"""Real PowerShell diagnostic entry/protocol tests with isolated fixtures."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[1]
PWSH=os.environ.get('BOXWEAVE_TEST_PWSH') or shutil.which('pwsh')

@unittest.skipUnless(PWSH and os.name=='nt','Requires Windows PowerShell/OpenSSH for native diagnostic fixtures')
class WindowsDiagnosticsTests(unittest.TestCase):
    def test_native_diagnostic_contract(self):
        for case in ('matched','missing','pin','invalid','replace','report','existing_report'):
            with self.subTest(case=case),tempfile.TemporaryDirectory(prefix='boxweave-diag-') as tmp:
                p=subprocess.run([PWSH,'-NoProfile','-NonInteractive','-File',str(ROOT/'tests/windows_diagnostics.ps1'),
                                  '-SetupPath',str(ROOT/'clients/windows/setup.ps1'),'-FixtureRoot',tmp,'-Case',case],
                                 capture_output=True,text=True,timeout=20)
                self.assertEqual(p.returncode,0,p.stdout+p.stderr)
                self.assertEqual(json.loads(p.stdout)['result'],'passed')

    def test_real_native_protocol_process(self):
        from test_diagnostics import PEER
        import sys
        for case in ('valid','second_page','empty_result','denied','bad_content','bad_error_type','wrong_id','invalid_json','cursor_loop','protocol','missing_capabilities','oversized','early_exit','blocked','write_blocked'):
            with self.subTest(case=case),tempfile.TemporaryDirectory(prefix='boxweave-probe-') as tmp:
                peer=Path(tmp)/'peer.py';peer.write_text(PEER)
                env=dict(os.environ,DIAGNOSTIC_CASE=case)
                p=subprocess.run([PWSH,'-NoProfile','-NonInteractive','-File',str(ROOT/'tests/windows_probe.ps1'),
                                  '-SetupPath',str(ROOT/'clients/windows/setup.ps1'),'-Python',sys.executable,'-Peer',str(peer)],
                                 capture_output=True,text=True,timeout=15,env=env)
                self.assertEqual(p.returncode,0,p.stdout+p.stderr)
                data=json.loads(p.stdout)
                self.assertEqual(data['accepted'],case in ('valid','second_page'),data)
                self.assertNotEqual(data['code'],'UNCLASSIFIED',data)
                self.assertEqual(data['cleanup'],'completed',data)

if __name__=='__main__':unittest.main()
