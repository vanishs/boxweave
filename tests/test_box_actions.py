"""M4/M5: durable Box action plans with no replay after uncertain dispatch."""
import sqlite3
import unittest
from unittest.mock import patch
from bwmini.domain import Fault
from bwmini.box_actions import BoxActions
from bwmini.auto_sleep import validate as validate_sleep
from test_slice import Base, TARGET, WebTests


class MockAdapter:
    allowed=frozenset(("shell_exec","write_file"))
    calls=[]
    def __init__(self,service,logical):
        if logical!=TARGET.logical:raise Fault("TARGET_NOT_ALLOWED",403)
        self.target=TARGET
    def validate(self,name,args):
        if name not in self.allowed:raise Fault("AGENT_BOX_TOOL_DENIED",403)
        if name=="shell_exec" and "command" not in args:raise Fault("COMMAND_REQUIRED",422)
    def execute(self,name,args,timeout=30):
        self.calls.append((name,args))
        return {"summary":"upstream_completed"}


class ActionTests(Base):
    def test_plan_confirm_dispatch_once_and_private_receipt(self):
        MockAdapter.calls=[]
        with patch("bwmini.box_actions.BoxAgent",MockAdapter):
            plan=self.s.box_actions.plan(TARGET.logical,"shell_exec",
                 {"command":"echo private-token","cwd":"/home/fixture-box"})
            self.assertNotIn("private-token",str(plan))
            receipt=self.s.box_actions.confirm(plan["id"],TARGET.logical)
            self.assertEqual(receipt["state"],"queued")
            with self.assertRaises(Fault):
                self.s.box_actions.confirm(plan["id"],TARGET.logical)
            self.assertTrue(self.s.box_actions.work_one())
            self.assertFalse(self.s.box_actions.work_one())
            saved=self.s.box_actions.receipt(receipt["id"])
            self.assertEqual(saved["state"],"succeeded")
            self.assertNotIn("private-token",str(saved))
            self.assertEqual(len(MockAdapter.calls),1)

    def test_uncertain_dispatch_never_replays(self):
        MockAdapter.calls=[]
        with patch("bwmini.box_actions.BoxAgent",MockAdapter):
            plan=self.s.box_actions.plan(TARGET.logical,"shell_exec",{"command":"true"})
            rec=self.s.box_actions.confirm(plan["id"],TARGET.logical)
            with self.store.db() as db:
                db.execute("UPDATE box_action_receipts SET state='dispatching' WHERE id=?",
                           (rec["id"],))
            self.s.box_actions.recover()
            self.assertEqual(self.s.box_actions.receipt(rec["id"])["state"],"unknown")
            self.assertFalse(self.s.box_actions.work_one())
            self.assertFalse(MockAdapter.calls)

    def test_v11_schema_and_bad_sleep_params(self):
        with self.store.db() as db:
            self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0],11)
        for args in ({"enabled":1,"idleThresholdMinutes":30},
                     {"enabled":True,"idleThresholdMinutes":0},
                     {"enabled":True,"idleThresholdMinutes":999999}):
            with self.assertRaises(Fault):validate_sleep(args)
        self.assertTrue(validate_sleep({"enabled":True,"idleThresholdMinutes":30})["enabled"])


class ActionWebTests(WebTests):
    def test_new_api_denies_unenrolled_fixture_and_requires_login(self):
        path="/api/box-action-plans"
        self.assertEqual(self.client.post(path,json={},headers={"Origin":"http://localhost"}).status_code,401)
        self.login()
        response=self.client.post(path,headers=self.headers,json={
            "logical":TARGET.logical,"action":"shell_exec","arguments":{"command":"true"}})
        self.assertEqual(response.status_code,409)
        result=self.client.get("/api/box-action-receipts")
        self.assertEqual(result.status_code,200)
        self.assertEqual(result.json["operations"],[])


if __name__=="__main__":
    unittest.main()
