"""Protected Gateway mutation classification and non-bypass fixtures."""
import io
import json
import os
from pathlib import Path
import stat
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
from contextlib import redirect_stdout

from bwmini import gateway_protection as protection
from bwmini import mcp_gate


class FakeProvider:
    def __init__(self):
        self.sent=[]
        self.responses=[]
        self.stdin=self
        self.stdout=self
    def write(self, raw):
        message=json.loads(raw)
        self.sent.append(message)
        if message.get("method")=="tools/list":
            tools=[{"name":n,"inputSchema":{"type":"object","properties":{}}}
                   for n in ("list_containers","get_container","get_system_info",
                             "connect","create_container","resize_container",
                             "delete_container","unverified_custom_method")]
            self.responses.append(json.dumps({
                "jsonrpc":"2.0","id":message["id"],"result":{"tools":tools}})+"\n")
        elif "id" in message:
            self.responses.append(json.dumps({
                "jsonrpc":"2.0","id":message["id"],
                "result":{"content":[{"type":"text","text":"{}"}]}})+"\n")
    def flush(self):pass
    def readline(self, *args):
        return (self.responses.pop(0).encode()
                if self.responses else b"")
    def poll(self):return 0
    def close(self):pass


class ProtectedGatewayTests(unittest.TestCase):
    def gateway(self, requests):
        provider=FakeProvider()
        output=io.StringIO()
        incoming=io.StringIO("".join(json.dumps(r)+"\n" for r in requests))
        with patch.object(mcp_gate.gateway_protection,"enabled",return_value=True), \
             patch.object(mcp_gate,"tool_policy",return_value=None), \
             patch.object(mcp_gate.subprocess,"Popen",return_value=provider), \
             patch.object(mcp_gate.gd,"record"), \
             patch.object(mcp_gate.sys,"stdin",incoming), \
             redirect_stdout(output):
            mcp_gate.main()
        return provider,[json.loads(row) for row in output.getvalue().splitlines()]

    def test_rejects_direct_and_unclassified_mutations_before_provider_call(self):
        tools=[("connect",{"box":"box1","exec":"echo unexpected"}),
               ("create_container",{"username":"newbox"}),
               ("resize_container",{"username":"box1","cpu":"4"}),
               ("delete_container",{"username":"box1"}),
               ("unverified_custom_method",{}),
               ("boxweave_nesting",{"action":"apply","logical":"box1"})]
        calls=[{"jsonrpc":"2.0","method":"tools/call","id":i,
                "params":{"name":name,"arguments":args}}
               for i,(name,args) in enumerate(tools,1)]
        provider,replies=self.gateway(calls)
        self.assertEqual(provider.sent,[])
        self.assertEqual(len(replies),len(calls))
        self.assertEqual(replies[0]["error"]["message"],"COORDINATION_USE_TRACKED_TASK")
        self.assertTrue(all(r["error"]["message"] in
            ("COORDINATION_USE_TRACKED_TASK","COORDINATION_TOOL_UNCLASSIFIED")
            for r in replies))

    def test_advertised_catalog_drops_unclassified_provider_mutators(self):
        provider,replies=self.gateway([{"jsonrpc":"2.0","method":"tools/list","id":8}])
        self.assertEqual(len(provider.sent),1)
        names={t["name"] for t in replies[0]["result"]["tools"]}
        self.assertIn("get_container",names)
        self.assertIn("boxweave_task_submit",names)
        self.assertIn("boxweave_box_lease",names)
        self.assertIn("boxweave_box_patch",names)
        self.assertIn("boxweave_box_read_file",names)
        self.assertNotIn("connect",names)
        self.assertNotIn("create_container",names)
        self.assertNotIn("resize_container",names)
        self.assertNotIn("unverified_custom_method",names)

    def test_source_only_mcp_policy_does_not_claim_per_agent_rbac(self):
        self.assertIsNone(protection.check_call(
            "boxweave_task_submit",{"box":"box1","request_key":"a"*32,"exec":"true"}))
        self.assertIsNone(protection.check_call("boxweave_nesting",{"action":"plan"}))
        self.assertIsNone(protection.check_call("list_containers",{}))
        with self.assertRaisesRegex(ValueError,"COORDINATION_TOOL_UNCLASSIFIED"):
            protection.check_call("boxweave_nesting",{"action":"apply"})
        with self.assertRaisesRegex(ValueError,"COORDINATION_USE_TRACKED_TASK"):
            protection.check_call("connect",{"box":"box1"})

    def test_absent_default_policy_is_legacy_not_false_enforcement(self):
        with patch.object(Path,"lstat",side_effect=FileNotFoundError):
            self.assertFalse(protection.enabled())
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaisesRegex(ValueError,"COORDINATION_POLICY_UNAVAILABLE"):
                protection.enabled(Path(d)/"not-installed.json")

    def test_root_only_policy_rejects_bad_modes_schema_and_symlinks(self):
        with tempfile.TemporaryDirectory() as d:
            folder=Path(d)
            folder.chmod(0o755)
            path=folder/"policy.json"
            path.write_text('{"version":1,"enabled":true}')
            path.chmod(0o644)
            with self.assertRaisesRegex(ValueError,"COORDINATION_POLICY_UNSAFE"):
                protection.enabled(path)
            original=Path.lstat
            def root_owned(obj):
                info=original(obj)
                if obj==path or obj==folder:
                    return SimpleNamespace(st_uid=0,st_mode=info.st_mode,
                        st_size=info.st_size,st_nlink=info.st_nlink,
                        st_ino=info.st_ino,st_dev=info.st_dev)
                return info
            with patch.object(Path,"lstat",root_owned):
                self.assertTrue(protection.enabled(path))
                path.write_text('{"version":1,"enabled":"yes"}')
                with self.assertRaisesRegex(ValueError,"COORDINATION_POLICY_INVALID"):
                    protection.enabled(path)
                path.write_text('{"version":1,"enabled":false}')
                self.assertFalse(protection.enabled(path))
                path.chmod(0o666)
                with self.assertRaisesRegex(ValueError,"COORDINATION_POLICY_UNSAFE"):
                    protection.enabled(path)
            path.unlink()
            path.symlink_to("/etc/passwd")
            with self.assertRaisesRegex(ValueError,"COORDINATION_POLICY_UNSAFE"):
                protection.enabled(path)


if __name__=="__main__":
    unittest.main()
