"""Explicitly approved, one-attempt nested-host MCP lifecycle experiment.

This code is development-only. It requires a private grant bound to the
machine, outer Box incarnation and a unique child name. Unknown writes stop
the workflow without retry or deletion. The independent observation is native
Incus readback, not the same MCP response.
"""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import sys
import time

from .engine import (AcceptanceFault, argv_file, capture, load, mcp_session,
                     now, private_mkdir, unique_file)

SCOPE = "test_owned_nested_lifecycle_only"
CONFIRM = "I_AUTHORIZE_TEST_OWNED_LIFECYCLE"
NAME = re.compile(r"^bwracc-[a-z0-9-]{5,36}$")
# Cold Provider creates have legitimately exceeded 75 seconds. Keep an explicit
# bounded budget per action and preserve UNKNOWN on any later timeout.
ACTION_TIMEOUT = {
    "create_container": 240,
    "stop_container": 120,
    "start_container": 120,
    "delete_container": 180,
}
CHILD_RESOURCES = {"cpu": "1", "memory": "1GB", "disk": "8GB"}


def grant_read(path, confirm):
    if confirm != CONFIRM:
        raise AcceptanceFault("LIVE_LIFECYCLE_CONFIRMATION_REQUIRED")
    path = Path(path)
    if path.is_symlink() or not path.is_file():
        raise AcceptanceFault("GRANT_MISSING")
    stat = path.stat()
    if stat.st_uid != os.geteuid() or stat.st_mode & 0o077:
        raise AcceptanceFault("GRANT_NOT_PRIVATE_OR_OWNED")
    grant = load(path)
    if grant.get("scope") != SCOPE or grant.get("approved") is not True:
        raise AcceptanceFault("GRANT_SCOPE_INVALID")
    if grant.get("hostname") != socket.gethostname():
        raise AcceptanceFault("GRANT_HOST_CHANGED")
    if grant.get("machine_id") != Path("/etc/machine-id").read_text().strip():
        raise AcceptanceFault("GRANT_MACHINE_CHANGED")
    if (not re.fullmatch(r"[a-f0-9-]{36}", str(grant.get("outer_uuid", ""))) or
            not re.fullmatch(r"\d{9,13}", str(grant.get("outer_created_at", "")))):
        raise AcceptanceFault("GRANT_OUTER_IDENTITY_MISSING")
    if not isinstance(grant.get("expires_epoch"), (float, int)) or not (
            0 < grant["expires_epoch"] - time.time() < 86400):
        raise AcceptanceFault("GRANT_EXPIRED_OR_UNBOUNDED")
    if not NAME.fullmatch(str(grant.get("child_name", ""))):
        raise AcceptanceFault("GRANT_UNSAFE_CHILD_NAME")
    return grant


def native_list():
    # Read only the nested Provider, under a preapproved test-owned inner host.
    result = capture(["sudo", "-n", "incus", "--force-local", "--project",
                      "default", "list", "--format=json"], timeout=20)
    if result["execution"] != "completed" or result["exit_code"] != 0:
        raise AcceptanceFault("NATIVE_INCUS_READ_UNAVAILABLE")
    try:
        items = json.loads(result["stdout"])
    except ValueError:
        raise AcceptanceFault("NATIVE_INCUS_RESPONSE_INVALID") from None
    if not isinstance(items, list):
        raise AcceptanceFault("NATIVE_INCUS_RESPONSE_INVALID")
    return {x.get("name"): x for x in items if isinstance(x, dict)
            and isinstance(x.get("name"), str)}


def incarnation(row):
    if not isinstance(row, dict):
        raise AcceptanceFault("NATIVE_IDENTITY_MISSING")
    config = row.get("config") or {}
    uuid = config.get("volatile.uuid")
    created = row.get("created_at")
    if not isinstance(uuid, str) or not re.fullmatch(r"[a-f0-9-]{36}", uuid):
        raise AcceptanceFault("NATIVE_UUID_MISSING")
    if not isinstance(created, str) or not created:
        raise AcceptanceFault("NATIVE_CREATED_MISSING")
    return {"uuid": uuid, "created_at": created}


def one_call(argv, tool, arguments):
    if tool not in ACTION_TIMEOUT:
        raise AcceptanceFault("UNREVIEWED_LIFECYCLE_ACTION")
    timeout = ACTION_TIMEOUT[tool]
    p = mcp_session(argv, [{"jsonrpc": "2.0", "id": "racc-lifecycle",
                            "method": "tools/call",
                            "params": {"name": tool, "arguments": arguments}}],
                    timeout=timeout)
    response = next((x for x in p["messages"] if
                     x.get("id") == "racc-lifecycle"), None)
    result = response.get("result", {}) if isinstance(response, dict) else {}
    success = (p["process"].get("execution") == "completed" and
               not (isinstance(response, dict) and response.get("error")) and
               isinstance(result, dict) and result.get("isError") is not True and
               isinstance(result.get("content"), list))
    return {"tool": tool, "request": arguments, "attempted_at": now(),
            "timeout_seconds": timeout,
            "response": response, "process": p["process"],
            "success_response": bool(success)}


def run(grant, confirmation, mcp_argv, output, *, with_readonly_breadth=False):
    approved = grant_read(grant, confirmation)
    name = approved["child_name"]
    dest = Path(output)
    if dest.exists() or dest.is_symlink():
        raise AcceptanceFault("EXPERIMENT_OUTPUT_ALREADY_EXISTS")
    private_mkdir(dest)
    unique_file(dest / "lab.json", {"scope": SCOPE, "child_name": name,
                "outer_uuid": approved["outer_uuid"],
                "outer_created_at": approved["outer_created_at"],
                "started_at": now()})
    inventory = native_list()
    if name + "-container" in inventory or name in inventory:
        raise AcceptanceFault("CHILD_NAME_ALREADY_EXISTS_REFUSE_REUSE")

    def append(index, tool, args, expected, *, original_id=None):
        prefix = f"{index:02d}-{tool}"
        # Pre-dispatch durable intent. An interrupted call must never be retried.
        unique_file(dest / (prefix + ".intent.json"),
                    {"name": name, "tool": tool, "arguments": args,
                     "original_incarnation": original_id,
                     "state": "MAY_BE_DISPATCHED_ONCE", "recorded_at": now()})
        attempt = one_call(mcp_argv, tool, args)
        unique_file(dest / (prefix + ".attempt.json"), attempt)
        try:
            children = native_list()
            observed = children.get(name + "-container") or children.get(name)
            before = incarnation(observed) if observed else None
            oracle = {"observed_at": now(), "original_incarnation": original_id,
                      "native_child": {"name": observed.get("name"), "status": observed.get("status")} if observed else None, "incarnation": before}
            unique_file(dest / (prefix + ".oracle.json"), oracle)
        except (AcceptanceFault, OSError, ValueError) as exc:
            unique_file(dest / (prefix + ".unknown.json"),
                        {"state": "UNKNOWN_NO_REPLAY", "cause": str(exc)})
            raise AcceptanceFault("UNKNOWN_PROVIDER_READ_NO_REPLAY") from exc
        if not attempt["success_response"]:
            response = attempt.get("response")
            error = response.get("error") if isinstance(response, dict) else None
            # The installed Gateway performs the server-owned key preflight
            # before forwarding create_container. Confirmed typed local
            # refusal plus an absent native child is not a Provider timeout.
            if (tool == "create_container" and isinstance(error, dict)
                    and error.get("message") == "MANAGED_IDENTITY_UNAVAILABLE"
                    and observed is None):
                unique_file(dest / (prefix + ".denied.json"), {
                    "state": "PRE_DISPATCH_DENIED",
                    "code": "MANAGED_IDENTITY_UNAVAILABLE",
                    "independent_native_child": "absent",
                    "recorded_at": now()})
                raise AcceptanceFault("MCP_CREATE_IDENTITY_PREFLIGHT_DENIED")
            unique_file(dest / (prefix + ".unknown.json"),
                        {"state": "UNKNOWN_NO_REPLAY",
                         "cause": "MCP_RESPONSE_NOT_ACKNOWLEDGED"})
            raise AcceptanceFault("MCP_WRITE_RESULT_UNKNOWN_NO_REPLAY")
        if expected == "absent":
            if observed is not None:
                raise AcceptanceFault("NATIVE_DELETE_SIDE_EFFECT_MISSING")
        elif observed is None:
            raise AcceptanceFault("NATIVE_EXPECTED_CHILD_ABSENT")
        elif original_id and oracle["incarnation"] != original_id:
            raise AcceptanceFault("CHILD_INCARNATION_CHANGED_NO_MORE_WRITES")
        elif expected == "running" and str(observed.get("status", "")).lower() not in ("running",):
            raise AcceptanceFault("NATIVE_RUNNING_STATE_MISMATCH")
        elif expected == "stopped" and str(observed.get("status", "")).lower() not in ("stopped",):
            raise AcceptanceFault("NATIVE_STOPPED_STATE_MISMATCH")
        return oracle

    try:
        created = append(1, "create_container",
                         {"username": name, **CHILD_RESOURCES}, "running")
        identity = created["incarnation"]
        if not identity:
            raise AcceptanceFault("CREATED_CHILD_IDENTITY_UNAVAILABLE")
        append(2, "stop_container", {"username": name}, "stopped", original_id=identity)
        append(3, "start_container", {"username": name}, "running", original_id=identity)
        append(4, "delete_container", {"username": name}, "absent", original_id=identity)
    except AcceptanceFault as exc:
        unique_file(dest / "result.json", {
                    "state": ("BLOCKED_PRE_DISPATCH" if
                              str(exc) == "MCP_CREATE_IDENTITY_PREFLIGHT_DENIED"
                              else "UNKNOWN_OR_FAIL"),
                    "reason": str(exc), "child_name": name,
                    "cleanup": "MANUAL_ORIGINAL_IDENTITY_ONLY",
                    "no_automatic_retry": True})
        raise
    result = {"state": "PARTIAL", "scenario_id": "SYS-LIFE-01",
              "limited_coverage": "CREATE_STOP_START_DELETE",
              "not_tested": ["resize", "guest exec", "cross-interface auth"],
              "oracle": "NATIVE_INCUS_EXACT_INCARNATION",
              "outer_uuid": approved["outer_uuid"],
              "child_name": name, "completed_at": now()}
    unique_file(dest / "result.json", result)
    return result


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--grant", required=True)
    p.add_argument("--confirm", required=True)
    p.add_argument("--mcp-argv-json", required=True)
    p.add_argument("--out", required=True)
    args = p.parse_args(argv)
    try:
        result = run(args.grant, args.confirm,
                     argv_file(args.mcp_argv_json, "mcp"), args.out)
        print(json.dumps(result, sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, KeyError) as exc:
        print(json.dumps({"status": "FAIL_CLOSED", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())

