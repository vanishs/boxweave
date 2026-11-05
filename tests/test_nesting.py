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

    def test_provider_never_substitutes_podman_for_missing_nesting_capability(self):
        calls=[]
        class Peer:
            def __init__(self,*a):pass
            def __enter__(self):return self
            def __exit__(self,*a):pass
            def discover(self):
                return {"create_container":{"inputSchema":{"properties":{k:{"type":"boolean"} if k in ("enable_podman","enable_nesting") else {} for k in props}}}}
            def tool(self,n,a):calls.append((n,a));return {"content":[]}
        provider=object.__new__(Provider);provider.argv=["/fixture"]
        spec={"cpu":"1","memory":"1GB","disk":"8GB","ssh_public_key":PUBLIC,"_nesting":True}
        props={"username","cpu","memory","disk","image","ssh_keys","enable_podman"}
        with patch("bwmini.provider.MCP",Peer):
            with self.assertRaises(Fault):provider.lifecycle_mutate("create","work",spec,60)
            self.assertFalse(calls)
            props.add("enable_nesting")
            provider.lifecycle_mutate("create","work",spec,60)
            self.assertTrue(calls[-1][1]["enable_nesting"])
            self.assertFalse(calls[-1][1]["enable_podman"])
            provider.lifecycle_mutate("create","work",{k:v for k,v in spec.items() if k!="_nesting"},60)
            self.assertNotIn("enable_nesting",calls[-1][1])

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
    @classmethod
    def setUpClass(cls):
        # Gateway fixture expectations must not depend on the host's
        # root-installed protected-mode policy. Individual protection
        # tests explicitly override this default where appropriate.
        base = patch.object(mcp_gate.gateway_protection, "enabled", return_value=False)
        base.start()
        cls.addClassCleanup(base.stop)

    def test_create_strips_extension_and_applies_it_only_after_success(self):
        import struct,base64
        from io import StringIO
        from types import SimpleNamespace
        key="ssh-ed25519 "+base64.b64encode(struct.pack(">I",11)+b"ssh-ed25519"+struct.pack(">I",32)+b"x"*32).decode()
        request={"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_container",
                    "arguments":{"username":"work","ssh_keys":[key],"enable_podman":False,"enable_nesting":True}}}
        observed={"name":"work-container","createdAt":"123","username":"work"}
        plan={"target":{"name":"work-container","created":"123"},"expected":{"identity":"pinned"}}
        for failure in (False,True):
            upstream=[{"jsonrpc":"2.0","id":1,**({"error":{"code":-1,"message":"creation failed"}} if failure else {"result":{"content":[]}})}]
            if not failure:upstream.append({"jsonrpc":"2.0","id":"boxweave-nesting-1","result":{"content":[{"type":"text","text":json.dumps({"container":observed})}]}})
            sent=io.BytesIO()
            process=SimpleNamespace(stdin=sent,stdout=io.BytesIO("".join(json.dumps(x)+"\n" for x in upstream).encode()),poll=lambda:0)
            class KeepBytes(io.BytesIO):
                def close(self):pass
            process.stdin=KeepBytes()
            output=StringIO()
            with patch.object(mcp_gate.sys,"stdin",StringIO(json.dumps(request)+"\n")),patch.object(mcp_gate.sys,"stdout",output),patch.object(mcp_gate.subprocess,"Popen",return_value=process),patch.object(mcp_gate,"managed_public_key",return_value=key),patch.object(mcp_gate,"nesting_invoke",side_effect=[plan,{"nesting":True}]) as invoke:
                mcp_gate.main()
            forwarded=json.loads(process.stdin.getvalue().splitlines()[0])
            self.assertNotIn("enable_nesting",forwarded["params"]["arguments"])
            self.assertFalse(forwarded["params"]["arguments"]["enable_podman"])
            self.assertEqual(invoke.call_count,0 if failure else 2)
            reply=json.loads(output.getvalue())
            if failure:self.assertIn("error",reply)
            else:self.assertTrue(json.loads(reply["result"]["content"][-1]["text"])["boxweave_nesting"]["nesting"])

    def peer(self,upstream):
        from types import SimpleNamespace
        class KeepBytes(io.BytesIO):
            def close(self):pass
        return SimpleNamespace(stdin=KeepBytes(),stdout=io.BytesIO("".join(json.dumps(x)+"\n" for x in upstream).encode()),
                               poll=lambda:0)

    def test_failed_create_preserves_upstream_error_and_exposes_safe_unknown(self):
        request = {"jsonrpc": "2.0", "id": 15, "method": "tools/call",
                   "params": {"name": "create_container",
                              "arguments": {"username": "work", "enable_nesting": False}}}
        original = {"jsonrpc": "2.0", "id": 15,
                    "error": {"code": -32000, "message": "upstream create failed"}}
        output = io.StringIO()
        peer = self.peer([original])
        with (patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(request)+"\n")),
              patch.object(mcp_gate.sys, "stdout", output),
              patch.object(mcp_gate.subprocess, "Popen", return_value=peer),
              patch.object(mcp_gate, "managed_public_key", return_value="ssh-ed25519 AAAA")):
            mcp_gate.main()
        reply = json.loads(output.getvalue())
        self.assertEqual(reply["error"]["code"], -32000)
        self.assertEqual(reply["error"]["message"], "upstream create failed")
        record = reply["error"]["data"]["boxweave_create"]
        self.assertEqual((record["state"], record["logical"], record["retry"]),
                         ("unknown", "work", "not_authorized"))
        self.assertEqual(record["inspect"], "get_container")
        sent = [json.loads(line) for line in peer.stdin.getvalue().splitlines()]
        self.assertEqual(len(sent), 1)
        self.assertEqual(sent[0]["params"]["name"], "create_container")

    def test_failed_create_result_annotation_is_not_success(self):
        reply = {"jsonrpc": "2.0", "id": 1,
                 "result": {"isError": True,
                            "content": [{"type": "text", "text": "remote error"}]}}
        annotated = mcp_gate.annotate_failed_create(reply, {"username": "work"})
        self.assertTrue(annotated["result"]["isError"])
        self.assertEqual(annotated["result"]["content"][1]["text"], "remote error")
        self.assertEqual(json.loads(annotated["result"]["content"][0]["text"])
                         ["boxweave_create"]["code"], "CREATE_OUTCOME_UNVERIFIED")
        self.assertEqual(mcp_gate.annotate_failed_create(
            {"result": {"isError": False, "content": []}},
            {"username": "work"})["result"]["content"], [])
        self.assertNotIn("data", mcp_gate.annotate_failed_create(
            {"error": {"code": -1}}, {"username": "../unsafe"})["error"])

    def test_nesting_helper_gap_is_named_not_masked_as_trust_gate(self):
        request={"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"boxweave_nesting",
                    "arguments":{"action":"inspect","logical":"work"}}}
        observed={"container":{"name":"work-container","createdAt":"123","username":"work"}}
        upstream=[{"jsonrpc":"2.0","id":"boxweave-nesting-7","result":{"content":[{"type":"text","text":json.dumps(observed)}]}}]
        output=io.StringIO()
        with patch.object(mcp_gate.sys,"stdin",io.StringIO(json.dumps(request)+"\n")),patch.object(mcp_gate.sys,"stdout",output),patch.object(mcp_gate.subprocess,"Popen",return_value=self.peer(upstream)),patch.object(mcp_gate,"nesting_invoke",side_effect=ValueError("NESTING_HELPER_UNAVAILABLE")):
            mcp_gate.main()
        reply=json.loads(output.getvalue())
        self.assertIn("NESTING_HELPER_UNAVAILABLE",reply["error"]["message"])
        self.assertNotIn("TRUST_OR_WORKLOAD_KEY_GATE",reply["error"]["message"])

    def test_nesting_partial_preserves_upstream_created_receipt(self):
        request={"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"create_container",
                    "arguments":{"username":"work","enable_nesting":True}}}
        receipt={"type":"text","text":"Created work-container; running at 10.100.0.20"}
        observed={"container":{"name":"work-container","createdAt":"123","username":"work"}}
        upstream=[{"jsonrpc":"2.0","id":12,"result":{"content":[receipt]}},
                  {"jsonrpc":"2.0","id":"boxweave-nesting-12","result":{"content":[{"type":"text","text":json.dumps(observed)}]}}]
        process=self.peer(upstream)
        output=io.StringIO()
        with (patch.object(mcp_gate.sys,"stdin",io.StringIO(json.dumps(request)+"\n")),
              patch.object(mcp_gate.sys,"stdout",output),
              patch.object(mcp_gate.subprocess,"Popen",return_value=process),
              patch.object(mcp_gate,"managed_public_key",return_value="ssh-ed25519 AAAA"),
              patch.object(mcp_gate,"nesting_invoke",side_effect=ValueError("SECRET_NON_ALLOWLIST"))):
            mcp_gate.main()
        reply=json.loads(output.getvalue())
        self.assertTrue(reply["result"]["isError"])
        self.assertEqual(reply["result"]["content"][1],receipt)
        partial=json.loads(reply["result"]["content"][0]["text"])["boxweave_nesting"]
        self.assertTrue(partial["box_created"])
        self.assertEqual(partial["nesting_outcome"],"not_verified")
        self.assertEqual(partial["code"],"NESTING_CONFIGURATION_UNVERIFIED_INSPECT_ORIGINAL_INSTANCE")
        self.assertNotIn("SECRET_NON_ALLOWLIST",json.dumps(reply))

    def test_created_box_survives_nesting_helper_gap_without_recreation(self):
        request={"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_container",
                    "arguments":{"username":"work","enable_nesting":True}}}
        observed={"name":"work-container","createdAt":"123","username":"work"}
        plan={"target":{"name":"work-container","created":"123"},"expected":{"identity":"pinned"}}
        upstream=[{"jsonrpc":"2.0","id":1,"result":{"content":[]}},
                  {"jsonrpc":"2.0","id":"boxweave-nesting-1","result":{"content":[{"type":"text","text":json.dumps({"container":observed})}]}}]
        process=self.peer(upstream)
        output=io.StringIO()
        with patch.object(mcp_gate.sys,"stdin",io.StringIO(json.dumps(request)+"\n")),patch.object(mcp_gate.sys,"stdout",output),patch.object(mcp_gate.subprocess,"Popen",return_value=process),patch.object(mcp_gate,"managed_public_key",return_value="ssh-ed25519 AAAA"),patch.object(mcp_gate,"nesting_invoke",side_effect=[plan,ValueError("NESTING_HELPER_UNAVAILABLE")]):
            mcp_gate.main()
        forwarded=[json.loads(line) for line in process.stdin.getvalue().splitlines()]
        self.assertEqual(sum(message.get("method")=="tools/call" and message["params"]["name"]=="create_container"
                             for message in forwarded),1)
        reply=json.loads(output.getvalue())
        self.assertNotIn("error", reply)
        self.assertTrue(reply["result"]["isError"])
        partial=json.loads(reply["result"]["content"][0]["text"])["boxweave_nesting"]
        self.assertEqual(partial["state"], "partial")
        self.assertTrue(partial["box_created"])
        self.assertTrue(partial["requested_nesting"])
        self.assertEqual(partial["nesting_outcome"], "not_verified")
        self.assertEqual(partial["username"], "work")
        self.assertEqual(partial["code"], "NESTING_HELPER_UNAVAILABLE")
        self.assertIn("do not repeat", partial["next_action"])
