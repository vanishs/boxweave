"""Exercise real PowerShell native argv modes against an isolated Linux WSL shim.

Set BOXWEAVE_TEST_PWSH or put pwsh on PATH. These are not live Windows enrollment
tests: only the named setup helper functions and shell bodies are loaded.
"""
import base64
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[1]
PWSH=os.environ.get('BOXWEAVE_TEST_PWSH') or shutil.which('pwsh')
KEY='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIoY7yK5+VYtbzXBFgHguKKWJFTx98DSeDtmDTfS1zA fixture'
PIN='[fixture.invalid]:2222 '+KEY

@unittest.skipUnless(PWSH and os.name=='posix','Requires PowerShell 7.3+ on Linux for isolated native argv regression')
class WindowsWslTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(prefix='boxweave-wsl-argv-')
        self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)
        self.fixture_home=self.root/'home with spaces'
        self.fixture_home.mkdir()
        self.bin=self.root/'bin';self.bin.mkdir()
        self.shim=self.bin/'wsl-fixture'
        self.shim.write_text("""#!/usr/bin/python3
import json,os,pathlib,sys
root=pathlib.Path(os.environ['BOXWEAVE_WSL_FIXTURE'])
with (root/'calls.jsonl').open('a') as f:f.write(json.dumps(sys.argv[1:])+'\\n')
a=sys.argv[1:]
if a[:5]!=['-d','Fixture Distro','-e','sh','-c']:sys.exit('Invalid WSL argv prefix')
os.execv('/bin/sh',a[3:])
""")
        self.shim.chmod(0o755)
        keygen=self.bin/'ssh-keygen'
        keygen.write_text("""#!/usr/bin/python3
import os,pathlib,sys
if '-f' in sys.argv:
 p=pathlib.Path(sys.argv[sys.argv.index('-f')+1])
 p.write_text('OPAQUE TEST FIXTURE')
 p.with_suffix('.pub').write_text(os.environ['BOXWEAVE_WSL_PUBLIC'])
 with (pathlib.Path(os.environ['BOXWEAVE_WSL_FIXTURE'])/'generations').open('a') as f:f.write('generated\\n')
""")
        keygen.chmod(0o755)
        self.peer=self.bin/'mcp-peer'
        self.peer.write_text("""#!/usr/bin/python3
import json,os,pathlib,sys
root=pathlib.Path(os.environ['BOXWEAVE_WSL_FIXTURE'])
if '--fail' in sys.argv:
 sys.stdin.buffer.readline()
 sys.stderr.write('FAKE_SECRET_DO_NOT_LOG\\n')
 sys.exit(127)
methods=[]
for raw in sys.stdin.buffer:
 assert raw.startswith(b'{') and raw.endswith(b'\\n')
 d=json.loads(raw.decode('utf-8'));methods.append(d['method'])
 if d['method']=='initialize':
  assert len(methods)==1
  result={'protocolVersion':'2024-11-05','serverInfo':{'name':'fixture'}}
 elif d['method']=='notifications/initialized':
  assert len(methods)==2
  continue
 elif d['method']=='tools/list':
  assert len(methods)==3
  result={'tools':[{'name':'list_containers'}]}
 else:
  assert methods==['initialize','notifications/initialized','tools/list','tools/call']
  assert d['params']=={'name':'list_containers','arguments':{}}
  (root/'handshake.json').write_text(json.dumps(methods))
  result={'content':[]}
 print(json.dumps({'jsonrpc':'2.0','id':d['id'],'result':result}),flush=True)
""")
        self.peer.chmod(0o755)
        self.env=dict(os.environ,HOME=str(self.fixture_home),
                      PATH=str(self.bin)+':'+os.environ['PATH'],
                      BOXWEAVE_WSL_FIXTURE=str(self.root),BOXWEAVE_WSL_PUBLIC=KEY,
                      POWERSHELL_TELEMETRY_OPTOUT='1',POWERSHELL_UPDATECHECK='Off')
    def run_case(self,case,mode='Legacy',body='',arguments=()):
        p=self.root/'fixture.json'
        p.write_text(json.dumps({'shim':str(self.shim),'peer':str(self.peer),'pin_base64':base64.b64encode(PIN.encode()).decode(),
                                 'body':body,'arguments':list(arguments)}))
        return subprocess.run([PWSH,'-NoLogo','-NoProfile','-NonInteractive','-File',
                               str(ROOT/'tests/windows_wsl.ps1'),'-SetupPath',str(ROOT/'clients/windows/setup.ps1'),
                               '-FixturePath',str(p),'-Mode',mode,'-Case',case],
                              capture_output=True,text=True,env=self.env,timeout=20)
    def test_original_legacy_call_reproduces_base64_invalid_input(self):
        self.fixture_home=self.root/'plain-home';self.fixture_home.mkdir()
        self.env['HOME']=str(self.fixture_home)
        r=self.run_case('original')
        self.assertNotEqual(r.returncode,0)
        self.assertIn('base64: invalid input',r.stderr)
        self.assertFalse((self.fixture_home/'.ssh/boxweave_known_hosts').exists())
    def test_pin_roundtrip_and_repeat_in_both_modes(self):
        for mode in ('Legacy','Standard'):
            for _ in range(2):
                r=self.run_case('pin',mode)
                self.assertEqual(r.returncode,0,r.stderr)
                self.assertEqual((self.fixture_home/'.ssh/boxweave_known_hosts').read_text(),PIN+'\n')
        calls=[json.loads(line) for line in (self.root/'calls.jsonl').read_text().splitlines()]
        self.assertTrue(all(len(a)==6 and '"' not in a[-1] and '\n' not in a[-1] for a in calls))
    def test_mismatch_preserves_existing_pin(self):
        p=self.fixture_home/'.ssh/boxweave_known_hosts';p.parent.mkdir();p.write_text('existing identity\n')
        r=self.run_case('pin')
        self.assertNotEqual(r.returncode,0)
        self.assertIn('pinned host key differs',r.stderr)
        self.assertIn('exit code 1',r.stderr)
        self.assertEqual(p.read_text(),'existing identity\n')
    def test_key_creation_and_reuse_in_both_modes(self):
        for mode in ('Legacy','Standard'):
            r=self.run_case('key',mode)
            self.assertEqual(r.returncode,0,r.stderr)
            self.assertEqual(r.stdout.strip(),KEY)
        self.assertEqual((self.root/'generations').read_text(),'generated\n')
    def test_home_with_spaces_in_both_modes(self):
        for mode in ('Legacy','Standard'):
            r=self.run_case('home',mode)
            self.assertEqual(r.returncode,0,r.stderr)
            self.assertEqual(r.stdout.strip(),str(self.fixture_home))
    def test_crlf_and_nested_quotes_are_encoded(self):
        body="set -eu\r\npath=\"a b\"\r\nif [ \"$(printf '%s' \"$path\")\" = \"a b\" ]; then printf '%s\\n' \"$1\"; fi\r\n"
        for mode in ('Legacy','Standard'):
            r=self.run_case('body',mode,body,['Ab+/=='])
            self.assertEqual(r.returncode,0,r.stderr)
            self.assertEqual(r.stdout,'Ab+/==\n')
    def test_script_source_is_not_exposed_to_native_quote_parsing(self):
        body="printf '%s\\n' \"nested $(printf '%s' \\\"value\\\")\"\r\n"
        r=self.run_case('command',body=body,arguments=['key','Ab+/=='])
        self.assertEqual(r.returncode,0,r.stderr)
        command=r.stdout.strip()
        self.assertNotIn('"',command)
        encoded=re.search(r"printf '%s' '([A-Za-z0-9+/=]+)'",command).group(1)
        self.assertEqual(base64.b64decode(encoded).decode(),body.replace('\r\n','\n'))
    def test_rejects_shell_metacharacters_before_execution(self):
        r=self.run_case('body',body='exit 0',arguments=['key;touch injected'])
        self.assertNotEqual(r.returncode,0)
        self.assertIn('Invalid WSL script argument',r.stderr)
        self.assertFalse((self.root/'calls.jsonl').exists())
    def test_nonzero_exit_is_propagated(self):
        r=self.run_case('body',body='exit 23')
        self.assertNotEqual(r.returncode,0)
        self.assertIn('exit code 23',r.stderr)

    def test_process_switches_are_not_unnecessarily_quoted(self):
        args=['-d','Ubuntu','-e','ssh','-T','-i','/home/user with spaces/.ssh/boxweave','-o','BatchMode=yes']
        r=self.run_case('process-arguments',arguments=args)
        self.assertEqual(r.returncode,0,r.stderr)
        self.assertEqual(r.stdout.strip(),'-d Ubuntu -e ssh -T -i "/home/user with spaces/.ssh/boxweave" -o BatchMode=yes')
    def test_empty_process_argument_is_preserved(self):
        r=self.run_case('process-arguments',arguments=['-N',''])
        self.assertEqual(r.returncode,0,r.stderr)
        self.assertEqual(r.stdout.strip(),'-N ""')
    def test_complete_powershell_mcp_handshake(self):
        for mode in ('Legacy','Standard'):
            r=self.run_case('mcp',mode)
            self.assertEqual(r.returncode,0,r.stderr)
            self.assertIn('MCP fixture passed',r.stdout)
            self.assertEqual(json.loads((self.root/'handshake.json').read_text()),
                             ['initialize','notifications/initialized','tools/list','tools/call'])
    def test_mcp_early_exit_reports_status_without_raw_stderr(self):
        r=self.run_case('mcp',arguments=['--fail'])
        self.assertNotEqual(r.returncode,0)
        self.assertIn('process exit 127',r.stderr)
        self.assertNotIn('FAKE_SECRET_DO_NOT_LOG',r.stderr)

if __name__=='__main__':unittest.main()
