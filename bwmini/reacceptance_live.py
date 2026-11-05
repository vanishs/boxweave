"""Authorized, test-owned live selftest contention probe.

Runs only inside a pre-approved disposable Box acting as a nested Provider host.
Never modifies the owner host; dispatched UNKNOWN is preserved without retry.
"""
from __future__ import annotations
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import sys
import time

from .reacceptance import (AcceptanceFault, canonical, capture, digest, load,
                           now, private_mkdir, unique_file)

CONFIRM = "I_AUTHORIZE_TEST_OWNED_NESTED_SELFTEST"
MAX_SECONDS = 540


def verify_lab(identity):
    p = Path(identity)
    if p.is_symlink() or p.stat().st_mode & 0o077 or p.stat().st_uid != os.geteuid():
        raise AcceptanceFault("LAB_PROVENANCE_FILE_NOT_PRIVATE")
    manifest = load(p)
    if manifest.get("scope") != "test_owned_nested_selftest_only" or manifest.get("approved") is not True:
        raise AcceptanceFault("LAB_NOT_EXPLICITLY_APPROVED")
    if manifest.get("hostname") != socket.gethostname():
        raise AcceptanceFault("LAB_HOSTNAME_CHANGED")
    mid = Path("/etc/machine-id").read_text().strip()
    if manifest.get("machine_id") != mid or not mid:
        raise AcceptanceFault("LAB_MACHINE_ID_CHANGED")
    if not re.fullmatch(r"[a-f0-9-]{36}", str(manifest.get("outer_uuid", ""))):
        raise AcceptanceFault("LAB_OUTER_UUID_MISSING")
    if not re.fullmatch(r"\d{9,13}", str(manifest.get("outer_created_at", ""))):
        raise AcceptanceFault("LAB_OUTER_INCARNATION_MISSING")
    if not 0 < manifest.get("expires_epoch", 0) - time.time() < 86400:
        raise AcceptanceFault("LAB_APPROVAL_EXPIRED_OR_UNBOUNDED")
    return manifest


def native_children():
    result = capture(["sudo", "-n", "incus", "list", "-f", "csv", "-c", "n"], timeout=18)
    if result["execution"] != "completed" or result["exit_code"] != 0:
        raise AcceptanceFault("NESTED_PROVIDER_READ_UNAVAILABLE")
    return set(result["stdout"].splitlines())


def plan_step(folder, name):
    response = capture(["sudo", "-n", "boxweave", "selftest", "plan", "--budget", "420"],
                       timeout=35)
    if response["execution"] != "completed" or response["exit_code"] != 0:
        raise AcceptanceFault("SELFTEST_PLAN_FAILED_" + name)
    try:
        payload = json.loads(response["stdout"])
        plan, token = payload["plan"], payload["confirmation"]
        if not re.fullmatch(r"boxweave-selftest-[a-z0-9]+", plan["box"]):
            raise ValueError()
        if digest(plan) != token:
            raise ValueError()
    except (ValueError, TypeError, KeyError):
        raise AcceptanceFault("SELFTEST_PLAN_MALFORMED_" + name) from None
    path = Path(folder) / (name + ".plan.json")
    unique_file(path, plan)
    unique_file(Path(folder) / (name + ".plan-response.json"), response)
    return {"name": name, "box": plan["box"], "file": str(path),
            "confirmation": token}


def selftest_cmd(item):
    return ["sudo", "-n", "boxweave", "selftest", "run", "--plan", item["file"],
            "--confirmation", item["confirmation"], "--budget", "420"]


def exact_step(result, box, expected_state):
    if result["execution"] != "completed":
        raise AcceptanceFault("SELFTEST_UNKNOWN_" + box)
    try:
        value = json.loads(result["stdout"])
    except ValueError:
        raise AcceptanceFault("SELFTEST_NONJSON_" + box) from None
    if expected_state == "passed":
        if result["exit_code"] != 0 or value.get("state") != "passed":
            raise AcceptanceFault("SELFTEST_NOT_PASSED_" + box)
        if sorted(step.get("step") for step in value.get("steps", [])) != [
                "create", "delete", "hello", "stop"]:
            raise AcceptanceFault("SELFTEST_STEPS_INCOMPLETE_" + box)
    else:
        if result["exit_code"] != 2 or value.get("code") != "SELFTEST_RUN_UNRECONCILED":
            raise AcceptanceFault("SELFTEST_COMPETITOR_NOT_REFUSED_" + box)
    return value


def bounded_run(original, timeout=MAX_SECONDS):
    with open(original["stdout"], "xb") as out, open(original["stderr"], "xb") as err:
        return subprocess.Popen(selftest_cmd(original), stdin=subprocess.DEVNULL,
                                stdout=out, stderr=err)


def probe(lab_file, directory, confirmation):
    if confirmation != CONFIRM:
        raise AcceptanceFault("LIVE_MUTATION_EXPLICIT_CONFIRMATION_REQUIRED")
    scope = verify_lab(lab_file)
    root = Path(directory)
    if root.exists():
        raise AcceptanceFault("LIVE_EVIDENCE_DIRECTORY_EXISTS")
    private_mkdir(root)
    unique_file(root / "lab.json", {
        "verified": True, "scope": "test_owned_nested_selftest_only",
        "hostname": scope["hostname"], "outer_uuid": scope["outer_uuid"],
        "outer_created_at": scope["outer_created_at"], "started_utc": now()})
    if any(x.startswith("boxweave-selftest-") for x in native_children()):
        raise AcceptanceFault("EXISTING_NESTED_SELFTEST_REFUSE_OVERLAP")
    original = plan_step(root, "original")
    competing = plan_step(root, "competing")
    first = {**original, "stdout": str(root / "original.stdout"),
             "stderr": str(root / "original.stderr")}
    p = bounded_run(first)
    unique_file(root / "original-dispatch.json", {"pid": p.pid, "started_utc": now(),
                                                   "box": first["box"]})
    # Observe an actual side effect, not a guessed timing overlap.
    deadline = time.monotonic() + 150
    observed = False
    while time.monotonic() < deadline:
        if first["box"] + "-container" in native_children():
            observed = True
            break
        if p.poll() is not None:
            break
        time.sleep(1)
    if not observed:
        unique_file(root / "unknown.json", {"state": "UNKNOWN",
                    "reason": "ORIGINAL_IN_FLIGHT_NOT_OBSERVED",
                    "original_pid": p.pid, "original_box": first["box"]})
        raise AcceptanceFault("ORIGINAL_UNKNOWN_PRESERVED_NO_RETRY")
    unique_file(root / "original-inflight.json", {
        "box": first["box"], "first_child_present": True, "observed_utc": now()})
    second = capture(selftest_cmd(competing), timeout=40)
    unique_file(root / "competing.json", second)
    exact_step(second, competing["box"], "refused")
    if competing["box"] + "-container" in native_children():
        raise AcceptanceFault("COMPETITOR_CREATED_FORBIDDEN_CHILD")
    try:
        rc = p.wait(timeout=MAX_SECONDS)
    except subprocess.TimeoutExpired:
        unique_file(root / "unknown.json", {"state": "UNKNOWN", "original_box": first["box"],
                                            "reason": "ORIGINAL_COMPLETION_TIMEOUT_NO_RETRY"})
        raise AcceptanceFault("ORIGINAL_UNKNOWN_PRESERVED_NO_RETRY") from None
    first_result = {"execution": "completed", "exit_code": rc,
                    "stdout": Path(first["stdout"]).read_text(errors="replace")[:65536],
                    "stderr": Path(first["stderr"]).read_text(errors="replace")[:65536]}
    unique_file(root / "original-complete.json", first_result)
    exact_step(first_result, first["box"], "passed")
    if first["box"] + "-container" in native_children():
        raise AcceptanceFault("ORIGINAL_CHILD_CLEANUP_UNVERIFIED")
    after = plan_step(root, "after-contention")
    last = capture(selftest_cmd(after), timeout=MAX_SECONDS)
    unique_file(root / "after-contention.json", last)
    exact_step(last, after["box"], "passed")
    remaining = {x for x in native_children() if x.startswith("boxweave-selftest-")}
    if remaining:
        raise AcceptanceFault("NESTED_SELFTEST_RESIDUE_PRESERVE_" + str(len(remaining)))
    success = {"state": "INSTALLED_BLACKBOX_PASS", "original_box": first["box"],
               "competing_box": competing["box"], "later_box": after["box"],
               "competing_exit": 2, "later_exit": 0, "residue": 0,
               "observed_utc": now(), "proof_kind": "same_agent_installed_two_process"}
    unique_file(root / "outcome.json", success)
    return success


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--lab", required=True, help="Approved private exact outer Box identity manifest")
    p.add_argument("--out", required=True, help="New private evidence directory, never reused")
    p.add_argument("--confirm", required=True)
    args = p.parse_args(argv)
    try:
        print(json.dumps(probe(args.lab, args.out, args.confirm), sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError) as exc:
        print(json.dumps({"status": "BLOCKED_OR_UNKNOWN", "code": str(exc),
                          "replay": "FORBIDDEN_IF_DISPATCHED"}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())

