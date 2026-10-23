import json
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest
import subprocess

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'server'))
import jobs


@unittest.skipUnless(os.name=='posix','POSIX worker tests')
class JobTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)/'jobs'

    def execute(self, code, **kwargs):
        job=jobs.start([sys.executable,'-c',code],root=self.root,**kwargs)
        deadline=time.monotonic()+10
        while time.monotonic()<deadline:
            receipt=jobs.status(job['job_id'],self.root,logs=True)
            if receipt['state'] in jobs.TERMINAL:return receipt
            time.sleep(.03)
        self.fail('Worker did not finish')

    def test_receipt_survives_parent_and_preserves_nonzero_exit(self):
        result=self.execute('print("hello");raise SystemExit(7)')
        self.assertEqual(result['state'],'failed');self.assertEqual(result['exit_code'],7)
        self.assertEqual(result['logs']['stdout'],'hello\n')
        self.assertEqual(jobs.status(result['job_id'],self.root)['exit_code'],7)
        self.assertEqual((self.root/result['job_id']/'receipt.json').stat().st_mode&0o777,0o600)

    def test_deadline_and_output_limit_are_explicit(self):
        result=self.execute('import time;time.sleep(20)',timeout=.15)
        self.assertEqual(result['state'],'timed_out')
        result=self.execute('print("x"*1000000)',output_limit=1024)
        self.assertEqual(result['state'],'output_limit');self.assertTrue(result['output_truncated'])

    def test_worker_disappearance_is_unknown_not_success(self):
        p=jobs.private_directory(self.root/('a'*32))
        jobs.atomic_json(p/'receipt.json',{'state':'running','created_epoch':0,'exit_code':0})
        result=jobs.status('a'*32,self.root)
        self.assertEqual(result['state'],'unknown');self.assertIsNone(result['exit_code'])

    def test_secret_lines_are_redacted_across_chunks(self):
        p=Path(self.tmp.name)/'log'
        log=jobs.SafeLog(p)
        for chunk in (b'tok',b'en=PRIVATE\n',b'{"password": "JSON_SECRET"}\n',b'eyJabc.def.',b'ghi\n',b'-----BEGIN PRIVATE KEY-----\n',b'PRIVATE_DATA\n',b'-----END PRIVATE KEY-----\n'):
            log.feed(chunk)
        log.close(); text=p.read_text()
        for value in ('PRIVATE\n','eyJabc','PRIVATE_DATA','JSON_SECRET'):self.assertNotIn(value,text)

    def test_path_and_credential_arguments_rejected(self):
        with self.assertRaises(ValueError):jobs.job_path(self.root,'../other')
        with self.assertRaises(ValueError):jobs.start(['program','--password','synthetic'],root=self.root)

    def test_cli_submission_survives_submitter_exit(self):
        cli=Path(__file__).resolve().parents[1]/'server/cli.py'
        r=subprocess.run([sys.executable,'-B',str(cli),'job','start','--state-dir',str(self.root),
                          '--',sys.executable,'-c','print("from cli")'],capture_output=True,text=True,timeout=5)
        self.assertEqual(r.returncode,0,r.stderr)
        job=json.loads(r.stdout);deadline=time.monotonic()+5
        while time.monotonic()<deadline:
            result=jobs.status(job['job_id'],self.root,True)
            if result['state'] in jobs.TERMINAL:break
            time.sleep(.03)
        self.assertEqual(result['state'],'succeeded')
        self.assertEqual(result['logs']['stdout'],'from cli\n')
