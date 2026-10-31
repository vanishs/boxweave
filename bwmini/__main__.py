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

def read_config(path):
    p=Path(path)
    if p.is_symlink() or p.stat().st_size>65536:
        raise Fault("INVALID_CONFIG",422)
    c=json.loads(p.read_text())
    return c

def load(path):
    c=read_config(path)
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
    p.add_argument("action",choices=["init-admin","reset-admin","serve","worker","projects","receipts","history","reconcile","environments","environment-plan","environment-confirm","environment-receipts","environment-receipt","environment-observe","model-wizard","model-select","model-confirm","model-handoff","access","access-memberships","access-network-plan","access-port-plan","access-confirm","access-receipts","access-receipt","access-observe","tenant-label","diagnose","job-run","job-receipts","selftest-plan","selftest-run"])
    p.add_argument("--config",required=True)
    p.add_argument("--once",action="store_true")
    p.add_argument("--limit", type=int, default=20)
    p.add_argument("--cursor")
    p.add_argument("--operation")
    p.add_argument("--logical")
    p.add_argument("--environment-action",choices=["create","start","stop","delete","resize"])
    p.add_argument("--plan")
    p.add_argument("--model-provider")
    p.add_argument("--model-id")
    p.add_argument("--access-mode")
    p.add_argument("--client-version")
    p.add_argument("--revision",type=int)
    p.add_argument("--native-selected",action="store_true")
    p.add_argument("--native-access-checked",action="store_true")
    p.add_argument("--network-tenant")
    p.add_argument("--source-tenant")
    p.add_argument("--destination-tenant")
    p.add_argument("--network-action",choices=["grant","revoke"])
    p.add_argument("--both-directions",action="store_true")
    p.add_argument("--port-profile")
    p.add_argument("--port-action",choices=["publish","unpublish"])
    p.add_argument("--access-plan")
    p.add_argument("--confirmation")
    p.add_argument("--access-operation")
    p.add_argument("--tenant")
    p.add_argument("--label")
    p.add_argument("--label-revision",type=int)
    p.add_argument("--argv")
    p.add_argument("--budget",type=int,default=60)
    p.add_argument("--cpu",choices=["1","2","3","4","5","6","7","8"])
    p.add_argument("--memory",choices=["1GB","2GB","4GB","8GB"])
    p.add_argument("--disk",choices=["8GB","10GB","20GB","25GB","30GB","40GB","50GB","100GB"])
    a=p.parse_args()
    groups={"access-network-plan":[a.source_tenant,a.destination_tenant,a.network_action],"access-port-plan":[a.port_profile,a.port_action],"access-confirm":[a.access_plan,a.confirmation],"tenant-label":[a.tenant,a.label,a.label_revision]}
    for action,values in groups.items():
        if a.action==action and any(v is None for v in values):p.error(action+" requires its complete action-specific arguments")
        if a.action not in (action,"selftest-run") and any(v is not None for v in values):p.error(action+" arguments are not supported by this action")
    if a.action=="selftest-run" and not (isinstance(a.plan,str) and isinstance(a.confirmation,str)):
        p.error("selftest-run requires --plan and --confirmation")
    if (a.both_directions and a.action!="access-network-plan"):p.error("--both-directions requires access-network-plan")
    if any(v is not None for v in (a.cpu,a.memory,a.disk)) and a.action!="environment-plan":
        p.error("resource choices are only supported by environment-plan")
    if (a.action in ("access-receipt","access-observe"))!=(a.access_operation is not None):p.error("access receipt/observation requires --access-operation")
    if a.network_tenant is not None and (a.action!="environment-plan" or a.environment_action!="create"):p.error("--network-tenant requires environment-plan --environment-action create")
    model_values=[a.model_provider,a.model_id,a.access_mode,a.client_version]
    if a.action=="model-select" and (any(v is None for v in model_values) or a.revision is None):p.error("model-select requires version/provider/model/access mode and revision")
    if a.action!="model-select" and any(v is not None for v in model_values):p.error("model selection flags require model-select")
    if a.action=="model-confirm" and (a.revision is None or not a.native_selected or not a.native_access_checked):p.error("model-confirm requires revision and both explicit native check flags")
    if a.action not in ("model-select","model-confirm") and a.revision is not None:p.error("revision is only supported for model mutation")
    if a.action!="model-confirm" and (a.native_selected or a.native_access_checked):p.error("native check flags require model-confirm")
    if a.action=="environment-plan" and (a.logical is None or a.environment_action is None):
        p.error("environment-plan requires --logical and --environment-action")
    if a.action=="environment-confirm" and (a.logical is None or a.plan is None):
        p.error("environment-confirm requires --logical and --plan")
    if a.logical is not None and a.action not in ("environment-plan","environment-confirm"):
        p.error("--logical is only supported for environment plans/confirmation")
    if a.environment_action is not None and a.action!="environment-plan":
        p.error("--environment-action is only supported for environment-plan")
    if a.plan is not None and a.action not in ("environment-confirm","selftest-run"):
        p.error("--plan is only supported for environment-confirm and selftest-run")
    if (a.action in ("reconcile","environment-receipt","environment-observe")) != (a.operation is not None):
        p.error("reconcile requires --operation; other actions do not accept it")
    if a.action != "history" and (a.cursor is not None or a.limit != 20):
        p.error("--limit and --cursor are supported only for history")
    try:
        if a.action=="diagnose":
            c=read_config(a.config)
            if c.get("mode")!="mcp":raise Fault("LIVE_DIAGNOSTIC_REQUIRES_MCP",422)
            from .deploy_check import read_provider
            connection=read_provider(c);provider=Provider(c)
            print(json.dumps({"mode":"mcp","connection":connection,"targets":[provider.observe(x).data() for x in c.get("targets",{})]}))
            return 0
        if a.action in ("job-run","job-receipts"):
            from . import jobs
            c=read_config(a.config);store=Store(c["database"])
            if a.action=="job-run":
                try:
                    argv=json.loads(a.argv) if isinstance(a.argv,str) else None
                except ValueError:
                    argv=None
                if not isinstance(argv,list):
                    raise Fault("JOB_ARGV_INVALID",422)
                print(json.dumps(jobs.run(store,argv,budget=a.budget),indent=2,sort_keys=True))
            else:
                print(json.dumps(jobs.receipts(store),indent=2,sort_keys=True))
            return 0
        if a.action=="selftest-plan":
            from .selftest import build_plan
            plan,checksum=build_plan()
            print(json.dumps({"plan":plan,"confirmation":checksum,
                  "next":"Save the plan object to a file, review it, then run selftest-run with --plan FILE --confirmation HEX"} ,indent=2,sort_keys=True))
            return 0
        if a.action=="selftest-run":
            from .selftest import load_plan,run as run_selftest
            if not isinstance(a.plan,str) or not isinstance(a.confirmation,str):
                raise Fault("SELFTEST_PLAN_REQUIRED",422)
            try:
                raw=json.loads(Path(a.plan).read_text(encoding="utf-8"))
            except (OSError,ValueError):
                raise Fault("SELFTEST_PLAN_INVALID",422) from None
            plan=load_plan(raw,a.confirmation)
            c,s=load(a.config)
            result=run_selftest(s.provider,plan,budget=a.budget)
            print(json.dumps(result,indent=2,sort_keys=True))
            return 0 if result["state"]=="passed" else 2
        if a.action.startswith("model-") or a.action in ("init-admin","reset-admin"):
            c=read_config(a.config);model_store=Store(c["database"])
        else:
            c,s=load(a.config)
        if a.action in ("init-admin","reset-admin"):
            if a.action=="reset-admin" and input("Type RESET to replace the local administrator and revoke browser sessions: ")!="RESET":
                raise Fault("RESET_DECLINED",422)
            password=getpass.getpass("New local administrator password: ")
            if not 12<=len(password)<=1024:
                raise Fault("PASSWORD_LENGTH",422)
            if password!=getpass.getpass("Repeat password: "):
                raise Fault("PASSWORD_MISMATCH",422)
            (model_store.reset_admin if a.action=="reset-admin" else model_store.admin)(generate_password_hash(password))
            print("Administrator saved; browser sessions revoked on reset; no password recorded.")
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
            resize={k:v for k,v in (("cpu",a.cpu),("memory",a.memory),("disk",a.disk)) if v is not None} or None
            print(json.dumps(s.lifecycle.plan(a.environment_action,a.logical,network_tenant=a.network_tenant,resize=resize),indent=2))
        elif a.action=="environment-confirm":
            print(json.dumps(s.lifecycle.confirm(a.plan,a.logical),indent=2))
        elif a.action=="environment-receipts":
            print(json.dumps(s.lifecycle.receipts(),indent=2))
        elif a.action=="environment-receipt":
            print(json.dumps(s.lifecycle.receipt(a.operation),indent=2))
        elif a.action=="environment-observe":
            print(json.dumps(s.lifecycle.observe(a.operation),indent=2))
        elif a.action=="access":print(json.dumps(s.access.settings(),indent=2))
        elif a.action=="access-memberships":print(json.dumps(s.access.memberships(),indent=2))
        elif a.action=="access-receipts":print(json.dumps(s.access.receipts(),indent=2))
        elif a.action=="access-receipt":print(json.dumps(s.access.receipt(a.access_operation),indent=2))
        elif a.action=="access-observe":print(json.dumps(s.access.observe(a.access_operation),indent=2))
        elif a.action=="access-confirm":print(json.dumps(s.access.confirm(a.access_plan,a.confirmation),indent=2))
        elif a.action=="tenant-label":print(json.dumps(s.access.label(a.tenant,a.label,a.label_revision),indent=2))
        elif a.action=="access-port-plan":print(json.dumps(s.access.port_plan(a.port_profile,a.port_action),indent=2))
        elif a.action=="access-network-plan":
            pairs=[(a.source_tenant,a.destination_tenant)]+([(a.destination_tenant,a.source_tenant)] if a.both_directions else [])
            changes=[{"destination":destination,"add_sources":[source] if a.network_action=="grant" else [],"remove_sources":[source] if a.network_action=="revoke" else []} for source,destination in pairs]
            print(json.dumps(s.access.network_plan(changes),indent=2))
        elif a.action.startswith("model-"):
            from .models import Models
            models=Models(model_store)
            if a.action=="model-wizard":result=models.read()
            elif a.action=="model-handoff":result=models.handoff()
            elif a.action=="model-select":result=models.save({"client_version":a.client_version,"provider":a.model_provider,"model":a.model_id,"access_mode":a.access_mode},a.revision)
            else:result=models.confirm(a.revision,a.native_selected,a.native_access_checked)
            print(json.dumps(result,indent=2))
        return 0
    except (Fault,OSError,ValueError,KeyError) as error:
        print(json.dumps({"code":getattr(error,"code","CONFIG_OR_RUNTIME_UNAVAILABLE")}))
        return 2

if __name__=="__main__":
    raise SystemExit(main())
