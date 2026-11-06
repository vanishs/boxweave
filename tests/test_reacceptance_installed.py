"""Regression tests for the separate, non-mutating installed breadth runner."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from tools.reacceptance import installed
from tools.reacceptance.engine import AcceptanceFault


class InstalledReadOnlyTests(unittest.TestCase):
    def test_required_target_schema_is_not_synthesized(self):
        ok, why = installed.positive_arguments(
            {"properties": {"username": {"type": "string"}},
             "required": ["username"]}, None)
        self.assertIsNone(ok)
        self.assertEqual(why, "REQUIRES_REVIEWED_SAFE_INPUT_username")
        ok, why = installed.positive_arguments(
            {"properties": {"username": {"type": "string"}},
             "required": ["username"]}, "racc-owned-1")
        self.assertEqual(ok, {"username": "racc-owned-1"})
        self.assertIsNone(why)
        self.assertEqual(installed.positive_arguments(
            {"required": ["password"], "properties": {}}, "racc-owned-1")[0], None)

    def test_runtime_catalog_blocks_all_unreviewed_mutators(self):
        fake = [{"name": "create_container", "inputSchema": {}},
                {"name": "list_containers", "inputSchema": {}},
                {"name": "get_container",
                 "inputSchema": {"required": ["username"],
                                 "properties": {"username": {"type": "string"}}}}]
        calls = []

        def session(argv, requests, **kwargs):
            calls.append(requests)
            if requests[0]["method"] == "tools/list":
                return {"messages": [{"id": "racc-list", "result": {"tools": fake}}],
                        "process": {"exit_code": 0}}
            return {"messages": [{"id": "racc-mvp", "result": {"content": []}}],
                    "process": {"exit_code": 0, "execution": "completed"}}

        with patch.object(installed, "mcp_session", side_effect=session):
            result = installed.inspect_mcp(["python3", "-m", "bwmini.mcp_gate"])
        self.assertEqual(result["catalogue"], "DISCOVERED")
        self.assertEqual(result["rows"]["stdio_mcp:create_container"]["status"], "BLOCKED")
        self.assertEqual(result["rows"]["stdio_mcp:get_container"]["status"], "BLOCKED")
        self.assertEqual(result["rows"]["stdio_mcp:list_containers"]["status"], "PARTIAL")
        self.assertEqual(len(calls), 3)
        self.assertFalse(any(x["params"]["name"] == "create_container"
                             for req in calls[1:] for x in req))

    def test_task_get_requires_original_identity_not_empty_positive(self):
        fake = [{"name": "boxweave_task_get",
                 "inputSchema": {"type": "object",
                                 "properties": {"task_id": {"type": "string"},
                                                "request_key": {"type": "string"}}}}]
        calls = []
        def session(argv, requests, **kwargs):
            calls.append(requests)
            return {"messages": [{"id": "racc-list", "result": {"tools": fake}}],
                    "process": {"execution": "completed", "exit_code": 0}}
        with patch.object(installed, "mcp_session", side_effect=session):
            r = installed.inspect_mcp(["python3", "-m", "bwmini.mcp_gate"])
        self.assertEqual(len(calls), 1)
        self.assertEqual(r["rows"]["stdio_mcp:boxweave_task_get"]["status"],
                         "BLOCKED")
        self.assertEqual(r["rows"]["stdio_mcp:boxweave_task_get"]["reason"],
                         "ORIGINAL_TASK_ID_OR_REQUEST_KEY_REQUIRED")

    def test_readonly_public_doctor_is_blocked_not_functional_fail(self):
        public = {"state": "installed_public_descriptor", "checks":
                  [{"id": "app_worker_unit", "state": "not_verified"}]}
        def mock_capture(argv, **kw):
            if argv[-1] == "doctor":
                return {"exit_code": 2, "execution": "completed",
                        "stdout": json.dumps(public), "stderr": ""}
            return {"exit_code": 2, "execution": "completed",
                    "stdout": "", "stderr": ""}
        with patch.object(installed, "help_tree", return_value={}), (
             patch.object(installed, "capture", side_effect=mock_capture)):
            result = installed.inspect_cli(["boxweave"])
        self.assertEqual(result["rows"]["host_cli:doctor"]["status"], "BLOCKED")

    def test_stale_target_cannot_be_used_for_real_probe(self):
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaisesRegex(AcceptanceFault, "UNSAFE_READ_ONLY_TARGET"):
                installed.execute(".", Path(d) / "output",
                                  owned="../foreign")
            self.assertFalse((Path(d) / "output").exists())

    def test_missing_transport_has_no_false_success(self):
        with tempfile.TemporaryDirectory() as d:
            result = installed.execute(".", Path(d) / "out")
            saved = json.loads((Path(d) / "out" / "original.json").read_text())
            self.assertEqual(result["passed_system_scenarios"], 0)
            self.assertFalse(result["has_independent_oracle"])
            self.assertEqual(saved["rows"]["transport:stdio_mcp"]["status"], "BLOCKED")
            self.assertTrue(all(row["status"] != "PASS" for row in saved["rows"].values()))
            with self.assertRaises(AcceptanceFault):
                installed.execute(".", Path(d) / "out")

    def test_help_tree_does_not_recurse_root_fallback(self):
        from tools.reacceptance import surfaces
        from unittest.mock import patch
        seen = []
        def fake_capture(argv, **kwargs):
            seen.append(argv)
            # A poorly behaving installed CLI prints root help even for
            # an unknown command: its root choices must not recurse again.
            return {"execution": "completed", "exit_code": 0,
                    "stdout": "usage: host_cli.py [-h] {alpha,beta} ...\\n\\n"
                    "positional arguments:\\n  {alpha,beta} ...",
                    "stderr": ""}
        with patch.object(surfaces, "capture", side_effect=fake_capture):
            result = surfaces.help_tree(["boxweave"])
        self.assertEqual(len(seen), 3)
        self.assertEqual(set(result), {"<root>", "alpha", "beta"})
        self.assertEqual(result["alpha"]["reason"],
                         "HELP_FELL_BACK_TO_DIFFERENT_COMMAND")

    def test_reject_untrusted_sudo_python_mcp(self):
        from tools.reacceptance import engine
        with tempfile.TemporaryDirectory() as d:
            cfg = Path(d) / "argv.json"
            cfg.write_text(json.dumps([
                "sudo", "-n", "python3", "/tmp/mcp_gate.py"]))
            with self.assertRaisesRegex(AcceptanceFault, "EXECUTABLE_NOT_ALLOWLISTED"):
                engine.argv_file(cfg, "mcp")
            cfg.write_text(json.dumps([
                "sudo", "-n", "python3",
                "/var/lib/boxweave-default/releases/0.49.0-" +
                "f" * 40 + "/mcp_gate.py"]))
            with self.assertRaisesRegex(AcceptanceFault, "EXECUTABLE_NOT_ALLOWLISTED"):
                engine.argv_file(cfg, "mcp")

    def test_installed_gateway_requires_actual_app_user_policy(self):
        from tools.reacceptance import engine
        with tempfile.TemporaryDirectory() as d:
            cfg = Path(d) / "mcp.json"
            trusted = "/usr/local/lib/boxweave/default/mcp_gate.py"
            good = ["sudo","-n","-u","boxweave","python3",trusted]
            cfg.write_text(json.dumps(good))
            with patch.object(engine,"trusted_installed_public_mcp",
                              return_value=True):
                self.assertEqual(engine.argv_file(cfg, "mcp"), good)
            cfg.write_text(json.dumps(["sudo","-n","python3",
                "/var/lib/boxweave-default/releases/0.49.0-" +
                "f"*40 + "/mcp_gate.py"]))
            with self.assertRaisesRegex(AcceptanceFault, "EXECUTABLE_NOT_ALLOWLISTED"):
                engine.argv_file(cfg, "mcp")

    def test_process_failure_never_passes(self):
        self.assertEqual(installed.result_kind(None), "NO_JSON_RPC_RESPONSE")
        self.assertEqual(installed.result_kind({"error": {"code": 400}}), "RPC_ERROR")
        self.assertEqual(installed.result_kind({"result": {"isError": True}}), "RPC_ERROR")


if __name__ == "__main__":
    unittest.main()

