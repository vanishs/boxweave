"""MCP agent self-guide: topic validation, delegated catalog and gateway."""
import io
import json
import unittest
from pathlib import Path
from contextlib import redirect_stdout
from unittest.mock import patch
from bwmini import mcp_gate


class GuideTests(unittest.TestCase):
    def test_static_topics_and_invalid_arguments(self):
        data = mcp_gate.boxweave_info({})
        self.assertEqual(set(data["sections"]), set(mcp_gate.INFO_TOPICS))
        self.assertEqual(set(mcp_gate.boxweave_info({"topic": "development"})["sections"]),
                         {"development"})
        self.assertIn("docs/DEVELOPMENT.md",
                      data["sections"]["documentation"]["entrypoints"])
        self.assertIn("not live runtime", data["scope"])
        self.assertEqual(data["source_version"],
                         (Path(__file__).resolve().parents[1] / "VERSION").read_text().strip())
        self.assertIn("docs/process/AGENT_TEST_DEPLOYMENT.md",
                      data["sections"]["documentation"]["entrypoints"])
        self.assertIn("new authorized disposable Box",
                      data["sections"]["development"]["steps"][1])
        for args in (None, [], {"topic": "bad"}, {"topic": 5},
                     {"topic": "all", "host": "localhost"}):
            with self.assertRaises(ValueError):
                mcp_gate.boxweave_info(args)

    def test_delegated_tool_description_and_allowlist(self):
        data = {"result": {"tools": [
            {"name": "create_container", "description": "ephemeral private key",
             "inputSchema": {"properties": {"ssh_keys": {}},
                             "required": ["username", "ssh_keys"]}},
            {"name": "connect", "description": "override host"},
            {"name": "list_containers"},
            {"name": "delete_container", "description": "delete"}]}}
        tools = mcp_gate.catalog(data)["result"]["tools"]
        self.assertEqual([tool["name"] for tool in tools][-4:],
                         ["boxweave_nesting", "boxweave_info",
                          "boxweave_diagnose", "boxweave_project_diagnose"])
        self.assertIn("managed workload SSH public key", tools[0]["description"])
        self.assertNotIn("ssh_keys", tools[0]["inputSchema"]["required"])
        self.assertIn("exact-target", tools[1]["description"])
        self.assertIn("Permanently", tools[3]["description"])
        self.assertTrue(tools[-1]["annotations"]["readOnlyHint"])
        filtered = mcp_gate.filter_tools({"result": {"tools": tools}},
                                         frozenset({"boxweave_info"}))
        self.assertEqual([t["name"] for t in filtered["result"]["tools"]],
                         ["boxweave_info"])

    def test_gateway_info_no_upstream_and_stdio_initialize(self):
        class KeepOpen(io.BytesIO):
            def close(self):
                pass

        class FakeProcess:
            def __init__(self, output=b""):
                self.stdin = KeepOpen()
                self.stdout = KeepOpen(output)
            def poll(self):
                return 0

        fake = FakeProcess()
        request = {"jsonrpc": "2.0", "id": 7, "method": "tools/call",
                   "params": {"name": "boxweave_info",
                              "arguments": {"topic": "security"}}}
        with (patch.object(mcp_gate, "tool_policy", return_value=None),
              patch.object(mcp_gate.subprocess, "Popen", return_value=fake),
              patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(request)+"\n")),
              redirect_stdout(io.StringIO()) as stdout):
            mcp_gate.main()
        self.assertEqual(fake.stdin.getvalue(), b"")
        response = json.loads(stdout.getvalue())["result"]
        content = json.loads(response["content"][0]["text"])
        self.assertEqual(set(content["sections"]), {"security"})

        upstream = {"jsonrpc": "2.0", "id": 8, "result": {
            "protocolVersion": "2024-11-05", "capabilities": {},
            "serverInfo": {"name": "containarium", "version": "0.120.1"}}}
        fake = FakeProcess((json.dumps(upstream)+"\n").encode())
        init = {"jsonrpc": "2.0", "id": 8, "method": "initialize", "params": {}}
        with (patch.object(mcp_gate, "tool_policy", return_value=None),
              patch.object(mcp_gate.subprocess, "Popen", return_value=fake),
              patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(init)+"\n")),
              redirect_stdout(io.StringIO()) as stdout):
            mcp_gate.main()
        metadata = json.loads(stdout.getvalue())["result"]
        self.assertEqual(metadata["serverInfo"]["name"], "boxweave")
        self.assertEqual(metadata["serverInfo"]["version"],
                         mcp_gate.SERVER_INFO_VERSION)
        self.assertIn("boxweave_info", metadata["instructions"])


if __name__ == "__main__":
    unittest.main()
