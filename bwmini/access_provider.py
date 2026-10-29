"""Narrow observed Containarium native-network/v1 and passthrough wire contracts."""
from datetime import datetime,timezone
import ipaddress
import json
import re
from .domain import Fault,Target
from .provider import text

KEY=re.compile(r'[a-z0-9][a-z0-9._-]{0,127}\Z')
def tenant(value):
    if not isinstance(value,str) or not KEY.fullmatch(value) or value.startswith('containarium-core-'):raise Fault('TENANT_NOT_ALLOWED',403)
    return value

def json_object(pairs):
    out={}
    for key,value in pairs:
        if key in out:raise Fault('ACCESS_CONTRACT_UNSUPPORTED')
        out[key]=value
    return out

def read_json(m,name,args):
    try:value=json.loads(text(m.tool(name,args)),object_pairs_hook=json_object)
    except (ValueError,TypeError,KeyError):raise Fault('ACCESS_CONTRACT_UNSUPPORTED') from None
    if not isinstance(value,dict):raise Fault('ACCESS_CONTRACT_UNSUPPORTED')
    return value

def guarded(guard,now,max_age):
    try:
        for key in ('core','tenant'):
            value=guard[key]
            if value['mode']!='GUARD_MODE_ENFORCE' or value['unsupported'] is not False or value.get('firewallDriver')!='nftables':return False
            if value.get('lastError') or value.get('staleSince') or value.get('unresolved') or value.get('unmanaged'):return False
            stamp=datetime.fromisoformat(value['lastPass'].replace('Z','+00:00'))
            if stamp.tzinfo is None:return False
            at=stamp.timestamp()
            if not -5<=now-at<=max_age:return False
            if not isinstance(value['entries'],list) or any(e.get('attached') is not True or e.get('lastError') for e in value['entries']):return False
        return True
    except (KeyError,TypeError,ValueError,AttributeError):return False

def policy_value(value,key):
    if not isinstance(value,dict) or set(value)!={'policy','revision'}:raise Fault('POLICY_CONTRACT_UNSUPPORTED')
    policy=value['policy'];revision=value['revision']
    if not isinstance(policy,dict) or policy.get('tenant')!=key or not isinstance(revision,str) or re.fullmatch(r'[a-f0-9]{64}',revision) is None:raise Fault('POLICY_CONTRACT_UNSUPPORTED')
    sources=policy.get('allowFromTenants',[])
    if not isinstance(sources,list) or any(not isinstance(v,str) for v in sources) or len(sources)>256 or len(sources)!=len(set(sources)) or any(tenant(v)==key for v in sources):raise Fault('POLICY_CONTRACT_UNSUPPORTED')
    for field in ('allowIntraTenant','allowMetadata'):
        if field in policy and type(policy[field]) is not bool:raise Fault('POLICY_CONTRACT_UNSUPPORTED')
    return {'policy':policy,'revision':revision}

def unchanged_fields(policy):
    if policy is None:return None
    value=dict(policy);value.pop('allowFromTenants',None)
    defaults={'allowIntraTenant':False,'allowMetadata':False,'egressCidrs':[],'egressDomains':[],'denyRules':[],
              'source':'','mode':'NETWORK_POLICY_MODE_UNSPECIFIED','egressPreset':'EGRESS_PRESET_UNSPECIFIED','egressMode':'NETWORK_POLICY_MODE_UNSPECIFIED'}
    for k,v in defaults.items():value.setdefault(k,v)
    return value

class Wire:
    def __init__(self,provider):self.provider=provider
    def capabilities(self):
        with self.provider.session() as m:
            tools=m.discover()
            required={'network_tenant_capabilities','network_guard_status'}
            if not required.issubset(tools):raise Fault('NETWORK_CONTRACT_UNSUPPORTED')
            cap=read_json(m,'network_tenant_capabilities',{})
            if cap.get('contract')!='native-network-tenant/v1' or cap.get('backendScope')!='local-lxc' or cap.get('nativeMembershipRead') is not True:raise Fault('NETWORK_CONTRACT_UNSUPPORTED')
            cap['_atomic_tools']=all(k in tools for k in ('get_network_policy','patch_tenant_access')) and 'changes' in tools.get('patch_tenant_access',{}).get('inputSchema',{}).get('properties',{})
            cap['_shared_field']='network_tenant' in tools.get('create_container',{}).get('inputSchema',{}).get('properties',{})
            return cap
    def guard(self):
        with self.provider.session() as m:
            tools=m.discover()
            if 'network_guard_status' not in tools:raise Fault('GUARD_UNAVAILABLE')
            return read_json(m,'network_guard_status',{})
    def policy(self,key):
        tenant(key)
        with self.provider.session() as m:
            m.discover()
            try:value=read_json(m,'get_network_policy',{'tenant':key})
            except Fault as e:
                if e.code=='POLICY_ABSENT':return {'policy':None,'revision':'absent'}
                raise
        return policy_value(value,key)
    def patch(self,changes):
        with self.provider.session() as m:
            tools=m.discover();props=tools.get('patch_tenant_access',{}).get('inputSchema',{}).get('properties',{})
            if 'changes' not in props:raise Fault('NETWORK_CONTRACT_UNSUPPORTED')
            result=read_json(m,'patch_tenant_access',{'changes':changes})
        policies=result.get('policies');state=result.get('applicationState','TENANT_ACCESS_APPLICATION_STATE_UNSPECIFIED')
        if not isinstance(policies,list) or len(policies)!=len(changes) or type(result.get('connectivityTested',False)) is not bool:raise Fault('POLICY_RESULT_UNSUPPORTED')
        wanted={c['destination'] for c in changes};seen=set();rows=[]
        for value in policies:
            key=value.get('policy',{}).get('tenant') if isinstance(value,dict) else None
            if key not in wanted or key in seen:raise Fault('POLICY_RESULT_UNSUPPORTED')
            seen.add(key);rows.append(policy_value(value,key))
        if state not in ['TENANT_ACCESS_APPLICATION_STATE_UNSPECIFIED','TENANT_ACCESS_APPLICATION_STATE_PENDING','TENANT_ACCESS_APPLICATION_STATE_GUARD_VERIFIED','TENANT_ACCESS_APPLICATION_STATE_DEGRADED']:raise Fault('POLICY_RESULT_UNSUPPORTED')
        return {'policies':rows,'application_state':state,'connectivity_tested':False}
    def container(self,logical):
        # Full inventory avoids interpreting an arbitrary get failure as absence.
        if logical not in self.provider.lifecycle_inventory():return None
        with self.provider.session() as m:
            m.discover();value=read_json(m,'get_container',{'username':logical}).get('container',{})
        try:
            target=Target(self.provider.config['provider'],self.provider.config['backend'],self.provider.config['project'],logical,value['name'],str(value['createdAt']),value['username'],value['name'])
            if value['username']!=logical:raise ValueError()
            ip=ipaddress.ip_address(value['network']['ipAddress'])
            if ip.version!=4 or ip.is_loopback or ip.is_multicast or ip.is_unspecified:raise ValueError()
            membership=value.get('networkTenant');source=value.get('networkTenantSource')
            if membership is not None:tenant(membership)
            return {'target':target,'ip':str(ip),'state':value['state'],'network_tenant':membership,'membership_source':source,'account_user':value['username']}
        except (KeyError,ValueError,TypeError):raise Fault('TARGET_FORMAT_UNSUPPORTED') from None
    def routes(self):
        with self.provider.session() as m:
            tools=m.discover()
            if not {'list_passthrough_routes','add_passthrough_route','delete_passthrough_route'}.issubset(tools):raise Fault('PORT_CONTRACT_UNSUPPORTED')
            required={'add_passthrough_route':{'external_port','protocol','target_ip','target_port','container_name','description'},'delete_passthrough_route':{'external_port','protocol'}}
            if any(not fields.issubset(tools[k].get('inputSchema',{}).get('properties',{})) for k,fields in required.items()):raise Fault('PORT_CONTRACT_UNSUPPORTED')
            value=read_json(m,'list_passthrough_routes',{})
        rows=value.get('routes');count=value.get('totalCount')
        if not isinstance(rows,list) or type(count) is not int or count!=len(rows) or value.get('nextCursor'):raise Fault('ROUTE_LIST_UNSUPPORTED')
        out=[];seen=set()
        for r in rows:
            if not isinstance(r,dict):raise Fault('ROUTE_LIST_UNSUPPORTED')
            protocol={'ROUTE_PROTOCOL_TCP':'tcp','ROUTE_PROTOCOL_UDP':'udp'}.get(r.get('protocol'))
            port=r.get('externalPort');target_port=r.get('targetPort')
            if protocol is None or type(port) is not int or type(target_port) is not int or not 1<=port<=65535 or not 1<=target_port<=65535 or (protocol,port) in seen or type(r.get('active',False)) is not bool:raise Fault('ROUTE_LIST_UNSUPPORTED')
            try:ip=str(ipaddress.IPv4Address(r['targetIp']))
            except (KeyError,ValueError,TypeError):raise Fault('ROUTE_LIST_UNSUPPORTED') from None
            if any(not isinstance(r.get(k,''),str) or len(r.get(k,''))>512 for k in ('containerName','description')):raise Fault('ROUTE_LIST_UNSUPPORTED')
            seen.add((protocol,port));out.append({'protocol':protocol,'external_port':port,'target_port':target_port,'target_ip':ip,'container':r.get('containerName',''),'description':r.get('description',''),'active':r.get('active',False)})
        return out
    def route_mutate(self,action,route):
        if action not in ('publish','unpublish'):raise Fault('ACTION_NOT_ALLOWED')
        name='add_passthrough_route' if action=='publish' else 'delete_passthrough_route'
        args={'external_port':route['external_port'],'protocol':route['protocol']}
        if action=='publish':args.update(target_ip=route['target_ip'],target_port=route['target_port'],container_name=route['container'],description=route['description'])
        with self.provider.session() as m:
            tools=m.discover();props=tools.get(name,{}).get('inputSchema',{}).get('properties',{})
            if not set(args).issubset(props):raise Fault('PORT_CONTRACT_UNSUPPORTED')
            m.tool(name,args)
