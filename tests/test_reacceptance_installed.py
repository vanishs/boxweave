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

    def test_process_failure_never_passes(self):
        self.assertEqual(installed.result_kind(None), "NO_JSON_RPC_RESPONSE")
        self.assertEqual(installed.result_kind({"error": {"code": 400}}), "RPC_ERROR")
        self.assertEqual(installed.result_kind({"result": {"isError": True}}), "RPC_ERROR")


if __name__ == "__main__":
    unittest.main()

