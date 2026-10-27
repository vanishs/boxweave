"""Interactive local administrator setup. Passwords never accepted as arguments."""

import argparse
import getpass
import json
import os
import sys
from pathlib import Path
from .store import Store


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument(
        "action", choices=["init-admin", "reset-password", "status", "serve"]
    )
    p.add_argument("--config", default="/etc/boxweave/web.json")
    a = p.parse_args(argv)
    cfg = json.loads(Path(a.config).read_text())
    if a.action == "serve":
        from .app import create_app
        from waitress import serve

        serve(
            create_app(cfg),
            host="127.0.0.1",
            port=cfg["backend_port"],
            threads=8,
            clear_untrusted_proxy_headers=True,
        )
        return 0
    if a.action == "status":
        # Read-only status never opens/migrates the database or provider token.
        import socket

        from .tenants import CONTRACT
        from .store import SCHEMA

        result = {k: cfg[k] for k in ("origin", "backend_port", "database")}
        result["tenant_capabilities"] = {
            "supported_contract": CONTRACT,
            "shared_creation_adapter": True,
            "atomic_tenant_access_adapter": True,
            "provider_state": "unprobed; use authenticated /api/tenants and /api/tenant-access",
            "reassignment": False,
        }
        result["supported_database_schema"] = SCHEMA
        try:
            with socket.create_connection(("127.0.0.1", cfg["backend_port"]), 2):
                result["listener"] = "reachable"
        except OSError:
            result["listener"] = "unreachable"
        print(json.dumps(result, indent=2))
        return 0 if result["listener"] == "reachable" else 2
    if not sys.stdin.isatty():
        p.error("Administrator password requires an interactive local terminal")
    if os.geteuid() != 0 and os.geteuid() != Path(cfg["database"]).parent.stat().st_uid:
        p.error("Run as the web service user or root")
    from .app import HASHER

    store = Store(cfg["database"])
    with store.db() as db:
        exists = db.execute("SELECT 1 FROM admin").fetchone()
        if a.action == "init-admin" and exists:
            p.error("Administrator already initialized; use reset-password")
        if a.action == "reset-password" and not exists:
            p.error("Initialize administrator first")
    password = getpass.getpass("New administrator password (at least 8 characters): ")
    if len(password) < 8 or len(password) > 1024:
        p.error("Use 8–1024 characters")
    if password != getpass.getpass("Repeat password: "):
        p.error("Passwords do not match")
    with store.db() as db:
        db.execute(
            "INSERT OR REPLACE INTO admin VALUES (1,?)", (HASHER.hash(password),)
        )
        db.execute("DELETE FROM sessions")
    print("Administrator credential updated; all sessions invalidated.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
