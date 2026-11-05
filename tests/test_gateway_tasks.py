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
        # Actual Provider stderr-only response: exit code is definitive.
        self.assertEqual(tasks.observed_exit({"result": {"content": [
            {"type": "text", "text": "exit_code: 7\n\n--- stderr ---\nmarker\n"}]}}), 7)

    def test_bounded_capacity_and_selector(self):
        with patch.object(tasks,"TASK_LIMIT",0):
            with self.assertRaisesRegex(tasks.TaskFault,"TASK_CAPACITY_EXCEEDED"):
                self.store.reserve(KEY,"box1",ID,tasks.command_digest("box1","hello"))
        for kwargs in ({},{ "task_id":KEY,"request_key":KEY }, {"task_id":"not a uuid"}):
            with self.assertRaisesRegex(tasks.TaskFault,"TASK_SELECTOR_INVALID"):
                self.store.get(**kwargs)


class RecoveryOperatorGatewayTests(unittest.TestCase):
    def setUp(self):
        from bwmini.operation_coordination import OperationCoordinator,box_resource
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.path=Path(self.temp.name)
        self.tasks=tasks.TaskStore(self.path/"tasks.sqlite")
        (self.path/"tasks.sqlite").touch(mode=0o600)
        self.tasks.initialize()
        self.claims=OperationCoordinator(lambda:sqlite3.connect(self.path/"claims.sqlite"))
        self.claims.initialize()
        record,_=self.tasks.reserve(KEY,"box1",ID,tasks.command_digest("box1","echo one"))
        self.op=record["task_id"]
        self.tasks.advance(self.op,"recorded","dispatch_intent")
        self.claims.acquire(self.op,ID,"gateway_"+self.op,[box_resource(ID)])
        self.claims.dispatch_intent(self.op)

    def test_original_gateway_receipt_matches_opaque_original_id(self):
        original=tasks.original_recovery_receipt(self.op,"gateway_"+self.op,ID,
                                                  tasks_path=self.path/"tasks.sqlite")
        self.assertEqual(original["state"],"unknown")
        self.assertTrue(original["exact_original_task"])
        self.assertTrue(original["identity_matches_claim"])
        mismatched=tasks.original_recovery_receipt(self.op,"gateway_"+self.op,"c"*64,
                                                   tasks_path=self.path/"tasks.sqlite")
        self.assertFalse(mismatched["identity_matches_claim"])
        missing=tasks.original_recovery_receipt(self.op,"gateway_"+self.op,ID,
                                                 tasks_path=self.path/"missing.sqlite")
        self.assertFalse(missing["exact_original_task"])
        self.assertEqual(missing["state"],"unverified")

    def test_app_receipts_have_explicit_positive_and_missing_states(self):
        path=self.path/"state.sqlite";path.touch(mode=0o600)
        with sqlite3.connect(path) as db:
            db.execute("CREATE TABLE environment_operations(id TEXT PRIMARY KEY, state TEXT,logical TEXT)")
            db.execute("INSERT INTO environment_operations VALUES(?,?,?)",(self.op,"unknown","box1"))
        found=tasks.original_recovery_receipt(self.op,"environment_"+self.op,ID,app_path=path)
        self.assertEqual(found["state"],"unknown")
        self.assertTrue(found["exact_original_task"])
        notfound=tasks.original_recovery_receipt("b"*32,"environment_"+"b"*32,ID,app_path=path)
        self.assertEqual(notfound["state"],"unverified")
        self.assertFalse(notfound["exact_original_task"])

    def test_recovery_review_challenge_is_always_separate_argv_safe(self):
        from unittest.mock import patch
        # Force an RNG token that begins with a '-' option prefix. The
        # generated challenge must remain safe as --challenge TOKEN (not just
        # --challenge=TOKEN), preserving 43 characters and the stored digest.
        with patch("bwmini.operation_coordination.secrets.token_urlsafe",
                   return_value="-" * 42):
            review = self.claims.recovery_review(self.op, "test-admin")
        self.assertEqual(review["challenge"], "r" + "-" * 42)
        self.assertRegex(review["challenge"], r"^r[A-Za-z0-9_-]{42}$")

    def test_operator_cli_inspect_release_and_audit_with_exact_confirmation(self):
        import io,json
        from contextlib import redirect_stdout,redirect_stderr
        class LocalClaims:
            def __init__(inner):inner.coordinator=self.claims
        with patch('bwmini.coordination_bridge.SharedClaims',LocalClaims),patch.object(
                tasks,'original_recovery_receipt',return_value={"source":"gateway","state":"unknown"}):
            output=io.StringIO()
            with redirect_stdout(output):
                code=tasks.recovery_cli(["inspect",self.op,"--reviewer","test-admin"])
            self.assertEqual(code,0)
            review=json.loads(output.getvalue());self.assertEqual(review["operation_id"],self.op)
            denied=io.StringIO()
            with redirect_stderr(denied):
                code=tasks.recovery_cli(["release",self.op,"--reviewer","test-admin",
                      "--challenge",review["challenge"],"--confirm","RELEASE "+self.op,
                      "--reason","Test-approved cannot establish outcome"])
            self.assertEqual(code,2)
            self.assertIn("RECOVERY_CONFIRMATION_REQUIRED",denied.getvalue())
            out=io.StringIO()
            with redirect_stdout(out):
                code=tasks.recovery_cli(["release",self.op,"--reviewer","test-admin",
                      "--challenge",review["challenge"],"--confirm","RELEASE "+self.op,
                      "--reason","Test-approved cannot establish outcome",
                      "--acknowledge-remote-may-still-run"])
            self.assertEqual(code,0)
            self.assertEqual(json.loads(out.getvalue())["released_resources"],1)
            out=io.StringIO()
            with redirect_stdout(out):
                code=tasks.recovery_cli(["audit",self.op,"--reviewer","test-admin"])
            self.assertEqual(code,0)
            self.assertEqual(len(json.loads(out.getvalue())["audits"]),1)
            self.assertEqual(self.tasks.get(task_id=self.op)["state"],"unknown")

if __name__=="__main__":
    unittest.main()
