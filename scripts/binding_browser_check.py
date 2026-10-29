"""Phase6 installed-source browser regression; fixture infrastructure, no remote writes."""
import argparse,hashlib,zipfile,json,os,secrets,signal,subprocess,sys,time,uuid,urllib.request
from pathlib import Path
from werkzeug.security import generate_password_hash
from playwright.sync_api import sync_playwright
from bwmini.__main__ import load
root=Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--profile-inputs',type=Path,help='Explicit operator-reviewed public metadata only; default is a committed nonexecuting fixture')
args=parser.parse_args();profile_path=args.profile_inputs or root/'examples/client-profiles.public.json'
assert profile_path.stat().st_size<=131072
value=json.loads(profile_path.read_text());inputs=value.get('profiles',value)
assert set(inputs)=={'codex-native','codex-wsl','zcode-native','zcode-wsl'}
state=root/'.runtime/binding-browser'/uuid.uuid4().hex[:12];state.mkdir(mode=0o700,parents=True)
cfg={'mode':'fixture','target':{'provider':'containarium','backend':'fixture','project':'default','logical':'fixture-box','instance':'fixture-box-container','created':'123','user':'fixture-box','hostname':'fixture-box-container'},'database':str(state/'state.sqlite3'),'host':'127.0.0.1','port':18658,'origin':'http://127.0.0.1:18658','dist':str(root/'web/dist'),'secure_cookie':False}
p=state/'config.json';p.write_text(json.dumps(cfg));_,s=load(p);password=secrets.token_urlsafe(24);s.store.admin(generate_password_hash(password))
server=subprocess.Popen([sys.executable,'-m','bwmini','serve','--config',str(p)],cwd=root,stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
try:
 deadline=time.monotonic()+20
 while True:
  try:urllib.request.urlopen(cfg['origin'],timeout=1).close();break
  except OSError:
   assert time.monotonic()<deadline;time.sleep(.2)
 with sync_playwright() as w:
  b=w.chromium.launch();page=b.new_page();errors=[];page.on('pageerror',lambda e:errors.append(str(e)));page.goto(cfg['origin']);page.get_by_label('Password',exact=True).fill(password);page.get_by_role('button',name='Sign in',exact=True).click();page.get_by_label('Project name',exact=True).fill('Retained project');page.get_by_role('button',name='Save binding',exact=True).click();page.get_by_text('Retained project',exact=True).wait_for()
  pid=page.request.get(cfg['origin']+'/api/projects').json()['projects'][0]['id']
  with page.expect_response(lambda r:r.url.endswith('/api/plans')) as response:page.get_by_role('button',name='Preview hello',exact=True).click()
  stale=response.value.json()['id']
  page.get_by_role('button',name='Edit binding Retained project',exact=True).click();assert page.get_by_label('Remote workspace',exact=True).input_value()=='/home/fixture-box';page.get_by_label('Remote workspace',exact=True).fill('/home/fixture-box/changed');page.get_by_role('button',name='Save binding',exact=True).click()
  csrf=page.request.get(cfg['origin']+'/api/session').json()['csrf'];r=page.request.post(cfg['origin']+'/api/plans/'+stale+'/confirm',data={'confirm':True},headers={'Origin':cfg['origin'],'X-CSRF-Token':csrf});assert r.status==409 and r.json()['code']=='REVISION_CONFLICT'
  page.get_by_role('button',name='Preview hello',exact=True).click();page.get_by_role('button',name='Confirm hello',exact=True).click();page.get_by_test_id('operation-state').filter(has_text='queued').wait_for()
  page.once('dialog',lambda x:x.accept());page.get_by_role('button',name='Unbind Retained project',exact=True).click();page.get_by_text('revision 3 · Unbound',exact=True).wait_for();assert page.get_by_role('button',name='Preview hello',exact=True).is_disabled()
  s.worker(once=True);assert s.list_receipts()[0]['state']=='failed' and s.provider.calls==[]
  page.reload();page.get_by_text('revision 3 · Unbound',exact=True).wait_for();assert page.request.get(cfg['origin']+'/api/projects').json()['projects'][0]['id']==pid
  page.get_by_role('button',name='Edit binding Retained project',exact=True).click();page.get_by_label('Reviewed environment').select_option('fixture-box');page.get_by_label('Remote workspace',exact=True).fill('/home/fixture-box/rebound');page.get_by_role('button',name='Save binding',exact=True).click();page.get_by_text('revision 4 · fixture-box · /home/fixture-box/rebound',exact=True).wait_for()
  rows=page.request.get(cfg['origin']+'/api/projects').json()['projects'];assert len(rows)==1 and rows[0]['bound'] and rows[0]['target']==cfg['target']
  page.get_by_role('button',name='Client setup',exact=True).click()
  downloads={}
  for name,data in inputs.items():
   page.get_by_label('Client profile',exact=True).select_option(data['profile'])
   fields={'agent_executable':'Node executable' if data['profile'].startswith('zcode/') else 'Codex executable','agent_version':'Exact Agent version','profile_directory':'Dedicated profile directory','host':'SSH host','user':'SSH user','port':'SSH port','transport_executable':'Transport executable','identity_file':'Private key path only','known_hosts':'Known-hosts path','server_key':'Public SSH server key'}
   if data['profile'].endswith('/wsl'):fields['distribution']='Explicit WSL distribution'
   if data['profile'].startswith('zcode/'):fields.update(agent_core='ZCode core script',builtin_path='Public ZCode catalog path',builtin_sha256='Public catalog SHA256')
   for key,label in fields.items():page.get_by_label(label,exact=True).fill(str(data[key]))
   page.get_by_label('Allow replacement of the BoxWeave registration; preserve unrelated settings',exact=True).check()
   with page.expect_response(lambda q:q.url.endswith('/api/client-profile')) as response:page.get_by_role('button',name='Generate public setup',exact=True).click()
   assert response.value.status==200;generated=response.value.json()
   for button,filename in [('Download public descriptor','boxweave-client.json'),('Download reviewed enrollment script','Register-BoxWeave.ps1')]:
    with page.expect_download() as download:page.get_by_role('button',name=button,exact=True).click()
    raw=Path(download.value.path()).read_bytes();downloads[name+'/'+filename]=raw
    if filename.endswith('.json'):assert hashlib.sha256(raw).hexdigest()==generated['descriptor_sha256']
    else:assert '--replace' in raw.decode() and generated['descriptor_sha256'] in raw.decode()
  out=state/'public-profiles.zip'
  with zipfile.ZipFile(out,'w',zipfile.ZIP_DEFLATED) as z:
   for name,raw in downloads.items():z.writestr(name,raw)
  profile_artifact={'profiles':4,'files':len(downloads),'sha256':hashlib.sha256(out.read_bytes()).hexdigest()}
  page.set_viewport_size({'width':390,'height':844});assert not page.evaluate('document.documentElement.scrollWidth>innerWidth')
  assert not errors
  result={'result':'passed','infrastructure':'fixture; zero provider execution calls','browser':b.version,'project_identity_preserved':True,'reselect_edit_unbind_reload_rebind':True,'stale_confirmation_rejected':True,'queued_receipt_retained_failed_without_execution':True,'javascript_errors':errors,'public_web_profile_downloads':profile_artifact,'profile_input_kind':'explicit_operator_metadata' if args.profile_inputs else 'committed_nonexecuting_fixture','artifact_directory':str(state)}
  (state/'result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result));b.close()
finally:
 os.killpg(server.pid,signal.SIGTERM);server.wait(timeout=10)
