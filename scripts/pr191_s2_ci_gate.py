"""Read-only exact-head S2 CI provenance gate for PR and controlled RC.
The existing native producer jobs retain all their original assertions.
This module does not dispatch, rerun, modify or approve a native effect.
"""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time

from pr191_s2_witness_map import evaluate_receipts, load_map
from pr191_n2_evidence_identity import evaluate as n2_identity

REPO = "vanishs/boxweave"
N2 = "v1-n2-installed-lifecycle.yml"
S25 = "v1-pr191-s25-scoped.yml"
RELEASE = "v1-release.yml"
UNMAPPED_N2 = {"scoped-retirement", "scoped-offline-empty", "scoped-offline-settled",
               "scoped-offline-original", "scoped-update-ack", "scoped-update-ack-receipt", "scoped-noquota"}
RC_REF = re.compile(r"refs/heads/release/(v1\.[0-9]+\.[0-9]+-rc\.[1-9][0-9]*)")


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
                r["repository"]["full_name"] != REPO or r["head_repository"]["full_name"] != REPO or
                r["head_repository"]["id"] != r["repository"]["id"]):
            continue
        if any(p["number"] == pr and p["head"]["sha"] == source and
               p["base"]["ref"] == "v1" and p["head"]["repo"]["id"] == r["repository"]["id"] and
               p["base"]["repo"]["id"] == r["repository"]["id"] for p in r["pull_requests"]):
            matches.append(r)
    if len(matches) > 1:
        raise ValueError("ambiguous exact-source CI runs: " + workflow)
    return matches[0] if matches else None


def checked_lane(run, jobs, artifacts, lane, source, identity_only=False):
    release_s25 = lane == "s25" and run["path"] == ".github/workflows/" + RELEASE
    job_name = ("v1-prerelease-nested-S2.5-isolated-runner" if release_s25 else
                "s25-scoped-nested-startup-and-lifecycle") if lane == "s25" else "installed-lifecycle (" + lane + ")"
    j = [x for x in records(jobs, "jobs") if x["name"] == job_name]
    if len(j) != 1 or j[0]["status"] != "completed" or j[0]["conclusion"] != "success" or j[0]["run_id"] != run["id"]:
        raise ValueError("missing/failed exact native job: " + lane)
    required = ["Retain sanitized lane locator"]
    if identity_only:
        if lane not in UNMAPPED_N2 or run["event"] != "push":
            raise ValueError("release-only unmapped identity required")
        required = ["Retain sanitized release-only N2 candidate identity"]
    elif lane != "s25":
        required.append("Index mapped lane only after all original lane checks")
    elif release_s25:
        required.extend(("Require the exact protected v1 HEAD before touching a disposable VM",
                         "Real nested independent Agent/Incus verifier"))
    for step in required:
        if len([x for x in j[0]["steps"] if x["name"] == step and
                x["status"] == "completed" and x["conclusion"] == "success"]) != 1:
            raise ValueError("missing successful evidence step: " + lane)
    name = ("pr191-n2-identity-" if identity_only else "pr191-s2-") + lane + "-attempt-" + str(run["run_attempt"])
    a = [x for x in records(artifacts, "artifacts") if x["name"] == name]
    if len(a) != 1 or a[0]["expired"] or not 0 < a[0]["size_in_bytes"] <= 65536:
        raise ValueError("missing/expired/oversized lane artifact: " + lane)
    meta = a[0]["workflow_run"]
    if (meta["id"] != run["id"] or meta["head_sha"] != source or
            meta["repository_id"] != run["repository"]["id"] or
            meta["head_repository_id"] != run["repository"]["id"]):
        raise ValueError("artifact provenance mismatch: " + lane)
    return {"lane": lane, "run_id": run["id"], "attempt": run["run_attempt"],
            "job_id": j[0]["id"], "artifact_id": a[0]["id"], "artifact_name": name}


def bind_receipt(receipt, location, event_sha, release_ref=None, pr=None):
    ci = receipt["ci"]
    if (receipt["report"]["lane"] != location["lane"] or
            ci["repository"] != REPO or ci["event"] != ("push" if release_ref else "pull_request") or
            ci["event_sha"] != event_sha or ci["run_id"] != location["run_id"] or
            ci["run_attempt"] != location["attempt"]):
        raise ValueError("receipt/original CI mismatch")
    if release_ref:
        workflow = RELEASE if location["lane"] == "s25" else N2
        job = "native-s25-gate" if location["lane"] == "s25" else "installed-lifecycle"
        if (ci["workflow_ref"] != REPO + "/.github/workflows/" + workflow + "@" + release_ref or
                ci["job"] != job or receipt["report"]["source"] != event_sha):
            raise ValueError("release receipt workflow/ref/job mismatch")
    else:
        workflow = S25 if location["lane"] == "s25" else N2
        job = "s25-bridgehead-lab" if location["lane"] == "s25" else "installed-lifecycle"
        if (type(pr) is not int or pr <= 0 or
                ci.get("workflow_ref") != REPO + "/.github/workflows/" + workflow + "@refs/pull/" + str(pr) + "/merge" or
                ci.get("job") != job):
            raise ValueError("PR receipt workflow/ref/job mismatch")


def release_context(env, source, workflows):
    ref = env.get("GITHUB_REF", "")
    match = RC_REF.fullmatch(ref)
    if (env.get("GITHUB_ACTIONS") != "true" or env.get("RUNNER_ENVIRONMENT") != "github-hosted" or
            env.get("GITHUB_REPOSITORY") != REPO or env.get("GITHUB_EVENT_NAME") != "push" or
            not match or not re.fullmatch(r"[0-9a-f]{40}", source) or env.get("GITHUB_SHA") != source or
            env.get("GITHUB_WORKFLOW_REF") not in {
                REPO + "/.github/workflows/" + w + "@" + ref for w in workflows}):
        raise ValueError("exact same-repository release push context required")
    tag = match[1]
    if tag[1:].split("-rc.")[0] != Path("VERSION").read_text().strip():
        raise ValueError("release branch and source VERSION mismatch")
    return ref, tag


def release_identity_receipt(raw, lane, source, env):
    release_context(env, source, (N2,))
    if lane not in UNMAPPED_N2 or env.get("GITHUB_JOB") != "installed-lifecycle":
        raise ValueError("exact unmapped release N2 identity required")
    if [line for line in raw.decode("utf-8", "strict").splitlines()
            if line.startswith("N2_EVIDENCE_LANE=")] != ["N2_EVIDENCE_LANE=" + lane]:
        raise ValueError("N2 lane identity mismatch")
    def number(key):
        value = env.get(key, "")
        if not re.fullmatch(r"[1-9][0-9]{0,19}", value):
            raise ValueError("invalid " + key)
        return int(value)
    return {"schema": 1, "kind": "N2_RELEASE_IDENTITY_ONLY",
            "report": {"lane": lane, **n2_identity(raw, source)},
            "ci": {"repository": REPO, "workflow_ref": env["GITHUB_WORKFLOW_REF"],
                   "job": env["GITHUB_JOB"], "event": "push", "event_sha": source,
                   "run_id": number("GITHUB_RUN_ID"), "run_attempt": number("GITHUB_RUN_ATTEMPT")}}


def check_release_identity(receipt, location, source, ref, digest):
    bind_receipt(receipt, location, source, ref)
    report = receipt["report"]
    if (set(receipt) != {"schema", "kind", "report", "ci"} or receipt["schema"] != 1 or
            receipt["kind"] != "N2_RELEASE_IDENTITY_ONLY" or report["lane"] not in UNMAPPED_N2 or
            set(report) != {"lane", "source", "candidate_sha256", "transcript_sha256"} or
            report["candidate_sha256"] != digest or
            not re.fullmatch(r"[0-9a-f]{64}", report["transcript_sha256"])):
        raise ValueError("unmapped N2 release identity mismatch")


def select_release_run(payload, workflow, source, ref, run_id=None, attempt=None):
    matches = [r for r in records(payload, "workflow_runs")
               if r["head_sha"] == source and r["event"] == "push" and
               r["head_branch"] == ref.removeprefix("refs/heads/") and
               r["path"] == ".github/workflows/" + workflow and
               r["repository"]["full_name"] == REPO and r["head_repository"]["full_name"] == REPO and
               r["repository"]["id"] == r["head_repository"]["id"] and
               (run_id is None or r["id"] == run_id and r["run_attempt"] == attempt)]
    if len(matches) > 1:
        raise ValueError("ambiguous exact release CI runs: " + workflow)
    return matches[0] if matches else None


def require_release_n2(run, jobs):
    # The 22 mapped N2 lanes plus the seven independent installed gates.
    lanes = {lane for row in load_map() for lane in row["requires"]} - {"s25"}
    lanes |= UNMAPPED_N2
    expected = {"installed-lifecycle (" + lane + ")" for lane in lanes}
    actual = records(jobs, "jobs")
    if (len(expected) != 29 or run["status"] != "completed" or run["conclusion"] != "success" or
            len(actual) != len(expected) or {j["name"] for j in actual} != expected or
            any(j["run_id"] != run["id"] or j["status"] != "completed" or
                j["conclusion"] != "success" for j in actual)):
        raise ValueError("release requires every retained N2 lane")
    for j in actual:
        for step in ("Require the exact protected v1 HEAD for a controlled RC producer",
                     "Require original installed lane assertions"):
            if len([x for x in j["steps"] if x["name"] == step and
                    x["status"] == "completed" and x["conclusion"] == "success"]) != 1:
                raise ValueError("release N2 missing successful required step: " + j["name"])


def checked_release_manifest(run, jobs, artifacts, source):
    j = [x for x in records(jobs, "jobs") if x["name"] == "v1-exact-source-two-arch-manifest"]
    if len(j) != 1 or j[0]["run_id"] != run["id"] or j[0]["status"] != "completed" or j[0]["conclusion"] != "success":
        raise ValueError("missing successful current-attempt candidate manifest job")
    if len([x for x in j[0]["steps"] if x["name"] == "Make and self-verify exact candidate manifest"
            and x["status"] == "completed" and x["conclusion"] == "success"]) != 1:
        raise ValueError("missing manifest verification step")
    a = [x for x in records(artifacts, "artifacts") if x["name"] == "v1-release-candidate-manifest"]
    if len(a) != 1 or a[0]["expired"] or not 0 < a[0]["size_in_bytes"] <= 300 * 1024 * 1024:
        raise ValueError("missing/expired/oversized manifest artifact")
    meta = a[0]["workflow_run"]
    if (meta["id"] != run["id"] or meta["head_sha"] != source or
            meta["repository_id"] != run["repository"]["id"] or
            meta["head_repository_id"] != run["repository"]["id"]):
        raise ValueError("release manifest artifact provenance mismatch")
    return {"job_id": j[0]["id"], "artifact_id": a[0]["id"], "run_id": run["id"],
            "attempt": run["run_attempt"]}


def release_candidate_digest(directory, tag, source):
    manifest = json.loads((directory / "release.json").read_text())
    name = "boxweave-" + tag + "-linux-amd64"
    asset = manifest["assets"][name]
    path = directory / name
    if path.is_symlink() or not path.is_file() or not 0 < path.stat().st_size <= 100 * 1024 * 1024:
        raise ValueError("invalid release asset file")
    data = path.read_bytes()
    from v1_release_manifest import validate_elf
    validate_elf(data, "amd64")
    digest = hashlib.sha256(data).hexdigest()
    if (manifest["source"] != source or manifest["tag"] != tag or manifest["version"] != tag[1:] or
            manifest["platform"] != "linux" or manifest["cgo_enabled"] is not True or
            asset["goarch"] != "amd64" or asset["size"] != len(data) or asset["sha256"] != digest):
        raise ValueError("release amd64 asset identity mismatch")
    return digest


def require_release_digests(receipts, digest):
    if any(r["report"]["candidate_sha256"] != digest for r in receipts):
        raise ValueError("native candidate differs from exact release amd64 asset")


def release_main(directory):
    source = os.environ["GITHUB_SHA"]
    ref, tag = release_context(os.environ, source, (RELEASE,))
    if subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip() != source:
        raise ValueError("checkout source mismatch")
    run_id, attempt = (int(os.environ[k]) for k in ("GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT"))
    if run_id <= 0 or attempt <= 0 or os.environ.get("GITHUB_JOB") != "native-s2-gate":
        raise ValueError("exact release consumer job required")
    if api("git/ref/heads/v1")["object"]["sha"] != source:
        raise ValueError("release must still match protected v1 HEAD")
    digest = release_candidate_digest(directory, tag, source)
    # Same release run: S25 and manifest jobs are dependencies, not the whole
    # run (waiting for ourselves to finish would deadlock).
    own = select_release_run({"total_count": 1, "workflow_runs": [
        api("actions/runs/" + str(run_id))]}, RELEASE, source, ref, run_id, attempt)
    if own is None:
        raise ValueError("release consumer run identity mismatch")
    deadline = time.monotonic() + 35 * 60
    while True:
        n2 = select_release_run(api("actions/workflows/" + N2 + "/runs?event=push&head_sha=" +
                                    source + "&per_page=100"), N2, source, ref)
        if n2 and n2["status"] == "completed":
            break
        if time.monotonic() >= deadline:
            raise ValueError("exact release N2 not completed within observation budget")
        time.sleep(30)
    lanes = sorted({lane for row in load_map() for lane in row["requires"]})
    metadata = {}
    extra = {}
    for run, is_s25 in ((n2, False), (own, True)):
        path = "actions/runs/" + str(run["id"])
        jobs = api(path + "/attempts/" + str(run["run_attempt"]) + "/jobs?per_page=100")
        artifacts = api(path + "/artifacts?per_page=100")
        if not is_s25:
            require_release_n2(run, jobs)
            for lane in sorted(UNMAPPED_N2):
                extra[lane] = checked_lane(run, jobs, artifacts, lane, source, identity_only=True)
        else:
            manifest_location = checked_release_manifest(run, jobs, artifacts, source)
        for lane in lanes:
            if (lane == "s25") == is_s25:
                metadata[lane] = checked_lane(run, jobs, artifacts, lane, source)
    report, receipts = collect_receipts(metadata, lanes, source, source, release_ref=ref)
    require_release_digests(receipts, digest)
    all_locations = list(metadata.values()) + list(extra.values())
    if len(all_locations) != 30 or any(len({m[key] for m in all_locations}) != 30
                                        for key in ("job_id", "artifact_id")):
        raise ValueError("incomplete or reused release native identities")
    with tempfile.TemporaryDirectory(prefix="pr191-rc-n2-") as root:
        for lane, loc in sorted(extra.items()):
            receipt = download_receipt(loc, Path(root) / lane, "pr191-n2-identity.json")
            check_release_identity(receipt, loc, source, ref, digest)
            receipts.append(receipt)
    if len({r["report"]["transcript_sha256"] for r in receipts}) != 30:
        raise ValueError("reused release transcript")
    print(json.dumps({"source": source, "release_ref": ref, "candidate_sha256": digest,
                      "jobs": [metadata[lane] for lane in lanes],
                      "manifest": manifest_location,
                      "additional_n2": [extra[lane] for lane in sorted(extra)],
                      "mapped_markers": report["original_marker_rows"],
                      "scope": "EXACT_RC_CI_S2_CROSSWALK",
                      "public_wan": report["public_wan"]}, sort_keys=True))
    print("P2_R4_SCOPED_RELEASE_S2_EXACT_CI_CROSSWALK=PASS")


def download_receipt(loc, directory, expected):
    directory.mkdir()
    subprocess.run(["gh", "run", "download", str(loc["run_id"]),
                    "--repo", REPO, "--name", loc["artifact_name"],
                    "--dir", str(directory)], check=True, timeout=90)
    files = list(directory.rglob("*"))
    if (len(files) != 1 or files[0].name != expected or files[0].is_symlink() or
            not files[0].is_file() or files[0].stat().st_size > 8192):
        raise ValueError("unexpected lane receipt contents")
    return json.loads(files[0].read_text())


def collect_receipts(metadata, lanes, source, event_sha, release_ref=None, pr=None):
    if set(metadata) != set(lanes):
        raise ValueError("incomplete required native lane inventory")
    if len({m["job_id"] for m in metadata.values()}) != len(lanes) or len({
            m["artifact_id"] for m in metadata.values()}) != len(lanes):
        raise ValueError("reused native job or artifact")
    receipts = []
    with tempfile.TemporaryDirectory(prefix="pr191-s2-ci-") as root:
        for lane in lanes:
            loc = metadata[lane]
            expected = "pr191-s25-lane.json" if lane == "s25" else "pr191-s2-lane.json"
            receipt = download_receipt(loc, Path(root) / lane, expected)
            bind_receipt(receipt, loc, event_sha, release_ref, pr)
            receipts.append(receipt)
    report = evaluate_receipts(receipts, source)
    return report, receipts


def pr_context(env, source, pr_text, event_sha):
    if not re.fullmatch(r"[1-9][0-9]{0,18}", pr_text):
        raise ValueError("invalid PR number")
    ref = "refs/pull/" + pr_text + "/merge"
    if (env.get("GITHUB_ACTIONS") != "true" or
            env.get("RUNNER_ENVIRONMENT") != "github-hosted" or
            env.get("GITHUB_REPOSITORY") != REPO or
            env.get("GITHUB_EVENT_NAME") != "pull_request" or
            env.get("GITHUB_SHA") != event_sha or env.get("GITHUB_REF") != ref or
            env.get("GITHUB_JOB") != "s2-native-blackbox" or
            env.get("GITHUB_WORKFLOW_REF") != REPO + "/.github/workflows/v1-s2-native.yml@" + ref or
            not all(re.fullmatch(r"[a-f0-9]{40}", x) for x in (source, event_sha))):
        raise ValueError("exact same-repository Scoped PR CI context required")
    return int(pr_text)


def main():
    if sys.argv[1:] == ["--release-version"]:
        source = subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip()
        _, tag = release_context(os.environ, source, (N2, RELEASE))
        print(tag[1:])
        return
    if len(sys.argv) == 3 and sys.argv[1] == "--release":
        release_main(Path(sys.argv[2]))
        return
    if len(sys.argv) == 4 and sys.argv[1] == "--release-n2-identity":
        source = subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip()
        receipt = release_identity_receipt(Path(sys.argv[2]).read_bytes(),
                                           os.environ["N2_EVIDENCE_LANE"], source, os.environ)
        with Path(sys.argv[3]).open("x", encoding="utf-8") as out:
            json.dump(receipt, out, sort_keys=True)
            out.write("\n")
        return
    source, pr_text, event_sha = sys.argv[1:]
    pr = pr_context(os.environ, source, pr_text, event_sha)
    if subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip() != source:
        raise ValueError("checkout source mismatch")
    lanes = sorted({lane for row in load_map() for lane in row["requires"]})
    # Producer workflows run independently; observe them only, never dispatch.
    deadline = time.monotonic() + 35 * 60
    while True:
        runs = {workflow: select_run(
            api("actions/workflows/" + workflow + "/runs?event=pull_request&head_sha=" +
                source + "&per_page=100"), workflow, source, pr)
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
    report, _ = collect_receipts(metadata, lanes, source, event_sha, pr=pr)
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
