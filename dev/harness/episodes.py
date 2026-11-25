"""Bind selected improvements to the existing local task DAG; no dispatcher."""
import copy
import hashlib
import re
from pathlib import Path

from . import evolution, records, tasks
from .common import HarnessError, git, observe, repo_path


def check_scope(proposal, root):
    for name in proposal["assets"]:
        target = repo_path(root, name)
        if target.relative_to(root.resolve()).as_posix() != name:
            raise HarnessError("Proposal asset resolves through a different path")
        if target.exists() and not target.is_file():
            raise HarnessError("Proposal assets must name files, not directories")


def selection(row):
    selected = evolution.outstanding_selection(row)
    if selected is None:
        raise HarnessError("No outstanding selected improvement")
    event = evolution.validate_decision(selected)
    evolution.validate_proposal(event["proposal"], row["asset"])
    if event["episode_id"] != evolution.episode_id(row["id"], event["basis"]):
        raise HarnessError("Selected episode identity mismatch")
    return event


def spec(row):
    event = selection(row)
    proposal = event["proposal"]
    return {"kind": "task", "schema_version": 1,
            "work_ref": event["episode_id"], "objective": proposal["objective"],
            "improvement": {"candidate_id": row["id"], "selection": event},
            "tasks": [{"id": step, "acceptance": proposal["acceptance"][step],
                       "depends_on": [evolution.EPISODE_STEPS[i-1]] if i else [],
                       "state": "PENDING"}
                      for i, step in enumerate(evolution.EPISODE_STEPS)],
            "attempts": [], "events": []}


def identity(data, expected):
    for key in ("kind", "schema_version", "work_ref", "objective", "improvement"):
        if data.get(key) != expected[key]:
            raise HarnessError("Task history does not match selected improvement: " + key)
    rows = tasks.validate(data)
    if list(rows) != list(evolution.EPISODE_STEPS):
        raise HarnessError("Selected episode task graph changed")
    for wanted in expected["tasks"]:
        row = rows[wanted["id"]]
        if any(row[k] != wanted[k] for k in ("id", "acceptance", "depends_on")):
            raise HarnessError("Selected episode acceptance or dependency changed")
    return rows


def validate_event(event):
    fields = {"step", "to", "attempt_ref", "actor_ref", "authority_ref", "checkpoint_ref",
              "evidence_ref", "reason", "reconcile_ref", "outcome"}
    if not isinstance(event, dict) or set(event) != fields:
        raise HarnessError("Invalid episode transition fields")
    if event["step"] not in evolution.EPISODE_STEPS or event["to"] not in tasks.STATES:
        raise HarnessError("Invalid episode step or state")
    for key in fields - {"step", "to", "reconcile_ref", "outcome"}:
        evolution.bounded(event[key], "episode " + key)
    if event["reconcile_ref"] is not None:
        evolution.bounded(event["reconcile_ref"], "episode reconciliation")
    allowed = {"WORKING": (None,), "RECORDED": (None,), "BLOCKED": ("blocked", "unknown"),
               "PENDING": ("resolved",)}
    if event["outcome"] not in allowed[event["to"]]:
        raise HarnessError("Invalid episode transition outcome")
    if (event["to"] == "PENDING") != (event["reconcile_ref"] is not None):
        raise HarnessError("Only an explicit reconciliation may return a blocked step to pending")
    return event


def check_source(source, clean=True):
    if (not isinstance(source, dict) or not isinstance(source.get("commit"), str)
            or not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", source["commit"])
            or source.get("worktree") not in ("clean", "dirty")
            or (clean and source["worktree"] != "clean")):
        raise HarnessError("Episode requires an observed exact source; clean except when preserving a blocker")


def bound_implementation(candidate, event, source, selected):
    impl = candidate.get("implementation")
    if not isinstance(impl, dict):
        raise HarnessError("No bound implementation attached")
    binding = impl.get("binding")
    if (not isinstance(binding, dict) or binding.get("work_ref") != selected["episode_id"]
            or binding.get("attempt_ref") != event["attempt_ref"]
            or type(binding.get("revision")) is not int
            or not selected["reviewed_revision"] + 1 < binding["revision"] <= candidate["revision"]
            or impl.get("asset_sha256") == selected["asset_sha256"]
            or impl.get("source", {}).get("commit") != source["commit"]
            or impl.get("source", {}).get("worktree") != "clean"
            or impl.get("evidence_ref") != event["evidence_ref"]):
        raise HarnessError("Implementation must be a new exact work-bound attachment for this episode attempt")
    return impl


def apply_event(data, event, source, candidate):
    """Pure replay of the existing task DAG, with explicit caller evidence."""
    event = validate_event(event)
    check_source(source, event["to"] != "BLOCKED")
    selected = selection(candidate)
    if data["improvement"] != {"candidate_id": candidate["id"], "selection": selected}:
        raise HarnessError("Candidate selection differs from task binding")
    proposal = selected["proposal"]
    if event["authority_ref"] != proposal["authority_ref"]:
        raise HarnessError("Episode authority reference differs from selected scope")
    updated = copy.deepcopy(data)
    rows = tasks.validate(updated)
    row = rows[event["step"]]
    to = event["to"]
    attempts = updated["attempts"]
    events = updated["events"]
    if event["evidence_ref"] in {item["event"]["evidence_ref"] for item in events}:
        raise HarnessError("Episode transition needs distinct evidence reference")
    if to == "WORKING":
        if event["step"] not in tasks.ready(updated):
            raise HarnessError("Episode step is not ready; reconcile existing claim")
        if len(attempts) >= proposal["max_attempts"]:
            raise HarnessError("Episode attempt budget exhausted")
        if event["attempt_ref"] in {item["attempt_ref"] for item in attempts}:
            raise HarnessError("Attempt reference already used")
        if event["step"] == "implement":
            expected_source = data["source"]["commit"] if events else proposal["source_sha"]
        else:
            expected_source = rows["implement"].get("recorded_source")
        if source["commit"] != expected_source:
            raise HarnessError("Source changed; reconcile exact source before starting")
        if event["step"] in ("review", "benefit"):
            excluded = {item["actor_ref"] for item in attempts if item["step"] == "implement"}
            if event["step"] == "benefit":
                excluded.update(item["actor_ref"] for item in attempts if item["step"] == "review")
            if event["actor_ref"] in excluded:
                raise HarnessError("Independent review/benefit requires distinct actor reference")
        attempts.append({key: event[key] for key in ("step", "attempt_ref", "actor_ref")})
        row.update(state=to, attempt_ref=event["attempt_ref"], actor_ref=event["actor_ref"],
                   checkpoint=event["checkpoint_ref"], note=event["reason"])
        row.pop("blocker", None)
    elif to in ("RECORDED", "BLOCKED"):
        if (row["state"] != "WORKING" or row.get("attempt_ref") != event["attempt_ref"]
                or row.get("actor_ref") != event["actor_ref"]):
            raise HarnessError("Transition does not match outstanding episode claim")
        if to == "RECORDED":
            if event["step"] == "implement":
                bound_implementation(candidate, event, source, selected)
            elif source["commit"] != rows["implement"].get("recorded_source"):
                raise HarnessError("Recorded step is not bound to implemented source")
            row.update(state=to, note=event["reason"], checkpoint=event["checkpoint_ref"],
                       evidence=event["evidence_ref"], recorded_source=source["commit"])
        else:
            row.update(state=to, note=event["reason"], checkpoint=event["checkpoint_ref"],
                       evidence=event["evidence_ref"], blocker=event["outcome"])
    else:
        if row["state"] != "BLOCKED" or row.get("attempt_ref") != event["attempt_ref"]:
            raise HarnessError("Reconcile the exact blocked episode attempt")
        if row.get("blocker") == "unknown" and event["actor_ref"] == row.get("actor_ref"):
            raise HarnessError("Uncertain outcome needs separate reconciliation actor reference")
        refs = {item["event"]["evidence_ref"] for item in events}
        refs.update(item["event"]["reconcile_ref"] for item in events)
        if event["reconcile_ref"] in refs or event["reconcile_ref"] == event["evidence_ref"]:
            raise HarnessError("Reconciliation requires distinct new review evidence")
        if event["step"] != "implement" and source["commit"] != rows["implement"].get("recorded_source"):
            raise HarnessError("Reconciliation does not match implemented source")
        row.update(state=to, note=event["reason"], checkpoint=event["checkpoint_ref"],
                   evidence=event["evidence_ref"], reconcile_ref=event["reconcile_ref"])
        row.pop("blocker", None)
    events.append({"event": event, "source": source, "candidate_revision": candidate["revision"]})
    updated["source"] = source
    tasks.validate(updated)
    return updated


def read_history(directory, expected, candidate, candidate_dir):
    last = records.latest(directory)
    previous = None
    for i in range(1, last["revision"] + 1):
        data = records.read_json(directory / f"{i:08d}.json")
        rows = identity(data, expected)
        if previous is None:
            if any(row["state"] != "PENDING" for row in rows.values()) or data.get("attempts") != [] or data.get("events") != []:
                raise HarnessError("Episode must start with an unclaimed task graph")
            check_source(data.get("source"))
            if data["source"]["commit"] != expected["improvement"]["selection"]["proposal"]["source_sha"]:
                raise HarnessError("Episode initial source differs from proposal")
            if data != dict(expected, source=data["source"], revision=1):
                raise HarnessError("Episode initial task record has unexpected fields")
        else:
            events = data.get("events")
            if (not isinstance(events, list) or len(events) != len(previous["events"]) + 1
                    or events[:-1] != previous["events"]):
                raise HarnessError("Episode transition history was rewritten")
            item = events[-1]
            if not isinstance(item, dict) or set(item) != {"event", "source", "candidate_revision"}:
                raise HarnessError("Invalid episode event record")
            revision = item["candidate_revision"]
            if (type(revision) is not int or not expected["improvement"]["selection"]["reviewed_revision"] < revision <= candidate["revision"]
                    or (previous["events"] and revision < previous["events"][-1]["candidate_revision"])):
                raise HarnessError("Invalid episode candidate revision")
            snap = records.read_json(candidate_dir / f"{revision:08d}.json")
            if snap.get("id") != candidate["id"] or snap.get("revision") != revision:
                raise HarnessError("Episode learning snapshot identity mismatch")
            replay = apply_event(previous, item["event"], item["source"], snap)
            if data != dict(replay, revision=i):
                raise HarnessError("Episode state differs from immutable transition replay")
        previous = data
    if previous != last:
        raise HarnessError("Episode task history changed while reading")
    return last


def check_live_scope(root, proposal, source, candidate, event):
    check_source(source, event["to"] != "BLOCKED")
    if event["to"] == "BLOCKED":
        return  # Preserve original evidence even when the worktree is dirty.
    check_scope(proposal, root)
    base = proposal["source_sha"]
    head = source["commit"]
    if git(root, "merge-base", "--is-ancestor", base, head) is None:
        raise HarnessError("Cannot prove proposal source ancestry")
    diff = git(root, "diff", "--name-only", "-z", "--no-renames", base, head, "--")
    if diff is None or set(filter(None, diff.split("\0"))) - set(proposal["assets"]):
        raise HarnessError("Changed source is outside selected proposal scope")
    if event["step"] == "implement" and event["to"] == "RECORDED":
        impl = bound_implementation(candidate, event, source, selection(candidate))
        path = repo_path(root, candidate["asset"])
        if not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != impl["asset_sha256"]:
            raise HarnessError("Implemented asset digest differs from current source")


def canonical_history_root(root, candidate_dir):
    inbox = Path(candidate_dir).parent
    if inbox.resolve() == (root / "dev/learning").resolve():
        return root / "docs/development/BDH_V07_EPISODES"
    return inbox.parent / "episodes"


def resolution_record(candidate, candidate_dir, root, history_root, *, task_revision,
                      outcome, review_ref, actor_ref, reason):
    selected = selection(candidate)
    expected = spec(candidate)
    if history_root.resolve() != canonical_history_root(root, candidate_dir).resolve():
        raise HarnessError("Resolution requires the canonical episode history root")
    if history_root.is_symlink():
        raise HarnessError("Unsafe episode history root")
    directory = history_root / selected["episode_id"]
    task = read_history(directory, expected, candidate, candidate_dir)
    if task["revision"] != task_revision:
        raise HarnessError("Resolution task revision conflict")
    rows = tasks.validate(task)
    if any(row["state"] != "RECORDED" for row in rows.values()):
        raise HarnessError("Resolve only after all episode steps are recorded")
    used = {item["actor_ref"] for item in task["attempts"]
            if item["step"] in ("implement", "benefit")}
    if actor_ref in used:
        raise HarnessError("Terminal review must be separate from implementation and benefit witness")
    refs = {item["event"]["evidence_ref"] for item in task["events"]}
    if review_ref in refs:
        raise HarnessError("Terminal review requires distinct evidence")
    benefit = rows["benefit"]
    # Require an actual later learning observation, not merely a task note.
    matches = [item for item in candidate.get("observations", [])[selected["observation_count"]:]
               if item["evidence"] == benefit["evidence"]
               and item["attempt_ref"] == benefit["attempt_ref"]
               and item["work_ref"] != selected["episode_id"]]
    if len(matches) != 1:
        raise HarnessError("Resolution requires exact later ordinary-task learning observation")
    witness = matches[0]
    if outcome == "benefit-confirmed" and (witness["outcome"] != "helped" or "residual" in witness):
        raise HarnessError("Confirmed benefit requires helpful witness without recorded residual")
    if outcome == "reopen" and witness["outcome"] == "helped" and "residual" not in witness:
        raise HarnessError("Reopen needs actual residual or non-helpful/uncertain witness")
    raw = (directory / f"{task_revision:08d}.json").read_bytes()
    return evolution.validate_resolution({"episode_id": selected["episode_id"],
        "task_revision": task_revision, "task_sha256": hashlib.sha256(raw).hexdigest(),
        "outcome": outcome, "review_ref": review_ref, "actor_ref": actor_ref, "reason": reason,
        "benefit_evidence": benefit["evidence"], "resolved_revision": candidate["revision"]})


def verify_resolution(candidate, candidate_dir, root, record):
    expected = resolution_record(candidate, candidate_dir, root,
        canonical_history_root(root, candidate_dir), task_revision=record["task_revision"],
        outcome=record["outcome"], review_ref=record["review_ref"],
        actor_ref=record["actor_ref"], reason=record["reason"])
    if expected != record:
        raise HarnessError("Episode resolution does not match terminal task and learning evidence")
    return record


def materialize(row, root, history_root, create=False, *, candidate_dir, event=None, task_expect=None):
    expected = spec(row)
    selected = selection(row)
    proposal = selected["proposal"]
    if history_root.resolve() != canonical_history_root(root, candidate_dir).resolve():
        raise HarnessError("Use the canonical episode history root; reconcile copies through Git")
    if history_root.is_symlink() or not history_root.is_dir():
        raise HarnessError("Episode history root missing or unsafe")
    directory = history_root / selected["episode_id"]
    if directory.is_symlink():
        raise HarnessError("Unsafe episode history directory")
    if event is not None and (create or not directory.exists()):
        raise HarnessError("Reconcile or materialize the exact episode before transition")
    if event is None and task_expect is not None:
        raise HarnessError("Task revision requires an explicit transition")
    if not directory.exists() and not create:
        return {"status": "SELECTED", "episode_id": selected["episode_id"],
                "candidate_id": row["id"], "selection": selected, "ready": [],
                "next_safe_action": "Materialize exact selected task history on authorized clean source"}
    if not directory.exists():
        check_scope(proposal, root)
        source = observe(root)
        if source["worktree"] != "clean" or source["commit"] != proposal["source_sha"]:
            raise HarnessError("Materialization requires exact clean proposal source")
        data = dict(expected, source=source)
        tasks.validate(data)
        try:
            records.publish(directory, 0, data)
        except HarnessError:
            # A concurrent or lost response may have created the same
            # revision. Never overwrite; validate its actual contents.
            if not directory.is_dir() or directory.is_symlink():
                raise
    data = read_history(directory, expected, row, candidate_dir)
    if event is not None:
        if type(task_expect) is not int or data["revision"] != task_expect:
            raise HarnessError("Episode task revision conflict")
        validate_event(event)
        source = observe(root)
        check_live_scope(root, proposal, source, row, event)
        updated = apply_event(data, event, source, row)
        records.publish(directory, task_expect, updated)
        data = read_history(directory, expected, row, candidate_dir)
    rows = tasks.validate(data)
    ready = tasks.ready(data)
    if any(item["state"] == "BLOCKED" for item in rows.values()):
        status = "UNKNOWN" if any(item.get("blocker") == "unknown" for item in rows.values()) else "BLOCKED"
        next_action = "Preserve original checkpoint and reconcile exact blocked attempt before any new claim"
    elif any(item["state"] == "WORKING" for item in rows.values()):
        status = "ACTIVE"
        next_action = "Resume exact recorded claim and checkpoint; do not replay uncertain effects"
    elif all(item["state"] == "RECORDED" for item in rows.values()):
        status = "OUTCOMES_RECORDED_REVIEW_REQUIRED"
        next_action = "Independently verify benefit evidence; do not infer promotion or closure"
    else:
        status = "READY"
        next_action = proposal["next_safe_action"] if not data["events"] else "Review scope and claim next ready step"
    return {"status": status, "episode_id": selected["episode_id"],
            "candidate_id": row["id"], "selection": selected, "revision": data["revision"],
            "history": str(directory), "ready": ready, "tasks": data["tasks"],
            "attempts_used": len(data["attempts"]), "attempt_budget": proposal["max_attempts"],
            "source": data["source"], "next_safe_action": next_action,
            "note": "Local caller-recorded lifecycle only; no dispatch, remote persistence, authority or independent evidence authenticity verified."}
