"""H1B pure planner tests. No Host paths or effect execution."""
import copy
import io
import json
import types
from unittest import mock
import unittest
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path

import uninstall_core as u


def fixture():
    uid = "11111111-1111-4111-8111-000000000001"
    box = {"uuid": uid, "incarnation": uid, "name": "test-box",
           "create_proof": True, "binding_state": "PINNED", "create_original": "a" * 32}
    report = {
        "version": 1, "scope": "user-1001", "observer_sha256": "a" * 64,
        "coverage": "BOUNDED_SOURCES_OBSERVED", "cross_source_atomic": False,
        "ledger_observation_sha256": "b" * 64,
        "installation": {"adapter": "PINNED_RC14", "mode": "SCOPED_V2",
                         "identity": {"runtime": "VERIFIED"},
                         "preimage": {"state": "OBSERVED"},
                         "root_policy": {"kernel_guard": "OBSERVED"},
                         "artifacts": [{"id": k, "evidence": {"state": "PRESENT"}} for k in
                                       ("scoped_root_journal", "scoped_guard_unit",
                                        "scoped_lifecycle_unit", "scoped_incus_dropin")],
                         "units": []},
        "database": {"state": "OBSERVED", "epoch_ns": 1},
        "native": {"evidence": {"state": "OBSERVED", "sha256": "c" * 64},
                   "ownership_comparison": "OBSERVED", "owned_matched": [uid],
                   "owned_unverified": [], "unowned_or_mismatched": 0,
                   "project_config": "AFTER", "profile_config": "AFTER",
                   "bridge_config": "RECORDED_PREIMAGE_MATCH",
                   "bridge_consumers": "OWNED_REFERENCES_ONLY",
                   "base_acl_consumers": "OWNED_REFERENCES_ONLY",
                   "base_acl_config": "RECORDED_PREIMAGE_MATCH",
                   "box_acl_references": "OBSERVED", "forward_references": "OBSERVED",
                   "epoch_ns": 2},
        "root_ledger_comparison": "MATCH",
        "ledger": {"boxes": [box], "recorded_claims": 9,
                   "nonterminal_or_unknown_originals": [], "peers": [], "forwards": [],
                   "legacy_pending_retirements": [], "legacy_compactions": []}}
    local = {k: {"state": "ABSENT"} for k in
             ("audit", "retired", "closed_journals", "scoped_fence", "purge_fence", "scriptlet")}
    local["state"] = {"state": "OBSERVED", "files": [], "unknown_entries": 0}
    return report, local


class MaintenancePreview(unittest.TestCase):
    def plan(self, report, local, retention="retain", host="d" * 64):
        result = u.maintenance_preview("user-1001", retention, host, report, local)
        self.assertFalse(result["ready"])
        self.assertFalse(result["apply_available"])
        self.assertFalse(result["effect_set_complete"])
        self.assertNotIn("review_sha256", result)
        return result

    def test_normal_and_stable_review(self):
        r, l = fixture()
        p = self.plan(r, l)
        self.assertEqual(p["status"], "REVIEW_ONLY")
        self.assertEqual(p["classification"], "NORMAL_REVIEW")
        self.assertEqual(p["blockers"], [])
        self.assertEqual(sum(x["kind"] == "STOP_DELETE_BOX" for x in p["proposed_effects"]), 1)
        r["native"]["epoch_ns"] = 3
        r["database"]["epoch_ns"] = 4
        self.assertEqual(self.plan(r, l)["preview_sha256"], p["preview_sha256"])
        for key, val in (("observer_sha256", "e" * 64), ("ledger_observation_sha256", "f" * 64)):
            q = copy.deepcopy(r)
            q[key] = val
            self.assertNotEqual(self.plan(q, l)["preview_sha256"], p["preview_sha256"])
        self.assertNotEqual(self.plan(r, l, host="e" * 64)["preview_sha256"], p["preview_sha256"])
        self.assertNotEqual(self.plan(r, l, retention="erase")["preview_sha256"], p["preview_sha256"])
        r["native"]["evidence"]["sha256"] = "d" * 64
        self.assertNotEqual(self.plan(r, l)["preview_sha256"], p["preview_sha256"])

    def test_offline_unknown_and_pending_create(self):
        r, l = fixture()
        r["installation"]["identity"]["runtime"] = "UNVERIFIED"
        r["ledger"]["nonterminal_or_unknown_originals"] = [
            {"id": "a" * 32, "action": "box_peer_set", "state": "UNKNOWN", "recorded_claims": 5}]
        p = self.plan(r, l)
        self.assertEqual(p["classification"], "OFFLINE_REVIEW")
        self.assertIn("ORIGINAL_UNKNOWN", {x["code"] for x in p["blockers"]})
        r["ledger"]["nonterminal_or_unknown_originals"][0]["action"] = "box_create"
        p = self.plan(r, l)
        self.assertEqual(p["classification"], "HARD_REFUSAL")

    def test_all_blockers_and_preservation(self):
        r, l = fixture()
        r["installation"]["preimage"]["state"] = "UNVERIFIED"
        r["installation"]["artifacts"].append({"id": "legacy_root_unit", "evidence": {"state": "PRESENT"}})
        r["installation"]["units"] = [{"name": "boxweave.service", "drop_ins": "PRESENT_UNREVIEWED",
                                       "effective_file": {"state": "UNVERIFIED"}}]
        r["ledger"]["foreign_scope_rows"] = 1
        r["ledger"]["orphan_fences"] = 1
        r["native"].update({"other_projects": 1, "unowned_or_mismatched": 1,
                            "base_acl_consumers": "FOREIGN_OR_UNVERIFIED",
                            "profile_config": "DRIFT", "forward_references": "UNVERIFIED"})
        l["mounted_artifacts"] = ["state"]
        l["state"]["unknown_entries"] = 1
        l["audit"]["state"] = "STARTED"
        l["retired"]["state"] = "PRESENT_UNVERIFIED"
        p = self.plan(r, l, retention="erase")
        codes = {x["code"] for x in p["blockers"]}
        self.assertTrue({"PREIMAGE_UNVERIFIED", "MIXED_INSTALLATION_ARTIFACTS",
                         "FOREIGN_LEDGER_SCOPE", "ORPHAN_NETWORK_EVIDENCE",
                         "NATIVE_IDENTITY_CONFLICT", "FOREIGN_OR_UNVERIFIED_CONSUMER",
                         "NATIVE_CONFIG_UNVERIFIED", "FORWARD_DEVICE_UNVERIFIED",
                         "EFFECTIVE_DROPIN_UNREVIEWED", "EFFECTIVE_UNIT_UNVERIFIED",
                         "MOUNTED_ARTIFACT", "STATE_INVENTORY_UNSAFE",
                         "INTERRUPTED_RETIREMENT", "HISTORICAL_ARCHIVE_PROVENANCE_REQUIRED",
                         "OTHER_PROJECTS_PRESERVED"} <= codes)
        self.assertEqual(p["status"], "BLOCKED")
        self.assertIn("unrelated Projects", p["preserve"])

    def test_missing_is_not_empty(self):
        p = self.plan({}, {"capture": {"state": "UNVERIFIED"}})
        self.assertEqual(p["status"], "BLOCKED")
        self.assertFalse(any(x["kind"] == "STOP_DELETE_BOX" for x in p["proposed_effects"]))
        self.assertIn("OBSERVER_UNAVAILABLE", {x["code"] for x in p["blockers"]})

    def test_parallel_local_change_discards_combined_observation(self):
        r, before = fixture()
        after = copy.deepcopy(before)
        after["state"]["unknown_entries"] = 1
        args = types.SimpleNamespace(project="user-1001", observer_binary=None,
                                     observer_sha256=None, retention="retain",
                                     maintenance_recheck=False)
        output = io.StringIO()
        with mock.patch.object(u.os, "geteuid", return_value=0), \
             mock.patch.object(u.pwd, "getpwnam", return_value=types.SimpleNamespace(pw_uid=1001)), \
             mock.patch.object(u, "machine_sha", return_value="a" * 64), \
             mock.patch.object(u, "maintenance_local_inventory", side_effect=[before, after]), \
             mock.patch.object(u, "maintenance_observe", return_value=r), \
             mock.patch.object(u, "apply") as apply, redirect_stdout(output):
            self.assertEqual(u.maintenance_main(args), 2)
            apply.assert_not_called()
        result = json.loads(output.getvalue())
        self.assertEqual(result["status"], "BLOCKED")
        self.assertEqual(result["observation"], {})
        self.assertEqual(result["local_inventory"]["capture"]["reason"], "LOCAL_EVIDENCE_CHANGED")

    def test_mount_boundary_and_cli_separation(self):
        self.assertTrue(u.maintenance_under_mount(Path("/var/lib/boxweave"), ["/var/lib/boxweave/ledger.sqlite"]))
        self.assertFalse(u.maintenance_under_mount(Path("/var/lib/boxweave"), ["/var/lib", "/var/lib/boxweave-retired"]))
        for argv in (["--apply", "--retention", "erase"],
                     ["--full", "--confirm-preview-sha256", "a" * 64],
                     ["--maintenance-plan", "--retention", "retain", "--confirm-review-sha256", "a" * 64],
                     ["--maintenance-plan", "--retention", "retain", "--observer-binary", "/tmp/observer"],
                     ["--maintenance-recheck", "--retention", "retain"]):
            with redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as cm:
                u.main(argv)
            self.assertEqual(cm.exception.code, 2)


if __name__ == "__main__":
    unittest.main()
