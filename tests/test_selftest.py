"""Confirmed selftest regressions: plan hash gate, live-only, full cleanup."""
import io
import json
import unittest
from unittest.mock import patch

from bwmini import host_cli, selftest
from bwmini.domain import Fault, Target

KEY = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakefakefakefakefakefakefakefakefakefakefake"


class Live:
    mode = "mcp"
    def __init__(self):
        self.boxes = {}
        self.writes = []
        self.hello_failures = 0
        self.inspect_calls = 0
    def lifecycle_inspect(self, logical):
        self.inspect_calls += 1
        return self.boxes.get(logical)
    def lifecycle_mutate(self, action, logical, spec, timeout):
        self.writes.append((action, logical))
        if action == "create":
            t = Target("containarium", "dev", "default", logical, logical + "-container", "7", logical, logical + "-container")
            self.boxes[logical] = {"target": t, "state": "CONTAINER_STATE_RUNNING"}
        elif action == "stop":
            self.boxes[logical]["state"] = "CONTAINER_STATE_STOPPED"
        elif action == "delete":
            del self.boxes[logical]
    def hello(self, target, path, timeout):
        if self.hello_failures:
            self.hello_failures -= 1
            raise Fault("NOT_CONNECTION_READY")
        if path != "/home/" + target.user:
            raise Fault("COMMAND_FAILED")
        return {"stdout": selftest.HELLO, "exit_code": 0,
                "identity": {"user": target.user, "hostname": target.hostname}}


class Clock:
    def __init__(self):
        self.now = [1000.0]
    def __call__(self):
        return self.now[0]
    def sleep(self, seconds):
        self.now[0] += seconds


class SelftestTests(unittest.TestCase):
    def test_plan_shape_and_stable_confirmation(self):
        plan, checksum = selftest.build_plan(now="2026-10-31T00:00:00+00:00")
        self.assertRegex(plan["box"], r"^boxweave-selftest-[0-9a-f]{6}$")
        self.assertEqual(plan["resources"], {"cpu": "1", "memory": "1GB", "disk": "8GB"})
        self.assertEqual(plan["command"], "print('hello world')")
        self.assertEqual(selftest.confirmation(plan), checksum)
        other, other_checksum = selftest.build_plan(now="2026-10-31T00:00:00+00:00")
        self.assertNotEqual(other["box"], plan["box"])
        self.assertNotEqual(other_checksum, checksum)

    def test_load_plan_accepts_only_confirmed_selftest_plans(self):
        plan, checksum = selftest.build_plan(now="T")
        self.assertEqual(selftest.load_plan(plan, checksum)["box"], plan["box"])
        for change in ({"box": "other-box"}, {"box": "boxweave-selftest-ZZZZZZ"},
                       {"resources": {"cpu": "4"}}, {"command": "rm -rf /"},
                       {"schema_version": 2}, {"created_utc": "x" * 65}, {"extra": 1}):
            with self.subTest(change=change), self.assertRaisesRegex(Fault, "SELFTEST_PLAN_INVALID"):
                selftest.load_plan({**plan, **change}, selftest.confirmation({**plan, **change}))
        for bad in ("", "zz", checksum.replace("a", "b")):
            with self.subTest(bad=bad), self.assertRaisesRegex(Fault, "CONFIRMATION_MISMATCH"):
                selftest.load_plan(plan, bad)

    def test_full_run_passes_and_cleans_up(self):
        provider, clock = Live(), Clock()
        plan, checksum = selftest.build_plan(now="T")
        result = selftest.run(provider, selftest.load_plan(plan, checksum), budget=120,
                              clock=clock, sleep=clock.sleep, key=KEY)
        self.assertEqual(result["state"], "passed")
        self.assertEqual([s["step"] for s in result["steps"]], ["create", "hello", "stop", "delete"])
        self.assertTrue(all(s["state"] == "succeeded" for s in result["steps"]))
        self.assertEqual(result["cleanup"], "not_needed")
        self.assertEqual(provider.boxes, {})
        self.assertEqual([w[0] for w in provider.writes], ["create", "stop", "delete"])

    def test_hello_failure_still_cleans_up_and_reports_failed(self):
        provider, clock = Live(), Clock()
        provider.hello_failures = 1
        plan, checksum = selftest.build_plan(now="T")
        result = selftest.run(provider, selftest.load_plan(plan, checksum), budget=120,
                              clock=clock, sleep=clock.sleep, key=KEY)
        self.assertEqual(result["state"], "failed")
        self.assertEqual(result["cleanup"], "performed")
        self.assertEqual(provider.boxes, {})
        hello = [s for s in result["steps"] if s["step"] == "hello"][0]
        self.assertEqual(hello["code"], "NOT_CONNECTION_READY")

    def test_fixture_mode_is_refused_before_any_remote_call(self):
        class Fixture:
            mode = "fixture"
        plan, checksum = selftest.build_plan(now="T")
        with self.assertRaisesRegex(Fault, "SELFTEST_LIVE_REQUIRED"):
            selftest.run(Fixture(), selftest.load_plan(plan, checksum), key=KEY)

    def test_create_that_never_becomes_running_hits_deadline(self):
        class Pending(Live):
            def lifecycle_inspect(self, logical):
                self.inspect_calls += 1
                box = self.boxes.get(logical)
                return box
            def lifecycle_mutate(self, action, logical, spec, timeout):
                if action == "create":
                    self.writes.append(("create", logical))
                    t = Target("containarium", "dev", "default", logical, logical + "-container", "7", logical, logical + "-container")
                    self.boxes[logical] = {"target": t, "state": "CONTAINER_STATE_EXECUTING"}
                    return
                super().lifecycle_mutate(action, logical, spec, timeout)
        provider, clock = Pending(), Clock()
        plan, checksum = selftest.build_plan(now="T")
        result = selftest.run(provider, selftest.load_plan(plan, checksum), budget=60,
                              clock=clock, sleep=clock.sleep, key=KEY)
        self.assertEqual(result["state"], "failed")
        create = [s for s in result["steps"] if s["step"] == "create"][0]
        self.assertEqual(create["code"], "DEADLINE_EXCEEDED")

    def test_host_selftest_wrapper_wiring_and_guards(self):
        info = {"application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave"}
        with patch.object(host_cli, "public_state", return_value=info), \
             patch.object(host_cli.os, "geteuid", return_value=1000), \
             patch("sys.stderr", new_callable=io.StringIO):
            self.assertEqual(host_cli.selftest_command(["plan"]), 2)
        captured = {}
        def fake_observe(argv, timeout, limit, cwd=None):
            captured["argv"], captured["cwd"] = argv, cwd
            return json.dumps({"plan": {}, "confirmation": "a" * 64})
        with patch.object(host_cli, "public_state", return_value=info), \
             patch.object(host_cli.os, "geteuid", return_value=0), \
             patch.object(host_cli.pwd, "getpwnam",
                          return_value=type("P", (), {"pw_name": "boxweave", "pw_uid": 1001})()), \
             patch.object(host_cli, "observe", side_effect=fake_observe), \
             patch("sys.stdout", new_callable=io.StringIO):
            self.assertEqual(host_cli.selftest_command(["plan", "--budget", "120"]), 0)
        self.assertIn("selftest-plan", captured["argv"])
        self.assertEqual(captured["cwd"], "/home/boxweave/boxweave/current")
        self.assertIn("--budget", captured["argv"])
        with patch.object(host_cli, "public_state", return_value=info), \
             patch.object(host_cli.os, "geteuid", return_value=0), \
             patch.object(host_cli.pwd, "getpwnam",
                          return_value=type("P", (), {"pw_name": "boxweave", "pw_uid": 1001})()), \
             patch("sys.stderr", new_callable=io.StringIO):
            self.assertEqual(host_cli.selftest_command(["run", "--plan", "/tmp/p.json"]), 2)


    def test_cli_argparse_guards_allow_selftest_plan_arguments(self):
        import io
        import json as json_module
        import tempfile
        from pathlib import Path as FSPath
        from bwmini.__main__ import main
        plan, checksum = selftest.build_plan(now="T")
        with tempfile.TemporaryDirectory() as tmp:
            plan_file = FSPath(tmp) / "plan.json"
            plan_file.write_text(json_module.dumps(plan))
            config = FSPath(tmp) / "fixture.json"
            config.write_text(json_module.dumps({"mode": "fixture", "database": str(FSPath(tmp) / "db.sqlite3"),
                                                 "target": {"provider": "containarium", "backend": "dev",
                                                            "project": "default", "logical": "v2-lifecycle-e2e",
                                                            "instance": "v2-lifecycle-e2e-container", "created": "1",
                                                            "user": "v2-lifecycle-e2e", "hostname": "v2-lifecycle-e2e-container"},
                                                 "fixture_counter": None}))
            out, err = io.StringIO(), io.StringIO()
            argv = ["__main__.py", "selftest-run", "--config", str(config), "--plan", str(plan_file),
                    "--confirmation", checksum, "--budget", "60"]
            with patch("sys.stdout", out), patch("sys.stderr", err), patch("sys.argv", argv):
                code = main()
            self.assertEqual(code, 2)  # reached the live-only refusal, past every argparse guard
            self.assertEqual(json_module.loads(out.getvalue())["code"], "SELFTEST_LIVE_REQUIRED")
            with patch("sys.stdout", out), patch("sys.stderr", err), \
                 patch("sys.argv", ["__main__.py", "selftest-run", "--config", str(config), "--plan", str(plan_file)]):
                with self.assertRaises(SystemExit) as refused:
                    main()
            self.assertEqual(refused.exception.code, 2)
            with patch("sys.stdout", out), patch("sys.stderr", err), \
                 patch("sys.argv", ["__main__.py", "environment-confirm", "--config", str(config), "--plan", "x"]):
                with self.assertRaises(SystemExit):
                    main()
            with patch("sys.stdout", out), patch("sys.stderr", err), \
                 patch("sys.argv", ["__main__.py", "job-run", "--config", str(config), "--confirmation", "x"]):
                with self.assertRaises(SystemExit):
                    main()


if __name__ == "__main__":
    unittest.main()
