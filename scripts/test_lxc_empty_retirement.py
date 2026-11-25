"""No new authority from a known inherited policy hint."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import uninstall_core as u

class LXCEmptyRetirementTests(unittest.TestCase):
    def test_unknown_or_additional_override_refused(self):
        for path in ("/tmp/other.conf", str(u.LXC_SERVICE_DROPIN)+" /tmp/extra.conf"):
            with patch.object(u,"scoped_properties",return_value={"FragmentPath":str(u.UNIT),"DropInPaths":path}),patch.object(u,"lxc_retirement_policy") as check:
                with self.assertRaisesRegex(u.UninstallError,"DROPIN_UNREVIEWED"):
                    u.scoped_effective_review(u.UNIT,True)
                check.assert_not_called()

    def test_known_policy_still_refused_for_live_drain(self):
        with patch.object(u,"scoped_properties",return_value={"FragmentPath":str(u.UNIT),"DropInPaths":str(u.LXC_SERVICE_DROPIN)}),patch.object(u,"lxc_retirement_policy") as check:
            with self.assertRaisesRegex(u.UninstallError,"DROPIN_UNREVIEWED"):
                u.scoped_effective_review(u.UNIT)
            check.assert_not_called()

    def test_exact_unit_body_drift_refused_before_effect(self):
        with patch.object(u,"lxc_retirement_policy",return_value={"sha256":"known"}),patch.object(u,"file_fingerprint",return_value={"sha256":"changed"}),patch.object(u,"scoped_effective_reviews") as read:
            plan={"unit":{"sha256":"old"},"scoped_effective_reviews":{u.UNIT.name:{"sha256":"known"}}}
            with self.assertRaisesRegex(u.UninstallError,"UNIT_BODY_DRIFT"):
                u.scoped_verify_effective_reviews(plan)
            read.assert_not_called()

    def test_review_hint_never_grants_apply_authority(self):
        for mode, coverage, want in (("SCOPED_V2","BOUNDED_SOURCES_OBSERVED","LXC_SCOPED_RETIREMENT_REVIEW_REQUIRED"),
                                     ("SCOPED_V2","UNVERIFIED","EFFECTIVE_DROPIN_UNREVIEWED"),
                                     ("LEGACY","BOUNDED_SOURCES_OBSERVED","EFFECTIVE_DROPIN_UNREVIEWED")):
            r={"coverage":coverage,"database":{"state":"OBSERVED"},"ledger_observation_sha256":"a"*64,
               "installation":{"mode":mode,"units":[{"name":u.UNIT.name,"drop_ins":"KNOWN_LXC255_SCOPED_RETIREMENT_REVIEW_REQUIRED","effective_file":{"state":"OBSERVED"}}]},"ledger":{"boxes":[]}}
            out=u.maintenance_preview("user-1001","retain","a"*64,r,{})
            self.assertTrue(any(x["code"]==want for x in out["blockers"]))
            self.assertFalse(out.get("apply_available",False))

if __name__=="__main__":
    unittest.main()
