"""Safety controls for external installed Verifier proof and partial status."""
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from tools.reacceptance import installed_verifier as subject
from tools.reacceptance.engine import AcceptanceFault


class IndependentInstalledVerifierTests(unittest.TestCase):
    def test_missing_lifecycle_and_selftest_never_pass(self):
        self.assertEqual(subject.check_lifecycle(None)["status"], "BLOCKED")
        self.assertEqual(subject.check_selftest(None)["status"], "BLOCKED")

    def test_partial_four_step_oracles_not_full_scenario_pass(self):
        with tempfile.TemporaryDirectory() as d:
            base = Path(d)
            (base/"result.json").write_text(json.dumps({
                "state": "PARTIAL", "scenario_id": "SYS-LIFE-01"}))
            original = {"uuid": "a"*36, "created_at": "2026-11-06"}
            for i, (name, status) in enumerate((
                    ("create_container", "Running"),
                    ("stop_container", "Stopped"),
                    ("start_container", "Running"),
                    ("delete_container", None)), 1):
                stem = f"{i:02d}-{name}"
                (base/(stem+".attempt.json")).write_text(json.dumps({
                    "tool": name, "success_response": True,
                    "process": {"execution": "completed", "exit_code": 0}}))
                (base/(stem+".oracle.json")).write_text(json.dumps({
                    "native_child": {"status": status} if status else None,
                    "incarnation": original if status else None}))
            result = subject.check_lifecycle(base)
            self.assertEqual(result["status"], "PARTIAL")
            self.assertEqual(len(result["steps"]), 4)
            (base/"03-start_container.oracle.json").write_text(json.dumps({
                "native_child": {"status": "Running"},
                "incarnation": {"uuid": "different"}}))
            self.assertEqual(subject.check_lifecycle(base)["status"], "BLOCKED")

    def test_selftest_fixture_requires_original_competing_later_oracle(self):
        with tempfile.TemporaryDirectory() as d:
            base = Path(d)
            steps = [{"step": t} for t in ("create", "hello", "stop", "delete")]
            p = {"state": "passed", "steps": steps}
            (base/"outcome.json").write_text(json.dumps({
                "state": "INSTALLED_BLACKBOX_PASS",
                "proof_kind": "same_agent_installed_two_process",
                "residue": 0, "competing_exit": 2, "later_exit": 0,
                "original_box": "test-a", "competing_box": "test-b",
                "later_box": "test-c"}))
            (base/"original-complete.json").write_text(json.dumps({
                "exit_code": 0, "stdout": json.dumps(p)}))
            (base/"after-contention.json").write_text(json.dumps({
                "exit_code": 0, "stdout": json.dumps(p)}))
            (base/"competing.json").write_text(json.dumps({
                "exit_code": 2, "stdout": json.dumps({
                    "code": "SELFTEST_RUN_UNRECONCILED"})}))
            self.assertEqual(subject.check_selftest(base)["status"], "PARTIAL")
            (base/"competing.json").write_text(json.dumps({
                "exit_code": 0, "stdout": json.dumps({"state": "passed"})}))
            self.assertEqual(subject.check_selftest(base)["status"], "BLOCKED")

    def test_verified_installed_package_cannot_sign_incomplete_release(self):
        with tempfile.TemporaryDirectory() as d:
            base = Path(d)
            (base/"boxweave.zip").write_bytes(b"candidate")
            digest = hashlib.sha256(b"candidate").hexdigest()
            source = "b"*40
            (base/"release.json").write_text(json.dumps({
                "source": source, "files": {"boxweave.zip": digest}}))
            data = {"version": "0.49.0",
                    "source": {"commit": source, "state": "clean"}}
            responses = {
                "version": {"exit_code": 0, "execution": "completed",
                            "stdout": "BoxWeave 0.49.0\n",
                            "stdout_sha256": "a"*64},
                "provenance": {"exit_code": 0, "execution": "completed",
                               "stdout": json.dumps({
                                   "version": "0.49.0", "source": source}),
                               "stdout_sha256": "b"*64},
                "doctor": {"exit_code": 0, "execution": "completed",
                            "stdout": json.dumps({
                                "checks": [{"id": x, "state": "passed"}
                                           for x in subject.REQUIRED_HOST_CHECKS]}),
                            "stdout_sha256": "c"*64},
                "installed": {"exit_code": 0, "execution": "completed",
                              "stdout": json.dumps({
                                  **data, "archive_sha256": digest}),
                              "stdout_sha256": "d"*64},
            }
            def fake_capture(argv, **kw):
                if "version" in argv: return responses["version"]
                if "provenance" in argv: return responses["provenance"]
                if "doctor" in argv: return responses["doctor"]
                return responses["installed"]
            with patch.object(subject, "verify_package", return_value=data), (
                 patch.object(subject, "capture", side_effect=fake_capture)):
                verdict = subject.inspect(base/"boxweave.zip",
                        base/"release.json", source, base/"verifier")
            self.assertEqual(verdict["host_app_parity"], "PASS")
            self.assertFalse(verdict["release_gate_approved"])
            self.assertFalse(any(x["status"] == "PASS"
                                 for x in verdict["case_matrix"].values()))
            with patch.object(subject, "verify_package", return_value=data), (
                 patch.object(subject, "capture", side_effect=fake_capture)):
                with self.assertRaisesRegex(AcceptanceFault, "VERIFIER_OUTPUT_EXISTS"):
                    subject.inspect(base/"boxweave.zip", base/"release.json",
                                    source, base/"verifier")


if __name__ == "__main__":
    unittest.main()

