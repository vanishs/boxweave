"""Lifecycle fixtures: one dispatch, exact identity and bounded read-only recovery."""
import base64
import json
import struct
from dataclasses import replace
from test_slice import Base, TARGET, HASH, PASSWORD
from bwmini.domain import Target, Fault
from bwmini.provider import Fixture, Provider
from bwmini.service import Service
from bwmini.store import Store
from bwmini.web import create_app

PUBLIC="ssh-ed25519 "+base64.b64encode(struct.pack(">I",11)+b"ssh-ed25519"+struct.pack(">I",32)+b"x"*32).decode()+" fixture"
NAME="v2-lifecycle-e2e"

class LifecycleFixture(Fixture):
    def __init__(self):
        super().__init__(TARGET)
        self.config={"targets":{TARGET.logical:TARGET.data()}}
        self.lifecycle_config={"allowed_names":[NAME],"cpu":"1","memory":"1GB","disk":"8GB","ssh_public_key":PUBLIC,"readiness_timeout":60}
        self.resources={}
        self.writes=[]
        self.ready_failures=0
        self.generation=1000
        self.lose_create=False
        self.inspect_count=0
        self.lazy_resize=False
        self.no_resources=False
    def lifecycle_inspect(self,logical):
        self.inspect_count+=1
        entry=self.resources.get(logical)
        if entry is not None and self.no_resources:
            entry={k:v for k,v in entry.items() if k!="resources"}
        return entry
    def lifecycle_mutate(self,action,logical,spec,timeout):
        self.writes.append((action,logical))
        if action=="create":
            self.generation+=1
            t=Target("containarium","dev","default",logical,logical+"-container",str(self.generation),logical,logical+"-container")
            self.resources[logical]={"target":t,"state":"CONTAINER_STATE_RUNNING","podman_enabled":False,
                                     "resources":{"cpu":"1","memory":"1GB","disk":"8GB"}}
            if self.lose_create:
                raise OSError("synthetic lost response")
        elif action=="start":self.resources[logical]["state"]="CONTAINER_STATE_RUNNING"
        elif action=="stop":self.resources[logical]["state"]="CONTAINER_STATE_STOPPED"
        elif action=="resize":
            if not self.lazy_resize:
                entry=self.resources[logical]
                entry["resources"]={**entry.get("resources",{}),**spec.get("_resize",{})}
        elif action=="delete":del self.resources[logical]
    def lifecycle_identity(self,target,timeout):
        if self.ready_failures:
            self.ready_failures-=1
            raise Fault("NOT_CONNECTION_READY")
        return {"user":target.user,"hostname":target.hostname,"stdout":target.user+"\n"+target.hostname+"\n","exit_code":0}

class LifecycleTests(Base):
    def setUp(self):
        super().setUp()
        self.provider=LifecycleFixture()
        self.s=Service(self.store,self.provider,clock=lambda:self.now[0])
        self.life=self.s.lifecycle
    def confirmed(self,action="create"):
        p=self.life.plan(action,NAME)
        return self.life.confirm(p["id"],NAME)
    def ready(self):
        op=self.confirmed()
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["state"],"succeeded")
        return op
    def test_create_stop_start_delete_and_receipt_restart(self):
        self.ready()
        for action,phase in [("stop","stopped"),("start","connection_ready"),("stop","stopped"),("delete","absent")]:
            op=self.confirmed(action)
            self.s.worker(once=True)
            row=self.life.receipt(op["id"])
            self.assertEqual((row["state"],row["phase"]),("succeeded",phase))
        again=Service(Store(self.store.path),self.provider,clock=lambda:self.now[0])
        again.worker(once=True)
        self.assertEqual(len(again.lifecycle.receipts()),5)
        self.assertEqual([x[0] for x in self.provider.writes],["create","stop","start","stop","delete"])
        self.assertTrue(self.life.settings()["environments"][0]["deleted"])
    def test_trust_failure_stops_observation_without_retry(self):
        def fail(*args):raise Fault("TRUST_FAILED")
        self.provider.lifecycle_identity=fail
        op=self.confirmed();self.s.worker(once=True)
        row=self.life.receipt(op["id"])
        self.assertEqual((row["state"],row["code"]),("failed","TRUST_FAILED"))
        count=self.provider.inspect_count
        self.now[0]+=30;self.s.worker(once=True)
        self.assertEqual(self.provider.inspect_count,count)
        self.assertEqual(self.provider.writes,[("create",NAME)])
    def test_running_is_not_connection_ready_and_deadline_not_reset(self):
        self.provider.ready_failures=2
        op=self.confirmed()
        self.s.worker(once=True)
        row=self.life.receipt(op["id"])
        self.assertEqual((row["state"],row["phase"]),("observing","running"))
        again=Service(Store(self.store.path),self.provider,clock=lambda:self.now[0])
        again.worker(once=True)
        self.assertEqual(len(self.provider.writes),1)
        self.now[0]+=61
        again.worker(once=True)
        row=again.lifecycle.receipt(op["id"])
        self.assertEqual((row["state"],row["deadline"]),("unknown",op["deadline"]))
    def test_lost_create_found_or_absent_never_recreates_or_adopts(self):
        self.provider.lose_create=True
        op=self.confirmed();self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["state"],"unknown")
        self.life.observe(op["id"])
        self.assertEqual(self.life.receipt(op["id"])["result"]["completion"],"unverified")
        del self.provider.resources[NAME]
        self.life.observe(op["id"])
        self.s.worker(once=True)
        self.assertEqual(self.provider.writes,[("create",NAME)])
        self.assertEqual(self.life.settings()["environments"],[])
        p=self.life.plan("create",NAME)
        with self.assertRaisesRegex(Fault,"OPERATION_PENDING"):self.life.confirm(p["id"],NAME)

    def test_resize_plan_confirm_and_observed_resources(self):
        self.ready()
        plan=self.life.plan("resize",NAME,resize={"memory":"2GB"})
        self.assertEqual(plan["resize"],{"memory":"2GB"})
        op=self.life.confirm(plan["id"],NAME)
        self.s.worker(once=True)
        row=self.life.receipt(op["id"])
        self.assertEqual((row["state"],row["phase"],row["code"]),("succeeded","resized","OK"))
        self.assertEqual(row["result"]["verification"],"observed")
        self.assertEqual(row["result"]["resources_observed"],{"memory":"2GB"})
        self.assertIn(("resize",NAME),self.provider.writes)

    def test_resize_rejects_bad_spec_and_wrong_action(self):
        self.ready()
        for spec in ({},{"cpu":"9"},{"disk":"3GB"},{"bogus":"1"},None):
            with self.subTest(spec=spec),self.assertRaisesRegex(Fault,"RESIZE_SPEC_INVALID"):
                self.life.plan("resize",NAME,resize=spec)
        with self.assertRaisesRegex(Fault,"RESIZE_RESIZE_ONLY"):
            self.life.plan("create",NAME,resize={"cpu":"2"})
        with self.assertRaisesRegex(Fault,"ACTION_NOT_ALLOWED"):
            self.life.plan("pause",NAME)

    def test_resize_mismatch_keeps_observing_until_deadline(self):
        self.ready()
        self.provider.lazy_resize=True
        plan=self.life.plan("resize",NAME,resize={"cpu":"4"})
        op=self.life.confirm(plan["id"],NAME)
        self.s.worker(once=True)
        row=self.life.receipt(op["id"])
        self.assertEqual((row["state"],row["phase"]),("observing","resizing"))
        self.now[0]+=61
        self.s.worker(once=True)
        row=self.life.receipt(op["id"])
        self.assertEqual((row["state"],row["code"]),("unknown","DEADLINE_EXCEEDED"))

    def test_resize_without_observed_resources_succeeds_unverified(self):
        self.ready()
        self.provider.no_resources=True
        plan=self.life.plan("resize",NAME,resize={"disk":"20GB"})
        op=self.life.confirm(plan["id"],NAME)
        self.s.worker(once=True)
        row=self.life.receipt(op["id"])
        self.assertEqual((row["state"],row["code"]),("succeeded","OK"))
        self.assertEqual(row["result"]["verification"],"not_available")
    def test_used_wrong_actor_and_wrong_confirmation(self):
        p=self.life.plan("create",NAME)
        with self.assertRaisesRegex(Fault,"NOT_FOUND"):self.life.confirm(p["id"],NAME,"other")
        with self.assertRaisesRegex(Fault,"EXACT_TARGET"):self.life.confirm(p["id"],"wrong")
        self.life.confirm(p["id"],NAME)
        with self.assertRaisesRegex(Fault,"PLAN_EXPIRED_OR_USED"):self.life.confirm(p["id"],NAME)
        with self.assertRaisesRegex(Fault,"NOT_FOUND"):self.life.receipt(self.life.receipts()[0]["id"],"other")
    def test_replaced_identity_or_state_stops_before_mutation(self):
        self.ready()
        plan=self.life.plan("stop",NAME)
        old=self.provider.resources[NAME]["target"]
        self.provider.resources[NAME]["target"]=replace(old,created="replacement")
        with self.assertRaisesRegex(Fault,"TARGET_CHANGED"):self.life.confirm(plan["id"],NAME)
        self.assertEqual(len(self.provider.writes),1)
        self.provider.resources[NAME]["target"]=old
        op=self.life.confirm(plan["id"],NAME)
        self.provider.resources[NAME]["state"]="CONTAINER_STATE_STOPPED"
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["code"],"STALE_RESOURCE_STATE")
        self.assertEqual(len(self.provider.writes),1)
    def test_worker_lost_before_ack_unknown_after_ack_observation_only(self):
        op=self.confirmed()
        with self.store.db() as db:db.execute("UPDATE environment_operations SET state='dispatching' WHERE id=?",(op["id"],))
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["state"],"unknown")
        self.assertEqual(self.provider.writes,[])
    def test_ack_saved_then_restart_only_observes(self):
        op=self.confirmed()
        self.provider.lifecycle_mutate("create",NAME,self.provider.lifecycle_config,60)
        with self.store.db() as db:db.execute("UPDATE environment_operations SET state='dispatching',acknowledged=1 WHERE id=?",(op["id"],))
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["state"],"succeeded")
        self.assertEqual(self.provider.writes,[("create",NAME)])
    def test_static_core_and_unmanaged_targets_refused(self):
        for logical in [TARGET.logical,"containarium-core-postgresql","unrelated"]:
            with self.assertRaisesRegex(Fault,"TARGET_NOT_ALLOWED"):self.life.plan("delete",logical)
        with self.assertRaisesRegex(Fault,"MANAGED_TARGET_REQUIRED"):self.life.plan("stop",NAME)
        with self.assertRaisesRegex(Fault,"ACTION_NOT_ALLOWED"):self.life.plan("shell",NAME)
    def test_config_change_and_clock_rollback_prevent_write(self):
        op=self.confirmed()
        self.provider.lifecycle_config["disk"]="10GB"
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["code"],"CONFIG_CHANGED")
        self.assertEqual(self.provider.writes,[])
        self.provider.lifecycle_config["disk"]="8GB"
        op=self.confirmed()
        self.now[0]=900
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["code"],"CLOCK_ROLLBACK")
        self.assertEqual(self.provider.writes,[])
    def test_profile_requires_full_public_key_and_protects_static_resources(self):
        self.provider.lifecycle_config["ssh_public_key"]="SHA256:fingerprint"
        with self.assertRaisesRegex(Fault,"MANAGED_PUBLIC_KEY_REQUIRED"):
            Service(self.store,self.provider)
        self.provider.lifecycle_config["ssh_public_key"]=PUBLIC
        self.provider.lifecycle_config["allowed_names"]=[TARGET.logical]
        with self.assertRaisesRegex(Fault,"LIFECYCLE_CONFIG_INVALID"):
            Service(self.store,self.provider)
    def test_web_origin_csrf_and_extra_override(self):
        self.store.admin(HASH)
        client=create_app(self.s,"http://localhost",self.root,secure=False).test_client()
        self.assertEqual(client.get("/api/environments").status_code,401)
        response=client.post("/api/login",json={"password":PASSWORD},headers={"Origin":"http://localhost"})
        headers={"Origin":"http://localhost","X-CSRF-Token":response.json["csrf"]}
        self.assertEqual(client.post("/api/environment-plans",json={"action":"create","logical":NAME,"host":"other"},headers=headers).status_code,400)
        self.assertEqual(client.post("/api/environment-plans",json={"action":"create","logical":NAME},headers={"Origin":"http://localhost"}).status_code,403)
        plan=client.post("/api/environment-plans",json={"action":"create","logical":NAME},headers=headers)
        self.assertEqual(plan.status_code,201)
        op=client.post("/api/environment-plans/"+plan.json["id"]+"/confirm",json={"confirm":True,"logical":NAME},headers=headers)
        self.assertEqual(op.status_code,202)
        self.s.worker(once=True)
        self.assertEqual(client.get("/api/environment-operations/"+op.json["id"]).json["state"],"succeeded")
    def test_delete_requires_stopped_and_no_force(self):
        self.ready()
        with self.assertRaisesRegex(Fault,"RESOURCE_STATE_NOT_ALLOWED"):self.life.plan("delete",NAME)
        self.assertEqual(self.provider.writes,[("create",NAME)])

    def test_created_environment_can_bind_execute_and_deletion_keeps_project(self):
        self.ready()
        p=self.s.bind("Managed project",NAME,"/home/"+NAME)
        op=self.s.confirm(self.s.plan(p["id"],1)["id"])
        self.s.worker(once=True)
        self.assertEqual(self.store.receipt(op["id"])["state"],"succeeded")
        for action in ("stop","delete"):
            self.confirmed(action);self.s.worker(once=True)
        self.assertEqual(self.store.project(p["id"])["name"],"Managed project")
        with self.assertRaisesRegex(Fault,"TARGET_CHANGED"):
            self.s.plan(p["id"],1)

    def test_transport_scope_change_blocks_queued_write_and_unknown_observation(self):
        op=self.confirmed()
        self.provider.config["backend"]="different-reviewed-backend"
        self.s.worker(once=True)
        self.assertEqual(self.life.receipt(op["id"])["code"],"CONFIG_CHANGED")
        self.assertEqual(self.provider.writes,[])
        self.provider.config.pop("backend")
        op=self.confirmed()
        with self.store.db() as db:db.execute("UPDATE environment_operations SET state='unknown' WHERE id=?",(op["id"],))
        count=self.provider.inspect_count
        self.provider.config["project"]="other-project"
        self.assertEqual(self.life.observe(op["id"])["code"],"CONFIG_CHANGED")
        self.assertEqual(self.provider.inspect_count,count)

class AdapterContractTests(Base):
    def configured(self):
        import sys
        p=Provider({"provider":"containarium","backend":"dev","project":"default",
                    "launcher":[sys.executable],"targets":{TARGET.logical:TARGET.data()}})
        return p
    def test_inventory_requires_complete_unique_count_before_absence(self):
        p=self.configured()
        class Session:
            raw=""
            def __enter__(self):return self
            def __exit__(self,*args):pass
            def discover(self):return {}
            def tool(self,*args):return {"content":[{"type":"text","text":self.raw}]}
        session=Session();p.session=lambda *args:session
        session.raw="Found 1 container(s):\n\n   Username: existing\n"
        self.assertEqual(p.lifecycle_inventory(),{"existing"})
        for raw in ["unknown error","Found 2 container(s):\n   Username: existing",
                    "Found 2 container(s):\n   Username: existing\n   Username: existing\n",
                    '{"containers":[],"nextCursor":"more"}']:
            session.raw=raw
            with self.assertRaisesRegex(Fault,"INVENTORY_CONTRACT_UNSUPPORTED"):p.lifecycle_inventory()
    def test_mutation_arguments_always_include_full_key_and_never_force(self):
        from unittest.mock import patch
        p=self.configured();calls=[]
        class Session:
            def __init__(self,*args):pass
            def __enter__(self):return self
            def __exit__(self,*args):pass
            def discover(self):
                props={k:{} for k in ["username","cpu","memory","disk","image","ssh_keys","enable_podman","force","waitForReady"]}
                return {n:{"inputSchema":{"properties":props}} for n in ["create_container","start_container","stop_container","delete_container"]}
            def tool(self,n,args):calls.append((n,args))
        spec=LifecycleFixture().lifecycle_config
        with patch("bwmini.provider.MCP",Session):
            for action in ("create","stop","start","delete"):p.lifecycle_mutate(action,NAME,spec,1800)
        self.assertEqual(calls[0][1]["ssh_keys"],[PUBLIC])
        self.assertIs(calls[0][1]["enable_podman"],False)
        self.assertIs(calls[1][1]["force"],False)
        self.assertIs(calls[2][1]["waitForReady"],False)
        self.assertIs(calls[3][1]["force"],False)
    def test_unsupported_create_key_contract_fails_without_tool_call(self):
        from unittest.mock import patch
        p=self.configured()
        class Session:
            def __init__(self,*args):pass
            def __enter__(self):return self
            def __exit__(self,*args):pass
            def discover(self):return {"create_container":{"inputSchema":{"properties":{"username":{}}}}}
            def tool(self,*args):raise AssertionError("Must not create without explicit key")
        with patch("bwmini.provider.MCP",Session),self.assertRaisesRegex(Fault,"CONTRACT_UNSUPPORTED"):
            p.lifecycle_mutate("create",NAME,LifecycleFixture().lifecycle_config,60)
