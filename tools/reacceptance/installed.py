"""Run bounded real installed read-only MCP and CLI black-box probes.

An inventory is never PASS. Every actual call records its bytes/digest and
the exact unsupported or missing-oracle reason. The runner deliberately has no
host-root, arbitrary shell, or mutation dispatcher.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
from .engine import (AcceptanceFault, argv_file, capture, catalogue,
                     mcp_session, now, private_mkdir, unique_file)
from .surfaces import help_tree

READ_ONLY_MCP = frozenset((
    "boxweave_info", "list_containers", "get_container", "get_metrics",
    "list_backends", "get_system_info", "list_routes", "list_network_policies",
    "get_metrics_export", "security_sentry_status", "list_runners",
    "boxweave_task_get", "boxweave_task_list",
))
READ_ONLY_HOST = frozenset(("version", "status", "provenance", "doctor"))
MAX_RECORDS = 500


def probe_call(argv, name, arguments):
    result = mcp_session(argv, [{
        "jsonrpc": "2.0", "id": "racc-mvp", "method": "tools/call",
        "params": {"name": name, "arguments": arguments},
    }], timeout=25)
    msg = next((r for r in result["messages"]
                if r.get("id") == "racc-mvp"), None)
    return {"observed_at": now(),
            "request": {"name": name, "arguments": arguments},
            "response": msg,
            "process": result["process"]}


def positive_arguments(schema, owned):
    if not isinstance(schema, dict):
        return None, "INVALID_RUNTIME_SCHEMA"
    props = schema.get("properties", {})
    required = schema.get("required", [])
    if not isinstance(props, dict) or not isinstance(required, list):
        return None, "INVALID_RUNTIME_SCHEMA"
    args = {}
    for prop in required:
        if prop == "username" and owned:
            args[prop] = owned
        elif prop == "box" and owned:
            args[prop] = owned
        else:
            return None, "REQUIRES_REVIEWED_SAFE_INPUT_" + str(prop)
    # An optional target property is still a prerequisite for a meaningful
    # positive test; an empty request otherwise produces a misleading FAIL.
    if not owned and any(k in props for k in ("username", "box", "logical")):
        return None, "VERIFIED_TEST_OWNED_TARGET_REQUIRED"
    if owned and "username" in props and "username" not in args:
        args["username"] = owned
    if owned and "box" in props and "box" not in args:
        args["box"] = owned
    if owned and "logical" in props and "logical" not in args:
        args["logical"] = owned
    return args, None


def result_kind(msg):
    if not isinstance(msg, dict):
        return "NO_JSON_RPC_RESPONSE"
    if "error" in msg or (isinstance(msg.get("result"), dict) and
                          msg["result"].get("isError")):
        return "RPC_ERROR"
    if "result" in msg:
        return "RPC_RESULT"
    return "INVALID_RPC_RESPONSE"


def inspect_mcp(argv, owned=None):
    if not argv:
        return {"catalogue": "NOT_CONFIGURED", "rows": {}, "evidence": {}}
    session = mcp_session(argv, [{
        "jsonrpc": "2.0", "id": "racc-list", "method": "tools/list",
    }], timeout=30)
    reply = next((r for r in session["messages"] if
                  r.get("id") == "racc-list"), {})
    tools = reply.get("result", {}).get("tools")
    if not isinstance(tools, list) or any(
            not isinstance(x, dict) or not isinstance(x.get("name"), str)
            for x in tools):
        return {"catalogue": "UNAVAILABLE", "rows": {},
                "evidence": {"reason": "ACTUAL_STDIO_TOOL_LIST_FAILED",
                             "session": session}}
    if len(tools) > MAX_RECORDS:
        raise AcceptanceFault("MCP_CATALOGUE_BUDGET_EXCEEDED")
    rows, evidence = {}, {}
    for t in tools:
        name = t["name"]
        key = "stdio_mcp:" + name
        if name not in READ_ONLY_MCP:
            rows[key] = {"status": "BLOCKED", "reason":
                         "MUTATING_OR_UNREVIEWED_TOOL_REQUIRES_SCENARIO_AND_GRANT"}
            continue
        schema = t.get("inputSchema") or {}
        args, blocker = positive_arguments(schema, owned)
        # The MCP schema for task_get allows either request_key or task_id
        # but may advertise no top-level "required". Never submit empty
        # arguments as though they were a valid positive workload.
        if name == "boxweave_task_get":
            blocker = "ORIGINAL_TASK_ID_OR_REQUEST_KEY_REQUIRED"
        elif (not blocker and isinstance(schema, dict) and
              (schema.get("anyOf") or schema.get("oneOf") or
               schema.get("dependentRequired"))):
            blocker = "COMPOSITE_SCHEMA_NEEDS_REVIEWED_POSITIVE_CASE"
        if blocker:
            rows[key] = {"status": "BLOCKED", "reason": blocker}
            continue
        pos = probe_call(argv, name, args)
        # Only use a string unknown property: no dangerous names/targets.
        invalid = probe_call(argv, name, {"__racc_unknown_field__": "invalid"})
        evidence[key] = {"positive": pos, "invalid": invalid}
        outcome = result_kind(pos["response"])
        rows[key] = {"status": "PARTIAL" if outcome == "RPC_RESULT" else "FAIL",
                     "reason": ("READONLY_POSITIVE_AND_INVALID_NO_INDEPENDENT_ORACLE"
                                if outcome == "RPC_RESULT" else "READONLY_POSITIVE_FAILED"),
                     "positive_kind": outcome,
                     "invalid_kind": result_kind(invalid["response"])}
    return {"catalogue": "DISCOVERED", "rows": rows, "evidence": evidence,
            "names": [x["name"] for x in tools],
            "initialize_process": session["process"]}


def inspect_cli(argv):
    if not argv:
        return {"catalogue": "NOT_CONFIGURED", "rows": {}, "evidence": {}}
    tree = help_tree(argv)
    rows, evidence = {}, {}
    for leaf, help_result in tree.items():
        if leaf == "<root>" or help_result.get("confirmed") is False:
            continue
        key = "installed_cli_leaf:" + leaf
        rows[key] = {"status": "PARTIAL", "reason": "HELP_ONLY_NO_POSITIVE"}
        evidence[key] = {"help": help_result}
    for name in sorted(READ_ONLY_HOST):
        key = "host_cli:" + name
        positive = capture(argv + [name], timeout=30)
        invalid = capture(argv + [name, "--racc-unknown-argument"], timeout=15)
        evidence[key] = {"positive": positive, "invalid": invalid}
        good = positive["execution"] == "completed" and positive["exit_code"] == 0
        text = (positive.get("stdout", "") + " " +
                positive.get("stderr", "")).upper()
        needs_admin = not good and name == "doctor" and (
            "ROOT_REQUIRED" in text or "REQUIRES_ROOT" in text)
        # Installed public CLI doctor intentionally does not inspect protected
        # app units without admin. Its JSON uses "not_verified" rather than
        # literal ROOT_REQUIRED; do not call this a defective installed doctor.
        if name == "doctor" and not good and not needs_admin:
            try:
                payload = json.loads(positive.get("stdout", ""))
                checks = payload.get("checks", [])
                needs_admin = (payload.get("state") == "installed_public_descriptor"
                               and isinstance(checks, list)
                               and any(isinstance(x, dict) and
                                       x.get("state") == "not_verified"
                                       for x in checks))
            except (TypeError, ValueError, AttributeError):
                pass
        rows[key] = {"status": ("PARTIAL" if good else
                                "BLOCKED" if needs_admin else "FAIL"),
                     "reason": ("READONLY_POSITIVE_NO_INDEPENDENT_ORACLE"
                                if good else
                                "DOCTOR_REQUIRES_EXPLICIT_ADMIN_READONLY_LANE"
                                if needs_admin else "INSTALLED_CLI_POSITIVE_FAILED"),
                     "invalid_exit_code": invalid["exit_code"]}
    return {"catalogue": "DISCOVERED", "rows": rows, "evidence": evidence}


def execute(root, out, host=None, mcp=None, owned=None):
    if owned is not None and not re.fullmatch(r"[a-z][a-z0-9-]{2,62}", owned):
        raise AcceptanceFault("UNSAFE_READ_ONLY_TARGET")
    dest = Path(out)
    if dest.exists() or dest.is_symlink():
        raise AcceptanceFault("RUN_OUTPUT_EXISTS")
    private_mkdir(dest)
    inv = catalogue(root, host_command=host, mcp_command=mcp)
    cli = inspect_cli(host)
    m = inspect_mcp(mcp, owned)
    rows = {x["key"]: {"status": "BLOCKED", "reason":
            "SOURCE_ONLY_NOT_AN_INSTALLED_CALL"} for x in inv["actions"]}
    rows.update(cli["rows"])
    rows.update(m["rows"])
    # Missing an installed transport is not equivalent to having no tools.
    if not host:
        rows["transport:installed_cli"] = {
            "status": "BLOCKED", "reason": "INSTALLED_CLI_NOT_CONFIGURED"}
    if not mcp:
        rows["transport:stdio_mcp"] = {
            "status": "BLOCKED", "reason": "INSTALLED_MCP_NOT_CONFIGURED"}
    for k in sorted(rows):
        if k.startswith("transport:"):
            continue
        if k not in cli["rows"] and k not in m["rows"]:
            rows[k]["reason"] = "NOT_EXECUTED_VIA_INSTALLED_TRANSPORT"
    original = {"schema_version": 1, "started_at": now(),
                "source_sha": inv["source_sha"],
                "target": {"read_only_username": owned},
                "cli": cli, "mcp": m, "rows": rows,
                "status": "PARTIAL_INSTALLED_READONLY_NOT_SYSTEM_ACCEPTANCE"}
    summary = {"schema_version": 1, "source_sha": inv["source_sha"],
               "row_count": len(rows),
               "recorded_evidence_entries": sum(len(x["evidence"]) for x in (m, cli)),
               "stdio_mcp_positive_attempts": len(m["evidence"]) if m["catalogue"] == "DISCOVERED" else 0,
               "cli_readonly_positive_attempts": sum(
                   1 for x in cli["evidence"] if x.startswith("host_cli:")),
               "cli_help_observations": sum(
                   1 for x in cli["evidence"] if x.startswith("installed_cli_leaf:")),
               "passed_system_scenarios": 0,
               "has_independent_oracle": False,
               "status": original["status"]}
    unique_file(dest / "original.json", original)
    unique_file(dest / "summary.json", summary)
    return summary


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--root", default=".")
    p.add_argument("--out", required=True)
    p.add_argument("--host-argv-json")
    p.add_argument("--mcp-argv-json")
    p.add_argument("--read-only-target")
    args = p.parse_args(argv)
    try:
        result = execute(args.root, args.out,
                         host=argv_file(args.host_argv_json, "host") if args.host_argv_json else None,
                         mcp=argv_file(args.mcp_argv_json, "mcp") if args.mcp_argv_json else None,
                         owned=args.read_only_target)
        print(json.dumps(result, sort_keys=True))
        return 0
    except (AcceptanceFault, ValueError, OSError, KeyError) as exc:
        print(json.dumps({"status": "FAIL_CLOSED", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())

