"""Gateway-private metadata-only idempotent Box task receipts.

No command body, stdout/stderr, client token, or secret is persisted.
The shared coordinator belongs to a different minimal DB and will be
integrated only after verified cross-entry dispatch claims.
"""
import hashlib
import hmac
import json
import os
from pathlib import Path
import pwd
import re
import secrets
import sqlite3
import stat
import time
from urllib.parse import quote

DEFAULT_DB = Path("/home/boxweave/.boxweave/gateway-tasks.sqlite3")
TASK_LIMIT = 10000
KEY = re.compile(r"[a-f0-9]{32}\Z")
BOX = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
DIGEST = re.compile(r"[a-f0-9]{64}\Z")

TOOLS = (
    {"name":"boxweave_task_submit",
     "description":"Deduplicated metadata-only connect(exec), under existing trust + connect tool policy. Key must be random 32-hex. Retained output disabled; cross-Web coordination not yet claimed.",
     "inputSchema":{"type":"object","additionalProperties":False,
       "properties":{"box":{"type":"string"},"request_key":{"type":"string"},
                     "exec":{"type":"string"},
                     "lease_token":{"type":"string","description":"Optional token for a protected-mode Box lease; never persisted."}},
       "required":["box","request_key","exec"]}},
    {"name":"boxweave_task_get",
     "description":"Read durable gateway task metadata by original task_id or request_key. Never retransmits.",
     "inputSchema":{"type":"object","additionalProperties":False,
       "properties":{"task_id":{"type":"string"},"request_key":{"type":"string"}}},
     "annotations":{"readOnlyHint":True,"destructiveHint":False}},
    {"name":"boxweave_task_list",
     "description":"Read up to 50 metadata-only gateway task receipts by Box. No command bodies or output.",
     "inputSchema":{"type":"object","additionalProperties":False,
       "properties":{"box":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":50}},
       "required":["box"]},"annotations":{"readOnlyHint":True,"destructiveHint":False}},
)

SCHEMA = """
CREATE TABLE IF NOT EXISTS gateway_tasks(
 task_id TEXT PRIMARY KEY, request_key TEXT NOT NULL UNIQUE,
 payload_digest TEXT NOT NULL, identity_digest TEXT NOT NULL,
 logical TEXT NOT NULL, state TEXT NOT NULL CHECK(state IN
 ('recorded','dispatch_intent','completed','unknown','failed_before_dispatch')),
 exit_code INTEGER, created REAL NOT NULL, updated REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS gateway_tasks_created ON gateway_tasks(created);
"""

class TaskFault(Exception):
    def __init__(self, code):
        self.code=code
        super().__init__(code)

def command_digest(box, command):
    if (not isinstance(box,str) or BOX.fullmatch(box) is None
            or not isinstance(command,str) or not 1<=len(command)<=4096
            or any(ord(c)<32 and c not in "\r\n\t" for c in command)):
        raise TaskFault("TASK_ARGUMENT_INVALID")
    return hashlib.sha256(json.dumps(
        {"operation":"exec","box":box,"command":command},
        sort_keys=True,separators=(",",":")).encode()).hexdigest()

def observed_exit(message):
    result=message.get("result",{})
    if not isinstance(result,dict) or result.get("isError"):
        raise TaskFault("TASK_RESULT_UNVERIFIED")
    structured=result.get("structuredContent")
    if isinstance(structured,dict) and type(structured.get("exit_code")) is int:
        return structured["exit_code"]
    text="\n".join(item.get("text","") for item in result.get("content",[])
                   if isinstance(item,dict) and isinstance(item.get("text"),str))
    if len(text)>2097152:
        raise TaskFault("TASK_RESULT_UNVERIFIED")
    # Real Provider responses without output contain only "exit_code: N\n".
    # Match the entire bare response; never infer completion from a prefix.
    bare=re.fullmatch(r"exit_code: (-?[0-9]+)\n",text)
    if bare:
        return int(bare[1])
    # The Provider emits a stderr-only section on failing commands with
    # diagnostics but no stdout. The observed numeric exit is still terminal.
    # Keep the same bounded, fixed-heading contract as the stdout branch.
    m=re.match(r"\Aexit_code: (-?[0-9]+)\n\n--- stdout ---\n",text)
    if m:
        return int(m[1])
    # Stderr-only formatting is emitted only when bytes actually exist.
    # Refuse an incomplete stderr heading with a missing body.
    stderr=re.match(r"\Aexit_code: (-?[0-9]+)\n\n--- stderr ---\n[\s\S]",text)
    if stderr:
        return int(stderr[1])
    raise TaskFault("TASK_RESULT_UNVERIFIED")

class TaskStore:
    def __init__(self,path=DEFAULT_DB,clock=time.time):
        self.path=Path(path)
        self.clock=clock

    def connect(self, initializing=False):
        if not self.path.is_absolute() or ".." in self.path.parts:
            raise TaskFault("TASK_STORAGE_UNSAFE")
        try:
            owner=os.geteuid()
            parent=self.path.parent.lstat()
            dbstat=self.path.lstat()
            if (not stat.S_ISDIR(parent.st_mode) or parent.st_uid!=owner or parent.st_mode&0o077
                    or not stat.S_ISREG(dbstat.st_mode) or dbstat.st_uid!=owner
                    or dbstat.st_mode&0o077 or dbstat.st_nlink!=1 or dbstat.st_size>64*1024*1024):
                raise TaskFault("TASK_STORAGE_UNSAFE")
            for ancestor in self.path.parent.parents:
                meta=ancestor.lstat()
                if (not stat.S_ISDIR(meta.st_mode) or
                    (meta.st_mode&0o022 and not (meta.st_uid==0 and meta.st_mode&stat.S_ISVTX))):
                    raise TaskFault("TASK_STORAGE_UNSAFE")
            for ext in ("-journal","-wal","-shm"):
                side=Path(str(self.path)+ext)
                try:
                    m=side.lstat()
                except FileNotFoundError:
                    continue
                if (not stat.S_ISREG(m.st_mode) or m.st_uid!=owner or m.st_mode&0o077
                        or m.st_nlink!=1):
                    raise TaskFault("TASK_STORAGE_UNSAFE")
            db=sqlite3.connect("file:"+quote(str(self.path),safe="/")+"?mode=rw",uri=True,timeout=2)
            try:
                later=self.path.lstat()
                if later.st_ino!=dbstat.st_ino or later.st_dev!=dbstat.st_dev:
                    raise TaskFault("TASK_STORAGE_UNSAFE")
                version=db.execute("PRAGMA user_version").fetchone()[0]
                if version not in ((0,1) if initializing else (1,)):
                    raise TaskFault("TASK_SCHEMA_UNSUPPORTED")
                if not initializing:
                    tables={r[0] for r in db.execute(
                        "SELECT name FROM sqlite_master WHERE type='table'")}
                    if tables!={"gateway_tasks"}:
                        raise TaskFault("TASK_SCHEMA_UNSUPPORTED")
                return db
            except BaseException:
                db.close()
                raise
        except (OSError,sqlite3.Error):
            raise TaskFault("TASK_TRACKING_UNAVAILABLE") from None

    def _transaction(self,callback,initializing=False):
        db=self.connect(initializing)
        try:
            db.execute("BEGIN IMMEDIATE")
            output=callback(db)
            db.commit()
            return output
        except sqlite3.Error:
            db.rollback()
            raise TaskFault("TASK_TRACKING_UNAVAILABLE") from None
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()

    def initialize(self):
        def init(db):
            existing={r[0] for r in db.execute("SELECT name FROM sqlite_master WHERE type='table'")}
            if existing and existing!={"gateway_tasks"}:
                raise TaskFault("TASK_SCHEMA_UNSUPPORTED")
            for line in SCHEMA.split(";"):
                if line.strip():db.execute(line)
            db.execute("PRAGMA user_version=1")
        self._transaction(init,initializing=True)

    @staticmethod
    def row(r):
        if r is None:raise TaskFault("TASK_NOT_FOUND")
        # A stale dispatch intent is unverified, even if the gateway died
        # before sending its Provider request. No reconnect replays it.
        state="unknown" if r[5]=="dispatch_intent" else r[5]
        return {"task_id":r[0],"request_key":r[1],"box":r[4],
                "state":state,"exit_code":r[6] if state=="completed" else None,
                "result":{"available":False},"coverage":"gateway_tracked_only"}

    def reserve(self,request_key,box,identity_digest,payload_digest):
        if (not isinstance(request_key,str) or KEY.fullmatch(request_key) is None
                or not isinstance(box,str) or BOX.fullmatch(box) is None
                or not isinstance(identity_digest,str) or DIGEST.fullmatch(identity_digest) is None
                or not isinstance(payload_digest,str) or DIGEST.fullmatch(payload_digest) is None):
            raise TaskFault("TASK_ARGUMENT_INVALID")
        def commit(db):
            current=db.execute("SELECT * FROM gateway_tasks WHERE request_key=?",(request_key,)).fetchone()
            if current is not None:
                if current[2]!=payload_digest or current[3]!=identity_digest or current[4]!=box:
                    raise TaskFault("REQUEST_KEY_CONFLICT")
                return self.row(current),False
            if db.execute("SELECT count(*) FROM gateway_tasks").fetchone()[0]>=TASK_LIMIT:
                raise TaskFault("TASK_CAPACITY_EXCEEDED")
            task_id=secrets.token_hex(16);now=self.clock()
            db.execute("INSERT INTO gateway_tasks VALUES(?,?,?,?,?,?,?,?,?)",
                       (task_id,request_key,payload_digest,identity_digest,box,"recorded",None,now,now))
            return self.row(db.execute("SELECT * FROM gateway_tasks WHERE task_id=?",(task_id,)).fetchone()),True
        return self._transaction(commit)

    def advance(self,task_id,previous,next_state,exit_code=None):
        if (not isinstance(task_id,str) or KEY.fullmatch(task_id) is None
                or (previous,next_state) not in (("recorded","dispatch_intent"),
                                                 ("recorded","failed_before_dispatch"),
                                                 ("dispatch_intent","completed"),
                                                 ("dispatch_intent","unknown"))):
            raise TaskFault("TASK_ARGUMENT_INVALID")
        if next_state=="completed" and type(exit_code) is not int:
            raise TaskFault("TASK_RESULT_UNVERIFIED")
        def commit(db):
            row=db.execute("SELECT state FROM gateway_tasks WHERE task_id=?",(task_id,)).fetchone()
            if row is None:raise TaskFault("TASK_NOT_FOUND")
            if row[0]!=previous:raise TaskFault("TASK_STATE_CONFLICT")
            db.execute("UPDATE gateway_tasks SET state=?,exit_code=?,updated=? WHERE task_id=?",
                       (next_state,exit_code,self.clock(),task_id))
        return self._transaction(commit)

    def get(self,*,task_id=None,request_key=None):
        if (task_id is None)==(request_key is None):
            raise TaskFault("TASK_SELECTOR_INVALID")
        key=task_id if task_id is not None else request_key
        if not isinstance(key,str) or KEY.fullmatch(key) is None:
            raise TaskFault("TASK_SELECTOR_INVALID")
        field="task_id" if task_id is not None else "request_key"
        return self._transaction(lambda db:self.row(db.execute(
            "SELECT * FROM gateway_tasks WHERE "+field+"=?",(key,)).fetchone()))

    def list(self,box,limit=20):
        if (not isinstance(box,str) or BOX.fullmatch(box) is None
                or type(limit) is not int or not 1<=limit<=50):
            raise TaskFault("TASK_ARGUMENT_INVALID")
        return self._transaction(lambda db:[self.row(r) for r in db.execute(
            "SELECT * FROM gateway_tasks WHERE logical=? ORDER BY created DESC LIMIT ?",
            (box,limit))])

def provision_claims(folder):
    """Prepare only the minimum shared claim DB; leave dispatcher disabled."""
    claim=Path(folder)/"claims.sqlite3"
    if not claim.exists():
        fd=os.open(claim,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600)
        os.close(fd)
    if __package__:
        from .coordination_runtime import private_connection
        from .operation_coordination import OperationCoordinator
    else:
        from coordination_runtime import private_connection
        from operation_coordination import OperationCoordinator
    OperationCoordinator(lambda: private_connection(claim)).initialize()


def provision():
    """Explicit app-user setup, invoked only by the approved installer/host CLI."""
    uid=os.geteuid()
    if uid==0 or pwd.getpwuid(uid).pw_name!="boxweave":
        raise TaskFault("TASK_PROVISION_USER_REQUIRED")
    folder=DEFAULT_DB.parent
    if not folder.exists():
        folder.mkdir(mode=0o700)
    m=folder.lstat()
    if not stat.S_ISDIR(m.st_mode) or m.st_uid!=uid or m.st_mode&0o077:
        raise TaskFault("TASK_STORAGE_UNSAFE")
    if not DEFAULT_DB.exists():
        fd=os.open(DEFAULT_DB,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600)
        os.close(fd)
    TaskStore().initialize()
    provision_claims(folder)
    print(json.dumps({"state":"ready","private_metadata_only":True,
                      "coordination_prepared":True,"coordination_enabled":False}))

def original_recovery_receipt(operation_id, receipt_ref, identity_digest, *, tasks_path=DEFAULT_DB,
                              app_path=Path("/home/boxweave/boxweave-state/state.sqlite3")):
    """Best-effort original receipt read, never an authority to re-dispatch.

    A missing/mismatched origin is *explicitly unverified*, never fabricated.
    The operator's forced decision remains risky even when a receipt exists.
    """
    if receipt_ref == "gateway_" + operation_id:
        try:
            store = TaskStore(tasks_path)
            def read(db):
                row = db.execute("SELECT * FROM gateway_tasks WHERE task_id=?",(operation_id,)).fetchone()
                if row is None:raise TaskFault("TASK_NOT_FOUND")
                return row
            record = store._transaction(read)
            row = store.row(record)
            matched = hmac.compare_digest(record[3],identity_digest)
            return {"source":"gateway", "state":row["state"],
                    "logical":row["box"],"exact_original_task":True,
                    "identity_matches_claim":matched, "exit_code":row["exit_code"]}
        except (TaskFault, OSError):
            return {"source":"gateway", "state":"unverified",
                    "exact_original_task":False,"identity_matches_claim":False}
    table = {"environment_":"environment_operations",
             "web_":"box_action_receipts", "access_":"access_operations"}
    for prefix,name in table.items():
        if receipt_ref != prefix + operation_id:
            continue
        try:
            info=app_path.lstat()
            parent=app_path.parent.lstat()
            if (not stat.S_ISREG(info.st_mode) or info.st_uid!=os.geteuid()
                    or info.st_mode&0o077 or info.st_nlink!=1
                    or not stat.S_ISDIR(parent.st_mode) or parent.st_uid!=os.geteuid()
                    or parent.st_mode&0o077):
                raise OSError()
            db=sqlite3.connect("file:"+quote(str(app_path),safe="/")+"?mode=ro",uri=True)
            try:
                cols="state,logical" if name!="access_operations" else "state"
                row=db.execute(f"SELECT {cols} FROM {name} WHERE id=?",(operation_id,)).fetchone()
                if row is None:raise OSError()
                return {"source":prefix[:-1],"state":row[0],
                        "logical":row[1] if len(row)>1 else None,
                        "exact_original_task":True,
                        "identity_matches_claim":False}
            finally:db.close()
        except (OSError,sqlite3.Error):
            return {"source":prefix[:-1],"state":"unverified",
                    "exact_original_task":False,
                    "identity_matches_claim":False}
    return {"source":"unrecognized", "state":"unverified",
            "exact_original_task":False,"identity_matches_claim":False}


def recovery_cli(argv):
    """Only called by the privileged installed host command (never MCP)."""
    import argparse
    import sys
    if __package__:
        from .coordination_bridge import SharedClaims
        from .operation_coordination import CoordinationFault
    else:
        from coordination_bridge import SharedClaims
        from operation_coordination import CoordinationFault
    parser=argparse.ArgumentParser(prog="sudo boxweave task recover")
    parser.add_argument("action",choices=("inspect","release","audit"))
    parser.add_argument("operation_id")
    parser.add_argument("--reviewer",required=True)
    parser.add_argument("--challenge")
    parser.add_argument("--reason")
    parser.add_argument("--confirm")
    parser.add_argument("--acknowledge-remote-may-still-run",action="store_true")
    args=parser.parse_args(argv)
    try:
        coord=SharedClaims().coordinator
        if args.action=="inspect":
            if args.challenge or args.reason or args.confirm or args.acknowledge_remote_may_still_run:
                raise CoordinationFault("RECOVERY_ARGUMENT_INVALID")
            view=coord.status(args.operation_id)
            source=original_recovery_receipt(args.operation_id,view["receipt_ref"],view["identity_digest"])
            result=coord.recovery_review(args.operation_id,args.reviewer)
            result["original_receipt"]=source
        elif args.action=="release":
            if not all((args.challenge,args.reason,args.confirm,args.acknowledge_remote_may_still_run)):
                raise CoordinationFault("RECOVERY_CONFIRMATION_REQUIRED")
            result=coord.recovery_force_release(args.operation_id,args.reviewer,
                args.challenge,args.reason,args.confirm,args.acknowledge_remote_may_still_run)
        else:
            if args.challenge or args.reason or args.confirm or args.acknowledge_remote_may_still_run:
                raise CoordinationFault("RECOVERY_ARGUMENT_INVALID")
            result={"operation_id":args.operation_id,
                    "audits":coord.recovery_audit(args.operation_id)}
        print(json.dumps(result,sort_keys=True))
        return 0
    except (CoordinationFault,TaskFault) as error:
        print(json.dumps({"state":"refused","code":error.code}),file=sys.stderr)
        return 2
    except (OSError,ValueError):
        print(json.dumps({"state":"unavailable","code":"RECOVERY_UNAVAILABLE"}),file=sys.stderr)
        return 2


if __name__=="__main__":
    import sys
    if sys.argv[1:]==["provision"]:
        provision()
    elif len(sys.argv)>2 and sys.argv[1]=="recover":
        raise SystemExit(recovery_cli(sys.argv[2:]))
    else:
        raise SystemExit("TASK_HELPER_PROVISION_OR_RECOVERY_ONLY")
