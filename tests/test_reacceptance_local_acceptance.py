"""Fixture calibration for the no-workflow, Agent-local evidence checker."""
from __future__ import annotations

import contextlib
import io
import json
import unittest

from tools.reacceptance import local_acceptance
from tools.reacceptance.engine import AcceptanceFault
from tests.test_reacceptance_signing_preflight import SigningPreflightTests


class LocalAcceptanceTests(unittest.TestCase):
    def setUp(self):
        self.fixture = SigningPreflightTests("test_complete_fixture_is_only_eligible_never_signature_or_authority")
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)

    def review(self):
        f = self.fixture
        return local_acceptance.check(f.assets, f.attestation, f.source)

    def test_consistent_fixture_is_not_authenticated_or_release_authorized(self):
        r = self.review()
        self.assertEqual(r["decision"], "LOCAL_EVIDENCE_CONSISTENT_NOT_RELEASE_AUTHORIZATION")
        self.assertFalse(r["remote_execution_authenticated"])
        self.assertFalse(r["independent_box_truth_verified_by_this_command"])
        self.assertFalse(r["release_authorized"])
        self.assertFalse(r["published"])

    def test_partial_full_graph_refused(self):
        f = self.fixture
        f.evidence["scenarios"]["SYS-LIFE-01"]["status"] = "PARTIAL"
        f.save()
        with self.assertRaisesRegex(AcceptanceFault, "REQUIRED_SCENARIO_NOT_VERIFIED"):
            self.review()

    def test_same_actor_or_box_refused(self):
        f = self.fixture
        f.evidence["verifier"]["actor_id"] = f.evidence["finder"]["actor_id"]
        f.save()
        with self.assertRaisesRegex(AcceptanceFault, "SAME_ACTOR_OR_SAME_BOX"):
            self.review()

    def test_case_oracle_tampering_refused(self):
        f = self.fixture
        (f.root / "cases/SYS-LIFE-01/oracle.json").write_text('{"independent_native":false}')
        with self.assertRaisesRegex(AcceptanceFault, "BYTES_MISMATCH"):
            self.review()

    def test_original_candidate_asset_tampering_refused(self):
        f = self.fixture
        (f.assets / "install.sh").write_bytes(b"not the accepted bytes")
        with self.assertRaisesRegex(AcceptanceFault, "CHECKSUM_FILE_MISMATCH"):
            self.review()

    def test_command_does_not_claim_publication(self):
        f = self.fixture
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            rc = local_acceptance.main(["--assets", str(f.assets),
                "--evidence", str(f.attestation), "--source", "f" * 40])
        self.assertEqual(rc, 2)
        self.assertFalse(json.loads(buf.getvalue())["published"])


if __name__ == "__main__":
    unittest.main()
