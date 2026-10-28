#!/usr/bin/env python3
"""BoxWeave client enrollment for macOS / WSL CLI agents. Requires Python 3.11+, standard library.

All connection parameters arrive through BOXWEAVE_* environment variables set
by the generated enrollment script; nothing is embedded or base64-packed.
The SSH transport is addressed through a managed "Host boxweave" block in the
user's ~/.ssh/config, so agent registrations stay stable when the server
address changes: edit the block (or the generated script's EDIT ME header and
rerun) instead of re-registering anything.
"""
import argparse
import base64
import datetime
import json
import os
from pathlib import Path
import queue
import re
import shlex
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import time
import tomllib

BEGIN = '<!-- BEGIN BOXWEAVE MANAGED -->'
END = '<!-- END BOXWEAVE MANAGED -->'
SSH_BEGIN = '# BEGIN BOXWEAVE MANAGED'
SSH_END = '# END BOXWEAVE MANAGED'
LAUNCHER = '/usr/local/bin/boxweave-mcp'
AGENTS = ('codex', 'cursor', 'claude-code', 'zcode')
# The stdio MCP channel is long-lived: without traffic-based keepalive a silently
# dropped connection (idle NAT/conntrack eviction) hangs every later call. ZCode
# also kills tool calls at its 30s default; create-class calls need more.
SSH_KEEPALIVE_ARGS = ['-o', 'ServerAliveInterval=15', '-o', 'ServerAliveCountMax=4']
ZCODE_TIMEOUT_MS = 180000
CODEX_TOOL_TIMEOUT_SEC = 180
POLICY = """## BoxWeave remote development (Containarium)
- Use MCP server boxweave to list/provision containers and its dynamic connect tool to work inside a box.
- Develop, install dependencies, test and deploy INSIDE the assigned project LXC. Inspect existing files/Git state and preserve uncommitted work.
- Inspect existing resources first; reuse a suitable project box. Never modify unrelated or containarium-core-* boxes.
- The Ubuntu MCP endpoint is {ssh_host}:{ssh_port}, user {linux_user}. The transport key is restricted to the MCP launcher, not a general host shell credential.
- For create_container.ssh_keys, explicitly supply this FULL managed Ubuntu-to-container public key:
  {container_public_key}
- Never substitute a client transport key, a fingerprint, or another nesting layer's key. Never read/export private keys or JWTs.
- A create timeout is ambiguous: inspect the original logical name before retrying; never duplicate or force-recreate to bypass a timeout.
- Running does not prove SSH readiness. Use one deadline per phase (normally 600 seconds), poll every 15-30 seconds, and verify connect with whoami && hostname before project work. Report actual stdout and exit codes.
- After IP reuse, allow bounded trusted host-key reconciliation. Never disable host verification or edit known_hosts to bypass a mismatch. Diagnose persistent failures read-only.
- Service activity, HTTP 200/401/403 and CLI exit 0 do not prove authenticated MCP execution. Test initialize/initialized, tools/list and an actual read-only tools/call.
- Keep identities separate at each nested MCP host; direct Incus execution is not evidence of nested authenticated MCP execution.
- Use systemd/Compose only for requested persistent workloads; verify health and application autostart. LXC boot.autostart requires separate evidence.
- Publish ports only when requested. Inspect routes/conflicts and verify actual client connectivity. Passthrough does not provide LAN-only source filtering or TLS.
- Obtain explicit approval for destructive/host-wide changes or public exposure. Never change host SSH/firewall/provider credentials to make a test pass.
"""

def atomic_write(path, text, mode=0o600):
    path = Path(path)
    if path.is_symlink(): raise ValueError('Refusing configuration symlink: '+str(path))
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists():
        stamp = datetime.datetime.now().strftime('%Y%m%d-%H%M%S-%f')
        backup = path.with_name(path.name+'.backup-'+stamp)
        shutil.copyfile(path,backup); backup.chmod(mode)
    fd, temporary = tempfile.mkstemp(prefix='.boxweave-',dir=path.parent)
    try:
        with os.fdopen(fd,'w',encoding='utf-8') as output: output.write(text)
        os.chmod(temporary,mode)
        os.replace(temporary,path)
    finally:
        if os.path.exists(temporary): os.unlink(temporary)

def glm_settings(original, provider, model=None, token=None):
    """Merge only GLM environment settings; credentials never enter server payloads."""
    endpoints = {'zai': 'https://api.z.ai/api/anthropic',
                 'zhipu': 'https://open.bigmodel.cn/api/anthropic'}
    if provider not in endpoints: raise ValueError('Invalid Claude provider')
    if model is not None and (not isinstance(model, str) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.:\[\]-]{0,127}', model)):
        raise ValueError('Invalid GLM model')
    if not isinstance(token, str) or not token.strip() or any(c in token for c in '\r\n'):
        raise ValueError('Set GLM_API_KEY in the client environment before setup')
    data = json.loads(original) if original.strip() else {}
    if not isinstance(data, dict) or not isinstance(data.get('env', {}), dict):
        raise ValueError('Invalid Claude settings object')
    env = data.setdefault('env', {})
    env.update(ANTHROPIC_BASE_URL=endpoints[provider], ANTHROPIC_AUTH_TOKEN=token,
               API_TIMEOUT_MS='3000000')
    env.pop('ANTHROPIC_API_KEY', None)
    if model:
        for key in ('ANTHROPIC_MODEL', 'ANTHROPIC_DEFAULT_OPUS_MODEL',
                    'ANTHROPIC_DEFAULT_SONNET_MODEL', 'ANTHROPIC_DEFAULT_HAIKU_MODEL'):
            env[key] = model
    return json.dumps(data, indent=2)+'\n'

def zcode_update(original, command, args, replace=False):
    """Preserve native user config and reject unreviewed server conflicts."""
    data = json.loads(original) if original.strip() else {}
    if not isinstance(data, dict): raise ValueError('Invalid ZCode JSON object')
    mcp = data.setdefault('mcp', {})
    if not isinstance(mcp, dict): raise ValueError('Invalid ZCode MCP object')
    servers = mcp.setdefault('servers', {})
    if not isinstance(servers, dict): raise ValueError('Invalid ZCode MCP servers')
    expected = {'type': 'stdio', 'command': command, 'args': args, 'timeoutMs': ZCODE_TIMEOUT_MS}
    old = servers.get('boxweave')
    if old is not None and not replace:
        if not isinstance(old, dict) or any(old.get(k) != v for k,v in expected.items()) or set(old)-set(expected):
            raise ValueError('Existing ZCode boxweave entry differs; rerun with --replace after review')
    features = data.get('features', {})
    if not isinstance(features, dict) or features.get('mcp', True) is not True:
        raise ValueError('Enable MCP in ZCode before setup')
    servers['boxweave'] = expected
    return json.dumps(data, indent=2)+'\n'

def managed(original, policy):
    if original.count(BEGIN) != original.count(END) or original.count(BEGIN)>1:
        raise ValueError('Malformed BoxWeave instruction markers')
    block = BEGIN+'\n'+policy.strip()+'\n'+END
    if BEGIN in original:
        if original.index(END)<original.index(BEGIN): raise ValueError('Reversed instruction markers')
        return original[:original.index(BEGIN)]+block+original[original.index(END)+len(END):]
    return original.rstrip()+'\n\n'+block+'\n'

def ssh_config_managed(original, block):
    """Idempotently place the Host boxweave transport block between markers."""
    if original.count(SSH_BEGIN) != original.count(SSH_END) or original.count(SSH_BEGIN) > 1:
        raise ValueError('Malformed BoxWeave ssh_config markers')
    text = SSH_BEGIN+'\n'+block.strip('\n')+'\n'+SSH_END
    if SSH_BEGIN in original:
        if original.index(SSH_END) < original.index(SSH_BEGIN): raise ValueError('Reversed BoxWeave ssh_config markers')
        return original[:original.index(SSH_BEGIN)]+text+original[original.index(SSH_END)+len(SSH_END):]
    return original.rstrip()+'\n\n'+text+'\n'

def ssh_transport_block(host, port, user, key, known):
    return (
        'Host boxweave\n'
        '    HostName '+host+'\n'
        '    Port '+str(port)+'\n'
        '    User '+user+'\n'
        '    IdentityFile '+str(key)+'\n'
        '    IdentitiesOnly yes\n'
        '    IdentityAgent none\n'
        '    BatchMode yes\n'
        '    UserKnownHostsFile '+str(known)+'\n'
        '    StrictHostKeyChecking yes\n'
        '    HostKeyAlgorithms ssh-ed25519\n'
        '    ConnectTimeout 15\n'
        '    ServerAliveInterval 15\n'
        '    ServerAliveCountMax 4\n')

def parse_ssh_transport_block(text):
    """Read the managed block back; returns dict or None when absent."""
    if SSH_BEGIN not in text: return None
    segment = text[text.index(SSH_BEGIN)+len(SSH_BEGIN):]
    if SSH_END not in segment: raise ValueError('Malformed BoxWeave ssh_config markers')
    segment = segment[:segment.index(SSH_END)]
    values = {}
    for line in segment.splitlines():
        line = line.strip()
        if not line or line.startswith('#') or '=' in line: continue
        parts = line.split(None, 1)
        if len(parts) == 2 and parts[0] not in values:
            values[parts[0]] = parts[1].strip('"')
    return values

def toml_update(original, command, args, replace=False):
    parsed = tomllib.loads(original)
    servers = parsed.get('mcp_servers',{})
    if not isinstance(servers,dict): raise ValueError('Invalid MCP TOML table')
    existing = servers.get('boxweave')
    if isinstance(existing,dict) and not replace and set(existing)-{'command','args','startup_timeout_sec','tool_timeout_sec','tools'}:
        raise ValueError('Existing BoxWeave MCP entry has extra settings; use --replace after review')
    expected = {'command':command,'args':args}
    if existing is not None and not isinstance(existing,dict): raise ValueError('Invalid BoxWeave MCP entry')
    if existing is not None and not replace and any(existing.get(k)!=v for k,v in expected.items()):
        raise ValueError('Existing boxweave MCP entry differs; rerun with --replace after review')
    header = '[mcp_servers.boxweave]'
    pattern = r'(?ms)^\[mcp_servers\.boxweave\][ \t]*(?:#[^\n]*)?\n.*?(?=^\[|\Z)'
    if existing is not None and not re.search(pattern,original):
        raise ValueError('Unsupported existing BoxWeave TOML spelling; edit it explicitly')
    for nested in re.findall(r'(?m)^\[mcp_servers\.boxweave\.([^\]\n]+)\]', original):
        if not re.fullmatch(r'tools(?:\.[A-Za-z0-9_-]+)*', nested):
            raise ValueError('Nested BoxWeave MCP settings require explicit migration')
    block = header+'\ncommand = '+json.dumps(command,ensure_ascii=False)+'\nargs = '+json.dumps(args,ensure_ascii=False)+'\nstartup_timeout_sec = 30\ntool_timeout_sec = '+str(CODEX_TOOL_TIMEOUT_SEC)+'\n'
    updated = re.sub(pattern,lambda _:block,original,count=1) if existing is not None else original.rstrip()+'\n\n'+block
    check = tomllib.loads(updated)
    if existing is not None and 'tools' in existing and check['mcp_servers']['boxweave'].get('tools') != existing['tools']:
        raise ValueError('Inline tool settings require explicit migration')
    if check.get('mcp_servers',{}).get('boxweave',{}).get('command') != command:
        raise ValueError('MCP table is inline or otherwise cannot be safely extended')
    return updated

def json_update(original, command, args, replace=False):
    data = json.loads(original) if original.strip() else {}
    if not isinstance(data,dict): raise ValueError('Invalid Cursor JSON object')
    servers = data.setdefault('mcpServers',{})
    if not isinstance(servers,dict): raise ValueError('Invalid Cursor MCP servers')
    existing = servers.get('boxweave')
    if existing is not None and not isinstance(existing,dict): raise ValueError('Invalid BoxWeave MCP entry')
    if existing is not None and not replace and (existing.get('command')!=command or existing.get('args')!=args):
        raise ValueError('Existing boxweave MCP entry differs; rerun with --replace after review')
    servers['boxweave'] = {'command':command,'args':args}
    return json.dumps(data,indent=2)+'\n'

def checked(args, **kwargs):
    result = subprocess.run(args, **kwargs)
    if result.returncode: raise RuntimeError('Command failed: '+Path(args[0]).name+' (exit '+str(result.returncode)+')')
    return result

class DiagnosticError(RuntimeError):
    def __init__(self, code, stage, message):
        super().__init__(message)
        self.code, self.stage = code, stage


class DiagnosticTimeout(DiagnosticError, TimeoutError): pass


def probe(command, args, timeout=30, evidence=None):
    """Bound every pipe operation with one monotonic deadline; never return payloads."""
    import selectors
    evidence = evidence if evidence is not None else {}
    stage = 'transport'
    started = time.monotonic()
    deadline = started + timeout
    p = subprocess.Popen([command, *args], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                         stderr=subprocess.DEVNULL, start_new_session=True)
    evidence['process_exit_code'] = None
    selector = selectors.DefaultSelector()
    buffer = b''
    observed = 0
    ident = 0

    def remaining():
        value = deadline-time.monotonic()
        if value <= 0: raise TimeoutError('MCP timeout')
        return value

    def wait(stream, event):
        selector.register(stream, event)
        try:
            if not selector.select(remaining()): raise TimeoutError('MCP timeout')
        finally: selector.unregister(stream)

    def send(value):
        data = (json.dumps(value)+'\n').encode()
        while data:
            wait(p.stdin, selectors.EVENT_WRITE)
            try: count = os.write(p.stdin.fileno(), data)
            except BlockingIOError: continue
            data = data[count:]

    def invalid():
        raise DiagnosticError('PROTOCOL_INVALID', stage, 'Invalid MCP response')

    def call(method, params):
        nonlocal ident, buffer, observed
        ident += 1
        send({'jsonrpc':'2.0','id':ident,'method':method,'params':params})
        for _ in range(256):
            while b'\n' not in buffer:
                wait(p.stdout, selectors.EVENT_READ)
                try: chunk = os.read(p.stdout.fileno(), 65536)
                except BlockingIOError: continue
                if not chunk:
                    evidence['process_exit_code'] = p.poll()
                    raise DiagnosticError('TRANSPORT_FAILED', stage, 'MCP stream closed')
                observed += len(chunk)
                buffer += chunk
                if len(buffer)>2097152 or observed>8388608:
                    raise DiagnosticError('RESPONSE_LIMIT', stage, 'MCP response limit exceeded')
            line, buffer = buffer.split(b'\n', 1)
            if not line.strip(): continue
            try: value = json.loads(line)
            except (ValueError, UnicodeError): invalid()
            if not isinstance(value, dict) or value.get('jsonrpc') != '2.0': invalid()
            if 'method' in value:
                if 'id' in value:
                    send({'jsonrpc':'2.0','id':value['id'],'error':{'code':-32601,'message':'Unsupported client request'}})
                continue
            if type(value.get('id')) is not int or value['id'] != ident: invalid()
            if ('result' in value) == ('error' in value): invalid()
            if 'error' in value:
                raise DiagnosticError('PROVIDER_CALL_FAILED' if stage=='provider_call' else 'PROTOCOL_INVALID',
                                      stage, 'MCP returned a protocol error')
            if not isinstance(value['result'], dict): invalid()
            return value['result']
        raise DiagnosticError('RESPONSE_LIMIT', stage, 'MCP message budget exceeded')

    try:
        os.set_blocking(p.stdin.fileno(), False)
        os.set_blocking(p.stdout.fileno(), False)
        stage = 'initialize'
        result = call('initialize', {'protocolVersion':'2024-11-05','capabilities':{},
                                    'clientInfo':{'name':'boxweave-client','version':'0.4.0'}})
        if result.get('protocolVersion') != '2024-11-05':
            raise DiagnosticError('PROTOCOL_UNSUPPORTED', stage, 'Unsupported negotiated MCP protocol')
        if not isinstance(result.get('capabilities'),dict) or not isinstance(result['capabilities'].get('tools'),dict): invalid()
        evidence['protocol_version'] = result['protocolVersion']
        evidence['initialize'] = 'passed'
        send({'jsonrpc':'2.0','method':'notifications/initialized'})
        stage = 'discovery'
        params, seen = {}, set()
        for page in range(16):
            result = call('tools/list', params)
            tools = result.get('tools')
            if not isinstance(tools,list) or any(not isinstance(t,dict) or not isinstance(t.get('name'),str) for t in tools): invalid()
            evidence['tool_pages'] = page+1
            if any(t['name']=='list_containers' for t in tools): break
            cursor = result.get('nextCursor')
            if cursor is None:
                raise DiagnosticError('TOOL_UNAVAILABLE', stage, 'Provider list_containers tool unavailable')
            if not isinstance(cursor,str) or not cursor or cursor in seen: invalid()
            seen.add(cursor); params={'cursor':cursor}
        else: raise DiagnosticError('RESPONSE_LIMIT', stage, 'MCP page limit exceeded')
        evidence['discovery'] = 'passed'
        stage = 'provider_call'
        result = call('tools/call', {'name':'list_containers','arguments':{}})
        if 'isError' in result and type(result['isError']) is not bool: invalid()
        if result.get('isError'):
            raise DiagnosticError('PROVIDER_CALL_FAILED', stage, 'Read-only provider verification failed')
        content = result.get('content')
        if not isinstance(content,list): invalid()
        for block in content:
            if not isinstance(block,dict) or block.get('type') not in ('text','image','audio','resource','resource_link'): invalid()
            if block['type']=='text' and not isinstance(block.get('text'),str): invalid()
            if block['type'] in ('image','audio') and (not isinstance(block.get('data'),str) or not isinstance(block.get('mimeType'),str)): invalid()
            if block['type']=='resource' and not isinstance(block.get('resource'),dict): invalid()
            if block['type']=='resource_link' and (not isinstance(block.get('uri'),str)): invalid()
        evidence['provider_call'] = 'passed'
        return True
    except TimeoutError:
        raise DiagnosticTimeout('DEADLINE_EXCEEDED', stage, 'MCP timeout') from None
    finally:
        selector.close()
        p.stdin.close()
        try: os.killpg(p.pid,signal.SIGTERM)
        except ProcessLookupError: pass
        try: p.wait(timeout=2)
        except subprocess.TimeoutExpired:
            try: os.killpg(p.pid,signal.SIGKILL)
            except ProcessLookupError: pass
            p.wait(timeout=2)
        p.stdout.close()
        evidence['cleanup'] = 'completed'
        evidence['elapsed_ms'] = round((time.monotonic()-started)*1000)


def host_token(host, port):
    return '['+host+']:'+str(port) if port != 22 or ':' in host else host


def validate_server_host_key(value):
    if not isinstance(value, str) or not re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/=]+(?: [^\r\n]+)?', value):
        raise ValueError('Invalid BOXWEAVE_SERVER_HOST_KEY')
    blob = base64.b64decode(value.split()[1], validate=True)
    if len(blob) != 51 or blob[:15] != b'\x00\x00\x00\x0bssh-ed25519':
        raise ValueError('Invalid BOXWEAVE_SERVER_HOST_KEY')
    return value


def connection_from_env():
    """Read and validate the connection parameters exported by the generated script."""
    d = {'ssh_host': os.environ.get('BOXWEAVE_SSH_HOST', ''),
         'ssh_port': os.environ.get('BOXWEAVE_SSH_PORT', ''),
         'linux_user': os.environ.get('BOXWEAVE_SSH_USER', ''),
         'launcher': LAUNCHER,
         'agent': os.environ.get('BOXWEAVE_AGENT', ''),
         'runtime': os.environ.get('BOXWEAVE_RUNTIME', ''),
         'transport': os.environ.get('BOXWEAVE_TRANSPORT', '')}
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9.:-]{0,252}', d['ssh_host']):
        raise ValueError('Invalid or missing BOXWEAVE_SSH_HOST')
    if not re.fullmatch(r'[0-9]+', d['ssh_port']) or not 1 <= int(d['ssh_port']) <= 65535:
        raise ValueError('Invalid or missing BOXWEAVE_SSH_PORT')
    d['ssh_port'] = int(d['ssh_port'])
    if not re.fullmatch(r'[a-z_][a-z0-9_-]{0,63}', d['linux_user']):
        raise ValueError('Invalid or missing BOXWEAVE_SSH_USER')
    if d['agent'] not in AGENTS:
        raise ValueError('Invalid or missing BOXWEAVE_AGENT')
    if d['runtime'] not in ('native', 'wsl') or d['transport'] not in ('native', 'wsl'):
        raise ValueError('Invalid or missing BOXWEAVE_RUNTIME/BOXWEAVE_TRANSPORT')
    d['os'] = 'macos' if sys.platform == 'darwin' else 'windows'
    if d['os'] == 'macos':
        if d['runtime'] != 'native' or d['transport'] != 'native':
            raise ValueError('macOS supports native runtime and SSH only')
    else:
        if d['runtime'] != 'wsl' or d['transport'] != 'wsl':
            raise ValueError('This setup.py covers macOS native and the Windows WSL CLI profile only; use setup.ps1 for native Windows')
        if d['agent'] in ('cursor', 'zcode'):
            raise ValueError('WSL runtime supports Codex/Claude Code CLI only')
    return d


def diagnose(args):
    report={'schema_version':1,'operation':'client_diagnose','overall':'incomplete',
            'connection':'unverified','registration':'unverified','guidance':'unverified',
            'agent_ui':'unverified','stages':[],'process_exit_code':None,'cleanup':'not_started'}
    started=time.monotonic()
    def record(stage,status,code):
        report['stages'].append({'id':stage,'status':status,'code':code,
                                 'elapsed_ms':round((time.monotonic()-started)*1000)})
    def fail(code,stage,status='incomplete'):
        record(stage,status,code)
        raise DiagnosticError(code,stage,'Diagnostic check did not pass')
    def safe_file(path):
        if any(x.is_symlink() for x in (path,*path.parents)):
            fail('INPUT_INVALID','input')
        if path.exists() and path.stat().st_size>1048576: fail('INPUT_INVALID','input')
    try:
        if args.replace or args.bootstrap_identity or args.project_path: fail('INPUT_INVALID','input')
        try:
            if not 1<=args.timeout<=300: raise ValueError()
            d=connection_from_env()
            env_key=os.environ.get('BOXWEAVE_SERVER_HOST_KEY')
            if env_key: validate_server_host_key(env_key)
        except (ValueError,TypeError,KeyError,OSError): fail('INPUT_INVALID','input')
        record('input','passed','OK')
        if d['os']=='macos' and sys.platform!='darwin': fail('PROFILE_UNSUPPORTED','runtime')
        if d['runtime']=='wsl':
            try: wsl='microsoft' in Path('/proc/sys/kernel/osrelease').read_text().lower()
            except OSError: wsl=False
            if not wsl: fail('PROFILE_UNSUPPORTED','runtime')
        ssh=shutil.which('ssh')
        if not ssh: fail('RUNTIME_MISSING','runtime')
        report.update(runtime=d['os']+'-'+d['runtime'],transport=d['transport'])
        record('runtime','passed','OK')
        key=Path.home()/'.ssh/boxweave';known=Path.home()/'.ssh/boxweave_known_hosts'
        config=Path.home()/'.ssh/config'
        for path in (key,known,config): safe_file(path)
        if not key.is_file(): fail('IDENTITY_MISSING','trust')
        if not known.is_file(): fail('PIN_MISSING','trust')
        block=parse_ssh_transport_block(config.read_text(encoding='utf-8') if config.exists() else '')
        if not block: fail('TRANSPORT_MISSING','trust')
        expected_fields={'HostName':d['ssh_host'],'Port':str(d['ssh_port']),'User':d['linux_user'],
                         'IdentityFile':str(key),'UserKnownHostsFile':str(known)}
        if any(block.get(name)!=value for name,value in expected_fields.items()):
            fail('TRANSPORT_MISMATCH','trust','failed')
        token=host_token(d['ssh_host'],d['ssh_port'])
        observed_pin=known.read_text().strip()
        if env_key:
            expected_pin=token+' '+' '.join(env_key.split()[:2])
            if observed_pin!=expected_pin: fail('PIN_MISMATCH','trust','failed')
        elif observed_pin.split()[0:1]!=[token]:
            fail('PIN_MISMATCH','trust','failed')
        record('trust','passed','OK')
        transport=['-T','-i',str(key),'-o','IdentitiesOnly=yes','-o','BatchMode=yes',
                   '-o','StrictHostKeyChecking=yes','-o','UserKnownHostsFile='+str(known),
                   '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15',
                   *SSH_KEEPALIVE_ARGS,
                   '-p',str(d['ssh_port']),d['linux_user']+'@'+d['ssh_host'],d['launcher']]
        expected_transport=['boxweave',d['launcher']]
        home=Path(args.agent_home or os.environ.get('CODEX_HOME') or Path.home()/'.codex').expanduser()
        if d['agent']=='cursor': home=Path(args.agent_home or Path.home()/'.cursor').expanduser()
        if d['agent']=='zcode': home=Path(args.agent_home or Path.home()/'.zcode').expanduser()
        if d['agent']=='claude-code':
            report['registration']='unsupported'
            record('registration','incomplete','REGISTRATION_UNSUPPORTED')
            home=Path(os.environ.get('CLAUDE_CONFIG_DIR') or Path.home()/'.claude')
        else:
            config_file=home/('config.toml' if d['agent']=='codex' else 'cli/config.json' if d['agent']=='zcode' else 'mcp.json');safe_file(config_file)
            try:
                parsed=(tomllib.loads(config_file.read_text()) if d['agent']=='codex' else json.loads(config_file.read_text()))
                entry=(parsed.get('mcp',{}).get('servers',{}).get('boxweave') if d['agent']=='zcode'
                       else parsed.get('mcp_servers' if d['agent']=='codex' else 'mcpServers',{}).get('boxweave'))
                allowed={'command','args','type','enable','enabled','timeoutMs'} if d['agent']=='zcode' else {'command','args','startup_timeout_sec','tool_timeout_sec','enabled'}
                if not isinstance(entry,dict) or set(entry)-allowed: raise ValueError()
                matched=entry.get('command')==ssh and entry.get('args')==expected_transport and entry.get('enabled',True) is True and entry.get('enable',True) is True
                if d['agent']=='zcode': matched=matched and entry.get('type','stdio')=='stdio' and parsed.get('features',{}).get('mcp',True) is True
                report['registration']='matched' if matched else 'mismatch'
                record('registration','passed' if matched else 'failed','OK' if matched else 'REGISTRATION_MISMATCH')
            except FileNotFoundError:
                report['registration']='missing';record('registration','incomplete','REGISTRATION_MISSING')
            except (ValueError,AttributeError,TypeError):
                report['registration']='unsupported';record('registration','incomplete','REGISTRATION_UNSUPPORTED')
        guidance=(home/'AGENTS.override.md' if d['agent']=='codex' else home/'boxweave-user-rules.md' if d['agent']=='cursor' else home/'AGENTS.md' if d['agent']=='zcode' else home/'rules/boxweave.md')
        if d['agent']=='codex':
            safe_file(guidance)
            if not guidance.exists() or not guidance.read_text().strip(): guidance=home/'AGENTS.md'
        safe_file(guidance)
        report['guidance']='present' if guidance.is_file() and BEGIN in guidance.read_text() else 'unverified'
        if d['agent']=='cursor': report['guidance']='manual'
        record('guidance','passed' if report['guidance']=='present' else 'unverified',
               'FILE_PRESENT' if report['guidance']=='present' else 'MANUAL_VERIFICATION_REQUIRED')
        evidence={}
        try:
            # Controlled profile: never execute commands from an agent registration.
            probe(ssh,['-F','/dev/null','-o','IdentityAgent=none',
                       '-o','GlobalKnownHostsFile=/dev/null','-o','UpdateHostKeys=no',*transport],
                  timeout=args.timeout,evidence=evidence)
            report['connection']='passed'
        except DiagnosticError as error:
            report['connection']='failed';record(error.stage,'failed',error.code)
        finally:
            for stage in ('initialize','discovery','provider_call'):
                if evidence.get(stage)=='passed': record(stage,'passed','OK')
            for name in ('protocol_version','process_exit_code','cleanup','tool_pages'):
                if name in evidence: report[name]=evidence[name]
    except DiagnosticError:
        pass
    except (OSError,ValueError,TypeError,AttributeError):
        record('input','incomplete','INPUT_UNAVAILABLE')
    except KeyboardInterrupt:
        record('interruption','incomplete','INTERRUPTED');report['overall']='incomplete'
        print(json.dumps(report));return 130
    states={x['status'] for x in report['stages']}
    report['overall']='failed' if 'failed' in states else 'incomplete' if 'incomplete' in states else 'passed'
    code={'passed':0,'failed':1,'incomplete':2}[report['overall']]
    if args.report:
        try:
            destination=Path(args.report).expanduser();safe_file(destination)
            if destination.exists(): raise ValueError()
            # Explicit export only; never overwrite a report or create parent directories.
            fd,temporary=tempfile.mkstemp(prefix='.boxweave-report-',dir=destination.parent)
            try:
                with os.fdopen(fd,'w') as output:
                    output.write(json.dumps(report,indent=2)+'\n');output.flush();os.fsync(output.fileno())
                os.link(temporary,destination)
            finally: os.unlink(temporary)
        except (OSError,ValueError,DiagnosticError):
            record('report','failed','REPORT_WRITE_FAILED');report['overall']='failed';code=1
    print(json.dumps(report,indent=2))
    return code


def container_public_key_from_env():
    value = os.environ.get('BOXWEAVE_CONTAINER_PUBLIC_KEY', '')
    if not isinstance(value, str) or '\n' in value or '\r' in value or not re.fullmatch(
            r'(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp256) [A-Za-z0-9+/=]+(?: [^\r\n]+)?', value):
        raise ValueError('Invalid or missing BOXWEAVE_CONTAINER_PUBLIC_KEY')
    return value


def setup(args):
    d=connection_from_env()
    host_key=validate_server_host_key(os.environ.get('BOXWEAVE_SERVER_HOST_KEY'))
    container_key=container_public_key_from_env()
    provider=os.environ.get('BOXWEAVE_CLAUDE_PROVIDER') or None
    model=os.environ.get('BOXWEAVE_GLM_MODEL') or None
    if (provider or model) and d['agent']!='claude-code':
        raise ValueError('GLM options require Claude Code')
    if model and not provider: raise ValueError('GLM model requires provider')
    glm_text=None
    if provider:
        directory=Path(os.environ.get('CLAUDE_CONFIG_DIR') or Path.home()/'.claude').expanduser()
        settings=directory/'settings.json'
        glm_text=glm_settings(settings.read_text() if settings.exists() else '', provider,
                              model, os.environ.get('GLM_API_KEY'))
    if d['os']=='macos' and sys.platform!='darwin': raise ValueError('Run this macOS command on the Mac')
    if d['runtime']=='wsl' and 'microsoft' not in Path('/proc/sys/kernel/osrelease').read_text().lower():
        raise ValueError('Run this WSL command inside the intended WSL distribution')
    ssh=shutil.which('ssh'); keygen=shutil.which('ssh-keygen')
    if not ssh or not keygen: raise ValueError('OpenSSH ssh and ssh-keygen are required')
    claude=shutil.which('claude') if d['agent']=='claude-code' else None
    if d['agent']=='claude-code' and not claude: raise ValueError('Install Claude Code in this runtime before setup')
    home=Path.home()
    sshdir=home/'.ssh'
    key=sshdir/'boxweave'
    known=sshdir/'boxweave_known_hosts'
    config=sshdir/'config'
    pub=key.with_suffix('.pub')
    pin=host_token(d['ssh_host'],d['ssh_port'])+' '+' '.join(host_key.split()[:2])+'\n'
    if known.exists():
        observed=known.read_text().strip()
        if observed and observed!=pin.strip():
            if observed.split()[0:1]!=[pin.split()[0]]:
                print('[boxweave] SSH address changed ({} -> {}); repinning the server host key.'.format(
                    observed.split()[0], pin.split()[0]), flush=True)
            else:
                raise ValueError('Pinned server identity changed; verify out-of-band')
    if key.exists()!=pub.exists(): raise ValueError('Incomplete SSH keypair; refusing overwrite')
    # Stage adapters before enrolling so malformed user configuration is not a partial enrollment.
    # The transport is addressed through the managed "Host boxweave" block, so every
    # registration is just: ssh boxweave <launcher>. Changing the server address later
    # only touches ~/.ssh/config, never the agent registrations.
    pending=[(config, ssh_config_managed(config.read_text(encoding='utf-8') if config.exists() else '',
                                         ssh_transport_block(d['ssh_host'],d['ssh_port'],d['linux_user'],key,known)))]
    if d['agent']=='codex':
        directory=Path(args.agent_home or os.environ.get('CODEX_HOME') or home/'.codex').expanduser()
        agent_config=directory/'config.toml'
        pending.append((agent_config,toml_update(agent_config.read_text() if agent_config.exists() else '',ssh,['boxweave',d['launcher']],args.replace)))
        rule=directory/'AGENTS.override.md'
        if not rule.exists() or not rule.read_text().strip(): rule=directory/'AGENTS.md'
        policy=POLICY.format(ssh_host=d['ssh_host'],ssh_port=d['ssh_port'],linux_user=d['linux_user'],
                             container_public_key=container_key)
        pending.append((rule,managed(rule.read_text() if rule.exists() else '',policy)))
    elif d['agent']=='zcode':
        directory=Path(args.agent_home or home/'.zcode').expanduser()
        agent_config=directory/'cli/config.json'
        pending.append((agent_config,zcode_update(agent_config.read_text() if agent_config.exists() else '',ssh,['boxweave',d['launcher']],args.replace)))
        policy=POLICY.format(ssh_host=d['ssh_host'],ssh_port=d['ssh_port'],linux_user=d['linux_user'],
                             container_public_key=container_key)
        pending.append((directory/'AGENTS.md',managed(directory.joinpath('AGENTS.md').read_text() if directory.joinpath('AGENTS.md').exists() else '',policy)))
    elif d['agent']=='cursor':
        directory=Path(args.agent_home or home/'.cursor').expanduser()
        agent_config=directory/'mcp.json'
        pending.append((agent_config,json_update(agent_config.read_text() if agent_config.exists() else '',ssh,[d['launcher']],args.replace)))
        policy=POLICY.format(ssh_host=d['ssh_host'],ssh_port=d['ssh_port'],linux_user=d['linux_user'],
                             container_public_key=container_key)
        pending.append((directory/'boxweave-user-rules.md',policy))
        if args.project_path:
            project=Path(args.project_path).expanduser().resolve(strict=True)
            if not project.is_dir(): raise ValueError('Project path is not a directory')
            pending.append((project/'.cursor/rules/boxweave.mdc','---\nalwaysApply: true\n---\n\n'+policy))
    else:
        # The official CLI owns its user registry, including custom configuration-home semantics.
        existing=subprocess.run([claude,'mcp','get','boxweave'],capture_output=True,timeout=20)
        if existing.returncode==0 and not args.replace:
            raise ValueError('Existing Claude boxweave entry; rerun with --replace after review')
        directory=Path(args.agent_home or os.environ.get('CLAUDE_CONFIG_DIR') or home/'.claude').expanduser()
        if args.agent_home:
            raise ValueError('For Claude Code set CLAUDE_CONFIG_DIR instead of --agent-home')
        policy=POLICY.format(ssh_host=d['ssh_host'],ssh_port=d['ssh_port'],linux_user=d['linux_user'],
                             container_public_key=container_key)
        pending.append((directory/'rules/boxweave.md',policy))
        if glm_text is not None: pending.append((directory/'settings.json',glm_text))
    for path,_ in pending:
        if path.is_symlink() or any(p.is_symlink() for p in path.parents if p!=home.parent):
            raise ValueError('Refusing symlinked configuration path')
    if args.bootstrap_identity and not Path(args.bootstrap_identity).expanduser().is_file():
        raise ValueError('Bootstrap identity not found')
    sshdir.mkdir(mode=0o700,exist_ok=True); sshdir.chmod(0o700)
    if key.is_symlink() or pub.is_symlink() or known.is_symlink(): raise ValueError('Refusing SSH identity symlink')
    if not key.exists():
        checked([keygen,'-q','-t','ed25519','-f',str(key),'-N','','-C','boxweave'],timeout=30)
    key.chmod(0o600)
    checked([keygen,'-lf',str(pub),'-E','sha256'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=10)
    line=pub.read_text().strip()
    if not re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/=]+(?: [^\r\n]+)?',line): raise ValueError('Invalid transport public key')
    atomic_write(known,pin)
    atomic_write(config,pending[0][1])
    enrollment='restrict,command="'+d['launcher']+'" '+line
    encoded=base64.b64encode(enrollment.encode()).decode()
    material=line.split()[1]
    script=(
        'set -eu\numask 077\nmkdir -p "$HOME/.ssh"\nchmod 700 "$HOME/.ssh"\n'
        'test ! -L "$HOME/.ssh/authorized_keys"\ntouch "$HOME/.ssh/authorized_keys"\n'
        'chmod 600 "$HOME/.ssh/authorized_keys"\n'
        'tmp=$(mktemp "$HOME/.ssh/.boxweave-XXXXXX")\ntrap \'rm -f "$tmp"\' EXIT\n'
        'awk -v key='+shlex.quote(material)+' \'{ keep=1; for(i=1;i<=NF;i++) if($i==key) keep=0; if(keep) print $0 }\' "$HOME/.ssh/authorized_keys" > "$tmp"\n'
        "printf '%s' '"+encoded+"' | base64 -d >> \"$tmp\"\nprintf '\n' >> \"$tmp\"\n"
        'chmod 600 "$tmp"\nmv -f "$tmp" "$HOME/.ssh/authorized_keys"\n')
    common=['-T','-o','StrictHostKeyChecking=yes','-o','UserKnownHostsFile='+str(known),
            '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15','-p',str(d['ssh_port'])]
    if args.bootstrap_identity:
        auth=['-o','IdentitiesOnly=yes','-o','IdentityAgent=none','-i',str(Path(args.bootstrap_identity).expanduser())]
    else:
        auth=['-o','PubkeyAuthentication=no','-o','PreferredAuthentications=keyboard-interactive,password']
    print('[boxweave] Enrolling public key using your EXISTING SSH login.',flush=True)
    remote="printf '%s' "+shlex.quote(base64.b64encode(script.encode()).decode())+" | base64 -d | sh"
    checked([ssh,*common,*auth,d['linux_user']+'@'+d['ssh_host'],remote],timeout=300)
    print('[boxweave] Enrollment complete; verifying authenticated MCP.',flush=True)
    # The same options as the managed Host block, but explicit: this probe must
    # not depend on the user's ~/.ssh/config being parsed as we wrote it.
    verify=['-F','/dev/null','-o','IdentityAgent=none','-T','-i',str(key),
            '-o','IdentitiesOnly=yes','-o','BatchMode=yes',
            '-o','StrictHostKeyChecking=yes','-o','UserKnownHostsFile='+str(known),
            '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15',
            *SSH_KEEPALIVE_ARGS,
            '-p',str(d['ssh_port']),d['linux_user']+'@'+d['ssh_host'],d['launcher']]
    probe(ssh,verify)
    if claude:
        if args.replace and existing.returncode==0:
            checked([claude,'mcp','remove','--scope','user','boxweave'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=20)
        checked([claude,'mcp','add','--scope','user','--transport','stdio','boxweave','--',ssh,'boxweave',d['launcher']],
                stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=20)
    for path,text in pending[1:]: atomic_write(path,text)
    print('MCP initialize/tools/list/read-only provider call: passed.')
    print('Agent registration: written (ssh boxweave). Reopen the agent and verify BoxWeave in its MCP UI.')
    if d['agent']=='cursor' and not args.project_path:
        print('Guidance: manual step required. Paste '+str(directory/'boxweave-user-rules.md')+' into Cursor Settings > Rules > User Rules.')
    else: print('Guidance: written; confirm loading in a new agent session.')

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--diagnose',action='store_true',help='Observe existing client connection; never enroll or repair')
    parser.add_argument('--timeout',type=float,default=30,help='Protocol budget in seconds (1-300); cleanup adds at most four seconds')
    parser.add_argument('--report',help='Create a new private diagnostic report file')
    parser.add_argument('--bootstrap-identity',help='Existing authorized SSH identity; never copied')
    parser.add_argument('--agent-home')
    parser.add_argument('--project-path',help='Optional local Cursor workspace for a project rule')
    parser.add_argument('--replace',action='store_true',help='Replace the reviewed boxweave MCP entry only')
    args=parser.parse_args()
    if args.diagnose: return diagnose(args)
    if args.report: parser.error('--report requires --diagnose')
    try: setup(args); return 0
    except (ValueError,OSError,RuntimeError,subprocess.TimeoutExpired) as error:
        print('BoxWeave setup incomplete: '+str(error),file=sys.stderr)
        print('Earlier enrollment/configuration steps may persist; no automatic identity rollback.',file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print('BoxWeave setup cancelled; completed steps may persist.',file=sys.stderr); return 130

if __name__=='__main__':
    raise SystemExit(main())
