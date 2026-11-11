#!/usr/bin/env python3
"""Read-only independent *evidence manifest*, not a substitute for native tests.

Run after the one-use native S2 acceptance on its stdout log. This process
never has Incus credentials and cannot create, modify or delete a Box.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import sys

MANDATORY = (
    "S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS",
    "S2_NATIVE_MCP_DISCOVERY_13_OF_13",
    "S2_NATIVE_PACKET_IPV4_IPV6_SAME_BRIDGE_DEFAULT_DENY",
    "S2_NATIVE_PACKET_INTERNET_TOGGLE_IPV4_IPV6_GATEWAY_EGRESS",
    "S2_NATIVE_SQLITE_FOREIGN_CORRUPT_FAIL_CLOSED",
    "S2_NATIVE_CREATE_ACK_LOSS_ORIGINAL_UNKNOWN",
    "S2_NATIVE_RUNNING_NOT_CONFUSED_WITH_GUEST_READY",
    "S2_P1_REAL_NATIVE_BOUND_VERIFYING_CONCURRENT_DELETE_FULL_LIFECYCLE",
    "S2_P1_REAL_NATIVE_BOUND_VERIFYING_TWELVE_SDK_DELETE_REFUSED",
    "S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_NATIVE_CREATE_UPDATE_EXEC_DELETE",
    "S2_P1_TWO_CONTROLLER_INDEPENDENT_NATIVE_AND_MCP_ISOLATION",
    "S2_P1_REAL_INSTALLED_OPERATION_LIST_INFLIGHT_REVOCATION_AND_EXPIRY_NO_LEAK",
    "S2_P1_FOUR_REAL_PRIVATE_READ_TOKEN_RACES_NATIVE_POSTREAD_EMPTY",
    "S2_MCP_REAL_GUEST_QUOTA_ORIGINAL_RECEIPT_OUTPUT_EXIT_ONE",
    "S2_P2_REAL_MALFORMED_EXEC_NATIVE_202_UNKNOWN_RESTART_CLEANUP",
    "S2_P2_STALLED_NATIVE_OUTPUT_CANCEL_NO_NATIVE_CHANGE",
    "S2_P2_ALTERNATE_POOL_LIVE_EXPANDED_ROOT_MATCHES_UUID_POOL_QUOTA",
    "S2_P0_NATIVE_202_COMBINED_WITH_SQLITE_COMMIT_FAULT_BOTH_PHASES",
    "S2_GC_APPROVED_SEVEN_DAY_COMPLETE_NATIVE_MCP_CYCLE",
)
SUITE_MARKER = "S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS"

def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--log", required=True, type=Path)
    p.add_argument("--source", required=True)
    p.add_argument("--output", type=Path)
    args = p.parse_args()
    if not re.fullmatch(r"[0-9a-f]{40}", args.source):
        raise SystemExit("source is not a full Git commit SHA")
    raw = args.log.read_bytes()
    if len(raw) > 32 * 1024 * 1024 or len(raw) < 200:
        raise SystemExit("missing or oversized S2 native witness log")
    lines = raw.decode("utf-8", "replace").splitlines()
    expected = f"S2_NATIVE_CANDIDATE_SOURCE_SHA={args.source}"
    if sum(line.strip() == expected for line in lines) != 1:
        raise SystemExit("wrong source SHA / duplicate source marker")
    hashes = []
    for line in lines:
        m = re.fullmatch(r"([0-9a-f]{64})  (/tmp/bw-v1-s2-ghci\.[^/\s]+/boxweave)", line.strip())
        if m:
            hashes.append(m.group(1))
    if len(hashes) != 1:
        raise SystemExit("candidate unmodified executable digest absent or ambiguous")
    found = set(line.strip() for line in lines)
    missing = [name for name in MANDATORY if name + "=PASS" not in found]
    if missing:
        raise SystemExit("missing required exact acceptance witnesses: " + ", ".join(missing))
    # Native gateway packet egress alone can never certify Internet access.
    if "S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS" in found and (
        "S2_NATIVE_EXTERNAL_IPV4_WAN_HTTPS_POSITIVE_CONTROL=PASS" not in found
    ):
        raise SystemExit("WAN toggle claimed success without verified public positive")
    wan = (
        "PASS"
        if "S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS" in found
        else "UNVERIFIED_GUEST_WAN_POSITIVE"
    )
    result = {
        "kind": "S2_NATIVE_EVIDENCE_INDEX_NOT_HARDWARE_SIGNOFF",
        "source_sha": args.source,
        "native_candidate_executable_sha256": hashes[0],
        "log_sha256": hashlib.sha256(raw).hexdigest(),
        "required_markers_checked": len(MANDATORY),
        "passed_required_markers": list(MANDATORY),
        "wan_public_ipv4_https": wan,
        "physical_power_removal": "BLOCKED_NO_AUTHORIZED_HARDWARE",
        "live_guest_readiness_positive": "UNSUPPORTED_NOT_CLAIMED",
        "network_dependent_delete_A_043_047": "DEFERRED_S4",
        "exit_status": "S2_PARTIAL_PENDING_INDEPENDENT_TC_A_REVIEW",
    }
    encoded = json.dumps(result, indent=2, sort_keys=True) + "\n"
    if args.output:
        if not args.output.parent.is_dir():
            raise SystemExit("evidence output directory does not exist")
        args.output.write_text(encoded, encoding="utf-8")
    print("S2_P3_INDEPENDENT_EXACT_SOURCE_BINARY_MARKER_AUDIT=PASS")
    print("S2_P3_MANDATORY_MARKERS_CHECKED=" + str(len(MANDATORY)))
    print("S2_P3_UNMODIFIED_NATIVE_BINARY_SHA256=" + hashes[0])
    print("S2_P3_PUBLIC_WAN_STATUS=" + wan)
    print("S2_P3_EXIT_STATUS=S2_PARTIAL_PENDING_INDEPENDENT_TC_A_REVIEW")
    return 0

if __name__ == "__main__":
    sys.exit(main())
