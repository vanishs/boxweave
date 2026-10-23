"""Behavioral diagnostics tests; no real credentials, enrollment or network."""
import argparse
import base64
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'server'))
import clients
import cli
spec=importlib.util.spec_from_file_location('diagnostic_setup',ROOT/'clients/posix/setup.py')
setup=importlib.util.module_from_spec(spec);spec.loader.exec_module(setup)
KEY='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIoY7yK5+VYtbzXBFgHguKKWJFTx98DSeDtmDTfS1zA fixture'

PEER=r"""#!/usr/bin/python3
import json,os,sys,time
case=os.environ.get('DIAGNOSTIC_CASE','valid')
if case=='blocked':
 time.sleep(10)
 sys.exit()
if case=='write_blocked':
 sys.stdin.readline()
 print(json.dumps({'jsonrpc':'2.0','method':'unknown','id':'x'*200000}),flush=True)
 time.sleep(10)
 sys.exit()
for raw in sys.stdin:
 d=json.loads(raw);m=d['method']
 if m=='notifications/initialized':continue
 if m=='initialize':r={'protocolVersion':'2024-11-05','capabilities':{'tools':{}}}
 elif m=='tools/list':
  if case=='second_page' and 'cursor' not in d['params']:r={'tools':[],'nextCursor':'next'}
  elif case=='cursor_loop':r={'tools':[],'nextCursor':'next'}
  else:r={'tools':[{'name':'list_containers'}]}
 else:
  r={'content':[]}
  if case=='empty_result':r={}
  if case=='denied':r={'isError':True,'content':[]}
  if case=='bad_content':r={'content':[{'type':'text','text':123}]}
  if case=='bad_error_type':r={'content':[],'isError':'false'}
 if case=='protocol' and m=='initialize':r['protocolVersion']='2099-01-01'
 if case=='missing_capabilities' and m=='initialize':r.pop('capabilities')
 if case=='oversized':print('x'*2200000,flush=True);time.sleep(10);sys.exit()
 if case=='early_exit':sys.stderr.write('FAKE_SECRET_DO_NOT_LOG');sys.exit(127)
 ident=d['id']+1 if case=='wrong_id' else d['id']
 if case=='invalid_json':print('not json',flush=True)
 else:print(json.dumps({'jsonrpc':'2.0','id':ident,'result':r}),flush=True)
"""

@unittest.skipUnless(os.name=='posix','POSIX process/pipe tests')
class DiagnosticTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
        self.home=Path(self.tmp.name);(self.home/'.ssh').mkdir()
        self.peer=self.home/'ssh';self.peer.write_text(PEER);self.peer.chmod(0o755)
        self.key=self.home/'.ssh/boxweave';self.key.write_text('OPAQUE FIXTURE: NEVER OPEN')
        self.known=self.home/'.ssh/boxweave_known_hosts';self.known.write_text('fixture.invalid '+KEY.rsplit(' ',1)[0]+'\n')
        self.data={'os':'macos','runtime':'native','transport':'native','agent':'codex',
                   'ssh_host':'fixture.invalid','ssh_port':22,'linux_user':'fixture',
                   'launcher':'/usr/local/bin/boxweave-mcp','server_public_key':KEY}
        args=['-T','-i',str(self.key),'-o','IdentitiesOnly=yes','-o','BatchMode=yes',
              '-o','StrictHostKeyChecking=yes','-o','UserKnownHostsFile='+str(self.known),
              '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15',
              '-p','22','fixture@fixture.invalid','/usr/local/bin/boxweave-mcp']
        self.config=self.home/'.codex/config.toml';self.config.parent.mkdir()
        self.config.write_text(setup.toml_update('',str(self.peer),args))
    def invoke(self,**changes):
        values=dict(data_base64=base64.b64encode(json.dumps(self.data).encode()).decode(),
                    replace=False,bootstrap_identity=None,agent_home=str(self.config.parent),
                    project_path=None,timeout=1,report=None)
        values.update(changes)
        old=Path.read_text
        def read(path,*a,**kw):
            if path==self.key:raise AssertionError('Private identity content opened')
            return old(path,*a,**kw)
        with patch.object(Path,'home',return_value=self.home),patch.object(sys,'platform','darwin'),\
             patch.object(setup.shutil,'which',return_value=str(self.peer)),\
             patch.object(Path,'read_text',read),contextlib.redirect_stdout(io.StringIO()) as output:
            code=setup.diagnose(argparse.Namespace(**values))
        return code,json.loads(output.getvalue())
    def test_diagnostic_is_readonly_and_passes_without_inventory(self):
        before={p:p.read_bytes() for p in (self.key,self.known,self.config)}
        code,report=self.invoke()
        self.assertEqual(code,0,report)
        self.assertEqual(report['connection'],'passed')
        self.assertEqual(report['agent_ui'],'unverified')
        self.assertEqual(report['registration'],'matched')
        self.assertEqual(before,{p:p.read_bytes() for p in before})
        text=json.dumps(report)
        self.assertNotIn(str(self.home),text);self.assertNotIn('OPAQUE',text)
    def test_missing_registration_does_not_hide_connection(self):
        self.config.unlink()
        code,report=self.invoke()
        self.assertEqual(code,2);self.assertEqual(report['connection'],'passed')
    def test_custom_registration_environment_is_not_claimed_matching(self):
        self.config.write_text(self.config.read_text()+'env = { EXTRA = "custom" }\n')
        code,report=self.invoke()
        self.assertEqual(code,2);self.assertEqual(report['registration'],'unsupported')
    def test_pin_mismatch_stops_before_protocol(self):
        self.known.write_text('wrong pin')
        with patch.object(setup,'probe',side_effect=AssertionError('Probe must not run')):
            code,report=self.invoke()
        self.assertEqual(code,1);self.assertEqual(report['connection'],'unverified')
    def test_export_refuses_existing_file_and_writes_private_new_file(self):
        target=self.home/'report.json'
        code,report=self.invoke(report=str(target))
        self.assertEqual(code,0,report)
        self.assertEqual(target.stat().st_mode&0o777,0o600)
        original=target.read_bytes()
        code,report=self.invoke(report=str(target))
        self.assertEqual(code,1);self.assertEqual(original,target.read_bytes())
    def test_invalid_input_and_mutation_flags_are_structured(self):
        for fields in ({'data_base64':'bad'},{'replace':True},{'bootstrap_identity':'unused'}):
            code,report=self.invoke(**fields)
            self.assertEqual(code,2);self.assertEqual(report['stages'][0]['code'],'INPUT_INVALID')
    def test_cli_diagnostic_branch_never_enrolls(self):
        with patch.object(setup,'diagnose',return_value=2) as diagnostic,patch.object(setup,'setup',side_effect=AssertionError()):
            with patch.object(sys,'argv',['setup.py','--diagnose','--data-base64','e30=']):
                self.assertEqual(setup.main(),2)
            diagnostic.assert_called_once()
    def test_protocol_fixtures(self):
        for case in ('valid','second_page','empty_result','denied','bad_content','bad_error_type','wrong_id','invalid_json','cursor_loop','protocol','missing_capabilities','oversized','early_exit'):
            with self.subTest(case=case),patch.dict(os.environ,{'DIAGNOSTIC_CASE':case}):
                evidence={}
                if case in ('valid','second_page'):
                    self.assertTrue(setup.probe(str(self.peer),[],timeout=1,evidence=evidence))
                    self.assertEqual(evidence['tool_pages'],2 if case=='second_page' else 1)
                else:
                    with self.assertRaises(setup.DiagnosticError):setup.probe(str(self.peer),[],timeout=1,evidence=evidence)
    def test_deadline_is_bounded(self):
        for case in ('blocked','write_blocked'):
            with patch.dict(os.environ,{'DIAGNOSTIC_CASE':case}):
                with self.assertRaises(setup.DiagnosticTimeout):
                    setup.probe(str(self.peer),[],timeout=.1)

class GeneratorTests(unittest.TestCase):
    def descriptor(self):
        return dict(ssh_host='fixture.invalid',ssh_port=22,linux_user='fixture',
                    launcher='/usr/local/bin/boxweave-mcp',server_public_key=KEY,container_public_key=KEY,
                    source_repo='example/boxweave',source_commit='a'*40,
                    client_sha256={'clients/windows/setup.ps1':'b'*64,'clients/posix/setup.py':'c'*64})
    def test_generated_diagnostics_pin_assets_and_select_early_entry(self):
        for os_name,runtime,transport in [('windows','native','native'),('windows','native','wsl'),('windows','wsl','wsl'),('macos','native','native')]:
            result=clients.render(self.descriptor(),dict(os=os_name,runtime=runtime,transport=transport,agent='codex'),{'diagnose':True})
            self.assertEqual(result['operation'],'client_diagnose')
            self.assertIn('-Diagnose' if runtime=='native' and os_name=='windows' else '--diagnose',result['command'])
            self.assertIn('a'*40,result['command'])
    def test_generator_rejects_mutating_diagnostic_options(self):
        with contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(cli.main(['client','--diagnose','--replace']),2)

if __name__=='__main__':unittest.main()
