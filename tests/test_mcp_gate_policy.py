"""M1 direct gateway allowlist: deny before forwarding, no client-side grants."""
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
from bwmini.mcp_gate import filter_tools, tool_policy


class GatePolicyTests(unittest.TestCase):
    def test_catalogue_and_unknown_tool_are_denied(self):
        catalog={"result":{"tools":[{"name":"get_container"},{"name":"get_secret"}]}}
        filtered=filter_tools(catalog, frozenset({"get_container"}))
        self.assertEqual([x["name"] for x in filtered["result"]["tools"]],["get_container"])
        self.assertNotIn("get_secret", frozenset({"get_container"}))
        self.assertEqual(filter_tools({"result":{"tools":[]}},None)["result"]["tools"], [])

    def test_operator_owned_policy_file_and_fail_closed(self):
        with tempfile.TemporaryDirectory() as folder:
            path=Path(folder)/"policy.json"
            policy={"version":1,"allowed_tools":["get_container"]}
            path.write_text(json.dumps(policy))
            path.chmod(0o600)
            # A temporary user-owned directory is deliberately not accepted
            # as an operator-owned system policy directory.
            with self.assertRaises(ValueError):tool_policy(path)
            path.write_text(json.dumps({"version":1,"allowed_tools":["x","x"]}))
            with self.assertRaises(ValueError):tool_policy(path)

    def test_denied_call_is_never_forwarded_to_upstream(self):
        import io
        from contextlib import redirect_stdout
        from bwmini import mcp_gate
        class KeepOpen(io.BytesIO):
            def close(self):
                pass
        class FakeProcess:
            def __init__(self):
                self.stdin=KeepOpen()
                self.stdout=KeepOpen()
            def poll(self):return 0
        fake=FakeProcess()
        request={"jsonrpc":"2.0","id":1,"method":"tools/call",
                 "params":{"name":"get_secret","arguments":{"name":"private"}}}
        with (patch.object(mcp_gate,"tool_policy",return_value=frozenset({"list_containers"})),
              patch.object(mcp_gate.subprocess,"Popen",return_value=fake),
              patch.object(mcp_gate.sys,"stdin",io.StringIO(json.dumps(request)+"\n")),
              redirect_stdout(io.StringIO()) as out):
            mcp_gate.main()
        self.assertEqual(fake.stdin.getvalue(),b"")
        self.assertEqual(json.loads(out.getvalue())["error"]["message"],"GATE_TOOL_NOT_ALLOWED")

    def test_malformed_catalogue_rejected(self):
        with self.assertRaises(ValueError):
            filter_tools({"result":{"tools":None}},frozenset({"get_container"}))


if __name__=="__main__":
    unittest.main()
