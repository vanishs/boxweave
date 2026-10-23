"""Behavior tests for client command generation and isolated client configuration."""
import argparse
import base64
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import shlex
import subprocess
import sys
import tempfile
import textwrap
import unittest
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'server'))
import clients
import cli

spec=importlib.util.spec_from_file_location('posix_setup',ROOT/'clients/posix/setup.py')
setup=importlib.util.module_from_spec(spec);spec.loader.exec_module(setup)
KEY='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIoY7yK5+VYtbzXBFgHguKKWJFTx98DSeDtmDTfS1zA public-fixture'

def descriptor():
    return {'schema_version':1,'ssh_host':'lab.example','ssh_port':2222,'linux_user':'developer',
            'launcher':'/usr/local/bin/boxweave-mcp','server_public_key':KEY,'container_public_key':KEY,
            'source_repo':'example/boxweave','source_commit':'a'*40,
            'client_sha256':{'clients/windows/setup.ps1':'b'*64,'clients/posix/setup.py':'c'*64}}

class Terminal(io.StringIO):
    def isatty(self): return True

class CommandTests(unittest.TestCase):
    def args(self,*args):
        return cli.parser().parse_args(['client',*args])
    def test_help_without_configuration(self):
        with contextlib.redirect_stdout(io.StringIO()) as out:
            with self.assertRaises(SystemExit) as e: self.args('--help')
        self.assertEqual(e.exception.code,0)
        self.assertIn('--transport',out.getvalue())
    def test_nonterminal_requires_explicit_os_and_agent(self):
        with patch('sys.stdin',io.StringIO()):
            with self.assertRaisesRegex(ValueError,'Missing --os'): clients.selection(self.args())
    def test_wizard_retries_and_selects_bridge(self):
        with patch('sys.stdin',Terminal('bad\n1\n2\n1\n2\n')), contextlib.redirect_stderr(io.StringIO()):
            result=clients.selection(self.args())
        self.assertEqual(result,{'os':'windows','agent':'cursor','runtime':'native','transport':'wsl'})
    def test_partial_wizard_only_missing_choices(self):
        with patch('sys.stdin',Terminal('3\n')),contextlib.redirect_stderr(io.StringIO()) as out:
            result=clients.selection(self.args('--os','macos'))
        self.assertEqual(result['agent'],'claude-code')
        self.assertNotIn('client OS',out.getvalue())
    def test_eof_cancels_without_reading_descriptor(self):
        with patch('sys.stdin',Terminal()),contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(clients.run(self.args()),130)
    def test_json_never_prompts(self):
        with patch('sys.stdin',Terminal()),contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(clients.run(self.args('--json')),2)
    def test_replace_wizard_and_flag_reach_generated_payload(self):
        cases = [(True, '2\n', [], True), (True, '1\n', [], False),
                 (True, '', ['--replace'], True), (False, '', [], False),
                 (False, '', ['--replace'], True)]
        for terminal, answer, flags, expected in cases:
            with self.subTest(terminal=terminal, flags=flags, expected=expected), tempfile.TemporaryDirectory() as tmp:
                config=Path(tmp)/'client.json';config.write_text(json.dumps(descriptor()))
                args=self.args('--os','macos','--agent','codex','--config',str(config),*flags)
                stdin=Terminal(answer) if terminal else io.StringIO(answer)
                with patch('sys.stdin',stdin),contextlib.redirect_stdout(io.StringIO()) as out,contextlib.redirect_stderr(io.StringIO()) as err:
                    self.assertEqual(clients.run(args),0)
                command=next(line for line in out.getvalue().splitlines() if line.startswith('bash -c '))
                payload=json.loads(base64.b64decode(shlex.split(shlex.split(command)[2])[-1]))
                self.assertEqual(payload['setup_options']['replace'],expected)
                self.assertEqual('registration?' in err.getvalue(),terminal and not flags)
    def test_trailing_notes_are_shell_comments_for_every_profile(self):
        profiles=[['--os','macos'], ['--os','windows','--runtime','wsl'],
                  ['--os','windows','--runtime','native','--transport','native'],
                  ['--os','windows','--runtime','native','--transport','wsl']]
        with tempfile.TemporaryDirectory() as tmp:
            config=Path(tmp)/'client.json';config.write_text(json.dumps(descriptor()))
            for profile in profiles:
                with self.subTest(profile=profile),patch('sys.stdin',io.StringIO()),contextlib.redirect_stdout(io.StringIO()) as out:
                    self.assertEqual(clients.run(self.args(*profile,'--agent','codex','--config',str(config))),0)
                tail=out.getvalue().split('\n\n',2)[-1].splitlines()[1:]
                notes=[line for line in tail if line.strip()]
                self.assertTrue(notes)
                self.assertTrue(all(line.startswith('# ') for line in notes))
    def test_replace_json_never_prompts_and_preserves_plain_notes(self):
        with tempfile.TemporaryDirectory() as tmp:
            config=Path(tmp)/'client.json';config.write_text(json.dumps(descriptor()))
            args=self.args('--os','macos','--agent','codex','--config',str(config),'--replace','--json')
            with patch('sys.stdin',Terminal()),contextlib.redirect_stdout(io.StringIO()) as out:
                self.assertEqual(clients.run(args),0)
            result=json.loads(out.getvalue())
            self.assertTrue(all(not note.startswith('#') for note in result['notes']))
            payload=json.loads(base64.b64decode(shlex.split(shlex.split(result['command'])[2])[-1]))
            self.assertTrue(payload['setup_options']['replace'])

    def test_complete_matrix(self):
        profiles=[]
        for agent in clients.AGENTS:
            profiles.append(clients.selection(self.args('--os','macos','--agent',agent)))
            for transport in ('native','wsl'):
                profiles.append(clients.selection(self.args('--os','windows','--agent',agent,'--runtime','native','--transport',transport)))
        for agent in ('codex','claude-code'):
            profiles.append(clients.selection(self.args('--os','windows','--agent',agent,'--runtime','wsl')))
        self.assertEqual(len(profiles),11)
        for p in profiles:
            result=clients.render(clients.validate(descriptor()),p)
            self.assertIn('a'*40,result['command'])
            if p['os']=='windows' and p['runtime']=='native':
                self.assertEqual(result['paste_into'],'Windows PowerShell')
                self.assertIn('Get-FileHash',result['command'])
                self.assertIn('ExecutionPolicy Bypass',result['command'])
                self.assertIn('LASTEXITCODE',result['command'])
            else:
                process=subprocess.run(['bash','-n','-c',shlex.split(result['command'])[2]],capture_output=True)
                self.assertEqual(process.returncode,0,process.stderr)
    def test_invalid_combinations(self):
        for options in (
            ['--os','macos','--agent','codex','--transport','wsl'],
            ['--os','windows','--agent','cursor','--runtime','wsl'],
            ['--os','windows','--agent','codex','--runtime','wsl','--transport','native']):
            with self.subTest(options=options),self.assertRaises(ValueError): clients.selection(self.args(*options))
    def test_descriptor_field_injection(self):
        for field,value in [('ssh_host','x;touch /tmp/NO'),('linux_user','-oBad'),
                            ('source_commit','main'),('launcher','/bin/sh; true'),('ssh_port',True),
                            ('server_public_key','ssh-ed25519 TEST')]:
            data=descriptor();data[field]=value
            with self.subTest(field=field),self.assertRaises(ValueError):clients.validate(data)
    def test_public_key_rejects_cross_type_blob(self):
        with self.assertRaises(ValueError):clients.public_key(KEY.replace('ssh-ed25519 ','ssh-rsa ',1))
    def test_override_only_changes_rendered_payload(self):
        with tempfile.TemporaryDirectory() as tmp:
            p=Path(tmp)/'client.json';original=json.dumps(descriptor());p.write_text(original)
            args=self.args('--os','macos','--agent','codex','--config',str(p),'--ssh-host','new.example','--json')
            with contextlib.redirect_stdout(io.StringIO()) as out: self.assertEqual(clients.run(args),0)
            result=json.loads(out.getvalue())
            command=shlex.split(shlex.split(result['command'])[2])
            encoded=command[-1]
            self.assertEqual(json.loads(base64.b64decode(encoded))['ssh_host'],'new.example')
            self.assertEqual(p.read_text(),original)
    def test_missing_descriptor_error_is_actionable(self):
        with contextlib.redirect_stderr(io.StringIO()) as out:
            self.assertEqual(clients.run(self.args('--os','macos','--agent','cursor','--config','/nonexistent/boxweave.json')),2)
        self.assertIn('update the server installer',out.getvalue())
    def test_ps_literal_quote(self):
        self.assertEqual(clients.psquote("space ' dollar $"),"'space '' dollar $'")
    def test_publisher_whitelists_and_hashes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);config=root/'config.json';host=root/'host.pub';box=root/'box.pub';dest=root/'public.json'
            host.write_text(KEY);box.write_text(KEY)
            d=descriptor();d.update(container_public_key_file=str(box),jwt_secret_file='/must/not/open',extra='NEVER_COPY')
            config.write_text(json.dumps(d))
            for name in d['client_sha256']:
                asset=root/name;asset.parent.mkdir(parents=True,exist_ok=True);asset.write_text('fixture')
            clients.publish(config,dest,root,'a'*40,host)
            data=clients.validate(json.loads(dest.read_text()))
            self.assertNotIn('jwt_secret_file',data);self.assertNotIn('NEVER_COPY',dest.read_text())
            self.assertEqual(len(data['client_sha256']['clients/posix/setup.py']),64)

class ConfigTests(unittest.TestCase):
    def test_toml_preserves_runtime_comments_and_other_servers(self):
        original='# Keep this\n[desktop]\nrunCodexInWindowsSubsystemForLinux = true\n[mcp_servers.other]\ncommand="other"\n'
        result=setup.toml_update(original,'/usr/bin/ssh',['user@host','a b'])
        self.assertTrue(result.startswith(original))
        self.assertEqual(setup.tomllib.loads(result)['mcp_servers']['other']['command'],'other')
        self.assertEqual(setup.toml_update(result,'/usr/bin/ssh',['user@host','a b']),result)
    def test_toml_non_ascii_paths(self):
        value=setup.toml_update('', 'ssh', ['/home/user/emoji-\U0001f600'])
        self.assertEqual(setup.tomllib.loads(value)['mcp_servers']['boxweave']['args'],['/home/user/emoji-\U0001f600'])
    def test_disabled_entry_is_not_silently_enabled(self):
        original='[mcp_servers.boxweave]\ncommand="ssh"\nargs=[]\nenabled=false\n'
        with self.assertRaisesRegex(ValueError,'extra settings'):setup.toml_update(original,'ssh',[])
    def test_toml_conflict_requires_replace(self):
        original='[mcp_servers.boxweave]\ncommand="other"\nargs=[]\n'
        with self.assertRaises(ValueError):setup.toml_update(original,'ssh',[])
        self.assertEqual(setup.tomllib.loads(setup.toml_update(original,'ssh',[],True))['mcp_servers']['boxweave']['command'],'ssh')
    def test_toml_rejects_unusual_tables_and_malformed_input(self):
        for original in ('not valid !','mcp_servers={boxweave={command="other"}}\n',
                         '[mcp_servers.boxweave]\ncommand="ssh"\nargs=[]\n[mcp_servers.boxweave.env]\nX="y"\n'):
            with self.subTest(original=original),self.assertRaises(ValueError):setup.toml_update(original,'ssh',[],True)
    def test_cursor_preserves_unrelated_data_and_repeat(self):
        original=json.dumps({'mcpServers':{'other':{'url':'https://example.invalid'}},'custom':True})
        result=setup.json_update(original,'ssh',['arg'])
        self.assertTrue(json.loads(result)['custom'])
        self.assertIn('other',json.loads(result)['mcpServers'])
        self.assertEqual(setup.json_update(result,'ssh',['arg']),result)
        with self.assertRaises(ValueError):setup.json_update(result,'different',[])
    def test_cursor_bad_shapes(self):
        for original in ('[]','null','{"mcpServers":null}','{"mcpServers":[]}'):
            with self.subTest(original=original),self.assertRaises(ValueError):setup.json_update(original,'ssh',[])
    def test_policy_preserves_user_content(self):
        result=setup.managed('User text\n','first')
        self.assertEqual(setup.managed(result,'first'),result)
        changed=setup.managed(result,'second')
        self.assertTrue(changed.startswith('User text\n'))
        self.assertNotIn('first',changed)
    def test_malformed_policy_markers(self):
        for text in (setup.BEGIN,setup.END,setup.END+setup.BEGIN,setup.BEGIN+setup.BEGIN+setup.END+setup.END):
            with self.subTest(text=text),self.assertRaises(ValueError):setup.managed(text,'policy')
    def test_atomic_backup_and_symlink(self):
        with tempfile.TemporaryDirectory() as tmp:
            target=Path(tmp)/'config';target.write_text('old')
            setup.atomic_write(target,'new')
            self.assertEqual(target.read_text(),'new')
            self.assertEqual(next(Path(tmp).glob('config.backup-*')).read_text(),'old')
            self.assertEqual(target.stat().st_mode & 0o777,0o600)
            link=Path(tmp)/'link';link.symlink_to(target)
            with self.assertRaises(ValueError):setup.atomic_write(link,'bad')
            self.assertEqual(target.read_text(),'new')
    def test_native_windows_rejected_by_posix_setup(self):
        d=descriptor();d.update(os='windows',runtime='native',transport='native',agent='codex',policy='text')
        with self.assertRaises(ValueError):setup.validate_data(d)

class ProbeTests(unittest.TestCase):
    def test_protocol_sequence_and_readonly_call(self):
        with tempfile.TemporaryDirectory() as tmp:
            script=Path(tmp)/'fake.py'
            script.write_text(textwrap.dedent('''\
                import json,sys
                phase=0
                for line in sys.stdin:
                    d=json.loads(line)
                    if d['method']=='initialize':
                        assert phase==0
                        phase=1
                        result={'protocolVersion':'2024-11-05'}
                    elif d['method']=='notifications/initialized':
                        assert phase==1
                        phase=2
                        continue
                    elif d['method']=='tools/list':
                        assert phase==2
                        phase=3
                        result={'tools':[{'name':'list_containers'}]}
                    else:
                        assert phase==3 and d['method']=='tools/call' and d['params']=={'name':'list_containers','arguments':{}}
                        result={'content':[]}
                    print(json.dumps({'jsonrpc':'2.0','id':d['id'],'result':result}),flush=True)
                '''))
            self.assertTrue(setup.probe(sys.executable,[str(script)],timeout=3))
    def test_timeout_is_not_an_exit_code(self):
        with self.assertRaisesRegex(TimeoutError,'timeout'):
            setup.probe(sys.executable,['-c','import time; time.sleep(10)'],timeout=.15)
    def test_closed_stream(self):
        with self.assertRaisesRegex(RuntimeError,'stream closed'):
            setup.probe(sys.executable,['-c','pass'],timeout=2)



class SetupFlowTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(prefix='boxweave-client-fixture-')
        self.addCleanup(self.tmp.cleanup)
        self.home=Path(self.tmp.name)
        bindir=self.home/'bin';bindir.mkdir()
        fake=textwrap.dedent('''\
            #!/usr/bin/python3
            import json,os,pathlib,sys
            root=pathlib.Path(os.environ['BOXWEAVE_TEST_ROOT'])
            name=pathlib.Path(sys.argv[0]).name
            args=sys.argv[1:]
            with (root/'calls.jsonl').open('a') as f: f.write(json.dumps([name,*args])+'\\n')
            if name=='ssh-keygen':
                if '-f' in args:
                    p=pathlib.Path(args[args.index('-f')+1]);p.write_text('FAKE IDENTITY FIXTURE')
                    p.with_suffix('.pub').write_text(os.environ['BOXWEAVE_PUBLIC_FIXTURE'])
                sys.exit(0)
            if name=='claude':
                exists=root/'claude-entry'
                if args[1]=='get': sys.exit(0 if exists.exists() else 1)
                if args[1]=='remove': exists.unlink()
                if args[1]=='add': exists.touch()
                sys.exit(0)
            if 'BatchMode=yes' not in args:
                sys.exit(0)
            phase=0
            for line in sys.stdin:
                d=json.loads(line)
                if d['method']=='initialize':
                    assert phase==0
                    phase=1
                    result={'protocolVersion':'2024-11-05'}
                elif d['method']=='notifications/initialized':
                    assert phase==1
                    phase=2
                    continue
                elif d['method']=='tools/list':
                    assert phase==2
                    phase=3
                    result={'tools':[{'name':'list_containers'}]}
                else:
                    assert phase==3 and d['method']=='tools/call'
                    assert d['params']=={'name':'list_containers','arguments':{}}
                    result={'content':[]}
                print(json.dumps({'jsonrpc':'2.0','id':d['id'],'result':result}),flush=True)
            ''')
        for name in ('ssh','ssh-keygen','claude'):
            tool=bindir/name;tool.write_text(fake);tool.chmod(0o755)
        self.env={'PATH':str(bindir)+':/usr/bin:/bin','BOXWEAVE_TEST_ROOT':str(self.home),
                  'BOXWEAVE_PUBLIC_FIXTURE':KEY,'CODEX_HOME':str(self.home/'.codex'),
                  'CLAUDE_CONFIG_DIR':str(self.home/'.claude')}
    def run_setup(self,agent='codex',replace=False,project=None):
        d=descriptor();d.update(os='macos',runtime='native',transport='native',agent=agent,policy='Fixture remote policy')
        args=argparse.Namespace(data_base64=base64.b64encode(json.dumps(d).encode()).decode(),
                                bootstrap_identity=None,agent_home=None,project_path=project,replace=replace)
        with patch.dict('os.environ',self.env),patch.object(Path,'home',return_value=self.home),patch.object(sys,'platform','darwin'),contextlib.redirect_stdout(io.StringIO()) as out:
            setup.setup(args)
        return out.getvalue()
    def calls(self):
        p=self.home/'calls.jsonl'
        return [json.loads(line) for line in p.read_text().splitlines()] if p.exists() else []
    def test_codex_complete_repeat_and_override(self):
        directory=self.home/'.codex';directory.mkdir()
        override=directory/'AGENTS.override.md';override.write_text('User override\n')
        out=self.run_setup()
        config=(directory/'config.toml').read_text()
        self.assertIn('passed',out)
        self.assertIn('User override',override.read_text())
        self.assertFalse((directory/'AGENTS.md').exists())
        self.run_setup()
        self.assertEqual((directory/'config.toml').read_text(),config)
        generates=[a for a in self.calls() if a[0]=='ssh-keygen' and '-f' in a]
        self.assertEqual(len(generates),1)
    def test_cursor_manual_guidance_and_project_rule(self):
        out=self.run_setup('cursor')
        self.assertIn('manual step required',out)
        self.assertTrue((self.home/'.cursor/boxweave-user-rules.md').is_file())
        project=self.home/'workspace';project.mkdir()
        out=self.run_setup('cursor',project=str(project))
        self.assertNotIn('manual step required',out)
        self.assertIn('alwaysApply: true',(project/'.cursor/rules/boxweave.mdc').read_text())
    def test_claude_user_scope_and_reviewed_replacement(self):
        self.run_setup('claude-code')
        self.assertTrue((self.home/'.claude/rules/boxweave.md').is_file())
        additions=[a for a in self.calls() if a[:3]==['claude','mcp','add']]
        self.assertEqual(additions[0][3:5],['--scope','user'])
        with self.assertRaisesRegex(ValueError,'Existing Claude'):self.run_setup('claude-code')
        self.run_setup('claude-code',replace=True)
    def test_bad_configuration_fails_before_keygen_or_enrollment(self):
        directory=self.home/'.codex';directory.mkdir()
        (directory/'config.toml').write_text('invalid !')
        with self.assertRaises(ValueError):self.run_setup()
        self.assertEqual(self.calls(),[])
    def test_changed_pin_and_incomplete_identity_fail_before_enrollment(self):
        sshdir=self.home/'.ssh';sshdir.mkdir()
        pin=sshdir/'boxweave_known_hosts';pin.write_text('changed')
        with self.assertRaisesRegex(ValueError,'identity changed'):self.run_setup()
        pin.unlink();(sshdir/'boxweave').write_text('OPAQUE FIXTURE')
        with self.assertRaisesRegex(ValueError,'Incomplete'):self.run_setup()
        self.assertEqual(self.calls(),[])

if __name__=='__main__': unittest.main()
