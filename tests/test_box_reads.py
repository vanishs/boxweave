"""R-01: read-only Box commands are bounded and never follow guest symlinks."""
import base64
import io
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

from bwmini import box_reads

class ReadOnlyBoxTests(unittest.TestCase):
    def test_rejects_shell_metacharacters_absolute_and_parent_escape(self):
        for path in ("../secret", "/etc/passwd", "a/../../b", "a//b", "a/./b", "a\nb"):
            with self.subTest(path=path), self.assertRaises(ValueError):
                box_reads.verified_args("boxweave_box_read_file", {"box":"dev","path":path})
        for box in ("../dev","bad;rm","",42):
            with self.subTest(box=box), self.assertRaises(ValueError):
                box_reads.verified_args("boxweave_box_list_directory",{"box":box,"path":""})

    def test_rejects_unsafe_limit_or_extra_authority(self):
        for bad in ({"offset":True},{"max_bytes":2048},{"host":"127.0.0.1"},{"user":"root"}):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                box_reads.verified_args("boxweave_box_read_file", {"box":"dev","path":"ok", **bad})

    def guest(self, root, tool, path, **kw):
        v=box_reads.verified_args(tool,{"box":"dev","path":path,**kw})
        # Fixture-only rewritten *root*; production guest script fixes /home/user.
        code=box_reads._GUEST.replace('"/home/"+v["box"]','sys.argv[2]')
        data=base64.urlsafe_b64encode(json.dumps(v).encode()).decode()
        p=subprocess.run([sys.executable,"-c",code,data,str(root)],capture_output=True,text=True,timeout=10)
        return p

    def test_read_file_exact_bytes_and_offset(self):
        with tempfile.TemporaryDirectory() as folder:
            (Path(folder)/"a.txt").write_bytes(b"abcd\x00\xffxyz")
            p=self.guest(folder,"boxweave_box_read_file","a.txt",offset=2,max_bytes=3)
            self.assertEqual(p.returncode,0,p.stderr)
            out=json.loads(p.stdout)
            self.assertEqual(base64.b64decode(out["data"]),b"cd\x00")
            self.assertFalse(out["eof"])

    def test_list_bounded_and_symlink_refused(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder)
            for n in ("a","b","c"): (root/n).write_text(n)
            (root/"link").symlink_to("/etc")
            p=self.guest(folder,"boxweave_box_list_directory","",limit=2)
            self.assertEqual(p.returncode,0,p.stderr)
            d=json.loads(p.stdout)
            self.assertEqual([x["name"] for x in d["entries"]],["a","b"])
            self.assertEqual(d["next_offset"],2)
            self.assertNotEqual(self.guest(folder,"boxweave_box_read_file","link/passwd").returncode,0)
            self.assertNotEqual(self.guest(folder,"boxweave_box_list_directory","link").returncode,0)

    def test_opaque_provider_result_must_be_observed_exit_zero(self):
        with self.assertRaises(ValueError):
            box_reads.typed_output({"result":{"structuredContent":{"exit_code":4,"stdout":"{}"}}})
        good={"result":{"structuredContent":{"exit_code":0,"stdout":json.dumps({"entries":[],"next_offset":None})}}}
        self.assertEqual(box_reads.typed_output(good)["entries"],[])
        with self.assertRaises(ValueError):
            box_reads.typed_output({"result":{"isError":True,"content":[]}})

    def test_standalone_helper_import(self):
        with tempfile.TemporaryDirectory() as folder:
            script=Path(folder)/"box_reads.py"
            shutil.copyfile(Path(box_reads.__file__),script)
            result=subprocess.run([sys.executable,"-E","-s","-c",
                "import sys;sys.path.insert(0,sys.argv[1]);import box_reads;print(len(box_reads.TOOLS))",
                folder],capture_output=True,text=True,timeout=10)
            self.assertEqual(result.returncode,0,result.stderr)
            self.assertEqual(result.stdout.strip(),"2")
