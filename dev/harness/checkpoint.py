"""Portable client-owned JSON checkpoints; never replay operations."""
import json
import os
import re
from datetime import datetime, timezone
from pathlib import Path

from .common import HarnessError, observe


def unique_keys(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate key")
        result[key] = value
    return result


def nonempty(value):
    return isinstance(value, str) and bool(value.strip())


def validate(data):
    if not isinstance(data, dict) or type(data.get("format_version")) is not int or data["format_version"] != 1:
        raise HarnessError("Expected checkpoint format_version 1")
    for key in ("work_ref", "attempt_ref", "objective", "stage", "next_safe_action", "updated_at_utc"):
        if not nonempty(data.get(key)):
            raise HarnessError(f"Missing checkpoint field: {key}")
    try:
        stamp = datetime.fromisoformat(data["updated_at_utc"].replace("Z", "+00:00"))
        if stamp.utcoffset() != timezone.utc.utcoffset(stamp):
            raise ValueError("not UTC")
    except ValueError as exc:
        raise HarnessError("Checkpoint timestamp must be UTC") from exc
    source = data.get("source")
    workspace = data.get("workspace")
    if not isinstance(source, dict) or not isinstance(workspace, dict):
        raise HarnessError("Missing source or workspace object")
    if not nonempty(source.get("repository")):
        raise HarnessError("Missing repository identity")
    commit = source.get("commit")
    if not isinstance(commit, str) or not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", commit):
        raise HarnessError("Expected exact source commit SHA")
    if source.get("worktree") not in ("clean", "dirty", "unknown"):
        raise HarnessError("Invalid worktree state")
    if workspace.get("kind") not in ("box", "local", "vm", "ci", "github"):
        raise HarnessError("Invalid workspace kind")
    if workspace.get("kind") == "box":
        for key in ("box_uuid", "layer", "path"):
            if not nonempty(workspace.get(key)):
                raise HarnessError(f"Missing Box workspace field: {key}")
    for key in ("verified_completed", "current_uncertainty", "original_operations", "do_not_repeat", "links"):
        if not isinstance(data.get(key), list):
            raise HarnessError(f"Expected list: {key}")
    for op in data["original_operations"]:
        if not isinstance(op, dict) or not nonempty(op.get("operation_id")):
            raise HarnessError("Original operations require operation_id")
    return data


def read_checkpoint(path):
    try:
        with Path(path).open(encoding="utf-8") as stream:
            data = json.load(stream, object_pairs_hook=unique_keys)
    except (OSError, ValueError) as exc:
        raise HarnessError("Cannot read valid checkpoint JSON") from exc
    return validate(data)


def register(parser):
    sub = parser.add_subparsers(dest="action", required=True)
    check = sub.add_parser("check", help="Validate and compare a saved checkpoint")
    check.add_argument("file")
    capture = sub.add_parser("capture", help="Create a new checkpoint (never overwrite)")
    capture.add_argument("file")
    for key in ("--work-ref", "--attempt-ref", "--objective", "--stage",
                "--repository", "--next-safe-action"):
        capture.add_argument(key, required=True)
    capture.add_argument("--kind", choices=("local", "box", "vm", "ci", "github"), default="local")
    capture.add_argument("--box-uuid")
    capture.add_argument("--layer")
    capture.add_argument("--operation-id", action="append", default=[])


def run(args, root):
    live = observe(root)
    if args.action == "capture":
        if not live["commit"] or live["worktree"] == "unknown":
            raise HarnessError("Cannot observe current Git source")
        workspace = {"kind": args.kind, "path": str(root)}
        if args.kind == "box":
            workspace.update(box_uuid=args.box_uuid, layer=args.layer, deletion_approval="not_approved")
        data = validate({
            "format_version": 1, "work_ref": args.work_ref, "attempt_ref": args.attempt_ref,
            "updated_at_utc": datetime.now(timezone.utc).isoformat(),
            "objective": args.objective, "stage": args.stage, "workspace": workspace,
            "source": {"repository": args.repository, "commit": live["commit"],
                       "worktree": live["worktree"]},
            "verified_completed": [], "current_uncertainty": ["Native/CI state not checked by bwdev"],
            "original_operations": [{"operation_id": op} for op in args.operation_id],
            "do_not_repeat": [], "next_safe_action": args.next_safe_action, "links": [],
        })
        # Exclusive creation: no overwrite, no claim of cross-writer CAS.
        try:
            fd = os.open(args.file, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(fd, "w", encoding="utf-8") as stream:
                json.dump(data, stream, indent=2, ensure_ascii=True)
                stream.write("\n")
        except OSError as exc:
            raise HarnessError("Cannot exclusively create checkpoint file") from exc
        return {"status": "CAPTURED", "file": str(args.file),
                "note": "Local file only. Save in authorized durable storage; no native/CI proof."}, 0
    data = read_checkpoint(args.file)
    issues = []
    if live["commit"] != data["source"]["commit"]:
        issues.append("source SHA differs or unavailable")
    if live["worktree"] != "clean" or data["source"]["worktree"] != "clean":
        issues.append("dirty or unknown worktree requires independent review")
    if data["original_operations"]:
        issues.append("original Operations require independent readback")
    if data["workspace"].get("path") != str(root):
        issues.append("workspace path differs or unavailable")
    if data["workspace"]["kind"] == "box":
        issues.append("exact Box identity requires independent readback")
    return {
        "status": "REVIEW_REQUIRED" if issues else "STRUCTURE_AND_GIT_MATCH",
        "work_ref": data["work_ref"], "source": live, "issues": issues,
        "next_safe_action": data["next_safe_action"],
        "note": "Never authorizes replay, takeover, or acceptance; verify repository identity, remote state and evidence separately.",
    }, 1 if issues else 0
