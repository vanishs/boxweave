"""Independent web security, graph persistence and batch semantics (fake provider)."""

import json
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "server"))
try:
    from webui.app import create_app, HASHER
    from webui.provider import ProviderError, Provider, limits
    from webui.graph import markdown, validate
    from webui.store import Store, Conflict
except ImportError:
    create_app = None


class FakeProvider:
    def __init__(self):
        self.data = {}
        self.calls = []
        self.fail = {}

    def boxes(self):
        return {"containers": list(self.data.values())}

    def box(self, n):
        if n not in self.data:
            raise ProviderError(404)
        return self.data[n]

    def execute(self, a, i):
        self.calls.append((a, i))
        n = i.get("username", "")
        if n in self.fail:
            raise self.fail[n]
        if a == "create":
            self.data[n] = {
                "username": n,
                "name": n + "-container",
                "state": "CONTAINER_STATE_RUNNING",
            }
        return {}

    def routes(self, k):
        return {"routes": []}


@unittest.skipIf(
    create_app is None,
    "Install server/webui/requirements.lock in an isolated environment",
)
class WebTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.password_hash = HASHER.hash("a sufficiently long password")

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.cfg = {
            "database": self.temp.name + "/db.sqlite",
            "origin": "https://console.test:18443",
            "static": self.temp.name,
        }
        self.provider = FakeProvider()
        self.app = create_app(self.cfg, self.provider)
        self.app.testing = True
        self.addCleanup(self.app.extensions["batch_pool"].shutdown, True)
        self.c = self.app.test_client()
        self.store = self.app.extensions["store"]
        self.csrf = ""

    def req(self, path, method="GET", data=None, csrf=True, origin=True):
        headers = {}
        if origin:
            headers["Origin"] = self.cfg["origin"]
        if csrf:
            headers["X-CSRF-Token"] = self.csrf
        return self.c.open(
            "/api/" + path,
            method=method,
            json=data,
            headers=headers,
            base_url=self.cfg["origin"],
        )

    def login(self):
        with self.store.db() as db:
            db.execute(
                "INSERT OR REPLACE INTO admin VALUES (1,?)", (self.password_hash,)
            )
        r = self.req("login", "POST", {"password": "a sufficiently long password"})
        self.assertEqual(r.status_code, 200)
        self.csrf = r.json["csrf"]
        return r

    def graph(self, title=""):
        return {
            "name": title,
            "prompt": "Inspect services",
            "nodes": [
                {
                    "id": "n1",
                    "type": "resource",
                    "title": "alpha",
                    "comment": "first\nsecond",
                    "resource": {"type": "box", "id": "alpha"},
                    "position": {"x": 1, "y": 2},
                }
            ],
            "edges": [],
        }

    def wait_job(self, jid):
        for _ in range(200):
            r = self.req("jobs/" + jid)
            if r.json["finished"]:
                return r.json
            time.sleep(0.01)
        self.fail("Job did not finish")

    def batch(self, action, items):
        plan = self.req("plans", "POST", {"action": action, "items": items})
        self.assertEqual(plan.status_code, 200, plan.json)
        result = self.req(
            "plans/" + plan.json["id"] + "/confirm", "POST", {"confirm": True}
        )
        self.assertEqual(result.status_code, 202, result.json)
        return self.wait_job(result.json["id"])

    def test_uninitialized_and_unauthenticated_denied(self):
        self.assertFalse(self.req("status").json["initialized"])
        for path in ["boxes", "metrics", "graphs", "jobs", "network/route"]:
            self.assertEqual(self.req(path).status_code, 401)
        self.assertEqual(self.req("graphs", "POST", self.graph()).status_code, 401)

    def test_secure_cookie_session_logout_and_csrf(self):
        cookie = self.login().headers["Set-Cookie"]
        self.assertIn("Secure", cookie)
        self.assertIn("HttpOnly", cookie)
        self.assertIn("SameSite=Strict", cookie)
        self.assertEqual(
            self.req("graphs", "POST", self.graph(), csrf=False).status_code, 403
        )
        self.assertEqual(
            self.req("graphs", "POST", self.graph(), origin=False).status_code, 403
        )
        self.assertEqual(self.req("logout", "POST", {}).status_code, 200)
        self.assertEqual(self.req("boxes").status_code, 401)

    def test_http_cookie_is_not_secure(self):
        self.cfg["origin"] = "http://console.test:18443"
        app = create_app(self.cfg, self.provider)
        self.addCleanup(app.extensions["batch_pool"].shutdown, True)
        self.app = app
        self.c = app.test_client()
        self.store = app.extensions["store"]
        self.assertNotIn("Secure;", self.login().headers["Set-Cookie"])

    def test_login_rate_limit_origin_and_expiry(self):
        self.assertEqual(
            self.req("login", "POST", {"password": "wrong"}, origin=False).status_code,
            403,
        )
        for _ in range(5):
            self.assertEqual(
                self.req("login", "POST", {"password": "wrong"}).status_code, 401
            )
        self.assertEqual(
            self.req("login", "POST", {"password": "wrong"}).status_code, 429
        )

    def test_expired_session(self):
        self.login()
        with self.store.db() as db:
            db.execute("UPDATE sessions SET expires=0")
        self.assertEqual(self.req("boxes").status_code, 401)

    def test_graph_persistence_duplicates_free_nodes_and_conflict(self):
        self.login()
        g = self.graph()
        g["nodes"] += [
            {**g["nodes"][0], "id": "n2", "comment": "different"},
            {
                "id": "n3",
                "type": "text",
                "title": "Note",
                "comment": "free content",
                "position": {"x": 50, "y": 60},
            },
        ]
        g["edges"] = [
            {
                "id": "e1",
                "source": "n1",
                "target": "n2",
                "title": "shared",
                "comment": "connection comment",
            }
        ]
        r = self.req("graphs", "POST", g)
        self.assertEqual(r.status_code, 201, r.json)
        saved = r.json
        update = {**g, "version": saved["version"], "name": "Named"}
        self.assertEqual(
            self.req("graphs/" + saved["id"], "PUT", update).status_code, 200
        )
        self.assertEqual(
            self.req("graphs/" + saved["id"], "PUT", update).status_code, 409
        )
        restored = Store(self.cfg["database"]).graph(saved["id"])
        self.assertEqual(restored["nodes"], validate(g)["nodes"])
        md = self.req("graphs/" + saved["id"] + "/markdown").json["markdown"]
        for value in [
            "# BoxWeave",
            "MCP: boxweave",
            "Inspect services",
            "first",
            "second",
            "different",
            "free content",
            "connection comment",
            " -- ",
        ]:
            self.assertIn(value, md)
        self.assertNotIn("CPU", md)
        self.assertNotIn("position", md)
        self.assertEqual(md, markdown(restored))

    def test_select_delete_and_history_retained(self):
        self.login()
        a = self.req("graphs", "POST", self.graph()).json
        b = self.req("graphs", "POST", self.graph("second")).json
        self.req("graphs/" + a["id"] + "/select", "POST", {})
        self.assertEqual(self.req("graphs").json["current"], a["id"])
        self.assertEqual(len(self.req("graphs").json["graphs"]), 2)
        self.assertEqual(
            self.req(
                "graphs/" + b["id"], "DELETE", {"confirm": False, "version": 1}
            ).status_code,
            400,
        )
        self.assertEqual(
            self.req(
                "graphs/" + b["id"], "DELETE", {"confirm": True, "version": 1}
            ).status_code,
            200,
        )

    def test_graph_validation_secrets_and_dangling_edges(self):
        self.login()
        g = self.graph()
        g["prompt"] = "-----BEGIN RSA PRIVATE KEY-----"
        self.assertEqual(self.req("graphs", "POST", g).status_code, 400)
        g = self.graph()
        g["edges"] = [{"id": "e", "source": "n1", "target": "missing"}]
        self.assertEqual(self.req("graphs", "POST", g).status_code, 400)
        g = self.graph()
        g["nodes"][0]["position"]["x"] = float("inf")
        self.assertEqual(self.req("graphs", "POST", g).status_code, 400)

    def test_no_empty_relations_and_markdown_escaping(self):
        g = self.graph()
        g["nodes"][0]["title"] = "[x]<script>"
        out = markdown(g)
        self.assertIn("\\[x\\]\\<script\\>", out)
        self.assertNotIn("Relations", out)

    def test_batch_partial_success_and_unknown_create(self):
        self.login()
        self.provider.fail = {
            "bad": ProviderError(400),
            "slow": ProviderError(502, True),
        }
        items = [
            {
                "username": n,
                "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
                "image": "images:ubuntu/24.04",
            }
            for n in ["alpha", "bad", "slow"]
        ]
        j = self.batch("create", items)
        self.assertEqual(
            [i["status"] for i in j["items"]], ["submitted", "failed", "unknown"]
        )
        self.assertIn("alpha", self.provider.data)
        r = self.req("jobs/" + j["id"] + "/reconcile", "POST", {})
        self.assertEqual(r.json["items"][0]["status"], "success")
        self.assertEqual(r.json["items"][2]["status"], "unknown")
        plan = self.req("plans", "POST", {"action": "create", "items": [items[2]]}).json
        self.assertEqual(
            self.req(
                "plans/" + plan["id"] + "/confirm", "POST", {"confirm": True}
            ).status_code,
            400,
        )

    def test_conflicts_validation_and_explicit_fields(self):
        self.login()
        self.provider.data["taken"] = {"username": "taken"}
        for username in ["taken", "Bad Name", "containarium-core-x", "../escape"]:
            r = self.req(
                "plans",
                "POST",
                {
                    "action": "create",
                    "items": [
                        {
                            "username": username,
                            "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
                        }
                    ],
                },
            )
            self.assertEqual(r.status_code, 400)
        for action in ["start", "stop", "delete"]:
            j = self.batch(action, [{"username": "alpha"}, {"username": "beta"}])
            self.assertTrue(all(i["status"] == "success" for i in j["items"]))
        self.batch("resize", [{"username": "alpha", "resources": {"cpu": "4"}}])
        self.assertEqual(
            self.provider.calls[-1],
            ("resize", {"username": "alpha", "resources": {"cpu": "4"}}),
        )
        self.assertEqual(
            self.req(
                "plans",
                "POST",
                {
                    "action": "resize",
                    "items": [{"username": "alpha", "resources": {"image": "other"}}],
                },
            ).status_code,
            400,
        )

    def test_one_time_confirmation(self):
        self.login()
        p = self.req(
            "plans", "POST", {"action": "stop", "items": [{"username": "alpha"}]}
        ).json
        self.assertEqual(
            self.req(
                "plans/" + p["id"] + "/confirm", "POST", {"confirm": False}
            ).status_code,
            400,
        )
        r = self.req("plans/" + p["id"] + "/confirm", "POST", {"confirm": True})
        self.wait_job(r.json["id"])
        self.assertEqual(
            self.req(
                "plans/" + p["id"] + "/confirm", "POST", {"confirm": True}
            ).status_code,
            409,
        )

    def test_no_proxy_or_unverified_operation(self):
        self.login()
        for path in [
            "proxy?url=http://example.com",
            "network/edit",
            "http://127.0.0.1:8080",
        ]:
            self.assertEqual(self.req(path).status_code, 404)
        self.assertEqual(
            self.req(
                "plans", "POST", {"action": "route-update", "items": [{}]}
            ).status_code,
            400,
        )
        self.assertEqual(
            self.req(
                "plans",
                "POST",
                {
                    "action": "stop",
                    "items": [{"username": "alpha", "url": "http://evil"}],
                },
            ).status_code,
            400,
        )

    def test_security_headers_and_path_traversal(self):
        self.login()
        r = self.req("boxes")
        self.assertIn("frame-ancestors 'none'", r.headers["Content-Security-Policy"])
        self.assertNotIn("token", r.get_data(as_text=True).lower())
        self.assertEqual(
            self.c.get(
                "/assets/../../server/tokens.py", base_url=self.cfg["origin"]
            ).status_code,
            404,
        )

    def test_database_mode_and_schema_failure(self):
        self.assertEqual(os.stat(self.cfg["database"]).st_mode & 0o777, 0o600)
        with self.store.db() as db:
            db.execute("PRAGMA user_version=99")
        with self.assertRaises(RuntimeError):
            Store(self.cfg["database"])

    def test_jobs_recover_as_unknown_without_replay(self):
        self.store.job(
            "interrupted",
            {
                "action": "stop",
                "finished": False,
                "items": [{"target": {"username": "alpha"}, "status": "running"}],
            },
        )
        a = create_app(self.cfg, self.provider)
        self.addCleanup(a.extensions["batch_pool"].shutdown, True)
        self.assertEqual(self.store.job("interrupted")["items"][0]["status"], "unknown")
        self.assertEqual(self.provider.calls, [])


@unittest.skipIf(create_app is None, "Requires Web dependencies")
class ProviderContractTests(unittest.TestCase):
    def test_only_loopback_config_and_private_token_files_are_accepted(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp) / "fixture.jwt"
            p.write_text("fixture-only")
            p.chmod(0o644)
            cfg = {
                "api_url": "http://127.0.0.1:18000",
                "token_file": str(p),
                "public_key": "ssh-ed25519 AAAA fixture",
            }
            provider = Provider(cfg)
            with self.assertRaises(ProviderError):
                provider.boxes()
            for origin in (
                "http://evil.test",
                "http://127.0.0.1/path",
                "http://user@localhost",
                "file:///tmp/test",
            ):
                with self.assertRaises(ValueError):
                    Provider({**cfg, "api_url": origin})

    def test_explicit_keys_and_nested_response_field_allowlists(self):
        from unittest.mock import patch
        from webui.provider import public_box

        cleaned = public_box(
            {
                "username": "alpha",
                "token": "should not pass",
                "resources": {"cpu": "2", "secret": "should not pass"},
                "network": {"ipAddress": "10.0.0.2", "token": "should not pass"},
            }
        )
        self.assertNotIn("should not pass", json.dumps(cleaned))
        cfg = {
            "api_url": "http://127.0.0.1:18000",
            "token_file": "/unused",
            "public_key": "ssh-ed25519 AAAA fixture",
        }
        provider = Provider(cfg)
        with patch.object(provider, "request", return_value={}) as request:
            provider.execute(
                "create",
                {
                    "username": "alpha",
                    "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
                    "image": "images:ubuntu/24.04",
                },
            )
            args = request.call_args.args
            self.assertEqual(args[:2], ("POST", "/v1/containers"))
            self.assertEqual(args[2]["sshKeys"], [cfg["public_key"]])
            self.assertIs(args[2]["async"], True)
            provider.execute(
                "passthrough-delete", {"externalPort": 12345, "protocol": "UDP"}
            )
            self.assertEqual(
                request.call_args.args[1],
                "/v1/network/passthrough/12345?protocol=ROUTE_PROTOCOL_UDP",
            )


if __name__ == "__main__":
    unittest.main()
