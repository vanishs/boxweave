"""Root-only fast local Provider-scope heartbeat for R-04 coordination.

Unlike SSH host-key inventory, this attestation never enters any guest.
A stale/missing heartbeat fails closed; it cannot authorize guest trust.
"""
import json
import os
import platform
from pathlib import Path
import stat
import subprocess
import time

try:
    from .hostkeys import atomic
except ImportError:
    from hostkeys import atomic

STATE = Path("/var/lib/boxweave-default/state.json")
PUBLIC = Path("/var/lib/boxweave-default/public")
SCOPE = PUBLIC / "scope.json"

def publish(*, state=STATE, directory=PUBLIC, now=time.time,
            inspect=subprocess.run, hostname=platform.node):
    if os.geteuid() != 0:
        raise ValueError("ROOT_REQUIRED")
    meta = state.lstat()
    parent = directory.lstat()
    if (not stat.S_ISREG(meta.st_mode) or meta.st_uid != 0
            or meta.st_nlink != 1 or meta.st_mode & 0o077
            or not stat.S_ISDIR(parent.st_mode) or parent.st_uid != 0
            or parent.st_mode & 0o022):
        raise ValueError("SCOPE_STATE_UNVERIFIED")
    record = json.loads(state.read_text())
    node = hostname()
    if (not isinstance(record, dict) or record.get("hostname") != node
            or not isinstance(record.get("managed_files"), dict)
            or not record["managed_files"]
            or not isinstance(node, str) or not node or len(node) > 128):
        raise ValueError("SCOPE_STATE_UNVERIFIED")
    # Bound a native local-daemon liveness proof. Do not use SSH key scans
    # or a caller-supplied backend identity to refresh R-04 authority.
    try:
        result = inspect(["/usr/bin/incus", "--force-local", "query", "/1.0"],
                         stdin=subprocess.DEVNULL, capture_output=True,
                         text=True, timeout=5, check=False)
        if result.returncode != 0 or not isinstance(json.loads(result.stdout), dict):
            raise ValueError()
    except (OSError, subprocess.SubprocessError, ValueError, TypeError):
        raise ValueError("SCOPE_NATIVE_PROVIDER_UNVERIFIED") from None
    data = {"observed_at": now(),
            "scope": {"provider":"containarium","backend":node,"project":"default"}}
    payload = json.dumps(data, sort_keys=True, separators=(",",":")).encode()
    # Never follow an existing symlink or overwrite a foreign-owned file.
    if SCOPE.exists() or SCOPE.is_symlink():
        existing = SCOPE.lstat()
        if (not stat.S_ISREG(existing.st_mode) or existing.st_uid != 0
                or existing.st_nlink != 1 or existing.st_mode & 0o022):
            raise ValueError("SCOPE_PUBLICATION_UNSAFE")
    atomic(SCOPE, payload, 0, 0, 0o644)
    return data

if __name__ == "__main__":
    publish()
