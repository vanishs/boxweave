import fcntl,hashlib,json,os,sqlite3,subprocess,tempfile,unittest,zipfile
from pathlib import Path
from unittest.mock import patch
from bwmini import distribution as d

class DistributionTests(unittest.TestCase):
 def setUp(self):
  self.tmp=tempfile.TemporaryDirectory();self.root=Path(self.tmp.name);self.prefix=self.root/'installed';self.source=self.root/'source';self.wheels=self.root/'wheels';self.wheels.mkdir();(self.wheels/'fake.whl').write_bytes(b'fixture-wheel')
  for n in ['bwmini/__main__.py','bwmini/deploy_check.py','web/dist/index.html','requirements.lock']:
   p=self.source/n;p.parent.mkdir(parents=True,exist_ok=True);p.write_text('fixture')
  (self.source/'VERSION').write_text('0.9.0\n');self.archive=self.root/'release.zip'
  self.info=d.build(self.source,self.wheels,self.archive,'f'*40,'modified')
  self.db=self.root/'state.sqlite3'
  with sqlite3.connect(self.db) as c:c.execute('create table retained(value text)');c.execute('insert into retained values (?)',('nondefault-project',))
  self.config=self.root/'input.json';self.settings={'mode':'mcp','host':'127.0.0.1','port':18981,'origin':'https://localhost:18982','database':str(self.db),'launcher':['/usr/bin/ssh','trusted-profile'],'operator_extension':{'keep':42}}
  self.config.write_text(json.dumps(self.settings));self.raw=self.config.read_bytes()
  def env_create(_,path):
   p=Path(path)/'bin/python';p.parent.mkdir(parents=True,exist_ok=True);p.write_text('fixture executable; never run')
  self.env=patch.object(d.venv.EnvBuilder,'create',env_create);self.env.start()
  self.run=patch.object(d.subprocess,'run',side_effect=lambda *a,**k:subprocess.CompletedProcess(a[0],0,json.dumps({'mcp':{'state':'authenticated_read_passed'},'admin_initialized':True}),''));self.run.start()
 def tearDown(self):self.run.stop();self.env.stop();self.tmp.cleanup()
 def install(self,**kw):return d.install(self.archive,self.info['sha256'],self.prefix,self.config,quiesced=True,**kw)
 def test_digest_rejected_before_prefix_or_state_change(self):
  before=self.db.read_bytes()
  with self.assertRaisesRegex(d.InstallError,'DIGEST'):d.install(self.archive,'0'*64,self.prefix,self.config,quiesced=True)
  self.assertFalse(self.prefix.exists());self.assertEqual(self.db.read_bytes(),before)
 def test_manifest_does_not_allow_path_escape_even_with_reviewed_archive_digest(self):
  p=self.root/'hostile.zip'
  with zipfile.ZipFile(p,'w') as z:z.writestr('../outside',b'bad');z.writestr('manifest.json',b'{}')
  with self.assertRaisesRegex(d.InstallError,'PATH'):d.verify(p,d.digest(p.read_bytes()))
  self.assertFalse((self.root/'outside').exists())
 def test_symlink_archive_member_is_rejected(self):
  p=self.root/'link.zip';i=zipfile.ZipInfo('bwmini/escape');i.external_attr=(0o120777<<16)
  with zipfile.ZipFile(p,'w') as z:z.writestr(i,'/etc/passwd');z.writestr('manifest.json',b'{}')
  with self.assertRaisesRegex(d.InstallError,'PATH'):d.verify(p,d.digest(p.read_bytes()))
 def test_install_rerun_preserves_settings_and_consistent_database_backup(self):
  first=self.install();current=(self.prefix/'current').resolve();second=self.install()
  self.assertEqual((self.prefix/'current').resolve(),current);self.assertEqual((self.prefix/'config.json').read_bytes(),self.raw)
  self.assertFalse((self.prefix/'update-in-progress.json').exists());self.assertEqual(first['state'],'installed_services_not_activated')
  with sqlite3.connect(second['backup']) as c:self.assertEqual(c.execute('select value from retained').fetchone()[0],'nondefault-project')
  self.assertEqual(json.loads((self.prefix/'runtime.json').read_text())['operator_extension'],{'keep':42})
 def test_operator_keys_in_runtime_refuse_update_until_moved_to_config(self):
  self.install()
  runtime=json.loads((self.prefix/'runtime.json').read_text());runtime['lifecycle']={'allowed_names':['demo-1']}
  (self.prefix/'runtime.json').write_text(json.dumps(runtime))
  with self.assertRaisesRegex(d.InstallError,'OPERATOR_RUNTIME_KEYS_REQUIRE_CONFIG_JSON:lifecycle'):
   d.install(self.archive,self.info['sha256'],self.prefix,None,quiesced=True)
  saved=json.loads((self.prefix/'config.json').read_text());saved['lifecycle']=runtime['lifecycle']
  (self.prefix/'config.json').write_text(json.dumps(saved))
  d.install(self.archive,self.info['sha256'],self.prefix,None,quiesced=True)
  self.assertEqual(json.loads((self.prefix/'runtime.json').read_text())['lifecycle'],{'allowed_names':['demo-1']})
 def test_config_conflict_refuses_publication(self):
  self.install();before=os.readlink(self.prefix/'current');self.config.write_text(json.dumps(self.settings|{'port':18983}))
  with self.assertRaisesRegex(d.InstallError,'CONFIG_CONFLICT'):self.install()
  self.assertEqual(os.readlink(self.prefix/'current'),before)
 def test_concurrent_install_is_not_allowed(self):
  self.prefix.mkdir(mode=0o700)
  with (self.prefix/'.install.lock').open('a') as lock:
   fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
   with self.assertRaisesRegex(d.InstallError,'INSTALL_BUSY'):self.install()
  self.assertFalse((self.prefix/'update-in-progress.json').exists())
 def test_interruption_before_atomic_pointer_keeps_old_release_and_repair_works(self):
  self.install();old=os.readlink(self.prefix/'current');(self.source/'VERSION').write_text('0.9.1\n');self.archive=self.root/'next.zip';self.info=d.build(self.source,self.wheels,self.archive,'e'*40,'modified')
  replace=d.os.replace
  def fail(src,dst):
   if Path(dst)==self.prefix/'current':raise OSError('injected publication interruption')
   return replace(src,dst)
  with patch.object(d.os,'replace',side_effect=fail):
   with self.assertRaises(OSError):self.install()
  self.assertEqual(os.readlink(self.prefix/'current'),old);self.assertTrue((self.prefix/'update-in-progress.json').exists())
  self.install();self.assertNotEqual(os.readlink(self.prefix/'current'),old);self.assertFalse((self.prefix/'update-in-progress.json').exists())
 def test_postpublication_failure_retains_marker_until_verified_repair(self):
  def fail(argv,**kw):
   return subprocess.CompletedProcess(argv,2 if '--mcp' in argv else 0,json.dumps({'mcp':{'state':'authenticated_read_passed'}}),'')
  with patch.object(d.subprocess,'run',side_effect=fail):
   with self.assertRaisesRegex(d.InstallError,'MCP_READ_FAILED'):self.install()
  self.assertTrue((self.prefix/'current').is_symlink());self.assertEqual(json.loads((self.prefix/'update-in-progress.json').read_text())['stage'],'published')
  self.install();self.assertFalse((self.prefix/'update-in-progress.json').exists())
 def test_changed_installed_file_is_not_silently_adopted(self):
  self.install();(self.prefix/'current/bwmini/__main__.py').write_text('tampered')
  with self.assertRaisesRegex(d.InstallError,'INSTALLED_FILE_CHANGED'):self.install()
 def test_downgrade_requires_consistent_backup_restore(self):
  self.install();(self.source/'VERSION').write_text('0.8.0\n');self.archive=self.root/'old.zip';self.info=d.build(self.source,self.wheels,self.archive,'e'*40,'modified')
  with self.assertRaisesRegex(d.InstallError,'DOWNGRADE'):self.install()
 def test_no_activation_without_explicit_quiesce(self):
  with self.assertRaisesRegex(d.InstallError,'QUIESCE_REQUIRED'):d.install(self.archive,self.info['sha256'],self.prefix,self.config)
 def test_health_timer_and_oneshot_states_are_distinct(self):
  self.assertTrue(d.unit_ready({'ActiveState':'inactive','Result':'success','ExecMainStatus':'0'},oneshot=True))
  self.assertFalse(d.unit_ready({'ActiveState':'inactive','Result':'exit-code','ExecMainStatus':'1'},oneshot=True))
  good={'ActiveState':'active','SubState':'waiting','UnitFileState':'enabled'}
  self.assertTrue(d.unit_ready(good,timer=True));self.assertFalse(d.unit_ready(good|{'UnitFileState':'disabled'},timer=True));self.assertFalse(d.unit_ready({},timer=True))
 def test_units_escape_paths_and_do_not_start_provider(self):
  content=d.unit_text(Path('/tmp/a space%p'),Path('/tmp/a space%p/runtime.json'),'serve').decode()
  self.assertIn('Description=BoxWeave serve\n',content);self.assertNotIn('Description=BoxWeave v2',content)
  self.assertIn('%%p',content);self.assertIn(r'WorkingDirectory=/tmp/a\x20space%%p/current',content);self.assertNotIn('containarium',content);self.assertNotIn('sudo',content)
 def test_builder_excludes_unreviewed_private_client_and_runtime_files(self):
  for name in ['clients/private-profile.json','.runtime/session.json']:
   p=self.source/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text('synthetic-do-not-distribute')
  output=self.root/'public-only.zip';d.build(self.source,self.wheels,output,'f'*40,'modified')
  with zipfile.ZipFile(output) as z:
   self.assertNotIn('clients/private-profile.json',z.namelist());self.assertTrue(all(b'synthetic-do-not-distribute' not in z.read(n) for n in z.namelist()))
 def test_first_install_requires_explicit_local_administrator_setup(self):
  with patch.object(d.subprocess,'run',return_value=subprocess.CompletedProcess([],0,json.dumps({'mcp':{'state':'authenticated_read_passed'},'admin_initialized':False}),'')):
   with self.assertRaisesRegex(d.InstallError,'ADMIN_SETUP_REQUIRED'):self.install()
  self.assertTrue((self.prefix/'current').is_symlink());self.assertTrue((self.prefix/'update-in-progress.json').exists())
  self.install();self.assertFalse((self.prefix/'update-in-progress.json').exists())
 def test_401_is_reachability_not_auth_and_failed_service_is_not_starting(self):
  import urllib.error
  self.install()
  with patch.object(d.urllib.request,'urlopen',side_effect=urllib.error.HTTPError('http://localhost',401,'Unauthorized',{},None)):
   value=d.health(self.prefix);self.assertTrue(value['web']['reachable']);self.assertFalse(value['web']['authenticated']);self.assertEqual(value['mcp']['state'],'not_checked')
   state=json.loads((self.prefix/'installed.json').read_text());state['units']=['owned.service'];(self.prefix/'installed.json').write_text(json.dumps(state))
   with patch.object(d,'properties',return_value={'ActiveState':'failed','Result':'exit-code'}):self.assertEqual(d.health(self.prefix)['state'],'failed')
   (self.prefix/'update-in-progress.json').write_text('{}');self.assertEqual(d.health(self.prefix)['state'],'incomplete')
 def test_health_emits_typed_unit_states_and_schema(self):
  import urllib.error
  self.install()
  state=json.loads((self.prefix/'installed.json').read_text())
  state['units']=['boxweave-v2-fixture01-serve.service','boxweave-v2-fixture01-worker.service']
  (self.prefix/'installed.json').write_text(json.dumps(state))
  with patch.object(d,'properties',return_value={'ActiveState':'active','SubState':'running','UnitFileState':'enabled'}), \
       patch.object(d.urllib.request,'urlopen',side_effect=OSError('fixture: no listener')):
   value=d.health(self.prefix)
  self.assertEqual(value['diagnostic_schema'],1)
  self.assertEqual(value['unit_states'],{'serve':'ready','worker':'ready'})
  self.assertEqual(value['state'],'starting')
  with patch.object(d,'properties',return_value={'ActiveState':'inactive'}), \
       patch.object(d.urllib.request,'urlopen',side_effect=OSError('fixture: no listener')):
   value=d.health(self.prefix)
  self.assertEqual(value['unit_states']['worker'],'inactive')
  self.assertEqual(value['state'],'starting')
 def test_health_mcp_request_never_ready_on_probe_or_admin_failure(self):
  self.install()
  for outcome in (('not json',''),
                  (json.dumps({'mcp':{'state':'authenticated_read_passed'}}),''),
                  (json.dumps({'mcp':'broken','admin_initialized':True}),''),
                  (json.dumps({'mcp':{'state':'authenticated_read_passed'},'admin_initialized':False}),'')):
   with self.subTest(outcome=outcome[0][:24]):
    with patch.object(d.subprocess,'run',return_value=subprocess.CompletedProcess([],0,outcome[0],outcome[1])):
     value=d.health(self.prefix,mcp=True)
    self.assertEqual(value['state'],'failed')
    self.assertIn(value.get('code'),('MCP_PROBE_INVALID','ADMIN_SETUP_REQUIRED'))
 def test_health_malformed_mcp_output_is_typed_not_generic(self):
  self.install()
  with patch.object(d.subprocess,'run',side_effect=subprocess.TimeoutExpired([],5)):
   value=d.health(self.prefix,mcp=True)
  self.assertEqual(value['state'],'failed');self.assertEqual(value['code'],'HEALTH_UNAVAILABLE')
 def test_origin_credentials_paths_and_insecure_cookie_are_rejected(self):
  for changed in [{'origin':'https://user:password@example.test'},{'origin':'https://example.test/path'},{'origin':'http://example.test','secure_cookie':False},{'secure_cookie':False}]:
   self.config.write_text(json.dumps(self.settings|changed))
   with self.assertRaises(d.InstallError):self.install()
   self.assertFalse(self.prefix.exists())
 def test_quiesce_failed_inactive_unit_has_no_running_process_to_stop(self):
  with patch.object(d,'properties',return_value={'ActiveState':'inactive','MainPID':'0'}),patch.object(d,'systemctl') as control:
   d.stop_owned_unit('owned.service');control.assert_not_called()
  with patch.object(d,'properties',return_value={'ActiveState':'active','MainPID':'123'}),patch.object(d,'systemctl') as control:
   d.stop_owned_unit('owned.service');control.assert_called_once_with('stop','owned.service')
  with patch.object(d,'properties',return_value={}):
   with self.assertRaisesRegex(d.InstallError,'STATE_UNAVAILABLE'):d.stop_owned_unit('owned.service')
 def test_local_reset_invalidates_session_without_provider_call(self):
  from bwmini.store import Store
  from bwmini.domain import Target
  from bwmini.provider import Fixture
  from bwmini.service import Service
  from bwmini.web import create_app
  from werkzeug.security import generate_password_hash
  store=Store(self.root/'auth.sqlite3');store.admin(generate_password_hash('synthetic-old-password'))
  provider=Fixture(Target('containarium','dev','default','test','test-container','123','test','test-container'));service=Service(store,provider)
  client=create_app(service,'http://localhost',self.root,secure=False).test_client()
  response=client.post('/api/login',json={'password':'synthetic-old-password'},headers={'Origin':'http://localhost'})
  self.assertEqual(response.status_code,200);self.assertEqual(client.get('/api/projects').status_code,200)
  store.reset_admin(generate_password_hash('synthetic-new-password'))
  self.assertEqual(client.get('/api/projects').status_code,401)
  self.assertNotEqual(client.post('/api/login',json={'password':'synthetic-old-password'},headers={'Origin':'http://localhost'}).status_code,200)
  self.assertEqual(client.post('/api/login',json={'password':'synthetic-new-password'},headers={'Origin':'http://localhost'}).status_code,200)
  self.assertEqual(provider.calls,[])
 def test_real_systemd_parser_accepts_units_in_space_containing_prefix(self):
  prefix=self.root/'a space';release=prefix/'current';binary=release/'.venv/bin/python';binary.parent.mkdir(parents=True);binary.symlink_to('/usr/bin/python3')
  p=self.root/'verify-package.service';p.write_bytes(d.unit_text(prefix,prefix/'runtime.json','serve'))
  # Bypass this test class's fake subprocess runner only for a read-only parser.
  import os
  code=os.spawnvp(os.P_WAIT,'systemd-analyze',['systemd-analyze','verify',str(p)])
  self.assertEqual(code,0)
 def test_config_rejects_external_listener_and_state_in_release(self):
  for value in [self.settings|{'host':'0.0.0.0'},self.settings|{'database':str(self.prefix/'releases/bad/state.sqlite3')}]:
   self.config.write_text(json.dumps(value))
   with self.assertRaises(d.InstallError):self.install()
if __name__=='__main__':unittest.main()
