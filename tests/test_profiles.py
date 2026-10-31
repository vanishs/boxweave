"""Profile preservation and protocol failures against real child processes, not live claims."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('profiles',Path(__file__).parents[1]/'clients/profile_client.py')
p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)

class Profiles(unittest.TestCase):
    def test_codex_preserves_other_config_and_tool_approval(self):
        text='model="keep"\n[mcp_servers.other]\ncommand="other"\n[mcp_servers.boxweave]\ncommand="old"\nargs=[]\n[mcp_servers.boxweave.tools.delete_container]\napproval="prompt"\n'
        with self.assertRaisesRegex(p.Refused,'REGISTRATION_CONFLICT'):p.codex_merge(text,['new','���� folder'])
        result=p.codex_merge(text,['new','���� folder'],True)
        obj=p.tomllib.loads(result)
        self.assertEqual(obj['model'],'keep');self.assertEqual(obj['mcp_servers']['other']['command'],'other')
        self.assertEqual(obj['mcp_servers']['boxweave']['tools']['delete_container']['approval'],'prompt')
        self.assertEqual(p.codex_merge(result,['new','���� folder']),result)
    def test_quoted_inline_or_multiline_transport_fails_closed(self):
        for text in ['[mcp_servers."boxweave"]\ncommand="old"\n', 'mcp_servers={boxweave={command="old"}}', '[mcp_servers.boxweave]\nargs=[\n"old"\n]\n']:
            with self.assertRaises(p.Refused):p.codex_merge(text,['new'],True)
    def test_zcode_workspace_preserves_unrelated_and_conflict(self):
        original=json.dumps({'editor':{'wrap':True},'mcp':{'servers':{'other':{'command':'keep'},'boxweave':{'command':'old','env':{'SYNTHETIC_SECRET':'do-not-print'}}}}})
        with self.assertRaisesRegex(p.Refused,'REGISTRATION_CONFLICT'):p.zcode_merge(original,['new'])
        result=p.zcode_merge(original,['new','���� folder'],True);v=json.loads(result)
        self.assertTrue(v['editor']['wrap']);self.assertEqual(v['mcp']['servers']['other'],{'command':'keep'})
        self.assertNotIn('SYNTHETIC_SECRET',result);self.assertEqual(p.zcode_merge(result,['new','���� folder']),result)
    def test_wsl_argv_explicit_distro_posix_null_and_no_shell(self):
        d={'profile':'codex/windows/wsl','connection':{'host':'server.example','port':22,'user':'trusted','identity_file':'/home/me/���� folder/key','known_hosts':'/home/me/���� folder/known','server_key':'ssh-ed25519 AAAA','executable':str(Path(sys.executable).resolve()),'distribution':'Ubuntu'}}
        a=p.ssh_argv(d,False)
        self.assertEqual(a[1:5],['--distribution','Ubuntu','--exec','/usr/bin/ssh'])
        self.assertIn('/dev/null',a);self.assertNotIn('sh',a);self.assertIn('/home/me/���� folder/key',a)
        self.assertEqual(a.count('ServerAliveInterval=15'),1)
        self.assertEqual(a.count('ServerAliveCountMax=4'),1)
        self.assertEqual(a.index('ServerAliveInterval=15')+2,a.index('ServerAliveCountMax=4'))
        del d['connection']['distribution']
        with self.assertRaises(p.Refused):p.ssh_argv(d,False)
    def test_native_registration_argv_preserves_keepalive(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder);(root/'key').touch();(root/'known').write_text('pin')
            d={'profile':'codex/windows/native','connection':{
                'host':'server.example','port':22,'user':'trusted',
                'identity_file':str(root/'key'),'known_hosts':str(root/'known'),
                'server_key':'ssh-ed25519 AAAA',
                'executable':str(Path(sys.executable).resolve())}}
            argv=p.ssh_argv(d,False)
            self.assertEqual(argv.count('ServerAliveInterval=15'),1)
            self.assertEqual(argv.count('ServerAliveCountMax=4'),1)
            config=p.tomllib.loads(p.codex_merge('',argv))
            saved=config['mcp_servers']['boxweave']['args']
            self.assertEqual(saved,argv)
            self.assertEqual(saved.count('ServerAliveInterval=15'),1)
            self.assertEqual(saved.count('ServerAliveCountMax=4'),1)

    def test_pin_mismatch_fails_before_launch(self):
        with tempfile.TemporaryDirectory() as folder:
            f=Path(folder);(f/'key').touch();(f/'known').write_text('wrong')
            d={'profile':'codex/windows/native','connection':{'host':'server.example','port':22,'user':'trusted','identity_file':str(f/'key'),'known_hosts':str(f/'known'),'server_key':'ssh-ed25519 AAAA','executable':str(Path(sys.executable).resolve())}}
            with self.assertRaisesRegex(p.Refused,'PIN_FILE_MISMATCH'):p.ssh_argv(d)
    def test_actual_eof_and_deadline_close_owned_process(self):
        for code,expected in [('pass','STREAM_CLOSED'),('import time;time.sleep(60)','PROTOCOL_DEADLINE')]:
            client=p.RPC([sys.executable,'-c',code],timeout=.15)
            with client,self.assertRaisesRegex(p.Refused,expected):client.call('read',{})
            self.assertIsNotNone(client.p.poll());self.assertFalse(client.thread.is_alive())
    def test_mcp_strict_jsonrpc_and_explicit_zcode_protocol(self):
        code='import sys,json; r=json.loads(sys.stdin.readline());print(json.dumps({"id":r["id"],"result":{"ok":True}}),flush=True)'
        with p.RPC([sys.executable,'-c',code],timeout=2) as c,self.assertRaisesRegex(p.Refused,'PROTOCOL_INVALID'):c.call('read',{})
        with p.RPC([sys.executable,'-c',code],timeout=2,zcode=True) as c:self.assertTrue(c.call('read',{})['ok'])
    def test_jsonrpc_wrong_id_and_output_limit(self):
        for code,expected in [('print(\'{"jsonrpc":"2.0","id":99,"result":{}}\',flush=True)','PROTOCOL_INVALID'),('print("x"*2097153,flush=True)','OUTPUT_LIMIT')]:
            with p.RPC([sys.executable,'-c',code],timeout=2) as c,self.assertRaisesRegex(p.Refused,expected):c.call('read',{})
    def test_atomic_backup_and_no_change_on_conflict(self):
        with tempfile.TemporaryDirectory() as folder:
            d={'profile':'zcode/windows/native','profile_directory':folder}
            first=p.write_config(d,['new']);self.assertTrue(first['changed']);config=p.config_path(d);before=config.read_bytes()
            self.assertFalse(p.write_config(d,['new'])['changed'])
            with self.assertRaisesRegex(p.Refused,'REGISTRATION_CONFLICT'):p.write_config(d,['other'])
            self.assertEqual(config.read_bytes(),before)
            changed=p.write_config(d,['other'],True);self.assertEqual(Path(changed['backup']).read_bytes(),before)
            self.assertFalse(list(config.parent.glob('.boxweave-stage-*')))
    def test_agent_environment_excludes_secrets_keeps_windows_ssh_runtime(self):
        with patch.dict(p.os.environ,{'API_KEY':'synthetic','PROGRAMDATA':'public-system-dir','ALLUSERSPROFILE':'public-system-dir'},clear=True):
            e=p.agent_env({'profile':'zcode/windows/native','profile_directory':'fresh','zcode_builtin':{'path':'reviewed-public'}})
            self.assertNotIn('API_KEY',e);self.assertEqual(e['PROGRAMDATA'],'public-system-dir')
            self.assertEqual(e['ZCODE_DATA_BASE_DIR'],'fresh')

    def test_program_output_and_deadline_are_bounded(self):
        with self.assertRaisesRegex(p.Refused,'OUTPUT_LIMIT'):p.program([sys.executable,'-c','print("x"*4194305)'])
        with self.assertRaisesRegex(p.Refused,'PROCESS_DEADLINE'):p.program([sys.executable,'-c','import time;time.sleep(60)'],timeout=.1)
    def test_duplicate_json_config_is_not_silently_rewritten(self):
        with self.assertRaisesRegex(p.Refused,'DUPLICATE_JSON_KEY'):p.zcode_merge('{"mcp":{},"mcp":{}}',['new'])
    def test_zcode_registration_mismatch_before_agent_connection(self):
        with tempfile.TemporaryDirectory() as folder:
            d={'profile':'zcode/windows/native','profile_directory':folder,'zcode_builtin':{'path':'public'},'agent_argv':['selected'],'agent_version':'0.17.0'}
            p.write_config(d,['old'])
            with patch.object(p,'program',return_value='0.17.0'),patch.object(p,'RPC',side_effect=AssertionError('must not launch MCP')):
                with self.assertRaisesRegex(p.Refused,'REGISTRATION_MISMATCH'):p.registration(d,['new'])
