"""D2-a diagnostic report: app-owned read-only reader and secure host export."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import sqlite3
import tempfile
import unittest
from unittest.mock import patch

from test_slice import Base, TARGET

from bwmini import diagnostics, host_cli
from bwmini.domain import Fault


def descriptor():
    return {"schema_version": 1, "phase": "ready", "version": "0.21.5",
            "application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave",
            "web_port": 18640}


def reader_args(config, selector):
    return ["python", "-m", "bwmini.diagnostics", "read-report",
            "--config", str(config), "--selector", json.dumps(selector)]


class ReaderTests(Base):
    def setUp(self):
        super().setUp()
        self.config = self.root / "config.json"
        self.config.write_text(json.dumps({"mode": "mcp", "database": str(self.store.path)}))

    def test_operation_report_has_allowlisted_receipt_trace_and_events(self):
        op = self.operation()
        self.s.work_one()
        before = hashlib.sha256(self.store.path.read_bytes()).hexdigest()
        payload = diagnostics.read_report(self.config, {
            "kind": "operation", "operation_kind": "hello",
            "operation_id": op["id"], "actor": "operator"})
        after = hashlib.sha256(self.store.path.read_bytes()).hexdigest()
        self.assertEqual(before, after)  # read-only proof: byte-identical state
        self.assertTrue(payload["found"])
        self.assertEqual(set(payload["receipt"]), {"id", "project", "state", "code", "created", "deadline"})
        self.assertEqual(payload["receipt"]["state"], "succeeded")
        self.assertNotIn("result", payload["receipt"])
        self.assertIsNotNone(payload["trace_id"])
        kinds = [event["event_type"] for event in payload["events"]]
        self.assertEqual(kinds[0], "claimed")
        self.assertEqual(kinds[-1], "observed")
        self.assertEqual(payload["events"][-1]["code"], "OK")

    def test_wrong_actor_and_unknown_operation_do_not_disclose(self):
        op = self.operation()
        for actor, oid in (("someone-else", op["id"]), ("operator", "missing-op")):
            with self.subTest(actor=actor):
                payload = diagnostics.read_report(self.config, {
                    "kind": "operation", "operation_kind": "hello",
                    "operation_id": oid, "actor": actor})
                self.assertFalse(payload["found"])
                self.assertIn("receipt_not_found_for_actor", payload["omissions"])

    def test_trace_selector_returns_linked_events_without_receipt_content(self):
        op = self.operation()
        self.s.work_one()
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        payload = diagnostics.read_report(self.config, {"kind": "trace", "trace_id": trace})
        self.assertTrue(payload["found"])
        self.assertEqual(payload["receipt"]["operation_kind"], "hello")
        self.assertGreaterEqual(payload["event_count"], 2)
        missing = diagnostics.read_report(self.config, {"kind": "trace", "trace_id": "b" * 32})
        self.assertFalse(missing["found"])

    def test_window_selector_is_time_bounded(self):
        op = self.operation()
        self.s.work_one()
        payload = diagnostics.read_report(self.config, {
            "kind": "window", "since": "1970-01-01T00:00:00Z", "until": "9999-01-01T00:00:00Z"})
        self.assertTrue(payload["found"])
        self.assertGreaterEqual(payload["event_count"], 2)
        self.assertIn("window_receipts_not_joined", payload["omissions"])
        empty = diagnostics.read_report(self.config, {
            "kind": "window", "since": "1970-01-01T00:00:00Z", "until": "1971-01-01T00:00:00Z"})
        self.assertEqual(empty["event_count"], 0)

    def test_invalid_selectors_fail_closed(self):
        for selector in ({"kind": "operation", "operation_kind": "selftest",
                          "operation_id": "x", "actor": "operator"},
                         {"kind": "operation", "operation_kind": "hello",
                          "operation_id": "../escape", "actor": "operator"},
                         {"kind": "trace", "trace_id": "nothex"},
                         {"kind": "window", "since": "yesterday", "until": "tomorrow"},
                         {"kind": "unknown"}):
            with self.subTest(selector=selector), self.assertRaises(Fault):
                diagnostics.read_report(self.config, selector)

    def test_pre_migration_receipt_exports_with_explicit_gap(self):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("INSERT INTO projects VALUES('p-pre','legacy','{}','d','/home/u',1)")
            db.execute("INSERT INTO operations VALUES('op-pre','p-pre',1,'d','operator','succeeded',1,2,NULL,'OK')")
        payload = diagnostics.read_report(self.config, {
            "kind": "operation", "operation_kind": "hello",
            "operation_id": "op-pre", "actor": "operator"})
        self.assertTrue(payload["found"])
        self.assertEqual(payload["events"], [])
        self.assertIn("trace_unavailable_pre_migration", payload["omissions"])

    def test_environment_receipt_excludes_spec_and_target_content(self):
        spec = json.dumps({"ssh_public_key": "ssh-ed25519 SYNTHETIC", "cpu": "2"})
        target = json.dumps(TARGET.data())
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("INSERT INTO environment_operations VALUES('env-1','operator','create','fixture-box',?,?,'x',"
                       "'queued','queued',1,9,1,0,NULL,NULL,'fixture')", (spec, target))
            db.execute("INSERT INTO diagnostic_trace_links VALUES('operator','environment','env-1',?, 'now')",
                       ("c" * 32,))
            db.execute("INSERT INTO diagnostic_events (event_id,trace_id,span_id,actor_scope,operation_kind,"
                       "operation_id,time_utc,component,phase,event_type,outcome,code,capture) "
                       "VALUES('e1',?, 's1','operator','environment','env-1','now','lifecycle','claimed',"
                       "'claimed','passed','CLAIMED','captured')", ("c" * 32,))
        payload = diagnostics.read_report(self.config, {
            "kind": "operation", "operation_kind": "environment",
            "operation_id": "env-1", "actor": "operator"})
        self.assertTrue(payload["found"])
        self.assertNotIn("spec", payload["receipt"])
        self.assertNotIn("target", payload["receipt"])
        rendered = json.dumps(payload)
        self.assertNotIn("SYNTHETIC", rendered)
        self.assertNotIn("fixture-box-container", rendered)

    def test_unsafe_state_file_fails_closed(self):
        private = self.root / "alt.sqlite3"
        private.write_bytes(b"not a database")
        os.chmod(private, 0o644)
        bad = self.root / "bad.json"
        bad.write_text(json.dumps({"mode": "mcp", "database": str(private)}))
        with self.assertRaisesRegex(Fault, "STATE_FILE_NOT_PRIVATE"):
            diagnostics.read_report(bad, {"kind": "trace", "trace_id": "c" * 32})


class FakeRecord:
    pw_uid = 18640
    pw_name = "boxweave"


READER_PAYLOAD = {
    "schema_version": 1, "reader": "app-owned-diagnostics", "found": True,
    "receipt": {"id": "op-1", "state": "succeeded", "code": "OK", "created": 1, "deadline": 2,
                "project": "p1"},
    "trace_id": "a" * 32,
    "events": [{"seq": 1, "event_id": "e1", "trace_id": "a" * 32, "span_id": "s1",
                "parent_span_id": None, "actor_scope": "operator", "operation_kind": "hello",
                "operation_id": "op-1", "time_utc": "2026-10-31T00:00:00.000Z", "elapsed_ms": 5,
                "component": "worker", "phase": "claimed", "event_type": "claimed",
                "outcome": "passed", "code": "CLAIMED", "capture": "captured",
                "target_ref": None}],
    "omissions": ["receipt_detail_fields_not_exported"], "event_count": 1,
}


class ReportCommandTests(unittest.TestCase):
    def report(self, output, root=True, reader="observed", extra=None):
        argv = ["report", "--kind", "hello", "--operation", "op-1", "--output", output]
        argv += (extra or [])
        payload = json.dumps(READER_PAYLOAD) if reader == "observed" else (
            json.dumps({"schema_version": 1, "found": False, "events": [], "omissions": ["x"]})
            if reader == "notfound" else None)
        with patch.object(host_cli.os, "geteuid", return_value=0 if root else 1000), \
             patch.object(host_cli, "web_observation",
                          return_value={"reachable": True, "http_status": 401, "authenticated": False}), \
             patch.object(host_cli, "service_observation",
                          side_effect=lambda name: {"LoadState": "loaded", "ActiveState": "active",
                                                    "UnitFileState": "enabled"}), \
             patch.object(host_cli.pwd, "getpwnam", return_value=FakeRecord()), \
             patch.object(host_cli, "run_bounded", return_value=payload) as execute, \
             patch.object(host_cli, "public_state", return_value=descriptor()), \
             patch("sys.stdout", new_callable=io.StringIO) as out, \
             patch("sys.stderr", new_callable=io.StringIO) as err:
            code = host_cli.main(argv)
        return code, out.getvalue(), err.getvalue(), execute

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.parent = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def test_full_export_is_private_complete_and_bounded(self):
        code, out, err, execute = self.report(str(self.parent / "report-1"))
        self.assertEqual(code, 0, err)
        report = self.parent / "report-1"
        self.assertEqual(sorted(item.name for item in report.iterdir()),
                         ["README.txt", "checks.json", "layers.json", "manifest.json",
                          "omissions.json", "operation.json", "timeline.jsonl"])
        self.assertEqual(oct(os.stat(report).st_mode & 0o777), oct(0o700))
        self.assertTrue(all(oct(os.stat(item).st_mode & 0o777) == oct(0o600)
                            for item in report.iterdir()))
        manifest = json.loads((report / "manifest.json").read_text())
        self.assertEqual(manifest["collection_status"], "complete")
        self.assertEqual(manifest["app_reader"], "observed")
        self.assertEqual(manifest["selector"]["operation_id"], "op-1")
        timeline = [json.loads(line) for line in
                    (report / "timeline.jsonl").read_text().splitlines() if line]
        self.assertEqual(len(timeline), 1)
        self.assertTrue(set(timeline[0]) <= set(host_cli.REPORT_EVENT_COLUMNS))
        self.assertIn("receipt_detail_fields_not_exported",
                      json.loads((report / "omissions.json").read_text())["omissions"])
        self.assertIn("not_verified", (report / "layers.json").read_text())
        argv = execute.call_args.args[0]
        self.assertIn("read-report", argv)
        self.assertIn("bwmini.diagnostics", argv)
        self.assertEqual(json.loads(out)["state"], "exported")

    def test_unprivileged_report_is_partial_without_reader(self):
        code, out, err, execute = self.report(str(self.parent / "report-2"), root=False)
        self.assertEqual(code, 2)
        execute.assert_not_called()
        report = self.parent / "report-2"
        manifest = json.loads((report / "manifest.json").read_text())
        self.assertEqual(manifest["collection_status"], "partial")
        self.assertEqual(manifest["diagnostic_conclusion"], "not_verified")
        self.assertIn("app_reader_requires_admin",
                      json.loads((report / "omissions.json").read_text())["omissions"])

    def test_reader_unavailable_is_partial_never_fatal(self):
        code, out, err, _ = self.report(str(self.parent / "report-3"), reader="down")
        self.assertEqual(code, 2)
        self.assertIn("app_reader_unavailable",
                      json.loads((self.parent / "report-3" / "omissions.json").read_text())["omissions"])

    def test_unknown_source_refuses_without_writing(self):
        code, out, err, _ = self.report(str(self.parent / "report-4"), reader="notfound")
        self.assertEqual(code, 1)
        self.assertFalse((self.parent / "report-4").exists())
        self.assertIn("REPORT_SOURCE_NOT_FOUND", err)

    def test_existing_target_is_never_overwritten(self):
        target = self.parent / "report-5"
        target.mkdir()
        (target / "sentinel.txt").write_text("keep")
        code, out, err, _ = self.report(str(target))
        self.assertEqual(code, 1)
        self.assertIn("REPORT_TARGET_EXISTS", err)
        self.assertEqual((target / "sentinel.txt").read_text(), "keep")
        self.assertFalse(any(item.name.startswith(".boxweave-report-")
                             for item in self.parent.iterdir()))

    def test_symlink_parent_refuses(self):
        real = self.parent / "real-dir"
        real.mkdir()
        link = self.parent / "link-dir"
        link.symlink_to(real)
        code, out, err, _ = self.report(str(link / "report-6"))
        self.assertEqual(code, 2)
        self.assertIn("REPORT_PARENT_UNSAFE", err)
        self.assertEqual(list(real.iterdir()), [])

    def test_invalid_selectors_reject_before_any_write(self):
        cases = ([["--trace", "nothex"]],
                 [["--trace", "a" * 32], ["--kind", "hello"]],
                 [["--since", "2026-10-31T00:00:00Z"]],
                 [["--since", "bad"], ["--until", "2026-10-31T01:00:00Z"]],
                 [["--since", "2026-10-31T00:00:00Z"], ["--until", "2026-11-01T00:00:01Z"]],
                 [["--kind", "selftest"], ["--operation", "x"]])
        for extra in cases:
            with self.subTest(extra=extra), \
                 patch.object(host_cli, "write_report_directory") as writer:
                argv_extra = [value for pair in extra for value in pair]
                code, out, err, _ = self.report(str(self.parent / "unused"), extra=argv_extra)
            self.assertEqual(code, 2, err)
            writer.assert_not_called()
            self.assertFalse((self.parent / "unused").exists())

    def test_missing_selector_is_refused(self):
        with patch.object(host_cli.os, "geteuid", return_value=1000), \
             patch.object(host_cli, "public_state", return_value=descriptor()), \
             patch("sys.stdout", new_callable=io.StringIO), \
             patch("sys.stderr", new_callable=io.StringIO) as err:
            code = host_cli.main(["report", "--output", str(self.parent / "unused")])
        self.assertEqual(code, 2)
        self.assertIn("SELECTOR_REQUIRED", err.getvalue())
        self.assertFalse((self.parent / "unused").exists())

    def test_window_report_exports_events_only(self):
        argv = ["report", "--since", "2026-10-31T00:00:00Z", "--until", "2026-10-31T01:00:00Z",
                "--output", str(self.parent / "report-7")]
        with patch.object(host_cli.os, "geteuid", return_value=1000), \
             patch.object(host_cli, "web_observation",
                          return_value={"reachable": True, "http_status": 401, "authenticated": False}), \
             patch.object(host_cli, "service_observation",
                          side_effect=lambda name: {"LoadState": "loaded", "ActiveState": "active",
                                                    "UnitFileState": "enabled"}), \
             patch.object(host_cli.pwd, "getpwnam", return_value=FakeRecord()), \
             patch.object(host_cli, "public_state", return_value=descriptor()):
            code = host_cli.main(argv)
        self.assertEqual(code, 2)
        manifest = json.loads((self.parent / "report-7" / "manifest.json").read_text())
        self.assertEqual(manifest["selector"]["kind"], "window")


if __name__ == "__main__":
    unittest.main()
