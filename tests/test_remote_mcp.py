"""Remote MCP protocol and authentication fixture tests (no Box/SSH/network)."""
import base64
import hashlib
from pathlib import Path
from unittest.mock import patch
import re
import tempfile
import unittest
from urllib.parse import parse_qs, urlsplit

from werkzeug.security import generate_password_hash

from bwmini.remote_auth import RemoteCredentials, RemoteError, valid_url
from bwmini.remote_mcp import create_remote_app, relay, validate_remote_config
from bwmini.remote_tls import caddyfile
from bwmini.store import Store

ORIGIN = "https://mcp.example.test"
RESOURCE = ORIGIN + "/mcp"
ACCEPT = "application/json, text/event-stream"
REQ = {"jsonrpc": "2.0", "id": 7, "method": "tools/list", "params": {}}


class RemoteTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.store = Store(Path(self.tmp.name) / "state" / "db.sqlite3")
        self.store.admin(generate_password_hash("fixture-admin-password"))
        self.now = [1000.]
        self.calls = []
        def dispatch(request):
            self.calls.append(request)
            return {"jsonrpc": "2.0", "id": request["id"],
                    "result": {"tools": [{"name": "list_containers"}]}}
        self.dispatch = dispatch
        self.auth = RemoteCredentials(self.store, RESOURCE, lambda: self.now[0])
        self.app = create_remote_app(
            self.store, {"enabled": True, "public_url": ORIGIN, "port": 18740},
            dispatch=dispatch, clock=lambda: self.now[0])
        self.app.testing = True
        self.client = self.app.test_client()
        self.headers = {"Accept": ACCEPT, "Content-Type": "application/json",
                        "MCP-Protocol-Version": "2025-11-25"}

    def tearDown(self):
        self.tmp.cleanup()

    def call(self, token=None, path="/mcp", payload=None, headers=None):
        h = dict(self.headers)
        if token:
            h["Authorization"] = "Bearer " + token
        if headers:
            h.update(headers)
        return self.client.post(path, json=payload or REQ, headers=h)

    def test_schema_migration_to_v12(self):
        with self.store.db() as db:
            self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], 12)
        self.assertIsNotNone(Store(self.store.path))

    def test_metadata_and_no_anonymous_tools(self):
        data = self.client.get("/.well-known/oauth-protected-resource/mcp").json
        self.assertEqual(data["resource"], RESOURCE)
        self.assertEqual(data["authorization_servers"], [ORIGIN])
        self.assertEqual(self.call().status_code, 401)
        self.assertIn("resource_metadata", self.call().headers["WWW-Authenticate"])
        self.assertEqual(self.calls, [])

    def test_api_bearer_stateless_init_and_call(self):
        token = self.auth.issue("api", "unit fixture")
        result = self.call(token)
        self.assertEqual(result.status_code, 200)
        self.assertEqual(result.json["result"]["tools"][0]["name"], "list_containers")
        self.assertEqual(self.calls, [REQ])
        init = self.call(token, payload={"jsonrpc": "2.0", "id": 1,
                                         "method": "initialize",
                                         "params": {"protocolVersion": "2025-11-25"}})
        self.assertEqual(init.json["result"]["protocolVersion"], "2025-11-25")
        self.assertEqual(init.json["result"]["serverInfo"]["name"], "boxweave")
        self.assertIn("isolated Linux Boxes", init.json["result"]["serverInfo"]["description"])
        self.assertIn("boxweave_info", init.json["result"]["instructions"])
        self.assertEqual(self.call(token, payload={"jsonrpc": "2.0",
                         "method": "notifications/initialized"}).status_code, 202)
        self.assertEqual(len(self.calls), 1)

    def test_bad_headers_origin_and_version_fail_before_dispatch(self):
        token = self.auth.issue("api", "fixture")
        self.assertEqual(self.call(token, headers={"Origin": "https://evil.test"}).status_code, 403)
        self.assertEqual(self.call(token, headers={"Host": "evil.example.test"}).status_code, 403)
        self.assertEqual(self.call(token, headers={"Accept": "application/json"}).status_code, 406)
        self.assertEqual(self.call(token, headers={"MCP-Protocol-Version": "1900"}).status_code, 400)
        self.assertEqual(self.client.get("/mcp", headers={"Authorization": "Bearer " + token}).status_code, 405)
        self.assertEqual(self.calls, [])

    def test_path_credential_is_not_anonymous_or_query_token(self):
        token = self.auth.issue("link", "one field")
        path = "/mcp/link/" + token
        self.assertEqual(self.call(path=path).status_code, 200)
        self.assertEqual(self.call(path="/mcp?token=" + token).status_code, 401)
        ident = [v for v in self.auth.list() if v["kind"] == "link"][0]["id"]
        self.auth.revoke(ident)
        self.assertEqual(self.call(path=path).status_code, 401)
        self.assertNotIn(token, str(self.auth.list()))

    def test_revoked_and_expired_access(self):
        token = self.auth.issue("api", "revoke")
        ident = self.auth.list()[0]["id"]
        self.assertEqual(self.call(token).status_code, 200)
        self.auth.revoke(ident)
        self.assertEqual(self.call(token).status_code, 401)
        access = self.auth.issue("access", "OAuth test", "client", RESOURCE, self.now[0] + 1)
        self.assertEqual(self.call(access).status_code, 200)
        self.now[0] += 2
        self.assertEqual(self.call(access).status_code, 401)

    def test_denied_authorization_attempts_commit_before_failure(self):
        # The rate ledger must survive failed password checks; a rolled-back
        # SQLite transaction would let repeated denials evade the 5/min gate.
        callback = "https://chatgpt.example.test/callback"
        client_id = self.auth.register("Rate ledger fixture", [callback])
        pending = self.auth.pending(client_id, callback, "A" * 43,
                                    RESOURCE, "rate-fixture-state")
        form = {"pending": pending, "password": "invalid-fixture-input",
                "approve": "yes"}
        for _ in range(5):
            self.assertEqual(self.client.post("/authorize", data=form,
                                              headers={"Origin": ORIGIN}).status_code, 403)
        with self.store.db() as db:
            self.assertEqual(db.execute(
                "SELECT count(*) FROM remote_attempts WHERE category='admin_authorize'"
            ).fetchone()[0], 5)
            self.assertEqual(db.execute("SELECT count(*) FROM remote_codes").fetchone()[0], 0)
        self.assertEqual(self.client.post("/authorize", data=form,
                                          headers={"Origin": ORIGIN}).status_code, 429)
        # A denied attempt must not consume a still-valid pending grant.
        self.now[0] += 61
        form["password"] = "fixture-admin-password"
        self.assertEqual(self.client.post("/authorize", data=form,
                                          headers={"Origin": ORIGIN}).status_code, 302)

    def test_dcr_authorization_pkce_refresh_and_reset(self):
        registered = self.client.post("/register", json={
            "client_name": "Realistic test",
            "redirect_uris": ["https://chatgpt.example.test/callback"],
            "token_endpoint_auth_method": "none"})
        self.assertEqual(registered.status_code, 201)
        client_id = registered.json["client_id"]
        verifier = "correct-test-pkce-verifier-" + "x" * 38
        challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
        params = {"client_id": client_id,
                  "redirect_uri": "https://chatgpt.example.test/callback",
                  "response_type": "code", "code_challenge_method": "S256",
                  "code_challenge": challenge, "state": "random-state-123",
                  "scope": "mcp:operator", "resource": RESOURCE}
        page = self.client.get("/authorize", query_string=params)
        self.assertEqual(page.status_code, 200)
        # Browser form submits to this origin then follows a 302 to the
        # registered OAuth callback. Without both headers Chrome either
        # sends Origin: null or blocks the cross-origin callback via CSP.
        self.assertEqual(page.headers["Referrer-Policy"], "same-origin")
        self.assertIn("form-action 'self' https://chatgpt.example.test;",
                      page.headers["Content-Security-Policy"])
        self.assertNotIn("chatgpt.example.test/callback",
                         page.headers["Content-Security-Policy"])
        self.assertIn("frame-ancestors 'none'",
                      page.headers["Content-Security-Policy"])
        pending = re.search('name="pending" value="([^"]+)"', page.get_data(as_text=True)).group(1)
        form = {"pending": pending, "password": "fixture-admin-password", "approve": "yes"}
        self.assertEqual(self.client.post("/authorize", data=form).status_code, 403)
        # The browser fix must not authorize opaque/null Origin or replay.
        self.assertEqual(self.client.post("/authorize", data=form,
                         headers={"Origin": "null"}).status_code, 403)
        self.assertEqual(self.client.post("/authorize", data=form,
                         headers={"Origin": "https://evil.example.test"}).status_code, 403)
        approved = self.client.post("/authorize", headers={"Origin": ORIGIN}, data=form)
        self.assertEqual(approved.status_code, 302)
        self.assertEqual(self.client.post("/authorize", headers={"Origin": ORIGIN},
                         data=form).status_code, 403)
        self.assertTrue(approved.headers["Location"].startswith(params["redirect_uri"]))
        code = parse_qs(urlsplit(approved.headers["Location"]).query)["code"][0]
        form = {"client_id": client_id, "redirect_uri": params["redirect_uri"],
                "code": code, "resource": RESOURCE, "grant_type": "authorization_code",
                "code_verifier": verifier}
        bad = dict(form, code_verifier="wrong-verifier-" + "y" * 35)
        self.assertEqual(self.client.post("/token", data=bad).status_code, 400)
        result = self.client.post("/token", data=form)
        self.assertEqual(result.status_code, 200)
        self.assertEqual(self.call(result.json["access_token"]).status_code, 200)
        self.assertEqual(self.client.post("/token", data=form).status_code, 400)
        refresh = self.client.post("/token", data={
            "grant_type": "refresh_token", "refresh_token": result.json["refresh_token"],
            "client_id": client_id, "resource": RESOURCE})
        self.assertEqual(refresh.status_code, 200)
        self.assertEqual(self.client.post("/token", data={
            "grant_type": "refresh_token", "refresh_token": result.json["refresh_token"],
            "client_id": client_id, "resource": RESOURCE}).status_code, 400)
        self.store.reset_admin(generate_password_hash("new-password-after-reset"))
        self.assertEqual(self.call(result.json["access_token"]).status_code, 401)
        self.assertEqual(self.call(refresh.json["access_token"]).status_code, 401)

    def test_authorization_headers_never_expand_for_rejected_redirect(self):
        metadata = self.client.get("/.well-known/oauth-authorization-server")
        self.assertEqual(metadata.headers["Referrer-Policy"], "no-referrer")
        self.assertIn("form-action 'self';", metadata.headers["Content-Security-Policy"])
        registered = self.client.post("/register", json={
            "client_name": "Browser CSP test",
            "redirect_uris": ["https://chatgpt.example.test/callback"]})
        self.assertEqual(registered.status_code, 201)
        params = {"client_id": registered.json["client_id"],
                  "redirect_uri": "https://unregistered.example.test/callback",
                  "response_type": "code", "code_challenge_method": "S256",
                  "code_challenge": "A" * 43, "state": "valid-state-1",
                  "scope": "mcp:operator", "resource": RESOURCE}
        rejected = self.client.get("/authorize", query_string=params)
        self.assertEqual(rejected.status_code, 400)
        self.assertEqual(rejected.headers["Referrer-Policy"], "no-referrer")
        self.assertIn("form-action 'self';",
                      rejected.headers["Content-Security-Policy"])
        self.assertNotIn("unregistered.example.test",
                         rejected.headers["Content-Security-Policy"])

    def test_redirect_dcr_and_bad_resource_rejected(self):
        for redirect in ("http://attacker.example/cb", "http://169.254.169.254/cb",
                         "https://foo.example/cb#fragment", "https://u:p@foo.example/cb"):
            self.assertFalse(valid_url(redirect))
        result = self.client.post("/register", json={
            "client_name": "unsafe", "redirect_uris": ["http://attacker.example/cb"]})
        self.assertEqual(result.status_code, 400)
        self.assertEqual(self.client.get("/authorize", query_string={
            "response_type": "code", "code_challenge_method": "S256",
            "client_id": "unregistered", "redirect_uri": "https://foo.example/cb",
            "code_challenge": "A" * 43, "state": "valid-state-1",
            "resource": RESOURCE}).status_code, 400)

    def test_http_validation(self):
        with self.assertRaises(ValueError):
            validate_remote_config({"enabled": True, "public_url": "http://mcp.test", "port": 18740})
        with self.assertRaises(ValueError):
            validate_remote_config({"enabled": True, "public_url": ORIGIN,
                                    "port": 18740, "host": "0.0.0.0"})

    def test_relay_spawns_gateway_with_importable_script_directory(self):
        # A real gateway process must import its sibling modules under the
        # exact spawn argv relay() uses; isolated mode (-I) once stripped the
        # script directory from sys.path and made every relay return 503.
        folder = Path(self.tmp.name) / "gatehome"
        folder.mkdir()
        (folder / "nesting.py").write_text("MARKER = 'fixture-sibling'\n")
        gate = folder / "gate.py"
        gate.write_text(
            "import json, sys\n"
            "from nesting import MARKER\n"
            "for line in sys.stdin:\n"
            "    request = json.loads(line)\n"
            "    if 'id' not in request:\n"
            "        continue\n"
            "    print(json.dumps({'jsonrpc': '2.0', 'id': request['id'],\n"
            "                      'result': {'marker': MARKER}}), flush=True)\n")
        reply = relay({"jsonrpc": "2.0", "id": 3, "method": "tools/list", "params": {}},
                      str(gate))
        self.assertEqual(reply["result"]["marker"], "fixture-sibling")

    def test_caddyfile_requires_installed_caddy_and_canonical_host(self):
        remote = {"enabled": True, "public_url": ORIGIN, "port": 18740}
        with self.assertRaises(ValueError):
            caddyfile(remote, caddy=str(Path(self.tmp.name) / "missing-caddy"))
        with self.assertRaises(ValueError):
            caddyfile({"enabled": True, "public_url": "https://localhost", "port": 18740},
                      caddy=str(Path(self.tmp.name) / "missing-caddy"))
        fake = Path(self.tmp.name) / "caddy"
        fake.write_text("#!/bin/sh\n")
        fake.chmod(0o755)
        config = caddyfile(remote, caddy=str(fake))
        self.assertIn("mcp.example.test {\n", config)
        self.assertIn("reverse_proxy 127.0.0.1:18740", config)
        self.assertNotIn("tls ", config)
        # A globally overwritten Referrer-Policy causes Origin: null in
        # ordinary browsers on the OAuth password form; upstream headers win.
        self.assertNotIn("header Referrer-Policy", config)
        self.assertIn("output discard", config)


class RemoteWebTests(unittest.TestCase):
    def setUp(self):
        from bwmini.domain import Target
        from bwmini.provider import Fixture
        from bwmini.service import Service
        from bwmini.web import create_app
        self.tmp = tempfile.TemporaryDirectory()
        store = Store(Path(self.tmp.name) / "state" / "db.sqlite3")
        store.admin(generate_password_hash("web-fixture-password"))
        target = Target("containarium", "dev", "default", "fixture-box",
                        "fixture-box-container", "123", "fixture-box",
                        "fixture-box-container")
        service = Service(store, Fixture(target))
        app = create_app(service, "http://127.0.0.1:18640", self.tmp.name,
                         secure=False, remote_config={"enabled": True,
                         "public_url": ORIGIN, "port": 18844})
        app.testing = True
        self.client = app.test_client()
        self.assertEqual(self.client.post("/api/login",
            json={"password": "web-fixture-password"},
            headers={"Origin": "http://127.0.0.1:18640"}).status_code, 200)
        self.csrf = self.client.get("/api/session").json["csrf"]
        self.headers = {"Origin": "http://127.0.0.1:18640",
                        "X-CSRF-Token": self.csrf}

    def tearDown(self):
        self.tmp.cleanup()

    def test_web_issue_revoke_and_secret_not_listed(self):
        response = self.client.post("/api/remote-mcp/credentials",
            json={"kind": "link", "label": "test coder"}, headers=self.headers)
        self.assertEqual(response.status_code, 201)
        self.assertIn("/mcp/link/", response.json["connection_url"])
        token = response.json["secret"]
        listing = self.client.get("/api/remote-mcp").json
        self.assertTrue(listing["enabled"])
        self.assertNotIn(token, str(listing))
        ident = listing["credentials"][0]["id"]
        result = self.client.delete("/api/remote-mcp/credentials/" + ident,
                                    json={}, headers=self.headers)
        self.assertEqual(result.status_code, 200)
        self.assertEqual(self.client.get("/api/remote-mcp").json["credentials"][0]["revoked"], 1)

    def test_web_remote_listener_tcp_probe_is_bounded_and_not_public_proof(self):
        # Probe is loopback-only and never reaches public_url or a Box.
        with patch("socket.create_connection") as connect:
            local = self.client.get("/api/remote-mcp")
            self.assertEqual(local.status_code, 200)
            self.assertTrue(local.json["local_tcp_reachable"])
            self.assertFalse(local.json["externally_verified"])
            connect.assert_called_once_with(("127.0.0.1", 18844), timeout=0.2)
        with patch("socket.create_connection", side_effect=OSError("port closed")):
            down = self.client.get("/api/remote-mcp")
            self.assertEqual(down.status_code, 200)
            self.assertFalse(down.json["local_tcp_reachable"])
            self.assertFalse(down.json["externally_verified"])
            self.assertEqual(down.json["mcp_url"], RESOURCE)

    def test_web_token_cannot_be_created_without_csrf(self):
        result = self.client.post("/api/remote-mcp/credentials",
            json={"kind": "api", "label": "test coder"},
            headers={"Origin": "http://127.0.0.1:18640"})
        self.assertEqual(result.status_code, 403)


if __name__ == "__main__":
    unittest.main()
