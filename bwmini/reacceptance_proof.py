"""Optional, explicitly authorized Containarium Proof Pack adapter.

Never exposed as general-purpose MCP or arbitrary host command.  Only a pinned,
pre-reviewed unit-test declaration in an exact owned nested child is accepted.
"""
from __future__ import annotations
import argparse
import json
import os
from pathlib import Path
import re
import sys

from .reacceptance import (AcceptanceFault, capture, digest, load, now,
                           private_mkdir, unique_file, upstream_proof_review)
from .reacceptance_live import verify_lab

CONFIRM = "I_AUTHORIZE_REVIEWED_ROOT_TESTS_IN_DISPOSABLE_CHILD"


def child_identity(path):
    f = Path(path)
    if f.is_symlink() or f.stat().st_mode & 0o077 or f.stat().st_uid != os.geteuid():
        raise AcceptanceFault("PROOF_CHILD_IDENTITY_NOT_PRIVATE")
    x = load(f)
    if (x.get("scope") != "test_owned_proof_child"
            or not re.fullmatch(r"[a-z][a-z0-9-]{2,62}-container", str(x.get("name", "")))
            or not re.fullmatch(r"[a-fA-F0-9-]{36}", str(x.get("uuid", "")))
            or not str(x.get("created_at", ""))):
        raise AcceptanceFault("PROOF_CHILD_IDENTITY_INVALID")
    return x


def exact_child(name, uuid, created_at):
    r = capture(["sudo", "-n", "incus", "info", name, "--format", "json"], timeout=20)
    if r["execution"] != "completed" or r["exit_code"] != 0:
        raise AcceptanceFault("PROOF_CHILD_PROVIDER_INFO_UNAVAILABLE")
    try:
        info = json.loads(r["stdout"])
        real_uuid = info.get("config", {}).get("volatile.uuid")
        actual_created = str(info.get("created_at", ""))
        state = str(info.get("status", "")).lower()
    except (ValueError, TypeError):
        raise AcceptanceFault("PROOF_CHILD_PROVIDER_INFO_INVALID") from None
    if real_uuid != uuid or actual_created != str(created_at) or state != "running":
        raise AcceptanceFault("PROOF_CHILD_INCARNATION_CHANGED")
    return {"name": name, "uuid": uuid, "created_at": actual_created}


def test_checkout(child, declaration, commit):
    name = child["name"]
    result = capture(["sudo", "-n", "incus", "exec", name, "--", "cat",
                      "/workspace/.github/containarium.yml"], timeout=20)
    if result["execution"] != "completed" or result["exit_code"] != 0:
        raise AcceptanceFault("PROOF_DECLARATION_NOT_READABLE_IN_CHILD")
    expected = Path(declaration).read_bytes()
    if digest(expected) != digest(result["stdout"].encode()):
        raise AcceptanceFault("PROOF_DECLARATION_CHANGED_IN_CHILD")
    check = capture(["sudo", "-n", "incus", "exec", name, "--", "git",
                     "-C", "/workspace", "rev-parse", "HEAD"], timeout=20)
    if check["execution"] != "completed" or check["exit_code"] != 0:
        raise AcceptanceFault("PROOF_CHILD_COMMIT_UNAVAILABLE")
    if check["stdout"].strip() != commit:
        raise AcceptanceFault("PROOF_CHECKOUT_CHANGED")
    return {"commit": commit, "declaration_sha256": digest(expected)}


def generate(lab_path, child_path, declaration, source_commit, out, confirm):
    if confirm != CONFIRM:
        raise AcceptanceFault("PROOF_ROOT_EXECUTION_NOT_CONFIRMED")
    scope = verify_lab(lab_path)
    child = child_identity(child_path)
    if child.get("outer_uuid") != scope["outer_uuid"]:
        raise AcceptanceFault("PROOF_CHILD_BOUND_TO_DIFFERENT_OUTER")
    root = Path(out)
    if root.exists():
        raise AcceptanceFault("PROOF_OUTPUT_EXISTS")
    # Review prior to any downstream mutation, with no implicit Makefile fallback.
    review = upstream_proof_review("/workspace", child["name"], declaration)
    if not re.fullmatch(r"[a-f0-9]{40}", source_commit):
        raise AcceptanceFault("PROOF_SOURCE_COMMIT_INVALID")
    binary = Path("/usr/local/bin/containariumd")
    metadata = binary.stat()
    if not binary.is_file() or metadata.st_uid != 0 or metadata.st_mode & 0o022:
        raise AcceptanceFault("PROOF_HOST_BINARY_UNTRUSTED")
    exact = exact_child(child["name"], child["uuid"], child["created_at"])
    checkout = test_checkout(child, declaration, source_commit)
    private_mkdir(root)
    unique_file(root / "preflight.json", {
        "lab": {"outer_uuid": scope["outer_uuid"], "hostname": scope["hostname"]},
        "child": exact, "checkout": checkout, "review": review,
        "started_utc": now(), "state": "DISPATCH_PENDING"})
    result = capture(["sudo", "-n", str(binary), "proof-pack", "generate",
                      child["name"], "--dir", "/workspace", "--step-timeout", "3m"],
                     timeout=120, limit=4*1024*1024)
    unique_file(root / "generator.json", result)
    if result["execution"] != "completed" or result["exit_code"] != 0:
        raise AcceptanceFault("PROOF_ORIGINAL_RESULT_NOT_PASSED_OR_UNKNOWN_NO_RETRY")
    try:
        pack = json.loads(result["stdout"])
    except ValueError:
        raise AcceptanceFault("PROOF_OUTPUT_NOT_JSON") from None
    # Different upstream protobuf JSON renderers may use one of two spellings.
    status = str(pack.get("status", ""))
    if "PASSED" not in status:
        raise AcceptanceFault("PROOF_PACK_NOT_PASSED")
    result_status = {"state": "GENERATED_NOT_SEALED", "child": exact,
                     "source_commit": source_commit,
                     "pack_sha256": digest(result["stdout"].encode()),
                     "producer_kind": "upstream_host_generated_root_inside_owned_child",
                     "black_box_side_effects_verified": False,
                     "created_utc": now()}
    unique_file(root / "outcome.json", result_status)
    return result_status


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--lab", required=True)
    p.add_argument("--child-identity", required=True)
    p.add_argument("--declaration", required=True)
    p.add_argument("--source-commit", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--confirm", required=True)
    args = p.parse_args(argv)
    try:
        output = generate(args.lab, args.child_identity, args.declaration,
                          args.source_commit, args.out, args.confirm)
        print(json.dumps(output, sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, TypeError) as exc:
        print(json.dumps({"state": "BLOCKED_OR_UNKNOWN",
            "code": str(exc), "retry_original": "FORBIDDEN_IF_DISPATCHED"}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())

