"""Deployment validation and independent authenticated read; diagnostic output is public metadata."""
import argparse,json,re,sqlite3,sys
from pathlib import Path
from .app_runtime import load,read_config
from .provider import Provider
from .mcp import MCP
from .domain import Fault

def require_minimum_runtime_version(value, code):
 match=re.fullmatch(r'v?([0-9]+)\.([0-9]+)\.([0-9]+)',value) if isinstance(value,str) else None
 if not match or tuple(int(n) for n in match.groups()) < (0,128,0):
  raise Fault(code)
 return value


class Observed(MCP):
 def call(self,method,params):
  value=super().call(method,params)
  if method=='initialize':self.server=value
  return value
def read_provider(c):
 p=Provider(c)
 with Observed(p.argv,30) as client:
  tools=client.discover()
  server=client.server.get('serverInfo',{})
  version=server.get('version') if isinstance(server,dict) else None
  require_minimum_runtime_version(version,'MCP_SERVER_BELOW_MINIMUM_OR_UNKNOWN')
  client.tool('list_containers',{})
  result={'state':'authenticated_read_passed','server':server,'protocol':client.server['protocolVersion'],'tools':len(tools),'read':'list_containers','minimum_mcp_server':'0.128.0'}
  if p.rest is not None:
   # Authenticated daemon runtime identity is independent from the binary
   # currently present on disk. Refuse old running processes after an update.
   details=p.rest.request('GET','/v1/system/info',timeout=20).get('info')
   runtime=details.get('daemonVersion') if isinstance(details,dict) else None
   require_minimum_runtime_version(runtime,'PROVIDER_RUNTIME_BELOW_MINIMUM_OR_UNKNOWN')
   p.rest.inventory()
   result['daemon_version']=runtime
   result['basic_management']='rest_authenticated_inventory_passed'
  else:result['basic_management']='legacy_mcp'
  return result
def check(config,mcp=False):
 c=read_config(config);p=Provider(c);result={'mode':'mcp','basic_management':'rest' if p.rest is not None else 'legacy_mcp'}
 if not mcp:
  c,s=load(config);result['schema']=7
 else:
  db=Path(c['database'])
  with sqlite3.connect('file:'+str(db)+'?mode=ro',uri=True) as connection:
   result['admin_initialized']=connection.execute('SELECT count(*) FROM admin').fetchone()[0]==1
  result['mcp']=read_provider(c)
 return result
def main():
 parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--config',required=True);parser.add_argument('--mcp',action='store_true');a=parser.parse_args()
 try:print(json.dumps(check(a.config,a.mcp)));return 0
 except (Fault,OSError,ValueError,KeyError,sqlite3.Error) as e:print(json.dumps({'code':getattr(e,'code','DEPLOYMENT_CHECK_FAILED')}));return 2
if __name__=='__main__':raise SystemExit(main())
