"""REQ010 phase6 defect regressions: unbinding is local, revisioned and nondestructive."""
import json,sqlite3,subprocess,sys
from pathlib import Path
from test_slice import Base,TARGET,HASH,PASSWORD
from bwmini.domain import Fault
from bwmini.web import create_app
from bwmini.store import Store

class BindingTests(Base):
 def test_unbind_preserves_project_receipts_and_stops_queued_execution(self):
  op=self.operation();p=self.store.project(op['project']);self.fixture.calls.clear()
  row=self.s.unbind(p['id'],p['revision']);self.assertEqual(row['revision'],2)
  with self.assertRaisesRegex(Fault,'PROJECT_UNBOUND'):self.s.plan(p['id'],2)
  self.s.worker(once=True);self.assertEqual(self.store.receipt(op['id'])['state'],'failed')
  self.assertEqual(self.fixture.calls,[])
  self.assertEqual(self.s.projects()[0]['name'],p['name']);self.assertFalse(self.s.projects()[0]['bound'])
  with self.store.db() as db:self.assertEqual(db.execute('SELECT count(*) FROM projects').fetchone()[0],1)
 def test_stale_confirmation_and_unbind_do_not_mutate(self):
  p=self.bind();plan=self.s.plan(p['id'],1);self.s.unbind(p['id'],1)
  with self.assertRaisesRegex(Fault,'REVISION_CONFLICT'):self.s.confirm(plan['id'])
  with self.assertRaisesRegex(Fault,'REVISION_CONFLICT'):self.s.unbind(p['id'],1)
  self.assertEqual(self.store.project(p['id'])['revision'],2)
 def test_rebind_reuses_identity_and_workspace_after_restart(self):
  p=self.bind();self.s.unbind(p['id'],1)
  self.s.bind('Retained',TARGET.logical,'/home/fixture-box/new',p['id'],2)
  from bwmini.service import Service
  again=Service(Store(self.store.path),self.fixture)
  row=again.projects()[0];self.assertTrue(row['bound']);self.assertEqual(row['target'],TARGET.data());self.assertEqual(row['workspace'],'/home/fixture-box/new')
 def test_unbind_does_not_need_reachable_provider_or_touch_graph(self):
  p=self.bind()
  with self.store.db() as db:db.execute('INSERT INTO graphs VALUES(?,?,?)',('graph',1,'{}'))
  self.fixture.observe=lambda _:self.fail('Unbind contacted provider')
  self.s.unbind(p['id'],1)
  with self.store.db() as db:self.assertEqual(db.execute('SELECT count(*) FROM graphs').fetchone()[0],1)
 def test_authenticated_origin_csrf_exact_body_and_revision_gate(self):
  self.store.admin(HASH);p=self.bind();app=create_app(self.s,'http://localhost',self.root,secure=False);c=app.test_client()
  url='/api/projects/'+p['id']+'/binding'
  self.assertEqual(c.delete(url,json={'revision':1},headers={'Origin':'http://localhost'}).status_code,401)
  login=c.post('/api/login',json={'password':PASSWORD},headers={'Origin':'http://localhost'});csrf=login.json['csrf'];good={'Origin':'http://localhost','X-CSRF-Token':csrf}
  self.assertNotEqual(c.delete(url,json={'revision':1},headers={'Origin':'http://other'}).status_code,200)
  self.assertNotEqual(c.delete(url,json={'revision':1},headers={'Origin':'http://localhost'}).status_code,200)
  self.assertNotEqual(c.delete(url,json={'revision':True},headers=good).status_code,200)
  self.assertEqual(c.delete(url,json={'revision':1},headers=good).status_code,200)
 def test_empty_target_diagnosis_authenticates_without_database_or_writes(self):
  config=self.root/'diagnostic.json';config.write_text(json.dumps({'mode':'mcp','provider':'containarium','backend':'fixture','project':'default','targets':{},'launcher':[sys.executable,str(Path(__file__).with_name('mcp_peer.py').absolute()),'diagnostic-empty']}))
  result=subprocess.run([sys.executable,'-m','bwmini','diagnose','--config',str(config)],capture_output=True,text=True)
  self.assertEqual(result.returncode,0,result.stdout+result.stderr);value=json.loads(result.stdout)
  self.assertEqual(value['targets'],[]);self.assertEqual(value['connection']['state'],'authenticated_read_passed');self.assertEqual(value['connection']['read'],'list_containers')
  config.write_text(json.dumps({'mode':'mcp','provider':'containarium','backend':'fixture','project':'default','targets':{},'launcher':['/usr/bin/false']}))
  failed=subprocess.run([sys.executable,'-m','bwmini','diagnose','--config',str(config)],capture_output=True,text=True);self.assertNotEqual(failed.returncode,0)
 def test_schema6_migrates_preserves_bound(self):
  p=self.bind()
  with self.store.db() as db:
    db.execute('DROP TABLE project_unbindings');db.execute('DROP TABLE job_runs')
    for t in ('diagnostic_events','diagnostic_trace_links','selftest_runs'):db.execute('DROP TABLE '+t)
    db.execute('DROP TABLE box_action_receipts');db.execute('DROP TABLE box_action_plans');db.execute('PRAGMA user_version=6')
  updated=Store(self.store.path)
  with updated.db() as db:self.assertEqual(db.execute('PRAGMA user_version').fetchone()[0],11)
  self.assertEqual(updated.project(p['id'])['target'],json.dumps(TARGET.data()))
