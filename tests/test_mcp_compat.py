"""P1: bounded read-only upstream MCP compatibility probe fixtures."""
import json
import unittest

from bwmini import mcp_compat as c


def fixtures():
    return {
        "list_containers": {"name": "list_containers", "inputSchema":
                            {"type": "object", "properties": {}}},
        "get_container": {"name": "get_container", "inputSchema":
                          {"type": "object", "properties": {"username": {"type": "string"}},
                           "required": ["username"]}},
        "connect": {"name": "connect", "inputSchema":
                    {"type": "object", "properties": {"box": {"type": "string"},
                                                      "exec": {"type": "string"}},
                     "required": ["box"]}},
        "create_container": {"name": "create_container", "inputSchema":
                             {"type": "object", "properties": {"username": {"type": "string"},
                                                               "ssh_keys": {"type": "array"}},
                              "required": ["username", "ssh_keys"]}},
        "debug_container": {"name": "debug_container", "inputSchema":
                            {"type": "object", "properties": {"username": {"type": "string"}},
                             "required": ["username"]}},
    }


class CompatibilityProbeTests(unittest.TestCase):
    def test_known_subset_and_fingerprint_stable(self):
        old = c.analyze(fixtures())
        self.assertEqual(old["state"], "compatible_subset")
        self.assertEqual(old["code"], "MCP_COMPAT_SUBSET_MATCH")
        self.assertEqual(len(old["fingerprint_sha256"]), 64)
        self.assertEqual(c.analyze(fixtures())["fingerprint_sha256"],
                         old["fingerprint_sha256"])
        self.assertEqual(old["response_contract"], "not_verified")
        self.assertEqual(old["upstream_version"], "not_verified")

    def test_box_or_username_contract_supported(self):
        f = fixtures()
        f["connect"]["inputSchema"]["properties"].pop("box")
        f["connect"]["inputSchema"]["properties"]["username"] = {"type": "string"}
        f["connect"]["inputSchema"]["required"] = ["username"]
        self.assertEqual(c.analyze(f)["state"], "compatible_subset")

    def test_change_missing_wrong_type_and_new_required(self):
        f = fixtures()
        f["create_container"]["inputSchema"]["properties"]["ssh_keys"]["type"] = "string"
        self.assertEqual(c.analyze(f)["state"], "schema_changed")
        self.assertIn("ssh_keys", next(v for v in c.analyze(f)["checks"]
                                      if v["tool"] == "create_container")["fields"])
        f = fixtures()
        f["connect"]["inputSchema"]["properties"].pop("box")
        self.assertIn("box|username", next(v for v in c.analyze(f)["checks"]
                                           if v["tool"] == "connect")["fields"])
        f = fixtures()
        f["list_containers"]["inputSchema"]["required"] = ["token"]
        self.assertIn("new_required_field", next(v for v in c.analyze(f)["checks"]
                                                 if v["tool"] == "list_containers")["fields"])
        f = fixtures()
        f.pop("get_container")
        self.assertEqual(c.analyze(f)["state"], "tool_missing")
        self.assertTrue(any(v["tool"] == "get_container" and v["state"] == "missing"
                            for v in c.analyze(f)["checks"]))

    def test_allowlist_excluded_not_misreported_missing(self):
        f = fixtures()
        f.pop("debug_container")
        r = c.analyze(f, allowed=frozenset({"get_container", "connect"}))
        self.assertEqual(r["state"], "compatible_subset")
        self.assertEqual(next(x for x in r["checks"] if x["tool"] == "debug_container")["state"],
                         "not_checked")
        self.assertEqual(c.analyze({}, allowed=frozenset())["state"], "not_verified")

    def test_paged_read_only_catalog_and_secrecy(self):
        calls, replies = [], [
            {"result": {"tools": [fixtures()["list_containers"]],
                        "nextCursor": "second"}},
            {"result": {"tools": [fixtures()["connect"], {
                "name": "private_secret", "description": "SECRET_DO_NOT_LEAK",
                "inputSchema": {"type": "object", "properties": {"password": {"type": "string"}}}
            }]}}
        ]
        def send(request):
            calls.append(request)
        def reply(ident, forward):
            self.assertFalse(forward)
            self.assertEqual(ident, calls[-1]["id"])
            return replies.pop(0)
        result = c.read_tools(send, reply)
        self.assertEqual(set(result), {"list_containers", "connect"})
        self.assertEqual([r["method"] for r in calls], ["tools/list", "tools/list"])
        self.assertEqual(calls[1]["params"], {"cursor": "second"})
        self.assertNotIn("SECRET_DO_NOT_LEAK", json.dumps(c.analyze(result)))

    def test_untrusted_catalog_fails_and_never_runs_tool(self):
        for response in (
            {"error": {"message": "SECRET"}},
            {"result": {"tools": None}},
            {"result": {"tools": [{"name": "connect"}, {"name": "connect"}]}},
            {"result": {"tools": [], "nextCursor": "x"}},
        ):
            calls = []
            def send(message):
                calls.append(message)
            def reply(_id, _forward):
                if len(calls) > 1:
                    return response
                return response
            if response.get("result", {}).get("nextCursor"):
                # The repeated cursor must be rejected.
                with self.assertRaises(ValueError):
                    c.read_tools(send, reply)
            else:
                with self.assertRaises(ValueError):
                    c.read_tools(send, reply)
            self.assertTrue(all(m["method"] == "tools/list" for m in calls))
        self.assertEqual(c.unavailable()["state"], "not_verified")


if __name__ == "__main__":
    unittest.main()

