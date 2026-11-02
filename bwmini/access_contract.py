"""Pure access configuration validation independent of service and Provider."""
import re
from .domain import Fault, name
from .access_provider import tenant

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

