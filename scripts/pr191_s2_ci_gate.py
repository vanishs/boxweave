"""Read-only exact-head S2 CI provenance gate for PR #191.
The existing native producer jobs retain all their original assertions.
This module does not dispatch, rerun, modify or approve a native effect.
"""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time

from pr191_s2_witness_map import evaluate_receipts, load_map

REPO = "vanishs/boxweave"
N2 = "v1-n2-installed-lifecycle.yml"
S25 = "v1-pr191-s25-scoped.yml"


def records(payload, key):
    values = payload[key]
    if not isinstance(values, list) or payload["total_count"] != len(values):
        raise ValueError("incomplete GitHub API page: " + key)
    return values


def api(path):
    result = subprocess.check_output(["gh", "api", "repos/" + REPO + "/" + path],
                                     text=True, timeout=60)
    return json.loads(result)


def select_run(payload, workflow, source, pr):
    matches = []
    for r in records(payload, "workflow_runs"):
        if (r["head_sha"] != source or r["event"] != "pull_request" or
                r["path"] != ".github/workflows/" + workflow or
                r["repository"]["full_name"] != REPO or r["head_repository"]["full_name"] != REPO):
            continue
        if any(p["number"] == pr and p["head"]["sha"] == source and
               p["base"]["ref"] == "v1" and p["head"]["repo"]["id"] == r["repository"]["id"] and
               p["base"]["repo"]["id"] == r["repository"]["id"] for p in r["pull_requests"]):
            matches.append(r)
    if len(matches) > 1:
        raise ValueError("ambiguous exact-source CI runs: " + workflow)
    return matches[0] if matches else None


def checked_lane(run, jobs, artifacts, lane, source):
    job_name = "s25-scoped-nested-startup-and-lifecycle" if lane == "s25" else "installed-lifecycle (" + lane + ")"
    j = [x for x in records(jobs, "jobs") if x["name"] == job_name]
    if len(j) != 1 or j[0]["status"] != "completed" or j[0]["conclusion"] != "success" or j[0]["run_id"] != run["id"]:
        raise ValueError("missing/failed exact native job: " + lane)
    required = ["Retain sanitized lane locator"]
    if lane != "s25":
        required.append("Index mapped lane only after all original lane checks")
    for step in required:
        if len([x for x in j[0]["steps"] if x["name"] == step and
                x["status"] == "completed" and x["conclusion"] == "success"]) != 1:
            raise ValueError("missing successful evidence step: " + lane)
    name = "pr191-s2-" + lane + "-attempt-" + str(run["run_attempt"])
    a = [x for x in records(artifacts, "artifacts") if x["name"] == name]
    if len(a) != 1 or a[0]["expired"] or a[0]["size_in_bytes"] > 65536:
        raise ValueError("missing/expired/oversized lane artifact: " + lane)
    meta = a[0]["workflow_run"]
    if (meta["id"] != run["id"] or meta["head_sha"] != source or
            meta["repository_id"] != run["repository"]["id"] or
            meta["head_repository_id"] != run["repository"]["id"]):
        raise ValueError("artifact provenance mismatch: " + lane)
    return {"lane": lane, "run_id": run["id"], "attempt": run["run_attempt"],
            "job_id": j[0]["id"], "artifact_id": a[0]["id"], "artifact_name": name}


def bind_receipt(receipt, location, event_sha):
    ci = receipt["ci"]
    if (receipt["report"]["lane"] != location["lane"] or
            ci["repository"] != REPO or ci["event"] != "pull_request" or
            ci["event_sha"] != event_sha or ci["run_id"] != location["run_id"] or
            ci["run_attempt"] != location["attempt"]):
        raise ValueError("receipt/original CI mismatch")


def main():
    source, pr_text, event_sha = sys.argv[1:]
    if (os.environ.get("GITHUB_ACTIONS") != "true" or
            os.environ.get("RUNNER_ENVIRONMENT") != "github-hosted" or
            os.environ.get("GITHUB_REPOSITORY") != REPO or
            os.environ.get("GITHUB_EVENT_NAME") != "pull_request" or
            os.environ.get("GITHUB_SHA") != event_sha or pr_text != "191" or
            not all(re.fullmatch(r"[a-f0-9]{40}", x) for x in (source, event_sha))):
        raise ValueError("exact same-repository PR #191 CI context required")
    if subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip() != source:
        raise ValueError("checkout source mismatch")
    lanes = sorted({lane for row in load_map() for lane in row["requires"]})
    # Producer workflows run independently; observe them only, never dispatch.
    deadline = time.monotonic() + 35 * 60
    while True:
        runs = {workflow: select_run(
            api("actions/workflows/" + workflow + "/runs?event=pull_request&head_sha=" +
                source + "&per_page=100"), workflow, source, int(pr_text))
                for workflow in (N2, S25)}
        if all(r and r["status"] == "completed" for r in runs.values()):
            break
        if time.monotonic() >= deadline:
            raise ValueError("exact producer runs not completed within observation budget")
        time.sleep(30)
    metadata = {}
    for workflow, run in runs.items():
        path = "actions/runs/" + str(run["id"])
        jobs = api(path + "/attempts/" + str(run["run_attempt"]) + "/jobs?per_page=100")
        artifacts = api(path + "/artifacts?per_page=100")
        for lane in lanes:
            if (lane == "s25") == (workflow == S25):
                metadata[lane] = checked_lane(run, jobs, artifacts, lane, source)
    if set(metadata) != set(lanes):
        raise ValueError("incomplete required native lane inventory")
    if len({m["job_id"] for m in metadata.values()}) != len(lanes) or len({
            m["artifact_id"] for m in metadata.values()}) != len(lanes):
        raise ValueError("reused native job or artifact")
    receipts = []
    with tempfile.TemporaryDirectory(prefix="pr191-s2-ci-") as root:
        for lane in lanes:
            loc = metadata[lane]
            directory = Path(root) / lane
            directory.mkdir()
            subprocess.run(["gh", "run", "download", str(loc["run_id"]),
                            "--repo", REPO, "--name", loc["artifact_name"],
                            "--dir", str(directory)], check=True, timeout=90)
            files = list(directory.rglob("*"))
            expected = "pr191-s25-lane.json" if lane == "s25" else "pr191-s2-lane.json"
            if (len(files) != 1 or files[0].name != expected or files[0].is_symlink() or
                    not files[0].is_file() or files[0].stat().st_size > 8192):
                raise ValueError("unexpected lane receipt contents")
            receipt = json.loads(files[0].read_text())
            bind_receipt(receipt, loc, event_sha)
            receipts.append(receipt)
    report = evaluate_receipts(receipts, source)
    # This is the original S2 coverage scope, not the whole N2 gate or public WAN.
    print(json.dumps({"source": source, "jobs": [metadata[lane] for lane in lanes],
                      "mapped_markers": report["original_marker_rows"],
                      "scope": "EXACT_CI_S2_CROSSWALK",
                      "public_wan": report["public_wan"],
                      "other_n2_lanes": "SEPARATE_REQUIRED_GATES"}, sort_keys=True))
    print("P1B_SCOPED_S2_EXACT_CI_CROSSWALK=PASS")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as exc:
        raise SystemExit(str(exc))
