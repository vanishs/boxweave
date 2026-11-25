"""Read-only exact-head CI navigation, not a substitute for GitHub rules."""
import json
import re
import shutil
import subprocess

from .common import HarnessError, observe
from .records import read_json

WORKFLOWS = {
    "quality": ".github/workflows/quality-gate.yml",
    "source": ".github/workflows/v1-s1-check.yml",
    "distribution": ".github/workflows/v1-release.yml",
    "installed": ".github/workflows/v1-n2-installed-lifecycle.yml",
}
SHA = re.compile(r"[0-9a-f]{40}")
REPO = re.compile(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+")


def identity(repo, sha):
    if not isinstance(repo, str) or not REPO.fullmatch(repo) or not isinstance(sha, str) or not SHA.fullmatch(sha):
        raise HarnessError("Expected owner/repository and exact 40-character head SHA")


def evaluate(data, repo, sha, required, pr=None, base_sha=None):
    identity(repo, sha)
    if (not isinstance(data, dict) or data.get("schema_version") != 1
            or data.get("repository") != repo or data.get("head_sha") != sha
            or not isinstance(data.get("runs"), list)):
        raise HarnessError("CI snapshot identity/schema mismatch")
    if (pr is None) != (base_sha is None) or (pr is not None and (type(pr) is not int or pr <= 0 or not isinstance(base_sha, str) or not SHA.fullmatch(base_sha))):
        raise HarnessError("Supply valid PR number and exact base SHA together")
    groups = {key: [] for key in required}
    other = 0
    for run in data["runs"]:
        if not isinstance(run, dict):
            raise HarnessError("Invalid CI run")
        if run.get("head_sha") != sha or run.get("event") != "pull_request":
            other += 1
            continue
        if pr is not None:
            associated = run.get("pull_requests")
            if not isinstance(associated, list) or not any(
                    isinstance(link, dict) and link.get("number") == pr
                    and isinstance(link.get("base"), dict) and link["base"].get("sha") == base_sha
                    and isinstance(link.get("head"), dict) and link["head"].get("sha") == sha
                    for link in associated):
                other += 1
                continue
        for key in required:
            if run.get("path") == WORKFLOWS[key]:
                groups[key].append(run)
    checks = []
    for key, runs in groups.items():
        # Multiple runs can refer to different PRs/base/attempts. Don't guess.
        row = {"check": key, "state": "MISSING" if not runs else "AMBIGUOUS", "runs": []}
        for run in runs:
            run_id = run.get("id")
            if type(run_id) is not int or run_id <= 0:
                raise HarnessError("Invalid run ID")
            row["runs"].append({"id": run_id, "status": run.get("status"),
                                "conclusion": run.get("conclusion"),
                                "url": f"https://github.com/{repo}/actions/runs/{run_id}"})
        if len(runs) == 1:
            run = runs[0]
            if run.get("status") != "completed":
                row["state"] = "PENDING"
            elif run.get("conclusion") == "success":
                row["state"] = "REPORTED_SUCCESS"
            elif run.get("conclusion") in ("failure", "timed_out", "action_required", "startup_failure"):
                row["state"] = "FAIL"
            elif run.get("conclusion") == "cancelled":
                row["state"] = "CANCELLED"
            else:
                row["state"] = "REVIEW_REQUIRED"
        checks.append(row)
    good = all(row["state"] == "REPORTED_SUCCESS" for row in checks)
    return {"status": "REPORTED_SUCCESS" if good else "REVIEW_REQUIRED", "repository": repo,
            "head_sha": sha, "pr": pr, "base_sha": base_sha, "checks": checks,
            "ignored_other_head_or_event": other,
            "note": "Run metadata only. Verify exact PR/base, job/artifact evidence and branch rules independently; a snapshot is not authority."}, 0 if good else 1


def live_snapshot(repo, sha):
    identity(repo, sha)
    if not shutil.which("gh"):
        raise HarnessError("gh unavailable; use authorized GitHub connector/exported run snapshot")
    endpoint = f"repos/{repo}/actions/runs?head_sha={sha}&event=pull_request&per_page=100"
    try:
        proc = subprocess.run(["gh", "api", "--hostname", "github.com", "--method", "GET", endpoint], capture_output=True,
                              text=True, timeout=30, check=False)
        if proc.returncode:
            raise HarnessError("GitHub read failed; inspect gh authentication externally")
        data = json.loads(proc.stdout)
    except (OSError, ValueError, subprocess.TimeoutExpired) as exc:
        raise HarnessError("GitHub read unavailable") from exc
    if not isinstance(data, dict) or not isinstance(data.get("workflow_runs"), list):
        raise HarnessError("Unexpected GitHub response")
    if type(data.get("total_count")) is not int or data["total_count"] > len(data["workflow_runs"]):
        raise HarnessError("Incomplete run list; use a complete authorized snapshot")
    return {"schema_version": 1, "repository": repo, "head_sha": sha, "runs": data["workflow_runs"]}


def inspect(root, repo, sha, snapshot=None, scope="full", pr=None, base_sha=None):
    identity(repo, sha)
    current = observe(root)
    if current["commit"] != sha:
        raise HarnessError("CI requested SHA differs from checkout; inspect original SHA separately")
    required = list(WORKFLOWS) if scope == "full" else ["quality"]
    data = read_json(snapshot) if snapshot else live_snapshot(repo, sha)
    result, code = evaluate(data, repo, sha, required, pr, base_sha)
    result["provenance"] = "CALLER_SNAPSHOT" if snapshot else "GH_API_READ"
    result["scope"] = scope
    return result, code


def register(parser):
    parser.add_argument("--repo", required=True)
    parser.add_argument("--sha", required=True)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--snapshot")
    source.add_argument("--live", action="store_true")
    parser.add_argument("--scope", choices=("full", "quality-only"), default="full")
    parser.add_argument("--pr", type=int)
    parser.add_argument("--base-sha")


def run(args, root):
    return inspect(root, args.repo, args.sha, args.snapshot, args.scope, args.pr, args.base_sha)
