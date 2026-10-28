"""Behavior tests for client script generation and isolated client configuration."""
import argparse
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
    return {'schema_version':2,'ssh_host':'lab.example','ssh_port':2222,'linux_user':'developer',
            'launcher':'/usr/local/bin/boxweave-mcp','server_public_key':KEY,'container_public_key':KEY,
            'source_repo':'example/boxweave','source_commit':'a'*40}

def env_for(d,profile):
    return {'BOXWEAVE_SSH_HOST':d['ssh_host'],'BOXWEAVE_SSH_PORT':str(d['ssh_port']),
            'BOXWEAVE_SSH_USER':d['linux_user'],'BOXWEAVE_SERVER_HOST_KEY':d['server_public_key'],
            'BOXWEAVE_CONTAINER_PUBLIC_KEY':d['container_public_key'],
            'BOXWEAVE_AGENT':profile['agent'],'BOXWEAVE_RUNTIME':profile['runtime'],
            'BOXWEAVE_TRANSPORT':profile['transport']}

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
        self.assertNotIn('--ssh-host',out.getvalue())
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
    def test_generated_script_has_editable_header_for_every_profile(self):
        profiles=[['--os','macos'], ['--os','windows','--runtime','wsl'],
                  ['--os','windows','--runtime','native','--transport','native'],
                  ['--os','windows','--runtime','native','--transport','wsl']]
        for profile in profiles:
            with self.subTest(profile=profile),tempfile.TemporaryDirectory() as tmp:
                config=Path(tmp)/'client.json';config.write_text(json.dumps(descriptor()))
                args=self.args(*profile,'--agent','codex','--config',str(config),'--json')
                with patch('sys.stdin',io.StringIO()),contextlib.redirect_stdout(io.StringIO()) as out:
                    self.assertEqual(clients.run(args),0)
                result=json.loads(out.getvalue())
                script=result['script']
                self.assertIn('EDIT ME',script)
                self.assertIn('lab.example',script)
                self.assertIn('2222',script)
                self.assertIn(KEY,script)
                self.assertIn('a'*40,script)
                self.assertNotIn('client_sha256',script)
                if profile[1]=='windows' and 'native' in profile:
                    self.assertEqual(result['paste_into'],'Windows PowerShell')
                    self.assertIn('$env:BOXWEAVE_SSH_HOST',script)
                    self.assertIn('ExecutionPolicy Bypass',script)
                    expected_transport = "'WSL'" if profile[-1]=='wsl' else "'Native'"
                    self.assertIn('$env:BOXWEAVE_TRANSPORT = '+expected_transport,script)
                else:
                    self.assertIn('BOXWEAVE_SSH_HOST=',script)
                    process=subprocess.run(['bash','-n','-c',script],capture_output=True)
                    self.assertEqual(process.returncode,0,process.stderr)
    def test_output_writes_script_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            config=Path(tmp)/'client.json';config.write_text(json.dumps(descriptor()))
            target=Path(tmp)/'out.sh'
            args=self.args('--os','macos','--agent','codex','--config',str(config),'--output',str(target))
            with patch('sys.stdin',io.StringIO()),contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(clients.run(args),0)
            script=target.read_text()
            self.assertIn('BOXWEAVE_SSH_HOST=lab.example',script)
            self.assertIn('clients/posix/setup.py',script)
    def test_glm_options_land_in_script_without_the_key(self):
        with tempfile.TemporaryDirectory() as tmp:
            config=Path(tmp)/'client.json';config.write_text(json.dumps(descriptor()))
            args=self.args('--os','macos','--agent','claude-code','--claude-provider','zhipu',
                           '--glm-model','glm-test','--config',str(config),'--json')
            with patch.dict('os.environ',{'GLM_API_KEY':'do-not-export'}), \
                 patch('sys.stdin',io.StringIO()),contextlib.redirect_stdout(io.StringIO()) as out:
                self.assertEqual(clients.run(args),0)
            script=json.loads(out.getvalue())['script']
            self.assertIn('BOXWEAVE_CLAUDE_PROVIDER=zhipu',script)
            self.assertIn('BOXWEAVE_GLM_MODEL=glm-test',script)
            self.assertNotIn('do-not-export',script)
            args.agent='codex'
            with contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(clients.run(args),2)
    def test_complete_matrix(self):
        profiles=[]
        for agent in clients.AGENTS:
            profiles.append(clients.selection(self.args('--os','macos','--agent',agent)))
            for transport in ('native','wsl'):
                profiles.append(clients.selection(self.args('--os','windows','--agent',agent,'--runtime','native','--transport',transport)))
        for agent in ('codex','claude-code'):
            profiles.append(clients.selection(self.args('--os','windows','--agent',agent,'--runtime','wsl')))
        self.assertEqual(len(profiles),14)
        for p in profiles:
            result=clients.render(clients.validate(descriptor()),p)
            if p['os']=='windows' and p['runtime']=='native':
                self.assertEqual(result['paste_into'],'Windows PowerShell')
            else:
                process=subprocess.run(['bash','-n','-c',result['script']],capture_output=True)
                self.assertEqual(process.returncode,0,process.stderr)
    def test_invalid_combinations(self):
        for options in (
            ['--os','macos','--agent','codex','--transport','wsl'],
            ['--os','windows','--agent','cursor','--runtime','wsl'],
            ['--os','windows','--agent','zcode','--runtime','wsl'],
            ['--os','windows','--agent','codex','--runtime','wsl','--transport','native']):
            with self.subTest(options=options),self.assertRaises(ValueError): clients.selection(self.args(*options))
    def test_descriptor_field_injection(self):
        for field,value in [('ssh_host','x;touch /tmp/NO'),('linux_user','-oBad'),
                            ('source_commit','main'),('launcher','/bin/sh; true'),('ssh_port',True),
                            ('server_public_key','ssh-ed25519 TEST'),('schema_version',1)]:
            data=descriptor();data[field]=value
            with self.subTest(field=field),self.assertRaises(ValueError):clients.validate(data)
    def test_public_key_rejects_cross_type_blob(self):
        with self.assertRaises(ValueError):clients.public_key(KEY.replace('ssh-ed25519 ','ssh-rsa ',1))
    def test_removed_legacy_flags_are_gone(self):
        for flag in ('--ssh-host','--ssh-port','--replace','--diagnose','--bootstrap-identity'):
            with self.subTest(flag=flag),contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit): self.args(flag,'x')
    def test_missing_descriptor_error_is_actionable(self):
        with contextlib.redirect_stderr(io.StringIO()) as out:
            self.assertEqual(clients.run(self.args('--os','macos','--agent','cursor','--config','/nonexistent/boxweave.json')),2)
        self.assertIn('update the server installer',out.getvalue())
    def test_ps_literal_quote(self):
        self.assertEqual(clients.ps_quote("space ' dollar $"),"'space '' dollar $'")
    def test_publisher_whitelists_descriptor_fields(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);config=root/'config.json';host=root/'host.pub';box=root/'box.pub';dest=root/'public.json'
            host.write_text(KEY);box.write_text(KEY)
            d=descriptor();d.update(container_public_key_file=str(box),jwt_secret_file='/must/not/open',extra='NEVER_COPY')
            config.write_text(json.dumps(d))
            clients.publish(config,dest,'a'*40,host)
            data=clients.validate(json.loads(dest.read_text()))
            self.assertNotIn('jwt_secret_file',data);self.assertNotIn('NEVER_COPY',dest.read_text())
            self.assertEqual(data['launcher'],'/usr/local/bin/boxweave-mcp')
            self.assertEqual(data['schema_version'],2)

class ConfigTests(unittest.TestCase):
    def test_toml_preserves_runtime_comments_and_other_servers(self):
        original='# Keep this\n[desktop]\nrunCodexInWindowsSubsystemForLinux = true\n[mcp_servers.other]\ncommand="other"\n'
        result=setup.toml_update(original,'/usr/bin/ssh',['boxweave','/usr/local/bin/boxweave-mcp'])
        self.assertTrue(result.startswith(original))
        self.assertEqual(setup.tomllib.loads(result)['mcp_servers']['other']['command'],'other')
        self.assertEqual(setup.toml_update(result,'/usr/bin/ssh',['boxweave','/usr/local/bin/boxweave-mcp']),result)
    def test_toml_preserves_tool_approval_tables(self):
        parent='[mcp_servers.boxweave]\ncommand="old"\nargs=[]\n'
        nested='[mcp_servers.boxweave.tools.list_containers]\napproval_mode="approve"\n[mcp_servers.boxweave.tools.connect]\napproval_mode="approve"\n'
        other='[mcp_servers.other]\ncommand="other"\n'
        result=setup.toml_update(parent+nested+other,'wsl.exe',['boxweave','/usr/local/bin/boxweave-mcp'],True)
        self.assertIn(nested+other,result)
        parsed=setup.tomllib.loads(result)['mcp_servers']
        self.assertEqual(parsed['boxweave']['tools']['connect']['approval_mode'],'approve')
        self.assertEqual(parsed['boxweave']['command'],'wsl.exe')
        self.assertEqual(setup.toml_update(result,'wsl.exe',['boxweave','/usr/local/bin/boxweave-mcp']),result)
        with self.assertRaises(ValueError):setup.toml_update(result,'different',[])
    def test_toml_non_ascii_paths(self):
        value=setup.toml_update('', 'ssh', ['boxweave','/home/user/emoji-\U0001f600'])
        self.assertEqual(setup.tomllib.loads(value)['mcp_servers']['boxweave']['args'],['boxweave','/home/user/emoji-\U0001f600'])
    def test_disabled_entry_is_not_silently_enabled(self):
        original='[mcp_servers.boxweave]\ncommand="ssh"\nargs=[]\nenabled=false\n'
        with self.assertRaisesRegex(ValueError,'extra settings'):setup.toml_update(original,'ssh',[])
    def test_toml_conflict_requires_replace(self):
        original='[mcp_servers.boxweave]\ncommand="other"\nargs=[]\n'
        with self.assertRaises(ValueError):setup.toml_update(original,'ssh',[])
        self.assertEqual(setup.tomllib.loads(setup.toml_update(original,'ssh',[],True))['mcp_servers']['boxweave']['command'],'ssh')
    def test_toml_rejects_unusual_tables_and_malformed_input(self):
        for original in ('not valid !','mcp_servers={boxweave={command="other"}}\n',
                         '[mcp_servers.boxweave]\ncommand="ssh"\nargs=[]\ntools={connect={approval_mode="approve"}}\n',
                         '[mcp_servers.boxweave]\ncommand="ssh"\nargs=[]\n[mcp_servers.boxweave.env]\nX="y"\n'):
            with self.subTest(original=original),self.assertRaises(ValueError):setup.toml_update(original,'ssh',[],True)
    def test_cursor_preserves_unrelated_data_and_repeat(self):
        original=json.dumps({'mcpServers':{'other':{'url':'https://example.invalid'}},'custom':True})
        result=setup.json_update(original,'ssh',['boxweave','/usr/local/bin/boxweave-mcp'])
        self.assertTrue(json.loads(result)['custom'])
        self.assertIn('other',json.loads(result)['mcpServers'])
        self.assertEqual(setup.json_update(result,'ssh',['boxweave','/usr/local/bin/boxweave-mcp']),result)
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
    def test_ssh_config_block_roundtrip_and_readdress(self):
        block=setup.ssh_transport_block('lab.example',2222,'developer','/h/.ssh/boxweave','/h/.ssh/boxweave_known_hosts')
        out=setup.ssh_config_managed('',block)
        self.assertEqual(out.count(setup.SSH_BEGIN),1)
        parsed=setup.parse_ssh_transport_block(out)
        self.assertEqual(parsed['HostName'],'lab.example')
        self.assertEqual(parsed['Port'],'2222')
        self.assertEqual(parsed['IdentityFile'],'/h/.ssh/boxweave')
        # Idempotent rewrite of the same block.
        self.assertEqual(setup.ssh_config_managed(out,block),out)
        # Readdressing replaces the block and preserves unrelated content.
        other=setup.ssh_transport_block('new.example',22,'developer','/h/.ssh/boxweave','/h/.ssh/boxweave_known_hosts')
        updated=setup.ssh_config_managed('Host other\n  User x\n\n'+out,other)
        self.assertTrue(updated.startswith('Host other'))
        self.assertNotIn('lab.example',updated)
        self.assertEqual(setup.parse_ssh_transport_block(updated)['HostName'],'new.example')
        self.assertEqual(setup.parse_ssh_transport_block(updated)['Port'],'22')
    def test_ssh_config_malformed_markers(self):
        block=setup.ssh_transport_block('h',22,'u','/k','/kn')
        for text in (setup.SSH_BEGIN+'\nx\n',setup.SSH_END+'\n'+setup.SSH_BEGIN,
                     setup.SSH_BEGIN+'\n'+setup.SSH_BEGIN+'\n'+setup.SSH_END):
            with self.subTest(text=text),self.assertRaises(ValueError):setup.ssh_config_managed(text,block)
    def test_host_pin_token_forms(self):
        self.assertEqual(setup.host_token('h.example',22),'h.example')
        self.assertEqual(setup.host_token('h.example',2222),'[h.example]:2222')
        self.assertEqual(setup.host_token('::1',22),'[::1]:22')

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
                        result={'capabilities':{'tools':{}},'protocolVersion':'2024-11-05'}
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
                    result={'capabilities':{'tools':{}},'protocolVersion':'2024-11-05'}
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
    def run_setup(self,agent='codex',replace=False,project=None,env_overrides=None):
        d=descriptor()
        profile={'agent':agent,'runtime':'native','transport':'native'}
        env=dict(self.env);env.update(env_for(d,profile))
        if env_overrides: env.update(env_overrides)
        args=argparse.Namespace(diagnose=False,timeout=30,report=None,
                                bootstrap_identity=None,agent_home=None,project_path=project,replace=replace)
        with patch.dict('os.environ',env),patch.object(Path,'home',return_value=self.home),patch.object(sys,'platform','darwin'),contextlib.redirect_stdout(io.StringIO()) as out:
            setup.setup(args)
        return out.getvalue()
    def calls(self):
        p=self.home/'calls.jsonl'
        return [json.loads(line) for line in p.read_text().splitlines()] if p.exists() else []

    def test_zcode_native_config_repeat_and_conflict(self):
        directory=self.home/'.zcode';(directory/'cli').mkdir(parents=True)
        config=directory/'cli/config.json'
        config.write_text(json.dumps({'theme':'dark','mcp':{'servers':{'other':{'command':'other'}}}}))
        (directory/'AGENTS.md').write_text('User preferences\n')
        self.run_setup('zcode')
        first=json.loads(config.read_text())
        self.assertEqual(first['theme'],'dark')
        self.assertEqual(first['mcp']['servers']['other'],{'command':'other'})
        self.assertEqual(first['mcp']['servers']['boxweave']['type'],'stdio')
        self.assertEqual(first['mcp']['servers']['boxweave']['args'],['boxweave','/usr/local/bin/boxweave-mcp'])
        self.assertTrue((directory/'AGENTS.md').read_text().startswith('User preferences'))
        self.assertIn(KEY,(directory/'AGENTS.md').read_text())
        self.run_setup('zcode')
        self.assertEqual(json.loads(config.read_text()),first)
        self.assertTrue(list(config.parent.glob('config.json.backup-*')))
        first['mcp']['servers']['boxweave']['enable']=False
        config.write_text(json.dumps(first))
        before=len(self.calls())
        with self.assertRaisesRegex(ValueError,'Existing ZCode'): self.run_setup('zcode')
        self.assertEqual(len(self.calls()),before)
        self.run_setup('zcode',replace=True)
        self.assertNotIn('enable',json.loads(config.read_text())['mcp']['servers']['boxweave'])
        self.assertFalse(any(c[0]=='claude' for c in self.calls()))

    def test_codex_complete_repeat_and_override(self):
        directory=self.home/'.codex';directory.mkdir()
        override=directory/'AGENTS.override.md';override.write_text('User override\n')
        out=self.run_setup()
        config=(directory/'config.toml').read_text()
        self.assertIn('passed',out)
        self.assertIn('User override',override.read_text())
        self.assertFalse((directory/'AGENTS.md').exists())
        self.assertIn("args = [\"boxweave\"",config)
        self.run_setup()
        self.assertEqual((directory/'config.toml').read_text(),config)
        generates=[a for a in self.calls() if a[0]=='ssh-keygen' and '-f' in a]
        self.assertEqual(len(generates),1)
    def test_ssh_config_block_written_and_stable_across_reruns(self):
        self.run_setup()
        config=self.home/'.ssh/config'
        first=config.read_text()
        self.assertIn('Host boxweave',first)
        self.assertIn('HostName lab.example',first)
        self.assertEqual(setup.parse_ssh_transport_block(first)['Port'],'2222')
        self.run_setup()
        self.assertEqual(config.read_text(),first)
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
        self.assertEqual(additions[0][-2:],['boxweave','/usr/local/bin/boxweave-mcp'])
        with self.assertRaisesRegex(ValueError,'Existing Claude'):self.run_setup('claude-code')
        self.run_setup('claude-code',replace=True)
    def test_bad_configuration_fails_before_keygen_or_enrollment(self):
        directory=self.home/'.codex';directory.mkdir()
        (directory/'config.toml').write_text('invalid !')
        with self.assertRaises(ValueError):self.run_setup()
        self.assertEqual(self.calls(),[])
    def test_changed_pin_same_host_fails_before_enrollment(self):
        sshdir=self.home/'.ssh';sshdir.mkdir()
        (sshdir/'boxweave_known_hosts').write_text('[lab.example]:2222 ssh-ed25519 DIFFERENTKEY')
        with self.assertRaisesRegex(ValueError,'Pinned server identity changed'):self.run_setup()
        self.assertEqual(self.calls(),[])
    def test_readdress_repins_and_continues(self):
        sshdir=self.home/'.ssh';sshdir.mkdir()
        (sshdir/'boxweave_known_hosts').write_text('[old.example]:2222 ssh-ed25519 OLDPIN')
        out=self.run_setup()
        self.assertIn('SSH address changed',out)
        pin=(sshdir/'boxweave_known_hosts').read_text()
        self.assertTrue(pin.startswith('[lab.example]:2222 '))
        self.assertIn(KEY.split()[1],pin)
    def test_incomplete_identity_fails_before_enrollment(self):
        sshdir=self.home/'.ssh';sshdir.mkdir()
        (sshdir/'boxweave').write_text('OPAQUE FIXTURE')
        with self.assertRaisesRegex(ValueError,'Incomplete'):self.run_setup()
        self.assertEqual(self.calls(),[])
    def test_invalid_env_input_fails_closed(self):
        for override in ({'BOXWEAVE_SSH_HOST':'bad host; rm'},{'BOXWEAVE_SSH_PORT':'99999'},
                         {'BOXWEAVE_SSH_PORT':''},{'BOXWEAVE_SSH_USER':'Bad'},
                         {'BOXWEAVE_SERVER_HOST_KEY':'ssh-rsa AAAA'},
                         {'BOXWEAVE_CONTAINER_PUBLIC_KEY':'not a key'}):
            with self.subTest(override=override),self.assertRaises(ValueError):
                self.run_setup(env_overrides=override)
        self.assertEqual(self.calls(),[])

class GLMTests(unittest.TestCase):
    def test_merge_preserves_settings_and_maps_models(self):
        original=json.dumps({'permissions': {'allow': ['Read']}, 'env': {
            'CUSTOM': 'keep', 'ANTHROPIC_API_KEY': 'old'}})
        for provider, url in [('zai','https://api.z.ai/api/anthropic'),
                              ('zhipu','https://open.bigmodel.cn/api/anthropic')]:
            result=json.loads(setup.glm_settings(original,provider,'glm-test','fixture-token'))
            self.assertEqual(result['permissions'], {'allow': ['Read']})
            self.assertEqual(result['env']['CUSTOM'],'keep')
            self.assertEqual(result['env']['ANTHROPIC_BASE_URL'],url)
            self.assertEqual(result['env']['ANTHROPIC_AUTH_TOKEN'],'fixture-token')
            self.assertNotIn('ANTHROPIC_API_KEY',result['env'])
            for tier in ('OPUS','SONNET','HAIKU'):
                self.assertEqual(result['env']['ANTHROPIC_DEFAULT_'+tier+'_MODEL'],'glm-test')

    def test_invalid_settings_and_missing_key_fail_closed(self):
        for original,provider,model,token in [
            ('[]','zai',None,'key'), ('{"env":[]}','zai',None,'key'),
            ('{}','other',None,'key'), ('{}','zai',None,''),
            ('{}','zai','bad model','key'), ('{}','zai',None,'key\n')]:
            with self.assertRaises(ValueError):
                setup.glm_settings(original,provider,model,token)

class ZCodeConfigTests(unittest.TestCase):
    def test_invalid_shapes_disabled_mcp_and_custom_entry(self):
        for original in ('[]','null','{"mcp":[]}','{"mcp":{"servers":null}}',
                         '{"features":{"mcp":false}}',
                         '{"mcp":{"servers":{"boxweave":{"command":"other"}}}}'):
            with self.subTest(original=original),self.assertRaises(ValueError):
                setup.zcode_update(original,'ssh',['boxweave','/usr/local/bin/boxweave-mcp'])
    def test_only_zcode_has_native_config_shape(self):
        result=json.loads(setup.zcode_update('{}','ssh',['boxweave','/usr/local/bin/boxweave-mcp']))
        self.assertNotIn('mcpServers',result)
        self.assertEqual(result['mcp']['servers']['boxweave']['args'],['boxweave','/usr/local/bin/boxweave-mcp'])
        self.assertEqual(setup.zcode_update(json.dumps(result),'ssh',['boxweave','/usr/local/bin/boxweave-mcp']),
                         json.dumps(result,indent=2)+'\n')

if __name__=='__main__': unittest.main()
