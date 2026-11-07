"""System-first scenario graph planner. Planning never counts as acceptance."""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import re

from .engine import AcceptanceFault, impacted, load_dependencies, now

# A logical action is matched to *actual discovered callable* surface entries.
# Nodes define a real sequence with the independent post-state requirements.
GRAPHS = {
    "SYS-LIFE-01": [
        ("finder", "create_container", "stdio_mcp", "write"),
        ("observer", "get_container", "stdio_mcp", "read"),
        ("finder", "resize_container", "stdio_mcp", "write"),
        ("observer", "get_container", "stdio_mcp", "read"),
        ("finder", "stop_container", "stdio_mcp", "write"),
        ("finder", "start_container", "stdio_mcp", "write"),
        ("observer", "connect", "stdio_mcp", "read"),
        ("finder", "delete_container", "stdio_mcp", "write"),
        ("observer", "list_containers", "stdio_mcp", "read")],
    "SYS-LIFE-02": [
        ("finder", "get_container", "stdio_mcp", "read"),
        ("finder", "resize_container", "stdio_mcp", "write"),
        ("competing_actor", "stop_container", "stdio_mcp", "write"),
        ("observer", "get_container", "stdio_mcp", "read")],
    "SYS-JOB-01": [
        ("finder", "boxweave_task_submit", "stdio_mcp", "write"),
        ("competing_actor", "boxweave_task_submit", "stdio_mcp", "write"),
        ("observer", "boxweave_task_get", "stdio_mcp", "read")],
    "SYS-COORD-01": [
        ("finder", "resize_container", "stdio_mcp", "write"),
        ("competing_actor", "stop_container", "stdio_mcp", "write"),
        ("observer", "get_container", "stdio_mcp", "read")],
    "SYS-UNKNOWN-01": [
        ("finder", "boxweave_task_submit", "stdio_mcp", "write"),
        ("observer", "boxweave_task_get", "stdio_mcp", "read")],
    "SYS-SELFTEST-01": [
        ("finder", "selftest", "host_cli", "write"),
        ("competing_actor", "selftest", "host_cli", "write"),
        ("observer", "selftest", "host_cli", "read")],
    "SYS-UPGRADE-01": [
        ("finder", "version", "host_cli", "read"),
        ("finder", "status", "host_cli", "read"),
        ("observer", "doctor", "host_cli", "read")],
    "SYS-UPSTREAM-01": [
        ("observer", "version", "host_cli", "read"),
        ("observer", "get_system_info", "stdio_mcp", "read")],
    "SYS-COMPOSE-01": [
        ("finder", "compose_discover", "stdio_mcp", "read"),
        ("observer", "compose_status", "stdio_mcp", "read")],
    "SYS-SSH-01": [
        ("finder", "connect", "stdio_mcp", "read"),
        ("observer", "get_container", "stdio_mcp", "read")],
    "SYS-REST-01": [
        ("finder", "get_container", "stdio_mcp", "read"),
        ("observer", "status", "host_cli", "read")],
    "SYS-REPORT-01": [
        ("finder", "version", "host_cli", "read")],
    "SYS-RELEASE-01": [
        ("finder", "version", "host_cli", "read"),
        ("verifier", "doctor", "host_cli", "read")],
    "SYS-BACKUP-01": [
        ("finder", "get_container", "stdio_mcp", "read")],
}
IMPACT = {
    "SYS-LIFE-01": ("bwmini/box_actions.py", "bwmini/mcp_gate.py", "bwmini/provider.py"),
    "SYS-LIFE-02": ("bwmini/box_actions.py", "bwmini/operation_coordination.py"),
    "SYS-JOB-01": ("bwmini/gateway_tasks.py", "bwmini/coordination_runtime.py"),
    "SYS-COORD-01": ("bwmini/coordination_runtime.py", "bwmini/operation_leases.py"),
    "SYS-UNKNOWN-01": ("bwmini/gateway_tasks.py", "bwmini/coordination_runtime.py"),
    "SYS-UPGRADE-01": ("bwmini/distribution.py", "install.sh", "bwmini/upstream_version.py"),
    "SYS-UPSTREAM-01": ("bwmini/upstream_version.py", "bwmini/provider_rest.py"),
    "SYS-COMPOSE-01": ("bwmini/mcp_gate.py", "bwmini/provider.py"),
    "SYS-AUTH-01": ("bwmini/web.py", "bwmini/remote_auth.py"),
    "SYS-RELEASE-01": (".github/workflows/ci.yml", "bwmini/distribution.py"),
    "SYS-REPORT-01": ("tools/reacceptance/", "bwmini/reacceptance.py"),
}
DANGEROUS = {"SYS-UNKNOWN-01", "SYS-LIFE-02", "SYS-JOB-01", "SYS-COORD-01",
             "SYS-RELEASE-01", "SYS-AUTH-01"}


def seed_rows(path):
    content = Path(path).read_text(encoding="utf-8")
    rows = []
    for line in content.splitlines():
        if not re.match(r"^\| `SYS-[A-Z]+(?:-[A-Z]+)*-[0-9][0-9]`", line):
            continue
        cells = [x.strip() for x in line.strip().strip("|").split("|")]
        match = re.match(r"`(SYS-[A-Z]+(?:-[A-Z]+)*-[0-9]{2})` (P[01])", cells[0])
        if match and len(cells) >= 5:
            rows.append({"id": match.group(1), "priority": match.group(2),
                         "journey": cells[1], "invariants": cells[2],
                         "oracle": cells[3], "limits": cells[4]})
    if len(rows) < 16:
        raise AcceptanceFault("SYSTEM_SCENARIO_SEEDS_INCOMPLETE")
    return rows


def plan(inventory, seed, changed=()):
    rows = seed_rows(seed)
    actual = {x.get("key"): x for x in inventory.get("actions", [])}
    if not actual or len(actual) != len([x for x in inventory.get("actions", [])
                                        if x.get("key") in actual]):
        raise AcceptanceFault("DYNAMIC_INVENTORY_REQUIRED")
    changed = sorted(set(changed))
    ranked = []
    for row in rows:
        sid = row["id"]
        graph = GRAPHS.get(sid, ())
        nodes = []
        for i, (actor, name, channel, mode) in enumerate(graph):
            action = channel + ":" + name
            nodes.append({
                "id": sid + ":" + str(i + 1), "actor": actor, "interface": channel,
                "action_key": action, "callable_in_inventory": action in actual,
                "operation_class": mode,
                "identity": "test_owned_exact_uuid_created_at_REQUIRED",
                "depends_on": [] if i == 0 else [sid + ":" + str(i)],
                "input": "schema_bound_arguments_REQUIRED_AT_EXECUTION",
                "prerequisites": (["test_owned_exact_native_incarnation_REQUIRED"]
                                  if channel == "stdio_mcp" and name not in
                                  ("list_containers", "get_system_info") else []),
                "postcondition": "independent_provider_or_receipt_read_REQUIRED"})
        missing = sorted({n["action_key"] for n in nodes if not n["callable_in_inventory"]})
        hit = [p for p in changed if any(p.startswith(prefix)
               for prefix in IMPACT.get(sid, ()))]
        priority = 100 if row["priority"] == "P0" else 60
        risk = priority + (35 if hit else 0) + (20 if sid in DANGEROUS else 0)
        status = ("BLOCKED_UNCOMPILED_GRAPH" if not nodes else
                  "BLOCKED_RUNTIME_ACTION_MISSING" if missing else "PLANNED_NOT_RUN")
        ranked.append({**row, "risk": risk, "changed_paths": hit, "status": status,
                       "graph": {"nodes": nodes, "barriers": (
                           [{"after": nodes[0]["id"], "actors": ["finder", "competing_actor"],
                             "mode": "actual_in_flight_observed"}
                            ] if sid in ("SYS-LIFE-02", "SYS-COORD-01", "SYS-JOB-01")
                           and nodes else []),
                           "cleanup": "exact_test_owned_identity_only",
                           "unknown_policy": "freeze_identity_no_retry"},
                       "missing_installed_actions": missing,
                       "variations": ["positive", "invalid", "denied_if_meaningful",
                                      "failure_recovery", "cleanup"],
                       "execution_evidence": None})
    ranked.sort(key=lambda item: (-item["risk"], item["id"]))
    return {"schema_version": 1, "created_at": now(),
            "source_sha": inventory.get("source_sha"),
            "planning_only": True, "changed_files": changed,
            "scenario_count": len(ranked), "scenarios": ranked,
            "action_inventory_count": len(actual),
            "action_breadth_executed": 0,
            "overall": "NOT_ACCEPTED_PLAN_ONLY"}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--inventory", required=True)
    p.add_argument("--seed", default="docs/verification/REACCEPTANCE_SYSTEM_SCENARIOS.md")
    p.add_argument("--changed", action="append", default=[])
    p.add_argument("--out", required=True)
    args = p.parse_args(argv)
    try:
        value = plan(json.loads(Path(args.inventory).read_text()), args.seed, args.changed)
        out = Path(args.out)
        if out.exists():
            raise AcceptanceFault("SCENARIO_PLAN_REFUSE_OVERWRITE")
        out.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        out.write_text(json.dumps(value, sort_keys=True, indent=2) + "\n")
        print(json.dumps({"state": value["overall"], "scenarios": value["scenario_count"],
                          "output": str(out)}))
        return 0
    except (AcceptanceFault, OSError, ValueError) as exc:
        print(json.dumps({"status": "FAIL_CLOSED", "code": str(exc)}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
