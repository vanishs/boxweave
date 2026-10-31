"""The same application operations serve HTTP and CLI adapters."""
import base64
import binascii
from . import diagnostics
import fcntl
import json
import math
import re
import time
import uuid
from pathlib import Path
from .domain import Target, Fault, name, workspace

def ident():
    return uuid.uuid4().hex

class Service:
    def __init__(self, store, provider, clock=time.time, budget=60):
        self.store, self.provider, self.clock = store, provider, clock
        self.budget = min(600, max(1, budget))
        from .lifecycle import Lifecycle
        self.lifecycle = Lifecycle(store,provider,clock)
        from .access import Access
        self.access = Access(self)
        self.lifecycle.network = self.access

    def observe_target(self, logical):
        if self.lifecycle.config is not None and logical in self.lifecycle.config["allowed_names"]:
            try:
                target=self.lifecycle.managed(logical)
            except Fault as error:
                if error.code=="MANAGED_TARGET_REQUIRED":
                    raise Fault("TARGET_CHANGED") from None
                raise
            current=self.provider.lifecycle_inspect(logical)
            if current is None or current["target"]!=target:
                raise Fault("TARGET_CHANGED")
            if current["state"]!="CONTAINER_STATE_RUNNING":
                raise Fault("NOT_CONNECTION_READY")
            return target
        return self.provider.observe(logical)

    def projects(self):
        with self.store.db() as db:
            rows=[dict(r) for r in db.execute("SELECT p.id,p.name,p.revision,p.target,p.workspace,u.project AS unbound FROM projects p LEFT JOIN project_unbindings u ON p.id=u.project ORDER BY p.rowid")]
        return [{"id":r["id"],"name":r["name"],"revision":r["revision"],"bound":r["unbound"] is None,"target":json.loads(r["target"]) if r["unbound"] is None else None,"workspace":r["workspace"] if r["unbound"] is None else None} for r in rows]

    def bind(self, label, logical, path, project=None, revision=None):
        if project is not None and type(revision) is not int:raise Fault("INVALID_REVISION",422)
        target = self.observe_target(logical)
        label, path = name(label), workspace(path, target.user)
        pid = project or ident()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT revision FROM projects WHERE id=?", (pid,)).fetchone()
            if row and row["revision"] != revision:
                raise Fault("REVISION_CONFLICT")
            if project and row is None:
                raise Fault("NOT_FOUND", 404)
            rev = row["revision"] + 1 if row else 1
            if row:
                db.execute("UPDATE projects SET name=?,target=?,digest=?,workspace=?,revision=? WHERE id=?",
                           (label,json.dumps(target.data()),target.digest(),path,rev,pid))
            else:
                db.execute("INSERT INTO projects VALUES(?,?,?,?,?,?)",
                           (pid,label,json.dumps(target.data()),target.digest(),path,rev))
            db.execute("DELETE FROM project_unbindings WHERE project=?",(pid,))
        return self.store.project(pid)

    def unbind(self, project, revision):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row=db.execute("SELECT revision FROM projects WHERE id=?",(project,)).fetchone()
            if row is None:raise Fault("NOT_FOUND",404)
            if type(revision) is not int or row["revision"]!=revision:raise Fault("REVISION_CONFLICT")
            if db.execute("SELECT 1 FROM project_unbindings WHERE project=?",(project,)).fetchone():raise Fault("PROJECT_UNBOUND")
            db.execute("INSERT INTO project_unbindings VALUES(?)",(project,))
            db.execute("UPDATE projects SET revision=revision+1 WHERE id=?",(project,))
        return self.store.project(project)

    def plan(self, project, revision, actor="operator"):
        if type(revision) is not int:raise Fault("INVALID_REVISION",422)
        p = self.store.project(project)
        with self.store.db() as db:
            if db.execute("SELECT 1 FROM project_unbindings WHERE project=?",(project,)).fetchone():raise Fault("PROJECT_UNBOUND")
        if p["revision"] != revision:
            raise Fault("REVISION_CONFLICT")
        current = self.observe_target(json.loads(p["target"])["logical"])
        if current.digest() != p["digest"]:
            raise Fault("TARGET_CHANGED")
        plan = ident()
        with self.store.db() as db:
            db.execute("INSERT INTO plans VALUES(?,?,?,?,?,?,0)",
                       (plan,project,revision,p["digest"],self.clock()+300,actor))
        return {"id":plan, "action":"hello", "project":project, "revision":revision,
                "target":current.data(), "workspace":p["workspace"]}

    def confirm(self, plan, actor="operator"):
        now = self.clock()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT * FROM plans WHERE id=? AND actor=?", (plan,actor)).fetchone()
            if row is None:
                raise Fault("NOT_FOUND", 404)
            if row["used"] or now > row["expires"]:
                raise Fault("PLAN_EXPIRED_OR_USED")
            p = db.execute("SELECT * FROM projects WHERE id=?", (row["project"],)).fetchone()
            if p["revision"] != row["revision"] or p["digest"] != row["digest"]:
                raise Fault("REVISION_CONFLICT")
            op = ident()
            diagnostics.link(db, actor, "hello", op)
            db.execute("UPDATE plans SET used=1 WHERE id=?", (plan,))
            db.execute("INSERT INTO operations VALUES(?,?,?,?,?,?,?,?,?,?)",
                       (op,p["id"],p["revision"],p["digest"],actor,"queued",now,now+self.budget,None,None))
        return self.store.receipt(op,actor)

    def list_receipts(self, actor="operator"):
        with self.store.db() as db:
            return [dict(r) for r in db.execute("SELECT * FROM operations WHERE actor=? ORDER BY created DESC LIMIT 100",(actor,))]

    def receipt_history(self, actor="operator", limit=20, cursor=None):
        if type(limit) is not int or not 1 <= limit <= 100:
            raise Fault("INVALID_LIMIT", 400)
        after = None
        if cursor is not None:
            try:
                if not isinstance(cursor, str) or not 1 <= len(cursor) <= 256:
                    raise ValueError()
                raw = base64.b64decode(cursor, altchars=b"-_", validate=True)
                after = json.loads(raw)
                if (not isinstance(after, dict) or set(after) != {"created", "id"}
                        or type(after["created"]) not in (int, float)
                        or not math.isfinite(after["created"])
                        or not isinstance(after["id"], str)
                        or re.fullmatch(r"[0-9a-f]{32}", after["id"]) is None):
                    raise ValueError()
            except (ValueError, TypeError, OverflowError, UnicodeError, binascii.Error):
                raise Fault("INVALID_CURSOR", 400) from None
        query = "SELECT id,project,state,code,created FROM operations WHERE actor=?"
        args = [actor]
        if after is not None:
            query += " AND (created<? OR (created=? AND id<?))"
            args.extend([after["created"], after["created"], after["id"]])
        query += " ORDER BY created DESC,id DESC LIMIT ?"
        args.append(limit + 1)
        with self.store.db() as db:
            rows = [dict(row) for row in db.execute(query, args)]
        more, rows = len(rows) > limit, rows[:limit]
        next_cursor = None
        if more:
            boundary = {"created": rows[-1]["created"], "id": rows[-1]["id"]}
            next_cursor = base64.urlsafe_b64encode(json.dumps(boundary, separators=(",", ":")).encode()).decode()
        return {"operations": rows, "next_cursor": next_cursor}

    def observations(self, operation, actor="operator"):
        self.store.receipt(operation, actor)
        with self.store.db() as db:
            return [dict(row) for row in db.execute(
                "SELECT id,at,target_status,code FROM operation_observations WHERE operation=? AND actor=? ORDER BY rowid DESC LIMIT 20",
                (operation, actor))]

    def reconcile(self, operation, actor="operator"):
        op = self.store.receipt(operation, actor)
        if op["state"] != "unknown":
            raise Fault("RECONCILIATION_NOT_REQUIRED")
        project = self.store.project(op["project"])
        status, code = "unavailable", "OBSERVATION_UNAVAILABLE"
        if project["revision"] != op["revision"] or project["digest"] != op["digest"]:
            status, code = "changed", "REVISION_CONFLICT"
        else:
            try:
                expected = Target(**json.loads(project["target"]))
                observed = self.observe_target(expected.logical)
                status, code = ("matching", "TARGET_MATCHES") if observed == expected else ("changed", "TARGET_CHANGED")
            except Fault as error:
                if error.code == "TARGET_CHANGED":
                    status, code = "changed", "TARGET_CHANGED"
            except Exception:
                pass  # No remote exception or command output is published.
        at = self.clock()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            current = db.execute("SELECT * FROM operations WHERE id=? AND actor=?", (operation, actor)).fetchone()
            latest = db.execute("SELECT revision,digest FROM projects WHERE id=?", (op["project"],)).fetchone()
            if current is None or current["state"] != "unknown":
                raise Fault("RECONCILIATION_NOT_REQUIRED")
            if latest["revision"] != op["revision"] or latest["digest"] != op["digest"]:
                status, code = "changed", "REVISION_CONFLICT"
            observation = ident()
            db.execute("INSERT INTO operation_observations VALUES(?,?,?,?,?,?)",
                       (observation, operation, actor, at, status, code))
        return {"id":observation, "operation":operation, "at":at, "target_status":status,
                "code":code, "state":"unknown", "completion":"unverified", "retry":"not_authorized"}

    def recover(self):
        # Caller holds the process-scoped worker lock. Another live worker cannot be recovered.
        with self.store.db() as db:
            rows = [dict(r) for r in db.execute(
                "SELECT id,actor FROM operations WHERE state='dispatching'")]
            db.execute("UPDATE operations SET state='unknown',code='WORKER_LOST',result=NULL WHERE state='dispatching'")
        for op in rows:
            trace = diagnostics.trace_for_operation(self.store, op["actor"], "hello", op["id"]) \
                or diagnostics.new_trace_id()
            diagnostics.emit(self.store, component="worker", phase="recovered", event_type="recovered",
                             outcome="unknown", code="WORKER_LOST", trace_id=trace,
                             actor_scope=op["actor"], operation_kind="hello", operation_id=op["id"])

    def work_one(self):
        if self.access.work_one():
            return True
        if self.lifecycle.work_one():
            return True
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT * FROM operations WHERE state='queued' ORDER BY created LIMIT 1").fetchone()
            if row is None:
                return False
            op = dict(row)
            # Sample after claiming the queue snapshot: a new enqueue may have
            # committed while this worker waited for the SQLite writer lock.
            now = self.clock()
            if now < op["created"] or now >= op["deadline"]:
                db.execute("UPDATE operations SET state='failed',code='DEADLINE_EXCEEDED' WHERE id=?",(op["id"],))
                expired = True
            else:
                db.execute("UPDATE operations SET state='dispatching' WHERE id=?",(op["id"],))
                expired = False
        trace = diagnostics.trace_for_operation(self.store, op["actor"], "hello", op["id"]) \
            or diagnostics.new_trace_id()
        if expired:
            diagnostics.emit(self.store, component="worker", phase="deadline", event_type="observed",
                             outcome="failed", code="DEADLINE_EXCEEDED", trace_id=trace,
                             actor_scope=op["actor"], operation_kind="hello", operation_id=op["id"])
            return True
        started = time.monotonic()
        diagnostics.emit(self.store, component="worker", phase="claimed", event_type="claimed",
                         outcome="passed", code="CLAIMED", trace_id=trace,
                         actor_scope=op["actor"], operation_kind="hello", operation_id=op["id"])
        state, code, result = "unknown", "OUTCOME_UNKNOWN", None
        detail_code = "OUTCOME_UNKNOWN"
        try:
            p = self.store.project(op["project"])
            if p["revision"] != op["revision"] or p["digest"] != op["digest"]:
                raise Fault("REVISION_CONFLICT")
            expected = Target(**json.loads(p["target"]))
            observed = self.observe_target(expected.logical)
            if observed != expected:
                raise Fault("TARGET_CHANGED")
            left = op["deadline"] - self.clock()
            if left <= 0:
                raise Fault("DEADLINE_EXCEEDED")
            candidate = self.provider.hello(expected, p["workspace"], left)
            if (not isinstance(candidate, dict) or not isinstance(candidate.get("stdout"), str)
                    or len(candidate["stdout"]) > 4096 or type(candidate.get("exit_code")) is not int):
                raise Fault("COMMAND_RESULT_UNEXPECTED")
            valid = candidate["stdout"] == "hello world\n" and candidate["exit_code"] == 0
            result = {"stdout": "hello world\n" if valid else "", "exit_code":candidate["exit_code"],
                      "output_withheld":not valid,
                      "identity":{"user":expected.user,"hostname":expected.hostname},
                      "evidence":"fixture" if self.provider.mode=="fixture" else "live_mcp",
                      "remote_cleanup":"not_applicable" if self.provider.mode=="fixture" else "not_verified"}
            state = "succeeded" if valid else "failed"
            code = "OK" if valid else ("COMMAND_FAILED" if candidate["exit_code"] != 0 else "COMMAND_RESULT_UNEXPECTED")
            detail_code = code
        except Fault as error:
            code = error.code
            detail_code = getattr(error, "error_class", None) or code
            # These fail before the hello side effect, not a claim all remote requests were cancelled.
            if code in {"TARGET_CHANGED","REVISION_CONFLICT","IDENTITY_MISMATCH","TARGET_NOT_ALLOWED","NOT_FOUND"}:
                state = "failed"
        except Exception:
            detail_code = "UNCLASSIFIED_EXCEPTION"
            pass  # Untrusted transport/OS details never enter the receipt.
        with self.store.db() as db:
            db.execute("UPDATE operations SET state=?,code=?,result=? WHERE id=?",
                       (state,code,json.dumps(result) if result is not None else None,op["id"]))
            db.execute("INSERT INTO audit VALUES(?,?,?,?)",(self.clock(),"hello",op["project"],code))
        diagnostics.emit(self.store, component="worker", phase="observed", event_type="observed",
                         outcome={"succeeded":"passed","failed":"failed"}.get(state,"unknown"),
                         code=detail_code, trace_id=trace, actor_scope=op["actor"],
                         operation_kind="hello", operation_id=op["id"],
                         elapsed_ms=int((time.monotonic()-started)*1000))
        return True

    def worker(self, once=False):
        lock = Path(str(self.store.path)+".worker")
        with open(lock,"a") as f:
            try:
                fcntl.flock(f, fcntl.LOCK_EX|fcntl.LOCK_NB)
            except BlockingIOError:
                raise Fault("WORKER_ALREADY_RUNNING")
            self.recover()
            self.lifecycle.recover()
            self.access.recover()
            while True:
                worked = self.work_one()
                if once:
                    break
                if not worked:
                    time.sleep(.2)
