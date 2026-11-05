"""Fixture-only crash/concurrency checks for the Phase 1 shared claim core."""
import hashlib
import os
import sqlite3
import tempfile
import threading
import unittest

from bwmini.operation_coordination import (
    CoordinationFault, OperationCoordinator, box_resource, name_resource,
    port_resource, tenant_resource,
)

SCOPE = hashlib.sha256(b"provider/project").hexdigest()
A = hashlib.sha256(b"box-a/incarnation-1").hexdigest()
B = hashlib.sha256(b"box-b/incarnation-1").hexdigest()


class CoordinationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        path = os.path.join(self.tmp.name, "coord.sqlite")
        self.store = OperationCoordinator(lambda: sqlite3.connect(path, timeout=5))
        self.store.initialize()

    def op(self, n):
        return f"{n:032x}"

    def assert_code(self, code, call):
        with self.assertRaises(CoordinationFault) as error:
            call()
        self.assertEqual(error.exception.code, code)

    def test_resources_and_scope(self):
        self.assertEqual(box_resource(A), "box:" + A)
        self.assertEqual(name_resource(SCOPE, "box-a"), "name:" + SCOPE + ":box-a")
        self.assertEqual(tenant_resource(SCOPE, "tenant-a"), "tenant:" + SCOPE + ":tenant-a")
        self.assertEqual(port_resource(SCOPE, "tcp", 443), "port:" + SCOPE + ":tcp:443")
        for invalid in ("/etc/passwd", "../box", "BOX", "x:extra"):
            self.assert_code("COORDINATION_RESOURCE_INVALID",
                             lambda v=invalid: name_resource(SCOPE, v))
        self.assert_code("COORDINATION_RESOURCE_INVALID",
                         lambda: port_resource(SCOPE, "tcp", True))

    def test_cannot_dispatch_twice(self):
        self.store.acquire(self.op(1), A, "web_1", [box_resource(A)])
        self.assert_code("OPERATION_EXISTS",
                         lambda: self.store.acquire(self.op(1), A, "web_1", [box_resource(A)]))
        self.store.dispatch_intent(self.op(1))
        self.assert_code("OPERATION_STATE_CONFLICT",
                         lambda: self.store.dispatch_intent(self.op(1)))
        self.assert_code("OPERATION_STATE_CONFLICT",
                         lambda: self.store.cancel_before_dispatch(self.op(1)))

    def test_dispatch_intent_requires_original_outcome_reconciliation(self):
        # A killed gateway can leave the shared claim in dispatch_intent
        # while its durable task receipt is surfaced as UNKNOWN.
        self.store.acquire(self.op(1), A, "gateway_1", [box_resource(A)])
        self.store.dispatch_intent(self.op(1))
        self.assert_code("RECOVERY_REQUIRED",
                         lambda: self.store.acquire(self.op(2), A, "gateway_2", [box_resource(A)]))
        self.assertEqual(self.store.status(self.op(1))["state"], "dispatch_intent")
        self.assertEqual(self.store.status(self.op(1))["resources"], [box_resource(A)])

    def test_unknown_is_not_lease_expiry_or_failure(self):
        self.store.acquire(self.op(1), A, "mcp_1", [box_resource(A)])
        self.store.dispatch_intent(self.op(1))
        self.assertEqual(self.store.recover_dispatched(), 1)
        self.assertEqual(self.store.status(self.op(1))["state"], "unknown")
        self.assert_code("RECOVERY_REQUIRED",
                         lambda: self.store.acquire(self.op(2), A, "mcp_2", [box_resource(A)]))
        self.assert_code("OPERATION_STATE_CONFLICT",
                         lambda: self.store.observed_terminal(self.op(1), "other"))
        self.store.observed_terminal(self.op(1), "mcp_1")
        self.store.acquire(self.op(2), A, "mcp_2", [box_resource(A)])

    def test_pre_dispatch_cancellation_only(self):
        self.store.acquire(self.op(1), A, "receipt_1", [box_resource(A)])
        self.store.cancel_before_dispatch(self.op(1))
        self.assertEqual(self.store.status(self.op(1))["resources"], [])
        self.assert_code("OPERATION_STATE_CONFLICT",
                         lambda: self.store.observed_terminal(self.op(1), "receipt_1"))
        self.store.acquire(self.op(2), A, "receipt_2", [box_resource(A)])

    def test_no_partial_multiresource_acquire(self):
        self.store.acquire(self.op(1), A, "receipt_1", [name_resource(SCOPE, "box-a")])
        self.assert_code("RESOURCE_BUSY", lambda: self.store.acquire(
            self.op(2), A, "receipt_2", [port_resource(SCOPE, "tcp", 8080),
                                          name_resource(SCOPE, "box-a")]))
        self.store.acquire(self.op(3), B, "receipt_3", [port_resource(SCOPE, "tcp", 8080)])
        self.assert_code("OPERATION_NOT_FOUND", lambda: self.store.status(self.op(2)))

    def test_unrelated_boxes_parallel(self):
        self.store.acquire(self.op(1), A, "mcp_a", [box_resource(A)])
        self.store.acquire(self.op(2), B, "mcp_b", [box_resource(B)])
        self.assertEqual(self.store.status(self.op(2))["state"], "accepted")

    def test_connections_compete_atomically(self):
        start = threading.Barrier(3)
        answers = []
        lock = threading.Lock()

        def contender(n):
            other = OperationCoordinator(self.store.connect)
            start.wait()
            try:
                other.acquire(self.op(n), A, f"receipt_{n}", [box_resource(A)])
                value = "acquired"
            except CoordinationFault as error:
                value = error.code
            with lock:
                answers.append(value)

        threads = [threading.Thread(target=contender, args=(n,)) for n in (1, 2)]
        for thread in threads:
            thread.start()
        start.wait()
        for thread in threads:
            thread.join(timeout=6)
            self.assertFalse(thread.is_alive())
        self.assertCountEqual(answers, ["acquired", "RESOURCE_BUSY"])

    def test_closed_store_fails_before_claim(self):
        unavailable = OperationCoordinator(
            lambda: sqlite3.connect(os.path.join(self.tmp.name, "not-a-dir", "x.db")))
        self.assert_code("COORDINATION_UNAVAILABLE",
                         lambda: unavailable.acquire(self.op(1), A, "receipt_1", [box_resource(A)]))

    def test_invalid_identity_and_resource_are_refused(self):
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: self.store.acquire(self.op(1), "changed", "r1", [box_resource(A)]))
        self.assert_code("COORDINATION_RESOURCE_INVALID",
                         lambda: self.store.acquire(self.op(1), A, "r1",
                                                    [box_resource(A), box_resource(A)]))
        self.assert_code("COORDINATION_RESOURCE_INVALID",
                         lambda: self.store.acquire(self.op(1), A, "r1", ["bad:resource"]))



class RecoveryClosureTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.db=os.path.join(self.tmp.name,"claims.sqlite3")
        self.time=[1000]
        self.coordinator=OperationCoordinator(
            lambda:sqlite3.connect(self.db,timeout=5),clock=lambda:self.time[0])
        self.coordinator.initialize()
        self.op_id="a"*32
        self.box=box_resource(A)
        self.name=name_resource(SCOPE,"box-a")
        self.coordinator.acquire(self.op_id,A,"gateway_"+self.op_id,[self.box,self.name])
        self.coordinator.dispatch_intent(self.op_id)

    def check_error(self,code,fn):
        with self.assertRaises(CoordinationFault) as error:fn()
        self.assertEqual(error.exception.code,code)

    def release(self,token,**kwargs):
        return self.coordinator.recovery_force_release(
            self.op_id,"operator",token,"Original task outcome unavailable; accepted risk",
            "RELEASE "+self.op_id,True,**kwargs)

    def test_two_stage_release_preserves_original_state_and_audits_exact_resources(self):
        view=self.coordinator.recovery_review(self.op_id,"operator")
        self.assertEqual(view["state"],"dispatch_intent")
        self.assertEqual(view["resources"],[self.box,self.name])
        self.check_error("RECOVERY_CONFIRMATION_REQUIRED",lambda:self.coordinator.recovery_force_release(
            self.op_id,"operator",view["challenge"],"A long valid reason here",
            "RELEASE "+self.op_id,False))
        self.check_error("RECOVERY_REVIEW_EXPIRED_OR_CHANGED",lambda:self.release("a"*43))
        result=self.release(view["challenge"])
        self.assertEqual(result["state"],"released_with_unknown_outcome")
        self.assertEqual(result["released_resources"],2)
        self.assertFalse(result["remote_cancellation"])
        self.assertEqual(self.coordinator.status(self.op_id)["state"],"dispatch_intent")
        self.assertEqual(self.coordinator.status(self.op_id)["resources"],[])
        events=self.coordinator.recovery_audit(self.op_id)
        self.assertEqual(len(events),1)
        self.assertEqual(events[0]["decision"],"forced_unknown_release")
        self.assertEqual(events[0]["resources"],[self.box,self.name])
        self.assertEqual(events[0]["reviewer"],"operator")
        with sqlite3.connect(self.db) as db:
            for statement in (
                    "DELETE FROM coordination_recovery_audit",
                    "UPDATE coordination_recovery_audit SET reason='tampered'"):
                with self.assertRaises(sqlite3.IntegrityError):db.execute(statement)
        self.coordinator.acquire("b"*32,A,"gateway_b",[self.box,self.name])
        self.check_error("RECOVERY_CLAIM_NOT_UNKNOWN",
                         lambda:self.coordinator.recovery_review(self.op_id,"operator"))

    def test_expired_or_replaced_review_fails_closed(self):
        one=self.coordinator.recovery_review(self.op_id,"operator")
        two=self.coordinator.recovery_review(self.op_id,"operator")
        self.check_error("RECOVERY_REVIEW_EXPIRED_OR_CHANGED",lambda:self.release(one["challenge"]))
        self.time[0]+=301
        self.check_error("RECOVERY_REVIEW_EXPIRED_OR_CHANGED",lambda:self.release(two["challenge"]))
        self.assertEqual(len(self.coordinator.status(self.op_id)["resources"]),2)
        self.assertFalse(self.coordinator.recovery_audit(self.op_id))

    def test_snapshot_change_and_wrong_operator_refused(self):
        one=self.coordinator.recovery_review(self.op_id,"operator")
        self.check_error("RECOVERY_REVIEW_EXPIRED_OR_CHANGED",lambda:
            self.coordinator.recovery_force_release(self.op_id,"other",one["challenge"],
                "Enough explanation for risk acceptance","RELEASE "+self.op_id,True))
        self.coordinator.unknown(self.op_id)
        self.check_error("RECOVERY_REVIEW_EXPIRED_OR_CHANGED",lambda:self.release(one["challenge"]))
        self.assertFalse(self.coordinator.recovery_audit(self.op_id))
        self.assertEqual(len(self.coordinator.status(self.op_id)["resources"]),2)

    def test_invalid_input_and_pre_dispatch_refused(self):
        one=self.coordinator.recovery_review(self.op_id,"operator")
        for text in ("short","reason with\nnewline", "x"*241):
            self.check_error("RECOVERY_ARGUMENT_INVALID",lambda t=text:self.coordinator.recovery_force_release(
                self.op_id,"operator",one["challenge"],t,"RELEASE "+self.op_id,True))
        self.check_error("RECOVERY_CONFIRMATION_REQUIRED",lambda:self.coordinator.recovery_force_release(
            self.op_id,"operator",one["challenge"],"A long valid reason here","RELEASE wrong",True))
        self.coordinator.acquire("c"*32,B,"gateway_c",[box_resource(B)])
        self.check_error("RECOVERY_CLAIM_NOT_UNKNOWN",lambda:
            self.coordinator.recovery_review("c"*32,"operator"))

    def test_review_and_release_are_durable_across_process_restart(self):
        self.coordinator.unknown(self.op_id)
        review=self.coordinator.recovery_review(self.op_id,"operator")
        restarted=OperationCoordinator(lambda:sqlite3.connect(self.db,timeout=5),
                                       clock=lambda:self.time[0])
        restarted.initialize()
        release=restarted.recovery_force_release(self.op_id,"operator",review["challenge"],
            "Uncertain remote execution, force release approved", "RELEASE "+self.op_id,True)
        self.assertEqual(release["released_resources"],2)
        self.assertEqual(len(self.coordinator.recovery_audit(self.op_id)),1)
        self.check_error("RECOVERY_CLAIM_NOT_UNKNOWN",lambda:self.release(review["challenge"]))

    def test_failed_audit_transaction_never_releases_claims(self):
        review=self.coordinator.recovery_review(self.op_id,"operator")
        with sqlite3.connect(self.db) as db:
            db.execute("CREATE TRIGGER fail_audit BEFORE INSERT ON coordination_recovery_audit "
                       "BEGIN SELECT RAISE(ABORT, 'simulated disk refusal'); END")
        self.check_error("COORDINATION_UNAVAILABLE",lambda:self.release(review["challenge"]))
        self.assertEqual(len(self.coordinator.status(self.op_id)["resources"]),2)
        self.assertEqual(self.coordinator.recovery_audit(self.op_id),[])

    def test_additive_schema_upgrade_keeps_old_unknown_claim(self):
        from bwmini.operation_coordination import _SCHEMA
        old=os.path.join(self.tmp.name,"old.sqlite3")
        with sqlite3.connect(old) as db:
            for statement in _SCHEMA.split(";"):
                if statement.strip() and "coordination_recovery_" not in statement:
                    db.execute(statement)
            db.execute("INSERT INTO coordination_version VALUES (1,1)")
            db.execute("INSERT INTO coordination_operations VALUES (?,?,?,?,?,?)",
                       (self.op_id,A,"gateway_"+self.op_id,"dispatch_intent",1,2))
            db.execute("INSERT INTO coordination_resources VALUES (?,?)",(self.box,self.op_id))
        upgraded=OperationCoordinator(lambda:sqlite3.connect(old,timeout=5))
        upgraded.initialize()
        self.assertEqual(upgraded.status(self.op_id)["resources"],[self.box])
        view=upgraded.recovery_review(self.op_id,"operator")
        self.assertEqual(view["state"],"dispatch_intent")
        upgraded.initialize()
        self.assertEqual(upgraded.status(self.op_id)["resources"],[self.box])

    def test_competing_release_has_one_winner_one_audit(self):
        review=self.coordinator.recovery_review(self.op_id,"operator")
        start=threading.Barrier(3)
        answers=[]
        def release():
            other=OperationCoordinator(lambda:sqlite3.connect(self.db,timeout=5),
                                       clock=lambda:self.time[0])
            start.wait()
            try:
                other.recovery_force_release(self.op_id,"operator",review["challenge"],
                    "Approved original unknown cannot be proven", "RELEASE "+self.op_id,True)
                answers.append("released")
            except CoordinationFault as error:answers.append(error.code)
        workers=[threading.Thread(target=release) for _ in range(2)]
        for w in workers:w.start()
        start.wait()
        for w in workers:w.join(timeout=5)
        self.assertCountEqual(answers,["released","RECOVERY_CLAIM_NOT_UNKNOWN"])
        self.assertEqual(len(self.coordinator.recovery_audit(self.op_id)),1)

if __name__ == "__main__":
    unittest.main()
