"""Exercise client configuration helpers using only disposable files."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[1]
PWSH=os.environ.get('BOXWEAVE_TEST_PWSH') or shutil.which('pwsh')

@unittest.skipUnless(PWSH,'Requires a PowerShell runtime for configuration fixtures')
class WindowsConfigTests(unittest.TestCase):
    def run_case(self,case):
        with tempfile.TemporaryDirectory(prefix='boxweave-config-') as tmp:
            result=subprocess.run([PWSH,'-NoLogo','-NoProfile','-NonInteractive','-File',
                str(ROOT/'tests/windows_config.ps1'),'-SetupPath',str(ROOT/'clients/windows/setup.ps1'),
                '-FixtureRoot',tmp,'-Case',case],capture_output=True,text=True,timeout=20)
            self.assertEqual(result.returncode,0,result.stdout+result.stderr)
            data=json.loads(result.stdout)
            self.assertEqual(data['result'],'passed')
            self.assertEqual(data['checks'],6)
    def test_create_replace_backup_and_cleanup(self):
        self.run_case('atomic')
    def test_codex_and_managed_rules_publication(self):
        self.run_case('adapters')

if __name__=='__main__':unittest.main()
