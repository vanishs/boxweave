"""Real Chromium/HTTP/SQLite graph workflow; infrastructure is explicitly fixture-only."""
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

root=Path(__file__).resolve().parents[1]
state=root/'.runtime/graphs'/secrets.token_hex(8);state.mkdir(parents=True,mode=0o700)
password=secrets.token_urlsafe(24);counter=state/'calls.txt';counter.write_text('')
origin='http://127.0.0.1:18643'
config={'mode':'fixture','database':str(state/'state.sqlite3'),'target':{'provider':'containarium','backend':'dev','project':'default','logical':'fixture-box','instance':'fixture-box-container','created':'123','user':'fixture-box','hostname':'fixture-box-container'},'fixture_counter':str(counter),'host':'127.0.0.1','port':18643,'origin':origin,'secure_cookie':False,'dist':str(root/'web/dist')}
path=state/'config.json';path.write_text(json.dumps(config));path.chmod(0o600)
_,service=load(path);service.store.admin(generate_password_hash(password))
server=None
try:
    log=(state/'server.log').open('w')
    server=subprocess.Popen([sys.executable,'-m','bwmini','serve','--config',str(path)],cwd=root,stdout=log,stderr=log,start_new_session=True);log.close()
    deadline=time.monotonic()+30
    while True:
        try:
            urllib.request.urlopen(origin,timeout=1).close();break
        except OSError:
            if time.monotonic()>deadline:raise
            time.sleep(.2)
    with sync_playwright() as p:
        browser=p.chromium.launch();context=browser.new_context(permissions=['clipboard-read','clipboard-write'],accept_downloads=True,viewport={'width':1280,'height':960})
        page=context.new_page();errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
        page.goto(origin);page.get_by_label('Password').fill(password);page.get_by_role('button',name='Sign in',exact=True).click()
        page.get_by_label('Project name').fill('Environment remains')
        page.get_by_role('button',name='Save binding',exact=True).click()
        page.get_by_text('Environment remains',exact=True).wait_for()
        page.get_by_role('button',name='Prompt Graph',exact=True).click()
        page.get_by_label('Graph name').fill('Temporary draft')
        assert page.get_by_role('button',name='Projects',exact=True).is_disabled()
        page.once('dialog',lambda d:d.accept());page.get_by_role('button',name='Discard draft',exact=True).click()
        page.get_by_role('status').filter(has_text='Draft discarded.').wait_for()
        assert page.get_by_role('button',name='Projects',exact=True).is_enabled()
        page.get_by_label('Graph name').fill('Review plan');page.get_by_label('Graph prompt').fill('Compare approaches without execution.')
        page.get_by_role('button',name='Add card',exact=True).click()
        page.get_by_label('Card title').fill('Investigate');page.get_by_label('Card text').fill('api_key=synthetic-confidential\n[untrusted](example)')
        page.get_by_label('Project context').fill('demo');page.get_by_label('Environment context').fill('fixture-box');page.get_by_label('Tenant context').fill('lab')
        page.get_by_label('Comment text').fill('Keep this context');page.get_by_role('button',name='Add comment',exact=True).click()
        page.get_by_label('Edit comment 1',exact=True).fill('Updated context')
        page.get_by_role('button',name='Add card',exact=True).click();page.get_by_label('Card title').fill('Compare');page.get_by_label('Card text').fill('Review the evidence')
        page.get_by_label('Selected card',exact=True).select_option(label='Investigate');page.get_by_label('Link selected card to').select_option(label='Compare');page.get_by_role('button',name='Add relation',exact=True).click()
        node=page.locator('.react-flow__node').first;node.scroll_into_view_if_needed();box=node.bounding_box();assert box
        page.mouse.move(box['x']+box['width']/2,box['y']+box['height']/2);page.mouse.down();page.mouse.move(box['x']+box['width']/2+70,box['y']+box['height']/2+30,steps=8);page.mouse.up()
        page.get_by_role('button',name='Save graph',exact=True).click();page.get_by_role('status').filter(has_text='Saved revision 1').wait_for()
        listing=page.request.get(origin+'/api/graphs').json()['graphs'];assert len(listing)==1;gid=listing[0]['id']
        saved=page.request.get(origin+'/api/graphs/'+gid).json();doc=saved['document']
        assert len(doc['cards'])==2 and len(doc['links'])==1 and len(doc['comments'])==1
        assert doc['comments'][0]['text']=='Updated context'
        assert doc['cards'][0]['context']['environment']=='fixture-box'
        assert (doc['cards'][0]['x'],doc['cards'][0]['y'])!=(0,0)
        page.reload();page.get_by_role('button',name='Prompt Graph',exact=True).click();page.get_by_label('Saved graph').select_option(gid)
        page.wait_for_function("document.querySelector('input')?.value==='Review plan'")
        assert page.get_by_label('Graph prompt').input_value()=='Compare approaches without execution.'
        second=context.new_page();second.goto(origin);second.get_by_role('button',name='Prompt Graph',exact=True).click();second.get_by_label('Saved graph').select_option(gid)
        second.wait_for_function("document.querySelector('input')?.value==='Review plan'")
        page.get_by_label('Graph prompt').fill('Local unsaved draft')
        second.get_by_label('Graph name').fill('Other editor');second.get_by_role('button',name='Save graph',exact=True).click();second.get_by_role('status').filter(has_text='Saved revision 2').wait_for()
        page.bring_to_front();page.get_by_role('button',name='Save graph',exact=True).click();page.get_by_role('status').filter(has_text='REVISION_CONFLICT').wait_for()
        assert page.get_by_label('Graph prompt').input_value()=='Local unsaved draft'
        page.once('dialog',lambda d:d.accept());page.get_by_role('button',name='Reload saved graph',exact=True).click();page.get_by_role('status').filter(has_text='Reloaded saved graph').wait_for();assert 'Other editor' in page.get_by_label('Saved graph').inner_text()
        page.get_by_role('button',name='Copy Markdown',exact=True).click();page.get_by_role('status').filter(has_text='Copied redacted Markdown').wait_for()
        copied=page.evaluate('navigator.clipboard.readText()');assert 'synthetic-confidential' not in copied and 'REDACTED' in copied and r'\[untrusted\]' in copied
        with service.store.db() as db:assert db.execute('SELECT count(*) FROM graph_copies').fetchone()[0]==1
        page.evaluate("() => {window.originalWrite=navigator.clipboard.writeText.bind(navigator.clipboard);navigator.clipboard.writeText=async()=>{throw new Error('deliberate clipboard rejection');};}")
        page.get_by_role('button',name='Copy Markdown',exact=True).click();page.get_by_role('status').filter(has_text='Clipboard failed; no successful copy recorded.').wait_for()
        with service.store.db() as db:assert db.execute('SELECT count(*) FROM graph_copies').fetchone()[0]==1
        page.evaluate('() => {navigator.clipboard.writeText=window.originalWrite;}')
        with page.expect_download() as download:page.get_by_role('button',name='Export Markdown',exact=True).click()
        exported=Path(download.value.path()).read_text();assert exported==copied
        page.get_by_label('Selected card',exact=True).select_option(label='Investigate')
        page.screenshot(path=str(state/'graph-desktop.png'),full_page=True)
        page.set_viewport_size({'width':390,'height':844})
        page.locator('.graph-canvas').scroll_into_view_if_needed();page.wait_for_timeout(300)
        page.locator('.react-flow__controls-fitview').click()
        deadline=time.monotonic()+10
        visible="() => {const c=document.querySelector('.graph-canvas').getBoundingClientRect();const nodes=[...document.querySelectorAll('.react-flow__node')];return nodes.length===2&&nodes.every(n=>{const r=n.getBoundingClientRect();return r.left>=c.left&&r.right<=c.right&&r.top>=c.top&&r.bottom<=c.bottom;});}"
        while not page.evaluate(visible):
            if time.monotonic()>deadline:
                print('fit_failure='+str(page.evaluate("() => ({canvas:document.querySelector('.graph-canvas').getBoundingClientRect().toJSON(),nodes:[...document.querySelectorAll('.react-flow__node')].map(n=>({text:n.textContent,rect:n.getBoundingClientRect().toJSON()})),transform:document.querySelector('.react-flow__viewport').style.transform})")),flush=True)
                page.screenshot(path=str(state/'fit-failure.png'),full_page=True)
                raise AssertionError('Mobile canvas did not fit both nodes; '+str(state))
            page.wait_for_timeout(100)
        page.screenshot(path=str(state/'graph-mobile.png'),full_page=True)
        assert not page.evaluate('document.documentElement.scrollWidth>innerWidth')
        page.once('dialog',lambda d:d.accept());page.get_by_role('button',name='Delete graph',exact=True).click();page.get_by_role('status').filter(has_text='Graph deleted').wait_for()
        page.get_by_role('button',name='Projects',exact=True).click();page.get_by_text('Environment remains',exact=True).wait_for()
        with service.store.db() as db:
            assert db.execute('SELECT count(*) FROM operations').fetchone()[0]==0
            assert db.execute('SELECT count(*) FROM projects').fetchone()[0]==1
        assert counter.read_text()=='' and not errors
        summary={'result':'passed','browser':'chromium','browser_version':browser.version,'infrastructure':'fixture, zero graph execution','checks':['card edit and drag','context and comment','undirected relation','save and full reload','concurrent edit conflict preserves draft','actual browser clipboard','clipboard rejection creates no success','escaped/redacted Markdown download','graph delete preserves binding','zero operations','desktop/mobile without overflow','mobile canvas fit'],'state':str(state)}
        (state/'result.json').write_text(json.dumps(summary,indent=2));print(json.dumps(summary));browser.close()
finally:
    if server and server.poll() is None:os.killpg(server.pid,signal.SIGTERM);server.wait(timeout=10)
