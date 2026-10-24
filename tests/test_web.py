"""Independent web security, graph persistence and batch semantics (fake provider)."""

import json
import io
import threading
from contextlib import contextmanager
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.error import HTTPError
from unittest.mock import patch
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


def upstream_missing_box(username):
    return {
        "code": 500,
        "error": (
            "rpc error: code = Unknown desc = failed to get container: "
            "failed to get container: Failed to fetch instance "
            f'"{username}-container" in project "default": Instance not found'
        ),
    }


@contextmanager
def http_provider_fixture(directory):
    """Real HTTP adapter round-trips; all resources and authorization are fake."""
    calls = []

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *args):
            pass

        def reply(self, status, data):
            encoded = json.dumps(data).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)

        def do_GET(self):
            calls.append(("GET", self.path, None))
            if self.path == "/v1/containers":
                self.reply(200, {"containers": []})
            else:
                self.reply(500, upstream_missing_box(self.path.rsplit("/", 1)[-1]))

        def do_POST(self):
            data = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
            calls.append(("POST", self.path, data))
            self.reply(
                200,
                {
                    "container": {
                        "username": data["username"],
                        "state": "CONTAINER_STATE_CREATING",
                    }
                },
            )

    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    token = Path(directory) / "http-fixture.jwt"
    token.write_text("synthetic-fixture-only")
    token.chmod(0o600)
    provider = Provider(
        {
            "api_url": f"http://127.0.0.1:{server.server_port}",
            "token_file": str(token),
            "public_key": "ssh-ed25519 AAAA fixture",
        }
    )
    thread.start()
    try:
        yield provider, calls
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)


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
                "network": {"ipAddress": "10.0.0.2"},
            }
        return {}

    def network_guard(self):
        import hashlib
        from datetime import datetime, timezone

        return {
            "core": {},
            "tenant": {
                "mode": "GUARD_MODE_ENFORCE",
                "firewallDriver": "nftables",
                "unsupported": False,
                "lastPass": datetime.now(timezone.utc).isoformat(),
                "entries": [
                    {
                        "container": c.get("name", n + "-container"),
                        "subject": n,
                        "ip": c.get("network", {}).get("ipAddress", "10.0.0.2"),
                        "aclName": "containarium-tenant-"
                        + hashlib.sha256(n.encode()).hexdigest()[:12],
                        "attached": True,
                    }
                    for n, c in self.data.items()
                ],
            },
        }

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

    def test_real_http_missing_instance_preflight_allows_one_create(self):
        with http_provider_fixture(self.temp.name) as (provider, calls):
            self.app = create_app(self.cfg, provider)
            self.addCleanup(self.app.extensions["batch_pool"].shutdown, True)
            self.c = self.app.test_client()
            self.store = self.app.extensions["store"]
            self.login()
            item = {
                "username": "boxweave-web-test",
                "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
                "image": "images:ubuntu/24.04",
            }
            job = self.batch("create", [item])
            self.assertEqual(job["items"][0]["status"], "submitted")
            writes = [c for c in calls if c[0] == "POST"]
            self.assertEqual(len(writes), 1)
            self.assertEqual(writes[0][1], "/v1/containers")
            self.assertEqual(writes[0][2]["username"], item["username"])
            self.assertEqual(writes[0][2]["sshKeys"], [provider.key])

    def test_unrelated_lookup_failure_never_reaches_create(self):
        self.login()
        with patch.object(self.provider, "box", side_effect=ProviderError(502)):
            job = self.batch(
                "create",
                [
                    {
                        "username": "alpha",
                        "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
                    }
                ],
            )
        self.assertEqual(job["items"][0]["status"], "failed")
        self.assertEqual(self.provider.calls, [])

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

    def test_exact_missing_instance_response_is_normalized_only_for_lookup(self):
        with tempfile.TemporaryDirectory() as temp:
            with http_provider_fixture(temp) as (provider, calls):
                with self.assertRaises(ProviderError) as caught:
                    provider.box("boxweave-web-test")
                self.assertEqual(caught.exception.status, 404)
                self.assertFalse(caught.exception.unknown)
                self.assertEqual([c[0] for c in calls], ["GET"])

    def test_other_errors_cannot_be_mistaken_for_missing_boxes(self):
        cfg = {
            "api_url": "http://127.0.0.1:18000",
            "token_file": "/unused",
            "public_key": "ssh-ed25519 AAAA fixture",
        }
        provider = Provider(cfg)
        cases = [
            (
                "GET",
                "/v1/containers/alpha",
                500,
                {"code": 500, "error": "Permission denied; fixture-sensitive-error"},
            ),
            ("GET", "/v1/containers/alpha", 500, upstream_missing_box("beta")),
            (
                "GET",
                "/v1/containers/alpha",
                500,
                {
                    "code": 500,
                    "error": upstream_missing_box("alpha")["error"].replace(
                        'project "default"', 'project "other"'
                    ),
                },
            ),
            ("GET", "/v1/containers/alpha", 500, b"not JSON"),
            ("GET", "/v1/containers/alpha", 500, b"x" * 8193),
            ("GET", "/v1/containers", 500, upstream_missing_box("alpha")),
            ("POST", "/v1/containers", 500, upstream_missing_box("alpha")),
            ("DELETE", "/v1/containers/alpha", 500, upstream_missing_box("alpha")),
            ("GET", "/v1/containers/alpha", 403, upstream_missing_box("alpha")),
            ("GET", "/v1/containers/alpha", 404, {"error": "not found"}),
        ]
        for method, path, status, data in cases:
            with self.subTest(method=method, path=path, status=status):
                raw = data if isinstance(data, bytes) else json.dumps(data).encode()
                error = HTTPError(
                    provider.origin + path, status, "fixture", {}, io.BytesIO(raw)
                )
                with patch("webui.provider.Path.lstat") as stat_call:
                    import stat

                    stat_call.return_value.st_mode = stat.S_IFREG | 0o600
                    with patch(
                        "webui.provider.Path.read_text",
                        return_value="synthetic-fixture-only",
                    ), patch.object(provider.opener, "open", side_effect=error):
                        with self.assertRaises(ProviderError) as caught:
                            provider.request(
                                method, path, {} if method == "POST" else None
                            )
                self.assertEqual(
                    caught.exception.status, status if status in (403, 404) else 502
                )
                self.assertEqual(
                    caught.exception.unknown, method != "GET" and status >= 500
                )
                self.assertNotIn("fixture-sensitive-error", str(caught.exception))
                self.assertTrue(error.fp.closed)

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
