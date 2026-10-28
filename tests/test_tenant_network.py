"""A/B contract integration through authenticated Web APIs; synthetic Provider."""

import copy
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import sys
import threading
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "server"))
import test_web as web
from webui.tenants import CONTRACT, tenant_id
from webui.provider import ProviderError
from webui.graph import markdown, validate


class NativeProvider(web.FakeProvider):
    def __init__(self):
        super().__init__()
        self.policies = {}
        self.mutex = threading.Lock()
        self.enabled = True
        self.allowed = True
        self.guard_enabled = True
        self.policy_error = None
        self.malformed = False

    def tenant_capabilities(self):
        return {
            "contract": CONTRACT if self.enabled else "unrecognized",
            "nativeMembershipRead": True,
            "sharedTenantCreate": True,
            "atomicTenantAccess": True,
            "networkPolicyWrite": self.allowed,
            "backendScope": "local-lxc",
        }

    def execute(self, action, item):
        result = super().execute(action, item)
        if action == "create":
            self.data[item["username"]].update(
                network={"ipAddress": "10.0.0." + str(len(self.data) + 2)},
                networkTenant=item.get("networkTenant", item["username"]),
                networkTenantSource="native" if item.get("networkTenant") else "name",
            )
        return result

    def network_guard(self):
        if not self.guard_enabled:
            return {}
        entries = []
        for n, b in self.data.items():
            key = b.get("networkTenant", n)
            entries.append(
                {
                    "container": b["name"],
                    "subject": key,
                    "ip": b["network"]["ipAddress"],
                    "aclName": "containarium-tenant-"
                    + hashlib.sha256(key.encode()).hexdigest()[:12],
                    "attached": True,
                }
            )
        return {
            "core": {},
            "tenant": {
                "mode": "GUARD_MODE_ENFORCE",
                "firewallDriver": "nftables",
                "lastPass": datetime.now(timezone.utc).isoformat(),
                "entries": entries,
            },
        }

    @staticmethod
    def revision(p):
        return (
            hashlib.sha256(json.dumps(p, sort_keys=True).encode()).hexdigest()
            if p
            else "absent"
        )

    def network_policy(self, key):
        p = copy.deepcopy(self.policies.get(key))
        return {"policy": p, "revision": self.revision(p)}

    def network_policies(self):
        return {"policies": copy.deepcopy(list(self.policies.values()))}

    def patch_tenant_access(self, changes):
        self.calls.append(("tenant-access", copy.deepcopy(changes)))
        if self.policy_error:
            raise self.policy_error
        if self.malformed:
            return {}
        with self.mutex:
            pending = copy.deepcopy(self.policies)
            for c in changes:
                old = pending.get(c["destination"])
                if self.revision(old) != c["expectedRevision"]:
                    raise ProviderError(409)
                policy = old or {"tenant": c["destination"]}
                allowed = set(policy.get("allowFromTenants", []))
                allowed.difference_update(c.get("removeSources", []))
                allowed.update(c.get("addSources", []))
                policy["allowFromTenants"] = sorted(allowed)
                pending[c["destination"]] = policy
            self.policies = pending
        return {
            "policies": [self.network_policy(c["destination"]) for c in changes],
            "applicationState": getattr(
                self, "application_state", "TENANT_ACCESS_APPLICATION_STATE_PENDING"
            ),
            "connectivityTested": False,
        }


class NetworkAPITests(unittest.TestCase):
    setUpClass = classmethod(web.WebTests.setUpClass.__func__)
    req = web.WebTests.req
    login = web.WebTests.login
    wait_job = web.WebTests.wait_job
    graph = web.WebTests.graph

    def setUp(self):
        web.WebTests.setUp(self)
        self.app.extensions["batch_pool"].shutdown(True)
        self.provider = NativeProvider()
        self.app = web.create_app(self.cfg, self.provider)
        self.app.testing = True
        self.addCleanup(self.app.extensions["batch_pool"].shutdown, True)
        self.c = self.app.test_client()
        self.store = self.app.extensions["store"]
        self.login()

    def create(self, names, tid=""):
        p = self.req(
            "plans",
            "POST",
            {
                "action": "create",
                "items": [
                    {
                        "username": n,
                        "tenantId": tid,
                        "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
                    }
                    for n in names
                ],
            },
        )
        self.assertEqual(p.status_code, 200, p.json)
        r = self.req("plans/" + p.json["id"] + "/confirm", "POST", {"confirm": True})
        self.assertEqual(r.status_code, 202, r.json)
        self.wait_job(r.json["id"])
        return p.json, self.req("jobs/" + r.json["id"] + "/reconcile", "POST", {}).json

    def tenants(self):
        for key, names in [
            ("allinone", ["box1", "box2", "box3"]),
            ("testing", ["test1", "test2"]),
        ]:
            for n in names:
                self.provider.data[n] = {
                    "username": n,
                    "name": n + "-container",
                    "state": "CONTAINER_STATE_RUNNING",
                    "network": {"ipAddress": "10.0.0.2"},
                    "networkTenant": key,
                    "networkTenantSource": "native",
                }
        return tenant_id("allinone"), tenant_id("testing")

    def access_plan(self, direction="forward", operation="grant"):
        src, dst = self.tenants()
        result = self.req(
            "tenant-access/plans",
            "POST",
            {
                "source": src,
                "destination": dst,
                "direction": direction,
                "operation": operation,
            },
        )
        self.assertEqual(result.status_code, 201, result.json)
        return result.json

    def confirm_access(self, plan):
        return self.req(
            "tenant-access/plans/" + plan["id"] + "/confirm", "POST", {"confirm": True}
        )

    def test_shared_creation_is_native_and_keeps_stable_local_id(self):
        local = self.req("tenants", "POST", {"name": "allinone"}).json
        tid = local["id"]
        plan, job = self.create(["box1", "box2", "box3"], tid)
        keys = {b["networkTenant"] for b in self.provider.data.values()}
        self.assertEqual(len(keys), 1)
        self.assertTrue(next(iter(keys)).startswith("bw-"))
        self.assertEqual({i["expectedTenant"]["id"] for i in plan["items"]}, {tid})
        self.assertEqual({i["status"] for i in job["items"]}, {"guard-verified"})
        self.assertTrue(
            all(
                i["verification"]["tenantVerified"]
                and i["verification"]["guardVerified"]
                and i["verification"]["networkConnectivity"] == "not_tested"
                for i in job["items"]
            )
        )
        snap = self.req("tenants").json
        self.assertEqual(len(snap["tenants"]), 1)
        self.assertEqual(snap["tenants"][0]["id"], tid)
        self.assertEqual(snap["tenants"][0]["boxCount"], 3)
        current = snap["tenants"][0]
        renamed = self.req(
            "tenants/" + tid, "PATCH", {"name": "Dev", "version": current["version"]}
        ).json
        self.assertEqual(renamed["id"], tid)
        self.assertEqual(len([c for c in self.provider.calls if c[0] == "create"]), 3)

    def test_individual_rows_keep_distinct_tenants_and_automatic_mode(self):
        src, dst = self.tenants()
        items = [
            {
                "username": n,
                "tenantId": t,
                "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
            }
            for n, t in [("front", src), ("worker", dst), ("independent", "")]
        ]
        p = self.req("plans", "POST", {"action": "create", "items": items}).json
        self.assertEqual(
            [i.get("networkTenant") for i in p["items"]], ["allinone", "testing", None]
        )
        self.assertEqual(p["items"][2]["expectedTenant"]["key"], "independent")

    def test_other_box_attachment_failure_does_not_disable_creation_capability(self):
        self.tenants()
        original = self.provider.network_guard

        def degraded():
            raw = original()
            raw["tenant"]["entries"][0]["attached"] = False
            raw["tenant"]["entries"][0]["lastError"] = "fixture attachment failure"
            raw["tenant"]["lastError"] = "fixture attachment failure"
            return raw

        self.provider.network_guard = degraded
        snapshot = self.req("tenants").json
        self.assertEqual(snapshot["guard"]["tenant"]["status"], "degraded")
        self.assertTrue(snapshot["capabilities"]["sharedTenantCreate"])
        self.assertFalse(snapshot["guard"]["tenant"]["connectivityTested"])

    def test_account_and_shared_membership_keys_remain_distinct(self):
        self.tenants()
        self.provider.data["box1"]["networkTenant"] = "account:testing"
        self.provider.data["box1"]["networkTenantSource"] = "NETWORK_TENANT_SOURCE_NATIVE"
        snapshot = self.req("tenants").json
        by_name = {box["username"]: box["tenant"] for box in snapshot["containers"]}
        self.assertEqual(by_name["box1"]["key"], "account:testing")
        self.assertEqual(by_name["test1"]["key"], "testing")
        self.assertNotEqual(by_name["box1"]["id"], by_name["test1"]["id"])

    def test_legacy_downstream_contract_remains_gated(self):
        original = self.provider.tenant_capabilities
        def legacy():
            return {**original(), "contract": "boxweave.tenant-network/v1"}
        self.provider.tenant_capabilities = legacy
        snapshot = self.req("tenants").json
        self.assertFalse(snapshot["capabilities"]["sharedTenantCreate"])
        self.assertFalse(snapshot["capabilities"]["networkPolicyWrite"])

    def test_capability_auth_and_guard_are_independently_required(self):
        src, dst = self.tenants()
        item = {
            "username": "new",
            "tenantId": src,
            "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
        }
        self.provider.enabled = False
        self.assertEqual(
            self.req(
                "plans", "POST", {"action": "create", "items": [item]}
            ).status_code,
            400,
        )
        self.provider.enabled = True
        self.provider.guard_enabled = False
        self.assertEqual(
            self.req(
                "plans", "POST", {"action": "create", "items": [item]}
            ).status_code,
            400,
        )
        self.provider.guard_enabled = True
        p = self.req("plans", "POST", {"action": "create", "items": [item]}).json
        self.provider.enabled = False
        self.assertEqual(
            self.req(
                "plans/" + p["id"] + "/confirm", "POST", {"confirm": True}
            ).status_code,
            400,
        )
        self.assertEqual(self.provider.calls, [])
        self.provider.enabled = True
        self.provider.allowed = False
        self.assertEqual(
            self.req(
                "tenant-access/plans",
                "POST",
                {
                    "source": src,
                    "destination": dst,
                    "direction": "both",
                    "operation": "grant",
                },
            ).status_code,
            400,
        )

    def test_running_native_box_needs_guard_evidence(self):
        src, _ = self.tenants()
        p = self.req(
            "plans",
            "POST",
            {
                "action": "create",
                "items": [
                    {
                        "username": "new",
                        "tenantId": src,
                        "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
                    }
                ],
            },
        ).json
        r = self.req("plans/" + p["id"] + "/confirm", "POST", {"confirm": True}).json
        self.wait_job(r["id"])
        self.provider.guard_enabled = False
        result = self.req("jobs/" + r["id"] + "/reconcile", "POST", {}).json["items"][0]
        self.assertNotEqual(result["status"], "guard-verified")
        self.assertTrue(result["verification"]["tenantVerified"])
        self.assertFalse(result["verification"]["guardVerified"])

    def test_direction_policy_preservation_revoke_and_replay(self):
        self.tenants()
        policy = {
            "tenant": "testing",
            "mode": "NETWORK_POLICY_MODE_ENFORCE",
            "egressCidrs": ["10.0.0.0/8"],
            "egressDomains": ["example.com"],
            "allowMetadata": True,
            "source": "managed",
            "egressPreset": "EGRESS_PRESET_ALLOW_LIST_ONLY",
            "egressMode": "NETWORK_POLICY_MODE_LOG_ONLY",
            "denyRules": [{"cidr": "10.2.0.0/16", "note": "keep"}],
            "allowFromTenants": ["existing"],
        }
        self.provider.policies["testing"] = copy.deepcopy(policy)
        plan = self.access_plan()
        self.assertEqual(plan["changes"][0]["destination"], "testing")
        self.assertEqual(self.provider.calls, [])
        result = self.confirm_access(plan)
        self.assertEqual(result.status_code, 200, result.json)
        self.assertEqual(result.json["items"][0]["status"], "configured")
        self.assertFalse(result.json["connectivityTested"])
        expected = {**policy, "allowFromTenants": ["allinone", "existing"]}
        self.assertEqual(self.provider.policies["testing"], expected)
        self.assertNotIn("allinone", self.provider.policies)
        self.assertEqual(self.confirm_access(plan).status_code, 400)
        self.assertEqual(len(self.provider.calls), 1)
        revoke = self.access_plan(operation="revoke")
        self.confirm_access(revoke)
        self.assertEqual(self.provider.policies["testing"], policy)

    def test_both_directions_atomic_conflict_and_unknown_are_not_success(self):
        plan = self.access_plan("both")
        self.provider.policies["testing"] = {
            "tenant": "testing",
            "egressCidrs": ["192.0.2.0/24"],
        }
        result = self.confirm_access(plan).json
        self.assertEqual(result["items"][0]["status"], "conflict")
        self.assertNotIn("allinone", self.provider.policies)
        plan = self.access_plan("both")
        result = self.confirm_access(plan).json
        self.assertEqual(result["items"][0]["status"], "configured")
        self.assertEqual(
            self.provider.policies["allinone"]["allowFromTenants"], ["testing"]
        )
        self.assertEqual(
            self.provider.policies["testing"]["allowFromTenants"], ["allinone"]
        )
        plan = self.access_plan("both", "revoke")
        self.provider.policy_error = ProviderError(502, True)
        result = self.confirm_access(plan).json
        self.assertEqual(result["items"][0]["status"], "unknown")
        self.assertEqual(self.confirm_access(plan).status_code, 400)

    def test_absent_policy_readback_with_gateway_emitted_defaults(self):
        # grpc-gateway emits zero values even when an absent preview had only a key.
        plan = self.access_plan()
        original = self.provider.network_policy

        def gateway_policy(key):
            result = original(key)
            if result["policy"] is not None:
                result["policy"] = {
                    "allowIntraTenant": False,
                    "egressCidrs": [],
                    "egressDomains": [],
                    "mode": "NETWORK_POLICY_MODE_UNSPECIFIED",
                    "allowMetadata": False,
                    "source": "",
                    "denyRules": [],
                    "egressPreset": "EGRESS_PRESET_UNSPECIFIED",
                    **result["policy"],
                }
            return result

        self.provider.network_policy = gateway_policy
        result = self.confirm_access(plan)
        self.assertEqual(result.status_code, 200, result.json)
        self.assertEqual(result.json["items"][0]["status"], "configured")

    def test_malformed_http_success_does_not_claim_configured(self):
        plan = self.access_plan()
        self.provider.malformed = True
        result = self.confirm_access(plan).json
        self.assertEqual(result["items"][0]["status"], "unknown")

    def test_auth_csrf_filters_and_immutable_policy_parameters(self):
        src, dst = self.tenants()
        body = {
            "source": src,
            "destination": dst,
            "direction": "both",
            "operation": "grant",
        }
        self.assertEqual(
            self.req("tenant-access/plans", "POST", body, csrf=False).status_code, 403
        )
        self.assertEqual(
            self.req(
                "tenant-access/plans", "POST", {**body, "policy": {"egressCidrs": []}}
            ).status_code,
            400,
        )
        self.assertEqual(self.req("tenant-access?tenant=forged").status_code, 404)
        self.assertEqual(self.req("tenant-access?arbitrary=route").status_code, 400)
        plan = self.access_plan()
        self.assertEqual(
            self.req(
                "tenant-access/plans/" + plan["id"] + "/confirm", "POST", {"confirm": 1}
            ).status_code,
            400,
        )

    def test_policy_references_block_metadata_delete(self):
        local = self.req("tenants", "POST", {"name": "local"}).json
        self.store.bind_tenant_key(
            local["id"], 1, "bw-" + local["id"].removeprefix("tenant_local_")
        )
        key = "bw-" + local["id"].removeprefix("tenant_local_")
        self.provider.policies["testing"] = {
            "tenant": "testing",
            "allowFromTenants": [key],
        }
        result = self.req(
            "tenants/" + local["id"], "DELETE", {"confirm": True, "version": 2}
        )
        self.assertEqual(result.status_code, 409)
        self.assertIn("Network Policies", result.json["error"])

    def test_access_graph_is_context_only_and_export_retains_both_tenants(self):
        self.tenants()
        plan = self.access_plan()
        self.confirm_access(plan)
        relation = self.req("tenant-access").json["relations"][0]
        a, b = relation["source"], relation["destination"]
        g = self.graph()
        g["nodes"][0].update(
            title="Tenant permission",
            resource={
                "type": "tenant-access",
                "id": "access:" + a["id"] + ":" + b["id"],
                "tenant": {"id": b["id"], "key": b["providerKey"]},
                "access": {
                    "source": {"id": a["id"], "key": a["providerKey"]},
                    "destination": {"id": b["id"], "key": b["providerKey"]},
                    "direction": "source-to-destination",
                    "policyStatus": "configured",
                    "verificationStatus": "not_tested",
                },
            },
        )
        count = len(self.provider.calls)
        saved = self.req("graphs", "POST", g)
        self.assertEqual(saved.status_code, 201, saved.json)
        output = markdown(saved.json)
        self.assertIn("allinone -> testing", output)
        self.assertIn("TCP not tested", output)
        self.assertEqual(len(self.provider.calls), count)
        self.assertGreater(self.store.tenant_graph_references(a["id"]), 0)
        g["nodes"][0]["resource"]["access"]["verificationStatus"] = "verified"
        with self.assertRaises(ValueError):
            validate(g)

    def test_typed_application_states_and_unknown_fail_closed(self):
        for state, expected in [
            ("TENANT_ACCESS_APPLICATION_STATE_PENDING", "configured"),
            ("TENANT_ACCESS_APPLICATION_STATE_GUARD_VERIFIED", "guard-verified"),
            ("TENANT_ACCESS_APPLICATION_STATE_DEGRADED", "degraded"),
            ("TENANT_ACCESS_APPLICATION_STATE_UNSPECIFIED", "unknown"),
            ("future-value", "unknown"),
        ]:
            with self.subTest(state=state):
                self.provider.application_state = state
                result = self.confirm_access(self.access_plan()).json
                self.assertEqual(result["items"][0]["status"], expected)
                self.assertFalse(result["connectivityTested"])

    def test_revoke_absent_destination_does_not_create_policy(self):
        self.confirm_access(self.access_plan())
        plan = self.access_plan("both", "revoke")
        self.assertEqual(len(plan["changes"]), 1)
        self.assertEqual(plan["changes"][0]["destination"], "testing")
        self.assertEqual(
            self.confirm_access(plan).json["items"][0]["status"], "configured"
        )
        self.assertNotIn("allinone", self.provider.policies)
        self.provider.policies.clear()
        src, dst = self.tenants()
        result = self.req(
            "tenant-access/plans",
            "POST",
            {
                "source": src,
                "destination": dst,
                "direction": "both",
                "operation": "revoke",
            },
        )
        self.assertEqual(result.status_code, 400)
        self.assertEqual(self.provider.policies, {})

    def test_grant_requires_fresh_guard_but_revoke_can_remain_pending(self):
        plan = self.access_plan()
        self.confirm_access(plan)
        src, dst = self.tenants()
        self.provider.guard_enabled = False
        request = {
            "source": src,
            "destination": dst,
            "direction": "forward",
            "operation": "grant",
        }
        self.assertEqual(
            self.req("tenant-access/plans", "POST", request).status_code, 400
        )
        request["operation"] = "revoke"
        preview = self.req("tenant-access/plans", "POST", request)
        self.assertEqual(preview.status_code, 201, preview.json)
        self.assertEqual(
            self.confirm_access(preview.json).json["items"][0]["status"], "configured"
        )

    def test_verified_local_backend_keeps_native_membership(self):
        self.tenants()
        for b in self.provider.data.values():
            b["backendId"] = "local-dev"
        original = self.provider.boxes
        self.provider.boxes = lambda: {**original(), "localBackendIds": ["local-dev"]}
        plan = self.access_plan()
        self.assertEqual(
            self.confirm_access(plan).json["items"][0]["status"], "configured"
        )
        self.provider.boxes = original
        src, dst = self.tenants()
        for b in self.provider.data.values():
            b["backendId"] = "remote"
        result = self.req(
            "tenant-access/plans",
            "POST",
            {
                "source": src,
                "destination": dst,
                "direction": "forward",
                "operation": "grant",
            },
        )
        self.assertNotEqual(result.status_code, 201)

    def test_independent_egress_mode_change_is_unknown(self):
        plan = self.access_plan()
        patch = self.provider.patch_tenant_access

        def concurrent_change(changes):
            result = patch(changes)
            self.provider.policies["testing"][
                "egressMode"
            ] = "NETWORK_POLICY_MODE_ENFORCE"
            result["policies"] = [
                self.provider.network_policy(c["destination"]) for c in changes
            ]
            return result

        self.provider.patch_tenant_access = concurrent_change
        result = self.confirm_access(plan).json
        self.assertEqual(result["items"][0]["status"], "unknown")
        self.assertFalse(result["connectivityTested"])


if __name__ == "__main__":
    unittest.main()
