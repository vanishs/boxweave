"""Deterministic harness behavior cases, not an Agent productivity benchmark."""
from . import ci, context, tasks, evolution, work, workflow
from .common import HarnessError, repo_path
from .records import read_json
import argparse


def _evaluate(root):
    data = read_json(repo_path(root, "dev/evals/cases.json"))
    if not isinstance(data, dict) or data.get("schema_version") != 1 or not isinstance(data.get("cases"), list):
        raise HarnessError("Invalid evaluation case manifest")
    results = []
    seen = set()
    for case in data["cases"]:
        key = case["id"]
        if key in seen:
            raise HarnessError("Duplicate evaluation case")
        seen.add(key)
        kind = case["kind"]
        if kind == "context":
            actual, _ = context.run(argparse.Namespace(path=case["paths"], query=case["query"],
                                                       limit=8, all=False), root)
            ids = [item["id"] for item in actual["assets"]]
            ok = all(item in ids for item in case["expected_ids"])
        elif kind == "ci":
            actual, _ = ci.evaluate(case["snapshot"], case["repository"], case["sha"], ["quality"])
            ids = [row["state"] for row in actual["checks"]]
            ok = ids == case["expected_states"]
        elif kind == "learning":
            ids = [item["id"] for item in evolution.select(case["rows"], **case["selection"])]
            ok = ids == case["expected_ids"]
        elif kind == "work":
            try:
                row = work.validate(case["envelope"])
                ids = [row["state"]]
            except HarnessError:
                ids = ["INVALID"]
            ok = ids == case["expected_states"]
        elif kind == "feedback":
            actual, _ = work.feedback.review(work.validate(case["envelope"]), root)
            ids = [actual["status"]]
            ok = ids == case["expected_states"]
        elif kind == "feedback-witness":
            # Reuse the canonical immutable witness; mutate only an in-memory copy.
            row = read_json(repo_path(root, case["envelope_path"]))
            if case["variant"] == "historical-improvement":
                row["learning"] = {"kind":"improved", "ref":"BDH-L014"}
            elif case["variant"] == "exact-unlink":
                row.update(state="RECORDED", boundary="completion", learning_links=[],
                           learning={"kind":"none","ref":"no-new-asset"})
                row["asset_feedback"][0].update(decision="not-applicable",candidate=None,reason="unrelated")
            else:
                raise HarnessError("Unknown feedback witness variant")
            actual, _ = work.feedback.review(work.validate(row), root)
            ids = [actual["status"]]
            ok = ids == case["expected_states"]
        elif kind == "finish-gate":
            ids = [workflow.finish_gate(**case["observations"])]
            ok = ids == case["expected_states"]
        elif kind == "task-continuation":
            decision = tasks.continuation(case["spec"])
            ids = [decision["decision"]] + [row["id"] for row in decision["actionable"]]
            ok = ids == case["expected_states"]
        elif kind == "task":
            ids = tasks.ready(case["spec"])
            ok = ids == case["expected_ready"]
        else:
            raise HarnessError("Unknown deterministic evaluation kind")
        results.append({"id": key, "status": "PASS" if ok else "FAIL", "observed": ids})
    if not results:
        raise HarnessError("No evaluation cases")
    success = all(row["status"] == "PASS" for row in results)
    return {"status": "PASS" if success else "FAIL", "cases": results,
            "note": "Behavior fixtures only. No comparative Agent runs, cost or productivity gain measured."}, 0 if success else 1


def evaluate(root):
    try:
        return _evaluate(root)
    except (KeyError, TypeError, AttributeError, ValueError) as exc:
        raise HarnessError("Malformed deterministic evaluation manifest") from exc


def register(parser):
    pass


def run(args, root):
    return evaluate(root)
