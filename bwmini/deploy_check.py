"""Deployment validation and independent authenticated read; diagnostic output is public metadata."""
import argparse,json,sqlite3,sys
from pathlib import Path
from .__main__ import load,read_config
from .provider import Provider
from .mcp import MCP
from .domain import Fault

class Observed(MCP):
 def call(self,method,params):
  value=super().call(method,params)
  if method=='initialize':self.server=value
  return value
def check(config,mcp=False):
 c=read_config(config);p=Provider(c);result={'mode':'mcp'}
 if not mcp:
  c,s=load(config);result['schema']=6
 else:
  db=Path(c['database'])
  with sqlite3.connect('file:'+str(db)+'?mode=ro',uri=True) as connection:
   result['admin_initialized']=connection.execute('SELECT count(*) FROM admin').fetchone()[0]==1
  with Observed(p.argv,30) as client:
   tools=client.discover();client.tool('list_containers',{})
   result['mcp']={'state':'authenticated_read_passed','server':client.server.get('serverInfo',{}),'protocol':client.server['protocolVersion'],'tools':len(tools),'read':'list_containers'}
 return result
def main():
 parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--config',required=True);parser.add_argument('--mcp',action='store_true');a=parser.parse_args()
 try:print(json.dumps(check(a.config,a.mcp)));return 0
 except (Fault,OSError,ValueError,KeyError,sqlite3.Error) as e:print(json.dumps({'code':getattr(e,'code','DEPLOYMENT_CHECK_FAILED')}));return 2
if __name__=='__main__':raise SystemExit(main())
