"""Receipt discovery is local observation, never infrastructure execution."""
import base64
import json
from pathlib import Path
import subprocess
import sys
from test_slice import Base, TARGET, PASSWORD, HASH
from bwmini.domain import Fault
from bwmini.service import Service
from bwmini.store import Store
from bwmini.web import create_app


class HistoryTests(Base):
    def seed(self, count=125):
        project = self.bind()
        with self.store.db() as db:
            for number in range(count):
                state = "unknown" if number == 0 else "succeeded"
                db.execute("INSERT INTO operations VALUES(?,?,?,?,?,?,?,?,?,?)",
                           (f"{number:032x}", project["id"], 1, project["digest"], "operator",
                            state, 1000., 1060., None, "WORKER_LOST" if number == 0 else "OK"))
            db.execute("INSERT INTO operations VALUES(?,?,?,?,?,?,?,?,?,?)",
                       ("f"*32, project["id"], 1, project["digest"], "someone-else",
                        "succeeded", 2000., 2060., None, "OK"))
        return project

    def test_tied_pages_beyond_100_actor_scope_and_restart_no_replay(self):
        self.seed()
        with self.store.db() as db:
            before = "\n".join(db.iterdump())
        class NoProvider:
            def __getattr__(self, key):
                raise AssertionError("History consulted provider: " + key)
        restarted = Service(Store(self.store.path), NoProvider())
        cursor, found = None, []
        while True:
            page = restarted.receipt_history(limit=17, cursor=cursor)
            found.extend(page["operations"])
            cursor = page["next_cursor"]
            if cursor is None:
                break
        self.assertEqual([item["id"] for item in found],
                         [f"{i:032x}" for i in reversed(range(125))])
        self.assertEqual(found[-1]["state"], "unknown")
        self.assertNotIn("result", found[0])
        self.assertNotIn("digest", found[0])
        self.assertEqual(len(restarted.receipt_history(actor="someone-else")["operations"]), 1)
        self.assertEqual(restarted.receipt_history(actor="nobody"),
                         {"operations": [], "next_cursor": None})
        self.assertEqual(self.store.receipt("0"*32)["code"], "WORKER_LOST")
        with self.assertRaisesRegex(Fault, "NOT_FOUND"):
            self.store.receipt("f"*32)
        with self.store.db() as db:
            self.assertEqual("\n".join(db.iterdump()), before)
        self.assertEqual(self.fixture.calls, [])

    def test_newer_insert_does_not_shift_older_page(self):
        project = self.seed(23)
        first = self.s.receipt_history(limit=10)
        with self.store.db() as db:
            db.execute("INSERT INTO operations VALUES(?,?,?,?,?,?,?,?,?,?)",
                       ("e"*32, project["id"], 1, project["digest"], "operator",
                        "queued", 1001., 1061., None, None))
        found = list(first["operations"])
        cursor = first["next_cursor"]
        while cursor:
            page = self.s.receipt_history(limit=10, cursor=cursor)
            found.extend(page["operations"])
            cursor = page["next_cursor"]
        self.assertEqual([row["id"] for row in found],
                         [f"{i:032x}" for i in reversed(range(23))])
        self.assertEqual(self.s.receipt_history(limit=1)["operations"][0]["id"], "e"*32)
        self.assertEqual(self.fixture.calls, [])

    def test_cursor_and_limit_validation(self):
        bad_cursors = ["", "x"*257, "not base64!", 42, True]
        for value in [[], {"created": True, "id": "0"*32},
                      {"created": float("inf"), "id": "0"*32},
                      {"created": 1, "id": "bad"},
                      {"created": 1, "id": "0"*32, "actor": "someone-else"}]:
            bad_cursors.append(base64.urlsafe_b64encode(json.dumps(value).encode()).decode())
        for cursor in bad_cursors:
            with self.subTest(cursor=cursor), self.assertRaisesRegex(Fault, "INVALID_CURSOR"):
                self.s.receipt_history(cursor=cursor)
        for limit in [0, 101, True, "20", 1.5]:
            with self.subTest(limit=limit), self.assertRaisesRegex(Fault, "INVALID_LIMIT"):
                self.s.receipt_history(limit=limit)
        self.assertEqual(self.s.receipt_history(limit=100),
                         {"operations": [], "next_cursor": None})

    def test_authenticated_history_invalid_queries_and_expiry(self):
        self.seed(23)
        self.store.admin(HASH)
        client = create_app(self.s, "http://localhost", self.root, secure=False,
                            clock=lambda:self.now[0]).test_client()
        self.assertEqual(client.get("/api/operation-history").status_code, 401)
        login = client.post("/api/login", json={"password": PASSWORD},
                            headers={"Origin": "http://localhost"})
        self.assertEqual(login.status_code, 200)
        first = client.get("/api/operation-history?limit=10")
        self.assertEqual(first.status_code, 200)
        self.assertEqual(len(first.json["operations"]), 10)
        second = client.get("/api/operation-history",
                            query_string={"limit": 10, "cursor": first.json["next_cursor"]})
        self.assertFalse(set(x["id"] for x in first.json["operations"]) &
                         set(x["id"] for x in second.json["operations"]))
        for query in ["limit=0", "limit=101", "limit=-1", "limit=true",
                      "limit=999999999", "limit=1&limit=2", "other=1",
                      "cursor=x&cursor=y", "cursor=bad"]:
            with self.subTest(query=query):
                self.assertEqual(client.get("/api/operation-history?" + query).status_code, 400)
        self.assertEqual(client.get("/api/operations/" + "f"*32).status_code, 404)
        self.now[0] += 3601
        self.assertEqual(client.get("/api/operation-history").status_code, 401)
        self.assertEqual(self.fixture.calls, [])

    def test_real_cli_history_and_original_receipts_contract(self):
        self.seed(23)
        config = self.root / "config.json"
        config.write_text(json.dumps({"mode": "fixture", "database": str(self.store.path),
                                     "target": TARGET.data()}))
        def run(*args):
            return subprocess.run([sys.executable, "-m", "bwmini", *args, "--config", str(config)],
                                  capture_output=True, text=True, timeout=10)
        first = run("history", "--limit", "10")
        self.assertEqual(first.returncode, 0, first.stdout)
        page = json.loads(first.stdout)
        self.assertEqual(len(page["operations"]), 10)
        second = run("history", "--limit", "10", "--cursor", page["next_cursor"])
        self.assertEqual(second.returncode, 0)
        self.assertNotEqual(json.loads(second.stdout)["operations"][0]["id"], page["operations"][0]["id"])
        legacy = run("receipts")
        self.assertEqual(legacy.returncode, 0)
        self.assertIsInstance(json.loads(legacy.stdout), list)
        invalid = run("history", "--limit", "101")
        self.assertEqual(invalid.returncode, 2)
        self.assertEqual(json.loads(invalid.stdout)["code"], "INVALID_LIMIT")
        self.assertEqual(run("projects", "--cursor", "bad").returncode, 2)
