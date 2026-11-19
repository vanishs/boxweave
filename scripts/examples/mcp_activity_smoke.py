#!/usr/bin/env python3
"""Example: write your own safe MCP suite with a few lines of Python.

Only works against a temporary authenticated local BoxWeave Schema v5 server.
Do not treat empty SQLite Activity pages as permission to replay UNKNOWN work.
"""
import os
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from mcp_testkit import LoopbackMCP, ProbeReport, token_from_env


def run() -> dict:
    report = ProbeReport("example-activity-read-only")
    endpoint = os.environ.get("BOXWEAVE_MCP_URL", "http://127.0.0.1:18650/mcp")
    work_ref = os.environ.get("BOXWEAVE_MCP_WORK_REF", "")
    with LoopbackMCP(endpoint, token=token_from_env(), timeout=12) as mcp:
        tools = mcp.all_tools()
        report.check("activity_tool_advertised", "box_activity_get" in tools)
        status = mcp.host_status()
        report.check("schema_v5", status["schema_version"] == 5)
        overview = mcp.activity("overview", limit=5)
        report.check("overview_bounded", overview["source"] == "SQLITE_SNAPSHOT")
        issues = mcp.activity("issues", limit=5)
        report.check("unknowns_never_all_clear", issues["attention_complete"] is False)
        if work_ref:
            manifest = mcp.recover_work(work_ref, max_items=100)
            report.check("unknown_requires_manual_review", manifest["safe_to_retry_unknown"] is False)
    return report.as_dict()


if __name__ == "__main__":
    import json
    print(json.dumps(run(), indent=2, sort_keys=True))
