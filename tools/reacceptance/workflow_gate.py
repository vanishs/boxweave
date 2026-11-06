"""Developer-side GitHub run provenance and complete release asset checks.

This reader cannot dispatch workflows, install software, sign, or publish.
GitHub API observations must be supplied through the authenticated official CLI.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

from bwmini.online import RELEASE_FILES
from .engine import AcceptanceFault

REPOSITORY = "vanishs/boxweave"
CANDIDATE_WORKFLOW = ".github/workflows/ci.yml"
VERIFIER_WORKFLOW = ".github/workflows/independent-verifier.yml"
ASSETS = frozenset((*RELEASE_FILES, "install.sh", "release.json", "SHA256SUMS"))


def require(value, reason):
    if not value:
        raise AcceptanceFault(reason)


def run_identity(run, source, workflow, event):
    require(isinstance(run, dict), "WORKFLOW_RUN_INVALID")
    require(run.get("repository", {}).get("full_name") == REPOSITORY and
            run.get("head_repository", {}).get("full_name") == REPOSITORY,
            "WORKFLOW_REPOSITORY_UNTRUSTED")
    require(run.get("head_sha") == source and run.get("head_branch") == "main",
            "WORKFLOW_SOURCE_MISMATCH")
    require(run.get("path") == workflow and run.get("event") == event,
            "WORKFLOW_ORIGIN_UNTRUSTED")
    require(run.get("status") == "completed" and run.get("conclusion") == "success",
            "WORKFLOW_NOT_SUCCESSFUL")
    require(type(run.get("run_attempt")) is int and 0 < run["run_attempt"] < 2**63 and
            type(run.get("id")) is int and 0 < run["id"] < 2**63,
            "WORKFLOW_ATTEMPT_INVALID")


def provenance(candidate, verifier, artifacts, source):
    require(isinstance(source, str) and re.fullmatch(r"[0-9a-f]{40}", source),
            "WORKFLOW_SOURCE_INVALID")
    run_identity(candidate, source, CANDIDATE_WORKFLOW, "push")
    run_identity(verifier, source, VERIFIER_WORKFLOW, "workflow_dispatch")
    require(type(candidate.get("id")) is int and type(verifier.get("id")) is int
            and candidate["id"] != verifier["id"], "SEPARATE_VERIFIER_RUN_REQUIRED")
    require(isinstance(artifacts, dict) and artifacts.get("total_count") ==
            len(artifacts.get("artifacts", [])), "CANDIDATE_ARTIFACT_LIST_INCOMPLETE")
    matches = [a for a in artifacts["artifacts"] if a.get("name") == "application-candidate"]
    require(len(matches) == 1, "CANDIDATE_ARTIFACT_AMBIGUOUS")
    artifact = matches[0]
    require(artifact.get("expired") is False and type(artifact.get("id")) is int and 0 < artifact["id"] < 2**63
            and re.fullmatch(r"sha256:[a-f0-9]{64}", str(artifact.get("digest", ""))),
            "CANDIDATE_ARTIFACT_UNAVAILABLE")
    origin = artifact.get("workflow_run", {})
    require(origin.get("id") == candidate["id"] and origin.get("head_sha") == source,
            "CANDIDATE_ARTIFACT_ORIGIN_MISMATCH")
    return {"candidate_run_id": candidate["id"],
            "candidate_run_attempt": candidate["run_attempt"],
            "candidate_artifact_id": artifact["id"],
            "candidate_artifact_digest": artifact["digest"],
            "verifier_run_id": verifier["id"],
            "verifier_run_attempt": verifier["run_attempt"]}


def asset_index(directory):
    root = Path(directory)
    require(root.is_dir() and not root.is_symlink(), "RELEASE_ASSET_DIRECTORY_INVALID")
    entries = list(root.iterdir())
    require({p.name for p in entries} == ASSETS, "RELEASE_ASSET_SET_MISMATCH")
    result = {}
    for p in entries:
        require(p.is_file() and not p.is_symlink() and
                0 < p.stat().st_size <= (180_000_000 if p.name == "boxweave.zip" else 8_000_000),
                "RELEASE_ASSET_UNSAFE_" + p.name)
        result[p.name] = hashlib.sha256(p.read_bytes()).hexdigest()
    manifest = json.loads((root / "release.json").read_text())
    require(set(manifest.get("files", {})) == set(RELEASE_FILES),
            "RELEASE_MANIFEST_ASSET_SET_MISMATCH")
    require(all(result[n] == digest for n, digest in manifest["files"].items()),
            "RELEASE_ASSET_DIGEST_MISMATCH")
    lines = (root / "SHA256SUMS").read_text().splitlines()
    expected = [result[n] + "  " + n for n in sorted(ASSETS - {"SHA256SUMS"})]
    require(sorted(lines) == sorted(expected), "RELEASE_CHECKSUM_FILE_MISMATCH")
    return result


def api(path):
    completed = subprocess.run(["gh", "api", path], capture_output=True,
                               text=True, timeout=30, check=False)
    require(completed.returncode == 0, "GITHUB_PROVENANCE_UNAVAILABLE")
    return json.loads(completed.stdout)


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--source", required=True)
    p.add_argument("--candidate-run-id", required=True)
    p.add_argument("--verifier-run-id", required=True)
    p.add_argument("--assets", required=True)
    p.add_argument("--attestation", required=True)
    p.add_argument("--context-out")
    args = p.parse_args(argv)
    try:
        require(all(re.fullmatch(r"[1-9][0-9]{0,19}", v) for v in
                    (args.candidate_run_id, args.verifier_run_id)), "WORKFLOW_ID_INVALID")
        prefix = "repos/" + REPOSITORY + "/actions/runs/"
        context = provenance(api(prefix + args.candidate_run_id),
                             api(prefix + args.verifier_run_id),
                             api(prefix + args.candidate_run_id + "/artifacts?per_page=100"),
                             args.source)
        evidence = json.loads(Path(args.attestation).read_text())
        require(evidence.get("workflow") == context, "SIGNED_WORKFLOW_CONTEXT_MISMATCH")
        require(evidence.get("release_assets_sha256") == asset_index(args.assets),
                "SIGNED_RELEASE_ASSET_INDEX_MISMATCH")
        if args.context_out:
            with Path(args.context_out).open("x") as stream:
                json.dump(context, stream, sort_keys=True)
        print(json.dumps({"decision": "PROVENANCE_AND_ALL_ASSETS_MATCH", **context}, sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, TypeError, KeyError, subprocess.TimeoutExpired) as exc:
        print(json.dumps({"decision": "DENIED", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
