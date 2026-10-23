import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'server'))
import provenance


class ProvenanceTests(unittest.TestCase):
    def test_public_view_filters_unknown_fields_and_missing_old_install(self):
        with tempfile.TemporaryDirectory() as tmp:
            p=Path(tmp)/'manifest.json'
            self.assertEqual(provenance.read_public(p)['status'],'not_recorded')
            provenance.atomic_json(p,{'schema_version':1,'state':'complete','secret':'NEVER_RENDER',
               'source':{'repository':'example/project','requested_ref':'main','resolved_commit':'a'*40,'token':'NEVER_RENDER'},
               'installer':{'differs_from_source':True},'component_sha256':{'secret':'NEVER_RENDER'}})
            result=provenance.read_public(p)
            self.assertEqual(result['source']['resolved_commit'],'a'*40)
            self.assertTrue(result['installer']['differs_from_source'])
            self.assertNotIn('NEVER_RENDER',json.dumps(result))

    def test_build_hashes_actual_installer_and_binary(self):
        with tempfile.TemporaryDirectory() as tmp:
            p=Path(tmp);(p/'server').mkdir()
            (p/'VERSION').write_text('1.2.3\n');(p/'reference-install.sh').write_text('original')
            (p/'patched.sh').write_text('patched');(p/'mcp-server-linux-amd64').write_text('backend')
            with patch.dict('os.environ',{'ARCH':'amd64'}),patch.object(provenance,'version',return_value=None):
                result=provenance.build(p,p/'patched.sh','example/project','main','a'*40,'v1.0.0','/missing')
            self.assertTrue(result['installer']['differs_from_source'])
            self.assertEqual(result['state'],'pending')
            self.assertEqual(result['mcp_backend']['sha256'],provenance.digest(p/'mcp-server-linux-amd64'))
            self.assertIsNone(result['daemon']['reported_version'])
