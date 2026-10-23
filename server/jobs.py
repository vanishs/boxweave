"""Explicit local command jobs with durable, private receipts (Linux/POSIX)."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import selectors
import signal
import socket
import subprocess
import sys
import threading
import time
import uuid

from provenance import atomic_json, utc

ROOT = Path.home() / '.local/state/boxweave/jobs'
TERMINAL = {'succeeded', 'failed', 'timed_out', 'output_limit', 'launch_failed'}


def private_directory(path):
    path = Path(path).absolute()
    if path.is_symlink(): raise ValueError('Refusing symlink directory')
    path.mkdir(parents=True, mode=0o700, exist_ok=True)
    if path.stat().st_uid != os.getuid(): raise ValueError('State directory has another owner')
    path.chmod(0o700)
    return path


def job_path(root, ident):
    if not re.fullmatch(r'[a-f0-9]{32}', ident): raise ValueError('Invalid job id')
    path = Path(root) / ident
    if path.is_symlink(): raise ValueError('Refusing symlink job')
    return path


def read_json(path):
    if path.is_symlink() or path.stat().st_size > 65536: raise ValueError('Invalid state file')
    return json.loads(path.read_text(encoding='utf-8'))


def start(argv, timeout=600, output_limit=1048576, root=ROOT, cwd=None):
    if os.name != 'posix': raise ValueError('Jobs require POSIX')
    if not argv or not 0 < timeout <= 86400 or not 1024 <= output_limit <= 16777216: raise ValueError('Invalid job limits')
    # Credentials do not belong in argv or receipts. Use the program's protected file interface.
    if any(re.search(r'(?i)(password|secret|token|authorization)[=:]|^--?(password|secret|token|authorization)$|eyJ[\w-]+\.', a) for a in argv):
        raise ValueError('Credential-like command arguments are not supported')
    if len(json.dumps(argv)) > 16384: raise ValueError('Command arguments too large')
    base = private_directory(root); ident = uuid.uuid4().hex
    p = private_directory(base / ident)
    request = {'argv': argv, 'cwd': str(Path(cwd or Path.cwd()).resolve()), 'timeout': timeout, 'output_limit': output_limit}
    atomic_json(p / 'request.json', request)
    receipt = {'schema_version': 1, 'job_id': ident, 'state': 'queued', 'created_at': utc(),
               'execution_host': socket.gethostname(), 'execution_uid': os.getuid(),
               'created_epoch': time.time(), 'exit_code': None, 'timeout_seconds': timeout,
               'command': {'executable': Path(argv[0]).name, 'argument_count': len(argv)-1,
                           'sha256': hashlib.sha256(json.dumps(argv).encode()).hexdigest()},
               'output_truncated': False, 'remote_descendants_termination': 'not_verified'}
    atomic_json(p / 'receipt.json', receipt)
    try:
        process=subprocess.Popen([sys.executable, '-B', str(Path(__file__).resolve()), '_worker', str(p)],
                                 stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
        # Reap while the submitting process lives; if it exits, the worker remains detached.
        threading.Thread(target=process.wait,daemon=True).start()
    except OSError:
        receipt.update(state='launch_failed', finished_at=utc()); atomic_json(p / 'receipt.json', receipt)
    return {'job_id': ident, 'state': receipt['state'], 'state_directory': str(base)}


class SafeLog:
    """Redact recognized credentials before writing; drop overlong lines, bound memory."""
    def __init__(self, path):
        self.stream = open(path, 'xb'); os.chmod(path, 0o600)
        self.pending = b''; self.discard = False; self.pem = False
        self.omitted = False

    def write_line(self, line):
        text = line.decode('utf-8', errors='replace')
        if '-----BEGIN ' in text and 'PRIVATE KEY-----' in text: self.pem = True
        if self.pem:
            if 'PRIVATE KEY-----' in text and '-----END ' in text: self.pem = False
            text = '[private key content omitted]\n'
        else:
            text = re.sub(r'eyJ[A-Za-z0-9_.-]+', '[JWT omitted]', text)
            text = re.sub(r'''(?i)((?:["']?(?:password|secret|token|authorization)["']?)\s*[=:]\s*).*''', r'\1[omitted]', text)
        self.stream.write(text.encode()); self.stream.flush()

    def feed(self, data):
        self.pending += data
        while b'\n' in self.pending:
            line, self.pending = self.pending.split(b'\n', 1)
            if not self.discard and len(line) <= 4096: self.write_line(line + b'\n')
            else:
                self.omitted = True; self.stream.write(b'[overlong line omitted]\n')
            self.discard = False
        if len(self.pending) > 4096:
            self.pending = b''; self.discard = True; self.omitted = True

    def close(self):
        if self.pending and not self.discard: self.write_line(self.pending)
        self.stream.close()


def worker(path):
    import fcntl
    p = private_directory(path)
    with open(p / 'worker.lock', 'a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        request, receipt = read_json(p / 'request.json'), read_json(p / 'receipt.json')
        receipt.update(state='running', started_at=utc(), worker_pid=os.getpid())
        atomic_json(p / 'receipt.json', receipt)
        proc, streams, selector = None, [], selectors.DefaultSelector()
        started = time.monotonic(); total = 0; reason = None
        def kill_group():
            if proc is not None:
                try: os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError: pass
        try:
            proc = subprocess.Popen(request['argv'], cwd=request['cwd'], stdin=subprocess.DEVNULL,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
            receipt['command_pid'] = proc.pid; atomic_json(p / 'receipt.json', receipt)
            for name, pipe in (('stdout', proc.stdout), ('stderr', proc.stderr)):
                log = SafeLog(p / (name + '.log')); streams.append(log)
                selector.register(pipe, selectors.EVENT_READ, log)
            while selector.get_map():
                remaining = request['timeout'] - (time.monotonic() - started)
                if remaining <= 0:
                    reason = 'timed_out'; kill_group(); break
                for key, _ in selector.select(min(remaining, 0.25)):
                    data = os.read(key.fd, 8192)
                    if not data:
                        selector.unregister(key.fileobj); key.fileobj.close(); continue
                    total += len(data)
                    if total > request['output_limit']:
                        reason = 'output_limit'; kill_group(); break
                    key.data.feed(data)
                if reason: break
            if reason is None:
                try: proc.wait(timeout=max(0.01, request['timeout'] - (time.monotonic()-started)))
                except subprocess.TimeoutExpired: reason = 'timed_out'; kill_group()
            try: code = proc.wait(timeout=3)
            except subprocess.TimeoutExpired: code = None
            receipt.update(state=reason or ('succeeded' if code == 0 else 'failed'), exit_code=code,
                           output_truncated=reason == 'output_limit')
        except Exception as error:
            kill_group()
            if proc:
                try: proc.wait(timeout=3)
                except subprocess.TimeoutExpired: pass
            receipt.update(state='launch_failed', error=type(error).__name__, exit_code=None)
        finally:
            for key in list(selector.get_map().values()): key.fileobj.close()
            selector.close()
            for log in streams: log.close()
            receipt['output_truncated'] = receipt.get('output_truncated',False) or any(log.omitted for log in streams)
            receipt.update(finished_at=utc(), elapsed_seconds=round(time.monotonic()-started, 3), output_bytes_observed=total)
            atomic_json(p / 'receipt.json', receipt)


def status(ident, root=ROOT, logs=False):
    import fcntl
    p = job_path(root, ident); receipt = read_json(p / 'receipt.json')
    if receipt.get('state') not in TERMINAL:
        # The lock, not a potentially recycled PID, establishes a running worker.
        with open(p / 'worker.lock', 'a') as lock:
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                if time.time() - receipt.get('created_epoch', 0) > 5:
                    receipt.update(recorded_state=receipt['state'], state='unknown', exit_code=None)
            except BlockingIOError: pass
    if logs:
        receipt['logs'] = {}
        for name in ('stdout','stderr'):
            file = p / (name + '.log')
            if file.is_file() and not file.is_symlink():
                with file.open('rb') as stream:
                    stream.seek(max(0, file.stat().st_size - 8192))
                    receipt['logs'][name] = stream.read(8192).decode(errors='replace')
    return receipt


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest='action', required=True)
    start_parser = sub.add_parser('start')
    start_parser.add_argument('--timeout', type=float, default=600)
    start_parser.add_argument('--output-limit', type=int, default=1048576)
    start_parser.add_argument('--state-dir', type=Path, default=ROOT)
    start_parser.add_argument('command', nargs=argparse.REMAINDER)
    query = sub.add_parser('status'); query.add_argument('id')
    query.add_argument('--state-dir', type=Path, default=ROOT); query.add_argument('--logs', action='store_true')
    hidden = sub.add_parser('_worker'); hidden.add_argument('path')
    args = p.parse_args(argv)
    try:
        if args.action == '_worker': worker(args.path); return 0
        if args.action == 'start':
            command = args.command[1:] if args.command[:1] == ['--'] else args.command
            result = start(command, args.timeout, args.output_limit, args.state_dir)
        else: result = status(args.id, args.state_dir, args.logs)
        print(json.dumps(result, indent=2)); return 0
    except (OSError, ValueError) as error:
        print(json.dumps({'error': type(error).__name__, 'message': str(error) if isinstance(error, ValueError) else 'Job state unavailable'})); return 2


if __name__ == '__main__': raise SystemExit(main())
