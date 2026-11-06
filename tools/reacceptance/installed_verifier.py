"""Independent *read-only* installed-verifier inspection for a pinned CI archive.

Produces observed evidence and per-scenario classifications, never a release
signature or approval. Operate in a separate, administrator-approved Box,
using original bytes from CI; any missing scope is BLOCKED, not PASS.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

from bwmini.distribution import InstallError, verify as verify_package
from .engine import AcceptanceFault, capture, now, private_mkdir, unique_file

VERSION = re.compile(r"^0\.[0-9]+\.[0-9]+$")
HEX40 = re.compile(r"^[a-f0-9]{40}$")
HEX64 = re.compile(r"^[a-f0-9]{64}$")
REQUIRED_HOST_CHECKS = frozenset((
    "public_descriptor", "web_loopback", "host_credential_timer",
    "host_trust_timer", "host_user_manager", "app_release_health",
    "app_serve_unit", "app_worker_unit", "authenticated_mcp",
    "admin_initialized",
))


def exact_read(argv, reason):
    record = capture(argv, timeout=60, limit=2 * 1024 * 1024)
    if record.get("execution") != "completed" or record.get("exit_code") != 0:
        raise AcceptanceFault(reason)
    return record


def captured_json(argv, reason):
    record = exact_read(argv, reason)
    try:
        value = json.loads(record["stdout"])
    except ValueError:
        raise AcceptanceFault(reason + "_NON_JSON") from None
    if not isinstance(value, dict):
        raise AcceptanceFault(reason + "_INVALID_DATA")
    return value, record


def check_lifecycle(path):
    if not path:
        return {"status": "BLOCKED", "reason": "VERIFIER_LIVE_LIFECYCLE_NOT_RUN"}
    base = Path(path)
    if not base.is_dir() or base.is_symlink():
        return {"status": "BLOCKED", "reason": "VERIFIER_LIFECYCLE_FILES_MISSING"}
    try:
        outcome = json.loads((base / "result.json").read_text())
        if outcome.get("state") != "PARTIAL" or outcome.get("scenario_id") != "SYS-LIFE-01":
            raise AcceptanceFault("VERIFIER_LIFECYCLE_NOT_CLEAN")
        observations, incarnation = [], None
        expected = ["Running", "Stopped", "Running", None]
        for i, name in enumerate(("create_container", "stop_container",
                                  "start_container", "delete_container"), start=1):
            stem = f"{i:02d}-{name}"
            proof = json.loads((base / (stem + ".attempt.json")).read_text())
            oracle = json.loads((base / (stem + ".oracle.json")).read_text())
            if proof.get("tool") != name or proof.get("success_response") is not True:
                raise AcceptanceFault("VERIFIER_LIFECYCLE_INCOMPLETE_STEP")
            process = proof.get("process") or {}
            if process.get("execution") != "completed" or process.get("exit_code") != 0:
                raise AcceptanceFault("VERIFIER_LIFECYCLE_UNACKNOWLEDGED")
            native = oracle.get("native_child")
            if (native or {}).get("status") != expected[i-1]:
                raise AcceptanceFault("VERIFIER_LIFECYCLE_NATIVE_STATE_MISMATCH")
            observed_id = oracle.get("incarnation")
            if i == 1:
                if not isinstance(observed_id, dict) or not observed_id.get("uuid"):
                    raise AcceptanceFault("VERIFIER_LIFECYCLE_ORIGINAL_IDENTITY_MISSING")
                incarnation = observed_id
            elif i < 4 and observed_id != incarnation:
                raise AcceptanceFault("VERIFIER_LIFECYCLE_INCARNATION_CHANGED")
            elif i == 4 and observed_id is not None:
                raise AcceptanceFault("VERIFIER_LIFECYCLE_DELETE_NOT_OBSERVED")
            observations.append({"action": name, "native_state": expected[i-1]})
        return {"status": "PARTIAL", "reason": "FOUR_LIFECYCLE_STEPS_REAL_NATIVE_ORACLES_BUT_FULL_GRAPH_INCOMPLETE",
                "native_incarnation": incarnation, "steps": observations,
                "evidence_sha256": hashlib.sha256((base / "result.json").read_bytes()).hexdigest()}
    except (OSError, ValueError, KeyError, TypeError, AcceptanceFault) as exc:
        return {"status": "BLOCKED", "reason": "VERIFIER_LIFECYCLE_PROOF_INVALID",
                "validation_error": str(exc)[:120]}


def check_selftest(path):
    if not path:
        return {"status": "BLOCKED", "reason": "VERIFIER_CONCURRENT_SELFTEST_NOT_RUN"}
    base = Path(path)
    if not base.is_dir() or base.is_symlink():
        return {"status": "BLOCKED", "reason": "VERIFIER_SELFTEST_FILES_MISSING"}
    try:
        receipt = json.loads((base / "outcome.json").read_text())
        original = json.loads((base / "original-complete.json").read_text())
        competing = json.loads((base / "competing.json").read_text())
        later = json.loads((base / "after-contention.json").read_text())
        first = json.loads(original.get("stdout", "{}"))
        second = json.loads(competing.get("stdout", "{}"))
        third = json.loads(later.get("stdout", "{}"))
        steps = sorted(x.get("step") for x in first.get("steps", []))
        later_steps = sorted(x.get("step") for x in third.get("steps", []))
        if (receipt.get("state") != "INSTALLED_BLACKBOX_PASS" or
            receipt.get("proof_kind") != "same_agent_installed_two_process" or
            receipt.get("residue") != 0 or
            receipt.get("competing_exit") != 2 or receipt.get("later_exit") != 0 or
            original.get("exit_code") != 0 or competing.get("exit_code") != 2 or
            later.get("exit_code") != 0 or
            first.get("state") != "passed" or third.get("state") != "passed" or
            second.get("code") != "SELFTEST_RUN_UNRECONCILED" or
            steps != ["create", "delete", "hello", "stop"] or later_steps != steps):
            raise AcceptanceFault("VERIFIER_CONCURRENT_SELFTEST_NOT_VERIFIED")
        return {"status": "PARTIAL", "reason": "SINGLE_BOX_TWO_PROCESS_BLACKBOX_NOT_INDEPENDENT_AGENT",
                "original_box": receipt["original_box"],
                "competing_box": receipt["competing_box"],
                "later_box": receipt["later_box"],
                "evidence_sha256": hashlib.sha256((base / "outcome.json").read_bytes()).hexdigest()}
    except (OSError, ValueError, KeyError, TypeError, AcceptanceFault) as exc:
        return {"status": "BLOCKED", "reason": "VERIFIER_SELFTEST_PROOF_INVALID",
                "validation_error": str(exc)[:120]}


def inspect(candidate, manifest_file, expected_source, output,
            *, lifecycle=None, selftest=None):
    if not HEX40.fullmatch(expected_source):
        raise AcceptanceFault("VERIFIER_SOURCE_INVALID")
    archive = Path(candidate)
    if not archive.is_file() or archive.is_symlink() or archive.stat().st_size > 180_000_000:
        raise AcceptanceFault("VERIFIER_CANDIDATE_INVALID")
    actual = hashlib.sha256(archive.read_bytes()).hexdigest()
    try:
        manifest = json.loads(Path(manifest_file).read_text())
        package = verify_package(archive, actual)
    except (OSError, ValueError, KeyError, InstallError) as exc:
        raise AcceptanceFault("VERIFIER_MANIFEST_OR_ARCHIVE_INVALID") from exc
    if (manifest.get("source") != expected_source or
        manifest.get("files", {}).get("boxweave.zip") != actual or
        package.get("source") != {"commit": expected_source, "state": "clean"}):
        raise AcceptanceFault("VERIFIER_EXACT_CANDIDATE_MISMATCH")
    version = package.get("version")
    if not isinstance(version, str) or not VERSION.fullmatch(version):
        raise AcceptanceFault("VERIFIER_PRODUCT_VERSION_INVALID")
    version_record = exact_read(["sudo", "-n", "boxweave", "version"], "VERIFIER_HOST_CLI_UNAVAILABLE")
    if version not in version_record["stdout"]:
        raise AcceptanceFault("VERIFIER_HOST_VERSION_MISMATCH")
    prov, prov_record = captured_json(
        ["sudo", "-n", "boxweave", "provenance"], "VERIFIER_PROVENANCE_UNAVAILABLE")
    doctor, doctor_record = captured_json(
        ["sudo", "-n", "boxweave", "doctor", "--mcp"],
        "VERIFIER_DOCTOR_UNAVAILABLE")
    app, app_record = captured_json(
        ["sudo", "-n", "-u", "boxweave", "/usr/bin/cat",
         "/home/boxweave/boxweave/installed.json"],
        "VERIFIER_APP_INSTALLATION_UNAVAILABLE")
    if (prov.get("version") != version or prov.get("source") != expected_source or
        app.get("version") != version or
        app.get("source") != {"commit": expected_source, "state": "clean"} or
        app.get("archive_sha256") != actual):
        raise AcceptanceFault("VERIFIER_HOST_APP_NOT_EXACT_CANDIDATE")
    checks = doctor.get("checks")
    if (not isinstance(checks, list) or not REQUIRED_HOST_CHECKS.issubset(
            {row.get("id") for row in checks if isinstance(row, dict) and
             row.get("state") == "passed"})):
        raise AcceptanceFault("VERIFIER_INSTALLED_DOCTOR_NOT_FULLY_PASSED")
    cases = {
        "SYS-LIFE-01": check_lifecycle(lifecycle),
        "SYS-SELFTEST-01": check_selftest(selftest),
        "SYS-UPGRADE-01": {"status": "BLOCKED",
                           "reason": "CLEAN_INSTALL_IS_NOT_UPGRADE"},
        "SYS-UNKNOWN-01": {"status": "BLOCKED",
                           "reason": "NO_SAFE_INDEPENDENT_UNKNOWN_REPLAY"},
        "SYS-JOB-01": {"status": "BLOCKED",
                       "reason": "NO_REAL_MULTI_ACTOR_TASK_TEST"},
        "SYS-RELEASE-01": {"status": "PARTIAL",
                           "reason": "EXACT_PRIVATE_CANDIDATE_INSTALLED_BUT_NOT_PUBLIC_RELEASE"},
        "SYS-REPORT-01": {"status": "BLOCKED",
                          "reason": "EXTERNAL_REPORT_GATE_NOT_PROVEN"},
    }
    dest = Path(output)
    if dest.exists() or dest.is_symlink():
        raise AcceptanceFault("VERIFIER_OUTPUT_EXISTS")
    private_mkdir(dest)
    result = {"schema_version": 1, "observed_at": now(),
              "state": "INDEPENDENT_INSTALLED_CANDIDATE_PARTIAL",
              "candidate_sha256": actual, "source_sha": expected_source,
              "version": version,
              "host_app_parity": "PASS",
              "doctor_checks_passed": len(REQUIRED_HOST_CHECKS),
              "case_matrix": cases,
              "release_gate_approved": False,
              "signature": "NOT_PRODUCED_UNTIL_ALL_REQUIRED_CASES_PASS"}
    unique_file(dest / "verifier-report.json", result)
    for key, value in (("host-version", version_record),
                       ("provenance", prov_record), ("doctor", doctor_record),
                       ("installed-app", app_record)):
        # No secrets are needed in report; the validated metadata is enough.
        clean = {"observed_at": now(),
                 "command_label": key, "exit_code": value.get("exit_code"),
                 "stdout_sha256": value.get("stdout_sha256"),
                 "stderr_sha256": value.get("stderr_sha256")}
        unique_file(dest / (key + ".proof.json"), clean)
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--candidate", required=True)
    parser.add_argument("--manifest", required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--lifecycle")
    parser.add_argument("--selftest")
    args = parser.parse_args(argv)
    try:
        print(json.dumps(inspect(args.candidate, args.manifest, args.source,
                                 args.out, lifecycle=args.lifecycle,
                                 selftest=args.selftest), sort_keys=True))
        return 0
    except (AcceptanceFault, ValueError, KeyError, TypeError, OSError) as exc:
        print(json.dumps({"state": "FAIL_CLOSED", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())

