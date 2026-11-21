#!/usr/bin/env python3
"""GitHub-only stable release gate driver. Dry-run is read-only.

Only a protected v1 manual workflow with separately configured owner and
verifier public keys may publish. Evidence is read as data at an immutable
commit; no code is checked out from that commit. Never rebuild the candidate.
"""
from __future__ import annotations

import argparse
import base64
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

from v1_stable_gate import REPO, SHA, TAG, MAX_JSON, candidate, obj, read_regular, require, verify


def gh(*args):
    p = subprocess.run(["gh", *args], capture_output=True, timeout=300, check=False)
    require(p.returncode == 0, "GitHub API/artifact operation failed: " + args[0])
    return p.stdout


def api(path):
    return obj(gh("api", path))


def protected_source(root):
    source = os.environ.get("GITHUB_SHA", "")
    require(os.environ.get("GITHUB_ACTIONS") == "true" and
            os.environ.get("GITHUB_EVENT_NAME") == "workflow_dispatch" and
            os.environ.get("GITHUB_REPOSITORY") == REPO and
            os.environ.get("GITHUB_REF") == "refs/heads/v1" and
            os.environ.get("GITHUB_REF_PROTECTED") == "true" and SHA.fullmatch(source),
            "stable gate requires protected v1 manual workflow")
    actual = subprocess.check_output(["git", "-C", str(root), "rev-parse", "HEAD"], timeout=15).decode().strip()
    require(actual == source, "checkout/source mismatch")
    require(api(f"repos/{REPO}/git/ref/heads/v1").get("object", {}).get("sha") == source,
            "protected v1 HEAD advanced")
    return source


def evidence_file(ref, tag, filename, limit):
    path = f"release-signoff/{tag}/{filename}"
    data = api(f"repos/{REPO}/contents/{path}?ref={ref}")
    require(data.get("path") == path and data.get("type") == "file" and data.get("encoding") == "base64",
            "evidence file identity mismatch")
    size = data.get("size")
    require(type(size) is int and 0 < size <= limit, "evidence file size outside bound")
    raw = base64.b64decode("".join(data["content"].split()), validate=True)
    require(len(raw) == size, "evidence size mismatch")
    return raw


def not_exists(path):
    p = subprocess.run(["gh", "api", path], capture_output=True, timeout=30, check=False)
    require(p.returncode != 0 and re.search(rb"\(HTTP 404\)\s*$", p.stderr) is not None,
            "existing tag/release or inability to establish absence")


def gate(root, run_id, evidence_ref, publish=False):
    source = protected_source(root)
    require(re.fullmatch(r"[1-9][0-9]{0,18}", run_id) is not None and SHA.fullmatch(evidence_ref),
            "invalid immutable evidence/run reference")
    tag = "v" + (root / "VERSION").read_text().strip()
    require(TAG.fullmatch(tag) is not None, "stable version required")
    # Key material must be configured by the owner OUTSIDE the candidate and
    # supplied by the protected environment. Nothing from evidence can set it.
    owner = os.environ.get("STABLE_OWNER_PUBLIC_KEY", "").encode()
    verifier = os.environ.get("STABLE_VERIFIER_PUBLIC_KEY", "").encode()
    require(owner and verifier, "stable release signer trust not configured")
    run = api(f"repos/{REPO}/actions/runs/{run_id}")
    require(run.get("id") == int(run_id) and type(run.get("run_attempt")) is int and
            0 < run["run_attempt"] < 2**31, "run ID/attempt mismatch")
    attempt = run["run_attempt"]
    jobs = api(f"repos/{REPO}/actions/runs/{run_id}/attempts/{attempt}/jobs?per_page=100")
    identity = candidate(run, jobs, source)
    artifact_name = f"v1-stable-assets-{attempt}"
    art = api(f"repos/{REPO}/actions/runs/{run_id}/artifacts?per_page=100")
    artifacts = art.get("artifacts")
    require(type(artifacts) is list and art.get("total_count") == len(artifacts),
            "incomplete artifact inventory")
    matches = [a for a in artifacts if a.get("name") == artifact_name]
    require(len(matches) == 1 and matches[0].get("expired") is False, "candidate artifact missing/expired")
    artifact = matches[0]
    require(type(artifact.get("id")) is int and artifact["id"] > 0 and
            type(artifact.get("size_in_bytes")) is int and 0 < artifact["size_in_bytes"] <= 220 * 1024 * 1024,
            "invalid artifact identity/size")
    raw = evidence_file(evidence_ref, tag, "signoff.json", MAX_JSON)
    sig1 = evidence_file(evidence_ref, tag, "owner.sig", 64)
    sig2 = evidence_file(evidence_ref, tag, "verifier.sig", 64)
    with tempfile.TemporaryDirectory(prefix="bw-stable-release-") as tmp:
        assets = Path(tmp) / "assets"
        assets.mkdir()
        gh("run", "download", run_id, "--repo", REPO, "--name", artifact_name, "--dir", str(assets))
        result = verify(assets, root, run, jobs, source, tag, raw, sig1, sig2, owner, verifier)
        require((result["run_id"], result["run_attempt"]) == identity, "candidate identity changed")
        # Recheck protected source and the latest run attempt after verification.
        require(protected_source(root) == source, "protected source changed")
        latest = api(f"repos/{REPO}/actions/runs/{run_id}")
        require(latest.get("run_attempt") == attempt and latest.get("conclusion") == "success"
                and latest.get("head_sha") == source, "candidate attempt changed")
        not_exists(f"repos/{REPO}/git/ref/tags/{tag}")
        not_exists(f"repos/{REPO}/releases/tags/{tag}")
        if publish:
            require(os.environ.get("STABLE_PUBLISH_AUTHORIZED") == "true",
                    "publication job not explicitly authorized")
            # No tag replacement, no stale artifact rebuild, no stable pointer
            # change. An interrupted publication requires reviewed reconciliation.
            require(protected_source(root) == source, "protected source changed before publication")
            gh("api", f"repos/{REPO}/git/refs", "-f", f"ref=refs/tags/{tag}", "-f", f"sha={source}")
            files = sorted(str(p) for p in assets.iterdir())
            gh("release", "create", tag, *files, "--repo", REPO, "--verify-tag", "--title", f"BoxWeave {tag}",
               "--notes", f"Signed S5 acceptance at {evidence_ref}; source {source}; candidate run {run_id} attempt {attempt}. Stable pointer unchanged.")
            readback = Path(tmp) / "readback"
            readback.mkdir()
            gh("release", "download", tag, "--repo", REPO, "--dir", str(readback))
            verified = verify(readback, root, run, jobs, source, tag, raw, sig1, sig2, owner, verifier)
            require(verified == result, "published readback differs")
            live = api(f"repos/{REPO}/releases/tags/{tag}")
            require(live.get("tag_name") == tag and live.get("draft") is False and
                    live.get("prerelease") is False and {a["name"] for a in live["assets"]} ==
                    {p.name for p in assets.iterdir()}, "published release identity/inventory mismatch")
            require(api(f"repos/{REPO}/git/ref/tags/{tag}").get("object", {}).get("sha") == source,
                    "published tag/source mismatch")
        result["publication"] = "PUBLISHED_EXACT_READBACK" if publish else "DRY_RUN_NO_PUBLICATION"
        return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--run-id", required=True)
    p.add_argument("--evidence-commit", required=True)
    p.add_argument("--publish", action="store_true")
    a = p.parse_args()
    try:
        result = gate(Path(__file__).resolve().parent.parent, a.run_id, a.evidence_commit, a.publish)
    except (OSError, ValueError, TypeError, KeyError, subprocess.SubprocessError) as e:
        p.exit(1, "STABLE_RELEASE_GATE_BLOCKED: " + str(e) + "\n")
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main()
