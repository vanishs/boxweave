"""Known-bad calibration of pre-sign eligibility; no signing identity is used."""
from __future__ import annotations

import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest
import zipfile

from bwmini.distribution import PLATFORM
from tools.reacceptance import release_gate, signing_preflight, workflow_gate
from tools.reacceptance.engine import AcceptanceFault


class SigningPreflightTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.assets = self.root / "assets"
        self.assets.mkdir()
        self.source = "a" * 40
        self.context = {"candidate_run_id": 101, "candidate_run_attempt": 1,
                        "candidate_artifact_id": 202,
                        "candidate_artifact_digest": "sha256:" + "c" * 64,
                        "verifier_run_id": 303, "verifier_run_attempt": 1}
        pieces = {"VERSION": b"0.49.1\n", "requirements.lock": b"",
                  "web/dist/index.html": b"<html/>",
                  "bwmini/__main__.py": b"fixture", "bwmini/deploy_check.py": b"fixture"}
        for name in release_gate.RELEASE_FILES:
            if name != "boxweave.zip":
                data = b"fixture helper " + name.encode()
                pieces["bwmini/" + name] = data
                (self.assets / name).write_bytes(data)
        manifest = {"format": 1, "version": "0.49.1", "platform": PLATFORM,
                    "source": {"commit": self.source, "state": "clean"},
                    "files": {name: release_gate.sha(data) for name, data in pieces.items()}}
        with zipfile.ZipFile(self.assets / "boxweave.zip", "w") as archive:
            for name, data in pieces.items():
                archive.writestr(name, data)
            archive.writestr("manifest.json", json.dumps(manifest))
        digest = release_gate.sha((self.assets / "boxweave.zip").read_bytes())
        release = {"format": 1, "version": "0.49.1", "source": self.source,
                   "files": {name: release_gate.sha((self.assets / name).read_bytes())
                             for name in release_gate.RELEASE_FILES}}
        (self.assets / "release.json").write_text(json.dumps(release))
        (self.assets / "install.sh").write_bytes(b"fixture installer")
        names = workflow_gate.ASSETS - {"SHA256SUMS"}
        (self.assets / "SHA256SUMS").write_text("".join(
            release_gate.sha((self.assets / name).read_bytes()) + "  " + name + "\n"
            for name in sorted(names)))
        self.evidence = {"schema_version": 1, "decision": "APPROVED",
            "source_sha": self.source, "candidate_sha256": digest,
            "release_manifest_sha256": release_gate.sha((self.assets / "release.json").read_bytes()),
            "release_assets_sha256": workflow_gate.asset_index(self.assets),
            "workflow": self.context,
            "finder": {"actor_id": "finder-1", "box_uuid": "11111111-1111-4111-8111-111111111111", "created_at": "1793900000"},
            "verifier": {"actor_id": "verifier-2", "box_uuid": "22222222-2222-4222-8222-222222222222", "created_at": "1793900010", "clean_installed_candidate": True, "installed_candidate_sha256": digest},
            "scenarios": {}}
        for sid in release_gate.FULL_SCENARIOS:
            folder = self.root / "cases" / sid
            folder.mkdir(parents=True)
            row = {"status": "PASS", "verifier_actor_id": "verifier-2"}
            for field, filename, actor in (("original_attempt_sha256", "original-attempt.json", "finder-1"),
                    ("oracle_sha256", "oracle.json", "verifier-2"),
                    ("installed_report_sha256", "installed-report.json", "verifier-2")):
                proof = {"schema_version": 1, "scenario_id": sid, "actor_id": actor,
                    "candidate_sha256": digest, "status": "PASS", "proof_kind": "installed_blackbox",
                    "process": {"execution": "completed", "exit_code": 0,
                                "stdout_sha256": "a" * 64, "stderr_sha256": "b" * 64},
                    "independent_native": True, "observed_expected_effect": True,
                    "full_graph_complete": True, "unresolved": [],
                    "nodes": {node: "PASS" for node in release_gate.REQUIRED_NODES[sid]}}
                data = json.dumps(proof).encode()
                (folder / filename).write_bytes(data)
                row[field] = release_gate.sha(data)
            self.evidence["scenarios"][sid] = row
        self.attestation = self.root / "release-gate.json"
        self.save()

    def save(self):
        self.attestation.write_text(json.dumps(self.evidence))

    def prepare(self):
        return signing_preflight.prepare(self.assets, self.attestation, self.source, self.context)

    def test_complete_fixture_is_only_eligible_never_signature_or_authority(self):
        result = self.prepare()
        self.assertEqual(result["decision"], "ELIGIBLE_FOR_PROTECTED_SIGNING_ONLY")
        self.assertIs(result["signature_produced"], False)
        self.assertIs(result["release_authorized"], False)
        self.assertFalse(any(p.suffix in (".pem", ".sig") for p in self.root.rglob("*")))

    def test_partial_report_fails_closed_before_signing(self):
        self.evidence["scenarios"]["SYS-LIFE-01"]["status"] = "PARTIAL"
        self.save()
        with self.assertRaisesRegex(AcceptanceFault, "REQUIRED_SCENARIO_NOT_VERIFIED"):
            self.prepare()

    def test_tampered_native_oracle_rejected(self):
        (self.root / "cases/SYS-LIFE-01/oracle.json").write_text('{"status":"PASS"}')
        with self.assertRaisesRegex(AcceptanceFault, "BYTES_MISMATCH"):
            self.prepare()

    def test_changed_installer_cannot_hide_outside_manifest(self):
        (self.assets / "install.sh").write_bytes(b"changed installer")
        with self.assertRaisesRegex(AcceptanceFault, "CHECKSUM_FILE_MISMATCH"):
            self.prepare()

    def test_wrong_workflow_attempt_and_unbound_assets_rejected(self):
        self.evidence["workflow"] = dict(self.context, candidate_run_attempt=2)
        self.save()
        with self.assertRaisesRegex(AcceptanceFault, "WORKFLOW_CONTEXT_MISMATCH"):
            self.prepare()
        self.evidence["workflow"] = self.context
        self.evidence["release_assets_sha256"].pop("install.sh")
        self.save()
        with self.assertRaisesRegex(AcceptanceFault, "ALL_ASSET_BINDING_MISMATCH"):
            self.prepare()

    def test_cli_denies_partial_proof_with_exit_two(self):
        self.evidence["scenarios"]["SYS-JOB-01"]["status"] = "BLOCKED"
        self.save()
        context = self.root / "context.json"
        context.write_text(json.dumps(self.context))
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            code = signing_preflight.main(["--assets", str(self.assets), "--attestation",
                str(self.attestation), "--source", self.source, "--expected-context", str(context)])
        self.assertEqual(code, 2)
        self.assertEqual(json.loads(output.getvalue())["decision"], "DENIED")


if __name__ == "__main__":
    unittest.main()
