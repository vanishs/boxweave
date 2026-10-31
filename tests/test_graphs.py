import copy
import json
from pathlib import Path
import sqlite3
import tempfile
import unittest
from bwmini.domain import Fault
from bwmini.graphs import Graphs,validate,markdown
from bwmini.store import Store,SCHEMA
from bwmini.service import Service
from bwmini.web import create_app
from test_slice import TARGET,HASH,PASSWORD
from bwmini.provider import Fixture

def document():
    context={'project':'demo','environment':'workload','tenant':'lab'}
    return {'name':'Review','prompt':'Plan only','cards':[{'id':'a','title':'First','body':'One','x':0,'y':0,'context':context},{'id':'b','title':'Second','body':'Two','x':250,'y':80,'context':dict(context)}],'comments':[{'id':'note','text':'Keep context','card':'a'}],'links':[{'id':'edge','source':'a','target':'b'}]}

class GraphTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.path=Path(self.tmp.name)/'state/state.sqlite3'
        self.store=Store(self.path);self.graphs=Graphs(self.store)
    def tearDown(self):self.tmp.cleanup()
    def test_persist_all_context_and_positions(self):
        g=self.graphs.save(document())
        again=Graphs(Store(self.path)).get(g['id'])
        self.assertEqual(again,g);self.assertEqual(again['document']['cards'][1]['y'],80)
    def test_two_editors_conflict_without_overwrite(self):
        g=self.graphs.save(document());other=Graphs(Store(self.path))
        changed=copy.deepcopy(g['document']);changed['prompt']='First editor wins'
        self.graphs.save(changed,g['id'],1)
        with self.assertRaisesRegex(Fault,'REVISION_CONFLICT'):other.save(document(),g['id'],1)
        self.assertEqual(other.get(g['id'])['document']['prompt'],'First editor wins')
    def test_invalid_graph_references_and_fields(self):
        variants=[]
        d=document();d['cards'][1]['id']='a';variants.append(d)
        d=document();d['links'][0]['target']='missing';variants.append(d)
        d=document();d['links'][0]['target']='a';variants.append(d)
        d=document();d['links'].append({'id':'reverse','source':'b','target':'a'});variants.append(d)
        d=document();d['comments'][0]['card']=[];variants.append(d)
        d=document();d['comments'][0]['card']='missing';variants.append(d)
        d=document();d['cards'][0]['x']=float('nan');variants.append(d)
        d=document();d['cards'][0]['x']=True;variants.append(d)
        d=document();d['exec']='do not run';variants.append(d)
        for value in variants:
            with self.subTest(value=value),self.assertRaises(Fault):validate(value)
    def test_graph_size_limits(self):
        d=document();d['cards'][0]['body']='x'*2001
        with self.assertRaises(Fault):validate(d)
        d=document();d['prompt']='界'*2000;d['cards'][0]['body']='界'*2000;d['cards'][1]['body']='界'*2000
        with self.assertRaisesRegex(Fault,'GRAPH_TOO_LARGE'):validate(d)
    def test_markdown_escapes_context_and_redacts_known_patterns(self):
        d=document();d['name']='[link](evil) <script>*'
        d['cards'][0]['body']='api_key=synthetic-confidential\npassword="private phrase"\neyJabcdefghijk.abcdefghijklmnop.qrstuvwxyzABCDE'
        out=markdown(d)
        self.assertIn(r'\[link\]',out);self.assertIn(r'\<script\>',out)
        self.assertNotIn('synthetic-confidential',out);self.assertNotIn('private phrase',out);self.assertNotIn('eyJabcdefghijk',out)
        self.assertIn('REDACTED',out);self.assertIn('First ↔ Second',out)
    def test_export_does_not_claim_copy_and_copy_revision_is_checked(self):
        g=self.graphs.save(document());self.graphs.export(g['id'])
        with self.store.db() as db:self.assertEqual(db.execute('SELECT count(*) FROM graph_copies').fetchone()[0],0)
        self.assertEqual(self.graphs.copied(g['id'],1)['status'],'reported_success')
        self.graphs.save(document(),g['id'],1)
        with self.assertRaisesRegex(Fault,'REVISION_CONFLICT'):self.graphs.copied(g['id'],1)
        with self.store.db() as db:self.assertEqual(db.execute('SELECT count(*) FROM graph_copies').fetchone()[0],1)
    def test_graph_delete_preserves_environment_and_cascades_own_events(self):
        service=Service(self.store,Fixture(TARGET));project=service.bind('Existing',TARGET.logical,'/home/fixture-box')
        g=self.graphs.save(document());self.graphs.copied(g['id'],1)
        with self.assertRaisesRegex(Fault,'REVISION_CONFLICT'):self.graphs.delete(g['id'],2)
        self.graphs.delete(g['id'],1)
        self.assertEqual(self.store.project(project['id'])['name'],'Existing')
        with self.store.db() as db:self.assertEqual(db.execute('SELECT count(*) FROM graph_copies').fetchone()[0],0)
        with self.assertRaisesRegex(Fault,'NOT_FOUND'):self.graphs.get(g['id'])
    def test_boolean_revision_rejected(self):
        g=self.graphs.save(document())
        with self.assertRaisesRegex(Fault,'INVALID_REVISION'):self.graphs.save(document(),g['id'],True)

class MigrationTests(unittest.TestCase):
    def test_schema_one_data_survives_atomic_upgrade(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'state.sqlite3'
            with sqlite3.connect(path) as db:
                db.executescript(SCHEMA+'PRAGMA user_version=1;')
                db.execute('INSERT INTO admin VALUES(1,?)',(HASH,))
                db.execute('INSERT INTO projects VALUES(?,?,?,?,?,?)',('existing','Preserved',json.dumps(TARGET.data()),TARGET.digest(),'/home/fixture-box',3))
                db.execute('INSERT INTO operations VALUES(?,?,?,?,?,?,?,?,?,?)',('prior-op','existing',3,TARGET.digest(),'operator','unknown',100,160,None,'WORKER_LOST'))
                db.execute('INSERT INTO sessions VALUES(?,?,?)',('prior-session','csrf',1000))
            path.chmod(0o600);store=Store(path)
            self.assertEqual(store.project('existing')['revision'],3)
            self.assertEqual(store.receipt('prior-op')['code'],'WORKER_LOST')
            with store.db() as db:
                self.assertEqual(db.execute('PRAGMA user_version').fetchone()[0],10)
                self.assertEqual(db.execute('SELECT password FROM admin').fetchone()[0],HASH)
                self.assertEqual(db.execute('SELECT id FROM sessions').fetchone()[0],'prior-session')
            self.assertEqual(Graphs(Store(path)).list(),[])
    def test_conflicting_schema_migration_rolls_back(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'state.sqlite3'
            with sqlite3.connect(path) as db:db.executescript(SCHEMA+'CREATE TABLE graphs (unexpected TEXT); PRAGMA user_version=1;')
            path.chmod(0o600)
            with self.assertRaisesRegex(Fault,'STATE_UNAVAILABLE'):Store(path)
            with sqlite3.connect(path) as db:
                self.assertEqual(db.execute('PRAGMA user_version').fetchone()[0],1)
                self.assertIsNone(db.execute("SELECT name FROM sqlite_master WHERE name='graph_copies'").fetchone())

class GraphAPITests(unittest.TestCase):
    def test_protected_graph_operations_never_call_provider(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);store=Store(root/'state/db.sqlite3');store.admin(HASH)
            class Forbidden:
                mode='fixture'
                def observe(self,*_):raise AssertionError('Graph called infrastructure')
                def hello(self,*_):raise AssertionError('Graph executed infrastructure')
            service=Service(store,Forbidden());client=create_app(service,'http://localhost',root,secure=False).test_client()
            self.assertEqual(client.get('/api/graphs').status_code,401)
            origin={'Origin':'http://localhost'}
            login=client.post('/api/login',json={'password':PASSWORD},headers=origin)
            headers={**origin,'X-CSRF-Token':login.json['csrf']}
            self.assertEqual(client.post('/api/graphs',json={'document':document()},headers=origin).status_code,403)
            self.assertEqual(client.post('/api/graphs',json={'document':document()},headers={'Origin':'http://other','X-CSRF-Token':headers['X-CSRF-Token']}).status_code,403)
            r=client.post('/api/graphs',json={'document':document()},headers=headers);self.assertEqual(r.status_code,201);g=r.json
            self.assertEqual(client.get('/api/graphs').json['graphs'][0]['id'],g['id'])
            self.assertEqual(client.get('/api/graphs/'+g['id']).json,g)
            self.assertEqual(client.get('/api/graphs/'+g['id']+'/markdown').status_code,200)
            self.assertEqual(client.post('/api/graphs/'+g['id']+'/copies',json={'revision':1},headers=headers).status_code,200)
            self.assertEqual(client.put('/api/graphs/'+g['id'],json={'document':document(),'revision':1},headers=headers).status_code,200)
            self.assertEqual(client.put('/api/graphs/'+g['id'],json={'document':document(),'revision':1},headers=headers).status_code,409)
            self.assertEqual(client.delete('/api/graphs/'+g['id'],json={'revision':2},headers=headers).status_code,200)
            with store.db() as db:self.assertEqual(db.execute('SELECT count(*) FROM operations').fetchone()[0],0)
