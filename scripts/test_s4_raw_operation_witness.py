"""S4 test-only raw Operation witness fail-closed regression tests."""
import importlib.util
import sqlite3
import tempfile
import unittest
from pathlib import Path

name = "s4_raw_operation_witness"
spec = importlib.util.spec_from_file_location(
    name, Path(__file__).with_name(name + ".py"))
witness = importlib.util.module_from_spec(spec)
spec.loader.exec_module(witness)


class WitnessTest(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        self.path = Path(self.dir.name) / "ledger.db"
        self.opid = "a" * 32
        self.native_id = "b" * 32
        with sqlite3.connect(self.path) as db:
            db.execute("CREATE TABLE operations (id,action,layer,box_uuid,"
                       "incarnation,resource_key,native_operation_id,state)")
            db.execute("CREATE TABLE exec_outputs (operation_id,exit_code,truncation)")
            db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?)",
                       (self.opid, "box_exec", "test-project", "owned-uuid",
                        "owned-uuid", "box:guest", self.native_id, "SUCCEEDED"))
            db.execute("INSERT INTO exec_outputs VALUES (?,?,?)",
                       (self.opid, 127, 0))

    def test_digest_finds_only_the_one_original(self):
        import hashlib
        value = "s4-stable-key"
        hashed = hashlib.sha256(value.encode()).hexdigest()
        with sqlite3.connect(self.path) as db:
            db.execute("ALTER TABLE operations ADD COLUMN key_digest TEXT DEFAULT ''")
            db.execute("UPDATE operations SET key_digest=? WHERE id=?", (hashed, self.opid))
        self.assertEqual(witness.discover_original(self.path, hashed), self.opid)
        self.assertIsNone(witness.discover_original(self.path, "0" * 64))

    def test_terminal_captured_and_guest_exit_not_inferred(self):
        def native(project, native_id):
            self.assertEqual((project, native_id), ("test-project", self.native_id))
            return {"native_operation_id": native_id, "status_code": 400,
                    "status": "Failure", "error_present": True}, None
        r = witness.collect(self.path, self.opid, 0.1, native=native)
        self.assertEqual(r["observation"], "CAPTURED_TERMINAL")
        self.assertEqual(r["original_sqlite"]["guest_exit_code"], 127)
        self.assertEqual(r["native_terminal"]["status_code"], 400)

    def test_missing_retained_native_not_a_fake_terminal(self):
        r = witness.collect(self.path, self.opid, 0.01,
                            native=lambda *_: (None, "NATIVE_NOT_RETAINED_OR_UNAVAILABLE"))
        self.assertEqual(r["observation"], "NOT_CAPTURED")
        self.assertIsNone(r["native_terminal"])

    def test_mismatched_native_id_not_original_evidence(self):
        r = witness.collect(self.path, self.opid, 0.01,
                            native=lambda *_: ({"native_operation_id": "foreign",
                                                 "status_code": 200}, None))
        self.assertFalse(r["original_id_matches"])

    def test_missing_sqlite_original_not_evidence(self):
        r = witness.collect(self.path, "c" * 32, 0.01)
        self.assertEqual(r["observation"], "NOT_CAPTURED")


    def test_pre_dispatch_native_event_terminal_exact_id_and_project(self):
        events = Path(self.dir.name) / "events.jsonl"
        data = {
            "source": "INCUS_OPERATION_EVENT_WEBSOCKET",
            "project": "test-project",
            "event_project": "test-project",
            "native_operation_id": self.native_id,
            "status_code": 400,
            "status": "Failure",
            "error_present": True,
            "raw_metadata_sha256": "c" * 64,
        }
        import json
        events.write_text(json.dumps(data) + "\n", encoding="utf-8")
        r = witness.collect(
            self.path, self.opid, 0.05,
            native=lambda p, i: (witness.event_terminal(events, p, i), None))
        self.assertEqual(r["observation"], "CAPTURED_TERMINAL")
        self.assertEqual(r["native_terminal"]["evidence_source"],
                         "INCUS_OPERATION_EVENT_WEBSOCKET")
        self.assertEqual(r["native_terminal"]["raw_metadata_sha256"], "c" * 64)
        self.assertEqual(r["original_sqlite"]["guest_exit_code"], 127)

    def test_foreign_or_corrupt_event_never_fabricates_terminal(self):
        events = Path(self.dir.name) / "events.jsonl"
        import json
        base = {
            "source": "INCUS_OPERATION_EVENT_WEBSOCKET",
            "project": "foreign-project",
            "event_project": "foreign-project",
            "native_operation_id": self.native_id,
            "status_code": 200,
            "status": "Success",
            "error_present": False,
            "raw_metadata_sha256": "d" * 64,
        }
        invalid = [base, {**base, "project": "test-project"},
                   {**base, "project": "test-project",
                    "event_project": "test-project",
                    "native_operation_id": "f" * 32},
                   {**base, "project": "test-project",
                    "event_project": "test-project",
                    "raw_metadata_sha256": "not-an-event-hash"}]
        events.write_text("\n".join(json.dumps(v) for v in invalid) + "\n",
                          encoding="utf-8")
        self.assertIsNone(witness.event_terminal(events, "test-project", self.native_id))
        r = witness.collect(
            self.path, self.opid, 0.01,
            native=lambda p, i: (witness.event_terminal(events, p, i),
                                 "NATIVE_NOT_RETAINED_OR_UNAVAILABLE"))
        self.assertEqual(r["observation"], "NOT_CAPTURED")


if __name__ == "__main__":
    unittest.main()
