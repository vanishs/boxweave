"""Durable Box action plans and single-dispatch worker.

A confirmed plan persists *before* attempting remote side effects. Recovered
dispatching work is UNKNOWN and NEVER automatically re-executed.
"""
import json
import uuid
from .agent_box import BoxAgent, WRITE_TOOLS
from .domain import Fault

SCHEMA = """
CREATE TABLE box_action_plans (
 id TEXT PRIMARY KEY, actor TEXT NOT NULL, logical TEXT NOT NULL,
 target_digest TEXT NOT NULL, action TEXT NOT NULL, arguments TEXT NOT NULL,
 created REAL NOT NULL, expires REAL NOT NULL, used INTEGER NOT NULL DEFAULT 0);
CREATE TABLE box_action_receipts (
 id TEXT PRIMARY KEY, plan TEXT NOT NULL UNIQUE REFERENCES box_action_plans(id),
 actor TEXT NOT NULL, logical TEXT NOT NULL, target_digest TEXT NOT NULL,
 action TEXT NOT NULL, state TEXT NOT NULL, created REAL NOT NULL,
 updated REAL NOT NULL, code TEXT NOT NULL);
CREATE INDEX box_action_receipts_state ON box_action_receipts(state, created);
"""

def new_id():
    return uuid.uuid4().hex

class BoxActions:
    def __init__(self, service):
        self.service=service
        self.store=service.store
        self.clock=service.clock

    def plan(self,logical,action,args,actor="operator"):
        if action not in WRITE_TOOLS:raise Fault("ACTION_NOT_ALLOWED",403)
        adapter=BoxAgent(self.service,logical)
        adapter.validate(action,args)
        serialized=json.dumps(args,sort_keys=True,separators=(",",":"))
        if len(serialized.encode())>16384:raise Fault("REQUEST_TOO_LARGE",413)
        now=self.clock()
        pid=new_id()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("""INSERT INTO box_action_plans
                (id,actor,logical,target_digest,action,arguments,created,expires)
                VALUES (?,?,?,?,?,?,?,?)""",
                (pid,actor,logical,adapter.target.digest(),action,serialized,now,now+120))
        # Never return the command or file content in a plan response.
        return {"id":pid,"logical":logical,"action":action,"expires":now+120,
                "confirmation_required":True,"state":"planned"}

    def confirm(self,pid,logical,actor="operator"):
        now=self.clock()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row=db.execute("SELECT * FROM box_action_plans WHERE id=? AND actor=?",
                           (pid,actor)).fetchone()
            if row is None:raise Fault("NOT_FOUND",404)
            if logical!=row["logical"]:raise Fault("TARGET_CHANGED")
            if row["used"] or now<row["created"] or now>=row["expires"]:
                raise Fault("PLAN_STALE")
            # This check is intentionally before enqueue/dispatch.
            target=BoxAgent(self.service,logical)
            if target.target.digest()!=row["target_digest"]:raise Fault("TARGET_CHANGED")
            if row["action"] not in target.allowed:raise Fault("AGENT_BOX_TOOL_DENIED",403)
            receipt=new_id()
            db.execute("UPDATE box_action_plans SET used=1 WHERE id=? AND used=0",(pid,))
            db.execute("""INSERT INTO box_action_receipts
               (id,plan,actor,logical,target_digest,action,state,created,updated,code)
               VALUES (?,?,?,?,? ,?,'queued',?,?,'QUEUED')""",
               (receipt,pid,actor,logical,row["target_digest"],row["action"],now,now))
        return {"id":receipt,"logical":logical,"action":row["action"],
                "state":"queued","code":"QUEUED"}

    def receipt(self,ident,actor="operator"):
        with self.store.db() as db:
            row=db.execute("""SELECT id,actor,logical,target_digest,action,state,
                                     created,updated,code
                              FROM box_action_receipts WHERE id=? AND actor=?""",
                           (ident,actor)).fetchone()
        if row is None:raise Fault("NOT_FOUND",404)
        return dict(row)

    def list(self,actor="operator"):
        with self.store.db() as db:
            rows=db.execute("""SELECT id,actor,logical,target_digest,action,state,
                                      created,updated,code
                               FROM box_action_receipts WHERE actor=?
                               ORDER BY created DESC LIMIT 100""",(actor,)).fetchall()
        return [dict(row) for row in rows]

    def recover(self):
        # The remote operation might have succeeded before the crash.
        with self.store.db() as db:
            db.execute("UPDATE box_action_receipts SET state='unknown',code='OUTCOME_UNKNOWN' "
                       "WHERE state IN ('validating','dispatching')")

    def work_one(self):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row=db.execute("""SELECT r.id, r.logical, r.target_digest, r.action,
                                     p.arguments
                              FROM box_action_receipts r
                              JOIN box_action_plans p ON p.id=r.plan
                              WHERE r.state='queued' ORDER BY r.created LIMIT 1""").fetchone()
            if row is None:return False
            op=dict(row)
            db.execute("UPDATE box_action_receipts SET state='validating',code='VALIDATING',updated=? WHERE id=?",
                       (self.clock(),op["id"]))
        # Validation failures occur without upstream tool invocation.
        try:
            adapter=BoxAgent(self.service,op["logical"])
            if adapter.target.digest()!=op["target_digest"]:
                raise Fault("TARGET_CHANGED")
            args=json.loads(op["arguments"])
            adapter.validate(op["action"],args)
        except Fault as error:
            with self.store.db() as db:
                db.execute("UPDATE box_action_receipts SET state='failed',code=?,updated=? WHERE id=?",
                           (error.code,self.clock(),op["id"]))
            return True
        except Exception:
            with self.store.db() as db:
                db.execute("UPDATE box_action_receipts SET state='unknown',code='VALIDATION_UNVERIFIED',updated=? WHERE id=?",
                           (self.clock(),op["id"]))
            return True

        with self.store.db() as db:
            db.execute("UPDATE box_action_receipts SET state='dispatching',code='DISPATCH_INTENT',updated=? WHERE id=?",
                       (self.clock(),op["id"]))
        try:
            adapter.execute(op["action"],args,timeout=30)
            state,code="succeeded","OK"
        except Exception:
            # Lost response is not proof of failure. Never replay.
            state,code="unknown","OUTCOME_UNKNOWN"
        with self.store.db() as db:
            db.execute("UPDATE box_action_receipts SET state=?,code=?,updated=? WHERE id=?",
                       (state,code,self.clock(),op["id"]))
        return True
