"""Behavioral diagnostics tests; no real credentials, enrollment or network."""
import argparse
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
        self.sshconfig=self.home/'.ssh/config'
        self.sshconfig.write_text(setup.ssh_config_managed('',
            setup.ssh_transport_block('fixture.invalid',22,'fixture',self.key,self.known)))
        self.env={'BOXWEAVE_SSH_HOST':'fixture.invalid','BOXWEAVE_SSH_PORT':'22','BOXWEAVE_SSH_USER':'fixture',
                  'BOXWEAVE_SERVER_HOST_KEY':KEY,'BOXWEAVE_AGENT':'codex',
                  'BOXWEAVE_RUNTIME':'native','BOXWEAVE_TRANSPORT':'native'}
        args=['boxweave','/usr/local/bin/boxweave-mcp']
        self.config=self.home/'.codex/config.toml';self.config.parent.mkdir()
        self.config.write_text(setup.toml_update('',str(self.peer),args))
    def invoke(self,**changes):
        values=dict(replace=False,bootstrap_identity=None,agent_home=str(self.config.parent),
                    project_path=None,timeout=1,report=None)
        values.update(changes)
        old=Path.read_text
        def read(path,*a,**kw):
            if path==self.key:raise AssertionError('Private identity content opened')
            return old(path,*a,**kw)
        with patch.dict('os.environ',self.env),patch.object(Path,'home',return_value=self.home),\
             patch.object(sys,'platform','darwin'),patch.object(setup.shutil,'which',return_value=str(self.peer)),\
             patch.object(Path,'read_text',read),contextlib.redirect_stdout(io.StringIO()) as output:
            code=setup.diagnose(argparse.Namespace(**values))
        return code,json.loads(output.getvalue())
    def test_diagnostic_is_readonly_and_passes_without_inventory(self):
        before={p:p.read_bytes() for p in (self.key,self.known,self.config,self.sshconfig)}
        code,report=self.invoke()
        self.assertEqual(code,0,report)
        self.assertEqual(report['connection'],'passed')
        self.assertEqual(report['agent_ui'],'unverified')
        self.assertEqual(report['registration'],'matched')
        self.assertEqual(before,{p:p.read_bytes() for p in before})
        text=json.dumps(report)
        self.assertNotIn(str(self.home),text);self.assertNotIn('OPAQUE',text)

    def test_zcode_registration_and_guidance_are_readonly(self):
        entry=setup.tomllib.loads(self.config.read_text())['mcp_servers']['boxweave']
        directory=self.home/'.zcode';(directory/'cli').mkdir(parents=True)
        config=directory/'cli/config.json'
        config.write_text(setup.zcode_update('{}',entry['command'],entry['args']))
        (directory/'AGENTS.md').write_text(setup.managed('User text','Fixture'))
        self.env['BOXWEAVE_AGENT']='zcode'
        before=config.read_bytes()
        code,report=self.invoke(agent_home=str(directory))
        self.assertEqual(code,0,report)
        self.assertEqual(report['registration'],'matched')
        self.assertEqual(report['guidance'],'present')
        self.assertEqual(config.read_bytes(),before)
        parsed=json.loads(config.read_text());parsed['mcp']['servers']['boxweave']['enable']=False
        config.write_text(json.dumps(parsed))
        code,report=self.invoke(agent_home=str(directory))
        self.assertEqual(code,1)
        self.assertEqual(report['registration'],'mismatch')
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
    def test_transport_block_mismatch_stops_before_protocol(self):
        self.sshconfig.write_text(setup.ssh_config_managed('',
            setup.ssh_transport_block('other.invalid',22,'fixture',self.key,self.known)))
        with patch.object(setup,'probe',side_effect=AssertionError('Probe must not run')):
            code,report=self.invoke()
        self.assertEqual(code,1)
        self.assertEqual(report['connection'],'unverified')
        self.assertEqual(report['stages'][-1]['code'],'TRANSPORT_MISMATCH')
    def test_missing_transport_block_reports_incomplete(self):
        self.sshconfig.unlink()
        code,report=self.invoke()
        self.assertEqual(code,2)
        self.assertEqual(report['connection'],'unverified')
    def test_export_refuses_existing_file_and_writes_private_new_file(self):
        target=self.home/'report.json'
        code,report=self.invoke(report=str(target))
        self.assertEqual(code,0,report)
        self.assertEqual(target.stat().st_mode&0o777,0o600)
        original=target.read_bytes()
        code,report=self.invoke(report=str(target))
        self.assertEqual(code,1);self.assertEqual(original,target.read_bytes())
    def test_invalid_input_and_mutation_flags_are_structured(self):
        for fields,env_changes in (({},{'BOXWEAVE_SSH_HOST':'bad host'}),
                                   ({},{'BOXWEAVE_SSH_PORT':'0'}),
                                   ({'replace':True},{}),({'bootstrap_identity':'unused'},{})):
            saved=dict(self.env);self.env.update(env_changes)
            try:
                code,report=self.invoke(**fields)
            finally:
                self.env=saved
            self.assertEqual(code,2);self.assertEqual(report['stages'][0]['code'],'INPUT_INVALID')
    def test_cli_diagnostic_branch_never_enrolls(self):
        with patch.object(setup,'diagnose',return_value=2) as diagnostic,patch.object(setup,'setup',side_effect=AssertionError()):
            with patch.object(sys,'argv',['setup.py','--diagnose']):
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
        return dict(schema_version=2,ssh_host='fixture.invalid',ssh_port=22,linux_user='fixture',
                    launcher='/usr/local/bin/boxweave-mcp',server_public_key=KEY,container_public_key=KEY,
                    source_repo='example/boxweave',source_commit='a'*40)
    def test_generated_script_documents_passthrough_flags_and_pins_source(self):
        for os_name,runtime,transport in [('windows','native','native'),('windows','native','wsl'),('windows','wsl','wsl'),('macos','native','native')]:
            result=clients.render(self.descriptor(),dict(os=os_name,runtime=runtime,transport=transport,agent='codex'))
            script=result['script']
            self.assertIn('a'*40,script)
            self.assertIn('EDIT ME',script)
            self.assertIn('--diagnose' if not (runtime=='native' and os_name=='windows') else '-Diagnose',script)
            if runtime=='wsl':
                process=subprocess.run(['bash','-n','-c',script],capture_output=True)
                self.assertEqual(process.returncode,0,process.stderr)
    def test_generator_has_no_diagnose_flag(self):
        with contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit): cli.main(['client','--diagnose'])

if __name__=='__main__':unittest.main()
