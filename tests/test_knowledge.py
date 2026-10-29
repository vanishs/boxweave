"""Maintenance gates detect broken handoff/authority/traceability rather than trusting prose."""
import json,tempfile,unittest
from pathlib import Path
from scripts.check_knowledge import audit,REQUIRED_ROLES
class KnowledgeTests(unittest.TestCase):
 def setUp(self):
  self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup);self.root=Path(self.tmp.name);docs=self.root/'docs';(docs/'process').mkdir(parents=True)
  self.authority={k:'docs/'+k+'.md' for k in sorted(REQUIRED_ROLES)}
  cold={'goals':'product','boundaries':'scope','run':'install','design':'architecture','tests':'tests','risks':'risks','legacy':'legacy','state':'status'}
  self.registry={'authority':self.authority,'cold_start':cold}
  for role,name in self.authority.items():(self.root/name).write_text('# '+role+'\n')
  (self.root/'docs/requirements.md').write_text('| REQ-001 | outcome |\n')
  (self.root/'docs/acceptance.md').write_text('## AC-001 — criterion\n')
  (self.root/'docs/traceability.md').write_text('| REQ-001 / AC-001 | test |\n')
  (self.root/'AGENTS.md').write_text('[Index](docs/INDEX.md)\n');(self.root/'README.md').write_text('# Start\n')
  self.index=self.root/'docs/INDEX.md';self.index.write_text('\n'.join('['+k+']('+k+'.md)' for k in sorted(REQUIRED_ROLES)))
  self.write()
 def write(self):(self.root/'docs/process/DOCUMENT_AUTHORITY.json').write_text(json.dumps(self.registry))
 def test_complete_public_handoff_routes_pass(self):
  result=audit(self.root);self.assertEqual(result['result'],'passed',result['errors']);self.assertEqual(len(result['cold_start_routes']),8)
 def test_broken_link_does_not_pass(self):
  with (self.root/'docs/install.md').open('a') as f:f.write('[missing](not-present.md)')
  self.assertTrue(any('Broken local link' in e for e in audit(self.root)['errors']))
 def test_duplicate_authority_does_not_pass(self):
  self.registry['authority']['tasks']=self.authority['status'];self.write()
  self.assertIn('Duplicate authority file',audit(self.root)['errors'])
 def test_new_requirement_without_acceptance_trace_does_not_pass(self):
  with (self.root/'docs/requirements.md').open('a') as f:f.write('| REQ-002 | newly added |\n')
  self.assertIn('Requirement/acceptance traceability gap',audit(self.root)['errors'])
 def test_cold_start_missing_route_and_outside_link_do_not_pass(self):
  self.index.write_text(self.index.read_text().replace('[install](install.md)','')+'\n[outside](../../outside.md)')
  errors=audit(self.root)['errors'];self.assertIn('Cold-start route missing: run',errors);self.assertTrue(any('Link leaves repository' in e for e in errors))
if __name__=='__main__':unittest.main()
