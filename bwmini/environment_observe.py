"""E0/E1 read-only Provider capability audit and bounded Box fingerprint.

No guest commands, Incus fallback, snapshot/clone writes or installation.
Every report is scoped to an already configured exact managed target.
"""
import hashlib
import json
import math
import re
import time

from .domain import Fault, Target
from .wire_results import text

_ACTIONS = ("snapshot_list", "snapshot_create", "snapshot_delete",
            "snapshot_restore", "clone_instance", "clone_snapshot",
            "identity_rekey")
# Names are only discovery candidates. No write becomes allowed by matching.
_CANDIDATES = {
    "snapshot_list": "list_container_snapshots",
    "snapshot_create": "create_container_snapshot",
    "snapshot_delete": "delete_container_snapshot",
    "snapshot_restore": "restore_container_snapshot",
    "clone_instance": "clone_container",
    "clone_snapshot": "clone_container_snapshot",
    "identity_rekey": "rekey_container_identity",
}
_STATES = ("CONTAINER_STATE_RUNNING", "CONTAINER_STATE_STOPPED")
_RESOURCE = ("cpu","memory","disk")
_TOKEN = re.compile(r"[A-Za-z0-9._/-]{1,80}\Z")

def _bounded(value, optional=False):
    if optional and value is None:
        return None
    if type(value) not in (str,int) or not _TOKEN.fullmatch(str(value)):
        raise Fault("ENVIRONMENT_OBSERVATION_UNVERIFIED")
    return str(value)

def _sha(obj):
    return hashlib.sha256(json.dumps(obj, sort_keys=True,
        separators=(",",":"), ensure_ascii=True).encode()).hexdigest()

def observe_exact(config, logical, payload, catalog, observed_at=None):
    """Accept a verified MCP observation, not an arbitrary requested target."""
    if not isinstance(config,dict):
        raise Fault("ENVIRONMENT_SCOPE_INVALID")
    targets=config.get("targets")
    if not isinstance(targets,dict) or logical not in targets:
        raise Fault("TARGET_NOT_ALLOWED",403)
    if not isinstance(payload,dict) or not isinstance(catalog,dict) or len(catalog)>256:
        raise Fault("ENVIRONMENT_OBSERVATION_UNVERIFIED")
    if not {"provider","backend","project","targets"} <= set(config):
        raise Fault("ENVIRONMENT_SCOPE_INVALID")
    try:
        row=payload["container"]
        if not isinstance(row,dict):
            raise ValueError()
        target=Target(config["provider"],config["backend"],config["project"],
                      logical,row["name"],str(row["createdAt"]),row["username"],
                      row["name"])
        expected=Target(**targets[logical])
        if target.data()!=expected.data():
            raise Fault("TARGET_CHANGED")
        state=row.get("state")
        if state not in _STATES:
            raise Fault("ENVIRONMENT_OBSERVATION_UNVERIFIED")
    except (KeyError, ValueError, TypeError):
        raise Fault("ENVIRONMENT_OBSERVATION_UNVERIFIED") from None
    # No IP addresses, SSH trust, hostnames beyond their immutable digest,
    # arbitrary metadata or user-supplied mount paths enter this report.
    resources=row.get("resources")
    if not isinstance(resources,dict):
        resources={}
    fields={key:_bounded(resources.get(key), optional=True) for key in _RESOURCE}
    tenant=_bounded(row.get("networkTenant"), optional=True)
    report={"identity_digest":target.digest(),"state":state,
            "cpu":fields["cpu"],"memory":fields["memory"],"disk":fields["disk"],
            "network_tenant":tenant}
    scope=_sha({"provider":target.provider,"backend":target.backend,
                "project":target.project})
    capabilities={}
    for name in _ACTIONS:
        candidate=catalog.get(_CANDIDATES[name])
        if candidate is None:
            capabilities[name]={"state":"unsupported",
                                "reason":"TOOL_NOT_EXPOSED"}
        else:
            # A tool name, or its textual description, is insufficient for
            # authorizing destructive actions. Verify schema and live
            # backend/volume coverage in an E2+ approval checkpoint.
            capabilities[name]={"state":"not_verified",
                                "reason":"CONTRACT_AND_STORAGE_NOT_PROVED"}
    completeness=all(report[k] is not None for k in (*_RESOURCE,"network_tenant"))
    when=time.time() if observed_at is None else observed_at
    if type(when) not in (int,float) or not math.isfinite(when) or when<0:
        raise Fault("ENVIRONMENT_OBSERVATION_UNVERIFIED")
    return {"provider_scope_digest":scope,
            "target_identity_digest":target.digest(),
            "observation":report,
            "fingerprint_sha256":_sha(report),
            "observation_status":"complete" if completeness else "partial",
            "storage_coverage":"not_verified",
            "capabilities":capabilities,
            "observed_at":float(when),
            "evidence":"authenticated_provider_read_only"}

def diff_fingerprints(before, after):
    """Return only bounded changes for the same immutable incarnation."""
    if not isinstance(before,dict) or not isinstance(after,dict):
        raise Fault("ENVIRONMENT_TARGET_CHANGED")
    for key in ("target_identity_digest","provider_scope_digest"):
        a,b=before.get(key),after.get(key)
        if (not isinstance(a,str) or re.fullmatch(r"[0-9a-f]{64}",a) is None
                or a!=b):
            raise Fault("ENVIRONMENT_TARGET_CHANGED")
    left,right=before.get("observation"),after.get("observation")
    if (not isinstance(left,dict) or not isinstance(right,dict)
            or any(report.get("fingerprint_sha256")!=_sha(obs)
                   for report,obs in ((before,left),(after,right)))
            or any(obs.get("identity_digest")!=report["target_identity_digest"]
                   for report,obs in ((before,left),(after,right)))):
        raise Fault("ENVIRONMENT_OBSERVATION_UNVERIFIED")
    allowed=("state", *_RESOURCE, "network_tenant")
    changes={}
    unknown=[]
    for k in allowed:
        if k not in left or k not in right or left[k] is None or right[k] is None:
            unknown.append(k)
        elif left[k]!=right[k]:
            changes[k]={"before":left[k],"after":right[k]}
    return {"identity_digest":before["target_identity_digest"],
            "changes":changes,"not_verified":unknown,
            "same_fingerprint":before.get("fingerprint_sha256")==after.get("fingerprint_sha256")}

class EnvironmentInspector:
    def __init__(self, provider):
        self.provider=provider

    def report(self, logical):
        config=self.provider.config
        if not isinstance(config,dict):
            raise Fault("ENVIRONMENT_SCOPE_INVALID")
        targets=config.get("targets")
        if not isinstance(targets,dict) or logical not in targets:
            raise Fault("TARGET_NOT_ALLOWED",403)
        with self.provider.session() as m:
            catalog=m.discover()
            if not isinstance(catalog,dict) or len(catalog)>256:
                raise Fault("ENVIRONMENT_READ_UNSUPPORTED")
            tool=catalog.get("get_container")
            spec=tool.get("inputSchema") if isinstance(tool,dict) else None
            if (not isinstance(spec,dict) or not isinstance(spec.get("properties"),dict)
                    or not isinstance(spec["properties"].get("username"),dict)):
                raise Fault("ENVIRONMENT_READ_UNSUPPORTED")
            result=m.tool("get_container",{"username":logical})
            try:
                payload=json.loads(text(result))
            except (ValueError,TypeError):
                raise Fault("ENVIRONMENT_OBSERVATION_UNVERIFIED") from None
        return observe_exact(config,logical,payload,catalog)
