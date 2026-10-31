"""Explicit infrastructure fixture for scoped Tenant and port state machines."""
from copy import deepcopy
from datetime import datetime,timezone
import hashlib
import json
from dataclasses import replace
from unittest.mock import patch
from pathlib import Path
import sys
from test_slice import Base,TARGET,HASH,PASSWORD
from test_lifecycle import LifecycleFixture,NAME
from bwmini.domain import Fault
from bwmini.service import Service
from bwmini.store import Store
from bwmini.web import create_app
from bwmini.access_provider import Wire,guarded,policy_value,unchanged_fields
from bwmini.mcp import MCP

PROFILE={'id':'hello-tcp','logical':TARGET.logical,'external_port':18849,'target_port':8080,'protocol':'tcp','audience':'private test only','approved':True}
CHANGE={'destination':'tenant-b','add_sources':['tenant-a'],'remove_sources':[]}
TABLES=('access_claims','access_events','access_operations','access_plans','access_routes','tenant_labels')

class FakeWire:
    def __init__(self,provider,now):
        self.provider=provider;self.now=now;self.policies={};self.routes_by_key={};self.writes=[];self.health=True;self.support=True;self.lost=False;self.ip='10.9.0.2';self.target=TARGET;self.active=True
        self.cap={'contract':'native-network-tenant/v1','backendScope':'local-lxc','nativeMembershipRead':True,'sharedTenantCreate':True,'atomicTenantAccess':True,'networkPolicyWrite':True,'_atomic_tools':True,'_shared_field':True}
    def capabilities(self):
        if not self.support:raise Fault('NETWORK_CONTRACT_UNSUPPORTED')
        return dict(self.cap)
    def guard(self):
        t=datetime.fromtimestamp(self.now[0],timezone.utc).isoformat()
        v={'mode':'GUARD_MODE_ENFORCE' if self.health else 'GUARD_MODE_OFF','firewallDriver':'nftables','unsupported':False,'lastPass':t,'lastError':'','staleSince':'','entries':[],'unresolved':[],'unmanaged':[]}
        return {'core':dict(v),'tenant':dict(v)}
    def policy(self,key):
        p=self.policies.get(key)
        return {'policy':deepcopy(p),'revision':hashlib.sha256(json.dumps(p,sort_keys=True).encode()).hexdigest() if p is not None else 'absent'}
    def patch(self,changes):
        for c in changes:
            if self.policy(c['destination'])['revision']!=c['expected_revision']:raise Fault('UPSTREAM_REVISION_CONFLICT')
        self.writes.append(('patch',deepcopy(changes)))
        for c in changes:
            p=self.policies.setdefault(c['destination'],{'tenant':c['destination'],'allowIntraTenant':True,'mode':'NETWORK_POLICY_MODE_ENFORCE'})
            allowed=set(p.get('allowFromTenants',[]));allowed.update(c['add_sources']);allowed.difference_update(c['remove_sources']);p['allowFromTenants']=sorted(allowed)
        if self.lost:raise OSError('synthetic dropped response')
        return {'policies':[self.policy(c['destination']) for c in changes],'application_state':'TENANT_ACCESS_APPLICATION_STATE_GUARD_VERIFIED' if self.health else 'TENANT_ACCESS_APPLICATION_STATE_DEGRADED','connectivity_tested':False}
    def container(self,logical):
        if logical==TARGET.logical:return {'target':self.target,'ip':self.ip,'state':'CONTAINER_STATE_RUNNING','network_tenant':'tenant-a','membership_source':'NETWORK_TENANT_SOURCE_NAME','account_user':TARGET.user}
        row=self.provider.resources.get(logical)
        return dict(row,ip='10.9.0.3',account_user=row['target'].user,membership_source='NETWORK_TENANT_SOURCE_EXPLICIT') if row else None
    def routes(self):return deepcopy(list(self.routes_by_key.values()))
    def route_mutate(self,action,route):
        self.writes.append((action,deepcopy(route)));key=(route['protocol'],route['external_port'])
        if action=='publish':self.routes_by_key[key]=dict(route,active=self.active)
        else:del self.routes_by_key[key]
        if self.lost:raise OSError('synthetic lost acknowledgement')

class AccessFixture(LifecycleFixture):
    def __init__(self,now):
        super().__init__();self.config.update(provider='containarium',backend='dev',project='default',access={'tenants':['tenant-a','tenant-b'],'port_profiles':[dict(PROFILE)],'guard_max_age':60});self.access_wire=FakeWire(self,now)
    def lifecycle_mutate(self,action,logical,spec,timeout):
        super().lifecycle_mutate(action,logical,spec,timeout)
        if action=='create' and spec.get('_network_tenant'):self.resources[logical]['network_tenant']=spec['_network_tenant']

class AccessTests(Base):
    def setUp(self):
        super().setUp();self.provider=AccessFixture(self.now);self.s=Service(self.store,self.provider,clock=lambda:self.now[0]);self.a=self.s.access;self.w=self.provider.access_wire
    def confirmed(self,plan):return self.a.confirm(plan['id'],plan['confirmation'])
    def port(self,action='publish'):
        op=self.confirmed(self.a.port_plan('hello-tcp',action));self.s.worker(once=True);return self.a.receipt(op['id'])
    def network(self,changes=None):
        op=self.confirmed(self.a.network_plan(changes or [deepcopy(CHANGE)]));self.s.worker(once=True);return self.a.receipt(op['id'])
    def test_metadata_rename_does_not_change_native_membership_or_policy(self):
        self.a.label('tenant-a','Friendly name',0);self.assertEqual(self.a.settings()['tenants'][0]['label'],'Friendly name');self.assertEqual(self.w.writes,[])
        self.assertEqual(self.w.container(TARGET.logical)['network_tenant'],'tenant-a')
        with self.assertRaisesRegex(Fault,'REVISION_CONFLICT'):self.a.label('tenant-a','stale',0)
    def test_network_directional_and_atomic_two_direction_preserves_other_fields(self):
        self.w.policies['tenant-b']={'tenant':'tenant-b','allowIntraTenant':False,'egressCidrs':['1.2.3.4/32'],'denyRules':[{'cidr':'9.0.0.0/8'}],'egressMode':'NETWORK_POLICY_MODE_LOG_ONLY','allowMetadata':False,'allowFromTenants':[]}
        before=unchanged_fields(deepcopy(self.w.policies['tenant-b']))
        second={'destination':'tenant-a','add_sources':['tenant-b'],'remove_sources':[]}
        r=self.network([deepcopy(CHANGE),second]);self.assertEqual(r['state'],'succeeded')
        self.assertTrue(r['result']['configured']);self.assertTrue(r['result']['guard_observed']);self.assertFalse(r['result']['connectivity_tested'])
        self.assertEqual(unchanged_fields(self.w.policies['tenant-b']),before);self.assertEqual(len(self.w.writes),1)
        self.assertEqual(self.w.policies['tenant-a']['allowFromTenants'],['tenant-b'])
    def test_stale_one_of_atomic_destinations_stops_all_before_write(self):
        changes=[deepcopy(CHANGE),{'destination':'tenant-a','add_sources':['tenant-b'],'remove_sources':[]}]
        p=self.a.network_plan(changes);op=self.confirmed(p)
        self.w.policies['tenant-a']={'tenant':'tenant-a','allowFromTenants':[]}
        self.s.worker(once=True);self.assertEqual(self.a.receipt(op['id'])['code'],'POLICY_CHANGED');self.assertEqual(self.w.writes,[])
        self.assertNotIn('tenant-b',self.w.policies)
    def test_guard_or_capability_failure_blocks_grant_and_shared_create(self):
        self.w.health=False
        for action in [lambda:self.a.network_plan([CHANGE]),lambda:self.s.lifecycle.plan('create',NAME,network_tenant='tenant-a')]:
            with self.assertRaisesRegex(Fault,'GUARD_NOT_READY'):action()
        self.w.health=True;self.w.cap['_atomic_tools']=False
        with self.assertRaisesRegex(Fault,'NETWORK_WRITE_UNSUPPORTED'):self.a.network_plan([CHANGE])
        # Shared capability is independent of atomic policy writes.
        self.s.lifecycle.plan('create',NAME,network_tenant='tenant-a')
        self.w.cap['_shared_field']=False
        with self.assertRaisesRegex(Fault,'SHARED_CREATE_UNSUPPORTED'):self.s.lifecycle.plan('create',NAME,network_tenant='tenant-a')
        self.assertEqual(self.w.writes,[]);self.assertEqual(self.provider.writes,[])
    def test_revoke_degraded_remains_configured_pending_guard_then_observes(self):
        self.network();self.w.health=False
        r=self.network([{'destination':'tenant-b','add_sources':[],'remove_sources':['tenant-a']}])
        self.assertEqual(r['state'],'observing');self.assertTrue(r['result']['configured']);self.assertFalse(r['result']['guard_observed'])
        count=len(self.w.writes);self.w.health=True;self.now[0]+=16;self.s.worker(once=True)
        self.assertEqual(self.a.receipt(r['id'])['state'],'succeeded');self.assertEqual(len(self.w.writes),count)
    def test_lost_network_ack_unknown_never_replays_or_promotes(self):
        self.w.lost=True;r=self.network();self.assertEqual(r['state'],'unknown')
        r=self.a.observe(r['id']);self.assertEqual(r['state'],'unknown');self.assertEqual(r['result']['retry'],'not_authorized')
        self.now[0]+=30;self.s.worker(once=True);self.assertEqual(len(self.w.writes),1)
        p=self.a.network_plan([CHANGE])
        with self.assertRaisesRegex(Fault,'ACCESS_OPERATION_PENDING'):self.confirmed(p)
    def test_publish_unpublish_exact_ownership_history_restart_no_replay(self):
        r=self.port();self.assertEqual(r['state'],'succeeded');self.assertFalse(r['result']['connectivity_tested'])
        r=self.port('unpublish');self.assertEqual(r['state'],'succeeded');self.assertEqual(self.w.routes(),[])
        again=Service(Store(self.store.path),self.provider,clock=lambda:self.now[0]);again.worker(once=True)
        self.assertEqual(len(again.access.receipts()),2);self.assertEqual(len(self.w.writes),2)
    def test_port_collision_replaced_ip_target_and_foreign_delete_stop(self):
        with self.assertRaisesRegex(Fault,'MANAGED_ROUTE_REQUIRED'):self.a.port_plan('hello-tcp','unpublish')
        p=self.a.port_plan('hello-tcp','publish');op=self.confirmed(p);self.w.ip='10.9.0.3';self.s.worker(once=True)
        self.assertEqual(self.a.receipt(op['id'])['code'],'TARGET_CHANGED');self.assertEqual(self.w.writes,[])
        self.w.ip='10.9.0.2';self.port();key=('tcp',18849);self.w.routes_by_key[key]['description']='foreign replacement'
        with self.assertRaisesRegex(Fault,'ROUTE_CHANGED'):self.a.port_plan('hello-tcp','unpublish')
        with self.assertRaisesRegex(Fault,'ROUTE_EXISTS'):self.a.port_plan('hello-tcp','publish')
    def test_unapproved_exposure_and_wrong_backend_fail_before_write(self):
        self.provider.config['access']['port_profiles'][0]['approved']=False
        with self.assertRaisesRegex(Fault,'EXPOSURE_NOT_APPROVED'):self.a.port_plan('hello-tcp','publish')
        self.provider.config['access']['port_profiles'][0]['approved']=True
        p=self.a.port_plan('hello-tcp','publish');op=self.confirmed(p);self.provider.config['backend']='other'
        self.s.worker(once=True);self.assertEqual(self.a.receipt(op['id'])['code'],'CONFIG_CHANGED');self.assertEqual(self.w.writes,[])
    def test_lost_publish_and_crash_do_not_adopt_recreate_or_remove_candidate(self):
        self.w.lost=True;r=self.port();self.assertEqual(r['state'],'unknown');self.a.observe(r['id']);self.s.worker(once=True)
        self.assertEqual(len(self.w.writes),1)
        with self.assertRaisesRegex(Fault,'MANAGED_ROUTE_REQUIRED'):self.a.port_plan('hello-tcp','unpublish')
    def test_process_loss_before_ack_and_saved_ack_recovery(self):
        op=self.confirmed(self.a.port_plan('hello-tcp','publish'))
        with self.store.db() as db:db.execute("UPDATE access_operations SET state='dispatching' WHERE id=?",(op['id'],))
        self.s.worker(once=True);self.assertEqual(self.a.receipt(op['id'])['state'],'unknown');self.assertEqual(self.w.writes,[])
    def test_saved_ack_resumes_observation_only_and_preserves_deadline(self):
        self.w.active=False;op=self.port();self.assertEqual(op['state'],'observing');deadline=op['deadline'];count=len(self.w.writes)
        with self.store.db() as db:db.execute("UPDATE access_operations SET state='dispatching' WHERE id=?",(op['id'],))
        self.s.worker(once=True);self.assertEqual(len(self.w.writes),count)
        self.now[0]=deadline+1;self.s.worker(once=True);self.assertEqual(self.a.receipt(op['id'])['state'],'unknown');self.assertEqual(self.a.receipt(op['id'])['deadline'],deadline)
    def test_actor_wrong_confirmation_used_plan_and_unknown_scope(self):
        p=self.a.port_plan('hello-tcp','publish')
        with self.assertRaisesRegex(Fault,'NOT_FOUND'):self.a.confirm(p['id'],p['confirmation'],'other')
        with self.assertRaisesRegex(Fault,'EXACT_CONFIRMATION'):self.a.confirm(p['id'],'wrong')
        op=self.confirmed(p)
        with self.assertRaisesRegex(Fault,'PLAN_EXPIRED_OR_USED'):self.confirmed(p)
        with self.assertRaisesRegex(Fault,'NOT_FOUND'):self.a.receipt(op['id'],'other')
        with self.store.db() as db:db.execute("UPDATE access_operations SET state='unknown' WHERE id=?",(op['id'],))
        self.provider.config['project']='other';self.assertEqual(self.a.observe(op['id'])['code'],'CONFIG_CHANGED');self.assertEqual(self.w.writes,[])
    def test_shared_create_sets_membership_and_keeps_account_user_separate(self):
        p=self.s.lifecycle.plan('create',NAME,network_tenant='tenant-a');op=self.s.lifecycle.confirm(p['id'],NAME);self.s.worker(once=True)
        r=self.s.lifecycle.receipt(op['id']);self.assertEqual(r['state'],'succeeded');self.assertEqual(r['result']['network_tenant'],'tenant-a');self.assertEqual(r['result']['account_user'],NAME)
    def test_shared_create_observed_membership_mismatch_stops(self):
        p=self.s.lifecycle.plan('create',NAME,network_tenant='tenant-a');op=self.s.lifecycle.confirm(p['id'],NAME)
        original=self.provider.lifecycle_mutate
        def wrong(*args):original(*args);self.provider.resources[NAME]['network_tenant']='tenant-b'
        self.provider.lifecycle_mutate=wrong;self.s.worker(once=True)
        self.assertEqual(self.s.lifecycle.receipt(op['id'])['code'],'NETWORK_MEMBERSHIP_MISMATCH')
    def test_http_auth_csrf_exact_fields_and_override_rejection(self):
        self.store.admin(HASH);c=create_app(self.s,'http://localhost',self.root,secure=False,clock=lambda:self.now[0]).test_client()
        self.assertEqual(c.get('/api/access').status_code,401)
        csrf=c.post('/api/login',json={'password':PASSWORD},headers={'Origin':'http://localhost'}).json['csrf'];h={'Origin':'http://localhost','X-CSRF-Token':csrf}
        self.assertEqual(c.post('/api/access-plans/port',json={'profile':'hello-tcp','action':'publish'},headers={'Origin':'http://localhost'}).status_code,403)
        self.assertEqual(c.post('/api/access-plans/port',json={'profile':'hello-tcp','action':'publish','host':'override'},headers=h).status_code,400)
        r=c.post('/api/access-plans/network',json={'changes':[CHANGE]},headers=h);self.assertEqual(r.status_code,201)
        self.assertEqual(self.w.writes,[])

    def test_real_sqlite_ack_save_failure_recovers_unknown_without_replay(self):
        p=self.a.port_plan('hello-tcp','publish');op=self.confirmed(p)
        with self.store.db() as db:db.executescript("CREATE TRIGGER access_ack_failure BEFORE UPDATE OF ack ON access_operations WHEN NEW.ack IS NOT NULL BEGIN SELECT RAISE(ABORT,'synthetic ack failure'); END;")
        with self.assertRaisesRegex(Fault,'STATE_UNAVAILABLE'):self.s.worker(once=True)
        self.assertEqual(len(self.w.writes),1)
        with self.store.db() as db:db.execute('DROP TRIGGER access_ack_failure')
        restarted=Service(Store(self.store.path),self.provider,clock=lambda:self.now[0]);restarted.worker(once=True)
        self.assertEqual(restarted.access.receipt(op['id'])['state'],'unknown');self.assertEqual(len(self.w.writes),1)
        self.assertEqual(restarted.access.observe(op['id'])['state'],'unknown')
    def test_schema_five_migration_preserves_model_intent_and_projects(self):
        from bwmini.models import Models
        project=self.bind();Models(self.store).save({'client_version':'3.16.3','provider':'zai','model':'GLM-5.3','access_mode':'native-existing'},0)
        with self.store.db() as db:
            for table in TABLES:db.execute('DROP TABLE '+table)
            db.execute('DROP TABLE project_unbindings');db.execute('DROP TABLE job_runs');db.execute('PRAGMA user_version=5')
        upgraded=Store(self.store.path)
        self.assertEqual(upgraded.project(project['id'])['revision'],1)
        self.assertEqual(Models(upgraded).read()['selection']['model'],'GLM-5.3')
        with upgraded.db() as db:self.assertEqual(db.execute('PRAGMA user_version').fetchone()[0],8)
    def test_membership_read_keeps_account_and_network_identity_distinct(self):
        rows=self.a.memberships()['memberships'];self.assertEqual(rows[0]['account_user'],TARGET.user);self.assertEqual(rows[0]['network_tenant'],'tenant-a')

class WireTests(Base):
    def provider(self):
        from bwmini.provider import Provider
        return Provider({'provider':'containarium','backend':'dev','project':'default','launcher':[sys.executable],'targets':{TARGET.logical:TARGET.data()}})
    def session(self,values):
        class Session:
            calls=[]
            def __enter__(self):return self
            def __exit__(self,*args):pass
            def discover(self):
                props={k:{} for k in ['external_port','protocol','target_ip','target_port','container_name','description','changes','network_tenant']}
                return {k:{'inputSchema':{'properties':props}} for k in ['network_tenant_capabilities','network_guard_status','get_network_policy','patch_tenant_access','create_container','list_passthrough_routes','add_passthrough_route','delete_passthrough_route']}
            def tool(self,name,args):
                self.calls.append((name,args));value=values[name]
                if isinstance(value,Exception):raise value
                return {'content':[{'type':'text','text':json.dumps(value)}]}
        return Session()
    def test_capabilities_require_exact_native_local_contract_but_keep_flags_independent(self):
        p=self.provider();session=self.session({'network_tenant_capabilities':{'contract':'native-network-tenant/v1','backendScope':'local-lxc','nativeMembershipRead':True,'sharedTenantCreate':True,'atomicTenantAccess':False,'networkPolicyWrite':False}});p.session=lambda *a:session
        c=Wire(p).capabilities();self.assertTrue(c['sharedTenantCreate']);self.assertFalse(c['atomicTenantAccess']);self.assertTrue(c['_shared_field'])
        for cap in [{},dict(c,contract='legacy'),dict(c,backendScope='remote'),dict(c,nativeMembershipRead=1)]:
            session.tool=lambda *a:{'content':[{'type':'text','text':json.dumps(cap)}]}
            with self.assertRaisesRegex(Fault,'NETWORK_CONTRACT_UNSUPPORTED'):Wire(p).capabilities()
    def test_routes_complete_count_duplicate_protocol_and_boolean_defaults(self):
        p=self.provider();valid={'externalPort':18849,'targetIp':'10.9.0.2','targetPort':8080,'protocol':'ROUTE_PROTOCOL_TCP','containerName':'fixture-box-container','description':'owned'}
        for value in [{'routes':[],'totalCount':0},{'routes':[valid],'totalCount':1}]:
            session=self.session({'list_passthrough_routes':value});p.session=lambda *a:session
            rows=Wire(p).routes()
            if rows:self.assertIs(rows[0]['active'],False)
        for value in [{},{'routes':[],'totalCount':1},{'routes':[],'totalCount':False},{'routes':[valid,valid],'totalCount':2},{'routes':[dict(valid,protocol='tcp')],'totalCount':1},{'routes':[dict(valid,targetPort='8080')],'totalCount':1},{'routes':[],'totalCount':0,'nextCursor':'more'}]:
            session=self.session({'list_passthrough_routes':value});p.session=lambda *a:session
            with self.assertRaisesRegex(Fault,'ROUTE_LIST_UNSUPPORTED'):Wire(p).routes()
    def test_policy_requires_full_revision_and_exact_tenant_preserves_zero_defaults(self):
        good={'policy':{'tenant':'tenant-a','allowFromTenants':['tenant-b']},'revision':'a'*64}
        self.assertEqual(policy_value(good,'tenant-a')['revision'],'a'*64)
        for value in [dict(good,revision='absent'),dict(good,policy={}),dict(good,policy={'tenant':'tenant-a','allowFromTenants':[{}]}),dict(good,policy={'tenant':'tenant-a','allowFromTenants':['tenant-b','tenant-b']})]:
            with self.assertRaises(Fault):policy_value(value,'tenant-a')
        self.assertEqual(unchanged_fields({'tenant':'tenant-a'}),unchanged_fields({'tenant':'tenant-a','allowMetadata':False,'allowIntraTenant':False,'egressCidrs':[],'allowFromTenants':[]}))
    def test_guard_requires_fresh_enforce_no_future_naive_or_errors(self):
        wire=FakeWire(None,self.now);g=wire.guard();self.assertTrue(guarded(g,1000,60))
        for change in [{'lastPass':'1970-01-01T00:00:00Z'},{'lastPass':'1970-01-01T00:17:00Z'},{'lastPass':'1970-01-01T00:16:40'},{'lastError':'error'},{'unsupported':True},{'unsupported':0},{'mode':'GUARD_MODE_OFF'},{'entries':[{'attached':False}]}]:
            bad=deepcopy(g);bad['tenant'].update(change);self.assertFalse(guarded(bad,1000,60))
    def test_only_exact_get_policy_404_maps_to_absence(self):
        prefix='Tool execution failed: API error (status 404): '
        message=prefix+json.dumps({'code':404,'error':'rpc error: code = NotFound desc = no network policy for tenant "tenant-a"'})
        for tool,key,expected in [('get_network_policy','tenant-a','POLICY_ABSENT'),('get_network_policy','tenant-b','PROVIDER_REJECTED'),('get_container','tenant-a','PROVIDER_REJECTED')]:
            reply={'jsonrpc':'2.0','id':1,'error':{'code':-32603,'message':message}}
            code='import json,sys;json.loads(sys.stdin.readline());print('+repr(json.dumps(reply))+',flush=True)'
            with MCP([sys.executable,'-c',code],3) as m,self.assertRaisesRegex(Fault,expected):m.call('tools/call',{'name':tool,'arguments':{'tenant':key}})
    def test_route_mutation_uses_exact_protocol_and_metadata_no_force_or_host_override(self):
        p=self.provider();session=self.session({'add_passthrough_route':{},'delete_passthrough_route':{}});p.session=lambda *a:session
        wire=Wire(p);route={'external_port':18849,'protocol':'udp','target_ip':'10.9.0.2','target_port':8080,'container':'fixture-box-container','description':'boxweave-v2:nonce'}
        wire.route_mutate('publish',route);wire.route_mutate('unpublish',route)
        self.assertEqual(session.calls[-1],('delete_passthrough_route',{'external_port':18849,'protocol':'udp'}))
        self.assertEqual(session.calls[-2][1]['target_ip'],'10.9.0.2');self.assertNotIn('force',session.calls[-2][1])
    def test_shared_create_passes_native_key_and_never_encryption_tenant(self):
        from test_lifecycle import PUBLIC
        p=self.provider();calls=[]
        class Session:
            def __init__(self,*a):pass
            def __enter__(self):return self
            def __exit__(self,*a):pass
            def discover(self):return {'create_container':{'inputSchema':{'properties':{k:{} for k in ['username','cpu','memory','disk','image','ssh_keys','enable_podman','network_tenant']}}}}
            def tool(self,name,args):calls.append((name,args))
        with patch('bwmini.provider.MCP',Session):p.lifecycle_mutate('create','approved-new',{'cpu':'1','memory':'1GB','disk':'8GB','ssh_public_key':PUBLIC,'_network_tenant':'tenant-a'},60)
        args=calls[0][1];self.assertEqual(args['network_tenant'],'tenant-a');self.assertNotIn('tenant_id',args);self.assertEqual(args['ssh_keys'],[PUBLIC]);self.assertFalse(args['enable_podman'])
