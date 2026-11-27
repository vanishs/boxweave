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

    def test_capture_requires_exact_schema7_and_settled_originals(self):
        # Read-only capture with synthetic observer inputs; no host calls.
        from contextlib import ExitStack
        from types import SimpleNamespace

        digest = "a" * 64
        local = {key: {"state": "ABSENT"} for key in (
            "backup", "scriptlet", "legacy_root_operations", "scoped_fence",
            "purge_fence", "audit")}
        local["state"] = {"state": "OBSERVED", "files": [
            {"name": name, "identity": {}, "safe": True}
            for name in ("ledger.sqlite", "install-manifest.json", u.SERVE_LOCK_NAME)]}
        local["scoped_journal"] = {"tree": {"sha256": digest}}
        report = {
            "coverage": "BOUNDED_SOURCES_OBSERVED",
            "ledger_observation_sha256": digest,
            "installation": {"adapter": "EXACT_RUNNING_OBSERVER_BINARY",
                "mode": "SCOPED_V2", "identity": {"static_contract": "VERIFIED",
                    "service_unit": "ACTIVE", "runtime": "VERIFIED",
                    "guard_unit": "ACTIVE", "lifecycle_unit": "ACTIVE"},
                "installed_sha256": digest, "manifest_sha256": digest,
                "unit_sha256": digest, "preimage": {"sha256": digest}},
            "native": {"base_acl_config": "RECORDED_PREIMAGE_MATCH",
                "bridge_config": "RECORDED_PREIMAGE_MATCH",
                "evidence": {"sha256": digest}}}
        patches = {
            "machine_sha": digest, "maintenance_local_inventory": local,
            "maintenance_observe": report, "maintenance_preview": {"blockers": []},
            "scoped_drain_scope": self.scope, "scoped_drain_global_inventory": None,
            "inspect_exec_archive": {}, "scoped_effective_reviews": {},
            "scoped_units": {}, "scoped_incus_active": 123, "command": ("", 0)}
        for active in (True, False):
            report["installation"]["identity"]["service_unit"] = "ACTIVE" if active else "INACTIVE"
            report["installation"]["identity"]["runtime"] = "VERIFIED" if active else "UNVERIFIED"
            for version, unresolved, expected in ((7, 0, []), (6, 0, ["DRAIN_UNSETTLED_ORIGINALS"]),
                    (8, 0, ["DRAIN_UNSETTLED_ORIGINALS"]), (7, 1, ["DRAIN_UNSETTLED_ORIGINALS"])):
                with self.subTest(active=active, version=version, unresolved=unresolved), ExitStack() as stack:
                    stack.enter_context(mock.patch.object(u.pwd, "getpwnam",
                        return_value=SimpleNamespace(pw_uid=1001)))
                    for name, result in patches.items():
                        stack.enter_context(mock.patch.object(u, name, return_value=result))
                    reader = stack.enter_context(mock.patch.object(u, "inspect_sqlite",
                        return_value={"version": version, "unresolved_operations": unresolved,
                                      "operations": 5, "blockers": []}))
                    capture = u.scoped_drain_capture("user-1001", active, offline_entry=not active)
                    self.assertEqual(capture["blockers"], expected)
                    self.assertTrue(reader.call_args.kwargs["scoped"])
                    self.assertEqual(reader.call_args.kwargs["cold"], not active)

    def test_offline_plan_binds_stopped_entry_and_original_scope(self):
        capture = {"scope": self.scope, "immutable": {"host": "h"},
                   "witness": {"ledger": "a"}, "blockers": []}
        with mock.patch.object(u.os.path, "lexists", return_value=False), \
                mock.patch.object(u, "machine_sha", return_value="a" * 64), \
                mock.patch.object(u, "scoped_inactive") as inactive, \
                mock.patch.object(u, "scoped_incus_active"), \
                mock.patch.object(u, "scoped_drain_capture", side_effect=lambda *args, **kwargs: copy.deepcopy(capture)) as read:
            offline = u.scoped_drain_plan("user-1001", offline=True)
            self.assertTrue(offline["ready"])
            self.assertIs(offline["offline_entry"], True)
            self.assertEqual(read.call_args.args[1], False)
            self.assertEqual(inactive.call_count, 2)
            normal = u.scoped_drain_plan("user-1001")
            self.assertTrue(normal["ready"])
            self.assertNotIn("offline_entry", normal)
            self.assertNotEqual(offline["review_sha256"], normal["review_sha256"])
            self.assertEqual(offline["steps"], normal["steps"])

    def test_offline_entry_refuses_active_or_unverified_controller(self):
        with mock.patch.object(u.os.path, "lexists", return_value=False), \
                mock.patch.object(u, "scoped_inactive", side_effect=u.UninstallError("not inactive")), \
                mock.patch.object(u, "scoped_drain_capture") as read:
            with self.assertRaisesRegex(u.UninstallError, "not inactive"):
                u.scoped_drain_plan("user-1001", offline=True)
            read.assert_not_called()

    def test_offline_resume_does_not_adopt_normal_receipt_or_stop_restarted_controller(self):
        record = {"project": "user-1001", "phase": "APPROVED", "approved_plan": {}}
        with mock.patch.object(u, "scoped_drain_read", return_value=record), \
                mock.patch.object(u, "scoped_drain_verify") as verify:
            with self.assertRaisesRegex(u.UninstallError, "OFFLINE_ORIGINAL_REQUIRED"):
                u.scoped_drain_resume_preview("user-1001", offline=True)
            verify.assert_not_called()
        record["approved_plan"]["offline_entry"] = True
        with mock.patch.object(u, "scoped_drain_read", return_value=record), \
                mock.patch.object(u, "scoped_inactive", side_effect=u.UninstallError("not inactive")), \
                mock.patch.object(u, "command") as command:
            with self.assertRaisesRegex(u.UninstallError, "not inactive"):
                u.scoped_drain_continue(record)
            command.assert_not_called()

    def test_offline_blockers_remain_no_approval(self):
        capture = {"scope": self.scope, "immutable": {"host": "h"}, "witness": {"ledger": "a"},
                   "blockers": ["DRAIN_UNSETTLED_ORIGINALS", "FOREIGN_LEDGER_SCOPE:ledger"]}
        with mock.patch.object(u.os.path, "lexists", return_value=False), \
                mock.patch.object(u, "machine_sha", return_value="a"*64), \
                mock.patch.object(u, "scoped_inactive"), mock.patch.object(u, "scoped_incus_active"), \
                mock.patch.object(u, "scoped_drain_capture", return_value=capture), \
                mock.patch("builtins.open") as terminal, mock.patch.object(u, "scoped_drain_begin") as begin:
            plan = u.scoped_drain_plan("user-1001", offline=True)
            self.assertFalse(plan["ready"])
            self.assertEqual(plan["blockers"], capture["blockers"])
            with self.assertRaisesRegex(u.UninstallError, "retirement blocked"):
                u.scoped_drain_approve("user-1001", plan, offline=True)
            terminal.assert_not_called()
            begin.assert_not_called()

    def test_offline_approval_rejects_normal_confirmation_and_stale_plan(self):
        import contextlib
        import io
        plan = {"ready": True, "offline_entry": True}
        for answer, error in (("REMOVE SCOPED user-1001", "OPERATOR_APPROVAL_REQUIRED"),
                              ("REMOVE SCOPED OFFLINE user-1001", "changed after approval")):
            with self.subTest(answer=answer), \
                    mock.patch("builtins.open", side_effect=[io.StringIO(answer+"\n"), io.StringIO()]), \
                    mock.patch.object(u, "scoped_lock", side_effect=lambda _: contextlib.nullcontext()), \
                    mock.patch.object(u, "scoped_drain_plan", return_value={**plan, "ready": False}), \
                    mock.patch.object(u, "scoped_drain_begin") as begin:
                with self.assertRaisesRegex(u.UninstallError, error):
                    u.scoped_drain_approve("user-1001", plan, offline=True)
                begin.assert_not_called()

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
