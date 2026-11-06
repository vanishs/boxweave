"""Black-box HTTP create tests: one dispatch, durable receipts, no retry."""
import json
from pathlib import Path
import tempfile
import threading
import time
import unittest
from concurrent.futures import ThreadPoolExecutor

from bwmini.remote_auth import RemoteCredentials
from bwmini.remote_create import CreateFault, CreateJournal, validated_request
from bwmini.remote_mcp import create_remote_app
from bwmini.store import Store

ACCEPT = "application/json, text/event-stream"
RESOURCE = "http://127.0.0.1:18844/mcp"


class CreateJournalTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.store = Store(Path(self.tmp.name) / "state" / "store.sqlite3")
        self.journal = CreateJournal(self.store)

    def test_migration_is_durable_without_storing_args_or_keys(self):
        with self.store.db() as db:
            self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], 14)
        logical, key, args, digest = validated_request({
            "username": "test-a", "cpu": "1", "request_key": "a"*32,
            "ssh_keys": ["ssh-ed25519 public-material-fixture"]})
        receipt, fresh = self.journal.reserve(logical, key, digest)
        self.assertTrue(fresh)
        self.assertEqual(receipt["state"], "submitted")
        assert receipt["task_id"] != key
        with self.store.db() as db:
            text = "\n".join(str(tuple(r)) for r in db.execute(
                "SELECT * FROM remote_create_tasks"))
        self.assertNotIn("public-material-fixture", text)
        self.assertNotIn("ssh-ed25519", text)

    def test_one_name_one_dispatch_regardless_of_retry_key(self):
        _, _, _, digest = validated_request({"username": "a", "cpu": "1"})
        first, fresh = self.journal.reserve("a", "a"*32, digest)
        self.assertTrue(fresh)
        second, fresh = self.journal.reserve("a", "b"*32, digest)
        self.assertFalse(fresh)
        self.assertEqual(first["task_id"], second["task_id"])
        self.assertEqual(first["request_key"], second["request_key"])
        with self.assertRaisesRegex(CreateFault, "CREATE_NAME_CONFLICT"):
            self.journal.reserve("a", "b"*32, "0"*64)
        with self.assertRaisesRegex(CreateFault, "CREATE_REQUEST_KEY_CONFLICT"):
            self.journal.reserve("b", "a"*32, digest)
        self.journal.complete(first["task_id"], True)
        self.assertEqual(self.journal.get(username="a")["state"], "succeeded")
        again, fresh = self.journal.reserve("a", None, digest)
        self.assertFalse(fresh)
        self.assertEqual(again["task_id"], first["task_id"])

    def test_durable_unknown_after_restart_never_replays(self):
        logical, key, _, digest = validated_request({"username": "b"})
        first, fresh = self.journal.reserve(logical, key, digest)
        self.assertTrue(fresh)
        CreateJournal(Store(self.store.path)).recover_interrupted()
        row = CreateJournal(Store(self.store.path)).get(username="b")
        self.assertEqual(row["state"], "unknown")
        self.assertEqual(row["code"], "PROCESS_RESTART_UNKNOWN")
        again, fresh = self.journal.reserve(logical, key, digest)
        self.assertFalse(fresh)
        self.assertEqual(first["task_id"], again["task_id"])
        self.journal.complete(first["task_id"], True)
        self.assertEqual(self.journal.get(username="b")["state"], "unknown")

    def test_simultaneous_reservation_is_atomic(self):
        _, _, _, digest = validated_request({"username": "atomic"})
        with ThreadPoolExecutor(max_workers=8) as pool:
            receipts = list(pool.map(
                lambda _: self.journal.reserve("atomic", None, digest),
                range(8)))
        self.assertEqual(len({r[0]["task_id"] for r in receipts}), 1)
        self.assertEqual(sum(fresh for _, fresh in receipts), 1)

    def test_validation_and_strict_read_only_selector(self):
        for args in (None, {"username": "../bad"},
                     {"username": "ok", "request_key": "short"},
                     {"username": "ok", "cpu": float("nan")}):
            with self.subTest(args=args), self.assertRaises(CreateFault):
                validated_request(args)
        with self.assertRaisesRegex(CreateFault, "CREATE_SELECTOR_INVALID"):
            self.journal.get()
        with self.assertRaisesRegex(CreateFault, "CREATE_SELECTOR_INVALID"):
            self.journal.get(username="../bad")


class HTTPCreateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.store = Store(Path(self.tmp.name) / "state" / "store.sqlite3")
        self.started = threading.Event()
        self.finish = threading.Event()
        self.upstream = []
        self.allow_create = True

        def dispatch(msg):
            if msg["method"] == "tools/list":
                names = ["list_containers"]
                if self.allow_create:
                    names.append("create_container")
                return {"jsonrpc": "2.0", "id": msg["id"], "result": {
                    "tools": [{"name": name,
                               "inputSchema": {"type": "object",
                                               "properties": {"username": {"type": "string"}}}}
                              for name in names]}}
            if msg["method"] == "tools/call":
                self.upstream.append(msg)
                self.started.set()
                self.finish.wait(3)
                return {"jsonrpc": "2.0", "id": msg["id"],
                        "result": {"content": [{"type": "text", "text": "created"}]}}
            raise AssertionError("unexpected method")

        self.app = create_remote_app(self.store, {}, dispatch=dispatch)
        self.client = self.app.test_client()
        self.token = RemoteCredentials(self.store, RESOURCE).issue("api", "fixture")
        self.headers = {"Authorization": "Bearer " + self.token,
                        "Content-Type": "application/json", "Accept": ACCEPT}

    def tearDown(self):
        self.finish.set()
        self.tmp.cleanup()

    def call(self, tool, args, ident=1, token=True):
        h = self.headers if token else {"Accept": ACCEPT}
        return self.client.post("/mcp", json={
            "jsonrpc": "2.0", "id": ident, "method": "tools/call",
            "params": {"name": tool, "arguments": args}}, headers=h)

    @staticmethod
    def result(reply):
        return json.loads(reply.json["result"]["content"][0]["text"])

    def test_background_response_and_chatgpt_replay_are_one_dispatch(self):
        args = {"username": "create-one", "cpu": "1",
                "request_key": "a"*32}
        first = self.call("create_container", args)
        self.assertEqual(first.status_code, 200)
        original = self.result(first)
        self.assertEqual(original["state"], "submitted")
        self.assertTrue(self.started.wait(2))
        second = self.call("create_container", {
            **args, "request_key": "b"*32}, ident=999)
        self.assertEqual(second.status_code, 200)
        self.assertEqual(self.result(second)["task_id"], original["task_id"])
        self.assertEqual(len(self.upstream), 1)
        self.assertNotIn("request_key",
                         self.upstream[0]["params"]["arguments"])
        get = self.call("boxweave_create_status",
                        {"task_id": original["task_id"]})
        self.assertEqual(self.result(get)["state"], "submitted")
        self.finish.set()
        for _ in range(150):
            get = self.call("boxweave_create_status", {"username": "create-one"})
            if self.result(get)["state"] == "succeeded":
                break
            time.sleep(.01)
        self.assertEqual(self.result(get)["state"], "succeeded")
        self.assertFalse(self.result(get)["guest_user_and_ssh_verified"])
        third = self.call("create_container", args, ident=1000)
        self.assertEqual(self.result(third)["task_id"], original["task_id"])
        self.assertEqual(len(self.upstream), 1)

    def test_conflicting_same_name_never_dispatches(self):
        self.call("create_container", {"username": "reused", "cpu": "1"})
        self.assertTrue(self.started.wait(2))
        fail = self.call("create_container", {"username": "reused", "cpu": "2"})
        self.assertTrue(fail.json["result"]["isError"])
        self.assertEqual(self.result(fail)["code"], "CREATE_NAME_CONFLICT")
        self.assertEqual(len(self.upstream), 1)

    def test_policy_refusal_is_before_persistent_intent(self):
        self.allow_create = False
        fail = self.call("create_container", {"username": "blocked"})
        self.assertEqual(self.result(fail)["code"], "GATE_TOOL_NOT_ALLOWED")
        self.assertEqual(len(self.upstream), 0)
        with self.assertRaisesRegex(CreateFault, "CREATE_TASK_NOT_FOUND"):
            CreateJournal(self.store).get(username="blocked")

    def test_catalog_and_authentication(self):
        payload = {"jsonrpc": "2.0", "id": 4, "method": "tools/list",
                   "params": {}}
        denied = self.client.post("/mcp", json=payload,
                                  headers={"Accept": ACCEPT})
        self.assertEqual(denied.status_code, 401)
        listed = self.client.post("/mcp", json=payload,
                                  headers=self.headers).json["result"]["tools"]
        names = [x["name"] for x in listed]
        self.assertIn("boxweave_create_status", names)
        create = next(x for x in listed if x["name"] == "create_container")
        self.assertIn("request_key", create["inputSchema"]["properties"])
        unauthenticated = self.call("create_container", {"username": "no-auth"},
                                    token=False)
        self.assertEqual(unauthenticated.status_code, 401)
        self.assertEqual(self.upstream, [])

    def test_status_selector_errors_are_mcp_results_not_http_500(self):
        invalid = self.call("boxweave_create_status", {
            "task_id": "a"*32, "username": "other"})
        self.assertEqual(invalid.status_code, 200)
        self.assertEqual(self.result(invalid)["code"], "CREATE_SELECTOR_INVALID")
        missing = self.call("boxweave_create_status", {"username": "absent"})
        self.assertEqual(self.result(missing)["code"], "CREATE_TASK_NOT_FOUND")
