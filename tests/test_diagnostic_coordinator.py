"""DC-01 fixtures: no host privileges, real Provider or credential bytes."""
import io
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch
from contextlib import redirect_stdout

from bwmini import diagnostic_probe as app
from bwmini import gate_diagnostics as gate
from bwmini import mcp_gate


class FakeRest:
    def __init__(self, error=None):
        self.error = error
        self.reads = 0

    def inventory(self):
        self.reads += 1
        if self.error:
            raise self.error
        return {"secret": "MUST_NOT_EXPORT", "containers": []}


class FakeProvider:
    def __init__(self, c, error=None):
        self.rest = FakeRest(error) if c.get("provider_api") else None
        self.argv = ["/not-executed/launcher", "SECRET"]


class FakeMCP:
    calls = []
    def __init__(self, argv, timeout):
        self.calls.append(("init", timeout))
    def __enter__(self):
        return self
    def __exit__(self, *args):
        return False
    def discover(self):
        self.calls.append(("discover", None))
    def tool(self, name, args):
        self.calls.append((name, args))
        return {"content": [{"text": "SECRET"}]}


class AppProbeTests(unittest.TestCase):
    def setUp(self):
        self.config = {"mode": "mcp", "provider": "containarium",
                       "provider_api": {"url": "http://127.0.0.1:9090",
                                        "token_file": "/private/secret"}}
        FakeMCP.calls = []

    def reader(self, _):
        return dict(self.config)

    def test_preflight_is_local_and_never_imports_live_provider(self):
        output = app.run("preflight", "/fake/runtime", "/fake/persisted",
                         reader=self.reader, provider_factory=FakeProvider,
                         mcp_factory=FakeMCP)
        self.assertEqual(output["state"], "passed")
        self.assertNotIn("token_file", json.dumps(output))
        self.assertFalse(FakeMCP.calls)
        self.assertNotIn("rest_inventory", [v["id"] for v in output["checks"]])

    def test_rest_and_mcp_independent_readiness(self):
        provider = FakeProvider(self.config)
        output = app.run("rest_read", "/fake/runtime", "/fake/persisted",
                         reader=self.reader, provider_factory=lambda c: provider,
                         mcp_factory=FakeMCP)
        self.assertEqual(output["state"], "passed")
        self.assertEqual(provider.rest.reads, 1)
        self.assertEqual(FakeMCP.calls[-1], ("list_containers", {}))
        self.assertNotIn("MUST_NOT_EXPORT", json.dumps(output))
        self.assertNotIn("SECRET", json.dumps(output))

    def test_binding_drift_is_failed_and_probes_never_mutate_config(self):
        calls = []
        def drift(path):
            calls.append(path)
            item = dict(self.config)
            if "persisted" in path:
                item["provider_api"] = {"url": "http://127.0.0.1:9091",
                                        "token_file": "/private/secret"}
            return item
        output = app.run("preflight", "/fake/runtime", "/fake/persisted",
                         reader=drift, provider_factory=FakeProvider)
        self.assertEqual(output["state"], "failed")
        self.assertIn("PROVIDER_BINDING_DRIFT", json.dumps(output))
        self.assertEqual(calls, ["/fake/runtime", "/fake/persisted"])

    def test_legacy_mcp_mode_stays_available_and_rest_is_unverified(self):
        def legacy(_):
            return {"mode": "mcp", "provider": "containarium", "launcher": ["/bin/false"]}
        output = app.run("rest_read", "/fake/runtime", "/fake/persisted",
                         reader=legacy, provider_factory=FakeProvider,
                         mcp_factory=FakeMCP)
        self.assertEqual(output["state"], "not_verified")
        self.assertIn("MCP_AUTHENTICATED_READ_PASSED", json.dumps(output))
        self.assertIn("REST_NOT_CONFIGURED", json.dumps(output))

    def test_config_errors_are_typed_without_original_details(self):
        def bad(_):
            raise OSError("SECRET path and token text")
        output = app.run("rest_read", "/fake/runtime", "/fake/persisted", reader=bad)
        self.assertEqual(output["state"], "not_verified")
        self.assertNotIn("SECRET", json.dumps(output))


class GatewayCoordinatorTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.prefix = Path(self.temp.name)
        self.workdir = self.prefix / "current"
        (self.workdir / ".venv/bin").mkdir(parents=True)
        (self.workdir / ".venv/bin/python").write_text("test")
        (self.prefix / "runtime.json").write_text("{}")
        (self.prefix / "config.json").write_text("{}")

    def tearDown(self):
        self.temp.cleanup()

    def payload(self, profile="preflight", state="passed"):
        return {"schema_version": 1, "profile": profile, "state": state,
                "coverage": "app_owned_probe", "checks": [
                    {"id": "config", "state": state, "code": "CONFIG_READ_PASSED"}],
                "omissions": []}

    def test_catalog_and_fixed_profile_schema(self):
        tools = mcp_gate.catalog({"result": {"tools": []}})["result"]["tools"]
        item = next(t for t in tools if t["name"] == "boxweave_diagnostic_run")
        self.assertTrue(item["annotations"]["readOnlyHint"])
        self.assertEqual(item["inputSchema"]["properties"]["profile"]["enum"],
                         ["preflight", "rest_read", "operation_trace"])
        self.assertFalse(item["inputSchema"]["additionalProperties"])

    def test_fixed_subprocess_argv_cwd_and_safe_projection(self):
        response = SimpleNamespace(returncode=0,
                                   stdout=json.dumps(self.payload()).encode())
        with patch.object(gate.subprocess, "run", return_value=response) as mocked:
            result = gate.query_coordinator({"profile": "preflight"}, prefix=self.prefix)
        argv = mocked.call_args.args[0]
        self.assertEqual(argv[1:4], ["-m", "bwmini.diagnostic_probe", "--profile"])
        self.assertEqual(mocked.call_args.kwargs["cwd"], str(self.workdir))
        self.assertEqual(mocked.call_args.kwargs["stdin"], gate.subprocess.DEVNULL)
        self.assertEqual(mocked.call_args.kwargs["stderr"], gate.subprocess.DEVNULL)
        self.assertLessEqual(mocked.call_args.kwargs["timeout"], 60)
        self.assertEqual(result["state"], "passed")

    def test_invalid_inputs_do_not_invoke_any_app_process(self):
        for query in (
            {"profile": "preflight", "command": "id"},
            {"profile": "preflight", "config": "/etc/shadow"},
            {"profile": "rest_read", "operation_id": "abc"},
            {"profile": "operation_trace"},
            {"profile": "operation_trace", "operation_kind": "environment",
             "operation_id": "../config"},
            {"profile": "selftest"},
        ):
            with self.subTest(query=query), patch.object(
                gate.subprocess, "run"
            ) as mocked, self.assertRaisesRegex(ValueError, "COORDINATOR_SELECTOR_INVALID"):
                gate.query_coordinator(query, prefix=self.prefix)
                mocked.assert_not_called()

    def test_missing_workdir_and_python_are_distinct(self):
        missing = self.prefix / "missing"
        self.assertEqual(gate.query_coordinator(
            {"profile": "preflight"}, prefix=missing)["checks"][0]["code"],
            "APP_WORKDIR_MISSING")
        (self.workdir / ".venv/bin/python").unlink()
        self.assertEqual(gate.query_coordinator(
            {"profile": "preflight"}, prefix=self.prefix)["checks"][0]["code"],
            "APP_PYTHON_MISSING")

    def test_secret_and_extra_fields_cannot_escape_gateway(self):
        modified = self.payload()
        modified["debug"] = "MUST_NOT_EXPORT_SECRET"
        with patch.object(gate.subprocess, "run", return_value=SimpleNamespace(
            returncode=0, stdout=json.dumps(modified).encode())):
            result = gate.query_coordinator({"profile": "preflight"}, prefix=self.prefix)
        self.assertEqual(result["state"], "not_verified")
        self.assertNotIn("MUST_NOT_EXPORT_SECRET", json.dumps(result))
        modified.pop("debug")
        modified["checks"][0]["code"] = "SECRET raw-bearer"
        with patch.object(gate.subprocess, "run", return_value=SimpleNamespace(
            returncode=0, stdout=json.dumps(modified).encode())):
            result = gate.query_coordinator({"profile": "preflight"}, prefix=self.prefix)
        self.assertEqual(result["state"], "not_verified")
        self.assertNotIn("raw-bearer", json.dumps(result))

    def test_timeout_and_import_failure_are_not_verified(self):
        from subprocess import TimeoutExpired
        with patch.object(gate.subprocess, "run", side_effect=TimeoutExpired("python", 55)):
            self.assertEqual(gate.query_coordinator(
                {"profile": "rest_read"}, prefix=self.prefix)["checks"][0]["code"],
                "APP_PROBE_TIMEOUT")
        with patch.object(gate.subprocess, "run", return_value=SimpleNamespace(
            returncode=1, stdout=b"ModuleNotFoundError: SECRET")):
            result = gate.query_coordinator({"profile": "rest_read"}, prefix=self.prefix)
        self.assertEqual(result["state"], "not_verified")
        self.assertNotIn("SECRET", json.dumps(result))

    def test_d2_trace_linkage_from_existing_scoped_reader(self):
        synthetic = {"state": "observed", "coverage": "app_owned_reader",
                     "trace_id": "a" * 32,
                     "events": [{"component": "lifecycle", "code": "CLAIMED"},
                                {"component": "provider_api", "code": "API_RESPONSE_RECEIVED"}],
                     "omissions": ["receipt_detail_fields_not_exported"]}
        with patch.object(gate, "query_app", return_value=synthetic) as mocked:
            result = gate.query_coordinator(
                {"profile": "operation_trace", "operation_kind": "environment",
                 "operation_id": "a" * 32}, prefix=self.prefix)
        self.assertEqual(result["state"], "passed")
        self.assertEqual(result["event_count"], 2)
        self.assertEqual(result["trace_id"], "a" * 32)
        self.assertEqual(mocked.call_args.args[0]["source"], "app")
        self.assertEqual(mocked.call_args.args[0]["operation_id"], "a" * 32)
        synthetic["events"] = [{"component": "lifecycle", "code": "CLAIMED"}]
        with patch.object(gate, "query_app", return_value=synthetic):
            incomplete = gate.query_coordinator(
                {"profile": "operation_trace", "operation_kind": "environment",
                 "operation_id": "a" * 32}, prefix=self.prefix)
        self.assertEqual(incomplete["state"], "not_verified")

    def test_gateway_mcp_dispatch_never_reaches_provider_for_diagnosis(self):
        class KeepOpen(io.BytesIO):
            def close(self):
                pass
        class ProviderProcess:
            def __init__(self):
                self.stdin = KeepOpen()
                self.stdout = io.BytesIO()
            def poll(self):
                return 0
        process = ProviderProcess()
        query = {"jsonrpc": "2.0", "id": 4, "method": "tools/call",
                 "params": {"name": "boxweave_diagnostic_run",
                            "arguments": {"profile": "preflight"}}}
        with (patch.object(mcp_gate, "tool_policy", return_value=None),
              patch.object(mcp_gate.gateway_protection, "enabled", return_value=False),
              patch.object(mcp_gate.subprocess, "Popen", return_value=process),
              patch.object(mcp_gate.gd, "query_coordinator",
                           return_value={"state": "passed"}) as coordinated,
              patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(query) + "\n")),
              redirect_stdout(io.StringIO()) as out):
            mcp_gate.main()
        value = json.loads(out.getvalue())
        self.assertEqual(value["id"], 4)
        self.assertIn("passed", value["result"]["content"][0]["text"])
        self.assertEqual(process.stdin.getvalue(), b"")
        coordinated.assert_called_once_with({"profile": "preflight"})


if __name__ == "__main__":
    unittest.main()
