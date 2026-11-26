"""Portable task envelope for all BoxWeave work; not a second task scheduler."""
import argparse
import re
import json
import os
from pathlib import Path

from . import context, evolution, feedback, learning
from .common import HarnessError, observe
from .records import latest, publish, read_json

PROFILES = {
    "guidance": ("knowledge.index", "Answer the question; do not claim operational effects"),
    "operations": ("knowledge.maintenance", "Verify target version, authority and observed poststate"),
    "diagnosis": ("knowledge.operations", "Read original evidence; never replay UNKNOWN"),
    "development": ("knowledge.workflow", "Use start/task/verify/finish and all applicable CI gates"),
    "research": ("knowledge.index", "Cite sources; distinguish findings, hypotheses and decisions"),
    "release": ("verification.acceptance", "Use existing release and owner gates"),
    "harness": ("skill.continuous-learning", "Use targeted regression, Eval and independent review"),
}
STATES = ("ACTIVE", "BLOCKED", "UNKNOWN", "HANDOFF", "ANSWERED", "RECORDED")
DISPOSITIONS = ("none", "reuse", "candidate", "deferred", "improved", "pending")
FIELDS = {"kind", "schema_version", "revision", "work_ref", "attempt_ref", "task_kind",
          "objective", "risk", "authority", "target", "state", "next_safe_action",
          "evidence", "learning", "used_assets", "original_operations", "asset_feedback", "boundary", "related_work", "learning_links", "feedback_protocol", "implementation_claim", "evolve_handoff"}


def string(value, field, limit=512):
    if not isinstance(value, str) or not value.strip() or len(value) > limit or any(ord(c) < 32 for c in value):
        raise HarnessError("Invalid bounded text: " + field)
    return value


def refs(value, field):
    if not isinstance(value, list) or len(value) > 32:
        raise HarnessError("Invalid reference list: " + field)
    for item in value:
        string(item, field)
    if len(set(value)) != len(value):
        raise HarnessError("Duplicate reference: " + field)
    return value


def validate(data):
    if (not isinstance(data, dict) or set(data) - FIELDS or data.get("kind") != "work"
            or type(data.get("schema_version")) is not int or data["schema_version"] != 1):
        raise HarnessError("Invalid portable work envelope")
    for key in ("work_ref", "attempt_ref", "objective", "authority", "next_safe_action"):
        string(data.get(key), key)
    if not isinstance(data.get("task_kind"), str) or data["task_kind"] not in PROFILES or data.get("risk") not in ("low", "standard", "high"):
        raise HarnessError("Unknown task kind or risk")
    if data.get("state") not in STATES:
        raise HarnessError("Unknown work state")
    target = data.get("target")
    if not isinstance(target, dict) or set(target) != {"kind", "ref", "version", "source_sha"}:
        raise HarnessError("Invalid target identity")
    if target["kind"] not in ("unknown", "host", "box", "repository", "service", "none"):
        raise HarnessError("Unknown target kind")
    for key in ("ref", "version"):
        if target[key] is not None:
            string(target[key], key)
    if target["source_sha"] is not None and (not isinstance(target["source_sha"], str)
            or not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", target["source_sha"])):
        raise HarnessError("Invalid target source SHA")
    refs(data.get("evidence"), "evidence")
    refs(data.get("used_assets"), "used_assets")
    refs(data.get("original_operations", []), "original_operations")
    refs(data.get("related_work", []), "related_work")
    refs(data.get("learning_links", []), "learning_links")
    items = data.get("asset_feedback", [])
    if not isinstance(items, list) or len(items) > 32:
        raise HarnessError("Invalid asset feedback")
    protocol = data.get("feedback_protocol")
    if "feedback_protocol" in data and (type(protocol) is not int or protocol != feedback.PROTOCOL):
        raise HarnessError("Unknown feedback protocol")
    seen = set()
    for item in items:
        if protocol == feedback.PROTOCOL:
            feedback.validate_item(item)
        else:
            if (not isinstance(item, dict) or set(item) != {"asset", "outcome", "evidence"}
                    or item["outcome"] not in ("helped", "misled", "missing", "uncertain")):
                raise HarnessError("Invalid asset feedback item")
            string(item["asset"], "feedback asset")
            string(item["evidence"], "feedback evidence")
        key = feedback.identity(item)
        if key in seen:
            raise HarnessError("Duplicate feedback identity")
        seen.add(key)
    if "boundary" in data and data["boundary"] not in ("failure", "correction", "decision", "handoff", "completion", "progress", "ci"):
        raise HarnessError("Unknown material boundary")
    disposition = data.get("learning")
    if disposition is not None:
        if (not isinstance(disposition, dict) or set(disposition) != {"kind", "ref"}
                or disposition.get("kind") not in DISPOSITIONS):
            raise HarnessError("Invalid learning disposition")
        string(disposition.get("ref"), "learning ref")
    claim = data.get("implementation_claim")
    if claim is not None:
        if (not isinstance(claim, dict) or set(claim) != {"candidate_revision", "implementation_ref", "validation_evidence", "handoff_ref"}
                or type(claim["candidate_revision"]) is not int or claim["candidate_revision"] < 2
                or not disposition or disposition["kind"] != "improved"):
            raise HarnessError("Invalid work implementation claim")
        for key in ("implementation_ref", "validation_evidence"):
            string(claim[key], key)
        if claim["handoff_ref"] is not None:
            string(claim["handoff_ref"], "handoff_ref")
    if data["state"] != "ACTIVE" and disposition is None:
        raise HarnessError("Boundary requires learning disposition")
    if data["state"] == "ANSWERED" and data["task_kind"] not in ("guidance", "research"):
        raise HarnessError("Only guidance/research may be answered; record actual outcome separately")
    if data["state"] == "RECORDED":
        if data["task_kind"] in ("operations", "diagnosis") and target["version"] is None:
            raise HarnessError("Recorded operational outcome requires explicit observed version/state")
        if not data["evidence"]:
            raise HarnessError("Recorded outcome requires evidence reference")
        if data["task_kind"] in ("operations", "diagnosis", "development", "release", "harness"):
            if target["kind"] in ("unknown", "none") or not target["ref"]:
                raise HarnessError("Recorded material outcome requires observed target reference")
    handoff = data.get("evolve_handoff")
    if handoff is not None:
        if (not data.get("boundary") or not isinstance(handoff, dict)
                or set(handoff) != {"status", "items"}
                or handoff["status"] not in ("PENDING", "UNRESOLVED", "NO_MATCHED_PENDING")
                or not isinstance(handoff["items"], list) or len(handoff["items"]) > 200):
            raise HarnessError("Invalid Evolve boundary snapshot")
        ids = set()
        for item in handoff["items"]:
            if (not isinstance(item, dict) or set(item) != {"id", "revision", "status", "basis", "next_safe_action"}
                    or item["status"] not in ("REVIEW_DUE", "SELECTION_RECORDED")
                    or type(item["revision"]) is not int or item["revision"] < 1):
                raise HarnessError("Invalid pending opportunity reference")
            for key in ("id", "basis", "next_safe_action"):
                string(item[key], "Evolve " + key)
            if item["id"] in ids:
                raise HarnessError("Duplicate pending opportunity")
            ids.add(item["id"])
        if (handoff["status"] == "PENDING") != bool(handoff["items"]):
            raise HarnessError("Inconsistent Evolve boundary snapshot")
    if data.get("boundary") and disposition is None:
        raise HarnessError("Material boundary requires learning disposition")
    if "revision" in data and (type(data["revision"]) is not int or data["revision"] < 1):
        raise HarnessError("Invalid work revision")
    return data


def advance(old, new, expected):
    validate(old)
    validate(new)
    if old["revision"] != expected or new.get("revision") is not None:
        raise HarnessError("Revision conflict; use fresh boundary envelope without revision")
    for key in ("work_ref", "task_kind", "objective", "risk", "authority"):
        if old[key] != new[key]:
            raise HarnessError("Changed immutable task scope: " + key)
    if not new.get("boundary") or not new.get("learning"):
        raise HarnessError("Boundary and learning disposition required")
    if old["state"] in ("ANSWERED", "RECORDED"):
        raise HarnessError("Closed work; start linked new work rather than overwrite")
    if old["state"] == "UNKNOWN" and new["state"] != "UNKNOWN":
        raise HarnessError("UNKNOWN requires independent reconciliation; retain original work")
    if old["state"] in ("BLOCKED", "HANDOFF") and new["state"] not in ("BLOCKED", "HANDOFF", "UNKNOWN"):
        if set(new["evidence"]) == set(old["evidence"]):
            raise HarnessError("Blocked work needs new reconciliation evidence")
    for key in ("evidence", "used_assets", "original_operations", "related_work"):
        if not set(old.get(key, [])).issubset(new.get(key, [])):
            raise HarnessError("Cannot remove original references: " + key)
    # Candidate links are review decisions, not original facts. Prior links
    # remain in immutable revisions when an incorrect association is corrected.
    if new.get("feedback_protocol") != feedback.PROTOCOL and not set(old.get("learning_links", [])).issubset(new.get("learning_links", [])):
        raise HarnessError("Cannot remove legacy learning links without explicit protocol migration")
    feedback.preserve(old, new)
    for key, prior in old["target"].items():
        if prior is not None and prior != "unknown" and new["target"][key] != prior:
            if key in ("version", "source_sha") and new["target"][key] is not None and new["target"]["ref"] == old["target"]["ref"] and set(new["evidence"]) - set(old["evidence"]):
                continue
            raise HarnessError("Target identity changed without new version evidence; use linked new work")
    return new


def read_history(directory):
    """Validate every work revision and transition, including imported copies."""
    last = validate(latest(directory))
    prior = None
    for number in range(1, last["revision"] + 1):
        current = validate(read_json(Path(directory) / f"{number:08d}.json"))
        if current.get("revision") != number:
            raise HarnessError("Work revision differs from history position")
        if prior is None:
            if current["state"] != "ACTIVE":
                raise HarnessError("Work history must begin with an active envelope")
        else:
            advance(prior, {key: value for key, value in current.items() if key != "revision"}, number - 1)
        prior = current
    if prior != last:
        raise HarnessError("Work history changed while reading")
    return last


def summary(row):
    issues = []
    if row["target"]["kind"] == "unknown" or row["target"]["version"] is None and row["task_kind"] in ("operations", "diagnosis"):
        issues.append("Target version/identity unobserved")
    if row.get("original_operations"):
        issues.append("Read original Operations and independently verify poststate")
    if row["state"] in ("UNKNOWN", "BLOCKED", "HANDOFF"):
        issues.append("Reconcile blocked, unknown or handed-off work before effects")
    if (row.get("learning") or {}).get("kind") == "pending":
        issues.append("Learning persistence pending")
    return {"status": "REVIEW_REQUIRED" if issues else "OBSERVED",
            "record": row, "issues": issues, "next_safe_action": row["next_safe_action"],
            "note": "Advisory only; verify exact target/source/authority externally; never replay original effects."}


def plan(root, task_kind, query, paths=(), domains=(), signals=(), inbox=None):
    if task_kind not in PROFILES:
        raise HarnessError("Unknown task kind")
    entry, gate = PROFILES[task_kind]
    selected, _ = context.run(argparse.Namespace(path=list(paths), query=query,
                                                limit=6, all=False), root)
    asset = next((row for row in context.assets(root) if row["id"] == entry), None)
    if asset is None or asset["status"].lower() in ("stale", "deprecated", "rejected"):
        raise HarnessError("Required profile asset missing")
    items = [asset] + [row for row in selected["assets"] if row["id"] != entry]
    inbox = Path(inbox) if inbox else root / "dev/learning"
    selected_learning = []
    findings = []
    learning_status = "UNAVAILABLE"
    if inbox.is_dir():
        audit, _ = learning.audit(inbox, root)
        learning_status = audit["status"]
        selected_learning = evolution.select(audit["records"], task_kind=task_kind, query=query,
            paths=list(paths) if task_kind == "guidance" else [*paths, *(p for row in items for p in row["paths"])],
            domains=domains, signals=signals, review_context=audit["review_context"])
        findings = audit["findings"]
    return {"status": "PLAN_ONLY", "task_kind": task_kind,
            "learning": {"status": learning_status, "related": [{key: row.get(key) for key in ("id", "asset", "state", "next_action",
                          "match_reasons", "recorded_work_refs", "reconsider", "opportunity")} for row in selected_learning],
                         "findings": findings},
            "required": ["dev/BOOTSTRAP.md", "AGENTS.md", "dev/INDEX.md"],
            "assets": items[:7], "outcome_gate": gate,
            "control_source": observe(root),
            "feedback": {"protocol": feedback.PROTOCOL, "next": "For material work, begin an envelope; record feedback at boundaries and reconcile before closing."},
            "note": "Control checkout is not target identity. No effect, permission, or record created. Guidance may use manual GitHub-only entry."}


def brief_plan(result):
    """Project the existing advisory plan, never change selection or gates."""
    related = []
    for row in result["learning"]["related"]:
        opportunity = row["opportunity"]
        decision = opportunity["decision"]
        related.append({
            **row,
            "opportunity": {
                "status": opportunity["status"],
                "reasons": opportunity["reasons"],
                "observation_count": opportunity["observation_count"],
                "evidence_count": len(opportunity["evidence"]),
                "decision": decision["decision"] if decision else None,
            },
        })
    return {
        **result,
        "view": "BRIEF",
        "assets": [{key: row[key] for key in ("id", "paths", "status")}
                   for row in result["assets"]],
        "learning": {**result["learning"], "related": related},
        "next_action": (
            "Read applicable contracts/local AGENTS.md; use verify plan, then "
            "verify run CHECK FRESH_DIRECTORY and verify inspect for selected "
            "registered source checks."
            if result["task_kind"] in ("development", "harness") else
            "Read applicable contracts/local AGENTS.md and use the smallest safe task workflow."
        ),
        "execution_route": (
            "Local bwdev plan executed in this checkout. Asset paths are discovery, "
            "not proof of executable or connector availability. Use authorized "
            "connectors directly or the documented manual fallback; no new authority."
        ),
        "detail": (
            "Same selection and audit findings as the full plan. For evidence, "
            "review basis, decisions or asset descriptions, rerun without --brief "
            "using the same inputs and reconcile current records. No BDH "
            "implementation or selection without explicit owner authorization."
        ),
    }


def register(parser):
    sub = parser.add_subparsers(dest="action", required=True)
    p = sub.add_parser("plan")
    p.add_argument("--kind", choices=sorted(PROFILES), required=True)
    p.add_argument("--query", default="")
    p.add_argument("--path", action="append", default=[])
    p.add_argument("--domain", action="append", default=[])
    p.add_argument("--signal", action="append", default=[])
    p.add_argument("--inbox")
    p.add_argument("--brief", action="store_true", help="Compact advisory view; omit evidence detail, not warnings")
    v = sub.add_parser("check", help="Validate a portable envelope without writing")
    v.add_argument("file")
    f = sub.add_parser("feedback", help="Reconcile material feedback with the learning inbox (read-only)")
    f.add_argument("file")
    f.add_argument("--inbox")
    b = sub.add_parser("begin", help="Publish a new local envelope from sanitized JSON")
    b.add_argument("directory")
    b.add_argument("file")
    b.add_argument("--inbox")
    s = sub.add_parser("status")
    s.add_argument("directory")
    s.add_argument("--inbox")
    a = sub.add_parser("boundary", help="Record a material boundary with expected local revision")
    a.add_argument("directory")
    a.add_argument("file")
    a.add_argument("--expect", required=True, type=int)
    a.add_argument("--inbox")
    r = sub.add_parser("resume", help="Read original envelope and reconciliation requirements")
    r.add_argument("directory")
    r.add_argument("--inbox")
    e = sub.add_parser("export", help="Exclusive local portable export, not remote persistence")
    e.add_argument("directory")
    e.add_argument("file")


def run(args, root):
    if args.action == "plan":
        result = plan(root, args.kind, args.query, args.path, args.domain, args.signal, args.inbox)
        return (brief_plan(result) if getattr(args, "brief", False) else result), 0
    inbox = getattr(args, "inbox", None)
    if args.action == "feedback":
        return feedback.review(validate(read_json(args.file)), root, inbox)
    if args.action in ("check", "begin"):
        row = validate(read_json(args.file))
        if args.action == "begin":
            if row["state"] != "ACTIVE" or row.get("revision") is not None:
                raise HarnessError("Begin requires a new active envelope, not an imported history")
            if row.get("feedback_protocol") != feedback.PROTOCOL:
                raise HarnessError("New work history requires feedback protocol 1; check legacy records read-only")
            row = publish(args.directory, 0, row)
    else:
        row = read_history(args.directory)
        if args.action == "boundary":
            updated = advance(row, read_json(args.file), args.expect)
            reviewed, code = feedback.review(updated, root, inbox)
            if updated["state"] in ("RECORDED", "ANSWERED"):
                if code or reviewed["status"] != "FEEDBACK_RECORDED":
                    return {"status": "REVIEW_REQUIRED", "published": False, "feedback": reviewed,
                            "note": "Record an explicit active or handoff boundary and reconcile before closing."}, 1
            evolve = reviewed["evolve"]
            # Snapshot computed by the boundary writer. This is not a
            # caller-provided proof or a gate on ordinary task completion.
            updated["evolve_handoff"] = {key: evolve[key] for key in ("status", "items")}
            row = publish(args.directory, args.expect, validate(updated))
        elif args.action == "resume":
            result = summary(row)
            result["feedback"], _ = feedback.review(row, root, inbox)
            result["issues"].extend(result["feedback"]["issues"])
            if result["issues"]:
                result["status"] = "REVIEW_REQUIRED"
            return result, 1 if result["issues"] else 0
        elif args.action == "export":
            fd = os.open(args.file, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(fd, "w", encoding="utf-8") as stream:
                json.dump(row, stream, indent=2, sort_keys=True, allow_nan=False)
                stream.write("\n")
            return {"status": "EXPORTED_LOCAL", "file": str(args.file),
                    "note": "Not remote persistence. Save approved sanitized record via Git/PR; if unavailable mark pending."}, 0
    reviewed, _ = feedback.review(row, root, inbox)
    return {"status": "RECORDED" if args.action in ("begin", "boundary") else "OBSERVED",
            "record": row, "feedback": reviewed,
            "note": "Caller record only; local publication is not remote persistence or target proof. Reconcile original evidence before effects."}, 0
