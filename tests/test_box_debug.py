"""A limited Platform debug read must not leak raw upstream SSH logs."""
import json
import unittest
from bwmini.box_debug import summary
from bwmini.domain import Fault
from test_slice import TARGET, Base, PASSWORD, HASH
from bwmini.web import create_app


class FakeRest:
    def __init__(self,response):
        self.response=response
        self.requested=[]
    def debug_container(self,logical):
        self.requested.append(("debug_container",logical))
        return self.response
    def container(self,logical):
        self.requested.append(("get_container",logical))
        return {"username":TARGET.logical,"name":TARGET.instance,
                "createdAt":TARGET.created}

class FakeProvider:
    mode="mcp"
    def __init__(self,rest):
        self.rest=rest
    def session(self):
        raise AssertionError("debug_container must never fall back to MCP")


class SummaryTests(unittest.TestCase):
    def test_safe_summary_drops_ssh_logs_and_raw_causes(self):
        rest=FakeRest({"containerState":"running","hostUserExists":True,
            "hostUserShellExists":False,"recentSshdRejections":["SECRET"],
            "likelyCause":"SENSITIVE","nextActions":["private"],
            "hostUserShell":"/bin/private"})
        result=summary(FakeProvider(rest),TARGET)
        self.assertTrue(result["host_user_exists"])
        self.assertFalse(result["host_user_shell_exists"])
        self.assertEqual(rest.requested,[
            ("debug_container",TARGET.logical),("get_container",TARGET.logical)])
        self.assertNotIn("SECRET",json.dumps(result))
        self.assertNotIn("SENSITIVE",json.dumps(result))
        self.assertFalse(result["boxweave_host_ssh_verified"])

    def test_missing_rest_and_invalid_response_fail_closed(self):
        with self.assertRaisesRegex(Fault,"PROVIDER_API_REQUIRED"):
            summary(FakeProvider(None),TARGET)
        for payload in ({}, {"containerState":"running",
                              "hostUserExists":"yes"}):
            with self.subTest(payload=payload),self.assertRaises(Fault):
                summary(FakeProvider(FakeRest(payload)),TARGET)

    def test_unknown_upstream_state_never_claimed_healthy(self):
        result=summary(FakeProvider(FakeRest({
            "containerState":"error: private detail"})),TARGET)
        self.assertEqual(result["container_state"],"unverified")


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
