#!/usr/bin/env python3
"""Read-only S2 grouped TC/A evidence crosswalk, with explicit exit exceptions.

This is a *scoped* exit check, not a physical-outage test or a final v1 release.
It runs only on the exact original disposable native acceptance transcript.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

HERE = Path(__file__).resolve().parent
POLICY = HERE / "s2_p3_scoped_acceptance.json"
SOURCE_RE = re.compile(r"^[0-9a-f]{40}$")
BINARY_RE = re.compile(r"^([0-9a-f]{64})  /tmp/bw-v1-s2-ghci\.[^/\s]+/boxweave$")


def evaluate(raw, source, policy=None):
    if not SOURCE_RE.fullmatch(source):
        raise ValueError("invalid source SHA")
    if len(raw) < 200 or len(raw) > 32 * 1024 * 1024:
        raise ValueError("missing or oversized original native transcript")
    text = raw.decode("utf-8", "replace")
    lines = [line.strip() for line in text.splitlines()]
    if lines.count("S2_NATIVE_CANDIDATE_SOURCE_SHA=" + source) != 1:
        raise ValueError("missing, duplicate or mismatched exact source")
    binaries = [m.group(1) for line in lines if (m := BINARY_RE.fullmatch(line))]
    if len(binaries) != 1:
        raise ValueError("missing/duplicate original *unmodified* candidate binary hash")
    markers = set(lines)
    if "S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS" not in markers:
        raise ValueError("native end-to-end suite never finished")
    if "S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS" in markers and \
       "S2_NATIVE_EXTERNAL_IPV4_WAN_HTTPS_POSITIVE_CONTROL=PASS" not in markers:
        raise ValueError("public WAN falsely claimed without independent TLS positive control")
    # A physical outage cannot be invented by adding PASS marker to a log.
    if "S2_NATIVE_PHYSICAL_POWER_OUTAGE=PASS" in markers:
        raise ValueError("hardware power outage marker unsupported by S2 surrogate")
    cases = json.loads(POLICY.read_text() if policy is None else json.dumps(policy))
    if len(cases) != 19 or len({c["id"] for c in cases}) != 19:
        raise ValueError("exact 10 TC + 9 grouped A-case rows required")
    if [c["id"] for c in cases[:10]] != [f"TC-{i:03d}" for i in range(1,11)]:
        raise ValueError("TC-001..010 must all be present and in order")
    for case in cases:
        if case["verdict"] not in ("PASS_TESTED_S2_SCOPE", "DEFERRED_S4"):
            raise ValueError("illegal S2 evidence disposition "+case["id"])
        if not case["required_markers"] or not case.get("limits"):
            raise ValueError("missing explicit native proof or scope limit: "+case["id"])
        missing = [x for x in case["required_markers"] if x+"=PASS" not in markers]
        if missing:
            raise ValueError("native evidence for "+case["id"]+" missing: "+", ".join(missing))
    deferred = [c for c in cases if c["verdict"] == "DEFERRED_S4"]
    if len(deferred) != 1 or deferred[0]["id"] != "A-043-047":
        raise ValueError("only S4 dependent Delete may be deferred from S2")
    required_surrogate = [
      "S2_SQLITE_WAL_FSYNC_BEFORE_REAL_KILL_UNKNOWN_NO_REPLAY",
      "S2_SQLITE_WAL_FSYNC_AFTER_REAL_KILL_UNKNOWN_NO_REPLAY",
      "S2_SQLITE_ACTUAL_ISOLATED_EXT4_ENOSPC_RECOVERY_UNKNOWN_NO_REPLAY",
      "S2_P0_NATIVE_202_COMBINED_WITH_SQLITE_COMMIT_FAULT_BOTH_PHASES",
    ]
    if any(x+"=PASS" not in markers for x in required_surrogate):
        raise ValueError("operator-approved crash/ENOSPC surrogate not independently proved")
    if "S2_NATIVE_RUNNING_NOT_CONFUSED_WITH_GUEST_READY=PASS" not in markers:
        raise ValueError("unprobed native Running may be falsely called guest-ready")
    wan = ("PASS" if "S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS" in markers
           else "UNVERIFIED_ENVIRONMENT_EXCEPTION")
    return {
      "kind":"S2_SCOPE_SPECIFIC_TC_A_ACCEPTANCE_NOT_FINAL_V1_RELEASE",
      "source_sha":source,
      "original_unmodified_candidate_sha256":binaries[0],
      "native_witness_log_sha256":hashlib.sha256(raw).hexdigest(),
      "grouped_cases":cases,
      "groups":len(cases),
      "supported_s2_scoped_cases_with_native_evidence":len(cases)-len(deferred),
      "s4_deferred_groups":len(deferred),
      "power_fault_acceptance":"ACCEPTED_SIMULATED_SIGKILL_WAL_FSYNC_ENOSPC_NOT_PHYSICAL",
      "physical_power_cycle_actual":"NOT_TESTED_NO_HARDWARE_CLAIM",
      "public_guest_wan_https":wan,
      "guest_ready_contract":"STOPPED_FALSE_RUNNING_UNVERIFIED_NULL",
      "candidate_release_status":"DRAFT_UNMERGED_NOT_RELEASED",
      "outcome":"S2_SUPPORTED_SCOPE_EVIDENCE_ACCEPTED_WITH_EXPLICIT_EXCEPTIONS",
    }


def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--log",type=Path,required=True)
    ap.add_argument("--source",required=True)
    ap.add_argument("--output",type=Path)
    args=ap.parse_args()
    try:
        report=evaluate(args.log.read_bytes(),args.source)
        rendered=json.dumps(report,indent=2,ensure_ascii=False)+"\n"
        if args.output:
            if not args.output.parent.is_dir():raise ValueError("manifest output parent does not exist")
            args.output.write_text(rendered,encoding="utf-8")
        print("S2_P3_ALL_NINETEEN_GROUPED_TC_A_EVIDENCE_SCOPED_REVIEW=PASS")
        print("S2_P3_SCOPED_NATIVE_GROUPS_VERIFIED="+str(report["supported_s2_scoped_cases_with_native_evidence"]))
        print("S2_P3_POWER_FAULT_SURROGATE=ACCEPTED_SIGKILL_WAL_ENOSPC_NOT_HARDWARE")
        print("S2_P3_PUBLIC_GUEST_WAN="+report["public_guest_wan_https"])
        print("S2_P3_SUPPORTED_SCOPE_WITH_EXCEPTIONS=ACCEPTED_NOT_FINAL_RELEASE")
    except (ValueError,OSError,KeyError,TypeError,json.JSONDecodeError) as err:
        print("S2_P3_SCOPE_REVIEW_REJECTED: "+str(err),file=sys.stderr)
        return 1
    return 0

if __name__=="__main__":
    sys.exit(main())
