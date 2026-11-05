"""P1-B2: shared claim protects real app lifecycle dispatch and observation."""
import hashlib
import json
import tempfile
import unittest
from unittest.mock import patch

from bwmini.coordination_bridge import SharedClaims, app_box
from bwmini.coordination_runtime import private_connection, scope_digest
from bwmini.operation_coordination import (
    OperationCoordinator, CoordinationFault, box_resource, name_resource)
from bwmini.service import Service
from test_slice import Base
from test_lifecycle import LifecycleFixture, NAME

SCOPE={"provider":"containarium","backend":"dev","project":"default"}


class LifecycleClaimTests(Base):
    def setUp(self):
        super().setUp()
        self.provider=LifecycleFixture()
        self.provider.config.update(SCOPE)
        self.path=self.root/"shared-claims.sqlite3"
        self.path.touch(mode=0o600)
        OperationCoordinator(lambda:private_connection(self.path)).initialize()
        self.claims=SharedClaims(self.path,clock=lambda:self.now[0])
        self.gateway=SharedClaims(self.path,clock=lambda:self.now[0])
        self.s=Service(self.store,self.provider,clock=lambda:self.now[0],
                       shared_claims=self.claims)
        self.life=self.s.lifecycle
        self.mock_scope=patch("bwmini.coordination_bridge.published_scope",
                              return_value=SCOPE)
        self.mock_scope.start()
        self.addCleanup(self.mock_scope.stop)

    def submit(self,action="create",resize=None):
        p=self.life.plan(action,NAME,resize=resize)
        return self.life.confirm(p["id"],NAME)["id"]

    def run_action(self,action="create",resize=None):
        ident=self.submit(action,resize)
        self.assertTrue(self.s.work_one())
        return ident,self.life.receipt(ident)

    def gateway_claim_name(self,logical,oid="a"*32):
        scope=scope_digest(**SCOPE)
        digest=hashlib.sha256(("gateway:"+logical).encode()).hexdigest()
        self.gateway.coordinator.acquire(oid,digest,"gateway_"+oid,
            [name_resource(scope,logical)])

    def test_create_claim_precedes_provider_write_and_releases_after_observed(self):
        original=self.provider.lifecycle_mutate
        seen=[]
        def during(action,logical,spec,timeout):
            active=self.life.receipts()[0]["id"]
            seen.append(self.claims.coordinator.status(active)["state"])
            with self.assertRaisesRegex(CoordinationFault,"RESOURCE_BUSY"):
                self.gateway_claim_name(NAME)
            return original(action,logical,spec,timeout)
        self.provider.lifecycle_mutate=during
        oid,result=self.run_action()
        self.assertEqual(result["state"],"succeeded")
        self.assertEqual(seen,["dispatch_intent"])
        self.assertEqual(self.gateway.coordinator.status(oid)["state"],
                         "observed_terminal")
        self.gateway_claim_name(NAME)
        self.assertEqual(self.provider.writes,[("create",NAME)])

    def test_complete_lifecycle_releases_only_observed_original_receipts(self):
        actions=("create","stop","start","stop","delete")
        for action in actions:
            with self.subTest(action=action):
                oid,result=self.run_action(action)
                self.assertEqual((result["state"],result["code"]),("succeeded","OK"))
                self.assertEqual(self.gateway.coordinator.status(oid)["state"],
                                 "observed_terminal")
        self.assertEqual([item[0] for item in self.provider.writes],list(actions))
        self.assertTrue(self.life.settings()["environments"][0]["deleted"])

    def test_existing_gateway_name_claim_denies_create_before_provider(self):
        self.gateway_claim_name(NAME)
        oid,result=self.run_action()
        self.assertEqual((result["state"],result["code"]),("failed","RESOURCE_BUSY"))
        self.assertEqual(self.provider.writes,[])
        self.assertEqual(self.gateway.coordinator.status("a"*32)["state"],"accepted")

    def test_lost_create_response_never_repeats_and_keeps_name(self):
        self.provider.lose_create=True
        oid,result=self.run_action()
        self.assertEqual(result["state"],"unknown")
        self.assertEqual(self.claims.coordinator.status(oid)["state"],"unknown")
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.gateway_claim_name(NAME)
        self.life.recover()
        self.s.worker(once=True)
        self.assertEqual(self.provider.writes,[("create",NAME)])

    def test_upstream_rejected_after_create_keeps_shared_claim_unknown(self):
        original = self.provider.lifecycle_mutate
        from bwmini.domain import Fault
        def created_then_error(action, logical, spec, timeout):
            original(action, logical, spec, timeout)
            raise Fault("PROVIDER_REJECTED")
        self.provider.lifecycle_mutate = created_then_error
        oid,receipt=self.run_action()
        self.assertEqual((receipt["state"], receipt["code"]),
                         ("unknown", "OUTCOME_UNKNOWN"))
        self.assertEqual(self.claims.coordinator.status(oid)["state"], "unknown")
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.gateway_claim_name(NAME)
        self.life.observe(oid)
        self.assertEqual(self.life.receipt(oid)["state"], "unknown")
        self.assertEqual(self.provider.writes, [("create", NAME)])

    def test_observation_pending_retains_claim_until_verified(self):
        self.provider.ready_failures=1
        oid,result=self.run_action()
        self.assertEqual(result["state"],"observing")
        self.assertEqual(self.gateway.coordinator.status(oid)["state"],"dispatch_intent")
        with self.assertRaisesRegex(CoordinationFault,"RESOURCE_BUSY"):
            self.gateway_claim_name(NAME)
        self.now[0]+=16
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(oid)["state"],"succeeded")
        self.assertEqual(self.gateway.coordinator.status(oid)["state"],"observed_terminal")
        self.gateway_claim_name(NAME)

    def test_expired_queued_action_never_claims_or_dispatches(self):
        oid = self.submit()
        row = self.life.receipt(oid)
        self.now[0] = row["deadline"] + 1
        self.assertTrue(self.s.work_one())
        result = self.life.receipt(oid)
        self.assertEqual((result["state"], result["code"]),
                         ("failed", "DEADLINE_EXCEEDED"))
        with self.assertRaisesRegex(CoordinationFault, "OPERATION_NOT_FOUND"):
            self.claims.coordinator.status(oid)
        self.gateway_claim_name(NAME)
        self.assertEqual(self.provider.writes, [])

    def test_observation_deadline_classifies_claim_as_unknown(self):
        # A dispatched, acknowledged create must retain its name claim after
        # deadline; other agents receive RECOVERY_REQUIRED, not RESOURCE_BUSY.
        self.provider.ready_failures = 100
        oid, row = self.run_action()
        self.assertEqual(row["state"], "observing")
        self.assertEqual(self.claims.coordinator.status(oid)["state"], "dispatch_intent")
        self.now[0] = row["deadline"] + 1
        self.assertTrue(self.s.work_one())
        self.assertEqual(self.life.receipt(oid)["state"], "unknown")
        self.assertEqual(self.claims.coordinator.status(oid)["state"], "unknown")
        with self.assertRaisesRegex(CoordinationFault, "RECOVERY_REQUIRED"):
            self.gateway_claim_name(NAME)
        self.assertEqual(self.provider.writes, [("create", NAME)])

    def test_resize_unverified_keeps_claim_not_false_release(self):
        self.run_action()
        self.provider.no_resources=True
        oid,result=self.run_action("resize",{"cpu":"2"})
        self.assertEqual(result["state"],"succeeded")
        self.assertEqual(result["result"]["verification"],"not_available")
        self.assertEqual(self.gateway.coordinator.status(oid)["state"],"unknown")
        target=self.provider.resources[NAME]["target"]
        identity,scope,logical=app_box(target,SCOPE)
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.gateway.acquire("e"*32,"gateway_e",identity,scope,logical)

    def test_existing_managed_stop_conflicts_on_box_identity(self):
        self.run_action()
        target=self.provider.resources[NAME]["target"]
        identity,scope,logical=app_box(target,SCOPE)
        self.gateway.acquire("a"*32,"gateway_a",identity,scope,logical)
        oid,result=self.run_action("stop")
        self.assertEqual((result["state"],result["code"]),("failed","RESOURCE_BUSY"))
        self.assertEqual(self.provider.writes,[("create",NAME)])

    def test_unprovisioned_store_fails_before_create(self):
        self.path.unlink()
        oid,result=self.run_action()
        self.assertEqual(result["state"],"failed")
        self.assertEqual(result["code"],"COORDINATION_UNAVAILABLE")
        self.assertEqual(self.provider.writes,[])
        self.assertFalse(self.path.exists())

    def test_untrusted_scope_fails_before_create(self):
        with patch("bwmini.coordination_bridge.published_scope",
                   return_value={**SCOPE,"backend":"different"}):
            oid,result=self.run_action()
        self.assertEqual(result["code"],"COORDINATION_SCOPE_MISMATCH")
        self.assertEqual(self.provider.writes,[])

    def test_independent_other_box_name_does_not_block_create(self):
        self.gateway_claim_name("other-box")
        oid,result=self.run_action()
        self.assertEqual(result["state"],"succeeded")
        self.assertEqual(self.gateway.coordinator.status("a"*32)["state"],"accepted")


if __name__=="__main__":
    unittest.main()
