"""Black-box fixture proofs for accepted REST create reconciliation."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from bwmini.async_create_observer import observe_one
from bwmini.remote_create import CreateJournal
from bwmini.store import Store
from bwmini.domain import Fault, Target


class Provider:
    rest = object()
    def __init__(self, logical):
        self.logical = logical
        self.target = Target("containarium", "dev", "default", logical,
                             logical + "-container", "1000", logical,
                             logical + "-container")
        self.state = "CONTAINER_STATE_RUNNING"
        self.identity_ready = True
        self.probes = 0
    def lifecycle_inspect(self, logical):
        self.probes += 1
        if self.state == "absent":
            return None
        return {"target": self.target, "state": self.state}
    def lifecycle_identity(self, target, timeout):
        if not self.identity_ready:
            raise Fault("TRUST_FAILED")
        return {"exit_code": 0, "user": target.user,
                "hostname": target.hostname}


class AsyncObserveTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.store = Store(Path(self.tmp.name) / "state" / "db.sqlite3")
        self.clock = lambda: 1040.
        self.provider = Provider("test-async")
        self.journal = CreateJournal(self.store, clock=lambda: 1000.)

    def prepare(self, nesting=None):
        receipt, fresh = self.journal.reserve(
            "test-async", "a" * 32, "b" * 64)
        self.assertTrue(fresh)
        self.journal.intent_options(receipt["task_id"], nesting=nesting)
        self.journal.begin_dispatch(receipt["task_id"], "b" * 64)
        self.journal.accept(receipt["task_id"])
        return receipt["task_id"]

    def test_ready_never_dispatched_twice_and_crash_only_observes(self):
        ident = self.prepare()
        self.journal.recover_interrupted()
        self.assertEqual(self.journal.get(task_id=ident)["state"], "accepted")
        with patch("bwmini.async_create_observer.coordination_bridge.SharedClaims"):
            self.assertTrue(observe_one(self.store, self.provider, self.clock))
        receipt = self.journal.get(task_id=ident)
        self.assertEqual(receipt["state"], "succeeded")
        self.assertTrue(receipt["guest_user_and_ssh_verified"])
        self.assertEqual(receipt["code"], "REST_GUEST_READY")
        self.assertFalse(self.journal.begin_dispatch(ident, "b" * 64))
        self.assertFalse(observe_one(self.store, self.provider, self.clock))

    def test_running_without_guest_is_not_ready(self):
        ident = self.prepare()
        self.provider.identity_ready = False
        self.assertTrue(observe_one(self.store, self.provider, self.clock))
        self.assertEqual(self.journal.get(task_id=ident)["state"], "accepted")
        self.assertFalse(observe_one(self.store, self.provider, self.clock))
        self.assertEqual(self.provider.probes, 1)

    def test_wrong_instance_identity_freezes_without_replay(self):
        ident = self.prepare()
        self.provider.target = Target("containarium", "dev", "default",
                                      "test-async", "unexpected-instance",
                                      "1000", "test-async", "unexpected-instance")
        with patch("bwmini.async_create_observer.coordination_bridge.SharedClaims"):
            self.assertTrue(observe_one(self.store, self.provider, self.clock))
        self.assertEqual(self.journal.get(task_id=ident)["state"], "unknown")

    def test_requested_nesting_exact_native_incarnation_then_ready(self):
        ident = self.prepare(nesting=True)
        calls = []
        def native(request):
            calls.append(request["action"])
            target = {"logical": "test-async", "name": "test-async-container",
                      "created": "1000", "uuid": "native-uuid",
                      "nesting": request["action"] == "apply"}
            if request["action"] == "plan":
                return {"target": target, "expected": {"id": "expected"}}
            return {"state": "configured", "target": target}
        with patch("bwmini.async_create_observer.nesting.invoke", side_effect=native), \
             patch("bwmini.async_create_observer.coordination_bridge.SharedClaims"):
            self.assertTrue(observe_one(self.store, self.provider, self.clock))
        self.assertEqual(calls, ["plan", "apply"])
        self.assertEqual(self.journal.get(task_id=ident)["state"], "succeeded")

    def test_crash_after_nesting_intent_does_not_replay_native_write(self):
        ident = self.prepare(nesting=True)
        self.assertTrue(self.journal.begin_nesting(ident))
        with patch("bwmini.async_create_observer.nesting.invoke") as native, \
             patch("bwmini.async_create_observer.coordination_bridge.SharedClaims"):
            observe_one(self.store, self.provider, self.clock)
        native.assert_not_called()
        self.assertEqual(self.journal.get(task_id=ident)["state"], "unknown")
