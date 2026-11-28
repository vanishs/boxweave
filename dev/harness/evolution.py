"""Deterministic cross-task learning selection; no executable candidate content."""
import hashlib
import json
import re
from pathlib import PurePosixPath

from .common import HarnessError

KINDS = ("guidance", "operations", "diagnosis", "development", "research", "release", "harness")
IMPACTS = ("blocker", "risk", "friction", "unknown")
EFFORTS = ("small", "large", "unknown")
OUTCOMES = ("helped", "misled", "missing", "recurred", "uncertain")
OPEN = ("CANDIDATE", "STALE", "DEFERRED")
REVIEWABLE = (*OPEN, "VERIFIED", "ACTIVE")
COST_UNITS = ("manual-actions", "tool-calls", "bytes", "milliseconds", "tokens")
DECISIONS = ("preserve", "defer", "reject", "select")
EPISODE_STEPS = ("implement", "validate", "review", "deliver", "benefit")
OBSERVATION_FIELDS = {"work_ref", "attempt_ref", "evidence", "observation", "outcome"}
STOP = {"task", "work", "with", "from", "this", "that", "test", "tests", "boxweave", "harness", "review", "evidence", "agent", "using", "actual", "record"}


def bounded(value, field):
    if not isinstance(value, str) or not value.strip() or len(value) > 512 or any(ord(c) < 32 for c in value):
        raise HarnessError("Invalid learning field: " + field)
    return value


def terms(value):
    return set(re.findall(r"[a-z0-9][a-z0-9-]{3,}", value.casefold())) - STOP


def tags(value, field):
    if not isinstance(value, list) or len(value) > 16:
        raise HarnessError("Invalid learning tags: " + field)
    for tag in value:
        bounded(tag, field)
        if not re.fullmatch(r"[a-z0-9][a-z0-9._-]{0,63}", tag):
            raise HarnessError("Invalid learning tag: " + field)
    if len(set(value)) != len(value):
        raise HarnessError("Duplicate learning tag: " + field)
    return value


def validate_residual(value):
    """Explicit caller-recorded friction; missing historical values stay unknown."""
    if not isinstance(value, dict) or set(value) != {"signature", "impact", "environment", "cost", "uncertainty"}:
        raise HarnessError("Invalid residual evidence fields")
    bounded(value["signature"], "residual signature")
    if value["impact"] not in IMPACTS:
        raise HarnessError("Invalid residual impact")
    for key in ("environment", "uncertainty"):
        if value[key] is not None:
            bounded(value[key], "residual " + key)
    cost = value["cost"]
    if cost is not None:
        if (not isinstance(cost, dict) or set(cost) != {"value", "unit", "provenance"}
                or type(cost["value"]) is not int or not 0 <= cost["value"] <= 10**12
                or cost["unit"] not in COST_UNITS):
            raise HarnessError("Invalid residual cost")
        bounded(cost["provenance"], "residual cost provenance")
    return value


def validate(row):
    for key in ("domains", "signals", "task_kinds"):
        tags(row.get(key, []), key)
    if any(key not in KINDS for key in row.get("task_kinds", [])):
        raise HarnessError("Unknown learning task kind")
    if row.get("impact", "unknown") not in IMPACTS or row.get("effort", "unknown") not in EFFORTS:
        raise HarnessError("Invalid impact/effort")
    observations = row.get("observations", [])
    if not isinstance(observations, list) or len(observations) > 200:
        raise HarnessError("Invalid learning observations")
    seen = {row["evidence"]}
    for item in observations:
        if not isinstance(item, dict) or not OBSERVATION_FIELDS.issubset(item) or set(item) - OBSERVATION_FIELDS - {"residual"}:
            raise HarnessError("Invalid observation fields")
        for key in ("work_ref", "attempt_ref", "evidence", "observation"):
            bounded(item[key], key)
        if item["outcome"] not in OUTCOMES:
            raise HarnessError("Invalid observation outcome")
        if item["evidence"] in seen:
            raise HarnessError("Repeated evidence reference; link original rather than count twice")
        seen.add(item["evidence"])
        if "residual" in item:
            validate_residual(item["residual"])
    deferred = row.get("reconsider")
    if deferred is not None:
        if not isinstance(deferred, dict) or set(deferred) != {"reason", "when", "observation_count"}:
            raise HarnessError("Invalid reconsideration condition")
        bounded(deferred["reason"], "deferral reason")
        bounded(deferred["when"], "reconsideration")
        count = deferred["observation_count"]
        if type(count) is not int or not 0 <= count <= len(observations):
            raise HarnessError("Invalid reconsideration observation count")
    reviews = row.get("opportunity_reviews", [])
    if not isinstance(reviews, list) or len(reviews) > 200:
        raise HarnessError("Invalid opportunity reviews")
    for event in reviews:
        validate_decision(event)
    resolutions = row.get("opportunity_resolutions", [])
    if not isinstance(resolutions, list) or len(resolutions) > len(reviews):
        raise HarnessError("Invalid episode resolution history")
    seen = set()
    for resolution in resolutions:
        validate_resolution(resolution)
        if resolution["episode_id"] in seen:
            raise HarnessError("Duplicate episode resolution")
        seen.add(resolution["episode_id"])
    return row


def validate_resolution(value):
    fields = {"episode_id", "task_revision", "task_sha256", "outcome", "review_ref",
              "actor_ref", "reason", "resolved_revision"}
    evidence_key = "failure_evidence" if isinstance(value, dict) and value.get("outcome") == "abandon" else "benefit_evidence"
    fields.add(evidence_key)
    if not isinstance(value, dict) or set(value) != fields:
        raise HarnessError("Invalid episode resolution fields")
    if not isinstance(value["episode_id"], str) or not re.fullmatch(r"BDH-E-[0-9a-f]{32}", value["episode_id"]):
        raise HarnessError("Invalid resolved episode ID")
    if not isinstance(value["task_sha256"], str) or not re.fullmatch(r"[0-9a-f]{64}", value["task_sha256"]):
        raise HarnessError("Invalid resolution task digest")
    for key in ("task_revision", "resolved_revision"):
        if type(value[key]) is not int or value[key] < 1:
            raise HarnessError("Invalid episode resolution revision")
    if value["outcome"] not in ("benefit-confirmed", "reopen", "abandon"):
        raise HarnessError("Invalid episode resolution outcome")
    for key in ("review_ref", "actor_ref", "reason", evidence_key):
        bounded(value[key], "episode resolution " + key)
    return value


def outstanding_selection(row):
    reviews = row.get("opportunity_reviews", [])
    if not reviews or reviews[-1]["decision"] != "select":
        return None
    selected = reviews[-1]
    if any(r["episode_id"] == selected["episode_id"]
           for r in row.get("opportunity_resolutions", [])):
        return None
    return selected


def proposal_path(path):
    bounded(path, "proposal asset")
    if (PurePosixPath(path).as_posix() != path or path.startswith("/")
            or any(part in (".", "..", ".git") for part in PurePosixPath(path).parts)):
        raise HarnessError("Proposal asset must be canonical and repository-relative")
    if not (path.startswith(("dev/", "docs/development/"))
            or re.fullmatch(r"scripts/test_bwdev[^/]*\.py", path)):
        raise HarnessError("Proposal outside bounded BDH source scope")
    return path


def validate_proposal(value, asset=None):
    fields = {"objective", "assets", "authority_ref", "risk", "source_sha",
              "baseline", "expected_outcome", "max_attempts", "acceptance", "next_safe_action"}
    if not isinstance(value, dict) or set(value) != fields:
        raise HarnessError("Invalid improvement proposal fields")
    for key in ("objective", "authority_ref", "baseline", "expected_outcome", "next_safe_action"):
        bounded(value[key], "proposal " + key)
    if value["risk"] not in ("low", "standard", "high"):
        raise HarnessError("Invalid proposal risk")
    if not isinstance(value["source_sha"], str) or not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", value["source_sha"]):
        raise HarnessError("Proposal requires exact source SHA")
    if type(value["max_attempts"]) is not int or not len(EPISODE_STEPS) <= value["max_attempts"] <= 50:
        raise HarnessError("Invalid improvement attempt budget")
    assets = value["assets"]
    if not isinstance(assets, list) or not 1 <= len(assets) <= 16:
        raise HarnessError("Proposal requires bounded affected assets")
    for path in assets:
        proposal_path(path)
    if len(set(assets)) != len(assets) or (asset is not None and asset not in assets):
        raise HarnessError("Proposal must uniquely include the candidate asset")
    acceptance = value["acceptance"]
    if not isinstance(acceptance, dict) or set(acceptance) != set(EPISODE_STEPS):
        raise HarnessError("Proposal must cover every episode step")
    for step in EPISODE_STEPS:
        bounded(acceptance[step], "acceptance " + step)
    return value


def episode_id(candidate_id, basis):
    return "BDH-E-" + hashlib.sha256((candidate_id + ":" + basis).encode()).hexdigest()[:32]


def validate_decision(event):
    fields = {"reviewed_revision", "basis", "decision", "reason", "review_ref",
              "when", "asset_sha256", "trigger_ref", "evidence", "covered_observation_count",
              "observation_count", "proposal", "episode_id"}
    if not isinstance(event, dict) or set(event) != fields:
        raise HarnessError("Invalid opportunity decision fields")
    if type(event["reviewed_revision"]) is not int or event["reviewed_revision"] < 1:
        raise HarnessError("Invalid reviewed revision")
    if not isinstance(event["basis"], str) or not re.fullmatch(r"[0-9a-f]{64}", event["basis"]):
        raise HarnessError("Invalid opportunity basis")
    if event["decision"] not in DECISIONS:
        raise HarnessError("Invalid opportunity decision")
    for key in ("reason", "review_ref", "when"):
        bounded(event[key], "decision " + key)
    if event["trigger_ref"] is not None:
        bounded(event["trigger_ref"], "decision trigger")
    digest = event["asset_sha256"]
    if digest is not None and (not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest)):
        raise HarnessError("Invalid reviewed asset digest")
    evidence = event["evidence"]
    if not isinstance(evidence, list) or len(evidence) > 201:
        raise HarnessError("Invalid decision evidence range")
    for ref in evidence:
        bounded(ref, "decision evidence")
    if len(set(evidence)) != len(evidence):
        raise HarnessError("Duplicate decision evidence")
    for key in ("covered_observation_count", "observation_count"):
        count = event[key]
        if type(count) is not int or not 0 <= count <= 200:
            raise HarnessError("Invalid reviewed observation range")
    if event["covered_observation_count"] > event["observation_count"]:
        raise HarnessError("Invalid reviewed observation range")
    if event["decision"] == "select":
        validate_proposal(event["proposal"])
        if not isinstance(event["episode_id"], str) or not re.fullmatch(r"BDH-E-[0-9a-f]{32}", event["episode_id"]):
            raise HarnessError("Invalid improvement episode identity")
    elif event["proposal"] is not None or event["episode_id"] is not None:
        raise HarnessError("Only selection may bind an improvement proposal")
    return event


def asset_drift(row, digest):
    impl = row.get("implementation")
    return ((isinstance(impl, dict) and impl.get("asset_sha256") != digest)
            or (row["state"] in ("VERIFIED", "ACTIVE") and row.get("reviewed_asset_sha256") != digest))


def decision_context(row, coverage, digest):
    return dict(coverage, asset_sha256=digest, asset_drift=asset_drift(row, digest))


def review_basis(row, result, coverage):
    if "asset_sha256" not in coverage or type(row.get("revision")) is not int:
        return None
    payload = {"id": row["id"], "revision": row["revision"], "state": row["state"],
               "asset": row["asset"], "asset_sha256": coverage["asset_sha256"],
               "review": {key: result[key] for key in
                          ("status", "reasons", "evidence", "covered_observation_count",
                           "coverage_event", "observation_count", "decision")}}
    return hashlib.sha256(json.dumps(payload, sort_keys=True, separators=(",", ":"),
                                     ensure_ascii=True).encode()).hexdigest()


def make_decision(row, coverage, *, basis, decision, reason, review_ref, when,
                  trigger_ref=None, proposal=None):
    if row["state"] not in REVIEWABLE:
        raise HarnessError("Closed candidate; use a reviewed follow-up")
    current = opportunity(row, coverage)
    if current["basis"] is None or basis != current["basis"]:
        raise HarnessError("Opportunity basis changed; review current evidence")
    previous = row.get("opportunity_reviews", [])
    if any(event["review_ref"] == review_ref for event in previous):
        raise HarnessError("Opportunity review reference already recorded")
    if trigger_ref is not None and (trigger_ref == review_ref or
            any(event["trigger_ref"] == trigger_ref for event in previous)):
        raise HarnessError("Review trigger must be distinct and new")
    if current["status"] != "REVIEW_DUE" and trigger_ref is None:
        raise HarnessError("No new review signal; explicit new trigger required")
    if outstanding_selection(row) is not None:
        raise HarnessError("Selected opportunity needs episode reconciliation before another decision")
    if decision == "select":
        validate_proposal(proposal, row["asset"])
    elif proposal is not None:
        raise HarnessError("Proposal only allowed when selecting")
    return validate_decision({"reviewed_revision": row["revision"], "basis": basis,
        "decision": decision, "reason": reason, "review_ref": review_ref, "when": when,
        "asset_sha256": coverage["asset_sha256"], "trigger_ref": trigger_ref,
        "evidence": current["evidence"], "covered_observation_count": current["covered_observation_count"],
        "observation_count": current["observation_count"], "proposal": proposal,
        "episode_id": episode_id(row["id"], basis) if decision == "select" else None})


def add_observation(row, item):
    result = dict(row, observations=[*row.get("observations", []), item])
    return validate(result)


def work_count(row):
    # Distinct caller references, not proof of independent tasks or recurrence.
    return len({row["work_ref"], *(item["work_ref"] for item in row.get("observations", []))})


def opportunity(row, coverage=None):
    """Advisory opportunity, not a task assignment or an approval.

    History coverage identifies the last attachment/lifecycle review event,
    not independent proof that all earlier observations were resolved.
    """
    observations = row.get("observations", [])
    coverage = coverage or {}
    count = coverage.get("covered_observation_count")
    if count is None and not row.get("implementation") and row["state"] in OPEN:
        count = 0
    if count is not None and (type(count) is not int or not 0 <= count <= len(observations)):
        raise HarnessError("Invalid opportunity coverage")
    condition = row.get("reconsider") if row["state"] == "DEFERRED" else None
    if condition is not None:
        count = max(count or 0, condition["observation_count"])
    # Without a validated history boundary, do not infer that an old
    # untyped negative observation is newer than an attachment/review.
    fresh = observations[count:] if count is not None else [
        item for item in observations if "residual" in item]
    reasons, evidence = [], []
    for item in fresh:
        if "residual" in item or item["outcome"] in ("misled", "missing", "recurred"):
            reason = "explicit-residual" if "residual" in item else "negative-outcome"
            if reason not in reasons:
                reasons.append(reason)
            evidence.append(item["evidence"])
    decision = coverage.get("opportunity_decision")
    decision_current = (decision is not None
                        and coverage.get("decision_event") == coverage.get("coverage_event"))
    same_asset = (decision_current and "asset_sha256" in coverage
                  and decision["asset_sha256"] == coverage["asset_sha256"])
    if (row["state"] == "STALE" or coverage.get("asset_drift")) and not same_asset:
        reasons.append("asset-review")
    if decision_current and not same_asset and "asset_sha256" in coverage:
        reasons.append("reviewed-asset-changed")
    reconsider_count = condition["observation_count"] if condition is not None else None
    if decision_current:
        if decision["decision"] == "defer":
            reconsider_count = count
        elif reconsider_count is not None:
            reconsider_count = max(reconsider_count, count)
    if reconsider_count is not None and len(observations) > reconsider_count:
        reasons.append("reconsideration-new-evidence")
        for item in observations[reconsider_count:]:
            if item["evidence"] not in evidence:
                evidence.append(item["evidence"])
    if row["state"] == "CANDIDATE" and not row.get("implementation") and not decision_current:
        reasons.append("unimplemented-candidate")
        if row["evidence"] not in evidence:
            evidence.insert(0, row["evidence"])
    status = ("REVIEW_DUE" if reasons else
              ("COVERAGE_UNKNOWN" if count is None else "NO_NEW_REVIEW_SIGNAL"))
    if decision_current and not reasons:
        status = "SELECTION_RECORDED" if decision["decision"] == "select" else "REVIEW_RECORDED"
    reviews = row.get("opportunity_reviews", [])
    selected = outstanding_selection(row)
    resolution = coverage.get("episode_resolution")
    if resolution is not None:
        if resolution["outcome"] in ("reopen", "abandon"):
            abandoned = resolution["outcome"] == "abandon"
            reasons.append("episode-abandoned" if abandoned else "episode-reopened")
            ref = resolution["failure_evidence" if abandoned else "benefit_evidence"]
            if ref not in evidence:
                evidence.append(ref)
        status = "REVIEW_DUE" if reasons else "EPISODE_RESOLVED"
    if selected is not None:
        status = "SELECTION_RECORDED"
    result = {"status": status, "reasons": reasons, "evidence": evidence,
              "covered_observation_count": count,
              "coverage_event": coverage.get("coverage_event"),
              "observation_count": len(observations),
              "decision": reviews[-1] if status == "SELECTION_RECORDED" else (decision if decision_current else None),
              "note": "Advisory only; evidence references are not independent proof or authority."}
    return dict(result, basis=review_basis(row, result, coverage))


def select(rows, *, task_kind=None, query="", paths=(), domains=(), signals=(), limit=8,
           include_all=False, review_context=None):
    if type(limit) is not int or not 1 <= limit <= 50:
        raise HarnessError("Learning limit must be 1..50")
    if task_kind is not None and task_kind not in KINDS:
        raise HarnessError("Unknown learning task kind")
    tags(list(domains), "domains")
    tags(list(signals), "signals")
    path_set = set(paths)
    words = terms(query)
    filtered = bool(task_kind or query.strip() or path_set or domains or signals)
    ranked = []
    for row in rows:
        validate(row)
        if row["state"] not in REVIEWABLE:
            continue
        review = opportunity(row, (review_context or {}).get(row["id"]))
        if row["state"] not in OPEN and review["status"] not in ("REVIEW_DUE", "SELECTION_RECORDED"):
            continue
        reasons = []
        if row["asset"] in path_set:
            reasons.append("asset")
        if set(domains).intersection(row.get("domains", [])):
            reasons.append("domain")
        if set(signals).intersection(row.get("signals", [])):
            reasons.append("failure-signal")
        tokens = terms(row["id"] + " " + row["observation"] + " " + row["asset"] + " " + " ".join(row.get("domains", [])))
        if words.intersection(tokens):
            reasons.append("query")
        if task_kind and task_kind in row.get("task_kinds", []) and (not row.get("domains") or reasons):
            reasons.append("task-kind")
        condition = row.get("reconsider")
        if row["state"] == "DEFERRED" and condition is not None and len(row.get("observations", [])) > condition["observation_count"]:
            # A changed observation only resurfaces when relevant or inspecting the global queue.
            if reasons or not filtered:
                reasons.append("new-observation")
        if filtered and not reasons and not include_all:
            continue
        if not reasons:
            reasons.append("global-queue" if not filtered else "explicit-all")
        if review["status"] == "SELECTION_RECORDED":
            action = "reconcile selected improvement episode; no dispatch"
        elif review["status"] == "EPISODE_RESOLVED":
            action = "retain resolved episode evidence; reconsider on new evidence"
        elif review["status"] == "REVIEW_RECORDED":
            action = "preserve recorded decision; reconsider on new evidence or explicit trigger"
        elif row["state"] == "DEFERRED":
            action = "review reconsideration condition; select or defer again"
        elif review["status"] == "REVIEW_DUE" and (row.get("implementation") or row["state"] != "CANDIDATE"):
            action = "review residual improvement opportunity"
        elif row.get("implementation"):
            action = "review linked implementation"
        else:
            action = "select and validate a bounded improvement"
        count = work_count(row)
        item = {**row, "next_action": action, "match_reasons": reasons,
                "recorded_work_refs": count, "opportunity": review}
        exact = sum(reason in ("asset", "domain", "failure-signal") for reason in reasons)
        impact = {"blocker": 0, "risk": 1, "friction": 2, "unknown": 3}[row.get("impact", "unknown")]
        effort = {"small": 0, "unknown": 1, "large": 2}[row.get("effort", "unknown")]
        ranked.append(((-exact, impact, -count, effort, row["id"]), item))
    return [item for _, item in sorted(ranked, key=lambda pair: pair[0])[:limit]]
