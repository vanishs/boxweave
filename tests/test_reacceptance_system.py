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

    def test_real_http_route_extraction_ignores_arbitrary_get_calls(self):
        from tools.reacceptance.engine import source_web_routes
        with tempfile.TemporaryDirectory() as t:
            script = Path(t) / "api.py"
            script.write_text('''def make_app():
    from flask import Flask, request
    app = Flask(__name__)
    @app.get("/api/real")
    def real():
        return request.headers.get("Authorization")
    @app.post("/api/submit")
    def submit():
        return request.form.get("code_verifier")
    @app.route("/mcp", methods=["GET", "POST"])
    def both():
        return request.args.get("not-a-route")
    return app
''')
            routes = source_web_routes(script)
        self.assertEqual(routes, ["GET /api/real", "GET /mcp",
                                  "POST /api/submit", "POST /mcp"])
        self.assertFalse(any("Authorization" in r or "code_verifier" in r
                             for r in routes))

    def test_installed_help_never_expands_option_value_sets_or_fallback(self):
        from unittest.mock import patch
        calls = []
        usages = {
            (): "usage: boxweave [-h] {setup,http,box}",
            ("setup",): "usage: boxweave setup [-h] [--mode {ssh,https,local}] [--format {text,json}]",
            ("http",): "usage: boxweave http [-h] {status,doctor,configure,restart}",
            ("box",): "usage: boxweave box [-h] {inspect,diagnose}",
            ("box", "inspect"): "usage: boxweave box inspect [-h]",
            ("box", "diagnose"): "usage: boxweave box diagnose [-h]",
        }
        def fake_capture(argv, **kw):
            key = tuple(argv[1:-1])
            calls.append(key)
            return {"execution": "completed", "exit_code": 0,
                    "stdout": usages.get(key, usages[("http",)]) + "\\n\\n",
                    "stderr": ""}
        with patch.object(surfaces, "capture", side_effect=fake_capture):
            discovered = surfaces.help_tree(["boxweave"])
        self.assertNotIn("setup ssh", discovered)
        self.assertNotIn("setup text", discovered)
        self.assertNotIn("http configure configure", discovered)
        self.assertTrue(discovered["box inspect"]["confirmed"])
        self.assertFalse(discovered["http configure"]["confirmed"])
        self.assertLess(len(calls), 20)

    def test_feature_mapping_is_specific_not_blanket_web_or_remote(self):
        from tools.reacceptance.engine import catalogue
        catalog = catalogue(ROOT)
        actions = {r["key"]: r["features"] for r in catalog["actions"]}
        self.assertNotIn("web_api:GET Authorization", actions)
        self.assertNotIn("web_api:GET code_verifier", actions)
        self.assertEqual(actions["web_api:GET /api/graphs"], ["F21"])
        self.assertEqual(actions["web_api:POST /api/login"], ["F18"])
        self.assertEqual(actions["web_api:GET /api/model-wizard"], ["F22"])
        self.assertEqual(actions["remote_api:POST /authorize"], ["F25"])
        self.assertEqual(actions["remote_api:POST /mcp"], ["F25"])
        self.assertEqual(actions["stdio_mcp:create_container"], ["F07", "F34"]) if (
            "stdio_mcp:create_container" in actions) else None
        self.assertFalse([r["key"] for r in catalog["actions"]
                          if r["surface"] in ("web_api", "remote_api")
                          and r["features"] == ["UNMAPPED"]])

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

    def test_opt_in_readonly_fixture_runs_only_after_native_create_identity(self):
        from tools.reacceptance import lifecycle, installed
        from unittest.mock import patch
        uuid = "a" * 8 + "-" + "b" * 4 + "-" + "c" * 4 + "-" + "d" * 4 + "-" + "e" * 12
        name = "bwracc-verified-12345"
        identity = {"uuid": uuid, "created_at": "2026-11-06T20:00:00Z"}
        def child(status):
            return {"name": name + "-container", "status": status,
                    "created_at": identity["created_at"],
                    "config": {"volatile.uuid": uuid}}
        calls = []
        def one_call(argv, tool, args):
            calls.append(tool)
            return {"success_response": True, "response": {
                "result": {"content": []}}, "process": {"execution": "completed"}}
        observed = [{}, {name + "-container": child("Running")},
                    {name + "-container": child("Running")},
                    {name + "-container": child("Stopped")},
                    {name + "-container": child("Running")}, {}]
        with tempfile.TemporaryDirectory() as t:
            with patch.object(lifecycle, "grant_read", return_value={
                    "child_name": name, "outer_uuid": uuid,
                    "outer_created_at": "1794000000"}), (
                 patch.object(lifecycle, "native_list", side_effect=observed)), (
                 patch.object(lifecycle, "one_call", side_effect=one_call)), (
                 patch.object(installed, "inspect_mcp", return_value={"catalogue": "DISCOVERED",
                     "rows": {"stdio_mcp:get_container": {"status": "PARTIAL"}}}) as probe):
                result = lifecycle.run("private-grant", lifecycle.CONFIRM,
                                      ["fake-mcp"], Path(t) / "output",
                                      with_readonly_breadth=True)
            self.assertEqual(calls, ["create_container", "stop_container",
                                     "start_container", "delete_container"])
            self.assertEqual(result["readonly_breadth"],
                             "PARTIAL_PRIVATE_EVIDENCE_NOT_ACCEPTANCE")
            self.assertTrue((Path(t) / "output/readonly-breadth.json").exists())
            self.assertEqual(probe.call_args.kwargs["verified_target"]["provider_uuid"], uuid)
            self.assertFalse(result.get("full_acceptance", False))

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
        pieces.update({"bwmini/" + name: b"fixture helper " + name.encode()
                       for name in release_gate.RELEASE_FILES if name != "boxweave.zip"})
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
        assets = {"boxweave.zip": self.package_sha}
        for name in release_gate.RELEASE_FILES:
            if name != "boxweave.zip":
                data = pieces["bwmini/" + name]
                (self.root / name).write_bytes(data)
                assets[name] = hashlib.sha256(data).hexdigest()
        self.release.write_text(json.dumps(
            {"format": 1, "version": "0.48.0", "source": self.source, "files": assets}))
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
                   "release_manifest_sha256": hashlib.sha256(self.release.read_bytes()).hexdigest(),
                   "finder": {"actor_id": "finder-1", "box_uuid": "11111111-1111-4111-8111-111111111111",
                              "created_at": "1793900000"},
                   "verifier": {"actor_id": verifier_id, "box_uuid": "22222222-2222-4222-8222-222222222222",
                                "created_at": "1793900100",
                                "clean_installed_candidate": True,
                                "installed_candidate_sha256": self.package_sha},
                   "scenarios": {sid: dict(row) for sid in release_gate.FULL_SCENARIOS}}
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
                if key == "original_attempt_sha256":
                    data["process"] = {"execution": "completed", "exit_code": 2,
                        "stdout_sha256": "a" * 64, "stderr_sha256": "b" * 64}
                else:
                    data.update(status="PASS", proof_kind="installed_blackbox")
                if key == "oracle_sha256":
                    data.update(independent_native=True, observed_expected_effect=True)
                if key == "installed_report_sha256":
                    data.update(full_graph_complete=True, unresolved=[],
                        nodes={node: "PASS" for node in release_gate.REQUIRED_NODES[sid]})
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

    def test_release_gate_tracks_every_canonical_scenario(self):
        import re
        seed = ROOT / "docs/verification/REACCEPTANCE_SYSTEM_SCENARIOS.md"
        canonical = set(re.findall(r"^\| `(SYS-(?:[A-Z]+-)+[0-9]{2})` P[01]", seed.read_text(), re.M))
        discovered = {row["id"] for row in scenarios.seed_rows(seed)}
        self.assertEqual(discovered, canonical, "Scenario parser silently dropped a canonical scenario")
        self.assertTrue(release_gate.MANDATORY <= canonical)
        self.assertEqual(release_gate.FULL_SCENARIOS, canonical)
        self.assertEqual(set(release_gate.REQUIRED_NODES), canonical)
        self.assertTrue(all(len(nodes) >= 3 for nodes in release_gate.REQUIRED_NODES.values()))

    def test_seven_mvp_scenarios_cannot_authorize_full_release(self):
        mvp = {"SYS-LIFE-01", "SYS-JOB-01", "SYS-UNKNOWN-01", "SYS-SELFTEST-01",
               "SYS-UPGRADE-01", "SYS-RELEASE-01", "SYS-REPORT-01"}
        a, b = self.signed(lambda d: d.update(scenarios={sid: row for sid, row in d["scenarios"].items()
                                                       if sid in mvp}))
        with self.assertRaisesRegex(AcceptanceFault, "FULL_SCENARIO_ACCOUNTING_REQUIRED"):
            self.check(a, b)

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
        a, b = self.signed(lambda d: d["verifier"].update(box_uuid="11111111-1111-4111-8111-111111111111"))
        with self.assertRaisesRegex(AcceptanceFault, "SAME_ACTOR"):
            self.check(a, b)
        a, b = self.signed(lambda d: d["scenarios"].pop("SYS-RELEASE-01"))
        with self.assertRaisesRegex(AcceptanceFault, "FULL_SCENARIO_ACCOUNTING_REQUIRED"):
            self.check(a, b)

    def test_signed_missing_oracle_and_invalid_install_rejected(self):
        a, b = self.signed()
        (self.root / "cases/SYS-LIFE-01/oracle.json").write_text('{"tampered":true}')
        with self.assertRaisesRegex(AcceptanceFault, "BYTES_MISMATCH"):
            self.check(a, b)
        a, b = self.signed(lambda d: d["verifier"].update(clean_installed_candidate=False))
        with self.assertRaisesRegex(AcceptanceFault, "CLEAN_INSTALL_UNPROVEN"):
            self.check(a, b)

    def resign_case(self, sid, filename, change):
        report, signature = self.signed()
        path = self.root / "cases" / sid / filename
        data = json.loads(path.read_text())
        change(data)
        path.write_text(json.dumps(data, sort_keys=True))
        field = {"original-attempt.json": "original_attempt_sha256",
                 "oracle.json": "oracle_sha256",
                 "installed-report.json": "installed_report_sha256"}[filename]
        receipt = json.loads(report.read_text())
        receipt["scenarios"][sid][field] = hashlib.sha256(path.read_bytes()).hexdigest()
        report.write_text(json.dumps(receipt, sort_keys=True))
        subprocess.run(["openssl", "dgst", "-sha256", "-sign", str(self.private),
                "-out", str(signature), str(report)], check=True, capture_output=True)
        return report, signature

    def test_signed_partial_failed_and_metadata_only_proofs_never_pass(self):
        for change in (lambda d: d.update(status="PARTIAL"),
                       lambda d: d.update(status="FAIL"),
                       lambda d: d.update(proof_kind="fixture")):
            with self.subTest(change=change):
                a, b = self.resign_case("SYS-LIFE-01", "installed-report.json", change)
                with self.assertRaisesRegex(AcceptanceFault, "NOT_INSTALLED_PASS"):
                    self.check(a, b)
        a, b = self.resign_case("SYS-LIFE-01", "installed-report.json",
            lambda d: d.update(nodes={"create": "PASS", "stop": "PASS", "start": "PASS", "delete": "PASS"}))
        with self.assertRaisesRegex(AcceptanceFault, "GRAPH_INCOMPLETE"):
            self.check(a, b)
        a, b = self.resign_case("SYS-LIFE-01", "oracle.json",
            lambda d: d.update(observed_expected_effect=False))
        with self.assertRaisesRegex(AcceptanceFault, "EFFECT_ORACLE_MISSING"):
            self.check(a, b)
        a, b = self.resign_case("SYS-JOB-01", "original-attempt.json", lambda d: d.pop("process"))
        with self.assertRaisesRegex(AcceptanceFault, "PROCESS_EVIDENCE_MISSING"):
            self.check(a, b)

    def test_required_scenario_override_cannot_weaken_gate(self):
        a, b = self.signed()
        with self.assertRaisesRegex(AcceptanceFault, "CANNOT_BE_WEAKENED"):
            release_gate.accept(self.package, self.release, a, b, self.public,
                                self.source, required={"SYS-LIFE-01"})

    def test_external_asset_mutation_or_manifest_rebinding_rejected(self):
        a, b = self.signed()
        (self.root / "online.py").write_bytes(b"changed executable")
        with self.assertRaisesRegex(AcceptanceFault, "ASSET_DIGEST_MISMATCH"):
            self.check(a, b)
        manifest = json.loads(self.release.read_text())
        manifest["files"]["online.py"] = hashlib.sha256(b"changed executable").hexdigest()
        self.release.write_text(json.dumps(manifest))
        with self.assertRaisesRegex(AcceptanceFault, "HELPER_CANDIDATE_MISMATCH"):
            self.check(a, b)

    def test_signed_manifest_binding_is_required(self):
        a, b = self.signed(lambda d: d.update(release_manifest_sha256="f" * 64))
        with self.assertRaisesRegex(AcceptanceFault, "VERIFIER_RELEASE_MANIFEST_MISMATCH"):
            self.check(a, b)

    def test_truthy_clean_install_string_refused(self):
        a, b = self.signed(lambda d: d["verifier"].update(clean_installed_candidate="false"))
        with self.assertRaisesRegex(AcceptanceFault, "CLEAN_INSTALL_UNPROVEN"):
            self.check(a, b)

    def test_signature_cannot_be_reused_for_other_workflow_run(self):
        context = {"candidate_run_id": 101, "candidate_run_attempt": 1, "candidate_artifact_id": 202,
                   "candidate_artifact_digest": "sha256:" + "a" * 64,
                   "verifier_run_id": 303, "verifier_run_attempt": 1}
        a, b = self.signed(lambda d: d.update(workflow=context))
        self.assertIn("APPROVED", release_gate.accept(self.package, self.release,
            a, b, self.public, self.source, expected_context=context)["decision"])
        with self.assertRaisesRegex(AcceptanceFault, "WORKFLOW_CONTEXT_MISMATCH"):
            release_gate.accept(self.package, self.release, a, b, self.public,
                self.source, expected_context=dict(context, verifier_run_id=304))

    def test_workflow_context_requires_all_attempts_and_exact_numeric_types(self):
        context = {"candidate_run_id": 101, "candidate_run_attempt": 1,
                   "candidate_artifact_id": 202,
                   "candidate_artifact_digest": "sha256:" + "a" * 64,
                   "verifier_run_id": 303, "verifier_run_attempt": 1}
        a, b = self.signed(lambda d: d.update(workflow=context))
        invalid = [dict(context, candidate_run_attempt=True),
                   dict(context, candidate_run_id="101"),
                   dict(context, candidate_artifact_id=0),
                   dict(context, candidate_artifact_digest="a" * 64),
                   {key: value for key, value in context.items() if key != "candidate_run_attempt"}]
        for values in invalid:
            with self.subTest(values=values), self.assertRaisesRegex(
                    AcceptanceFault, "EXPECTED_WORKFLOW_CONTEXT_INVALID"):
                release_gate.accept(self.package, self.release, a, b,
                    self.public, self.source, expected_context=values)
        for field in ("candidate_run_attempt", "verifier_run_attempt"):
            changed = dict(context, **{field: 2})
            with self.subTest(field=field), self.assertRaisesRegex(
                    AcceptanceFault, "WORKFLOW_CONTEXT_MISMATCH"):
                release_gate.accept(self.package, self.release, a, b,
                    self.public, self.source, expected_context=changed)
        a, b = self.signed(lambda d: d.update(workflow=dict(context, candidate_run_attempt=True)))
        with self.assertRaisesRegex(AcceptanceFault, "WORKFLOW_CONTEXT_MISMATCH"):
            release_gate.accept(self.package, self.release, a, b,
                self.public, self.source, expected_context=context)

    def test_applicable_extra_failure_or_unknown_cannot_authorize_release(self):
        for status in ("FAIL", "UNKNOWN", "PARTIAL", "BLOCKED"):
            a, b = self.signed(lambda d: d["scenarios"]["SYS-BACKUP-01"].update(status=status))
            with self.subTest(status=status), self.assertRaises(AcceptanceFault):
                self.check(a, b)

    def test_unsupported_feature_requires_exact_native_capability_evidence(self):
        sid = "SYS-BACKUP-01"
        folder = self.root / "cases" / sid
        folder.mkdir(parents=True)
        response_path = folder / "capability-response.json"
        stdout, stderr = "", "Unsupported capability: whole-box encrypted backup"
        response = {"execution": "completed", "exit_code": 2, "stdout": stdout, "stderr": stderr,
                    "stdout_sha256": hashlib.sha256(stdout.encode()).hexdigest(),
                    "stderr_sha256": hashlib.sha256(stderr.encode()).hexdigest()}
        response_path.write_text(json.dumps(response))
        proof = {"schema_version": 1, "scenario_id": sid, "actor_id": "verifier-2",
                 "candidate_sha256": self.package_sha, "result": "UNSUPPORTED",
                 "independent_native": True, "channel": "native_cli",
                 "capability": "whole-box encrypted backup",
                 "observed_response_sha256": hashlib.sha256(response_path.read_bytes()).hexdigest()}
        path = folder / "capability.json"
        path.write_text(json.dumps(proof))

        def issue():
            return self.signed(lambda d: d["scenarios"][sid].update(status="BLOCKED",
                applicability="unsupported", reason="Installed native Provider lacks whole-box encrypted backup",
                capability_evidence_sha256=hashlib.sha256(path.read_bytes()).hexdigest()))

        a, b = issue()
        result = self.check(a, b)
        self.assertEqual(result["unsupported_scenarios"], [sid])
        self.assertFalse(result["full_system_acceptance"])
        response_path.unlink()
        with self.assertRaisesRegex(AcceptanceFault, "NATIVE_RESPONSE_MISSING"):
            self.check(a, b)
        response_path.write_text('{"execution":"completed"}')
        with self.assertRaisesRegex(AcceptanceFault, "NATIVE_RESPONSE_BYTES_MISMATCH"):
            self.check(a, b)
        response_path.write_text(json.dumps(dict(response, stderr="changed output")))
        proof["observed_response_sha256"] = hashlib.sha256(response_path.read_bytes()).hexdigest()
        path.write_text(json.dumps(proof))
        a, b = issue()
        with self.assertRaisesRegex(AcceptanceFault, "NATIVE_OUTPUT_BYTES_MISMATCH"):
            self.check(a, b)
        response_path.write_text(json.dumps(dict(response, exit_code=True)))
        proof["observed_response_sha256"] = hashlib.sha256(response_path.read_bytes()).hexdigest()
        path.write_text(json.dumps(proof))
        a, b = issue()
        with self.assertRaisesRegex(AcceptanceFault, "NATIVE_PROCESS_INCOMPLETE"):
            self.check(a, b)
        path.write_text('{"result":"UNSUPPORTED"}')
        with self.assertRaisesRegex(AcceptanceFault, "CAPABILITY_BYTES_MISMATCH"):
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
