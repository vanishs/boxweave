#!/usr/bin/env python3
"""Deliberate false-PASS and missing-native-negative controls for S2 case reviewer."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

HERE=Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location("s2_scope",HERE/"s2_p3_scoped_acceptance.py")
target=importlib.util.module_from_spec(spec)
spec.loader.exec_module(target)
POLICY=json.loads((HERE/"s2_p3_scoped_acceptance.json").read_text())
SHA="a"*40
HASH="b"*64
markers={"S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS"}
for case in POLICY:markers.update(case["required_markers"])
def witness(extras=(), omit=()):
    lines=["S2_NATIVE_CANDIDATE_SOURCE_SHA="+SHA,
           HASH+"  /tmp/bw-v1-s2-ghci.known/boxweave"]
    lines += [x+"=PASS" for x in sorted(markers) if x not in omit]
    lines += list(extras)
    return ("\n".join(lines)+"\n").encode()

class ExitReviewerNegatives(unittest.TestCase):
    def test_valid_scoped_report_is_not_a_hardware_release(self):
        result=target.evaluate(witness(),SHA)
        self.assertEqual(result["groups"],19)
        self.assertEqual(result["supported_s2_scoped_cases_with_native_evidence"],18)
        self.assertEqual(result["s4_deferred_groups"],1)
        self.assertEqual(result["public_guest_wan_https"],"UNVERIFIED_ENVIRONMENT_EXCEPTION")
        self.assertEqual(result["physical_power_cycle_actual"],"NOT_TESTED_NO_HARDWARE_CLAIM")
        self.assertEqual(result["candidate_release_status"],"DRAFT_UNMERGED_NOT_RELEASED")
    def test_each_of_19_rows_cannot_be_forged_with_missing_first_native_witness(self):
        for case in POLICY:
            with self.subTest(id=case["id"]),self.assertRaisesRegex(ValueError,"missing"):
                target.evaluate(witness(omit={case["required_markers"][0]}),SHA)
    def test_wrong_or_duplicate_source_refused(self):
        with self.assertRaises(ValueError):target.evaluate(witness(),"c"*40)
        with self.assertRaises(ValueError):target.evaluate(witness(["S2_NATIVE_CANDIDATE_SOURCE_SHA="+SHA]),SHA)
    def test_missing_or_duplicate_candidate_binary_refused(self):
        lines=witness().decode().splitlines()
        with self.assertRaises(ValueError):
            target.evaluate(("\n".join(lines[1:])+"\n").encode(),SHA)
        with self.assertRaises(ValueError):
            target.evaluate(witness([HASH+"  /tmp/bw-v1-s2-ghci.other/boxweave"]),SHA)
    def test_false_wan_gate_blocked(self):
        with self.assertRaisesRegex(ValueError,"WAN falsely claimed"):
            target.evaluate(witness(["S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS"]),SHA)
    def test_actual_wan_requires_both_toggles_and_positive(self):
        result=target.evaluate(witness(["S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS",
           "S2_NATIVE_EXTERNAL_IPV4_WAN_HTTPS_POSITIVE_CONTROL=PASS"]),SHA)
        self.assertEqual(result["public_guest_wan_https"],"PASS")
    def test_unsupported_hardware_marker_rejected(self):
        with self.assertRaisesRegex(ValueError,"hardware"):
            target.evaluate(witness(["S2_NATIVE_PHYSICAL_POWER_OUTAGE=PASS"]),SHA)
    def test_surrogate_cannot_be_removed(self):
        for key in ["S2_SQLITE_WAL_FSYNC_BEFORE_REAL_KILL_UNKNOWN_NO_REPLAY",
                    "S2_SQLITE_WAL_FSYNC_AFTER_REAL_KILL_UNKNOWN_NO_REPLAY"]:
            with self.subTest(key=key),self.assertRaises(ValueError):
                target.evaluate(witness(omit={key}),SHA)
    def test_policy_missing_or_duplicate_row_refused(self):
        with self.assertRaisesRegex(ValueError,"exact 10 TC"):
            target.evaluate(witness(),SHA,policy=POLICY[:-1])
        with self.assertRaisesRegex(ValueError,"exact 10 TC"):
            target.evaluate(witness(),SHA,policy=POLICY[:-1]+[POLICY[0]])
    def test_s4_only_deferred(self):
        p=json.loads(json.dumps(POLICY))
        p[0]["verdict"]="DEFERRED_S4"
        with self.assertRaisesRegex(ValueError,"only S4"):
            target.evaluate(witness(),SHA,policy=p)

if __name__=="__main__":
    unittest.main()
