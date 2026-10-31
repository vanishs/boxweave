"""Reviewed standalone Linux app distribution builder/installer. Never installs a provider."""
import argparse,fcntl,hashlib,json,os,platform,re,shutil,sqlite3,stat,subprocess,sys,tempfile,time,urllib.error,urllib.request,urllib.parse,uuid,venv,zipfile
from pathlib import Path

PLATFORM = 'linux-x86_64-cpython3-min312'

class InstallError(Exception): pass
def require(ok,code):
 if not ok:raise InstallError(code)
def digest(data):return hashlib.sha256(data).hexdigest()
def encoded(value):return json.dumps(value,sort_keys=True,indent=2).encode()+b'\n'
def atomic(path,data,mode=0o600):
 path=Path(path);tmp=path.with_name('.'+path.name+'.'+uuid.uuid4().hex)
 try:
  fd=os.open(tmp,os.O_CREAT|os.O_EXCL|os.O_WRONLY,mode)
  with os.fdopen(fd,'wb') as f:f.write(data);f.flush();os.fsync(f.fileno())
  os.replace(tmp,path)
  fd=os.open(path.parent,os.O_RDONLY)
  try:os.fsync(fd)
  finally:os.close(fd)
 finally:tmp.unlink(missing_ok=True)
def regular(path):
 require(not path.is_symlink() and path.is_file(),'UNSAFE_FILE')
 return path.read_bytes()
def safe_name(name):
 p=Path(name)
 return bool(name) and not name.startswith(('/','\\')) and '\\' not in name and ':' not in name and all(x not in ('','..','.') for x in name.split('/')) and str(p)==name
def verify(archive,sha):
 require(re.fullmatch('[0-9a-f]{64}',sha or '') is not None,'EXPECTED_DIGEST_REQUIRED')
 raw=regular(Path(archive));require(len(raw)<=100_000_000 and digest(raw)==sha,'ARCHIVE_DIGEST_MISMATCH')
 with zipfile.ZipFile(archive) as z:
  infos=z.infolist();names=[x.filename for x in infos]
  require(len(names)==len(set(names)) and 1<len(names)<=2048,'ARCHIVE_MEMBERS_INVALID')
  require(all(safe_name(x.filename) and not x.is_dir() and x.file_size<=30_000_000 and not stat.S_ISLNK(x.external_attr>>16) for x in infos),'ARCHIVE_PATH_INVALID')
  require(sum(x.file_size for x in infos)<=180_000_000,'ARCHIVE_TOO_LARGE')
  manifest=json.loads(z.read('manifest.json'));require(set(manifest)=={'format','version','platform','source','files'},'MANIFEST_INVALID')
  require(manifest['format']==1 and re.fullmatch(r'0\.[0-9]+\.[0-9]+',manifest['version']) is not None,'MANIFEST_INVALID')
  require(manifest['platform'] == PLATFORM, 'PLATFORM_UNSUPPORTED')
  require(set(manifest['source'])=={'commit','state'} and re.fullmatch('[0-9a-f]{40}',manifest['source']['commit']) is not None and manifest['source']['state'] in ('clean','modified'),'PROVENANCE_INVALID')
  files=manifest['files'];require(isinstance(files,dict) and set(files)|{'manifest.json'}==set(names),'MANIFEST_MEMBERS_MISMATCH')
  require({'bwmini/__main__.py','bwmini/deploy_check.py','requirements.lock','VERSION','web/dist/index.html'}<=set(files),'PACKAGE_INCOMPLETE')
  require(all(n.startswith(('bwmini/','clients/','web/dist/','wheels/')) or n in ('VERSION','requirements.lock','boxweave.sh','install.sh') for n in files),'PACKAGE_CONTENT_INVALID')
  for name,sha256 in files.items():require(re.fullmatch('[0-9a-f]{64}',sha256) is not None and digest(z.read(name))==sha256,'MEMBER_DIGEST_MISMATCH')
  require(z.read('VERSION').decode().strip()==manifest['version'],'VERSION_MISMATCH')
  return manifest
def build(root,wheels,output,commit,source_state):
 root=Path(root).resolve();files={}
 modules=('__init__','__main__','bootstrap','online','client_setup','hostkeys','mcp_gate','nesting','manage','access','access_provider','deploy_check','distribution','domain','graphs','jobs','lifecycle','mcp','models','profiles','provider','selftest','service','store','web')
 clients=('profile_client.py','README.md','wsl/prepare_identity.py','windows/register-native.ps1','windows/native-package.example.json','windows/prepare-identity.ps1','windows/bootstrap-native.ps1','windows/diagnose-native.ps1','windows/diagnose-connection.ps1','windows/connection.example.json')
 public=[root/'bwmini'/(n+'.py') for n in modules]+[root/'clients'/n for n in clients]+[root/'boxweave.sh',root/'install.sh']+list((root/'web/dist').rglob('*'))
 for p in public:
  if p.is_file():
   require(not p.is_symlink(),'SYMLINK_SOURCE');files[p.relative_to(root).as_posix()]=p.read_bytes()
 for n in ('requirements.lock','VERSION'):files[n]=regular(root/n)
 for p in Path(wheels).glob('*.whl'):files['wheels/'+p.name]=regular(p)
 require(bool([x for x in files if x.startswith('wheels/')]),'OFFLINE_WHEELS_REQUIRED')
 # pip enforces the independently locked wheel digests during installation.
 manifest={'format':1,'version':files['VERSION'].decode().strip(),'platform':PLATFORM,'source':{'commit':commit,'state':source_state},'files':{n:digest(b) for n,b in files.items()}}
 require(not Path(output).exists(),'OUTPUT_EXISTS')
 with zipfile.ZipFile(output,'x',compression=zipfile.ZIP_DEFLATED) as z:
  for n,b in sorted(files.items()):
   i=zipfile.ZipInfo(n,(2026,1,1,0,0,0));i.compress_type=zipfile.ZIP_DEFLATED;i.external_attr=0o100600<<16;z.writestr(i,b)
  i=zipfile.ZipInfo('manifest.json',(2026,1,1,0,0,0));i.compress_type=zipfile.ZIP_DEFLATED;i.external_attr=0o100600<<16;z.writestr(i,encoded(manifest))
 sha=digest(Path(output).read_bytes());verify(output,sha)
 return {'archive':str(output),'sha256':sha,'version':manifest['version'],'source':manifest['source']}
def properties(unit):
 r=subprocess.run(['systemctl','--user','show',unit,'--property=ActiveState,SubState,Result,ExecMainStatus,UnitFileState,MainPID'],capture_output=True,text=True,timeout=10)
 return dict(x.split('=',1) for x in r.stdout.splitlines() if '=' in x) if r.returncode==0 else {}
def unit_ready(props,timer=False,oneshot=False):
 if timer:return props.get('UnitFileState')=='enabled' and props.get('ActiveState')=='active' and props.get('SubState')=='waiting'
 if oneshot:return props.get('Result')=='success' and props.get('ExecMainStatus')=='0' and props.get('ActiveState') in ('active','inactive')
 return props.get('UnitFileState')=='enabled' and props.get('ActiveState')=='active' and props.get('SubState')=='running'
def systemctl(*args):
 p=subprocess.run(['systemctl','--user',*args],capture_output=True,timeout=30)
 require(p.returncode==0,'USER_SERVICE_CONTROL_FAILED')
def stop_owned_unit(name):
 props=properties(name)
 require(bool(props.get('ActiveState')) and props.get('MainPID','').isdigit(),'USER_SERVICE_STATE_UNAVAILABLE')
 if props['ActiveState'] in ('inactive','failed') and props['MainPID']=='0':return
 systemctl('stop',name)
def q(value):return '"'+str(value).replace('\\','\\\\').replace('"','\\"').replace('%','%%')+'"'
def unit_text(prefix,config,action):
 return ('[Unit]\nDescription=BoxWeave '+action+'\n[Service]\nType=simple\nWorkingDirectory='+str(prefix/'current').replace('%','%%').replace(' ',r'\x20').replace(chr(34),r'\x22')+'\nExecStart='+q(prefix/'current/.venv/bin/python')+' -m bwmini '+action+' --config '+q(config)+'\nRestart=on-failure\nRestartSec=2\nUMask=0077\n[Install]\nWantedBy=default.target\n').encode()
def config_read(path,prefix):
 raw=regular(Path(path));require(len(raw)<=65536,'CONFIG_TOO_LARGE');c=json.loads(raw)
 require(isinstance(c,dict) and c.get('mode')=='mcp' and c.get('host','127.0.0.1')=='127.0.0.1','CONFIG_INVALID')
 db=Path(c.get('database',''));require(db.is_absolute() and not db.is_symlink() and not db.is_relative_to(prefix/'releases'),'STATE_PATH_INVALID')
 require(type(c.get('port')) is int and 1024<=c['port']<=65535 and isinstance(c.get('origin'),str),'CONFIG_INVALID')
 u=urllib.parse.urlsplit(c['origin']);require(u.scheme in ('http','https') and bool(u.hostname) and u.username is None and u.password is None and not u.path and not u.query and not u.fragment,'ORIGIN_INVALID')
 require(u.scheme=='https' or u.hostname in ('127.0.0.1','localhost'),'ORIGIN_INVALID')
 require(c.get('secure_cookie',True)==(u.scheme=='https'),'COOKIE_ORIGIN_INVALID')
 argv=c.get('launcher');require(isinstance(argv,list) and bool(argv) and all(isinstance(x,str) for x in argv) and Path(argv[0]).is_absolute(),'CONFIG_INVALID')
 return raw,c
def installed(prefix):
 p=prefix/'installed.json'
 return json.loads(regular(p)) if p.exists() else None
def current_release(prefix):
 p=prefix/'current'
 require(not p.exists() or p.is_symlink(),'FOREIGN_CURRENT')
 if not p.is_symlink():return None
 target=p.resolve();require(target.parent==prefix/'releases' and re.fullmatch('[0-9a-f]{64}',target.name) is not None,'FOREIGN_CURRENT')
 return target
def release_check(release,manifest):
 for name,sha in manifest['files'].items():require(digest(regular(release/name))==sha,'INSTALLED_FILE_CHANGED')
 require(json.loads(regular(release/'manifest.json'))==manifest,'INSTALLED_MANIFEST_CHANGED')
def health(prefix,mcp=False):
 prefix=Path(prefix).resolve();state=installed(prefix);result={'state':'starting','incomplete':(prefix/'update-in-progress.json').exists(),'installed':state,'services':{},'web':{'reachable':False,'authenticated':False},'mcp':{'state':'not_checked'},'boot_autostart':'unverified','confinement':'unverified'}
 if not state:return result
 try:
  release=current_release(prefix);manifest=json.loads(regular(release/'manifest.json'));release_check(release,manifest)
  require(release.name==state['archive_sha256'] and digest(regular(release/'manifest.json'))==state['manifest_sha256'],'PROVENANCE_CHANGED')
  raw,c=config_read(prefix/'config.json',prefix)
  for unit in state['units']:result['services'][unit]=properties(unit)
  try:
   urllib.request.urlopen('http://127.0.0.1:'+str(c['port'])+'/api/projects',timeout=3).close();result['web']={'reachable':True,'authenticated':False,'status':200}
  except urllib.error.HTTPError as e:result['web']={'reachable':True,'authenticated':False,'status':e.code}
  except OSError:pass
  if mcp:
   probe=subprocess.run([str(release/'.venv/bin/python'),'-m','bwmini.deploy_check','--config',str(prefix/'runtime.json'),'--mcp'],cwd=release,capture_output=True,text=True,timeout=100)
   require(probe.returncode==0,'MCP_READ_FAILED');checked=json.loads(probe.stdout);result['mcp']=checked['mcp'];result['administrator_initialized']=checked['admin_initialized']
   require(checked['admin_initialized'],'ADMIN_SETUP_REQUIRED')
  ready=all(unit_ready(p) for p in result['services'].values()) and bool(state['units']) and result['web']['reachable']
  failed=any(p.get('ActiveState')=='failed' or p.get('Result') in ('exit-code','signal','core-dump','start-limit-hit') for p in result['services'].values())
  result['state']='incomplete' if result['incomplete'] else 'failed' if failed else 'ready' if ready else 'starting'
 except (InstallError,OSError,ValueError,subprocess.SubprocessError) as e:result['state']='failed';result['code']=str(e) if isinstance(e,InstallError) else 'HEALTH_UNAVAILABLE'
 return result
def install(archive,sha,prefix,config=None,activate=False,quiesced=False):
 require(sys.platform=='linux' and platform.machine()=='x86_64' and sys.version_info[:2]>=(3,12) and platform.python_implementation()=='CPython','PLATFORM_UNSUPPORTED')
 manifest=verify(archive,sha) # No untrusted package code executes before this.
 prefix=Path(prefix).absolute();require(prefix==prefix.resolve() and '\n' not in str(prefix),'PREFIX_UNSAFE')
 require(os.geteuid()!=0,'RUN_AS_APPLICATION_USER')
 require(activate or quiesced,'QUIESCE_REQUIRED')
 if prefix.exists():require(prefix.stat().st_uid==os.getuid() and not prefix.is_symlink() and not prefix.stat().st_mode&0o077,'PREFIX_NOT_PRIVATE')
 saved=prefix/'config.json';config=Path(config) if config else saved
 raw,c=config_read(config,prefix)
 if saved.exists():require(regular(saved)==raw,'CONFIG_CONFLICT')
 prefix.mkdir(mode=0o700,parents=True,exist_ok=True)
 with (prefix/'.install.lock').open('a') as lock:
  try:fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
  except BlockingIOError:raise InstallError('INSTALL_BUSY')
  old=installed(prefix);current=current_release(prefix)
  if current:
   old_manifest=json.loads(regular(current/'manifest.json'))
   require(tuple(map(int,manifest['version'].split('.')))>=tuple(map(int,old_manifest['version'].split('.'))),'DOWNGRADE_REQUIRES_BACKUP')
  identifier='boxweave-v2-'+digest(str(prefix).encode())[:12]
  units={identifier+'-'+action+'.service':unit_text(prefix,prefix/'runtime.json',action) for action in ('serve','worker')}
  directory=Path.home()/'.config/systemd/user'
  if activate:
   require(bool(os.environ.get('XDG_RUNTIME_DIR')),'USER_BUS_REQUIRED')
   directory.mkdir(mode=0o700,parents=True,exist_ok=True)
   for name,content in units.items():
    f=directory/name
    if f.exists():
     previous=regular(f);legacy=content.replace(('WorkingDirectory='+str(prefix/'current').replace('%','%%').replace(' ',r'\x20').replace(chr(34),r'\x22')).encode(),('WorkingDirectory='+q(prefix/'current')).encode())
     owned=old is not None and name in old['units'] and (digest(previous)==old.get('unit_sha256',{}).get(name) or ('unit_sha256' not in old and previous in (content,legacy,content.replace(b'Description=BoxWeave ',b'Description=BoxWeave v2 '),legacy.replace(b'Description=BoxWeave ',b'Description=BoxWeave v2 '))))
     require(owned,'FOREIGN_SERVICE')
  marker=prefix/'update-in-progress.json'
  def mark(stage,**kw):atomic(marker,encoded({'stage':stage,'archive_sha256':sha,'previous':current.name if current else None,'time':time.time(),**kw}))
  mark('verified')
  if not saved.exists():atomic(saved,raw)
  runtime=dict(c);runtime['dist']=str(prefix/'current/web/dist');atomic(prefix/'runtime.json',encoded(runtime))
  releases=prefix/'releases';releases.mkdir(mode=0o700,exist_ok=True);release=releases/sha
  if not release.exists():
   staging=releases/('.stage-'+uuid.uuid4().hex);staging.mkdir(mode=0o700)
   with zipfile.ZipFile(archive) as z:
    for name in z.namelist():
     path=staging/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(z.read(name));path.chmod(0o600)
   os.replace(staging,release)
  release_check(release,manifest);mark('staged')
  if activate:
   for name in units:
    if (directory/name).exists():stop_owned_unit(name)
  environment=release/'.venv'
  if environment.exists():
   require(not environment.is_symlink(),'UNSAFE_ENVIRONMENT')
   environments=prefix/'environment-backups';environments.mkdir(mode=0o700,exist_ok=True)
   os.replace(environment,environments/(sha+'-'+str(time.time_ns())))
  venv.EnvBuilder(with_pip=True).create(environment)
  p=subprocess.run([str(environment/'bin/python'),'-m','pip','install','--no-index','--no-deps','--require-hashes','--only-binary=:all:','--find-links',str(release/'wheels'),'-r',str(release/'requirements.lock')],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=300)
  require(p.returncode==0,'OFFLINE_DEPENDENCIES_FAILED')
  mark('quiesced')
  database=Path(c['database']);database.parent.mkdir(mode=0o700,parents=True,exist_ok=True)
  with Path(str(database)+'.worker').open('a') as worker_lock:
   try:fcntl.flock(worker_lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
   except BlockingIOError:raise InstallError('WORKER_NOT_QUIESCED')
   backup=None
   if database.exists():
    require(not database.is_symlink(),'STATE_PATH_INVALID')
    backups=prefix/'backups';backups.mkdir(mode=0o700,exist_ok=True);backup=backups/(str(time.time_ns())+'.sqlite3')
    with sqlite3.connect('file:'+str(database)+'?mode=ro',uri=True) as source,sqlite3.connect(backup) as destination:source.backup(destination)
    backup.chmod(0o600)
   mark('backup',backup=str(backup) if backup else None)
   staged_config=prefix/'candidate-config.json';candidate=dict(runtime);candidate['dist']=str(release/'web/dist');atomic(staged_config,encoded(candidate))
   check=subprocess.run([str(environment/'bin/python'),'-m','bwmini.deploy_check','--config',str(staged_config)],cwd=release,capture_output=True,text=True,timeout=30)
   require(check.returncode==0,'CANDIDATE_CONFIG_OR_STATE_FAILED')
   mark('publishing',backup=str(backup) if backup else None)
   link=prefix/('.current-'+uuid.uuid4().hex);link.symlink_to(release,target_is_directory=True);os.replace(link,prefix/'current')
   atomic(prefix/'installed.json',encoded({'version':manifest['version'],'archive_sha256':sha,'manifest_sha256':digest(regular(release/'manifest.json')),'source':manifest['source'],'units':list(units) if activate else [],'unit_sha256':{n:digest(b) for n,b in units.items()} if activate else {},'backup':str(backup) if backup else None}))
   mark('published',backup=str(backup) if backup else None)
  if activate:
   for name,content in units.items():atomic(directory/name,content)
   systemctl('daemon-reload')
   for name in units:systemctl('enable','--now',name)
   deadline=time.monotonic()+30
   while True:
    status=health(prefix)
    if status['web']['reachable'] and all(unit_ready(p) for p in status['services'].values()):break
    require(time.monotonic()<deadline,'POSTPUBLICATION_HEALTH_FAILED');time.sleep(.5)
  else:
   status=health(prefix);require(status['state']!='failed','POSTPUBLICATION_VERIFY_FAILED')
  # Verify a real read independently from HTTP reachability before clearing marker.
  checked=subprocess.run([str(environment/'bin/python'),'-m','bwmini.deploy_check','--config',str(prefix/'runtime.json'),'--mcp'],cwd=release,capture_output=True,text=True,timeout=100)
  require(checked.returncode==0,'MCP_READ_FAILED')
  require(json.loads(checked.stdout).get('admin_initialized') is True,'ADMIN_SETUP_REQUIRED')
  mark('checked',backup=str(backup) if backup else None);marker.unlink()
  return {'state':'ready' if activate else 'installed_services_not_activated','archive_sha256':sha,'version':manifest['version'],'backup':str(backup) if backup else None,'mcp':json.loads(checked.stdout)['mcp']}
def main():
 p=argparse.ArgumentParser(description=__doc__);sub=p.add_subparsers(dest='action',required=True)
 b=sub.add_parser('build');b.add_argument('--root',required=True);b.add_argument('--wheels',required=True);b.add_argument('--output',required=True);b.add_argument('--source-commit',required=True);b.add_argument('--source-state',choices=['clean','modified'],required=True)
 i=sub.add_parser('install');i.add_argument('--archive','--package',dest='archive',required=True);i.add_argument('--sha256',required=True);i.add_argument('--prefix',required=True);i.add_argument('--config');i.add_argument('--activate',action='store_true');i.add_argument('--quiesced',action='store_true')
 h=sub.add_parser('health');h.add_argument('--prefix',required=True);h.add_argument('--mcp',action='store_true')
 a=p.parse_args()
 try:
  if a.action=='build':result=build(a.root,a.wheels,a.output,a.source_commit,a.source_state)
  elif a.action=='install':result=install(a.archive,a.sha256,a.prefix,a.config,a.activate,a.quiesced)
  else:result=health(a.prefix,a.mcp)
  print(json.dumps(result));return 0 if result.get('state') not in ('failed','incomplete','starting') else 2
 except (InstallError,OSError,ValueError,KeyError,subprocess.SubprocessError,zipfile.BadZipFile) as e:
  print(json.dumps({'state':'incomplete' if a.action=='install' and (Path(a.prefix)/'update-in-progress.json').exists() else 'failed','code':str(e) if isinstance(e,InstallError) else 'INSTALL_OR_RUNTIME_FAILED'}));return 2
if __name__=='__main__':raise SystemExit(main())
