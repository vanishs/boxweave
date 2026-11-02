"""Small, durable resource-claim core for the unified BoxWeave operation boundary.

No Provider calls, permissions, network access, task outputs, or application
receipt databases live here. The host integration must supply a separately
authorized connection factory and write receipts before dispatch.
"""
from contextlib import contextmanager
import re
import sqlite3
import time


class CoordinationFault(Exception):
    """Non-secret, typed refusal. Never contains original commands or tokens."""

    def __init__(self, code):
        self.code = code
        super().__init__(code)


_SCHEMA = """
CREATE TABLE IF NOT EXISTS coordination_version (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    version INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS coordination_operations (
    operation_id TEXT PRIMARY KEY,
    identity_digest TEXT NOT NULL,
    receipt_ref TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN (
        'accepted', 'dispatch_intent', 'unknown', 'observed_terminal',
        'cancelled_before_dispatch')),
    created REAL NOT NULL,
    updated REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS coordination_resources (
    resource TEXT PRIMARY KEY,
    operation_id TEXT NOT NULL REFERENCES coordination_operations(operation_id)
);
CREATE INDEX IF NOT EXISTS coordination_resources_operation
    ON coordination_resources(operation_id);
"""


def _digest(value):
    if not isinstance(value, str) or re.fullmatch(r"[0-9a-f]{64}", value) is None:
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    return value


def _logical(value):
    if not isinstance(value, str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}", value) is None:
        raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
    return value


def box_resource(identity_digest):
    """Exact immutable Box incarnation, not a reusable logical name."""
    return "box:" + _digest(identity_digest)


def name_resource(scope_digest, logical):
    """Names must also be claimed across Box creation/recreation."""
    return "name:" + _digest(scope_digest) + ":" + _logical(logical)


def tenant_resource(scope_digest, tenant):
    return "tenant:" + _digest(scope_digest) + ":" + _logical(tenant)


def port_resource(scope_digest, protocol, external_port):
    if protocol not in ("tcp", "udp") or type(external_port) is not int or not 1 <= external_port <= 65535:
        raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
    return "port:" + _digest(scope_digest) + ":" + protocol + ":" + str(external_port)


def _resource(value):
    if not isinstance(value, str) or len(value) > 160:
        raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
    parts = value.split(":")
    if len(parts) == 2 and parts[0] == "box":
        _digest(parts[1])
    elif len(parts) == 3 and parts[0] in ("name", "tenant"):
        _digest(parts[1])
        _logical(parts[2])
    elif len(parts) == 4 and parts[0] == "port":
        if re.fullmatch(r"[1-9][0-9]{0,4}", parts[3]) is None:
            raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
        port_resource(parts[1], parts[2], int(parts[3]))
    else:
        raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
    return value


def _operation_id(value):
    if not isinstance(value, str) or re.fullmatch(r"[0-9a-f]{32}", value) is None:
        raise CoordinationFault("COORDINATION_OPERATION_INVALID")
    return value


def _receipt(value):
    if not isinstance(value, str) or re.fullmatch(r"[A-Za-z0-9_-]{1,80}", value) is None:
        raise CoordinationFault("COORDINATION_RECEIPT_INVALID")
    return value


class OperationCoordinator:
    """One authoritative claim store shared only via reviewed host permissions.

    Each method commits before returning; callers MUST NOT hold a transaction
    over a remote call. No method executes, retries or cancels guest work.
    """

    def __init__(self, connect, clock=time.time):
        self.connect = connect
        self.clock = clock

    @contextmanager
    def _transaction(self):
        db = None
        try:
            db = self.connect()
            db.execute("PRAGMA foreign_keys=ON")
            db.execute("BEGIN IMMEDIATE")
            yield db
            db.commit()
        except (sqlite3.Error, OSError) as exc:
            if db is not None:
                db.rollback()
            raise CoordinationFault("COORDINATION_UNAVAILABLE") from exc
        except BaseException:
            if db is not None:
                db.rollback()
            raise
        finally:
            if db is not None:
                db.close()

    def initialize(self):
        """Run only from the explicit trusted coordinator provisioning path."""
        with self._transaction() as db:
            for statement in _SCHEMA.split(";"):
                if statement.strip():
                    db.execute(statement)
            row = db.execute("SELECT version FROM coordination_version WHERE singleton=1").fetchone()
            if row is None:
                db.execute("INSERT INTO coordination_version VALUES (1, 1)")
            elif row[0] != 1:
                raise CoordinationFault("COORDINATION_SCHEMA_UNSUPPORTED")

    def acquire(self, operation_id, identity_digest, receipt_ref, resources):
        _operation_id(operation_id)
        _digest(identity_digest)
        _receipt(receipt_ref)
        if not isinstance(resources, (list, tuple)) or not 1 <= len(resources) <= 8:
            raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
        keys = sorted({_resource(r) for r in resources})
        if len(keys) != len(resources):
            raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
        with self._transaction() as db:
            if db.execute("SELECT 1 FROM coordination_operations WHERE operation_id=?",
                          (operation_id,)).fetchone():
                raise CoordinationFault("OPERATION_EXISTS")
            for key in keys:
                owner = db.execute(
                    "SELECT o.state FROM coordination_resources r "
                    "JOIN coordination_operations o ON o.operation_id=r.operation_id "
                    "WHERE r.resource=?", (key,)).fetchone()
                if owner:
                    raise CoordinationFault(
                        "RECOVERY_REQUIRED" if owner[0] == "unknown" else "RESOURCE_BUSY")
            now = self.clock()
            db.execute("INSERT INTO coordination_operations VALUES (?,?,?,?,?,?)",
                       (operation_id, identity_digest, receipt_ref, "accepted", now, now))
            db.executemany("INSERT INTO coordination_resources VALUES (?,?)",
                           [(key, operation_id) for key in keys])

    def _transition(self, operation_id, previous, state, release=False, receipt_ref=None):
        _operation_id(operation_id)
        if receipt_ref is not None:
            _receipt(receipt_ref)
        with self._transaction() as db:
            row = db.execute(
                "SELECT state,receipt_ref FROM coordination_operations WHERE operation_id=?",
                (operation_id,)).fetchone()
            if row is None:
                raise CoordinationFault("OPERATION_NOT_FOUND")
            if row[0] not in previous or (receipt_ref is not None and row[1] != receipt_ref):
                raise CoordinationFault("OPERATION_STATE_CONFLICT")
            db.execute("UPDATE coordination_operations SET state=?,updated=? WHERE operation_id=?",
                       (state, self.clock(), operation_id))
            if release:
                db.execute("DELETE FROM coordination_resources WHERE operation_id=?", (operation_id,))

    def dispatch_intent(self, operation_id):
        """Commit BEFORE attempting the first remote side effect."""
        self._transition(operation_id, ("accepted",), "dispatch_intent")

    def unknown(self, operation_id):
        """Conservative uncertainty; resources remain claimed indefinitely."""
        self._transition(operation_id, ("dispatch_intent", "unknown"), "unknown")

    def cancel_before_dispatch(self, operation_id):
        """Never use for operations with a durable dispatch intent."""
        self._transition(operation_id, ("accepted",), "cancelled_before_dispatch", release=True)

    def observed_terminal(self, operation_id, receipt_ref):
        """Only after original receipt/target evidence was durably recorded."""
        self._transition(operation_id, ("dispatch_intent", "unknown"), "observed_terminal",
                         release=True, receipt_ref=receipt_ref)

    def recover_dispatched(self):
        """After a verified gateway/worker restart; NEVER re-dispatch."""
        with self._transaction() as db:
            return db.execute(
                "UPDATE coordination_operations SET state='unknown',updated=? "
                "WHERE state='dispatch_intent'", (self.clock(),)).rowcount

    def status(self, operation_id):
        _operation_id(operation_id)
        with self._transaction() as db:
            row = db.execute(
                "SELECT identity_digest,receipt_ref,state FROM coordination_operations "
                "WHERE operation_id=?", (operation_id,)).fetchone()
            if row is None:
                raise CoordinationFault("OPERATION_NOT_FOUND")
            resources = db.execute(
                "SELECT resource FROM coordination_resources WHERE operation_id=? ORDER BY resource",
                (operation_id,)).fetchall()
            return {"identity_digest": row[0], "receipt_ref": row[1],
                    "state": row[2], "resources": [r[0] for r in resources]}
