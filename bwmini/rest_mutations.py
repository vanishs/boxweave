"""One-shot, provider-identity-bound REST mutation receipts for Agent MCP.

Web/CLI already use the same REST transport and R-04 SharedClaims. This
root-owned adapter is only enabled when the app's exact config and Gateway
protected coordination share the same reviewed scope and private claim store.
"""
import hashlib
import json
import re
import secrets
import time

try:
    from . import rest_routes, rest_wire, coordination_bridge, resource_size
    from .gateway_create import AppStore, GatewayRESTFault
except ImportError:
    import rest_routes
    import rest_wire
    import coordination_bridge
    import resource_size
    from gateway_create import AppStore, GatewayRESTFault

KEY = re.compile(r"[0-9a-f]{32}\Z")
ACTIONS = frozenset(("start_container", "stop_container",
                     "delete_container", "resize_container"))
MAX_TASKS = 10000


def receipt(row):
    return {"task_id": row["task_id"], "request_key": row["request_key"],
            "username": row["logical"], "action": row["action"],
            "state": row["state"], "code": row["code"],
            "target": {"name": row["instance_name"],
                       "createdAt": row["instance_created"]},
            "upstream_acknowledged": row["state"] in ("accepted", "succeeded"),
            "verified": row["state"] == "succeeded",
            "retry": "not_authorized"}


def args_key(tool, args):
    if tool not in ACTIONS or not isinstance(args, dict):
        raise GatewayRESTFault("REST_OPERATION_INVALID")
    supplied = args.get("request_key")
    if supplied is not None and (not isinstance(supplied, str)
                                 or KEY.fullmatch(supplied) is None):
        raise GatewayRESTFault("REST_REQUEST_KEY_INVALID")
    without = {k:v for k,v in args.items() if k != "request_key"}
    try:
        raw = json.dumps({"tool":tool,"args":without}, allow_nan=False,
                         sort_keys=True,separators=(",",":"))
    except (TypeError, ValueError):
        raise GatewayRESTFault("REST_OPERATION_INVALID") from None
    if len(raw) > 8192:
        raise GatewayRESTFault("REST_OPERATION_TOO_LARGE")
    digest = hashlib.sha256(raw.encode()).hexdigest()
    return supplied or digest[:32], digest, without


class MutationJournal:
    def __init__(self, store, clock=time.time):
        self.store, self.clock = store, clock

    def get(self, *, task_id=None, key=None):
        if (task_id is None) == (key is None):
            raise GatewayRESTFault("REST_TASK_SELECTOR_INVALID")
        value = task_id if task_id is not None else key
        if not isinstance(value, str) or KEY.fullmatch(value) is None:
            raise GatewayRESTFault("REST_TASK_SELECTOR_INVALID")
        column = "task_id" if task_id is not None else "request_key"
        with self.store.db() as db:
            row = db.execute("SELECT * FROM remote_rest_tasks WHERE "
                             + column + "=?", (value,)).fetchone()
        if row is None:
            raise GatewayRESTFault("REST_TASK_NOT_FOUND")
        return receipt(row)

    def check_key(self, key, logical, action, digest):
        with self.store.db() as db:
            row = db.execute("SELECT * FROM remote_rest_tasks "
                             "WHERE request_key=?", (key,)).fetchone()
        if row is None:
            return None
        if (row["logical"] != logical or row["action"] != action
                or row["payload_sha256"] != digest):
            raise GatewayRESTFault("REST_REQUEST_KEY_CONFLICT")
        return receipt(row)

    def reserve(self, key, digest, tool, row, identity, spec):
        logical = rest_routes.logical_name(row["username"])
        action = tool.removesuffix("_container")
        if action not in ("start", "stop", "delete", "resize"):
            raise GatewayRESTFault("REST_OPERATION_INVALID")
        if (row.get("name") != logical+"-container"
                or not isinstance(row.get("createdAt"), (str, int))
                or not str(row["createdAt"])):
            raise GatewayRESTFault("REST_TARGET_IDENTITY_UNVERIFIED")
        raw = json.dumps(spec, sort_keys=True, separators=(",",":"))
        if len(raw) > 4096:
            raise GatewayRESTFault("REST_OPERATION_TOO_LARGE")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            prior = db.execute("SELECT * FROM remote_rest_tasks "
                               "WHERE request_key=?", (key,)).fetchone()
            if prior is not None:
                if prior["payload_sha256"] != digest or prior["logical"] != logical:
                    raise GatewayRESTFault("REST_REQUEST_KEY_CONFLICT")
                return receipt(prior), False
            if db.execute("SELECT count(*) FROM remote_rest_tasks").fetchone()[0] >= MAX_TASKS:
                raise GatewayRESTFault("REST_TASK_JOURNAL_FULL")
            task = secrets.token_hex(16)
            now = self.clock()
            db.execute("INSERT INTO remote_rest_tasks "
                       "(task_id,request_key,logical,action,payload_sha256,"
                       "identity_digest,instance_name,instance_created,spec_json,"
                       "state,code,created,updated) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)",
                       (task,key,logical,action,digest,identity,row["name"],
                        str(row["createdAt"]),raw,"submitted","REST_PENDING",now,now))
            saved = db.execute("SELECT * FROM remote_rest_tasks "
                               "WHERE task_id=?", (task,)).fetchone()
            return receipt(saved), True

    def begin(self, task_id):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            cur=db.execute("UPDATE remote_rest_tasks SET code='REST_DISPATCH_INTENT',"
                           "updated=? WHERE task_id=? AND state='submitted' "
                           "AND code='REST_PENDING'", (self.clock(), task_id))
            return cur.rowcount == 1

    def accepted(self, task_id):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_rest_tasks SET state='accepted',"
                       "code='REST_ACKNOWLEDGED',updated=? WHERE task_id=? "
                       "AND state='submitted' AND code='REST_DISPATCH_INTENT'",
                       (self.clock(), task_id))

    def unknown(self, task_id):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_rest_tasks SET state='unknown',"
                       "code='REST_OUTCOME_UNKNOWN',updated=? WHERE task_id=? "
                       "AND state IN ('submitted','accepted')",
                       (self.clock(),task_id))

    def ready(self, task_id):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_rest_tasks SET state='succeeded',"
                       "code='REST_DESIRED_STATE_OBSERVED',updated=? "
                       "WHERE task_id=? AND state='accepted'",
                       (self.clock(), task_id))

    def poll(self, interval=15):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT * FROM remote_rest_tasks "
                             "WHERE state='accepted' AND last_probe_at<=? "
                             "ORDER BY created,task_id LIMIT 1",
                             (self.clock()-interval,)).fetchone()
            if row is not None:
                db.execute("UPDATE remote_rest_tasks SET last_probe_at=? "
                           "WHERE task_id=?", (self.clock(), row["task_id"]))
        return dict(row) if row is not None else None

    def recover(self):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE remote_rest_tasks SET state='unknown',"
                       "code='PROCESS_RESTART_UNKNOWN',updated=? "
                       "WHERE state='submitted'",
                       (self.clock(),))


def dispatch(configuration, tool, args, *, protected):
    if not protected or configuration.get("mode") != "coordinated":
        raise GatewayRESTFault("REST_COORDINATION_REQUIRED")
    key,digest,params = args_key(tool,args)
    logical = rest_routes.logical_name(params.get("username"))
    method,path,body = rest_routes.route(tool,params)
    ledger = MutationJournal(AppStore(configuration["state_db"]))
    prior = ledger.check_key(key,logical,tool.removesuffix("_container"),digest)
    if prior is not None:
        return prior
    rest = rest_wire.Client({k:configuration[k] for k in ("url","token_file")})
    try:
        observed = rest.request(
            "GET","/v1/containers/"+logical,timeout=20)
        row = rest_routes.validate_read_result(
            "get_container",observed,username=logical)["container"]
    except (rest_wire.WireFault, rest_routes.RouteFault):
        raise GatewayRESTFault("REST_TARGET_UNVERIFIED") from None
    state = row.get("state")
    if (tool == "start_container" and state != "CONTAINER_STATE_STOPPED"
            or tool == "stop_container" and state != "CONTAINER_STATE_RUNNING"
            or tool == "resize_container" and state not in (
                "CONTAINER_STATE_RUNNING", "CONTAINER_STATE_STOPPED")):
        raise GatewayRESTFault("REST_RESOURCE_STATE_NOT_ALLOWED")
    if tool == "resize_container" and body.get("disk") is not None:
        current = row.get("resources")
        if not isinstance(current,dict) or not isinstance(current.get("disk"),str):
            raise GatewayRESTFault("RESOURCE_DISK_SIZE_UNVERIFIED")
        try:
            resource_size.require_nonshrinking_disk(body["disk"],current["disk"])
        except ValueError as exc:
            raise GatewayRESTFault(str(exc)) from None
    scope = coordination_bridge.published_scope()
    identity,scope_id,observed_name = coordination_bridge.gateway_box(
        row,scope,logical)
    if observed_name != logical:
        raise GatewayRESTFault("REST_TARGET_IDENTITY_UNVERIFIED")
    task,created = ledger.reserve(key,digest,tool,row,identity,body or {})
    if not created:
        return task
    task_id = task["task_id"]
    claims=coordination_bridge.SharedClaims()
    try:
        claims.acquire(task_id,"gateway_"+task_id,identity,scope_id,logical)
        claims.dispatch_intent(task_id)
        if not ledger.begin(task_id):
            return ledger.get(task_id=task_id)
    except Exception:
        # The receipt survives; no outbound call has happened.
        ledger.unknown(task_id)
        raise GatewayRESTFault("REST_PREDISPATCH_UNVERIFIED") from None
    try:
        # The single REST request can be accepted while daemon state still
        # transitions. The app worker independently verifies the same target.
        rest.request(method,path,body=body,timeout=60)
        ledger.accepted(task_id)
    except Exception:
        ledger.unknown(task_id)
        try: claims.unknown(task_id)
        except Exception: pass
    return ledger.get(task_id=task_id)
