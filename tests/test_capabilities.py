"""M0: read-only MCP capability discovery must not call tools or grant rights."""
import unittest
from bwmini.capabilities import classify, snapshot
from bwmini.domain import Fault
from test_slice import WebTests


class CapabilityTests(unittest.TestCase):
    def test_source_inventory_never_grants_or_executes(self):
        value = classify({
            "get_container": {"name": "get_container", "inputSchema": {"type": "object", "properties": {}}},
            "get_secret": {"name": "get_secret", "inputSchema": {"type": "object", "properties": {}}},
            "surprise_admin": {"name": "surprise_admin", "inputSchema": {"type": "object"}},
        })
        self.assertEqual(value["count"], 3)
        self.assertEqual(value["authorization"], "not_established_by_discovery")
        self.assertEqual(value["direct_agent_gateway"], "not_checked")
        self.assertEqual(value["tools"][0]["app_mapping"], "typed_adapter")
        self.assertEqual(value["tools"][1]["risk_hint"], "sensitive_review")
        self.assertEqual(value["tools"][2]["app_mapping"], "not_integrated")
        self.assertEqual(len(value["tools"][0]["schema_sha256"]), 64)

    def test_bad_catalogue_rejects_without_dispatch(self):
        for catalogue in (
            [], {"bad-name": {"name": "bad-name", "inputSchema": {"type": "object"}}},
            {"foo": {"name": "bar", "inputSchema": {"type": "object"}}},
            {"foo": "not-an-object"},
        ):
            with self.subTest(catalogue=catalogue), self.assertRaises(Fault):
                classify(catalogue)

    def test_missing_schema_is_honestly_unverified(self):
        value = classify({"future_tool": {"name": "future_tool"}})
        self.assertEqual(value["tools"][0]["schema"], "unverified")
        self.assertIsNone(value["tools"][0]["schema_sha256"])

    def test_fixture_not_marked_live(self):
        class Fixture:
            mode = "fixture"
        self.assertEqual(snapshot(Fixture())["state"], "not_checked")

    def test_live_client_never_calls_tools(self):
        calls = []
        class FakeClient:
            def __enter__(self): return self
            def __exit__(self, *args): pass
            def discover(self):
                calls.append("tools/list")
                return {"list_containers": {"name": "list_containers",
                                            "inputSchema": {"type": "object"}}}
            def tool(self, *_): raise AssertionError("tools/call must never execute")
        class Provider:
            mode = "mcp"
            def session(self): return FakeClient()
        self.assertEqual(snapshot(Provider())["count"], 1)
        self.assertEqual(calls, ["tools/list"])


class CapabilityWebTests(WebTests):
    def test_capabilities_requires_login_and_refuses_fixture_success(self):
        self.assertEqual(self.client.get("/api/mcp-capabilities").status_code, 401)
        self.login()
        result = self.client.get("/api/mcp-capabilities")
        self.assertEqual(result.status_code, 200)
        self.assertEqual(result.json["state"], "not_checked")
        self.assertEqual(result.json["count"], 0)
        self.assertEqual(result.json["authorization"], "not_established_by_discovery")


if __name__ == "__main__":
    unittest.main()
