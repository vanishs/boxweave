"""Conservative verification plan, fixed source-only checks, bounded evidence."""
import hashlib
import os
import re
import runpy
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

from .common import HarnessError, git, observe
from .records import latest, publish


CHECKS = {
    "bwdev": ("python", "dev/harness/check_runner.py", "test_bwdev*.py"),
    "scope": ("python", "dev/harness/check_runner.py", "test_ci_change_scope.py"),
    "recovery-fixture": ("python", "dev/harness/check_runner.py", "test_pr177_installed_recovery.py"),
    "format": ("python", "scripts/check_format.py"),
}


def changed_paths(root):
    base = git(root, "merge-base", "HEAD", "refs/remotes/origin/v1")
    if not base:
        return None, None
    results = [
        git(root, "diff", "--no-renames", "--name-only", "-z", base, "HEAD", "--"),
        git(root, "diff", "--cached", "--no-renames", "--name-only", "-z", "HEAD", "--"),
        git(root, "diff", "--no-renames", "--name-only", "-z", "--"),
        git(root, "ls-files", "--others", "--exclude-standard", "-z"),
    ]
    if any(result is None for result in results):
        return None, None
    return base, sorted({path for result in results for path in result.split("\x00") if path})


def plan(root):
    source = observe(root)
    base, paths = changed_paths(root)
    docs = False
    if paths:
        try:
            classify = runpy.run_path(str(root / "scripts/ci_change_scope.py"))["docs_only"]
            docs = bool(classify(paths))
        except (OSError, KeyError, TypeError, ValueError, SyntaxError):
            pass
    recommendations = ["scope", "format"]
    if paths is None or any(p.startswith("dev/") or p.startswith("scripts/test_bwdev") for p in paths):
        recommendations.append("bwdev")
    if paths is None or any("recover" in p or "lifecycle" in p for p in paths):
        recommendations.append("recovery-fixture")
    return {
        "status": "PLAN_ONLY", "source": source, "merge_base": base,
        "changed_paths": paths, "scope": "DOCS_ONLY_CANDIDATE" if docs else "FULL_OR_UNKNOWN",
        "recommended_local_checks": recommendations,
        "available_checks": sorted(CHECKS),
        "required_gate": "quality-gate on exact PR head; existing CI classifier is authoritative",
        "native_acceptance": "REVIEW_REQUIRED_BY_APPLICABLE_PRODUCT_CONTRACT",
        "note": "Never skips required CI, selects native effects, or authorizes a release. Local base can be stale.",
    }


def execute(root, check, directory, timeout, retain_log=False):
    if not 1 <= timeout <= 600:
        raise HarnessError("Timeout must be 1..600 seconds")
    directory = Path(directory)
    if directory.exists() or directory.is_symlink():
        raise HarnessError("Evidence directory already exists; inspect rather than rerun")
    # Reserve a fresh directory before running: concurrent callers must not both execute.
    try:
        directory.mkdir(mode=0o700)
    except FileExistsError as exc:
        raise HarnessError("Evidence directory already exists; inspect rather than rerun") from exc
    before = observe(root)
    command = [sys.executable, *CHECKS[check][1:]]
    result = "BLOCKED"
    code = None
    reason = None
    # Output is hashed without exposing logs or credentials in persisted evidence.
    with tempfile.TemporaryFile() as log:
        if check == "format" and not shutil.which("gofmt"):
            reason = "gofmt missing; repository format gate not runnable"
        elif not before["commit"]:
            reason = "Git source unavailable"
        else:
            try:
                proc = subprocess.run(command, cwd=root, stdout=log, stderr=subprocess.STDOUT,
                                      timeout=timeout, check=False,
                                      env=dict(os.environ, PYTHONDONTWRITEBYTECODE="1", GOTOOLCHAIN="local"))
                code = proc.returncode
                result = "PASS" if code == 0 else "FAIL"
            except subprocess.TimeoutExpired:
                result, reason = "UNKNOWN", "local test timed out"
            except OSError:
                reason = "cannot start registered check"
        log.seek(0)
        if retain_log:
            fd = os.open(directory / "output.log", os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(fd, "wb") as saved:
                shutil.copyfileobj(log, saved)
            log.seek(0)
        digest = hashlib.sha256()
        count = 0
        for chunk in iter(lambda: log.read(65536), b""):
            digest.update(chunk)
            count += len(chunk)
    after = observe(root)
    bound = bool(before["commit"] and before["commit"] == after["commit"]
                 and before["worktree"] == after["worktree"] == "clean")
    data = publish(directory / "record", 0, {
        "kind": "verification", "schema_version": 1, "check": check,
        "argv": list(CHECKS[check]), "source_before": before, "source_after": after,
        "result": result, "exit_code": code, "reason": reason,
        "output_sha256": digest.hexdigest(), "output_bytes": count,
        "binding": "CLEAN_HEAD_OBSERVED" if bound else "UNBOUND",
        "log_retained": retain_log,
    })
    return {
        **data, "status": "LOCAL_RESULT",
        "note": "Local recorded result, not an independent CI or native acceptance. Output retained locally only when explicitly requested.",
    }, 0 if result == "PASS" else 1


def inspect(root, directory):
    directory = Path(directory)
    data = latest(directory / "record" if (directory / "record").is_dir() else directory)
    if data.get("kind") != "verification" or data.get("schema_version") != 1:
        raise HarnessError("Unsupported verification record")
    check = data.get("check")
    result = data.get("result")
    code = data.get("exit_code")
    digest = data.get("output_sha256")
    count = data.get("output_bytes")
    if (not isinstance(check, str) or check not in CHECKS
            or result not in ("PASS", "FAIL", "BLOCKED", "UNKNOWN")
            or not isinstance(data.get("argv"), list)
            or not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest)
            or type(count) is not int or count < 0
            or (result == "PASS" and (type(code) is not int or code != 0))
            or (result == "FAIL" and (type(code) is not int or code == 0))
            or (result in ("BLOCKED", "UNKNOWN") and code is not None)):
        raise HarnessError("Invalid verification result")
    current = observe(root)
    before, after = data.get("source_before"), data.get("source_after")
    matched = (data["argv"] == list(CHECKS[check]) and isinstance(before, dict) and isinstance(after, dict)
               and data.get("binding") == "CLEAN_HEAD_OBSERVED"
               and current["commit"] and before.get("commit") == after.get("commit") == current["commit"]
               and before.get("worktree") == after.get("worktree") == current["worktree"] == "clean")
    if data.get("log_retained"):
        log = directory / "output.log"
        if not log.is_file() or log.is_symlink():
            matched = False
        else:
            h = hashlib.sha256()
            size = 0
            with log.open("rb") as stream:
                for chunk in iter(lambda: stream.read(65536), b""):
                    h.update(chunk)
                    size += len(chunk)
            matched = matched and h.hexdigest() == digest and size == count
    return {
        "status": "LOCAL_RECORD_MATCH" if matched else "REVIEW_REQUIRED",
        "check": data["check"], "result": data["result"], "source": current,
        "note": "Checks source identity only, not authenticity, current native state or CI acceptance.",
    }, 0 if matched else 1


def register(parser):
    sub = parser.add_subparsers(dest="action", required=True)
    sub.add_parser("plan")
    run = sub.add_parser("run", help="Run one explicitly registered source-only check")
    run.add_argument("check", choices=sorted(CHECKS))
    run.add_argument("directory", help="Fresh local evidence directory")
    run.add_argument("--timeout", type=int, default=120)
    run.add_argument("--retain-log", action="store_true", help="Keep potentially sensitive local output; do not publish blindly")
    sub.add_parser("inspect").add_argument("directory")


def run(args, root):
    if args.action == "plan":
        return plan(root), 0
    if args.action == "run":
        return execute(root, args.check, args.directory, args.timeout, args.retain_log)
    return inspect(root, args.directory)
