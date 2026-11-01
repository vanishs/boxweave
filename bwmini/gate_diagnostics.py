"""Private, bounded gateway diagnostics and exact-Box D4 policy.

This module runs as the unprivileged gateway account. Never persist MCP
arguments, outputs, exceptions, prompts, commands, credentials or raw logs.
"""
import contextlib
import datetime
import json
import os
from pathlib import Path
import re
import secrets
import sqlite3
import stat
import subprocess
import time

SETTINGS_PATH = Path("/etc/boxweave/diagnostics.json")
PROJECT_POLICY = Path("/etc/boxweave/project-diagnostics.json")
JOURNAL = Path("/home/boxweave/.local/state/boxweave-gateway/diagnostics.sqlite3")
APP_PREFIX = Path("/home/boxweave/boxweave")
DEFAULTS = {"enabled": True, "auto_diagnose": True,
            "retention_days": 7, "max_storage_mb": 50}
TOOL = re.compile(r"[a-z][a-z0-9_]{0,127}\Z")
TRACE = re.compile(r"[0-9a-f]{32}\Z")
LOGICAL = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
SERVICE = re.compile(r"[A-Za-z0-9_.@-]{1,80}\Z")
CHECKS = frozenset(("processes", "ports", "service", "logs"))
KINDS = frozenset(("hello", "environment", "access"))
EVENT_LIMIT = 25


def _owner_file(path, *, owner=0, private=False):
    info = path.lstat()
    if (not stat.S_ISREG(info.st_mode) or info.st_uid != owner
            or info.st_nlink != 1 or info.st_size > 16384
            or (info.st_mode & (0o077 if private else 0o022))):
        raise ValueError("DIAGNOSTIC_POLICY_UNSAFE")
    return json.loads(path.read_text(encoding="utf-8"))


def validate_settings(value):
    if (not isinstance(value, dict) or set(value) - set(DEFAULTS)
            or any(type(value.get(key, DEFAULTS[key])) is not bool
                   for key in ("enabled", "auto_diagnose"))
            or type(value.get("retention_days", 7)) is not int
            or not 1 <= value.get("retention_days", 7) <= 90
            or type(value.get("max_storage_mb", 50)) is not int
            or not 1 <= value.get("max_storage_mb", 50) <= 512):
        raise ValueError("DIAGNOSTIC_SETTINGS_INVALID")
    return DEFAULTS | value


def settings(path=SETTINGS_PATH):
    if not path.exists() and not path.is_symlink():
        return DEFAULTS.copy()
    return validate_settings(_owner_file(path))


def _journal_parent(path, *, create):
    """Only create a private leaf under an existing owner-only home directory."""
    parent = path.parent
    if create:
        parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    for folder in (parent, parent.parent):
        data = folder.lstat()
        if not stat.S_ISDIR(data.st_mode) or data.st_uid != os.geteuid() or data.st_mode & 0o077:
            raise ValueError("DIAGNOSTIC_STORAGE_UNSAFE")
    if path.exists() or path.is_symlink():
        data = path.lstat()
        if (not stat.S_ISREG(data.st_mode) or data.st_uid != os.geteuid()
                or data.st_nlink != 1 or data.st_mode & 0o077):
            raise ValueError("DIAGNOSTIC_STORAGE_UNSAFE")


def _open(path, *, write):
    _journal_parent(path, create=write)
    if not write and not path.is_file():
        return None
    if write:
        # Reject symlink/hardlink before SQLite opens the file.
        if not path.exists():
            fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
            os.close(fd)
        db = sqlite3.connect(str(path), timeout=0.5)
        db.execute("PRAGMA busy_timeout=500")
        db.execute("CREATE TABLE IF NOT EXISTS events ("
                   "seq INTEGER PRIMARY KEY, trace TEXT NOT NULL, utc REAL NOT NULL,"
                   "tool TEXT NOT NULL, outcome TEXT NOT NULL, code TEXT NOT NULL,"
                   "elapsed_ms INTEGER NOT NULL)")
        db.execute("CREATE INDEX IF NOT EXISTS idx_gateway_trace ON events(trace,seq)")
        return db
    db = sqlite3.connect("file:" + str(path) + "?mode=ro", uri=True, timeout=0.5)
    db.execute("PRAGMA query_only=ON")
    return db


def classify(message):
    """No unreviewed Provider response text or error payload enters the journal."""
    if not isinstance(message, dict):
        return "unknown", "UNVERIFIED"
    if "error" in message:
        return "unknown", "UPSTREAM_JSONRPC_ERROR"
    result = message.get("result")
    if not isinstance(result, dict):
        return "unknown", "RESPONSE_UNVERIFIED"
    if result.get("isError") is True:
        return "failed", "UPSTREAM_TOOL_ERROR"
    return "succeeded", "MCP_RESPONSE_RECEIVED"


def record(tool, message, elapsed_ms, *, path=JOURNAL, config=None, trace=None, clock=time.time):
    """Best-effort, never throws. Caller handles original MCP outcome unchanged."""
    token = trace or secrets.token_hex(16)
    try:
        policy = validate_settings(config) if config is not None else settings()
        if not policy["enabled"] or not TOOL.fullmatch(tool):
            return token
        outcome, code = classify(message)
        _record_row(path, policy, token, tool, outcome, code,
                    max(0, min(3_600_000, int(elapsed_ms))), clock())
    except (OSError, ValueError, TypeError, OverflowError, sqlite3.Error):
        pass
    return token


def _record_row(path, policy, trace, tool, outcome, code, elapsed, now):
    with contextlib.closing(_open(path, write=True)) as db:
        # Journal is short-lived support metadata, not an operation receipt.
        with db:
            db.execute("INSERT INTO events(trace,utc,tool,outcome,code,elapsed_ms)"
                       " VALUES(?,?,?,?,?,?)", (trace, now, tool, outcome, code, elapsed))
            db.execute("DELETE FROM events WHERE utc<?",
                       (now - policy["retention_days"] * 86400,))
            db.execute("DELETE FROM events WHERE seq NOT IN"
                       " (SELECT seq FROM events ORDER BY seq DESC LIMIT 20000)")
        if path.stat().st_size > policy["max_storage_mb"] * 1024 * 1024:
            # Drop oldest half instead of any unbounded vacuum or journal growth.
            with db:
                db.execute("DELETE FROM events WHERE seq IN"
                           " (SELECT seq FROM events ORDER BY seq LIMIT"
                           " (SELECT (COUNT(*)+1)/2 FROM events))")


def query_gateway(args, *, path=JOURNAL, config=None):
    policy = validate_settings(config) if config is not None else settings()
    if not policy["enabled"]:
        return {"source": "gateway", "state": "disabled", "events": [], "coverage": "not_checked"}
    if not isinstance(args, dict) or set(args) - {"source", "trace_id", "recent"}:
        raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
    trace = args.get("trace_id")
    recent = args.get("recent")
    if (trace is None) == (recent is None):
        raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
    if trace is not None and (not isinstance(trace, str) or not TRACE.fullmatch(trace)):
        raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
    if recent is not None and (type(recent) is not int or not 1 <= recent <= EVENT_LIMIT):
        raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
    try:
        connection = _open(path, write=False)
        if connection is None:
            return {"source": "gateway", "state": "not_verified", "coverage": "no_records", "events": []}
        with contextlib.closing(connection) as db:
            if trace is not None:
                rows = db.execute("SELECT trace,utc,tool,outcome,code,elapsed_ms"
                                  " FROM events WHERE trace=? AND utc>=?"
                                  " ORDER BY seq DESC LIMIT ?", (trace, time.time() - policy["retention_days"] * 86400, EVENT_LIMIT)).fetchall()
            else:
                rows = db.execute("SELECT trace,utc,tool,outcome,code,elapsed_ms"
                                  " FROM events WHERE utc>=?"
                                  " ORDER BY seq DESC LIMIT ?",
                                  (time.time() - policy["retention_days"] * 86400, recent)).fetchall()
        return {"source": "gateway", "state": "observed" if rows else "not_verified",
                "coverage": "gateway_only", "events": [
                    {"trace_id": t, "time_utc": datetime.datetime.fromtimestamp(utc, datetime.timezone.utc).isoformat(),
                     "tool": tool, "outcome": outcome, "code": code, "elapsed_ms": duration}
                    for t, utc, tool, outcome, code, duration in rows],
                "root_cause": "not_verified"}
    except (OSError, sqlite3.Error, ValueError):
        return {"source": "gateway", "state": "not_verified",
                "coverage": "journal_unavailable", "events": []}


def query_app(args, *, prefix=APP_PREFIX):
    """Fixed app-owned reader, invoked with no actor/path/command from caller."""
    if not isinstance(args, dict) or args.get("source") != "app":
        raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
    if set(args) == {"source", "trace_id"}:
        trace = args["trace_id"]
        if not isinstance(trace, str) or not TRACE.fullmatch(trace):
            raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
        selector = {"kind": "trace", "trace_id": trace, "actor": "operator"}
    elif set(args) == {"source", "operation_kind", "operation_id"}:
        kind, ident = args["operation_kind"], args["operation_id"]
        if kind not in KINDS or not isinstance(ident, str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,62}", ident):
            raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
        selector = {"kind": "operation", "operation_kind": kind,
                    "operation_id": ident, "actor": "operator"}
    else:
        raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
    try:
        proc = subprocess.run(
            [str(prefix / "current/.venv/bin/python"), "-m", "bwmini.diagnostics",
             "--config", str(prefix / "config.json"),
             "--selector", json.dumps(selector, separators=(",", ":"))],
            cwd=str(prefix / "current"), stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=10,
            check=False)
        if proc.returncode or len(proc.stdout) > 131072:
            raise ValueError("APP_READER_UNAVAILABLE")
        data = json.loads(proc.stdout)
        if not isinstance(data, dict) or type(data.get("found")) is not bool:
            raise ValueError("APP_READER_UNAVAILABLE")
        # Drop all arbitrary fields; even the app reader's receipt could evolve.
        events = data.get("events", [])
        if not isinstance(events, list):
            raise ValueError("APP_READER_UNAVAILABLE")
        allowed = ("time_utc", "component", "phase", "outcome", "code", "capture")
        timeline = [{k: e[k] for k in allowed if k in e and isinstance(e[k], (int, str))
                     and len(str(e[k])) <= 128} for e in events[:25] if isinstance(e, dict)]
        return {"source": "app", "state": "observed" if data["found"] else "not_verified",
                "coverage": "app_owned_reader", "trace_id": data.get("trace_id") if
                isinstance(data.get("trace_id"), str) and TRACE.fullmatch(data["trace_id"]) else None,
                "events": timeline, "omissions": [
                    s for s in data.get("omissions", [])[:8] if isinstance(s, str) and
                    re.fullmatch(r"[a-z_]{1,64}", s)],
                "root_cause": "not_verified"}
    except (OSError, subprocess.TimeoutExpired, ValueError, TypeError, KeyError):
        return {"source": "app", "state": "not_verified",
                "coverage": "app_reader_unavailable", "events": [],
                "root_cause": "not_verified"}


def validate_project_policy(value):
    """Operator allowlist binds an incarnation, not just a logical Box name."""
    if not isinstance(value, dict) or set(value) != {"version", "boxes"} or value["version"] != 1:
        raise ValueError("PROJECT_POLICY_INVALID")
    boxes = value["boxes"]
    if not isinstance(boxes, dict) or len(boxes) > 64:
        raise ValueError("PROJECT_POLICY_INVALID")
    for name, row in boxes.items():
        if (not isinstance(name, str) or not LOGICAL.fullmatch(name)
                or not isinstance(row, dict)
                or set(row) != {"instance_name", "created", "checks", "services"}
                or not isinstance(row["instance_name"], str)
                or not re.fullmatch(r"[A-Za-z0-9_.-]{1,96}", row["instance_name"])
                or not isinstance(row["created"], str) or not 1 <= len(row["created"]) <= 100
                or not isinstance(row["checks"], list) or len(row["checks"]) > 4
                or any(c not in CHECKS for c in row["checks"])
                or not isinstance(row["services"], list) or len(row["services"]) > 24
                or any(not isinstance(s, str) or not SERVICE.fullmatch(s)
                       for s in row["services"])):
            raise ValueError("PROJECT_POLICY_INVALID")
    return boxes


def project_grant(args, observed, *, policy_path=PROJECT_POLICY, policy=None):
    if (not isinstance(args, dict) or set(args) - {"box", "check", "service"}
            or not isinstance(args.get("box"), str) or not LOGICAL.fullmatch(args["box"])
            or args.get("check") not in CHECKS):
        raise ValueError("PROJECT_SELECTOR_INVALID")
    check, box = args["check"], args["box"]
    if check in ("service", "logs"):
        if not isinstance(args.get("service"), str) or not SERVICE.fullmatch(args["service"]):
            raise ValueError("PROJECT_SELECTOR_INVALID")
    elif "service" in args:
        raise ValueError("PROJECT_SELECTOR_INVALID")
    if policy is None:
        try:
            policy = _owner_file(policy_path)
        except FileNotFoundError:
            raise ValueError("PROJECT_DIAG_NOT_AUTHORIZED") from None
    boxes = validate_project_policy(policy)
    grant = boxes.get(box)
    if grant is None or check not in grant["checks"] or (
            check in ("service", "logs") and args["service"] not in grant["services"]):
        raise ValueError("PROJECT_DIAG_NOT_AUTHORIZED")
    if (not isinstance(observed, dict) or observed.get("username") != box
            or observed.get("name") != grant["instance_name"]
            or str(observed.get("createdAt")) != grant["created"]
            or observed.get("state") != "CONTAINER_STATE_RUNNING"):
        raise ValueError("PROJECT_TARGET_CHANGED")
    return grant


def command_for_check(args):
    kind = args["check"]
    if kind == "processes":
        return "ps -eo stat=,comm= | head -n 40"
    if kind == "ports":
        return "ss -lntH | head -n 40"
    service = args["service"]
    if not SERVICE.fullmatch(service):
        raise ValueError("PROJECT_SELECTOR_INVALID")
    if kind == "service":
        return "systemctl --user is-active -- " + service
    if kind == "logs":
        return "journalctl --user -u " + service + " --no-pager -n 40 --output=cat"
    raise ValueError("PROJECT_SELECTOR_INVALID")


def project_summary(args, response):
    """Interpret known execution envelope, never return raw stdout/stderr."""
    if not isinstance(response, dict) or response.get("isError"):
        return {"state": "not_verified", "code": "PROJECT_READ_UNAVAILABLE"}
    structured = response.get("structuredContent")
    if isinstance(structured, dict) and type(structured.get("exit_code")) is int:
        code, out = structured["exit_code"], structured.get("stdout")
    else:
        blocks = response.get("content", [])
        text = "".join(b.get("text", "") for b in blocks if isinstance(b, dict))
        found = re.fullmatch(r"exit_code: (-?\d+)\n\n--- stdout ---\n([\s\S]{0,8192})", text)
        if not found:
            return {"state": "not_verified", "code": "PROJECT_READ_FORMAT_UNSUPPORTED"}
        code, out = int(found[1]), found[2]
    if not isinstance(out, str) or len(out.encode("utf-8")) > 8192:
        return {"state": "not_verified", "code": "PROJECT_OUTPUT_LIMIT"}
    if code:
        return {"state": "not_verified", "code": "PROJECT_COMMAND_EXIT_NONZERO",
                "exit_code": max(-255, min(255, code))}
    lines = out.splitlines()[:40]
    kind = args["check"]
    summary = {"state": "observed", "check": kind, "exit_code": 0,
               "scope": "selected_box_guest", "root_cause": "not_verified"}
    if kind == "processes":
        summary["sample_count"] = len(lines)
        summary["zombie_count"] = sum(l.lstrip().startswith("Z") for l in lines)
    elif kind == "ports":
        summary["sample_listening_tcp_count"] = len(lines)
    elif kind == "service":
        status = out.strip()
        summary["service_state"] = status if status in ("active", "inactive", "failed",
            "activating", "deactivating", "unknown") else "not_verified"
    else:
        summary["sample_log_lines"] = len(lines)
        summary["error_hint_count"] = sum(bool(re.search(r"error|exception|failed|timeout|refused",
                                                            line, flags=re.I)) for line in lines)
    return summary
