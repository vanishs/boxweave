#!/usr/bin/env python3
"""Loopback MCP test helper CLI. Python suites should import mcp_testkit directly."""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from mcp_testkit import (LoopbackMCP, MCPError, ProbeReport,
                         token_from_env, token_from_file)


def cli_parser() -> argparse.ArgumentParser:
    ap = argparse.ArgumentParser(description="Read-only-by-default local BoxWeave MCP protocol tester")
    ap.add_argument("--url", default="http://127.0.0.1:18650/mcp", help="loopback http://127.0.0.1:PORT/mcp only")
    creds = ap.add_mutually_exclusive_group()
    creds.add_argument("--token-env", default="BOXWEAVE_MCP_TOKEN", help="environment variable containing Bearer token")
    creds.add_argument("--token-file", type=Path, help="owner-only file (chmod 600), never CLI argv raw token")
    ap.add_argument("--timeout", type=float, default=12.0)
    ap.add_argument("--report", type=Path, help="write sanitized JSON report (no tokens/arguments)")
    ap.add_argument("--allow-mutations", action="store_true", help="opt in for disposable test fixtures only")
    ap.add_argument("--ack-disposable", action="store_true", help="acknowledge safe isolated environment for mutation")
    subs = ap.add_subparsers(dest="action", required=True)
    subs.add_parser("inspect", help="get current tool catalog, read-only annotations and negotiated protocol")
    p = subs.add_parser("smoke", help="read-only metadata, tool catalog, history and optional Activity tests")
    p.add_argument("--expected-tools", type=int)
    p.add_argument("--expected-schema", type=int)
    p.add_argument("--require-activity", action="store_true")
    p = subs.add_parser("call", help="typed tool call; default blocks all mutation and unannotated tools")
    p.add_argument("name")
    p.add_argument("--args-file", type=Path, help="JSON object containing tool parameters")
    p = subs.add_parser("recover", help="read-only bounded original/UNKNOWN recovery by work_ref")
    p.add_argument("--work-ref", required=True)
    p.add_argument("--max-items", type=int, default=100)
    return ap


def main(argv: list[str] | None = None) -> int:
    args = cli_parser().parse_args(argv)
    if args.allow_mutations and not args.ack_disposable:
        print("ERROR: --allow-mutations requires --ack-disposable", file=sys.stderr)
        return 2
    if args.ack_disposable and not args.allow_mutations:
        print("ERROR: --ack-disposable is meaningful only with --allow-mutations", file=sys.stderr)
        return 2
    report = ProbeReport("boxweave-python-loopback-" + args.action)
    try:
        token = token_from_file(args.token_file) if args.token_file else token_from_env(args.token_env)
        with LoopbackMCP(args.url, token=token, timeout=args.timeout,
                         allow_mutations=args.allow_mutations) as mcp:
            tools = mcp.all_tools()
            if args.action == "inspect":
                result = {
                    "tool_count": len(tools), "names": sorted(tools),
                    "read_only": sorted(n for n, t in tools.items() if (t.get("annotations") or {}).get("readOnlyHint") is True),
                    "protocol": mcp.protocol,
                    "scope": "LOCAL_LOOPBACK_ONLY",
                }
                report.check("tools_list", bool(tools))
            elif args.action == "smoke":
                report.check("tools_list_nonempty", bool(tools))
                if args.expected_tools is not None:
                    report.check("expected_tool_count", len(tools) == args.expected_tools)
                status = mcp.host_status()
                report.check("host_status_schema_present", isinstance(status.get("schema_version"), int))
                if args.expected_schema is not None:
                    report.check("expected_schema_version", status["schema_version"] == args.expected_schema)
                page = mcp.operation_list(limit=2)
                report.check("operation_list_bounded", isinstance(page.get("operations"), list))
                if "box_activity_get" in tools:
                    act = mcp.activity("issues", limit=2)
                    report.check("activity_source_sqlite", act.get("source") == "SQLITE_SNAPSHOT")
                    report.check("activity_does_not_claim_complete_native", act.get("attention_complete") is False)
                elif args.require_activity:
                    report.check("required_activity_exposed", False)
                result = report.as_dict()
            elif args.action == "call":
                params = json.loads(args.args_file.read_text(encoding="utf-8")) if args.args_file else {}
                if not isinstance(params, dict):
                    raise ValueError("tool args-file must contain a JSON object")
                reply = mcp.call(args.name, params)
                report.check("tool_result_valid", isinstance(reply.structured, dict))
                # Do not print arbitrary guest/file/output data. Tests may
                # inspect the returned structure in memory via the library.
                result = {"tool": args.name, "keys": sorted(reply.structured),
                          "tracking_id": reply.structured.get("tracking_id"),
                          "scope": "LOCAL_LOOPBACK_ONLY", "outcome": "MCP_CALL_RETURNED_NOT_NATIVE_PROOF"}
            else:
                result = mcp.recover_work(args.work_ref, max_items=args.max_items)
                report.check("manifest_is_explicitly_read_only", result["safe_to_retry_unknown"] is False)
        if args.report:
            report.write(args.report)
        print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    except (MCPError, ValueError, KeyError, OSError, json.JSONDecodeError, AssertionError) as exc:
        # CLI never echoes exception text from server or raw credential.
        if args.report:
            try:
                report.write(args.report)
            except OSError:
                pass
        print(f"ERROR: {type(exc).__name__}: local MCP probe failed (details omitted)", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
