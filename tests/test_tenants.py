"""Tenant truth, migration, API and graph regression tests; synthetic Provider only."""

import copy
from datetime import datetime, timezone, timedelta
import hashlib
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "server"))
import test_web as web
from webui.tenants import guard_view, resolve_box, tenant_id, TenantService
from webui.provider import public_box, ProviderError
from webui.graph import validate, markdown
from webui.store import Store


def box(name="alpha", **extra):
    return {
        "username": name,
        "name": name + "-container",
        "state": "CONTAINER_STATE_RUNNING",
        "network": {"ipAddress": "10.0.0.2"},
        **extra,
    }


def raw_guard(key="dev", **extra):
    return {
        "mode": "GUARD_MODE_ENFORCE",
        "firewallDriver": "nftables",
        "lastPass": datetime.now(timezone.utc).isoformat(),
        "entries": [
            {
                "container": "alpha-container",
                "subject": key,
                "ip": "10.0.0.2",
                "aclName": "containarium-tenant-"
                + hashlib.sha256(key.encode()).hexdigest()[:12],
                "attached": True,
            }
        ],
        **extra,
    }


class TruthTests(unittest.TestCase):
    def test_native_guard_subject_overrides_cloud_and_name(self):
        b = resolve_box(
            box(labels={"cloud_org_id": "cloud", "tenant": "fake"}),
            guard_view(raw_guard("native")),
        )
        self.assertEqual(b["tenant"]["key"], "native")
        self.assertEqual(b["tenant"]["guard"]["status"], "verified")
        self.assertTrue(b["tenant"]["declaredLabelIgnored"])
        self.assertFalse(b["tenant"]["guard"]["connectivityTested"])

    def test_official_fallback_is_inferred_never_acl_verified(self):
        guard = guard_view({})
        self.assertEqual(guard["status"], "unknown")
        for data, key in [
            (box(labels={"cloud_org_id": "cloud", "tenant": "fake"}), "cloud"),
            (box(), "alpha"),
        ]:
            t = resolve_box(data, guard)["tenant"]
            self.assertEqual(t["key"], key)
            self.assertFalse(t["confirmed"])
            self.assertNotEqual(t["guard"]["status"], "verified")
        self.assertIsNone(resolve_box({"username": "alpha"}, guard)["tenant"]["id"])

    def test_unknown_and_remote_backend_never_borrow_local_evidence(self):
        for fields in (
            {"unresolved": ["alpha-container"]},
            {"unmanaged": ["alpha-container"]},
        ):
            self.assertIsNone(
                resolve_box(box(), guard_view(raw_guard(**fields)))["tenant"]["id"]
            )
        self.assertIsNone(
            resolve_box(box(backendId="peer"), guard_view(raw_guard()))["tenant"]["id"]
        )

    def test_guard_requires_enforce_fresh_driver_and_matching_acl(self):
        cases = [
            ({"unsupported": True}, "unsupported"),
            ({"mode": "GUARD_MODE_OBSERVE"}, "pending"),
            (
                {
                    "lastPass": (
                        datetime.now(timezone.utc) - timedelta(seconds=181)
                    ).isoformat()
                },
                "unknown",
            ),
            (
                {
                    "lastPass": (
                        datetime.now(timezone.utc) + timedelta(seconds=60)
                    ).isoformat()
                },
                "unknown",
            ),
            ({"staleSince": "yesterday"}, "degraded"),
            ({"firewallDriver": "xtables"}, "unsupported"),
            ({"lastError": "sensitive diagnostics withheld"}, "degraded"),
        ]
        for change, state in cases:
            with self.subTest(change=change):
                self.assertEqual(
                    resolve_box(box(), guard_view(raw_guard(**change)))["tenant"][
                        "guard"
                    ]["status"],
                    state,
                )
        for field, value, state in [
            ("attached", False, "degraded"),
            ("aclName", "wrong", "degraded"),
            ("ip", "10.0.0.99", "pending"),
            ("lastError", "private diagnostic", "degraded"),
        ]:
            raw = raw_guard()
            raw["entries"][0][field] = value
            parsed = guard_view(raw)
            self.assertNotIn("private diagnostic", json.dumps(parsed))
            self.assertEqual(
                resolve_box(box(), parsed)["tenant"]["guard"]["status"], state
            )

    def test_ambiguous_guard_and_name_only_guard_do_not_verify(self):
        raw = raw_guard()
        raw["entries"] *= 2
        self.assertEqual(
            resolve_box(box(), guard_view(raw))["tenant"]["guard"]["status"], "degraded"
        )
        self.assertEqual(
            resolve_box(box(), guard_view(raw_guard(entries=[])))["tenant"]["guard"][
                "status"
            ],
            "pending",
        )

    def test_provider_only_exposes_attribution_allowlist(self):
        result = public_box(
            box(
                labels={
                    "tenant": "declared",
                    "cloud_org_id": "dev",
                    "secret": "do not return",
                }
            )
        )
        self.assertEqual(
            result["labels"], {"tenant": "declared", "cloud_org_id": "dev"}
        )
        self.assertNotIn("do not return", json.dumps(result))

    def test_grouping_uses_provider_key_not_display_name_or_ip(self):
        with tempfile.TemporaryDirectory() as d:
            provider = web.FakeProvider()
            provider.data = {
                n: box(n, labels={"cloud_org_id": key})
                for n, key in [("alpha", "dev"), ("beta", "dev"), ("test", "testing")]
            }
            provider.network_guard = lambda: {"tenant": raw_guard(entries=[])}
            store = Store(Path(d) / "db")
            snapshot = TenantService(store, provider).snapshot()
            self.assertEqual(
                {t["providerKey"]: t["boxCount"] for t in snapshot["tenants"]},
                {"dev": 2, "testing": 1},
            )
            self.assertEqual(store.tenants(), [])  # Discovery is read-only.
            self.assertEqual(
                len(
                    TenantService(store, provider).filtered(snapshot, tenant_id("dev"))
                ),
                2,
            )


class MigrationTests(unittest.TestCase):
    def test_v1_all_tables_survive_and_second_open_is_idempotent(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "db"
            store = Store(path)
            with store.db() as db:
                db.execute("DROP TABLE tenants")
                db.execute("PRAGMA user_version=1")
                db.execute("INSERT INTO admin VALUES (1,?)", ("synthetic hash",))
                db.execute(
                    "INSERT INTO sessions VALUES (?,?,?)",
                    ("session", "csrf", 9999999999),
                )
                db.execute("INSERT INTO attempts VALUES (?,?)", ("address", 1))
                db.execute(
                    "INSERT INTO settings VALUES (?,?)", ("current", "old-graph")
                )
                db.execute(
                    "INSERT INTO graphs VALUES (?,?,?,?)",
                    (
                        "old-graph",
                        3,
                        json.dumps(
                            {"name": "Old", "prompt": "Keep", "nodes": [], "edges": []}
                        ),
                        1,
                    ),
                )
                db.execute("INSERT INTO jobs VALUES (?,?)", ("job", "{}"))
                db.execute("INSERT INTO plans VALUES (?,?,?,?)", ("plan", "{}", 1, 0))
                db.execute(
                    "INSERT INTO audit VALUES (?,?,?,?)", (1, "old", "target", "ok")
                )
                before = {
                    t: [tuple(r) for r in db.execute("SELECT * FROM " + t)]
                    for t in [
                        "admin",
                        "sessions",
                        "attempts",
                        "settings",
                        "graphs",
                        "jobs",
                        "plans",
                        "audit",
                    ]
                }
            backup = Path(d) / "before.db"
            with sqlite3.connect(path) as src, sqlite3.connect(backup) as dst:
                src.backup(dst)
            Store(path)
            Store(path)
            with sqlite3.connect(path) as db:
                self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], 2)
                for table, rows in before.items():
                    self.assertEqual(
                        db.execute("SELECT * FROM " + table).fetchall(), rows
                    )
            with sqlite3.connect(backup) as db:
                self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], 1)
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)

    def test_failed_migration_rolls_back_version_and_old_data(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "db"
            Store(path)
            with sqlite3.connect(path) as db:
                db.execute("PRAGMA user_version=1")
                db.execute("INSERT INTO settings VALUES ('keep','value')")
            with self.assertRaises(sqlite3.OperationalError):
                Store(path)  # Conflicting table simulates migration failure.
            with sqlite3.connect(path) as db:
                self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], 1)
                self.assertEqual(
                    db.execute("SELECT value FROM settings").fetchone()[0], "value"
                )


class TenantAPITests(unittest.TestCase):
    setUpClass = classmethod(web.WebTests.setUpClass.__func__)
    setUp = web.WebTests.setUp
    req = web.WebTests.req
    login = web.WebTests.login
    graph = web.WebTests.graph
    wait_job = web.WebTests.wait_job

    def test_auth_csrf_origin_and_strict_metadata(self):
        self.assertEqual(self.req("tenants").status_code, 401)
        self.login()
        for kw in ({"csrf": False}, {"origin": False}):
            self.assertEqual(
                self.req("tenants", "POST", {"name": "dev"}, **kw).status_code, 403
            )
        for fields in (
            {"name": "dev", "providerKey": "core"},
            {"name": "dev", "id": "fake"},
            {"name": ""},
        ):
            self.assertEqual(self.req("tenants", "POST", fields).status_code, 400)
        self.assertEqual(self.provider.calls, [])

    def test_metadata_crud_selection_rename_and_conflict(self):
        self.login()
        created = self.req(
            "tenants", "POST", {"name": "dev", "description": "Local only"}
        ).json
        tid = created["id"]
        self.assertTrue(tid.startswith("tenant_local_"))
        self.req("tenant-selection", "POST", {"id": tid})
        edited = self.req(
            "tenants/" + tid, "PATCH", {"name": "friendly", "version": 1}
        ).json
        self.assertEqual(edited["id"], tid)
        self.assertIsNone(edited["providerKey"])
        self.assertEqual(
            self.req(
                "tenants/" + tid, "PATCH", {"name": "stale", "version": 1}
            ).status_code,
            409,
        )
        self.assertEqual(self.req("tenants").json["current"], tid)
        self.assertEqual(
            self.req(
                "tenants/" + tid, "DELETE", {"version": 2, "confirm": True}
            ).status_code,
            200,
        )
        self.assertEqual(self.req("tenants").json["current"], "")
        self.assertEqual(self.provider.calls, [])
        with self.store.db() as db:
            self.assertEqual(
                db.execute(
                    "SELECT COUNT(*) FROM audit WHERE action LIKE 'tenant-%'"
                ).fetchone()[0],
                3,
            )

    def test_discovered_metadata_keeps_native_key_and_blocks_nonempty_delete(self):
        self.login()
        self.provider.data = {"alpha": box()}
        tid = tenant_id("alpha")
        response = self.req("tenants/" + tid, "PATCH", {"name": "dev", "version": 0})
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json["providerKey"], "alpha")
        self.assertEqual(
            self.req("boxes").json["containers"][0]["tenant"]["name"], "dev"
        )
        self.assertEqual(
            self.req(
                "tenants/" + tid, "DELETE", {"version": 1, "confirm": True}
            ).status_code,
            409,
        )
        self.assertIn("alpha", self.provider.data)

    def test_graph_references_block_metadata_delete(self):
        self.login()
        t = self.req("tenants", "POST", {"name": "dev"}).json
        g = self.graph()
        g["nodes"][0]["resource"]["tenant"] = {
            "id": t["id"],
            "name": "dev",
            "source": "local",
        }
        self.assertEqual(self.req("graphs", "POST", g).status_code, 201)
        result = self.req(
            "tenants/" + t["id"], "DELETE", {"version": 1, "confirm": True}
        )
        self.assertEqual(result.status_code, 409)
        self.assertIn("Prompt Graph", result.json["error"])

    def test_scope_validation_routes_exact_identity_and_unknown_bucket(self):
        self.login()
        self.provider.data = {"alpha": box(), "beta": box("beta")}
        self.provider.routes = lambda k: {
            "routes": [
                {"containerName": "alpha-container", "domain": "a"},
                {"containerName": "beta-container", "domain": "b"},
                {"targetIp": "10.0.0.2", "domain": "orphan"},
            ]
        }
        tid = tenant_id("alpha")
        self.assertEqual(
            [b["username"] for b in self.req("boxes?tenant=" + tid).json["containers"]],
            ["alpha"],
        )
        self.assertEqual(
            [
                r["domain"]
                for r in self.req("network/route?tenant=" + tid).json["routes"]
            ],
            ["a"],
        )
        self.assertEqual(
            [
                r["domain"]
                for r in self.req("network/route?tenant=unresolved").json["routes"]
            ],
            ["orphan"],
        )
        self.assertEqual(
            self.req("metrics?tenant=" + tid).json["containers"][0]["username"], "alpha"
        )
        self.assertEqual(self.req("boxes?tenant=forged").status_code, 404)
        self.assertEqual(self.req("boxes?labels=core").status_code, 400)
        for view in ["boxes", "network", "monitoring"]:
            self.assertEqual(self.req("tenants/" + tid + "/" + view).status_code, 200)

    def test_unresolved_routes_and_guard_references_block_deletion(self):
        self.login()
        t = self.req("tenants", "POST", {"name": "empty"}).json
        self.provider.routes = lambda kind: {
            "routes": [{"containerName": "missing-container", "domain": "orphan"}]
        }
        result = self.req(
            "tenants/" + t["id"], "DELETE", {"version": 1, "confirm": True}
        )
        self.assertEqual(result.status_code, 409)
        self.assertIn("Unresolved network routes", result.json["error"])
        self.assertEqual(self.provider.calls, [])
        self.provider.routes = lambda kind: {"routes": []}
        tid = tenant_id("dev")
        self.store.save_tenant(tid, "Dev", "", "dev", 0)
        self.provider.network_guard = lambda: {"tenant": raw_guard("dev")}
        result = self.req("tenants/" + tid, "DELETE", {"version": 1, "confirm": True})
        self.assertEqual(result.status_code, 409)
        self.assertIn("Guard still references", result.json["error"])

    def test_unsupported_shared_create_and_reassignment_rejected(self):
        self.login()
        item = {
            "username": "new",
            "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
        }
        for changes in (
            {"tenantId": tenant_id("dev")},
            {"labels": {"tenant": "dev"}},
            {"tenant_id": "dev"},
        ):
            result = self.req(
                "plans", "POST", {"action": "create", "items": [{**item, **changes}]}
            )
            self.assertEqual(result.status_code, 400)
        result = self.req(
            "plans", "POST", {"action": "resize", "items": [{**item, "tenantId": ""}]}
        )
        self.assertEqual(result.status_code, 400)
        self.assertEqual(self.provider.calls, [])
        plan = self.req(
            "plans",
            "POST",
            {
                "action": "create",
                "items": [item, {**item, "username": "next", "tenantId": ""}],
            },
        ).json
        self.assertEqual(
            [i["expectedTenant"]["key"] for i in plan["items"]], ["new", "next"]
        )
        self.assertTrue(
            all(i["expectedTenant"]["guardStatus"] == "pending" for i in plan["items"])
        )
        self.assertFalse(plan["capabilities"]["sharedTenantCreate"])

    def test_create_running_is_not_success_without_acl_and_reconcile_never_replays(
        self,
    ):
        self.login()
        item = {
            "username": "new",
            "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
        }
        plan = self.req("plans", "POST", {"action": "create", "items": [item]}).json
        result = self.req(
            "plans/" + plan["id"] + "/confirm", "POST", {"confirm": True}
        ).json
        job = self.wait_job(result["id"])
        self.assertEqual(job["items"][0]["status"], "submitted")
        self.provider.network_guard = lambda: {}
        pending = self.req("jobs/" + result["id"] + "/reconcile", "POST", {}).json
        self.assertEqual(pending["items"][0]["status"], "verification-pending")
        with self.store.db() as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM jobs").fetchone()[0], 1)
        self.assertEqual(len(self.provider.calls), 1)
        self.assertEqual(
            self.req(
                "plans", "POST", {"action": "create", "items": [item]}
            ).status_code,
            400,
        )
        self.provider.network_guard = lambda: {
            "tenant": raw_guard(
                "wrong",
                entries=[
                    {**raw_guard("wrong")["entries"][0], "container": "new-container"}
                ],
            )
        }
        bad = self.req("jobs/" + result["id"] + "/reconcile", "POST", {}).json
        self.assertEqual(bad["items"][0]["status"], "degraded")
        self.assertEqual(len(self.provider.calls), 1)

    def test_unavailable_or_partial_inventory_never_claims_verified(self):
        self.login()
        self.provider.data = {"alpha": box()}
        self.provider.boxes = lambda: {
            "containers": list(self.provider.data.values()),
            "unreachableBackends": [{"backendId": "peer"}],
        }
        snapshot = self.req("tenants").json
        self.assertNotEqual(snapshot["tenants"][0]["guardStatus"], "verified")
        self.provider.network_guard = lambda: (_ for _ in ()).throw(ProviderError(403))
        self.assertEqual(
            self.req("tenants").json["containers"][0]["tenant"]["guard"]["status"],
            "unknown",
        )


class TenantGraphTests(unittest.TestCase):
    def graph(self, keys):
        return {
            "name": "",
            "prompt": "Keep user prompt",
            "nodes": [
                {
                    "id": "n" + str(i),
                    "type": "resource",
                    "title": "box" + str(i),
                    "comment": "Keep comment " + str(i),
                    "resource": {
                        "type": "box",
                        "id": "box" + str(i),
                        **(
                            {"tenant": {"id": tenant_id(key), "key": key}}
                            if key
                            else {}
                        ),
                    },
                    "position": {"x": 0, "y": 0},
                }
                for i, key in enumerate(keys)
            ],
            "edges": [],
        }

    def test_single_mixed_and_legacy_markdown(self):
        one = markdown(self.graph(["dev", "dev"]))
        self.assertIn("\nTenant: dev\n", one)
        self.assertNotIn("[Tenant:", one)
        mixed = markdown(self.graph(["dev", "testing", None]))
        self.assertIn("box0 [Tenant: dev]", mixed)
        self.assertIn("box1 [Tenant: testing]", mixed)
        self.assertIn("box2 [Tenant: Unknown]", mixed)
        for content in ["Keep user prompt", "Keep comment 0", "Keep comment 1"]:
            self.assertIn(content, mixed)
        self.assertNotIn("Tenant:", markdown(self.graph([None])))

    def test_local_empty_tenant_is_never_exported_as_real_network(self):
        g = self.graph(["dev"])
        g["nodes"][0]["resource"] = {
            "type": "tenant",
            "id": "tenant_local_" + "a" * 32,
            "tenant": {
                "id": "tenant_local_" + "a" * 32,
                "name": "dev",
                "source": "local",
            },
        }
        self.assertIn("local metadata only", markdown(g))

    def test_invalid_tenant_and_secret_context_rejected(self):
        for ref in (
            {"id": None},
            {"id": tenant_id("dev"), "key": "token=synthetic"},
            {"id": "bad"},
            {"id": tenant_id("dev"), "acl": "forged"},
        ):
            g = self.graph(["dev"])
            g["nodes"][0]["resource"]["tenant"] = ref
            with self.assertRaises(ValueError):
                validate(g)


if __name__ == "__main__":
    unittest.main()
