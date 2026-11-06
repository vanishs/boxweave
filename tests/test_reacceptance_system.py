"""Adversarial tests for the external, fail-closed reacceptance tooling."""
from __future__ import annotations

import base64
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
import zipfile

from bwmini import distribution
from tools.reacceptance import upstream, scenarios, surfaces, execution, agents, release_gate
from tools.reacceptance.engine import AcceptanceFault


ROOT = Path(__file__).resolve().parents[1]


class ExternalRunnerTests(unittest.TestCase):
    def test_legacy_shim_import_is_the_same_external_engine(self):
        from bwmini import reacceptance
        from tools.reacceptance import engine
        self.assertIs(reacceptance, engine)

    def test_no_prod_app_runtime_depends_on_dev_shim(self):
        import ast
        for file in distribution.production_modules(ROOT):
            root = ast.parse(file.read_text())
            imports = [x.module for x in ast.walk(root)
                       if isinstance(x, ast.ImportFrom)]
            imports += [a.name for x in ast.walk(root) if isinstance(x, ast.Import)
                        for a in x.names]
            for source in imports:
                self.assertNotIn("reacceptance", source or "", file.as_posix())

    def test_surface_help_discovery_never_mutates(self):
        with tempfile.TemporaryDirectory() as t:
            tool = Path(t) / "fake.py"
            invoked = Path(t) / "writes"
            tool.write_text('''import sys
from pathlib import Path
argv=sys.argv[1:]
if argv[-1:] == ["--help"]:
    print("usage: test {selftest,status,box}")
else:
    Path(%r).write_text("FORBIDDEN")
''' % str(invoked))
            rows = surfaces.help_tree([sys.executable, str(tool)])
            self.assertIn("selftest", rows)
            self.assertFalse(invoked.exists())
            self.assertTrue(rows["selftest"]["help_is_not_a_functional_positive"])

    def test_catalogue_falls_back_to_blocked_not_fake_pass(self):
        r = surfaces.breadth(ROOT)
        self.assertTrue(r["rows"])
        self.assertTrue(all(x["status"] in ("BLOCKED", "PARTIAL")
                            for x in r["rows"].values()))
        self.assertEqual(r["status"], "PARTIAL_NOT_FULL_ACCEPTANCE")

    def test_scenario_graphs_rank_changed_but_do_not_pass(self):
        from tools.reacceptance.engine import catalogue
        inv = catalogue(ROOT)
        outcome = scenarios.plan(inv,
            ROOT / "docs/verification/REACCEPTANCE_SYSTEM_SCENARIOS.md",
            ["bwmini/gateway_tasks.py"])
        self.assertGreaterEqual(outcome["scenario_count"], 16)
        found = {x["id"]: x for x in outcome["scenarios"]}
        self.assertGreater(found["SYS-JOB-01"]["risk"], found["SYS-COMPOSE-01"]["risk"])
        self.assertIn(found["SYS-JOB-01"]["status"],
                      ("BLOCKED_RUNTIME_ACTION_MISSING", "PLANNED_NOT_RUN"))
        self.assertFalse(any(x["status"] == "PASS" for x in outcome["scenarios"]))

    def test_once_only_dedup_and_unknown_target_freeze(self):
        with tempfile.TemporaryDirectory() as path:
            run = execution.OnceOnly(path)
            a = {"owner": "owned-racc", "provider_uuid": "a" * 8 + "-" + "b" * 4 +
                 "-" + "c" * 4 + "-" + "d" * 4 + "-" + "e" * 12,
                 "created_at": "1793900000", "test_owned": True}
            key = "a" * 32
            first = run.reserve(key, {"op": "stop"}, a)
            self.assertEqual(first["state"], "RESERVED_ONLY_NEEDS_APPROVED_EXECUTOR")
            with self.assertRaisesRegex(AcceptanceFault, "FROZEN"):
                run.reserve(key, {"op": "stop"}, a)
            with self.assertRaisesRegex(AcceptanceFault, "FROZEN"):
                run.reserve("b" * 32, {"op": "resize"}, a)
            self.assertIn("UNKNOWN", run.observe(a["provider_uuid"])["state"])
            b = dict(a, provider_uuid="f"*8 + "-" + "e"*4 + "-" + "d"*4 +
                     "-" + "c"*4 + "-" + "b"*12)
            self.assertEqual(run.reserve("c" * 32, {"op": "read"}, b)["state"],
                             "RESERVED_ONLY_NEEDS_APPROVED_EXECUTOR")

    def test_foreign_target_never_reserved(self):
        with tempfile.TemporaryDirectory() as path:
            run = execution.OnceOnly(path)
            with self.assertRaisesRegex(AcceptanceFault, "FOREIGN_OR_UNPINNED"):
                run.reserve("a" * 32, {"op": "stop"}, {"owner": "owner", "test_owned": False})
            self.assertEqual(list(Path(path).glob("dispatch-*")), [])

    def test_agent_handoff_does_not_claim_spawn(self):
        result = agents.handoffs({"original_attempt_sha256": "a"*64},
                     {"actor_id": "finder1", "box_uuid": "a"*36},
                     {"actor_id": "fixer2", "branch": "repair/test"}, "b"*64)
        self.assertFalse(result["independent_acceptance"])
        self.assertIn("MANUAL", result["status"])
        with self.assertRaisesRegex(AcceptanceFault, "NOT_DISTINCT"):
            agents.handoffs({"original_attempt_sha256": "a"*64},
                     {"actor_id": "one", "box_uuid": "a"*36},
                     {"actor_id": "one", "branch": "repair/test"}, "b"*64)


class UpstreamPinTests(unittest.TestCase):
    def fake_fetch(self, url):
        if url.endswith("/git/ref/tags/v0.131.0"):
            return {"object": {"sha": "a"*40, "type": "commit"}}
        if url.endswith("/git/ref/tags/v0.130.0"):
            return {"object": {"sha": "b"*40, "type": "commit"}}
        if url.startswith(upstream.API + "contents/"):
            data = b"reference content (never executable)"
            return {"type": "file", "encoding": "base64",
                    "content": base64.b64encode(data).decode(),
                    "sha": hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()}
        raise AssertionError("Unexpected or floating upstream request: " + url)

    def observations(self):
        return {"schema_version": 1, "components": {
            "daemon": {"status": "OBSERVED", "version": "0.131.0", "binary_path": "/test/daemon"},
            "cli": {"status": "OBSERVED", "version": "0.130.0", "binary_path": "/test/cli"},
            "mcp": {"status": "UNAVAILABLE", "reason": "VERSION_NOT_EXPOSED"}}}

    def test_real_component_versions_not_boxweave_baseline(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d) / "cache"
            result = upstream.resolve(self.observations(), root,
                            ("docs/DOCMAP.md",), fetch=self.fake_fetch)
            self.assertTrue(result["split_runtime"])
            self.assertEqual(result["components"]["daemon"]["commit"], "a"*40)
            self.assertEqual(result["components"]["cli"]["commit"], "b"*40)
            self.assertEqual(result["components"]["mcp"]["status"], "UNAVAILABLE")
            offline = upstream.resolve(self.observations(), root,
                              ("docs/DOCMAP.md",), offline=True,
                              fetch=lambda _: self.fail("offline made network request"))
            self.assertEqual(offline["components"]["daemon"]["commit"], "a"*40)
            pin = root / "v0.131.0.json"
            pin.write_text('{"tag":"v0.131.0","commit":"' + "f"*40 + '"}')
            with self.assertRaisesRegex(AcceptanceFault, "CACHE_PIN_TAMPERED"):
                upstream.resolve(self.observations(), root, ("docs/DOCMAP.md",),
                                 offline=True, fetch=self.fake_fetch)
            pin.write_text(json.dumps({"tag": "v0.131.0", "commit": "a"*40},
                                      sort_keys=True, separators=(",", ":")) + "\n")
            blob = root / ("a"*40) / "docs/DOCMAP.md"
            blob.write_text("tampered")
            with self.assertRaisesRegex(AcceptanceFault, "CACHE_TAMPERED"):
                upstream.resolve(self.observations(), root, ("docs/DOCMAP.md",),
                                 offline=True, fetch=self.fake_fetch)

    def test_unresolved_offline_ref_is_not_latest(self):
        with tempfile.TemporaryDirectory() as d:
            out = upstream.resolve(self.observations(), Path(d) / "cache",
                       ("docs/DOCMAP.md",), offline=True,
                       fetch=lambda _: self.fail("offline network"))
            self.assertEqual(out["components"]["daemon"]["status"],
                             "UNRESOLVED_UPSTREAM_REF")
            self.assertEqual(out["state"], "NO_OBSERVED_VERSION_PIN")

    def test_invalid_tag_and_untrusted_path_rejected(self):
        with self.assertRaisesRegex(AcceptanceFault, "VERSION_INVALID"):
            upstream.semver("latest")
        for item in ("../../install.sh", "docs/../../secret", "README.md",
                     "https://evil.example/", "docs/testing/../x"):
            with self.subTest(item=item), self.assertRaises(AcceptanceFault):
                upstream.safe_path(item)
        with self.assertRaisesRegex(AcceptanceFault, "TAG_TARGET"):
            upstream.release_commit("0.131.0",
                lambda _: {"object": {"sha": "a"*40, "type": "tree"}})


@unittest.skipUnless(shutil.which("openssl"), "openssl unavailable")
class ReleaseGateTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.source = "a"*40
        self.package = self.root / "boxweave.zip"
        pieces = {"bwmini/__main__.py": b"fixture",
                  "bwmini/deploy_check.py": b"fixture", "requirements.lock": b"",
                  "VERSION": b"0.48.0\n", "web/dist/index.html": b"<html/>"}
        manifest = {"format": 1, "version": "0.48.0",
                    "platform": distribution.PLATFORM,
                    "source": {"commit": self.source, "state": "clean"},
                    "files": {name: hashlib.sha256(data).hexdigest()
                              for name, data in pieces.items()}}
        with zipfile.ZipFile(self.package, "w") as z:
            for name, data in pieces.items():
                z.writestr(name, data)
            z.writestr("manifest.json", json.dumps(manifest))
        self.package_sha = hashlib.sha256(self.package.read_bytes()).hexdigest()
        self.release = self.root / "release.json"
        self.release.write_text(json.dumps(
            {"source": self.source, "files": {"boxweave.zip": self.package_sha}}))
        self.private = self.root / "private.pem"
        self.public = self.root / "public.pem"
        subprocess.run(["openssl", "genpkey", "-algorithm", "RSA",
                "-pkeyopt", "rsa_keygen_bits:2048", "-out", str(self.private)],
                check=True, capture_output=True)
        subprocess.run(["openssl", "pkey", "-in", str(self.private),
                "-pubout", "-out", str(self.public)], check=True, capture_output=True)

    def signed(self, change=None):
        verifier_id = "verifier-2"
        row = {"status": "PASS", "original_attempt_sha256": "a"*64,
               "oracle_sha256": "b"*64, "installed_report_sha256": "c"*64,
               "verifier_actor_id": verifier_id}
        receipt = {"schema_version": 1, "decision": "APPROVED",
                   "source_sha": self.source, "candidate_sha256": self.package_sha,
                   "finder": {"actor_id": "finder-1", "box_uuid": "1"*36,
                              "created_at": "1793900000"},
                   "verifier": {"actor_id": verifier_id, "box_uuid": "2"*36,
                                "created_at": "1793900100",
                                "clean_installed_candidate": True,
                                "installed_candidate_sha256": self.package_sha},
                   "scenarios": {sid: dict(row) for sid in release_gate.MANDATORY}}
        if change:
            change(receipt)
        for sid, values in receipt["scenarios"].items():
            folder = self.root / "cases" / sid
            folder.mkdir(parents=True, exist_ok=True)
            for key, filename, actor in (
                    ("original_attempt_sha256", "original-attempt.json", "finder-1"),
                    ("oracle_sha256", "oracle.json", verifier_id),
                    ("installed_report_sha256", "installed-report.json", verifier_id)):
                data = {"schema_version": 1, "scenario_id": sid, "actor_id": actor,
                        "candidate_sha256": self.package_sha if actor == verifier_id else "original"}
                contents = json.dumps(data, sort_keys=True).encode()
                (folder / filename).write_bytes(contents)
                values[key] = hashlib.sha256(contents).hexdigest()
        report = self.root / "release-gate.json"
        sig = self.root / "release-gate.sig"
        report.write_text(json.dumps(receipt, sort_keys=True))
        subprocess.run(["openssl", "dgst", "-sha256", "-sign", str(self.private),
                "-out", str(sig), str(report)], check=True, capture_output=True)
        return report, sig

    def check(self, *args):
        receipt, signature = args or self.signed()
        return release_gate.accept(self.package, self.release, receipt, signature,
                                   self.public, self.source)

    def test_signed_independent_full_scenario_receipt_accepts_exact_bytes(self):
        approved = self.check()
        self.assertEqual(approved["decision"], "APPROVED_FOR_THIS_EXACT_CANDIDATE_ONLY")
        self.assertTrue(approved["public_release_not_yet_verified"])

    def test_changed_candidate_denied(self):
        report, sig = self.signed()
        self.package.write_bytes(b"tampered")
        with self.assertRaises(AcceptanceFault):
            self.check(report, sig)

    def test_forged_success_json_denied_without_valid_signature(self):
        report, sig = self.signed()
        receipt = json.loads(report.read_text())
        receipt["verifier"]["box_uuid"] = "different"
        report.write_text(json.dumps(receipt))
        with self.assertRaisesRegex(AcceptanceFault, "SIGNATURE_NOT_TRUSTED"):
            self.check(report, sig)

    def test_signed_same_actor_and_missing_scenario_rejected(self):
        a, b = self.signed(lambda d: d["verifier"].update(box_uuid="1"*36))
        with self.assertRaisesRegex(AcceptanceFault, "SAME_ACTOR"):
            self.check(a, b)
        a, b = self.signed(lambda d: d["scenarios"].pop("SYS-RELEASE-01"))
        with self.assertRaisesRegex(AcceptanceFault, "REQUIRED_SCENARIO_NOT_VERIFIED"):
            self.check(a, b)

    def test_signed_missing_oracle_and_invalid_install_rejected(self):
        a, b = self.signed()
        (self.root / "cases/SYS-LIFE-01/oracle.json").write_text('{"tampered":true}')
        with self.assertRaisesRegex(AcceptanceFault, "BYTES_MISMATCH"):
            self.check(a, b)
        a, b = self.signed(lambda d: d["verifier"].update(clean_installed_candidate=False))
        with self.assertRaisesRegex(AcceptanceFault, "CLEAN_INSTALL_UNPROVEN"):
            self.check(a, b)

    def test_wrong_source_and_release_manifest_refused(self):
        a, b = self.signed()
        with self.assertRaisesRegex(AcceptanceFault, "CANDIDATE_SOURCE_MISMATCH"):
            release_gate.accept(self.package, self.release, a, b,
                                self.public, "f"*40)
        self.release.write_text(json.dumps({"source": self.source,
                     "files": {"boxweave.zip": "f"*64}}))
        with self.assertRaisesRegex(AcceptanceFault, "RELEASE_MANIFEST_DIGEST_MISMATCH"):
            self.check(a, b)


if __name__ == "__main__":
    unittest.main()
