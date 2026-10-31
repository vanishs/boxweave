"""Explicit local job receipts: bounded argv, durable outcome, output only hashed."""
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from bwmini import host_cli, jobs
from bwmini.domain import Fault
from bwmini.store import Store


class JobsTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.store = Store(Path(self.tmp.name) / "state" / "jobs.sqlite3")
        self.now = [1000.0]
        self.clock = lambda: self.now[0]

    def test_store_migrates_to_job_schema(self):
        with self.store.db() as db:
            self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], 8)
            columns = {r[1] for r in db.execute("PRAGMA table_info(job_runs)")}
        self.assertIn("output_sha256", columns)
        self.assertNotIn("output", columns)

    def test_argv_validation_refuses_unbounded_input(self):
        for argv in (None, [], [""] , [42], ["x"] * 65, ["a" * 4097], ["ok", "bad\narg"],
                     ["a" * 8000, "b" * 8000, "c" * 8000]):
            with self.subTest(argv=argv if not isinstance(argv, list) else f"list({len(argv)})"):
                with self.assertRaises(Fault):
                    jobs.validate(argv)
        self.assertEqual(jobs.validate(["echo", "hi"]), ["echo", "hi"])

    def test_successful_job_is_recorded_without_output_text(self):
        row = jobs.run(self.store, [sys.executable, "-c", "print('hello')"],
                       clock=self.clock,
                       spawner=lambda argv, budget: (subprocess.CompletedProcess(argv, 0), b"hello\n"))
        self.assertEqual((row["state"], row["code"], row["exit_code"]), ("succeeded", "OK", 0))
        self.assertEqual(row["output_sha256"], __import__("hashlib").sha256(b"hello\n").hexdigest())
        self.assertEqual(row["output_bytes"], 6)
        self.assertEqual(json.loads(row["argv"]), [sys.executable, "-c", "print('hello')"])
        self.assertNotIn("output", row)
        rows = jobs.receipts(self.store)
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["id"], row["id"])
        self.assertNotIn("output", rows[0])

    def test_real_subprocess_success_and_failure(self):
        good = jobs.run(self.store, [sys.executable, "-c", "print('ok')"], clock=self.clock)
        self.assertEqual((good["state"], good["exit_code"]), ("succeeded", 0))
        bad = jobs.run(self.store, [sys.executable, "-c", "raise SystemExit(2)"], clock=self.clock)
        self.assertEqual((bad["state"], bad["code"], bad["exit_code"]), ("failed", "JOB_EXITED", 2))

    def test_timeout_and_spawn_failures_are_durable(self):
        slow = jobs.run(self.store, [sys.executable, "-c", "import time; time.sleep(5)"],
                        budget=1, clock=self.clock)
        self.assertEqual((slow["state"], slow["code"]), ("failed", "JOB_TIMEOUT"))
        self.assertIsNone(slow["exit_code"])
        missing = jobs.run(self.store, ["/nonexistent-boxweave-binary"], clock=self.clock)
        self.assertEqual((missing["state"], missing["code"]), ("failed", "JOB_SPAWN_FAILED"))
        self.assertEqual(len(jobs.receipts(self.store)), 2)

    def test_host_job_command_maps_receipt_state_to_exit(self):
        import io
        info = {"application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave"}
        captured = {}
        def fake_observe(argv, timeout, limit, cwd=None):
            captured["argv"], captured["timeout"], captured["cwd"] = argv, timeout, cwd
            return json.dumps({"state": "succeeded", "id": "r1"})
        with (patch.object(host_cli, "public_state", return_value=info),
              patch.object(host_cli.os, "geteuid", return_value=0),
              patch.object(host_cli.pwd, "getpwnam",
                           return_value=type("P", (), {"pw_name": "boxweave", "pw_uid": 1001})()),
              patch.object(host_cli, "observe", side_effect=fake_observe),
              patch("sys.stdout", new_callable=io.StringIO)):
            code = host_cli.job_command(["--budget", "90", "--", "systemctl", "--user", "status", "x"])
        self.assertEqual(code, 0)
        self.assertEqual(captured["timeout"], 150)
        self.assertIn("-m", captured["argv"])
        self.assertIn("job-run", captured["argv"])
        self.assertEqual(captured["cwd"], "/home/boxweave/boxweave/current")
        with (patch.object(host_cli, "public_state", return_value=info),
              patch.object(host_cli.os, "geteuid", return_value=0),
              patch.object(host_cli.pwd, "getpwnam",
                           return_value=type("P", (), {"pw_name": "boxweave", "pw_uid": 1001})()),
              patch.object(host_cli, "observe",
                           return_value=json.dumps({"state": "failed", "code": "JOB_EXITED"})),
              patch("sys.stdout", new_callable=io.StringIO)):
            self.assertEqual(host_cli.job_command(["--", "false"]), 1)

    def test_host_job_command_refuses_without_root_or_separator(self):
        import io
        info = {"application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave"}
        with patch.object(host_cli, "public_state", return_value=info), \
             patch.object(host_cli.os, "geteuid", return_value=1000), \
             patch("sys.stderr", new_callable=io.StringIO):
            self.assertEqual(host_cli.job_command(["--", "echo", "hi"]), 2)
        with patch.object(host_cli, "public_state", return_value=info), \
             patch.object(host_cli.os, "geteuid", return_value=0), \
             patch("sys.stderr", new_callable=io.StringIO):
            self.assertEqual(host_cli.job_command(["echo", "hi"]), 2)
            self.assertEqual(host_cli.job_command(["--budget", "x", "--", "echo"]), 2)


if __name__ == "__main__":
    unittest.main()
