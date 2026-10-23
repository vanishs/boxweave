import os
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'server'))
from mcp_client import Client, MCPError


@unittest.skipUnless(os.name=='posix','POSIX stdio tests')
class MCPClientTests(unittest.TestCase):
    def launcher(self, body):
        tmp=tempfile.TemporaryDirectory();self.addCleanup(tmp.cleanup)
        p=Path(tmp.name)/'server'
        p.write_text('#!'+sys.executable+'\n'+body);p.chmod(0o755)
        return p

    def test_real_stdio_initialization_notification_and_tool_call(self):
        p=self.launcher('''import sys,json
initialized=False
for line in sys.stdin:
 d=json.loads(line)
 if d['method']=='notifications/initialized': initialized=True;continue
 if d['method']=='initialize': result={'protocolVersion':'2024-11-05','serverInfo':{'name':'fake'}}
 elif initialized: result={'content':[{'type':'text','text':'hello world'}]}
 else: raise SystemExit(3)
 print(json.dumps({'jsonrpc':'2.0','id':d['id'],'result':result}),flush=True)
''')
        client=Client(p)
        try:self.assertEqual(client.tool('list_containers',{})['content'][0]['text'],'hello world')
        finally:client.close()

    def test_no_response_is_bounded(self):
        p=self.launcher('import time\ntime.sleep(30)\n')
        with self.assertRaises(TimeoutError):Client(p,timeout=.1)

    def test_unbounded_response_is_rejected(self):
        p=self.launcher('import sys\nsys.stdout.write("x"*2200000);sys.stdout.flush()\n')
        with self.assertRaises(MCPError):Client(p,timeout=2)
