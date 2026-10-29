"""Actual Chromium and SQLite; every network/port resource here is an explicit fixture."""
import json
from pathlib import Path
import secrets
import sys
import threading
import time
from playwright.sync_api import sync_playwright,expect
from werkzeug.serving import make_server,WSGIRequestHandler
from werkzeug.security import generate_password_hash
from bwmini.store import Store
from bwmini.service import Service
from bwmini.web import create_app
root=Path(__file__).resolve().parents[1];sys.path.insert(0,str(root/'tests'))
from test_access import AccessFixture,NAME
state=root/'.runtime/access-browser'/secrets.token_hex(8);state.mkdir(parents=True,mode=0o700)
now=[time.time()];provider=AccessFixture(now);service=Service(Store(state/'state/state.sqlite3'),provider,clock=lambda:now[0])
password=secrets.token_urlsafe(24);service.store.admin(generate_password_hash(password));origin='http://127.0.0.1:18650'
class Quiet(WSGIRequestHandler):
    def log(self,*args):pass
server=make_server('127.0.0.1',18650,create_app(service,origin,root/'web/dist',secure=False),threaded=True,request_handler=Quiet)
thread=threading.Thread(target=server.serve_forever,daemon=True);stopping=threading.Event()
def work():
    while not stopping.is_set():service.work_one();stopping.wait(.1)
worker=threading.Thread(target=work,daemon=True)
try:
    thread.start();worker.start()
    with sync_playwright() as pw:
        browser=pw.chromium.launch();page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
        page.goto(origin);page.get_by_label('Password',exact=True).fill(password);page.get_by_role('button',name='Sign in',exact=True).click()
        page.get_by_role('button',name='Network & ports',exact=True).click();page.get_by_label('Label for tenant-a',exact=True).fill('Team A')
        with page.expect_response(lambda r:'/api/tenants/tenant-a/label' in r.url and r.request.method=='PUT'):page.get_by_role('button',name='Save label tenant-a',exact=True).click()
        expect(page.get_by_label('Source network Tenant',exact=True)).to_contain_text('Team A')
        page.get_by_label('Both directions in one atomic change',exact=True).check()
        def access_action(action):
            kind='network' if action in ('grant','revoke') else 'port'
            with page.expect_response(lambda r:r.url.endswith('/api/access-plans/'+kind) and r.request.method=='POST') as pending:
                page.get_by_role('button',name='Preview '+action,exact=True).click()
            plan=pending.value.json();field=page.get_by_label('Exact access confirmation',exact=True);field.wait_for();confirm=page.get_by_role('button',name='Confirm access change',exact=True);assert not confirm.is_enabled();field.fill(plan['confirmation'])
            with page.expect_response(lambda r:'/api/access-plans/' in r.url and r.url.endswith('/confirm') and r.request.method=='POST') as pending:confirm.click()
            op=pending.value.json();page.locator('.access-panel .receipt small').filter(has_text=op['id']).wait_for()
            page.get_by_test_id('access-state').filter(has_text='unknown' if provider.access_wire.lost else 'succeeded').wait_for(timeout=20000)
            return op['id']
        for action in ('grant','revoke','publish','unpublish'):access_action(action)
        assert len(provider.access_wire.writes)==4 and provider.access_wire.routes()==[]
        page.reload();page.get_by_role('button',name='Network & ports',exact=True).click()
        history=page.get_by_label('Access receipts',exact=True);expect(history.get_by_role('button',name='Open access receipt',exact=False)).to_have_count(4)
        assert len(provider.access_wire.writes)==4
        provider.access_wire.lost=True;unknown=access_action('publish');before=len(provider.access_wire.writes)
        page.get_by_role('button',name='Observe access state',exact=True).click();expect(page.get_by_test_id('access-state')).to_have_text('unknown')
        assert len(provider.access_wire.writes)==before
        page.get_by_role('button',name='Preview unpublish',exact=True).click();page.get_by_role('alert').filter(has_text='MANAGED_ROUTE_REQUIRED').wait_for()
        page.get_by_role('button',name='Environments',exact=True).click()
        page.get_by_label('New environment network Tenant',exact=True).select_option('tenant-a')
        page.get_by_role('button',name='Preview create',exact=True).click();page.get_by_label('Type the exact environment name',exact=True).fill(NAME)
        with page.expect_response(lambda r:'/api/environment-plans/' in r.url and r.url.endswith('/confirm') and r.request.method=='POST') as pending:page.get_by_role('button',name='Confirm create',exact=True).click()
        op=pending.value.json();page.locator('.environment-panel .receipt small').filter(has_text=op['id']).wait_for()
        page.get_by_test_id('environment-state').filter(has_text='succeeded').wait_for()
        page.get_by_role('button',name='Network & ports',exact=True).click();page.locator('.access-member').filter(has_text=NAME).filter(has_text='Network Tenant: tenant-a').wait_for()
        assert service.access.receipt(unknown)['state']=='unknown' and len(provider.access_wire.writes)==5
        page.set_viewport_size({'width':390,'height':844});assert page.evaluate('document.documentElement.scrollWidth <= window.innerWidth+1')
        page.screenshot(path=str(state/'mobile.jpg'),type='jpeg',quality=65,full_page=True)
        assert not errors;browser.close()
    result={'result':'passed','infrastructure':'fixture','state':str(state),'access_writes':5,'shared_creates':1,'checks':['metadata distinct from membership','atomic both-direction grant/revoke','publish/unpublish','exact confirmation','history reload no replay','unknown no adoption/retry','shared new membership/account separation','mobile no overflow']}
    (state/'result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
finally:stopping.set();worker.join(timeout=5);server.shutdown();server.server_close();thread.join(timeout=5)
