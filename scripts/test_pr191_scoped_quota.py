"""Source-only refusal regression; never execute a quota workload."""
import os
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[1]


class QuotaBoundaryTests(unittest.TestCase):
    def test_non_hosted_environment_refuses(self):
        env = dict(os.environ, GITHUB_ACTIONS="false", RUNNER_ENVIRONMENT="local")
        for name in ("s2_ext4_quota_pool_lab.sh", "s2_mcp_quota_witness.sh",
                     "pr191_scoped_s2_quota.sh", "pr191_scoped_s2_direct_quota.sh",
                     "s2_native_guest_quota_witness.sh"):
            with self.subTest(name=name):
                result = subprocess.run(["sh", "-e", str(ROOT / "scripts" / name)],
                                        env=env, capture_output=True, timeout=5)
                self.assertNotEqual(result.returncode, 0)

    def test_fixture_source_extraction_preserves_native_oracles(self):
        old = (ROOT / "scripts/accept_s2_native.sh").read_text()
        setup = (ROOT / "scripts/s2_ext4_quota_pool_lab.sh").read_text()
        witness = (ROOT / "scripts/s2_mcp_quota_witness.sh").read_text()
        installed = (ROOT / "scripts/pr191_scoped_s2_quota.sh").read_text()
        n2 = (ROOT / "scripts/n2_installed_lifecycle.sh").read_text()
        for name in ("s2_ext4_quota_pool_lab.sh", "s2_mcp_quota_witness.sh"):
            self.assertIn(". scripts/" + name, old)
        self.assertIn("S2_NATIVE_EXT4_PROJECT_QUOTA_BACKING_CONFIGURED=PASS", setup)
        self.assertIn("S2_MCP_REAL_GUEST_QUOTA_ORIGINAL_RECEIPT_OUTPUT_EXIT_ONE=PASS", witness)
        self.assertIn("S2_MCP_NATIVE_OUTPUT_404_TYPED_WITHOUT_FALSE_EXPIRY=PASS", witness)
        self.assertIn(". scripts/s2_mcp_quota_witness.sh", installed)
        self.assertLess(n2.index(". scripts/s2_ext4_quota_pool_lab.sh"),
                        n2.index("sudo sh install.sh --plan --scoped-peer-fresh"))
        self.assertIn("P1B_SCOPED_QUOTA_FIXED_LEDGER_AND_NATIVE_ABSENCE=PASS", installed)
        direct = (ROOT / "scripts/pr191_scoped_s2_direct_quota.sh").read_text()
        native = (ROOT / "scripts/s2_native_guest_quota_witness.sh").read_text()
        self.assertIn(". scripts/s2_native_guest_quota_witness.sh", old)
        self.assertIn(". scripts/s2_native_guest_quota_witness.sh", direct)
        self.assertIn(". scripts/pr191_scoped_s2_direct_quota.sh", installed)
        self.assertIn("S2_NATIVE_REAL_GUEST_RECORDED_EXEC_QUOTA_PARTIAL_OUTPUT=PASS", native)
        self.assertIn("P1B_SCOPED_DIRECT_NATIVE_NO_SYNTHETIC_MCP_RECEIPTS=PASS", direct)
        self.assertNotIn("incus delete", direct)
        self.assertIn("quota_mcp cleanup", direct)


if __name__ == "__main__":
    unittest.main()
