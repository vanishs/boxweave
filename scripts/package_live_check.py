"""Approved isolated package install/rerun/failure/repair and installed-Web acceptance."""
import argparse,hashlib,json,os,secrets,signal,socket,sqlite3,subprocess,sys,time
from pathlib import Path
from playwright.sync_api import sync_playwright
from werkzeug.security import generate_password_hash
from bwmini.__main__ import load

def main():
 a=argparse.ArgumentParser();a.add_argument('--approved-package-lab',action='store_true');a.add_argument('--expected-sha256',required=True);a.add_argument('--resume-reviewed-install',action='store_true');args=a.parse_args();assert args.approved_package_lab
 r=Path('/home/boxweave-v2-test/boxweave-v2-package-lab');prefix=r/'installed';archive=r/'candidate.zip';installer=r/'reviewed-installer.py'
 assert (not prefix.exists() and not (r/'state.sqlite3').exists()) or args.resume_reviewed_install,'Inspect original evidence; no blind rerun'
 if args.resume_reviewed_install:
  with sqlite3.connect('file:'+str(r/'state.sqlite3')+'?mode=ro',uri=True) as previous:
   assert all(previous.execute('select count(*) from '+n).fetchone()[0]==0 for n in ['operations','environment_operations','graphs']),'Resume only after verified pre-dispatch test failure'
 sha=hashlib.sha256(archive.read_bytes()).hexdigest();assert sha==args.expected_sha256
 cfg=json.loads(Path('/home/boxweave-v2-test/boxweave-v2-access-live/live.json').read_text());assert cfg['backend']=='bw-v2-access-lab'
 cfg.update(database=str(r/'state.sqlite3'),port=18652,origin='http://127.0.0.1:18652',dist=str(prefix/'current/web/dist'),secure_cookie=False);cfg['lifecycle']['allowed_names']=['v2-net-a'];cfg['operator_extension']={'retained':'nondefault-value'}
 config=r/'input.json';config.write_text(json.dumps(cfg));config.chmod(0o600);raw=config.read_bytes();c,s=load(config)
 assert s.provider.lifecycle_inspect('v2-net-a') is None
 password=secrets.token_urlsafe(32);(s.store.reset_admin if args.resume_reviewed_install else s.store.admin)(generate_password_hash(password))
 evidence={'scope':'private loopback installed app; no new outer public route','artifact_sha256':sha,'installs':[],'lifecycle':[],'model_inference':'not_requested'}
 def save(phase): (r/'progress.json').write_text(json.dumps({'phase':phase}));(r/'evidence.json').write_text(json.dumps(evidence,indent=2))
 def command(argv,expected=0,timeout=600):
  q=subprocess.run(argv,stdin=subprocess.DEVNULL,capture_output=True,text=True,timeout=timeout)
  (r/('command-'+str(time.time_ns())+'.log')).write_text(q.stdout+q.stderr)
  assert q.returncode==expected,('command exit',q.returncode)
  return json.loads(q.stdout)
 def install(p=prefix,conf=config,expected=0):
  result=command(['/usr/bin/python3',str(installer),'install','--archive',str(archive),'--sha256',sha,'--prefix',str(p),'--config',str(conf),'--activate'],expected)
  evidence['installs'].append(result);return result
 save('first installation');v=install();assert v['state']=='ready' and v['mcp']['state']=='authenticated_read_passed';save('installed browser workflow')
 health=command(['/usr/bin/python3',str(installer),'health','--prefix',str(prefix),'--mcp']);assert health['state']=='ready' and health['web']['status']==401 and health['web']['authenticated'] is False;evidence['health']=health
 with sync_playwright() as w:
  browser=w.chromium.launch();context=browser.new_context(permissions=['clipboard-read','clipboard-write'],accept_downloads=True,viewport={'width':1280,'height':960});page=context.new_page();errors=[];page.on('pageerror',lambda e:errors.append(str(e)));page.goto(c['origin']);page.get_by_label('Password',exact=True).fill(password);page.get_by_role('button',name='Sign in',exact=True).click()
  def lifecycle(action):
   save(action+' v2-net-a');page.get_by_role('button',name='Environments',exact=True).click();page.get_by_label('Authorized new environment').select_option('v2-net-a');page.get_by_role('button',name='Preview '+action,exact=True).click();page.get_by_label('Type the exact environment name',exact=True).fill('v2-net-a')
   with page.expect_response(lambda x:'/api/environment-plans/' in x.url and x.url.endswith('/confirm'),timeout=90000) as q:page.get_by_role('button',name='Confirm '+action,exact=True).click()
   assert q.value.status==202;op=q.value.json()['id'];deadline=time.monotonic()+1810
   while True:
    v=page.request.get(c['origin']+'/api/environment-operations/'+op).json()
    if v['state'] in ('succeeded','failed','unknown'):break
    assert time.monotonic()<deadline;time.sleep(2)
   evidence['lifecycle'].append(v);save(action+' observed');assert v['state']=='succeeded',v['code'];return v
  lifecycle('create')
  page.get_by_role('button',name='Projects',exact=True).click();page.get_by_label('Reviewed environment').select_option('v2-net-a');page.get_by_label('Project name',exact=True).fill('Installed package project');page.get_by_role('button',name='Save binding',exact=True).click();page.get_by_role('button',name='Preview hello',exact=True).first.click();page.get_by_role('button',name='Confirm hello',exact=True).click();page.get_by_test_id('operation-state').filter(has_text='succeeded').wait_for(timeout=120000)
  evidence['hello']=json.loads(s.list_receipts()[0]['result']);assert evidence['hello']['stdout']=='hello world\n' and evidence['hello']['exit_code']==0
  page.get_by_role('button',name='Prompt Graph',exact=True).click();page.get_by_label('Graph name',exact=True).fill('Package retention');page.get_by_label('Graph prompt',exact=True).fill('Preserve without execution');page.get_by_role('button',name='Add card',exact=True).click();page.get_by_label('Card title',exact=True).fill('Retained card');page.get_by_label('Card text',exact=True).fill('api_key=synthetic-package-secret');page.get_by_label('Comment text',exact=True).fill('Retained comment');page.get_by_role('button',name='Add comment',exact=True).click();page.get_by_role('button',name='Save graph',exact=True).click();page.get_by_role('status').filter(has_text='Saved revision 1').wait_for()
  gid=page.request.get(c['origin']+'/api/graphs').json()['graphs'][0]['id'];graph_before=page.request.get(c['origin']+'/api/graphs/'+gid).json()
  with page.expect_download() as download:page.get_by_role('button',name='Export Markdown',exact=True).click()
  md=Path(download.value.path()).read_text();assert 'synthetic-package-secret' not in md and 'REDACTED' in md
  page.get_by_role('button',name='Model setup',exact=True).click();page.get_by_label('Model provider',exact=True).select_option('zai');page.get_by_label('GLM model',exact=True).select_option('GLM-5.3');page.get_by_label('Native access method',exact=True).select_option('native-existing');page.get_by_role('button',name='Save model selection',exact=True).click();page.get_by_test_id('model-status').filter(has_text='native_setup_pending').wait_for(timeout=10000)
  model_before=page.request.get(c['origin']+'/api/model-wizard').json()
  with s.store.db() as db:counts={n:db.execute('select count(*) from '+n).fetchone()[0] for n in ['projects','graphs','operations','environment_operations','access_operations']}
  save('normal installer rerun');install();assert (prefix/'config.json').read_bytes()==raw
  page.reload();assert page.get_by_role('button',name='Projects',exact=True).is_visible();assert page.request.get(c['origin']+'/api/graphs/'+gid).json()==graph_before;assert page.request.get(c['origin']+'/api/model-wizard').json()==model_before
  with s.store.db() as db:assert counts=={n:db.execute('select count(*) from '+n).fetchone()[0] for n in counts}
  evidence['retention']={'settings_bytes_unchanged':True,'project_graph_comment_model_session_receipts_preserved':True,'no_replay':True,'redacted_markdown':True}
  state=json.loads((prefix/'installed.json').read_text());api=next(n for n in state['units'] if '-serve.' in n)
  command_line=['systemctl','--user','stop',api];subprocess.run(command_line,check=True,capture_output=True)
  down=command(['/usr/bin/python3',str(installer),'health','--prefix',str(prefix),'--mcp'],expected=2);assert down['mcp']['state']=='authenticated_read_passed' and not down['web']['reachable'];evidence['web_down_mcp_works']=True
  subprocess.run(['systemctl','--user','start',api],check=True,capture_output=True);time.sleep(2);page.reload()
  lifecycle('stop');lifecycle('delete');assert s.provider.lifecycle_inspect('v2-net-a') is None;evidence['dedicated_workload_absent']=True
  assert not errors;evidence['browser']={'name':'Chromium','version':browser.version,'errors':errors};browser.close()
 save('tampered artifact refusal')
 bad=r/'tampered.zip';data=bytearray(archive.read_bytes());data[len(data)//2]^=1;bad.write_bytes(data)
 v=command(['/usr/bin/python3',str(installer),'install','--archive',str(bad),'--sha256',sha,'--prefix',str(r/'tampered-prefix'),'--config',str(config),'--activate'],expected=2);assert v['code']=='ARCHIVE_DIGEST_MISMATCH' and not (r/'tampered-prefix').exists();evidence['tamper_before_side_effect']=True
 save('real postpublication failure')
 badcfg=dict(cfg);badcfg.update(database=str(r/'failure-state.sqlite3'),port=18654,origin='http://127.0.0.1:18654');badconfig=r/'failure-input.json';badconfig.write_text(json.dumps(badcfg));badconfig.chmod(0o600);_,failure=load(badconfig);failure.store.admin(generate_password_hash(secrets.token_urlsafe(32)))
 occupant=socket.socket();occupant.bind(('127.0.0.1',18654));occupant.listen()
 badprefix=r/'failure-installed'
 try:
  v=install(badprefix,badconfig,2);assert v['code']=='POSTPUBLICATION_HEALTH_FAILED' and (badprefix/'update-in-progress.json').exists()
 finally:occupant.close()
 marker=json.loads((badprefix/'update-in-progress.json').read_text());assert marker['stage']=='published';evidence['real_failure_marker_stage']=marker['stage'];save('repair after own port-conflict removed')
 v=install(badprefix,badconfig);assert v['state']=='ready' and not (badprefix/'update-in-progress.json').exists();evidence['real_repair_ready']=True
 for n in json.loads((badprefix/'installed.json').read_text())['units']:subprocess.run(['systemctl','--user','disable','--now',n],check=True,capture_output=True)
 evidence['failure_test_units_stopped_and_disabled']=True
 evidence['final_health']=command(['/usr/bin/python3',str(installer),'health','--prefix',str(prefix),'--mcp']);assert evidence['final_health']['state']=='ready'
 evidence['result']='passed';save('completed');print(json.dumps({'result':'passed','installs':len(evidence['installs']),'hello':evidence['hello'],'retention':evidence['retention'],'real_repair':True,'dedicated_workload_absent':True}))
if __name__=='__main__':main()
