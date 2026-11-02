"""BOI optimistic atomic guest patch: exact, bounded, no symlink, no replay."""
import base64
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from bwmini import box_patch

class PatchTests(unittest.TestCase):
    def setUp(self):
        self.folder=tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.root=Path(self.folder.name)
        self.path=self.root/"main.py"
        self.path.write_text("x = 1\n")
        self.sha=hashlib.sha256(self.path.read_bytes()).hexdigest()

    def verified(self, **changes):
        args={"box":"bwphase2-1102","path":"main.py",
              "expected_sha256":self.sha,"new_text":"x = 2\n",
              "request_key":"a"*32}
        args.update(changes)
        return box_patch.verify_patch(args)

    def guest(self,checked):
        # Only the test fixture substitutes the home directory; the source
        # helper always fixes /home/<verified target>.
        code=box_patch._GUEST_PATCH.replace('"/home/"+v["box"]','sys.argv[2]')
        payload=base64.urlsafe_b64encode(json.dumps(checked).encode()).decode()
        return subprocess.run([sys.executable,"-c",code,payload,str(self.root)],
                              capture_output=True,text=True,timeout=10)

    def test_atomic_replacement_readback(self):
        result=self.guest(self.verified())
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertEqual(self.path.read_text(),"x = 2\n")
        receipt=json.loads(result.stdout)
        self.assertEqual(receipt["sha256"],hashlib.sha256(b"x = 2\n").hexdigest())
        self.assertEqual(box_patch.parse_receipt({"result":{"structuredContent":{
            "exit_code":0,"stdout":result.stdout}}}),receipt)
        self.assertEqual(list(self.root.iterdir()),[self.path])

    def test_stale_refuses_without_changing(self):
        checked=self.verified(expected_sha256="f"*64)
        result=self.guest(checked)
        self.assertNotEqual(result.returncode,0)
        self.assertEqual(self.path.read_text(),"x = 1\n")

    def test_symlink_and_hardlink_refused(self):
        original=self.root/"original"
        original.write_text("x = 1\n")
        self.path.unlink()
        self.path.symlink_to(original)
        self.assertNotEqual(self.guest(self.verified()).returncode,0)
        self.assertEqual(original.read_text(),"x = 1\n")
        self.path.unlink()
        self.path.hardlink_to(original)
        self.assertNotEqual(self.guest(self.verified()).returncode,0)

    def test_directory_symlink_escape_refused(self):
        (self.root/"link").symlink_to("/tmp")
        checked=self.verified(path="link/anything")
        self.assertNotEqual(self.guest(checked).returncode,0)

    def test_invalid_inputs_rejected(self):
        for patch in ({"path":"../private"}, {"path":"/tmp/test"},
                      {"new_text":"x"*1025}, {"new_text":42},
                      {"expected_sha256":"bad"}, {"request_key":"a"*30},
                      {"exec":"ls"}, {"box":"../../root"}):
            with self.subTest(patch=patch),self.assertRaises(ValueError):
                self.verified(**patch)

    def test_failed_result_never_claimed_completed(self):
        for item in ({"result":{"structuredContent":{"exit_code":1,"stdout":"{}"}}},
                     {"result":{"isError":True}},
                     {"result":{"structuredContent":{"exit_code":True,"stdout":"{}"}}}):
            with self.assertRaises(ValueError):
                box_patch.parse_receipt(item)
