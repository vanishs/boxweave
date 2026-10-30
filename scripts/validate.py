"""One fixture-only maintenance lane; never installs dependencies or runs live lab drivers."""
import argparse,datetime,importlib.util,json,os,shutil,subprocess,sys,time,uuid
from pathlib import Path
BROWSERS=('browser_check','history_browser_check','graph_browser_check','access_browser_check','model_browser_check','lifecycle_browser_check','binding_browser_check')
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--browser',action='store_true',help='Also run all seven loopback fixture browser workflows');p.add_argument('--browser-only',action='store_true',help='Run only the seven fixture browser workflows against an existing build');a=p.parse_args()
 root=Path(__file__).resolve().parents[1]
 if sys.platform!='linux' or os.geteuid()==0:p.error('Use the prepared non-root Linux project environment; docs-only checks are portable')
 missing=[x for x in ('flask','waitress')+ (('playwright',) if (a.browser or a.browser_only) else ()) if importlib.util.find_spec(x) is None]
 if missing or shutil.which('npm') is None:p.error('Prepared locked dependencies/Node are missing; see FINAL_HANDOFF, no automatic installation')
 folder=root/'.runtime/validation'/uuid.uuid4().hex;folder.mkdir(mode=0o700,parents=True)
 deadline=time.monotonic()+600;start=time.time()
 report={'state':'running','started_utc':datetime.datetime.fromtimestamp(start,datetime.timezone.utc).isoformat(),'deadline_utc':datetime.datetime.fromtimestamp(start+600,datetime.timezone.utc).isoformat(),'scope':'fixture-only tests/build; no live MCP/profile enrollment/deployment','steps':[]}
 def save():
  tmp=folder/'result.tmp';tmp.write_text(json.dumps(report,indent=2));tmp.chmod(0o600);os.replace(tmp,folder/'result.json')
 commands=[('knowledge',[sys.executable,str(root/'scripts/check_knowledge.py')]),('unit',[sys.executable,'-m','unittest','discover','-s','tests','-v']),('build',['npm','--prefix','web','run','build'])]
 if a.browser_only:commands=[]
 if a.browser or a.browser_only:commands += [(name,[sys.executable,str(root/'scripts'/ (name+'.py'))]) for name in BROWSERS]
 save()
 env=dict(os.environ,PYTHONPATH=str(root))
 for name,argv in commands:
  step={'name':name,'state':'running','exit_code':None};report['steps'].append(step);save();print('Running '+name,flush=True);began=time.monotonic()
  try:
   remaining=deadline-time.monotonic()
   if remaining<=0:raise subprocess.TimeoutExpired(argv,0)
   with (folder/(name+'.log')).open('xb') as log:
    os.chmod(log.name,0o600);q=subprocess.run(argv,cwd=root,env=env,stdin=subprocess.DEVNULL,stdout=log,stderr=subprocess.STDOUT,timeout=remaining)
   step.update(state='passed' if q.returncode==0 else 'failed',exit_code=q.returncode)
  except subprocess.TimeoutExpired:step.update(state='timeout',descendant_cleanup='unverified; inspect this run before any rerun')
  except OSError:step.update(state='failed',code='PROCESS_UNAVAILABLE')
  step['elapsed_seconds']=round(time.monotonic()-began,3);save()
  if step['state']!='passed':report['state']=step['state'];save();print(json.dumps({'state':report['state'],'step':name,'report':str(folder/'result.json')}));return 1
 report['state']='passed';save();print(json.dumps({'state':'passed','steps':len(commands),'report':str(folder/'result.json')}));return 0
if __name__=='__main__':raise SystemExit(main())
