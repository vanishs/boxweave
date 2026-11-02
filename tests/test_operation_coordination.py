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


if __name__ == "__main__":
    unittest.main()
