"""Structured learning candidates and advisory lifecycle/asset audit."""
import hashlib
import re
from pathlib import Path

from . import evolution
from .common import HarnessError, observe, repo_path
from .records import latest, publish, read_json

STATES = {"CANDIDATE", "DEFERRED", "VERIFIED", "ACTIVE", "STALE", "DEPRECATED", "REJECTED"}
TRANSITIONS = {
    "CANDIDATE": {"DEFERRED", "VERIFIED", "REJECTED"},
    "DEFERRED": {"CANDIDATE", "REJECTED"},
    "VERIFIED": {"ACTIVE", "STALE", "DEFERRED"},
    "ACTIVE": {"STALE", "DEPRECATED"},
    "STALE": {"CANDIDATE", "DEPRECATED"},
    "DEPRECATED": set(),
    "REJECTED": set(),
}


def nonempty(value):
    return isinstance(value, str) and bool(value.strip())


def safe_id(value):
    if not isinstance(value, str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}", value):
        raise HarnessError("Invalid learning ID")
    return value


def fingerprint(observation, asset):
    text = " ".join(observation.casefold().split()) + "\x00" + asset
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def asset_digest(root, asset):
    path = repo_path(root, asset)
    if not path.is_file():
        return None
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(65536), b""):
            digest.update(chunk)
    return digest.hexdigest()


def validate(data, root):
    if data.get("kind") != "learning" or data.get("schema_version") != 1:
        raise HarnessError("Unsupported learning record")
    safe_id(data.get("id"))
    for key in ("work_ref", "observation", "evidence", "asset", "validation"):
        if not nonempty(data.get(key)):
            raise HarnessError(f"Missing learning field: {key}")
    repo_path(root, data["asset"])
    if data.get("state") not in sorted(STATES) or data.get("change_class") not in ("L1", "L2", "L3", "L4", "L5"):
        raise HarnessError("Invalid lifecycle state or change class")
    if data.get("fingerprint") != fingerprint(data["observation"], data["asset"]):
        raise HarnessError("Learning fingerprint mismatch")
    if data["state"] in ("VERIFIED", "ACTIVE"):
        for key in ("review_ref", "validation_evidence"):
            if not nonempty(data.get(key)):
                raise HarnessError(f"Missing reviewed disposition field: {key}")
        if not isinstance(data.get("reviewed_asset_sha256"), str) or not re.fullmatch(r"[0-9a-f]{64}", data["reviewed_asset_sha256"]):
            raise HarnessError("Reviewed asset digest required")
        if data["change_class"] == "L4" and not nonempty(data.get("independent_review_ref")):
            raise HarnessError("L4 needs an independent review reference")
        if data["change_class"] == "L5" and not nonempty(data.get("owner_ref")):
            raise HarnessError("L5 needs an explicit owner authorization reference")
    return evolution.validate(data)


def read_history(directory, root):
    """Check imported learning revisions, not just the newest snapshot."""
    last = validate(latest(directory), root)
    prior = None
    immutable = ("kind", "schema_version", "id", "work_ref", "observation",
                 "evidence", "asset", "validation", "change_class", "fingerprint", "source")
    for number in range(1, last["revision"] + 1):
        current = validate(read_json(Path(directory) / f"{number:08d}.json"), root)
        if current.get("revision") != number:
            raise HarnessError("Learning revision differs from history position")
        if prior is None:
            if current["state"] != "CANDIDATE":
                raise HarnessError("Learning history must begin with a candidate")
        else:
            if any(current.get(key) != prior.get(key) for key in immutable):
                raise HarnessError("Cannot rewrite original learning capture")
            original = prior.get("observations", [])
            if current.get("observations", [])[:len(original)] != original:
                raise HarnessError("Cannot remove or rewrite original learning observations")
            if prior["state"] in ("DEPRECATED", "REJECTED"):
                raise HarnessError("Closed learning history has a further revision")
            if current["state"] != prior["state"] and current["state"] not in TRANSITIONS[prior["state"]]:
                raise HarnessError("Invalid historical learning lifecycle transition")
        prior = current
    if prior != last:
        raise HarnessError("Learning history changed while reading")
    return last


def all_records(inbox, root):
    if not inbox.is_dir() or inbox.is_symlink():
        raise HarnessError("Learning inbox missing or unsafe")
    result = []
    for path in sorted(inbox.iterdir()):
        if path.name.startswith("."):
            continue
        safe_id(path.name)
        data = read_history(path, root)
        if data["id"] != path.name:
            raise HarnessError("Learning ID does not match directory")
        result.append(data)
    return result


def audit(inbox, root):
    rows = all_records(inbox, root)
    seen = {}
    findings = []
    source = observe(root)
    for row in rows:
        key = row["fingerprint"]
        if key in seen:
            findings.append({"id": row["id"], "issue": "DUPLICATE", "other": seen[key]})
        seen[key] = row["id"]
        implementation = row.get("implementation")
        if implementation is not None:
            if (not isinstance(implementation, dict)
                    or not nonempty(implementation.get("ref"))
                    or not nonempty(implementation.get("evidence_ref"))
                    or implementation.get("asset_sha256") != asset_digest(root, row["asset"])):
                findings.append({"id": row["id"], "issue": "IMPLEMENTATION_CHANGED_OR_MISSING"})
        if row["state"] in ("VERIFIED", "ACTIVE"):
            if asset_digest(root, row["asset"]) != row["reviewed_asset_sha256"]:
                findings.append({"id": row["id"], "issue": "ASSET_CHANGED_OR_MISSING"})
            prior = row.get("reviewed_source")
            if (not isinstance(prior, dict) or not source["commit"]
                    or prior.get("commit") != source["commit"]
                    or prior.get("worktree") != "clean" or source["worktree"] != "clean"):
                findings.append({"id": row["id"], "issue": "SOURCE_REVIEW_REQUIRED"})
    return {
        "status": "REVIEW_REQUIRED" if findings else "OBSERVED",
        "records": rows, "findings": findings,
        "note": "Exact-text duplicate and asset/source drift hints only; no semantic validation, authorization, or automatic promotion.",
    }, 1 if findings else 0


def register(parser):
    sub = parser.add_subparsers(dest="action", required=True)
    capture = sub.add_parser("capture", help="Record a new candidate")
    capture.add_argument("inbox")
    capture.add_argument("id")
    for key in ("--work-ref", "--observation", "--evidence", "--asset", "--validation"):
        capture.add_argument(key, required=True)
    capture.add_argument("--change-class", choices=("L1", "L2", "L3", "L4", "L5"), required=True)
    capture.add_argument("--domain", action="append", default=[])
    capture.add_argument("--task-kind", choices=evolution.KINDS, action="append", default=[])
    capture.add_argument("--signal", action="append", default=[])
    capture.add_argument("--impact", choices=evolution.IMPACTS, default="unknown")
    capture.add_argument("--effort", choices=evolution.EFFORTS, default="unknown")
    observation = sub.add_parser("observe", help="Link another material observation without duplicating candidate")
    observation.add_argument("inbox")
    observation.add_argument("id")
    observation.add_argument("--expect", type=int, required=True)
    for field in ("--work-ref", "--attempt-ref", "--evidence", "--observation"):
        observation.add_argument(field, required=True)
    observation.add_argument("--outcome", choices=evolution.OUTCOMES, required=True)
    defer = sub.add_parser("defer", help="Record why and when to reconsider a candidate")
    defer.add_argument("inbox")
    defer.add_argument("id")
    defer.add_argument("--expect", type=int, required=True)
    defer.add_argument("--reason", required=True)
    defer.add_argument("--when", required=True)
    attach = sub.add_parser("attach", help="Link implemented improvement and tests without promoting the candidate")
    attach.add_argument("inbox")
    attach.add_argument("id")
    attach.add_argument("--expect", type=int, required=True)
    attach.add_argument("--implementation-ref", required=True)
    attach.add_argument("--validation-evidence", required=True)
    for action in ("list", "audit"):
        sub.add_parser(action).add_argument("inbox")
    nxt = sub.add_parser("next")
    nxt.add_argument("inbox")
    nxt.add_argument("--kind", choices=evolution.KINDS)
    nxt.add_argument("--query", default="")
    nxt.add_argument("--path", action="append", default=[])
    nxt.add_argument("--domain", action="append", default=[])
    nxt.add_argument("--signal", action="append", default=[])
    nxt.add_argument("--limit", type=int, default=8)
    nxt.add_argument("--all", action="store_true")
    review = sub.add_parser("review", help="Record an explicit externally reviewed disposition")
    review.add_argument("inbox")
    review.add_argument("id")
    review.add_argument("--expect", required=True, type=int)
    review.add_argument("--to", choices=sorted(STATES), required=True)
    review.add_argument("--review-ref", required=True)
    review.add_argument("--validation-evidence")
    review.add_argument("--independent-review-ref")
    review.add_argument("--owner-ref")


def run(args, root):
    inbox = Path(args.inbox)
    if args.action in ("list", "audit", "next"):
        data, code = audit(inbox, root)
        if args.action == "next":
            data["records"] = evolution.select(data["records"], task_kind=args.kind,
                query=args.query, paths=args.path, domains=args.domain,
                signals=args.signal, limit=args.limit, include_all=args.all)
            data["note"] += " Selection is advisory; recorded work refs are not independently confirmed recurrence."
        return data, code if args.action in ("audit", "next") else 0
    key = safe_id(args.id)
    directory = inbox / key
    if args.action == "capture":
        rows = all_records(inbox, root)
        asset = repo_path(root, args.asset).relative_to(root.resolve()).as_posix()
        data = {
            "kind": "learning", "schema_version": 1, "id": key, "state": "CANDIDATE",
            "work_ref": args.work_ref, "observation": args.observation, "evidence": args.evidence,
            "asset": asset, "validation": args.validation, "change_class": args.change_class,
            "fingerprint": fingerprint(args.observation, asset), "source": observe(root),
            "domains": args.domain, "task_kinds": args.task_kind, "signals": args.signal,
            "impact": args.impact, "effort": args.effort, "observations": [],
        }
        validate(data, root)
        if any(row["fingerprint"] == data["fingerprint"] for row in rows):
            raise HarnessError("Duplicate candidate; inspect existing record")
        result = publish(directory, 0, data)
    else:
        if not inbox.is_dir() or inbox.is_symlink():
            raise HarnessError("Learning inbox missing or unsafe")
        data = read_history(directory, root)
        if data["id"] != key or data["revision"] != args.expect:
            raise HarnessError("Learning identity/revision conflict")
        if args.action == "observe":
            if data["state"] in ("REJECTED", "DEPRECATED"):
                raise HarnessError("Closed candidate; use reviewed follow-up")
            data = evolution.add_observation(data, {key: getattr(args, key) for key in
                ("work_ref", "attempt_ref", "evidence", "observation", "outcome")})
            result = publish(directory, args.expect, validate(data, root))
            return {"status": "RECORDED", "record": result,
                    "note": "Observation linked; no automatic promotion or independent recurrence claim."}, 0
        if args.action == "defer":
            if data["state"] not in ("CANDIDATE", "VERIFIED"):
                raise HarnessError("Only candidate or verified may be explicitly deferred")
            data.update(state="DEFERRED", reconsider={"reason": args.reason, "when": args.when,
                "observation_count": len(data.get("observations", []))})
            result = publish(directory, args.expect, validate(data, root))
            return {"status": "RECORDED", "record": result,
                    "note": "Deferred, not closed. Relevant future task may reconsider; no effect authorized."}, 0
        if args.action == "attach":
            if data["state"] not in ("CANDIDATE", "STALE") or not nonempty(args.implementation_ref) or not nonempty(args.validation_evidence):
                raise HarnessError("Attach requires a candidate/stale record and explicit implementation and evidence references")
            digest = asset_digest(root, data["asset"])
            if digest is None:
                raise HarnessError("Implemented asset not present")
            data["implementation"] = {"ref": args.implementation_ref,
                                      "evidence_ref": args.validation_evidence,
                                      "asset_sha256": digest, "source": observe(root)}
            result = publish(directory, args.expect, data)
            return {"status": "RECORDED", "record": result,
                    "note": "Implementation linked; candidate state unchanged, independent review still required."}, 0
        if args.to not in TRANSITIONS[data["state"]] or not nonempty(args.review_ref):
            raise HarnessError("Invalid reviewed lifecycle transition")
        data.update(state=args.to, review_ref=args.review_ref)
        if args.to in ("VERIFIED", "ACTIVE"):
            if not nonempty(args.validation_evidence):
                raise HarnessError("Explicit validation evidence reference required")
            digest = asset_digest(root, data["asset"])
            if digest is None:
                raise HarnessError("Reviewed asset does not exist")
            current = observe(root)
            if args.to == "ACTIVE" and (digest != data.get("reviewed_asset_sha256")
                    or current != data.get("reviewed_source") or current["worktree"] != "clean"):
                raise HarnessError("Asset/source changed since verification; revalidate before activation")
            data.update(validation_evidence=args.validation_evidence,
                        independent_review_ref=args.independent_review_ref,
                        owner_ref=args.owner_ref, reviewed_asset_sha256=digest, reviewed_source=current)
        validate(data, root)
        result = publish(directory, args.expect, data)
    return {
        "status": "RECORDED", "record": result,
        "note": "Caller-recorded disposition only; does not validate referenced evidence or edit the canonical asset/index. Existing owner and PR review gates apply.",
    }, 0
