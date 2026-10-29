"""Public Web client setup is data generation, not enrollment or authority expansion."""
import copy,hashlib,json
from test_slice import Base,HASH,PASSWORD
from bwmini.web import create_app
from bwmini.profiles import generate
from bwmini.domain import Fault
DATA={'profile':'codex/windows/native','agent_executable':'C:/Tools/codex.exe','agent_core':'','agent_version':'codex-cli 0.165.0-alpha.1.2','profile_directory':'C:/profiles/中文 profile','host':'192.168.2.233','port':18447,'user':'bw-v2-mcp-nested','identity_file':'C:/Keys/id_ed25519','known_hosts':'C:/Keys/known_hosts','server_key':'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBHd0xW02aHQFLqGddjcelVuXWYtsp/O1VqcXTrrzOa9','transport_executable':'C:/Windows/System32/OpenSSH/ssh.exe','distribution':'','builtin_path':'','builtin_sha256':'','replace':False}
class PublicProfileTests(Base):
 def test_all_four_profiles_hashes_and_replacement_intent_survive_artifacts(self):
  for agent in ('codex','zcode'):
   for transport in ('native','wsl'):
    d=copy.deepcopy(DATA);d['profile']=agent+'/windows/'+transport;d['replace']=True
    if agent=='zcode':d.update(agent_executable='C:/Tools/node.exe',agent_core='C:/ZCode/zcode.cjs',agent_version='0.17.0',builtin_path='C:/ZCode/public.json',builtin_sha256='a'*64)
    if transport=='wsl':d.update(transport_executable='C:/Windows/System32/wsl.exe',distribution='Ubuntu',identity_file='/home/user/中文 key/id_ed25519',known_hosts='/home/user/中文 key/known_hosts')
    v=generate(d);descriptor=json.loads(v['descriptor_text']);self.assertEqual(descriptor,v['descriptor']);self.assertEqual(v['descriptor_sha256'],hashlib.sha256(v['descriptor_text'].encode()).hexdigest())
    self.assertIn(v['descriptor_sha256'],v['enrollment_script']);self.assertIn(descriptor['source_sha256'],v['enrollment_script']);self.assertIn('--replace',v['enrollment_script']);self.assertEqual(v['execution'],'not_performed')
  self.assertNotIn('--replace',generate(DATA)['enrollment_script'])
 def test_malformed_secret_unknown_fields_and_wrong_runtime_are_refused(self):
  for changed in [{'private_key':'SYNTHETIC_SECRET'},{'server_key':'-----BEGIN OPENSSH PRIVATE KEY-----'},{'host':'host;run'},{'agent_version':'api_key=SYNTHETIC_SECRET'},{'port':True},{'profile_directory':'relative'},{'profile':'codex/windows/wsl'},{'user':'root'},{'replace':1}]:
   with self.assertRaises(Fault):generate(DATA|changed)
 def test_endpoint_auth_origin_csrf_exact_input_and_zero_provider_calls(self):
  self.store.admin(HASH);c=create_app(self.s,'http://localhost',self.root,secure=False).test_client();url='/api/client-profile'
  self.assertEqual(c.post(url,json=DATA,headers={'Origin':'http://localhost'}).status_code,401)
  token=c.post('/api/login',json={'password':PASSWORD},headers={'Origin':'http://localhost'}).json['csrf'];headers={'Origin':'http://localhost','X-CSRF-Token':token}
  self.assertEqual(c.post(url,json=DATA,headers={'Origin':'http://localhost'}).status_code,403)
  self.assertEqual(c.post(url,json=DATA,headers=headers).status_code,200)
  r=c.post(url,json=DATA|{'private_key':'SYNTHETIC_SECRET'},headers=headers);self.assertNotEqual(r.status_code,200);self.assertNotIn('SYNTHETIC_SECRET',r.get_data(as_text=True));self.assertEqual(self.fixture.calls,[])
