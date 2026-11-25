"""Structured learning candidates and advisory lifecycle/asset audit."""
import hashlib
import re
from pathlib import Path

from . import episodes, evolution
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
    implementation = data.get("implementation")
    if isinstance(implementation, dict) and "binding" in implementation:
        binding = implementation["binding"]
        if not isinstance(binding, dict) or set(binding) != {"work_ref", "attempt_ref", "revision"}:
            raise HarnessError("Invalid implementation binding")
        for field in ("work_ref", "attempt_ref"):
            evolution.bounded(binding[field], "implementation " + field)
        if (type(binding["revision"]) is not int or binding["revision"] < 2
                or binding["revision"] > data.get("revision", binding["revision"])):
            raise HarnessError("Invalid implementation event revision")
        source = implementation.get("source")
        if (not isinstance(source, dict) or source.get("worktree") != "clean"
                or not isinstance(source.get("commit"), str)
                or not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", source["commit"])):
            raise HarnessError("Bound implementation requires a clean observed source")
        for field in ("ref", "evidence_ref"):
            evolution.bounded(implementation.get(field), "implementation " + field)
        if not isinstance(implementation.get("asset_sha256"), str) or not re.fullmatch(r"[0-9a-f]{64}", implementation["asset_sha256"]):
            raise HarnessError("Invalid implementation asset digest")
    return evolution.validate(data)


def read_history(directory, root, review_context=None):
    """Check imported learning revisions, not just the newest snapshot."""
    last = validate(latest(directory), root)
    prior = None
    coverage = {"covered_observation_count": 0, "coverage_event": None}
    immutable = ("kind", "schema_version", "id", "work_ref", "observation",
                 "evidence", "asset", "validation", "change_class", "fingerprint", "source")
    for number in range(1, last["revision"] + 1):
        current = validate(read_json(Path(directory) / f"{number:08d}.json"), root)
        if current.get("revision") != number:
            raise HarnessError("Learning revision differs from history position")
        reviews = current.get("opportunity_reviews", [])
        prior_reviews = (prior or {}).get("opportunity_reviews", [])
        if reviews[:len(prior_reviews)] != prior_reviews or len(reviews) > len(prior_reviews) + 1:
            raise HarnessError("Cannot rewrite opportunity review history")
        if prior is None:
            if reviews:
                raise HarnessError("Initial capture cannot contain a review")
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
        if len(reviews) > len(prior_reviews):
            event = reviews[-1]
            # A decision is exactly one append-only event, not a combined
            # state/observation/attachment edit with ambiguous evidence.
            if any(current.get(k) != prior.get(k) for k in set(current) | set(prior)
                   if k not in ("revision", "opportunity_reviews")):
                raise HarnessError("Opportunity decision must be a separate revision")
            ctx = evolution.decision_context(prior, coverage, event["asset_sha256"])
            actual = evolution.make_decision(prior, ctx, basis=event["basis"],
                decision=event["decision"], reason=event["reason"], review_ref=event["review_ref"],
                when=event["when"], trigger_ref=event["trigger_ref"], proposal=event["proposal"])
            if event != actual:
                raise HarnessError("Opportunity review does not match prior revision")
            coverage = {"covered_observation_count": len(current.get("observations", [])),
                        "coverage_event": number, "decision_event": number,
                        "opportunity_decision": event}
        impl = current.get("implementation")
        previous_impl = (prior or {}).get("implementation")
        if impl != previous_impl and isinstance(impl, dict) and "binding" in impl:
            if prior is None or impl["binding"]["revision"] != number:
                raise HarnessError("Implementation binding does not identify its attachment event")
            if isinstance(previous_impl, dict):
                if previous_impl.get("asset_sha256") == impl["asset_sha256"]:
                    raise HarnessError("Unchanged asset is revalidation, not a new bound improvement")
                if previous_impl.get("evidence_ref") == impl["evidence_ref"]:
                    raise HarnessError("New bound improvement requires distinct validation evidence")
        if prior is not None and ((impl is not None and impl != previous_impl)
                or (current["state"] in ("VERIFIED", "ACTIVE") and current["state"] != prior["state"])):
            coverage = {"covered_observation_count": len(current.get("observations", [])),
                        "coverage_event": number}
        prior = current
    if prior != last:
        raise HarnessError("Learning history changed while reading")
    if review_context is not None:
        review_context[last["id"]] = coverage
    return last


def all_records(inbox, root, review_context=None):
    if not inbox.is_dir() or inbox.is_symlink():
        raise HarnessError("Learning inbox missing or unsafe")
    result = []
    for path in sorted(inbox.iterdir()):
        if path.name.startswith("."):
            continue
        safe_id(path.name)
        data = read_history(path, root, review_context)
        if data["id"] != path.name:
            raise HarnessError("Learning ID does not match directory")
        result.append(data)
    return result


def audit(inbox, root):
    review_context = {}
    rows = all_records(inbox, root, review_context)
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
    for row in rows:
        review_context[row["id"]] = evolution.decision_context(
            row, review_context[row["id"]], asset_digest(root, row["asset"]))
    return {
        "status": "REVIEW_REQUIRED" if findings else "OBSERVED",
        "records": rows, "findings": findings, "review_context": review_context,
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
    observation.add_argument("--residual-file", help="Explicit typed residual evidence JSON; omitted means unknown")
    decide = sub.add_parser("decide", help="Record an explicit opportunity review; no dispatch")
    decide.add_argument("inbox")
    decide.add_argument("id")
    decide.add_argument("--expect", type=int, required=True)
    decide.add_argument("--basis", required=True)
    decide.add_argument("--decision", choices=evolution.DECISIONS, required=True)
    for field in ("--reason", "--review-ref", "--when"):
        decide.add_argument(field, required=True)
    decide.add_argument("--trigger-ref", help="New explicit review/reconsideration evidence, if no new signal")
    decide.add_argument("--proposal", help="Bounded JSON proposal required for select")
    opportunity = sub.add_parser("opportunity", help="Inspect one reviewable candidate even without a new signal")
    opportunity.add_argument("inbox")
    opportunity.add_argument("id")
    episode = sub.add_parser("episode", help="Inspect or explicitly materialize selected episode task history")
    episode.add_argument("inbox")
    episode.add_argument("id")
    episode.add_argument("--history-root", required=True)
    episode.add_argument("--expect", type=int, required=True)
    episode.add_argument("--materialize", action="store_true")
    episode.add_argument("--event", help="Explicit JSON task transition; never dispatches work")
    episode.add_argument("--task-expect", type=int, help="Expected task revision for transition")
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
    attach.add_argument("--work-ref", help="Bind a new implementation to the originating work")
    attach.add_argument("--attempt-ref", help="Bind a new implementation to the originating attempt")
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
                signals=args.signal, limit=args.limit, include_all=args.all, review_context=data["review_context"])
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
        review_context = {}
        data = read_history(directory, root, review_context)
        if data["id"] != key or (args.action != "opportunity" and data["revision"] != args.expect):
            raise HarnessError("Learning identity/revision conflict")
        if args.action == "opportunity":
            if data["state"] not in evolution.REVIEWABLE:
                raise HarnessError("Closed candidate; use reviewed follow-up")
            ctx = evolution.decision_context(data, review_context[key], asset_digest(root, data["asset"]))
            return {"status": "OBSERVED", "id": key, "revision": data["revision"],
                    "opportunity": evolution.opportunity(data, ctx)}, 0
        if args.action == "episode":
            event = read_json(args.event) if args.event is not None else None
            return episodes.materialize(data, root, Path(args.history_root), args.materialize,
                                        candidate_dir=directory, event=event, task_expect=args.task_expect), 0
        if args.action == "decide":
            ctx = evolution.decision_context(data, review_context[key], asset_digest(root, data["asset"]))
            proposal = read_json(args.proposal) if args.proposal is not None else None
            if args.decision == "select":
                evolution.validate_proposal(proposal, data["asset"])
                episodes.check_scope(proposal, root)
                source = observe(root)
                if source["worktree"] != "clean" or source["commit"] != proposal["source_sha"]:
                    raise HarnessError("Select requires the exact clean proposal source")
            event = evolution.make_decision(data, ctx, basis=args.basis, decision=args.decision,
                reason=args.reason, review_ref=args.review_ref, when=args.when,
                trigger_ref=args.trigger_ref, proposal=proposal)
            data["opportunity_reviews"] = [*data.get("opportunity_reviews", []), event]
            result = publish(directory, args.expect, validate(data, root))
            return {"status": "RECORDED", "record": result,
                    "note": "Explicit opportunity decision only; no task assignment, effect or promotion."}, 0
        if args.action == "observe":
            if data["state"] in ("REJECTED", "DEPRECATED"):
                raise HarnessError("Closed candidate; use reviewed follow-up")
            item = {key: getattr(args, key) for key in
                    ("work_ref", "attempt_ref", "evidence", "observation", "outcome")}
            if getattr(args, "residual_file", None) is not None:
                item["residual"] = evolution.validate_residual(read_json(args.residual_file))
            data = evolution.add_observation(data, item)
            result = publish(directory, args.expect, validate(data, root))
            return {"status": "RECORDED", "record": result,
                    "note": "Observation linked; no automatic promotion or independent recurrence claim."}, 0
        if (data.get("opportunity_reviews") and data["opportunity_reviews"][-1]["decision"] == "select"
                and (args.action == "defer" or (args.action == "review" and args.to in ("REJECTED", "DEPRECATED")))):
            raise HarnessError("Outstanding selected episode needs explicit reconciliation")
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
            prior_impl = data.get("implementation")
            work_ref = getattr(args, "work_ref", None)
            attempt_ref = getattr(args, "attempt_ref", None)
            if (work_ref is None) != (attempt_ref is None):
                raise HarnessError("Implementation binding requires both work and attempt references")
            impl = {"ref": args.implementation_ref, "evidence_ref": args.validation_evidence,
                    "asset_sha256": digest, "source": observe(root)}
            if work_ref is not None:
                if isinstance(prior_impl, dict):
                    if prior_impl.get("asset_sha256") == digest:
                        raise HarnessError("Unchanged asset is revalidation, not a new bound improvement")
                    if prior_impl.get("evidence_ref") == args.validation_evidence:
                        raise HarnessError("New bound improvement requires distinct validation evidence")
                impl["binding"] = {"work_ref": work_ref, "attempt_ref": attempt_ref,
                                   "revision": args.expect + 1}
            data["implementation"] = impl
            validate(dict(data, revision=args.expect + 1), root)
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
