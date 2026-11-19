#!/usr/bin/env python3
"""Mock loopback wire-level regression suite; no Incus, OAuth or external server."""
from __future__ import annotations

import json
import os
from pathlib import Path
import socket
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import unittest
from unittest.mock import patch

from mcp_testkit import (LoopbackMCP, FaultInjectingMCP, MCPProtocolError,
                         MCPTransportError, MCPToolError, MCPUnsafeMutation,
                         ProbeReport, token_from_env, token_from_file)


class MockMCPHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    calls = []
    unsafe_calls = 0
    mode = "normal"
    lock = threading.Lock()

    def log_message(self, fmt, *args):  # no output containing sensitive values
        pass

    def do_POST(self):
        if self.path != "/mcp":
            return self.respond(404, {})
        length = int(self.headers.get("Content-Length", "0"))
        try:
            payload = json.loads(self.rfile.read(length))
        except ValueError:
            return self.respond(400, {})
        with self.lock:
            self.calls.append({"method": payload.get("method"),
                               "name": (payload.get("params") or {}).get("name"),
                               "auth": self.headers.get("Authorization", "")})
        if self.mode == "redirect":
            return self.respond(307, {})
        if self.mode == "invalid-json":
            return self.respond(200, "not-json", raw=True)
        method, params = payload.get("method"), payload.get("params") or {}
        if method == "notifications/initialized":
            return self.respond(202, None)
        if method == "initialize":
            result = {"protocolVersion": "2025-11-25", "capabilities": {"tools": {}},
                      "serverInfo": {"name": "mock-boxweave", "version": "test"}}
        elif method == "tools/list":
            result = {"tools": [
                {"name": n, "annotations": {"readOnlyHint": True}}
                for n in ("host_status", "operation_list", "operation_get", "box_activity_get", "error_tool", "legacy_text")
            ] + [{"name": "box_create", "annotations": {"readOnlyHint": False}},
                 {"name": "unannotated"}], "nextCursor": ""}
        elif method == "tools/call":
            name, args = params.get("name"), params.get("arguments") or {}
            if name in ("box_create", "unannotated"):
                with self.lock:
                    type(self).unsafe_calls += 1
                result = {"content": [], "structuredContent": {"operation_id": "unsafe-id", "state": "ADMITTED"}}
            elif name == "host_status":
                result = {"content": [], "structuredContent": {"schema_version": 5, "tracking_id": "abc", "mcp_ready": True}}
            elif name == "operation_list":
                cursor = args.get("cursor")
                ops = [{"operation_id": "a"*32, "state": "UNKNOWN", "work_ref": "work/demo"}]
                result = {"content": [], "structuredContent": {"page": {
                    "operations": ops if not cursor else [], "next_cursor": ""}}}
            elif name == "operation_get":
                result = {"content": [], "structuredContent": {"operation": {
                    "operation_id": args["operation_id"], "state": "UNKNOWN",
                    "next_action": {"kind": "HOST_LOCAL_REVIEW"}}}}
            elif name == "legacy_text":
                result = {"content": [{"type":"text","text":json.dumps({"tracking_id":"legacy","answer":123})}]}
            elif name == "box_activity_get":
                view = args.get("view", "overview")
                if view == "issues":
                    activity = {"issues": [{"reason": "UNRESOLVED_OPERATION",
                                "operation_id": "a"*32, "resource_ref": "box:demo"}],
                                "source": "SQLITE_SNAPSHOT", "attention_complete": False,
                                "truncated": False}
                elif view == "evidence":
                    activity = {"evidence": {"original": {"operation_id": args["operation_id"]},
                              "limitations": ["NO_NATIVE_POSTSTATE"]},
                              "source": "SQLITE_SNAPSHOT", "attention_complete": False}
                else:
                    activity = {"operations": [{"operation_id": "a"*32}],
                                "source": "SQLITE_SNAPSHOT", "attention_complete": False}
                result = {"content": [], "structuredContent": {"activity": activity}}
            else:
                result = {"isError": True, "content": [{"type": "text", "text": "sensitive-data"}]}
        else:
            result = None
        if self.mode == "wrong-id" and method != "notifications/initialized":
            payload_id = -42
        else:
            payload_id = payload.get("id")
        response = {"jsonrpc": "2.0", "id": payload_id, "result": result}
        if self.mode == "sse":
            data = "event: message\ndata: " + json.dumps(response) + "\n\n"
            return self.respond(200, data, content_type="text/event-stream", raw=True)
        return self.respond(200, response)

    def respond(self, status, value, *, content_type="application/json", raw=False):
        if value is None:
            body = b""
        elif raw:
            body = value.encode("utf-8")
        else:
            body = json.dumps(value).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Content-Type", content_type)
        self.send_header("Cache-Control", "private, no-store")
        self.end_headers()
        self.wfile.write(body)


class LoopbackSuite(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), MockMCPHandler)
        cls.port = cls.server.server_port
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=5)

    def setUp(self):
        MockMCPHandler.calls = []
        MockMCPHandler.unsafe_calls = 0
        MockMCPHandler.mode = "normal"

    def client(self, cls=LoopbackMCP):
        return cls(f"http://127.0.0.1:{self.port}/mcp", token="test_secret_never_log")

    def test_initialize_and_catalog(self):
        with self.client() as c:
            self.assertEqual(c.protocol, "2025-11-25")
            self.assertEqual(len(c.all_tools()), 8)
            self.assertEqual(c.host_status()["schema_version"], 5)
            self.assertIn("no-store", c.last_cache_control)
        self.assertTrue(all(x["auth"] == "Bearer test_secret_never_log" for x in MockMCPHandler.calls))

    def test_mutation_unannotated_rejected_without_http_effect(self):
        with self.client() as c:
            with self.assertRaises(MCPUnsafeMutation):
                c.call("box_create", {"name": "should-not-create"})
            with self.assertRaises(MCPUnsafeMutation):
                c.call("unannotated")
        self.assertEqual(MockMCPHandler.unsafe_calls, 0)

    def test_disposable_explicit_mutation_opt_in(self):
        with LoopbackMCP(f"http://127.0.0.1:{self.port}/mcp", token="testing", allow_mutations=True) as c:
            result = c.call("box_create", {"name": "fixture"})
            self.assertEqual(result.structured["state"], "ADMITTED")
        self.assertEqual(MockMCPHandler.unsafe_calls, 1)

    def test_pagination_recovery_and_unknown_no_auto_retry(self):
        with self.client() as c:
            operations = list(c.walk_operations(work_ref="work/demo"))
            self.assertEqual(len(operations), 1)
            issues = list(c.walk_activity("issues", work_ref="work/demo"))
            self.assertEqual(len(issues), 1)
            manifest = c.recover_work("work/demo")
            self.assertEqual(manifest["unknown_originals"], ["a"*32])
            self.assertTrue(manifest["requires_manual_review"])
            self.assertFalse(manifest["safe_to_retry_unknown"])
            self.assertEqual(c.poll_operation("a"*32, seconds=0)["next_action"]["kind"], "HOST_LOCAL_REVIEW")
        self.assertEqual(MockMCPHandler.unsafe_calls, 0)

    def test_fault_injection_after_response_read_only(self):
        with self.client(FaultInjectingMCP) as c:
            c.drop_next_read_reply = True
            with self.assertRaises(MCPTransportError):
                c.activity("issues")
            self.assertEqual(c.activity("issues")["issues"][0]["reason"], "UNRESOLVED_OPERATION")
            self.assertFalse(c.drop_next_read_reply)
        self.assertEqual(MockMCPHandler.unsafe_calls, 0)

    def test_http_redirect_refused(self):
        with self.client() as c:
            MockMCPHandler.mode = "redirect"
            with self.assertRaises(MCPTransportError):
                c.host_status()

    def test_wrong_response_id_and_invalid_json_rejected(self):
        with self.client() as c:
            for mode in ("wrong-id", "invalid-json"):
                MockMCPHandler.mode = mode
                with self.subTest(mode=mode), self.assertRaises(MCPProtocolError):
                    c.host_status()
                MockMCPHandler.mode = "normal"

    def test_sse_bounded_one_reply(self):
        with self.client() as c:
            MockMCPHandler.mode = "sse"
            self.assertEqual(c.host_status()["schema_version"], 5)

    def test_error_result_redacts_secret(self):
        with self.client() as c:
            with self.assertRaises(MCPToolError) as cm:
                c.call("error_tool")
            self.assertNotIn("sensitive-data", str(cm.exception))

    def test_legacy_text_fallback_is_not_arbitrary_text(self):
        with self.client() as c:
            self.assertEqual(c.call("legacy_text").structured["answer"],123)

    def test_remote_url_and_auth_in_url_forbidden(self):
        invalid = ["https://127.0.0.1:8000/mcp", "http://example.com:8000/mcp",
                   "http://127.0.0.2:8000/mcp", "http://127.0.0.1/mcp",
                   "http://token@127.0.0.1:8000/mcp", "http://127.0.0.1:8000/mcp?x=1",
                   "http://127.0.0.1:8000/other", "http://127.0.0.1:8000/mcp#f"]
        for url in invalid:
            with self.subTest(url=url), self.assertRaises(ValueError):
                LoopbackMCP(url, token="x")

    def test_ipv6_loopback_allowed_but_not_other_ipv6(self):
        LoopbackMCP("http://[::1]:18650/mcp", token="x")
        with self.assertRaises(ValueError):
            LoopbackMCP("http://[fe80::1]:18650/mcp", token="x")

    def test_secure_token_file_and_env(self):
        with tempfile.TemporaryDirectory() as d:
            file = Path(d) / "token"
            file.write_text("topsecret\n", encoding="utf-8")
            os.chmod(file, 0o600)
            self.assertEqual(token_from_file(file), "topsecret")
            os.chmod(file, 0o644)
            with self.assertRaises(ValueError):
                token_from_file(file)
        with patch.dict(os.environ, {"BOXWEAVE_MCP_TOKEN": "env_secret"}):
            self.assertEqual(token_from_env(), "env_secret")
        with self.assertRaises(ValueError):
            token_from_env("BAD-NAME")

    def test_report_excludes_arbitrary_secret_details(self):
        report = ProbeReport("local-mcp-probe")
        report.check("safe_check", True, detail="secret_not_printed")
        with self.assertRaises(AssertionError):
            report.check("expected_failure", False, detail="private_token")
        raw = json.dumps(report.as_dict())
        self.assertNotIn("secret_not_printed", raw)
        self.assertNotIn("private_token", raw)
        with self.assertRaises(ValueError):
            report.check("unsafe token in name", True)

    def test_cli_smoke_and_safe_default(self):
        from mcp_lab import main
        with patch.dict(os.environ, {"BOXWEAVE_MCP_TOKEN": "cli-secret"}):
            self.assertEqual(main(["--url", f"http://127.0.0.1:{self.port}/mcp", "smoke",
                                   "--expected-tools", "8", "--expected-schema", "5", "--require-activity"]), 0)
            self.assertEqual(main(["--url", f"http://127.0.0.1:{self.port}/mcp", "call", "box_create"]), 1)
        self.assertEqual(MockMCPHandler.unsafe_calls, 0)


if __name__ == "__main__":
    unittest.main()
