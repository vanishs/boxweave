"""E1 immutable app-owned fingerprint capture, actor and Web auth tests."""
import copy
import hashlib
import json
from pathlib import Path
import sqlite3
import tempfile
import unittest

from werkzeug.security import generate_password_hash
from bwmini.domain import Fault, Target
from bwmini.store import Store
from bwmini.provider import Fixture
from bwmini.service import Service
from bwmini.web import create_app
from bwmini.environment_observe import observe_exact
from bwmini.environment_history import (
    save_capture, read_capture, list_captures, compare_captures,
)

TARGET=Target("containarium","dev","default","fixture-box",
              "fixture-box-container","123","fixture-box","fixture-box-container")
CONFIG={"provider":"containarium","backend":"dev","project":"default",
        "targets":{"fixture-box":TARGET.data()}}
CONTAINER={"container":{"username":"fixture-box","name":"fixture-box-container",
            "createdAt":"123","state":"CONTAINER_STATE_RUNNING",
            "networkTenant":"test",
            "resources":{"cpu":"1","memory":"1GB","disk":"10GB"}}}

def report():
    return observe_exact(CONFIG,"fixture-box",CONTAINER,{},observed_at=10)

class EnvironmentHistoryTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)
        self.store=Store(self.root/"state"/"state.sqlite3")

    def test_private_v13_schema_and_older_migration(self):
        with self.store.db() as db:
            self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0],13)
            db.execute("DROP TABLE environment_captures")
            db.execute("PRAGMA user_version=12")
        Store(self.store.path)
        with self.store.db() as db:
            self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0],13)
            self.assertIsNotNone(db.execute("SELECT name FROM sqlite_master WHERE "
                                            "name='environment_captures'").fetchone())

    def test_save_get_list_diff_immutable(self):
        original=report()
        a=save_capture(self.store,original,clock=lambda:100)
        self.assertEqual(a["fingerprint_sha256"],original["fingerprint_sha256"])
        self.assertEqual(len(list_captures(self.store)),1)
        changed=copy.deepcopy(CONTAINER)
        changed["container"]["resources"]["memory"]="2GB"
        b=save_capture(self.store,observe_exact(CONFIG,"fixture-box",changed,{}),
                       clock=lambda:200)
        result=compare_captures(self.store,a["capture_id"],b["capture_id"])
        self.assertEqual(set(result["changes"]),{"memory"})
        self.assertEqual(len(list_captures(self.store)),2)
        original["observation"]["memory"]="999GB"
        self.assertEqual(read_capture(self.store,a["capture_id"])["observation"]["memory"],"1GB")

    def test_protect_secret_and_actor_and_foreign_identity(self):
        value=report()
        value["private_token"]="DO_NOT_STORE_ABC"
        a=save_capture(self.store,value)
        self.assertNotIn(b"DO_NOT_STORE_ABC",self.store.path.read_bytes())
        with self.assertRaisesRegex(Fault,"ENVIRONMENT_ACTOR_NOT_ALLOWED"):
            read_capture(self.store,a["capture_id"],actor="other")
        with self.assertRaisesRegex(Fault,"ENVIRONMENT_ACTOR_NOT_ALLOWED"):
            save_capture(self.store,value,actor="other")
        foreign=Target("containarium","dev","default","other-box",
                       "other-box-container","777","other-box","other-box-container")
        config={**CONFIG,"targets":{"other-box":foreign.data()}}
        observed={"container":{**CONTAINER["container"],
                  "username":"other-box","name":"other-box-container","createdAt":"777"}}
        other=observe_exact(config,"other-box",observed,{})
        b=save_capture(self.store,other)
        with self.assertRaisesRegex(Fault,"ENVIRONMENT_TARGET_CHANGED"):
            compare_captures(self.store,a["capture_id"],b["capture_id"])

    def test_reject_arbitrary_capture_selector_and_future_schema(self):
        with self.assertRaises(Fault):
            read_capture(self.store,"../../state")
        with self.assertRaises(Fault):
            list_captures(self.store,limit=True)
        with sqlite3.connect(self.store.path) as db:
            db.execute("PRAGMA user_version=14")
        with self.assertRaisesRegex(Fault,"SCHEMA_UNSUPPORTED"):
            Store(self.store.path)

    def test_web_operator_auth_capture_and_compare(self):
        self.store.admin(generate_password_hash("test-only-password"))
        fixture=Fixture(TARGET)
        fixture.environment_report=lambda logical: report()
        service=Service(self.store,fixture,clock=lambda:1000)
        app=create_app(service,"http://localhost",self.root/"dist",secure=False,
                       clock=lambda:1000)
        client=app.test_client()
        self.assertEqual(client.get("/api/boxes/fixture-box/environment").status_code,401)
        headers={"Origin":"http://localhost"}
        login=client.post("/api/login",json={"password":"test-only-password"},headers=headers)
        self.assertEqual(login.status_code,200)
        headers["X-CSRF-Token"]=login.json["csrf"]
        self.assertEqual(client.get("/api/boxes/fixture-box/environment").status_code,200)
        denied=client.post("/api/boxes/fixture-box/environment/capture",json={"token":"x"},headers=headers)
        self.assertEqual(denied.status_code,400)
        saved=client.post("/api/boxes/fixture-box/environment/capture",json={},headers=headers)
        self.assertEqual(saved.status_code,200,saved.json)
        saved_id=saved.json["capture_id"]
        listing=client.get("/api/environment-captures")
        self.assertEqual(len(listing.json["captures"]),1)
        left=client.get("/api/environment-captures/"+saved_id)
        self.assertEqual(left.json["observation"]["cpu"],"1")
        compare=client.post("/api/environment-captures/compare",
                            json={"left":saved_id,"right":saved_id},headers=headers)
        self.assertEqual(compare.status_code,200,compare.json)
        self.assertEqual(compare.json["changes"],{})
        self.assertEqual(client.post("/api/boxes/fixture-box/environment/capture",
                                    json={}).status_code,403)
