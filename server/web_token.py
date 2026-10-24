#!/usr/bin/env python3
"""Root-only independent web credential issuer. Never opens the MCP token."""
import base64
import time
import json
import os
import pwd
import subprocess
import tempfile
from pathlib import Path


def main():
    if os.geteuid() != 0:
        raise SystemExit("Must run as root")
    cfg = json.loads(Path("/etc/boxweave/config.json").read_text())
    account = pwd.getpwnam("boxweave-web")
    target = Path("/var/lib/boxweave-web-auth/provider.jwt")
    if target.parent.is_symlink() or target.is_symlink():
        raise SystemExit("Refusing credential symlink")
    args = [
        cfg["containarium_daemon_binary"],
        "token",
        "generate",
        "--username",
        "boxweave-web",
        "--roles",
        "admin",
        "--scopes",
        "containers:read,containers:write,routes:read,routes:write",
        "--expiry",
        "2h",
        "--secret-file",
        cfg["jwt_secret_file"],
        "--raw",
    ]
    result = subprocess.run(args, capture_output=True, text=True, timeout=30)
    if (
        result.returncode
        or not result.stdout.strip().startswith("eyJ")
        or len(result.stdout.strip().split(".")) != 3
    ):
        raise SystemExit("Web credential issuance failed (details suppressed)")
    try:
        payload = result.stdout.strip().split(".")[1]
        claims = json.loads(
            base64.urlsafe_b64decode(payload + "=" * (-len(payload) % 4))
        )
        required = {
            "containers:read",
            "containers:write",
            "routes:read",
            "routes:write",
        }
        if (
            claims.get("username") != "boxweave-web"
            or set(claims.get("scopes", [])) != required
            or claims.get("exp", 0) > time.time() + 7300
            or claims.get("exp", 0) < time.time() + 60
        ):
            raise ValueError("Unexpected claims")
    except (ValueError, TypeError, KeyError):
        raise SystemExit("Web issuer returned unexpected subject, scopes or expiry")
    fd, temp = tempfile.mkstemp(prefix=".provider.", dir=target.parent)
    try:
        with os.fdopen(fd, "w") as f:
            f.write(result.stdout.strip() + "\n")
            f.flush()
            os.fsync(f.fileno())
        os.chmod(temp, 0o600)
        os.chown(temp, account.pw_uid, account.pw_gid)
        os.replace(temp, target)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)
    print(
        "Independent web credential rotated; previous credential expires within two hours."
    )


if __name__ == "__main__":
    main()
