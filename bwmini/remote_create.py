"""Durable, one-dispatch HTTP MCP create receipts.

The coordinated REST Gateway is the only migrated create writer. An accepted name
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
# These codes are emitted only after BoxWeave has proved zero Provider POST.
# Never infer this state from a timeout, loss of worker process or arbitrary
# upstream response. Frozen names remain non-replayable even in this state.
PREDISPATCH_FAILURE_CODES = frozenset((
    "REST_CREATE_PREFLIGHT_UNVERIFIED", "REST_RESOURCE_EXISTS",
    "REST_CREATE_PREDISPATCH_UNVERIFIED", "COORDINATION_SCOPE_UNVERIFIED",
    "REST_APP_CONFIG_UNVERIFIED", "REST_APP_COORDINATION_NOT_ALIGNED",
    "REST_STATE_DB_UNSAFE",
))


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


EXTRA_SCHEMA = """
CREATE TABLE IF NOT EXISTS remote_create_options(
 task_id TEXT PRIMARY KEY REFERENCES remote_create_tasks(task_id),
 requested_nesting INTEGER CHECK(requested_nesting IN (0,1)),
 postcondition TEXT NOT NULL DEFAULT 'not_started'
   CHECK(postcondition IN ('not_started','intent','configured','unknown')),
 instance_uuid TEXT,
 instance_created TEXT,
 last_probe_at REAL NOT NULL DEFAULT 0,
 source TEXT NOT NULL CHECK(source IN ('http','ssh','web'))
);
CREATE TABLE IF NOT EXISTS remote_rest_tasks(
 task_id TEXT PRIMARY KEY,
 request_key TEXT NOT NULL UNIQUE,
 logical TEXT NOT NULL,
 action TEXT NOT NULL CHECK(action IN ('start','stop','delete','resize')),
 payload_sha256 TEXT NOT NULL,
 identity_digest TEXT NOT NULL,
 instance_name TEXT NOT NULL,
 instance_created TEXT NOT NULL,
 spec_json TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('submitted','accepted','succeeded','unknown')),
 code TEXT NOT NULL,
 created REAL NOT NULL,
 updated REAL NOT NULL,
 last_probe_at REAL NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS remote_rest_tasks_poll
 ON remote_rest_tasks(state,last_probe_at);
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
    # This is a permanent, non-replayable intent: a typo in an
    # unadvertised field must not consume the logical Box name.
    # The live upstream tool catalogue is the allowlist, even if its
    # JSON Schema omits additionalProperties:false.
    if any(name not in properties for name in payload):
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
    # REST 'CREATING' is only an accepted intent, NEVER a ready/complete Box.
    # Retain the old SQLite CHECK constraint and migration compatibility.
    if state == "submitted" and row["code"] == "REST_CREATE_ACCEPTED":
        state = "accepted"
    elif state == "submitted" and row["code"] in PREDISPATCH_FAILURE_CODES:
        state = "failed_before_dispatch"
    result = {
        "task_id": row["task_id"],
        "request_key": row["request_key"],
        "username": row["logical"],
        "state": state,
        "code": row["code"],
        "upstream_acknowledged": state in ("accepted", "succeeded"),
        "guest_user_and_ssh_verified": (state == "succeeded"
                                         and row["code"] == "REST_GUEST_READY"),
    }
    if state in ("submitted", "accepted"):
        result["next"] = ("Poll boxweave_create_status; accepted is provisioning, "
                          "not guest ready. Never resend a create.")
    elif state == "succeeded":
        result["next"] = "Read get_container, then independently check guest SSH readiness."
    elif state == "failed_before_dispatch":
        result["next"] = ("No Provider POST was dispatched for this task. "
                          "The original name remains reserved; inspect or "
                          "resolve the failure without replaying the original request.")
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
                       "WHERE state='submitted' AND code!='REST_CREATE_ACCEPTED' "
                       "AND code NOT IN (" +
                       ",".join("?" for _ in PREDISPATCH_FAILURE_CODES) + ")",
                       (self.clock(), *sorted(PREDISPATCH_FAILURE_CODES)))

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
                        "REST_GATE_PENDING", now, now))
            row = db.execute("SELECT * FROM remote_create_tasks WHERE task_id=?",
                             (task_id,)).fetchone()
            return task_receipt(row), True

    def fail_before_dispatch(self, task_id, code, *, username=None, digest=None):
        """Persist proof that THIS request did not reach Provider REST POST.

        Only a still-pending original receipt is eligible. Any recorded
        dispatch intent/accepted/UNKNOWN or historical row is immutable.
        Never free its name or silently permit another create.
        """
        if code not in PREDISPATCH_FAILURE_CODES:
            raise CreateFault("CREATE_RESULT_INVALID")
        if not isinstance(task_id, str) or KEY.fullmatch(task_id) is None:
            raise CreateFault("CREATE_SELECTOR_INVALID")
        if digest is not None and (not isinstance(digest,str)
                or re.fullmatch(r"[0-9a-f]{64}",digest) is None):
            raise CreateFault("CREATE_ARGUMENTS_INVALID")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row=db.execute("SELECT logical,payload_sha256,state,code "
                           "FROM remote_create_tasks WHERE task_id=?",
                           (task_id,)).fetchone()
            if (row is None or (username is not None and row["logical"]!=username)
                    or (digest is not None and row["payload_sha256"]!=digest)
                    or row["state"]!="submitted" or row["code"]!="REST_GATE_PENDING"):
                return False
            db.execute("UPDATE remote_create_tasks SET code=?,updated=? "
                       "WHERE task_id=? AND state='submitted' "
                       "AND code='REST_GATE_PENDING'",
                       (code,self.clock(),task_id))
            return True

    def begin_dispatch(self, task_id, digest):
        """Atomically bind exactly one outbound invocation to this intent.

        Historical rows are NEVER eligible; an old submitted intent might
        already have reached upstream before a process failed. No replay.
        """
        if not isinstance(task_id, str) or KEY.fullmatch(task_id) is None:
            raise CreateFault("CREATE_SELECTOR_INVALID")
        if not isinstance(digest, str) or re.fullmatch(r"[a-f0-9]{64}", digest) is None:
            raise CreateFault("CREATE_ARGUMENTS_INVALID")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT state,code,payload_sha256 FROM "
                             "remote_create_tasks WHERE task_id=?", (task_id,)).fetchone()
            if row is None or row["payload_sha256"] != digest:
                raise CreateFault("CREATE_TASK_MISMATCH")
            if row["state"] != "submitted" or row["code"] != "REST_GATE_PENDING":
                return False
            db.execute("UPDATE remote_create_tasks SET code='REST_GATE_DISPATCH_INTENT', "
                       "updated=? WHERE task_id=? AND state='submitted' "
                       "AND code='REST_GATE_PENDING'", (self.clock(), task_id))
            return True

    def intent_options(self, task_id, *, nesting=None, source="http"):
        if nesting is not None and type(nesting) is not bool:
            raise CreateFault("NESTING_BOOLEAN_REQUIRED")
        if source not in ("http", "ssh", "web"):
            raise CreateFault("CREATE_SOURCE_INVALID")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            state = db.execute(
                "SELECT state,code FROM remote_create_tasks WHERE task_id=?",
                (task_id,)).fetchone()
            if state is None or state["state"] != "submitted":
                raise CreateFault("CREATE_TASK_NOT_DISPATCHABLE")
            existing = db.execute(
                "SELECT requested_nesting,source FROM remote_create_options "
                "WHERE task_id=?", (task_id,)).fetchone()
            setting = None if nesting is None else int(nesting)
            if existing is not None:
                if (existing["requested_nesting"] != setting or
                        existing["source"] != source):
                    raise CreateFault("CREATE_OPTIONS_CONFLICT")
                return
            db.execute(
                "INSERT INTO remote_create_options(task_id,requested_nesting,source) "
                "VALUES(?,?,?)", (task_id, setting, source))

    def accepted_observation(self, interval=15):
        # Persisted, cross-process read-only poll throttle. No worker can
        # accidentally multiply upstream calls after a restart.
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute(
                "SELECT t.task_id,t.logical,t.created,o.requested_nesting,"
                "o.postcondition,o.instance_uuid,o.instance_created "
                "FROM remote_create_tasks AS t JOIN remote_create_options AS o "
                "ON o.task_id=t.task_id WHERE t.state='submitted' AND "
                "t.code='REST_CREATE_ACCEPTED' AND o.last_probe_at<=? "
                "ORDER BY t.created,t.task_id LIMIT 1",
                (self.clock()-interval,)).fetchone()
            if row is not None:
                db.execute("UPDATE remote_create_options SET last_probe_at=? "
                           "WHERE task_id=?", (self.clock(), row["task_id"]))
        return dict(row) if row is not None else None

    def freeze(self, task_id, code="UPSTREAM_CREATE_OUTCOME_UNKNOWN"):
        if code not in ("UPSTREAM_CREATE_OUTCOME_UNKNOWN",
                        "NESTING_POSTCREATE_UNKNOWN",
                        "READINESS_OUTCOME_UNKNOWN"):
            raise CreateFault("CREATE_RESULT_INVALID")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_create_tasks SET state='unknown',code=?,"
                       "updated=? WHERE task_id=? AND state='submitted'",
                       (code, self.clock(), task_id))

    def mark_ready(self, task_id):
        """Only caller-supplied independent live guest proof may mark ready."""
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute(
                "SELECT requested_nesting,postcondition FROM remote_create_options "
                "WHERE task_id=?", (task_id,)).fetchone()
            if (row is None or (row["requested_nesting"] is not None
                                and row["postcondition"] != "configured")):
                raise CreateFault("CREATE_POSTCONDITION_UNVERIFIED")
            db.execute("UPDATE remote_create_tasks SET state='succeeded',"
                       "code='REST_GUEST_READY',updated=? WHERE task_id=? "
                       "AND state='submitted' AND code='REST_CREATE_ACCEPTED'",
                       (self.clock(), task_id))

    def begin_nesting(self, task_id):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute(
                "SELECT postcondition FROM remote_create_options WHERE task_id=?",
                (task_id,)).fetchone()
            if row is None or row["postcondition"] != "not_started":
                return False
            db.execute("UPDATE remote_create_options SET postcondition='intent' "
                       "WHERE task_id=? AND postcondition='not_started'", (task_id,))
            return True

    def finish_nesting(self, task_id, uuid, created):
        if not isinstance(uuid, str) or not uuid or not isinstance(created, str) or not created:
            raise CreateFault("NESTING_IDENTITY_INVALID")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_create_options SET postcondition='configured',"
                       "instance_uuid=?,instance_created=? WHERE task_id=? "
                       "AND postcondition='intent'", (uuid, created, task_id))

    def nesting_already_set(self, task_id, uuid, created):
        if not uuid or not created:
            raise CreateFault("NESTING_IDENTITY_INVALID")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_create_options SET postcondition='configured',"
                       "instance_uuid=?,instance_created=? WHERE task_id=? "
                       "AND postcondition='not_started'", (uuid, created, task_id))

    def accept(self, task_id):
        """Record only a durable upstream acceptance, never terminal success."""
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_create_tasks SET code='REST_CREATE_ACCEPTED', "
                       "updated=? WHERE task_id=? AND state='submitted' "
                       "AND code='REST_GATE_DISPATCH_INTENT'",
                       (self.clock(), task_id))

    def complete(self, task_id, succeeded):
        # A timeout, server restart, or journal race cannot become a success.
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_create_tasks SET state=?, code=?, updated=? "
                       "WHERE task_id=? AND state='submitted' "
                       "AND code!='REST_CREATE_ACCEPTED' AND code NOT IN (" +
                       ",".join("?" for _ in PREDISPATCH_FAILURE_CODES) + ")",
                       ("succeeded" if succeeded else "unknown",
                        "UPSTREAM_CREATE_ACKNOWLEDGED" if succeeded
                        else "UPSTREAM_CREATE_OUTCOME_UNKNOWN",
                        self.clock(), task_id, *sorted(PREDISPATCH_FAILURE_CODES)))

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
