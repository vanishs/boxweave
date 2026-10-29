"""Actual Chromium/local SQLite model wizard; native model access is deliberately a fixture report."""
import json
from pathlib import Path
import secrets
import threading
from playwright.sync_api import sync_playwright,expect
from werkzeug.serving import make_server,WSGIRequestHandler
from werkzeug.security import generate_password_hash
from bwmini.store import Store
from bwmini.service import Service
from bwmini.provider import Fixture
from bwmini.domain import Target
from bwmini.web import create_app
root=Path(__file__).resolve().parents[1];state=root/'.runtime/model-browser'/secrets.token_hex(8);state.mkdir(parents=True,mode=0o700)
target=Target('containarium','dev','default','fixture-box','fixture-box-container','123','fixture-box','fixture-box-container')
provider=Fixture(target);service=Service(Store(state/'state/state.sqlite3'),provider)
password=secrets.token_urlsafe(24);service.store.admin(generate_password_hash(password));origin='http://127.0.0.1:18648'
class Quiet(WSGIRequestHandler):
    def log(self,*args):pass
server=make_server('127.0.0.1',18648,create_app(service,origin,root/'web/dist',secure=False),threaded=True,request_handler=Quiet)
thread=threading.Thread(target=server.serve_forever,daemon=True)
try:
    thread.start()
    with sync_playwright() as pw:
        browser=pw.chromium.launch();page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
        page.goto(origin);page.get_by_label('Password',exact=True).fill(password);page.get_by_role('button',name='Sign in',exact=True).click()
        page.get_by_role('button',name='Model setup',exact=True).click()
        page.get_by_test_id('model-status').filter(has_text='not_selected').wait_for()
        page.get_by_label('Model provider',exact=True).select_option('bigmodel')
        page.get_by_role('button',name='Save model selection',exact=True).click()
        page.get_by_test_id('model-status').filter(has_text='native_setup_pending').wait_for()
        page.reload();page.get_by_role('button',name='Model setup',exact=True).click()
        page.get_by_label('Model provider',exact=True).wait_for();assert page.get_by_label('Model provider',exact=True).input_value()=='bigmodel'
        page.get_by_role('button',name='Show native setup steps',exact=True).click();handoff=page.get_by_label('Public model setup handoff',exact=True);handoff.wait_for()
        report=json.loads(handoff.input_value());assert report['selection']['model']=='GLM-5.3' and report['inference']=='unverified'
        assert report['credentials']=='not_collected' and not page.locator('input[type=password]').count()
        confirm=page.get_by_role('button',name='Record my native check',exact=True);assert not confirm.is_enabled()
        page.get_by_label('I selected this provider and model in ZCode',exact=True).check();assert not confirm.is_enabled()
        page.get_by_label('ZCode reports my provider access is available',exact=True).check();confirm.click()
        page.get_by_test_id('model-status').filter(has_text='operator_reported_ready').wait_for()
        page.get_by_label('GLM model',exact=True).select_option('GLM-5.3-Flash');page.get_by_role('button',name='Save model selection',exact=True).click()
        page.get_by_test_id('model-status').filter(has_text='native_setup_pending').wait_for()
        # Another authenticated editor changes the revision; stale UI cannot overwrite it.
        csrf=page.request.get(origin+'/api/session').json()['csrf'];current=page.request.get(origin+'/api/model-wizard').json()
        value=dict(current['selection'],revision=current['revision'],provider='zai')
        result=page.request.put(origin+'/api/model-wizard',data=value,headers={'Origin':origin,'X-CSRF-Token':csrf});assert result.ok
        page.get_by_label('GLM model',exact=True).select_option('GLM-5.3');page.get_by_role('button',name='Save model selection',exact=True).click()
        page.get_by_role('alert').filter(has_text='MODEL_REVISION_CONFLICT').wait_for()
        page.get_by_role('button',name='Reload model setup',exact=True).click()
        expect(page.get_by_label('GLM model',exact=True)).to_have_value('GLM-5.3-Flash')
        page.set_viewport_size({'width':390,'height':844});page.get_by_role('button',name='Show native setup steps',exact=True).click();handoff.wait_for()
        assert page.evaluate('document.documentElement.scrollWidth <= window.innerWidth+1')
        page.screenshot(path=str(state/'mobile.jpg'),type='jpeg',quality=65,full_page=True)
        assert not errors and provider.calls==[];browser.close()
    result={'result':'passed','state':str(state),'infrastructure':'fixture','native_access':'fixture_operator_report_only','inference':'not_requested','provider_execution_calls':0,'checks':['save/reload','public native handoff','two explicit native confirmations','changed selection clears readiness','stale revision rejected','mobile no overflow']}
    (state/'result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
finally:server.shutdown();server.server_close();thread.join(timeout=5)
