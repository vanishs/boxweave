"""Actual Chromium/HTTP/SQLite receipt browsing, fixture infrastructure only."""
import json
import os
from pathlib import Path
import secrets
import signal
import socket
import subprocess
import sys
import time
import urllib.request
from playwright.sync_api import sync_playwright
from werkzeug.security import generate_password_hash
from bwmini.__main__ import load

root = Path(__file__).resolve().parents[1]
state = root / ".runtime" / "history-browser" / secrets.token_hex(8)
state.mkdir(parents=True, mode=0o700)
password = secrets.token_urlsafe(24)
counter = state / "calls.txt"
counter.write_text("")
config = {"mode":"fixture", "database":str(state/"state.sqlite3"),
          "target":{"provider":"containarium","backend":"dev","project":"default",
                    "logical":"fixture-box","instance":"fixture-box-container","created":"123",
                    "user":"fixture-box","hostname":"fixture-box-container"},
          "fixture_counter":str(counter), "host":"127.0.0.1","port":18644,
          "origin":"http://127.0.0.1:18644","secure_cookie":False,
          "dist":str(root/"web/dist"),"operation_timeout":60}
path = state/"config.json"
path.write_text(json.dumps(config))
path.chmod(0o600)
_, service = load(path)
service.store.admin(generate_password_hash(password))
project = service.bind("History fixture", "fixture-box", "/home/fixture-box")
with service.store.db() as db:
    for i in range(25):
        db.execute("INSERT INTO operations VALUES(?,?,?,?,?,?,?,?,?,?)",
                   (f"{i:08x}"+"0"*24, project["id"],1,project["digest"],"operator",
                    "unknown",1000.,1060.,None,"WORKER_LOST"))

def start(action):
    return subprocess.Popen([sys.executable,"-m","bwmini",action,"--config",str(path)],
                            cwd=root,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,
                            start_new_session=True)
def stop(process):
    if process and process.poll() is None:
        os.killpg(process.pid, signal.SIGTERM)
        process.wait(timeout=5)
def ready():
    deadline=time.monotonic()+30
    while True:
        try:
            urllib.request.urlopen(config["origin"],timeout=1).close()
            return
        except OSError:
            if time.monotonic()>deadline:
                raise RuntimeError("History browser server not ready")
            time.sleep(.2)

server=worker=None
try:
    with socket.socket() as check:
        check.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        check.bind(("127.0.0.1",18644))
    server=start("serve")
    worker=start("worker")
    ready()
    with sync_playwright() as playwright:
        browser=playwright.chromium.launch()
        page=browser.new_page(viewport={"width":1280,"height":960})
        errors=[]
        page.on("pageerror",lambda error:errors.append(str(error)))
        page.goto(config["origin"])
        page.get_by_label("Password").fill(password)
        page.get_by_role("button",name="Sign in",exact=True).click()
        page.get_by_role("button",name="Preview hello",exact=True).click()
        page.get_by_role("button",name="Confirm hello",exact=True).click()
        page.get_by_test_id("operation-state").filter(has_text="succeeded").wait_for()
        page.get_by_role("button",name="Refresh receipt history",exact=True).click()
        history=page.get_by_label("Receipt history",exact=True)
        history.get_by_role("button",name="Open receipt",exact=False).first.wait_for()
        assert history.get_by_role("button",name="Open receipt",exact=False).count()==20
        history.get_by_role("button",name="Older receipts",exact=True).click()
        history.get_by_role("button",name="Open receipt 00000000",exact=True).wait_for()
        assert history.get_by_role("button",name="Open receipt",exact=False).count()==6
        history.get_by_role("button",name="Open receipt 00000000",exact=True).click()
        page.get_by_test_id("operation-state").filter(has_text="unknown").wait_for()
        page.get_by_text("Completion is unknown.",exact=False).wait_for()
        page.locator(".receipt small").filter(has_text="0"*32).wait_for()
        assert page.locator(".receipt pre").count()==0
        # A failed history refresh must retain selected evidence and the older page.
        page.route("**/api/operation-history?*", lambda route: route.fulfill(
            status=503, content_type="application/json", body='{"code":"STATE_UNAVAILABLE"}'))
        page.get_by_role("button",name="Refresh receipt history",exact=True).click()
        page.get_by_role("alert").filter(has_text="STATE_UNAVAILABLE").wait_for()
        assert history.get_by_role("button",name="Open receipt",exact=False).count()==6
        assert page.get_by_test_id("operation-state").inner_text()=="unknown"
        page.unroute("**/api/operation-history?*")
        page.locator(".receipt").scroll_into_view_if_needed()
        page.screenshot(path=str(state/"history-desktop.jpg"),type="jpeg",quality=65)
        stop(worker); worker=start("worker")
        stop(server); server=start("serve")
        ready()
        page.reload()
        page.get_by_role("button",name="Refresh receipt history",exact=True).click()
        history.get_by_role("button",name="Older receipts",exact=True).click()
        history.get_by_role("button",name="Open receipt 00000000",exact=True).click()
        page.get_by_test_id("operation-state").filter(has_text="unknown").wait_for()
        assert counter.read_text().splitlines()==["hello"]
        with service.store.db() as db:
            assert db.execute("SELECT count(*) FROM operations").fetchone()[0]==26
            assert db.execute("SELECT count(*) FROM audit").fetchone()[0]==1
        page.set_viewport_size({"width":390,"height":844})
        page.locator(".receipt").scroll_into_view_if_needed()
        page.screenshot(path=str(state/"history-mobile.jpg"),type="jpeg",quality=65)
        assert not page.evaluate("document.documentElement.scrollWidth > innerWidth")
        assert not errors, errors
        browser.close()
    result={"result":"passed","browser":"Chromium","infrastructure":"fixture",
            "operations":26,"executions":1,"checks":["20+6 pagination","unknown detail","failed refresh retains evidence",
            "receipt rediscovery after API/worker restart","no replay","mobile no overflow"],
            "evidence_dir":str(state)}
    (state/"result.json").write_text(json.dumps(result,indent=2))
    print(json.dumps(result))
finally:
    stop(worker)
    stop(server)
