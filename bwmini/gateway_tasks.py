"""Gateway-private metadata-only idempotent Box task receipts.

No command body, stdout/stderr, client token, or secret is persisted.
The shared coordinator belongs to a different minimal DB and will be
integrated only after verified cross-entry dispatch claims.
"""
import hashlib
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
                     "exec":{"type":"string"}},
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
    m=re.match(r"\Aexit_code: (-?\d+)\n\n--- stdout ---\n",text)
    if m and len(text)<=2097152:
        return int(m[1])
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

if __name__=="__main__":
    import sys
    if sys.argv[1:]!=["provision"]:
        raise SystemExit("TASK_HELPER_PROVISION_ONLY")
    provision()
