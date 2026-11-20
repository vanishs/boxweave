#!/usr/bin/env python3
"""Self-contained temporary-Git regression checks for the CI change-scope decision."""
from __future__ import annotations

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from ci_change_scope import docs_only

CHECKER = Path(__file__).with_name("ci_change_scope.py")


class ChangeScopeTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.git("init", "-q")
        self.git("config", "user.name", "CI Test")
        self.git("config", "user.email", "ci@example.invalid")
        self.write("README.md", "baseline\n")
        self.git("add", ".")
        self.git("commit", "-qm", "base")
        self.base = self.git("rev-parse", "HEAD")

    def git(self, *args: str) -> str:
        return subprocess.check_output(["git", *args], cwd=self.repo, text=True).strip()

    def write(self, path: str, content: str) -> None:
        target = self.repo / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")

    def commit(self, path: str) -> None:
        self.git("add", path)
        self.git("commit", "-qm", "test change")

    def classify(self, event: str = "push", ref: str = "refs/heads/feat/v1-test",
                 before: str | None = None, base_ref: str = "") -> dict[str, str]:
        result = self.repo / "output.txt"
        result.write_text("", encoding="utf-8")
        env = dict(os.environ, CI_EVENT_NAME=event, CI_REF=ref,
                   CI_BEFORE_SHA=self.base if before is None else before,
                   CI_BASE_REF=base_ref, GITHUB_OUTPUT=str(result))
        proc = subprocess.run([sys.executable, str(CHECKER)], cwd=self.repo,
                              env=env, text=True, capture_output=True, check=True)
        self.assertIn("CI_CHANGE_SCOPE=", proc.stdout)
        return dict(line.split("=", 1) for line in result.read_text().splitlines())

    def test_docs_directory_non_document_inputs_require_full(self) -> None:
        for name in ("docs/example.py", "docs/policy.json", "docs/image.svg"):
            with self.subTest(name=name):
                self.write(name, "not textual documentation\n")
                self.commit(name)
                self.assertEqual(self.classify()["full"], "true")

    def test_source_to_document_rename_requires_full(self) -> None:
        self.write("old.go", "package demo\n")
        self.commit("old.go")
        before = self.git("rev-parse", "HEAD")
        self.git("mv", "old.go", "new.md")
        self.git("commit", "-qm", "rename")
        self.assertEqual(self.classify(before=before)["full"], "true")

    def test_shared_local_classifier(self) -> None:
        self.assertTrue(docs_only(["README.md", "docs/manual.rst"]))
        self.assertFalse(docs_only([]))
        self.assertFalse(docs_only(["docs/image.svg"]))
        self.assertFalse(docs_only([".github/guide.md"]))

    def test_markdown_and_docs_skip_expensive_jobs(self) -> None:
        self.write("docs/acceptance.md", "PASS scoped\n")
        self.commit("docs/acceptance.md")
        self.assertEqual(self.classify()["full"], "false")

    def test_readme_only_skips_expensive_jobs(self) -> None:
        self.write("README.md", "updated\n")
        self.commit("README.md")
        self.assertEqual(self.classify()["reason"], "DOCS_ONLY")

    def test_source_change_requires_full(self) -> None:
        self.write("internal/new.go", "package internal\n")
        self.commit("internal/new.go")
        self.assertEqual(self.classify()["full"], "true")

    def test_workflow_change_requires_full(self) -> None:
        self.write(".github/workflows/ci.yml", "name: CI\n")
        self.commit(".github/workflows/ci.yml")
        self.assertEqual(self.classify()["full"], "true")

    def test_missing_push_base_fails_closed(self) -> None:
        self.write("README.md", "updated\n")
        self.commit("README.md")
        self.assertEqual(self.classify(before="0" * 40)["full"], "true")

    def test_protected_branch_runs_full_even_for_docs(self) -> None:
        self.write("README.md", "updated\n")
        self.commit("README.md")
        self.assertEqual(self.classify(ref="refs/heads/v1")["full"], "true")

    def test_release_branch_runs_full_even_for_docs(self) -> None:
        self.write("README.md", "updated\n")
        self.commit("README.md")
        self.assertEqual(self.classify(ref="refs/heads/release/v1.0.0-rc.8")["full"], "true")

    def test_manual_dispatch_runs_full(self) -> None:
        self.write("README.md", "updated\n")
        self.commit("README.md")
        self.assertEqual(self.classify(event="workflow_dispatch")["full"], "true")

    def test_pull_request_includes_entire_diff_not_last_commit(self) -> None:
        self.git("update-ref", "refs/remotes/origin/v1", self.base)
        self.write("internal/changed.go", "package internal\n")
        self.commit("internal/changed.go")
        self.write("README.md", "last commit is docs\n")
        self.commit("README.md")
        self.assertEqual(self.classify(event="pull_request", ref="refs/pull/42/merge",
                                       base_ref="v1")["full"], "true")

    def test_pull_request_docs_only_is_light(self) -> None:
        self.git("update-ref", "refs/remotes/origin/v1", self.base)
        self.write("README.md", "changed docs\n")
        self.commit("README.md")
        self.assertEqual(self.classify(event="pull_request", ref="refs/pull/42/merge",
                                       base_ref="v1")["full"], "false")


if __name__ == "__main__":
    unittest.main()
