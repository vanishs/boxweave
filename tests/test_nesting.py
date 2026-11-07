"""Nesting trust, stale-plan, compatibility and independent-dispatch regressions."""
import copy, io, json, unittest
from unittest.mock import patch
from bwmini import nesting, mcp_gate
from bwmini.domain import Fault
from bwmini.provider import Provider
from test_lifecycle import LifecycleFixture, NAME, PUBLIC
from test_slice import Base
from bwmini.service import Service

class NestingBoundaryTests(unittest.TestCase):
    def target(self):
        return {"name":"work-container","type":"container","created_at":"2026-10-29T00:00:00Z",
                "status":"Running","config":{"volatile.uuid":"incarnation-1","user.containarium.label.os-type":"ubuntu_2404"},
                "expanded_config":{"security.nesting":"false","security.privileged":"false"},
                "expanded_devices":{"eth0":{"network":"incusbr0"}}}

    def test_plan_is_read_only_apply_preserves_other_settings_and_never_restarts(self):
        item=self.target(); calls=[]
        def run(argv):
            calls.append(argv)
            if "query" in argv:return json.dumps(item)
            self.assertEqual(argv[-2:],["work-container","security.nesting=true"])
            item["expanded_config"]["security.nesting"]="true"
            return ""
        with patch.object(nesting,"run",side_effect=run):
            plan=nesting.handle({"action":"plan","logical":"work","nesting":True})
            self.assertEqual(len(calls),1)
            self.assertEqual(calls[0], [nesting.INCUS, "--force-local", "query",
                                         "/1.0/instances/work-container?project=default"])
            result=nesting.handle({"action":"apply","logical":"work","nesting":True,"expected":plan["expected"],"confirm":"work-container"})
        self.assertTrue(result["changed"]);self.assertFalse(result["restart_performed"])
        self.assertEqual(result["effective_nested_execution"],"not_verified")
        self.assertEqual(item["expanded_config"]["security.privileged"],"false")
        self.assertEqual(sum("set" in c for c in calls),1)
        self.assertFalse(any(any(v in c for v in ("start","stop","restart","delete")) for c in calls))

    def test_incus_query_rejects_global_project_but_accepts_rest_project(self):
        calls=[]
        def fake_run(argv):
            calls.append(argv)
            if "--project" in argv and "query" in argv:
                raise ValueError("LOCAL_INCUS_OBSERVATION_OR_CHANGE_FAILED")
            return json.dumps(self.target())
        with patch.object(nesting, "run", side_effect=fake_run):
            observed=nesting.handle({"action":"inspect","logical":"work"})
        self.assertEqual(observed["target"]["name"],"work-container")
        self.assertEqual(calls,[[nesting.INCUS,"--force-local","query",
                                 "/1.0/instances/work-container?project=default"]])

    def test_stale_incarnation_or_config_and_wrong_confirmation_cannot_write(self):
        for change in ("uuid","config","confirmation"):
            item=self.target();calls=[]
            def run(argv):
                calls.append(argv);return json.dumps(item)
            with patch.object(nesting,"run",side_effect=run):
                plan=nesting.handle({"action":"plan","logical":"work","nesting":True})
                if change=="uuid":item["config"]["volatile.uuid"]="replacement"
                if change=="config":item["expanded_config"]["security.privileged"]="true"
                with self.assertRaisesRegex(ValueError,"STALE_PLAN"):
                    nesting.handle({"action":"apply","logical":"work","nesting":True,"expected":plan["expected"],"confirm":"wrong" if change=="confirmation" else "work-container"})
            self.assertTrue(all("query" in c for c in calls))

    def test_unmanaged_core_and_nonboolean_inputs_refused(self):
        for name in ("containarium-core-postgres","../work","work;id",""):
            with patch.object(nesting,"run") as run,self.assertRaises(ValueError):
                nesting.handle({"action":"inspect","logical":name})
            run.assert_not_called()
        item=self.target();item["config"].pop("user.containarium.label.os-type")
        with patch.object(nesting,"run",return_value=json.dumps(item)),self.assertRaises(ValueError):
            nesting.handle({"action":"inspect","logical":"work"})
        with patch.object(nesting,"run",return_value=json.dumps(self.target())):
            for value in ("true",1,None):
                with self.assertRaises(ValueError):
                    nesting.handle({"action":"plan","logical":"work","nesting":value})


    def test_nesting_helper_safe_failure_code_survives_sudo_boundary(self):
        from types import SimpleNamespace
        for code in ("OWNED_PROVIDER_REQUIRED","MANAGED_INSTANCE_IDENTITY_REQUIRED",
                     "LOCAL_INCUS_OBSERVATION_OR_CHANGE_FAILED"):
            result=SimpleNamespace(returncode=2,stdout=json.dumps({"state":"failed","code":code}))
            with patch.object(nesting.subprocess,"run",return_value=result):
                with self.assertRaisesRegex(ValueError,"^"+code+"$"):
                    nesting.invoke({"action":"inspect","logical":"work"})

    def test_nesting_helper_unknown_failure_is_redacted(self):
        from types import SimpleNamespace
        for raw in ('not-json',json.dumps({"state":"failed","code":"KEY_MATERIAL"})):
            result=SimpleNamespace(returncode=2,stdout=raw)
            with patch.object(nesting.subprocess,"run",return_value=result):
                with self.assertRaisesRegex(ValueError,"NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE"):
                    nesting.invoke({"action":"inspect","logical":"work"})

    def test_mount_error_is_diagnosis_not_assertion_of_parent_config(self):
        result=nesting.diagnostic('Operation not permitted - Failed to mount "proc" onto /rootfs/proc')
        self.assertTrue(result["nested_proc_mount_denied"])
        self.assertEqual(result["parent_nesting"],"not_verified")
        self.assertIn("other_mount_confinement",result["cause"])
        self.assertFalse(nesting.diagnostic("unrelated failure")["nested_proc_mount_denied"])

    def test_mcp_catalog_is_independent_of_podman(self):
        original={"result":{"tools":[{"name":"create_container","inputSchema":{"properties":{"enable_podman":{"type":"boolean"}}}}]}}
        out=mcp_gate.catalog(copy.deepcopy(original))
        props=out["result"]["tools"][0]["inputSchema"]["properties"]
        self.assertIn("enable_nesting",props);self.assertIn("enable_podman",props)
        self.assertIn("boxweave_nesting", {t["name"] for t in out["result"]["tools"]})
        self.assertIn("boxweave_info", {t["name"] for t in out["result"]["tools"]})
        self.assertNotIn("enable_nesting",original["result"]["tools"][0]["inputSchema"]["properties"])

    def test_rest_create_keeps_nesting_independent_of_podman(self):
        from bwmini.provider_rest import RESTClient
        calls=[]
        provider=object.__new__(Provider)
        provider.argv=["/fixture"]
        provider.rest=RESTClient({"url":"http://127.0.0.1:18080",
                                  "token_file":"/tmp/no-read"})
        provider.rest.request=lambda method,route,body=None,timeout=30: (
            calls.append((method,route,body)) or
            {"container":{"username":"work","state":"CONTAINER_STATE_CREATING"}})
        spec={"cpu":"1","memory":"1GB","disk":"8GB",
              "ssh_public_key":PUBLIC,"_nesting":True}
        with patch("bwmini.provider.MCP",
                   side_effect=AssertionError("legacy MCP create forbidden")):
            provider.lifecycle_mutate("create","work",spec,60)
        self.assertEqual(len(calls),1)
        method,route,body=calls[0]
        self.assertEqual((method,route),("POST","/v1/containers"))
        self.assertIs(body["enablePodman"],False)
        self.assertNotIn("enable_nesting",body)
        self.assertIs(body["async"],True)
        with self.assertRaisesRegex(Fault,"NESTING_BOOLEAN_REQUIRED"):
            provider.lifecycle_mutate("create","work",{**spec,"_nesting":"true"},60)
        self.assertEqual(len(calls),1)

class NestingLifecycleTests(Base):
    def setUp(self):
        super().setUp()
        self.provider=LifecycleFixture()
        self.s=Service(self.store,self.provider,clock=lambda:self.now[0])
        self.life=self.s.lifecycle
    def test_independent_selection_is_frozen_in_plan_and_receipt(self):
        checks=[]
        self.provider.nesting_check=lambda value:checks.append(value)
        plan=self.life.plan("create",NAME,nesting=True)
        self.assertTrue(plan["nesting"])
        op=self.life.confirm(plan["id"],NAME);self.s.worker(once=True)
        result=self.life.receipt(op["id"])
        self.assertEqual(result["state"],"succeeded")
        self.assertTrue(result["result"]["requested_nesting"])
        self.assertEqual(result["result"]["effective_nested_execution"],"not_verified")
        self.assertTrue(checks)
    def test_unsupported_nesting_cannot_produce_a_create_plan(self):
        with self.assertRaisesRegex(Fault,"INDEPENDENT_NESTING_UNSUPPORTED"):
            self.life.plan("create",NAME,nesting=True)
        self.assertFalse(self.provider.writes)

class GateProtocolTests(unittest.TestCase):
    """The eight migrated lifecycle tools never send upstream MCP calls."""

    def peer(self):
        from types import SimpleNamespace
        class KeepBytes(io.BytesIO):
            def close(self): pass
        return SimpleNamespace(stdin=KeepBytes(),stdout=io.BytesIO(),
                               poll=lambda:0)

    def gateway(self,requests,*,config=None,rest_response=None,create_response=None):
        if config is None:
            config={"mode":"coordinated"}
        process=self.peer()
        output=io.StringIO()
        incoming=io.StringIO("".join(json.dumps(x)+"\n" for x in requests))
        calls=[]
        def create(cfg,args,rpc_id,*,normalize,protected):
            calls.append(("create",dict(args),protected))
            return create_response or {"task_id":"a"*32,"state":"accepted",
                                      "guest_user_and_ssh_verified":False}
        def mutate(cfg,name,args,*,protected):
            calls.append((name,dict(args),protected))
            return {"task_id":"b"*32,"state":"accepted"}
        def read(cfg,name,args):
            calls.append((name,dict(args),True))
            return rest_response or {"container":{"name":"work-container",
                     "createdAt":"123","username":"work"}}
        with (patch.object(mcp_gate.sys,"stdin",incoming),
              patch.object(mcp_gate.sys,"stdout",output),
              patch.object(mcp_gate.subprocess,"Popen",return_value=process),
              patch.object(mcp_gate,"tool_policy",return_value=None),
              patch.object(mcp_gate.gateway_protection,"enabled",return_value=False),
              patch.object(mcp_gate.gateway_rest,"policy",return_value=config),
              patch.object(mcp_gate.gateway_rest,"execute_read",side_effect=read),
              patch.object(mcp_gate.gateway_create,"dispatch",side_effect=create),
              patch.object(mcp_gate.rest_mutations,"dispatch",side_effect=mutate),
              patch.object(mcp_gate.gd,"record")):
            mcp_gate.main()
        replies=[json.loads(line) for line in output.getvalue().splitlines()]
        self.assertEqual(process.stdin.getvalue(),b"")
        return replies,calls

    def test_create_keeps_nesting_in_single_rest_intent(self):
        request={"jsonrpc":"2.0","id":1,"method":"tools/call",
                 "params":{"name":"create_container",
                           "arguments":{"username":"work","enable_nesting":True}}}
        replies,calls=self.gateway([request])
        self.assertEqual(len(calls),1)
        self.assertEqual(calls[0],("create",{"username":"work",
                            "enable_nesting":True},True))
        receipt=json.loads(replies[0]["result"]["content"][0]["text"])
        self.assertEqual(receipt["state"],"accepted")
        self.assertFalse(receipt["guest_user_and_ssh_verified"])

    def test_unknown_receipt_is_not_a_failure_or_automatic_retry(self):
        request={"jsonrpc":"2.0","id":2,"method":"tools/call",
                 "params":{"name":"create_container",
                           "arguments":{"username":"work","enable_nesting":True}}}
        replies,calls=self.gateway([request],create_response={
            "task_id":"a"*32,"state":"unknown",
            "code":"UPSTREAM_CREATE_OUTCOME_UNKNOWN"})
        receipt=json.loads(replies[0]["result"]["content"][0]["text"])
        self.assertEqual(receipt["state"],"unknown")
        self.assertEqual(len(calls),1)
        self.assertNotIn("error",replies[0])

    def test_migrated_reads_and_mutations_never_forward_mcp(self):
        names=("list_containers","get_container","debug_container",
               "start_container","stop_container","delete_container",
               "resize_container")
        requests=[{"jsonrpc":"2.0","id":i+1,"method":"tools/call",
                   "params":{"name":name,"arguments":(
                       {} if name=="list_containers" else
                       {"username":"work","cpu":"2"} if name=="resize_container" else
                       {"username":"work"})}}
                  for i,name in enumerate(names)]
        replies,calls=self.gateway(requests)
        self.assertEqual(len(replies),len(names))
        self.assertTrue(all("result" in reply for reply in replies))
        self.assertEqual([name for name,_,_ in calls],list(names))
        self.assertTrue(all(protected for _,_,protected in calls))

    def test_nesting_helper_failure_preserves_typed_error_with_rest_identity(self):
        request={"jsonrpc":"2.0","id":7,"method":"tools/call",
                 "params":{"name":"boxweave_nesting",
                           "arguments":{"action":"inspect","logical":"work"}}}
        with patch.object(mcp_gate,"nesting_invoke",
                          side_effect=ValueError("NESTING_HELPER_UNAVAILABLE")):
            replies,calls=self.gateway([request])
        self.assertEqual(replies[0]["error"]["message"],
                         "NESTING_HELPER_UNAVAILABLE")
        self.assertEqual(calls[0][0],"get_container")

    def test_unavailable_rest_refuses_create_without_upstream_mcp(self):
        request={"jsonrpc":"2.0","id":7,"method":"tools/call",
                 "params":{"name":"create_container",
                           "arguments":{"username":"work"}}}
        process=self.peer()
        output=io.StringIO()
        with (patch.object(mcp_gate.sys,"stdin",
                           io.StringIO(json.dumps(request)+"\n")),
              patch.object(mcp_gate.sys,"stdout",output),
              patch.object(mcp_gate.subprocess,"Popen",return_value=process),
              patch.object(mcp_gate,"tool_policy",return_value=None),
              patch.object(mcp_gate.gateway_protection,"enabled",return_value=False),
              patch.object(mcp_gate.gateway_rest,"policy",
                           side_effect=mcp_gate.gateway_rest.GatewayRESTFault(
                               "REST_APP_CONFIG_UNVERIFIED")),
              patch.object(mcp_gate.gd,"record")):
            mcp_gate.main()
        self.assertEqual(process.stdin.getvalue(),b"")
        reply=json.loads(output.getvalue())
        self.assertEqual(reply["error"]["message"],
                         "REST_APP_CONFIG_UNVERIFIED")
