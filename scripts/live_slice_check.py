"""Real loopback Web/worker/MCP smoke, fresh private state only."""
import argparse
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import sys
import time
import urllib.request
from werkzeug.security import generate_password_hash
from bwmini.__main__ import load

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config",required=True)
    parser.add_argument("--browser",action="store_true")
    args=parser.parse_args()
    path=Path(args.config).absolute()
    cfg=json.loads(path.read_text())
    if cfg.get("mode")!="mcp" or cfg.get("host")!="127.0.0.1" or cfg.get("origin")!=f"http://127.0.0.1:{cfg['port']}":
        raise RuntimeError("Reviewed loopback MCP config required")
    if Path(cfg["database"]).exists():
        raise RuntimeError("Use fresh test database; existing state preserved")
    root=Path(__file__).resolve().parents[1]
    c,service=load(path)
    password=secrets.token_urlsafe(32)
    service.store.admin(generate_password_hash(password))
    logical=next(iter(c["targets"]))
    expected=c["targets"][logical]
    origin=c["origin"]
    def start(action):
        return subprocess.Popen([sys.executable,"-m","bwmini",action,"--config",str(path)],
            cwd=root,stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
    def stop(p):
        if p and p.poll() is None:
            os.killpg(p.pid,signal.SIGTERM);p.wait(timeout=10)
    def ready():
        deadline=time.monotonic()+30
        while True:
            try:
                urllib.request.urlopen(origin,timeout=1).close();return
            except OSError:
                if time.monotonic()>deadline:raise
                time.sleep(.2)
    web=worker=None
    try:
        web=start("serve");worker=start("worker");ready()
        if args.browser:
            from playwright.sync_api import sync_playwright
            with sync_playwright() as pw:
                browser=pw.chromium.launch()
                page=browser.new_page(viewport={"width":1280,"height":960})
                errors=[];page.on("pageerror",lambda e:errors.append(str(e)))
                page.goto(origin)
                page.get_by_label("Password").fill(password)
                page.get_by_role("button",name="Sign in",exact=True).click()
                page.get_by_text("Live MCP mode",exact=False).wait_for()
                page.get_by_label("Project name").fill("Current real MCP check")
                page.get_by_role("button",name="Save binding",exact=True).click()
                page.get_by_role("button",name="Preview hello",exact=True).first.click()
                page.get_by_role("button",name="Confirm hello",exact=True).click()
                page.get_by_test_id("operation-state").filter(has_text="succeeded").wait_for(timeout=120000)
                page.get_by_text("evidence: live_mcp",exact=False).wait_for()
                with service.store.db() as db:
                    before=db.execute("SELECT count(*) FROM audit").fetchone()[0]
                    receipt=dict(db.execute("SELECT * FROM operations").fetchone())
                stop(worker);worker=start("worker");stop(web);web=start("serve");ready()
                page.reload()
                page.get_by_role("button",name="Preview hello",exact=True).first.wait_for()
                page.get_by_role("button",name="Load latest receipt",exact=True).click()
                page.get_by_test_id("operation-state").filter(has_text="succeeded").wait_for()
                page.get_by_role("button",name="Refresh receipt history",exact=True).click()
                page.get_by_label("Receipt history",exact=True).get_by_role("button",name="Open receipt",exact=False).first.wait_for()
                page.set_viewport_size({"width":390,"height":844})
                assert not page.evaluate("document.documentElement.scrollWidth > innerWidth")
                assert not errors
                browser.close()
        else:
            jar=http.cookiejar.CookieJar()
            client=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPCookieProcessor(jar))
            csrf=None
            def request(route,data=None):
                headers={"Origin":origin}
                if csrf:headers["X-CSRF-Token"]=csrf
                raw=None if data is None else json.dumps(data).encode()
                if raw is not None:headers["Content-Type"]="application/json"
                with client.open(urllib.request.Request(origin+route,data=raw,headers=headers),timeout=60) as response:
                    return json.loads(response.read())
            csrf=request("/api/login",{"password":password})["csrf"]
            assert request("/api/session")["mode"]=="mcp"
            assert request("/api/resources")["mode"]=="mcp"
            p=request("/api/projects",{"name":"Current live check","logical":logical,"workspace":"/home/"+expected["user"]})
            plan=request("/api/plans",{"project":p["id"],"revision":p["revision"]})
            op=request("/api/plans/"+plan["id"]+"/confirm",{"confirm":True})
            deadline=time.monotonic()+120
            while True:
                receipt=request("/api/operations/"+op["id"])
                if receipt["state"] in ("succeeded","failed","unknown"):break
                if time.monotonic()>deadline:raise TimeoutError("Receipt pending")
                time.sleep(.5)
            assert receipt["state"]=="succeeded",receipt["code"]
            with service.store.db() as db:before=db.execute("SELECT count(*) FROM audit").fetchone()[0]
            stop(worker);worker=start("worker");stop(web);web=start("serve");ready()
            assert request("/api/operations/"+op["id"])==receipt
            assert request("/api/operation-history")["operations"][0]["id"]==op["id"]
        with service.store.db() as db:
            after=db.execute("SELECT count(*) FROM audit").fetchone()[0]
            current=dict(db.execute("SELECT * FROM operations WHERE id=?",(receipt["id"],)).fetchone())
        result=json.loads(current["result"])
        assert before==after==1
        assert result["stdout"]=="hello world\n" and result["exit_code"]==0 and result["evidence"]=="live_mcp"
        assert result["identity"]=={"user":expected["user"],"hostname":expected["hostname"]}
        summary={"result":"passed","client":"Chromium" if args.browser else "HTTP","evidence":"live_mcp",
                 "operation":current["id"],"identity":result["identity"],"stdout":result["stdout"],
                 "exit_code":result["exit_code"],"audit_count":after,"restart_no_replay":True,"history":True}
        path.with_suffix(".result.json").write_text(json.dumps(summary,indent=2))
        print(json.dumps(summary))
    finally:
        stop(worker);stop(web)

if __name__=="__main__":main()
