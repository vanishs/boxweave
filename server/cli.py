#!/usr/bin/env python3
"""BoxWeave diagnostics and explicitly requested local jobs. Standard library only."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import threading
from urllib.parse import quote, urlsplit
from urllib.error import HTTPError, URLError
from urllib.request import build_opener, HTTPRedirectHandler, ProxyHandler

sys.path.insert(0, str(Path(__file__).resolve().parent))
from provenance import read_public

CONFIG = Path('/etc/boxweave/config.json')
SERVICES = ('incus.service', 'containarium.service', 'boxweave-hostkeys.timer', 'boxweave-token.timer')
ONESHOTS = ('boxweave-hostkeys.service', 'boxweave-token.service')
VERSION_FILE = Path(__file__).resolve().parent / 'VERSION'
if not VERSION_FILE.is_file():
    VERSION_FILE = Path(__file__).resolve().parents[1] / 'VERSION'


def product_version():
    try:
        value = VERSION_FILE.read_text(encoding='utf-8').strip()
    except OSError:
        return 'unknown'
    return value if re.fullmatch(r'[0-9A-Za-z.+-]{1,64}', value) else 'unknown'


def command(args, timeout=5, output_limit=262144):
    """No shell, stdin, credentials, or unbounded command wait."""
    try:
        p = subprocess.Popen(args, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                             start_new_session=os.name == 'posix',
                             env={**os.environ, 'LC_ALL': 'C', 'SYSTEMD_COLORS': '0'})
        chunks, overflow = [], threading.Event()
        def stop():
            try:
                if os.name == 'posix':
                    os.killpg(p.pid, signal.SIGKILL)
                else:
                    p.kill()
            except ProcessLookupError:
                pass
        def read():
            size = 0
            while True:
                b = p.stdout.read1(8192)
                if not b:
                    return
                size += len(b)
                if size > output_limit:
                    overflow.set()
                    stop()
                    return
                chunks.append(b)
        reader = threading.Thread(target=read, daemon=True)
        reader.start()
        timed_out = False
        try:
            p.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
            stop()
            p.wait(timeout=2)
        reader.join(timeout=1)
        pipe_timeout = reader.is_alive()
        if pipe_timeout:
            stop()
            reader.join(timeout=1)
        if not reader.is_alive():
            p.stdout.close()
        error = 'timeout' if timed_out or pipe_timeout else 'output_limit' if overflow.is_set() else None if p.returncode == 0 else 'nonzero_exit'
        return {'exit': p.returncode, 'stdout': '' if error else b''.join(chunks).decode(errors='replace'), 'error': error}
    except FileNotFoundError:
        return {'exit': None, 'stdout': '', 'error': 'not_installed'}
    except subprocess.TimeoutExpired:
        return {'exit': None, 'stdout': '', 'error': 'timeout'}
    except OSError:
        return {'exit': None, 'stdout': '', 'error': 'unavailable'}


def load_config(path):
    try:
        # Read the configuration itself, never the secret/token paths it refers to.
        if path.stat().st_size > 65536:
            return None, 'configuration_too_large'
        data = json.loads(path.read_text(encoding='utf-8'))
        if not isinstance(data, dict):
            return None, 'invalid_configuration'
        return data, None
    except FileNotFoundError:
        return None, 'not_configured'
    except PermissionError:
        return None, 'configuration_not_readable (try sudo for diagnostics)'
    except (OSError, ValueError):
        return None, 'invalid_configuration'


def component_versions():
    result = {}
    for name, args in [('incus_client', ['incus', '--version']),
                       ('containarium_cli', ['containarium', 'version'])]:
        r = command(args)
        match = re.search(r'(?<!\w)v?(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)', r['stdout']) if r['exit'] == 0 else None
        result[name] = {'version': match.group(1) if match else None,
                        'status': 'available' if match else r['error'] or 'version_not_reported'}
    return result


def service_state(unit):
    r = command(['systemctl', 'show', unit, '-p', 'LoadState', '-p', 'ActiveState',
                 '-p', 'SubState', '-p', 'Result', '-p', 'MainPID', '-p', 'UnitFileState',
                 '-p', 'ExecMainStatus', '-p', 'ExecMainStartTimestampMonotonic'])
    values = dict(line.split('=', 1) for line in r['stdout'].splitlines() if '=' in line)
    # Only return bounded state identifiers, never ExecStart/Environment or raw output.
    safe = {key: value for key, value in values.items()
            if key in ('LoadState', 'ActiveState', 'SubState', 'Result', 'MainPID',
                       'UnitFileState', 'ExecMainStatus', 'ExecMainStartTimestampMonotonic')
            and re.fullmatch(r'[a-z0-9_-]{1,64}', value)}
    if safe.get('LoadState') == 'not-found':
        status = 'missing'
    elif safe.get('ActiveState') == 'activating':
        status = 'starting'
    elif safe.get('ActiveState') == 'failed':
        status = 'failed'
    elif safe.get('ActiveState') == 'active':
        status = 'active'
    elif safe.get('ActiveState') == 'inactive':
        status = 'inactive'
    else:
        status = 'unknown'
    return {'status': status, 'state': safe, 'error': r['error'] if not safe else None}


def environment():
    r = command(['systemd-detect-virt', '--container'])
    kind = r['stdout'].strip()
    container = r['exit'] == 0
    physical = r['exit'] == 1 and kind == 'none'
    evidence = {'container': container if container or physical else None,
                'virtualization': kind if re.fullmatch(r'[a-z0-9_-]{1,40}', kind) else 'unknown'}
    try:
        label = Path('/proc/self/attr/apparmor/current').read_text().strip()
    except OSError:
        label = None
    ns = {}
    for key in ('.ns_name', '.ns_level', '.ns_stacked', '.stacked'):
        try:
            ns[key] = (Path('/sys/kernel/security/apparmor') / key).read_text().strip()[:256]
        except OSError:
            ns[key] = None
    evidence['apparmor'] = {'process_label': label, 'namespace': ns,
                            'effective_device_filtering': 'not_verified'}
    evidence['notes'] = []
    if container:
        evidence['notes'].append('Nested host: inspect parent-side confinement separately; a child namespace label is not the complete host policy.')
    if label == 'unconfined' and (ns.get('.stacked') == 'yes' or ns.get('.ns_stacked') == 'yes'):
        evidence['notes'].append('Stacking is reported. Host-side parent and process attribution are still required; do not mark confinement passed or failed from this label alone.')
    return evidence


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def api_reachability(config):
    """Optional unauthenticated, loopback-only probe. Never follows redirects or reads bodies."""
    uri = config.get('api_url', 'http://127.0.0.1:8080')
    if not isinstance(uri, str):
        return {'status': 'skipped', 'reason': 'invalid_api_url'}
    try:
        u = urlsplit(uri)
        if (u.scheme not in ('http', 'https') or u.hostname not in ('127.0.0.1', '::1', 'localhost')
                or u.username or u.password or u.query or u.fragment or u.path not in ('', '/')):
            return {'status': 'skipped', 'reason': 'probe_requires_plain_loopback_origin_without_credentials'}
        opener = build_opener(ProxyHandler({}), NoRedirect())
        try:
            with opener.open(uri.rstrip('/') + '/v1/containers', timeout=3) as response:
                code = response.status
        except HTTPError as error:
            code = error.code
            error.close()
        return {'status': 'reachable' if code in (200, 401, 403) else 'unexpected_http_status',
                'http_status': code, 'authenticated_health': 'not_verified'}
    except (ValueError, OSError, URLError):
        return {'status': 'unreachable', 'authenticated_health': 'not_verified'}


def inspect_box(name, project):
    for value in (name, project):
        if not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}', value):
            raise ValueError('Invalid instance or project name')
    path = '/1.0/instances/' + quote(name, safe='') + '?project=' + quote(project, safe='')
    r = command(['incus', '--force-local', 'query', path], timeout=10)
    if r['exit'] != 0:
        return {'status': 'error', 'error': r['error'], 'name': name, 'project': project,
                'note': 'Exact-name query failed; no suffix substitution or automatic retry was performed.'}, 1
    try:
        d = json.loads(r['stdout'])
        if not isinstance(d, dict) or d.get('name') != name:
            raise ValueError('Unexpected instance identity')
        cfg = d.get('expanded_config', {})
        devices = d.get('expanded_devices', {})
        if not isinstance(cfg, dict) or not isinstance(devices, dict):
            raise ValueError('Invalid metadata')
        binding = {k: d.get(k) for k in ('created_at', 'config', 'expanded_config', 'devices', 'expanded_devices', 'profiles', 'type')}
        selected = ('security.nesting', 'security.privileged', 'boot.autostart', 'security.idmap.isolated',
                    'security.idmap.size', 'security.idmap.base', 'security.syscalls.intercept.bpf', 'nvidia.runtime')
        summary = {'status': 'ok', 'name': name, 'project': project, 'state': d.get('status'), 'type': d.get('type'),
                   'uuid': d.get('config', {}).get('volatile.uuid'),
                   'config_sha256': hashlib.sha256(json.dumps(binding, sort_keys=True).encode()).hexdigest(),
                   'security': {k: cfg.get(k) for k in selected},
                   'devices': {key: {k: value.get(k) for k in ('type', 'network', 'pool', 'path') if k in value}
                               for key, value in devices.items() if isinstance(value, dict)},
                   'raw_config': {key: {'present': bool(cfg.get(key)),
                                       'sha256': hashlib.sha256(str(cfg.get(key, '')).encode()).hexdigest()}
                                  for key in ('raw.lxc', 'raw.apparmor')},
                   'notes': ['UUIDs may be inherited by clones. Bind operations to host, project, exact name and fresh configuration evidence.',
                             'A snapshot is not an authorization token or a complete security assessment.']}
        if not any(v.get('type') == 'nic' for v in devices.values() if isinstance(v, dict)):
            summary['notes'].append('No NIC device is present. An internal nested bridge address is not a verified SSH or LAN endpoint.')
        return summary, 0
    except (ValueError, TypeError, AttributeError):
        return {'status': 'error', 'error': 'invalid_instance_metadata', 'name': name, 'project': project}, 1


def health(args):
    config, error = load_config(args.config)
    services = {unit: service_state(unit) for unit in SERVICES}
    oneshots = {unit: service_state(unit) for unit in ONESHOTS}
    result = {'command': args.command, 'boxweave': product_version(),
              'configuration': {'status': 'available' if config is not None else error}, 'services': services,
              'oneshots': oneshots}
    if config is not None:
        web_path = args.config.parent / 'web.json'
        web_config, web_error = load_config(web_path)
        web_units = {unit: service_state(unit) for unit in ('boxweave-web.service','boxweave-web-caddy.service','boxweave-web-token.timer')}
        result['web'] = {'configuration': 'available' if web_config else web_error, 'services': web_units}
        if web_config:
            import socket
            port = web_config.get('backend_port')
            if type(port) is int and 1024 <= port <= 65535:
                try:
                    with socket.create_connection(('127.0.0.1', port), 1): result['web']['listener'] = 'reachable'
                except OSError: result['web']['listener'] = 'unreachable'
        result['configuration']['provider'] = config.get('provider') if config.get('provider') == 'containarium' else 'unknown'
        result['configuration']['schema_version'] = config.get('schema_version') if type(config.get('schema_version')) is int else None
    if args.command == 'doctor':
        result['environment'] = environment()
        result['components'] = component_versions()
        result['notes'] = ['Read-only diagnostics; no repairs, service activation, package updates, or security-policy changes.',
                           'inactive and starting are observations, not proof of corruption. Recheck readiness before a dependent action.',
                           'BoxWeave, the provider CLI, the MCP backend and the Incus server have separate versions.']
        if args.api:
            result['api'] = api_reachability(config) if config is not None else {'status': 'skipped', 'reason': 'configuration_unavailable'}
    try:
        marker = args.config.parent / 'update-in-progress'
        marker.stat(); pending = True
    except FileNotFoundError:
        pending = False
    except OSError:
        pending = None
    installation = installation_state(config is not None, services, oneshots, pending)
    if 'api' in result:
        api = result['api']['status']
        if api == 'unexpected_http_status': installation.update(state='failed')
        elif api in ('unreachable', 'skipped') and installation['state'] == 'ready': installation.update(state='incomplete')
        if api != 'reachable': installation['reasons'].append('requested_api_probe_' + api)
    if config is not None and (args.config.parent / 'web.json').exists():
        web = result.get('web', {})
        if web.get('listener') != 'reachable' or any(r['status'] != 'active' for r in web.get('services', {}).values()):
            installation['reasons'].append('web_not_ready')
            if installation['state'] == 'ready': installation['state'] = 'incomplete'
    result['installation'] = installation
    result['installation_state'] = result['overall'] = installation['state']
    return result, 0 if installation['state'] == 'ready' else 1 if installation['state'] == 'failed' else 2


def installation_state(configured, services, oneshots, pending):
    reasons, failed, starting = [], False, False
    if not configured: reasons.append('configuration_unavailable')
    if pending is not False: reasons.append('update_pending' if pending else 'update_marker_unreadable')
    for unit, record in {**services, **oneshots}.items():
        status, state = record['status'], record.get('state', {})
        if status == 'failed' or state.get('Result') not in (None, 'success'):
            failed = True; reasons.append(unit + ':failed')
        elif status == 'starting' or (unit in oneshots and status == 'active' and state.get('SubState') != 'exited'):
            starting = True; reasons.append(unit + ':starting')
        elif unit in services:
            if status != 'active': reasons.append(unit + ':' + status)
            if unit.endswith('.timer') and state.get('UnitFileState') != 'enabled': reasons.append(unit + ':not_enabled')
            if unit == 'containarium.service' and state.get('UnitFileState') != 'enabled': reasons.append(unit + ':not_enabled')
        else:
            completed = (status in ('inactive', 'active') and state.get('Result') == 'success'
                         and state.get('ExecMainStatus') == '0'
                         and state.get('ExecMainStartTimestampMonotonic', '0').isdigit()
                         and int(state.get('ExecMainStartTimestampMonotonic', '0')) > 0)
            if not completed: reasons.append(unit + ':successful_run_not_verified')
    state = 'failed' if failed else 'incomplete' if pending is not False or not configured else 'starting' if starting else 'incomplete' if reasons else 'ready'
    return {'state': state, 'pending_update': pending, 'reasons': reasons,
            'scope': 'local_installation_readiness; not authenticated MCP or security certification'}


def parser():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--json', action='store_true', help='Machine-readable output')
    p.add_argument('--version', action='version', version='BoxWeave ' + product_version())
    commands = p.add_subparsers(dest='command', required=True)
    def output_options(sub):
        sub.add_argument('--json', action='store_true', default=argparse.SUPPRESS)
    v = commands.add_parser('version', help='BoxWeave source version; no provider connection')
    output_options(v)
    v.add_argument('--components', action='store_true', help='Also query client binary versions (not server/API health)')
    v.add_argument('--verbose', action='store_true', help='Include recorded installation provenance')
    v.add_argument('--manifest', type=Path, default=CONFIG.parent / 'install-manifest.json')
    for name in ('status', 'doctor'):
        sub = commands.add_parser(name, help='Read-only ' + name)
        output_options(sub)
        sub.add_argument('--config', type=Path, default=CONFIG)
        if name == 'doctor':
            sub.add_argument('--api', action='store_true', help='Unauthenticated loopback reachability probe; no token access')
    box = commands.add_parser('box', help='Read-only instance information').add_subparsers(dest='box_command', required=True)
    inspect = box.add_parser('inspect', help='Query an exact Incus name; never appends -container')
    output_options(inspect)
    inspect.add_argument('name')
    inspect.add_argument('--project', default='default', help='Explicit Incus project (default: default)')
    from clients import add_parser
    add_parser(commands)
    web = commands.add_parser('web', help='Independent web administrator and health commands')
    web.add_argument('arguments', nargs=argparse.REMAINDER)
    job = commands.add_parser('job', help='Explicit local command jobs with durable receipts')
    job.add_argument('arguments', nargs=argparse.REMAINDER)
    smoke = commands.add_parser('selftest', help='Plan or run an approved nested MCP smoke test')
    smoke.add_argument('arguments', nargs=argparse.REMAINDER)
    return p


def main(argv=None):
    args = parser().parse_args(argv)
    if args.command == 'web':
        os.execv('/usr/bin/python3', ['/usr/bin/python3', '/usr/local/libexec/boxweave/web_entry.py', *args.arguments])
    if args.command == 'client':
        from clients import run
        return run(args)
    if args.command in ('job', 'selftest'):
        if args.command == 'job':
            from jobs import main as entry
        else:
            from smoke import main as entry
        return entry(args.arguments)
    if args.command == 'version':
        result, code = {'boxweave': product_version()}, 0
        if args.components:
            result['components'] = component_versions()
        if args.verbose:
            result['provenance'] = read_public(args.manifest)
    elif args.command == 'box':
        try:
            result, code = inspect_box(args.name, args.project)
        except ValueError as error:
            result, code = {'status': 'error', 'error': str(error)}, 2
    else:
        result, code = health(args)
    if args.json:
        print(json.dumps(result, ensure_ascii=True, indent=2))
    elif args.command == 'version' and not args.verbose:
        print('BoxWeave ' + result['boxweave'])
        for name, value in result.get('components', {}).items():
            print(name + ': ' + (value['version'] or value['status']))
    else:
        print(json.dumps(result, ensure_ascii=True, indent=2))
    return code


if __name__ == '__main__':
    raise SystemExit(main())
