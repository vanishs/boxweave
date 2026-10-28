"""Real browser/application/SQLite, explicitly synthetic infrastructure."""
import json
import os
from pathlib import Path
import secrets
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request
from playwright.sync_api import sync_playwright
from werkzeug.security import generate_password_hash
from bwmini.__main__ import load

root=Path(__file__).resolve().parents[1]
state=root/".runtime"/"browser"/secrets.token_hex(8)
state.mkdir(parents=True,mode=0o700,exist_ok=False)
password=secrets.token_urlsafe(24)
counter=state/"calls.txt"
counter.write_text("")
config={"mode":"fixture","database":str(state/"state.sqlite3"),
"target":{"provider":"containarium","backend":"dev","project":"default","logical":"fixture-box",
"instance":"fixture-box-container","created":"123","user":"fixture-box","hostname":"fixture-box-container"},
"fixture_counter":str(counter),"host":"127.0.0.1","port":18640,"origin":"http://127.0.0.1:18640",
"secure_cookie":False,"dist":str(root/"web/dist"),"operation_timeout":60}
path=state/"config.json";path.write_text(json.dumps(config));path.chmod(0o600)
_,service=load(path)
with service.store.db() as db:
    if not db.execute("SELECT 1 FROM admin").fetchone():
        service.store.admin(generate_password_hash(password))
    else:
        db.execute("UPDATE admin SET password=? WHERE id=1",(generate_password_hash(password),))
        db.execute("DELETE FROM sessions")
def start(action):
    return subprocess.Popen([sys.executable,"-m","bwmini",action,"--config",str(path)],
                            cwd=root,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
def stop(p):
    if p.poll() is None:
        os.killpg(p.pid,signal.SIGTERM);p.wait(timeout=5)
server=worker=None
try:
    with socket.socket() as check:
        check.bind(("127.0.0.1",18640))
    server=start("serve");worker=start("worker")
    deadline=time.monotonic()+30
    while True:
        try:
            urllib.request.urlopen(config["origin"],timeout=1).close();break
        except OSError:
            if time.monotonic()>deadline:raise RuntimeError("Browser fixture not ready")
            time.sleep(.2)
    with sync_playwright() as p:
        browser=p.chromium.launch()
        page=browser.new_page(viewport={"width":1280,"height":960})
        errors=[]
        page.on("pageerror",lambda e:errors.append(str(e)))
        page.goto(config["origin"])
        page.get_by_label("Password").fill(password)
        page.get_by_role("button",name="Sign in",exact=True).click()
        page.get_by_text("Fixture mode",exact=False).wait_for()
        page.get_by_label("Project name").fill("Browser project")
        page.get_by_role("button",name="Save binding",exact=True).click()
        page.get_by_role("button",name="Preview hello",exact=True).first.click()
        page.get_by_role("button",name="Confirm hello",exact=True).click()
        page.get_by_test_id("operation-state").filter(has_text="succeeded").wait_for()
        page.get_by_text("hello world",exact=True).wait_for()
        calls=counter.read_text().splitlines()
        if calls!=["hello"]:raise AssertionError("Unexpected infrastructure fixture call count")
        page.screenshot(path=str(state/"slice-desktop.png"),full_page=True)
        stop(worker);worker=start("worker")
        stop(server);server=start("serve");time.sleep(1)
        page.reload()
        page.get_by_role("button",name="Load latest receipt",exact=True).click()
        page.get_by_test_id("operation-state").filter(has_text="succeeded").wait_for()
        if counter.read_text().splitlines()!=calls:raise AssertionError("Restart replayed operation")
        page.set_viewport_size({"width":390,"height":844})
        page.screenshot(path=str(state/"slice-mobile.png"),full_page=True)
        if page.evaluate("document.documentElement.scrollWidth > innerWidth"):
            raise AssertionError("Mobile horizontal overflow")
        if errors:raise AssertionError("Browser script error")
        browser.close()
    print(json.dumps({"browser":"chromium","infrastructure":"fixture","result":"passed",
      "checks":["login","binding","preview","confirm","hello-receipt","api-worker-restart","no-replay","mobile-no-overflow"],
      "screenshots":["slice-desktop.png","slice-mobile.png"],"call_count":len(calls),"evidence_dir":str(state)}))
finally:
    if worker:stop(worker)
    if server:stop(server)
