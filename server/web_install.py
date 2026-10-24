#!/usr/bin/env python3
"""Mandatory web preparation and publication, separate from MCP service lifecycle."""
import argparse
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import pwd
import shutil
import socket
import subprocess
import tempfile
import time
from urllib.request import urlopen, Request

ROOT = Path("/etc/boxweave")
RELEASES = Path("/opt/boxweave-web/releases")


def run(args, **kw):
    subprocess.run(args, check=True, timeout=kw.pop("timeout", 120), **kw)


def atomic(path, data, mode=0o644):
    path = Path(path)
    if path.is_symlink() or (path.exists() and not path.is_file()):
        raise RuntimeError("Invalid managed file: " + str(path))
    fd, tmp = tempfile.mkstemp(prefix="." + path.name, dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(tmp, mode)
        os.replace(tmp, path)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def settings(mode=None, host=None, port=None, backend=None, old=None):
    d = dict(old or {})
    d["mode"] = mode or d.get("mode", "https")
    d["host"] = host or d.get("host") or detect_host()
    d["port"] = port if port is not None else d.get("port", 18443)
    d["backend_port"] = backend if backend is not None else d.get("backend_port", 18444)
    address = ipaddress.ip_address(d["host"])
    if (
        address.version != 4
        or not address.is_private
        or address.is_unspecified
        or address.is_link_local
    ):
        raise ValueError("Web host must be an explicit private IPv4 address")
    if d["mode"] not in ("http", "https"):
        raise ValueError("Invalid web mode")
    if (
        any(
            type(d[k]) != int or not 1024 <= d[k] <= 65535
            for k in ("port", "backend_port")
        )
        or d["port"] == d["backend_port"]
    ):
        raise ValueError("Invalid or conflicting web ports")
    d["origin"] = f"{d['mode']}://{d['host']}:{d['port']}"
    return d


def detect_host():
    data = json.loads(
        subprocess.check_output(
            ["ip", "-j", "-4", "addr", "show", "scope", "global"], text=True, timeout=5
        )
    )
    addresses = [
        a["local"]
        for i in data
        if not i.get("ifname", "").startswith(
            ("lo", "incus", "lxc", "docker", "podman", "veth", "br-")
        )
        for a in i.get("addr_info", [])
        if ipaddress.ip_address(a["local"]).is_private
    ]
    if len(set(addresses)) != 1:
        raise ValueError("LAN address is ambiguous; specify --web-host")
    return addresses[0]


def units(d):
    release = d["release"]
    service = f"""[Unit]
Description=BoxWeave independent web backend
After=network.target boxweave-web-token.service
[Service]
User=boxweave-web
Group=boxweave-web
WorkingDirectory=/var/lib/boxweave-web
Environment=PYTHONPATH={release}/server
ExecStart={release}/venv/bin/python -m webui.command serve
Restart=on-failure
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/boxweave-web
ReadOnlyPaths=/var/lib/boxweave-web-auth
InaccessiblePaths=-/etc/containarium -/etc/boxweave/config.json
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
[Install]
WantedBy=multi-user.target
"""
    caddy = f"""[Unit]
Description=BoxWeave dedicated HTTPS entry
After=network.target boxweave-web.service
[Service]
User=boxweave-web-caddy
Group=boxweave-web-caddy
Environment=XDG_DATA_HOME=/var/lib/boxweave-web-caddy
Environment=XDG_CONFIG_HOME=/var/lib/boxweave-web-caddy/config
ExecStart={release}/bin/caddy run --config /etc/boxweave/web.Caddyfile --adapter caddyfile
Restart=on-failure
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/boxweave-web-caddy
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
[Install]
WantedBy=multi-user.target
"""
    token = """[Unit]
Description=Rotate independent BoxWeave web provider credential
After=containarium.service
[Service]
Type=oneshot
User=root
ExecStart=/usr/bin/python3 /usr/local/libexec/boxweave/web_token.py
UMask=0077
TimeoutStartSec=60
"""
    timer = """[Unit]
Description=Hourly independent BoxWeave web credential refresh
[Timer]
OnBootSec=2min
OnUnitInactiveSec=1h
Persistent=true
[Install]
WantedBy=timers.target
"""
    return {
        "boxweave-web.service": service,
        "boxweave-web-caddy.service": caddy,
        "boxweave-web-token.service": token,
        "boxweave-web-token.timer": timer,
    }


def caddyfile(d):
    tls = "tls internal" if d["mode"] == "https" else ""
    return f"""{{
 admin off
 auto_https disable_redirects
 skip_install_trust
}}
{d['origin']} {{
 bind {d['host']}
 {tls}
 reverse_proxy 127.0.0.1:{d['backend_port']} {{
  header_up Host {d['host']}:{d['port']}
  header_up -X-Forwarded-For
  header_up -X-Forwarded-Host
  header_up -X-Forwarded-Proto
 }}
}}
"""


def prepare(source, d):
    # Package operations are limited to missing web prerequisites, never provider upgrades.
    missing = []
    for binary, package in [("node", "nodejs"), ("npm", "npm")]:
        if not shutil.which(binary):
            missing.append(package)
    check = subprocess.run(
        ["/usr/bin/python3", "-c", "import venv,ensurepip"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    if check.returncode:
        missing.append("python3-venv")
    if missing:
        run(["apt-get", "update"], timeout=180)
        run(
            ["apt-get", "install", "-y", "--no-install-recommends", *missing],
            timeout=300,
        )
    # Extract only a signature/hash-verified distro binary; never activate or edit distro Caddy.
    run(["apt-get", "download", "caddy"], cwd=source, timeout=180)
    packages = list(source.glob("caddy_*.deb"))
    if len(packages) != 1:
        raise RuntimeError("Ambiguous Caddy package")
    run(["dpkg-deb", "--extract", str(packages[0]), str(source / "caddy-package")])
    old = (
        json.loads((ROOT / "web.json").read_text())
        if (ROOT / "web.json").exists()
        else None
    )
    for key in ("port", "backend_port"):
        if old and d[key] == old[key] and d["host"] == old["host"]:
            # Only reuse listeners belonging to the exact managed units.
            unit = (
                "boxweave-web-caddy.service"
                if key == "port"
                else "boxweave-web.service"
            )
            active = (
                subprocess.run(["systemctl", "is-active", "--quiet", unit]).returncode
                == 0
            )
            if active:
                continue
        s = socket.socket()
        try:
            s.bind((d["host"] if key == "port" else "127.0.0.1", d[key]))
        finally:
            s.close()
    run(
        ["npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"],
        cwd=source / "web",
        timeout=240,
    )
    run(["npm", "run", "build"], cwd=source / "web", timeout=180)
    run(["/usr/bin/python3", "-m", "venv", str(source / "web-venv")])
    run(
        [
            str(source / "web-venv/bin/python"),
            "-m",
            "pip",
            "install",
            "--require-hashes",
            "-r",
            str(source / "server/webui/requirements.lock"),
        ],
        timeout=240,
    )
    run([str(source / "web-venv/bin/python"), "-c", "import flask,argon2,waitress"])
    # Fingerprint the exact source and build; a release is never overwritten.
    h = hashlib.sha256()
    for p in sorted(
        [
            *(source / "web/dist").rglob("*"),
            *(source / "server/webui").glob("*.py"),
            source / "server/webui/requirements.lock",
            source / "caddy-package/usr/bin/caddy",
        ]
    ):
        if p.is_file():
            h.update(p.relative_to(source).as_posix().encode())
            h.update(p.read_bytes())
    d["release"] = str(RELEASES / h.hexdigest()[:24])
    d["source_hash"] = h.hexdigest()
    atomic(source / "web.Caddyfile", caddyfile(d).encode())
    run(
        [
            str(source / "caddy-package/usr/bin/caddy"),
            "validate",
            "--config",
            str(source / "web.Caddyfile"),
            "--adapter",
            "caddyfile",
        ],
        env={
            **os.environ,
            "XDG_DATA_HOME": str(source / "caddy-validate-data"),
            "XDG_CONFIG_HOME": str(source / "caddy-validate-config"),
        },
        stdout=subprocess.DEVNULL,
    )
    (source / "web-prepared.json").write_text(json.dumps(d))


def publish(source):
    d = json.loads((source / "web-prepared.json").read_text())
    cfg = json.loads((ROOT / "config.json").read_text())
    key = Path(cfg["container_public_key_file"]).read_text().strip()
    if not key.startswith(("ssh-ed25519 ", "ssh-rsa ")) or "\n" in key:
        raise ValueError("Invalid managed public key")
    for username in ("boxweave-web", "boxweave-web-caddy"):
        try:
            pwd.getpwnam(username)
        except KeyError:
            run(
                [
                    "useradd",
                    "--system",
                    "--home-dir",
                    "/nonexistent",
                    "--shell",
                    "/usr/sbin/nologin",
                    username,
                ]
            )
    for directory, user, mode in [
        ("/var/lib/boxweave-web", "boxweave-web", 0o700),
        ("/var/lib/boxweave-web-auth", "boxweave-web", 0o750),
        ("/var/lib/boxweave-web-caddy", "boxweave-web-caddy", 0o700),
    ]:
        p = Path(directory)
        if p.is_symlink():
            raise ValueError("Refusing data-directory symlink")
        p.mkdir(exist_ok=True)
        account = pwd.getpwnam(user)
        os.chown(
            p, 0 if directory.endswith("-auth") else account.pw_uid, account.pw_gid
        )
        p.chmod(mode)
    RELEASES.mkdir(parents=True, exist_ok=True)
    for directory in (RELEASES, RELEASES.parent):
        directory.chmod(0o755)
    release = Path(d["release"])
    if release.is_symlink():
        raise ValueError("Invalid release directory")
    if not release.exists():
        temp = Path(tempfile.mkdtemp(prefix=".stage-", dir=RELEASES))
        shutil.copytree(source / "web-venv", temp / "venv", symlinks=True)
        (temp / "bin").mkdir()
        shutil.copy2(source / "caddy-package/usr/bin/caddy", temp / "bin/caddy")
        shutil.copytree(source / "web/dist", temp / "static")
        shutil.copytree(
            source / "server/webui",
            temp / "server/webui",
            ignore=shutil.ignore_patterns("__pycache__"),
        )
        for p in temp.rglob("*"):
            if not p.is_symlink():
                p.chmod(0o755 if p.is_dir() or p.stat().st_mode & 0o111 else 0o644)
        temp.chmod(0o755)
        temp.rename(release)
    d.update(
        database="/var/lib/boxweave-web/graphs.sqlite3",
        static=str(release / "static"),
        api_url=cfg["api_url"],
        token_file="/var/lib/boxweave-web-auth/provider.jwt",
        public_key=key,
    )
    # Refuse pre-existing unrelated units/masks and stop only our independent web process.
    for unit in units(d):
        unit_path = Path("/etc/systemd/system") / unit
        if unit_path.is_symlink():
            raise ValueError("Refusing masked/symlinked web unit")
        if unit_path.exists() and "BoxWeave" not in unit_path.read_text():
            raise ValueError("Unmanaged web unit conflict")
    backup_dir = Path(
        tempfile.mkdtemp(prefix="web-update-", dir="/var/backups/boxweave")
    )
    backup_dir.chmod(0o700)
    if (ROOT / "web.json").exists():
        shutil.copy2(ROOT / "web.json", backup_dir / "web.json")
    for unit in ("boxweave-web.service", "boxweave-web-caddy.service"):
        if subprocess.run(["systemctl", "is-active", "--quiet", unit]).returncode == 0:
            run(["systemctl", "stop", unit])
    # Database backup is consistent and private; migrations run as the service user.
    database = Path(d["database"])
    if database.is_symlink():
        raise ValueError("Refusing database symlink")
    if database.exists():
        import sqlite3

        backup = database.with_name(
            "graphs.pre-migration-" + str(time.time_ns()) + ".sqlite3"
        )
        with sqlite3.connect(database) as src, sqlite3.connect(backup) as dst:
            src.backup(dst)
        account = pwd.getpwnam("boxweave-web")
        os.chown(backup, account.pw_uid, account.pw_gid)
        backup.chmod(0o600)
    atomic(ROOT / "web.json", json.dumps(d, indent=2).encode())
    run(
        [
            "runuser",
            "-u",
            "boxweave-web",
            "--",
            "env",
            "PYTHONPATH=" + str(release / "server"),
            str(release / "venv/bin/python"),
            "-c",
            f'from webui.store import Store; Store({d["database"]!r})',
        ]
    )
    cf = caddyfile(d)
    atomic(source / "web.Caddyfile", cf.encode())
    units_data = units(d)
    targets = {
        ROOT / "web.Caddyfile": cf.encode(),
        **{Path("/etc/systemd/system") / k: v.encode() for k, v in units_data.items()},
    }
    for path in targets:
        if path.is_symlink():
            raise ValueError("Refusing masked or symlinked managed file")
        if path.exists():
            shutil.copy2(path, backup_dir / path.name)
    for path, data in targets.items():
        atomic(path, data)
    for name in ("web_token.py", "web_entry.py"):
        target = Path("/usr/local/libexec/boxweave") / name
        if target.exists():
            shutil.copy2(target, backup_dir / name)
        atomic(target, (source / "server" / name).read_bytes())
    run(["systemctl", "daemon-reload"])
    run(["systemctl", "start", "boxweave-web-token.service"])
    run(
        [
            "systemctl",
            "enable",
            "boxweave-web.service",
            "boxweave-web-caddy.service",
            "boxweave-web-token.timer",
        ]
    )
    run(
        [
            "systemctl",
            "restart",
            "boxweave-web.service",
            "boxweave-web-caddy.service",
            "boxweave-web-token.timer",
        ]
    )
    deadline = time.monotonic() + 30
    while True:
        try:
            with urlopen(
                Request(
                    f'http://127.0.0.1:{d["backend_port"]}/healthz',
                    headers={"Host": d["host"] + ":" + str(d["port"])},
                ),
                timeout=2,
            ) as response:
                if response.status == 200:
                    break
        except OSError:
            if time.monotonic() > deadline:
                raise RuntimeError(
                    "Web backend health failed; MCP timers remain independent"
                )
            time.sleep(1)
    run(
        [
            "systemctl",
            "is-active",
            "--quiet",
            "boxweave-web.service",
            "boxweave-web-caddy.service",
            "boxweave-web-token.timer",
        ]
    )
    manifest_path = ROOT / "install-manifest.json"
    manifest = json.loads(manifest_path.read_text())
    manifest["web"] = {
        "release": d["release"],
        "sha256": d["source_hash"],
        "mode": d["mode"],
        "origin": d["origin"],
        "units": list(units_data),
        "database_schema": 1,
    }
    manifest["component_sha256"].update(
        {
            str(p.relative_to(source)): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in [
                *source.glob("server/web*.py"),
                *source.glob("server/webui/*.py"),
                source / "web/package-lock.json",
                source / "server/webui/requirements.lock",
                source / "caddy-package/usr/bin/caddy",
            ]
        }
    )
    atomic(manifest_path, json.dumps(manifest, indent=2).encode(), 0o600)
    print("Web console: " + d["origin"])
    print("Mode: " + d["mode"] + "; port: " + str(d["port"]))
    print("Initialize administrator: sudo boxweave web init-admin")
    print("Diagnostics: sudo boxweave web status; sudo boxweave doctor")
    print("CA trust guide: docs/WEB-UI.md (preserve /var/lib/boxweave-web-caddy)")
    if d["mode"] == "http":
        print("WARNING: HTTP credentials and sessions are not encrypted.")


def main():
    p = argparse.ArgumentParser()
    p.add_argument("phase", choices=["prepare", "publish"])
    p.add_argument("source", type=Path)
    for k in ("mode", "host", "port", "backend-port"):
        p.add_argument("--" + k, type=int if "port" in k else str)
    a = p.parse_args()
    if os.geteuid() != 0:
        raise SystemExit("Requires root")
    if a.phase == "prepare":
        old = (
            json.loads((ROOT / "web.json").read_text())
            if (ROOT / "web.json").exists()
            else {}
        )
        prepare(a.source, settings(a.mode, a.host, a.port, a.backend_port, old))
    else:
        publish(a.source)


if __name__ == "__main__":
    main()
