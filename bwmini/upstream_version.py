"""Conservative local Containarium baseline resolver and native upgrade coordinator.

A release tag is a target, not a BoxWeave compatibility allowlist. Never log JWTs.
Host-wide Provider upgrades are asynchronous and must be observed after dispatch.
"""
import datetime
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import tempfile
import time
from urllib import request, error

BASELINE = "0.128.0"
ASSETS = ("containariumd", "containarium", "mcp-server")
PINNED = {
    "containariumd": "7c19f21b07d8e988ebbfd39243e22cf97d97d9595598b9860f5bc4123e41e09f",
    "containarium": "74560e96c4216cfb83218499682c816a9db78e4f367a4c194a587d650b83f696",
    "mcp-server": "d1c645843ef2989fc5bff53ec57b6612ff931e3b94c9f42d040ee8d7c065b0ba",
}
API = "https://api.github.com/repos/FootprintAI/Containarium/releases"
DOWNLOAD = "https://github.com/FootprintAI/Containarium/releases/download/v"
STATE = Path("/var/lib/boxweave-default")
SEMVER = re.compile(r"(?:v)?([0-9]+\.[0-9]+\.[0-9]+)\Z")


class UpgradeFailure(Exception):
    pass


def require(condition, code):
    if not condition:
        raise UpgradeFailure(code)


def version_tuple(value):
    match = SEMVER.fullmatch(value) if isinstance(value, str) else None
    require(match is not None, "UPSTREAM_VERSION_INVALID")
    return tuple(int(x) for x in match.group(1).split("."))


def normalize_version(value):
    version_tuple(value)
    return value.lstrip("v")


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def trusted(path):
    path = Path(path)
    if path.is_symlink() or not path.is_file():
        return False
    s = path.stat()
    return s.st_uid == 0 and s.st_nlink == 1 and not s.st_mode & 0o022 and bool(s.st_mode & stat.S_IXUSR)


def installed_version(path):
    require(trusted(path), "UPSTREAM_BINARY_UNTRUSTED")
    try:
        proc = subprocess.run([str(path), "version"], capture_output=True,
                              text=True, timeout=10, check=False)
    except (OSError, subprocess.SubprocessError):
        raise UpgradeFailure("UPSTREAM_VERSION_UNKNOWN") from None
    found = set(re.findall(r"(?<![0-9.])v?([0-9]+\.[0-9]+\.[0-9]+)(?![0-9.])",
                           proc.stdout + " " + proc.stderr))
    require(proc.returncode == 0 and len(found) == 1, "UPSTREAM_VERSION_UNKNOWN")
    return found.pop()


def _get_json(url):
    require(url.startswith(API + "/"), "UPSTREAM_RELEASE_SOURCE_INVALID")
    req = request.Request(url, headers={"User-Agent": "BoxWeave-upstream-manager",
                                        "Accept": "application/vnd.github+json"})
    with request.urlopen(req, timeout=20) as response:
        data = response.read(262145)
    require(len(data) <= 262144, "UPSTREAM_RELEASE_METADATA_TOO_LARGE")
    return json.loads(data)


def release(target=None, latest=False):
    """Resolve official published tag and exact SHA-256 asset digests."""
    require(not (latest and target), "UPSTREAM_TARGET_CONFLICT")
    if latest:
        metadata = _get_json(API + "/latest")
        tag = metadata.get("tag_name")
        selected = normalize_version(tag)
    else:
        selected = normalize_version(target or BASELINE)
        metadata = None
    require(version_tuple(selected) >= version_tuple(BASELINE),
            "UPSTREAM_BELOW_BOXWEAVE_BASELINE")
    if selected == BASELINE and not latest:
        return selected, dict(PINNED)
    if metadata is None:
        metadata = _get_json(API + "/tags/v" + selected)
    require(metadata.get("tag_name") == "v" + selected
            and not metadata.get("draft") and not metadata.get("prerelease"),
            "UPSTREAM_RELEASE_NOT_PUBLISHED")
    assets = {a.get("name"): a for a in metadata.get("assets", [])}
    output = {}
    for name in ASSETS:
        item = assets.get(name + "-linux-amd64", {})
        digest = item.get("digest")
        require(item.get("browser_download_url") == DOWNLOAD + selected + "/" + name + "-linux-amd64"
                and isinstance(digest, str) and re.fullmatch(r"sha256:[a-f0-9]{64}", digest),
                "UPSTREAM_ASSET_DIGEST_UNAVAILABLE")
        output[name] = digest[7:]
    return selected, output


def download_assets(folder, selected, digests):
    """Get binaries into a private root-owned stage, verify before any host write."""
    require(folder.is_dir() and not folder.is_symlink(), "UPSTREAM_STAGE_INVALID")
    for name in ASSETS:
        dest = folder / name
        url = DOWNLOAD + selected + "/" + name + "-linux-amd64"
        req = request.Request(url, headers={"User-Agent": "BoxWeave-upstream-manager"})
        with request.urlopen(req, timeout=60) as response, dest.open("xb") as output:
            digest = hashlib.sha256()
            size = 0
            while True:
                block = response.read(1048576)
                if not block:
                    break
                size += len(block)
                require(size <= 180000000, "UPSTREAM_ASSET_TOO_LARGE")
                digest.update(block)
                output.write(block)
            output.flush()
            os.fsync(output.fileno())
        require(digest.hexdigest() == digests[name], "UPSTREAM_ASSET_DIGEST_MISMATCH")
        dest.chmod(0o700)
        # mcp-server is a stdio server and has NO 'version' subcommand.
        # Its digest binds the clean-host binary to the published release;
        # the later real MCP initialize/read validates its running version.
        if name != "mcp-server":
            require(installed_version(dest) == selected, "UPSTREAM_ASSET_VERSION_MISMATCH")
    return folder


def _token(binary, secret, destination):
    secret = Path(secret)
    require(secret.is_file() and not secret.is_symlink() and secret.stat().st_uid == 0
            and not secret.stat().st_mode & 0o077, "UPSTREAM_SIGNING_SECRET_UNTRUSTED")
    with destination.open("xb") as out:
        args = [str(binary), "token", "generate", "--username", "boxweave-upstream",
                "--roles", "admin", "--scopes", "containers:read,containers:write",
                "--expiry", "10m", "--secret-file", str(secret), "--raw"]
        proc = subprocess.run(args, stdin=subprocess.DEVNULL, stdout=out,
                              stderr=subprocess.DEVNULL, timeout=20, check=False)
        out.flush()
        os.fsync(out.fileno())
    require(proc.returncode == 0 and 20 <= destination.stat().st_size <= 8192,
            "UPSTREAM_TOKEN_ISSUANCE_FAILED")
    return destination.read_text().strip()


def mcp_version(binary, daemon_cli, url, secret):
    """Read mcp-server's advertised version via the real stdio MCP initialize.

    Containarium v0.128.0 mcp-server/main.go does not implement 'version'.
    The handshake requires no authorization to mutate a Provider and never
    submits an upstream write.
    """
    require(trusted(binary), "UPSTREAM_MCP_BINARY_UNTRUSTED")
    require(re.fullmatch(r"http://127\\.0\\.0\\.1:[0-9]{4,5}", url) is not None,
            "UPSTREAM_LOOPBACK_REQUIRED")
    with tempfile.TemporaryDirectory(prefix="boxweave-upstream-mcp-", dir=STATE) as folder:
        token_file = Path(folder) / "jwt"
        _token(daemon_cli, secret, token_file)
        payload = json.dumps({
            "jsonrpc": "2.0", "id": 1, "method": "initialize",
            "params": {"protocolVersion": "2024-11-05",
                       "capabilities": {},
                       "clientInfo": {"name": "boxweave-upstream-probe",
                                      "version": "1.0.0"}}
        }) + "\\n"
        env = {"HOME": "/root", "PATH": "/usr/bin:/bin",
               "CONTAINARIUM_SERVER_URL": url,
               "CONTAINARIUM_MCP_ALLOW_INSECURE": "true",
               "CONTAINARIUM_JWT_TOKEN_FILE": str(token_file)}
        try:
            result = subprocess.run([str(binary)], input=payload,
                                    capture_output=True, text=True,
                                    env=env, timeout=15, check=False)
        except (OSError, subprocess.SubprocessError):
            raise UpgradeFailure("UPSTREAM_MCP_VERSION_UNVERIFIED") from None
    require(result.returncode == 0 and len(result.stdout) <= 65536,
            "UPSTREAM_MCP_VERSION_UNVERIFIED")
    lines = result.stdout.strip().splitlines()
    require(len(lines) == 1, "UPSTREAM_MCP_VERSION_UNVERIFIED")
    try:
        reply = json.loads(lines[0])
        require(reply.get("id") == 1 and isinstance(reply.get("result"), dict),
                "UPSTREAM_MCP_VERSION_UNVERIFIED")
        version = reply["result"]["serverInfo"]["version"]
        return normalize_version(version)
    except (TypeError, ValueError, KeyError, AttributeError):
        raise UpgradeFailure("UPSTREAM_MCP_VERSION_UNVERIFIED") from None


def _api(url, token, method, path, body=None, timeout=12):
    require(re.fullmatch(r"http://127\.0\.0\.1:[0-9]{4,5}", url) is not None,
            "UPSTREAM_LOOPBACK_REQUIRED")
    payload = json.dumps(body).encode() if body is not None else None
    req = request.Request(url + path, data=payload, method=method,
                          headers={"Authorization": "Bearer " + token,
                                   "Content-Type": "application/json"})
    with request.build_opener(request.ProxyHandler({})).open(req, timeout=timeout) as resp:
        data = resp.read(131073)
        require(len(data) <= 131072, "UPSTREAM_RESPONSE_TOO_LARGE")
        require(resp.status == 200, "UPSTREAM_HTTP_STATUS_UNEXPECTED")
        return json.loads(data)


def native_upgrade(binary, secret, url, selected, *, deadline=180):
    """Use Containarium's own watchdog upgrade. Never replay uncertain POSTs."""
    with tempfile.TemporaryDirectory(prefix="boxweave-upstream-token-", dir=STATE) as tmp:
        token = _token(binary, secret, Path(tmp) / "jwt")
        try:
            response = _api(url, token, "POST", "/v1/backends/upgrade",
                            {"github_tag": "v" + selected}, timeout=65)
        except error.HTTPError as exc:
            # A definite HTTP rejection occurred before a successful upgrade receipt.
            # Never leak its response body (which may contain private diagnostics).
            if exc.code in (400, 401, 403, 404, 409, 422):
                raise UpgradeFailure("UPSTREAM_UPGRADE_REJECTED_HTTP_" + str(exc.code)) from None
            raise UpgradeFailure("UPSTREAM_UPGRADE_OUTCOME_UNKNOWN_NO_RETRY") from None
        except (OSError, TimeoutError, error.URLError):
            raise UpgradeFailure("UPSTREAM_UPGRADE_OUTCOME_UNKNOWN_NO_RETRY") from None
        require(isinstance(response, dict) and response.get("upgradeId"),
                "UPSTREAM_UPGRADE_RECEIPT_UNKNOWN_NO_RETRY")
        receipt = response["upgradeId"]
        start = time.monotonic()
        while time.monotonic() - start < deadline:
            time.sleep(5)
            try:
                info = _api(url, token, "GET", "/v1/system/info")
                running = info.get("info", {}).get("daemonVersion", "")
                if running == selected:
                    _api(url, token, "GET", "/v1/containers")
                    return {"status": "running_verified", "version": selected, "upgrade_id": receipt}
            except (OSError, error.URLError, ValueError, KeyError):
                continue  # The daemon may be restarting; never re-submit the POST.
        raise UpgradeFailure("UPSTREAM_UPGRADE_PENDING_RECONCILE_ORIGINAL_JOB")


def _atomic_binary(path, staged):
    parent = path.parent
    require(parent.is_dir() and parent.stat().st_uid == 0
            and not parent.stat().st_mode & 0o022, "UPSTREAM_BINARY_DIRECTORY_UNTRUSTED")
    fd, temp = tempfile.mkstemp(prefix=".bw-upstream-", dir=parent)
    try:
        with os.fdopen(fd, "wb") as output:
            os.fchmod(output.fileno(), 0o755)
            output.write(staged.read_bytes())
            output.flush()
            os.fsync(output.fileno())
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def refresh_clients(folder, selected, old, daemon_path, *, managed, url, secret):
    """Only replace conventional root-controlled local CLI and MCP binaries."""
    targets = {"containarium": Path("/usr/local/bin/containarium"),
               "mcp-server": Path("/usr/local/bin/mcp-server")}
    # Check all local binary ownership before replacing any component.
    for dest in targets.values():
        require(not dest.exists() or trusted(dest) or
                (dest == targets["containarium"] and dest.is_symlink()
                 and dest.resolve() == daemon_path.resolve()
                 and trusted(daemon_path)), "UPSTREAM_CLIENT_UNTRUSTED")
    for name, dest in targets.items():
        # A normal upstream CLI symlink to the upgraded daemon is retained.
        if dest.is_symlink():
            require(name == "containarium" and dest.resolve() == daemon_path.resolve(),
                    "UPSTREAM_CLIENT_FOREIGN_SYMLINK")
            require(version_tuple(installed_version(daemon_path)) >= version_tuple(selected), "UPSTREAM_CLI_DAEMON_MISMATCH")
            continue
        require(not dest.exists() or trusted(dest), "UPSTREAM_CLIENT_UNTRUSTED")
        if dest.exists():
            observed = (mcp_version(dest, daemon_path, url, secret)
                        if name == "mcp-server" else installed_version(dest))
            if version_tuple(observed) >= version_tuple(selected):
                continue
        # Save previous verified binary; never overwrite an operator's backup.
        if dest.exists():
            backups = STATE / "upstream-binary-backups"
            backups.mkdir(mode=0o700, exist_ok=True)
            require(not backups.is_symlink() and backups.stat().st_uid == 0,
                    "UPSTREAM_BACKUP_DIRECTORY_UNTRUSTED")
            backup = backups / (name + "-" + sha256(dest.read_bytes()))
            if not backup.exists():
                with backup.open("xb") as out:
                    out.write(dest.read_bytes())
                backup.chmod(0o600)
        _atomic_binary(dest, folder / name)
        observed = (mcp_version(dest, daemon_path, url, secret)
                    if name == "mcp-server" else installed_version(dest))
        require(observed == selected, "UPSTREAM_CLIENT_VERSION_NOT_APPLIED")
        if managed:
            old.setdefault("managed_files", {})[str(dest)] = sha256(dest.read_bytes())


def ensure(old, *, cli, url, secret, target=None, latest=False, standalone=False):
    """Upgrade below-baseline or explicitly requested newer Provider.

    Leaves a healthy >=baseline Provider untouched on an ordinary app install.
    """
    require(os.geteuid() == 0, "ROOT_REQUIRED")
    selected, digests = release(target, latest)
    current = installed_version(cli)
    require(version_tuple(selected) >= version_tuple(BASELINE), "UPSTREAM_BELOW_BOXWEAVE_BASELINE")
    # Upgrade only the required old components; never downgrade a newer daemon
    # or an independently upgraded CLI on an ordinary BoxWeave install.
    if version_tuple(current) > version_tuple(selected) and (target is not None or latest):
        raise UpgradeFailure("UPSTREAM_DOWNGRADE_REFUSED")
    daemon_needs_upgrade = version_tuple(current) < version_tuple(selected)
    mcp = Path("/usr/local/bin/mcp-server")
    mcp_old = mcp_version(mcp, cli, url, secret) if mcp.exists() and trusted(mcp) else None
    client_needs_upgrade = not mcp_old or version_tuple(mcp_old) < version_tuple(selected)
    if not daemon_needs_upgrade and not client_needs_upgrade:
        return {"status": "kept_compatible", "running_version": current, "baseline": BASELINE}
    with tempfile.TemporaryDirectory(prefix="upstream-stage-", dir=STATE) as tmp:
        folder = Path(tmp)
        download_assets(folder, selected, digests)
        if daemon_needs_upgrade:
            if standalone:
                require(Path(cli) == Path("/usr/local/bin/containariumd"),
                        "MANAGED_UPSTREAM_BINARY_LOCATION_CHANGED")
                require(trusted(cli), "UPSTREAM_BINARY_UNTRUSTED")
                backups = STATE / "upstream-binary-backups"
                backups.mkdir(mode=0o700, exist_ok=True)
                require(not backups.is_symlink() and backups.stat().st_uid == 0,
                        "UPSTREAM_BACKUP_DIRECTORY_UNTRUSTED")
                backup = backups / ("containariumd-" + sha256(Path(cli).read_bytes()))
                if not backup.exists():
                    with backup.open("xb") as out:
                        out.write(Path(cli).read_bytes())
                    backup.chmod(0o600)
                _atomic_binary(Path(cli), folder / "containariumd")
                subprocess.run(["systemctl", "restart", "boxweave-provider.service"],
                               stdin=subprocess.DEVNULL, check=True, timeout=45,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                # systemctl restart may return before the HTTP listener is ready.
                # Poll only GET: no retry of an uncertain write.
                with tempfile.TemporaryDirectory(prefix="upstream-check-", dir=STATE) as private:
                    token = _token(Path(cli), secret, Path(private) / "jwt")
                    verified = False
                    deadline = time.monotonic() + 120
                    while time.monotonic() < deadline:
                        try:
                            info = _api(url, token, "GET", "/v1/system/info")
                            if info.get("info", {}).get("daemonVersion") == selected:
                                _api(url, token, "GET", "/v1/containers")
                                verified = True
                                break
                        except (OSError, error.URLError, ValueError, KeyError):
                            pass
                        time.sleep(4)
                    require(verified, "UPSTREAM_RUNNING_DAEMON_VERSION_UNVERIFIED")
                old.setdefault("managed_files", {})[str(cli)] = sha256(Path(cli).read_bytes())
                receipt = {"status": "running_verified", "version": selected}
            else:
                receipt = native_upgrade(cli, secret, url, selected)
        else:
            receipt = {"status": "running_version_unchanged", "version": current}
        refresh_clients(folder, selected, old, Path(cli), managed=standalone,
                        url=url, secret=secret)
        old["upstream_version_observed"] = current if version_tuple(current) > version_tuple(selected) else selected
        old["upstream_version_updated_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        return {**receipt, "baseline": BASELINE, "clients_verified": True}


def execute_installed(argv):
    """Root-owned host CLI entry with the same lock as the streamed installer."""
    import argparse
    parser = argparse.ArgumentParser(prog="sudo boxweave upstream")
    parser.add_argument("command", choices=("status", "upgrade", "update"))
    select = parser.add_mutually_exclusive_group()
    select.add_argument("--version")
    select.add_argument("--latest", action="store_true")
    args = parser.parse_args(argv)
    require(os.geteuid() == 0, "ROOT_REQUIRED")
    marker = STATE / "state.json"
    require(marker.is_file() and not marker.is_symlink() and marker.stat().st_uid == 0
            and not marker.stat().st_mode & 0o077, "MANAGED_INSTALL_REQUIRED")
    lock = os.open(STATE / ".install.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        old = json.loads(marker.read_text())
        require(old.get("hostname") == __import__("socket").gethostname()
                and old.get("phase") == "ready", "UPSTREAM_HOST_IDENTITY_OR_INSTALL_UNREADY")
        standalone = old.get("provider_mode") == "standalone"
        provider = old.get("external_provider") if not standalone else None
        cli = Path(provider["cli"] if provider else "/usr/local/bin/containariumd")
        url = provider["url"] if provider else "http://127.0.0.1:18080"
        secret = provider["secret"] if provider else "/etc/boxweave-provider/jwt.secret"
        current = installed_version(cli)
        result = {"baseline": BASELINE, "installed": current,
                  "mode": old.get("provider_mode"), "maximum": None}
        if args.command in ("upgrade", "update"):
            require(args.version is not None or args.latest, "UPSTREAM_TARGET_REQUIRED")
            result["upgrade"] = ensure(old, cli=cli, url=url, secret=secret,
                                       target=args.version, latest=args.latest, standalone=standalone)
            fd, name = tempfile.mkstemp(prefix=".upstream-state-", dir=STATE)
            try:
                with os.fdopen(fd, "w") as output:
                    os.fchmod(output.fileno(), 0o600)
                    json.dump(old, output, sort_keys=True)
                    output.flush()
                    os.fsync(output.fileno())
                os.replace(name, marker)
            finally:
                if os.path.exists(name):
                    os.unlink(name)
        print(json.dumps(result, sort_keys=True))
        return 0
    finally:
        os.close(lock)


def main(argv):
    try:
        return execute_installed(argv)
    except (UpgradeFailure, OSError, ValueError, KeyError, subprocess.SubprocessError):
        import sys
        error_ = sys.exc_info()[1]
        code = str(error_) if isinstance(error_, UpgradeFailure) else "UPSTREAM_UPGRADE_INCOMPLETE_INSPECT_ORIGINAL_JOB"
        print(json.dumps({"state": "incomplete", "code": code, "retry": "Inspect original daemon/job before another upgrade."}))
        return 2
