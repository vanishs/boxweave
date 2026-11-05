"""Reviewed-identity and private SQLite adapter for the shared claim core.

This adapter does not provision files, authorize callers, or enable coordination.
Only the explicit installer/provisioner may prepare the shared host state.
"""
import hashlib
import json
import os
from pathlib import Path
import re
import sqlite3
import stat
from urllib.parse import quote

# Standalone forced-command helpers have no package parent. Import only the
# explicitly bundled sibling, never a path supplied by an MCP caller.
if __package__:
    from .operation_coordination import CoordinationFault
else:
    from operation_coordination import CoordinationFault

MAX_DB_BYTES = 64 * 1024 * 1024


def _field(value):
    if not isinstance(value, str) or not 1 <= len(value) <= 200 or any(ord(ch) < 32 for ch in value):
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    return value


def scope_digest(provider, backend, project):
    """A controller-verified Provider/backend/project scope, never request data."""
    fields = {"provider": _field(provider), "backend": _field(backend),
              "project": _field(project)}
    return hashlib.sha256(json.dumps(fields, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def canonical_box_identity(logical, instance, created, user, *, provider, backend, project):
    """Unique immutable incarnation *within* the verified upstream scope."""
    if not isinstance(logical, str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}", logical) is None:
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    if not isinstance(user, str) or re.fullmatch(r"[a-z_][a-z0-9_-]{0,63}", user) is None:
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    fields = {"scope": scope_digest(provider, backend, project),
              "logical": logical, "instance": _field(instance),
              "created": _field(created), "user": user}
    return hashlib.sha256(json.dumps(fields, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def app_target_identity(target):
    """Call only AFTER the app's existing Provider/Target observation and pin."""
    return canonical_box_identity(
        target.logical, target.instance, target.created, target.user,
        provider=target.provider, backend=target.backend, project=target.project)


def gateway_target_identity(container, *, scope, logical):
    """Require root-reviewed scope and the exact already-trusted guest.

    Do not infer scope from untrusted tools/call arguments, IP or hostname.
    """
    if not isinstance(container, dict) or not isinstance(scope, dict):
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    if set(scope) != {"provider", "backend", "project"}:
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    try:
        user, instance, created = container["username"], container["name"], container["createdAt"]
        if user != logical or type(created) not in (str, int):
            raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
        return canonical_box_identity(
            logical, instance, str(created), user,
            provider=scope["provider"], backend=scope["backend"], project=scope["project"])
    except (KeyError, TypeError, ValueError):
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID") from None


def _owned_regular(path, owner):
    meta = path.lstat()
    if (not stat.S_ISREG(meta.st_mode) or meta.st_uid != owner
            or meta.st_mode & 0o077 or meta.st_nlink != 1
            or meta.st_size > MAX_DB_BYTES):
        raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
    return meta


def _checked_parents(parent, owner):
    # Root-owned public /usr, /var and / are traversable but not writable.
    # A sticky root-owned /tmp is permitted for isolated fixture tests.
    for ancestor in (parent, *parent.parents):
        meta = ancestor.lstat()
        if not stat.S_ISDIR(meta.st_mode):
            raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
        if ancestor == parent:
            if meta.st_uid != owner or meta.st_mode & 0o077:
                raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
        elif (meta.st_mode & 0o022 and not
              (meta.st_uid == 0 and meta.st_mode & stat.S_ISVTX and
               stat.S_ISDIR(meta.st_mode))):
            raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")


def private_connection(path):
    """Open pre-provisioned private coordination SQLite; never auto-create.

    This contract deliberately requires the same trusted effective UID on
    Web worker and standalone SSH/HTTP gateway. Cross-UID deployment is
    unsupported without a separate reviewed mediation boundary.
    """
    candidate = Path(path)
    if not candidate.is_absolute() or ".." in candidate.parts:
        raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
    try:
        owner = os.geteuid()
        _checked_parents(candidate.parent, owner)
        before = _owned_regular(candidate, owner)
        for suffix in ("-wal", "-shm", "-journal"):
            side = Path(str(candidate) + suffix)
            try:
                _owned_regular(side, owner)
            except FileNotFoundError:
                pass
        url = "file:" + quote(str(candidate), safe="/") + "?mode=rw"
        connection = sqlite3.connect(url, uri=True, timeout=2)
        try:
            after = _owned_regular(candidate, owner)
            if (before.st_dev, before.st_ino) != (after.st_dev, after.st_ino):
                raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
            if connection.execute("PRAGMA user_version").fetchone()[0] not in (0, 1):
                raise CoordinationFault("COORDINATION_SCHEMA_UNSUPPORTED")
            present = {r[0] for r in connection.execute(
                "SELECT name FROM sqlite_master WHERE type='table'")}
            base = {"coordination_operations", "coordination_resources", "coordination_version"}
            # Only reviewed additive schema generations; a partial or
            # unexpected table set fails closed, never silently reinitializes.
            leased = base | {"coordination_leases"}
            recovered = leased | {"coordination_recovery_reviews", "coordination_recovery_audit"}
            if present and present not in (base, leased, recovered):
                raise CoordinationFault("COORDINATION_SCHEMA_UNSUPPORTED")
            if present:
                row = connection.execute(
                    "SELECT version FROM coordination_version WHERE singleton=1").fetchone()
                if row is None or row[0] != 1:
                    raise CoordinationFault("COORDINATION_SCHEMA_UNSUPPORTED")
            return connection
        except BaseException:
            connection.close()
            raise
    except (OSError, sqlite3.Error) as exc:
        raise CoordinationFault("COORDINATION_UNAVAILABLE") from exc
