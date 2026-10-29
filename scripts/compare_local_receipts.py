"""Same-input local old/new receipt observations; no remote Provider comparison."""
import importlib.util,json,os,subprocess,sys,tempfile,time,zipfile
from pathlib import Path
from bwmini.store import Store
from bwmini.service import Service
from bwmini.provider import Fixture
from bwmini.domain import Target,Fault
root=Path(__file__).resolve().parents[1];old=root/'.runtime/phase6/legacy/server/jobs.py'
sys.path.insert(0,str(old.parent));spec=importlib.util.spec_from_file_location('phase6_old_jobs',old);jobs=importlib.util.module_from_spec(spec);spec.loader.exec_module(jobs)
target=Target('containarium','local-fixture','default','fixture-box','fixture-box-container','123','fixture-box','fixture-box-container')
records=[]
with tempfile.TemporaryDirectory(prefix='bw-comparison-') as tmp:
 for exit_code in (0,7):
  code='print("hello world");raise SystemExit('+str(exit_code)+')';argv=[sys.executable,'-c',code]
  case=Path(tmp)/str(exit_code);case.mkdir(mode=0o700)
  receipt=jobs.start(argv,root=case/'old');deadline=time.monotonic()+10
  while True:
   before=jobs.status(receipt['job_id'],case/'old',logs=True)
   if before['state'] in jobs.TERMINAL:break
   assert time.monotonic()<deadline;time.sleep(.03)
  class Local(Fixture):
   def hello(self,target,path,timeout):
    p=subprocess.run(argv,capture_output=True,text=True,timeout=timeout)
    return {'stdout':p.stdout,'exit_code':p.returncode}
  service=Service(Store(case/'new/state.sqlite3'),Local(target))
  project=service.bind('same input',target.logical,'/home/fixture-box');op=service.confirm(service.plan(project['id'],1)['id']);service.worker(once=True);after=service.store.receipt(op['id']);result=json.loads(after['result'])
  assert before['state']==after['state'] and before['exit_code']==result['exit_code']==exit_code
  records.append({'input':code,'old':{'state':before['state'],'exit_code':before['exit_code'],'stdout':before['logs']['stdout']},'new':{'state':after['state'],'exit_code':result['exit_code'],'stdout':result['stdout'],'output_withheld':result['output_withheld']},'classification':'same successful result' if exit_code==0 else 'approved narrower output disclosure; same failure/exit','evidence':'actual local Python processes, isolated SQLite/old job files; fixture identity, no remote integration'})
 # Current Store7 versus actual old Store6 code from the verified phase5 artifact.
 p=root/'.runtime/task012/boxweave-v2-0.9.0-f602acf0996b.zip'
 import hashlib
 assert hashlib.sha256(p.read_bytes()).hexdigest()=='046d8f0ca8117f7ee9740b5db671fdfc7b146a5064380df777c5e4908e8c2df2'
 path=Path(tmp)/'version-test/state.sqlite3';current=Store(path)
 with zipfile.ZipFile(p) as z:old_code=z.read('bwmini/store.py')
 namespace={'__name__':'bwmini.phase6_old_store','__package__':'bwmini'};exec(compile(old_code,'verified-v090-store.py','exec'),namespace)
 try:namespace['Store'](path)
 except Fault as error:assert error.code=='SCHEMA_UNSUPPORTED';older_refused=True
 else:raise AssertionError('Old binary accepted schema7')
 old_root=Path(tmp)/'v090';old_root.mkdir(mode=0o700)
 with zipfile.ZipFile(p) as z:
  for n in z.namelist():
   if n.startswith('bwmini/') and n.endswith('.py'):
    f=old_root/n;f.parent.mkdir(parents=True,exist_ok=True);f.write_bytes(z.read(n))
 config=Path(tmp)/'dead-launcher.json';config.write_text(json.dumps({'mode':'mcp','provider':'containarium','backend':'fixture','project':'default','targets':{},'launcher':['/usr/bin/false'],'database':str(Path(tmp)/'diagnostic/state.sqlite3')}))
 env=dict(os.environ,PYTHONPATH=str(old_root));old_diagnosis=subprocess.run([sys.executable,'-m','bwmini','diagnose','--config',str(config)],cwd=old_root,env=env,capture_output=True,text=True)
 assert old_diagnosis.returncode==0
 env['PYTHONPATH']=str(root);new_diagnosis=subprocess.run([sys.executable,'-m','bwmini','diagnose','--config',str(config)],cwd=root,env=env,capture_output=True,text=True)
 assert new_diagnosis.returncode!=0
 false_ready={'old_v090_exit':old_diagnosis.returncode,'new_exit':new_diagnosis.returncode,'old_no_provider_check':json.loads(old_diagnosis.stdout),'new_failure':json.loads(new_diagnosis.stdout),'same_input':'empty targets plus /usr/bin/false launcher'}
result={'result':'passed','records':records,'v090_refuses_schema7':older_refused,'false_ready_counterexample':false_ready,'scope':'local isolated execution comparison, not old/new live infrastructure or performance benchmark'}
(root/'.runtime/phase6/same-input.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
