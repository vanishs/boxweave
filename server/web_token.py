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
from urllib.request import Request, build_opener, HTTPRedirectHandler
from urllib.parse import urlsplit
import ipaddress


BASE_SCOPES = {"containers:read", "containers:write", "routes:read", "routes:write"}
NETWORK_SCOPES = {"network-policy:read", "network-policy:write"}


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def network_contract_available(origin, token):
    """Authenticate with a freshly minted baseline token; never read MCP keys."""
    try:
        url = urlsplit(origin)
        if (
            url.scheme not in ("http", "https")
            or not (
                url.hostname == "localhost"
                or ipaddress.ip_address(url.hostname).is_loopback
            )
            or url.username
            or url.password
            or url.path not in ("", "/")
        ):
            return False
        request = Request(
            origin.rstrip("/") + "/v1/network-tenants/capabilities",
            headers={"Authorization": "Bearer " + token},
        )
        with build_opener(NoRedirect()).open(request, timeout=5) as response:
            data = json.loads(response.read(65537))
        return (
            isinstance(data, dict)
            and data.get("contract") == "native-network-tenant/v1"
            and data.get("atomicTenantAccess") is True
            and data.get("backendScope") == "local-lxc"
        )
    except (OSError, ValueError, TypeError):
        return False


def issue(args):
    result = subprocess.run(args, capture_output=True, text=True, timeout=30)
    if (
        result.returncode
        or not result.stdout.strip().startswith("eyJ")
        or len(result.stdout.strip().split(".")) != 3
    ):
        raise SystemExit("Web credential issuance failed (details suppressed)")
    return result


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
    required = set(BASE_SCOPES)
    result = issue(args)
    if network_contract_available(cfg["api_url"], result.stdout.strip()):
        required.update(NETWORK_SCOPES)
        args[args.index("--scopes") + 1] = ",".join(sorted(required))
        result = issue(args)
    try:
        payload = result.stdout.strip().split(".")[1]
        claims = json.loads(
            base64.urlsafe_b64decode(payload + "=" * (-len(payload) % 4))
        )
        if (
            claims.get("username") != "boxweave-web"
            or "admin" not in claims.get("roles", [])
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
