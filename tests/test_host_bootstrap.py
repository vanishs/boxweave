"""The host entry must stay inert for help and invalid release digests."""
from pathlib import Path
import hashlib
import email.message
import io
import json
import os
import re
import subprocess
import sys
import unittest
import urllib.error
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / 'install.sh'

class BootstrapSafetyTests(unittest.TestCase):
    def test_no_args_reject_nonroot_before_download(self):
        p = subprocess.run(['bash', str(SCRIPT)], capture_output=True, text=True, timeout=5)
        self.assertEqual(p.returncode, 2)
        self.assertIn('DEFAULT_INSTALL_REQUIRES_ROOT', p.stderr)
        self.assertEqual(p.stdout, '')

    def test_shell_syntax(self):
        self.assertEqual(subprocess.run(['bash','-n',str(SCRIPT)], capture_output=True).returncode,0)
    def test_help_has_no_host_writes(self):
        p = subprocess.run(['bash',str(SCRIPT),'--help'],capture_output=True,text=True,timeout=5)
        self.assertEqual(p.returncode,0)
        self.assertIn('plan|apply',p.stdout)
    def test_missing_or_invalid_digest_never_reaches_install(self):
        p = subprocess.run(['bash',str(SCRIPT),'apply','--app-user','nobody',
                            '--archive','/tmp/not-a-release','--sha256','bad',
                            '--config','/tmp/not-config'],capture_output=True,text=True,timeout=5)
        self.assertEqual(p.returncode,2)
        self.assertIn('SHA256',p.stderr)

    def test_package_alias_is_recognized_without_executing(self):
        p = subprocess.run(['bash', str(SCRIPT), 'apply', '--app-user', 'nobody',
                            '--package', '/tmp/nonexistent-boxweave-package',
                            '--sha256', 'bad', '--config', '/tmp/nonexistent-config'],
                           capture_output=True, text=True, timeout=5)
        self.assertEqual(p.returncode, 2)
        self.assertIn('SHA256', p.stderr)

    def test_missing_distribution_digest_rejected_without_execution(self):
        p = subprocess.run(['bash',str(SCRIPT),'apply','--app-user','nobody',
                            '--archive','/tmp/not-a-release','--sha256','0'*64,
                            '--config','/tmp/not-config'],capture_output=True,text=True,timeout=5)
        self.assertEqual(p.returncode,2)
        self.assertIn('distribution-sha256',p.stderr)

    def test_version_selector_rejects_bad_input_without_download(self):
        for args, expected in (
            (['--version'], 'Missing value'),
            (['--version','v0.25.1'], 'Invalid --version'),
            (['--version','../0.25.1'], 'Invalid --version'),
            (['--version','0.25.1','--version','0.25.2'], 'Duplicate --version'),
            (['--version','0.25.1','--other'], 'Unknown streamed installer option'),
        ):
            with self.subTest(args=args):
                run = subprocess.run(['bash',str(SCRIPT),*args],
                                     capture_output=True,text=True,timeout=5)
                self.assertEqual(run.returncode,2)
                self.assertIn(expected,run.stderr)
                self.assertEqual(run.stdout,'')
        if os.geteuid() != 0:
            valid=subprocess.run(['bash',str(SCRIPT),'--version','0.25.1'],
                                 capture_output=True,text=True,timeout=5)
            self.assertEqual(valid.returncode,2)
            self.assertIn('DEFAULT_INSTALL_REQUIRES_ROOT',valid.stderr)

    def test_streamed_version_is_bound_to_exact_verified_release_offline(self):
        source=SCRIPT.read_text()
        body=re.search(r"<<'PY_DEFAULT_RELEASE'\n(.*?)\nPY_DEFAULT_RELEASE",source,re.S)
        self.assertIsNotNone(body)
        code=compile(body.group(1),str(SCRIPT)+':streamed','exec')
        published=(ROOT/'PUBLISHED_RELEASE').read_text().strip()
        pin=re.search(r'python3 - "(0\.[0-9]+\.[0-9]+)" "\$\{streamed_options\[@\]\}"',source)
        self.assertIsNotNone(pin)
        self.assertEqual(pin.group(1),published)

        manifest_source='a'*40
        payload=b"reviewed-example-installer-placeholder"
        for requested in ('0.25.1','0.25.2',''):
            with self.subTest(requested=requested):
                version=requested or published
                prefix='https://github.com/vanishs/boxweave/releases/download/v'+version+'/'
                api='https://api.github.com/repos/vanishs/boxweave'
                manifest={'format':1,'version':version,'source':manifest_source,
                          'files':{'online.py':hashlib.sha256(payload).hexdigest()}}
                metadata=json.dumps(manifest).encode()
                info={'tag_name':'v'+version,'draft':False,'prerelease':False,'assets':[{
                    'name':'release.json','digest':'sha256:'+hashlib.sha256(metadata).hexdigest(),
                    'browser_download_url':prefix+'release.json'},
                    {'name':'online.py','digest':'sha256:'+hashlib.sha256(payload).hexdigest(),
                     'browser_download_url':prefix+'online.py'}]}
                mapping={
                    api+'/releases/tags/v'+version:json.dumps(info).encode(),
                    prefix+'release.json':metadata,
                    api+'/git/ref/tags/v'+version:json.dumps({
                        'object':{'type':'commit','sha':manifest_source}}).encode(),
                    prefix+'online.py':payload,
                }
                urls=[]
                def fetch(request,timeout):
                    urls.append(request.full_url)
                    return io.BytesIO(mapping[request.full_url])
                fake=subprocess.CompletedProcess(['python'],0)
                with (patch.dict(os.environ,{'BOXWEAVE_REQUESTED_VERSION':requested}),
                      patch.object(sys,'argv',['-',published,'--provider-url','http://127.0.0.1:8080']),
                      patch('urllib.request.urlopen',side_effect=fetch),
                      patch('subprocess.run',return_value=fake) as launched):
                    with self.assertRaises(SystemExit) as status:
                        exec(code,{'__name__':'__main__'})
                self.assertEqual(status.exception.code,0)
                self.assertEqual(urls[0],api+'/releases/tags/v'+version)
                self.assertEqual(urls[-1],prefix+'online.py')
                command=launched.call_args.args[0]
                self.assertEqual(command[command.index('--version')+1],version)
                self.assertEqual(command[-2:],['--provider-url','http://127.0.0.1:8080'])

    def test_streamed_504_uses_digest_checked_api_asset(self):
        source=SCRIPT.read_text()
        body=re.search(r"<<'PY_DEFAULT_RELEASE'\n(.*?)\nPY_DEFAULT_RELEASE",source,re.S)
        self.assertIsNotNone(body)
        code=compile(body.group(1),'streamed','exec')
        version='0.46.0'
        api='https://api.github.com/repos/vanishs/boxweave'
        prefix='https://github.com/vanishs/boxweave/releases/download/v'+version+'/'
        payload=b'reviewed-streamed-helper'
        manifest={'format':1,'version':version,'source':'a'*40,
                  'files':{'online.py':hashlib.sha256(payload).hexdigest()}}
        metadata=json.dumps(manifest).encode()
        release={'tag_name':'v'+version,'draft':False,'prerelease':False,
                 'assets':[{'name':name,'id':aid,'url':api+'/releases/assets/'+str(aid),
                            'digest':'sha256:'+hashlib.sha256(data).hexdigest(),
                            'browser_download_url':prefix+name}
                           for name,aid,data in [('release.json',777,metadata),('online.py',778,payload)]]}
        mapping={api+'/releases/tags/v'+version:json.dumps(release).encode(),
                 prefix+'release.json':metadata,
                 api+'/git/ref/tags/v'+version:json.dumps({'object':{'type':'commit','sha':'a'*40}}).encode(),
                 api+'/releases/assets/778':payload}
        class AssetResponse(io.BytesIO):
            status=200
            headers=email.message.Message()
        AssetResponse.headers['Content-Type']='application/octet-stream'
        for corrupt in (False,True):
            with self.subTest(corrupt=corrupt):
                seen=[]
                def fetch(request,timeout):
                    seen.append((request.full_url,request.headers.get('Accept')))
                    if request.full_url==prefix+'online.py':
                        raise urllib.error.HTTPError(request.full_url,504,'Gateway Timeout',None,None)
                    data=mapping[request.full_url]
                    return AssetResponse(data+b'tampered' if corrupt and request.full_url.endswith('/778') else data)
                with (patch.dict(os.environ,{'BOXWEAVE_REQUESTED_VERSION':version}),
                      patch.object(sys,'argv',['-','0.43.1']),
                      patch('urllib.request.urlopen',side_effect=fetch),
                      patch('subprocess.run',return_value=subprocess.CompletedProcess(['python'],0)) as launched):
                    with self.assertRaises(SystemExit) as status:
                        exec(code,{'__name__':'__main__'})
                self.assertEqual(status.exception.code,2 if corrupt else 0)
                self.assertIn((api+'/releases/assets/778','application/octet-stream'),seen)
                if corrupt: launched.assert_not_called()
                else: launched.assert_called_once()

    def test_streamed_version_mismatch_fails_without_launch(self):
        source=SCRIPT.read_text()
        body=re.search(r"<<'PY_DEFAULT_RELEASE'\n(.*?)\nPY_DEFAULT_RELEASE",source,re.S)
        self.assertIsNotNone(body)
        code=compile(body.group(1),'streamed','exec')
        manifest={'format':1,'version':'0.25.1','source':'a'*40,
                  'files':{'online.py':'0'*64}}
        metadata=json.dumps(manifest).encode()
        prefix='https://github.com/vanishs/boxweave/releases/download/v0.25.2/'
        info={'tag_name':'v0.25.2','draft':False,'prerelease':False,'assets':[{
            'name':'release.json','digest':'sha256:'+hashlib.sha256(metadata).hexdigest(),
            'browser_download_url':prefix+'release.json'}]}
        def fetch(request,timeout):
            return io.BytesIO(json.dumps(info).encode() if request.full_url.endswith('/releases/tags/v0.25.2') else metadata)
        with (patch.dict(os.environ,{'BOXWEAVE_REQUESTED_VERSION':'0.25.2'}),
              patch.object(sys,'argv',['-','0.25.1']),
              patch('urllib.request.urlopen',side_effect=fetch),
              patch('subprocess.run') as launched):
            with self.assertRaises(SystemExit) as status:
                exec(code,{'__name__':'__main__'})
        self.assertEqual(status.exception.code,2)
        launched.assert_not_called()

if __name__ == '__main__':
    unittest.main()
