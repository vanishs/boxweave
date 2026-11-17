#!/usr/bin/env python3
"""Conservatively identify documentation-only GitHub Actions changes.

This is a CI cost optimization, NOT a way to bypass the required quality-gate
status. Missing/uncertain comparison data falls back to the full checks.
The release branch always receives the full artifact/native release workflow.
"""
from __future__ import annotations

import json
import os
import pathlib
import re
import subprocess
import sys

DOC_SUFFIXES = (".md", ".mdx", ".rst")
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")


def docs_only(paths: list[str]) -> bool:
    """Return true only when every changed path is a non-executable document."""
    return bool(paths) and all(path.lower().endswith(DOC_SUFFIXES) for path in paths)


def git(*args: str) -> bytes:
    return subprocess.check_output(["git", *args], stderr=subprocess.PIPE)


def comparison_base() -> str | None:
    event_name = os.environ.get("GITHUB_EVENT_NAME", "")
    if event_name == "pull_request":
        base_ref = os.environ.get("GITHUB_BASE_REF", "")
        if not base_ref or base_ref.startswith("-"):
            return None
        return git("merge-base", "HEAD", "origin/" + base_ref).decode().strip()
    if event_name == "push":
        # Release branches must run their entire publish and provenance pipeline.
        if os.environ.get("GITHUB_REF", "").startswith("refs/heads/release/"):
            return None
        event_file = os.environ.get("GITHUB_EVENT_PATH", "")
        if not event_file:
            return None
        payload = json.loads(pathlib.Path(event_file).read_text(encoding="utf-8"))
        before = payload.get("before", "")
        if not isinstance(before, str) or not FULL_SHA.fullmatch(before):
            return None
        if before == "0" * 40:
            return None  # New branch: no safe comparison baseline.
        return before
    return None  # workflow_dispatch: always run the full check.


def main() -> int:
    only_docs = False
    try:
        base = comparison_base()
        if base and FULL_SHA.fullmatch(base):
            # Disable rename detection: a source -> .md rename must still show
            # the removed source filename and must NOT be treated as docs-only.
            changed = git("diff", "--no-renames", "--name-only", "-z", base, "HEAD", "--")
            paths = [p.decode("utf-8") for p in changed.split(b"\x00") if p]
            only_docs = docs_only(paths)
    except (OSError, ValueError, UnicodeError, subprocess.CalledProcessError) as exc:
        # Never silently mark a code change docs-only on a comparison failure.
        print(f"Change scope uncertain ({exc}); running full CI", file=sys.stderr)
    print(f"docs_only={'true' if only_docs else 'false'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
