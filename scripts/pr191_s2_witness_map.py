#!/usr/bin/env python3
"""Read-only lane-local crosswalk check, never native/phase acceptance."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re

from pr191_n2_evidence_identity import evaluate as n2_identity
from s2_p3_evidence_audit import MANDATORY, LIST_WITNESS

HERE = Path(__file__).resolve().parent


def load_map():
    rows = json.loads((HERE / "pr191_s2_witness_map.json").read_text())
    original = set(MANDATORY)
    for case in json.loads((HERE / "s2_p3_scoped_acceptance.json").read_text()):
        original.update(case["required_markers"])
    names = [old for row in rows for old in row["old"]]
    if len(names) != 72 or len(set(names)) != 72 or set(names) != original:
        raise ValueError("incomplete or duplicate original crosswalk")
    direct = 0
    for row in rows:
        if not row["requires"] or any(not patterns for patterns in row["requires"].values()):
            raise ValueError("empty lane requirement")
        if "disposition" not in row:
            direct += len(row["old"])
            if len(row["old"]) != 1 or len(row["requires"]) != 1 or list(row["requires"].values()) != [[row["old"][0] + "=PASS"]]:
                raise ValueError("nonidentical direct marker")
        elif row["disposition"] != "SCOPED_REPLACEMENT_REVIEW_REQUIRED":
            raise ValueError("unknown replacement disposition")
    if direct != 48:
        raise ValueError("direct/changed crosswalk mismatch")
    return rows


def requirements(rows, lane):
    return sorted({pattern for row in rows for pattern in row["requires"].get(lane, [])})


def check_original_lineage(lines, lane):
    """Reject mixed originals even when all individual marker shapes match."""
    def one(prefix):
        hits = [(i, line.split()) for i, line in enumerate(lines) if line.startswith(prefix+" ")]
        if len(hits) != 1:
            raise ValueError("missing or ambiguous lineage "+prefix)
        return hits[0]

    if lane in ("scoped-create-ack", "scoped-delete-ack"):
        hits = [line for line in lines if line.startswith("P1B_SCOPED_ACK_ORIGINAL_RETAINED ")]
        if len(hits) != 2 or hits[0] != hits[1]:
            raise ValueError("ACK original not retained identically twice")
    elif re.fullmatch(r"scoped-(create|delete|exec)-wal-(before|after)",lane):
        cut_i, cut = one("P1B_SCOPED_WAL_EXACT_CONTROLLER_CUT=PASS")
        proof_i, proof = one("P1B_SCOPED_WAL_SQLITE_NATIVE_CROSSCHECK=PASS")
        done_i, done = one("P1B_SCOPED_INSTALLED_WAL_CUT=PASS")
        if not cut_i < proof_i < done_i or cut[1] != proof[1] or cut[1] != done[1] or cut[2] != proof[3]:
            raise ValueError("WAL cut and crosscheck original mismatch")
        if lane.startswith("scoped-exec-"):
            final_i, final = one("P1B_SCOPED_EXEC_WAL_ATOMIC=PASS")
            if not proof_i < final_i < done_i or final[1] != proof[1] or final[3:5] != proof[3:5]:
                raise ValueError("Exec WAL final original/native mismatch")
            expected = "UNKNOWN" if proof[2] == "VERIFYING" else "SUCCEEDED"
            if final[2] != expected:
                raise ValueError("Exec WAL final state contradicts pre-restart state")
    elif re.fullmatch(r"scoped-update-ack-wal-(before|after)",lane):
        cut_i, cut = one("P1B_SCOPED_UPDATE_NATIVE_ACK_WAL_CUT=PASS")
        restart_i, restart = one("P1B_SCOPED_UPDATE_NATIVE_ACK_WAL_RESTART_RETAINED=PASS")
        if not cut_i < restart_i or cut[1:3] != restart[1:3]:
            raise ValueError("Update ACK WAL original mismatch")


def evaluate(raw, source, lane, rows=None):
    rows = load_map() if rows is None else rows
    if not re.fullmatch(r"[0-9a-f]{40}", source):
        raise ValueError("invalid exact source")
    if not raw or len(raw) > 32 * 1024 * 1024:
        raise ValueError("missing or oversized transcript")
    lines = raw.decode("utf-8", "strict").splitlines()
    if lane == "s25":
        if [x for x in lines if x.startswith("S25_LAB_SOURCE_SHA=")] != ["S25_LAB_SOURCE_SHA=" + source]:
            raise ValueError("S25 source mismatch")
        digests = [x.split("=",1)[1] for x in lines if x.startswith("S25_LAB_CANDIDATE_SHA256=")]
        if len(digests) != 1 or not re.fullmatch(r"[0-9a-f]{64}", digests[0]):
            raise ValueError("S25 candidate identity missing or ambiguous")
        identity = {"source": source, "candidate_sha256": digests[0],
                    "transcript_sha256": hashlib.sha256(raw).hexdigest()}
    else:
        identity = n2_identity(raw, source)
        if [x for x in lines if x.startswith("N2_EVIDENCE_LANE=")] != ["N2_EVIDENCE_LANE=" + lane]:
            raise ValueError("N2 lane mismatch")
    patterns = requirements(rows, lane)
    if not patterns:
        raise ValueError("no crosswalk requirements for lane")
    missing = [p for p in patterns if not (any(re.fullmatch(p[3:], x) for x in lines) if p.startswith("re:") else p in lines)]
    if lane == "scoped-auth-races":
        prefixes = tuple(w.split("=")[0] + "=" for w in LIST_WITNESS)
        actual = tuple(x for x in lines if x.startswith(prefixes))
        if actual != LIST_WITNESS:
            missing.append("exact ordered operation_list witness")
    if missing:
        raise ValueError("missing lane witnesses: " + ", ".join(missing))
    check_original_lineage(lines,lane)
    return {"lane": lane, **identity, "requirements_observed": len(patterns),
            "scope": "LANE_LOG_CHECK_ONLY", "acceptance": "INDEPENDENT_REVIEW_REQUIRED"}


def evaluate_set(logs, source):
    """Index distinct original lane transcripts. Caller verifies CI provenance."""
    rows = load_map()
    expected = {lane for row in rows for lane in row["requires"]}
    if set(logs) != expected:
        raise ValueError("lane set mismatch: missing=" + ",".join(sorted(expected-set(logs))) +
                         " unexpected=" + ",".join(sorted(set(logs)-expected)))
    reports = [evaluate(logs[lane], source, lane, rows) for lane in sorted(expected)]
    if len({r["transcript_sha256"] for r in reports}) != len(reports):
        raise ValueError("reused transcript")
    return {
        "source": source, "lanes": reports,
        "original_marker_rows": sum(len(r["old"]) for r in rows),
        "replacement_review_required": [m for r in rows if "disposition" in r for m in r["old"]],
        "n2_binary_digests": sorted({r["candidate_sha256"] for r in reports if r["lane"] != "s25"}),
        "scope": "COMPLETE_CROSSWALK_LOG_INDEX_ONLY",
        "acceptance": "VERIFY_TRUSTED_CI_JOBS_AND_REPLACEMENT_SEMANTICS",
        "public_wan": "SEPARATE_C04_GATE_NOT_INFERRED",
    }


def ci_receipt(report, env):
    """Sanitized CI locator, not authenticated CI conclusion or acceptance."""
    if env.get("GITHUB_ACTIONS") != "true" or env.get("RUNNER_ENVIRONMENT") != "github-hosted":
        raise ValueError("CI receipt requires a declared hosted Actions context")
    repo = env.get("GITHUB_REPOSITORY", "")
    ref = env.get("GITHUB_WORKFLOW_REF", "")
    job = env.get("GITHUB_JOB", "")
    event = env.get("GITHUB_EVENT_NAME", "")
    event_sha = env.get("GITHUB_SHA", "")
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo):
        raise ValueError("invalid repository context")
    if report["lane"] == "s25":
        routes = {"s25-bridgehead-lab": "v1-pr191-s25-scoped.yml",
                  "native-s25-gate": "v1-release.yml"}
    else:
        routes = {"installed-lifecycle": "v1-n2-installed-lifecycle.yml"}
    path, sep, gitref = ref.partition("@")
    if job not in routes or path != repo+"/.github/workflows/"+routes[job] or not sep or not gitref.startswith("refs/") or not re.fullmatch(r"[A-Za-z0-9_./-]{6,300}",gitref):
        raise ValueError("wrong workflow or job identity")
    if event not in ("pull_request","push","workflow_dispatch") or not re.fullmatch(r"[0-9a-f]{40}",event_sha):
        raise ValueError("invalid event identity")
    if event != "pull_request" and report["source"] != event_sha:
        raise ValueError("non-PR event source mismatch")
    def number(key):
        value = env.get(key,"")
        if not re.fullmatch(r"[1-9][0-9]{0,19}",value):
            raise ValueError("invalid "+key)
        return int(value)
    return {"schema":1, "kind":"DECLARED_CI_LANE_LOG_INDEX",
            "map_sha256":hashlib.sha256((HERE/"pr191_s2_witness_map.json").read_bytes()).hexdigest(),
            "report":report,
            "ci":{"repository":repo,"workflow_ref":ref,"job":job,
                  "event":event,"event_sha":event_sha,"run_id":number("GITHUB_RUN_ID"),
                  "run_attempt":number("GITHUB_RUN_ATTEMPT")},
            "conclusion":"VERIFY_ORIGINAL_CI_JOB"}


def evaluate_receipts(receipts, source):
    """Index complete CI-declared receipts; verify their real jobs separately."""
    rows = load_map()
    expected = {lane for row in rows for lane in row["requires"]}
    seen = {}
    for receipt in receipts:
        try:
            report = receipt["report"]
            ci = receipt["ci"]
            lane = report["lane"]
            if lane not in expected or lane in seen:
                raise ValueError("unexpected or repeated lane")
            if set(report) != {"lane","source","candidate_sha256","transcript_sha256",
                               "requirements_observed","scope","acceptance"}:
                raise ValueError("unexpected lane report shape")
            if (report["source"] != source or
                not re.fullmatch(r"[0-9a-f]{40}",source) or
                not all(re.fullmatch(r"[0-9a-f]{64}",report[k]) for k in ("candidate_sha256","transcript_sha256")) or
                type(report["requirements_observed"]) is not int or
                report["requirements_observed"] != len(requirements(rows,lane)) or
                report["scope"] != "LANE_LOG_CHECK_ONLY" or
                report["acceptance"] != "INDEPENDENT_REVIEW_REQUIRED"):
                raise ValueError("invalid or mismatched report")
            env = {"GITHUB_ACTIONS":"true","RUNNER_ENVIRONMENT":"github-hosted",
                   "GITHUB_REPOSITORY":ci["repository"],"GITHUB_WORKFLOW_REF":ci["workflow_ref"],
                   "GITHUB_JOB":ci["job"],"GITHUB_EVENT_NAME":ci["event"],
                   "GITHUB_SHA":ci["event_sha"],"GITHUB_RUN_ID":str(ci["run_id"]),
                   "GITHUB_RUN_ATTEMPT":str(ci["run_attempt"])}
            if ci_receipt(report,env) != receipt:
                raise ValueError("receipt metadata or mapping mismatch")
            seen[lane] = receipt
        except (KeyError, TypeError, AttributeError) as exc:
            raise ValueError("malformed lane receipt") from exc
    if set(seen) != expected:
        raise ValueError("missing lane receipts: "+",".join(sorted(expected-set(seen))))
    if len({r["ci"]["repository"] for r in seen.values()}) != 1:
        raise ValueError("mixed repositories")
    if len({r["report"]["transcript_sha256"] for r in seen.values()}) != len(expected):
        raise ValueError("reused transcript digest")
    return {"source":source,"lanes":[seen[lane] for lane in sorted(expected)],
            "original_marker_rows":72,
            "replacement_review_required":[m for row in rows if "disposition" in row for m in row["old"]],
            "scope":"CI_DECLARED_RECEIPT_INDEX_ONLY",
            "acceptance":"VERIFY_ORIGINAL_CI_JOBS_AND_REPLACEMENT_SEMANTICS",
            "public_wan":"SEPARATE_C04_GATE_NOT_INFERRED"}


def main():
    p = argparse.ArgumentParser()
    files = p.add_mutually_exclusive_group(required=True)
    files.add_argument("--log", type=Path)
    files.add_argument("--directory", type=Path)
    files.add_argument("--receipt-directory",type=Path)
    p.add_argument("--source", required=True)
    p.add_argument("--lane")
    p.add_argument("--ci-receipt", type=Path)
    a = p.parse_args()
    try:
        if a.receipt_directory:
            if a.lane or a.ci_receipt:
                raise ValueError("receipt directory cannot select or write a lane")
            paths = sorted(a.receipt_directory.rglob("*.json"))
            if len(paths) > 23 or any(p.stat().st_size > 8192 for p in paths):
                raise ValueError("unexpected receipt count or size")
            result = evaluate_receipts([json.loads(p.read_text()) for p in paths],a.source)
        elif a.directory:
            if a.lane:
                raise ValueError("directory mode cannot select a lane")
            result = evaluate_set({p.stem: p.read_bytes() for p in a.directory.glob("*.log")}, a.source)
        else:
            if not a.lane:
                raise ValueError("single-log mode requires lane")
            result = evaluate(a.log.read_bytes(), a.source, a.lane)
        if a.ci_receipt:
            if not a.log:
                raise ValueError("CI receipt requires single original transcript")
            receipt = ci_receipt(result, os.environ)
            with a.ci_receipt.open("x",encoding="utf-8") as out:
                json.dump(receipt,out,sort_keys=True)
                out.write("\n")
        print(json.dumps(result, sort_keys=True))
    except (OSError, ValueError) as exc:
        raise SystemExit(str(exc))


if __name__ == "__main__":
    main()
