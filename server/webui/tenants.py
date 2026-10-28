"""Provider-authoritative network Tenant views and local display metadata."""

import hashlib
import re
import time
from datetime import datetime
from .graph import text
from .provider import ProviderError

CREATE_BLOCK = (
    "Shared network Tenant creation is unavailable: "
    "Provider has no ready native network Tenant contract. "
    "A compatible daemon with supported enforced NIC ACLs is required; "
    "local grouping or post-create relabeling is not a substitute."
)
CONTRACT = "native-network-tenant/v1"
# Only these enum values establish membership under the merged upstream contract.
MEMBERSHIP_SOURCES = {
    "NETWORK_TENANT_SOURCE_NATIVE",
    "NETWORK_TENANT_SOURCE_CLOUD_ORG_ID",
    "NETWORK_TENANT_SOURCE_NAME",
}
CAPABILITIES = {
    "sharedTenantCreate": False,
    "reassignTenant": False,
    "networkPolicyWrite": False,
    "reason": CREATE_BLOCK,
    "contract": "unavailable",
}


def tenant_id(key):
    return "tenant_" + hashlib.sha256(key.encode()).hexdigest()


def key_text(value):
    if not isinstance(value, str):
        return None
    value = value.strip()
    try:
        text(value, 200)
    except ValueError:
        return None
    if not value or any(ord(c) < 32 for c in value):
        return None
    return value


def guard_view(raw, now=None):
    """ACL attachment evidence is distinct from bidirectional connectivity tests."""
    now = time.time() if now is None else now
    if not isinstance(raw, dict):
        raw = {}
    age = None
    last_pass = raw.get("lastPass")
    if not isinstance(last_pass, str):
        last_pass = ""
    try:
        stamp = datetime.fromisoformat(last_pass.replace("Z", "+00:00"))
        if stamp.tzinfo is not None:
            age = now - stamp.timestamp()
    except (ValueError, TypeError):
        pass
    fresh = age is not None and -30 <= age <= 180 and not raw.get("staleSince")
    status = (
        "unknown"
        if not raw
        else (
            "unsupported"
            if raw.get("unsupported") is True
            else (
                "degraded"
                if raw.get("lastError") or raw.get("staleSince")
                else (
                    "pending"
                    if raw.get("mode") != "GUARD_MODE_ENFORCE"
                    else "unknown" if not fresh else "ready"
                )
            )
        )
    )

    def names(field):
        return (
            [
                s
                for s in raw.get(field, [])
                if isinstance(s, str) and key_text(s) is not None
            ]
            if isinstance(raw.get(field, []), list)
            else []
        )

    entries = []
    for e in raw.get("entries", []) if isinstance(raw.get("entries", []), list) else []:
        if not isinstance(e, dict):
            continue
        entries.append(
            {k: key_text(e.get(k)) for k in ("container", "subject", "ip", "aclName")}
        )
        entries[-1]["attached"] = e.get("attached") is True
        entries[-1]["hasError"] = bool(e.get("lastError"))
    if status == "ready":
        if raw.get("firewallDriver") != "nftables":
            status = "unsupported"
        elif (
            any(e["hasError"] or not e["attached"] for e in entries)
            or names("unresolved")
            or names("unmanaged")
        ):
            status = "degraded"
    return {
        "status": status,
        "mode": key_text(raw.get("mode")) or "GUARD_MODE_UNSPECIFIED",
        "firewallDriver": key_text(raw.get("firewallDriver")) or "unknown",
        "unsupported": raw.get("unsupported") is True,
        "lastPass": last_pass if age is not None else "",
        "fresh": fresh,
        "ageSeconds": round(age) if age is not None else None,
        "hasError": bool(raw.get("lastError")),
        "entries": entries,
        "unresolved": names("unresolved"),
        "unmanaged": names("unmanaged"),
        "connectivityTested": False,
    }


def creation_guard_capable(guard):
    """Match upstream creation capability, independently of other NIC entries.

    Full convergence is still required for verified ACL/policy evidence.
    """
    return (
        guard["fresh"]
        and guard["mode"] == "GUARD_MODE_ENFORCE"
        and guard["firewallDriver"] == "nftables"
        and not guard["unsupported"]
    )


def resolve_box(box, guard, local_backend_ids=()):
    name = box.get("name", "")
    labels = box.get("labels", {}) if isinstance(box.get("labels"), dict) else {}
    entries = [e for e in guard["entries"] if e.get("container") == name]
    entry = entries[0] if len(entries) == 1 else None
    key = key_text(box.get("networkTenant"))
    provider_source = box.get("networkTenantSource")
    provider_confirmed = bool(
        key
        and isinstance(provider_source, str)
        and provider_source in MEMBERSHIP_SOURCES
    )
    source = "provider_" + provider_source if provider_confirmed else "unknown"
    # Legacy Providers omit the native fields. An unknown native source must
    # remain unconfirmed rather than acquiring authority through legacy fallback.
    legacy_membership = (
        "networkTenant" not in box and "networkTenantSource" not in box
    )
    if legacy_membership and not key and entry and guard["fresh"]:
        key = key_text(entry.get("subject"))
        source = "guard" if key else "unknown"
    if legacy_membership and not key:
        key = key_text(labels.get("cloud_org_id"))
        if key:
            source = "cloud_org_id"
        elif isinstance(name, str) and name.endswith("-container"):
            key = key_text(name[: -len("-container")])
            source = "name" if key else "unknown"
    if (
        (box.get("backendId") and box["backendId"] not in local_backend_ids)
        or name in guard["unresolved"]
        or name in guard["unmanaged"]
    ):
        key = None
        source = "unknown"
    membership = {
        "id": tenant_id(key) if key else None,
        "key": key,
        "source": source,
        "confirmed": bool(
            key and (source == "guard" or provider_confirmed)
        ),
        "declaredLabelIgnored": bool(labels.get("tenant")),
    }
    status = guard["status"]
    reason = "Guard status is unavailable, stale or not enforcing"
    if key is None:
        status = "unknown"
        reason = "Provider network Tenant is unresolved"
    elif not membership["confirmed"] and not legacy_membership:
        status = "unknown"
        reason = "Provider network Tenant source is missing or unrecognized"
    elif guard["status"] == "unsupported":
        status = "unsupported"
        reason = "Provider cannot support the required NIC ACLs"
    elif len(entries) > 1:
        status = "degraded"
        reason = "Guard entry is ambiguous"
    elif not entry:
        status = "pending" if guard["status"] == "ready" else guard["status"]
        reason = "No matching Guard ACL entry yet"
    elif len(entries) != 1 or entry.get("hasError"):
        status = "degraded"
        reason = "Guard entry is ambiguous or reports an error"
    elif guard["status"] == "ready":
        expected = (
            "containarium-tenant-" + hashlib.sha256(key.encode()).hexdigest()[:12]
        )
        if guard["firewallDriver"] != "nftables":
            status = "unsupported"
            reason = "Firewall driver cannot establish required ACL support"
        elif (
            entry.get("subject") != key
            or entry.get("aclName") != expected
            or entry.get("attached") is not True
        ):
            status = "degraded"
            reason = "Tenant ACL name, subject or attachment does not match"
        elif (
            not box.get("network", {}).get("ipAddress")
            or entry.get("ip") != box["network"]["ipAddress"]
        ):
            status = "pending"
            reason = "Guard address does not match the current Box address"
        else:
            status = "verified"
            reason = "Fresh Provider Guard and matching attached ACL; connectivity not tested"
    membership["guard"] = {
        "lastPass": guard["lastPass"],
        "status": status,
        "reason": reason,
        "connectivityTested": False,
    }
    return {**box, "tenant": membership}


class TenantService:
    def __init__(self, store, provider):
        self.store = store
        self.provider = provider

    def capabilities(self):
        result = dict(CAPABILITIES)
        try:
            raw = self.provider.tenant_capabilities()
        except (ProviderError, AttributeError):
            return result
        if (
            not isinstance(raw, dict)
            or raw.get("contract") != CONTRACT
            or raw.get("backendScope") != "local-lxc"
            or raw.get("nativeMembershipRead") is not True
        ):
            result["reason"] = (
                "Provider has no recognized authenticated local-LXC Tenant network contract"
            )
            return result
        result.update(
            contract=CONTRACT,
            nativeMembershipRead=True,
            sharedTenantCreate=raw.get("sharedTenantCreate") is True,
            atomicTenantAccess=raw.get("atomicTenantAccess") is True,
            networkPolicyWrite=raw.get("atomicTenantAccess") is True
            and raw.get("networkPolicyWrite") is True,
            reason=raw.get("reason") or "",
        )
        if result["atomicTenantAccess"] and not result["networkPolicyWrite"]:
            result["policyReason"] = (
                "Provider credential lacks network-policy:write; rotate the independent Web credential after approved Provider installation"
            )
        return result

    @staticmethod
    def annotate(box, guard, records, local_backend_ids=()):
        resolved = resolve_box(box, guard, local_backend_ids)
        for record in records:
            if (
                record.get("providerKey")
                and record["providerKey"] == resolved["tenant"]["key"]
            ):
                resolved["tenant"]["id"] = record["id"]
                resolved["tenant"]["name"] = record["name"]
                break
        return resolved

    def creation_target(self, snapshot, tid, username):
        if not tid:
            return {
                "id": tenant_id(username),
                "key": username,
                "source": "expected_name_fallback",
                "guardStatus": "pending",
            }
        if not snapshot["capabilities"]["sharedTenantCreate"]:
            raise ValueError(snapshot["capabilities"].get("reason") or CREATE_BLOCK)
        tenant = self.record(snapshot, tid)
        key = tenant.get("providerKey")
        if not key:
            key = "bw-" + tid.removeprefix("tenant_local_")
            tenant = self.store.bind_tenant_key(tid, tenant["version"], key)
            # Repeated rows sharing this Tenant reuse one immutable declaration.
            self.record(snapshot, tid).update(tenant)
        if not re.fullmatch(
            r"[a-zA-Z0-9][a-zA-Z0-9:._-]{0,127}", key
        ) or key.startswith(("account:", "core-", "containarium-core-")):
            raise ValueError(
                "Provider cannot safely create into this reserved or unsupported network Tenant"
            )
        return {
            "id": tid,
            "key": key,
            "source": "explicit_network_tenant",
            "guardStatus": "pending",
        }

    def snapshot(self):
        inventory = self.provider.boxes()
        try:
            raw = self.provider.network_guard()
        except (ProviderError, AttributeError):
            raw = {}
        if not isinstance(raw, dict):
            raw = {}
        guards = {k: guard_view(raw.get(k, {})) for k in ("core", "tenant")}
        records = {t["id"]: t for t in self.store.tenants()}
        boxes = [
            self.annotate(
                b,
                guards["tenant"],
                records.values(),
                inventory.get("localBackendIds", []),
            )
            for b in inventory.get("containers", [])
        ]
        capabilities = self.capabilities()
        if (
            inventory.get("unreachableBackends")
            or not creation_guard_capable(guards["tenant"])
        ):
            capabilities["sharedTenantCreate"] = False
            if capabilities.get("contract") == CONTRACT:
                capabilities["reason"] = (
                    "Fresh supported enforced Provider Guard capability is required for shared creation"
                )

        for b in boxes:
            membership = b["tenant"]
            tid = membership["id"]
            if tid and tid not in records:
                records[tid] = {
                    "id": tid,
                    "name": membership["key"],
                    "description": "",
                    "providerKey": membership["key"],
                    "version": 0,
                    "origin": "discovered",
                }
        for t in records.values():
            members = [b for b in boxes if b["tenant"]["id"] == t["id"]]
            t.update(
                boxCount=len(members),
                running=sum(
                    b.get("state") == "CONTAINER_STATE_RUNNING" for b in members
                ),
                stopped=sum(
                    b.get("state") == "CONTAINER_STATE_STOPPED" for b in members
                ),
            )
            t["membershipStatus"] = (
                "confirmed"
                if members and all(b["tenant"]["confirmed"] for b in members)
                else "inferred" if members else "empty"
            )
            t["unknown"] = len(members) - t["running"] - t["stopped"]
            states = {b["tenant"]["guard"]["status"] for b in members}
            t["guardStatus"] = (
                "empty"
                if not members
                else (
                    "verified"
                    if states == {"verified"}
                    and not inventory.get("unreachableBackends")
                    and not guards["tenant"]["unresolved"]
                    and not guards["tenant"]["unmanaged"]
                    else (
                        "unsupported"
                        if "unsupported" in states
                        else (
                            "degraded"
                            if "degraded" in states
                            or guards["tenant"]["unresolved"]
                            or guards["tenant"]["unmanaged"]
                            else (
                                "unknown"
                                if "unknown" in states
                                or inventory.get("unreachableBackends")
                                else "pending"
                            )
                        )
                    )
                )
            )
            t["providerKeyState"] = (
                "confirmed"
                if members and all(b["tenant"]["confirmed"] for b in members)
                else "inferred" if members else "declared"
            )
            t["connectivityTested"] = False
        for b in boxes:
            if b["tenant"]["id"] in records:
                b["tenant"]["name"] = records[b["tenant"]["id"]]["name"]
        return {
            "tenants": sorted(records.values(), key=lambda t: (t["name"], t["id"])),
            "containers": boxes,
            "guard": guards,
            "capabilities": capabilities,
            "unreachableBackends": inventory.get("unreachableBackends", []),
            "localBackendIds": inventory.get("localBackendIds", []),
            "current": self.store.setting("current_tenant") or "",
        }

    @staticmethod
    def record(snapshot, tid):
        for t in snapshot["tenants"]:
            if t["id"] == tid:
                return t
        raise KeyError(tid)

    def filtered(self, snapshot, tid):
        if not tid:
            return snapshot["containers"]
        if tid != "unresolved":
            self.record(snapshot, tid)
        return [
            b
            for b in snapshot["containers"]
            if b["tenant"]["id"] == (None if tid == "unresolved" else tid)
        ]

    def routes(self, snapshot, kind, tid=""):
        boxes = self.filtered(snapshot, tid)
        # Associate by a current exact resource identifier, never by a reusable IP.
        mapping = {
            alias: b["tenant"]
            for b in snapshot["containers"]
            for alias in (b.get("name"), b.get("username"))
            if alias
        }
        allowed = {
            alias
            for b in boxes
            for alias in (b.get("name"), b.get("username"))
            if alias
        }
        routes = [
            {**r, "tenant": mapping.get(r.get("containerName"))}
            for r in self.provider.routes(kind).get("routes", [])
        ]
        return [
            r
            for r in routes
            if not tid
            or r.get("containerName") in allowed
            or tid == "unresolved"
            and not r.get("tenant")
        ]

    def deletion_blockers(self, snapshot, tenant):
        blockers = []
        if snapshot["unreachableBackends"]:
            blockers.append("Provider inventory is incomplete")
        if tenant["boxCount"]:
            blockers.append("Tenant still contains Boxes")
        if tenant.get("providerKey") and any(
            e.get("subject") == tenant["providerKey"]
            for e in snapshot["guard"]["tenant"]["entries"]
        ):
            blockers.append("Provider Guard still references this Tenant")
        for kind in ("route", "passthrough"):
            routes = self.routes(snapshot, kind)
            if any((r.get("tenant") or {}).get("id") == tenant["id"] for r in routes):
                blockers.append("Tenant still has associated network routes")
            if any(not (r.get("tenant") or {}).get("id") for r in routes):
                blockers.append(
                    "Unresolved network routes prevent safe reference checks"
                )
        if self.store.tenant_graph_references(tenant["id"]):
            blockers.append("Saved Prompt Graph cards reference this Tenant")
        if (
            not snapshot["guard"]["tenant"]["fresh"]
            or snapshot["guard"]["tenant"]["unresolved"]
            or snapshot["guard"]["tenant"]["unmanaged"]
        ):
            blockers.append("Guard inventory is unavailable or stale")
        return blockers
