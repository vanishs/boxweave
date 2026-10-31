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

    def test_malformed_catalogue_rejected(self):
        with self.assertRaises(ValueError):
            filter_tools({"result":{"tools":None}},frozenset({"get_container"}))


if __name__=="__main__":
    unittest.main()
