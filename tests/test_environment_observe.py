"""E0/E1 authenticated Provider capability and immutable fingerprint fixtures."""
import copy
import unittest

from bwmini.domain import Fault, Target
from bwmini.environment_observe import (
    EnvironmentInspector, diff_fingerprints, observe_exact,
)

TARGET=Target("containarium","local","default","phasebox",
              "phasebox-container","1793614975","phasebox","phasebox-container")
CONFIG={"provider":"containarium","backend":"local","project":"default",
        "targets":{"phasebox":TARGET.data()}}
CONTAINER={"container":{"username":"phasebox","name":"phasebox-container",
            "createdAt":"1793614975","state":"CONTAINER_STATE_RUNNING",
            "networkTenant":"boxweave-test",
            "resources":{"cpu":"1","memory":"1GB","disk":"10GB"}}}
CATALOG={"get_container":{"name":"get_container",
                 "inputSchema":{"type":"object","properties":{"username":{"type":"string"}}}}}

class FakeMCP:
    def __init__(self,catalog,payload):
        self.catalog,self.payload,self.calls=catalog,payload,[]
    def __enter__(self):return self
    def __exit__(self,*args):return None
    def discover(self):
        self.calls.append("initialize+tools/list")
        return self.catalog
    def tool(self,name,args):
        self.calls.append((name,args))
        return {"content":[{"type":"text","text":__import__("json").dumps(self.payload)}]}

class EnvTests(unittest.TestCase):
    def test_missing_snapshot_contract_is_unsupported(self):
        x=observe_exact(CONFIG,"phasebox",CONTAINER,CATALOG,observed_at=7)
        self.assertEqual(x["observation_status"],"complete")
        self.assertEqual(x["storage_coverage"],"not_verified")
        self.assertEqual({v["state"] for v in x["capabilities"].values()}, {"unsupported"})
        self.assertEqual(x["observed_at"],7.)
        self.assertNotIn("10.100.",str(x))

    def test_tool_name_alone_is_not_permission(self):
        tools=CATALOG|{"restore_container_snapshot":{"description":"I support restore!"}}
        x=observe_exact(CONFIG,"phasebox",CONTAINER,tools)
        self.assertEqual(x["capabilities"]["snapshot_restore"]["state"],"not_verified")
        self.assertEqual(x["capabilities"]["snapshot_create"]["state"],"unsupported")

    def test_fingerprint_deterministic_and_diff(self):
        a=observe_exact(CONFIG,"phasebox",CONTAINER,CATALOG,observed_at=4)
        b=observe_exact(CONFIG,"phasebox",CONTAINER,CATALOG,observed_at=9)
        self.assertEqual(a["fingerprint_sha256"],b["fingerprint_sha256"])
        updated=copy.deepcopy(CONTAINER)
        updated["container"]["resources"]["cpu"]="2"
        updated["container"]["state"]="CONTAINER_STATE_STOPPED"
        c=observe_exact(CONFIG,"phasebox",updated,CATALOG)
        changes=diff_fingerprints(a,c)["changes"]
        self.assertEqual(set(changes),{"cpu","state"})

    def test_optional_missing_fields_are_not_false_diff(self):
        older=copy.deepcopy(CONTAINER)
        del older["container"]["networkTenant"]
        a=observe_exact(CONFIG,"phasebox",older,CATALOG)
        b=observe_exact(CONFIG,"phasebox",CONTAINER,CATALOG)
        diff=diff_fingerprints(a,b)
        self.assertIn("network_tenant",diff["not_verified"])
        self.assertNotIn("network_tenant",diff["changes"])

    def test_exact_target_and_unowned_refused(self):
        with self.assertRaisesRegex(Fault,"TARGET_NOT_ALLOWED"):
            observe_exact(CONFIG,"other",CONTAINER,CATALOG)
        for key,value in [("createdAt","2"),("username","other"),
                          ("name","other-container")]:
            broken=copy.deepcopy(CONTAINER);broken["container"][key]=value
            with self.subTest(key=key),self.assertRaisesRegex(Fault,"TARGET_CHANGED"):
                observe_exact(CONFIG,"phasebox",broken,CATALOG)
        foreign=copy.deepcopy(CONFIG);foreign["project"]="foreign"
        with self.assertRaises(Fault):
            observe_exact(foreign,"phasebox",CONTAINER,CATALOG)

    def test_invalid_or_unsafe_resource_refused(self):
        for replacement in ({"cpu":"1; rm -rf /"}, {"memory":{"secret":"token"}}):
            bad=copy.deepcopy(CONTAINER)
            bad["container"]["resources"].update(replacement)
            with self.assertRaisesRegex(Fault,"ENVIRONMENT_OBSERVATION_UNVERIFIED"):
                observe_exact(CONFIG,"phasebox",bad,CATALOG)
        bad=copy.deepcopy(CONTAINER);bad["container"]["state"]="arbitrary"
        with self.assertRaises(Fault):
            observe_exact(CONFIG,"phasebox",bad,CATALOG)

    def test_authenticated_adapter_read_only(self):
        class Provider:
            config=CONFIG
            def __init__(self,client):self.client=client
            def session(self):return self.client
        client=FakeMCP(CATALOG,CONTAINER)
        report=EnvironmentInspector(Provider(client)).report("phasebox")
        self.assertEqual(report["observation"]["cpu"],"1")
        self.assertEqual(client.calls,["initialize+tools/list",
                                       ("get_container",{"username":"phasebox"})])
        self.assertEqual(len(report["target_identity_digest"]),64)

    def test_refuses_unverified_get_schema(self):
        class Provider:
            config=CONFIG
            def session(self):return FakeMCP({"get_container":{"name":"get_container"}},CONTAINER)
        with self.assertRaisesRegex(Fault,"ENVIRONMENT_READ_UNSUPPORTED"):
            EnvironmentInspector(Provider()).report("phasebox")
