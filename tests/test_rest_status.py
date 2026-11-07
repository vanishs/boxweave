"""Standard gRPC/HTTP semantics at the Containarium REST error boundary."""
import io
import json
import unittest
import urllib.error
from unittest.mock import patch

from bwmini import rest_status, rest_wire, gateway_create, gateway_rest, provider_rest
from bwmini.domain import Fault


class CanonicalProviderErrorTests(unittest.TestCase):
    def envelope(self, name, status, desc="test"):
        return json.dumps({
            "code": status,
            "error": "rpc error: code = " + name + " desc = " + desc,
            "message": desc,
        }).encode()

    def test_all_native_grpc_codes_use_standard_http_mappings(self):
        for code in rest_status.GRPCCode:
            if code is rest_status.GRPCCode.OK:
                continue
            name = "".join(x.capitalize() for x in code.name.split("_"))
            if code is rest_status.GRPCCode.CANCELLED:
                name = "Canceled"
            http = rest_status.http_status_for(code)
            with self.subTest(grpc=code.name):
                got = rest_status.normalize(
                    "POST", "/v1/containers/demo/stop", http,
                    self.envelope(name, http))
                self.assertEqual(got.grpc_code, code)
                self.assertEqual(got.http_status, http)
                self.assertEqual(got.upstream_http_status, http)
                self.assertTrue(got.typed)

    def test_inconsistent_status_and_unstructured_responses_fail_closed(self):
        cases = [
            (500, self.envelope("NotFound", 500)),
            (404, self.envelope("Internal", 404)),
            (500, b'{"code":500,"error":"Instance not found","message":"Instance not found"}'),
            (500, b'{"code":500,"error":"rpc error: code = NotFound desc = missing","message":"wrong"}'),
            (409, b'{"code":409,"error":"conflict","message":"conflict"}'),
            (404, b"not found"),
            (500, b""),
            (500, b"x" * 4097),
            (500, b'{"code":500,"code":500,"error":"e","message":"e"}'),
        ]
        for http, raw in cases:
            with self.subTest(http=http, body=raw[:55]):
                got = rest_status.normalize(
                    "GET", "/v1/containers/demo", http, raw)
                self.assertIsNone(got.grpc_code)
                self.assertFalse(got.typed)
                self.assertEqual(got.http_status, http)

    def test_http_only_unambiguous_statuses(self):
        expected = {
            401: rest_status.GRPCCode.UNAUTHENTICATED,
            403: rest_status.GRPCCode.PERMISSION_DENIED,
            429: rest_status.GRPCCode.RESOURCE_EXHAUSTED,
            501: rest_status.GRPCCode.UNIMPLEMENTED,
            503: rest_status.GRPCCode.UNAVAILABLE,
            504: rest_status.GRPCCode.DEADLINE_EXCEEDED,
        }
        for http, code in expected.items():
            with self.subTest(http=http):
                got = rest_status.normalize(
                    "GET", "/v1/containers", http, b'{"error":"redacted"}')
                self.assertEqual(got.grpc_code, code)
                self.assertEqual(got.http_status, http)
                self.assertEqual(got.origin, "http")
                self.assertFalse(got.typed)

    def test_exact_v0128_get_container_500_compatibility(self):
        logical = "bw50qa-async-a"
        route = "/v1/containers/" + logical
        detail = ("failed to get container: failed to get container: "
                  'Failed to fetch instance "' + logical +
                  '-container" in project "default": Instance not found')
        payload = {
            "code": 500,
            "error": "rpc error: code = Unknown desc = " + detail,
            "message": detail,
        }
        body = json.dumps(payload).encode()
        got = rest_status.normalize("GET", route, 500, body)
        self.assertEqual(got.grpc_code, rest_status.GRPCCode.NOT_FOUND)
        self.assertEqual(got.http_status, 404)
        self.assertEqual(got.upstream_http_status, 500)
        self.assertEqual(got.origin, "compatibility")
        bad = [
            ("POST", route, 500, body),
            ("GET", "/v1/containers", 500, body),
            ("GET", "/v1/containers/another", 500, body),
            ("GET", route + "/debug", 500, body),
            ("GET", route, 503, body),
            ("GET", route, 500, json.dumps({
                **payload, "message": detail.replace("default", "other")
            }).encode()),
            ("GET", route, 500, body.replace(b"Instance not found", b"not found")),
            ("GET", route, 500, body[:-1] + b',"code":500}'),
            ("GET", route, 500, body + b"x"),
            ("GET", route, 500, body + b"x" * 4097),
        ]
        for method, path, http, raw in bad:
            with self.subTest(method=method, route=path, http=http, size=len(raw)):
                result = rest_status.normalize(method, path, http, raw)
                self.assertFalse(result.typed and result.grpc_code is
                                 rest_status.GRPCCode.NOT_FOUND)

    def test_wire_reports_normalized_and_raw_status_without_body(self):
        class Owner:
            url = "http://127.0.0.1:18080"
            def _credential(self):
                return "placeholder-not-real"
            class Opener:
                def open(self, req, timeout):
                    raw = json.dumps({
                        "code": 503,
                        "error": "rpc error: code = Unavailable desc = private backend secret",
                        "message": "private backend secret",
                    }).encode()
                    raise urllib.error.HTTPError(
                        req.full_url, 503, "Service unavailable", {},
                        io.BytesIO(raw))
            opener = Opener()
        with self.assertRaises(rest_wire.WireFault) as caught:
            rest_wire.request(Owner(), "GET", "/v1/containers")
        error = caught.exception
        self.assertEqual(error.code, "API_HTTP_REJECTED")
        self.assertEqual(error.grpc_code, rest_status.GRPCCode.UNAVAILABLE)
        self.assertEqual(error.http_status, 503)
        self.assertEqual(error.upstream_http_status, 503)
        self.assertEqual(error.status_origin, "grpc")
        self.assertNotIn("private backend secret", str(error))

    def test_app_uses_canonical_provider_status_without_leaking_body(self):
        # No live requests: test the same RESTClient boundary used by Web/CLI.
        class R:
            grpc_code = rest_status.GRPCCode.NOT_FOUND
            http_status = 404
            upstream_http_status = 500
            status_origin = "compatibility"
            code = "API_HTTP_REJECTED"
            status = 409
        fake = rest_wire.WireFault(
            R.code, grpc_code=R.grpc_code, http_status=R.http_status,
            upstream_http_status=R.upstream_http_status,
            status_origin=R.status_origin)
        client = object.__new__(provider_rest.RESTClient)
        with patch.object(rest_wire, "request", side_effect=fake):
            with self.assertRaises(Fault) as caught:
                client.request("GET", "/v1/containers/demo")
        self.assertEqual(caught.exception.code, "NOT_FOUND")
        self.assertEqual(caught.exception.status, 404)
        self.assertEqual(caught.exception.upstream_http_status, 500)

    def test_gateway_read_uses_canonical_semantics(self):
        fake = rest_wire.WireFault(
            "API_HTTP_REJECTED", grpc_code=rest_status.GRPCCode.UNAVAILABLE,
            http_status=503, upstream_http_status=503, status_origin="grpc")
        with patch.object(rest_wire.Client, "request", side_effect=fake):
            with self.assertRaisesRegex(gateway_rest.GatewayRESTFault, "UNAVAILABLE"):
                gateway_rest.execute_read(
                    {"url":"http://127.0.0.1:18080",
                     "token_file":"/tmp/placeholder-not-read"},
                    "list_containers", {})

    def test_unclassified_http_errors_expose_canonical_unknown_not_private_transport_codes(self):
        # An untyped HTTP response cannot safely prove NOT_FOUND/CONFLICT.
        for upstream in (400, 404, 409, 500):
            with self.subTest(upstream=upstream):
                unknown = rest_wire.WireFault(
                    "API_HTTP_REJECTED", status=upstream, grpc_code=None,
                    http_status=upstream, upstream_http_status=upstream,
                    status_origin="unclassified")
                app = object.__new__(provider_rest.RESTClient)
                with patch.object(rest_wire, "request", side_effect=unknown):
                    with self.assertRaises(Fault) as caught:
                        app.request("GET", "/v1/containers/demo")
                self.assertEqual(caught.exception.code, "UNKNOWN")
                self.assertEqual(caught.exception.status, 500)
                self.assertIsNone(caught.exception.grpc_code)
                self.assertEqual(caught.exception.upstream_http_status, upstream)
                with patch.object(rest_wire.Client, "request", side_effect=unknown):
                    with self.assertRaisesRegex(
                            gateway_rest.GatewayRESTFault, "^UNKNOWN$"):
                        gateway_rest.execute_read(
                            {"url":"http://127.0.0.1:18080",
                             "token_file":"/tmp/placeholder-not-read"},
                            "list_containers", {})

    def test_transport_loss_is_standard_unavailable_for_reads_unknown_for_writes(self):
        class Owner:
            url = "http://127.0.0.1:18080"
            def _credential(self):
                return "placeholder-not-real"
            class Opener:
                def open(self, req, timeout):
                    raise urllib.error.URLError("do-not-reveal")
            opener = Opener()
        for method, expected_code, expected_http in (
            ("GET", rest_status.GRPCCode.UNAVAILABLE, 503),
            ("POST", rest_status.GRPCCode.UNKNOWN, 500),
            ("DELETE", rest_status.GRPCCode.UNKNOWN, 500),
            ("PUT", rest_status.GRPCCode.UNKNOWN, 500),
        ):
            with self.subTest(method=method):
                with self.assertRaises(rest_wire.WireFault) as caught:
                    rest_wire.request(
                        Owner(), method, "/v1/containers/demo",
                        body={} if method != "GET" else None)
                err = caught.exception
                self.assertEqual(err.grpc_code, expected_code)
                self.assertEqual(err.http_status, expected_http)
                self.assertIsNone(err.upstream_http_status)
                self.assertEqual(err.status_origin, "transport")
                self.assertNotIn("do-not-reveal", str(err))
