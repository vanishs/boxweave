"""Shared, separately provisioned operation-claim entrypoints.

Standalone root-published gateway and package-owned app deliberately use one
private SQLite file, one exact Provider scope and one durable claim namespace.
This module does not enable protection of legacy forwarded MCP mutations.
"""
import os
import hashlib
from pathlib import Path
import json
import stat
import time

if __package__:
    from .coordination_runtime import (
        app_target_identity, gateway_target_identity, private_connection, scope_digest)
    from .operation_coordination import (
        OperationCoordinator, CoordinationFault, box_resource, name_resource,
        tenant_resource, port_resource)
else:
    from coordination_runtime import (
        app_target_identity, gateway_target_identity, private_connection, scope_digest)
    from operation_coordination import (
        OperationCoordinator, CoordinationFault, box_resource, name_resource,
        tenant_resource, port_resource)

DEFAULT_CLAIMS = Path("/home/boxweave/.boxweave/claims.sqlite3")
PUBLIC_SCOPE = Path("/var/lib/boxweave-default/public/targets.json")
_SCOPE_KEYS = {"provider", "backend", "project"}

def checked_scope(value):
    if not isinstance(value, dict) or set(value) != _SCOPE_KEYS:
        raise CoordinationFault("COORDINATION_SCOPE_UNVERIFIED")
    scope_digest(value["provider"], value["backend"], value["project"])
    return value

def published_scope(path=PUBLIC_SCOPE, *, now=time.time):
    """Only a recently root-published scope; never app-supplied scope metadata."""
    try:
        parent=path.parent.lstat()
        meta=path.lstat()
        if (not stat.S_ISDIR(parent.st_mode) or parent.st_uid!=0 or parent.st_mode&0o022
                or not stat.S_ISREG(meta.st_mode) or meta.st_uid!=0
                or meta.st_mode&0o022 or meta.st_nlink!=1 or meta.st_size>2_097_152):
            raise CoordinationFault("COORDINATION_SCOPE_UNVERIFIED")
        value=json.loads(path.read_text())
        if (not isinstance(value,dict)
                or type(value.get("observed_at")) not in (int,float)
                or not 0<=now()-value["observed_at"]<=45):
            raise CoordinationFault("COORDINATION_SCOPE_UNVERIFIED")
        return checked_scope(value.get("scope"))
    except (OSError, UnicodeError, ValueError, TypeError):
        raise CoordinationFault("COORDINATION_SCOPE_UNVERIFIED") from None

def app_box(target, scope):
    checked_scope(scope)
    if {k:getattr(target,k,None) for k in _SCOPE_KEYS}!=scope:
        raise CoordinationFault("COORDINATION_SCOPE_MISMATCH")
    return (app_target_identity(target), scope_digest(**scope), target.logical)

def gateway_box(observed, scope, logical):
    scope=checked_scope(scope)
    return (gateway_target_identity(observed,scope=scope,logical=logical),
            scope_digest(**scope),logical)

def app_environment_keys(config, action, logical, target, spec):
    """Exact app-owned lifecycle keys, including pre-creation logical names."""
    if action not in ("create", "start", "stop", "delete", "resize"):
        raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
    if not isinstance(config,dict) or not isinstance(spec,dict):
        raise CoordinationFault("COORDINATION_SCOPE_UNVERIFIED")
    scope=published_scope()
    if {k:config.get(k) for k in _SCOPE_KEYS} != scope:
        raise CoordinationFault("COORDINATION_SCOPE_MISMATCH")
    scope_id=scope_digest(**scope)
    if target is None:
        if action!="create":
            raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
        namekey=name_resource(scope_id,logical)
        identity=hashlib.sha256(json.dumps({
            "scope":scope_id,"logical":logical,"resource":"pre_creation"
        },sort_keys=True,separators=(",",":")).encode()).hexdigest()
        resources=[namekey]
    else:
        if action=="create":
            raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
        identity,other_scope,observed_name=app_box(target,scope)
        if observed_name!=logical or other_scope!=scope_id:
            raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
        resources=[box_resource(identity),name_resource(scope_id,logical)]
    tenant=spec.get("_network_tenant")
    if tenant is not None:
        resources.append(tenant_resource(scope_id,tenant))
    return identity,resources


def app_access_keys(config, payload):
    """Resources for a previously validated, exact-scope Access intent.

    Access.preflight must independently pin the Provider revision/target.
    This does not grant new privileges or perform any Provider operation.
    """
    scope = published_scope()
    if not isinstance(config, dict) or {
        k: config.get(k) for k in _SCOPE_KEYS} != scope:
        raise CoordinationFault("COORDINATION_SCOPE_MISMATCH")
    if not isinstance(payload, dict):
        raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
    scope_id = scope_digest(**scope)
    if payload.get("kind") == "network":
        changes = payload.get("changes")
        if not isinstance(changes, list) or not 1 <= len(changes) <= 2:
            raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
        keys = [tenant_resource(scope_id, c["destination"]) for c in changes
                if isinstance(c, dict) and "destination" in c]
        if len(keys) != len(changes) or len(set(keys)) != len(keys):
            raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
        identity = hashlib.sha256(json.dumps({
            "scope": scope_id, "kind": "network", "resources": sorted(keys)
        }, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
        return identity, keys
    if payload.get("kind") == "port":
        if not __package__:
            raise CoordinationFault("COORDINATION_RESOURCE_INVALID")
        from .domain import Target
        try:
            target = Target(**payload["target"])
            route = payload["route"]
            identity, actual_scope, logical = app_box(target, scope)
            if actual_scope != scope_id or route["container"] != target.instance:
                raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
            keys = [box_resource(identity), name_resource(scope_id, logical),
                    port_resource(scope_id, route["protocol"], route["external_port"])]
            return identity, keys
        except (KeyError, TypeError, ValueError) as exc:
            raise CoordinationFault("COORDINATION_RESOURCE_INVALID") from exc
    raise CoordinationFault("COORDINATION_RESOURCE_INVALID")


class SharedClaims:
    def __init__(self, path=DEFAULT_CLAIMS, *, clock=time.time):
        self.path=Path(path)
        self.clock=clock
        self.coordinator=OperationCoordinator(
            lambda:private_connection(self.path), clock=clock)

    def acquire(self, operation_id, receipt_ref, identity, scope_id, logical, lease_token=None):
        resources=[box_resource(identity),name_resource(scope_id,logical)]
        self.coordinator.acquire(operation_id,identity,receipt_ref,resources,
                                 lease_token=lease_token)

    def acquire_environment(self, operation_id, receipt_ref, config,
                            action, logical, target, spec):
        identity,resources=app_environment_keys(config,action,logical,target,spec)
        self.coordinator.acquire(operation_id,identity,receipt_ref,resources)

    def acquire_access(self, operation_id, receipt_ref, config, payload):
        identity, resources = app_access_keys(config, payload)
        self.coordinator.acquire(operation_id, identity, receipt_ref, resources)

    def dispatch_intent(self, operation_id):
        self.coordinator.dispatch_intent(operation_id)

    def unknown(self, operation_id):
        self.coordinator.unknown(operation_id)

    def cancel_before_dispatch(self, operation_id):
        self.coordinator.cancel_before_dispatch(operation_id)

    def observed_terminal(self, operation_id, receipt_ref):
        self.coordinator.observed_terminal(operation_id,receipt_ref)

    def lease(self, action, identity, token=None, seconds=600):
        return self.coordinator.lease(action,box_resource(identity),identity,
                                      token=token,seconds=seconds)
