"""Explicit run modes. Fixtures cannot be an implicit live fallback."""
import argparse
import getpass
import json
import os
from pathlib import Path
from werkzeug.security import generate_password_hash
from .domain import Target, Fault
from .provider import Fixture, Provider
from .service import Service
from .store import Store

def load(path):
    p=Path(path)
    if p.is_symlink() or p.stat().st_size>65536:
        raise Fault("INVALID_CONFIG",422)
    c=json.loads(p.read_text())
    mode=c.get("mode")
    if mode=="fixture":
        provider=Fixture(Target(**c["target"]),c.get("fixture_counter"))
    elif mode=="mcp":
        provider=Provider(c)
    else:
        raise Fault("EXPLICIT_MODE_REQUIRED",422)
    store=Store(c["database"])
    return c,Service(store,provider,budget=c.get("operation_timeout",60))

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("action",choices=["init-admin","serve","worker","projects","receipts","diagnose"])
    p.add_argument("--config",required=True)
    p.add_argument("--once",action="store_true")
    a=p.parse_args()
    try:
        c,s=load(a.config)
        if a.action=="init-admin":
            password=getpass.getpass("New local administrator password: ")
            if not 12<=len(password)<=1024:
                raise Fault("PASSWORD_LENGTH",422)
            if password!=getpass.getpass("Repeat password: "):
                raise Fault("PASSWORD_MISMATCH",422)
            s.store.admin(generate_password_hash(password))
            print("Administrator initialized; no password recorded.")
        elif a.action=="worker":
            s.worker(a.once)
        elif a.action=="serve":
            from waitress import serve
            from .web import create_app
            host=c.get("host","127.0.0.1")
            # Private-address development HTTP must be deliberately opted in.
            if host!="127.0.0.1" and not c.get("allow_private_http"):
                raise Fault("LISTENER_NOT_APPROVED",422)
            port=c.get("port",18640)
            serve(create_app(s,c["origin"],Path(c["dist"]).absolute(),secure=c.get("secure_cookie",True)),
                  host=host,port=port,threads=4,max_request_body_size=16384,
                  clear_untrusted_proxy_headers=True)
        elif a.action=="projects":
            print(json.dumps(s.projects(),indent=2))
        elif a.action=="receipts":
            print(json.dumps(s.list_receipts(),indent=2))
        elif a.action=="diagnose":
            if s.provider.mode!="mcp":
                raise Fault("LIVE_DIAGNOSTIC_REQUIRES_MCP",422)
            print(json.dumps({"mode":"mcp","targets":[s.provider.observe(x).data() for x in c["targets"]]}))
        return 0
    except (Fault,OSError,ValueError,KeyError) as error:
        print(json.dumps({"code":getattr(error,"code","CONFIG_OR_RUNTIME_UNAVAILABLE")}))
        return 2

if __name__=="__main__":
    raise SystemExit(main())
