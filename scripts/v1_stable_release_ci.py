#!/usr/bin/env python3
"""Publish verified assets from the current protected v1 manual workflow."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

from v1_stable_gate import REPO, SHA, obj, require, verify_assets


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


def not_exists(path):
    p = subprocess.run(["gh", "api", path], capture_output=True, timeout=30, check=False)
    require(p.returncode != 0 and re.search(rb"\(HTTP 404\)\s*$", p.stderr) is not None,
            "existing tag/release or inability to establish absence")


def publish(root, assets):
    require(os.environ.get("STABLE_PUBLISH_AUTHORIZED") == "true",
            "publication job not explicitly authorized")
    source = protected_source(root)
    tag = "v" + (root / "VERSION").read_text().strip()
    hashes = verify_assets(assets, tag, source, root)
    not_exists(f"repos/{REPO}/git/ref/tags/{tag}")
    not_exists(f"repos/{REPO}/releases/tags/{tag}")
    require(protected_source(root) == source, "protected source changed before publication")
    gh("api", f"repos/{REPO}/git/refs", "-f", f"ref=refs/tags/{tag}", "-f", f"sha={source}")
    files = sorted(str(p) for p in assets.iterdir())
    gh("release", "create", tag, *files, "--repo", REPO, "--verify-tag", "--title", f"BoxWeave {tag}",
       "--notes", f"Source {source}. Built and verified in the same Actions run. Stable pointer unchanged.")
    with tempfile.TemporaryDirectory(prefix="bw-stable-readback-") as tmp:
        readback = Path(tmp)
        gh("release", "download", tag, "--repo", REPO, "--dir", str(readback))
        require(verify_assets(readback, tag, source, root) == hashes, "published readback differs")
    live = api(f"repos/{REPO}/releases/tags/{tag}")
    require(live.get("tag_name") == tag and live.get("draft") is False and
            live.get("prerelease") is False and {a["name"] for a in live["assets"]} ==
            {p.name for p in assets.iterdir()}, "published release identity/inventory mismatch")
    require(api(f"repos/{REPO}/git/ref/tags/{tag}").get("object", {}).get("sha") == source,
            "published tag/source mismatch")
    return {"publication": "PUBLISHED_EXACT_READBACK", "source": source, "tag": tag,
            "release_json_sha256": hashes[0], "sha256sums_sha256": hashes[1]}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--assets", required=True, type=Path)
    a = p.parse_args()
    try:
        result = publish(Path(__file__).resolve().parent.parent, a.assets)
    except (OSError, ValueError, TypeError, KeyError, subprocess.SubprocessError) as e:
        p.exit(1, "STABLE_RELEASE_BLOCKED: " + str(e) + "\n")
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main()
