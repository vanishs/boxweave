#!/usr/bin/env python3
"""Regression tests for the local hook fast path and document byte checks."""
from __future__ import annotations

import io
import pathlib
import subprocess
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import check_format  # noqa: E402
import local_hook_scope as scope  # noqa: E402


HEAD = "b" * 40
V1 = "a" * 40
BASE = "c" * 40
REMOTE = "d" * 40
ZERO = "0" * 40


def pushed(filename: str, remote_sha: str = ZERO, ref: str = "refs/heads/docs/test") -> str:
    return f"refs/heads/docs/test {HEAD} {ref} {remote_sha}"


class LocalHookScopeTests(unittest.TestCase):
    def test_staged_docs_and_mixed_changes(self):
        with patch.object(scope, "changed_paths", return_value=["README.md", "docs/note.mdx"]):
            self.assertTrue(scope.staged_docs_only())
        for paths in ([], ["README.md", "internal/native/box.go"],
                      ["cmd/old.go", "docs/new.md"], ["docs/a.md", "go.mod"],
                      ["docs/a.md", ".github/workflows/quality-gate.yml"]):
            with self.subTest(paths=paths):
                with patch.object(scope, "changed_paths", return_value=paths):
                    self.assertFalse(scope.staged_docs_only())

    def simulate_push(self, paths: bytes, lines: list[str]) -> bool:
        def fake_git(*args: str) -> bytes:
            if args == ("rev-parse", "--verify", "refs/remotes/origin/v1"):
                return (V1 + "\n").encode()
            if args[:2] == ("merge-base", "--is-ancestor"):
                return b""
            if args == ("merge-base", HEAD, V1):
                return (BASE + "\n").encode()
            if args[:3] == ("diff", "--no-renames", "--name-only"):
                return paths
            raise AssertionError(f"unexpected git call {args!r}")
        with patch.object(scope, "git", side_effect=fake_git):
            return scope.push_docs_only("origin", lines)

    def test_new_docs_branch_uses_v1_tracking_ref(self):
        self.assertTrue(self.simulate_push(b"README.md\x00docs/note.md\x00", [pushed("README.md")]))

    def test_existing_docs_branch_uses_complete_pr_diff(self):
        self.assertTrue(self.simulate_push(b"docs/note.md\x00", [pushed("docs/note.md", REMOTE)]))

    def test_renamed_code_or_mixed_config_is_full(self):
        for names in (b"internal/old.go\x00docs/new.md\x00",
                      b"docs/note.md\x00.github/workflows/quality-gate.yml\x00",
                      b"docs/note.md\x00go.mod\x00", b""):
            with self.subTest(paths=names):
                self.assertFalse(self.simulate_push(names, [pushed("docs/note.md")]))

    def test_release_force_delete_or_unknown_remote_is_full(self):
        self.assertFalse(self.simulate_push(b"README.md\x00",
                                           [pushed("README.md", ref="refs/heads/release/v1.0.0-rc.9")]))
        self.assertFalse(self.simulate_push(b"README.md\x00",
                                           [pushed("README.md").replace(HEAD, ZERO)]))
        self.assertFalse(self.simulate_push(b"README.md\x00", []))
        self.assertFalse(scope.push_docs_only("https://git.example/repo", [pushed("README.md")]))
        with patch.object(scope, "git", side_effect=subprocess.CalledProcessError(1, ["git"])):
            with self.assertRaises(subprocess.CalledProcessError):
                scope.push_docs_only("origin", [pushed("README.md")])

    def test_multiple_pushed_refs_all_must_be_docs_only(self):
        lines = [pushed("a.md"), pushed("b.md")]
        self.assertTrue(self.simulate_push(b"a.md\x00", lines))
        self.assertFalse(self.simulate_push(b"src/main.go\x00", lines))


class DocumentFormattingTests(unittest.TestCase):
    def test_utf8_newlines_and_whitespace_are_mechanical(self):
        self.assertEqual([], check_format.check_file("docs/notes.md", "合法 Markdown\n".encode()))
        self.assertEqual([], check_format.check_file("docs/notes.mdx", b"# Head\n"))
        self.assertEqual([], check_format.check_file("docs/notes.rst", b"Heading\n"))
        self.assertTrue(check_format.check_file("docs/notes.md", b"\xff"))
        self.assertTrue(check_format.check_file("docs/notes.mdx", b"no final newline"))
        self.assertTrue(check_format.check_file("docs/notes.rst", b"one\r\n"))
        self.assertTrue(check_format.check_file("docs/notes.md", b"bad tab\t\n"))

    def test_paths0_checks_only_named_staged_files(self):
        stream = io.TextIOWrapper(io.BytesIO(b"README.md\x00docs/note.rst\x00"), encoding="utf-8")
        with patch.object(sys, "stdin", stream):
            with patch.object(sys, "argv", ["check_format.py", "--cached", "--paths0"]):
                with patch.object(check_format, "source_bytes", side_effect=lambda name, cached: b"ok\n") as read:
                    self.assertEqual(0, check_format.main())
                    self.assertEqual(2, read.call_count)
                    self.assertTrue(all(call.args[1] for call in read.call_args_list))


if __name__ == "__main__":
    unittest.main()
