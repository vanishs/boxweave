"""Dynamic installed-surface breadth accounting and safe black-box attempts.

Every exposed entry gets a row. A help page or read-only response is never a
functional PASS; writes require separately reviewed exact-target scenario grants.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import re
import sys

from .engine import (AcceptanceFault, SAFE_MCP_TOOLS, capture, catalogue,
                     mcp_session, now, unique_file)

CLI_READ_ONLY = frozenset(("version", "status", "provenance", "doctor"))
MAX_DEPTH = 3
MAX_LEAVES = 250


def help_tree(argv):
    if not isinstance(argv, list) or not argv:
        raise AcceptanceFault("HOST_COMMAND_INVALID")
    queue = [()]
    seen = set()
    leaves = {}
    while queue and len(seen) < MAX_LEAVES:
        path = queue.pop(0)
        if path in seen or len(path) > MAX_DEPTH:
            continue
        seen.add(path)
        result = capture(argv + list(path) + ["--help"], timeout=12)
        row = {"argv_suffix": list(path) + ["--help"], "process": result,
               "help_is_not_a_functional_positive": True}
        leaves[" ".join(path) if path else "<root>"] = row
        if result["execution"] != "completed" or result["exit_code"] != 0:
            continue
        # argparse exposes the installed parser's command sets in {a,b,...}.
        # Source-discovered early dispatch remains a separate inventory path.
        for match in re.finditer(r"\{([a-zA-Z0-9_,-]+)\}", result["stdout"]):
            for part in match.group(1).split(","):
                if re.fullmatch(r"[a-z][a-z0-9_-]{0,48}", part):
                    child = path + (part,)
                    if child not in seen and len(child) <= MAX_DEPTH:
                        queue.append(child)
    if queue:
        raise AcceptanceFault("CLI_SURFACE_DISCOVERY_BUDGET_EXCEEDED")
    return leaves


def actual_mcp(mcp_command):
    if not mcp_command:
        return {"status": "NOT_CONFIGURED", "tools": []}
    r = mcp_session(mcp_command, [{"jsonrpc": "2.0", "id": "racc-catalog",
                                  "method": "tools/list"}], timeout=20)
    for item in r["messages"]:
        if item.get("id") != "racc-catalog":
            continue
        tools = item.get("result", {}).get("tools")
        if isinstance(tools, list) and all(isinstance(x, dict) and
                isinstance(x.get("name"), str) for x in tools):
            return {"status": "DISCOVERED", "tools": tools,
                    "stderr_sha256": r["process"].get("stderr_sha256")}
    return {"status": "UNAVAILABLE", "tools": [],
            "reason": "REAL_STDIO_TOOLS_LIST_NOT_AVAILABLE"}


def attempt_mcp(mcp_command, name, arguments):
    result = mcp_session(mcp_command, [{"jsonrpc": "2.0", "id": "racc-call",
                       "method": "tools/call", "params": {
                           "name": name, "arguments": arguments}}], timeout=20)
    response = next((x for x in result["messages"] if x.get("id") == "racc-call"), None)
    return {"execution": result["process"]["execution"],
            "exit_code": result["process"]["exit_code"],
            "stdout_sha256": result["process"]["stdout_sha256"],
            "stderr_sha256": result["process"]["stderr_sha256"],
            "response": response, "captured_at": now()}


def breadth(root, host_command=None, mcp_command=None, *, http=None):
    """Run only guaranteed nonmutating calls; no policy grants are inferred."""
    inv = catalogue(root, host_command=host_command, mcp_command=mcp_command)
    rows = {x["key"]: {"status": "BLOCKED", "reason": "NO_ACTUAL_BLACKBOX_ATTEMPT",
                      "surface": x["surface"]} for x in inv["actions"]}
    evidence = {}
    if host_command:
        cli = help_tree(host_command)
        for name, result in cli.items():
            if name == "<root>":
                continue
            key = "installed_cli_leaf:" + name
            rows[key] = {"status": "PARTIAL", "reason": "HELP_ONLY_NOT_A_FUNCTIONAL_TEST",
                         "surface": "installed_cli_leaf"}
            evidence[key] = {"help": result}
        for name in inv.get("source", {}).get("host_cli", []):
            key = "host_cli:" + name
            if key in rows and rows[key]["status"] == "BLOCKED":
                rows[key]["reason"] = "SOURCE_REGISTRATION_NOT_A_EXECUTED_INSTALLED_COMMAND"
        for action in CLI_READ_ONLY:
            key = "host_cli:" + action
            if key in rows:
                positive = capture(host_command + [action], timeout=15)
                malformed = capture(host_command + [action, "--racc-invalid-arg"],
                                    timeout=15)
                evidence[key] = {"positive": positive, "invalid": malformed}
                rows[key] = {"status": "PARTIAL", "surface": "host_cli",
                             "reason": "READ_ONLY_POSITIVE_AND_INVALID_CAPTURED_NO_INDEPENDENT_ORACLE"}
    else:
        cli = {}
    mcp = actual_mcp(mcp_command)
    for tool in mcp["tools"]:
        name = tool["name"]
        key = "stdio_mcp:" + name
        rows.setdefault(key, {"status": "BLOCKED", "surface": "stdio_mcp",
                              "reason": "RUNTIME_DISCOVERED_NOT_IN_SOURCE_INVENTORY"})
        if name not in SAFE_MCP_TOOLS:
            rows[key] = {"status": "BLOCKED", "surface": "stdio_mcp",
                         "reason": "NOT_ON_REVIEWED_READONLY_AUTOCALL_ALLOWLIST"}
            continue
        schema = tool.get("inputSchema") or {}
        if schema.get("required"):
            rows[key] = {"status": "BLOCKED", "surface": "stdio_mcp",
                         "reason": "REQUIRED_EXACT_TEST_TARGET_INPUT_MISSING"}
            continue
        positive = attempt_mcp(mcp_command, name, {})
        malformed = attempt_mcp(mcp_command, name, {"__racc_unknown_field__": {"invalid": True}})
        evidence[key] = {"positive": positive, "invalid": malformed}
        rows[key] = {"status": "PARTIAL", "surface": "stdio_mcp",
                     "reason": "ACTUAL_STDIO_CALLS_NO_INDEPENDENT_ORACLE_OR_DENIAL"}
    # Authenticated HTTP/MCP/Web catalogs are distinct from stdio and must not be
    # silently replaced by an outer ChatGPT tool list or source annotations.
    if http is None:
        missing_http = "AUTHENTICATED_HTTP_MCP_NOT_CONFIGURED"
    else:
        missing_http = "HTTP_CAPTURE_IS_NOT_A_VERIFIED_AUTHENTICATED_SESSION"
    for key in rows:
        if rows[key]["status"] == "BLOCKED" and not rows[key].get("reason"):
            rows[key]["reason"] = missing_http
    return {"schema_version": 1, "created_at": now(),
            "source_sha": inv["source_sha"], "inventory": inv,
            "stdio_catalog": {"state": mcp["status"],
                              "tool_names": [x["name"] for x in mcp["tools"]]},
            "installed_cli_help": cli,
            "missing_http": missing_http,
            "rows": rows, "attempts": evidence,
            "count": len(rows), "status": "PARTIAL_NOT_FULL_ACCEPTANCE"}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--root", default=".")
    p.add_argument("--host-argv-json")
    p.add_argument("--mcp-argv-json")
    p.add_argument("--out", required=True)
    args = p.parse_args(argv)
    from .engine import argv_file
    try:
        out = Path(args.out)
        if out.exists():
            raise AcceptanceFault("BREADTH_OUTPUT_ALREADY_EXISTS")
        r = breadth(args.root,
                    argv_file(args.host_argv_json, "host") if args.host_argv_json else None,
                    argv_file(args.mcp_argv_json, "mcp") if args.mcp_argv_json else None)
        out.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        unique_file(out, r)
        print(json.dumps({"status": r["status"], "named_rows": r["count"]}))
        return 0
    except (OSError, ValueError, AcceptanceFault) as exc:
        print(json.dumps({"status": "FAIL_CLOSED", "code": str(exc)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
