import copy
import unittest
from unittest import mock

import uninstall_core as u


class ScopedDrainPlanTests(unittest.TestCase):
    def setUp(self):
        self.a = "11111111-1111-4111-8111-111111111111"
        self.b = "22222222-2222-4222-8222-222222222222"
        self.scope = {"boxes": [{"uuid": x, "incarnation": x, "name": n,
                                 "create_proof": True, "binding_state": "PINNED"}
                                for x, n in ((self.a, "a"), (self.b, "b"))],
                      "retired": [],
                      "peers": [{"source": self.a, "source_incarnation": self.a,
                                 "target": self.b, "target_incarnation": self.b,
                                 "state": "CONFIRMED", "last_verified_original": "a" * 32}],
                      "forwards": [{"box": self.a, "incarnation": self.a,
                                    "state": "CONFIRMED", "id": "b" * 32}],
                      "power": {self.a: "Running", self.b: "Stopped"}}

    def test_root_inventory_refuses_foreign_before_dispatch(self):
        import json
        for b in self.scope["boxes"]:
            b["acl"] = "bw-peer-" + b["name"]
        acls = [{"name": "bw-peer-quarantine"}] + [{"name": b["acl"]} for b in self.scope["boxes"]]
        instances = [{"project": "user-1001", "name": b["name"], "config": {"volatile.uuid": b["uuid"]}}
                     for b in self.scope["boxes"]]
        def response(a, i):
            return [(json.dumps(a), 0), (json.dumps(i), 0)]
        with mock.patch.object(u, "command", side_effect=response(acls, instances)):
            u.scoped_drain_global_inventory("user-1001", self.scope)
        with mock.patch.object(u, "command", side_effect=response(acls + [{"name": "foreign"}], instances)) as call:
            with self.assertRaisesRegex(u.UninstallError, "ACL_INVENTORY_NOT_EXCLUSIVE"):
                u.scoped_drain_global_inventory("user-1001", self.scope)
            self.assertEqual(call.call_count, 1)
        with mock.patch.object(u, "command", side_effect=response(acls, instances + [{"project": "foreign"}])):
            with self.assertRaisesRegex(u.UninstallError, "INSTANCE_INVENTORY_NOT_EXCLUSIVE"):
                u.scoped_drain_global_inventory("user-1001", self.scope)

    def test_exact_dependency_order(self):
        steps = u.scoped_drain_steps(self.scope)
        self.assertEqual([s["action"] for s in steps],
                         ["box_peer_set", "port_forward_delete", "box_stop",
                          "box_delete", "box_stop", "box_delete"])
        self.assertEqual(steps[0]["target"], self.b)
        self.assertEqual(steps[1]["forward"], "b" * 32)

    def test_uncertain_dependencies_and_mismatched_incarnation_refused(self):
        for field, key, val in (("peers", "state", "UNKNOWN"),
                                ("forwards", "incarnation", self.b),
                                ("boxes", "create_proof", False)):
            bad = copy.deepcopy(self.scope)
            bad[field][0][key] = val
            with self.assertRaises(u.UninstallError):
                u.scoped_drain_steps(bad)

    def test_prefix_preserves_unrelated_identity(self):
        rec = {"approved_plan": {"capture": {"scope": self.scope}},
               "steps": u.scoped_drain_steps(self.scope), "completed": [
                   {"id": "1"*32}, {"id": "2"*32}, {"id": "3"*32}]}
        after = u.scoped_drain_expected(rec, {"id": "4"*32})
        self.assertEqual(after["boxes"], [self.scope["boxes"][1]])
        self.assertEqual(after["retired"][0]["uuid"], self.a)
        self.assertEqual(after["retired"][0]["delete_original"], "4"*32)
        self.assertEqual(after["power"], {self.b: "Stopped"})
        self.assertEqual(after["peers"], [])
        self.assertEqual(after["forwards"], [])

    def test_original_unknown_cannot_be_used_as_success(self):
        rec = {"steps": [{"action": "box_stop", "box": self.a}],
               "index": 0, "actor": {"id": "c"*32}}
        original = {"found": True, "receipt": {"state": "UNKNOWN", "action": "box_stop",
                    "box_uuid": self.a, "incarnation": self.a,
                    "work_ref": "scoped-drain-"+"c"*32, "operation_id": "d"*32}}
        import json
        with mock.patch.object(u, "command", return_value=(json.dumps(original), 1)) as call:
            with self.assertRaisesRegex(u.UninstallError, "UNSETTLED"):
                u.scoped_drain_call(rec, True)
            self.assertEqual(call.call_count, 1)

    def test_exact_original_read_only(self):
        rec = {"steps": [{"action": "box_stop", "box": self.a}],
               "index": 0, "actor": {"id": "c"*32}}
        import json
        receipt = {"found": True, "receipt": {"state": "SUCCEEDED", "action": "box_stop",
                    "box_uuid": self.a, "incarnation": self.a,
                    "work_ref": "scoped-drain-"+"c"*32, "operation_id": "d"*32}}
        with mock.patch.object(u, "command", return_value=(json.dumps(receipt), 0)) as call:
            self.assertEqual(u.scoped_drain_call(rec, True), {"id": "d"*32})
            self.assertIn("--read-only", call.call_args.args[0])
            self.assertEqual(call.call_count, 1)

    def test_drift_blocks_next_effect(self):
        rec = {"project": "user-1001", "approved_plan": {"capture": {"immutable": {"host": "h"},
                "scope": self.scope}}, "steps": u.scoped_drain_steps(self.scope),
                "completed": [], "last_witness": {"ledger": "a"}}
        fresh = {"blockers": [], "immutable": {"host": "h"},
                 "scope": self.scope, "witness": {"ledger": "b"}}
        with mock.patch.object(u, "scoped_drain_capture", return_value=fresh):
            with self.assertRaisesRegex(u.UninstallError, "WITNESS_DRIFT"):
                u.scoped_drain_verify(rec)


if __name__ == "__main__":
    unittest.main()
