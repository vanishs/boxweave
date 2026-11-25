"""Learning lifecycle, deduplication, stale asset and authority guard tests."""
import argparse
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
from harness import learning, records
from harness.common import HarnessError


class LearningTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "repo"
        self.root.mkdir()
        self.inbox = Path(self.tmp.name) / "inbox"
        self.inbox.mkdir()
        self.asset = self.root / "guide.md"
        self.asset.write_text("original\n")
        self.git("init", "-q")
        self.git("add", "guide.md")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "commit", "-qm", "fixture")
        self.parser = argparse.ArgumentParser()
        learning.register(self.parser)

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.root), *args], text=True).strip()

    def invoke(self, action, *args):
        return learning.run(self.parser.parse_args([action, str(self.inbox), *args]), self.root)

    def capture(self, key="L001", observation="Use actual evidence", change_class="L2"):
        return self.invoke("capture", key, "--work-ref", "fixture", "--observation", observation,
                           "--evidence", "fixture:test", "--asset", "guide.md",
                           "--validation", "inspect", "--change-class", change_class)

    def review(self, key, revision, state, *extra):
        return self.invoke("review", key, "--expect", str(revision), "--to", state,
                           "--review-ref", "fixture:review", *extra)

    def test_deduplicate_and_conflict(self):
        self.capture()
        with self.assertRaises(HarnessError):
            self.capture("L002", "  USE actual   evidence ")
        with self.assertRaises(HarnessError):
            self.capture("L001", "different")
        self.review("L001", 1, "DEFERRED")
        with self.assertRaises(HarnessError):
            self.review("L001", 1, "REJECTED")

    def test_review_and_asset_staleness(self):
        self.capture()
        with self.assertRaises(HarnessError):
            self.review("L001", 1, "ACTIVE")
        self.review("L001", 1, "VERIFIED", "--validation-evidence", "fixture:tests")
        self.review("L001", 2, "ACTIVE", "--validation-evidence", "fixture:tests")
        self.assertEqual(self.invoke("audit")[1], 0)
        self.asset.write_text("changed\n")
        result, code = self.invoke("audit")
        self.assertEqual(code, 1)
        self.assertIn("ASSET_CHANGED_OR_MISSING", [row["issue"] for row in result["findings"]])

    def test_activation_refuses_changed_source(self):
        self.capture()
        self.review("L001", 1, "VERIFIED", "--validation-evidence", "fixture:tests")
        self.asset.write_text("new\n")
        with self.assertRaises(HarnessError):
            self.review("L001", 2, "ACTIVE", "--validation-evidence", "fixture:tests")

    def test_sensitive_classes_require_review_references(self):
        self.capture(change_class="L5")
        with self.assertRaises(HarnessError):
            self.review("L001", 1, "VERIFIED", "--validation-evidence", "fixture:tests")
        self.review("L001", 1, "VERIFIED", "--validation-evidence", "fixture:tests",
                    "--owner-ref", "fixture:explicit-owner-decision")
        self.capture("L002", "another observation", "L4")
        with self.assertRaises(HarnessError):
            self.review("L002", 1, "VERIFIED", "--validation-evidence", "fixture:tests")

    def test_audit_detects_duplicate_after_separate_history_merge(self):
        result, _ = self.capture()
        data = dict(result["record"], id="L002")
        records.publish(self.inbox / "L002", 0, data)
        result, code = self.invoke("audit")
        self.assertEqual(code, 1)
        self.assertIn("DUPLICATE", [row["issue"] for row in result["findings"]])

    def test_repository_candidate_history(self):
        result, code = learning.audit(ROOT / "dev/learning", ROOT)
        self.assertIn(code, (0, 1))  # Genuine future drift is a review finding, not malformed data.
        self.assertIn("BDH-L012", [row["id"] for row in result["records"]])

    def test_attach_preserves_candidate_and_revision(self):
        self.capture()
        result, code = self.invoke("attach", "L001", "--expect", "1",
                                   "--implementation-ref", "fixture:patch",
                                   "--validation-evidence", "fixture:test")
        self.assertEqual(code, 0)
        self.assertEqual(result["record"]["state"], "CANDIDATE")
        self.assertEqual(result["record"]["revision"], 2)
        self.assertEqual(result["record"]["implementation"]["evidence_ref"], "fixture:test")
        self.assertEqual(self.invoke("next")[0]["records"][0]["id"], "L001")
        with self.assertRaises(HarnessError):
            self.invoke("attach", "L001", "--expect", "1", "--implementation-ref",
                        "stale", "--validation-evidence", "stale")

    def test_mutation_refuses_rewritten_capture(self):
        self.capture()
        directory = self.inbox / "L001"
        old = records.latest(directory)
        records.publish(directory, 1, dict(old, evidence="different original evidence"))
        with self.assertRaises(HarnessError):
            self.invoke("attach", "L001", "--expect", "2",
                        "--implementation-ref", "fixture:patch", "--validation-evidence", "fixture:tests")
        self.assertEqual(records.latest(directory)["revision"], 2)

    def test_invalid_id_and_external_asset(self):
        with self.assertRaises(HarnessError):
            self.capture("../outside")
        with self.assertRaises(HarnessError):
            learning.asset_digest(self.root, "../outside")


if __name__ == "__main__":
    unittest.main()
