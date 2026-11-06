"""Fetch and verify the original immutable GitHub candidate wrapper, without execution."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import stat
import subprocess
import zipfile

from .engine import AcceptanceFault
from .workflow_gate import api, run_identity, require, ASSETS, asset_index, REPOSITORY, CANDIDATE_WORKFLOW


def unpack(archive, digest, output):
    require(re.fullmatch(r"sha256:[a-f0-9]{64}", str(digest)), "CANDIDATE_WRAPPER_PIN_INVALID")
    archive = Path(archive)
    require(archive.is_file() and not archive.is_symlink() and
            0 < archive.stat().st_size <= 180_000_000, "CANDIDATE_WRAPPER_SIZE_INVALID")
    data = archive.read_bytes()
    require(len(data) <= 180_000_000 and "sha256:" + hashlib.sha256(data).hexdigest() == digest,
            "CANDIDATE_WRAPPER_DIGEST_MISMATCH")
    out = Path(output)
    require(not out.exists() and not out.is_symlink(), "CANDIDATE_OUTPUT_EXISTS")
    with zipfile.ZipFile(archive) as package:
        items = package.infolist()
        require(len(items) == len(ASSETS) and {i.filename for i in items} == ASSETS,
                "CANDIDATE_WRAPPER_MEMBERS_INVALID")
        require(sum(i.file_size for i in items) <= 200_000_000 and
                all(not i.is_dir() and not i.flag_bits & 1 and
                    stat.S_IFMT(i.external_attr >> 16) in (0, stat.S_IFREG)
                    for i in items), "CANDIDATE_WRAPPER_UNSAFE")
        # Validate all members before creating any output. Never use extractall.
        contents = {i.filename: package.read(i) for i in items}
    out.mkdir(mode=0o700)
    for name, content in contents.items():
        with (out / name).open("xb") as stream:
            stream.write(content)
    return asset_index(out)


def fetch(run_id, source, output):
    require(re.fullmatch(r"[1-9][0-9]{0,19}", str(run_id)), "CANDIDATE_RUN_ID_INVALID")
    prefix = "repos/" + REPOSITORY + "/actions/"
    run = api(prefix + "runs/" + str(run_id))
    run_identity(run, source, CANDIDATE_WORKFLOW, "push")
    listing = api(prefix + "runs/" + str(run_id) + "/artifacts?per_page=100")
    require(listing.get("total_count") == len(listing.get("artifacts", [])),
            "CANDIDATE_ARTIFACT_LIST_INCOMPLETE")
    matches = [a for a in listing["artifacts"] if a.get("name") == "application-candidate"]
    require(len(matches) == 1, "CANDIDATE_ARTIFACT_AMBIGUOUS")
    item = matches[0]
    require(type(item.get("id")) is int and item["id"] > 0 and
            item.get("expired") is False and
            item.get("workflow_run", {}).get("id") == int(run_id) and
            item.get("workflow_run", {}).get("head_sha") == source,
            "CANDIDATE_ARTIFACT_ORIGIN_INVALID")
    wrapper = Path(str(output) + ".original-wrapper.zip")
    with wrapper.open("xb") as stream:
        result = subprocess.run(["gh", "api", prefix + "artifacts/" + str(item["id"]) + "/zip"],
                                stdout=stream, stderr=subprocess.DEVNULL, timeout=90, check=False)
    require(result.returncode == 0, "CANDIDATE_ORIGINAL_DOWNLOAD_FAILED")
    index = unpack(wrapper, item.get("digest"), output)
    return {"source_sha": source, "candidate_run_id": int(run_id),
            "candidate_run_attempt": run["run_attempt"],
            "candidate_artifact_id": item["id"], "candidate_artifact_digest": item["digest"],
            "release_assets_sha256": index}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--candidate-run-id", required=True)
    p.add_argument("--source", required=True)
    p.add_argument("--out", required=True)
    args = p.parse_args(argv)
    try:
        print(json.dumps(fetch(args.candidate_run_id, args.source, args.out), sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, KeyError, zipfile.BadZipFile, subprocess.TimeoutExpired) as exc:
        print(json.dumps({"state": "BLOCKED", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
