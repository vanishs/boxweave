"""Gateway MCP contract integration: read-only probe and gated restore forwarding."""
import io
import json
import unittest
from contextlib import redirect_stdout
from unittest.mock import patch

from bwmini import mcp_gate
from tests.test_mcp_compat import fixtures


class KeptBytes(io.BytesIO):
    def close(self):
        pass


class EchoStdout:
    def close(self):
        pass

    def __init__(self, process):
        self.process = process
        self.last = -1

    def readline(self, limit=0):
        messages = [json.loads(l) for l in
                    self.process.stdin.getvalue().decode().splitlines()]
        if len(messages) <= self.last + 1:
            return b""
        self.last += 1
        item = messages[self.last]
        if item["method"] == "initialize":
            return (json.dumps({"jsonrpc": "2.0", "id": item["id"],
                                "result": {"serverInfo": {
                                    "name": "containarium-mcp", "version": "0.120.1"},
                                    "capabilities": {"tools": {}}}}) + "\n").encode()
        if item["method"] == "tools/list":
            tools = list(fixtures().values())
            return (json.dumps({"jsonrpc": "2.0", "id": item["id"],
                                "result": {"tools": tools}}) + "\n").encode()
        if item["method"] == "tools/call":
            if item["params"]["name"] == "get_container":
                username = item["params"]["arguments"]["username"]
                if username != "demo":
                    return (json.dumps({"jsonrpc":"2.0","id":item["id"],
                                        "error":{"code":-32000,"message":"not found"}})+"\n").encode()
                observed={"container":{"name":"demo-container","createdAt":"123",
                          "username":"demo","network":{"ipAddress":"10.203.0.2"},
                          "state":"CONTAINER_STATE_RUNNING"}}
                return (json.dumps({"jsonrpc":"2.0","id":item["id"],
                                    "result":{"content":[{"type":"text","text":json.dumps(observed)}],
                                              "isError":False}})+"\n").encode()
            return (json.dumps({"jsonrpc": "2.0", "id": item["id"],
                                "result": {"content": [{"type": "text", "text": "ok"}],
                                           "isError": False}}) + "\n").encode()
        raise AssertionError(item)


class FakeProcess:
    def __init__(self):
        self.stdin = KeptBytes()
        self.stdout = EchoStdout(self)
    def poll(self):
        return 0


class GatewayFlowTests(unittest.TestCase):
    def exercise(self, request, *, approved=None, registry=None):
        fake = FakeProcess()
        hooks = [patch.object(mcp_gate, "tool_policy", return_value=None),
                 patch.object(mcp_gate.subprocess, "Popen", return_value=fake),
                 patch.object(mcp_gate.sys, "stdin",
                              io.StringIO("\n".join(json.dumps(x) for x in
                                  (request if isinstance(request, list) else [request])) + "\n"))]
        if approved is not None:
            hooks.append(patch.object(mcp_gate.restore_guard, "approve",
                                      return_value=approved))
        if registry is not None:
            hooks.append(patch.object(mcp_gate, "trusted_registry", return_value=registry))
        for hook in hooks:
            hook.__enter__()
        try:
            with redirect_stdout(io.StringIO()) as out:
                mcp_gate.main()
        finally:
            for hook in reversed(hooks):
                hook.__exit__(None, None, None)
        output = [json.loads(x) for x in out.getvalue().splitlines()]
        sent = [json.loads(x) for x in fake.stdin.getvalue().decode().splitlines()]
        return output, sent

    def test_connect_two_names_same_forwarded_identity(self):
        import time
        target={"name":"demo-container","created":"123","user":"demo",
                "ip":"10.203.0.2","keys":["fixture"]}
        registry={"observed_at":time.time(),"targets":{"demo":target}}
        for box in ("demo","demo-container"):
            with self.subTest(box=box):
                req={"jsonrpc":"2.0","id":21,"method":"tools/call",
                     "params":{"name":"connect","arguments":{"box":box,"exec":"id -un"}}}
                output,sent=self.exercise(req,registry=registry)
                self.assertNotIn("error",output[-1])
                self.assertEqual([x["params"]["name"] for x in sent],
                                 ["get_container","connect"])
                self.assertEqual(sent[0]["params"]["arguments"],{"username":"demo"})
                self.assertEqual(sent[1]["params"]["arguments"],{"box":"demo","exec":"id -un"})

    def test_connect_unknown_and_colliding_targets_do_not_forward(self):
        import time
        target={"name":"demo-container","created":"123","user":"demo",
                "ip":"10.203.0.2","keys":["fixture"]}
        registry={"observed_at":time.time(),"targets":{"demo":target}}
        req={"jsonrpc":"2.0","id":22,"method":"tools/call",
             "params":{"name":"connect","arguments":{"box":"absent-container","exec":"true"}}}
        out,sent=self.exercise(req,registry=registry)
        self.assertEqual(sent,[])
        self.assertEqual(out[-1]["error"]["message"],"TRUST_NOT_READY")
        ambiguous={**registry,"targets":{**registry["targets"],
                   "demo-container":{**target,"name":"demo-container-container","user":"demo-container"}}}
        req["params"]["arguments"]["box"]="demo-container"
        out,sent=self.exercise(req,registry=ambiguous)
        self.assertEqual(sent,[])
        self.assertEqual(out[-1]["error"]["message"],"BOX_TARGET_AMBIGUOUS")

    def test_connect_upstream_missing_target_is_typed(self):
        import time
        target={"name":"missing-container","created":"123","user":"missing",
                "ip":"10.203.0.2","keys":["fixture"]}
        registry={"observed_at":time.time(),"targets":{"missing":target}}
        req={"jsonrpc":"2.0","id":23,"method":"tools/call",
             "params":{"name":"connect","arguments":{"box":"missing-container","exec":"true"}}}
        out,sent=self.exercise(req,registry=registry)
        self.assertEqual(len(sent),1)
        self.assertEqual(sent[0]["params"]["arguments"],{"username":"missing"})
        self.assertEqual(out[-1]["error"]["message"],"TARGET_READ_FAILED")

    def test_compat_uses_only_read_only_catalog(self):
        request = {"jsonrpc": "2.0", "id": 12, "method": "tools/call",
                   "params": {"name": "boxweave_diagnose",
                              "arguments": {"source": "compat"}}}
        output, sent = self.exercise(request)
        self.assertEqual([x["method"] for x in sent], ["tools/list"])
        value = json.loads(output[-1]["result"]["content"][0]["text"])
        self.assertEqual(value["state"], "compatible_subset")
        self.assertEqual(len(value["fingerprint_sha256"]), 64)

    def test_privileged_restore_catalog_guard_works_on_paginated_page(self):
        upstream = {"name": "restore_box_backup", "inputSchema": {
            "type": "object", "properties": {
                "box": {"type": "string"}, "backup": {"type": "string"},
                "age_identity_file": {"type": "string"},
                "as": {"type": "string"}, "arbitrary_secret_path": {"type": "string"}}}}
        result = mcp_gate.guard_restore_catalog({"result": {"tools": [upstream]}})
        adapted = result["result"]["tools"][0]
        self.assertNotIn("age_identity_file", adapted["inputSchema"]["properties"])
        self.assertNotIn("arbitrary_secret_path", adapted["inputSchema"]["properties"])
        self.assertIn("key_id", adapted["inputSchema"]["properties"])
        self.assertEqual(len(mcp_gate.guard_restore_catalog({"result": {"tools": [
            {"name": "restore_box_backup", "inputSchema": {}}]}})["result"]["tools"]), 0)

    def test_compat_reuses_advertised_upstream_mcp_version(self):
        init = {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}
        check = {"jsonrpc": "2.0", "id": 2, "method": "tools/call",
                 "params": {"name": "boxweave_diagnose",
                            "arguments": {"source": "compat"}}}
        output, sent = self.exercise([init, check])
        self.assertEqual([x["method"] for x in sent], ["initialize", "tools/list"])
        self.assertEqual(output[0]["result"]["serverInfo"]["version"],
                         mcp_gate.SERVER_INFO_VERSION)
        result = json.loads(output[-1]["result"]["content"][0]["text"])
        self.assertEqual(result["upstream_version"], "0.120.1")
        self.assertEqual(result["version_source"], "initialize.serverInfo.version")

    def test_guarded_restore_never_forwards_alias_or_confirmation(self):
        request = {"jsonrpc": "2.0", "id": 13, "method": "tools/call",
                   "params": {"name": "restore_box_backup",
                              "arguments": {"box": "demo", "backup": "id",
                                            "as": "demo-restored", "key_id": "demo.key",
                                            "confirm": "id"}}}
        rewritten = {"box": "demo", "backup": "id", "as": "demo-restored",
                     "age_identity_file": "/home/boxweave/boxweave/.restore-keys/demo.key",
                     "force": False}
        output, sent = self.exercise(request, approved=rewritten)
        self.assertEqual(len(sent), 1)
        self.assertEqual(sent[0]["method"], "tools/call")
        self.assertEqual(sent[0]["params"]["arguments"], rewritten)
        self.assertNotIn("key_id", sent[0]["params"]["arguments"])
        self.assertNotIn("confirm", sent[0]["params"]["arguments"])
        self.assertFalse(output[-1]["result"]["isError"])


if __name__ == "__main__":
    unittest.main()

