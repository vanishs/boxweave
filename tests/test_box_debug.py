"""A limited Platform debug read must not leak raw upstream SSH logs."""
import json
import unittest
from bwmini.box_debug import summary
from bwmini.domain import Fault
from test_slice import TARGET, Base, PASSWORD, HASH
from bwmini.web import create_app


class FakeClient:
    def __init__(self, response, schema=None):
        self.response = response
        self.requested = []
        self.schema = ({"type": "object", "properties": {"username": {"type": "string"}}}
                       if schema is None else schema)
    def __enter__(self): return self
    def __exit__(self, *_): return False
    def discover(self):
        return {"debug_container": {"inputSchema": self.schema}}
    def tool(self, name, args):
        self.requested.append((name, args))
        return {"content": [{"type": "text", "text": json.dumps(self.response)}]}


class FakeProvider:
    mode = "mcp"
    def __init__(self, client):
        self.client = client
    def session(self): return self.client


class SummaryTests(unittest.TestCase):
    def test_safe_summary_drops_ssh_logs_and_raw_causes(self):
        client = FakeClient({"containerState": "running", "hostUserExists": True,
            "hostUserShellExists": False, "recentSshdRejections": ["SECRET"],
            "likelyCause": "SENSITIVE", "nextActions": ["private"], "hostUserShell": "/bin/private"})
        result = summary(FakeProvider(client), TARGET)
        self.assertEqual(result["host_user_exists"], True)
        self.assertEqual(result["host_user_shell_exists"], False)
        self.assertEqual(client.requested, [("debug_container", {"username": TARGET.logical})])
        self.assertNotIn("SECRET", json.dumps(result))
        self.assertNotIn("SENSITIVE", json.dumps(result))
        self.assertFalse(result["boxweave_host_ssh_verified"])

    def test_invalid_schema_and_response_fail_closed(self):
        for payload, schema in (
            ({}, {}),
            ({"containerState": "running", "hostUserExists": "yes"}, None),
        ):
            with self.subTest(payload=payload), self.assertRaises(Fault):
                summary(FakeProvider(FakeClient(payload, schema)), TARGET)

    def test_unknown_upstream_state_never_claimed_healthy(self):
        result = summary(FakeProvider(FakeClient({"containerState": "error: private detail"})), TARGET)
        self.assertEqual(result["container_state"], "unverified")


class DebugWebTests(Base):
    def setUp(self):
        super().setUp()
        self.store.admin(HASH)
        self.app = create_app(self.s, "http://localhost", self.root/"dist", secure=False, clock=lambda:self.now[0])
        self.client = self.app.test_client()

    def test_requires_auth_and_bound_exact_target(self):
        uri = "/api/boxes/"+TARGET.logical+"/debug-summary"
        self.assertEqual(self.client.get(uri).status_code, 401)
        self.client.post("/api/login", json={"password": PASSWORD},
                         headers={"Origin": "http://localhost"})
        self.assertEqual(self.client.get(uri).status_code, 409)  # fixture is not live
        self.assertEqual(self.client.get("/api/boxes/forbidden/debug-summary").status_code, 403)


if __name__ == "__main__":
    unittest.main()
