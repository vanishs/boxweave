#!/usr/bin/env python3
"""Publish verified assets from the current protected v1 manual workflow."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import subprocess

from v1_stable_gate import REPO, SHA, obj, require, sha, verify_assets


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


def optional_api(path):
    p = subprocess.run(["gh", "api", path], capture_output=True, timeout=30, check=False)
    if p.returncode == 0:
        return obj(p.stdout)
    require(re.search(rb"\(HTTP 404\)\s*$", p.stderr) is not None,
            "unable to establish absence: " + path)
    return None


def find_release(tag):
    # The tag endpoint may omit drafts; authenticated listing includes them.
    matches = []
    for page in range(1, 101):
        batch = json.loads(gh("api", f"repos/{REPO}/releases?per_page=30&page={page}"))
        require(type(batch) is list and len(batch) <= 30, "invalid release page")
        matches.extend(r for r in batch if r.get("tag_name") == tag)
        if len(batch) < 30:
            require(len(matches) <= 1, "ambiguous releases for tag")
            return matches[0] if matches else None
    raise ValueError("incomplete release inventory")


def require_tag(tag, source):
    ref = optional_api(f"repos/{REPO}/git/ref/tags/{tag}")
    require(ref is not None and ref.get("ref") == f"refs/tags/{tag}" and
            ref.get("object", {}).get("type") == "commit" and
            ref["object"].get("sha") == source, "existing tag/source mismatch")


def release_inventory(release, tag, expected):
    require(release.get("tag_name") == tag and
            type(release.get("id")) is int and release["id"] > 0 and
            type(release.get("draft")) is bool and release.get("prerelease") is False,
            "release identity mismatch")
    inventory = {}
    ids = set()
    for entry in release["assets"]:
        name, asset_id = entry["name"], entry["id"]
        require(name in expected and name not in inventory and type(asset_id) is int and
                asset_id > 0 and asset_id not in ids, "foreign or duplicate release asset")
        inventory[name] = entry
        ids.add(asset_id)
    return inventory


def verify_remote_assets(inventory, expected, allow_starter=False):
    starters = []
    for name, entry in inventory.items():
        if allow_starter and entry.get("state") == "starter" and entry.get("size") == 0:
            starters.append(entry["id"])
            continue
        size, digest = expected[name]
        require(entry.get("state") == "uploaded" and entry.get("size") == size and
                entry.get("digest") in (None, "sha256:" + digest),
                "existing asset metadata differs: " + name)
        payload = gh("api", f"repos/{REPO}/releases/assets/{entry['id']}",
                     "-H", "Accept: application/octet-stream")
        require(len(payload) == size and sha(payload) == digest,
                "existing asset bytes differ: " + name)
    return starters


def publish(root, assets):
    require(os.environ.get("STABLE_PUBLISH_AUTHORIZED") == "true",
            "publication job not explicitly authorized")
    source = protected_source(root)
    tag = "v" + (root / "VERSION").read_text().strip()
    hashes = verify_assets(assets, tag, source, root)
    expected = {p.name: (p.stat().st_size, sha(p.read_bytes())) for p in assets.iterdir()}
    notes = (f"Source {source}. Verified stable candidate.\n"
             f"release.json SHA256: {hashes[0]}\nSHA256SUMS SHA256: {hashes[1]}")
    # No existing public release is edited, even on retry.
    release = find_release(tag)
    tag_path = f"repos/{REPO}/git/ref/tags/{tag}"
    ref = optional_api(tag_path)
    require(protected_source(root) == source, "protected source changed before publication")
    if ref is None:
        require(release is None, "release exists without its exact tag")
        gh("api", f"repos/{REPO}/git/refs", "-f", f"ref=refs/tags/{tag}", "-f", f"sha={source}")
    require_tag(tag, source)
    if release is None:
        release = obj(gh("api", f"repos/{REPO}/releases", "--method", "POST",
                         "-f", f"tag_name={tag}", "-f", f"target_commitish={source}",
                         "-f", f"name=BoxWeave {tag}", "-f", f"body={notes}",
                         "-F", "draft=true", "-F", "prerelease=false"))
        require(release.get("draft") is True, "new release is not a draft")
    inventory = release_inventory(release, tag, expected)
    release_id = release["id"]
    endpoint = f"repos/{REPO}/releases/{release_id}"
    if release["draft"]:
        require(release.get("body") == notes, "draft belongs to another candidate")
        # Verify every surviving asset before writing anything. A GitHub
        # interrupted upload can leave a zero-byte starter; only that exact
        # empty placeholder in our matching draft is eligible for deletion.
        starters = verify_remote_assets(inventory, expected, allow_starter=True)
        require(protected_source(root) == source, "protected source changed before upload")
        require_tag(tag, source)
        for asset_id in starters:
            gh("api", f"repos/{REPO}/releases/assets/{asset_id}", "--method", "DELETE")
        for name in sorted(expected):
            if name not in inventory or inventory[name]["id"] in starters:
                gh("release", "upload", tag, str(assets / name), "--repo", REPO)
        draft = api(endpoint)
        require(draft.get("id") == release_id and draft.get("draft") is True and
                draft.get("body") == notes, "draft changed during upload")
        inventory = release_inventory(draft, tag, expected)
        require(set(inventory) == set(expected), "incomplete draft asset inventory")
        verify_remote_assets(inventory, expected)
        require(protected_source(root) == source, "protected source changed before publication")
        require_tag(tag, source)
        gh("api", endpoint, "--method", "PATCH", "-F", "draft=false")
    live = api(endpoint)
    require(live.get("id") == release_id and live.get("draft") is False,
            "published release identity mismatch")
    inventory = release_inventory(live, tag, expected)
    require(set(inventory) == set(expected), "incomplete published asset inventory")
    verify_remote_assets(inventory, expected)
    require_tag(tag, source)
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
