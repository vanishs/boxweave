"""Deterministic cross-task learning selection; no executable candidate content."""
import re

from .common import HarnessError

KINDS = ("guidance", "operations", "diagnosis", "development", "research", "release", "harness")
IMPACTS = ("blocker", "risk", "friction", "unknown")
EFFORTS = ("small", "large", "unknown")
OUTCOMES = ("helped", "misled", "missing", "recurred", "uncertain")
OPEN = ("CANDIDATE", "STALE", "DEFERRED")
REVIEWABLE = (*OPEN, "VERIFIED", "ACTIVE")
COST_UNITS = ("manual-actions", "tool-calls", "bytes", "milliseconds", "tokens")
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
    return row


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
    if row["state"] == "STALE" or coverage.get("asset_drift"):
        reasons.append("asset-review")
    if condition is not None and len(observations) > condition["observation_count"]:
        reasons.append("reconsideration-new-evidence")
        for item in observations[condition["observation_count"]:]:
            if item["evidence"] not in evidence:
                evidence.append(item["evidence"])
    if row["state"] == "CANDIDATE" and not row.get("implementation"):
        reasons.append("unimplemented-candidate")
        if row["evidence"] not in evidence:
            evidence.insert(0, row["evidence"])
    return {"status": "REVIEW_DUE" if reasons else
            ("COVERAGE_UNKNOWN" if count is None else "NO_NEW_REVIEW_SIGNAL"),
            "reasons": reasons, "evidence": evidence,
            "covered_observation_count": count,
            "coverage_event": coverage.get("coverage_event"),
            "observation_count": len(observations),
            "note": "Advisory only; evidence references are not independent proof or authority."}


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
        if row["state"] not in OPEN and review["status"] != "REVIEW_DUE":
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
        if row["state"] == "DEFERRED":
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
