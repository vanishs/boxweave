"""Bounded, non-secret installation provenance; Python standard library only."""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import selectors
import signal
import stat
import subprocess
import tempfile
import time


def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def atomic_json(path, value):
    path = Path(path)
    fd, temp = tempfile.mkstemp(prefix='.' + path.name + '.', dir=path.parent)
    try:
        with os.fdopen(fd, 'w', encoding='utf-8') as stream:
            json.dump(value, stream, indent=2, ensure_ascii=True)
            stream.write('\n'); stream.flush(); os.fsync(stream.fileno())
        os.chmod(temp, 0o600)
        os.replace(temp, path)
    finally:
        if os.path.exists(temp): os.unlink(temp)


def digest(path):
    try:
        if not stat.S_ISREG(Path(path).stat().st_mode): return None
        h = hashlib.sha256()
        with open(path, 'rb') as stream:
            for chunk in iter(lambda: stream.read(65536), b''): h.update(chunk)
        return h.hexdigest()
    except OSError:
        return None


def bounded_output(args, limit=4096):
    proc=None
    try:
        proc=subprocess.Popen(args,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,start_new_session=True)
        data=b'';deadline=time.monotonic()+5
        with selectors.DefaultSelector() as selector:
            selector.register(proc.stdout,selectors.EVENT_READ)
            while True:
                remaining=deadline-time.monotonic()
                if remaining<=0 or not selector.select(remaining): return None
                chunk=os.read(proc.stdout.fileno(),8192)
                if not chunk: break
                data+=chunk
                if len(data)>limit:return None
        proc.wait(timeout=max(.01,deadline-time.monotonic()))
        return data.decode(errors='replace') if proc.returncode==0 else None
    except (OSError, subprocess.TimeoutExpired):
        return None
    finally:
        if proc:
            if proc.poll() is None:
                try:os.killpg(proc.pid,signal.SIGKILL)
                except ProcessLookupError:pass
                proc.wait(timeout=2)
            proc.stdout.close()


def version(binary):
    text=bounded_output([binary,'--version'])
    match=re.search(r'(?<!\w)v?(\d+\.\d+\.\d+(?:[-+][\w.-]+)?)',text or '')
    return match.group(1) if match else None


def incus_server_version():
    # Installation has already validated/started Incus; do not call this from diagnostics.
    try:
        data=json.loads(bounded_output(['incus','--force-local','query','/1.0'],65536) or '{}')
        data=data.get('metadata',data)
        value=data.get('environment',{}).get('server_version')
        return value if isinstance(value,str) and re.fullmatch(r'[0-9A-Za-z.+-]{1,64}',value) else None
    except (ValueError,AttributeError):return None


def build(directory, installer, repo, ref, commit, backend, daemon):
    p = Path(directory)
    names = ('cli.py', 'jobs.py', 'mcp_client.py', 'smoke.py', 'provenance.py', 'hostkeys.py', 'tokens.py', 'clients.py')
    files = {'server/' + name: digest(p / 'server' / name) for name in names}
    for client in ('clients/windows/setup.ps1', 'clients/posix/setup.py'):
        files[client] = digest(p / client)
    files['VERSION'] = digest(p / 'VERSION')
    local_hash, reference_hash = digest(installer), digest(p / 'reference-install.sh')
    return {'schema_version': 1, 'state': 'pending', 'prepared_at': utc(),
            'completed_at': None, 'boxweave_version': (p / 'VERSION').read_text().strip(),
            'source': {'repository': repo, 'requested_ref': ref, 'resolved_commit': commit},
            'installer': {'sha256': local_hash, 'reference_sha256': reference_hash,
                          'differs_from_source': local_hash != reference_hash if local_hash and reference_hash else None},
            'mcp_backend': {'release': backend, 'sha256': digest(p / ('mcp-server-linux-' + os.environ['ARCH']))},
            'daemon': {'reported_version': version(daemon), 'sha256': digest(daemon)},
            'incus_server_version': incus_server_version(),
            'component_sha256': files}


def read_public(path):
    """Never dump arbitrary manifest fields or follow credential references."""
    try:
        p = Path(path)
        if p.stat().st_size > 65536: raise ValueError('Manifest too large')
        d = json.loads(p.read_text(encoding='utf-8'))
        if not isinstance(d, dict) or d.get('schema_version') != 1: raise ValueError('Invalid manifest')
        def text(value, pattern, limit=200):
            return value if isinstance(value, str) and len(value) <= limit and re.fullmatch(pattern, value) else None
        def section(key, fields):
            source = d.get(key, {})
            if not isinstance(source, dict): source = {}
            return {name: text(source.get(name), pattern) for name, pattern in fields.items()}
        result = {'status': 'available', 'schema_version': 1,
                  'state': text(d.get('state'), r'(pending|complete)'),
                  'boxweave_version': text(d.get('boxweave_version'), r'[0-9A-Za-z.+-]+'),
                  'incus_server_version': text(d.get('incus_server_version'), r'[0-9A-Za-z.+-]+'),
                  'prepared_at': text(d.get('prepared_at'), r'[0-9T:.+Z-]+'),
                  'completed_at': text(d.get('completed_at'), r'[0-9T:.+Z-]+'),
                  'source': section('source', {'repository': r'[\w.-]+/[\w.-]+', 'requested_ref': r'[\w.-]+', 'resolved_commit': r'[a-f0-9]{40}'}),
                  'installer': section('installer', {'sha256': r'[a-f0-9]{64}', 'reference_sha256': r'[a-f0-9]{64}'}),
                  'mcp_backend': section('mcp_backend', {'release': r'v[0-9.]+', 'sha256': r'[a-f0-9]{64}'}),
                  'daemon': section('daemon', {'reported_version': r'[0-9A-Za-z.+-]+', 'sha256': r'[a-f0-9]{64}'})}
        installer = d.get('installer', {})
        flag = installer.get('differs_from_source') if isinstance(installer, dict) else None
        result['installer']['differs_from_source'] = flag if type(flag) is bool else None
        hashes = d.get('component_sha256', {})
        result['component_sha256'] = {key: value for key, value in hashes.items()
                                     if key in ('VERSION', 'clients/windows/setup.ps1', 'clients/posix/setup.py', *('server/' + n for n in ('cli.py','jobs.py','mcp_client.py','smoke.py','provenance.py','hostkeys.py','tokens.py','clients.py')))
                                     and text(value, r'[a-f0-9]{64}')} if isinstance(hashes, dict) else {}
        return result
    except FileNotFoundError:
        return {'status': 'not_recorded'}
    except PermissionError:
        return {'status': 'not_readable'}
    except (OSError, ValueError, TypeError):
        return {'status': 'invalid'}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest='action', required=True)
    b = sub.add_parser('build')
    for key in ('directory','installer','repo','ref','commit','backend','daemon'): b.add_argument('--' + key, required=True)
    c = sub.add_parser('complete'); c.add_argument('path')
    args = p.parse_args()
    if args.action == 'build':
        data = build(args.directory, args.installer, args.repo, args.ref, args.commit, args.backend, args.daemon)
        atomic_json(Path(args.directory) / 'install-manifest.json', data)
    else:
        path = Path(args.path); data = json.loads(path.read_text())
        data.update(state='complete', completed_at=utc()); atomic_json(path, data)


if __name__ == '__main__': main()
