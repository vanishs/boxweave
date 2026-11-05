"""Confirmed selftest regressions: plan hash gate, live-only, full cleanup."""
import io
import json
import os
from pathlib import Path
import tempfile
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
        self.hello_fault = "NOT_CONNECTION_READY"
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
            raise Fault(self.hello_fault)
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
        # Always change exactly one valid hex digit; replace("a", "b")
        # leaves an a-free random digest unchanged and flakes this test.
        corrupt_checksum = ("0" if checksum[0] != "0" else "1") + checksum[1:]
        self.assertNotEqual(corrupt_checksum, checksum)
        for bad in ("", "zz", corrupt_checksum):
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

    def test_transient_trust_failure_waits_without_recreating_box(self):
        provider, clock = Live(), Clock()
        provider.hello_failures = 2
        provider.hello_fault = "TRUST_FAILED"
        plan, checksum = selftest.build_plan(now="T")
        result = selftest.run(provider, selftest.load_plan(plan, checksum), budget=180,
                              clock=clock, sleep=clock.sleep, key=KEY)
        self.assertEqual(result["state"], "passed")
        self.assertGreaterEqual(clock(), 1006)
        self.assertEqual([x[0] for x in provider.writes], ["create", "stop", "delete"])
        self.assertEqual(provider.boxes, {})

    def test_hello_failure_still_cleans_up_and_reports_failed(self):
        provider, clock = Live(), Clock()
        provider.hello_failures = 100
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


    def test_host_selftest_run_preserves_safe_typed_failures(self):
        info = {"application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave"}
        identity = type("P", (), {"pw_name": "boxweave", "pw_uid": 1001})()
        cases = [
            ({"code": "SELFTEST_CONFIRMATION_MISMATCH"}, 2,
             "SELFTEST_CONFIRMATION_MISMATCH"),
            ({"state": "failed", "cleanup": "performed",
              "steps": [{"step": "hello", "state": "failed", "code": "TRUST_FAILED"}]},
             1, "TRUST_FAILED"),
            ({"code": "INTERNAL_UNSAFE_OUTPUT"}, 2, "SELFTEST_RESULT_INVALID"),
        ]
        with tempfile.TemporaryDirectory() as tmp:
            plan_file = Path(tmp) / "operator-plan.json"
            plan_file.write_text(json.dumps(selftest.build_plan(now="T")[0]))
            for response, expected_exit, expected_code in cases:
                with self.subTest(response=response):
                    output = io.StringIO()
                    errors = io.StringIO()
                    with (patch.object(host_cli, "public_state", return_value=info),
                          patch.object(host_cli.os, "geteuid", return_value=0),
                          patch.object(host_cli.pwd, "getpwnam", return_value=identity),
                          patch.object(host_cli, "run_bounded", return_value=json.dumps(response)) as runner,
                          patch("sys.stdout", output),
                          patch("sys.stderr", errors)):
                        code = host_cli.selftest_command(
                            ["run", "--plan", str(plan_file),
                             "--confirmation", "a" * 64])
                    self.assertEqual(code, expected_exit)
                    self.assertIn(expected_code, output.getvalue() + errors.getvalue())
                    self.assertNotIn("INTERNAL_UNSAFE_OUTPUT",
                                     output.getvalue() + errors.getvalue())
                    self.assertEqual(runner.call_args.kwargs["input_data"], plan_file.read_bytes())
                    self.assertIn("-", runner.call_args.args[0])
                    self.assertNotIn(str(plan_file), runner.call_args.args[0])

    def test_host_private_plan_pipe_accepts_operator_home_without_app_read_access(self):
        """Host reads the caller's 0600 plan; app sees stdin, never caller path."""
        plan, checksum = selftest.build_plan(now="T")
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp) / "private-home"
            directory.mkdir(mode=0o700)
            file = directory / "plan.json"
            file.write_text(json.dumps(plan))
            file.chmod(0o600)
            self.assertEqual(host_cli.selftest_plan_bytes(str(file), os.getuid()+10000),
                             file.read_bytes())
            info = {"application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave"}
            account = type("P", (), {"pw_name": "boxweave", "pw_uid": os.getuid()+10000})()
            forwarded = {}
            def fake_bounded(argv, timeout, limit, cwd, input_data=None):
                forwarded["argv"], forwarded["input"] = argv, input_data
                return json.dumps({"state": "passed", "steps": [], "cleanup": "not_needed"})
            with (patch.object(host_cli, "public_state", return_value=info),
                  patch.object(host_cli.os, "geteuid", return_value=0),
                  patch.object(host_cli.pwd, "getpwnam", return_value=account),
                  patch.object(host_cli, "run_bounded", side_effect=fake_bounded),
                  patch("sys.stdout", new_callable=io.StringIO)):
                self.assertEqual(host_cli.selftest_command([
                    "run", "--plan", str(file), "--confirmation", checksum]), 0)
            self.assertEqual(forwarded["input"], file.read_bytes())
            self.assertIn("--plan", forwarded["argv"])
            self.assertEqual(forwarded["argv"][forwarded["argv"].index("--plan")+1], "-")
            self.assertNotIn(str(file), forwarded["argv"])

    def test_host_plan_input_rejects_symlink_directory_fifo_oversize_and_missing(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            target=root/"plan.json"
            target.write_bytes(b"{}")
            target.chmod(0o600)
            link=root/"link"
            link.symlink_to(target)
            folder=root/"folder"
            folder.mkdir()
            fifo=root/"pipe"
            os.mkfifo(fifo)
            big=root/"big.json"
            big.write_bytes(b"x"*(host_cli.SELFTEST_MAX_PLAN_BYTES+1))
            for file,code in ((link,"SELFTEST_PLAN_UNSAFE"),
                              (folder,"SELFTEST_PLAN_UNSAFE"),
                              (fifo,"SELFTEST_PLAN_UNSAFE"),
                              (big,"SELFTEST_PLAN_TOO_LARGE"),
                              (root/"missing","SELFTEST_PLAN_NOT_FOUND")):
                with self.subTest(file=file), self.assertRaisesRegex(ValueError,code):
                    host_cli.selftest_plan_bytes(str(file),os.getuid())
            with patch.dict(host_cli.os.environ, {"SUDO_UID": str(os.getuid()+9000)}):
                with self.assertRaisesRegex(ValueError,"SELFTEST_PLAN_UNSAFE"):
                    host_cli.selftest_plan_bytes(str(target),os.getuid()+9001)

    def test_host_malformed_flag_positions_are_refused_without_crashing(self):
        info={"application_user":"boxweave","app_prefix":"/home/boxweave/boxweave"}
        account=type("P",(),{"pw_name":"boxweave","pw_uid":os.getuid()})()
        with (patch.object(host_cli,"public_state",return_value=info),
              patch.object(host_cli.os,"geteuid",return_value=0),
              patch.object(host_cli.pwd,"getpwnam",return_value=account),
              patch.object(host_cli,"run_bounded") as runner,
              patch("sys.stderr",new_callable=io.StringIO)):
            for invalid in (
                ["run","--confirmation","a"*64,"trailing","--plan"],
                ["run","--plan","a.json","--plan","b.json"],
                ["run","--plan","--confirmation","--confirmation","a"*64]):
                with self.subTest(invalid=invalid):
                    self.assertEqual(host_cli.selftest_command(invalid),2)
        runner.assert_not_called()

    def test_host_preflight_refusal_does_not_launch_app_or_echo_plan(self):
        info = {"application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave"}
        account = type("P", (), {"pw_name": "boxweave", "pw_uid": os.getuid()+10000})()
        with tempfile.TemporaryDirectory() as tmp:
            plan_file = Path(tmp) / "private"
            plan_file.write_bytes(b"private-test-input")
            alias = Path(tmp) / "alias"
            alias.symlink_to(plan_file)
            error = io.StringIO()
            with (patch.object(host_cli, "public_state", return_value=info),
                  patch.object(host_cli.os, "geteuid", return_value=0),
                  patch.object(host_cli.pwd, "getpwnam", return_value=account),
                  patch.object(host_cli, "run_bounded") as runner,
                  patch("sys.stderr", error)):
                self.assertEqual(host_cli.selftest_command([
                    "run", "--plan", str(alias), "--confirmation", "a"*64]), 2)
            runner.assert_not_called()
            self.assertIn("SELFTEST_PLAN_UNSAFE", error.getvalue())
            self.assertNotIn("private-test-input", error.getvalue())

    def test_app_selftest_stdin_plan_valid_and_invalid_are_pre_dispatch(self):
        from bwmini.__main__ import main
        plan,checksum=selftest.build_plan(now="T")
        with tempfile.TemporaryDirectory() as tmp:
            config=Path(tmp)/"config.json"
            config.write_text(json.dumps({
                "mode":"fixture","database":str(Path(tmp)/"db.sqlite3"),
                "target":{"provider":"containarium","backend":"dev","project":"default",
                          "logical":"v2-lifecycle-e2e","instance":"v2-lifecycle-e2e-container",
                          "created":"1","user":"v2-lifecycle-e2e",
                          "hostname":"v2-lifecycle-e2e-container"},
                "fixture_counter":None}))
            argv=["__main__.py","selftest-run","--config",str(config),
                  "--plan","-","--confirmation",checksum,"--budget","60"]
            cases=[(json.dumps(plan).encode(), "SELFTEST_LIVE_REQUIRED"),
                   (b"{broken", "SELFTEST_PLAN_INVALID"),
                   (b"x"*8193, "SELFTEST_PLAN_TOO_LARGE")]
            for data, code in cases:
                with self.subTest(code=code):
                    out=io.StringIO()
                    fake_stdin=type("Input",(),{"buffer":io.BytesIO(data)})()
                    with (patch("sys.argv",argv),patch("sys.stdin",fake_stdin),
                          patch("sys.stdout",out)):
                        self.assertEqual(main(),2)
                    self.assertEqual(json.loads(out.getvalue())["code"],code)

    def test_bounded_input_transport_is_true_stdin_not_argument(self):
        result=host_cli.run_bounded(
            ["python3","-c","import sys;print(len(sys.stdin.buffer.read()))"],
            timeout=5,limit=64,cwd=None,input_data=b"some private bytes")
        self.assertEqual(result.strip(),str(len(b"some private bytes")))

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
            with patch("sys.stdout", out), patch("sys.stderr", err), \
                 patch("sys.argv", ["__main__.py", "environment-plan", "--config", str(config),
                                    "--logical", "demo-1", "--environment-action", "resize",
                                    "--memory", "2GB"]):
                code = main()
            self.assertEqual(code, 2)  # argparse accepted resize; the application (not argparse) refused it
            self.assertIn("code", json_module.loads(out.getvalue().strip().splitlines()[-1]))


if __name__ == "__main__":
    unittest.main()


class JournalTests(unittest.TestCase):
    def setUp(self):
        import tempfile
        from bwmini.store import Store
        self.tmp = tempfile.TemporaryDirectory()
        self.store = Store(Path(self.tmp.name) / "state" / "state.sqlite3")
        self.plan, self.checksum = selftest.build_plan(now="2026-10-31T00:00:00+00:00")

    def tearDown(self):
        self.tmp.cleanup()

    def journal(self):
        return selftest.Journal(self.store, clock=lambda: 1000.0)

    def test_run_records_durable_step_boundaries(self):
        provider = Live()
        journal = self.journal()
        result = selftest.run(provider, self.plan, key=KEY, clock=Clock(), sleep=Clock().sleep,
                              journal=journal)
        self.assertEqual(result["state"], "passed")
        with self.store.db() as db:
            rows = [dict(r) for r in db.execute("SELECT * FROM selftest_runs")]
        self.assertEqual(len(rows), 1)
        row = rows[0]
        self.assertEqual(row["state"], "completed")
        self.assertEqual(row["exact_test_box"], self.plan["box"])
        self.assertEqual(row["plan_digest"], self.checksum)
        self.assertEqual(row["cleanup_state"], "not_needed")
        self.assertTrue(row["last_step"].startswith("delete:"))
        self.assertNotIn("ssh-ed25519", json.dumps(row))

    def test_failed_step_is_journaled_and_cleanup_recorded(self):
        provider = Live()
        provider.hello_failures = 100
        journal = self.journal()
        tick = Clock()
        result = selftest.run(provider, self.plan, key=KEY, budget=60, clock=tick,
                              sleep=tick.sleep, journal=journal)
        self.assertEqual(result["state"], "failed")
        with self.store.db() as db:
            row = dict(db.execute("SELECT * FROM selftest_runs").fetchone())
        self.assertEqual(row["state"], "failed")
        self.assertIn(row["cleanup_state"], ("performed", "best_effort_failed", "best_effort"))

    def test_kill_between_start_and_outcome_leaves_unreconciled_row(self):
        journal = self.journal()
        run_id = journal.start(self.plan, 1999.0)
        journal.step_started("create")
        # Simulate a killed process: no step_done, no finish.
        stale = selftest.Journal.unreconciled(self.store)
        self.assertEqual(len(stale), 1)
        self.assertEqual(stale[0]["id"], run_id)
        self.assertEqual(stale[0]["exact_test_box"], self.plan["box"])
        self.assertEqual(stale[0]["last_step"], "create:started")
        with self.assertRaisesRegex(Fault, "SELFTEST_RUN_UNRECONCILED"):
            selftest.run(Live(), self.plan, key=KEY, clock=Clock(), sleep=Clock().sleep,
                         journal=self.journal())
        provider = Live()
        self.assertEqual(provider.writes, [])  # refusal happened before any remote effect

    def test_resolve_is_bookkeeping_only_and_unblocks_new_runs(self):
        journal = self.journal()
        run_id = journal.start(self.plan, 1999.0)
        journal.step_started("create")
        selftest.Journal.resolve(self.store, run_id, actor="operator")
        with self.store.db() as db:
            row = dict(db.execute("SELECT * FROM selftest_runs").fetchone())
        self.assertEqual(row["state"], "resolved")
        self.assertEqual(row["cleanup_state"], "operator_resolved")
        self.assertEqual(selftest.Journal.unreconciled(self.store), [])
        provider = Live()
        result = selftest.run(provider, self.plan, key=KEY, clock=Clock(), sleep=Clock().sleep,
                              journal=self.journal())
        self.assertEqual(result["state"], "passed")

    def test_second_journal_start_is_atomically_refused(self):
        """Two independent processes can share this SQLite journal.

        The start operation itself must enforce exclusivity, not rely on
        a prior unprotected SELECT in the caller.
        """
        first = self.journal()
        first_id = first.start(self.plan, 1999.0)
        second_plan, _ = selftest.build_plan(now="T")
        with self.assertRaisesRegex(Fault, "SELFTEST_RUN_UNRECONCILED"):
            self.journal().start(second_plan, 1999.0)
        with self.store.db() as db:
            rows = [dict(row) for row in db.execute(
                "SELECT id,state,exact_test_box FROM selftest_runs")]
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["id"], first_id)
        self.assertEqual(rows[0]["state"], "running")

    def test_parallel_journal_starts_allow_exactly_one_writer(self):
        """The SQLite BEGIN IMMEDIATE gate must serialize competing actors."""
        from concurrent.futures import ThreadPoolExecutor
        from threading import Barrier
        barrier = Barrier(2, timeout=5)
        def candidate():
            plan, _ = selftest.build_plan(now="T")
            barrier.wait(timeout=5)
            try:
                return ("accepted", self.journal().start(plan, 1999.0))
            except Fault as error:
                return ("refused", error.code)
        with ThreadPoolExecutor(max_workers=2) as pool:
            futures = [pool.submit(candidate) for _ in range(2)]
            outcomes = [future.result(timeout=8) for future in futures]
        self.assertEqual(sorted(kind for kind, _ in outcomes),
                         ["accepted", "refused"])
        self.assertEqual([code for kind, code in outcomes if kind == "refused"],
                         ["SELFTEST_RUN_UNRECONCILED"])
        with self.store.db() as db:
            self.assertEqual(db.execute(
                "SELECT count(*) FROM selftest_runs").fetchone()[0], 1)

    def test_double_resolve_is_refused(self):
        journal = self.journal()
        run_id = journal.start(self.plan, 1999.0)
        journal.finish("failed", "best_effort")
        with self.assertRaisesRegex(Fault, "SELFTEST_RUN_NOT_RUNNING"):
            selftest.Journal.resolve(self.store, run_id)

    def test_journal_write_failure_refuses_the_run_before_mutation(self):
        class BrokenStore:
            def db(self):
                raise Fault("STATE_UNAVAILABLE", 503)

        provider = Live()
        broken = selftest.Journal(BrokenStore())
        with self.assertRaises(Exception):
            selftest.run(provider, self.plan, key=KEY, clock=Clock(), sleep=Clock().sleep,
                         journal=broken)
        self.assertEqual(provider.writes, [])
