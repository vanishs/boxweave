"""Contract tests for shared REST route mapping, transport and async lifecycle."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from bwmini import rest_routes, rest_wire
from bwmini.provider_rest import RESTClient
from bwmini.domain import Fault


class Reply:
    status = 200
    def __init__(self, obj):
        self.obj = obj
    def __enter__(self):
        return self
    def __exit__(self, *a):
        return False
    def read(self, _size=-1):
        return json.dumps(self.obj).encode()


class RouteContractTests(unittest.TestCase):
    def test_exact_eight_routes_and_no_dynamic_rest_gateway(self):
        name = "bwracc-demo"
        params = {"username": name}
        cases = {
            "list_containers": ({}, "GET", "/v1/containers"),
            "get_container": (params, "GET", "/v1/containers/" + name),
            "debug_container": (params, "GET", "/v1/containers/" + name + "/debug"),
            "start_container": (params, "POST", "/v1/containers/" + name + "/start"),
            "stop_container": (params, "POST", "/v1/containers/" + name + "/stop"),
            "delete_container": (params, "DELETE", "/v1/containers/" + name),
            "resize_container": ({**params, "cpu": "2"}, "PUT",
                                 "/v1/containers/" + name + "/resize"),
            "create_container": ({**params, "ssh_keys": ["ssh-ed25519 AAA"],
                                  "cpu": "1"}, "POST", "/v1/containers"),
        }
        self.assertEqual(set(cases), set(rest_routes._NAMES))
        for tool, (args, method, path) in cases.items():
            with self.subTest(tool=tool):
                route_method, route_path, body = rest_routes.route(tool, args)
                self.assertEqual((route_method, route_path), (method, path))
                if tool == "create_container":
                    self.assertIs(body["async"], True)
                    self.assertEqual(body["sshKeys"], ["ssh-ed25519 AAA"])
                    self.assertEqual(body["resources"]["cpu"], "1")

    def test_upstream_optional_create_parameter_mapping_no_silent_drop(self):
        args = {"username": "created", "ssh_keys": ["ssh-ed25519 AAA"],
                "enable_podman": False, "gpu": "0", "gpus": ["1"],
                "region": "europe", "tenant_id": "default",
                "idle_stop_minutes": 120, "network_tenant": "group",
                "os_type": "ubuntu", "isolation": "container"}
        _, _, body = rest_routes.route("create_container", args)
        self.assertEqual(body["enablePodman"], False)
        self.assertEqual(body["gpu"], "0")
        self.assertEqual(body["gpus"], ["1"])
        self.assertEqual(body["tenantId"], "default")
        self.assertEqual(body["networkTenant"], "group")
        self.assertEqual(body["osType"], "OS_TYPE_UBUNTU_2404")
        self.assertEqual(body["isolation"], "ISOLATION_TYPE_CONTAINER")
        self.assertEqual(body["idleStopMinutes"], 120)

    def test_reject_all_unknown_create_fields_and_unsafe_routes_before_post(self):
        base = {"username": "created", "ssh_keys": ["ssh-ed25519 AAA"]}
        for changed in ({"debug": True}, {"request_key": "a" * 32},
                        {"ssh_keys": []}, {"enable_podman": "false"},
                        {"idle_stop_minutes": 1.5}, {"os_type": "netbsd"},
                        {"isolation": "other"},
                        {"disable_auto_sleep": True, "idle_stop_minutes": 30}):
            with self.subTest(changed=changed), self.assertRaises(
                    rest_routes.RouteFault):
                rest_routes.route("create_container", {**base, **changed})
        for logical in ("../root", "containarium-core-state", "hi/other", ""):
            with self.subTest(logical=logical), self.assertRaises(
                    rest_routes.RouteFault):
                rest_routes.route("get_container", {"username": logical})
        with self.assertRaises(rest_routes.RouteFault):
            rest_routes.route("delete_container", {"username": "safe", "force": True})
        with self.assertRaises(rest_routes.RouteFault):
            rest_routes.validate_read_result("list_containers", {
                "containers": [], "nextCursor": "opaque"})
        with self.assertRaises(rest_routes.RouteFault):
            rest_routes.validate_read_result("get_container", {
                "container": {"username": "foreign"}}, username="safe")

    def test_app_rest_create_is_async_but_not_ready(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            key = root / "jwt"
            key.write_text("a" * 72)
            key.chmod(0o400)
            client = RESTClient({"url": "http://127.0.0.1:18080",
                                 "token_file": str(key)})
            captured = []
            def fake(request, timeout):
                captured.append((request.full_url, request.get_method(),
                                 json.loads(request.data)))
                return Reply({"container": {"username": "demo",
                                              "state": "CONTAINER_STATE_CREATING"}})
            client.opener.open = fake
            response = client.mutate("create", "demo", {
                "cpu": "1", "memory": "1GB", "disk": "10GB",
                "ssh_public_key": "ssh-ed25519 AAA"}, timeout=60)
            self.assertEqual(response["container"]["state"],
                             "CONTAINER_STATE_CREATING")
            self.assertEqual(len(captured), 1)
            self.assertEqual(captured[0][1], "POST")
            self.assertEqual(captured[0][2]["async"], True)

    def test_shared_wire_refuses_nonloopback_and_unsafe_token(self):
        with tempfile.TemporaryDirectory() as tmp:
            for endpoint in ("http://localhost:8080", "https://127.0.0.1:8080",
                             "http://127.0.0.1:8080@evil.test", "http://evil.test"):
                with self.subTest(endpoint=endpoint), self.assertRaises(
                        rest_wire.WireFault):
                    rest_wire.Client({"url": endpoint,
                                      "token_file": str(Path(tmp) / "jwt")})
            key = Path(tmp) / "jwt"
            key.write_text("a" * 72)
            key.chmod(0o600)
            client = rest_wire.Client({"url": "http://127.0.0.1:18080",
                                       "token_file": str(key)})
            with self.assertRaises(rest_wire.WireFault):
                client.request("GET", "/v1/containers")
            key.chmod(0o400)
            response = client.opener.open = lambda request, timeout: Reply(
                {"containers": []})
            self.assertEqual(client.request("GET", "/v1/containers"),
                             {"containers": []})
            with self.assertRaises(rest_wire.WireFault):
                client.request("POST", "http://evil.test/v1/containers", {})
