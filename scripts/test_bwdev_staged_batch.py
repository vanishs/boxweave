"""Staged batch and compact append export regression."""
import hashlib
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

    def test_framed_export_exact_reconstruction_and_selective_reads(self):
        for i in range(8):
            self.write("file-%02d.txt" % i, "unicode: " + chr(0x6c34) * 80)
        self.git("add", ".")
        payload = batch.export(self.root, *batch.snapshot(self.root))
        raw = batch.framed_export(payload, 3, 2200)
        self.assertEqual(raw, batch.framed_export(payload, 3, 2200))
        header, body = raw.split(b"\n", 1)
        index = json.loads(header)
        self.assertEqual(index["body_bytes"], len(body))
        self.assertEqual(index["rows"], len(payload["rows"]))
        rebuilt, end = [], 0
        for number, frame in enumerate(index["frames"], 1):
            self.assertEqual(frame["number"], number)
            self.assertEqual(frame["offset"], end)
            self.assertLessEqual(frame["rows"], 3)
            self.assertLessEqual(frame["bytes"], 2200)
            part = body[end:end+frame["bytes"]]
            self.assertEqual(hashlib.sha256(part).hexdigest(), frame["sha256"])
            parsed = json.loads(part)
            for key in ("parent", "base", "tree"):
                self.assertEqual(parsed[key], payload[key])
            self.assertEqual(len(parsed["rows"]), frame["rows"])
            rebuilt.extend(parsed["rows"])
            end += frame["bytes"]
        self.assertEqual(end, len(body))
        result = dict(payload, rows=rebuilt)
        self.assertEqual(result, payload)
        self.assertEqual(hashlib.sha256(batch.encoded(result)).hexdigest(),
                         index["payload_sha256"])
        last = index["frames"][-1]
        offset = len(header)+1+last["offset"]
        self.assertEqual(raw[offset:offset+last["bytes"]], body[last["offset"]:])
        self.assertLess(last["bytes"], len(raw))

    def test_frame_bounds_and_duplicate_refusal(self):
        payload = batch.export(self.root, *batch.snapshot(self.root))
        for rows, size in ((0, 65536), (101, 65536), (1, 1023), (1, batch.MAX_EXPORT+1)):
            with self.assertRaises(batch.BatchError):
                batch.framed_export(payload, rows, size)
        with self.assertRaises(batch.BatchError):
            batch.framed_export(dict(payload, rows=payload["rows"]*2))
        self.write("large.txt", "x"*2000)
        self.git("add", "large.txt")
        with self.assertRaisesRegex(batch.BatchError, "Single row"):
            batch.framed_export(batch.export(self.root, *batch.snapshot(self.root)), 1, 1024)

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

    def commit(self, message):
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "commit", "-qm", message)
        return self.git("rev-parse", "HEAD").decode().strip()

    def committed_range(self):
        first = self.commit("first")
        self.write("docs/second.md", "second change\n")
        self.git("add", "docs/second.md")
        second = self.commit("second")
        head, tree = batch.snapshot(self.root)
        self.assertEqual(head, second)
        return first, second, tree

    def test_committed_batch_two_local_commits_exact_export(self):
        first, head, tree = self.committed_range()
        with patch.dict(batch.CHECKS, {"format": ("python", "-c", "print('fixture')")}):
            result, code = batch.run(self.root, self.parent, tree, self.home / "committed",
                                     [], [], 10, head)
        self.assertEqual(code, 0)
        self.assertEqual(result["status"], "LOCAL_COMMITTED_PASS")
        self.assertEqual(result["local_commits"], [first, head])
        self.assertEqual(result["binding"], "LOCAL_COMMIT_RANGE")
        self.assertEqual(result["export"]["parent"], self.parent)
        self.assertEqual(result["export"]["tree"], tree)
        self.assertEqual(result["checks"][0]["argv"], ["git", "diff", "--check", self.parent, head])
        self.assertEqual({r["path"] for r in result["export"]["rows"]}, {self.doc, "docs/second.md"})
        self.assertEqual(result["native_ci"], "NOT_ESTABLISHED")
        framed = batch.framed_export(result["export"], 1, 65536)
        header, body = framed.split(b"\n", 1)
        index = json.loads(header)
        self.assertEqual(index["tree"], tree)
        rows = []
        for frame in index["frames"]:
            raw = body[frame["offset"]:frame["offset"]+frame["bytes"]]
            self.assertEqual(hashlib.sha256(raw).hexdigest(), frame["sha256"])
            rows.extend(json.loads(raw)["rows"])
        self.assertEqual(rows, result["export"]["rows"])

    def test_committed_batch_refuses_dirty_stale_and_divergent(self):
        first, head, tree = self.committed_range()
        for parent, candidate, expected_tree in ((self.parent, first, tree),
                                                  (self.parent, head, "0"*40),
                                                  (head, head, tree)):
            with self.assertRaises(batch.BatchError):
                batch.committed_snapshot(self.root, parent, candidate, expected_tree)
        self.write("staged.txt", "do not lose")
        self.git("add", "staged.txt")
        with self.assertRaises(batch.BatchError):
            batch.committed_snapshot(self.root, self.parent, head, tree)

    def test_committed_batch_refuses_divergent_parent(self):
        first, head, tree = self.committed_range()
        self.git("checkout", "-q", "--detach", self.parent)
        self.write("divergence", "other history")
        self.git("add", "divergence")
        divergent = self.commit("divergent")
        self.git("checkout", "-q", "--detach", head)
        with self.assertRaises(batch.BatchError):
            batch.committed_snapshot(self.root, divergent, head, tree)

    def test_committed_batch_source_drift_and_failure_no_export(self):
        _, head, tree = self.committed_range()
        with patch.dict(batch.CHECKS, {"format": ("python", "-c", "raise SystemExit(7)")}):
            result, code = batch.run(self.root, self.parent, tree, self.home / "failed",
                                     [], [], 10, head)
        self.assertEqual(code, 1)
        self.assertNotIn("export", result)
        with patch.dict(batch.CHECKS, {"format": ("python", "-c", "open('drift','w').write('keep')")}):
            result, code = batch.run(self.root, self.parent, tree, self.home / "drift",
                                     [], [], 10, head)
        self.assertEqual(code, 1)
        self.assertEqual(result["checks"][-1]["status"], "SOURCE_CHANGED")
        self.assertNotIn("export", result)
        self.assertEqual((self.root / "drift").read_text(), "keep")
