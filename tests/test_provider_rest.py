
"""Source/fixture acceptance for app-owned Containarium REST lifecycle.

Tests use an in-memory fake transport; they are not live Provider acceptance.
"""
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
import urllib.error
from unittest.mock import patch

from bwmini import diagnostics
from bwmini.domain import Fault, Target
from bwmini.provider import Provider
from bwmini.provider_rest import RESTClient, trace_operation
from bwmini.store import Store


class Response:
    def __init__(self, value, status=200):
        self.status = status
        self.data = value if isinstance(value, bytes) else json.dumps(value).encode()
    def __enter__(self):
        return self
    def __exit__(self, exc_type, exc, tb):
        return False
    def read(self, size=-1):
        return self.data[:size]


class ProviderRESTTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.token = self.root / "private-token"
        self.set_token("a" * 72)
        self.config = {
            "url": "http://127.0.0.1:18080",
            "token_file": str(self.token),
        }
        self.rest = RESTClient(self.config)
        self.sent = []
        self.responses = []
        self.rest.opener.open = self.send

    def set_token(self, value):
        tmp = self.root / "new-token"
        tmp.write_text(value)
        tmp.chmod(0o400)
        os.replace(tmp, self.token)

    def send(self, request, timeout=None):
        self.sent.append((request.get_method(), request.full_url,
                          request.data, request.get_header("Authorization"), timeout))
        if not self.responses:
            raise AssertionError("unexpected network request")
        next_ = self.responses.pop(0)
        if isinstance(next_, Exception):
            raise next_
        return Response(next_)

    def queue(self, *values):
        self.responses.extend(values)

    @staticmethod
    def box(created="1738000000", state="CONTAINER_STATE_RUNNING"):
        return {
            "name": "work-container", "username": "work",
            "createdAt": created, "state": state,
            "resources": {"cpu": "2", "memory": "2GB", "disk": "20GB"},
            "podmanEnabled": False,
            "networkTenant": "tenant-a",
            "networkTenantSource": "NETWORK_TENANT_SOURCE_NATIVE",
            "network": {"ipAddress": "10.0.10.15"}
        }

    def provider(self, targets=None):
        return Provider({
            "provider": "containarium", "backend": "dev", "project": "default",
            "launcher": [sys.executable], "targets": targets or {},
            "provider_api": self.config,
        })

    def test_no_remote_endpoint_or_unsafe_file_config(self):
        for url in ("https://example.com", "http://localhost:18080",
                    "http://127.0.0.1", "http://127.0.0.1:18080/extra",
                    "http://127.0.0.1:18080@evil.test:8080",
                    "http://127.0.0.1:18080?token=secret", "http://[::1]:18080"):
            with self.subTest(url=url), self.assertRaises(Fault):
                RESTClient({"url": url, "token_file": str(self.token)})
        with self.assertRaises(Fault):
            RESTClient({"url": self.config["url"], "token_file": "relative"})
        self.token.chmod(0o600)
        with self.assertRaisesRegex(Fault, "PROVIDER_API_TOKEN_UNSAFE"):
            self.rest.inventory()

    def test_inventory_and_get_are_rest_only_and_match_exact_target(self):
        row = self.box()
        self.queue({"containers": [row], "totalCount": 1},
                   {"container": row}, {"container": row})
        p = self.provider({"work": Target(
            "containarium", "dev", "default", "work", "work-container",
            "1738000000", "work", "work-container").data()})
        # The mocked transport is on the Provider's own RESTClient.
        p.rest.opener.open = self.send
        with patch.object(p, "session", side_effect=AssertionError("MCP lifecycle used")):
            state = p.lifecycle_inspect("work")
            self.assertEqual(state["target"].created, "1738000000")
            self.assertEqual(state["network_tenant"], "tenant-a")
            self.assertEqual(p.observe("work").logical, "work")
        self.assertTrue(all(x[0] == "GET" for x in self.sent))
        self.assertTrue(all(x[1].startswith(self.config["url"] + "/v1/") for x in self.sent))

    def test_create_start_stop_resize_delete_use_exact_rest_contract(self):
        p = self.provider()
        p.rest.opener.open = self.send
        actions = [
            ("create", {"cpu": "2", "memory": "2GB", "disk": "20GB",
                        "ssh_public_key": "ssh-ed25519 AAAtest",
                        "_network_tenant": "tenant-a"}, "POST", "/v1/containers"),
            ("start", {}, "POST", "/v1/containers/work/start"),
            ("stop", {}, "POST", "/v1/containers/work/stop"),
            ("resize", {"_resize": {"cpu": "3"}}, "PUT", "/v1/containers/work/resize"),
            ("delete", {}, "DELETE", "/v1/containers/work")
        ]
        self.queue(*[{"message": "acknowledged"} for _ in actions])
        with patch("bwmini.provider.MCP", side_effect=AssertionError("MCP used")):
            for action, spec, method, path in actions:
                p.lifecycle_mutate(action, "work", spec, timeout=60)
                self.assertEqual(self.sent[-1][0:2], (method, self.config["url"] + path))
        create = json.loads(self.sent[0][2])
        self.assertEqual(create["resources"], {"cpu": "2", "memory": "2GB", "disk": "20GB"})
        self.assertEqual(create["sshKeys"], ["ssh-ed25519 AAAtest"])
        self.assertEqual(create["networkTenant"], "tenant-a")
        self.assertEqual(json.loads(self.sent[1][2]), {"waitForReady": False})
        self.assertEqual(json.loads(self.sent[2][2]), {"force": False})
        self.assertEqual(json.loads(self.sent[3][2]), {"cpu": "3"})
        self.assertIsNone(self.sent[4][2])
        self.assertTrue(all(x[3] == "Bearer " + "a" * 72 for x in self.sent))

    def test_rest_disk_shrink_and_unsupported_size_refused_before_put(self):
        for current, requested, expected in (
                ("25GB", "24GB", "RESOURCE_DISK_SHRINK_REFUSED"),
                ("unknown", "26GB", "RESOURCE_DISK_SIZE_UNVERIFIED"),
                ("25GB", "invalid", "RESOURCE_DISK_SIZE_UNVERIFIED")):
            with self.subTest(current=current, requested=requested):
                self.sent.clear()
                row = self.box()
                row["resources"]["disk"] = current
                self.queue({"container": row})
                with self.assertRaisesRegex(Fault, expected):
                    self.rest.mutate("resize", "work", {"_resize": {"disk": requested}}, timeout=30)
                self.assertEqual([x[0] for x in self.sent], ["GET"])
                self.assertTrue(self.sent[0][1].endswith("/v1/containers/work"))

    def test_rest_disk_equal_or_grow_requires_read_then_put(self):
        for disk in ("25GB", "26GB"):
            with self.subTest(disk=disk):
                self.sent.clear()
                self.queue({"container": self.box()}, {"message": "acknowledged"})
                self.rest.mutate("resize", "work", {"_resize": {"disk": disk}}, timeout=30)
                self.assertEqual([x[0] for x in self.sent], ["GET", "PUT"])
                self.assertEqual(json.loads(self.sent[1][2]), {"disk": disk})

    def test_token_rotation_and_unsupported_redirect(self):
        self.queue({}, {})
        self.rest.inventory()
        self.set_token("b" * 72)
        self.rest.inventory()
        self.assertNotEqual(self.sent[0][3], self.sent[1][3])
        self.assertEqual(self.sent[1][3], "Bearer " + "b" * 72)
        self.queue(urllib.error.HTTPError(
            "http://127.0.0.1:18080/v1/containers", 302, "redirect", {}, io.BytesIO(b"secret")))
        with self.assertRaisesRegex(Fault, "API_HTTP_REJECTED"):
            self.rest.inventory()

    def test_partial_inventory_is_unknown_not_empty(self):
        self.queue({"containers": [], "unreachableBackends": [{"id": "offline"}]})
        with self.assertRaisesRegex(Fault, "INVENTORY_CONTRACT_UNSUPPORTED"):
            self.rest.inventory()
        self.queue({"containers": [{"username": "work"}, {"username": "work"}]})
        with self.assertRaisesRegex(Fault, "INVENTORY_CONTRACT_UNSUPPORTED"):
            self.rest.inventory()
        self.queue({})
        self.assertEqual(self.rest.inventory(), set())

    def test_malformed_duplicate_json_and_transport_errors_fail_closed(self):
        self.queue(b'{"container":{"username":"work"},"container":{"username":"work"}}')
        with self.assertRaisesRegex(Fault, "PROVIDER_API_RESPONSE_INVALID"):
            self.rest.container("work")
        self.queue(urllib.error.HTTPError(
            self.config["url"] + "/v1/containers/work", 401, "secret", {},
            io.BytesIO(b'{"error":"sensitive"}')))
        with self.assertRaisesRegex(Fault, "API_AUTH_REJECTED"):
            self.rest.mutate("stop", "work", {}, timeout=30)
        self.queue(urllib.error.URLError("network secret"))
        with self.assertRaisesRegex(Fault, "API_OUTCOME_UNKNOWN"):
            self.rest.mutate("delete", "work", {}, timeout=30)
        self.queue(urllib.error.URLError("network secret"))
        with self.assertRaisesRegex(Fault, "API_READ_UNAVAILABLE"):
            self.rest.inventory()

    def test_mcp_special_nesting_not_silently_sent_to_rest(self):
        p = self.provider()
        p.rest.opener.open = self.send
        with patch("bwmini.provider.MCP") as fake:
            tool = fake.return_value.__enter__.return_value
            tool.discover.return_value = {
                "create_container": {"inputSchema": {"properties": {
                    "username": {}, "cpu": {}, "memory": {}, "disk": {},
                    "image": {}, "ssh_keys": {}, "enable_podman": {},
                    "enable_nesting": {"type": "boolean"}}}}}
            p.lifecycle_mutate("create", "work", {
                "cpu": "2", "memory": "2GB", "disk": "20GB",
                "ssh_public_key": "ssh-ed25519 AAAtest", "_nesting": True
            }, 30)
            tool.tool.assert_called_once()
            self.assertEqual(tool.tool.call_args.args[0], "create_container")
            self.assertEqual(tool.tool.call_args.args[1]["enable_nesting"], True)
        self.assertEqual(self.sent, [])

    def test_trace_correlates_without_persisting_secrets(self):
        self.rest.store = Store(self.root / "state.sqlite3")
        trace = diagnostics.new_trace_id()
        secret = "a" * 72
        self.queue({"message": "server-private-secret-" + secret})
        with trace_operation(trace, "op123"):
            self.rest.mutate("start", "work", {}, timeout=30)
        events = diagnostics.events_for_trace(self.rest.store, trace)
        self.assertTrue(any(e["component"] == "provider_api" and
                            e["code"] == "API_RESPONSE_RECEIVED" and
                            e["operation_id"] == "op123" for e in events))
        self.assertNotIn(secret, json.dumps(events))
        self.assertNotIn("server-private-secret", json.dumps(events))


if __name__ == "__main__":
    unittest.main()
