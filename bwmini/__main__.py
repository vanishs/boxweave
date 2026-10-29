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
    p.add_argument("action",choices=["init-admin","serve","worker","projects","receipts","history","reconcile","environments","environment-plan","environment-confirm","environment-receipts","environment-receipt","environment-observe","diagnose"])
    p.add_argument("--config",required=True)
    p.add_argument("--once",action="store_true")
    p.add_argument("--limit", type=int, default=20)
    p.add_argument("--cursor")
    p.add_argument("--operation")
    p.add_argument("--logical")
    p.add_argument("--environment-action",choices=["create","start","stop","delete"])
    p.add_argument("--plan")
    a=p.parse_args()
    if a.action=="environment-plan" and (a.logical is None or a.environment_action is None):
        p.error("environment-plan requires --logical and --environment-action")
    if a.action=="environment-confirm" and (a.logical is None or a.plan is None):
        p.error("environment-confirm requires --logical and --plan")
    if a.logical is not None and a.action not in ("environment-plan","environment-confirm"):
        p.error("--logical is only supported for environment plans/confirmation")
    if a.environment_action is not None and a.action!="environment-plan":
        p.error("--environment-action is only supported for environment-plan")
    if a.plan is not None and a.action!="environment-confirm":
        p.error("--plan is only supported for environment-confirm")
    if (a.action in ("reconcile","environment-receipt","environment-observe")) != (a.operation is not None):
        p.error("reconcile requires --operation; other actions do not accept it")
    if a.action != "history" and (a.cursor is not None or a.limit != 20):
        p.error("--limit and --cursor are supported only for history")
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
        elif a.action=="history":
            print(json.dumps(s.receipt_history(limit=a.limit, cursor=a.cursor),indent=2))
        elif a.action=="reconcile":
            print(json.dumps(s.reconcile(a.operation),indent=2))
        elif a.action=="environments":
            print(json.dumps(s.lifecycle.settings(),indent=2))
        elif a.action=="environment-plan":
            print(json.dumps(s.lifecycle.plan(a.environment_action,a.logical),indent=2))
        elif a.action=="environment-confirm":
            print(json.dumps(s.lifecycle.confirm(a.plan,a.logical),indent=2))
        elif a.action=="environment-receipts":
            print(json.dumps(s.lifecycle.receipts(),indent=2))
        elif a.action=="environment-receipt":
            print(json.dumps(s.lifecycle.receipt(a.operation),indent=2))
        elif a.action=="environment-observe":
            print(json.dumps(s.lifecycle.observe(a.operation),indent=2))
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
