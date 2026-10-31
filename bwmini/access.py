"""Exact network/port intents with one dispatch; no generic provider proxy."""
import hashlib
import json
import re
import time
import uuid
from . import diagnostics
from .domain import Fault,Target,name
from .access_provider import Wire,tenant,guarded,unchanged_fields

SCHEMA="""
CREATE TABLE access_plans(id TEXT PRIMARY KEY,actor TEXT NOT NULL,payload TEXT NOT NULL,expires REAL NOT NULL,used INTEGER NOT NULL);
CREATE TABLE access_operations(id TEXT PRIMARY KEY,actor TEXT NOT NULL,payload TEXT NOT NULL,state TEXT NOT NULL,code TEXT NOT NULL,created REAL NOT NULL,deadline REAL NOT NULL,next_poll REAL NOT NULL,ack TEXT,result TEXT);
CREATE TABLE access_claims(resource TEXT PRIMARY KEY,operation TEXT NOT NULL REFERENCES access_operations(id));
CREATE TABLE access_events(operation TEXT NOT NULL REFERENCES access_operations(id),at REAL NOT NULL,state TEXT NOT NULL,code TEXT NOT NULL);
CREATE TABLE access_routes(resource TEXT PRIMARY KEY,route TEXT NOT NULL,target TEXT NOT NULL,deleted INTEGER NOT NULL);
CREATE TABLE tenant_labels(tenant TEXT PRIMARY KEY,label TEXT NOT NULL,revision INTEGER NOT NULL);
"""
def uid():return uuid.uuid4().hex
def encode(value):return json.dumps(value,sort_keys=True)

def validate_config(c):
    if c is None:return
    if not isinstance(c,dict) or set(c)!={'tenants','port_profiles','guard_max_age'}:raise Fault('ACCESS_CONFIG_INVALID',422)
    tenants=c['tenants'];ports=c['port_profiles']
    if not isinstance(tenants,list) or len(tenants)>16 or any(not isinstance(v,str) for v in tenants) or len(set(tenants))!=len(tenants):raise Fault('ACCESS_CONFIG_INVALID',422)
    for key in tenants:tenant(key)
    if type(c['guard_max_age']) is not int or not 1<=c['guard_max_age']<=120:raise Fault('ACCESS_CONFIG_INVALID',422)
    if not isinstance(ports,list) or len(ports)>16:raise Fault('ACCESS_CONFIG_INVALID',422)
    ids=set();keys=set()
    for p in ports:
        if not isinstance(p,dict) or set(p)!={'id','logical','external_port','target_port','protocol','audience','approved'}:raise Fault('ACCESS_CONFIG_INVALID',422)
        if not isinstance(p['id'],str) or re.fullmatch(r'[a-z][a-z0-9-]{0,31}',p['id']) is None or p['id'] in ids:raise Fault('ACCESS_CONFIG_INVALID',422)
        if not isinstance(p['logical'],str) or re.fullmatch(r'[a-z][a-z0-9-]{0,62}',p['logical']) is None or p['logical'].startswith('containarium-core-'):raise Fault('ACCESS_CONFIG_INVALID',422)
        if p['protocol'] not in ('tcp','udp') or any(type(p[k]) is not int or not 1<=p[k]<=65535 for k in ('external_port','target_port')) or type(p['approved']) is not bool:raise Fault('ACCESS_CONFIG_INVALID',422)
        name(p['audience']);key=(p['protocol'],p['external_port'])
        if key in keys:raise Fault('ACCESS_CONFIG_INVALID',422)
        keys.add(key);ids.add(p['id'])

class Access:
    def __init__(self,service):
        self.s=service;self.store=service.store;self.provider=service.provider;self.clock=service.clock
        self.config=vars(self.provider).get('config',{}).get('access');validate_config(self.config)
        self.wire=vars(self.provider).get('access_wire') or Wire(self.provider)
    def scope(self):return hashlib.sha256(encode({'transport':self.s.lifecycle.scope(),'access':self.config}).encode()).hexdigest()
    def actor(self,value):
        if not isinstance(value,str) or re.fullmatch(r'[A-Za-z0-9_-]{1,64}',value) is None:raise Fault('INVALID_ACTOR',422)
    def enabled(self):
        if self.config is None:raise Fault('ACCESS_DISABLED')
    def allowed_tenant(self,key):
        self.enabled();tenant(key)
        if key not in self.config['tenants']:raise Fault('TENANT_NOT_ALLOWED',403)
    def network_ready(self,grant=False,shared=False):
        self.enabled();cap=self.wire.capabilities()
        if shared:
            if cap.get('sharedTenantCreate') is not True or cap.get('_shared_field') is not True:raise Fault('SHARED_CREATE_UNSUPPORTED')
        elif cap.get('atomicTenantAccess') is not True or cap.get('networkPolicyWrite') is not True or cap.get('_atomic_tools') is not True:raise Fault('NETWORK_WRITE_UNSUPPORTED')
        guard=self.wire.guard();healthy=guarded(guard,self.clock(),self.config['guard_max_age'])
        if (grant or shared) and not healthy:raise Fault('GUARD_NOT_READY')
        return healthy
    def shared_check(self,key):self.allowed_tenant(key);self.network_ready(shared=True)
    def settings(self):
        if self.config is None:return {'enabled':False,'tenants':[],'port_profiles':[],'network':{'supported':False,'code':'ACCESS_DISABLED'}}
        network={'supported':False,'code':'NO_TENANTS_CONFIGURED'}
        if self.config['tenants']:
            try:
                cap=self.wire.capabilities();health=guarded(self.wire.guard(),self.clock(),self.config['guard_max_age'])
                network={'supported':True,'contract':cap['contract'],'shared_create':cap.get('sharedTenantCreate') is True and cap.get('_shared_field') is True,'atomic_write':cap.get('atomicTenantAccess') is True and cap.get('networkPolicyWrite') is True and cap.get('_atomic_tools') is True,'guard_observed':health,'packet_tested':False}
            except Fault as e:network={'supported':False,'code':e.code}
        with self.store.db() as db:labels={r['tenant']:dict(r) for r in db.execute('SELECT * FROM tenant_labels')}
        return {'enabled':True,'network':network,'tenants':[{'key':key,'label':labels.get(key,{}).get('label',key),'revision':labels.get(key,{}).get('revision',0)} for key in self.config['tenants']],
                'port_profiles':[dict(p) for p in self.config['port_profiles']],'source_filtering':False,'tls':False,'connectivity_tested':False}
    def memberships(self):
        self.enabled();self.wire.capabilities()
        names=set(vars(self.provider).get('config',{}).get('targets',{}))
        with self.store.db() as db:names.update(r[0] for r in db.execute('SELECT logical FROM managed_environments WHERE deleted=0'))
        if len(names)>32:raise Fault('MEMBERSHIP_SCOPE_TOO_LARGE')
        rows=[]
        for logical in sorted(names):
            try:
                target,value=self.target(logical)
                if value.get('network_tenant') is None:raise Fault('MEMBERSHIP_UNAVAILABLE')
                rows.append({'logical':logical,'instance':target.instance,'account_user':value['account_user'],'network_tenant':value['network_tenant'],'membership_source':value.get('membership_source'),'state':value['state']})
            except Fault as e:rows.append({'logical':logical,'code':e.code,'membership':'unverified'})
        return {'memberships':rows,'existing_membership_editable':False}

    def label(self,key,label,revision):
        self.allowed_tenant(key);label=name(label)
        if type(revision) is not int or revision<0:raise Fault('INVALID_REVISION',422)
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE');row=db.execute('SELECT * FROM tenant_labels WHERE tenant=?',(key,)).fetchone()
            if revision!=(row['revision'] if row else 0):raise Fault('REVISION_CONFLICT')
            if not row:db.execute('INSERT INTO tenant_labels VALUES(?,?,1)',(key,label))
            elif row['label']!=label:db.execute('UPDATE tenant_labels SET label=?,revision=revision+1 WHERE tenant=?',(label,key))
        return {'tenant':key,'membership_changed':False,'enforcement_changed':False,'label':label}
    def target(self,logical,allow_absent=False):
        config=vars(self.provider).get('config',{});expected=config.get('targets',{}).get(logical)
        if expected:expected=Target(**expected)
        else:expected=self.s.lifecycle.managed(logical)
        value=self.wire.container(logical)
        if value is None and allow_absent:return expected,None
        if value is None or value['target']!=expected:raise Fault('TARGET_CHANGED')
        return expected,value
    def profile(self,key):
        self.enabled();rows=[p for p in self.config['port_profiles'] if p['id']==key]
        if len(rows)!=1:raise Fault('PORT_PROFILE_NOT_ALLOWED',403)
        return rows[0]
    def resource(self,route):return 'port:'+route['protocol']+':'+str(route['external_port'])
    def find_route(self,route):
        rows=[r for r in self.wire.routes() if (r['protocol'],r['external_port'])==(route['protocol'],route['external_port'])]
        if len(rows)>1:raise Fault('ROUTE_LIST_UNSUPPORTED')
        return rows[0] if rows else None
    def same_route(self,a,b):return {k:v for k,v in a.items() if k!='active'}=={k:v for k,v in b.items() if k!='active'}
    def network_plan(self,changes,actor='operator'):
        self.actor(actor);self.enabled()
        if not isinstance(changes,list) or not 1<=len(changes)<=2:raise Fault('INVALID_NETWORK_CHANGE',422)
        destinations=set();clean=[];before=[]
        for change in changes:
            if not isinstance(change,dict) or set(change)!={'destination','add_sources','remove_sources'}:raise Fault('INVALID_NETWORK_CHANGE',422)
            destination=change['destination'];self.allowed_tenant(destination)
            if destination in destinations:raise Fault('INVALID_NETWORK_CHANGE',422)
            destinations.add(destination)
            for k in ('add_sources','remove_sources'):
                values=change[k]
                if not isinstance(values,list) or len(values)>16 or any(not isinstance(x,str) for x in values) or len(set(values))!=len(values):raise Fault('INVALID_NETWORK_CHANGE',422)
                for source in values:
                    self.allowed_tenant(source)
                    if source==destination:raise Fault('INVALID_NETWORK_CHANGE',422)
            if set(change['add_sources'])&set(change['remove_sources']) or not (change['add_sources'] or change['remove_sources']):raise Fault('INVALID_NETWORK_CHANGE',422)
            clean.append({'destination':destination,'add_sources':sorted(change['add_sources']),'remove_sources':sorted(change['remove_sources'])})
        self.network_ready(grant=any(c['add_sources'] for c in clean))
        for change in clean:
            value=self.wire.policy(change['destination']);change['expected_revision']=value['revision'];before.append(value)
        payload={'kind':'network','action':'patch','changes':clean,'before':before,'scope':self.scope()}
        return self.plan(payload,actor)
    def port_plan(self,profile,action,actor='operator'):
        self.actor(actor);p=self.profile(profile)
        if action not in ('publish','unpublish'):raise Fault('ACTION_NOT_ALLOWED',400)
        if action=='publish' and not p['approved']:raise Fault('EXPOSURE_NOT_APPROVED',403)
        target,value=self.target(p['logical'],allow_absent=action=='unpublish')
        route={'protocol':p['protocol'],'external_port':p['external_port'],'target_port':p['target_port'],'target_ip':value['ip'] if value else '', 'container':target.instance,'description':''}
        resource=self.resource(route)
        with self.store.db() as db:owned=db.execute('SELECT * FROM access_routes WHERE resource=?',(resource,)).fetchone()
        current=self.find_route(route)
        if action=='publish':
            if value['state']!='CONTAINER_STATE_RUNNING':raise Fault('NOT_CONNECTION_READY')
            if current is not None:raise Fault('ROUTE_EXISTS')
            if owned and not owned['deleted']:raise Fault('MANAGED_ROUTE_MISSING_REVIEW_REQUIRED')
            route['description']='boxweave-v2:'+uid()
        else:
            if not owned or owned['deleted']:raise Fault('MANAGED_ROUTE_REQUIRED',403)
            expected=json.loads(owned['route']);pinned=Target(**json.loads(owned['target']))
            if pinned!=target or current is None or not self.same_route(current,expected):raise Fault('ROUTE_CHANGED')
            if value and value['ip']!=expected['target_ip']:raise Fault('TARGET_CHANGED')
            route=expected
        return self.plan({'kind':'port','action':action,'profile':profile,'target':target.data(),'route':route,'audience':p['audience'],'scope':self.scope()},actor)
    def summary(self,payload):
        if payload['kind']=='network':return {'kind':'network','action':'patch','changes':payload['changes'],'connectivity_tested':False}
        return {'kind':'port','action':payload['action'],'profile':payload['profile'],'target':payload['target'],'route':{k:v for k,v in payload['route'].items() if k!='description'},'audience':payload['audience'],'source_filtering':False,'tls':False,'connectivity_tested':False}
    def phrase(self,payload):return ('confirm network '+','.join(sorted(c['destination'] for c in payload['changes']))) if payload['kind']=='network' else payload['action']+' '+payload['profile']
    def plan(self,payload,actor):
        ident=uid();expires=self.clock()+300
        with self.store.db() as db:db.execute('INSERT INTO access_plans VALUES(?,?,?,?,0)',(ident,actor,encode(payload),expires))
        return {'id':ident,'expires':expires,'confirmation':self.phrase(payload),'summary':self.summary(payload),'scope':{k:vars(self.provider).get('config',{}).get(k) for k in ('provider','backend','project')}}
    def preflight(self,payload):
        if payload['scope']!=self.scope():raise Fault('CONFIG_CHANGED')
        if payload['kind']=='network':
            for change in payload['changes']:self.allowed_tenant(change['destination'])
            self.network_ready(grant=any(c['add_sources'] for c in payload['changes']))
            for change in payload['changes']:
                if self.wire.policy(change['destination'])['revision']!=change['expected_revision']:raise Fault('POLICY_CHANGED')
        else:
            profile=self.profile(payload['profile'])
            if payload['action']=='publish' and not profile['approved']:raise Fault('EXPOSURE_NOT_APPROVED',403)
            target,value=self.target(profile['logical'],allow_absent=payload['action']=='unpublish')
            if target.data()!=payload['target'] or (value and value['ip']!=payload['route']['target_ip']):raise Fault('TARGET_CHANGED')
            current=self.find_route(payload['route'])
            if payload['action']=='publish':
                if not value or value['state']!='CONTAINER_STATE_RUNNING':raise Fault('NOT_CONNECTION_READY')
                if current is not None:raise Fault('ROUTE_EXISTS')
            elif current is None or not self.same_route(current,payload['route']):raise Fault('ROUTE_CHANGED')
    def confirm(self,ident,confirmation,actor='operator'):
        self.actor(actor)
        with self.store.db() as db:row=db.execute('SELECT * FROM access_plans WHERE id=? AND actor=?',(ident,actor)).fetchone()
        if not row:raise Fault('NOT_FOUND',404)
        payload=json.loads(row['payload'])
        if confirmation!=self.phrase(payload):raise Fault('EXACT_CONFIRMATION_REQUIRED',422)
        if row['used'] or self.clock()>=row['expires']:raise Fault('PLAN_EXPIRED_OR_USED')
        self.preflight(payload);operation=uid();now=self.clock()
        resources=['tenant:'+c['destination'] for c in payload['changes']] if payload['kind']=='network' else [self.resource(payload['route'])]
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE')
            if not db.execute('UPDATE access_plans SET used=1 WHERE id=? AND actor=? AND used=0 AND expires>?',(ident,actor,now)).rowcount:raise Fault('PLAN_EXPIRED_OR_USED')
            for resource in resources:
                if db.execute('SELECT 1 FROM access_claims WHERE resource=?',(resource,)).fetchone():raise Fault('ACCESS_OPERATION_PENDING')
            db.execute('INSERT INTO access_operations VALUES(?,?,?,?,?,?,?,?,NULL,NULL)',(operation,actor,encode(payload),'queued','QUEUED',now,now+600,now))
            for resource in resources:db.execute('INSERT INTO access_claims VALUES(?,?)',(resource,operation))
            diagnostics.link(db,actor,'access',operation)
        return self.receipt(operation,actor)
    def receipt(self,ident,actor='operator'):
        self.actor(actor)
        with self.store.db() as db:row=db.execute('SELECT * FROM access_operations WHERE id=? AND actor=?',(ident,actor)).fetchone()
        if not row:raise Fault('NOT_FOUND',404)
        return {'id':row['id'],'state':row['state'],'code':row['code'],'created':row['created'],'deadline':row['deadline'],'summary':self.summary(json.loads(row['payload'])),'result':json.loads(row['result']) if row['result'] else None,'evidence':'live_mcp' if self.provider.mode=='mcp' else 'fixture'}
    def receipts(self,actor='operator'):
        self.actor(actor)
        with self.store.db() as db:ids=[r[0] for r in db.execute('SELECT id FROM access_operations WHERE actor=? ORDER BY created DESC,id DESC LIMIT 100',(actor,))]
        return [self.receipt(i,actor) for i in ids]
    def save(self,job,state,code,ack=None,result=None):
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE');db.execute('UPDATE access_operations SET state=?,code=?,next_poll=?,ack=COALESCE(?,ack),result=? WHERE id=?',(state,code,self.clock()+15,encode(ack) if ack is not None else None,encode(result) if result is not None else None,job['id']))
            db.execute('INSERT INTO access_events VALUES(?,?,?,?)',(job['id'],self.clock(),state,code))
            if state in ('succeeded','failed'):db.execute('DELETE FROM access_claims WHERE operation=?',(job['id'],))
            if state=='succeeded' and json.loads(job['payload'])['kind']=='port':
                p=json.loads(job['payload']);db.execute('INSERT INTO access_routes VALUES(?,?,?,?) ON CONFLICT(resource) DO UPDATE SET route=excluded.route,target=excluded.target,deleted=excluded.deleted',(self.resource(p['route']),encode(p['route']),encode(p['target']),int(p['action']=='unpublish')))
    def recover(self):
        with self.store.db() as db:
            rows=[dict(r) for r in db.execute("SELECT id,actor,ack FROM access_operations WHERE state='dispatching'")]
            db.execute("UPDATE access_operations SET state=CASE WHEN ack IS NULL THEN 'unknown' ELSE 'observing' END,code=CASE WHEN ack IS NULL THEN 'OUTCOME_UNKNOWN' ELSE 'ACKNOWLEDGED' END WHERE state='dispatching'")
        for job in rows:
            trace = diagnostics.trace_for_operation(self.store, job['actor'], 'access', job['id']) \
                or diagnostics.new_trace_id()
            acknowledged = job['ack'] is not None
            diagnostics.emit(self.store, component='access', phase='recovered', event_type='recovered',
                             outcome='unknown', code='ACKNOWLEDGED' if acknowledged else 'OUTCOME_UNKNOWN',
                             trace_id=trace, actor_scope=job['actor'], operation_kind='access',
                             operation_id=job['id'])
    def observe(self,ident,actor='operator'):
        self.actor(actor)
        with self.store.db() as db:row=db.execute('SELECT * FROM access_operations WHERE id=? AND actor=?',(ident,actor)).fetchone()
        if not row:raise Fault('NOT_FOUND',404)
        job=dict(row);p=json.loads(job['payload'])
        if job['state'] not in ('observing','unknown'):return self.receipt(ident,actor)
        if p['scope']!=self.scope():self.save(job,'unknown','CONFIG_CHANGED');return self.receipt(ident,actor)
        try:
            if p['kind']=='port':
                current=self.find_route(p['route']);result={'route_state':'present' if current else 'absent','connectivity_tested':False}
                desired=(current is None) if p['action']=='unpublish' else current is not None and self.same_route(current,p['route']) and current['active']
                if current and not self.same_route(current,p['route']):raise Fault('ROUTE_CHANGED')
            else:
                policies=[self.wire.policy(c['destination']) for c in p['changes']];result={'configured':False,'guard_observed':False,'connectivity_tested':False}
                desired=True
                for change,before,after in zip(p['changes'],p['before'],policies):
                    policy=after['policy']
                    sources=set((before['policy'] or {}).get('allowFromTenants',[]));sources.update(change['add_sources']);sources.difference_update(change['remove_sources'])
                    if policy is None or set(policy.get('allowFromTenants',[]))!=sources:desired=False
                    if before['policy'] is not None and policy is not None and unchanged_fields(before['policy'])!=unchanged_fields(policy):raise Fault('POLICY_CHANGED')
                result['configured']=desired
                if desired:
                    cap=self.wire.capabilities();guard=self.wire.guard();health=guarded(guard,self.clock(),self.config['guard_max_age'])
                    ack=json.loads(job['ack']) if job['ack'] else {}
                    after_commit=all(datetime_stamp(guard.get(k,{}).get('lastPass',''))>=ack.get('at',float('inf')) for k in ('core','tenant'))
                    result['guard_observed']=health and (ack.get('response',{}).get('application_state')=='TENANT_ACCESS_APPLICATION_STATE_GUARD_VERIFIED' or after_commit)
                    desired=desired and result['guard_observed']
            if job['state']=='unknown' or not job['ack']:
                result.update(completion='unverified',retry='not_authorized');self.save(job,'unknown','OUTCOME_UNKNOWN',result=result)
            elif self.clock()<job['created'] or self.clock()>=job['deadline']:self.save(job,'unknown','DEADLINE_EXCEEDED',result=result)
            else:self.save(job,'succeeded' if desired else 'observing','OK' if desired else 'GUARD_OR_ROUTE_PENDING',result=result)
        except Fault as e:
            if e.code=='STATE_UNAVAILABLE':raise
            state='failed' if e.code in ('TRUST_FAILED','ROUTE_CHANGED','POLICY_CHANGED') and job['state']!='unknown' else 'unknown' if job['state']=='unknown' or self.clock()>=job['deadline'] else 'observing'
            self.save(job,state,e.code,result={'completion':'unverified','connectivity_tested':False})
        return self.receipt(ident,actor)
    def work_one(self):
        with self.store.db() as db:
            db.execute('BEGIN IMMEDIATE');row=db.execute("SELECT * FROM access_operations WHERE state='queued' OR (state='observing' AND next_poll<=?) ORDER BY created,id LIMIT 1",(self.clock(),)).fetchone()
            if not row:return False
            job=dict(row);execute=job['state']=='queued'
            if execute:db.execute("UPDATE access_operations SET state='dispatching' WHERE id=?",(job['id'],))
        trace = diagnostics.trace_for_operation(self.store, job['actor'], 'access', job['id']) \
            or diagnostics.new_trace_id()
        if not execute:self.observe(job['id'],job['actor']);return True
        started = time.monotonic()
        diagnostics.emit(self.store, component='access', phase='claimed', event_type='claimed',
                         outcome='passed', code='CLAIMED', trace_id=trace, actor_scope=job['actor'],
                         operation_kind='access', operation_id=job['id'])
        try:
            if self.clock()<job['created'] or self.clock()>=job['deadline']:raise Fault('DEADLINE_EXCEEDED')
            p=json.loads(job['payload']);self.preflight(p)
            if self.clock()<job['created'] or self.clock()>=job['deadline']:raise Fault('DEADLINE_EXCEEDED')
        except Fault as e:
            if e.code=='STATE_UNAVAILABLE':raise
            self.save(job,'failed',e.code);return True
        try:
            response=self.wire.patch(p['changes']) if p['kind']=='network' else self.wire.route_mutate(p['action'],p['route'])
            self.save(job,'observing','ACKNOWLEDGED',ack={'at':self.clock(),'response':response})
            diagnostics.emit(self.store, component='access', phase='submitted', event_type='acknowledged',
                             outcome='passed', code='ACKNOWLEDGED', trace_id=trace, actor_scope=job['actor'],
                             operation_kind='access', operation_id=job['id'],
                             elapsed_ms=int((time.monotonic()-started)*1000))
        except Exception as e:
            if isinstance(e,Fault) and e.code=='STATE_UNAVAILABLE':raise
            self.save(job,'unknown','OUTCOME_UNKNOWN')
            diagnostics.emit(self.store, component='access', phase='submitted', event_type='unknown',
                             outcome='unknown', code=diagnostics.error_class(e), trace_id=trace,
                             actor_scope=job['actor'], operation_kind='access', operation_id=job['id'],
                             elapsed_ms=int((time.monotonic()-started)*1000))
            return True
        self.observe(job['id'],job['actor']);return True

def datetime_stamp(value):
    from datetime import datetime
    try:
        d=datetime.fromisoformat(value.replace('Z','+00:00'))
        return d.timestamp() if d.tzinfo is not None else float('-inf')
    except (AttributeError,ValueError,TypeError):return float('-inf')
