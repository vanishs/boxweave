"""Host CLI contracts: no privileged host changes, network or credential access."""
import io
import json
import os
import subprocess
import sys
from contextlib import ExitStack
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from bwmini import host_cli


def fixture():
    return {"schema_version": 1, "phase": "ready", "version": "0.17.9",
            "application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave",
            "web_port": 18640, "host": "10.0.0.5", "port": 22,
            "server_key": "ssh-ed25519 public", "managed_public_key": "ssh-ed25519 public"}


HEALTHY_UNIT = {"LoadState": "loaded", "ActiveState": "active", "UnitFileState": "enabled"}
MANAGER_UNIT = {"LoadState": "loaded", "ActiveState": "active", "UnitFileState": "static"}
MCP_READ = {"state": "authenticated_read_passed", "protocol": "2024-11-05", "tools": 4, "read": "list_containers"}


def health_fixture(state="ready", units=None, version="0.17.9", schema=1, mcp=None, admin=None, code=None):
    value = {"state": state, "diagnostic_schema": schema, "incomplete": False,
             "unit_states": {"serve": "ready", "worker": "ready"} if units is None else units,
             "installed": {"version": version}, "services": {},
             "web": {"reachable": True, "authenticated": False, "status": 401},
             "mcp": {"state": "not_checked"}}
    if mcp is not None:
        value["mcp"] = mcp
    if admin is not None:
        value["administrator_initialized"] = admin
    if code is not None:
        value["code"] = code
    return value


class FakeRecord:
    pw_uid = 18640
    pw_name = "boxweave"


def checks_by_id(result):
    return {c["id"]: c for c in result["checks"]}


class DoctorTests(unittest.TestCase):
    def doctor(self, health, mcp=False, root=True, web=None, service_side=None):
        """Run doctor() under complete host stubs; returns (result, reader_mock).

        health: parsed dict (JSON-encoded for the reader), a raw reader output
        string, or None for a reader launch failure.
        """
        web = web or {"reachable": True, "http_status": 401, "authenticated": False}
        reader_output = health if isinstance(health, str) else (
            json.dumps(health) if health is not None else None)
        with ExitStack() as stack:
            stack.enter_context(patch.object(host_cli.os, "geteuid",
                                             return_value=0 if root else 1000))
            stack.enter_context(patch.object(host_cli, "web_observation", return_value=web))
            stack.enter_context(patch.object(host_cli, "service_observation",
                                             side_effect=service_side or
                                             (lambda name: MANAGER_UNIT if name.startswith("user@")
                                              else HEALTHY_UNIT)))
            reader = stack.enter_context(patch.object(host_cli, "run_bounded",
                                                      return_value=reader_output))
            stack.enter_context(patch.object(host_cli.pwd, "getpwnam", return_value=FakeRecord()))
            result = host_cli.doctor(fixture(), mcp=mcp)
        return result, reader

    def doctor_main(self, health, argv, root=True, reader_output=None):
        """Run main(doctor) under complete host stubs; returns (exit_code, stdout)."""
        output = reader_output if reader_output is not None else (
            json.dumps(health) if isinstance(health, dict) else health)
        with ExitStack() as stack:
            stack.enter_context(patch.object(host_cli.os, "geteuid",
                                             return_value=0 if root else 1000))
            stack.enter_context(patch.object(host_cli, "web_observation",
                                             return_value={"reachable": True, "http_status": 401,
                                                           "authenticated": False}))
            stack.enter_context(patch.object(host_cli, "service_observation",
                                             side_effect=lambda name: MANAGER_UNIT
                                             if name.startswith("user@") else HEALTHY_UNIT))
            stack.enter_context(patch.object(host_cli, "run_bounded", return_value=output))
            stack.enter_context(patch.object(host_cli.pwd, "getpwnam", return_value=FakeRecord()))
            stack.enter_context(patch.object(host_cli, "public_state", return_value=fixture()))
            out = io.StringIO()
            stack.enter_context(patch("sys.stdout", out))
            code = host_cli.main(argv)
        return code, out.getvalue()

    def test_d0_t01_worker_failed_is_failed_despite_reachable_web(self):
        health = health_fixture(state="failed", units={"serve": "ready", "worker": "inactive"},
                                code="HEALTH_STATE_FAILED")
        result, _ = self.doctor(health)
        by_id = checks_by_id(result)
        self.assertEqual(result["overall"], "failed")
        self.assertEqual(result["diagnostic_schema"], 1)
        self.assertEqual(by_id["web_loopback"]["state"], "passed")
        self.assertEqual(by_id["web_loopback"]["code"], "HTTP_REACHABLE")
        self.assertEqual(by_id["app_release_health"]["state"], "failed")
        self.assertEqual(by_id["app_serve_unit"]["state"], "passed")
        self.assertEqual(by_id["app_worker_unit"], {"id": "app_worker_unit", "state": "failed",
                                                    "code": "APP_UNIT_INACTIVE",
                                                    "scope": "app_user_systemd", "required": True})
        self.assertEqual(by_id["authenticated_mcp"]["state"], "not_checked")
        self.assertFalse(by_id["authenticated_mcp"]["required"])

    def test_d0_t02_unprivileged_doctor_is_incomplete_never_passed(self):
        result, reader = self.doctor(health_fixture(), root=False)
        by_id = checks_by_id(result)
        self.assertEqual(result["overall"], "incomplete")
        self.assertEqual(by_id["app_release_health"]["code"], "APP_HEALTH_REQUIRES_ADMIN")
        self.assertIn("sudo boxweave doctor", by_id["app_release_health"]["detail"])
        self.assertEqual(by_id["app_serve_unit"]["code"], "APP_UNIT_VISIBILITY_UNAVAILABLE")
        self.assertEqual(by_id["app_worker_unit"]["code"], "APP_UNIT_VISIBILITY_UNAVAILABLE")
        self.assertEqual(result["coverage"]["app_authenticated_health"], "not_verified_requires_admin")
        reader.assert_not_called()

    def test_d0_t04_unprivileged_mcp_doctor_never_invokes_reader(self):
        code, _ = self.doctor_main(health_fixture(), ["doctor", "--mcp"], root=False, reader_output="")
        self.assertEqual(code, 2)

    def test_d0_t05_reader_failures_are_not_verified_not_crash(self):
        for broken in (None, "not json", json.dumps({"state": "failed", "code": "X"}),
                       json.dumps({"state": "ready", "diagnostic_schema": 2, "unit_states": {}}),
                       json.dumps({"state": "ready", "diagnostic_schema": 1})):
            with self.subTest(broken=broken is None):
                result, _ = self.doctor(broken)
                by_id = checks_by_id(result)
                self.assertEqual(result["overall"], "incomplete")
                self.assertEqual(by_id["app_release_health"]["state"], "not_verified")
                self.assertIn(by_id["app_release_health"]["code"],
                              ("APP_HEALTH_READER_UNAVAILABLE", "INVALID_APP_HEALTH_RESULT"))

    def test_d0_t06_mcp_pass_never_masks_failed_worker(self):
        health = health_fixture(state="failed", units={"serve": "ready", "worker": "inactive"},
                                code="MCP_READ_FAILED", mcp=MCP_READ, admin=True)
        result, _ = self.doctor(health, mcp=True)
        by_id = checks_by_id(result)
        self.assertEqual(result["overall"], "failed")
        self.assertEqual(by_id["authenticated_mcp"]["state"], "passed")
        self.assertEqual(by_id["admin_initialized"]["state"], "passed")
        self.assertEqual(by_id["app_worker_unit"]["state"], "failed")
        self.assertEqual(result["mcp"]["state"], "observed")

    def test_d0_t07_missing_admin_initialization_is_not_success(self):
        health = health_fixture(state="ready", units={"serve": "ready", "worker": "ready"},
                                mcp=MCP_READ, admin=False)
        result, _ = self.doctor(health, mcp=True)
        by_id = checks_by_id(result)
        self.assertEqual(result["overall"], "failed")
        self.assertEqual(by_id["admin_initialized"], {"id": "admin_initialized", "state": "failed",
                                                      "code": "ADMIN_SETUP_REQUIRED",
                                                      "scope": "app_user_mcp", "required": True})

    def test_all_promised_checks_pass_is_exit_zero(self):
        health = health_fixture(state="ready", mcp=MCP_READ, admin=True)
        code, out = self.doctor_main(health, ["doctor", "--mcp"])
        self.assertEqual(code, 0)
        self.assertEqual(json.loads(out)["overall"], "passed")

    def test_release_mismatch_is_not_verified(self):
        result, _ = self.doctor(health_fixture(version="0.16.0"))
        by_id = checks_by_id(result)
        self.assertEqual(result["overall"], "incomplete")
        self.assertEqual(by_id["app_release_health"]["code"], "APP_RELEASE_DESCRIPTOR_MISMATCH")

    def test_empty_unit_list_is_never_ready(self):
        result, _ = self.doctor(health_fixture(units={}))
        by_id = checks_by_id(result)
        self.assertEqual(by_id["app_release_health"]["code"], "APP_UNIT_LIST_EMPTY")
        self.assertEqual(by_id["app_serve_unit"]["code"], "APP_UNIT_ABSENT")
        self.assertEqual(result["overall"], "incomplete")

    def test_doctor_output_contract_and_coverage_are_frozen(self):
        result, _ = self.doctor(health_fixture())
        self.assertEqual(sorted(result),
                         ["application_user", "checks", "coverage", "diagnostic_schema",
                          "external_web_access", "mcp", "overall", "services", "state",
                          "user_manager", "version", "web", "web_on_server"])
        self.assertEqual(result["state"], "installed_public_descriptor")
        self.assertEqual(sorted(result["coverage"]),
                         ["agent_ui", "app_authenticated_health", "authenticated_mcp",
                          "effective_nesting", "model_inference", "workload_execute"])
        self.assertEqual(result["coverage"]["workload_execute"], "not_verified")
        self.assertEqual(result["coverage"]["agent_ui"], "not_verified")
        self.assertEqual(result["mcp"], {"state": "not_checked", "reason": "NOT_REQUESTED"})
        self.assertEqual([c["id"] for c in result["checks"]],
                         ["public_descriptor", "web_loopback", "host_credential_timer",
                          "host_trust_timer", "host_user_manager", "app_release_health",
                          "app_serve_unit", "app_worker_unit", "authenticated_mcp"])
        self.assertTrue(all(set(c) <= {"id", "state", "code", "scope", "required", "detail"}
                            for c in result["checks"]))

    def test_web_down_is_a_failed_required_check(self):
        result, _ = self.doctor(health_fixture(),
                                web={"reachable": False, "authenticated": False})
        by_id = checks_by_id(result)
        self.assertEqual(by_id["web_loopback"]["state"], "failed")
        self.assertEqual(by_id["web_loopback"]["code"], "WEB_LOOPBACK_UNREACHABLE")
        self.assertEqual(result["overall"], "failed")

    def test_host_timer_failures_are_reported(self):
        result, _ = self.doctor(health_fixture(), service_side=lambda name: {"state": "unavailable"}
                                if "trust" in name else HEALTHY_UNIT)
        by_id = checks_by_id(result)
        self.assertEqual(by_id["host_trust_timer"]["state"], "not_verified")
        self.assertEqual(by_id["host_trust_timer"]["code"], "UNIT_STATE_UNAVAILABLE")
        self.assertEqual(by_id["host_credential_timer"]["state"], "passed")

    def test_stopped_worker_unit_fails_doctor_exit(self):
        health = health_fixture(state="failed", units={"serve": "ready", "worker": "inactive"},
                                code="HEALTH_STATE_FAILED")
        code, out = self.doctor_main(health, ["doctor"])
        self.assertEqual(code, 1)
        self.assertEqual(json.loads(out)["overall"], "failed")


class HostCliTests(unittest.TestCase):
    def test_installed_style_isolated_interpreter_client_help(self):
        # Exactly the production -I invocation; normal imports in unittest
        # masked this regression in 0.17.9. No root, host state or SSH needed.
        root = Path(__file__).resolve().parents[1]
        env = dict(os.environ, PYTHONPATH=str(root / "bwmini"))
        process = subprocess.run([sys.executable, "-I", str(root / "bwmini/host_cli.py"),
                                  "client", "--help"],
                                 cwd="/", env=env, capture_output=True, text=True, timeout=20)
        self.assertEqual(process.returncode, 0, process.stderr)
        self.assertIn("--admin-user", process.stdout)
        self.assertNotIn("ModuleNotFoundError", process.stderr)
        self.assertNotIn("Traceback", process.stderr)

    def test_host_cli_never_invokes_app_owned_root_interpreter(self):
        online = Path("bwmini/online.py").read_text()
        self.assertIn("exec /usr/bin/python3 -I /usr/local/lib/boxweave/default/host_cli.py", online)
        self.assertNotIn("b'cd /home/boxweave/boxweave/current", online)
        self.assertNotIn('b"cd /home/boxweave/boxweave/current', online)
        self.assertIn('atomic(public_path,public_payload,0o644)', online)

    def test_noargs_is_actionable_without_root_or_private_home(self):
        with patch.object(host_cli, "public_state", return_value=fixture()), \
             patch("sys.stdout", new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main([]), 0)
        text = out.getvalue()
        self.assertIn("Configure your client", text)
        self.assertRegex(text, r"(?m)^  boxweave client$")
        self.assertRegex(text, r"(?m)^  boxweave client --help  # More options$")
        self.assertLess(text.index("  boxweave client\n"), text.index("Optional operations:"))
        self.assertIn("127.0.0.1:18640", text)
        self.assertIn("ssh -N -L", text)
        self.assertIn("not the boxweave service account", text)
        self.assertIn("BOXWEAVE_BATCH", text)
        self.assertIn("--no-enroll", text)
        self.assertIn("--already-enrolled", text)
        self.assertNotRegex(text, r"(?m)^\\s*[1-9]\\.\\s")

    def test_box_root_shows_help_without_inspecting(self):
        with patch.object(host_cli, "inspect") as inspected, \
             patch("sys.stdout", new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main(["box"]), 0)
        self.assertIn("diagnose", out.getvalue())
        inspected.assert_not_called()

    def test_only_exact_readonly_incus_names(self):
        with patch.object(host_cli, "observe") as execute:
            for name in ("", "../etc", "bad;cmd", "with/slash"):
                result, code = host_cli.inspect(name, "default")
                self.assertEqual(code, 2)
                self.assertEqual(result["code"], "INVALID_EXACT_TARGET")
            execute.assert_not_called()
        record = {"name": "box1", "status": "Running", "type": "container",
                  "expanded_config": {"security.nesting": "false", "private.token": "never-export"}}
        with patch.object(host_cli, "observe", return_value=json.dumps(record)) as query:
            result, code = host_cli.inspect("box1", "default")
        self.assertEqual(code, 0)
        self.assertEqual(result["name"], "box1")
        self.assertNotIn("never-export", json.dumps(result))
        self.assertEqual(query.call_args.args[0],
                         ["incus", "--force-local", "query", "/1.0/instances/box1?project=default"])

    def test_unprivileged_app_health_never_runs_app_code(self):
        with patch.object(host_cli.os, "geteuid", return_value=1000), \
             patch.object(host_cli, "run_bounded") as execute:
            result = host_cli.app_health(fixture(), mcp=False)
        self.assertEqual(result["code"], "APP_HEALTH_REQUIRES_ADMIN")
        self.assertIn("sudo boxweave doctor", result["next"])
        execute.assert_not_called()

    def test_app_health_parses_strict_reader_result(self):
        good = health_fixture(version="0.17.9", units={"serve": "ready", "worker": "ready"})
        with patch.object(host_cli.os, "geteuid", return_value=0), \
             patch.object(host_cli, "run_bounded", return_value=json.dumps(good)) as execute, \
             patch.object(host_cli.pwd, "getpwnam", return_value=FakeRecord()):
            result = host_cli.app_health(fixture(), mcp=False)
        self.assertEqual(result["state"], "ready")
        self.assertEqual(result["unit_states"], {"serve": "ready", "worker": "ready"})
        argv = execute.call_args.args[0]
        self.assertEqual(argv[4:9], ["env", "XDG_RUNTIME_DIR=/run/user/18640",
                                     "/home/boxweave/boxweave/current/.venv/bin/python",
                                     "-m", "bwmini.distribution"])
        self.assertEqual(argv[9:12], ["health", "--prefix", "/home/boxweave/boxweave"])
        self.assertNotIn("--mcp", argv)
        with patch.object(host_cli.os, "geteuid", return_value=0), \
             patch.object(host_cli, "run_bounded", return_value=json.dumps(good)) as execute, \
             patch.object(host_cli.pwd, "getpwnam", return_value=FakeRecord()):
            host_cli.app_health(fixture(), mcp=True)
        self.assertIn("--mcp", execute.call_args.args[0])
        for broken in (None, "not json", json.dumps({"state": "ready"}),
                        json.dumps({"state": "ready", "diagnostic_schema": 1, "unit_states": [],
                                    "installed": {"version": "other"}})):
            with self.subTest(broken=broken is None), \
                 patch.object(host_cli.os, "geteuid", return_value=0), \
                 patch.object(host_cli, "run_bounded", return_value=broken), \
                 patch.object(host_cli.pwd, "getpwnam", return_value=FakeRecord()):
                refused = host_cli.app_health(fixture(), mcp=False)
            self.assertEqual(refused["state"], "not_verified")

    def test_public_state_refuses_symlink_and_writeable_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            descriptor = Path(tmp) / "public.json"
            descriptor.write_text(json.dumps(fixture()))
            descriptor.chmod(0o666)
            with patch.object(host_cli, "PUBLIC_STATE", descriptor):
                with self.assertRaises(ValueError):
                    host_cli.public_state()
            descriptor.chmod(0o600)
            link = Path(tmp) / "link.json"
            link.symlink_to(descriptor)
            with patch.object(host_cli, "PUBLIC_STATE", link):
                with self.assertRaises(ValueError):
                    host_cli.public_state()

    def test_status_observation_is_explicitly_not_authenticated(self):
        with patch.object(host_cli, "web_observation", return_value={"reachable": True, "authenticated": False}):
            result = host_cli.status(fixture())
        self.assertEqual(result["mcp"], "not_checked")
        self.assertEqual(result["app_health"], "not_checked")
        self.assertFalse(result["web"]["authenticated"])

    def provenance(self):
        return {"schema_version": 1, "version": "0.21.0", "source": "a" * 40,
                "installed_utc": "2026-10-31T00:00:00+00:00", "provider_mode": "standalone",
                "host": "10.0.0.5", "port": 22, "web_port": 18640,
                "application_user": "boxweave", "managed_files": 12,
                "phases": [{"name": "ready", "started_utc": "2026-10-31T00:00:00+00:00"}]}

    def test_provenance_validator_accepts_only_the_published_shape(self):
        good = self.provenance()
        host_cli.check_provenance(good)
        for change in ({"source": "short"}, {"source": "Z" * 40}, {"version": "bad"},
                       {"provider_mode": "fixture"}, {"managed_files": -1},
                       {"phases": [{"name": "Ready", "started_utc": "x"}]},
                       {"phases": [{"name": "ready"}]}, {"extra": 1}, {"port": 70000},
                       {"host": "bad host"}, {"installed_utc": "x" * 65}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                host_cli.check_provenance({**good, **change})
        with self.assertRaises(ValueError):
            host_cli.check_provenance({k: v for k, v in good.items() if k != "phases"})

    def test_provenance_command_prints_descriptor_or_refusal(self):
        with patch.object(host_cli, "provenance_state", return_value=self.provenance()), \
             patch("sys.stdout", new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main(["provenance"]), 0)
        self.assertEqual(json.loads(out.getvalue())["version"], "0.21.0")
        with patch.object(host_cli, "provenance_state", side_effect=ValueError("INVALID_PROVENANCE_DESCRIPTOR")), \
             patch("sys.stderr", new_callable=io.StringIO) as err:
            self.assertEqual(host_cli.main(["provenance"]), 2)
        self.assertIn("INVALID_PROVENANCE_DESCRIPTOR", err.getvalue())

    def test_provenance_refuses_world_writable_descriptor_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "provenance.json"
            path.write_text(json.dumps(self.provenance()))
            path.chmod(0o666)
            with patch.object(host_cli, "PROVENANCE_STATE", path), self.assertRaises(ValueError):
                host_cli.provenance_state()


if __name__ == "__main__":
    unittest.main()
