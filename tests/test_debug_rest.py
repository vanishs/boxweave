"""T evidence for selected debug_container REST read, standalone Gateway and Web."""
import io
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread
import os
import sys
from pathlib import Path
import tempfile
import unittest
from contextlib import redirect_stdout
from unittest.mock import patch

from bwmini import debug_rest, mcp_gate
from bwmini.box_debug import summary
from bwmini.domain import Fault
from bwmini.provider_rest import RESTClient
from bwmini.provider import Provider
from test_slice import TARGET


SENSITIVE = {
    "containerState": "running",
    "hostUserExists": True,
    "hostUserShellExists": True,
    "recentSshdRejections": ["private ssh journal 123"],
    "likelyCause": "private cause 456",
    "nextActions": ["private command 789"],
    "hostUserShell": "/bin/secret",
    "sourceRepo": "https://github.com/FootprintAI/Containarium",
    "daemonVersion": "0.128.0",
}


class Response:
    status = 200
    def __init__(self, data):
        self.data = data if isinstance(data, bytes) else json.dumps(data).encode()
    def __enter__(self): return self
    def __exit__(self, *_): return False
    def read(self, limit=-1): return self.data[:limit]


class Transport:
    def __init__(self, *outcomes):
        self.outcomes = list(outcomes)
        self.calls = []
    def open(self, request, timeout=None):
        self.calls.append((request.get_method(), request.full_url,
                           request.get_header("Authorization"), timeout))
        value = self.outcomes.pop(0)
        if isinstance(value, Exception): raise value
        return Response(value)


class DebugTransportTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.token_file = Path(temp.name) / "token"
        self.token_file.write_text("a" * 72)
        self.token_file.chmod(0o400)
        self.cfg = {"url": "http://127.0.0.1:18080", "token_file": str(self.token_file)}

    def test_exact_single_rest_get_revalidates_rotated_credential(self):
        transport = Transport(SENSITIVE, SENSITIVE)
        self.assertEqual(debug_rest.read(self.cfg, TARGET.logical, opener=transport), SENSITIVE)
        replacement = self.token_file.with_suffix(".new")
        replacement.write_text("b" * 72)
        replacement.chmod(0o400)
        os.replace(replacement, self.token_file)
        debug_rest.read(self.cfg, TARGET.logical, opener=transport)
        self.assertEqual(transport.calls[0][0], "GET")
        self.assertEqual(transport.calls[0][1],
                         self.cfg["url"] + "/v1/containers/" + TARGET.logical + "/debug")
        self.assertEqual(transport.calls[0][2], "Bearer " + "a" * 72)
        self.assertEqual(transport.calls[1][2], "Bearer " + "b" * 72)

    def test_real_loopback_http_get_authenticated_and_denial(self):
        received = []
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                received.append((self.path, self.headers.get("Authorization")))
                if self.headers.get("Authorization") != "Bearer " + "a" * 72:
                    self.send_response(401)
                    self.end_headers()
                    self.wfile.write(b"PRIVATE-ERROR")
                    return
                self.send_response(200)
                self.send_header("Content-Type","application/json")
                self.end_headers()
                self.wfile.write(json.dumps(SENSITIVE).encode())
            def log_message(self, *args):
                pass
        server = ThreadingHTTPServer(("127.0.0.1",0), Handler)
        worker = Thread(target=server.serve_forever, daemon=True)
        worker.start()
        try:
            self.cfg["url"] = "http://127.0.0.1:" + str(server.server_port)
            self.assertEqual(debug_rest.read(self.cfg,"work"), SENSITIVE)
            self.assertEqual(received[-1],("/v1/containers/work/debug", "Bearer " + "a" * 72))
            replacement = self.token_file.with_suffix(".new")
            replacement.write_text("b" * 72)
            replacement.chmod(0o400)
            os.replace(replacement,self.token_file)
            with self.assertRaisesRegex(debug_rest.DebugRESTError, "DEBUG_REST_AUTH_REJECTED"):
                debug_rest.read(self.cfg,"work")
        finally:
            server.shutdown()
            worker.join(3)
            server.server_close()

    def test_request_refuses_redirects_proxies_path_traversal_bad_token(self):
        for name in ("..", "x/../y", "containarium-core-/etc", "x?token=1", "", "X"):
            with self.subTest(name=name), self.assertRaises(debug_rest.DebugRESTError):
                debug_rest.route(name)
        for url in ("https://host", "http://127.0.0.1:18080/path",
                    "http://localhost:18080", "http://127.0.0.1:18080@evil:18080"):
            with self.subTest(url=url), self.assertRaises(debug_rest.DebugRESTError):
                debug_rest.read({**self.cfg, "url":url}, "work", opener=Transport(SENSITIVE))
        self.token_file.chmod(0o644)
        with self.assertRaisesRegex(debug_rest.DebugRESTError, "DEBUG_REST_TOKEN_UNSAFE"):
            debug_rest.read(self.cfg, "work", opener=Transport(SENSITIVE))
        self.assertEqual(Transport().calls, [])

    def test_invalid_duplicate_oversize_and_type_are_rejected(self):
        for data in (b'{"containerState":"running","containerState":"missing"}',
                     {"containerState":"running","hostUserExists":"true"},
                     {"containerState":"unknown"},
                     {"containerState":"running","recentSshdRejections":[5]},
                     {"containerState":"running","nextActions":["x"*33000]}):
            with self.subTest(data=str(data)[:40]), self.assertRaises(debug_rest.DebugRESTError):
                debug_rest.read(self.cfg, "work", opener=Transport(data))

    def test_app_rest_and_web_projection_no_mcp(self):
        p = type("P", (), {"mode":"mcp", "rest":RESTClient(self.cfg)})()
        p.rest.opener.open = Transport(SENSITIVE, {"container": {
            "username": TARGET.logical, "name": TARGET.instance, "createdAt": TARGET.created}}).open
        p.session = lambda: self.fail("MCP fallback used")
        result = summary(p, TARGET)
        self.assertEqual(result["target"], TARGET.logical)
        self.assertEqual(result["host_user_exists"], True)
        self.assertNotIn("private ssh", json.dumps(result))
        self.assertNotIn("private cause", json.dumps(result))
        self.assertNotIn("/bin/secret", json.dumps(result))
        self.assertEqual(result["sensitive_details"], "withheld")
        self.assertFalse(result["boxweave_host_ssh_verified"])

    def test_app_recreated_box_is_refused_after_rest_debug(self):
        p = type("P", (), {"mode":"mcp", "rest":RESTClient(self.cfg)})()
        p.rest.opener.open = Transport(SENSITIVE, {"container": {
            "username": TARGET.logical, "name": TARGET.instance, "createdAt": "new-incarnation"}}).open
        with self.assertRaisesRegex(Fault, "TARGET_CHANGED"):
            summary(p, TARGET)

    def test_app_debug_only_rollback_does_not_disable_lifecycle_rest(self):
        cfg = {"provider":"containarium", "backend":"dev", "project":"default",
               "launcher":[sys.executable], "targets":{},
               "provider_api":self.cfg, "debug_rest_enabled":False}
        p = Provider(cfg)
        self.assertIsNotNone(p.rest)
        self.assertFalse(p.debug_rest_enabled)
        cfg["debug_rest_enabled"] = "false"
        with self.assertRaisesRegex(Fault, "DEBUG_REST_CONFIG_INVALID"):
            Provider(cfg)

    def test_app_failure_does_not_fallback_to_mcp(self):
        p = type("P", (), {"mode":"mcp", "rest":RESTClient(self.cfg)})()
        p.rest.opener.open = Transport({"containerState":"running",
                                        "hostUserExists":"bad"}).open
        p.session = lambda: self.fail("MCP fallback used")
        with self.assertRaises(Fault):
            summary(p,TARGET)

    def test_root_policy_disabled_missing_and_unsafe_fail_closed(self):
        self.assertIsNone(debug_rest.gateway_config())
        with tempfile.TemporaryDirectory() as tmp:
            policy = Path(tmp)/"debug-rest.json"
            policy.write_text(json.dumps({"version":1,"url":self.cfg["url"],
                "token_file":self.cfg["token_file"],"enabled":True}))
            policy.chmod(0o644)
            with self.assertRaisesRegex(debug_rest.DebugRESTError,"DEBUG_REST_POLICY_UNSAFE"):
                debug_rest.gateway_config(policy)
            class Meta:
                def __init__(self, original):
                    self.st_mode=original.st_mode
                    self.st_uid=0
                    self.st_nlink=original.st_nlink
                    self.st_size=original.st_size
                    self.st_ino=original.st_ino
                    self.st_dev=original.st_dev
            original = Path.lstat
            def as_root(path):
                item = original(path)
                return Meta(item) if path in (policy, policy.parent) else item
            with patch.object(Path,"lstat",as_root):
                self.assertEqual(debug_rest.gateway_config(policy), self.cfg)
                policy.write_text('{"version":1,"enabled":true,"url":"http://evil","token_file":"x"}')
                with self.assertRaises(debug_rest.DebugRESTError):
                    debug_rest.gateway_config(policy)


class FakeProvider:
    def __init__(self):
        self.stdin=self
        self.stdout=self
        self.sent=[]
        self.returns=[]
    def write(self, raw):
        req=json.loads(raw)
        self.sent.append(req)
        if req.get("method") == "tools/list":
            msg={"result":{"tools":[
                {"name":"debug_container","inputSchema":{"type":"object",
                  "properties":{"username":{"type":"string"}}}},
                {"name":"get_container","inputSchema":{"type":"object","properties":{}}},
            ]}}
        else:
            msg={"result":{"content":[{"type":"text","text":"legacy"}]}}
        self.returns.append((json.dumps({"jsonrpc":"2.0","id":req["id"],**msg})+"\n").encode())
    def flush(self): pass
    def readline(self,*_): return self.returns.pop(0) if self.returns else b""
    def poll(self): return 0
    def close(self): pass


class GatewayDebugTests(unittest.TestCase):
    def run_gateway(self, requests, cfg, protected=False, permitted=None, response=SENSITIVE):
        provider = FakeProvider()
        incoming=io.StringIO("".join(json.dumps(req)+"\n" for req in requests))
        out=io.StringIO()
        with (patch.object(mcp_gate.subprocess, "Popen",return_value=provider),
              patch.object(mcp_gate.sys,"stdin",incoming),
              patch.object(mcp_gate,"tool_policy",return_value=permitted),
              patch.object(mcp_gate.gateway_protection,"enabled",return_value=protected),
              patch.object(mcp_gate.debug_rest,"gateway_config",return_value=cfg),
              patch.object(mcp_gate.debug_rest,"read",return_value=response) as rest,
              patch.object(mcp_gate.gd,"record") as recorded,
              redirect_stdout(out)):
            mcp_gate.main()
        return provider,[json.loads(l) for l in out.getvalue().splitlines()],rest,recorded

    def test_gateway_selected_rest_no_mcp_dispatch_and_no_raw_log(self):
        call={"jsonrpc":"2.0","id":7,"method":"tools/call",
              "params":{"name":"debug_container","arguments":{"username":"work"}}}
        for protected in (False,True):
            provider,out,rest,recorded=self.run_gateway([call],{"url":"http://127.0.0.1:18080",
                "token_file":"/tmp/safe"},protected=protected,
                permitted=frozenset({"debug_container"}))
            self.assertEqual(provider.sent,[])
            self.assertEqual(json.loads(out[0]["result"]["content"][0]["text"]), SENSITIVE)
            rest.assert_called_once()
            self.assertNotIn("private ssh",str(recorded.call_args))
            self.assertNotIn("private cause",str(recorded.call_args))

    def test_not_selected_keeps_legacy_and_protected_refuses(self):
        call={"jsonrpc":"2.0","id":7,"method":"tools/call",
              "params":{"name":"debug_container","arguments":{"username":"work"}}}
        provider,out,_,_=self.run_gateway([call],None)
        self.assertEqual(len(provider.sent),1)
        self.assertEqual(out[0]["result"]["content"][0]["text"],"legacy")
        provider,out,_,_=self.run_gateway([call],None,protected=True)
        self.assertEqual(provider.sent,[])
        self.assertEqual(out[0]["error"]["message"],"COORDINATION_TOOL_UNCLASSIFIED")

    def test_host_allowlist_denies_before_rest_and_catalog_respects_policy(self):
        call={"jsonrpc":"2.0","id":7,"method":"tools/call",
              "params":{"name":"debug_container","arguments":{"username":"work"}}}
        provider,out,rest,_=self.run_gateway([call],{"url":"http://127.0.0.1:18080",
            "token_file":"/tmp/safe"},protected=True,permitted=frozenset({"get_container"}))
        self.assertEqual(out[0]["error"]["message"],"GATE_TOOL_NOT_ALLOWED")
        self.assertEqual(provider.sent,[])
        rest.assert_not_called()
        listing={"jsonrpc":"2.0","id":9,"method":"tools/list"}
        _,out,_,_=self.run_gateway([listing],{"url":"http://127.0.0.1:18080",
            "token_file":"/tmp/safe"},protected=True)
        self.assertIn("debug_container",[row["name"] for row in out[0]["result"]["tools"]])
        _,out,_,_=self.run_gateway([listing],None,protected=True)
        self.assertNotIn("debug_container",[row["name"] for row in out[0]["result"]["tools"]])


if __name__ == "__main__":
    unittest.main()
