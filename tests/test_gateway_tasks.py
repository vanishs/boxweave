"""R-02 gateway-private dedupe/transport-loss tests, with real SQLite."""
import os
from pathlib import Path
import sqlite3
import tempfile
import threading
from concurrent.futures import ThreadPoolExecutor
import unittest
from unittest.mock import patch
from bwmini import gateway_tasks as tasks

KEY="a"*32
ID="b"*64

class TrackedTaskTests(unittest.TestCase):
    def setUp(self):
        self.folder=tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.path=Path(self.folder.name)/"tasks.sqlite"
        self.path.touch(mode=0o600)
        self.store=tasks.TaskStore(self.path)
        self.store.initialize()

    def test_duplicate_returns_same_receipt_without_dispatch(self):
        d=tasks.command_digest("box1","pytest -q")
        first,new=self.store.reserve(KEY,"box1",ID,d)
        self.assertTrue(new)
        again,new=self.store.reserve(KEY,"box1",ID,d)
        self.assertFalse(new)
        self.assertEqual(first["task_id"],again["task_id"])
        self.assertEqual(again["state"],"recorded")

    def test_same_key_changed_target_or_payload_fails(self):
        d=tasks.command_digest("box1","echo safe")
        self.store.reserve(KEY,"box1",ID,d)
        for box,identity,digest in [("box2",ID,d),("box1","a"*64,d),
                                     ("box1",ID,tasks.command_digest("box1","echo other"))]:
            with self.subTest(box=box),self.assertRaisesRegex(tasks.TaskFault,"REQUEST_KEY_CONFLICT"):
                self.store.reserve(KEY,box,identity,digest)

    def test_transport_death_reports_unknown_without_replay(self):
        meta,new=self.store.reserve(KEY,"box1",ID,tasks.command_digest("box1","pytest"))
        self.store.advance(meta["task_id"],"recorded","dispatch_intent")
        restarted=tasks.TaskStore(self.path)
        self.assertEqual(restarted.get(request_key=KEY)["state"],"unknown")
        same,new=restarted.reserve(KEY,"box1",ID,tasks.command_digest("box1","pytest"))
        self.assertFalse(new)
        self.assertEqual(same["state"],"unknown")
        with self.assertRaises(tasks.TaskFault):
            restarted.advance(meta["task_id"],"recorded","dispatch_intent")

    def test_exit_nonzero_is_known_but_output_not_retained(self):
        receipt,_=self.store.reserve(KEY,"box1",ID,tasks.command_digest("box1","pytest"))
        self.store.advance(receipt["task_id"],"recorded","dispatch_intent")
        self.store.advance(receipt["task_id"],"dispatch_intent","completed",exit_code=7)
        record=self.store.get(task_id=receipt["task_id"])
        self.assertEqual(record["state"],"completed")
        self.assertEqual(record["exit_code"],7)
        self.assertEqual(record["result"],{"available":False})
        self.assertNotIn("pytest",repr(record))
        self.assertNotIn("pytest",self.path.read_bytes().decode(errors="replace"))

    def test_racing_duplicate_keys_are_atomic(self):
        def job(_):
            return self.store.reserve(KEY,"box1",ID,tasks.command_digest("box1","date"))
        with ThreadPoolExecutor(max_workers=8) as pool:
            output=list(pool.map(job,range(8)))
        self.assertEqual(sum(new for _,new in output),1)
        self.assertEqual(len({x["task_id"] for x,_ in output}),1)

    def test_explicit_claim_provision_private_idempotent(self):
        tasks.provision_claims(self.folder.name)
        claim=Path(self.folder.name)/"claims.sqlite3"
        self.assertEqual(claim.stat().st_mode & 0o777, 0o600)
        with sqlite3.connect(claim) as db:
            tables={r[0] for r in db.execute(
                "SELECT name FROM sqlite_master WHERE type='table'")}
        self.assertIn("coordination_leases",tables)
        self.assertIn("coordination_operations",tables)
        tasks.provision_claims(self.folder.name)
        self.assertEqual(claim.stat().st_mode & 0o777, 0o600)

    def test_unprovisioned_or_corrupt_database_refuses(self):
        self.path.unlink()
        with self.assertRaisesRegex(tasks.TaskFault,"TASK_TRACKING_UNAVAILABLE"):
            self.store.get(request_key=KEY)
        self.path.touch(mode=0o600)
        with self.assertRaisesRegex(tasks.TaskFault,"TASK_SCHEMA_UNSUPPORTED"):
            self.store.get(request_key=KEY)
        with sqlite3.connect(self.path) as db:
            db.execute("PRAGMA user_version=12")
        with self.assertRaisesRegex(tasks.TaskFault,"TASK_SCHEMA_UNSUPPORTED"):
            self.store.initialize()

    def test_symlink_and_group_writable_refused(self):
        self.path.unlink()
        self.path.symlink_to("/etc/passwd")
        with self.assertRaisesRegex(tasks.TaskFault,"TASK_STORAGE_UNSAFE"):
            self.store.get(request_key=KEY)
        self.path.unlink()
        self.path.touch(mode=0o600)
        self.path.chmod(0o660)
        with self.assertRaisesRegex(tasks.TaskFault,"TASK_STORAGE_UNSAFE"):
            self.store.initialize()

    def test_no_arbitrary_command_or_result_acceptance(self):
        for command in ("", "x"*5000, "\x00python"):
            with self.assertRaisesRegex(tasks.TaskFault,"TASK_ARGUMENT_INVALID"):
                tasks.command_digest("box1",command)
        for payload in ({"result":{"isError":True}},{"result":{"structuredContent":{"exit_code":True}}},{}):
            with self.assertRaises(tasks.TaskFault):
                tasks.observed_exit(payload)
        self.assertEqual(tasks.observed_exit({"result":{"structuredContent":{"exit_code":7}}}),7)

    def test_real_provider_empty_output_exit_is_authoritative(self):
        for code in (0, 1, 7, -1):
            self.assertEqual(tasks.observed_exit({"result": {
                "content": [{"type": "text", "text": f"exit_code: {code}\n"}]}}), code)
        for raw in ("exit_code: 0", "exit_code: 0\nSECRET",
                    "exit_code: 0\n\n", "exit_code: 0\n\n--- stderr ---\n",
                    "prefix exit_code: 0\n", "exit_code: +0\n"):
            with self.subTest(raw=raw), self.assertRaisesRegex(
                    tasks.TaskFault, "TASK_RESULT_UNVERIFIED"):
                tasks.observed_exit({"result": {"content": [
                    {"type": "text", "text": raw}]}})
        self.assertEqual(tasks.observed_exit({"result": {"content": [
            {"type": "text", "text": "exit_code: 2\n\n--- stdout ---\noutput\n"}]}}), 2)

    def test_bounded_capacity_and_selector(self):
        with patch.object(tasks,"TASK_LIMIT",0):
            with self.assertRaisesRegex(tasks.TaskFault,"TASK_CAPACITY_EXCEEDED"):
                self.store.reserve(KEY,"box1",ID,tasks.command_digest("box1","hello"))
        for kwargs in ({},{ "task_id":KEY,"request_key":KEY }, {"task_id":"not a uuid"}):
            with self.assertRaisesRegex(tasks.TaskFault,"TASK_SELECTOR_INVALID"):
                self.store.get(**kwargs)

if __name__=="__main__":
    unittest.main()
