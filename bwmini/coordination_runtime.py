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

from .operation_coordination import CoordinationFault

MAX_DB_BYTES = 64 * 1024 * 1024


def canonical_box_identity(logical, instance, created, user):
    """Same incarnation key for independently verified app and gateway data."""
    if not isinstance(logical, str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}", logical) is None:
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    if not isinstance(user, str) or re.fullmatch(r"[a-z_][a-z0-9_-]{0,63}", user) is None:
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    for value in (instance, created):
        if not isinstance(value, str) or not 1 <= len(value) <= 200 or any(ord(ch) < 32 for ch in value):
            raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    fields = {"logical": logical, "instance": instance, "created": created, "user": user}
    return hashlib.sha256(json.dumps(fields, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def app_target_identity(target):
    """Call only AFTER the app's existing Provider/Target observation and pin."""
    return canonical_box_identity(target.logical, target.instance, target.created, target.user)


def gateway_target_identity(container):
    """Call only AFTER the gateway's trusted registry and Provider comparison."""
    if not isinstance(container, dict):
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID")
    try:
        return canonical_box_identity(
            container["username"], container["name"],
            str(container["createdAt"]), container["username"])
    except (KeyError, TypeError, ValueError):
        raise CoordinationFault("COORDINATION_IDENTITY_INVALID") from None


def private_connection(path):
    """Open a pre-provisioned private DB; missing/unsafe means NO dispatch.

    The path must be server-owned configuration, never an MCP/Web request arg.
    The app and SSH/HTTP gateway may share only this minimal database, not the
    application DB or gateway-private task/output store.
    """
    candidate = Path(path)
    if not candidate.is_absolute() or ".." in candidate.parts:
        raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
    try:
        parent = candidate.parent.lstat()
        before = candidate.lstat()
        owner = os.geteuid()
        if (not stat.S_ISDIR(parent.st_mode) or parent.st_uid != owner
                or parent.st_mode & 0o077
                or not stat.S_ISREG(before.st_mode) or before.st_uid != owner
                or before.st_mode & 0o077 or before.st_nlink != 1
                or before.st_size > MAX_DB_BYTES):
            raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
        url = "file:" + quote(str(candidate), safe="/") + "?mode=rw"
        connection = sqlite3.connect(url, uri=True, timeout=2)
        try:
            after = candidate.lstat()
            if (before.st_dev, before.st_ino) != (after.st_dev, after.st_ino):
                raise CoordinationFault("COORDINATION_STORAGE_UNSAFE")
            return connection
        except BaseException:
            connection.close()
            raise
    except (OSError, sqlite3.Error) as exc:
        raise CoordinationFault("COORDINATION_UNAVAILABLE") from exc
