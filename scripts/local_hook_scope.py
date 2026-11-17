#!/usr/bin/env python3
"""Conservative docs-only detection for the repository's local Git hooks.

pre-commit: classify *staged* files (unstaged changes do not grant an exemption).
pre-push: classify every pushed branch against the locally tracked remote v1
base; on any missing ref, shallow history, force push, mixed path or ambiguity,
return nonzero so the calling hook executes its original full verification.

GitHub's independent, required quality-gate remains the authoritative merge
check and uses its own up-to-date PR merge base.
"""
from __future__ import annotations

import re
import subprocess
import sys

from ci_change_scope import docs_only

SHA = re.compile(r"[0-9a-f]{40}\Z")
REMOTE = re.compile(r"[A-Za-z0-9_.-]+\Z")


def git(*args: str) -> bytes:
    return subprocess.check_output(("git", *args), stderr=subprocess.PIPE)


def changed_paths(*diff_args: str) -> list[str]:
    output = git("diff", "--no-renames", "--name-only", "-z", *diff_args, "--")
    return [p.decode("utf-8") for p in output.split(b"\x00") if p]


def staged_docs_only() -> bool:
    # A staged source-to-document rename is represented by a source deletion
    # and document addition; this must not be incorrectly exempted.
    return docs_only(changed_paths("--cached", "HEAD"))


def push_docs_only(remote: str, lines: list[str]) -> bool:
    if not REMOTE.fullmatch(remote) or not lines:
        return False
    remote_v1 = f"refs/remotes/{remote}/v1"
    base_head = git("rev-parse", "--verify", remote_v1).decode().strip()
    if not SHA.fullmatch(base_head):
        return False

    for line in lines:
        items = line.split()
        if len(items) != 4:
            return False
        local_ref, local_sha, remote_ref, remote_sha = items
        if not (local_ref.startswith("refs/heads/")
                and remote_ref.startswith("refs/heads/")
                and SHA.fullmatch(local_sha) and SHA.fullmatch(remote_sha)):
            return False
        if local_sha == "0" * 40 or remote_ref.startswith("refs/heads/release/"):
            return False
        if remote_sha != "0" * 40:
            # A force push is never eligible for the fast path.
            git("merge-base", "--is-ancestor", remote_sha, local_sha)

        # Compare the complete proposed PR branch with the local v1 base,
        # not just the latest pushed commit, to avoid skipping a mixed PR.
        baseline = git("merge-base", local_sha, base_head).decode().strip()
        if not SHA.fullmatch(baseline):
            return False
        if not docs_only(changed_paths(baseline, local_sha)):
            return False
    return True


def main() -> int:
    try:
        if len(sys.argv) == 2 and sys.argv[1] == "pre-commit":
            eligible = staged_docs_only()
        elif len(sys.argv) == 3 and sys.argv[1] == "pre-push":
            eligible = push_docs_only(sys.argv[2], sys.stdin.read().splitlines())
        else:
            return 1
        if eligible:
            print("Docs-only changes: lightweight local checks apply.")
            return 0
    except (OSError, ValueError, UnicodeError, subprocess.CalledProcessError) as exc:
        print(f"Local change scope uncertain ({exc}); using full checks", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
