"""Failure navigation cannot turn missing test names or compile errors into PASS."""
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "dev"))
import go_failure_index as index


class FailureIndexTests(unittest.TestCase):
    def test_comparison_and_package_errors(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            for file, tests in (("a_test.go", ["TestKeep", "TestAmbiguous"]),
                                ("b_test.go", ["TestAmbiguous"])):
                (root / file).write_text("package p\n" + "".join(
                    f"func {name}(t *testing.T) {{}}\n" for name in tests))
            subprocess.run(["git", "-C", str(root), "add", "."], check=True)
            old = root / "old.log"
            old.write_text("--- FAIL: TestGone (0s)\n--- FAIL: TestKeep (0s)\n")
            log = root / "new.log"
            log.write_text("--- FAIL: TestKeep (0s)\n    --- FAIL: TestKeep/sub (0s)\n"
                           "--- FAIL: TestAmbiguous (0s)\n--- FAIL: TestMissing (0s)\n"
                           "--- FAIL: TestKeep (0s)\nFAIL\tpkg/a\t0.1s\nFAIL\n"
                           "FAIL\tpkg/b [build failed]\nFAIL\n")
            result = index.summarize(root, log, old)
            for expected in ("top_level_failure_lines=4 unique_names=3",
                             "package_failure_records=pkg/a,pkg/b",
                             "a_test.go (1): TestKeep",
                             "AMBIGUOUS:a_test.go,b_test.go (1): TestAmbiguous",
                             "MISSING (1): TestMissing", "TestGone: MISSING",
                             "NOT proof of PASS"):
                self.assertIn(expected, result)
            self.assertNotIn("TestKeep/sub", result)
            log.write_text("FAIL\tpkg/b [build failed]\nFAIL\n")
            result = index.summarize(root, log, old)
            self.assertIn("top_level_failure_lines=0", result)
            self.assertIn("package_failure_records=pkg/b", result)
            self.assertIn("not acceptance", result)

    def test_batch_keeps_exit_evidence_and_detects_drift(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            root = base / "repo"
            root.mkdir()
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            source = root / "a_test.go"
            source.write_text("package p\nfunc TestKeep(t *testing.T) {}\n")
            subprocess.run(["git", "-C", str(root), "add", "."], check=True)
            subprocess.run(["git", "-C", str(root), "-c", "user.name=Test",
                            "-c", "user.email=test@example.invalid", "commit",
                            "-qm", "fixture"], check=True)
            bindir = base / "bin"
            bindir.mkdir()
            go = bindir / "go"
            go.write_text("#!/bin/sh\nprintf '%s\\n' '--- FAIL: TestKeep (0s)' >&2\nexit 1\n")
            go.chmod(0o700)
            with patch.dict(os.environ, {"PATH": str(bindir) + os.pathsep + os.environ["PATH"]}):
                first = base / "first"
                self.assertEqual(index.run_batch(root, ["./"], None, first), 1)
                result = json.loads((first / "result.json").read_text())
                self.assertEqual(result["guest_test_exit"], 1)
                self.assertFalse(result["source_drift"])
                self.assertIn("TestKeep", (first / "failure-index.txt").read_text())
                with self.assertRaises(FileExistsError):
                    index.run_batch(root, ["./"], None, first)
                with self.assertRaises(ValueError):
                    index.run_batch(root, ["-exec"], None, base / "bad")
                with self.assertRaises(ValueError):
                    index.run_batch(root, ["./"], None, root / "bad")
                go.write_text("#!/bin/sh\nprintf changed >> a_test.go\nexit 0\n")
                second = base / "second"
                self.assertEqual(index.run_batch(root, ["./"], None, second), 2)
                self.assertTrue(json.loads((second / "result.json").read_text())["source_drift"])
                go.write_text("#!/bin/sh\nexit 0\n")
                third = base / "third"
                self.assertEqual(index.run_batch(root, ["./"], None, third), 2)
                self.assertFalse(json.loads((third / "result.json").read_text())["reported_run"])
                go.write_text("#!/bin/sh\nprintf '%s\\n' '=== RUN   TestKeep' '--- PASS: TestKeep (0s)'\n")
                fourth = base / "fourth"
                self.assertEqual(index.run_batch(root, ["./"], None, fourth), 0)
                self.assertEqual(json.loads((fourth / "result.json").read_text())["reported_pass"], ["TestKeep"])
