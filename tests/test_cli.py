import contextlib
import copy
import importlib.util
import io
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('boxweave_cli', Path(__file__).resolve().parents[1] / 'server/cli.py')
cli = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cli)


class CLITests(unittest.TestCase):
    def instance(self):
        return {'name': 'literal-restore', 'status': 'Stopped', 'type': 'container',
                'created_at': '2026-01-01', 'config': {'volatile.uuid': 'cloned-uuid'},
                'expanded_config': {'security.nesting': 'true', 'environment.PASSWORD': 'NEVER_RENDER_ME',
                                    'raw.lxc': 'sensitive-config-content'},
                'expanded_devices': {'eth0': {'type': 'none'}, 'root': {'type': 'disk', 'pool': 'default', 'path': '/', 'source': 'PRIVATE_SOURCE'}},
                'devices': {}, 'profiles': ['default']}

    def inspect(self, data):
        with patch.object(cli, 'command', return_value={'exit': 0, 'stdout': json.dumps(data), 'error': None}) as run:
            result, code = cli.inspect_box('literal-restore', 'lab')
        return result, code, run.call_args.args[0]

    def test_version_does_not_contact_provider(self):
        with patch.object(cli, 'command', side_effect=AssertionError('No command expected')), contextlib.redirect_stdout(io.StringIO()) as out:
            self.assertEqual(cli.main(['version', '--json']), 0)
        self.assertEqual(json.loads(out.getvalue())['boxweave'], (Path(__file__).resolve().parents[1] / 'VERSION').read_text().strip())

    def test_exact_name_explicit_project_force_local(self):
        data, code, command = self.inspect(self.instance())
        self.assertEqual(code, 0)
        self.assertEqual(command, ['incus', '--force-local', 'query', '/1.0/instances/literal-restore?project=lab'])
        self.assertTrue(any('No NIC' in x for x in data['notes']))

    def test_metadata_does_not_leak_secret_fields(self):
        result, _, _ = self.inspect(self.instance())
        text = json.dumps(result)
        for secret in ('NEVER_RENDER_ME', 'sensitive-config-content', 'PRIVATE_SOURCE'):
            self.assertNotIn(secret, text)
        self.assertTrue(result['raw_config']['raw.lxc']['present'])

    def test_cloned_uuid_is_not_configuration_identity(self):
        original = self.instance()
        clone = copy.deepcopy(original)
        clone['expanded_config']['boot.autostart'] = 'true'
        a, _, _ = self.inspect(original)
        b, _, _ = self.inspect(clone)
        self.assertEqual(a['uuid'], b['uuid'])
        self.assertNotEqual(a['config_sha256'], b['config_sha256'])

    def test_wrong_returned_identity_rejected(self):
        data = self.instance(); data['name'] = 'other'
        result, code, _ = self.inspect(data)
        self.assertEqual(code, 1)
        self.assertEqual(result['error'], 'invalid_instance_metadata')

    def test_failed_exact_query_never_tries_other_name(self):
        with patch.object(cli, 'command', return_value={'exit': 1, 'stdout': '', 'error': 'nonzero_exit'}) as run:
            result, code = cli.inspect_box('literal-restore', 'default')
        self.assertEqual(code, 1); run.assert_called_once()
        self.assertNotIn('start', run.call_args.args[0])

    def test_invalid_name_never_executes(self):
        with patch.object(cli, 'command') as run:
            with self.assertRaises(ValueError):cli.inspect_box('bad;name', 'default')
        run.assert_not_called()

    def test_starting_is_not_failed(self):
        with patch.object(cli, 'command', return_value={'exit': 0, 'stdout': 'LoadState=loaded\nActiveState=activating\nSubState=start\n', 'error': None}):
            self.assertEqual(cli.service_state('incus.service')['status'], 'starting')

    def test_missing_service_is_not_healthy(self):
        with patch.object(cli, 'command', return_value={'exit': 0, 'stdout': 'LoadState=not-found\nActiveState=inactive\n', 'error': None}):
            self.assertEqual(cli.service_state('incus.service')['status'], 'missing')

    def test_nested_unconfined_not_used_as_full_policy_verdict(self):
        def read(path, *args, **kwargs):
            if str(path).endswith('current'):return 'unconfined'
            return 'yes' if str(path).endswith(('.stacked', '.ns_stacked')) else '1'
        with patch.object(cli, 'command', return_value={'exit': 0, 'stdout': 'lxc\n', 'error': None}), patch.object(Path, 'read_text', read):
            result = cli.environment()
        self.assertTrue(result['container'])
        self.assertEqual(result['apparmor']['effective_device_filtering'], 'not_verified')
        self.assertTrue(any('Host-side' in x for x in result['notes']))

    def test_nonlocal_or_credential_api_never_probed(self):
        with patch.object(cli, 'build_opener') as opener:
            for uri in ('http://example.com', 'http://user:password@127.0.0.1:8080', 'http://127.0.0.1/a?token=secret'):
                self.assertEqual(cli.api_reachability({'api_url': uri})['status'], 'skipped')
        opener.assert_not_called()

    def test_auth_response_means_reachable_not_authenticated_health(self):
        from urllib.error import HTTPError
        with patch.object(cli, 'build_opener') as factory:
            factory.return_value.open.side_effect = HTTPError('http://127.0.0.1', 401, 'Unauthorized', {}, io.BytesIO(b'SECRET_BODY'))
            result = cli.api_reachability({'api_url': 'http://127.0.0.1:8080'})
        self.assertEqual(result['status'], 'reachable')
        self.assertEqual(result['authenticated_health'], 'not_verified')
        self.assertNotIn('SECRET_BODY', json.dumps(result))

    def test_configuration_is_not_dumped(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'config.json'
            path.write_text(json.dumps({'provider': 'containarium', 'schema_version': 1, 'jwt': 'SECRET_JWT'}))
            def healthy(unit):
                if unit in cli.ONESHOTS:
                    return {'status':'inactive','state':{'Result':'success','ExecMainStatus':'0','ExecMainStartTimestampMonotonic':'1'}}
                return {'status':'active','state':{'UnitFileState':'enabled'}}
            with patch.object(cli, 'service_state', side_effect=healthy), contextlib.redirect_stdout(io.StringIO()) as out:
                code = cli.main(['status', '--config', str(path), '--json'])
            self.assertEqual(code, 0);self.assertNotIn('SECRET_JWT', out.getvalue())

    def test_pending_update_never_reports_ready(self):
        services={unit:{'status':'active','state':{'UnitFileState':'enabled'}} for unit in cli.SERVICES}
        oneshots={unit:{'status':'inactive','state':{'Result':'success','ExecMainStatus':'0','ExecMainStartTimestampMonotonic':'1'}} for unit in cli.ONESHOTS}
        self.assertEqual(cli.installation_state(True,services,oneshots,False)['state'],'ready')
        self.assertEqual(cli.installation_state(True,services,oneshots,True)['state'],'incomplete')
        services['boxweave-token.timer']['state']['UnitFileState']='disabled'
        self.assertEqual(cli.installation_state(True,services,oneshots,False)['state'],'incomplete')

    def test_starting_and_failed_have_distinct_states(self):
        services={unit:{'status':'active','state':{'UnitFileState':'enabled'}} for unit in cli.SERVICES}
        oneshots={unit:{'status':'inactive','state':{'Result':'success','ExecMainStatus':'0','ExecMainStartTimestampMonotonic':'1'}} for unit in cli.ONESHOTS}
        services['containarium.service']['status']='starting'
        self.assertEqual(cli.installation_state(True,services,oneshots,False)['state'],'starting')
        services['containarium.service']['status']='failed'
        self.assertEqual(cli.installation_state(True,services,oneshots,False)['state'],'failed')
        services['containarium.service']['status']='active'
        oneshots['boxweave-token.service']['state']['ExecMainStartTimestampMonotonic']='0'
        self.assertEqual(cli.installation_state(True,services,oneshots,False)['state'],'incomplete')

    def test_command_time_budget(self):
        result = cli.command([sys.executable, '-c', 'import time;time.sleep(30)'], timeout=0.1)
        self.assertEqual(result['error'], 'timeout')
        self.assertEqual(result['stdout'], '')

    def test_command_output_budget(self):
        result = cli.command([sys.executable, '-c', 'print("x"*1000000)'], output_limit=1024)
        self.assertEqual(result['error'], 'output_limit')
        self.assertEqual(result['stdout'], '')


if __name__ == '__main__':unittest.main()
