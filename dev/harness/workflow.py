"""Compose existing observations into an advisory development handoff."""
import argparse
import json
import os
from pathlib import Path

from . import checkpoint, ci, context, evolution, feedback, learning, profiles, tasks, verification, work
from .common import HarnessError, observe


def start(args, root):
    environment, code = profiles.observe_profile(root, args.profile)
    selected, _ = context.run(argparse.Namespace(path=args.path, query=args.query,
                                                limit=8, all=False), root)
    data = {"status": "REVIEW_REQUIRED" if code else "START_PLAN",
            "environment": environment, "context": selected,
            "verification": verification.plan(root),
            "feedback": {"protocol": feedback.PROTOCOL, "next": "Use work begin/boundary/feedback and supply --work-history at finish."},
            "note": "Read-only start unless explicit task initialization requested. No provisioning or native readiness claim."}
    inbox = Path(args.inbox) if args.inbox else root / "dev/learning"
    if inbox.is_dir():
        learned, _ = learning.audit(inbox, root)
        relevant = {p for row in selected["assets"] for p in row["paths"]}
        candidates = [row for row in learned["records"] if row["state"] in evolution.OPEN]
        related = evolution.select(learned["records"], task_kind="development", query=args.query,
            paths=sorted(relevant | set(args.path)), domains=getattr(args, "domain", []),
            signals=getattr(args, "signal", []), review_context=learned["review_context"])
        data["learning"] = {
            "pending_count": len(candidates),
            "related": [{key: row.get(key) for key in ("id", "asset", "state", "next_action",
                "match_reasons", "recorded_work_refs", "reconsider", "opportunity")} for row in related],
            "findings": learned["findings"],
        }
    else:
        data["learning"] = {"status": "UNAVAILABLE", "related": [], "note": "Learning inbox unavailable; do not infer no reusable findings."}
    if args.spec or args.history:
        if not args.spec or not args.history:
            raise HarnessError("Supply both --spec and --history for task initialization")
        if code:
            data["task"] = "NOT_INITIALIZED: environment review required"
        else:
            data["task"], _ = tasks.run(argparse.Namespace(action="init", directory=args.history,
                                                        spec=args.spec), root)
    return data, code


def finish_gate(ready, working, pending_output, operations_reviewed, issues):
    """Advisory decision from supplied observations, not remote verification."""
    if pending_output or ready or working:
        return "CONTINUE_OR_RESOLVE"
    if not operations_reviewed or issues:
        return "REVIEW_REQUIRED"
    return "HANDOFF_READY"


def summary(args, root, finishing=False):
    data, _ = tasks.run(argparse.Namespace(action="status", directory=args.history), root)
    source = observe(root)
    issues = []
    if not source["commit"] or source["worktree"] != "clean":
        issues.append("Current source is not an observed clean HEAD")
    if data["source"] != source:
        issues.append("Task revision source differs; reconcile before continuing")
    result = {"work_ref": data["work_ref"], "source": source, "task_revision": data["revision"],
              "tasks": data["tasks"], "ready": data["ready"], "issues": issues}
    if args.checkpoint:
        cp = checkpoint.read_checkpoint(args.checkpoint)
        if args.repo and cp["source"]["repository"] != args.repo:
            raise HarnessError("Checkpoint repository differs from requested repository")
        if cp["work_ref"] != data["work_ref"]:
            raise HarnessError("Checkpoint work_ref differs from task history")
        result["checkpoint"], code = checkpoint.run(
            argparse.Namespace(action="check", file=args.checkpoint), root)
        if code:
            issues.append("Checkpoint needs independent review")
    elif finishing:
        issues.append("No checkpoint supplied")
    result["local_evidence"] = []
    for path in args.evidence:
        item, code = verification.inspect(root, path)
        result["local_evidence"].append({"path": path, **item})
        if code or item["result"] != "PASS":
            issues.append("Local evidence needs review: " + path)
    if finishing and not args.evidence:
        issues.append("No local evidence supplied")
    if args.inbox:
        audit, code = learning.audit(Path(args.inbox), root)
        result["learning"] = {"records": [{"id": row["id"], "state": row["state"], "work_ref": row["work_ref"],
                                          "observed_work_refs": sorted({row["work_ref"], *(item["work_ref"] for item in row.get("observations", []))})}
                                         for row in audit["records"]], "findings": audit["findings"]}
        if code:
            issues.append("Learning audit needs review")
    work_history = getattr(args, "work_history", None)
    disposition_text = args.learning_disposition or ""
    if work_history:
        envelope = work.read_history(work_history)
        if envelope["work_ref"] != data["work_ref"]:
            raise HarnessError("Feedback work_ref differs from task history")
        if finishing:
            if envelope["objective"] != data["objective"]:
                issues.append("Feedback objective differs from task history")
            if envelope["task_kind"] not in ("development", "harness", "release"):
                issues.append("Feedback task kind differs from source delivery")
            if args.checkpoint and envelope["attempt_ref"] != cp["attempt_ref"]:
                issues.append("Feedback attempt differs from checkpoint")
            target = envelope["target"]
            if target["kind"] != "repository" or target["ref"] != args.repo or target["source_sha"] != source["commit"]:
                issues.append("Feedback target differs from exact delivery repository and source")
        result["feedback"], code = feedback.review(envelope, root, args.inbox)
        result["work_state"] = envelope["state"]
        if code or (finishing and result["feedback"]["status"] != "FEEDBACK_RECORDED"):
            issues.append("Material learning feedback needs review")
        if finishing and envelope["state"] != "RECORDED":
            issues.append("Work envelope does not record a completed outcome")
        if envelope.get("learning"):
            decision = envelope["learning"]
            disposition_text = decision["kind"] + ":" + decision["ref"]
        if args.learning_disposition or getattr(args, "related_candidate", []):
            issues.append("Use the structured work history instead of legacy learning flags")
    elif finishing:
        issues.append("No structured work feedback history supplied")
    if finishing:
        if any(row["state"] != "RECORDED" for row in data["tasks"]):
            issues.append("Task history contains unfinished work")
        disposition = args.learning_disposition or ""
        if work_history:
            pass
        elif disposition.startswith("no-new-asset:"):
            if not disposition.split(":", 1)[1].strip():
                issues.append("No-new-asset disposition requires a reason")
        elif disposition.startswith("candidate:"):
            key = disposition.split(":", 1)[1].strip()
            if not args.inbox or not any(row["id"] == key and data["work_ref"] in row["observed_work_refs"] for row in result.get("learning", {}).get("records", [])):
                issues.append("Learning candidate is not present in supplied inbox")
        else:
            issues.append("Missing learning disposition (candidate:ID or no-new-asset:REASON)")
        related = getattr(args, "related_candidate", [])
        if len(set(related)) != len(related) or any(not any(row["id"] == key and
                data["work_ref"] in row["observed_work_refs"] for row in result.get("learning", {}).get("records", []))
                for key in related):
            issues.append("Related learning candidate missing or not linked to work ref")
        result["related_candidates"] = related
        result["learning_disposition"] = disposition_text or None
        if not args.ci_snapshot:
            issues.append("No exact-head CI snapshot supplied")
        if not args.repo or not args.pr or not args.base_sha:
            issues.append("Missing explicit repository, PR and base SHA for delivery review")
    if args.ci_snapshot:
        if not args.repo or not source["commit"]:
            raise HarnessError("CI snapshot requires repository and observed SHA")
        # Default to full CI. A docs-only scope exception needs explicit policy review.
        result["ci"], code = ci.inspect(root, args.repo, source["commit"], args.ci_snapshot,
                                        pr=args.pr, base_sha=args.base_sha)
        if code:
            issues.append("Exact-head CI needs review")
    if finishing:
        pending = getattr(args, "pending_output", [])
        reviewed = getattr(args, "operations_reviewed", False)
        working = [row["id"] for row in data["tasks"] if row["state"] == "WORKING"]
        if not reviewed:
            issues.append("Original Operation/output review not reported")
        if pending:
            issues.append("Unconsumed required Operation output: " + ", ".join(pending))
        result["finish_gate"] = {
            "decision": finish_gate(data["ready"], working, pending, reviewed, issues),
            "ready": data["ready"], "working": working, "pending_output": pending,
            "operation_review": "CALLER_REPORTED" if reviewed else "NOT_REPORTED",
            "note": "Continue safe ready work or resolve original evidence; otherwise record the concrete blocker and next action. No remote proof or authority granted.",
        }
    result["status"] = "REVIEW_REQUIRED" if issues else "HANDOFF_SUMMARY"
    result["note"] = ("Advisory observations only. Caller must independently verify authority, "
                      "native state, CI jobs/base and learning disposition. Never grants merge permission.")
    lines = ["## BDH handoff", "Work ref: " + data["work_ref"],
             "Source: " + str(source["commit"]), "Task revision: " + str(data["revision"]),
             "Ready: " + (", ".join(data["ready"]) or "none"),
             "Review issues: " + ("; ".join(issues) or "none reported by local observations"),
             "Learning: " + (disposition_text or "not supplied"),
             "Structured feedback: " + result.get("feedback", {}).get("status", "NOT_SUPPLIED"),
             "Independent review and existing gates remain required."]
    result["pr_summary"] = "\n".join(lines)
    if args.export:
        # Explicit local export; caller chooses a fresh path and persists an approved copy.
        fd = os.open(args.export, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(result, stream, indent=2, sort_keys=True, allow_nan=False)
            stream.write("\n")
    return result, 1 if issues else 0


def register_start(parser):
    parser.add_argument("--profile", default="development")
    parser.add_argument("--path", action="append", default=[])
    parser.add_argument("--query", default="")
    parser.add_argument("--inbox", help="Optional explicit learning history; defaults to dev/learning")
    parser.add_argument("--domain", action="append", default=[])
    parser.add_argument("--signal", action="append", default=[])
    parser.add_argument("--spec")
    parser.add_argument("--history")


def register_summary(parser):
    parser.add_argument("history")
    parser.add_argument("--checkpoint")
    parser.add_argument("--evidence", action="append", default=[])
    parser.add_argument("--inbox")
    parser.add_argument("--repo")
    parser.add_argument("--ci-snapshot")
    parser.add_argument("--pr", type=int)
    parser.add_argument("--base-sha")
    parser.add_argument("--learning-disposition")
    parser.add_argument("--work-history", help="Structured work envelope history; required to claim feedback closure")
    parser.add_argument("--related-candidate", action="append", default=[])
    parser.add_argument("--export")


class Start:
    register = staticmethod(register_start)
    run = staticmethod(start)


class Status:
    register = staticmethod(register_summary)
    run = staticmethod(summary)


class Resume:
    register = staticmethod(register_summary)

    @staticmethod
    def run(args, root):
        if not args.checkpoint:
            raise HarnessError("Resume requires an explicit checkpoint")
        return summary(args, root)


class Finish:
    @staticmethod
    def register(parser):
        register_summary(parser)
        parser.add_argument("--operations-reviewed", action="store_true",
                            help="Report review of current-task original Operations and required outputs, including none")
        parser.add_argument("--pending-output", action="append", default=[],
                            help="Original Operation ID with required output still unconsumed; repeat as needed")

    @staticmethod
    def run(args, root):
        return summary(args, root, finishing=True)
