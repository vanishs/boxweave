#!/usr/bin/env python3
"""Regression tests for the CI documentation-only classifier."""
from __future__ import annotations

import importlib.util
import pathlib
import unittest
from unittest.mock import patch

SCRIPT = pathlib.Path(__file__).with_name("ci_change_scope.py")
spec = importlib.util.spec_from_file_location("ci_change_scope", SCRIPT)
assert spec is not None and spec.loader is not None
scope = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scope)


class DocumentationScopeTest(unittest.TestCase):
    def test_only_markdown_and_textual_docs(self):
        self.assertTrue(scope.docs_only(["README.md", "docs/research/note.md"]))
        self.assertTrue(scope.docs_only(["docs/product/guide.mdx", "docs/manual.rst"]))

    def test_no_files_is_not_docs_only(self):
        self.assertFalse(scope.docs_only([]))

    def test_mixed_code_and_docs_is_full(self):
        self.assertFalse(scope.docs_only(["README.md", "internal/native/box.go"]))
        self.assertFalse(scope.docs_only(["docs/note.md", ".github/workflows/quality-gate.yml"]))
        self.assertFalse(scope.docs_only(["docs/image.svg", "README.md"]))
        self.assertFalse(scope.docs_only(["docs/structure.json"]))

    def test_release_and_manual_dispatch_never_skip(self):
        with patch.dict("os.environ", {
            "GITHUB_EVENT_NAME": "push",
            "GITHUB_REF": "refs/heads/release/v1.0.0-rc.9",
        }):
            self.assertIsNone(scope.comparison_base())
        with patch.dict("os.environ", {"GITHUB_EVENT_NAME": "workflow_dispatch"}):
            self.assertIsNone(scope.comparison_base())

    def test_first_branch_push_is_full(self):
        import json
        import tempfile
        with tempfile.TemporaryDirectory() as temp:
            event = pathlib.Path(temp, "event.json")
            event.write_text(json.dumps({"before": "0" * 40}), encoding="utf-8")
            with patch.dict("os.environ", {
                "GITHUB_EVENT_NAME": "push",
                "GITHUB_REF": "refs/heads/v1",
                "GITHUB_EVENT_PATH": str(event),
            }):
                self.assertIsNone(scope.comparison_base())


if __name__ == "__main__":
    unittest.main()
