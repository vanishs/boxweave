#!/usr/bin/env python3
"""Independent S4 test witness of one FRESH native Incus Operation.

Read-only: original SQLite WAL + native 'incus query'; no MCP writes, no
re-dispatch, no credentials, raw output or unredacted native error messages.
Run while a disposable original is still active: Incus may expire the object.
"""
import argparse
import datetime
import json
import re
import sqlite3
import subprocess
import time
from pathlib import Path
from urllib.parse import quote

HEX32 = re.compile(r"^[0-9a-f]{32}$")
NATIVE = re.compile(r"^[0-9a-fA-F-]{32,36}$")


def journal(db_path, op_id):
    uri = Path(db_path).resolve().as_uri() + "?mode=ro"
    with sqlite3.connect(uri, uri=True, timeout=3) as db:
        row = db.execute(
            "SELECT id,action,layer,box_uuid,incarnation,resource_key,"
            "native_operation_id,state FROM operations WHERE id=?",
            (op_id,),
        ).fetchone()
        if not row:
            return None
        names = ("operation_id", "action", "project", "box_uuid",
                 "incarnation", "resource_key", "native_operation_id", "state")
        result = dict(zip(names, row))
        if result["action"] == "box_exec":
            out = db.execute(
                "SELECT exit_code,truncation FROM exec_outputs WHERE operation_id=?",
                (op_id,),
            ).fetchone()
            result["guest_exit_code"] = None if out is None else out[0]
            result["output_truncated"] = None if out is None else bool(out[1])
        return result


def query_original(project, native_id):
    route = "/1.0/operations/" + native_id + "?project=" + quote(project, safe="")
    try:
        proc = subprocess.run(
            ["incus", "query", route], capture_output=True, text=True,
            timeout=5, check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        return None, "NATIVE_READ_UNAVAILABLE"
    if proc.returncode != 0:
        # A 404 does NOT establish whether the original ever ran.
        return None, "NATIVE_NOT_RETAINED_OR_UNAVAILABLE"
    try:
        response = json.loads(proc.stdout)
        op = response["metadata"]
        if not isinstance(op, dict):
            return None, "NATIVE_METADATA_MISSING"
        if str(op.get("id", "")).rsplit("/", 1)[-1] != native_id:
            return None, "NATIVE_ORIGINAL_ID_MISMATCH"
        code, status = op.get("status_code"), op.get("status")
        if not isinstance(code, int) or not isinstance(status, str):
            return None, "NATIVE_STATUS_UNVERIFIED"
        return {
            "native_operation_id": native_id,
            "status_code": code,
            "status": status,
            "error_present": bool(op.get("err")),
        }, None
    except (ValueError, KeyError, TypeError):
        return None, "NATIVE_RESPONSE_INVALID"


def collect(db_path, op_id, seconds, read=journal, native=query_original):
    deadline = time.monotonic() + seconds
    first, observed = None, None
    reason = "NO_NATIVE_ORIGINAL_RECEIPT"
    while True:
        first = read(db_path, op_id)
        if first is None:
            reason = "SQLITE_ORIGINAL_NOT_FOUND"
        elif first["native_operation_id"]:
            native_id = first["native_operation_id"]
            if not NATIVE.fullmatch(native_id):
                reason = "NATIVE_ID_FORMAT_UNVERIFIED"
                break
            observed, reason = native(first["project"], native_id)
            if observed is not None and observed.get("status_code") in (200, 400, 401):
                break
            if observed is not None:
                reason = "NATIVE_NOT_TERMINAL"
        if time.monotonic() >= deadline:
            break
        time.sleep(min(.1, max(0, deadline - time.monotonic())))
    terminal = bool(observed is not None and first is not None and
                    observed.get("status_code") in (200, 400, 401) and
                    observed.get("native_operation_id") == first["native_operation_id"])
    return {
        "source": "INDEPENDENT_INCUS_CLI_READ_ONLY",
        "timestamp_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "observation": "CAPTURED_TERMINAL" if terminal else "NOT_CAPTURED",
        "reason": None if terminal else reason,
        "original_sqlite": first,
        "journal_after": read(db_path, op_id),
        "native_terminal": observed if terminal else None,
        "original_id_matches": terminal,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--db", required=True)
    parser.add_argument("--operation-id", required=True)
    parser.add_argument("--timeout-seconds", type=float, default=15)
    args = parser.parse_args()
    if not HEX32.fullmatch(args.operation_id) or not (0 < args.timeout_seconds <= 120):
        parser.error("invalid original ID or bounded timeout")
    if not Path(args.db).is_file():
        parser.error("SQLite file unavailable")
    result = collect(args.db, args.operation_id, args.timeout_seconds)
    print(json.dumps(result, sort_keys=True, separators=(",", ":")))
    return 0 if result["original_id_matches"] else 2


if __name__ == "__main__":
    raise SystemExit(main())
