"""Exact app-owned lifecycle plans, durable single dispatch and observation-only recovery."""
import base64
import hashlib
from pathlib import Path
import json
import re
import time
import uuid
from . import diagnostics
from .domain import Target, Fault

SCHEMA = """
CREATE TABLE environment_plans (id TEXT PRIMARY KEY, actor TEXT NOT NULL, action TEXT NOT NULL,
 logical TEXT NOT NULL, spec TEXT NOT NULL, target TEXT, expected_state TEXT, expires REAL NOT NULL, used INTEGER NOT NULL);
CREATE TABLE environment_operations (id TEXT PRIMARY KEY, actor TEXT NOT NULL, action TEXT NOT NULL,
 logical TEXT NOT NULL, spec TEXT NOT NULL, target TEXT, expected_state TEXT, state TEXT NOT NULL,
 phase TEXT NOT NULL, created REAL NOT NULL, deadline REAL NOT NULL, next_poll REAL NOT NULL,
 acknowledged INTEGER NOT NULL, code TEXT, result TEXT, evidence TEXT NOT NULL);
CREATE UNIQUE INDEX environment_one_pending ON environment_operations(logical)
 WHERE state IN ('queued','dispatching','observing','unknown');
CREATE TABLE managed_environments (logical TEXT PRIMARY KEY, target TEXT NOT NULL,
 created_by TEXT NOT NULL REFERENCES environment_operations(id), deleted INTEGER NOT NULL);
CREATE TABLE environment_events (operation TEXT NOT NULL REFERENCES environment_operations(id),
 at REAL NOT NULL, state TEXT NOT NULL, phase TEXT NOT NULL, code TEXT NOT NULL);
"""

RESIZE = {"cpu": ("1", "2", "3", "4", "5", "6", "7", "8"),
          "memory": ("1GB", "2GB", "4GB", "8GB"),
          "disk": ("8GB", "10GB", "20GB", "25GB", "30GB", "40GB", "50GB", "100GB")}

def uid():
    return uuid.uuid4().hex

class Lifecycle:
    def __init__(self, store, provider, clock=time.time):
        self.store, self.provider, self.clock = store, provider, clock
        self.config = vars(provider).get("lifecycle_config")
        if self.config is not None:
            c = self.config
            keys = {"allowed_names","cpu","memory","disk","ssh_public_key","readiness_timeout"}
            if not isinstance(c, dict) or set(c) != keys:
                raise Fault("LIFECYCLE_CONFIG_INVALID",422)
            names = c["allowed_names"]
            protected = set(vars(provider).get("config",{}).get("targets",{}))
            if (not isinstance(names,list) or not 1 <= len(names) <= 32
                    or any(not isinstance(n,str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}",n) is None
                           or n.startswith("containarium-core-") or n in protected for n in names)
                    or len(set(names)) != len(names)):
                raise Fault("LIFECYCLE_CONFIG_INVALID",422)
            if (c["cpu"] not in RESIZE["cpu"]
                    or c["memory"] not in RESIZE["memory"]
                    or c["disk"] not in RESIZE["disk"]
                    or type(c["readiness_timeout"]) is not int or not 1 <= c["readiness_timeout"] <= 1800):
                raise Fault("LIFECYCLE_CONFIG_INVALID",422)
            try:
                key = c["ssh_public_key"]
                if not isinstance(key,str) or len(key)>1024 or re.fullmatch(
                        r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}(?: [A-Za-z0-9@._-]{1,80})?",key) is None:
                    raise ValueError()
                blob = base64.b64decode(key.split()[1],validate=True)
                if len(blob)!=51 or blob[:19] != b"\x00\x00\x00\x0bssh-ed25519\x00\x00\x00\x20":
                    raise ValueError()
            except (ValueError,TypeError):
                raise Fault("MANAGED_PUBLIC_KEY_REQUIRED",422) from None

    def scope(self):
        config=vars(self.provider).get("config",{})
        argv=vars(self.provider).get("argv",[])
        files=[]
        for index,arg in enumerate(argv):
            path=None
            if index==0:path=arg
            elif arg.startswith("UserKnownHostsFile="):path=arg.split("=",1)[1]
            elif index>0 and argv[index-1]=="-i":path=arg
            if path:
                try:
                    info=Path(path).stat()
                    files.append([path,info.st_dev,info.st_ino,info.st_mtime_ns,info.st_size,info.st_mode])
                except OSError:
                    files.append([path,"unavailable"])
        value={"provider":config.get("provider"),"backend":config.get("backend"),"project":config.get("project"),"launcher":argv,"files":files,"access":config.get("access")}
        return hashlib.sha256(json.dumps(value,sort_keys=True).encode()).hexdigest()

    def spec(self):
        return {**self.config,"_scope":self.scope()}

    def matches_spec(self,spec):
        return {k:v for k,v in spec.items() if k not in ("_network_tenant","_nesting","_resize")}==self.spec()

    def network_check(self,spec):
        if "_nesting" in spec:
            check=getattr(self.provider,"nesting_check",None)
            if check is None:raise Fault("INDEPENDENT_NESTING_UNSUPPORTED",409)
            check(spec["_nesting"])
        if "_network_tenant" in spec:
            network=vars(self).get("network")
            if network is None:raise Fault("SHARED_CREATE_UNSUPPORTED")
            network.shared_check(spec["_network_tenant"])

    def settings(self):
        if self.config is None:
            return {"enabled":False,"environments":[]}
        with self.store.db() as db:
            rows = [dict(r) for r in db.execute("SELECT logical,target,deleted FROM managed_environments ORDER BY logical")]
        return {"enabled":True,"allowed_names":self.config["allowed_names"],
                "resources":{k:self.config[k] for k in ("cpu","memory","disk")},
                "resize":{k:list(v) for k,v in RESIZE.items()},
                "readiness_timeout":self.config["readiness_timeout"],
                "shared_network_tenants":(vars(self).get("network").config or {}).get("tenants",[]) if vars(self).get("network") else [],
                "environments":[{"logical":r["logical"],"target":json.loads(r["target"]),"deleted":bool(r["deleted"])} for r in rows]}

    def check_name(self, logical):
        if self.config is None:
            raise Fault("LIFECYCLE_DISABLED",409)
        if not isinstance(logical,str) or logical not in self.config["allowed_names"]:
            raise Fault("TARGET_NOT_ALLOWED",403)

    def managed(self, logical):
        self.check_name(logical)
        with self.store.db() as db:
            r = db.execute("SELECT * FROM managed_environments WHERE logical=? AND deleted=0",(logical,)).fetchone()
        if r is None:
            raise Fault("MANAGED_TARGET_REQUIRED",403)
        return Target(**json.loads(r["target"]))

    def preflight(self, action, logical, target, expected_state):
        observed = self.provider.lifecycle_inspect(logical)
        if action == "create":
            if observed is not None:
                raise Fault("RESOURCE_EXISTS")
            with self.store.db() as db:
                r = db.execute("SELECT deleted FROM managed_environments WHERE logical=?",(logical,)).fetchone()
            if r is not None and not r["deleted"]:
                raise Fault("MANAGED_RESOURCE_MISSING_REVIEW_REQUIRED")
        else:
            pinned = self.managed(logical)
            if observed is None or observed["target"] != target or pinned != target:
                raise Fault("TARGET_CHANGED")
            if observed["state"] != expected_state:
                raise Fault("STALE_RESOURCE_STATE")

    def plan(self, action, logical, actor="operator", network_tenant=None, nesting=None, resize=None):
        self.check_name(logical)
        if action not in ("create","start","stop","delete","resize"):
            raise Fault("ACTION_NOT_ALLOWED",400)
        if resize is not None and action!="resize":
            raise Fault("RESIZE_RESIZE_ONLY",422)
        target, state = None, None
        if action != "create":
            target = self.managed(logical)
            value = self.provider.lifecycle_inspect(logical)
            if value is None or value["target"] != target:
                raise Fault("TARGET_CHANGED")
            state = value["state"]
            if action == "resize":
                pass  # a running or stopped managed box is both resizable
            else:
                wanted = "CONTAINER_STATE_RUNNING" if action=="stop" else "CONTAINER_STATE_STOPPED"
                if state != wanted:
                    raise Fault("RESOURCE_STATE_NOT_ALLOWED")
        self.preflight(action,logical,target,state)
        spec = self.spec()
        if action == "resize":
            if (not isinstance(resize,dict) or not resize
                    or set(resize)-set(RESIZE) or any(resize.get(k) not in RESIZE[k] for k in resize)):
                raise Fault("RESIZE_SPEC_INVALID",422)
            spec["_resize"]=dict(resize)
        if network_tenant is not None:
            if action!="create":raise Fault("NETWORK_TENANT_CREATE_ONLY",422)
            spec["_network_tenant"]=network_tenant
            self.network_check(spec)
        if nesting is not None:
            if action!="create":raise Fault("NESTING_CREATE_ONLY",422)
            spec["_nesting"]=nesting
            self.network_check(spec)
        ident = uid()
        with self.store.db() as db:
            db.execute("INSERT INTO environment_plans VALUES(?,?,?,?,?,?,?,?,0)",
                       (ident,actor,action,logical,json.dumps(spec),json.dumps(target.data()) if target else None,state,self.clock()+300))
        return {"id":ident,"action":action,"logical":logical,"target":target.data() if target else None,
                "expected_state":state,"resources":{k:spec[k] for k in ("cpu","memory","disk")},
                "resize":spec.get("_resize"),
                "readiness_timeout":spec["readiness_timeout"],"network_tenant":spec.get("_network_tenant"),"nesting":spec.get("_nesting"),"force":False,"scope":{k:vars(self.provider).get("config",{}).get(k) for k in ("provider","backend","project")}}

    def confirm(self, plan, logical, actor="operator"):
        with self.store.db() as db:
            row = db.execute("SELECT * FROM environment_plans WHERE id=? AND actor=?",(plan,actor)).fetchone()
        if row is None:
            raise Fault("NOT_FOUND",404)
        row = dict(row)
        if logical != row["logical"]:
            raise Fault("EXACT_TARGET_CONFIRMATION_REQUIRED",400)
        self.check_name(logical)
        if not self.matches_spec(json.loads(row["spec"])):
            raise Fault("CONFIG_CHANGED")
        self.network_check(json.loads(row["spec"]))
        if row["used"] or self.clock()>row["expires"]:
            raise Fault("PLAN_EXPIRED_OR_USED")
        target = Target(**json.loads(row["target"])) if row["target"] else None
        self.preflight(row["action"],logical,target,row["expected_state"])
        now = self.clock()
        ident = uid()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            current = db.execute("SELECT used,expires FROM environment_plans WHERE id=? AND actor=?",(plan,actor)).fetchone()
            if current is None or current["used"] or now>current["expires"]:
                raise Fault("PLAN_EXPIRED_OR_USED")
            if db.execute("SELECT 1 FROM environment_operations WHERE logical=? AND state IN ('queued','dispatching','observing','unknown')",(logical,)).fetchone():
                raise Fault("OPERATION_PENDING")
            db.execute("UPDATE environment_plans SET used=1 WHERE id=?",(plan,))
            db.execute("INSERT INTO environment_operations VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                       (ident,actor,row["action"],logical,row["spec"],row["target"],row["expected_state"],
                        "queued","queued",now,now+self.config["readiness_timeout"],now,0,None,None,"fixture" if self.provider.mode=="fixture" else "live_mcp"))
            diagnostics.link(db, actor, "environment", ident)
            db.execute("INSERT INTO environment_events VALUES(?,?,?,?,?)",(ident,now,"queued","queued","CONFIRMED"))
        return self.receipt(ident,actor)

    def receipt(self, ident, actor="operator"):
        with self.store.db() as db:
            row = db.execute("SELECT * FROM environment_operations WHERE id=? AND actor=?",(ident,actor)).fetchone()
        if row is None:
            raise Fault("NOT_FOUND",404)
        d = dict(row)
        d.pop("spec")
        for key in ("target","result"):
            d[key] = json.loads(d[key]) if d[key] else None
        d["acknowledged"] = bool(d["acknowledged"])
        return d

    def receipts(self, actor="operator"):
        with self.store.db() as db:
            return [dict(r) for r in db.execute(
                "SELECT id,action,logical,state,phase,code,created,deadline,evidence FROM environment_operations WHERE actor=? ORDER BY created DESC,id DESC LIMIT 100",(actor,))]

    def save(self, job, state, phase, code, target=None, result=None, acknowledged=None):
        encoded = json.dumps(target.data()) if target else job["target"]
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT state FROM environment_operations WHERE id=?",(job["id"],)).fetchone()
            if row is None or row["state"] in ("succeeded","failed"):
                return
            db.execute("UPDATE environment_operations SET state=?,phase=?,code=?,target=?,result=?,next_poll=?,acknowledged=? WHERE id=?",
                       (state,phase,code,encoded,json.dumps(result) if result is not None else None,
                        self.clock()+15,job["acknowledged"] if acknowledged is None else acknowledged,job["id"]))
            db.execute("INSERT INTO environment_events VALUES(?,?,?,?,?)",
                       (job["id"],self.clock(),state,phase,code))
            if state=="succeeded" and job["action"]=="create":
                db.execute("INSERT INTO managed_environments VALUES(?,?,?,0) ON CONFLICT(logical) DO UPDATE SET target=excluded.target,created_by=excluded.created_by,deleted=0",
                           (job["logical"],encoded,job["id"]))
            if state=="succeeded" and job["action"]=="delete":
                db.execute("UPDATE managed_environments SET deleted=1 WHERE logical=? AND target=?",(job["logical"],encoded))

    def recover(self):
        with self.store.db() as db:
            rows = [dict(r) for r in db.execute(
                "SELECT id,actor FROM environment_operations WHERE state='dispatching'")]
            db.execute("UPDATE environment_operations SET state=CASE acknowledged WHEN 1 THEN 'observing' ELSE 'unknown' END,code='WORKER_LOST' WHERE state='dispatching'")
        for job in rows:
            trace = diagnostics.trace_for_operation(self.store, job["actor"], "environment", job["id"]) \
                or diagnostics.new_trace_id()
            diagnostics.emit(self.store, component="worker", phase="recovered", event_type="recovered",
                             outcome="unknown", code="WORKER_LOST", trace_id=trace,
                             actor_scope=job["actor"], operation_kind="environment", operation_id=job["id"])

    def observe(self, ident, actor="operator"):
        self.receipt(ident,actor)
        with self.store.db() as db:
            job = dict(db.execute("SELECT * FROM environment_operations WHERE id=? AND actor=?",(ident,actor)).fetchone())
        if job["state"] in ("queued","dispatching","succeeded","failed"):
            return self.receipt(ident,actor)
        if not self.matches_spec(json.loads(job["spec"])):
            self.save(job,"unknown","scope_changed","CONFIG_CHANGED")
            return self.receipt(ident,actor)
        if job["state"]=="observing" and (self.clock()<job["created"] or self.clock()>=job["deadline"]):
            self.save(job,"unknown","deadline","DEADLINE_EXCEEDED")
            return self.receipt(ident,actor)
        target = Target(**json.loads(job["target"])) if job["target"] else None
        try:
            value = self.provider.lifecycle_inspect(job["logical"])
            if job["state"]=="unknown" or not job["acknowledged"]:
                self.save(job,"unknown","reconciled","OUTCOME_UNKNOWN",result={
                    "resource_state":value["state"] if value else "absent",
                    "candidate":value["target"].data() if value else None,
                    "completion":"unverified","retry":"not_authorized"})
                return self.receipt(ident,actor)
            selected_tenant=json.loads(job["spec"]).get("_network_tenant")
            if selected_tenant and value and value.get("network_tenant")!=selected_tenant:
                raise Fault("NETWORK_MEMBERSHIP_MISMATCH")
            if target and value and value["target"]!=target:
                raise Fault("TARGET_CHANGED")
            if job["action"]=="delete":
                if value is None:
                    self.save(job,"succeeded","absent","OK",target=target,
                              result={"resource_state":"absent","readiness":"not_applicable","outcome":"observed_desired_state"})
                else:
                    self.save(job,"observing","deleting","WAITING_FOR_ABSENCE",target=target)
            elif value is None:
                self.save(job,"observing","submitted","WAITING_FOR_RESOURCE",target=target)
            elif job["action"]=="resize":
                requested=json.loads(job["spec"]).get("_resize") or {}
                observed=value.get("resources")
                if not requested or observed is None:
                    self.save(job,"succeeded","resized","OK",target=target,result={
                        "resources_requested":requested,"resources_observed":None,
                        "verification":"not_available","resource_state":value["state"],
                        "outcome":"observed_desired_state"})
                elif all(observed.get(k)==v for k,v in requested.items()):
                    self.save(job,"succeeded","resized","OK",target=target,result={
                        "resources_requested":requested,
                        "resources_observed":{k:observed.get(k) for k in requested},
                        "verification":"observed","resource_state":value["state"],
                        "outcome":"observed_desired_state"})
                else:
                    self.save(job,"observing","resizing","WAITING_FOR_RESIZE",target=target)
            elif job["action"]=="stop":
                state = "succeeded" if value["state"]=="CONTAINER_STATE_STOPPED" else "observing"
                self.save(job,state,"stopped" if state=="succeeded" else "stopping","OK" if state=="succeeded" else "WAITING_FOR_STOP",target=target,
                          result={"resource_state":value["state"],"readiness":"not_checked","outcome":"observed_desired_state"} if state=="succeeded" else None)
            else:
                target = target or value["target"]
                if value["state"]!="CONTAINER_STATE_RUNNING":
                    self.save(job,"observing","submitted","NOT_RUNNING",target=target)
                else:
                    self.save(job,"observing","running","CONNECTION_READINESS_PENDING",target=target)
                    remaining=job["deadline"]-self.clock()
                    if remaining<=0:
                        raise Fault("DEADLINE_EXCEEDED")
                    identity=self.provider.lifecycle_identity(target,remaining)
                    self.save(job,"succeeded","connection_ready","OK",target=target,result={
                        "resource_state":value["state"],"readiness":"identity_verified",
                        "identity":identity,"network_tenant":value.get("network_tenant"),"account_user":target.user,"observed_podman":value.get("podman_enabled"),
                        "requested_podman":False,"requested_nesting":json.loads(job["spec"]).get("_nesting"),"effective_nested_execution":"not_verified","nesting_restart_guidance":"Restart this configured box before relying on a changed nesting capability; verify child MCP execution separately." if "_nesting" in json.loads(job["spec"]) else None,"remote_cleanup":"unverified","outcome":"observed_desired_state"})
        except Fault as error:
            if error.code=="STATE_UNAVAILABLE":
                raise
            if error.code in ("TARGET_CHANGED","IDENTITY_MISMATCH","TRUST_FAILED","NETWORK_MEMBERSHIP_MISMATCH"):
                self.save(job,"failed","blocked",error.code)
            elif self.clock()>=job["deadline"] or error.code=="DEADLINE_EXCEEDED":
                self.save(job,"unknown","deadline","DEADLINE_EXCEEDED")
            else:
                self.save(job,"observing" if job["acknowledged"] and job["state"]!="unknown" else "unknown",
                          "running" if target else "submitted","OBSERVATION_UNAVAILABLE",target=target)
        return self.receipt(ident,actor)

    def work_one(self):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row=db.execute("SELECT * FROM environment_operations WHERE state='queued' OR (state='observing' AND next_poll<=?) ORDER BY created,id LIMIT 1",(self.clock(),)).fetchone()
            if row is None:
                return False
            job=dict(row)
            now=self.clock()
            if now<job["created"] or now>=job["deadline"]:
                code="CLOCK_ROLLBACK" if now<job["created"] else "DEADLINE_EXCEEDED"
                db.execute("UPDATE environment_operations SET state=?,phase='deadline',code=? WHERE id=?",
                           ("failed" if job["state"]=="queued" else "unknown",code,job["id"]))
                trace = diagnostics.trace_for_operation(self.store, job["actor"], "environment", job["id"]) \
                    or diagnostics.new_trace_id()
                diagnostics.emit(self.store, component="lifecycle", phase="deadline", event_type="observed",
                                 outcome="failed", code=code, trace_id=trace,
                                 actor_scope=job["actor"], operation_kind="environment", operation_id=job["id"])
                return True
            execute=job["state"]=="queued"
            if execute:
                db.execute("UPDATE environment_operations SET state='dispatching',phase='submitted' WHERE id=?",(job["id"],))
        trace = diagnostics.trace_for_operation(self.store, job["actor"], "environment", job["id"]) \
            or diagnostics.new_trace_id()
        if execute:
            started = time.monotonic()
            diagnostics.emit(self.store, component="lifecycle", phase="claimed", event_type="claimed",
                             outcome="passed", code="CLAIMED", trace_id=trace,
                             actor_scope=job["actor"], operation_kind="environment", operation_id=job["id"])
            try:
                self.check_name(job["logical"])
                if not self.matches_spec(json.loads(job["spec"])):
                    raise Fault("CONFIG_CHANGED")
                target=Target(**json.loads(job["target"])) if job["target"] else None
                self.network_check(json.loads(job["spec"]))
                self.preflight(job["action"],job["logical"],target,job["expected_state"])
                self.provider.lifecycle_mutate(job["action"],job["logical"],json.loads(job["spec"]),job["deadline"]-self.clock())
                self.save(job,"observing","submitted","ACKNOWLEDGED",acknowledged=1)
                diagnostics.emit(self.store, component="lifecycle", phase="submitted", event_type="acknowledged",
                                 outcome="passed", code="ACKNOWLEDGED", trace_id=trace,
                                 actor_scope=job["actor"], operation_kind="environment", operation_id=job["id"],
                                 elapsed_ms=int((time.monotonic()-started)*1000))
            except Fault as error:
                if error.code=="STATE_UNAVAILABLE":
                    raise
                known={"TARGET_CHANGED","TARGET_NOT_ALLOWED","RESOURCE_EXISTS","STALE_RESOURCE_STATE","CONFIG_CHANGED","MANAGED_TARGET_REQUIRED","MANAGED_RESOURCE_MISSING_REVIEW_REQUIRED","CONTRACT_UNSUPPORTED","LIFECYCLE_DISABLED","TENANT_NOT_ALLOWED","ACCESS_DISABLED","SHARED_CREATE_UNSUPPORTED","NETWORK_CONTRACT_UNSUPPORTED","GUARD_NOT_READY","RESIZE_SPEC_INVALID","PROVIDER_REJECTED"}
                classified = error.code in known
                self.save(job,"failed" if classified else "unknown","not_submitted" if classified else "submitted",
                          error.code if classified else "OUTCOME_UNKNOWN")
                diagnostics.emit(self.store, component="lifecycle",
                                 phase="not_submitted" if classified else "submitted",
                                 event_type="rejected" if classified else "unknown",
                                 outcome="failed" if classified else "unknown",
                                 code=getattr(error,"error_class",None) or (error.code if classified else "OUTCOME_UNKNOWN"),
                                 trace_id=trace, actor_scope=job["actor"],
                                 operation_kind="environment", operation_id=job["id"],
                                 elapsed_ms=int((time.monotonic()-started)*1000))
                return True
            except Exception:
                self.save(job,"unknown","submitted","OUTCOME_UNKNOWN")
                diagnostics.emit(self.store, component="lifecycle", phase="submitted", event_type="unknown",
                                 outcome="unknown", code="UNCLASSIFIED_EXCEPTION", trace_id=trace,
                                 actor_scope=job["actor"], operation_kind="environment",
                                 operation_id=job["id"], elapsed_ms=int((time.monotonic()-started)*1000))
                return True
        self.observe(job["id"],job["actor"])
        return True
