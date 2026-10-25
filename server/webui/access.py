"""Reviewed Tenant-wide ingress edits; Provider owns atomicity and policy data."""

import json
import re
import time
import uuid
from .graph import text
from .provider import ProviderError
from .tenants import CONTRACT

POLICY_FIELDS = {
    "tenant",
    "allowIntraTenant",
    "egressCidrs",
    "egressDomains",
    "mode",
    "allowMetadata",
    "source",
    "denyRules",
    "allowFromTenants",
    "egressPreset",
    "egressMode",
}


def public_policy(policy):
    def clean(value):
        if isinstance(value, str):
            try:
                return text(value, 20000)
            except ValueError:
                return "[redacted]"
        if isinstance(value, list):
            return [clean(v) for v in value]
        if isinstance(value, dict):
            return {k: clean(v) for k, v in value.items()}
        return value

    return clean({k: v for k, v in policy.items() if k in POLICY_FIELDS})


def policy_state(policy):
    """Compare protobuf semantics across omitted vs emitted default JSON fields."""
    defaults = {
        "allowIntraTenant": False,
        "egressCidrs": [],
        "egressDomains": [],
        "mode": "NETWORK_POLICY_MODE_UNSPECIFIED",
        "allowMetadata": False,
        "source": "",
        "denyRules": [],
        "egressPreset": "EGRESS_PRESET_UNSPECIFIED",
        "egressMode": "NETWORK_POLICY_MODE_UNSPECIFIED",
    }
    return {
        **defaults,
        **{k: v for k, v in public_policy(policy).items() if k != "allowFromTenants"},
    }


class AccessService:
    def __init__(self, store, provider, tenants):
        self.store, self.provider, self.tenants = store, provider, tenants

    def snapshot(self, snapshot, tid=""):
        if tid and tid != "unresolved":
            self.tenants.record(snapshot, tid)
        try:
            raw = self.provider.network_policies()
            policies = [
                public_policy(p) for p in raw.get("policies", []) if isinstance(p, dict)
            ]
            state = "available"
        except (AttributeError, ProviderError):
            policies, state = [], "unknown"
        by_key = {
            t.get("providerKey"): t for t in snapshot["tenants"] if t.get("providerKey")
        }
        relations = []
        for policy in policies:
            destination = by_key.get(policy.get("tenant"))
            for key in policy.get("allowFromTenants", []):
                source = by_key.get(key)
                if tid == "unresolved" and source and destination:
                    continue
                if (
                    tid
                    and tid != "unresolved"
                    and not any(t and t["id"] == tid for t in (source, destination))
                ):
                    continue
                relations.append(
                    {
                        "source": source
                        or {"id": None, "name": key, "providerKey": key},
                        "destination": destination
                        or {
                            "id": None,
                            "name": policy.get("tenant"),
                            "providerKey": policy.get("tenant"),
                        },
                        "direction": "source-to-destination",
                        "policyStatus": "configured",
                        "guardStatus": (
                            destination.get("guardStatus", "unknown")
                            if destination
                            else "unknown"
                        ),
                        "verificationStatus": "not_tested",
                    }
                )
        if tid == "unresolved":
            policies = [
                p
                for p in policies
                if p.get("tenant") not in by_key
                or any(k not in by_key for k in p.get("allowFromTenants", []))
            ]
        elif tid:
            key = self.tenants.record(snapshot, tid).get("providerKey")
            policies = [
                p
                for p in policies
                if p.get("tenant") == key or key in p.get("allowFromTenants", [])
            ]
        with self.store.db() as db:
            receipts = [
                json.loads(r[0])
                for r in db.execute(
                    "SELECT data FROM jobs ORDER BY rowid DESC LIMIT 100"
                )
            ]
        return {
            "policies": policies,
            "relations": relations,
            "policyReadState": state,
            "capabilities": snapshot["capabilities"],
            "recentOperations": [
                j for j in receipts if j.get("action") == "tenant-access"
            ][:20],
            "connectivityTested": False,
            "scopeNotice": "Tenant-wide ingress: all source Boxes may initiate connections to destination Boxes. This is not a Box/port rule; egress, listeners and other controls still apply.",
        }

    def deletion_blockers(self, snapshot, tenant):
        if not tenant.get("providerKey"):
            return []
        state = self.snapshot(snapshot)
        if state["policyReadState"] != "available":
            return ["Network Policy inventory is unavailable"]
        key = tenant["providerKey"]
        if any(
            p.get("tenant") == key or key in p.get("allowFromTenants", [])
            for p in state["policies"]
        ):
            return ["Provider Network Policies still reference this Tenant"]
        return []

    def plan(self, data):
        if (
            set(data) != {"source", "destination", "direction", "operation"}
            or data["direction"] not in ("forward", "reverse", "both")
            or data["operation"] not in ("grant", "revoke")
        ):
            raise ValueError(
                "Explicit source, destination, direction and grant/revoke required"
            )
        snap = self.tenants.snapshot()
        if not snap["capabilities"].get("networkPolicyWrite"):
            raise ValueError(
                snap["capabilities"].get("policyReason")
                or "Provider lacks the authenticated atomic Tenant access contract"
            )
        if snap["unreachableBackends"]:
            raise ValueError("Provider inventory is incomplete")
        if (
            data["operation"] == "grant"
            and snap["guard"]["tenant"]["status"] != "ready"
        ):
            raise ValueError("Grants require fresh enforced Provider Guard evidence")
        source, destination = [
            self.tenants.record(snap, data[k]) for k in ("source", "destination")
        ]
        if source["id"] == destination["id"]:
            raise ValueError("Select two distinct network Tenants")
        for tenant in (source, destination):
            if (
                not tenant.get("providerKey")
                or not tenant["boxCount"]
                or tenant["membershipStatus"] != "confirmed"
            ):
                raise ValueError(
                    "Cross-Tenant access requires current Provider-confirmed network membership"
                )
        pairs = (
            [(source, destination)]
            if data["direction"] == "forward"
            else [(destination, source)]
        )
        if data["direction"] == "both":
            pairs = [(source, destination), (destination, source)]
        changes, preview = [], []
        for src, dst in pairs:
            current = self.provider.network_policy(dst["providerKey"])
            revision = current.get("revision")
            if revision != "absent" and not (
                isinstance(revision, str) and re.fullmatch("[a-f0-9]{64}", revision)
            ):
                raise ValueError("Provider did not supply a safe full-policy revision")
            if data["operation"] == "revoke" and revision == "absent":
                continue
            field = "addSources" if data["operation"] == "grant" else "removeSources"
            changes.append(
                {
                    "destination": dst["providerKey"],
                    field: [src["providerKey"]],
                    "expectedRevision": revision,
                }
            )
            policy = current.get("policy") or {"tenant": dst["providerKey"]}
            before = policy.get("allowFromTenants", [])
            after = set(before)
            (
                after.add(src["providerKey"])
                if data["operation"] == "grant"
                else after.discard(src["providerKey"])
            )
            preview.append(
                {
                    "source": src,
                    "destination": dst,
                    "before": before,
                    "after": sorted(after),
                    "preservedPolicy": public_policy(policy),
                    "expectedRevision": revision,
                }
            )
        if not changes:
            raise ValueError(
                "No existing destination policy to revoke; nothing was written"
            )
        pid = str(uuid.uuid4())
        record = {
            "action": "tenant-access",
            "changes": changes,
            "preview": preview,
            "intent": data,
            "scopeNotice": self.snapshot(snap)["scopeNotice"],
        }
        with self.store.db() as db:
            db.execute(
                "INSERT INTO plans(id,data,expires) VALUES (?,?,?)",
                (pid, json.dumps(record), time.time() + 300),
            )
            db.execute(
                "INSERT INTO audit VALUES (?,?,?,?)",
                (time.time(), "tenant-access-preview", pid, "success"),
            )
        return {"id": pid, **record, "expiresIn": 300}

    def confirm(self, pid, data):
        if set(data) != {"confirm"} or data["confirm"] is not True:
            raise ValueError("Explicit policy confirmation required")
        # Revalidate the authenticated feature/permission contract before consuming.
        if not self.tenants.capabilities().get("networkPolicyWrite"):
            raise ValueError("Provider no longer authorizes atomic Tenant access")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT * FROM plans WHERE id=?", (pid,)).fetchone()
            if not row:
                raise KeyError(pid)
            plan = json.loads(row["data"])
            if (
                plan.get("action") != "tenant-access"
                or row["used"]
                or row["expires"] < time.time()
            ):
                raise ValueError("Policy plan expired, consumed or invalid")
            db.execute("UPDATE plans SET used=1 WHERE id=?", (pid,))
        jid = str(uuid.uuid4())
        receipt = {
            "id": jid,
            "action": "tenant-access",
            "finished": False,
            "created": time.time(),
            "intent": plan["intent"],
            "changes": plan["changes"],
            "preview": plan["preview"],
            "items": [{"status": "running", "target": plan["changes"]}],
            "connectivityTested": False,
        }
        self.store.job(jid, receipt)
        posted = False
        try:
            result = self.provider.patch_tenant_access(plan["changes"])
            posted = True
            returned = {
                p.get("policy", {}).get("tenant"): p
                for p in result.get("policies", [])
                if isinstance(p, dict)
            }
            if len(returned) != len(plan["changes"]):
                raise ProviderError(502, True)
            for change in plan["changes"]:
                entry = returned.get(change["destination"], {})
                revision = entry.get("revision")
                if not isinstance(revision, str) or not re.fullmatch(
                    "[a-f0-9]{64}", revision
                ):
                    raise ProviderError(502, True)
                observed = self.provider.network_policy(change["destination"])
                if observed.get("revision") != revision:
                    raise ProviderError(502, True)
                before = next(
                    p["preservedPolicy"]
                    for p in plan["preview"]
                    if p["destination"]["providerKey"] == change["destination"]
                )
                after = public_policy(observed.get("policy", {}))
                if policy_state(before) != policy_state(after):
                    raise ProviderError(502, True)
                allowed = observed.get("policy", {}).get("allowFromTenants", [])
                if any(
                    src not in allowed for src in change.get("addSources", [])
                ) or any(src in allowed for src in change.get("removeSources", [])):
                    raise ProviderError(502, True)
            state = {
                "TENANT_ACCESS_APPLICATION_STATE_PENDING": "pending",
                "TENANT_ACCESS_APPLICATION_STATE_GUARD_VERIFIED": "guard_verified",
                "TENANT_ACCESS_APPLICATION_STATE_DEGRADED": "degraded",
            }.get(result.get("applicationState"))
            if state is None:
                raise ProviderError(502, True)
            receipt["items"][0]["status"] = (
                "guard-verified"
                if state == "guard_verified"
                else "degraded" if state == "degraded" else "configured"
            )
            receipt["result"] = {
                "policies": [
                    {
                        "policy": public_policy(p.get("policy", {})),
                        "revision": p.get("revision"),
                    }
                    for p in result.get("policies", [])
                ],
                "applicationState": state,
                "connectivityTested": False,
            }
        except ProviderError as error:
            receipt["items"][0]["status"] = (
                "unknown"
                if error.unknown or posted
                else "conflict" if error.status == 409 else "failed"
            )
            receipt["items"][0]["message"] = (
                "Outcome unknown; inspect current policies and do not replay automatically"
                if error.unknown or posted
                else (
                    "Policy changed since preview; create a new reviewed plan"
                    if error.status == 409
                    else "Provider rejected the policy operation"
                )
            )
        receipt["finished"] = True
        self.store.job(jid, receipt)
        with self.store.db() as db:
            db.execute(
                "INSERT INTO audit VALUES (?,?,?,?)",
                (
                    time.time(),
                    "tenant-access-confirm",
                    jid,
                    receipt["items"][0]["status"],
                ),
            )
        return receipt
