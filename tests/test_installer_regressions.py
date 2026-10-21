"""Regression checks for issues found on fresh Ubuntu 26.04 installations.

These are non-destructive contract checks. They do not replace an actual clean
Ubuntu integration test against Incus and Containarium.
"""
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = (ROOT / 'install.sh').read_text()


class InstallerRegressionTests(unittest.TestCase):
    def test_installer_parses(self):
        subprocess.run(['bash', '-n', str(ROOT / 'install.sh')], check=True)

    def test_upstream_apt_writes_do_not_inherit_private_umask(self):
        self.assertIn('umask 077', SCRIPT)
        self.assertIn('(umask 022; bash "$TMP_INSTALL")', SCRIPT)
        self.assertNotIn('bash "$tmp_install"', SCRIPT)

    def test_previous_incomplete_apt_configuration_is_verified_before_repair(self):
        self.assertIn('4EFC590696CB15B87C73A3AD82CC8797C838DCFD', SCRIPT)
        self.assertIn('chmod 0644 "$keyring"', SCRIPT)
        self.assertIn('chmod 0644 "$sourcelist"', SCRIPT)
        self.assertIn('chmod 0644 "$preferences"', SCRIPT)
        self.assertLess(SCRIPT.index('repair_zabbly_apt_permissions\nif ! command -v containarium'),
                        SCRIPT.index('(umask 022; bash "$TMP_INSTALL")'))

    def test_uninitialized_existing_incus_is_fixed_without_touching_containers(self):
        self.assertIn('ensure_incus_initialized()', SCRIPT)
        self.assertIn('incus admin init --auto', SCRIPT)
        self.assertIn("Incus has containers but no storage pool; manual review required", SCRIPT)
        self.assertIn('ensure_incus_initialized\n', SCRIPT)

    def test_first_boot_has_configurable_readiness_wait_with_progress(self):
        self.assertIn('BOXWEAVE_API_READY_TIMEOUT:-900', SCRIPT)
        self.assertIn('--api-ready-timeout', SCRIPT)
        self.assertIn('systemctl is-active --quiet containarium.service', SCRIPT)
        self.assertIn('Core containers may still be initializing.', SCRIPT)
        self.assertIn('REST API did not become ready within', SCRIPT)
        self.assertNotIn('seq 1 90', SCRIPT)

    def test_cleanup_and_phase_error_reporting(self):
        for snippet in ('trap cleanup EXIT', 'trap report_failure ERR',
                        "INSTALL_PHASE='provider installation'",
                        "INSTALL_PHASE='Containarium REST API readiness'",
                        "INSTALL_PHASE='BoxWeave service verification'"):
            with self.subTest(snippet=snippet):
                self.assertIn(snippet, SCRIPT)


if __name__ == '__main__':
    unittest.main()
