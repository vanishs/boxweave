"""Conservative selection and local evidence binding regression."""
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
from harness import verification
from harness.common import HarnessError


class VerificationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "repo"
        self.root.mkdir()
        self.git("init", "-q")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "commit", "--allow-empty", "-qm", "fixture")
        self.directory = Path(self.tmp.name) / "evidence"

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.root), *args], text=True).strip()

    def test_missing_base_conservative(self):
        result = verification.plan(self.root)
        self.assertEqual(result["scope"], "FULL_OR_UNKNOWN")
        self.assertIsNone(result["changed_paths"])
        self.assertIn("quality-gate", result["required_gate"])

    def test_changed_paths_include_dirty_and_untracked(self):
        self.git("update-ref", "refs/remotes/origin/v1", "HEAD")
        (self.root / "tracked").write_text("initial")
        self.git("add", "tracked")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "commit", "-qm", "tracked")
        (self.root / "tracked").write_text("dirty")
        (self.root / "new").write_text("untracked")
        self.assertEqual(verification.changed_paths(self.root)[1], ["new", "tracked"])

    def test_classifier_reuses_real_policy(self):
        with patch.object(verification, "changed_paths", return_value=("base", ["docs/INDEX.md"])):
            self.assertEqual(verification.plan(ROOT)["scope"], "DOCS_ONLY_CANDIDATE")
        with patch.object(verification, "changed_paths", return_value=("base", [".github/PULL_REQUEST_TEMPLATE.md"])):
            self.assertEqual(verification.plan(ROOT)["scope"], "FULL_OR_UNKNOWN")

    def test_bound_and_stale_local_evidence(self):
        with patch.dict(verification.CHECKS, {"fixture": ("python", "-c", "print('fixture')")}):
            result, code = verification.execute(self.root, "fixture", self.directory, 10)
            self.assertEqual(code, 0)
            self.assertEqual(result["binding"], "CLEAN_HEAD_OBSERVED")
            self.assertEqual(verification.inspect(self.root, self.directory)[1], 0)
            with self.assertRaises(HarnessError):
                verification.execute(self.root, "fixture", self.directory, 10)
            (self.root / "dirty").write_text("edit")
            self.assertEqual(verification.inspect(self.root, self.directory)[1], 1)

    def test_staged_change_not_hidden_by_worktree_revert(self):
        (self.root / "tracked").write_text("initial")
        self.git("add", "tracked")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "commit", "-qm", "tracked")
        self.git("update-ref", "refs/remotes/origin/v1", "HEAD")
        (self.root / "tracked").write_text("staged")
        self.git("add", "tracked")
        (self.root / "tracked").write_text("initial")
        self.assertEqual(verification.changed_paths(self.root)[1], ["tracked"])

    def test_changed_commit_stale(self):
        with patch.dict(verification.CHECKS, {"fixture": ("python", "-c", "pass")}):
            verification.execute(self.root, "fixture", self.directory, 10)
            self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                     "commit", "--allow-empty", "-qm", "new candidate")
            self.assertEqual(verification.inspect(self.root, self.directory)[1], 1)

    def test_optional_log_is_private_and_matches_digest(self):
        import hashlib
        with patch.dict(verification.CHECKS, {"fixture": ("python", "-c", "print('fixture')")}):
            result, code = verification.execute(self.root, "fixture", self.directory, 10, retain_log=True)
            self.assertEqual(code, 0)
            log = self.directory / "output.log"
            self.assertEqual(log.stat().st_mode & 0o777, 0o600)
            self.assertEqual(hashlib.sha256(log.read_bytes()).hexdigest(), result["output_sha256"])
            self.assertEqual(verification.inspect(self.root, self.directory)[1], 0)
            log.write_text("tampered")
            self.assertEqual(verification.inspect(self.root, self.directory)[1], 1)

    def test_timeout_is_unknown(self):
        with patch.dict(verification.CHECKS, {"fixture": ("python", "-c", "import time; time.sleep(5)")}):
            result, code = verification.execute(self.root, "fixture", self.directory, 1)
            self.assertEqual((result["result"], result["exit_code"], code), ("UNKNOWN", None, 1))

    def test_failure_and_dirty_never_bound(self):
        (self.root / "dirty").write_text("edit")
        with patch.dict(verification.CHECKS, {"fixture": ("python", "-c", "raise SystemExit(7)")}):
            result, code = verification.execute(self.root, "fixture", self.directory, 10)
            self.assertEqual((result["result"], result["exit_code"], code), ("FAIL", 7, 1))
            self.assertEqual(result["binding"], "UNBOUND")
            self.assertEqual(verification.inspect(self.root, self.directory)[1], 1)


if __name__ == "__main__":
    unittest.main()
