#!/usr/bin/env python3
"""Safe docs-only GitHub Actions classification. Unverifiable changes run full CI."""
from __future__ import annotations

import os
import pathlib
import re
import subprocess


def git(*args: str) -> bytes:
    return subprocess.check_output(("git", *args), stderr=subprocess.DEVNULL)


def is_documentation(path: str) -> bool:
    # Workflow, source, installer and policy/config changes are ALWAYS full.
    if path.startswith(".github/"):
        return False
    return path.startswith("docs/") or path.endswith((".md", ".markdown", ".mdx"))


def classify() -> tuple[bool, str]:
    event = os.environ.get("CI_EVENT_NAME", "")
    ref = os.environ.get("CI_REF", "")
    if event == "workflow_dispatch":
        return True, "MANUAL_FULL"
    if ref == "refs/heads/v1" or ref.startswith("refs/heads/release/"):
        return True, "PROTECTED_OR_RELEASE_FULL"
    try:
        if event == "push":
            base = os.environ.get("CI_BEFORE_SHA", "")
            if not re.fullmatch(r"[a-fA-F0-9]{40}", base) or base == "0" * 40:
                return True, "MISSING_PUSH_BASE"
            git("cat-file", "-e", base + "^{commit}")
        elif event == "pull_request":
            target = os.environ.get("CI_BASE_REF", "")
            if not target or not re.fullmatch(r"[a-zA-Z0-9._/-]+", target) or ".." in target:
                return True, "MISSING_PR_BASE"
            base = git("merge-base", "HEAD", "origin/" + target).decode().strip()
        else:
            return True, "UNSUPPORTED_EVENT"
        paths = [item.decode("utf-8", "surrogateescape") for item in
                 git("diff", "--name-only", "-z", "--no-ext-diff", base, "HEAD").split(b"\0") if item]
        if not paths:
            return True, "EMPTY_OR_UNVERIFIABLE_DIFF"
        if all(is_documentation(path) for path in paths):
            return False, "DOCS_ONLY"
        return True, "SOURCE_OR_WORKFLOW_CHANGED"
    except (OSError, subprocess.CalledProcessError, UnicodeError):
        return True, "UNVERIFIABLE_DIFF"


def main() -> None:
    full, reason = classify()
    value = "true" if full else "false"
    print(f"CI_CHANGE_SCOPE={reason} full={value}")
    output = os.environ.get("GITHUB_OUTPUT")
    if output:
        with pathlib.Path(output).open("a", encoding="utf-8") as handle:
            handle.write(f"full={value}\nreason={reason}\n")


if __name__ == "__main__":
    main()
