"""Real app BoxAction dispatch vs the same SSH/HTTP gateway SQLite resource."""

import json
import sqlite3
import unittest
from unittest.mock import patch

from bwmini import coordination_bridge
from bwmini.coordination_runtime import private_connection
from bwmini.operation_coordination import OperationCoordinator, CoordinationFault
from bwmini.domain import Fault, Target
from bwmini.__main__ import load
from bwmini.service import Service
from test_slice import Base, TARGET
from test_box_actions import MockAdapter

SCOPE={"provider":"containarium","backend":"dev","project":"default"}
OTHER=Target("containarium","dev","default","other-box",
             "other-box-container","456","other-box","other-box-container")


class CoordinatedBoxActionTests(Base):
    def setUp(self):
        super().setUp()
        self.claim_path=self.root/"claims.sqlite3"
        self.claim_path.touch(mode=0o600)
        OperationCoordinator(lambda:private_connection(self.claim_path)).initialize()
        self.gateway=coordination_bridge.SharedClaims(
            self.claim_path,clock=lambda:self.now[0])
        self.s.shared_claims=coordination_bridge.SharedClaims(
            self.claim_path,clock=lambda:self.now[0])
        self.scope=patch("bwmini.coordination_bridge.published_scope",return_value=SCOPE)
        self.scope.start()
        self.addCleanup(self.scope.stop)

    def submit(self,command="echo safe"):
        p=self.s.box_actions.plan(TARGET.logical,"shell_exec",
                                  {"command":command})
        return self.s.box_actions.confirm(p["id"],TARGET.logical)["id"]

    def test_original_web_receipt_commits_before_release(self):
        MockAdapter.calls=[]
        ident,scope,logical=coordination_bridge.app_box(TARGET,SCOPE)
        real_execute=MockAdapter.execute

        def during_dispatch(adapter,action,args,timeout=30):
            self.assertEqual(self.gateway.coordinator.status(self.active)["state"],
                             "dispatch_intent")
            with self.assertRaisesRegex(CoordinationFault,"RESOURCE_BUSY"):
                self.gateway.acquire("d"*32,"gateway_d",ident,scope,logical)
            return real_execute(adapter,action,args,timeout=timeout)

        with patch("bwmini.box_actions.BoxAgent",MockAdapter), \
             patch.object(MockAdapter,"execute",during_dispatch):
            self.active=self.submit()
            self.assertTrue(self.s.box_actions.work_one())
        self.assertEqual(self.s.box_actions.receipt(self.active)["state"],"succeeded")
        self.assertEqual(len(MockAdapter.calls),1)
        self.gateway.acquire("d"*32,"gateway_d",ident,scope,logical)
        self.assertEqual(self.gateway.coordinator.status("d"*32)["state"],"accepted")

    def test_gateway_claim_blocks_web_before_guest_write(self):
        ident,scope,logical=coordination_bridge.app_box(TARGET,SCOPE)
        self.gateway.acquire("a"*32,"gateway_a",ident,scope,logical)
        MockAdapter.calls=[]
        with patch("bwmini.box_actions.BoxAgent",MockAdapter):
            rid=self.submit()
            self.assertTrue(self.s.box_actions.work_one())
        result=self.s.box_actions.receipt(rid)
        self.assertEqual((result["state"],result["code"]),("failed","RESOURCE_BUSY"))
        self.assertEqual(MockAdapter.calls,[])

    def test_gateway_unknown_stays_busy_after_recovery(self):
        ident,scope,logical=coordination_bridge.app_box(TARGET,SCOPE)
        self.gateway.acquire("a"*32,"gateway_a",ident,scope,logical)
        self.gateway.dispatch_intent("a"*32)
        self.gateway.unknown("a"*32)
        MockAdapter.calls=[]
        with patch("bwmini.box_actions.BoxAgent",MockAdapter):
            rid=self.submit()
            self.s.box_actions.work_one()
        self.assertEqual(self.s.box_actions.receipt(rid)["code"],"RECOVERY_REQUIRED")
        self.assertEqual(MockAdapter.calls,[])

    def test_web_lost_remote_reply_never_replays_or_releases(self):
        ident,scope,logical=coordination_bridge.app_box(TARGET,SCOPE)
        MockAdapter.calls=[]
        with patch("bwmini.box_actions.BoxAgent",MockAdapter), \
             patch.object(MockAdapter,"execute",side_effect=TimeoutError("lost")):
            rid=self.submit()
            self.s.box_actions.work_one()
        self.assertEqual(self.s.box_actions.receipt(rid)["state"],"unknown")
        self.assertEqual(self.gateway.coordinator.status(rid)["state"],"unknown")
        self.assertFalse(self.s.box_actions.work_one())
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.gateway.acquire("d"*32,"gateway_d",ident,scope,logical)

    def test_different_box_parallel_with_web_claim(self):
        ident,scope,logical=coordination_bridge.app_box(TARGET,SCOPE)
        other,same_scope,name=coordination_bridge.app_box(OTHER,SCOPE)
        self.gateway.acquire("a"*32,"gateway_a",other,same_scope,name)
        with patch("bwmini.box_actions.BoxAgent",MockAdapter):
            rid=self.submit()
            self.assertTrue(self.s.box_actions.work_one())
        self.assertEqual(self.s.box_actions.receipt(rid)["state"],"succeeded")

    def test_unprovisioned_db_fails_closed_and_preserves_original_receipt(self):
        self.claim_path.unlink()
        MockAdapter.calls=[]
        with patch("bwmini.box_actions.BoxAgent",MockAdapter):
            rid=self.submit()
            self.s.box_actions.work_one()
        saved=self.s.box_actions.receipt(rid)
        self.assertEqual(saved["state"],"failed")
        self.assertIn(saved["code"],("COORDINATION_UNAVAILABLE","COORDINATION_STORAGE_UNSAFE"))
        self.assertFalse(MockAdapter.calls)
        self.assertFalse(self.claim_path.exists())

    def test_scope_mismatch_refuses_without_write(self):
        MockAdapter.calls=[]
        with patch("bwmini.coordination_bridge.published_scope",return_value={
                 "provider":"containarium","backend":"different","project":"default"}), \
             patch("bwmini.box_actions.BoxAgent",MockAdapter):
            rid=self.submit()
            self.s.box_actions.work_one()
        self.assertEqual(self.s.box_actions.receipt(rid)["code"],"COORDINATION_SCOPE_MISMATCH")
        self.assertFalse(MockAdapter.calls)

    def test_acknowledged_autosleep_retains_claim_for_explicit_resolution(self):
        self.s.provider.mode = "mcp"
        with patch("bwmini.auto_sleep.execute",
                   return_value={"result":"upstream_acknowledged"}):
            plan=self.s.box_actions.plan(TARGET.logical,"toggle_auto_sleep",
                 {"enabled":True,"idleThresholdMinutes":30})
            rid=self.s.box_actions.confirm(plan["id"],TARGET.logical)["id"]
            self.s.box_actions.work_one()
        self.assertEqual(self.s.box_actions.receipt(rid)["state"],"acknowledged")
        self.assertEqual(self.gateway.coordinator.status(rid)["state"],"unknown")
        ident,scope,logical=coordination_bridge.app_box(TARGET,SCOPE)
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.gateway.acquire("e"*32,"gateway_e",ident,scope,logical)

    def test_opaque_task_receipt_does_not_expose_command(self):
        with patch("bwmini.box_actions.BoxAgent",MockAdapter):
            rid=self.submit("echo secret-should-not-surface")
            self.s.box_actions.work_one()
        visible=self.s.box_actions.receipt(rid)
        self.assertNotIn("secret-should-not-surface",json.dumps(visible))

    def test_invalid_app_coordination_config_refuses(self):
        base={"mode":"fixture","target":TARGET.data(),
              "database":str(self.root/"alternate-state.sqlite3")}
        with patch("bwmini.__main__.read_config",return_value=base|{
            "coordination":{"enabled":"maybe"}}), self.assertRaisesRegex(
                Fault,"COORDINATION_CONFIG_INVALID"):
            load("fixture")
        with patch("bwmini.__main__.read_config",return_value=base|{
            "coordination":{"enabled":True}}), self.assertRaisesRegex(
                Fault,"COORDINATION_LIVE_PROVIDER_REQUIRED"):
            load("fixture")

if __name__=="__main__":
    unittest.main()
