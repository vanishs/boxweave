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
            result=nesting.handle({"action":"apply","logical":"work","nesting":True,"expected":plan["expected"],"confirm":"work-container"})
        self.assertTrue(result["changed"]);self.assertFalse(result["restart_performed"])
        self.assertEqual(result["effective_nested_execution"],"not_verified")
        self.assertEqual(item["expanded_config"]["security.privileged"],"false")
        self.assertEqual(sum("set" in c for c in calls),1)
        self.assertFalse(any(any(v in c for v in ("start","stop","restart","delete")) for c in calls))

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
        self.assertEqual(out["result"]["tools"][-1]["name"],"boxweave_nesting")
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
            with patch.object(mcp_gate.sys,"stdin",StringIO(json.dumps(request)+"\n")),patch.object(mcp_gate.sys,"stdout",output),patch.object(mcp_gate.subprocess,"Popen",return_value=process),patch.object(mcp_gate,"nesting_invoke",side_effect=[plan,{"nesting":True}]) as invoke:
                mcp_gate.main()
            forwarded=json.loads(process.stdin.getvalue().splitlines()[0])
            self.assertNotIn("enable_nesting",forwarded["params"]["arguments"])
            self.assertFalse(forwarded["params"]["arguments"]["enable_podman"])
            self.assertEqual(invoke.call_count,0 if failure else 2)
            reply=json.loads(output.getvalue())
            if failure:self.assertIn("error",reply)
            else:self.assertTrue(json.loads(reply["result"]["content"][-1]["text"])["boxweave_nesting"]["nesting"])
