import copy
import unittest
import uninstall_core as u


class OfflineScopeTests(unittest.TestCase):
    def setUp(self):
        self.box = "a" * 32
        self.original = {"id": "b"*32, "state": "UNKNOWN", "action": "box_stop",
                         "retirement_scope": "EXACT_OWNED_POWER_ORIGINAL",
                         "claim_protection": "RETAINED", "recorded_claims": 1,
                         "native_receipt": True, "box_uuid": self.box, "incarnation": self.box}
        self.report = {"installation": {"mode": "LEGACY", "adapter": "PINNED_RC13",
                       "identity": {"installed_identity": "VERIFIED", "service_unit": "INACTIVE"}},
                       "native": {"legacy_lease_proof": "VERIFIED", "base_acl_config": "RECORDED_PREIMAGE_MATCH",
                       "owned_matched": [self.box], "owned_power": {self.box: "Stopped"}},
                       "ledger": {"boxes": [{"uuid": self.box, "incarnation": self.box,
                       "create_proof": True, "binding_state": "PINNED"}],
                       "nonterminal_or_unknown_originals": [self.original]}}
        self.preview = {"blockers": [{"code": "ORIGINAL_UNKNOWN", "resource": self.original["id"], "severity": "HARD"}]}

    def test_exact_unknown_retained(self):
        before = copy.deepcopy(self.report)
        blockers, eligible = u.offline_scope_blockers(self.report, self.preview)
        self.assertEqual(blockers, [])
        self.assertEqual(eligible, [self.original])
        self.assertEqual(self.report, before)

    def test_unknown_ownership_mismatch(self):
        for field, value in (("recorded_claims", 2), ("action", "box_create"),
                             ("native_receipt", False), ("incarnation", "foreign"),
                             ("retirement_scope", "UNPROVEN"), ("state", "RUNNING")):
            with self.subTest(field=field):
                report = copy.deepcopy(self.report)
                report["ledger"]["nonterminal_or_unknown_originals"][0][field] = value
                blockers, eligible = u.offline_scope_blockers(report, self.preview)
                self.assertTrue(blockers)
                self.assertFalse(eligible)

    def test_native_and_service_refusals(self):
        for section, field, value in (("native", "owned_power", {self.box: "Running"}),
                    ("native", "legacy_lease_proof", "UNVERIFIED"),
                    ("native", "base_acl_config", "DRIFT"),
                    ("installation", "mode", "SCOPED_V2"),
                    ("ledger", "peers", [{"state": "UNKNOWN"}]),
                    ("ledger", "forwards", [{"state": "ACTIVE"}]),
                    ("ledger", "legacy_compactions", ["x"])):
            with self.subTest(field=field):
                report = copy.deepcopy(self.report)
                report[section][field] = value
                self.assertTrue(u.offline_scope_blockers(report, self.preview)[0])

    def test_unrelated_hard_blocker_preserved(self):
        self.preview["blockers"].append({"code":"FOREIGN_CONSUMER","resource":"foreign","severity":"HARD"})
        self.assertIn("FOREIGN_CONSUMER:foreign", u.offline_scope_blockers(self.report, self.preview)[0])


if __name__ == "__main__":
    unittest.main()
