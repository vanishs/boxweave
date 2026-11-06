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
CREATE TABLE IF NOT EXISTS remote_create_tasks(
 task_id TEXT PRIMARY KEY,
 request_key TEXT NOT NULL UNIQUE,
 logical TEXT NOT NULL UNIQUE,
 payload_sha256 TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('submitted','succeeded','unknown')),
 code TEXT NOT NULL,
 created REAL NOT NULL,
 updated REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS remote_create_tasks_updated ON remote_create_tasks(updated);
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



def validate_upstream_payload(payload, schema):
    """Check the advertised upstream JSON-Schema subset before reserving a name.

    This deliberately does NOT claim all Containarium semantic constraints.
    Unrecognized schema directives remain upstream-owned, while recognized
    types, required fields, enums and bounds fail before a durable intent.
    """
    if (not isinstance(schema, dict) or schema.get("type") != "object"
            or not isinstance(schema.get("properties"), dict)
            or not isinstance(payload, dict)):
        raise CreateFault("CREATE_SCHEMA_UNAVAILABLE")
    properties = schema["properties"]
    if not isinstance(properties.get("username"), dict):
        raise CreateFault("CREATE_SCHEMA_UNAVAILABLE")
    required = schema.get("required", [])
    if (not isinstance(required, list)
            or any(not isinstance(name, str) for name in required)):
        raise CreateFault("CREATE_SCHEMA_UNAVAILABLE")
    if any(name not in payload for name in required):
        raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
    if schema.get("additionalProperties") is False and any(
            name not in properties for name in payload):
        raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")

    def check(value, definition, depth=0):
        if not isinstance(definition, dict) or depth > 5:
            raise CreateFault("CREATE_SCHEMA_UNAVAILABLE")
        kind = definition.get("type")
        types = (kind,) if isinstance(kind, str) else kind
        if not isinstance(types, (list, tuple)) or not types:
            raise CreateFault("CREATE_SCHEMA_UNAVAILABLE")
        validators = {
            "string": lambda x: isinstance(x, str),
            "boolean": lambda x: isinstance(x, bool),
            "integer": lambda x: type(x) is int,
            "number": lambda x: type(x) in (int, float),
            "array": lambda x: isinstance(x, list),
            "object": lambda x: isinstance(x, dict),
            "null": lambda x: x is None,
        }
        if any(k not in validators for k in types):
            raise CreateFault("CREATE_SCHEMA_UNAVAILABLE")
        if not any(validators[k](value) for k in types):
            raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
        if "enum" in definition and value not in definition["enum"]:
            raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
        if "minimum" in definition and type(value) in (int, float) and value < definition["minimum"]:
            raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
        if "maximum" in definition and type(value) in (int, float) and value > definition["maximum"]:
            raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
        if isinstance(value, str):
            if "minLength" in definition and len(value) < definition["minLength"]:
                raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
            if "maxLength" in definition and len(value) > definition["maxLength"]:
                raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
            if "pattern" in definition:
                try:
                    valid = re.search(definition["pattern"], value) is not None
                except (TypeError, re.error):
                    raise CreateFault("CREATE_SCHEMA_UNAVAILABLE") from None
                if not valid:
                    raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
        if isinstance(value, list):
            if "minItems" in definition and len(value) < definition["minItems"]:
                raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
            if "maxItems" in definition and len(value) > definition["maxItems"]:
                raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
            if "items" in definition:
                for member in value:
                    check(member, definition["items"], depth + 1)
        if isinstance(value, dict):
            child_fields = definition.get("properties", {})
            if not isinstance(child_fields, dict):
                raise CreateFault("CREATE_SCHEMA_UNAVAILABLE")
            if definition.get("additionalProperties") is False and any(
                    name not in child_fields for name in value):
                raise CreateFault("CREATE_UPSTREAM_ARGUMENT_INVALID")
            for name, member in value.items():
                if name in child_fields:
                    check(member, child_fields[name], depth + 1)

    for name, value in payload.items():
        if name in properties:
            check(value, properties[name])


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
