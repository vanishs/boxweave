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
    if isinstance(existing,dict) and not replace and set(existing)-{'command','args','startup_timeout_sec'}:
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
    block = header+'\ncommand = '+json.dumps(command,ensure_ascii=False)+'\nargs = '+json.dumps(args,ensure_ascii=False)+'\nstartup_timeout_sec = 30\n'
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

def probe(command, args, timeout=30):
    """Bounded thread-based pipes; no raw provider output or SSH diagnostics."""
    p = subprocess.Popen([command,*args],stdin=subprocess.PIPE,stdout=subprocess.PIPE,
                         stderr=subprocess.DEVNULL,start_new_session=True)
    inbox = queue.Queue(maxsize=64)
    def reader():
        try:
            while True:
                line = p.stdout.readline(2*1024*1024+1)
                if not line: break
                if len(line)>2*1024*1024: raise ValueError()
                inbox.put_nowait(line)
        except (ValueError,queue.Full): pass
        finally:
            try: inbox.put_nowait(None)
            except queue.Full: pass
    threading.Thread(target=reader,daemon=True).start()
    deadline = time.monotonic()+timeout
    def send(value):
        p.stdin.write((json.dumps(value)+'\n').encode()); p.stdin.flush()
    def call(ident,method,params):
        send({'jsonrpc':'2.0','id':ident,'method':method,'params':params})
        for _ in range(256):
            remaining=deadline-time.monotonic()
            if remaining<=0: raise TimeoutError('MCP handshake/tool discovery timeout')
            try: raw=inbox.get(timeout=remaining)
            except queue.Empty: raise TimeoutError('MCP handshake/tool discovery timeout') from None
            if raw is None: raise RuntimeError('MCP stream closed; process exit '+str(p.poll()))
            value=json.loads(raw)
            if value.get('id')==ident and 'method' not in value:
                if 'error' in value: raise RuntimeError('MCP returned a protocol error')
                return value['result']
            if 'method' in value and 'id' in value:
                send({'jsonrpc':'2.0','id':value['id'],'error':{'code':-32601,'message':'Unsupported client request'}})
        raise RuntimeError('MCP message budget exceeded')
    try:
        result=call(1,'initialize',{'protocolVersion':'2024-11-05','capabilities':{},
                                 'clientInfo':{'name':'boxweave-client','version':'0.2.0'}})
        if result.get('protocolVersion')!='2024-11-05': raise RuntimeError('Unsupported negotiated MCP protocol')
        send({'jsonrpc':'2.0','method':'notifications/initialized'})
        tools=call(2,'tools/list',{}).get('tools',[])
        names=[t.get('name','') for t in tools]
        target=next((n for n in names if n=='list_containers'),None)
        if not target: raise RuntimeError('Provider list_containers tool unavailable')
        result=call(3,'tools/call',{'name':target,'arguments':{}})
        if result.get('isError'): raise RuntimeError('Read-only provider verification failed')
        return True
    finally:
        p.stdin.close()
        try: os.killpg(p.pid,signal.SIGTERM)
        except ProcessLookupError: pass
        try: p.wait(timeout=2)
        except subprocess.TimeoutExpired:
            os.killpg(p.pid,signal.SIGKILL); p.wait(timeout=2)
        p.stdout.close()

def validate_data(d):
    if not isinstance(d,dict): raise ValueError('Invalid client data')
    for k,pattern in {'ssh_host':r'[A-Za-z0-9][A-Za-z0-9.:-]{0,252}',
                      'linux_user':r'[a-z_][a-z0-9_-]{0,63}',
                      'launcher':r'/[A-Za-z0-9_./-]+'}.items():
        if not isinstance(d.get(k),str) or not re.fullmatch(pattern,d[k]): raise ValueError('Invalid '+k)
    if type(d.get('ssh_port')) is not int or not 1<=d['ssh_port']<=65535: raise ValueError('Invalid port')
    if d.get('agent') not in ('codex','cursor','claude-code'): raise ValueError('Unsupported agent')
    if d.get('os')=='macos':
        if d.get('runtime')!='native' or d.get('transport')!='native': raise ValueError('Invalid macOS profile')
    elif d.get('os')=='windows':
        if d.get('runtime')!='wsl' or d.get('transport')!='wsl' or d['agent']=='cursor': raise ValueError('Invalid WSL CLI profile')
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
        "printf '%s' '"+encoded+"' | base64 -d >> \"$tmp\"\nprintf '\\n' >> \"$tmp\"\n"
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
    parser.add_argument('--bootstrap-identity',help='Existing authorized SSH identity; never copied')
    parser.add_argument('--agent-home')
    parser.add_argument('--project-path',help='Optional local Cursor workspace for a project rule')
    parser.add_argument('--replace',action='store_true',help='Replace the reviewed boxweave MCP entry only')
    args=parser.parse_args()
    try: setup(args); return 0
    except (ValueError,OSError,RuntimeError,subprocess.TimeoutExpired) as error:
        print('BoxWeave setup incomplete: '+str(error),file=sys.stderr)
        print('Earlier enrollment/configuration steps may persist; no automatic identity rollback.',file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print('BoxWeave setup cancelled; completed steps may persist.',file=sys.stderr); return 130

if __name__=='__main__':
    raise SystemExit(main())
