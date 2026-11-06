"""Adversarial checks for workflow origin and complete publication bytes."""
import copy
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from tools.reacceptance import workflow_gate as gate
from tools.reacceptance.engine import AcceptanceFault


class WorkflowGateTests(unittest.TestCase):
    def setUp(self):
        self.source = "a" * 40
        self.candidate = {"id": 10, "repository": {"full_name": gate.REPOSITORY},
            "head_repository": {"full_name": gate.REPOSITORY}, "head_sha": self.source,
            "head_branch": "main", "path": gate.CANDIDATE_WORKFLOW, "event": "push",
            "status": "completed", "conclusion": "success", "run_attempt": 1}
        self.verifier = {**copy.deepcopy(self.candidate), "id": 11,
            "path": gate.VERIFIER_WORKFLOW, "event": "workflow_dispatch"}
        self.artifacts = {"total_count": 1, "artifacts": [{"id": 20,
            "name": "application-candidate", "expired": False,
            "digest": "sha256:" + "b" * 64,
            "workflow_run": {"id": 10, "head_sha": self.source}}]}

    def check(self):
        return gate.provenance(self.candidate, self.verifier, self.artifacts, self.source)

    def test_exact_distinct_successful_workflow_context(self):
        self.assertEqual(self.check()["candidate_artifact_id"], 20)

    def test_untrusted_verifier_origin_and_state_rejected(self):
        for key, bad in (("path", gate.CANDIDATE_WORKFLOW), ("event", "pull_request"),
                         ("head_sha", "b" * 40), ("head_branch", "other"),
                         ("conclusion", "failure"), ("status", "in_progress"),
                         ("id", 10), ("id", -1), ("run_attempt", True), ("run_attempt", -1)):
            with self.subTest(key=key):
                old = self.verifier[key]
                self.verifier[key] = bad
                with self.assertRaises(AcceptanceFault):
                    self.check()
                self.verifier[key] = old
        self.verifier["head_repository"]["full_name"] = "attacker/boxweave"
        with self.assertRaises(AcceptanceFault):
            self.check()

    def test_ambiguous_expired_and_truncated_artifacts_rejected(self):
        self.artifacts["total_count"] = 101
        with self.assertRaisesRegex(AcceptanceFault, "INCOMPLETE"):
            self.check()
        self.artifacts["total_count"] = 1
        self.artifacts["artifacts"][0]["expired"] = True
        with self.assertRaisesRegex(AcceptanceFault, "UNAVAILABLE"):
            self.check()
        self.artifacts["artifacts"][0]["expired"] = False
        self.artifacts["artifacts"].append(copy.deepcopy(self.artifacts["artifacts"][0]))
        self.artifacts["total_count"] = 2
        with self.assertRaisesRegex(AcceptanceFault, "AMBIGUOUS"):
            self.check()

    def assets(self, root):
        for name in gate.ASSETS - {"release.json", "SHA256SUMS"}:
            (root / name).write_bytes(name.encode())
        digest = lambda n: hashlib.sha256((root / n).read_bytes()).hexdigest()
        (root / "release.json").write_text(json.dumps({"files": {
            n: digest(n) for n in gate.RELEASE_FILES}}))
        (root / "SHA256SUMS").write_text("".join(digest(n) + "  " + n + "\n"
            for n in sorted(gate.ASSETS - {"SHA256SUMS"})))

    def test_every_published_byte_and_exact_asset_set_verified(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            self.assets(root)
            index = gate.asset_index(root)
            self.assertEqual(set(index), gate.ASSETS)
            (root / "online.py").write_text("modified")
            with self.assertRaisesRegex(AcceptanceFault, "DIGEST_MISMATCH"):
                gate.asset_index(root)
            self.assets(root)
            (root / "install.sh").write_text("modified")
            with self.assertRaisesRegex(AcceptanceFault, "CHECKSUM_FILE_MISMATCH"):
                gate.asset_index(root)
            self.assets(root)
            (root / "surprise.sh").write_text("extra")
            with self.assertRaisesRegex(AcceptanceFault, "ASSET_SET_MISMATCH"):
                gate.asset_index(root)
