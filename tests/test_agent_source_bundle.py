"""Non-network contract checks for the developer-only Agent source bundle helper."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[1]
HELPER = ROOT / "scripts" / "agent_source_bundle.py"


class AgentSourceBundleTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.home = Path(self.temp.name)
        self.repo = self.home / "repo"
        self.repo.mkdir()
        self.output = self.home / "bundle.zip"
        self.git("init", "-q")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("config", "user.name", "Fixture")
        (self.repo / "VERSION").write_text("0.26.0\n")
        (self.repo / "hello.py").write_text('print("original")\n')
        (self.repo / ".gitignore").write_text("ignored.txt\n")
        self.git("add", ".")
        self.git("commit", "-qm", "test fixture")

    def git(self, *args):
        return subprocess.run(["git", "-C", str(self.repo), *args],
                              check=True, capture_output=True, text=True).stdout.strip()

    def helper(self, *args):
        return subprocess.run([sys.executable, str(HELPER), *args],
                              capture_output=True, text=True, timeout=15)

    def pack(self):
        return self.helper("pack", "--repo", str(self.repo), "--output", str(self.output))

    def test_local_modified_and_untracked_code_is_the_exact_snapshot(self):
        (self.repo / "hello.py").write_text('print("modified")\n')
        (self.repo / "new_test.py").write_text("assert True\n")
        (self.repo / "ignored.txt").write_text("should-not-exist\n")
        result = self.pack()
        self.assertEqual(result.returncode, 0, result.stderr)
        meta = json.loads(result.stdout)
        self.assertEqual(meta["state"], "ready")
        self.assertEqual(meta["source"]["state"], "modified")
        self.assertEqual(meta["source"]["commit"], self.git("rev-parse", "HEAD"))
        self.assertEqual(meta["sha256"], hashlib.sha256(self.output.read_bytes()).hexdigest())
        with zipfile.ZipFile(self.output) as archive:
            self.assertEqual(archive.read("source/hello.py"), b'print("modified")\n')
            self.assertEqual(archive.read("source/new_test.py"), b"assert True\n")
            self.assertNotIn("source/ignored.txt", archive.namelist())
            self.assertFalse(any(name.startswith("source/.git/") for name in archive.namelist()))
            self.assertEqual(json.loads(archive.read("boxweave-source-manifest.json"))["source"],
                             meta["source"])
        checked = self.helper("verify", "--archive", str(self.output), "--sha256", meta["sha256"])
        self.assertEqual(checked.returncode, 0, checked.stderr)
        self.assertEqual(json.loads(checked.stdout)["state"], "verified")

    def test_clean_and_deleted_worktrees_are_distinguished(self):
        clean = self.pack()
        self.assertEqual(clean.returncode, 0, clean.stderr)
        self.assertEqual(json.loads(clean.stdout)["source"]["state"], "clean")
        self.output.unlink()
        (self.repo / "hello.py").unlink()
        removed = self.pack()
        self.assertEqual(removed.returncode, 0, removed.stderr)
        self.assertEqual(json.loads(removed.stdout)["source"]["state"], "modified")
        with zipfile.ZipFile(self.output) as archive:
            self.assertNotIn("source/hello.py", archive.namelist())

    def test_secret_filename_and_symlink_are_refused_without_bundle(self):
        (self.repo / ".env").write_text("DO_NOT_SHARE=fixture\n")
        result = self.pack()
        self.assertEqual(result.returncode, 2)
        self.assertFalse(self.output.exists())
        (self.repo / ".env").unlink()
        os.symlink("hello.py", self.repo / "alias.py")
        result = self.pack()
        self.assertEqual(result.returncode, 2)
        self.assertFalse(self.output.exists())

    def test_exclusive_output_and_explicit_digest(self):
        result = self.pack()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.pack().returncode, 2)
        self.assertEqual(self.helper("verify", "--archive", str(self.output),
                                     "--sha256", "0" * 64).returncode, 2)
        data = bytearray(self.output.read_bytes())
        data[-1] ^= 1
        self.output.write_bytes(data)
        self.assertEqual(self.helper("verify", "--archive", str(self.output),
                                     "--sha256", hashlib.sha256(bytes(data)).hexdigest()).returncode, 2)

    def test_output_inside_worktree_is_refused(self):
        target = self.repo / "inside.zip"
        self.assertEqual(self.helper("pack", "--repo", str(self.repo),
                                     "--output", str(target)).returncode, 2)
        self.assertFalse(target.exists())


if __name__ == "__main__":
    unittest.main()
