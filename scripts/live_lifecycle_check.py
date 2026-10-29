"""Approved exact-name live lifecycle via Chromium/Web/worker and independent MCP."""
import argparse
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import sys
import time
import urllib.request
from playwright.sync_api import sync_playwright
from werkzeug.security import generate_password_hash
from bwmini.__main__ import load

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--config",required=True)
    p.add_argument("--allow-destructive-test",action="store_true")
    a=p.parse_args()
    path=Path(a.config).resolve()
    cfg=json.loads(path.read_text())
    if not a.allow_destructive_test or cfg.get("mode")!="mcp" or cfg.get("host")!="127.0.0.1":
        raise RuntimeError("Explicit isolated live test authorization required")
    if cfg.get("lifecycle",{}).get("allowed_names")!=["v2-lifecycle-e2e"]:
        raise RuntimeError("Only dedicated authorized lifecycle name allowed")
    if Path(cfg["database"]).exists():
        raise RuntimeError("Existing test state preserved; reconcile original name before another run")
    root=Path(__file__).resolve().parents[1]
    c,s=load(path)
    if s.provider.lifecycle_inspect("v2-lifecycle-e2e") is not None:
        raise RuntimeError("Original logical name exists; no create/retry")
    original=s.provider.observe("v2-live-hello")
    password=secrets.token_urlsafe(32)
    s.store.admin(generate_password_hash(password))
    progress=path.with_name("progress.json")
    receipts=[]
    def start(action):
        return subprocess.Popen([sys.executable,"-m","bwmini",action,"--config",str(path)],
            cwd=root,stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
    def stop(process):
        if process and process.poll() is None:
            os.killpg(process.pid,signal.SIGTERM);process.wait(timeout=10)
    def ready():
        deadline=time.monotonic()+30
        while True:
            try:urllib.request.urlopen(c["origin"],timeout=1).close();return
            except OSError:
                if time.monotonic()>deadline:raise
                time.sleep(.2)
    web=worker=None
    try:
        web=start("serve");worker=start("worker");ready()
        with sync_playwright() as pw:
            browser=pw.chromium.launch()
            page=browser.new_page(viewport={"width":1280,"height":960})
            errors=[];page.on("pageerror",lambda e:errors.append(str(e)))
            page.goto(c["origin"])
            page.get_by_label("Password").fill(password)
            page.get_by_role("button",name="Sign in",exact=True).click()
            page.get_by_role("button",name="Environments",exact=True).click()
            page.get_by_label("Authorized new environment").wait_for()
            for action in ("create","stop","start","stop","delete"):
                progress.write_text(json.dumps({"action":action,"status":"previewing","completed":len(receipts)}))
                page.get_by_role("button",name="Preview "+action,exact=True).click()
                field=page.get_by_label("Type the exact environment name")
                field.wait_for(timeout=60000);field.fill("v2-lifecycle-e2e")
                with page.expect_response(lambda r:'/api/environment-plans/' in r.url and r.url.endswith('/confirm') and r.request.method=='POST',timeout=60000) as response:
                    page.get_by_role("button",name="Confirm "+action,exact=True).click()
                queued=response.value.json()
                if response.value.status!=202:raise RuntimeError("Confirmation rejected")
                oid=queued["id"]
                page.locator(".environment-panel .receipt small").filter(has_text=oid).wait_for(timeout=60000)
                deadline=time.monotonic()+c["lifecycle"]["readiness_timeout"]+10
                while True:
                    r=page.request.get(c["origin"]+"/api/environment-operations/"+oid,timeout=60000).json()
                    progress.write_text(json.dumps({"action":action,"status":r["state"],"phase":r["phase"],"code":r["code"],"operation":oid,"completed":len(receipts)}))
                    if r["state"] in ("succeeded","failed","unknown"):break
                    if time.monotonic()>deadline:raise TimeoutError("Readiness phase ended")
                    page.wait_for_timeout(1000)
                receipts.append(r)
                path.with_name("receipts.json").write_text(json.dumps(receipts,indent=2))
                if r["state"]!="succeeded":
                    raise RuntimeError("Lifecycle outcome requires reconciliation: "+str(r["code"]))
                page.get_by_test_id("environment-state").filter(has_text="succeeded").wait_for(timeout=60000)
                if action=="create":
                    page.get_by_role("button",name="Projects",exact=True).click()
                    page.get_by_label("Reviewed environment").select_option("v2-lifecycle-e2e",timeout=60000)
                    page.get_by_label("Project name").fill("Lifecycle-bound project")
                    page.get_by_role("button",name="Save binding",exact=True).click()
                    page.get_by_role("button",name="Preview hello",exact=True).first.click()
                    page.get_by_role("button",name="Confirm hello",exact=True).click()
                    page.get_by_test_id("operation-state").filter(has_text="succeeded").wait_for(timeout=120000)
                    page.get_by_text("evidence: live_mcp",exact=False).wait_for()
                    page.get_by_role("button",name="Environments",exact=True).click()
                    page.get_by_label("Authorized new environment").wait_for()
            assert s.provider.lifecycle_inspect("v2-lifecycle-e2e") is None
            assert s.provider.observe("v2-live-hello")==original
            with s.store.db() as db:
                before=db.execute("SELECT count(*) FROM environment_operations").fetchone()[0]
                hello_count=db.execute("SELECT count(*) FROM audit").fetchone()[0]
            assert before==5 and hello_count==1
            stop(worker);worker=start("worker");stop(web);web=start("serve");ready()
            page.reload()
            page.get_by_role("button",name="Environments",exact=True).click()
            history=page.get_by_label("Environment receipts",exact=True)
            history.get_by_role("button",name="Open operation",exact=False).first.wait_for(timeout=60000)
            assert history.get_by_role("button",name="Open operation",exact=False).count()==5
            page.get_by_role("button",name="Projects",exact=True).click()
            page.get_by_role("button",name="Preview hello",exact=True).first.click()
            page.get_by_role("alert").filter(has_text="TARGET_CHANGED").wait_for()
            with s.store.db() as db:
                assert db.execute("SELECT count(*) FROM environment_operations").fetchone()[0]==before
                assert db.execute("SELECT count(*) FROM audit").fetchone()[0]==hello_count
            assert not errors
            browser.close()
        summary={"result":"passed","evidence":"live_mcp","actions":[r["action"] for r in receipts],
                 "operations":[r["id"] for r in receipts],"new_target":receipts[0]["target"],
                 "create_identity":receipts[0]["result"]["identity"],"start_identity":receipts[2]["result"]["identity"],
                 "requested_podman":False,"observed_podman":receipts[0]["result"]["observed_podman"],
                 "deleted_readback":True,"original_hello_preserved":True,"bound_hello_count":1,
                 "restart_no_replay":True,"stale_binding_blocked":True}
        path.with_name("result.json").write_text(json.dumps(summary,indent=2))
        progress.write_text(json.dumps({"status":"passed","completed":5}))
        print(json.dumps(summary))
    finally:
        stop(worker);stop(web)
if __name__=="__main__":main()
