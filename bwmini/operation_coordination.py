"""Small, durable resource-claim core for the unified BoxWeave operation boundary.

No Provider calls, permissions, network access, task outputs, or application
receipt databases live here. The host integration must supply a separately
authorized connection factory and write receipts before dispatch.
"""
from contextlib import contextmanager
import hashlib
import json
import hmac
import re
import secrets
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
CREATE TABLE IF NOT EXISTS coordination_leases (
    resource TEXT PRIMARY KEY,
    identity_digest TEXT NOT NULL,
    token_digest TEXT NOT NULL,
    generation INTEGER NOT NULL,
    expires REAL NOT NULL,
    updated REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS coordination_resources_operation
    ON coordination_resources(operation_id);
CREATE TABLE IF NOT EXISTS coordination_recovery_reviews (
    operation_id TEXT PRIMARY KEY REFERENCES coordination_operations(operation_id),
    reviewer TEXT NOT NULL,
    token_digest TEXT NOT NULL,
    snapshot_digest TEXT NOT NULL,
    expires REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS coordination_recovery_audit (
    event_id TEXT PRIMARY KEY,
    at REAL NOT NULL,
    operation_id TEXT NOT NULL REFERENCES coordination_operations(operation_id),
    reviewer TEXT NOT NULL,
    reason TEXT NOT NULL,
    original_state TEXT NOT NULL,
    identity_digest TEXT NOT NULL,
    receipt_ref TEXT NOT NULL,
    resources_json TEXT NOT NULL,
    decision TEXT NOT NULL CHECK (decision = 'forced_unknown_release')
);
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
            # Append-only at the application SQL interface. Filesystem root
            # remains the ultimate administrator; this is not a WORM store.
            db.execute("CREATE TRIGGER IF NOT EXISTS recovery_audit_no_update "
                       "BEFORE UPDATE ON coordination_recovery_audit BEGIN "
                       "SELECT RAISE(ABORT, 'RECOVERY_AUDIT_IMMUTABLE'); END")
            db.execute("CREATE TRIGGER IF NOT EXISTS recovery_audit_no_delete "
                       "BEFORE DELETE ON coordination_recovery_audit BEGIN "
                       "SELECT RAISE(ABORT, 'RECOVERY_AUDIT_IMMUTABLE'); END")

    def acquire(self, operation_id, identity_digest, receipt_ref, resources, lease_token=None):
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
                lease = db.execute("SELECT identity_digest,token_digest,expires FROM coordination_leases WHERE resource=?",
                                   (key,)).fetchone()
                if lease and lease[2] > self.clock():
                    if (lease[0] != identity_digest or
                            not isinstance(lease_token,str) or
                            not hmac.compare_digest(lease[1], hashlib.sha256(
                                lease_token.encode("utf-8")).hexdigest())):
                        raise CoordinationFault("RESOURCE_BUSY")
                owner = db.execute(
                    "SELECT o.state FROM coordination_resources r "
                    "JOIN coordination_operations o ON o.operation_id=r.operation_id "
                    "WHERE r.resource=?", (key,)).fetchone()
                if owner:
                    # A dispatch_intent is already an uncertain remote write:
                    # it may still be running or its worker may have died.
                    # The durable task receipt also presents this as UNKNOWN.
                    # Require original-outcome reconciliation in either case;
                    # never release claims or replay the mutation here.
                    raise CoordinationFault(
                        "RECOVERY_REQUIRED" if owner[0] in ("dispatch_intent", "unknown")
                        else "RESOURCE_BUSY")
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

    def lease(self, action, resource, identity_digest, token=None, seconds=600):
        """Optional 10-minute operator lease; never removes in-flight claims."""
        key = _resource(resource)
        _digest(identity_digest)
        if key != box_resource(identity_digest):
            raise CoordinationFault("LEASE_ARGUMENT_INVALID")
        if (action not in ("acquire","renew","release","status") or
                type(seconds) is not int or not 1 <= seconds <= 600):
            raise CoordinationFault("LEASE_ARGUMENT_INVALID")
        if action in ("renew","release"):
            if not isinstance(token,str) or re.fullmatch(r"[A-Za-z0-9_-]{43}",token) is None:
                raise CoordinationFault("LEASE_TOKEN_INVALID")
        with self._transaction() as db:
            now=self.clock()
            old=db.execute("SELECT identity_digest,token_digest,generation,expires "
                           "FROM coordination_leases WHERE resource=?",(key,)).fetchone()
            if action=="status":
                return {"state":"leased" if old and old[3]>now else "free",
                        "expires":old[3] if old and old[3]>now else None}
            if action=="acquire":
                if old and old[3]>now:
                    raise CoordinationFault("RESOURCE_BUSY")
                # Expiry of the lease does not release any separate
                # unknown or dispatching resource claim.
                claimed=db.execute("SELECT 1 FROM coordination_resources WHERE resource=?",
                                   (key,)).fetchone()
                if claimed:
                    raise CoordinationFault("RESOURCE_BUSY")
                new_token=secrets.token_urlsafe(32)
                generation=(old[2]+1) if old else 1
                expires=now+seconds
                db.execute("INSERT OR REPLACE INTO coordination_leases VALUES(?,?,?,?,?,?)",
                           (key,identity_digest,hashlib.sha256(new_token.encode()).hexdigest(),
                            generation,expires,now))
                return {"state":"leased","lease_token":new_token,
                        "generation":generation,"expires":expires}
            if old is None or old[3]<=now:
                raise CoordinationFault("LEASE_EXPIRED")
            if (old[0]!=identity_digest or not hmac.compare_digest(
                    old[1],hashlib.sha256(token.encode()).hexdigest())):
                raise CoordinationFault("LEASE_TOKEN_INVALID")
            if action=="release":
                db.execute("DELETE FROM coordination_leases WHERE resource=?",(key,))
                return {"state":"released","generation":old[2]}
            expires=now+seconds
            db.execute("UPDATE coordination_leases SET expires=?,updated=? WHERE resource=?",
                       (expires,now,key))
            return {"state":"leased","lease_token":token,
                    "generation":old[2],"expires":expires}

    @staticmethod
    def _reviewer(value):
        if not isinstance(value, str) or re.fullmatch(r"[A-Za-z0-9_][A-Za-z0-9_.@-]{0,63}", value) is None:
            raise CoordinationFault("RECOVERY_REVIEWER_INVALID")
        return value

    @staticmethod
    def _snapshot(db, operation_id):
        op = db.execute(
            "SELECT identity_digest,receipt_ref,state FROM coordination_operations WHERE operation_id=?",
            (operation_id,)).fetchone()
        if op is None:
            raise CoordinationFault("OPERATION_NOT_FOUND")
        keys = [r[0] for r in db.execute(
            "SELECT resource FROM coordination_resources WHERE operation_id=? ORDER BY resource", (operation_id,))]
        if op[2] not in ("unknown", "dispatch_intent") or not keys:
            raise CoordinationFault("RECOVERY_CLAIM_NOT_UNKNOWN")
        value = {"operation_id": operation_id, "identity_digest": op[0],
                 "receipt_ref": op[1], "state": op[2], "resources": keys}
        fingerprint = hashlib.sha256(json.dumps(value, sort_keys=True,
                separators=(",", ":")).encode()).hexdigest()
        return value, fingerprint

    def recovery_review(self, operation_id, reviewer):
        """One-use, short-lived challenge bound to exact UNKNOWN claims.

        Does not infer remote completion, modify original receipts or release.
        Only a privileged host CLI may expose this method to an operator.
        """
        _operation_id(operation_id)
        reviewer = self._reviewer(reviewer)
        with self._transaction() as db:
            value, fingerprint = self._snapshot(db, operation_id)
            challenge = secrets.token_urlsafe(32)
            expires = self.clock() + 300
            db.execute("INSERT OR REPLACE INTO coordination_recovery_reviews "
                       "VALUES(?,?,?,?,?)", (operation_id, reviewer,
                        hashlib.sha256(challenge.encode()).hexdigest(),fingerprint,expires))
            return {**value, "reviewer": reviewer,
                    "expires": expires, "challenge": challenge,
                    "warning": "ORIGINAL_REMOTE_OUTCOME_UNPROVEN;NO_RETRY_OR_CANCELLATION"}

    def recovery_force_release(self, operation_id, reviewer, challenge, reason,
                               confirmation, acknowledged=False):
        """Release only an exact reviewed UNKNOWN claim with atomic audit.

        Never mark the original receipt successful, run a command, retry or
        stop a guest process. Audit append and resource deletion commit once.
        """
        _operation_id(operation_id)
        reviewer = self._reviewer(reviewer)
        if (not isinstance(challenge,str) or re.fullmatch(r"[A-Za-z0-9_-]{43}",challenge) is None
                or not isinstance(reason,str) or not 12 <= len(reason) <= 240
                or any(ord(c) < 32 or ord(c) == 127 for c in reason)):
            raise CoordinationFault("RECOVERY_ARGUMENT_INVALID")
        if acknowledged is not True or confirmation != "RELEASE " + operation_id:
            raise CoordinationFault("RECOVERY_CONFIRMATION_REQUIRED")
        with self._transaction() as db:
            value, fingerprint = self._snapshot(db, operation_id)
            row = db.execute("SELECT reviewer,token_digest,snapshot_digest,expires "
                             "FROM coordination_recovery_reviews WHERE operation_id=?",
                             (operation_id,)).fetchone()
            if (row is None or row[0] != reviewer or row[3] <= self.clock()
                    or not hmac.compare_digest(row[1],hashlib.sha256(challenge.encode()).hexdigest())
                    or not hmac.compare_digest(row[2],fingerprint)):
                raise CoordinationFault("RECOVERY_REVIEW_EXPIRED_OR_CHANGED")
            event_id = secrets.token_hex(16)
            db.execute("INSERT INTO coordination_recovery_audit VALUES(?,?,?,?,?,?,?,?,?,?)",
                       (event_id,self.clock(),operation_id,reviewer,reason,value["state"],
                        value["identity_digest"],value["receipt_ref"],
                        json.dumps(value["resources"],separators=(",", ":")),"forced_unknown_release"))
            db.execute("DELETE FROM coordination_resources WHERE operation_id=?",(operation_id,))
            db.execute("DELETE FROM coordination_recovery_reviews WHERE operation_id=?",(operation_id,))
            return {"state":"released_with_unknown_outcome","operation_id":operation_id,
                    "audit_id":event_id,"released_resources":len(value["resources"]),
                    "original_receipt_unchanged":True,"remote_cancellation":False}

    def recovery_audit(self, operation_id):
        _operation_id(operation_id)
        with self._transaction() as db:
            return [dict(zip(("event_id","at","reviewer","reason","original_state",
                              "identity_digest","receipt_ref","resources","decision"),
                             (r[0],r[1],r[2],r[3],r[4],r[5],r[6],json.loads(r[7]),r[8])))
                    for r in db.execute("SELECT event_id,at,reviewer,reason,original_state,"
                                        "identity_digest,receipt_ref,resources_json,decision "
                                        "FROM coordination_recovery_audit WHERE operation_id=? ORDER BY at",
                                        (operation_id,))]

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
