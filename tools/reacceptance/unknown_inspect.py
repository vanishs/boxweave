"""Read-only exact-incarnation inspection of an originally UNKNOWN MCP create.

Never retries, releases a lease, deletes, starts or stops a Box. Retains the
original finder evidence and creates a new immutable observation packet for
an administrator or an independent Verifier to inspect.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import stat
import sys

from .engine import AcceptanceFault, now, private_mkdir, unique_file
from . import lifecycle

MAX_RECORD = 4_194_304


def original_record(directory, filename):
    p = directory / filename
    meta = p.lstat()
    if (not stat.S_ISREG(meta.st_mode) or meta.st_uid != os.geteuid() or
            meta.st_mode & 0o077 or meta.st_size <= 0 or
            meta.st_size > MAX_RECORD):
        raise AcceptanceFault("UNKNOWN_ORIGINAL_RECORD_UNTRUSTED_" + filename)
    content = p.read_bytes()
    try:
        data = json.loads(content)
    except (ValueError, UnicodeError):
        raise AcceptanceFault("UNKNOWN_ORIGINAL_RECORD_INVALID_" + filename) from None
    if not isinstance(data, dict):
        raise AcceptanceFault("UNKNOWN_ORIGINAL_RECORD_INVALID_" + filename)
    return data, hashlib.sha256(content).hexdigest()


def inspect(original_dir, grant_file, confirmation, out):
    grant = lifecycle.grant_read(grant_file, confirmation)
    original = Path(original_dir)
    meta = original.lstat()
    if (not stat.S_ISDIR(meta.st_mode) or meta.st_uid != os.geteuid() or
            meta.st_mode & 0o077):
        raise AcceptanceFault("UNKNOWN_ORIGINAL_DIRECTORY_UNTRUSTED")
    lab, lab_sha = original_record(original, "lab.json")
    intent, intent_sha = original_record(original, "01-create_container.intent.json")
    attempt, attempt_sha = original_record(original, "01-create_container.attempt.json")
    result, result_sha = original_record(original, "result.json")
    oracle, oracle_sha = original_record(original, "01-create_container.oracle.json")
    name = grant["child_name"]
    if (lab.get("scope") != lifecycle.SCOPE or lab.get("child_name") != name or
            lab.get("outer_uuid") != grant["outer_uuid"] or
            str(lab.get("outer_created_at")) != str(grant["outer_created_at"]) or
            intent.get("name") != name or
            intent.get("tool") != "create_container" or
            intent.get("state") != "MAY_BE_DISPATCHED_ONCE" or
            (intent.get("arguments") or {}).get("username") != name or
            attempt.get("tool") != "create_container" or
            (attempt.get("request") or {}).get("username") != name or
            attempt.get("success_response") is not False or
            (attempt.get("process") or {}).get("execution") != "UNKNOWN" or
            result.get("no_automatic_retry") is not True or
            result.get("child_name") != name or
            result.get("state") not in ("UNKNOWN_OR_FAIL", "UNKNOWN")):
        raise AcceptanceFault("UNKNOWN_ORIGINAL_CASE_NOT_DISPATCHED_OR_INCONSISTENT")
    former = oracle.get("incarnation")
    if former is not None and not isinstance(former, dict):
        raise AcceptanceFault("UNKNOWN_ORIGINAL_ORACLE_INVALID")
    children = lifecycle.native_list()
    row = children.get(name + "-container")
    current = lifecycle.incarnation(row) if row is not None else None
    if row is None:
        classification = "NOT_PRESENT_AT_LATER_READ_ORIGINAL_UNCERTAIN"
    elif former is None:
        classification = "PRESENT_BUT_ORIGINAL_IDENTITY_UNVERIFIED"
    elif current != former:
        classification = "INCARNATION_CHANGED_HARD_BLOCK"
    else:
        classification = "SAME_ORIGINAL_SIDE_EFFECT_EXISTS_STILL_UNKNOWN"
    evidence = {
        "schema_version": 1, "state": classification,
        "original_outcome": "UNKNOWN_NO_REPLAY",
        "original_side_effect_replayed": False,
        "release_acceptance": False, "cleanup_authorized": False,
        "child_name": name, "outer_uuid": grant["outer_uuid"],
        "outer_created_at": grant["outer_created_at"],
        "original_incarnation": former,
        "observed_incarnation": current,
        "native_status": row.get("status") if row is not None else None,
        "observed_at": now(),
        "original_evidence_sha256": {
            "lab": lab_sha, "intent": intent_sha, "attempt": attempt_sha,
            "original_oracle": oracle_sha, "original_result": result_sha
        }
    }
    target = Path(out)
    if target.exists() or target.is_symlink():
        raise AcceptanceFault("UNKNOWN_RECONCILE_OUTPUT_EXISTS")
    private_mkdir(target)
    unique_file(target / "read-only-reconcile.json", evidence)
    return evidence


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original", required=True)
    parser.add_argument("--grant", required=True)
    parser.add_argument("--confirm", required=True)
    parser.add_argument("--out", required=True)
    args = parser.parse_args(argv)
    try:
        print(json.dumps(inspect(args.original, args.grant, args.confirm,
                                 args.out), sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, KeyError, TypeError) as exc:
        print(json.dumps({"state": "BLOCKED", "reason": str(exc),
                          "original_replay": "FORBIDDEN"}))
        return 2


if __name__ == "__main__":
    sys.exit(main())

