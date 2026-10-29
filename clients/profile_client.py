"""Operator-trusted Windows profile setup. Python 3.11+, stdlib only; no model requests."""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import queue
import re
import subprocess
import threading
import time
import tomllib
import uuid

PROFILES={a+'/windows/'+t for a in ('codex','zcode') for t in ('native','wsl')}
TRANSPORT={'command','args','url','env','env_vars','cwd','experimental_environment','bearer_token_env_var','http_headers','env_http_headers','http_headers_helper','oauth'}

class Refused(Exception):pass

def fail(code):raise Refused(code)

def json_object(pairs):
    result={}
    for key,value in pairs:
        if key in result:fail('DUPLICATE_JSON_KEY')
        result[key]=value
    return result

def checked_path(value, exists=True):
    p=Path(value)
    if not p.is_absolute():fail('ABSOLUTE_PATH_REQUIRED')
    for part in (p,*p.parents):
        if part.exists() and (part.is_symlink() or bool(getattr(part.stat(),'st_file_attributes',0)&1024)):
            fail('REPARSE_PATH_REFUSED')
    if exists and not p.is_file():fail('FILE_REQUIRED')
    return p

def private_windows(path):
    # Inspect ACL metadata only, never read the identity or authentication file.
    code="$env:PSModulePath=$PSHOME+'\\Modules';$p='"+str(path).replace("'","''")+"';$a=Get-Acl -LiteralPath $p;$sid=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value;if($a.Owner -ne [Security.Principal.WindowsIdentity]::GetCurrent().Name){exit 2};foreach($r in $a.Access){if($r.AccessControlType -eq 'Allow' -and $r.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value -notin @($sid,'S-1-5-18')){exit 3}};Write-Output verified"
    if program(['powershell.exe','-NoProfile','-NonInteractive','-EncodedCommand',base64.b64encode(code.encode('utf-16le')).decode()])!='verified':fail('PRIVATE_ACL_REQUIRED')

def program(argv, env=None, timeout=30):
    process=subprocess.Popen(argv,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,env=env)
    done=queue.Queue(maxsize=1);chunks=[];total=[0]
    def read():
        try:
            while True:
                b=process.stdout.read(65536)
                if not b:break
                total[0]+=len(b)
                if total[0]>4194304:
                    process.kill();break
                chunks.append(b)
            done.put(True)
        except (OSError,ValueError):done.put(False)
    reader=threading.Thread(target=read,daemon=True);reader.start()
    try:
        try:ok=done.get(timeout=timeout)
        except queue.Empty:fail('PROCESS_DEADLINE')
        process.wait(timeout=3)
        if total[0]>4194304:fail('OUTPUT_LIMIT')
        if not ok or process.returncode!=0:fail('PROCESS_FAILED')
        return b''.join(chunks).decode('utf-8',errors='strict').strip()
    finally:
        if process.poll() is None:process.kill();process.wait(timeout=3)
        process.stdout.close();reader.join(timeout=2)

class RPC:
    """One deadline, bounded lines/total/messages; drain diagnostics without exposing them."""
    def __init__(self,argv,env=None,timeout=90,zcode=False):
        self.zcode=zcode
        self.deadline=time.monotonic()+timeout;self.id=0;self.total=0;self.q=queue.Queue(maxsize=256)
        self.p=subprocess.Popen(argv,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,env=env)
        self.stopped=threading.Event()
        def read():
            try:
                while not self.stopped.is_set():
                    line=self.p.stdout.readline(2097153)
                    self.q.put(line,timeout=1)
                    if not line or len(line)>2097152:break
            except (ValueError,OSError,queue.Full):pass
        self.thread=threading.Thread(target=read,daemon=True);self.thread.start()
    def send(self,value):
        if self.zcode:value={k:v for k,v in value.items() if k!='jsonrpc'}
        data=(json.dumps(value,ensure_ascii=True)+'\n').encode()
        # All requests here are small fixed read-only protocol messages.
        if len(data)>65536:fail('REQUEST_LIMIT')
        done=queue.Queue(maxsize=1)
        def write():
            try:self.p.stdin.write(data);self.p.stdin.flush();done.put(True)
            except (BrokenPipeError,OSError,ValueError):done.put(False)
        threading.Thread(target=write,daemon=True).start()
        left=self.deadline-time.monotonic()
        if left<=0:fail('PROTOCOL_DEADLINE')
        try:ok=done.get(timeout=left)
        except queue.Empty:fail('PROTOCOL_DEADLINE')
        if not ok:fail('STREAM_CLOSED')
    def call(self,method,params):
        self.id+=1;self.send({'jsonrpc':'2.0','id':self.id,'method':method,'params':params})
        for _ in range(256):
            left=self.deadline-time.monotonic()
            if left<=0:fail('PROTOCOL_DEADLINE')
            try:line=self.q.get(timeout=left)
            except queue.Empty:fail('PROTOCOL_DEADLINE')
            self.total+=len(line)
            if not line:fail('STREAM_CLOSED')
            if len(line)>2097152 or self.total>8388608:fail('OUTPUT_LIMIT')
            try:r=json.loads(line)
            except (ValueError,UnicodeError):fail('PROTOCOL_INVALID')
            if not isinstance(r,dict) or (not self.zcode and r.get('jsonrpc')!='2.0'):fail('PROTOCOL_INVALID')
            if 'method' in r:
                if 'id' in r:self.send({'jsonrpc':'2.0','id':r['id'],'error':{'code':-32601,'message':'Unsupported client request'}})
                continue
            if type(r.get('id')) is not int or r['id']!=self.id:fail('PROTOCOL_INVALID')
            if 'error' in r:fail('REMOTE_REJECTED')
            if not isinstance(r.get('result'),dict):fail('PROTOCOL_INVALID')
            return r['result']
        fail('MESSAGE_LIMIT')
    def __enter__(self):return self
    def __exit__(self,*args):
        self.stopped.set()
        if self.p.poll() is None:
            if os.name=='nt':
                subprocess.run(['taskkill.exe','/PID',str(self.p.pid),'/T','/F'],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=10)
            else:
                self.p.terminate()
            try:self.p.wait(timeout=3)
            except subprocess.TimeoutExpired:self.p.kill();self.p.wait(timeout=3)
        self.p.stdin.close();self.p.stdout.close();self.thread.join(timeout=2)
        # Local process termination is not proof of remote command cancellation.

def descriptor(path):
    p=checked_path(path)
    if p.stat().st_size>32768:fail('DESCRIPTOR_LIMIT')
    d=json.loads(p.read_text(encoding='utf-8-sig'),object_pairs_hook=json_object)
    if set(d)!=({'schema','profile','source_sha256','agent_argv','agent_version','profile_directory','connection'}|({'zcode_builtin'} if str(d.get('profile','')).startswith('zcode/') else set())) or d['schema']!=2 or d['profile'] not in PROFILES:
        fail('DESCRIPTOR_INVALID')
    if d['source_sha256']!=hashlib.sha256(Path(__file__).read_bytes()).hexdigest():fail('PAYLOAD_MISMATCH')
    if not isinstance(d['agent_argv'],list) or not d['agent_argv'] or any(not isinstance(a,str) or '\n' in a or '\r' in a for a in d['agent_argv']):fail('AGENT_INVALID')
    checked_path(d['agent_argv'][0]);checked_path(d['profile_directory'],False)
    if d['profile'].startswith('zcode/'):
        b=d['zcode_builtin']
        if set(b)!={'path','sha256'} or hashlib.sha256(checked_path(b['path']).read_bytes()).hexdigest()!=b['sha256']:fail('BUILTIN_CONFIG_CHANGED')
    return d

def ssh_argv(d,preflight=True):
    c=d['connection'];wsl=d['profile'].endswith('/wsl')
    required={'host','port','user','identity_file','known_hosts','server_key','executable'}|({'distribution'} if wsl else set())
    if set(c)!=required or type(c['port']) is not int or not 1<=c['port']<=65535:fail('CONNECTION_INVALID')
    if not re.fullmatch(r'[a-zA-Z0-9.-]{1,253}',c['host']) or not re.fullmatch(r'[a-z_][a-z0-9_-]{0,62}',c['user']):fail('CONNECTION_INVALID')
    if not re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/]+={0,2}',c['server_key']):fail('PUBLIC_PIN_REQUIRED')
    checked_path(c['executable'])
    token=c['host'] if c['port']==22 else '['+c['host']+']:'+str(c['port'])
    expected=token+' '+c['server_key']
    if wsl:
        if not isinstance(c['distribution'],str) or not re.fullmatch(r'[A-Za-z0-9_.-]{1,80}',c['distribution']):fail('EXPLICIT_DISTRO_REQUIRED')
        for k in ('identity_file','known_hosts'):
            if not PurePosixPath(c[k]).is_absolute() or any(x in c[k] for x in ('\n','\r','\\')):fail('POSIX_PATH_REQUIRED')
        if preflight:
            # Constants plus separately marshalled JSON; no login shell or distribution fallback.
            code="import os,sys,json,stat;from pathlib import Path;c=json.loads(sys.argv[1]);p=Path(c['identity_file']);h=Path(c['known_hosts']);assert all(not x.is_symlink() for x in (p,h,*p.parents,*h.parents));assert p.stat().st_uid==os.getuid() and stat.S_IMODE(p.stat().st_mode)&0o077==0;assert all(x.stat().st_uid==os.getuid() and stat.S_IMODE(x.stat().st_mode)&0o022==0 for x in (p.parent,h.parent));assert h.read_text().strip()==sys.argv[2];print('verified')"
            if program([c['executable'],'--distribution',c['distribution'],'--exec','/usr/bin/python3','-c',code,json.dumps(c),expected])!='verified':fail('WSL_IDENTITY_OR_PIN_INVALID')
        prefix=[c['executable'],'--distribution',c['distribution'],'--exec','/usr/bin/ssh'];null='/dev/null'
    else:
        for k in ('identity_file','known_hosts'):checked_path(c[k])
        if preflight and os.name=='nt':private_windows(c['identity_file'])
        if preflight and Path(c['known_hosts']).read_text(encoding='utf-8-sig').strip()!=expected:fail('PIN_FILE_MISMATCH')
        prefix=[c['executable']];null='NUL'
    return prefix+['-F',null,'-o','BatchMode=yes','-o','StrictHostKeyChecking=yes','-o','IdentitiesOnly=yes','-o','ConnectTimeout=15','-o','UserKnownHostsFile='+c['known_hosts'],'-i',c['identity_file'],'-p',str(c['port']),c['user']+'@'+c['host'],'/usr/local/bin/boxweave-mcp']

def probe(argv):
    with RPC(argv) as m:
        r=m.call('initialize',{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'boxweave-profile-diagnostic','version':'0.6.0'}})
        if r.get('protocolVersion')!='2024-11-05' or not isinstance(r.get('capabilities',{}).get('tools'),dict):fail('PROTOCOL_UNSUPPORTED')
        m.send({'jsonrpc':'2.0','method':'notifications/initialized'})
        names=set();cursor=None;seen=set()
        for _ in range(16):
            r=m.call('tools/list',{} if cursor is None else {'cursor':cursor})
            if not isinstance(r.get('tools'),list):fail('DISCOVERY_INVALID')
            for t in r['tools']:
                if not isinstance(t,dict) or not isinstance(t.get('name'),str) or t['name'] in names:fail('DISCOVERY_INVALID')
                names.add(t['name'])
            cursor=r.get('nextCursor')
            if cursor is None:break
            if not isinstance(cursor,str) or not cursor or cursor in seen:fail('CURSOR_INVALID')
            seen.add(cursor)
        else:fail('DISCOVERY_LIMIT')
        if 'list_containers' not in names:fail('READ_TOOL_MISSING')
        r=m.call('tools/call',{'name':'list_containers','arguments':{}})
        if r.get('isError') or not isinstance(r.get('content'),list) or not r['content'] or any(not isinstance(b,dict) or b.get('type')!='text' or not isinstance(b.get('text'),str) for b in r['content']) or not any(b['text'].strip() for b in r['content']):fail('READ_FAILED')
        return {'protocol':'2024-11-05','tool_count':len(names),'actual_read':'list_containers','result':'passed','remote_cleanup':'unverified'}

def codex_merge(original,argv,replace=False):
    parsed=tomllib.loads(original);servers=parsed.get('mcp_servers',{});old=servers.get('boxweave')
    desired={'command':argv[0],'args':argv[1:]}
    if old is not None and {k:v for k,v in old.items() if k in TRANSPORT}==desired:return original
    if old is not None and not replace:fail('REGISTRATION_CONFLICT')
    lines=original.splitlines();out=[];in_main=False;skip=False;found=False
    for line in lines:
        if re.match(r'\s*\[',line):
            if 'boxweave' in line and ('"' in line or "'" in line):fail('MANUAL_TOML_MIGRATION_REQUIRED')
            in_main=bool(re.fullmatch(r'\s*\[mcp_servers\.boxweave\]\s*(?:#.*)?',line))
            skip=bool(re.match(r'\s*\[mcp_servers\.boxweave\.(env|http_headers|env_http_headers|oauth)\]',line))
            if skip:continue
            out.append(line)
            if in_main:
                found=True;out.extend(''+k+' = '+json.dumps(v,ensure_ascii=False) for k,v in desired.items())
            continue
        if skip:continue
        if in_main:
            match=re.match(r'\s*([A-Za-z_]+)\s*=\s*(.*)',line)
            if match and match[1] in TRANSPORT:
                try:tomllib.loads(line)
                except tomllib.TOMLDecodeError:fail('MANUAL_TOML_MIGRATION_REQUIRED')
                continue
        out.append(line)
    if old is not None and not found:fail('MANUAL_TOML_MIGRATION_REQUIRED')
    if not found:out.extend(['','[mcp_servers.boxweave]']+[k+' = '+json.dumps(v,ensure_ascii=False) for k,v in desired.items()])
    candidate='\n'.join(out).rstrip()+'\n';after=tomllib.loads(candidate)
    before_other={k:v for k,v in parsed.items() if k!='mcp_servers'}
    if before_other!={k:v for k,v in after.items() if k!='mcp_servers'}:fail('PRESERVATION_FAILED')
    if {k:v for k,v in servers.items() if k!='boxweave'}!={k:v for k,v in after.get('mcp_servers',{}).items() if k!='boxweave'}:fail('PRESERVATION_FAILED')
    if old and {k:v for k,v in old.items() if k not in TRANSPORT}!={k:v for k,v in after['mcp_servers']['boxweave'].items() if k not in TRANSPORT}:fail('PRESERVATION_FAILED')
    return candidate

def zcode_merge(original,argv,replace=False):
    value=json.loads(original or '{}',object_pairs_hook=json_object)
    if not isinstance(value,dict):fail('JSON_CONFIG_UNSUPPORTED')
    m=value.setdefault('mcp',{});servers=m.setdefault('servers',{})
    if not isinstance(servers,dict):fail('JSON_CONFIG_UNSUPPORTED')
    desired={'command':argv[0],'args':argv[1:]};old=servers.get('boxweave')
    if old==desired:return original
    if old is not None and not replace:fail('REGISTRATION_CONFLICT')
    servers['boxweave']=desired
    return json.dumps(value,ensure_ascii=False,indent=2)+'\n'

def agent_env(d):
    env={k:v for k,v in os.environ.items() if k.upper() in {'PATH','SYSTEMROOT','WINDIR','COMSPEC','TEMP','TMP','PATHEXT','SYSTEMDRIVE','PROGRAMDATA','ALLUSERSPROFILE'}}
    p=d['profile_directory']
    if d['profile'].startswith('codex/'):
        env['CODEX_HOME']=p
    else:
        for k in ('ZCODE_HOME','ZCODE_DATA_BASE_DIR','HOME','USERPROFILE','APPDATA','LOCALAPPDATA'):env[k]=p
        env['DO_NOT_TRACK']='1'
        env['ZCODE_BUILTIN_PROVIDER_CONFIG_FILE']=d['zcode_builtin']['path']
        env['ZCODE_PERSONAL_PROVIDER_CONFIG_FILE']=str(Path(p)/'.zcode'/'v2'/'provider_config.json')
    return env

def config_path(d):
    p=Path(d['profile_directory'])
    return p/'config.toml' if d['profile'].startswith('codex/') else p/'.zcode'/'config.json'

def registration(d,argv):
    env=agent_env(d)
    version=program(d['agent_argv']+['--version'],env)
    if version!=d['agent_version']:fail('AGENT_VERSION_CHANGED')
    if d['profile'].startswith('codex/'):
        rows=json.loads(program(d['agent_argv']+['mcp','list','--json'],env))
        rows=[r for r in rows if r.get('name')=='boxweave']
        if len(rows)!=1:fail('REGISTRATION_MISSING')
        t=rows[0].get('transport',{})
        if t.get('command')!=argv[0] or t.get('args')!=argv[1:]:fail('REGISTRATION_MISMATCH')
        if rows[0].get('enabled') is not True:fail('REGISTRATION_DISABLED')
        return {'result':'verified_cli','agent_ui':'unverified'}
    config=json.loads(config_path(d).read_text(encoding='utf-8-sig'),object_pairs_hook=json_object)
    server=config.get('mcp',{}).get('servers',{}).get('boxweave',{})
    if {k:v for k,v in server.items() if k in TRANSPORT}!={'command':argv[0],'args':argv[1:]}:fail('REGISTRATION_MISMATCH')
    if server.get('enabled',True) is not True:fail('REGISTRATION_DISABLED')
    workspace={'workspacePath':d['profile_directory'],'workspaceKey':'boxweave-profile-'+hashlib.sha256(d['profile_directory'].encode()).hexdigest()[:16]}
    with RPC(d['agent_argv']+['app-server'],env,120,zcode=True) as client:
        r=client.call('mcp/list',{'workspace':workspace,'mode':'connect'})
        state=r.get('statuses',{}).get('boxweave',{})
        if state.get('status')!='connected' or not isinstance(state.get('toolCount'),int) or state['toolCount']<=0:fail('AGENT_MCP_NOT_CONNECTED')
        return {'result':'verified_agent_core','agent_ui':'unverified','tool_count':state['toolCount']}

def write_config(d,argv,replace=False):
    p=config_path(d);checked_path(str(p),False)
    if not Path(d['profile_directory']).is_dir():fail('EXPLICIT_PROFILE_DIRECTORY_REQUIRED')
    if os.name=='nt':private_windows(d['profile_directory'])
    p.parent.mkdir(exist_ok=True);lock=p.with_suffix(p.suffix+'.boxweave-lock')
    try:fd=os.open(lock,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    except FileExistsError:fail('CONFIG_LOCKED')
    stage=p.with_name('.boxweave-stage-'+uuid.uuid4().hex);backup=None
    try:
        os.close(fd);existed=p.exists();before=p.read_bytes() if existed else b''
        if len(before)>1048576:fail('CONFIG_LIMIT')
        original=before.decode('utf-8-sig')
        merge=codex_merge if d['profile'].startswith('codex/') else zcode_merge
        candidate=merge(original,argv,replace).encode('utf-8')
        if candidate==before:return {'changed':False,'backup':None}
        with stage.open('xb') as f:f.write(candidate);f.flush();os.fsync(f.fileno())
        if existed!=p.exists() or (existed and p.read_bytes()!=before):fail('CONFIG_CHANGED')
        if existed:
            backup=p.with_name(p.name+'.boxweave-'+uuid.uuid4().hex+'.bak')
            with backup.open('xb') as f:f.write(before)
        os.replace(stage,p)
        return {'changed':True,'backup':str(backup) if backup else None}
    finally:
        stage.unlink(missing_ok=True);lock.unlink(missing_ok=True)

def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('action',choices=['register','diagnose']);parser.add_argument('--descriptor',required=True);parser.add_argument('--replace',action='store_true');args=parser.parse_args()
    result={'connection':{'result':'unverified'},'registration':{'result':'unverified'},'overall':'incomplete','agent_ui':'unverified','model':'not_requested'}
    try:
        if os.name!='nt':fail('WINDOWS_CLIENT_REQUIRED')
        d=descriptor(args.descriptor)
        if program(d['agent_argv']+['--version'],agent_env(d))!=d['agent_version']:fail('AGENT_VERSION_CHANGED')
        argv=ssh_argv(d);connection=probe(argv)
        result={'profile':d['profile'],'connection':connection,'model':'not_requested','agent_ui':'unverified','overall':'incomplete'}
        if args.action=='register':result['write']=write_config(d,argv,args.replace)
        result['registration']=registration(d,argv)
        print(json.dumps(result));return 0
    except (Refused,OSError,ValueError,KeyError,TypeError,subprocess.SubprocessError) as error:
        code=str(error) if isinstance(error,Refused) else 'INPUT_OR_RUNTIME_INVALID'
        result.update(code=code,overall='incomplete',agent_ui='unverified',model='not_requested');print(json.dumps(result));return 2
if __name__=='__main__':raise SystemExit(main())
