"""Deterministic cross-task learning selection; no executable candidate content."""
import re

from .common import HarnessError

KINDS = ("guidance", "operations", "diagnosis", "development", "research", "release", "harness")
IMPACTS = ("blocker", "risk", "friction", "unknown")
EFFORTS = ("small", "large", "unknown")
OUTCOMES = ("helped", "misled", "missing", "recurred", "uncertain")
OPEN = ("CANDIDATE", "STALE", "DEFERRED")
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
        if not isinstance(item, dict) or set(item) != {"work_ref", "attempt_ref", "evidence", "observation", "outcome"}:
            raise HarnessError("Invalid observation fields")
        for key in ("work_ref", "attempt_ref", "evidence", "observation"):
            bounded(item[key], key)
        if item["outcome"] not in OUTCOMES:
            raise HarnessError("Invalid observation outcome")
        if item["evidence"] in seen:
            raise HarnessError("Repeated evidence reference; link original rather than count twice")
        seen.add(item["evidence"])
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


def select(rows, *, task_kind=None, query="", paths=(), domains=(), signals=(), limit=8, include_all=False):
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
        if row["state"] not in OPEN:
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
        if row.get("implementation"):
            action = "review linked implementation"
        elif row["state"] == "DEFERRED":
            action = "review reconsideration condition; select or defer again"
        else:
            action = "select and validate a bounded improvement"
        count = work_count(row)
        item = {**row, "next_action": action, "match_reasons": reasons,
                "recorded_work_refs": count}
        exact = sum(reason in ("asset", "domain", "failure-signal") for reason in reasons)
        impact = {"blocker": 0, "risk": 1, "friction": 2, "unknown": 3}[row.get("impact", "unknown")]
        effort = {"small": 0, "unknown": 1, "large": 2}[row.get("effort", "unknown")]
        ranked.append(((-exact, impact, -count, effort, row["id"]), item))
    return [item for _, item in sorted(ranked, key=lambda pair: pair[0])[:limit]]
