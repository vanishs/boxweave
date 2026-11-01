"""Optional reviewed HTTPS ingress via the operator-installed Caddy executable.

No package installation, firewall modification, DNS mutation, reverse-proxy
adoption or automatic Caddy launch takes place during application install.
"""
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
from urllib.parse import urlsplit

from .remote_mcp import validate_remote_config

CADDY = "/usr/bin/caddy"


def caddyfile(remote, caddy=CADDY):
    remote = validate_remote_config(remote)
    host = urlsplit(remote["public_url"]).hostname
    if not re.fullmatch(r"[a-zA-Z0-9.-]{1,253}", host) or "." not in host:
        raise ValueError("TLS_HOST_INVALID")
    if not Path(caddy).is_file() or not os.access(caddy, os.X_OK):
        raise ValueError("CADDY_NOT_INSTALLED")
    # The server only binds loopback. HTTPS is a separately managed process.
    # Do not override Referrer-Policy here: the authorized OAuth form must
    # retain same-origin, while other paths use the application's no-referrer.
    # Rewriting all responses to no-referrer breaks browser OAuth consent.
    return (host + " {\n"
            "  encode gzip\n"
            "  reverse_proxy 127.0.0.1:" + str(remote["port"]) + " {\n"
            "    flush_interval -1\n"
            "  }\n"
            "  log {\n    output discard\n  }\n"
            "}\n")


def run_caddy(remote, caddy=CADDY):
    config = caddyfile(remote, caddy)
    # Validate before a long-lived process. No unrelated system config touched.
    with tempfile.TemporaryDirectory(prefix="boxweave-caddy-") as folder:
        cfg = Path(folder) / "Caddyfile"
        cfg.write_text(config)
        cfg.chmod(0o600)
        subprocess.run([caddy, "validate", "--config", str(cfg),
                        "--adapter", "caddyfile"], check=True, timeout=20)
        return subprocess.call([caddy, "run", "--config", str(cfg),
                                "--adapter", "caddyfile"])
