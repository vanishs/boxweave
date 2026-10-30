"""Explicit root coordinator for an independent, non-root BoxWeave application.
Provider installation remains a separately reviewed prerequisite. Never reads a JWT.
"""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import platform
import pwd
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import time
from urllib.parse import urlsplit

SCOPES = "containers:read,containers:write,ssh:write,routes:read,routes:write,security:read,network-policy:read,network-policy:write"
BASE = Path("/var/lib/boxweave-bootstrap")
PUBLIC = Path("/usr/local/lib/boxweave")
UNIT_DIR = Path("/etc/systemd/system")


class Refused(Exception):
    pass


def require(value, code):
    if not value:
        raise Refused(code)


def safe_path(path):
    path = Path(path)
    require(path.is_absolute() and path == path.absolute() and
            not any(c in str(path) for c in "\n\r\x00") and ".." not in path.parts,
            "ABSOLUTE_SAFE_PATH_REQUIRED")
    for parent in (path, *path.parents):
        require(not parent.is_symlink(), "SYMLINK_REFUSED")
    return path


def reviewed(path, expected):
    require(isinstance(expected, str) and re.fullmatch("[a-f0-9]{64}", expected), "SHA256_REQUIRED")
    path = safe_path(path)
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, "rb") as f:
        info = os.fstat(f.fileno())
        require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1, "REGULAR_ARTIFACT_REQUIRED")
        data = f.read(180_000_001)
    require(len(data) <= 180_000_000, "ARTIFACT_TOO_LARGE")
    require(hashlib.sha256(data).hexdigest() == expected, "ARTIFACT_DIGEST_MISMATCH")
    return data


def root_private(path, directory=False):
    path = safe_path(path)
    for parent in path.parents:
        meta = parent.stat()
        require(meta.st_uid == 0 and not meta.st_mode & 0o022, "ROOT_OWNED_PARENT_REQUIRED")
    info = path.stat()
    require(info.st_uid == 0 and not info.st_mode & 0o077 and
            (stat.S_ISDIR(info.st_mode) if directory else stat.S_ISREG(info.st_mode) and info.st_nlink == 1),
            "ROOT_PRIVATE_PATH_REQUIRED")
    return path


def endpoint(value):
    try:
        u = urlsplit(value)
        require(u.scheme == "http" and u.hostname == "127.0.0.1" and
                u.username is None and u.password is None and
                not u.path and not u.query and not u.fragment and
                u.port is not None and 1024 <= u.port <= 65535,
                "LOOPBACK_PROVIDER_URL_REQUIRED")
    except ValueError:
        raise Refused("LOOPBACK_PROVIDER_URL_REQUIRED") from None
    return value


def atomic(path, data, mode=0o600, owner=None):
    path = safe_path(path)
    fd, tmp = tempfile.mkstemp(prefix=".stage-", dir=path.parent)
    try:
        os.fchmod(fd, mode)
        if owner:
            os.fchown(fd, *owner)
        with os.fdopen(fd, "wb") as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp, path)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def make_directory(path, mode):
    path.mkdir(mode=mode, parents=True)
    path.chmod(mode)  # Preserve the intended public traversal bits under umask 077.


def encoded(value):
    return (json.dumps(value, sort_keys=True, indent=2) + "\n").encode()


def run(argv, deadline, env=None):
    try:
        result = subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL, env=env,
                                timeout=max(0.01, deadline-time.monotonic()))
    except subprocess.TimeoutExpired:
        raise Refused("PHASE_DEADLINE_EXCEEDED") from None
    require(result.returncode == 0, "COMMAND_FAILED:" + Path(argv[0]).name)


def launcher_text(binary, token, url, home):
    # All paths are generated from a restricted username, never operator shell fragments.
    return ("#!/bin/sh\nexec /usr/bin/env -i HOME=" + home +
            " PATH=/usr/bin:/bin CONTAINARIUM_MCP_ALLOW_INSECURE=true CONTAINARIUM_SERVER_URL=" + url +
            " CONTAINARIUM_JWT_TOKEN_FILE=" + str(token) + " " + str(binary) + "\n").encode()


def rotate(state_path):
    require(os.geteuid() == 0, "ROOT_REQUIRED")
    state_path = root_private(state_path)
    root_private(state_path.parent, True)
    state = json.loads(state_path.read_text())
    uid = pwd.getpwnam(state["user"]).pw_uid
    for name in ("provider", "mcp"):
        reviewed(state[name], state[name + "_sha256"])
    root_private(state["secret"])
    token = safe_path(state["token"])
    parent = token.parent.stat()
    require(parent.st_uid == 0 and not parent.st_mode & 0o022, "ROOT_OWNED_CREDENTIAL_DIRECTORY_REQUIRED")
    lock = os.open(state_path.parent / ".rotate.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        fd, tmp = tempfile.mkstemp(prefix=".token-", dir=token.parent)
        try:
            # Provider stdout goes directly into the private file. No JWT is read,
            # placed in argv/environment, returned, or written to journal output.
            with os.fdopen(fd, "wb") as output:
                result = subprocess.run([state["provider"], "token", "generate",
                    "--username", state["user"], "--roles", "admin", "--scopes", SCOPES,
                    "--expiry", "48h", "--secret-file", state["secret"], "--raw"],
                    stdin=subprocess.DEVNULL, stdout=output, stderr=subprocess.DEVNULL, timeout=20)
                require(result.returncode == 0, "TOKEN_ISSUANCE_FAILED")
                output.flush()
                os.fsync(output.fileno())
                info = os.fstat(output.fileno())
                require(20 <= info.st_size <= 8192, "TOKEN_OUTPUT_INVALID")
                os.fchmod(output.fileno(), 0o400)
                os.fchown(output.fileno(), uid, pwd.getpwnam(state["user"]).pw_gid)
            os.replace(tmp, token)
        finally:
            if os.path.exists(tmp):
                os.unlink(tmp)
    finally:
        os.close(lock)
    return {"state": "rotated", "credential_contents": "not_read", "expiry_hours": 48}


def prepare(args):
    require(re.fullmatch("[a-z_][a-z0-9_-]{0,30}", args.app_user) and args.app_user != "root",
            "APPLICATION_USER_INVALID")
    endpoint(args.provider_url)
    require(1024 <= args.port <= 65535 and args.port != urlsplit(args.provider_url).port, "APP_PORT_INVALID")
    artifacts = {
        "package": reviewed(args.package, args.sha256),
        "distribution": reviewed(args.distribution, args.distribution_sha256),
        "mcp": reviewed(args.mcp_binary, args.mcp_sha256),
        "provider": reviewed(args.provider_cli, args.provider_cli_sha256),
    }
    root_private(args.secret_file)
    require(shutil.which("incus") and shutil.which("containarium"), "REVIEWED_PROVIDER_REQUIRED")
    return artifacts


def apply(args, artifacts):
    require(os.geteuid() == 0, "ROOT_REQUIRED")
    require(args.ack_root_changes == "INITIALIZE_BOXWEAVE", "ROOT_CHANGE_ACK_REQUIRED")
    require(sys.platform == "linux" and platform.machine() == "x86_64" and
            sys.version_info[:2] == (3, 12), "UBUNTU_2404_PYTHON312_REQUIRED")
    os_release = platform.freedesktop_os_release()
    require(os_release.get("ID") == "ubuntu" and os_release.get("VERSION_ID") == "24.04",
            "UBUNTU_2404_REQUIRED")
    # No APT, upstream upgrade, host SSH/firewall, route, or public listener changes.
    deadline = time.monotonic() + 600
    for tool in ("runuser", "useradd", "loginctl", "systemctl"):
        require(shutil.which(tool), "PREREQUISITE_MISSING:" + tool)
    require(Path("/usr/lib/python3.12/ensurepip").is_dir(), "PYTHON_VENV_PREREQUISITE_REQUIRED")
    user = args.app_user
    folder = BASE / user
    public = PUBLIC / user
    state_path = folder / "state.json"
    for parent in (BASE, PUBLIC):
        safe_path(parent)
        if not parent.exists():
            make_directory(parent, 0o755)
        info = parent.stat()
        require(info.st_uid == 0 and not info.st_mode & 0o022, "ROOT_OWNED_BASE_REQUIRED")
    fresh = not folder.exists()
    if fresh:
        require(not public.exists(), "FOREIGN_BOOTSTRAP_PATH")
        try:
            pwd.getpwnam(user)
        except KeyError:
            pass
        else:
            raise Refused("EXISTING_APPLICATION_USER_NOT_ADOPTED")
        folder.mkdir(mode=0o700)
    root_private(folder, True)
    lock = os.open(folder / ".install.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise Refused("BOOTSTRAP_BUSY") from None
        old = json.loads(root_private(state_path).read_text()) if state_path.exists() else None
        require(fresh or old is not None, "BOOTSTRAP_STATE_MISSING_REVIEW_REQUIRED")
        if old:
            require(old["user"] == user and old["url"] == args.provider_url and old["port"] == args.port and
                    old["secret"] == str(safe_path(args.secret_file)) and old.get("ssh_entry", False) == args.ssh_entry, "BOOTSTRAP_SETTINGS_CONFLICT")
            for name, data in (("mcp", artifacts["mcp"]), ("provider", artifacts["provider"])):
                require(old[name + "_sha256"] == hashlib.sha256(data).hexdigest(), "PROVIDER_UPGRADE_NOT_ALLOWED")
            for path, sha in old["managed_files"].items():
                reviewed(path, sha)
        else:
            # Snapshot reviewed bytes before any privileged execution or account setup.
            make_directory(public, 0o755)
            run(["useradd", "--create-home", "--shell", "/bin/sh" if args.ssh_entry else "/usr/sbin/nologin", user], deadline)
        # Only this managed namespace is public; the token itself remains mode 0400.
        PUBLIC.chmod(0o755)
        public.chmod(0o755)
        account = pwd.getpwnam(user)
        home = safe_path(account.pw_dir)
        require(home == Path("/home") / user, "APPLICATION_HOME_UNEXPECTED")
        home.chmod(0o700)
        owner = (account.pw_uid, account.pw_gid)
        prefix = home / "boxweave"
        # The two provider executables are snapshotted into root-controlled paths.
        for name in ("mcp", "provider"):
            target = public / name
            if not old:
                atomic(target, artifacts[name], 0o755)
        launcher = public / "launcher"
        token = public / "application.token"
        payload = launcher_text(public / "mcp", token, args.provider_url, str(home))
        # Existing bytes were checked against the previous managed digest above.
        atomic(launcher, payload, 0o755)
        bootstrap = folder / "bootstrap.py"
        atomic(bootstrap, Path(__file__).read_bytes(), 0o600)
        rotation = "boxweave-credential-" + user
        service = ("[Unit]\nDescription=Rotate independent BoxWeave application credential\n"
                   "[Service]\nType=oneshot\nUMask=0077\nExecStart=/usr/bin/python3 " +
                   str(bootstrap) + " rotate --state " + str(state_path) + "\n").encode()
        timer = ("[Unit]\nDescription=BoxWeave application credential refresh\n[Timer]\n"
                 "OnCalendar=*-*-* 00,12:00:00\nRandomizedDelaySec=300\nPersistent=true\n"
                 "[Install]\nWantedBy=timers.target\n").encode()
        units = {UNIT_DIR / (rotation + ".service"): service, UNIT_DIR / (rotation + ".timer"): timer}
        for path in units:
            require(old is not None or not path.exists(), "FOREIGN_CREDENTIAL_UNIT")
        state = {"format": 1, "ssh_entry": args.ssh_entry, "user": user, "url": args.provider_url, "port": args.port,
                 "secret": str(safe_path(args.secret_file)), "mcp": str(public / "mcp"),
                 "mcp_sha256": args.mcp_sha256, "provider": str(public / "provider"),
                 "provider_sha256": args.provider_cli_sha256, "token": str(token), "prefix": str(prefix),
                 "managed_files": {str(p): hashlib.sha256(b).hexdigest()
                    for p, b in {**units, launcher: payload, public/"mcp": artifacts["mcp"],
                                 public/"provider": artifacts["provider"]}.items()}}
        atomic(state_path, encoded(state))
        rotate(state_path)
        for path, data in units.items():
            atomic(path, data, 0o644)
        run(["systemctl", "daemon-reload"], deadline)
        run(["systemctl", "enable", "--now", rotation + ".timer"], deadline)
        run(["loginctl", "enable-linger", user], deadline)
        run(["systemctl", "start", "user@" + str(account.pw_uid) + ".service"], deadline)
        runtime = "/run/user/" + str(account.pw_uid)
        require(Path(runtime, "bus").exists(), "APPLICATION_USER_BUS_MISSING")
        stage = home / ".boxweave-bootstrap"
        if not stage.exists():
            stage.mkdir(mode=0o700)
            os.chown(stage, *owner)
        require(stage.stat().st_uid == account.pw_uid and not stage.stat().st_mode & 0o077,
                "APPLICATION_STAGE_UNSAFE")
        for name in ("package", "distribution"):
            atomic(stage / name, artifacts[name], 0o600, owner)
        config = {"mode": "mcp", "database": str(home / "boxweave-state/state.sqlite3"),
                  "provider": "containarium", "backend": platform.node(), "project": "default",
                  "launcher": [str(launcher)], "targets": {}, "host": "127.0.0.1", "port": args.port,
                  "origin": "http://127.0.0.1:" + str(args.port), "secure_cookie": False,
                  "operation_timeout": 60}
        config_path = stage / "application.json"
        argv = ["runuser", "-u", user, "--", "env", "XDG_RUNTIME_DIR=" + runtime,
                "/usr/bin/python3", str(stage / "distribution"), "install", "--package",
                str(stage / "package"), "--sha256", args.sha256, "--prefix", str(prefix), "--activate"]
        if not (prefix / "config.json").exists():
            atomic(config_path, encoded(config), 0o600, owner)
            argv += ["--config", str(config_path)]
        # App installer output is bounded public state, but keep it private on disk.
        with (folder / "application-install.log").open("wb") as log:
            result = subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=log, stderr=log,
                                    timeout=max(0.01, deadline-time.monotonic()))
        if result.returncode:
            return {"state": "application_setup_incomplete", "exit_code": result.returncode,
                    "log": str(folder / "application-install.log"), "prefix": str(prefix),
                    "next": "Inspect the private log; initialize the administrator locally, then rerun the same command."}
        # A waiting/enabled timer is independently checked, never inferred from enable exit 0.
        observed = subprocess.run(["systemctl", "show", rotation + ".timer",
                                  "-p", "ActiveState", "-p", "SubState", "-p", "UnitFileState"],
                                 capture_output=True, text=True, timeout=10)
        require(observed.returncode == 0 and
                {"ActiveState=active", "SubState=waiting", "UnitFileState=enabled"} <= set(observed.stdout.splitlines()),
                "CREDENTIAL_TIMER_NOT_READY")
        return {"state": "ready", "prefix": str(prefix), "application_user": user,
                "credential_scopes": SCOPES.split(","), "credential_timer": "enabled_waiting",
                "provider_installation": "separately_prepared", "public_exposure": False,
                "reboot": "not_verified"}
    finally:
        os.close(lock)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    for name in ("plan", "apply"):
        p = sub.add_parser(name)
        p.add_argument("--app-user", required=True)
        p.add_argument("--archive", "--package", dest="package", required=True)
        for flag in ("sha256", "distribution", "distribution-sha256",
                     "mcp-binary", "mcp-sha256", "provider-cli", "provider-cli-sha256",
                     "secret-file", "provider-url"):
            p.add_argument("--" + flag, required=True)
        p.add_argument("--port", type=int, default=18640)
        p.add_argument("--ack-root-changes")
        p.add_argument("--ssh-entry", action="store_true", help="Create the new app account with a shell for explicitly enrolled forced-command SSH keys")
    p = sub.add_parser("rotate")
    p.add_argument("--state", required=True)
    args = parser.parse_args(argv)
    try:
        if args.action == "rotate":
            result = rotate(args.state)
        else:
            artifacts = prepare(args)
            result = ({"state": "plan_checked", "changes": ["dedicated app account/user bus",
                       "independent scoped launcher/rotating credential", "private app install"],
                       "provider": "must already be reviewed and installed", "writes": False}
                      if args.action == "plan" else apply(args, artifacts))
        print(json.dumps(result, sort_keys=True))
        return 0 if result["state"] != "application_setup_incomplete" else 2
    except (Refused, OSError, ValueError, KeyError, subprocess.SubprocessError):
        # Never include subprocess output, secret bytes, arbitrary paths or token exceptions.
        error = sys.exc_info()[1]
        print(json.dumps({"state": "failed", "code": str(error) if isinstance(error, Refused)
                          else "BOOTSTRAP_OR_RUNTIME_FAILED"}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
