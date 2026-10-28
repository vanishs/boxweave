import fcntl
import json
import os
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from werkzeug.security import generate_password_hash
from bwmini.domain import Target, Fault
from bwmini.store import Store
from bwmini.service import Service
from bwmini.provider import Fixture, Provider, execution
from bwmini.mcp import MCP
from bwmini.web import create_app

TARGET=Target("containarium","dev","default","fixture-box","fixture-box-container","123","fixture-box","fixture-box-container")
PASSWORD="synthetic-test-password"
HASH=generate_password_hash(PASSWORD)
PEER=str(Path(__file__).with_name("mcp_peer.py").absolute())

class Base(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.root=Path(self.tmp.name)
        self.store=Store(self.root/"state"/"state.sqlite3")
        self.fixture=Fixture(TARGET)
        self.now=[1000.]
        self.s=Service(self.store,self.fixture,clock=lambda:self.now[0])
    def tearDown(self):
        self.tmp.cleanup()
    def bind(self):
        return self.s.bind("My project",TARGET.logical,"/home/fixture-box")
    def operation(self):
        p=self.bind()
        return self.s.confirm(self.s.plan(p["id"],p["revision"])["id"])

class ApplicationTests(Base):
    def test_binding_persists_and_untrusted_path_rejected(self):
        p=self.bind()
        self.assertEqual(Store(self.store.path).project(p["id"])["revision"],1)
        with self.assertRaises(Fault):
            self.s.bind("bad",TARGET.logical,"/home/fixture-box/../../etc")

    def test_revision_conflict_and_rebind_with_existing_plan(self):
        p=self.bind()
        plan=self.s.plan(p["id"],1)
        newer=self.s.bind("Updated",TARGET.logical,"/home/fixture-box",p["id"],1)
        self.assertEqual(newer["revision"],2)
        with self.assertRaisesRegex(Fault,"REVISION_CONFLICT"):
            self.s.confirm(plan["id"])
        with self.assertRaisesRegex(Fault,"REVISION_CONFLICT"):
            self.s.bind("Stale",TARGET.logical,"/home/fixture-box",p["id"],1)

    def test_normal_hello_and_receipt_after_restart_no_replay(self):
        op=self.operation()
        self.assertTrue(self.s.work_one())
        row=self.store.receipt(op["id"])
        self.assertEqual(row["state"],"succeeded")
        self.assertEqual(json.loads(row["result"])["stdout"],"hello world\n")
        restarted=Service(Store(self.store.path),self.fixture,clock=lambda:self.now[0])
        restarted.worker(once=True)
        self.assertEqual(self.fixture.calls,["identity","hello"])
        self.assertEqual(restarted.list_receipts()[0]["id"],op["id"])

    def test_plan_cannot_be_confirmed_twice(self):
        p=self.bind(); plan=self.s.plan(p["id"],1)
        self.s.confirm(plan["id"])
        with self.assertRaisesRegex(Fault,"PLAN_EXPIRED_OR_USED"):
            self.s.confirm(plan["id"])

    def test_plan_expired(self):
        p=self.bind(); plan=self.s.plan(p["id"],1);self.now[0]+=301
        with self.assertRaisesRegex(Fault,"PLAN_EXPIRED_OR_USED"):
            self.s.confirm(plan["id"])

    def test_target_change_before_dispatch_has_zero_hello_calls(self):
        op=self.operation()
        self.fixture.target=Target(**{**TARGET.data(),"created":"changed"})
        self.s.work_one()
        self.assertEqual(self.store.receipt(op["id"])["code"],"TARGET_CHANGED")
        self.assertEqual(self.fixture.calls,[])

    def test_binding_change_after_confirm_has_zero_hello_calls(self):
        op=self.operation()
        self.s.bind("Changed",TARGET.logical,"/home/fixture-box",op["project"],1)
        self.s.work_one()
        self.assertEqual(self.store.receipt(op["id"])["code"],"REVISION_CONFLICT")
        self.assertEqual(self.fixture.calls,[])

    def test_crash_after_dispatch_intent_recovers_unknown_never_replays(self):
        op=self.operation()
        with self.store.db() as db:
            db.execute("UPDATE operations SET state='dispatching' WHERE id=?",(op["id"],))
        self.s.worker(once=True)
        row=self.store.receipt(op["id"])
        self.assertEqual((row["state"],row["code"],row["result"]),("unknown","WORKER_LOST",None))
        self.assertEqual(self.fixture.calls,[])

    def test_response_loss_is_unknown_and_not_replayed(self):
        op=self.operation()
        def lost(*_):
            self.fixture.calls.append("sent")
            raise OSError("synthetic transport loss")
        self.fixture.hello=lost
        self.s.work_one();self.s.worker(once=True)
        row=self.store.receipt(op["id"])
        self.assertEqual(row["state"],"unknown")
        self.assertIsNone(row["result"])
        self.assertEqual(self.fixture.calls,["sent"])

    def test_expired_deadline_and_clock_rollback_do_not_reset(self):
        op=self.operation();self.now[0]+=61
        self.s.work_one()
        self.assertEqual(self.store.receipt(op["id"])["code"],"DEADLINE_EXCEEDED")
        self.assertEqual(self.fixture.calls,[])

    def test_backward_clock_prevents_dispatch(self):
        op=self.operation();self.now[0]=900
        self.s.work_one()
        self.assertEqual(self.store.receipt(op["id"])["state"],"failed")
        self.assertEqual(self.fixture.calls,[])

    def test_worker_lock_prevents_concurrent_recovery(self):
        with open(str(self.store.path)+".worker","a") as lock:
            fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
            with self.assertRaisesRegex(Fault,"WORKER_ALREADY_RUNNING"):
                self.s.worker(once=True)

    def test_unknown_schema_is_not_repaired(self):
        with self.store.db() as db: db.execute("PRAGMA user_version=9")
        with self.assertRaisesRegex(Fault,"SCHEMA_UNSUPPORTED"):
            Store(self.store.path)

    def test_receipt_id_does_not_authorize_another_actor(self):
        op=self.operation()
        with self.assertRaisesRegex(Fault,"NOT_FOUND"):
            self.store.receipt(op["id"],"other")

    def test_malformed_provider_output_is_not_saved_to_receipt(self):
        op=self.operation()
        self.fixture.hello=lambda *_: {"stdout":"synthetic-secret-value","exit_code":0}
        self.s.work_one()
        row=self.store.receipt(op["id"])
        self.assertIsNone(row["result"])
        self.assertNotIn("synthetic-secret-value",json.dumps(row))

    def test_unsafe_ancestor_state_symlink_is_rejected(self):
        real=self.root/"real";real.mkdir()
        link=self.root/"link";link.symlink_to(real,target_is_directory=True)
        with self.assertRaisesRegex(Fault,"UNSAFE_STATE_PATH"):
            Store(link/"nested"/"state.sqlite3")

    def test_invalid_live_config_does_not_create_state(self):
        from bwmini.__main__ import load
        config=self.root/"bad.json";db=self.root/"new-state"/"state.sqlite3"
        config.write_text(json.dumps({"mode":"mcp","database":str(db),"launcher":["/nonexistent/launcher"]}))
        with self.assertRaises(Fault): load(config)
        self.assertFalse(db.parent.exists())

class WebTests(Base):
    def setUp(self):
        super().setUp()
        self.store.admin(HASH)
        self.app=create_app(self.s,"http://localhost",self.root/"dist",secure=False,clock=lambda:self.now[0])
        self.client=self.app.test_client()
        self.headers={"Origin":"http://localhost"}
    def login(self):
        r=self.client.post("/api/login",json={"password":PASSWORD},headers=self.headers)
        self.assertEqual(r.status_code,200)
        self.headers["X-CSRF-Token"]=r.json["csrf"]
        return r

    def test_authentication_origin_csrf_and_cookie(self):
        self.assertEqual(self.client.get("/api/resources").status_code,401)
        self.assertEqual(self.client.post("/api/login",json={"password":PASSWORD}).status_code,403)
        r=self.login()
        self.assertIn("HttpOnly",r.headers["Set-Cookie"])
        self.assertIn("SameSite=Strict",r.headers["Set-Cookie"])
        self.assertEqual(self.client.post("/api/projects",json={},headers={"Origin":"http://localhost"}).status_code,403)
        self.assertEqual(self.client.get("/api/session").json["mode"],"fixture")

    def test_http_binding_preview_confirm_query_logout(self):
        self.login()
        p=self.client.post("/api/projects",json={"name":"UI","logical":TARGET.logical,"workspace":"/home/fixture-box"},headers=self.headers)
        self.assertEqual(p.status_code,201)
        plan=self.client.post("/api/plans",json={"project":p.json["id"],"revision":1},headers=self.headers)
        self.assertEqual(plan.status_code,201)
        r=self.client.post("/api/plans/"+plan.json["id"]+"/confirm",json={"confirm":True},headers=self.headers)
        self.assertEqual(r.status_code,202)
        self.s.worker(once=True)
        self.assertEqual(self.client.get("/api/operations/"+r.json["id"]).json["result"]["evidence"],"fixture")
        self.assertEqual(self.client.post("/api/logout",json={},headers=self.headers).status_code,200)
        self.assertEqual(self.client.get("/api/operations").status_code,401)

    def test_session_expiration_does_not_expose_receipts(self):
        self.login();self.now[0]+=3601
        self.assertEqual(self.client.get("/api/operations").status_code,401)

    def test_login_rate_limit_and_request_limit(self):
        for _ in range(5):
            self.assertEqual(self.client.post("/api/login",json={"password":"wrong"},headers=self.headers).status_code,401)
        self.assertEqual(self.client.post("/api/login",json={"password":PASSWORD},headers=self.headers).status_code,429)
        self.now[0]+=61
        self.assertEqual(self.client.post("/api/login",data='x'*17000,headers=self.headers,content_type="application/json").status_code,413)

    def test_extra_command_field_is_rejected(self):
        self.login()
        r=self.client.post("/api/projects",json={"name":"x","logical":TARGET.logical,"workspace":"/home/fixture-box","command":"anything"},headers=self.headers)
        self.assertEqual(r.status_code,400)
        self.assertEqual(self.fixture.calls,[])

class ProtocolTests(unittest.TestCase):
    def test_pagination_lifecycle_and_real_protocol_fixture(self):
        with MCP([sys.executable,PEER,"paged"]) as m:
            self.assertIn("connect",m.discover())
            result=m.tool("get_container",{"username":"fixture-box"})
            self.assertIn("container",json.loads(result["content"][0]["text"]))

    def test_wrong_id_and_repeated_cursor_rejected(self):
        for mode in ("wrong-id","repeat"):
            with self.subTest(mode=mode),MCP([sys.executable,PEER,mode]) as m:
                with self.assertRaises(Fault): m.discover()

    def test_empty_result_and_tool_error_rejected(self):
        for mode in ("malformed","denied"):
            with self.subTest(mode=mode),MCP([sys.executable,PEER,mode]) as m:
                m.discover()
                with self.assertRaises(Fault): m.tool("get_container",{})

    def provider(self,mode="valid"):
        return Provider({"provider":"containarium","backend":"dev","project":"default",
                         "launcher":[sys.executable,PEER,mode],"targets":{"fixture-box":TARGET.data()}})

    def test_provider_identity_and_hello_using_synthetic_peer(self):
        p=self.provider();self.assertEqual(p.observe("fixture-box"),TARGET)
        self.assertEqual(p.hello(TARGET,"/home/fixture-box",10)["exit_code"],0)

    def test_identity_mismatch_prevents_hello(self):
        with self.assertRaisesRegex(Fault,"IDENTITY_MISMATCH"):
            self.provider("identity").hello(TARGET,"/home/fixture-box",10)

    def test_missing_live_launcher_never_falls_back_to_fixture(self):
        with self.assertRaisesRegex(Fault,"MCP_LAUNCHER_UNAVAILABLE"):
            Provider({"launcher":["/nonexistent/independent-launcher"]})

    def test_guest_markers_do_not_fake_success(self):
        with self.assertRaisesRegex(Fault,"EXECUTION_FORMAT_UNSUPPORTED"):
            execution({"content":[{"text":"guest says exit_code: 0\nhello world"}]})
