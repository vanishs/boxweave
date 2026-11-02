"""D3/D4 gateway diagnostics fixtures: no real Provider, Box or host privileges."""
import json
import os
from pathlib import Path
import sqlite3
import tempfile
import time
import unittest
from unittest.mock import patch

from bwmini import gate_diagnostics as d, mcp_gate


class GatewayDiagnosticsTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        root = Path(self.tmp.name) / "owned"
        root.mkdir(mode=0o700)
        root.chmod(0o700)
        self.path = root / "private" / "journal.sqlite3"
        self.config = {"enabled": True, "auto_diagnose": True,
                       "retention_days": 7, "max_storage_mb": 1}

    def tearDown(self):
        self.tmp.cleanup()

    def test_durable_record_and_query_across_readers(self):
        trace = d.record("create_container", {"result": {"content": [{"text": "password=EXAMPLE_SECRET"}]}},
                         12, path=self.path, config=self.config)
        result = d.query_gateway({"source": "gateway", "trace_id": trace},
                                 path=self.path, config=self.config)
        self.assertEqual(result["state"], "observed")
        self.assertEqual(result["events"][0]["tool"], "create_container")
        self.assertEqual(result["events"][0]["outcome"], "succeeded")
        self.assertNotIn("EXAMPLE_SECRET", json.dumps(result))
        self.assertNotIn(b"EXAMPLE_SECRET", self.path.read_bytes())
        self.assertEqual(self.path.stat().st_mode & 0o777, 0o600)
        self.assertEqual(d.query_gateway({"source": "gateway", "recent": 1},
                                         path=self.path, config=self.config)["events"][0]["trace_id"], trace)

    def test_record_error_unknown_and_no_payload_capture(self):
        trace = d.record("connect", {"error": {"message": "JWT=EXAMPLE_SECRET"}},
                         34, path=self.path, config=self.config)
        row = d.query_gateway({"source": "gateway", "trace_id": trace},
                              path=self.path, config=self.config)["events"][0]
        self.assertEqual((row["outcome"], row["code"]), ("unknown", "UPSTREAM_JSONRPC_ERROR"))
        self.assertNotIn(b"EXAMPLE_SECRET", self.path.read_bytes())
        self.assertEqual(d.classify({"result": {"isError": True}})[0], "failed")
        self.assertEqual(d.classify({})[0], "unknown")

    def test_disabled_missing_and_invalid_selectors(self):
        trace = d.record("create_container", {}, 1, path=self.path,
                         config={**self.config, "enabled": False})
        self.assertIsInstance(trace, str)
        self.assertFalse(self.path.exists())
        self.assertEqual(d.query_gateway({"source": "gateway", "recent": 1},
                         path=self.path, config=self.config)["state"], "not_verified")
        for selector in ({"source": "gateway"}, {"source": "gateway", "recent": 50},
                         {"source": "gateway", "recent": True},
                         {"source": "gateway", "trace_id": "../etc/passwd"},
                         {"source": "gateway", "trace_id": "0" * 32, "recent": 1},
                         {"source": "gateway", "recent": 1, "actor": "admin"}):
            with self.subTest(selector=selector), self.assertRaisesRegex(ValueError, "DIAGNOSTIC_SELECTOR_INVALID"):
                d.query_gateway(selector, path=self.path, config=self.config)
        with self.assertRaises(ValueError):
            d.validate_settings({"retention_days": 1000})
        with self.assertRaises(ValueError):
            d.validate_settings({"max_storage_mb": 0})
        self.assertEqual(d.DEFAULTS["retention_days"], 7)

    def test_path_symlink_and_permissive_directory_fail_safe(self):
        other = self.path.parent.parent / "other.db"
        other.write_text("foreign")
        self.path.parent.symlink_to(other)
        d.record("connect", {}, 1, path=self.path, config=self.config)
        self.assertEqual(d.query_gateway({"source": "gateway", "recent": 1},
                                         path=self.path, config=self.config)["state"], "not_verified")
        self.path.parent.unlink()
        self.path.parent.mkdir(mode=0o755)
        self.path.parent.chmod(0o755)
        d.record("connect", {}, 1, path=self.path, config=self.config)
        self.assertFalse(self.path.exists())

    def test_retention_enforced_and_tool_schema(self):
        trace = d.record("connect", {"result": {}}, 1,
                         path=self.path, config=self.config)
        with sqlite3.connect(self.path) as conn:
            conn.execute("UPDATE events SET utc=? WHERE trace=?", (time.time() - 10 * 86400, trace))
        d.record("debug_container", {"result": {}}, 2,
                 path=self.path, config=self.config)
        self.assertEqual(d.query_gateway({"source": "gateway", "trace_id": trace},
                                         path=self.path, config=self.config)["events"], [])
        names = {t["name"] for t in mcp_gate.catalog({"result": {"tools": []}})["result"]["tools"]}
        self.assertTrue({"boxweave_diagnose", "boxweave_project_diagnose"} <= names)
        filtered = mcp_gate.filter_tools(mcp_gate.catalog({"result": {"tools": []}}),
                                         frozenset(["boxweave_diagnose"]))
        self.assertEqual([t["name"] for t in filtered["result"]["tools"]], ["boxweave_diagnose"])

    def test_app_fixed_reader_and_no_arbitrary_actor_or_paths(self):
        for query in ({"source": "app", "trace_id": "bad"},
                      {"source": "app", "trace_id": "a" * 32, "actor": "other"},
                      {"source": "app", "operation_kind": "environment", "operation_id": "../secret"},
                      {"source": "app", "operation_kind": "bad", "operation_id": "foo"}):
            with self.subTest(query=query), self.assertRaises(ValueError):
                d.query_app(query)
        with patch.object(d.subprocess, "run", side_effect=OSError("SECRET")):
            report = d.query_app({"source": "app", "trace_id": "a" * 32})
            self.assertEqual(report["state"], "not_verified")
            self.assertNotIn("SECRET", json.dumps(report))
        self.assertIn("boxweave", d.APP_PREFIX.as_posix())

    def test_d4_default_deny_and_exact_identity(self):
        args = {"box": "testbox", "check": "service", "service": "app.service"}
        with self.assertRaisesRegex(ValueError, "PROJECT_DIAG_NOT_AUTHORIZED"):
            d.preauthorize_project(args, policy_path=Path(self.tmp.name) / "missing.json")
        policy = {"version": 1, "boxes": {"testbox": {
            "instance_name": "testbox-container", "created": "created1",
            "checks": ["processes", "service", "logs", "ports"],
            "services": ["app.service"]}}}
        obs = {"username": "testbox", "name": "testbox-container",
               "createdAt": "created1", "state": "CONTAINER_STATE_RUNNING"}
        self.assertEqual(d.project_grant(args, obs, policy=policy)["created"], "created1")
        for changes in ({"createdAt": "new"}, {"name": "other"},
                        {"state": "CONTAINER_STATE_STOPPED"}, {"username": "other"}):
            with self.subTest(changes=changes), self.assertRaisesRegex(ValueError, "PROJECT_TARGET_CHANGED"):
                d.project_grant(args, obs | changes, policy=policy)
        for bad in ({"box": "other", "check": "ports"},
                    {"box": "testbox", "check": "logs", "service": "other.service"},
                    {"box": "testbox", "check": "service", "service": "a;cat /etc/shadow"},
                    {"box": "testbox", "check": "logs", "service": "-n"},
                    {"box": "testbox", "check": "ports", "service": "app.service"}):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                d.preauthorize_project(bad, policy=policy)
        self.assertEqual(d.command_for_check(args),
                         "systemctl --user is-active -- app.service")

    def test_d4_read_summaries_never_expose_raw_output(self):
        samples = [
            ({"box": "testbox", "check": "processes"}, "S python\nZ private", "zombie_count", 1),
            ({"box": "testbox", "check": "ports"}, "LISTEN SECRET\n", "sample_listening_tcp_count", 1),
            ({"box": "testbox", "check": "logs", "service": "app.service"},
             "Token=EXAMPLE_SECRET\nERROR private\n", "error_hint_count", 1),
            ({"box": "testbox", "check": "service", "service": "app.service"},
             "active\n", "service_state", "active"),
        ]
        for args, output, field, value in samples:
            with self.subTest(args=args):
                response = {"structuredContent": {"exit_code": 0, "stdout": output}}
                result = d.project_summary(args, response)
                self.assertEqual(result[field], value)
                self.assertNotIn("SECRET", json.dumps(result))
                self.assertNotIn("private", json.dumps(result))
        self.assertEqual(d.project_summary({"check": "logs"}, {"isError": True})["state"], "not_verified")
        self.assertEqual(d.project_summary({"check": "logs"},
                         {"structuredContent": {"exit_code": 0, "stdout": "x" * 9999}})["state"],
                         "not_verified")

    def test_d4_service_inactive_and_failed_are_valid_systemctl_observations(self):
        args = {"box": "testbox", "check": "service", "service": "app.service"}
        for service_state, code in (("inactive", 3), ("failed", 3), ("unknown", 4)):
            with self.subTest(service_state=service_state):
                result = d.project_summary(args, {"structuredContent": {
                    "exit_code": code, "stdout": service_state + "\n"}})
                self.assertEqual(result["state"], "observed")
                self.assertEqual(result["service_state"], service_state)
                self.assertEqual(result["exit_code"], code)
        for bad_code, output in ((1, "failed\n"), (3, "active\n"),
                                 (3, "arbitrary input\n")):
            with self.subTest(bad_code=bad_code, output=output):
                result = d.project_summary(args, {"structuredContent": {
                    "exit_code": bad_code, "stdout": output}})
                self.assertEqual(result["state"], "not_verified")
        unrelated = d.project_summary({"check": "ports"}, {"structuredContent": {
            "exit_code": 3, "stdout": "inactive\n"}})
        self.assertEqual(unrelated["state"], "not_verified")

    def test_mcp_gateway_passthrough_records_without_rewriting_provider(self):
        import io
        from contextlib import redirect_stdout
        from unittest.mock import patch
        class Fake:
            def __init__(self, source):
                self.stdin = io.BytesIO()
                self.stdout = io.BytesIO(source)
            def poll(self):
                return 0
        request = {"jsonrpc": "2.0", "id": 8, "method": "tools/call",
                   "params": {"name": "list_containers", "arguments": {}}}
        upstream = {"jsonrpc": "2.0", "id": 8, "result": {"content": [
            {"type": "text", "text": "EXAMPLE_PRIVATE_BODY"}]}}
        fake = Fake((json.dumps(upstream) + "\n").encode())
        recorded = []
        with (patch.object(mcp_gate, "tool_policy", return_value=None),
              patch.object(mcp_gate.subprocess, "Popen", return_value=fake),
              patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(request) + "\n")),
              patch.object(mcp_gate.gd, "record", side_effect=lambda *a, **kw: recorded.append((a, kw))),
              redirect_stdout(io.StringIO()) as output):
            mcp_gate.main()
        self.assertEqual(json.loads(output.getvalue()), upstream)
        self.assertEqual([entry[0][0] for entry in recorded], ["list_containers"])

    def test_d4_denies_before_upstream_and_returns_no_raw_log_on_success(self):
        import io
        from contextlib import redirect_stdout
        from unittest.mock import patch
        class KeepOpen(io.BytesIO):
            def close(self):
                pass
        class Fake:
            def __init__(self, source):
                self.stdin = KeepOpen()
                self.stdout = io.BytesIO(source)
            def poll(self):
                return 0
        request = {"jsonrpc": "2.0", "id": 7, "method": "tools/call",
                   "params": {"name": "boxweave_project_diagnose",
                              "arguments": {"box": "devbox", "check": "ports"}}}
        fake = Fake(b"")
        with (patch.object(mcp_gate, "tool_policy", return_value=None),
              patch.object(mcp_gate.subprocess, "Popen", return_value=fake),
              patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(request) + "\n")),
              patch.object(mcp_gate.gd, "preauthorize_project",
                           side_effect=ValueError("PROJECT_DIAG_NOT_AUTHORIZED")),
              redirect_stdout(io.StringIO()) as output):
            mcp_gate.main()
        self.assertIn("PROJECT_DIAG_NOT_AUTHORIZED", output.getvalue())
        self.assertEqual(fake.stdin.getvalue(), b"")

        observed = {"username": "devbox", "name": "devbox-container",
                    "createdAt": "123", "state": "CONTAINER_STATE_RUNNING"}
        container_reply = {"jsonrpc": "2.0", "id": "boxweave-nesting-7",
                           "result": {"content": [{"type": "text",
                                                 "text": json.dumps({"container": observed})}]}}
        exec_reply = {"jsonrpc": "2.0", "id": "boxweave-d4-" + "a" * 16,
                      "result": {"structuredContent": {
                          "exit_code": 0, "stdout": "LISTEN private_token\n"}}}
        fake = Fake((json.dumps(container_reply) + "\n" +
                     json.dumps(exec_reply) + "\n").encode())
        with (patch.object(mcp_gate, "tool_policy", return_value=None),
              patch.object(mcp_gate.subprocess, "Popen", return_value=fake),
              patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(request) + "\n")),
              patch.object(mcp_gate.secrets, "token_hex", side_effect=lambda n: "a" * (2 * n)),
              patch.object(mcp_gate.gd, "preauthorize_project", return_value={}),
              patch.object(mcp_gate.gd, "project_grant", return_value={}),
              patch.object(mcp_gate, "trusted", return_value=None),
              patch.object(mcp_gate, "REGISTRY", type("RegistryFixture", (), {"read_text": lambda self: "{}"})()),
              redirect_stdout(io.StringIO()) as output):
            mcp_gate.main()
        result = json.loads(output.getvalue())
        summary = json.loads(result["result"]["content"][0]["text"])
        self.assertEqual(summary["sample_listening_tcp_count"], 1)
        self.assertNotIn("private_token", json.dumps(summary))
        sent = fake.stdin.getvalue().decode()
        self.assertIn('"name": "get_container"', sent)
        self.assertIn('"name": "connect"', sent)


if __name__ == "__main__":
    unittest.main()
