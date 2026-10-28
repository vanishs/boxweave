"""The same application operations serve HTTP and CLI adapters."""
import fcntl
import json
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

    def projects(self):
        with self.store.db() as db:
            return [dict(r) for r in db.execute("SELECT id,name,revision FROM projects ORDER BY rowid")]

    def bind(self, label, logical, path, project=None, revision=None):
        target = self.provider.observe(logical)
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
        return self.store.project(pid)

    def plan(self, project, revision, actor="operator"):
        p = self.store.project(project)
        if p["revision"] != revision:
            raise Fault("REVISION_CONFLICT")
        current = self.provider.observe(json.loads(p["target"])["logical"])
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
            db.execute("UPDATE plans SET used=1 WHERE id=?", (plan,))
            db.execute("INSERT INTO operations VALUES(?,?,?,?,?,?,?,?,?,?)",
                       (op,p["id"],p["revision"],p["digest"],actor,"queued",now,now+self.budget,None,None))
        return self.store.receipt(op,actor)

    def list_receipts(self, actor="operator"):
        with self.store.db() as db:
            return [dict(r) for r in db.execute("SELECT * FROM operations WHERE actor=? ORDER BY created DESC LIMIT 100",(actor,))]

    def recover(self):
        # Caller holds the process-scoped worker lock. Another live worker cannot be recovered.
        with self.store.db() as db:
            db.execute("UPDATE operations SET state='unknown',code='WORKER_LOST' WHERE state='dispatching'")

    def work_one(self):
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
                return True
            db.execute("UPDATE operations SET state='dispatching' WHERE id=?",(op["id"],))
        state, code, result = "unknown", "OUTCOME_UNKNOWN", None
        try:
            p = self.store.project(op["project"])
            if p["revision"] != op["revision"] or p["digest"] != op["digest"]:
                raise Fault("REVISION_CONFLICT")
            expected = Target(**json.loads(p["target"]))
            observed = self.provider.observe(expected.logical)
            if observed != expected:
                raise Fault("TARGET_CHANGED")
            left = op["deadline"] - self.clock()
            if left <= 0:
                raise Fault("DEADLINE_EXCEEDED")
            candidate = self.provider.hello(expected, p["workspace"], left)
            if candidate.get("stdout") != "hello world\n" or type(candidate.get("exit_code")) is not int or candidate["exit_code"] != 0:
                raise Fault("COMMAND_RESULT_UNEXPECTED")
            result = {"stdout":"hello world\n", "exit_code":0,
                      "identity":{"user":expected.user,"hostname":expected.hostname},
                      "evidence":"fixture" if self.provider.mode=="fixture" else "live_mcp",
                      "remote_cleanup":"not_applicable" if self.provider.mode=="fixture" else "not_verified"}
            state, code = "succeeded", "OK"
        except Fault as error:
            code = error.code
            # These fail before the hello side effect, not a claim all remote requests were cancelled.
            if code in {"TARGET_CHANGED","REVISION_CONFLICT","IDENTITY_MISMATCH","TARGET_NOT_ALLOWED","NOT_FOUND"}:
                state = "failed"
        except Exception:
            pass  # Untrusted transport/OS details never enter the receipt.
        with self.store.db() as db:
            db.execute("UPDATE operations SET state=?,code=?,result=? WHERE id=?",
                       (state,code,json.dumps(result) if result is not None else None,op["id"]))
            db.execute("INSERT INTO audit VALUES(?,?,?,?)",(self.clock(),"hello",op["project"],code))
        return True

    def worker(self, once=False):
        lock = Path(str(self.store.path)+".worker")
        with open(lock,"a") as f:
            try:
                fcntl.flock(f, fcntl.LOCK_EX|fcntl.LOCK_NB)
            except BlockingIOError:
                raise Fault("WORKER_ALREADY_RUNNING")
            self.recover()
            while True:
                worked = self.work_one()
                if once:
                    break
                if not worked:
                    time.sleep(.2)
