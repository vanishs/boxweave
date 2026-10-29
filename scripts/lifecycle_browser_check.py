"""Real Chromium/HTTP/SQLite lifecycle UI; explicitly in-memory infrastructure fixture."""
import json
from pathlib import Path
import secrets
import socket
import sys
import threading
import time
from playwright.sync_api import sync_playwright
from werkzeug.serving import make_server, WSGIRequestHandler
from werkzeug.security import generate_password_hash
from bwmini.store import Store
from bwmini.service import Service
from bwmini.web import create_app
sys.path.insert(0,str(Path(__file__).resolve().parents[1]/"tests"))
from test_lifecycle import LifecycleFixture, NAME

root=Path(__file__).resolve().parents[1]
state=root/".runtime"/"lifecycle-browser"/secrets.token_hex(8)
state.mkdir(parents=True,mode=0o700)
password=secrets.token_urlsafe(24)
provider=LifecycleFixture()
service=Service(Store(state/"state/state.sqlite3"),provider)
service.store.admin(generate_password_hash(password))
origin="http://127.0.0.1:18645"
class Quiet(WSGIRequestHandler):
    def log(self,kind,message,*args):pass
server=make_server("127.0.0.1",18645,create_app(service,origin,root/"web/dist",secure=False),threaded=True,request_handler=Quiet)
http=threading.Thread(target=server.serve_forever,daemon=True)
stop=threading.Event()
def work():
    while not stop.is_set():
        service.work_one()
        stop.wait(.1)
worker=threading.Thread(target=work,daemon=True)
try:
    http.start();worker.start()
    with sync_playwright() as pw:
        browser=pw.chromium.launch()
        page=browser.new_page(viewport={"width":1280,"height":960})
        errors=[];page.on("pageerror",lambda e:errors.append(str(e)))
        page.goto(origin)
        page.get_by_label("Password").fill(password)
        page.get_by_role("button",name="Sign in",exact=True).click()
        page.get_by_role("button",name="Environments",exact=True).click()
        page.get_by_label("Authorized new environment").wait_for()
        for action in ("create","stop","start","stop","delete"):
            page.get_by_role("button",name="Preview "+action,exact=True).click()
            field=page.get_by_label("Type the exact environment name")
            field.wait_for()
            confirm=page.get_by_role("button",name="Confirm "+action,exact=True)
            assert not confirm.is_enabled()
            field.fill(NAME)
            with page.expect_response(lambda r:'/api/environment-plans/' in r.url and r.url.endswith('/confirm') and r.request.method=='POST') as response:
                confirm.click()
            oid=response.value.json()["id"]
            page.locator(".environment-panel .receipt small").filter(has_text=oid).wait_for()
            page.get_by_test_id("environment-state").filter(has_text="succeeded").wait_for(timeout=20000)
        before=list(provider.writes)
        page.reload()
        page.get_by_role("button",name="Environments",exact=True).click()
        history=page.get_by_label("Environment receipts",exact=True)
        history.get_by_role("button",name="Open operation",exact=False).first.wait_for()
        assert history.get_by_role("button",name="Open operation",exact=False).count()==5
        history.get_by_role("button",name="Open operation",exact=False).first.click()
        page.get_by_test_id("environment-state").filter(has_text="succeeded").wait_for()
        assert provider.writes==before
        page.set_viewport_size({"width":390,"height":844})
        page.locator(".environment-panel").scroll_into_view_if_needed()
        page.screenshot(path=str(state/"mobile.jpg"),type="jpeg",quality=65,full_page=True)
        assert not page.evaluate("document.documentElement.scrollWidth > innerWidth")
        assert not errors
        browser.close()
    result={"result":"passed","infrastructure":"fixture","writes":[a for a,n in before],
            "checks":["exact confirmation required","create/stop/start/stop/delete","receipt reload no replay","mobile no overflow"],
            "state":str(state)}
    (state/"result.json").write_text(json.dumps(result,indent=2))
    print(json.dumps(result))
finally:
    stop.set();worker.join(5)
    server.shutdown();server.server_close();http.join(5)
