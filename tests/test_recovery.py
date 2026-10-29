"""Real SQLite failure injection and actual process-death recovery."""
import json
import multiprocessing
import os
import sqlite3
from concurrent.futures import ThreadPoolExecutor
from contextlib import contextmanager
from pathlib import Path
from test_slice import Base, TARGET, HASH, PASSWORD
from bwmini.web import create_app
import subprocess
import sys
from bwmini.domain import Fault
from bwmini.service import Service
from bwmini.store import Store
from bwmini.provider import Fixture


def crash_worker(path, marker, when):
    store = Store(path)
    provider = Fixture(TARGET)
    if when == "before_send":
        def stop(*_):
            os._exit(71)
        provider.observe = stop
    else:
        def sent(*_):
            Path(marker).write_text("sent")
            os._exit(72)
        provider.hello = sent
    Service(store, provider, clock=lambda:1000.).work_one()


class RecoveryTests(Base):
    def test_real_sqlite_full_rolls_back_intent_and_plan_use(self):
        p = self.bind()
        plan = self.s.plan(p["id"], 1)
        with self.store.db() as db:
            db.execute("CREATE TABLE allocation_guard (data BLOB)")
            db.execute("CREATE TRIGGER full_insert AFTER INSERT ON operations BEGIN INSERT INTO allocation_guard VALUES(zeroblob(1048576)); END")
        original = self.store.db
        @contextmanager
        def full_db():
            with original() as db:
                pages = db.execute("PRAGMA page_count").fetchone()[0]
                db.execute(f"PRAGMA max_page_count={pages}")
                yield db
        self.store.db = full_db
        with self.assertRaisesRegex(Fault, "STATE_UNAVAILABLE"):
            self.s.confirm(plan["id"])
        self.store.db = original
        with original() as db:
            self.assertEqual(db.execute("SELECT count(*) FROM operations").fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT used FROM plans WHERE id=?", (plan["id"],)).fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT count(*) FROM allocation_guard").fetchone()[0], 0)
        self.assertEqual(self.fixture.calls, [])

    def test_claim_write_failure_cannot_dispatch(self):
        op = self.operation()
        with self.store.db() as db:
            db.execute("CREATE TRIGGER deny_claim BEFORE UPDATE OF state ON operations WHEN NEW.state='dispatching' BEGIN SELECT RAISE(ABORT,'synthetic claim failure'); END")
        with self.assertRaisesRegex(Fault, "STATE_UNAVAILABLE"):
            self.s.work_one()
        self.assertEqual(self.store.receipt(op["id"])["state"], "queued")
        self.assertEqual(self.fixture.calls, [])
        with self.store.db() as db:
            db.execute("DROP TRIGGER deny_claim")
        self.s.worker(once=True)
        self.assertEqual(self.fixture.calls, ["identity", "hello"])

    def test_result_audit_failure_is_atomic_then_unknown_without_replay(self):
        op = self.operation()
        deadline = op["deadline"]
        with self.store.db() as db:
            db.execute("CREATE TRIGGER deny_audit BEFORE INSERT ON audit BEGIN SELECT RAISE(ABORT,'synthetic audit failure'); END")
        with self.assertRaisesRegex(Fault, "STATE_UNAVAILABLE"):
            self.s.work_one()
        row = self.store.receipt(op["id"])
        self.assertEqual(row["state"], "dispatching")
        self.assertIsNone(row["result"])
        self.assertEqual(self.fixture.calls, ["identity", "hello"])
        self.s.worker(once=True)
        row = self.store.receipt(op["id"])
        self.assertEqual((row["state"], row["code"], row["deadline"]), ("unknown", "WORKER_LOST", deadline))
        self.assertIsNone(row["result"])
        self.assertEqual(self.fixture.calls, ["identity", "hello"])

    def test_actual_process_death_before_send_and_after_remote_effect(self):
        for when in ("before_send", "after_effect"):
            with self.subTest(when=when):
                op = self.operation()
                marker = self.root / (when + ".txt")
                child = multiprocessing.get_context("fork").Process(
                    target=crash_worker, args=(str(self.store.path), str(marker), when))
                child.start(); child.join(10)
                self.assertFalse(child.is_alive())
                self.assertEqual(child.exitcode, 71 if when == "before_send" else 72)
                self.assertEqual(self.store.receipt(op["id"])["state"], "dispatching")
                self.s.worker(once=True)
                row = self.store.receipt(op["id"])
                self.assertEqual((row["state"], row["result"], row["deadline"]), ("unknown", None, op["deadline"]))
                self.assertEqual(marker.exists(), when == "after_effect")
                self.assertEqual(self.fixture.calls, [])

    def test_two_real_sqlite_claims_cannot_duplicate_send(self):
        self.operation()
        with ThreadPoolExecutor(max_workers=2) as pool:
            results = list(pool.map(lambda _:self.s.work_one(), range(2)))
        self.assertEqual(sorted(results), [False, True])
        self.assertEqual(self.fixture.calls, ["identity", "hello"])

    def test_nonzero_command_exit_is_failure_not_transport_unknown(self):
        op = self.operation()
        self.fixture.hello = lambda *_:{"stdout":"synthetic-private-error", "exit_code":7}
        self.s.work_one()
        row = self.store.receipt(op["id"])
        result = json.loads(row["result"])
        self.assertEqual((row["state"], row["code"], result["exit_code"]), ("failed", "COMMAND_FAILED", 7))
        self.assertTrue(result["output_withheld"])
        self.assertNotIn("synthetic-private-error", json.dumps(row))

    def test_reconcile_records_matching_but_never_proves_completion_or_replays(self):
        op = self.operation()
        with self.store.db() as db:
            db.execute("UPDATE operations SET state='dispatching' WHERE id=?", (op["id"],))
        self.s.worker(once=True)
        before = self.store.receipt(op["id"])
        observed = self.s.reconcile(op["id"])
        self.assertEqual((observed["target_status"], observed["completion"], observed["retry"]), ("matching", "unverified", "not_authorized"))
        self.assertEqual(self.store.receipt(op["id"]), before)
        restarted = Service(Store(self.store.path), self.fixture, clock=lambda:1000.)
        self.assertEqual(len(restarted.observations(op["id"])), 1)
        restarted.worker(once=True)
        self.assertEqual(self.fixture.calls, [])
        with self.assertRaisesRegex(Fault, "NOT_FOUND"):
            restarted.reconcile(op["id"], actor="other")

    def test_reconcile_stale_binding_and_concurrent_edit_are_changed(self):
        op = self.operation()
        with self.store.db() as db:
            db.execute("UPDATE operations SET state='unknown' WHERE id=?", (op["id"],))
        original = self.fixture.observe
        def concurrent(logical):
            value = original(logical)
            self.fixture.observe = original
            self.s.bind("Changed", TARGET.logical, "/home/fixture-box", op["project"], 1)
            return value
        self.fixture.observe = concurrent
        observed = self.s.reconcile(op["id"])
        self.assertEqual(observed["code"], "REVISION_CONFLICT")
        self.fixture.observe = lambda *_: (_ for _ in ()).throw(AssertionError("must not observe stale binding"))
        self.assertEqual(self.s.reconcile(op["id"])["code"], "REVISION_CONFLICT")
        self.assertEqual(self.fixture.calls, [])

    def test_failed_observation_is_sanitized_and_state_stays_unknown(self):
        op = self.operation()
        with self.store.db() as db:
            db.execute("UPDATE operations SET state='unknown' WHERE id=?", (op["id"],))
        self.fixture.observe = lambda *_: (_ for _ in ()).throw(OSError("synthetic-private-message"))
        observed = self.s.reconcile(op["id"])
        self.assertEqual(observed["code"], "OBSERVATION_UNAVAILABLE")
        self.assertNotIn("synthetic-private-message", json.dumps(observed))
        self.assertEqual(self.store.receipt(op["id"])["state"], "unknown")

    def test_schema_two_upgrade_preserves_receipts_graphs_and_deadlines(self):
        op = self.operation()
        with self.store.db() as db:
            for table in ("environment_events","managed_environments","environment_operations","environment_plans"):
                db.execute("DROP TABLE "+table)
            db.execute("DROP TABLE operation_observations")
            db.execute("DROP TABLE model_selections")
            for table in ('access_claims', 'access_events', 'access_operations', 'access_plans', 'access_routes', 'tenant_labels'):db.execute("DROP TABLE "+table)
            db.execute("INSERT INTO graphs VALUES('test',1,'{}')")
            db.execute("DROP TABLE project_unbindings");db.execute("PRAGMA user_version=2")
        upgraded = Store(self.store.path)
        self.assertEqual(upgraded.receipt(op["id"]), op)
        with upgraded.db() as db:
            self.assertEqual(db.execute("SELECT count(*) FROM graphs").fetchone()[0], 1)
            self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], 7)

    def test_restart_keeps_expired_deadline_and_blocks_dispatch(self):
        op = self.operation()
        restarted = Service(Store(self.store.path), self.fixture, clock=lambda:2000.)
        restarted.worker(once=True)
        row = self.store.receipt(op["id"])
        self.assertEqual((row["code"], row["deadline"]), ("DEADLINE_EXCEEDED", op["deadline"]))
        self.assertEqual(self.fixture.calls, [])

    def test_reconciliation_http_authorization_and_cli(self):
        op = self.operation()
        with self.store.db() as db:
            db.execute("UPDATE operations SET state='unknown' WHERE id=?", (op["id"],))
        self.store.admin(HASH)
        client = create_app(self.s, "http://localhost", self.root, secure=False).test_client()
        url = "/api/operations/" + op["id"] + "/reconcile"
        self.assertEqual(client.post(url, json={}, headers={"Origin":"http://localhost"}).status_code, 401)
        login = client.post("/api/login", json={"password":PASSWORD}, headers={"Origin":"http://localhost"})
        self.assertEqual(client.post(url, json={}, headers={"Origin":"http://localhost"}).status_code, 403)
        headers={"Origin":"http://localhost", "X-CSRF-Token":login.json["csrf"]}
        self.assertEqual(client.post(url, json={"retry":True}, headers=headers).status_code, 400)
        response=client.post(url, json={}, headers=headers)
        self.assertEqual((response.status_code,response.json["state"]),(200,"unknown"))
        self.assertEqual(len(client.get("/api/operations/"+op["id"]+"/observations").json["observations"]),1)
        config=self.root/"config.json"
        config.write_text(json.dumps({"mode":"fixture","database":str(self.store.path),"target":TARGET.data()}))
        result=subprocess.run([sys.executable,"-m","bwmini","reconcile","--config",str(config),"--operation",op["id"]],capture_output=True,text=True,timeout=10)
        self.assertEqual(result.returncode,0,result.stdout)
        self.assertEqual(json.loads(result.stdout)["completion"],"unverified")
        self.assertEqual(self.fixture.calls,[])
