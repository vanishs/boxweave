import json
from pathlib import Path
import subprocess
import sys
from test_slice import Base,PASSWORD,HASH,TARGET
from bwmini.models import Models
from bwmini.domain import Fault
from bwmini.store import Store
from bwmini.web import create_app

SELECT={'client_version':'3.16.3','provider':'zai','model':'GLM-5.3','access_mode':'account'}
class ModelTests(Base):
    def setUp(self):super().setUp();self.models=Models(self.store)
    def test_save_reload_idempotence_and_no_provider_calls(self):
        first=self.models.save(SELECT,0);again=self.models.save(SELECT,first['revision'])
        self.assertEqual(first,again);self.assertEqual(Models(Store(self.store.path)).read(),first)
        self.assertEqual(first['status'],'native_setup_pending');self.assertEqual(first['inference'],'unverified')
        self.assertEqual(self.fixture.calls,[])
    def test_confirmation_requires_explicit_checks_and_is_only_operator_report(self):
        r=self.models.save(SELECT,0)
        for selected,checked in [(False,True),(True,False),(1,True)]:
            with self.assertRaisesRegex(Fault,'NATIVE_CHECK_REQUIRED'):self.models.confirm(r['revision'],selected,checked)
        r=self.models.confirm(r['revision'],True,True)
        self.assertEqual((r['status'],r['evidence'],r['inference']),('operator_reported_ready','operator_report','unverified'))
        self.assertEqual(self.models.confirm(r['revision'],True,True),r)
    def test_changed_selection_clears_completion_and_rejects_stale_editor(self):
        r=self.models.save(SELECT,0);r=self.models.confirm(r['revision'],True,True)
        changed=self.models.save(dict(SELECT,provider='bigmodel'),r['revision'])
        self.assertEqual(changed['status'],'native_setup_pending')
        with self.assertRaisesRegex(Fault,'MODEL_REVISION_CONFLICT'):self.models.save(SELECT,r['revision'])
        with self.assertRaisesRegex(Fault,'MODEL_REVISION_CONFLICT'):self.models.confirm(r['revision'],True,True)
    def test_actor_isolation(self):
        self.models.save(SELECT,0,'alice');self.assertIsNone(self.models.read('bob')['selection'])
        with self.assertRaisesRegex(Fault,'MODEL_SELECTION_REQUIRED'):self.models.confirm(0,True,True,'bob')
    def test_invalid_secret_fields_models_versions_revision_never_persist(self):
        for value in [dict(SELECT,api_key='SYNTHETIC_SECRET'),dict(SELECT,provider='SYNTHETIC_SECRET'),dict(SELECT,model='unknown'),dict(SELECT,access_mode='arbitrary'),dict(SELECT,client_version='unknown')]:
            with self.assertRaises(Fault):self.models.save(value,0)
        for revision in [True,-1,2**53,'1']:
            with self.assertRaises(Fault):self.models.save(SELECT,revision)
        self.assertEqual(self.models.read()['status'],'not_selected')
    def test_handoff_is_public_allowlisted_intent_no_inference_claim(self):
        with self.assertRaisesRegex(Fault,'MODEL_SELECTION_REQUIRED'):self.models.handoff()
        self.models.save(dict(SELECT,access_mode='api-key'),0)
        value=self.models.handoff();s=json.dumps(value)
        self.assertEqual(value['credentials'],'not_collected');self.assertEqual(value['mcp_connection'],'separate_check')
        self.assertNotIn('SYNTHETIC_SECRET',s);self.assertEqual(value['inference'],'unverified')
    def test_schema_four_upgrade_preserves_projects_and_lifecycle_state(self):
        project=self.bind()
        with self.store.db() as db:
            db.execute('DROP TABLE model_selections')
            for table in ('access_claims', 'access_events', 'access_operations', 'access_plans', 'access_routes', 'tenant_labels'):db.execute('DROP TABLE '+table)
            db.execute('DROP TABLE project_unbindings');db.execute('DROP TABLE job_runs')
            for table in ('diagnostic_events','diagnostic_trace_links','selftest_runs'):db.execute('DROP TABLE '+table)
            db.execute('DROP TABLE box_action_receipts');db.execute('DROP TABLE box_action_plans');db.execute('PRAGMA user_version=4')
        upgraded=Store(self.store.path)
        self.assertEqual(upgraded.project(project['id'])['revision'],1)
        self.assertEqual(Models(upgraded).read()['revision'],0)
        with upgraded.db() as db:self.assertEqual(db.execute('PRAGMA user_version').fetchone()[0],11)
    def test_http_auth_csrf_invalid_fields_and_secret_redaction(self):
        self.store.admin(HASH);c=create_app(self.s,'http://localhost',self.root,secure=False,clock=lambda:self.now[0]).test_client()
        self.assertEqual(c.get('/api/model-wizard').status_code,401)
        login=c.post('/api/login',json={'password':PASSWORD},headers={'Origin':'http://localhost'}).json
        h={'Origin':'http://localhost','X-CSRF-Token':login['csrf']}
        self.assertEqual(c.put('/api/model-wizard',json=dict(SELECT,revision=0),headers={'Origin':'http://localhost'}).status_code,403)
        bad=c.put('/api/model-wizard',json=dict(SELECT,revision=0,api_key='SYNTHETIC_SECRET'),headers=h)
        self.assertEqual(bad.status_code,400);self.assertNotIn('SYNTHETIC_SECRET',bad.text)
        self.assertEqual(c.put('/api/model-wizard',json=dict(SELECT,revision=0),headers=h).json['status'],'native_setup_pending')
        self.assertEqual(c.get('/api/model-wizard/handoff').json['inference'],'unverified')
        self.assertEqual(self.fixture.calls,[])
    def test_cli_selected_model_and_invalid_inputs_without_secret_echo(self):
        config=self.root/'config.json';config.write_text(json.dumps({'mode':'fixture','database':str(self.store.path),'target':TARGET.data()}))
        base=[sys.executable,'-m','bwmini']
        argv=['model-select','--config',str(config),'--revision','0','--client-version','3.16.3','--model-provider','bigmodel','--model-id','GLM-5.3','--access-mode','account']
        result=subprocess.run(base+argv,capture_output=True,text=True);self.assertEqual(result.returncode,0,result.stderr)
        self.assertEqual(json.loads(result.stdout)['selection']['provider'],'bigmodel')
        argv[argv.index('GLM-5.3')]='SYNTHETIC_SECRET';argv[argv.index('--revision')+1]='1'
        result=subprocess.run(base+argv,capture_output=True,text=True);self.assertNotEqual(result.returncode,0);self.assertNotIn('SYNTHETIC_SECRET',result.stdout+result.stderr)

    def test_existing_native_setup_never_assumes_account_or_api_key_mode(self):
        r=self.models.save(dict(SELECT,access_mode='native-existing'),0)
        value=self.models.handoff()
        self.assertIn('Keep the existing native ZCode connection',value['steps'][2])
        self.assertNotIn('API key',value['steps'][2])
        self.assertEqual(r['status'],'native_setup_pending')

    def test_model_cli_does_not_require_infrastructure_launcher(self):
        config=self.root/'models-only.json';config.write_text(json.dumps({'database':str(self.store.path),'mode':'mcp','launcher':['/not-installed']}))
        r=subprocess.run([sys.executable,'-m','bwmini','model-wizard','--config',str(config)],capture_output=True,text=True)
        self.assertEqual(r.returncode,0,r.stderr);self.assertEqual(json.loads(r.stdout)['status'],'not_selected')
