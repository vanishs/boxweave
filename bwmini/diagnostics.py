"""Typed, privacy-safe diagnostic evidence; allowlisted fields only, never message content.

Diagnostic events are secondary observations: writing them must never change
operation semantics, and reading them never bypasses actor scoping. No freeform
detail field exists by design - a generic regex scrubber is not a redaction
policy. Trace and event identifiers are unpredictable opaque correlation
labels, not authority tokens.
"""
import re
import secrets
import sqlite3
import time
from dataclasses import dataclass, asdict
from datetime import datetime, timezone

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
