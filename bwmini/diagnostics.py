"""Typed, privacy-safe diagnostic evidence; allowlisted fields only, never message content.

Diagnostic events are secondary observations: writing them must never change
operation semantics, and reading them never bypasses actor scoping. No freeform
detail field exists by design - a generic regex scrubber is not a redaction
policy. Trace and event identifiers are unpredictable opaque correlation
labels, not authority tokens.
"""
import argparse
import contextlib
import json
import os
import re
import stat
import secrets
import sqlite3
import time
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from pathlib import Path

from .domain import Fault

SCHEMA_VERSION = 1

KINDS = ("hello", "environment", "access", "job", "selftest")
COMPONENTS = ("web", "worker", "service", "lifecycle", "access", "mcp", "provider", "selftest", "gate")
EVENT_TYPES = ("accepted", "claimed", "submitted", "acknowledged", "observed", "rejected", "unknown", "recovered")
OUTCOMES = ("passed", "failed", "unknown", "not_verified")
CAPTURE_STATES = ("captured", "truncated", "dropped", "unavailable")
# Private error shapes; the public Fault code on receipts never changes.
ERROR_CLASSES = ("UPSTREAM_JSONRPC_REJECTION", "TOOL_IS_ERROR", "TRANSPORT_EOF", "TIMEOUT",
                 "MALFORMED_PROTOCOL", "OUTPUT_LIMIT", "UNCLASSIFIED_EXCEPTION")
CODE_PATTERN = re.compile(r"[A-Z][A-Z0-9_]{0,63}\Z")
ID_PATTERN = re.compile(r"[0-9a-f]{16,64}\Z")
SCOPE_PATTERN = re.compile(r"[A-Za-z0-9_-]{1,64}\Z")
MAX_TARGET_REF = 128
MAX_EVENTS = 20000
MAX_EVENTS_PER_TRACE = 64
RETENTION_SECONDS = 7 * 24 * 3600


def new_trace_id():
    return secrets.token_hex(16)


def new_span_id():
    return secrets.token_hex(8)


def new_event_id():
    return secrets.token_hex(16)


def utc_now(clock=time.time):
    return datetime.fromtimestamp(clock(), timezone.utc).isoformat(timespec="milliseconds")


@dataclass(frozen=True)
class DiagnosticEvent:
    event_id: str
    trace_id: str
    span_id: str
    parent_span_id: str | None
    actor_scope: str
    operation_kind: str | None
    operation_id: str | None
    time_utc: str
    elapsed_ms: int | None
    component: str
    phase: str
    event_type: str
    outcome: str
    code: str
    capture: str
    target_ref: str | None


def build(*, component, phase, event_type, outcome, code, trace_id, actor_scope="operator",
          operation_kind=None, operation_id=None, elapsed_ms=None, target_ref=None,
          capture="captured", clock=time.time):
    """Validate strictly; a programmer error here raises, emit() contains it."""
    if component not in COMPONENTS or event_type not in EVENT_TYPES or outcome not in OUTCOMES \
            or capture not in CAPTURE_STATES or (operation_kind is not None and operation_kind not in KINDS):
        raise ValueError("DIAGNOSTIC_ENUM_INVALID")
    if not CODE_PATTERN.fullmatch(code) or not SCOPE_PATTERN.fullmatch(actor_scope) \
            or not ID_PATTERN.fullmatch(trace_id):
        raise ValueError("DIAGNOSTIC_FIELD_INVALID")
    if len(phase) > 64 or not all(ord(c) >= 32 for c in phase):
        raise ValueError("DIAGNOSTIC_FIELD_INVALID")
    for value in (operation_id, target_ref):
        if value is not None and (not isinstance(value, str) or not value or len(value) > MAX_TARGET_REF):
            raise ValueError("DIAGNOSTIC_FIELD_INVALID")
    if target_ref is not None and any(ord(c) < 32 for c in target_ref):
        raise ValueError("DIAGNOSTIC_FIELD_INVALID")
    if elapsed_ms is not None and (type(elapsed_ms) is not int or elapsed_ms < 0):
        raise ValueError("DIAGNOSTIC_FIELD_INVALID")
    return DiagnosticEvent(event_id=new_event_id(), trace_id=trace_id, span_id=new_span_id(),
                           parent_span_id=None, actor_scope=actor_scope,
                           operation_kind=operation_kind, operation_id=operation_id,
                           time_utc=utc_now(clock), elapsed_ms=elapsed_ms, component=component,
                           phase=phase[:64], event_type=event_type, outcome=outcome, code=code,
                           capture=capture, target_ref=target_ref)


def error_class(error):
    """Fixed private class for an exception; never its text, args or repr."""
    if isinstance(error, BaseException):
        classified = getattr(error, "error_class", None)
        if classified in ERROR_CLASSES:
            return classified
    return "UNCLASSIFIED_EXCEPTION"


def attach(error, error_class):
    """Tag a Fault with its private shape; the public code and status stay unchanged."""
    if error_class in ERROR_CLASSES:
        error.error_class = error_class
    return error


def link(db, actor, kind, operation_id, trace_id=None, clock=time.time):
    """Insert the trace link inside the caller's operation transaction."""
    trace_id = trace_id or new_trace_id()
    db.execute("INSERT INTO diagnostic_trace_links VALUES(?,?,?,?,?)",
               (actor, kind, operation_id, trace_id, utc_now(clock)))
    return trace_id


def trace_for_operation(store, actor, kind, operation_id):
    with store.db() as db:
        row = db.execute("SELECT trace_id FROM diagnostic_trace_links WHERE actor=? AND operation_kind=? AND operation_id=?",
                         (actor, kind, operation_id)).fetchone()
    return row["trace_id"] if row else None


def events_for_trace(store, trace_id, limit=200):
    if type(limit) is not int or not 1 <= limit <= 1000:
        raise ValueError("DIAGNOSTIC_LIMIT_INVALID")
    with store.db() as db:
        return [dict(row) for row in db.execute(
            "SELECT * FROM diagnostic_events WHERE trace_id=? ORDER BY seq LIMIT ?", (trace_id, limit))]


def prune(store, retention=RETENTION_SECONDS, cap=MAX_EVENTS, clock=time.time):
    """Bounded maintenance of owned diagnostics only; business rows are untouched."""
    with store.db() as db:
        db.execute("BEGIN IMMEDIATE")
        db.execute("DELETE FROM diagnostic_events WHERE time_utc<?",
                   (utc_now(clock=lambda: clock() - retention),))
        db.execute("DELETE FROM diagnostic_events WHERE seq IN (SELECT seq FROM diagnostic_events "
                   "ORDER BY seq DESC LIMIT -1 OFFSET ?)", (cap,))
        for row in db.execute("SELECT trace_id FROM diagnostic_events GROUP BY trace_id "
                              "HAVING count(*)>?", (MAX_EVENTS_PER_TRACE,)).fetchall():
            db.execute("DELETE FROM diagnostic_events WHERE trace_id=? AND seq NOT IN "
                       "(SELECT seq FROM diagnostic_events WHERE trace_id=? ORDER BY seq DESC LIMIT ?)",
                       (row["trace_id"], row["trace_id"], MAX_EVENTS_PER_TRACE))


def emit(store, clock=time.time, **fields):
    """Safe_emit: own bounded transaction; any failure is a silent capture gap.

    Never raises, never writes to stdout/stderr (those may be MCP protocol
    streams), and never blocks or retries a user operation.
    """
    try:
        event = build(clock=clock, **fields)
    except (ValueError, TypeError):
        return None
    try:
        with store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            total = db.execute("SELECT count(*) FROM diagnostic_events").fetchone()[0]
            if total >= MAX_EVENTS:
                db.execute("DELETE FROM diagnostic_events WHERE seq IN "
                           "(SELECT seq FROM diagnostic_events ORDER BY seq LIMIT ?)",
                           (total - MAX_EVENTS + 1,))
            per_trace = db.execute("SELECT count(*) FROM diagnostic_events WHERE trace_id=?",
                                   (event.trace_id,)).fetchone()[0]
            if per_trace >= MAX_EVENTS_PER_TRACE:
                return None
            row = asdict(event)
            db.execute("INSERT INTO diagnostic_events "
                       "(event_id,trace_id,span_id,parent_span_id,actor_scope,operation_kind,"
                       "operation_id,time_utc,elapsed_ms,component,phase,event_type,outcome,code,"
                       "capture,target_ref) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                       tuple(row.values()))
    except (sqlite3.Error, ValueError, Fault):
        return None
    return event


# ---------------------------------------------------------------------------
# App-owned read-only report reader (D2). Runs as the application user; the
# root-owned host CLI invokes it with fixed arguments and a byte budget.
# It never instantiates Store(), never migrates, and never writes state.

RECEIPT_SOURCES = {
    "hello": ("operations", ("id", "project", "state", "code", "created", "deadline"),
              "SELECT {fields} FROM operations WHERE id=? AND actor=?"),
    "environment": ("environment_operations",
                    ("id", "actor", "action", "logical", "state", "phase", "code", "created",
                     "deadline", "acknowledged", "evidence", "expected_state"),
                    "SELECT {fields} FROM environment_operations WHERE id=? AND actor=?"),
    "access": ("access_operations", ("id", "actor", "state", "code", "created", "deadline"),
               "SELECT {fields} FROM access_operations WHERE id=? AND actor=?"),
}
# Deliberately excluded from every receipt export: spec, target, result detail,
# payload, ack response, workspace paths - configuration and document content
# never enter a report; the omissions manifest says so explicitly.
MAX_REPORT_EVENTS = 1000
ID_PATTERN_BROAD = re.compile(r"[A-Za-z0-9][A-Za-z0-9_-]{0,127}\Z")


def _read_only_connection(database):
    """Read-only, query-only SQLite connection to the configured state; fail closed."""
    path = Path(database)
    info = path.lstat()
    if path.is_symlink() or not stat.S_ISREG(info.st_mode):
        raise Fault("UNSAFE_STATE_PATH", 422)
    if info.st_uid != os.getuid() or info.st_nlink != 1 or info.st_mode & 0o077:
        raise Fault("STATE_FILE_NOT_PRIVATE", 422)
    connection = sqlite3.connect("file:" + str(path) + "?mode=ro", uri=True, timeout=5)
    connection.row_factory = sqlite3.Row
    connection.execute("PRAGMA query_only=ON")
    return connection


def _state_database(config_path):
    config = json.loads(Path(config_path).read_text(encoding="utf-8"))
    database = config.get("database")
    if not isinstance(database, str) or not database.startswith("/") or len(database) > 4096:
        raise Fault("CONFIG_INVALID", 422)
    return database


def read_report(config_path, selector):
    """Return one typed report payload, or a typed refusal; read-only throughout."""
    kind = selector.get("kind")
    payload = {"schema_version": SCHEMA_VERSION, "reader": "app-owned-diagnostics",
               "found": False, "events": [], "omissions": []}
    database = _state_database(config_path)
    if kind == "operation":
        operation_kind = selector.get("operation_kind")
        operation_id = selector.get("operation_id")
        actor = selector.get("actor")
        if operation_kind not in RECEIPT_SOURCES or not isinstance(operation_id, str) \
                or not ID_PATTERN_BROAD.fullmatch(operation_id) or not isinstance(actor, str) \
                or not SCOPE_PATTERN.fullmatch(actor or ""):
            raise Fault("INVALID_SELECTOR", 422)
        table, fields, query = RECEIPT_SOURCES[operation_kind]
        with contextlib.closing(_read_only_connection(database)) as connection:
            row = connection.execute(query.format(fields=",".join(fields)), (operation_id, actor)).fetchone()
            if row is None:
                payload["omissions"].append("receipt_not_found_for_actor")
                return payload
            payload["found"] = True
            payload["receipt"] = {field: row[field] for field in fields}
            link = connection.execute(
                "SELECT trace_id FROM diagnostic_trace_links WHERE actor=? AND operation_kind=? AND operation_id=?",
                (actor, operation_kind, operation_id)).fetchone()
            if link is None:
                payload["omissions"].append("trace_unavailable_pre_migration")
            else:
                payload["trace_id"] = link["trace_id"]
                payload["events"] = [dict(item) for item in connection.execute(
                    "SELECT * FROM diagnostic_events WHERE trace_id=? ORDER BY seq LIMIT ?",
                    (link["trace_id"], MAX_REPORT_EVENTS + 1))]
            payload["omissions"].append("receipt_detail_fields_not_exported")
    elif kind == "trace":
        trace_id = selector.get("trace_id")
        if not isinstance(trace_id, str) or not ID_PATTERN.fullmatch(trace_id):
            raise Fault("INVALID_SELECTOR", 422)
        with contextlib.closing(_read_only_connection(database)) as connection:
            link = connection.execute(
                "SELECT actor,operation_kind,operation_id FROM diagnostic_trace_links WHERE trace_id=?",
                (trace_id,)).fetchone()
            if link is None:
                payload["omissions"].append("trace_not_found")
                return payload
            payload["found"] = True
            payload["trace_id"] = trace_id
            payload["receipt"] = {"actor": link["actor"], "operation_kind": link["operation_kind"],
                                  "operation_id": link["operation_id"]}
            payload["events"] = [dict(item) for item in connection.execute(
                "SELECT * FROM diagnostic_events WHERE trace_id=? ORDER BY seq LIMIT ?",
                (trace_id, MAX_REPORT_EVENTS + 1))]
    elif kind == "window":
        since, until = selector.get("since"), selector.get("until")
        for value in (since, until):
            if not isinstance(value, str) or not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z", value):
                raise Fault("INVALID_SELECTOR", 422)
        with contextlib.closing(_read_only_connection(database)) as connection:
            payload["found"] = True
            payload["events"] = [dict(item) for item in connection.execute(
                "SELECT * FROM diagnostic_events WHERE time_utc>=? AND time_utc<=? ORDER BY seq LIMIT ?",
                (since, until, MAX_REPORT_EVENTS + 1))]
        payload["omissions"].append("window_receipts_not_joined")
    else:
        raise Fault("INVALID_SELECTOR", 422)
    if len(payload["events"]) > MAX_REPORT_EVENTS:
        payload["events"] = payload["events"][:MAX_REPORT_EVENTS]
        payload["omissions"].append("events_truncated_at_limit")
    payload["event_count"] = len(payload["events"])
    return payload


def main_report():
    parser = argparse.ArgumentParser(description="App-owned read-only diagnostic report reader")
    parser.add_argument("--config", required=True)
    parser.add_argument("--selector", required=True, help="JSON selector, host-validated")
    arguments = parser.parse_args()
    try:
        selector = json.loads(arguments.selector)
        if not isinstance(selector, dict):
            raise ValueError
        print(json.dumps(read_report(arguments.config, selector), sort_keys=True))
        return 0
    except (Fault, OSError, ValueError, KeyError, sqlite3.Error) as error:
        print(json.dumps({"state": "reader_failed",
                          "code": error.code if isinstance(error, Fault) else "READER_UNAVAILABLE"}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main_report())
