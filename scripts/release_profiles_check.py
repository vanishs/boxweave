"""Four actual Windows-hosted Agent transport checks; fixed existing workload command only."""
import argparse,hashlib,importlib.util,json,os,re,subprocess,sys
from pathlib import Path

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--descriptors',nargs=4,required=True);parser.add_argument('--result',required=True);a=parser.parse_args();assert os.name=='nt'
 root=Path(__file__).resolve().parents[1];p=root/'clients/profile_client.py';spec=importlib.util.spec_from_file_location('profile_client',p);helper=importlib.util.module_from_spec(spec);spec.loader.exec_module(helper)
 records=[]
 def envelope(value):
  assert not value.get('isError');text='\n'.join(x['text'] for x in value['content']);m=re.fullmatch(r'exit_code: (-?\d+)\n\n--- stdout ---\n([\s\S]*)',text);assert m;return int(m[1]),m[2]
 for item in a.descriptors:
  d=helper.descriptor(item);argv=helper.ssh_argv(d);before=helper.config_path(d).read_bytes()
  registration=helper.registration(d,argv);read=helper.probe(argv)
  with helper.RPC(argv,timeout=90) as client:
   server=client.call('initialize',{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'boxweave-release-fixed-hello','version':'0.9.0'}});client.send({'jsonrpc':'2.0','method':'notifications/initialized'});tools=client.call('tools/list',{})['tools'];assert 'connect' in [t['name'] for t in tools]
   raw=client.call('tools/call',{'name':'get_container','arguments':{'username':'v2-live-hello'}});assert not raw.get('isError')
   target=json.loads('\n'.join(x['text'] for x in raw['content']))['container']
   assert target['name']=='v2-live-hello-container' and str(target['createdAt'])=='1793213048' and target['username']=='v2-live-hello' and target['state']=='CONTAINER_STATE_RUNNING'
   hello_dispatches=[]
   class IdentityRefused(Exception):pass
   def guarded(expected):
    code,identity=envelope(client.call('tools/call',{'name':'connect','arguments':{'box':'v2-live-hello','exec':'whoami && hostname'}}))
    if code!=0 or identity!=expected:raise IdentityRefused()
    hello_dispatches.append(True)
    return envelope(client.call('tools/call',{'name':'connect','arguments':{'box':'v2-live-hello','exec':"python3 -c 'print(\"hello world\")'"}})),identity
   try:guarded('wrong-identity\nv2-live-hello-container\n')
   except IdentityRefused:pass
   else:raise AssertionError('Wrong execution identity was accepted')
   assert hello_dispatches==[];mismatch_stopped=True
   (code,out),identity=guarded('v2-live-hello\nv2-live-hello-container\n');assert len(hello_dispatches)==1
   assert code==0 and out=='hello world\n'
  assert helper.config_path(d).read_bytes()==before
  records.append({'profile':d['profile'],'agent_version':d['agent_version'],'registration':registration,'read':read,'server':server.get('serverInfo'),'target':{k:target[k] for k in ['name','username','createdAt','state']},'identity_stdout':identity,'stdout':out,'exit_code':code,'wrong_expected_identity_stops_before_hello':mismatch_stopped,'config_preserved':True,'execution_via':'selected strict MCP transport; actual Agent registration verified separately'})
 result={'result':'passed','os':'Windows','python':sys.version.split()[0],'profiles':records,'model_inference':'not_requested','gui':'prior separate same-version observations SRC070/071; not synthesized by this check'}
 Path(a.result).write_text(json.dumps(result,indent=2),encoding='utf-8');print(json.dumps({'result':'passed','profiles':len(records),'stdout':'hello world\n','exit_code':0,'model_inference':'not_requested'}))
if __name__=='__main__':main()
