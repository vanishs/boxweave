"""Approved new-lab Web/worker/MCP/packet tests. Never silently adopt/retry unknown writes."""
import argparse
import base64
import json
import os
from pathlib import Path
import secrets
import signal
import socket
import subprocess
import sys
import time
import urllib.request
from playwright.sync_api import sync_playwright,expect
from werkzeug.security import generate_password_hash
from bwmini.__main__ import load
from bwmini.domain import Fault
from bwmini.provider import execution
from bwmini.mcp import MCP

NAMES=['v2-net-a','v2-net-peer','v2-net-b']
A,PEER,B=NAMES

class ObservedMCP(MCP):
    def call(self,method,params):
        result=super().call(method,params)
        if method=='initialize':self.initialize_result=result
        return result

def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--config',required=True);parser.add_argument('--approved-access-lab',action='store_true');args=parser.parse_args()
    path=Path(args.config).resolve();cfg=json.loads(path.read_text());root=Path(__file__).resolve().parents[1]
    assert args.approved_access_lab and cfg['mode']=='mcp' and cfg['host']=='127.0.0.1'
    assert cfg['backend']=='bw-v2-access-lab' and cfg['lifecycle']['allowed_names']==NAMES and cfg['access']['tenants']==[A,B]
    assert all(p['external_port']==18849 and p['target_port']==18849 and p['logical']==A and p['approved'] is True for p in cfg['access']['port_profiles'])
    assert sorted(p['protocol'] for p in cfg['access']['port_profiles'])==['tcp','udp']
    assert not Path(cfg['database']).exists(),'Existing state must be reconciled, never blindly rerun'
    c,s=load(path);provider=s.provider;wire=s.access.wire
    assert all(provider.lifecycle_inspect(n) is None for n in NAMES),'Original logical name exists, no create/retry'
    assert wire.routes()==[],'Do not modify existing routes'
    assert s.access.network_ready(grant=True)
    with ObservedMCP(provider.argv,30) as m:
        tools=m.discover();info=m.initialize_result.get('serverInfo',{});assert info.get('version')=='0.120.1'
        m.tool('list_containers',{});backend={'server':info,'tools':len(tools),'protocol':m.initialize_result['protocolVersion'],'read':'list_containers'}
    password=secrets.token_urlsafe(32);s.store.admin(generate_password_hash(password));progress=path.with_name('progress.json')
    records={'backend':backend,'lifecycle':[],'access':[],'packets':[],'identities':[],'cleanup':{},'hello':None}
    phase='starting';listeners={};web=worker=None
    def persist():path.with_name('receipts.json').write_text(json.dumps(records,indent=2))
    def report(**values):progress.write_text(json.dumps({'phase':phase,**values}));persist()
    def start(action):return subprocess.Popen([sys.executable,'-m','bwmini',action,'--config',str(path)],cwd=root,stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
    def stop(process):
        if process and process.poll() is None:os.killpg(process.pid,signal.SIGTERM);process.wait(timeout=10)
    def ready():
        deadline=time.monotonic()+30
        while True:
            try:urllib.request.urlopen(c['origin'],timeout=1).close();return
            except OSError:
                if time.monotonic()>deadline:raise
                time.sleep(.2)
    def inner(logical,code):
        assert logical in NAMES
        target=s.lifecycle.managed(logical);identity=provider.lifecycle_identity(target,30);records['identities'].append(identity)
        with provider.session(30) as m:
            tools=m.discover();props=tools['connect']['inputSchema']['properties'];key='box' if 'box' in props else 'username'
            exit_code,out=execution(m.tool('connect',{key:logical,'exec':"python3 - <<'PY'\n"+code+"\nPY"}))
        assert exit_code==0, 'Remote test helper failed; retain resource for review'
        value=json.loads(out);persist();return value
    def launch_listener(logical,ports):
        nonce=secrets.token_hex(12)
        script="""import selectors,socket,time
sel=selectors.DefaultSelector();sockets=[]
for port,kind in PORTS:
 s=socket.socket(socket.AF_INET,socket.SOCK_STREAM if kind=='tcp' else socket.SOCK_DGRAM);s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('0.0.0.0',port))
 if kind=='tcp':s.listen(8)
 s.setblocking(False);sockets.append(s);sel.register(s,selectors.EVENT_READ,kind)
end=time.monotonic()+1200
try:
 while time.monotonic()<end:
  for key,event in sel.select(min(1,max(0,end-time.monotonic()))):
   if key.data=='tcp':
    c,a=key.fileobj.accept();c.settimeout(2)
    try:
     if c.recv(128)==b'probe':c.sendall(NONCE.encode())
    finally:c.close()
   else:
    b,a=key.fileobj.recvfrom(128)
    if b==b'probe':key.fileobj.sendto(NONCE.encode(),a)
finally:
 sel.close()
 for s in sockets:s.close()
"""
        script='PORTS='+repr(ports)+'\nNONCE='+repr(nonce)+'\n'+script
        encoded=base64.b64encode(script.encode()).decode()
        code="import os,json,base64,subprocess,getpass,socket\nfrom pathlib import Path\nassert getpass.getuser()=="+repr(logical)+" and socket.gethostname()=="+repr(logical+'-container')+"\np=Path.home()/"+repr('.boxweave-access-'+nonce)+";assert not p.exists();p.mkdir(mode=0o700)\nf=p/'listener.py';f.write_bytes(base64.b64decode("+repr(encoded)+"));f.chmod(0o600)\nlog=(p/'listener.log').open('xb');os.chmod(log.name,0o600)\nr=subprocess.Popen(['/usr/bin/python3',str(f)],stdin=subprocess.DEVNULL,stdout=log,stderr=log,start_new_session=True);log.close()\n(p/'pid').write_text(str(r.pid));print(json.dumps({'pid':r.pid,'path':str(p),'nonce':"+repr(nonce)+"}))"
        value=inner(logical,code);listeners[logical]=value
        return value
    def stop_listener(logical):
        item=listeners[logical]
        code="import os,signal,json\nfrom pathlib import Path\np=Path("+repr(item['path'])+");assert p.parent==Path.home() and p.name=="+repr('.boxweave-access-'+item['nonce'])+"\npid=int((p/'pid').read_text());proc=Path('/proc')/str(pid)\nif proc.exists():\n cmd=(proc/'cmdline').read_bytes().split(bytes([0]));assert str(p/'listener.py').encode() in cmd and proc.stat().st_uid==os.getuid() and os.getpgid(pid)==pid;os.killpg(pid,signal.SIGTERM)\nprint(json.dumps({'stopped_owned_listener':True}))"
        assert inner(logical,code)['stopped_owned_listener']
    def inner_probe(source,destination,expected):
        value=wire.container(destination);assert value and value['target']==s.lifecycle.managed(destination)
        nonce=listeners[destination]['nonce'];ip=value['ip']
        code="import socket,json\nr={'reachable':False}\ntry:\n with socket.create_connection(("+repr(ip)+",18761),timeout=3) as c:\n  c.settimeout(3);c.sendall(b'probe');r={'reachable':c.recv(128).decode()=="+repr(nonce)+"}\nexcept OSError as e:r['error_type']=type(e).__name__\nprint(json.dumps(r))"
        result=inner(source,code);assert result['reachable'] is expected,(source,destination,result)
        records['packets'].append({'from':source,'to':destination,'tcp_port':18761,'reachable':result['reachable'],'expected':expected});persist()
    def external_probe(protocol,expected):
        ip=cfg['test_backend_ip'];nonce=listeners[A]['nonce'];result={'reachable':False}
        try:
            if protocol=='tcp':
                with socket.create_connection((ip,18849),timeout=3) as q:q.settimeout(3);q.sendall(b'probe');result['reachable']=q.recv(128).decode()==nonce
            else:
                with socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as q:q.settimeout(3);q.sendto(b'probe',(ip,18849));value,address=q.recvfrom(128);result['reachable']=value.decode()==nonce and address==(ip,18849)
        except OSError as error:result['error_type']=type(error).__name__
        assert result['reachable'] is expected,(protocol,result)
        records['packets'].append({'from':'same-outer-Tenant app','to':'new nested backend','protocol':protocol,'external_port':18849,'reachable':result['reachable'],'expected':expected});persist()
    try:
        web=start('serve');worker=start('worker');ready()
        with sync_playwright() as pw:
            browser=pw.chromium.launch();page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda error:errors.append(str(error)))
            page.goto(c['origin']);page.get_by_label('Password',exact=True).fill(password);page.get_by_role('button',name='Sign in',exact=True).click()
            def lifecycle(action,logical,membership=''):
                nonlocal phase
                phase=action+' '+logical;report(state='previewing')
                page.get_by_role('button',name='Environments',exact=True).click();page.get_by_label('Authorized new environment').select_option(logical)
                page.get_by_label('New environment network Tenant',exact=True).select_option(membership)
                page.get_by_role('button',name='Preview '+action,exact=True).click()
                field=page.get_by_label('Type the exact environment name',exact=True);field.wait_for(timeout=90000);field.fill(logical)
                with page.expect_response(lambda r:'/api/environment-plans/' in r.url and r.url.endswith('/confirm') and r.request.method=='POST',timeout=90000) as response:page.get_by_role('button',name='Confirm '+action,exact=True).click()
                assert response.value.status==202;oid=response.value.json()['id'];deadline=time.monotonic()+cfg['lifecycle']['readiness_timeout']+10
                while True:
                    r=page.request.get(c['origin']+'/api/environment-operations/'+oid,timeout=60000).json();report(state=r['state'],code=r['code'],operation=oid)
                    if r['state'] in ('succeeded','failed','unknown'):break
                    if time.monotonic()>deadline:raise TimeoutError('Original lifecycle deadline ended')
                    page.wait_for_timeout(1000)
                records['lifecycle'].append(r);persist();assert r['state']=='succeeded',r['code']
                page.locator('.environment-panel .receipt small').filter(has_text=oid).wait_for(timeout=60000)
                page.get_by_test_id('environment-state').filter(has_text='succeeded').wait_for(timeout=60000)
                return r
            def access(action,profile='hello-tcp',both=True,source=A,destination=B):
                nonlocal phase
                phase='access '+action;report(state='previewing');page.get_by_role('button',name='Network & ports',exact=True).click()
                if action in ('grant','revoke'):
                    page.get_by_label('Source network Tenant',exact=True).select_option(source);page.get_by_label('Destination network Tenant',exact=True).select_option(destination)
                    page.get_by_label('Both directions in one atomic change',exact=True).set_checked(both)
                else:page.get_by_label('Port profile',exact=True).select_option(profile)
                kind='network' if action in ('grant','revoke') else 'port'
                with page.expect_response(lambda r:r.url.endswith('/api/access-plans/'+kind) and r.request.method=='POST',timeout=90000) as response:page.get_by_role('button',name='Preview '+action,exact=True).click()
                assert response.value.status==201;plan=response.value.json();page.get_by_label('Exact access confirmation',exact=True).fill(plan['confirmation'])
                with page.expect_response(lambda r:'/api/access-plans/' in r.url and r.url.endswith('/confirm') and r.request.method=='POST',timeout=90000) as response:page.get_by_role('button',name='Confirm access change',exact=True).click()
                assert response.value.status==202;oid=response.value.json()['id'];deadline=time.monotonic()+610
                while True:
                    r=page.request.get(c['origin']+'/api/access-operations/'+oid,timeout=60000).json();report(state=r['state'],code=r['code'],operation=oid)
                    if r['state'] in ('succeeded','failed','unknown'):break
                    if time.monotonic()>deadline:raise TimeoutError('Original access deadline ended')
                    page.wait_for_timeout(1000)
                records['access'].append(r);persist();assert r['state']=='succeeded',r['code'];return r
            lifecycle('create',A);launch_listener(A,[(18761,'tcp'),(18849,'tcp'),(18849,'udp')]);inner_probe(A,A,True)
            lifecycle('create',PEER,A);inner_probe(PEER,A,True)
            assert wire.container(PEER)['network_tenant']==A and wire.container(PEER)['account_user']==PEER
            lifecycle('stop',PEER);lifecycle('delete',PEER);assert provider.lifecycle_inspect(PEER) is None
            lifecycle('create',B);launch_listener(B,[(18761,'tcp')]);inner_probe(B,B,True);inner_probe(B,A,False);inner_probe(A,B,False)
            access('grant');inner_probe(A,B,True);inner_probe(B,A,True)
            # Fresh Web service preview becomes stale after a separate authorized single-direction edit.
            stale=s.access.network_plan([{'destination':A,'add_sources':[],'remove_sources':[B]},{'destination':B,'add_sources':[],'remove_sources':[A]}])
            old_a=wire.policy(A);access('revoke',both=False,source=B,destination=A)
            before={key:wire.policy(key) for key in (A,B)}
            try:s.access.confirm(stale['id'],stale['confirmation'])
            except Fault as error:assert error.code=='POLICY_CHANGED'
            else:raise AssertionError('Stale application plan accepted')
            assert {key:wire.policy(key) for key in (A,B)}==before
            # Independent real upstream atomic-CAS rejection; no mutation replay on a transport failure.
            changes=[{'destination':A,'expected_revision':old_a['revision'],'add_sources':[B],'remove_sources':[]},{'destination':B,'expected_revision':before[B]['revision'],'add_sources':[],'remove_sources':[A]}]
            with provider.session() as m:
                m.discover()
                try:m.tool('patch_tenant_access',{'changes':changes})
                except Fault as error:assert error.code=='PROVIDER_REJECTED'
                else:raise AssertionError('Stale native atomic batch accepted')
            assert {key:wire.policy(key) for key in (A,B)}==before;records['atomic_conflict_no_partial_write']=True
            access('grant');access('revoke');inner_probe(A,B,False);inner_probe(B,A,False)
            for protocol in ('tcp','udp'):external_probe(protocol,False);access('publish',profile='hello-'+protocol);external_probe(protocol,True)
            for protocol in ('tcp','udp'):access('unpublish',profile='hello-'+protocol);external_probe(protocol,False)
            inner_probe(A,A,True);assert wire.routes()==[]
            page.get_by_role('button',name='Projects',exact=True).click();page.get_by_label('Reviewed environment').select_option(A,timeout=60000);page.get_by_label('Project name',exact=True).fill('Access lab project');page.get_by_role('button',name='Save binding',exact=True).click()
            page.get_by_role('button',name='Preview hello',exact=True).first.click();page.get_by_role('button',name='Confirm hello',exact=True).click();page.get_by_test_id('operation-state').filter(has_text='succeeded').wait_for(timeout=120000)
            page.get_by_text('evidence: live_mcp',exact=False).wait_for();records['hello']=json.loads(s.list_receipts()[0]['result']);assert records['hello']['stdout']=='hello world\n' and records['hello']['exit_code']==0
            stop_listener(A);stop_listener(B)
            for logical in (B,A):lifecycle('stop',logical);lifecycle('delete',logical);assert provider.lifecycle_inspect(logical) is None
            assert wire.routes()==[]
            policies={key:wire.policy(key) for key in (A,B)};assert all(v['policy'].get('allowFromTenants',[])==[] for v in policies.values())
            records['cleanup']={'workloads_absent':NAMES,'test_routes_absent':True,'listeners_stopped':True,'empty_ingress_policy_records_retained':[A,B]};persist()
            with s.store.db() as db:counts=[db.execute('SELECT count(*) FROM '+table).fetchone()[0] for table in ['environment_operations','access_operations','audit']]
            stop(worker);worker=start('worker');stop(web);web=start('serve');ready();page.reload();page.get_by_role('button',name='Network & ports',exact=True).click();page.get_by_label('Access receipts',exact=True).get_by_role('button',name='Open access receipt',exact=False).first.wait_for(timeout=60000)
            with s.store.db() as db:assert counts==[db.execute('SELECT count(*) FROM '+table).fetchone()[0] for table in ['environment_operations','access_operations','audit']]
            records['restart_no_replay']=True;assert not errors;browser.close()
        records['result']='passed';path.with_name('result.json').write_text(json.dumps(records,indent=2));phase='completed';report(state='passed')
        print(json.dumps({'result':'passed','evidence':'live_mcp_and_packets','lifecycle_operations':len(records['lifecycle']),'access_operations':len(records['access']),'packet_checks':len(records['packets']),'cleanup':records['cleanup'],'restart_no_replay':True}))
    finally:stop(worker);stop(web)
if __name__=='__main__':main()
