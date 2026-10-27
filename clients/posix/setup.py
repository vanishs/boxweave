#!/usr/bin/env python3
"""BoxWeave macOS / WSL CLI enrollment. Requires Python 3.11+, standard library."""
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
# The stdio MCP channel is long-lived: without traffic-based keepalive a silently
# dropped connection (idle NAT/conntrack eviction) hangs every later call. ZCode
# also kills tool calls at its 30s default; create-class calls need more.
SSH_KEEPALIVE_ARGS = ['-o', 'ServerAliveInterval=15', '-o', 'ServerAliveCountMax=4']
ZCODE_TIMEOUT_MS = 180000
CODEX_TOOL_TIMEOUT_SEC = 180

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

def toml_update(original, command, args, replace=False):
    parsed = tomllib.loads(original)
    servers = parsed.get('mcp_servers',{})
    if not isinstance(servers,dict): raise ValueError('Invalid MCP TOML table')
    existing = servers.get('boxweave')
    if isinstance(existing,dict) and not replace and set(existing)-{'command','args','startup_timeout_sec','tool_timeout_sec'}:
        raise ValueError('Existing BoxWeave MCP entry has extra settings; use --replace after review')
    expected = {'command':command,'args':args}
    if existing is not None and not isinstance(existing,dict): raise ValueError('Invalid BoxWeave MCP entry')
    if existing is not None and not replace and any(existing.get(k)!=v for k,v in expected.items()):
        raise ValueError('Existing boxweave MCP entry differs; rerun with --replace after review')
    header = '[mcp_servers.boxweave]'
    pattern = r'(?ms)^\[mcp_servers\.boxweave\][ \t]*(?:#[^\n]*)?\n.*?(?=^\[|\Z)'
    if existing is not None and not re.search(pattern,original):
        raise ValueError('Unsupported existing BoxWeave TOML spelling; edit it explicitly')
    if re.search(r'(?m)^\[mcp_servers\.boxweave\.',original):
        raise ValueError('Nested BoxWeave MCP settings require explicit migration')
    block = header+'\ncommand = '+json.dumps(command,ensure_ascii=False)+'\nargs = '+json.dumps(args,ensure_ascii=False)+'\nstartup_timeout_sec = 30\ntool_timeout_sec = '+str(CODEX_TOOL_TIMEOUT_SEC)+'\n'
    updated = re.sub(pattern,lambda _:block,original,count=1) if existing is not None else original.rstrip()+'\n\n'+block
    check = tomllib.loads(updated)
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
                                    'clientInfo':{'name':'boxweave-client','version':'0.3.0'}})
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
            if block['type']=='resource_link' and not isinstance(block.get('uri'),str): invalid()
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


def diagnostic_profile(data):
    if not isinstance(data,dict): raise ValueError()
    for key,pattern in {'ssh_host':r'[A-Za-z0-9][A-Za-z0-9.:-]{0,252}',
                        'linux_user':r'[a-z_][a-z0-9_-]{0,63}',
                        'launcher':r'/[A-Za-z0-9_./-]+'}.items():
        if not isinstance(data.get(key),str) or not re.fullmatch(pattern,data[key]): raise ValueError()
    if type(data.get('ssh_port')) is not int or not 1<=data['ssh_port']<=65535: raise ValueError()
    if data.get('agent') not in ('codex','cursor','claude-code','zcode'): raise ValueError()
    if (data.get('os'),data.get('runtime'),data.get('transport')) not in (
        ('macos','native','native'),('windows','wsl','wsl')): raise ValueError()
    if data['agent'] in ('cursor','zcode') and data['runtime']=='wsl': raise ValueError()
    key=data.get('server_public_key','')
    if not isinstance(key,str) or not re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/=]+(?: [^\r\n]+)?',key): raise ValueError()
    blob=base64.b64decode(key.split()[1],validate=True)
    if len(blob)!=51 or blob[:15]!=b'\x00\x00\x00\x0bssh-ed25519': raise ValueError()
    return data


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
            if len(args.data_base64)>65536: raise ValueError()
            d=diagnostic_profile(json.loads(base64.b64decode(args.data_base64,validate=True)))
            options=d.get('setup_options',{})
            if not isinstance(options,dict): raise ValueError()
            if any(options.get(k) for k in ('replace','bootstrap_identity','project_path')): raise ValueError()
            if not 1<=args.timeout<=300: raise ValueError()
        except (ValueError,TypeError,KeyError): fail('INPUT_INVALID','input')
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
        for path in (key,known): safe_file(path)
        if not key.is_file(): fail('IDENTITY_MISSING','trust')
        if not known.is_file(): fail('PIN_MISSING','trust')
        host=d['ssh_host']
        token='['+host+']:'+str(d['ssh_port']) if d['ssh_port']!=22 or ':' in host else host
        expected=token+' '+' '.join(d['server_public_key'].split()[:2])
        if known.read_text().strip()!=expected: fail('PIN_MISMATCH','trust','failed')
        record('trust','passed','OK')
        transport=['-T','-i',str(key),'-o','IdentitiesOnly=yes','-o','BatchMode=yes',
                   '-o','StrictHostKeyChecking=yes','-o','UserKnownHostsFile='+str(known),
                   '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15',
                   *SSH_KEEPALIVE_ARGS,
                   '-p',str(d['ssh_port']),d['linux_user']+'@'+host,d['launcher']]
        home=Path(args.agent_home or options.get('agent_home') or
                  os.environ.get('CODEX_HOME') or Path.home()/'.codex').expanduser()
        if d['agent']=='cursor': home=Path(args.agent_home or options.get('agent_home') or Path.home()/'.cursor').expanduser()
        if d['agent']=='zcode': home=Path(args.agent_home or options.get('agent_home') or Path.home()/'.zcode').expanduser()
        if d['agent']=='claude-code':
            report['registration']='unsupported'
            record('registration','incomplete','REGISTRATION_UNSUPPORTED')
            home=Path(os.environ.get('CLAUDE_CONFIG_DIR') or Path.home()/'.claude')
        else:
            config=home/('config.toml' if d['agent']=='codex' else 'cli/config.json' if d['agent']=='zcode' else 'mcp.json');safe_file(config)
            try:
                parsed=(tomllib.loads(config.read_text()) if d['agent']=='codex' else json.loads(config.read_text()))
                entry=(parsed.get('mcp',{}).get('servers',{}).get('boxweave') if d['agent']=='zcode'
                       else parsed.get('mcp_servers' if d['agent']=='codex' else 'mcpServers',{}).get('boxweave'))
                allowed={'command','args','type','enable','enabled','timeoutMs'} if d['agent']=='zcode' else {'command','args','startup_timeout_sec','tool_timeout_sec','enabled'}
                if not isinstance(entry,dict) or set(entry)-allowed: raise ValueError()
                matched=entry.get('command')==ssh and entry.get('args')==transport and entry.get('enabled',True) is True and entry.get('enable',True) is True
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
            probe(ssh,['-F','/dev/null',*transport[:1],'-o','IdentityAgent=none',
                       '-o','GlobalKnownHostsFile=/dev/null','-o','UpdateHostKeys=no',*transport[1:]],
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


def validate_data(d):
    if not isinstance(d,dict): raise ValueError('Invalid client data')
    for k,pattern in {'ssh_host':r'[A-Za-z0-9][A-Za-z0-9.:-]{0,252}',
                      'linux_user':r'[a-z_][a-z0-9_-]{0,63}',
                      'launcher':r'/[A-Za-z0-9_./-]+'}.items():
        if not isinstance(d.get(k),str) or not re.fullmatch(pattern,d[k]): raise ValueError('Invalid '+k)
    if type(d.get('ssh_port')) is not int or not 1<=d['ssh_port']<=65535: raise ValueError('Invalid port')
    if d.get('agent') not in ('codex','cursor','claude-code','zcode'): raise ValueError('Unsupported agent')
    if d.get('os')=='macos':
        if d.get('runtime')!='native' or d.get('transport')!='native': raise ValueError('Invalid macOS profile')
    elif d.get('os')=='windows':
        if d.get('runtime')!='wsl' or d.get('transport')!='wsl' or d['agent'] in ('cursor','zcode'): raise ValueError('Invalid WSL CLI profile')
    else: raise ValueError('Unsupported OS')
    if not isinstance(d.get('policy'),str) or len(d['policy'])>20000: raise ValueError('Invalid guidance')
    for key in ('server_public_key','container_public_key'):
        value=d.get(key,'')
        if not isinstance(value,str) or '\n' in value or '\r' in value or not re.fullmatch(r'(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp256) [A-Za-z0-9+/=]+(?: [^\r\n]+)?',value):
            raise ValueError('Invalid '+key)
    if not d['server_public_key'].startswith('ssh-ed25519 '): raise ValueError('Invalid host key')
    return d

def setup(args):
    d=validate_data(json.loads(base64.b64decode(args.data_base64,validate=True)))
    options=d.get('setup_options',{})
    if not isinstance(options,dict): raise ValueError('Invalid setup options')
    for name in ('bootstrap_identity','agent_home','project_path','replace'):
        if not getattr(args,name,None) and options.get(name) is not None:
            if name!='replace' and not isinstance(options[name],str): raise ValueError('Invalid setup option')
            if name=='replace' and type(options[name]) is not bool: raise ValueError('Invalid replacement flag')
            setattr(args,name,options[name])
    provider=options.get('claude_provider')
    if (provider or options.get('glm_model')) and d['agent']!='claude-code':
        raise ValueError('GLM options require Claude Code')
    if options.get('glm_model') and not provider: raise ValueError('GLM model requires provider')
    glm_text=None
    if provider:
        directory=Path(os.environ.get('CLAUDE_CONFIG_DIR') or Path.home()/'.claude').expanduser()
        settings=directory/'settings.json'
        glm_text=glm_settings(settings.read_text() if settings.exists() else '', provider,
                              options.get('glm_model'), os.environ.get('GLM_API_KEY'))
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
    pub=key.with_suffix('.pub')
    host=d['ssh_host']
    token='['+host+']:'+str(d['ssh_port']) if d['ssh_port']!=22 or ':' in host else host
    pin=token+' '+' '.join(d['server_public_key'].split()[:2])+'\n'
    if known.exists() and known.read_text().strip()!=pin.strip(): raise ValueError('Pinned server identity changed; verify out-of-band')
    if key.exists()!=pub.exists(): raise ValueError('Incomplete SSH keypair; refusing overwrite')
    transport=['-T','-i',str(key),'-o','IdentitiesOnly=yes','-o','BatchMode=yes',
               '-o','StrictHostKeyChecking=yes','-o','UserKnownHostsFile='+str(known),
               '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15',
               *SSH_KEEPALIVE_ARGS,
               '-p',str(d['ssh_port']),d['linux_user']+'@'+host,d['launcher']]
    # Stage adapters before enrolling so malformed user configuration is not a partial enrollment.
    pending=[]
    if d['agent']=='codex':
        directory=Path(args.agent_home or os.environ.get('CODEX_HOME') or home/'.codex').expanduser()
        config=directory/'config.toml'
        pending.append((config,toml_update(config.read_text() if config.exists() else '',ssh,transport,args.replace)))
        rule=directory/'AGENTS.override.md'
        if not rule.exists() or not rule.read_text().strip(): rule=directory/'AGENTS.md'
        pending.append((rule,managed(rule.read_text() if rule.exists() else '',d['policy'])))
    elif d['agent']=='zcode':
        directory=Path(args.agent_home or home/'.zcode').expanduser()
        config=directory/'cli/config.json'
        pending.append((config,zcode_update(config.read_text() if config.exists() else '',ssh,transport,args.replace)))
        rule=directory/'AGENTS.md'
        pending.append((rule,managed(rule.read_text() if rule.exists() else '',d['policy'])))
    elif d['agent']=='cursor':
        directory=Path(args.agent_home or home/'.cursor').expanduser()
        config=directory/'mcp.json'
        pending.append((config,json_update(config.read_text() if config.exists() else '',ssh,transport,args.replace)))
        pending.append((directory/'boxweave-user-rules.md',d['policy']))
        if args.project_path:
            project=Path(args.project_path).expanduser().resolve(strict=True)
            if not project.is_dir(): raise ValueError('Project path is not a directory')
            pending.append((project/'.cursor/rules/boxweave.mdc','---\nalwaysApply: true\n---\n\n'+d['policy']))
    else:
        # The official CLI owns its user registry, including custom configuration-home semantics.
        existing=subprocess.run([claude,'mcp','get','boxweave'],capture_output=True,timeout=20)
        if existing.returncode==0 and not args.replace:
            raise ValueError('Existing Claude boxweave entry; rerun with --replace after review')
        directory=Path(args.agent_home or os.environ.get('CLAUDE_CONFIG_DIR') or home/'.claude').expanduser()
        if args.agent_home:
            raise ValueError('For Claude Code set CLAUDE_CONFIG_DIR instead of --agent-home')
        pending.append((directory/'rules/boxweave.md',d['policy']))
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
    checked([ssh,*common,*auth,d['linux_user']+'@'+host,remote],timeout=300)
    print('[boxweave] Enrollment complete; verifying authenticated MCP.',flush=True)
    probe(ssh,transport)
    if claude:
        if args.replace and existing.returncode==0:
            checked([claude,'mcp','remove','--scope','user','boxweave'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=20)
        checked([claude,'mcp','add','--scope','user','--transport','stdio','boxweave','--',ssh,*transport],
                stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=20)
    for path,text in pending: atomic_write(path,text)
    print('MCP initialize/tools/list/read-only provider call: passed.')
    print('Agent registration: written. Reopen the agent and verify BoxWeave in its MCP UI.')
    if d['agent']=='cursor' and not args.project_path:
        print('Guidance: manual step required. Paste '+str(directory/'boxweave-user-rules.md')+' into Cursor Settings > Rules > User Rules.')
    else: print('Guidance: written; confirm loading in a new agent session.')

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--data-base64',required=True,help='Public connection descriptor emitted by boxweave client')
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
