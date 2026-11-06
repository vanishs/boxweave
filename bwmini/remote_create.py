"""Durable, one-dispatch HTTP MCP create receipts.

The existing MCP gateway remains the only Provider writer. An accepted name
is consumed forever until an operator explicitly chooses a different name:
neither HTTP retries nor process restarts can replay an ambiguous create.
Only bounded metadata and SHA256 of the arguments are retained, never keys
or the upstream tool response.
"""
import hashlib
import json
import re
import secrets
import time

NAME = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
KEY = re.compile(r"[0-9a-f]{32}\Z")
TASK_LIMIT = 10000

SCHEMA = """
CREATE TABLE remote_create_tasks(
 task_id TEXT PRIMARY KEY,
 request_key TEXT NOT NULL UNIQUE,
 logical TEXT NOT NULL UNIQUE,
 payload_sha256 TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('submitted','succeeded','unknown')),
 code TEXT NOT NULL,
 created REAL NOT NULL,
 updated REAL NOT NULL
);
CREATE INDEX remote_create_tasks_updated ON remote_create_tasks(updated);
"""


class CreateFault(Exception):
    def __init__(self, code):
        self.code = code
        super().__init__(code)


def validated_request(args):
    """Accept the upstream schema without treating a client key as a tool arg."""
    if not isinstance(args, dict):
        raise CreateFault("CREATE_ARGUMENTS_INVALID")
    logical = args.get("username")
    if not isinstance(logical, str) or NAME.fullmatch(logical) is None:
        raise CreateFault("CREATE_NAME_INVALID")
    key = args.get("request_key")
    if key is not None and (not isinstance(key, str) or KEY.fullmatch(key) is None):
        raise CreateFault("CREATE_REQUEST_KEY_INVALID")
    payload = {k: v for k, v in args.items() if k != "request_key"}
    try:
        serialized = json.dumps(payload, sort_keys=True, separators=(",", ":"),
                                ensure_ascii=True, allow_nan=False)
    except (TypeError, ValueError, OverflowError):
        raise CreateFault("CREATE_ARGUMENTS_INVALID") from None
    if len(serialized) > 32768:
        raise CreateFault("CREATE_ARGUMENTS_TOO_LARGE")
    digest = hashlib.sha256(serialized.encode("ascii")).hexdigest()
    return logical, key, payload, digest


def task_receipt(row):
    state = row["state"]
    result = {
        "task_id": row["task_id"],
        "request_key": row["request_key"],
        "username": row["logical"],
        "state": state,
        "code": row["code"],
        "upstream_acknowledged": state == "succeeded",
        "guest_user_and_ssh_verified": False,
    }
    if state == "submitted":
        result["next"] = "Poll boxweave_create_status; never resend a create."
    elif state == "succeeded":
        result["next"] = "Read get_container, then independently check guest SSH readiness."
    else:
        result["next"] = ("Outcome unknown. Read get_container and debug_container; "
                          "do not repeat create, delete, or adopt automatically.")
    return result


class CreateJournal:
    def __init__(self, store, clock=time.time):
        self.store, self.clock = store, clock

    def recover_interrupted(self):
        # The process may have died after dispatch; never replay old intent.
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_create_tasks SET state='unknown', "
                       "code='PROCESS_RESTART_UNKNOWN', updated=? "
                       "WHERE state='submitted'", (self.clock(),))

    def reserve(self, logical, request_key, digest):
        if not isinstance(digest, str) or not re.fullmatch(r"[a-f0-9]{64}", digest):
            raise CreateFault("CREATE_ARGUMENTS_INVALID")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            by_name = db.execute("SELECT * FROM remote_create_tasks WHERE logical=?",
                                 (logical,)).fetchone()
            by_key = (db.execute("SELECT * FROM remote_create_tasks WHERE request_key=?",
                                 (request_key,)).fetchone()
                      if request_key is not None else None)
            if by_key is not None and (by_key["logical"] != logical
                                       or by_key["payload_sha256"] != digest):
                raise CreateFault("CREATE_REQUEST_KEY_CONFLICT")
            if by_name is not None:
                if by_name["payload_sha256"] != digest:
                    raise CreateFault("CREATE_NAME_CONFLICT")
                # A ChatGPT retry can regenerate its request_key. The exact
                # name + payload still resolves to the ORIGINAL invocation.
                return task_receipt(by_name), False
            if by_key is not None:
                raise CreateFault("CREATE_REQUEST_KEY_CONFLICT")
            if db.execute("SELECT count(*) FROM remote_create_tasks").fetchone()[0] >= TASK_LIMIT:
                raise CreateFault("CREATE_JOURNAL_FULL")
            task_id = secrets.token_hex(16)
            key = request_key or secrets.token_hex(16)
            now = self.clock()
            db.execute("INSERT INTO remote_create_tasks "
                       "(task_id, request_key, logical, payload_sha256, state, "
                       "code, created, updated) VALUES(?,?,?,?,?,?,?,?)",
                       (task_id, key, logical, digest, "submitted",
                        "DISPATCH_INTENT_RECORDED", now, now))
            row = db.execute("SELECT * FROM remote_create_tasks WHERE task_id=?",
                             (task_id,)).fetchone()
            return task_receipt(row), True

    def complete(self, task_id, succeeded):
        # A timeout, server restart, or journal race cannot become a success.
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_create_tasks SET state=?, code=?, updated=? "
                       "WHERE task_id=? AND state='submitted'",
                       ("succeeded" if succeeded else "unknown",
                        "UPSTREAM_CREATE_ACKNOWLEDGED" if succeeded
                        else "UPSTREAM_CREATE_OUTCOME_UNKNOWN",
                        self.clock(), task_id))

    def get(self, *, task_id=None, request_key=None, username=None):
        keys = [(name, value) for name, value in
                (("task_id", task_id), ("request_key", request_key),
                 ("logical", username)) if value is not None]
        if len(keys) != 1:
            raise CreateFault("CREATE_SELECTOR_INVALID")
        field, value = keys[0]
        if (not isinstance(value, str) or
                (NAME if field == "logical" else KEY).fullmatch(value) is None):
            raise CreateFault("CREATE_SELECTOR_INVALID")
        with self.store.db() as db:
            row = db.execute("SELECT * FROM remote_create_tasks WHERE " +
                             field + "=?", (value,)).fetchone()
        if row is None:
            raise CreateFault("CREATE_TASK_NOT_FOUND")
        return task_receipt(row)
