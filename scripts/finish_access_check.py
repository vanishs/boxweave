"""Finish only the approved access-lab observation and cleanup; never retry a write."""
import json,os,signal,socket,subprocess,sys,time
from pathlib import Path
from bwmini.__main__ import load
from bwmini.provider import execution

def main():
 r=Path(sys.argv[1]).resolve();c,s=load(r/'network-3.json');p=s.provider
 assert c['backend']=='bw-v2-access-lab' and c['test_backend_ip']=='10.206.30.9'
 expected={'v2-net-a':'1793254450','v2-net-b':'1793254620'}
 for n,created in expected.items():
  t=p.observe(n);assert t.created==created;p.lifecycle_identity(t,30)
 assert all(x['state']=='succeeded' for x in s.access.receipts())
 resuming=len(sys.argv)==3 and sys.argv[2]=='--resume-observation'
 assert len(s.access.receipts())==(7 if resuming else 6)
 result={'initial_udp_immediate_probe':'timeout; retained in original run log','packets':[],'unpublish':[],'cleanup':[]}
 if resuming:
  result=json.loads((r/'completion.json').read_text());assert len(result['unpublish'])==1 and result['unpublish'][0]['summary']['profile']=='hello-tcp' and result['unpublish'][0]['state']=='succeeded';result['initial_tcp_unpublish_probe']='immediate packet remained reachable; original completion.log retained'
 def save(): (r/'completion.json').write_text(json.dumps(result,indent=2))
 def inner(n,code):
  p.lifecycle_identity(p.observe(n),30)
  with p.session() as m:
   m.discover();ec,out=execution(m.tool('connect',{'box':n,'exec':"python3 - <<'PY2'\n"+code+"\nPY2"}))
  assert ec==0;return json.loads(out)
 active=inner('v2-net-a',"from pathlib import Path\nimport json\na=[]\nfor d in Path.home().glob('.boxweave-access-*'):\n p=Path('/proc')/(d/'pid').read_text()\n if p.exists() and str(d/'listener.py').encode() in (p/'cmdline').read_bytes().split(bytes([0])):a.append(d.name.rsplit('-',1)[-1])\nprint(json.dumps(a))")
 assert len(active)==1;nonce=active[0]
 def probe(protocol,wanted):
  deadline=time.monotonic()+30;attempts=[]
  while True:
   yes=False
   try:
    with socket.socket(socket.AF_INET,socket.SOCK_STREAM if protocol=='tcp' else socket.SOCK_DGRAM) as q:
     q.settimeout(3);q.connect((c['test_backend_ip'],18849));q.send(b'probe');yes=q.recv(128).decode()==nonce
   except OSError as e: attempts.append(type(e).__name__)
   if yes==wanted:break
   if time.monotonic()>=deadline:raise AssertionError((protocol,wanted,attempts))
   time.sleep(1)
  result['packets'].append({'protocol':protocol,'reachable':yes,'expected':wanted,'prior_observation_errors':attempts});save()
 if not resuming:probe('tcp',True);probe('udp',True)
 worker=None
 def start():
  return subprocess.Popen([sys.executable,'-m','bwmini','worker','--config',str(r/'network-3.json')],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
 def stop(q):
  if q and q.poll() is None:os.killpg(q.pid,signal.SIGTERM);q.wait(timeout=10)
 try:
  worker=start()
  for protocol in ('tcp','udp'):
   if resuming and protocol=='tcp':
    probe(protocol,False);continue
   plan=s.access.port_plan('hello-'+protocol,'unpublish');op=s.access.confirm(plan['id'],plan['confirmation'])
   deadline=time.monotonic()+610
   while True:
    v=s.access.receipt(op['id'])
    if v['state'] in ('succeeded','failed','unknown'):break
    assert time.monotonic()<deadline;time.sleep(1)
   result['unpublish'].append(v);save();assert v['state']=='succeeded';probe(protocol,False)
  assert s.access.wire.routes()==[]
  b=s.bind('Access completion','v2-net-a','/home/v2-net-a');v=s.plan(b['id'],b['revision']);op=s.confirm(v['id'])
  deadline=time.monotonic()+65
  while True:
   row=s.store.receipt(op['id'],'operator')
   if row['state'] in ('succeeded','failed','unknown'):break
   assert time.monotonic()<deadline;time.sleep(.5)
  assert row['state']=='succeeded';result['hello']=json.loads(row['result']);assert result['hello']['stdout']=='hello world\n' and result['hello']['exit_code']==0;save()
  before=[(x['id'],x['state']) for x in s.access.receipts()];stop(worker);worker=start();time.sleep(2)
  assert before==[(x['id'],x['state']) for x in s.access.receipts()];result['restart_no_replay']=True;save()
 finally:stop(worker)
 for n in expected:
  cleaned=inner(n,"import json,os,signal,re\nfrom pathlib import Path\na=[]\nfor d in Path.home().glob('.boxweave-access-*'):\n assert re.fullmatch(r'[.]boxweave-access-[0-9a-f]{24}',d.name) and not d.is_symlink()\n pid=int((d/'pid').read_text());p=Path('/proc')/str(pid)\n if p.exists() and (p/'cmdline').read_bytes():\n  assert p.stat().st_uid==os.getuid() and str(d/'listener.py').encode() in (p/'cmdline').read_bytes().split(bytes([0])) and os.getpgid(pid)==pid;os.killpg(pid,signal.SIGTERM)\n a.append(str(d))\nprint(json.dumps({'owned_listener_cleanup':a}))")
  t=p.observe(n);assert t.created==expected[n]
  for action,desired in [('stop','CONTAINER_STATE_STOPPED'),('delete',None)]:
   current=p.lifecycle_inspect(n);assert current and current['target']==t
   if action=='stop':assert current['state']=='CONTAINER_STATE_RUNNING'
   else:assert current['state']=='CONTAINER_STATE_STOPPED'
   p.lifecycle_mutate(action,n,{},120)
   deadline=time.monotonic()+120
   while True:
    current=p.lifecycle_inspect(n)
    if desired is None and current is None:break
    if current and current['target']==t and current['state']==desired:break
    assert current is None or current['target']==t
    assert time.monotonic()<deadline;time.sleep(2)
   result['cleanup'].append({'logical':n,'created':t.created,'action':action,'via':'independent authenticated MCP; no lifecycle adoption','observed':desired or 'absent'});save()
 assert p.lifecycle_inspect('v2-net-peer') is None
 result['empty_policy_records_retained']=['v2-net-a','v2-net-b']
 assert all(not (s.access.wire.policy(n)['policy'] or {}).get('allowFromTenants',[]) for n in expected)
 assert not s.access.wire.routes()
 result['result']='passed';save();print(json.dumps({'result':'passed','packets':result['packets'],'cleanup':result['cleanup'],'restart_no_replay':True,'hello':result['hello']}))
if __name__=='__main__':main()
