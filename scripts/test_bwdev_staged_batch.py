"""Staged batch and compact append export regression."""
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "dev"))
import staged_batch as batch


class StagedBatchTests(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.home = Path(tmp.name)
        self.root = self.home / "repo"
        self.root.mkdir()
        self.git("init", "-q")
        self.doc = sorted(batch.APPEND_DOCS)[0]
        self.write(self.doc, "base " * 200 + "\n")
        self.git("add", ".")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "commit", "-qm", "base")
        self.parent = self.git("rev-parse", "HEAD").decode().strip()
        self.write(self.doc, "base " * 200 + "\nnew\n")
        self.git("add", self.doc)

    def git(self, *args):
        return batch.git(self.root, *args)

    def write(self, name, text):
        p = self.root / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(text)

    def test_append_exact_and_smaller(self):
        parent, tree = batch.snapshot(self.root)
        full = batch.export(self.root, parent, tree, compact=False)
        small = batch.export(self.root, parent, tree)
        row = small["rows"][0]
        self.assertEqual(row["append"], "new\n")
        old = self.git("cat-file", "blob", row["base_blob"]).decode()
        self.assertEqual(old + row["append"], full["rows"][0]["content"])
        self.assertLess(len(batch.encoded(small)), len(batch.encoded(full)))

    def test_nonappend_full_and_deletion_refusal(self):
        self.write(self.doc, "replacement")
        self.git("add", self.doc)
        p, t = batch.snapshot(self.root)
        self.assertEqual(batch.export(self.root, p, t)["rows"][0]["content"], "replacement")
        (self.root / self.doc).unlink()
        self.git("add", "-u", self.doc)
        with self.assertRaises(batch.BatchError):
            batch.export(self.root, *batch.snapshot(self.root))

    def test_dirty_and_wrong_tree_refused(self):
        p, t = batch.snapshot(self.root)
        self.write("untracked", "keep me")
        with self.assertRaises(batch.BatchError):
            batch.snapshot(self.root)
        (self.root / "untracked").unlink()
        with self.assertRaises(batch.BatchError):
            batch.run(self.root, p, "0" * 40, self.home / "wrong", [], [], 10)
        self.assertFalse((self.home / "wrong").exists())

    def test_go_scope_required(self):
        self.write("internal/state/x.go", "package state\n")
        self.git("add", "internal/state/x.go")
        payload = batch.export(self.root, *batch.snapshot(self.root))
        with self.assertRaises(batch.BatchError):
            batch.commands(self.root, payload, [], [])
        with self.assertRaises(batch.BatchError):
            batch.commands(self.root, payload, ["./internal/../state"], [])
        names = [x[0] for x in batch.commands(self.root, payload, ["./internal/state"], [])]
        self.assertEqual(names, ["diff-check", "go-test", "go-vet", "format"])


    def test_batch_pass_and_exclusive_evidence(self):
        p, t = batch.snapshot(self.root)
        directory = self.home / "proof"
        with patch.dict(batch.CHECKS, {"format": ("python", "-c", "print('fixture')")}):
            result, code = batch.run(self.root, p, t, directory, [], [], 10)
            self.assertEqual(code, 0)
            self.assertEqual(result["export"]["tree"], t)
            self.assertEqual(result["native_ci"], "NOT_ESTABLISHED")
            self.assertEqual(json.loads((directory / "record.json").read_text())["status"], "LOCAL_STAGED_PASS")
            with self.assertRaises(FileExistsError):
                batch.run(self.root, p, t, directory, [], [], 10)

    def test_failure_no_export(self):
        p, t = batch.snapshot(self.root)
        with patch.dict(batch.CHECKS, {"format": ("python", "-c", "raise SystemExit(7)")}):
            result, code = batch.run(self.root, p, t, self.home / "failure", [], [], 10)
        self.assertEqual(code, 1)
        self.assertEqual(result["checks"][-1]["exit_code"], 7)
        self.assertNotIn("export", result)

    def test_mid_check_source_change_refused(self):
        p, t = batch.snapshot(self.root)
        with patch.dict(batch.CHECKS, {"format": ("python", "-c", "open('unexpected','w').write('x')")}):
            result, code = batch.run(self.root, p, t, self.home / "drift", [], [], 10)
        self.assertEqual(code, 1)
        self.assertEqual(result["checks"][-1]["status"], "SOURCE_CHANGED")
        self.assertNotIn("export", result)
        self.assertEqual((self.root / "unexpected").read_text(), "x")
