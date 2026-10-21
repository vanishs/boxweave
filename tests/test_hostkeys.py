import importlib.util
import ipaddress
from pathlib import Path
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[1] / 'server' / 'hostkeys.py'
spec = importlib.util.spec_from_file_location('knownhosts_sync', SOURCE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class HostKeySyncTests(unittest.TestCase):
    def box(self, name='example-container'):
        return {
            'name': name, 'status': 'Running',
            'config': {'volatile.uuid': 'unique', 'user.containarium.label.os-type': 'ubuntu',
                       'user.containarium.sshd_policy': 'yes'},
            'state': {'network': {'eth0': {'addresses': [
                {'family': 'inet', 'scope': 'global', 'address': '10.143.82.2'}
            ]}}},
        }

    def test_project_container_identification(self):
        self.assertTrue(module.project(self.box()))
        self.assertFalse(module.project(self.box('containarium-core-test-container')))
        b = self.box()
        del b['config']['user.containarium.sshd_policy']
        self.assertFalse(module.project(b))
        b = self.box()
        b['status'] = 'Stopped'
        self.assertFalse(module.project(b))

    def test_network_is_not_hardcoded(self):
        b = self.box()
        allowed = [ipaddress.ip_network('10.143.82.0/24')]
        self.assertEqual(module.ip_for(b, allowed), '10.143.82.2')
        with self.assertRaises(ValueError):
            module.ip_for(b, [ipaddress.ip_network('192.168.42.0/24')])

    def test_auto_subnet_discovery(self):
        network_json = [{'name':'incusbr0','type':'bridge','managed':True,
                         'config':{'ipv4.address':'10.143.82.1/24'}}]
        with patch.object(module, 'as_json', return_value=network_json):
            self.assertEqual(module.get_ipv4_networks({}), [ipaddress.ip_network('10.143.82.0/24')])

    def test_reject_nonprivate_networks(self):
        with self.assertRaises(ValueError):
            module.get_ipv4_networks({'allowed_container_cidrs':['8.8.8.0/24']})

    def test_missing_known_hosts_is_new(self):
        self.assertEqual(module.existing_keys('fake-user', Path('/nonexistent/nope'), '10.1.2.3'), set())

    def test_full_set_is_required(self):
        self.assertEqual(set(module.TYPES), {'ed25519', 'ecdsa', 'rsa'})


if __name__ == '__main__':
    unittest.main()
