"""D1 diagnostic evidence: typed events, v9 migration, transaction links, noninterference."""
import json
import os
import sqlite3
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from test_slice import Base, TARGET, PASSWORD, HASH

from bwmini import diagnostics
from bwmini.domain import Fault
from bwmini.store import Store
from bwmini.service import Service
from bwmini.provider import Fixture


def events(store, trace_id):
    return diagnostics.events_for_trace(store, trace_id, limit=1000)


def codes(entries):
    return [(e["event_type"], e["outcome"], e["code"]) for e in entries]


class TypeTests(unittest.TestCase):
    def test_build_validates_enums_fields_and_shapes(self):
        event = diagnostics.build(component="worker", phase="claimed", event_type="claimed",
                                  outcome="passed", code="CLAIMED", trace_id="a" * 32)
        self.assertEqual(event.capture, "captured")
        self.assertIsNone(event.parent_span_id)
        for kwargs in (dict(component="nope", phase="p", event_type="claimed", outcome="passed",
                            code="C", trace_id="a" * 32),
                       dict(component="worker", phase="p", event_type="claimed", outcome="passed",
                            code="lowercase", trace_id="a" * 32),
                       dict(component="worker", phase="p", event_type="claimed", outcome="passed",
                            code="C", trace_id="short"),
                       dict(component="worker", phase="p" * 65, event_type="claimed", outcome="passed",
                            code="C", trace_id="a" * 32),
                       dict(component="worker", phase="p", event_type="claimed", outcome="passed",
                            code="C", trace_id="a" * 32, target_ref="x" * 129),
                       dict(component="worker", phase="p", event_type="claimed", outcome="passed",
                            code="C", trace_id="a" * 32, target_ref="bad\ncontrol"),
                       dict(component="worker", phase="p", event_type="claimed", outcome="passed",
                            code="C", trace_id="a" * 32, elapsed_ms=-1),
                       dict(component="worker", phase="p", event_type="claimed", outcome="passed",
                            code="C", trace_id="a" * 32, operation_kind="unknown_kind")):
            with self.subTest(kwargs=sorted(kwargs)), self.assertRaises(ValueError):
                diagnostics.build(**kwargs)

    def test_error_class_never_carries_message_content(self):
        fault = Fault("PROVIDER_REJECTED")
        self.assertEqual(diagnostics.error_class(fault), "UNCLASSIFIED_EXCEPTION")
        diagnostics.attach(fault, "TOOL_IS_ERROR")
        self.assertEqual(diagnostics.error_class(fault), "TOOL_IS_ERROR")
        self.assertEqual(fault.code, "PROVIDER_REJECTED")
        diagnostics.attach(fault, "NOT_IN_ENUM")
        self.assertEqual(diagnostics.error_class(fault), "TOOL_IS_ERROR")
        self.assertEqual(diagnostics.error_class(ValueError("secret value")), "UNCLASSIFIED_EXCEPTION")

    def test_mcp_error_shapes_are_privately_classified_public_code_unchanged(self):
        from bwmini.mcp import MCP
        with patch.object(MCP, "__init__", lambda self, argv, timeout=30: None), \
             patch.object(MCP, "send", lambda self, obj: None):
            client = MCP([])
            client.deadline = 1
            client.ident = 0
            client.total = 0
            client.buffer = b'{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"rejected"}}\n'
            with self.assertRaises(Fault) as caught:
                client.call("tools/call", {"name": "list_containers", "arguments": {}})
            self.assertEqual(caught.exception.code, "PROVIDER_REJECTED")
            self.assertEqual(getattr(caught.exception, "error_class", None), "UPSTREAM_JSONRPC_REJECTION")
            client.buffer = b'{"jsonrpc":"2.0","id":2,"result":{"isError":true,"content":[]}}\n'
            client.ident = 1
            with self.assertRaises(Fault) as caught:
                client.tool("list_containers", {})
            self.assertEqual(caught.exception.code, "PROVIDER_REJECTED")
            self.assertEqual(getattr(caught.exception, "error_class", None), "TOOL_IS_ERROR")
            client.buffer = b'garbage bytes \xff\n'
            client.ident = 2
            with self.assertRaises(Fault) as caught:
                client.call("tools/call", {"name": "list_containers", "arguments": {}})
            self.assertEqual(caught.exception.code, "MCP_INVALID")
            self.assertEqual(getattr(caught.exception, "error_class", None), "MALFORMED_PROTOCOL")


class MigrationTests(unittest.TestCase):
    def test_d1_t01_v8_database_upgrades_to_v9_preserving_rows(self):
        from bwmini.lifecycle import SCHEMA as LIFECYCLE
        from bwmini.access import SCHEMA as ACCESS
        import bwmini.models as models
        import bwmini.jobs as jobs
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "state.sqlite3"
            db = sqlite3.connect(path)
            from bwmini.store import SCHEMA as CORE, GRAPH_SCHEMA, OBSERVATION_SCHEMA
            for statement in (CORE, GRAPH_SCHEMA, OBSERVATION_SCHEMA, LIFECYCLE, models.SCHEMA,
                              ACCESS, jobs.SCHEMA):
                for part in statement.split(";"):
                    if part.strip():
                        db.execute(part)
            db.execute("CREATE TABLE project_unbindings (project TEXT PRIMARY KEY REFERENCES projects(id))")
            db.execute("INSERT INTO projects VALUES('p1','proj','{}','digest','/home/u',1)")
            db.execute("INSERT INTO operations VALUES('op-old','p1',1,'digest','operator','succeeded',5,6,NULL,'OK')")
            db.execute("PRAGMA user_version=8")
            db.commit()
            db.close()
            os.chmod(path, 0o600)
            store = Store(path)
            with store.db() as check:
                version = check.execute("PRAGMA user_version").fetchone()[0]
                self.assertEqual(version, 10)
                self.assertEqual(check.execute("SELECT count(*) FROM operations").fetchone()[0], 1)
                self.assertIsNotNone(check.execute(
                    "SELECT sql FROM sqlite_master WHERE name='diagnostic_trace_links'").fetchone())
                self.assertIsNotNone(check.execute(
                    "SELECT sql FROM sqlite_master WHERE name='diagnostic_events_time_order'").fetchone())
                self.assertIsNotNone(check.execute(
                    "SELECT sql FROM sqlite_master WHERE name='selftest_runs'").fetchone())
            self.assertEqual(diagnostics.trace_for_operation(store, "operator", "hello", "op-old"), None)

    def test_d1_t02_unsupported_future_schema_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "state.sqlite3"
            db = sqlite3.connect(path)
            db.execute("CREATE TABLE forward(value TEXT)")
            db.execute("PRAGMA user_version=11")
            db.commit()
            db.close()
            os.chmod(path, 0o600)
            with self.assertRaisesRegex(Fault, "SCHEMA_UNSUPPORTED"):
                Store(path)


class ServiceDiagnosticsTests(Base):
    def test_confirm_links_trace_in_the_same_transaction(self):
        op = self.operation()
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        self.assertIsNotNone(trace)
        self.assertRegex(trace, r"[0-9a-f]{32}")
        self.assertEqual(events(self.store, trace), [])
        self.assertEqual(op["state"], "queued")

    def test_confirm_failure_rolls_back_the_link_together(self):
        p = self.bind()
        plan = self.s.plan(p["id"], p["revision"])["id"]
        original_link = diagnostics.link

        def broken_link(db, *args, **kwargs):
            raise RuntimeError("injected diagnostic failure")

        with patch.object(diagnostics, "link", broken_link):
            with self.assertRaises(RuntimeError):
                self.s.confirm(plan)
        with self.store.db() as db:
            self.assertEqual(db.execute("SELECT used FROM plans WHERE id=?", (plan,)).fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT count(*) FROM operations").fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT count(*) FROM diagnostic_trace_links").fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT count(*) FROM diagnostic_events").fetchone()[0], 0)
        # The same plan remains usable once the diagnostic path is healthy again.
        op = self.s.confirm(plan)
        self.assertIsNotNone(diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"]))

    def test_d1_t05_worker_success_records_claimed_and_observed(self):
        op = self.operation()
        self.assertTrue(self.s.work_one())
        self.assertEqual(self.store.receipt(op["id"])["state"], "succeeded")
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        entries = events(self.store, trace)
        self.assertEqual(codes(entries)[0], ("claimed", "passed", "CLAIMED"))
        self.assertEqual(codes(entries)[-1], ("observed", "passed", "OK"))
        self.assertIsNotNone(entries[-1]["elapsed_ms"])
        self.assertTrue(all(e["operation_id"] == op["id"] for e in entries))
        self.assertTrue(all(e["actor_scope"] == "operator" for e in entries))

    def test_d1_t06_upstream_rejection_classified_while_receipt_code_unchanged(self):
        def rejecting_hello(target, workspace, budget):
            raise diagnostics.attach(Fault("PROVIDER_REJECTED"), "TOOL_IS_ERROR")

        self.fixture.hello = rejecting_hello
        op = self.operation()
        self.assertTrue(self.s.work_one())
        receipt = self.store.receipt(op["id"])
        self.assertEqual(receipt["state"], "unknown")  # public semantics unchanged
        self.assertEqual(receipt["code"], "PROVIDER_REJECTED")
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        self.assertIn(("observed", "unknown", "TOOL_IS_ERROR"), codes(events(self.store, trace)))

    def test_unexpected_exception_records_fixed_class_only(self):
        def exploding_hello(target, workspace, budget):
            raise RuntimeError("untrusted transport detail SECRET-VALUE")

        self.fixture.hello = exploding_hello
        op = self.operation()
        self.assertTrue(self.s.work_one())
        receipt = self.store.receipt(op["id"])
        self.assertEqual(receipt["state"], "unknown")
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        rendered = json.dumps(events(self.store, trace))
        self.assertIn("UNCLASSIFIED_EXCEPTION", rendered)
        self.assertNotIn("SECRET-VALUE", rendered)

    def test_d1_t06_worker_kill_after_dispatch_preserves_unknown_without_replay(self):
        op = self.operation()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE operations SET state='dispatching' WHERE id=?", (op["id"],))
        hello_calls = len(self.fixture.calls)
        self.s.recover()
        receipt = self.store.receipt(op["id"])
        self.assertEqual(receipt["state"], "unknown")
        self.assertEqual(receipt["code"], "WORKER_LOST")
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        self.assertIn(("recovered", "unknown", "WORKER_LOST"), codes(events(self.store, trace)))
        self.assertEqual(len(self.fixture.calls), hello_calls)  # no remote replay

    def test_pre_migration_operation_is_processed_with_unlinked_trace(self):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("INSERT INTO projects VALUES('p-pre','legacy','{}','d','/home/u',1)")
            db.execute("INSERT INTO operations VALUES('op-pre','p-pre',1,'d','operator','queued',1,9,NULL,NULL)")
        self.assertTrue(self.s.work_one())
        receipt = self.store.receipt("op-pre", "operator")
        self.assertIn(receipt["state"], ("failed", "unknown"))
        with self.store.db() as db:
            total = db.execute("SELECT count(*) FROM diagnostic_events WHERE operation_id='op-pre'").fetchone()[0]
        self.assertGreaterEqual(total, 1)  # events exist though no trace link does

    def test_d1_t07_broken_diagnostic_writer_never_blocks_the_operation(self):
        op = self.operation()
        with patch.object(diagnostics, "build", side_effect=ValueError("writer broken")):
            self.assertTrue(self.s.work_one())
        self.assertEqual(self.store.receipt(op["id"])["state"], "succeeded")
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        self.assertEqual(events(self.store, trace), [])  # capture gap, no fabricated evidence

    def test_d1_t07_sqlite_failure_is_a_silent_capture_gap(self):
        op = self.operation()
        broken = Store(self.root / "state" / "broken.sqlite3")

        def broken_db():
            raise sqlite3.OperationalError("disk full")

        with patch.object(broken, "db", side_effect=broken_db):
            self.assertIsNone(diagnostics.emit(broken, component="worker", phase="p", event_type="observed",
                                               outcome="unknown", code="LOAD", trace_id="a" * 32))
        # The real operation proceeds untouched and its events still land.
        self.assertTrue(self.s.work_one())
        self.assertEqual(self.store.receipt(op["id"])["state"], "succeeded")
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        self.assertGreaterEqual(len(events(self.store, trace)), 2)

    def test_d1_t11_actors_are_isolated(self):
        p = self.bind()
        plan_a = self.s.plan(p["id"], p["revision"], actor="alice")["id"]
        op_a = self.s.confirm(plan_a, actor="alice")
        plan_b = self.s.plan(p["id"], p["revision"], actor="bob")["id"]
        op_b = self.s.confirm(plan_b, actor="bob")
        trace_a = diagnostics.trace_for_operation(self.store, "alice", "hello", op_a["id"])
        self.assertIsNone(diagnostics.trace_for_operation(self.store, "bob", "hello", op_a["id"]))
        self.assertIsNone(diagnostics.trace_for_operation(self.store, "alice", "environment", op_a["id"]))
        self.assertNotEqual(trace_a, diagnostics.trace_for_operation(self.store, "bob", "hello", op_b["id"]))

    def test_d1_t13_caps_prune_only_owned_diagnostics(self):
        op = self.operation()
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        with patch.object(diagnostics, "MAX_EVENTS", 6), patch.object(diagnostics, "MAX_EVENTS_PER_TRACE", 4):
            for index in range(10):
                diagnostics.emit(self.store, component="worker", phase="p", event_type="observed",
                                 outcome="unknown", code="LOAD", trace_id=trace,
                                 operation_kind="hello", operation_id=op["id"])
        self.assertLessEqual(len(events(self.store, trace)), 4)
        with self.store.db() as db:
            total = db.execute("SELECT count(*) FROM diagnostic_events").fetchone()[0]
            self.assertLessEqual(total, 7)
            self.assertEqual(db.execute("SELECT count(*) FROM operations").fetchone()[0], 1)
        self.assertEqual(self.store.receipt(op["id"])["state"], "queued")  # business row untouched

    def test_events_for_trace_limit_is_bounded(self):
        op = self.operation()
        trace = diagnostics.trace_for_operation(self.store, "operator", "hello", op["id"])
        with self.assertRaises(ValueError):
            diagnostics.events_for_trace(self.store, trace, limit=0)
        with self.assertRaises(ValueError):
            diagnostics.events_for_trace(self.store, trace, limit=1001)


if __name__ == "__main__":
    unittest.main()
