"""Audit material asset feedback without granting authority or executing it."""
from pathlib import Path

from . import context, evolution, learning
from .common import HarnessError, repo_path

PROTOCOL = 1
FACTS = ("asset", "evidence", "outcome", "attempt_ref", "observation")
ITEM_FIELDS = {*FACTS, "decision", "candidate", "reason"}
DECISIONS = ("linked", "no-change", "not-applicable", "pending")


def validate_item(item):
    if not isinstance(item, dict) or not ITEM_FIELDS.issubset(item) or set(item) - ITEM_FIELDS - {"correction_ref"}:
        raise HarnessError("Invalid feedback protocol item")
    for key in ("asset", "evidence", "attempt_ref", "observation", "reason"):
        evolution.bounded(item[key], "feedback " + key)
    if "correction_ref" in item:
        evolution.bounded(item["correction_ref"], "feedback correction reference")
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
        if old.get("feedback_protocol") == PROTOCOL and item.get("decision") == "linked" and (
                found.get("decision"), found.get("candidate")) != (item.get("decision"), item.get("candidate")):
            ref = found.get("correction_ref")
            if not ref or ref not in new["evidence"] or ref in old["evidence"]:
                raise HarnessError("Changing linked feedback requires new explicit correction review evidence")


def asset_paths(root, asset, registry):
    if asset in registry:
        return registry[asset]
    path = repo_path(root, asset).relative_to(root.resolve()).as_posix()
    return [path] if path == asset and (root / path).is_file() else []


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
    registry = {asset["id"]: asset["paths"] for asset in context.assets(root)} if items or row["used_assets"] else {}
    accounted = {item["asset"] for item in items}
    for asset in row["used_assets"]:
        if asset not in accounted:
            issues.append("Material used asset lacks feedback: " + asset)
    linked = {item["candidate"] for item in items if item["decision"] == "linked"}
    if set(row.get("learning_links", [])) != linked:
        issues.append("Work learning links differ from linked feedback")
    inbox = Path(inbox) if inbox else root / "dev/learning"
    candidates = []
    findings = []
    if inbox.is_dir() and not inbox.is_symlink():
        audit, _ = learning.audit(inbox, root)
        candidates = audit["records"]
        findings = audit["findings"]
    known = {candidate["id"]: candidate for candidate in candidates}
    duplicate_ids = {key for finding in findings if finding["issue"] == "DUPLICATE"
                     for key in (finding["id"], finding["other"])}
    if not inbox.is_dir():
        issues.append("Learning inbox unavailable; do not infer no related candidates")
    if inbox.is_symlink():
        issues.append("Unsafe learning inbox")
    seen_paths = {}
    # Check the current and represented attempts, not future attempts of
    # the same work_ref (which must not invalidate an earlier snapshot).
    represented_attempts = {row["attempt_ref"], *(item["attempt_ref"] for item in items)}
    for candidate in candidates:
        for observation in candidate.get("observations", []):
            if observation["work_ref"] != row["work_ref"] or observation["attempt_ref"] not in represented_attempts:
                continue
            if not any(candidate["asset"] in asset_paths(root, item["asset"], registry)
                       and matching_observation(candidate, row, item) == "OBSERVATION"
                       and all(item[key] == observation[key] for key in ("attempt_ref", "evidence", "observation", "outcome"))
                       for item in items):
                issues.append("Recorded work observation missing from feedback: " + candidate["id"] + " / " + observation["evidence"])
    for item in items:
        validate_item(item)
        paths = asset_paths(root, item["asset"], registry)
        suggestions = [{"id": candidate["id"], "revision": candidate["revision"], "state": candidate["state"]}
                       for candidate in candidates if candidate["asset"] in paths
                       and candidate["state"] not in ("DEPRECATED", "REJECTED")]
        result = {"asset": item["asset"], "evidence": item["evidence"], "decision": item["decision"],
                  "candidate": item["candidate"], "related": suggestions}
        problems = []
        if not paths:
            problems.append("Unknown asset ID or nonexistent canonical path")
        for path in paths:
            key = (path, item["evidence"])
            previous = seen_paths.get(key)
            if previous is not None and previous != item["asset"]:
                problems.append("Duplicate feedback through asset alias: " + previous)
            seen_paths[key] = item["asset"]
        if item["evidence"] not in row["evidence"]:
            problems.append("Feedback evidence absent from work record")
        if item["asset"] not in row["used_assets"] and item["outcome"] != "missing":
            problems.append("Feedback asset absent from used assets")
        exact = [candidate["id"] for candidate in candidates
                 if candidate["asset"] in paths and matching_observation(candidate, row, item)]
        if exact and (item["decision"] != "linked" or exact != [item["candidate"]]):
            problems.append("Exact recorded observation must remain linked; reconcile conflicting history explicitly: " + ", ".join(exact))
        if item.get("correction_ref") and item["correction_ref"] not in row["evidence"]:
            problems.append("Correction review evidence absent from work record")
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
                if candidate["id"] in duplicate_ids:
                    problems.append("Linked candidate has a duplicate capture; reconcile inbox")
                if disposition and disposition["ref"] == candidate["id"]:
                    if disposition["kind"] == "deferred" and candidate["state"] != "DEFERRED":
                        problems.append("Deferral not recorded in candidate")
                    if disposition["kind"] == "improved":
                        if not candidate.get("implementation"):
                            problems.append("Improvement has no attached implementation evidence")
                        elif any(f["id"] == candidate["id"] and f["issue"] == "IMPLEMENTATION_CHANGED_OR_MISSING"
                                 for f in findings):
                            problems.append("Attached implementation drifted; revalidate before claiming improvement")
                        else:
                            impl = candidate["implementation"]
                            binding = impl.get("binding")
                            claim = row.get("implementation_claim")
                            if not isinstance(binding, dict) or not isinstance(claim, dict):
                                problems.append("Historical attachment is not a work-bound improvement; record a bound implementation claim")
                            else:
                                attempts = {other["attempt_ref"] for other in items
                                            if other["decision"] == "linked" and other["candidate"] == candidate["id"]
                                            and candidate["asset"] in asset_paths(root, other["asset"], registry)
                                            and matching_observation(candidate, row, other) == "OBSERVATION"}
                                if (binding["work_ref"] != row["work_ref"]
                                        or binding["attempt_ref"] not in attempts
                                        or binding["revision"] != claim["candidate_revision"]
                                        or impl["ref"] != claim["implementation_ref"]
                                        or impl["evidence_ref"] != claim["validation_evidence"]
                                        or impl["source"]["commit"] != row["target"]["source_sha"]):
                                    problems.append("Implementation claim does not match originating work, attempt, attachment revision, evidence and source")
                                if any(claim[key] not in row["evidence"] for key in ("implementation_ref", "validation_evidence")):
                                    problems.append("Implementation and validation references must be in work evidence")
                                if row["attempt_ref"] != binding["attempt_ref"] and (
                                        not claim.get("handoff_ref") or claim["handoff_ref"] not in row["evidence"]
                                        or claim["handoff_ref"] in (impl["ref"], impl["evidence_ref"], item["evidence"])):
                                    problems.append("Later attempt requires explicit handoff reconciliation evidence")
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
