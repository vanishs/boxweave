"""Bounded Git observation and output."""
import json
import os
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


class HarnessError(Exception):
    """Actionable validation error."""


def git(root, *args):
    try:
        result = subprocess.run(
            ["git", "-C", str(root), *args], capture_output=True,
            text=True, encoding="utf-8", errors="replace", timeout=15,
            check=False, env=dict(os.environ, GIT_OPTIONAL_LOCKS="0"),
        )
    except (OSError, subprocess.TimeoutExpired):
        return None
    return result.stdout.rstrip("\n") if result.returncode == 0 else None


def observe(root):
    status = git(root, "status", "--porcelain=v1", "--untracked-files=normal")
    return {
        "commit": git(root, "rev-parse", "--verify", "HEAD"),
        "branch": git(root, "symbolic-ref", "--quiet", "--short", "HEAD"),
        "worktree": "unknown" if status is None else ("dirty" if status else "clean"),
        "base_commit": git(root, "rev-parse", "--verify", "refs/remotes/origin/v1"),
    }


def repo_path(root, path):
    if not isinstance(path, str) or not path or Path(path).is_absolute():
        raise HarnessError("Expected repository-relative path")
    candidate = (root / path).resolve()
    if not candidate.is_relative_to(root.resolve()):
        raise HarnessError("Path leaves repository")
    return candidate


def emit(data, as_json):
    if as_json:
        print(json.dumps(data, ensure_ascii=True, sort_keys=True, indent=2))
    else:
        for key, value in data.items():
            print(f"{key}: {json.dumps(value, ensure_ascii=True, sort_keys=True)}")
