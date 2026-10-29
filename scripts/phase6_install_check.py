"""Phase6 real isolated schema6 ->7 install; original services/state remain unchanged."""
import hashlib,json,os,secrets,sqlite3,subprocess,sys,time
from pathlib import Path
from playwright.sync_api import sync_playwright
from werkzeug.security import generate_password_hash
from bwmini.store import Store
r=Path('/home/boxweave-v2-test/boxweave-v2-package-lab/phase6-install/second');old=r.parent.parent/'installed';source=json.loads((old/'runtime.json').read_text());original=Path(source['database']);before=hashlib.sha256(original.read_bytes()).hexdigest()
def props(n):return subprocess.run(['systemctl','--user','show',n,'--property=MainPID','--value'],capture_output=True,text=True,check=True).stdout.strip()
old_units=json.loads((old/'installed.json').read_text())['units'];pids={n:props(n) for n in old_units}
clone=r/'state.sqlite3';assert not clone.exists()
with sqlite3.connect('file:'+str(original)+'?mode=ro',uri=True) as src,sqlite3.connect(clone) as dst:src.backup(dst)
clone.chmod(0o600);password=secrets.token_urlsafe(32);Store(clone).reset_admin(generate_password_hash(password))
with sqlite3.connect(clone) as db:assert db.execute('pragma user_version').fetchone()[0]==6
cfg=dict(source);cfg.update(database=str(clone),port=18660,origin='http://127.0.0.1:18660',secure_cookie=False)
config=r/'input.json';config.write_text(json.dumps(cfg));config.chmod(0o600)
prefix=r/'installed';q=subprocess.run(['/usr/bin/python3',str(r/'reviewed-installer.py'),'install','--archive',str(r/'candidate.zip'),'--sha256','58feeb1823197bc2a3d3aa3df8cdac4a04019d77b011d87a214f8fbcf96efb72','--prefix',str(prefix),'--config',str(config),'--activate'],capture_output=True,text=True,timeout=600)
(r/'installer.log').write_text(q.stdout+q.stderr);assert q.returncode==0,q.stdout;installed=json.loads((prefix/'installed.json').read_text())
try:
 with sqlite3.connect('file:'+str(clone)+'?mode=ro',uri=True) as db:assert db.execute('pragma user_version').fetchone()[0]==7
 with sqlite3.connect('file:'+installed['backup']+'?mode=ro',uri=True) as db:assert db.execute('pragma user_version').fetchone()[0]==6
 with sync_playwright() as w:
  browser=w.chromium.launch();page=browser.new_page();page.goto(cfg['origin']);page.get_by_label('Password',exact=True).fill(password);page.get_by_role('button',name='Sign in',exact=True).click();page.get_by_role('button',name='Client setup',exact=True).wait_for()
  projects=page.request.get(cfg['origin']+'/api/projects').json()['projects'];assert len(projects)==1;project=projects[0]
  page.once('dialog',lambda x:x.accept());page.get_by_role('button',name='Unbind '+project['name'],exact=True).click();page.get_by_text('revision '+str(project['revision']+1)+' · Unbound',exact=True).wait_for()
  after=page.request.get(cfg['origin']+'/api/projects').json()['projects'];assert len(after)==1 and not after[0]['bound']
  page.get_by_role('button',name='Client setup',exact=True).click();page.get_by_role('button',name='Generate public setup',exact=True).wait_for()
  data=json.loads((r/'public-profile-inputs.json').read_text())['codex-native'];token=page.request.get(cfg['origin']+'/api/session').json()['csrf']
  response=page.request.post(cfg['origin']+'/api/client-profile',data=data,headers={'Origin':cfg['origin'],'X-CSRF-Token':token});assert response.status==200 and response.json()['execution']=='not_performed'
  browser.close()
 assert hashlib.sha256(original.read_bytes()).hexdigest()==before and pids=={n:props(n) for n in old_units}
 result={'result':'passed','installer':json.loads(q.stdout),'schema_migrated':{'from':6,'to':7,'backup':6},'installed_browser_unbind_preserves_project':True,'installed_client_generation':True,'original_database_bytes_unchanged':True,'original_service_pids_unchanged':True,'no_workload_mutation':True}
finally:
 for name in installed['units']:subprocess.run(['systemctl','--user','disable','--now',name],capture_output=True,check=True)
result['isolated_test_services_stopped_disabled']=True;(r/'result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
