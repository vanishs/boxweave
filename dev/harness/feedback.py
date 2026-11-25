"""Audit material asset feedback without granting authority or executing it."""
from pathlib import Path

from . import context, evolution, learning
from .common import HarnessError, repo_path

PROTOCOL = 1
FACTS = ("asset", "evidence", "outcome", "attempt_ref", "observation")
ITEM_FIELDS = {*FACTS, "decision", "candidate", "reason"}
DECISIONS = ("linked", "no-change", "not-applicable", "pending")


def validate_item(item):
    if not isinstance(item, dict) or set(item) != ITEM_FIELDS:
        raise HarnessError("Invalid feedback protocol item")
    for key in ("asset", "evidence", "attempt_ref", "observation", "reason"):
        evolution.bounded(item[key], "feedback " + key)
    if item["outcome"] not in evolution.OUTCOMES or item["decision"] not in DECISIONS:
        raise HarnessError("Invalid feedback outcome or decision")
    if item["decision"] == "linked":
        evolution.bounded(item["candidate"], "feedback candidate")
    elif item["candidate"] is not None:
        raise HarnessError("Only linked feedback may name a candidate")
    return item


def identity(item):
    return (item["asset"], item["evidence"])


def preserve(old, new):
    """An immutable fact can be reviewed again, but not disappear or change."""
    prior = old.get("asset_feedback", [])
    current = {identity(item): item for item in new.get("asset_feedback", [])}
    if old.get("feedback_protocol") == PROTOCOL and new.get("feedback_protocol") != PROTOCOL:
        raise HarnessError("Cannot downgrade feedback protocol")
    for item in prior:
        found = current.get(identity(item))
        if found is None or any(found.get(key) != item[key] for key in FACTS if key in item):
            raise HarnessError("Cannot remove or rewrite original asset feedback")


def asset_paths(root, asset, registry):
    if asset in registry:
        return registry[asset]
    path = repo_path(root, asset).relative_to(root.resolve()).as_posix()
    return [path] if path == asset else []


def matching_observation(candidate, row, item):
    expected = {key: item[key] for key in ("attempt_ref", "evidence", "observation", "outcome")}
    expected["work_ref"] = row["work_ref"]
    if expected in candidate.get("observations", []):
        return "OBSERVATION"
    # Older captures contain no attempt/outcome. Do not call them later reuse.
    if (candidate["work_ref"] == row["work_ref"]
            and candidate["evidence"] == item["evidence"]
            and candidate["observation"] == item["observation"]):
        return "INITIAL_CAPTURE"
    return None


def _review(row, root, inbox=None):
    """Read-only local reconciliation of caller records, not proof of effects."""
    issues = []
    results = []
    if row.get("feedback_protocol") != PROTOCOL:
        return {"status": "REVIEW_REQUIRED", "issues": ["Feedback protocol not declared; review legacy record explicitly"],
                "items": [], "note": "No automatic migration or closure."}, 1
    disposition = row.get("learning")
    items = row.get("asset_feedback", [])
    if row["state"] == "ACTIVE" and not row.get("boundary") and not disposition and not items:
        return {"status": "NOT_DUE", "issues": [], "items": [],
                "note": "No material boundary recorded yet."}, 0
    if not disposition:
        issues.append("Missing learning disposition")
    elif disposition["kind"] == "pending":
        issues.append("Learning persistence or review pending")
    elif disposition["kind"] == "reuse" and not any(item["outcome"] == "helped" for item in items):
        issues.append("Reuse disposition lacks observed helpful asset feedback")
    if disposition and disposition["kind"] in ("candidate", "deferred", "improved"):
        if not any(item["decision"] == "linked" and item["candidate"] == disposition["ref"] for item in items):
            issues.append("Learning disposition lacks matching linked feedback")
    registry = {asset["id"]: asset["paths"] for asset in context.assets(root)} if items else {}
    inbox = Path(inbox) if inbox else root / "dev/learning"
    candidates = learning.all_records(inbox, root) if inbox.is_dir() and not inbox.is_symlink() and items else []
    known = {candidate["id"]: candidate for candidate in candidates}
    if items and not inbox.is_dir():
        issues.append("Learning inbox unavailable; do not infer no related candidates")
    if items and inbox.is_symlink():
        issues.append("Unsafe learning inbox")
    for item in items:
        validate_item(item)
        paths = asset_paths(root, item["asset"], registry)
        suggestions = [{"id": candidate["id"], "revision": candidate["revision"], "state": candidate["state"]}
                       for candidate in candidates if candidate["asset"] in paths
                       and candidate["state"] not in ("DEPRECATED", "REJECTED")]
        result = {"asset": item["asset"], "evidence": item["evidence"], "decision": item["decision"],
                  "candidate": item["candidate"], "related": suggestions}
        problems = []
        if item["evidence"] not in row["evidence"]:
            problems.append("Feedback evidence absent from work record")
        if item["asset"] not in row["used_assets"] and item["outcome"] != "missing":
            problems.append("Feedback asset absent from used assets")
        if item["decision"] == "pending":
            problems.append("Feedback review pending")
        elif item["decision"] == "no-change" and suggestions:
            problems.append("Related candidates need explicit review: link or mark not-applicable with a reason")
        elif item["decision"] == "linked":
            candidate = known.get(item["candidate"])
            if candidate is None:
                problems.append("Linked candidate missing")
            elif candidate["state"] in ("DEPRECATED", "REJECTED"):
                problems.append("Linked candidate closed; review follow-up")
            elif candidate["asset"] not in paths:
                problems.append("Candidate asset differs from feedback asset")
            elif candidate["id"] not in row.get("learning_links", []):
                problems.append("Candidate absent from work learning links")
            else:
                match = matching_observation(candidate, row, item)
                result["match"] = match
                result["candidate_revision"] = candidate["revision"]
                if not match:
                    problems.append("Exact feedback observation missing; use learn observe with the current revision")
                elif match == "INITIAL_CAPTURE" and disposition and disposition["kind"] == "reuse":
                    problems.append("Initial capture is not evidence of later reuse")
                if disposition and disposition["ref"] == candidate["id"]:
                    if disposition["kind"] == "deferred" and candidate["state"] != "DEFERRED":
                        problems.append("Deferral not recorded in candidate")
                    if disposition["kind"] == "improved" and not candidate.get("implementation"):
                        problems.append("Improvement has no attached implementation evidence")
        result["issues"] = problems
        result["status"] = "REVIEW_REQUIRED" if problems else "RECORDED"
        results.append(result)
        issues.extend(item["asset"] + ": " + problem for problem in problems)
    return {"status": "REVIEW_REQUIRED" if issues else "FEEDBACK_RECORDED",
            "issues": issues, "items": results,
            "note": "Checks supplied local records only. No proof of omitted observations, remote persistence, independent recurrence, promotion or product acceptance."}, 1 if issues else 0

def review(row, root, inbox=None):
    """An invalid/unavailable local inbox never becomes an empty successful review."""
    try:
        return _review(row, root, inbox)
    except (HarnessError, OSError, UnicodeError, KeyError, TypeError, ValueError) as exc:
        return {"status": "REVIEW_REQUIRED", "issues": ["Feedback inputs need review: " + str(exc)],
                "items": [], "note": "Preserve the original work and reconcile local inputs."}, 1
