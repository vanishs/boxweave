"""The host entry must stay inert for help and invalid release digests."""
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / 'install.sh'

class BootstrapSafetyTests(unittest.TestCase):
    def test_no_args_reject_nonroot_before_download(self):
        p = subprocess.run(['bash', str(SCRIPT)], capture_output=True, text=True, timeout=5)
        self.assertEqual(p.returncode, 2)
        self.assertIn('DEFAULT_INSTALL_REQUIRES_ROOT', p.stderr)
        self.assertEqual(p.stdout, '')

    def test_shell_syntax(self):
        self.assertEqual(subprocess.run(['bash','-n',str(SCRIPT)], capture_output=True).returncode,0)
    def test_help_has_no_host_writes(self):
        p = subprocess.run(['bash',str(SCRIPT),'--help'],capture_output=True,text=True,timeout=5)
        self.assertEqual(p.returncode,0)
        self.assertIn('plan|apply',p.stdout)
    def test_missing_or_invalid_digest_never_reaches_install(self):
        p = subprocess.run(['bash',str(SCRIPT),'apply','--app-user','nobody',
                            '--archive','/tmp/not-a-release','--sha256','bad',
                            '--config','/tmp/not-config'],capture_output=True,text=True,timeout=5)
        self.assertEqual(p.returncode,2)
        self.assertIn('SHA256',p.stderr)

    def test_package_alias_is_recognized_without_executing(self):
        p = subprocess.run(['bash', str(SCRIPT), 'apply', '--app-user', 'nobody',
                            '--package', '/tmp/nonexistent-boxweave-package',
                            '--sha256', 'bad', '--config', '/tmp/nonexistent-config'],
                           capture_output=True, text=True, timeout=5)
        self.assertEqual(p.returncode, 2)
        self.assertIn('SHA256', p.stderr)

    def test_missing_distribution_digest_rejected_without_execution(self):
        p = subprocess.run(['bash',str(SCRIPT),'apply','--app-user','nobody',
                            '--archive','/tmp/not-a-release','--sha256','0'*64,
                            '--config','/tmp/not-config'],capture_output=True,text=True,timeout=5)
        self.assertEqual(p.returncode,2)
        self.assertIn('distribution-sha256',p.stderr)

if __name__ == '__main__':
    unittest.main()
