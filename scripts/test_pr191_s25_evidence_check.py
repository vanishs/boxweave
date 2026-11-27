#!/usr/bin/env python3
"""Synthetic transcript calibration only; never native acceptance."""
from pathlib import Path
import subprocess
import tempfile
import unittest

CHECK = Path(__file__).with_name("pr191_s25_evidence_check.sh")
SOURCE = "a" * 40
# Snapshot of the original PR/release required witness lines.
MARKERS = (
    'S25_HTTP_BAD_HOST_ORIGIN_PATH_AND_TOKEN_DENIAL=PASS',
    'S25_ORIGINAL_KEY_DIFFERENT_INTENT_DENIED=PASS',
    'S25_MCP_QUIET_RUNNING_BOUNDED_WAIT=PASS',
    'S25_MCP_QUIET_FINISH_UNCANCELLED=PASS',
    'S25_V_QUIET_ORIGINAL_SQLITE_RECONCILED=PASS',
    'S25_L1_MCP_L2_UPDATE_STOP_START_SAME_UUID=PASS',
    'S25_V_L1_MCP_L2_UPDATE_STOP_START_SAME_NATIVE_UUID=PASS',
    'S25_V_MCP_L2_NATIVE_IPV4_IPV6_GATEWAY_EGRESS=PASS',
    'S25_MCP_GUEST_GIT_TEST_ORIGINAL_FAILURE=PASS',
    'S25_MCP_GUEST_GIT_FIX_AND_RETEST=PASS',
    'S25_V_MCP_ORIGINAL_GIT_FAIL_AND_REPAIR_OPS=PASS',
    'S25_SIMWAN6_SEPARATELY_ROUTED_TLS_HOST_CONTROL=PASS',
    'S25_SIMWAN6_OUTER_MCP_IPV6_TLS_PER_NIC_ON_OFF_ON=PASS',
    'S25_V_SIMWAN6_OUTER_SDK_ORIGINALS_NATIVE_NIC_ROUTED_TLS_PEER_DENIAL=PASS',
    'S25_V_SIMWAN6_OUTER_ORIGINAL_MCP_PEER_DELETED=PASS',
    'S25_SIMWAN6_INNER_MCP_IPV6_TLS_PER_NIC_ON_OFF_ON=PASS',
    'S25_V_SIMWAN6_INNER_SDK_ORIGINALS_NATIVE_NIC_ROUTED_TLS_PEER_DENIAL=PASS',
    'S25_V_SIMWAN6_INNER_ORIGINAL_MCP_PEER_DELETED=PASS',
    'S25_SIMWAN6_BOTH_LAYER_RAW_PACKET_AND_NATIVE_EVIDENCE_ARCHIVED=PASS',
    'S25_R2_REAL_SERVICE_SIGKILL_DISTINCT_PID_HTTP_AUTH_RECOVERY=PASS',
    'S25_R2_SIGKILL_ORIGINAL_TWO_AGENT_SQLITE_AND_NATIVE_NO_REPLAY=PASS',
    'S25_R12_TWO_AGENT_RESOURCE_CONFLICT_DENIED=PASS',
    'S25_R12_REAL_MCP_REPLY_LOST_AFTER_DISPATCH_NO_RETRY=PASS',
    'S25_R12_INDEPENDENT_AGENT_RECOVERED_MCP_LOST_REPLY_ORIGINAL=PASS',
    'S25_R12_REVOKED_ORIGINAL_AGENT_HTTP_401=PASS',
    'S25_R12_POST_REVOKE_SECOND_AGENT_ORIGINAL_RECEIPT_INTACT=PASS',
    'S25_V_R12_LOST_ACK_EXACT_ONE_NATIVE_EXEC_ORIGINAL=PASS',
    'P1B_S25_OUTER_EXACT_IMAGE_EXPORTED=PASS',
    'P1B_S25_INNER_EXACT_IMAGE_IMPORTED=PASS',
    'P1B_S25_V_TWO_LAYER_IMAGE_IDENTITY=PASS',
    'P1B_S25_INNER_LOCAL_DAEMON_SOCKET_BOUNDARY=PASS',
    'P1B_S25_SCOPED_CROSS_FIXTURE_TOKEN_AUTH_DENIED=PASS',
    'P1B_S25_SCOPED_SAME_NAME_DISTINCT_NATIVE_NO_CROSS_ADMISSION=PASS',
    'P1B_S25_SCOPED_TWO_LAYER_FIXED_LEDGER_ACL_GUARD=PASS',
    'P1B_S25_SCOPED_STARTUP_LIFECYCLE=PASS',
    'S25_V_INNER_ORIGINAL_MCP_L2_DELETE_NATIVE_ABSENCE=PASS',
    'S25_V_EXACT_NATIVE_L0_CLEANUP_ORIGINAL_SQLITE=PASS',
    'S25_R12_L1_FOREIGN_PACKET_AND_SOCKET_INDEPENDENT_V=PASS',
    'S25_V_R12_L1_FOREIGN_NATIVE_UUID_CLEANUP=PASS',
    'P1B_S25_SCOPED_FOREIGN_PACKET=PASS',
    'P1B_S25_SCOPED_ACTIVE_LEGACY_MAINTENANCE_REFUSED_NO_EFFECT=PASS',
    'P1B_S25_SCOPED_OFFLINE_LEGACY_MAINTENANCE_REFUSED_NO_EFFECT=PASS',
    'P1B_S25_SCOPED_MCP_ORIGINAL_SUCCEEDED_RECEIPTS_AFTER_MAINTENANCE=PASS',
    'P1B_S25_V_SCOPED_ORIGINAL_OPS_EXACT_BINARY_SQLITE_NATIVE_MAINTENANCE=PASS',
    'P1B_S25_SCOPED_MAINTENANCE_NO_EFFECT_ORIGINALS=PASS',
    'P1B_S25_SCOPED_NATIVE_BOOTSTRAP=outer:PASS',
    'P1B_S25_SCOPED_STALE_REVIEW_REFUSED=outer:PASS',
    'P1B_S25_SCOPED_FRESH_INSTALL=outer:PASS',
    'P1B_S25_SCOPED_NATIVE_BOOTSTRAP=inner:PASS',
    'P1B_S25_SCOPED_STALE_REVIEW_REFUSED=inner:PASS',
    'P1B_S25_SCOPED_FRESH_INSTALL=inner:PASS',
    'P1B_S25_SCOPED_CROSS_ID_REFUSED=L0:PASS',
    'P1B_S25_SCOPED_CROSS_ID_REFUSED=L1:PASS',
    'P1B_S25_LEGACY_SUCCESSFUL_UPGRADE_ROLLBACK=OUT_OF_SCOPE_170',
    'S25_TARGETED_MCP_DUAL_AGENT_RECOVERY_VERIFIER=PASS',
)
ORIGINALS = ("P1B_S25_SCOPED_ORIGINAL synthetic-L0",
             "P1B_S25_SCOPED_ORIGINAL synthetic-L1")


class EvidenceCheck(unittest.TestCase):
    def check(self, lines, source=SOURCE):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "synthetic.log"
            path.write_text("\n".join(lines) + "\n")
            return subprocess.run(["sh", str(CHECK), str(path), source],
                                  capture_output=True, timeout=10).returncode

    def test_full_fixture(self):
        self.assertEqual(self.check(["S25_LAB_SOURCE_SHA=" + SOURCE,
                                     *MARKERS, *ORIGINALS]), 0)

    def test_each_required_marker_is_mandatory(self):
        for marker in MARKERS:
            with self.subTest(marker=marker):
                lines = ["S25_LAB_SOURCE_SHA=" + SOURCE, *ORIGINALS,
                         *(m for m in MARKERS if m != marker),
                         "echo " + marker]
                self.assertNotEqual(self.check(lines), 0)

    def test_source_and_original_counts(self):
        lines = ["S25_LAB_SOURCE_SHA=" + SOURCE, *MARKERS, *ORIGINALS]
        for bad in (lines[1:], lines + [lines[0]],
                    ["S25_LAB_SOURCE_SHA=" + "b" * 40, *lines[1:]],
                    lines[:-1], lines + [ORIGINALS[0]]):
            self.assertNotEqual(self.check(bad), 0)
        self.assertNotEqual(self.check(lines, "invalid"), 0)

    def test_legacy_success_is_not_substitute(self):
        lines = ["S25_LAB_SOURCE_SHA=" + SOURCE, *ORIGINALS,
                 *(m for m in MARKERS if "OUT_OF_SCOPE_170" not in m),
                 "P1B_S25_LEGACY_SUCCESSFUL_UPGRADE_ROLLBACK=PASS"]
        self.assertNotEqual(self.check(lines), 0)


if __name__ == "__main__":
    unittest.main()
