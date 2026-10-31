"""No-host management CLI contracts."""
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from bwmini import manage

class ManagementTests(unittest.TestCase):
    def test_version_reads_only_bounded_version(self):
        with tempfile.TemporaryDirectory() as root:
            p = Path(root) / 'VERSION'
            p.write_text('0.11.0\n')
            self.assertEqual(manage.version_from(root), '0.11.0')
            p.write_text('suspicious\n')
            self.assertEqual(manage.version_from(root), 'unknown')
    def test_exact_inspection_rejects_unsafe_names_without_subprocess(self):
        with patch.object(manage, 'command') as command:
            for name in ('../etc', 'thing;whoami', '', 'one/two'):
                with self.assertRaises(ValueError): manage.inspect_box(name, 'default')
            command.assert_not_called()
    def test_inspect_does_not_return_raw_config_or_secrets(self):
        response = {"name":"box1","status":"Running","type":"container",
                    "expanded_config":{"security.nesting":"true","user.secret":"PRIVATE"},
                    "expanded_devices":{"eth0":{"nictype":"bridged"}}}
        with patch.object(manage, 'command', return_value={"state":"ok","text":json.dumps(response)}):
            outcome, code = manage.inspect_box('box1', 'default')
        self.assertEqual(code, 0)
        self.assertEqual(outcome['security']['security.nesting'], 'true')
        self.assertNotIn('PRIVATE', json.dumps(outcome))
    def test_client_delegates_to_single_setup_generator(self):
        with patch('bwmini.client_setup.main', return_value=0) as generator:
            code = manage.main(['client', '--agent', 'codex', '--transport', 'wsl'])
        self.assertEqual(code, 0)
        generator.assert_called_once_with(['--agent', 'codex', '--transport', 'wsl'])
    def test_missing_install_is_not_ready(self):
        with patch.object(manage, 'app_health', return_value={"state":"starting"}):
            with patch('sys.stdout', new_callable=io.StringIO):
                self.assertEqual(manage.main(['status','--prefix','/tmp/not-installed']), 2)
    def test_doctor_carries_overall_and_unverified_coverage(self):
        with patch.object(manage, 'app_health', return_value={"state":"starting"}) as health, \
             patch('sys.stdout', new_callable=io.StringIO) as out:
            self.assertEqual(manage.main(['doctor','--prefix','/tmp/not-installed']), 2)
        outcome = json.loads(out.getvalue())
        health.assert_called_once_with('/tmp/not-installed', mcp=False)
        self.assertEqual(outcome['overall'], 'starting')
        self.assertEqual(outcome['diagnostic_schema'], 1)
        self.assertEqual(outcome['coverage'], {'workload_execute': 'not_verified',
                                               'parent_host_policy': 'not_verified',
                                               'agent_ui': 'not_verified',
                                               'model_inference': 'not_verified'})
        self.assertEqual(outcome['host_observations']['effective_confinement'], 'not_verified')
    def test_doctor_mcp_flag_reaches_app_health(self):
        with patch.object(manage, 'app_health', return_value={"state":"failed"}) as health, \
             patch('sys.stdout', new_callable=io.StringIO):
            self.assertEqual(manage.main(['doctor','--prefix','/tmp/none','--mcp']), 1)
        health.assert_called_once_with('/tmp/none', mcp=True)
if __name__ == '__main__': unittest.main()
