"""Explicit task DAG and local revisioned progress; no effect dispatcher."""
import re
from pathlib import Path

from .common import HarnessError, observe
from .records import latest, publish, read_json

STATES = {"PENDING", "WORKING", "BLOCKED", "RECORDED"}


def nonempty(value):
    return isinstance(value, str) and bool(value.strip())


def validate(data):
    if not isinstance(data, dict) or not nonempty(data.get("work_ref")) or not nonempty(data.get("objective")):
        raise HarnessError("Task record requires work_ref and objective")
    tasks = data.get("tasks")
    if not isinstance(tasks, list) or not tasks or len(tasks) > 200:
        raise HarnessError("Expected 1..200 tasks")
    rows = {}
    for row in tasks:
        if not isinstance(row, dict) or not isinstance(row.get("id"), str):
            raise HarnessError("Invalid task row")
        key = row["id"]
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}", key) or key in rows:
            raise HarnessError("Invalid or duplicate task ID")
        if not nonempty(row.get("acceptance")):
            raise HarnessError("Task requires an observable acceptance criterion")
        deps = row.get("depends_on")
        if not isinstance(deps, list) or any(not isinstance(d, str) for d in deps) or len(set(deps)) != len(deps):
            raise HarnessError("Invalid task dependencies")
        state = row.get("state")
        if not isinstance(state, str) or state not in STATES:
            raise HarnessError("Invalid task state")
        if state in ("BLOCKED", "RECORDED") and not nonempty(row.get("note")):
            raise HarnessError("Blocked or recorded task requires a note/evidence reference")
        rows[key] = row
    visited = set()
    active = set()

    def visit(key):
        if key not in rows or key in active:
            raise HarnessError("Unknown or cyclic dependency")
        if key in visited:
            return
        active.add(key)
        for dep in rows[key]["depends_on"]:
            visit(dep)
        active.remove(key)
        visited.add(key)

    for key in rows:
        visit(key)
    for row in rows.values():
        if row["state"] in ("WORKING", "RECORDED") and any(rows[d]["state"] != "RECORDED" for d in row["depends_on"]):
            raise HarnessError("Active/recorded task has an unfinished dependency")
    return rows


def ready(data):
    rows = validate(data)
    return [row["id"] for row in rows.values() if row["state"] == "PENDING"
            and all(rows[d]["state"] == "RECORDED" for d in row["depends_on"])]


def continuation(data):
    """Cheap phase-wide navigation from existing caller-recorded task history."""
    rows = validate(data)
    pending = ready(data)
    runnable = [row for row in rows.values() if row["state"] == "WORKING"]
    runnable += [rows[key] for key in pending]
    blocked = [{"id": row["id"], "note": row["note"]} for row in rows.values()
               if row["state"] == "BLOCKED"]
    waiting = [{"id": row["id"],
                "on": [key for key in row["depends_on"] if rows[key]["state"] != "RECORDED"]}
               for row in rows.values() if row["state"] == "PENDING" and row["id"] not in pending]
    return {
        "decision": "CONTINUE" if runnable else ("REVIEW_BLOCKERS" if blocked or waiting else "REVIEW_EXIT"),
        "actionable": [{key: row[key] for key in ("id", "state", "acceptance")} for row in runnable],
        "blocked": blocked,
        "waiting": waiting,
        "recorded_count": sum(row["state"] == "RECORDED" for row in rows.values()),
        "note": "Caller-recorded task graph only; review original Operations, evidence and full phase scope. No stop, acceptance or authority granted.",
    }


def register(parser):
    sub = parser.add_subparsers(dest="action", required=True)
    init = sub.add_parser("init", help="Create local task history from a JSON spec")
    init.add_argument("directory")
    init.add_argument("spec")
    replan = sub.add_parser("replan", help="Revise pending dependencies or append tasks without removing acceptance")
    replan.add_argument("directory")
    replan.add_argument("spec")
    replan.add_argument("--expect", type=int, required=True)
    replan.add_argument("--expect-work-ref", help="Require the active phase/work identity")
    replan.add_argument("--note", required=True)
    for action in ("status", "next"):
        query = sub.add_parser(action)
        query.add_argument("directory")
        query.add_argument("--expect-work-ref", help="Require the active phase/work identity")
    update = sub.add_parser("transition", help="Publish a new local revision")
    update.add_argument("directory")
    update.add_argument("task_id")
    update.add_argument("--expect", type=int, required=True)
    update.add_argument("--expect-work-ref", help="Require the active phase/work identity")
    update.add_argument("--to", choices=sorted(STATES), required=True)
    update.add_argument("--note", required=True)
    update.add_argument("--checkpoint", help="Reference to a saved checkpoint")


def run(args, root):
    directory = Path(args.directory)
    if args.action == "init":
        spec = read_json(args.spec)
        if isinstance(spec, dict) and "improvement" in spec:
            raise HarnessError("Materialize a selected episode through learn episode")
        if not isinstance(spec, dict) or not isinstance(spec.get("tasks"), list):
            raise HarnessError("Expected task specification object")
        for row in spec["tasks"]:
            if not isinstance(row, dict):
                raise HarnessError("Invalid task row")
            row["state"] = "PENDING"
        validate(spec)
        spec["kind"] = "task"
        spec["schema_version"] = 1
        spec["source"] = observe(root)
        data = publish(directory, 0, spec)
    else:
        data = latest(directory)
        if data.get("kind") != "task" or data.get("schema_version") != 1:
            raise HarnessError("Unsupported task record kind/version")
        expected_ref = getattr(args, "expect_work_ref", None)
        if expected_ref is not None and data.get("work_ref") != expected_ref:
            raise HarnessError("Task history work_ref differs from active scope; reconcile phase history")
        rows = validate(data)
        if "improvement" in data:
            raise HarnessError("Use learn episode to inspect or advance selected improvement histories")
        if args.action == "replan":
            if args.expect != data["revision"] or not nonempty(args.note):
                raise HarnessError("Revision conflict or missing replan reason")
            spec = read_json(args.spec)
            if (not isinstance(spec, dict) or spec.get("work_ref") != data["work_ref"]
                    or spec.get("objective") != data["objective"] or not isinstance(spec.get("tasks"), list)):
                raise HarnessError("Replan must preserve work identity and objective")
            proposed = []
            for row in spec["tasks"]:
                if not isinstance(row, dict) or not isinstance(row.get("id"), str):
                    raise HarnessError("Invalid proposed task")
                original = rows.get(row["id"])
                if original:
                    if row.get("acceptance") != original["acceptance"]:
                        raise HarnessError("Replan cannot change existing acceptance")
                    if original["state"] != "PENDING" and row.get("depends_on") != original["depends_on"]:
                        raise HarnessError("Replan cannot change active/recorded/blocked dependencies")
                    row = dict(original, depends_on=row.get("depends_on"))
                else:
                    row = dict(row, state="PENDING")
                proposed.append(row)
            if not set(rows).issubset({row["id"] for row in proposed}):
                raise HarnessError("Replan cannot remove existing tasks")
            data["tasks"] = proposed
            data["replan_note"] = args.note
            validate(data)
            data["source"] = observe(root)
            data = publish(directory, args.expect, data)
        if args.action == "transition":
            if args.expect != data["revision"]:
                raise HarnessError("Revision conflict; reload before editing")
            if args.task_id not in rows:
                raise HarnessError("Unknown task ID")
            row = rows[args.task_id]
            allowed = {"PENDING": {"WORKING", "BLOCKED"}, "WORKING": {"RECORDED", "BLOCKED", "PENDING"},
                       "BLOCKED": {"PENDING"}, "RECORDED": {"PENDING"}}
            if args.to not in allowed[row["state"]] or not nonempty(args.note):
                raise HarnessError("Invalid transition or empty note")
            if args.to == "WORKING" and args.task_id not in ready(data):
                raise HarnessError("Dependencies are not ready")
            row.update(state=args.to, note=args.note)
            if args.checkpoint:
                row["checkpoint"] = args.checkpoint
            validate(data)
            data["source"] = observe(root)
            data = publish(directory, args.expect, data)
    eligible = ready(data)
    return {
        "status": "OBSERVED", "revision": data["revision"], "work_ref": data["work_ref"],
        "objective": data["objective"],
        "ready": eligible, "tasks": data["tasks"] if args.action != "next" else [],
        "continuation": continuation(data),
        "source": data.get("source"), "current_source": observe(root),
        "note": "RECORDED is a task note, not verified evidence or acceptance. Local revisions only; persist and reconcile through Git/PR.",
    }, 0
