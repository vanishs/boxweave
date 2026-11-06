"""Scenario-first owner report; inventory-only rows can never be PASS."""
from __future__ import annotations
import argparse
import json
from pathlib import Path

from .engine import AcceptanceFault

CLOSED = {"PASS", "FAIL", "BLOCKED", "PARTIAL", "UNKNOWN", "SKIPPED", "NOT_APPLICABLE"}


def render(plan, inventory, results=None, verdict=None):
    results = results or {}
    if not isinstance(results, dict):
        raise AcceptanceFault("SCENARIO_RESULTS_INVALID")
    scenario_results = results.get("scenarios", {})
    if not isinstance(scenario_results, dict):
        raise AcceptanceFault("SCENARIO_RESULTS_INVALID")
    lines = ["# BoxWeave system-first reacceptance", "",
             "**Acceptance state: PARTIAL / NOT VERIFIED unless separately proven.**",
             "",
             "## L2/L3 scenario and invariant matrix", "",
             "| Scenario | Risk | Invariants | Outcome | Evidence |",
             "| --- | ---: | --- | --- | --- |"]
    for row in plan.get("scenarios", []):
        evidence = scenario_results.get(row["id"], {})
        if not isinstance(evidence, dict):
            raise AcceptanceFault("SCENARIO_RESULT_INVALID")
        status = evidence.get("status", row["status"])
        # A caller cannot mark a scenario PASS without independent candidate install and an oracle.
        if status == "PASS" and not (
                evidence.get("independent_verifier") and
                evidence.get("installed_candidate_sha256") and
                evidence.get("oracle_sha256") and
                evidence.get("original_attempt_sha256")):
            status = "PARTIAL_UNVERIFIED_CLAIM"
        lines.append("| " + " | ".join((row["id"], str(row["risk"]),
                      row["invariants"].replace("|", "/"), status,
                      str(evidence.get("evidence_uri", "NOT_RECORDED")))) + " |")
    lines += ["", "## Defects, suspects and blockers", ""]
    defects = results.get("defects", [])
    if not defects:
        lines.append("No confirmed defect has been established by this report.")
    for d in defects:
        lines.append("- " + str(d.get("id", "UNTRACKED")) + ": " +
                     str(d.get("status", "SUSPECT")) + " — " +
                     str(d.get("summary", "unclassified")))
    lines += ["", "## L1 complete named action appendix", "",
              "| Interface/action | Classification | Result | Missing proof/blocker |",
              "| --- | --- | --- | --- |"]
    action_results = results.get("actions", {})
    if not isinstance(action_results, dict):
        raise AcceptanceFault("ACTION_RESULTS_INVALID")
    for action in inventory.get("actions", []):
        key = action["key"]
        r = action_results.get(key, {})
        if not isinstance(r, dict):
            raise AcceptanceFault("ACTION_RESULT_INVALID")
        outcome = r.get("status", "NOT_RUN")
        if outcome == "PASS" and not r.get("oracle_sha256"):
            outcome = "PARTIAL_MISSING_ORACLE"
        lines.append("| " + key.replace("|", "/") + " | " +
                     str(action.get("class", "unclassified")) + " | " +
                     outcome + " | " + str(r.get("reason", "not tested")) + " |")
    lines += ["", "## Publication", "",
              "**BLOCKED** until an exact private candidate passes installed independent",
              "reverification and the release gate checks original attempt and digest parity.",
              ""]
    return "\n".join(lines)


def main(argv=None):
    p = argparse.ArgumentParser()
    p.add_argument("--plan", required=True)
    p.add_argument("--inventory", required=True)
    p.add_argument("--results")
    p.add_argument("--out", required=True)
    args = p.parse_args(argv)
    try:
        output = Path(args.out)
        if output.exists():
            raise AcceptanceFault("REPORT_REFUSE_OVERWRITE")
        result = render(json.loads(Path(args.plan).read_text()),
                        json.loads(Path(args.inventory).read_text()),
                        json.loads(Path(args.results).read_text()) if args.results else None)
        output.write_text(result)
        print(json.dumps({"report": str(output), "state": "PARTIAL_NOT_RELEASE_ACCEPTANCE"}))
        return 0
    except (ValueError, OSError, AcceptanceFault) as exc:
        print(json.dumps({"state": "FAIL_CLOSED", "code": str(exc)}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
