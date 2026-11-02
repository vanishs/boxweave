"""Shared, separately provisioned operation-claim entrypoints.

Standalone root-published gateway and package-owned app deliberately use one
private SQLite file, one exact Provider scope and one durable claim namespace.
This module does not enable protection of legacy forwarded MCP mutations.
"""
import os
from pathlib import Path
import json
import stat
import time

if __package__:
    from .coordination_runtime import (
        app_target_identity, gateway_target_identity, private_connection, scope_digest)
    from .operation_coordination import (
        OperationCoordinator, CoordinationFault, box_resource, name_resource)
else:
    from coordination_runtime import (
        app_target_identity, gateway_target_identity, private_connection, scope_digest)
    from operation_coordination import (
        OperationCoordinator, CoordinationFault, box_resource, name_resource)

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
