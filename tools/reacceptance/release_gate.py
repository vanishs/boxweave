"""Fail-closed publication authorization for a privately installed exact candidate.

Only a signed, externally supplied Verifier receipt can authorize release. This
module does not create signatures, approve itself, or publish a release.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
from zipfile import BadZipFile

from .engine import AcceptanceFault
from bwmini.distribution import InstallError, verify as verify_package

HEX40 = re.compile(r"^[0-9a-f]{40}$")
HEX64 = re.compile(r"^[0-9a-f]{64}$")
MANDATORY = frozenset(("SYS-LIFE-01", "SYS-JOB-01", "SYS-UNKNOWN-01",
                       "SYS-SELFTEST-01", "SYS-UPGRADE-01", "SYS-RELEASE-01",
                       "SYS-REPORT-01"))


def sha(data):
    return hashlib.sha256(data).hexdigest()


def check_signature(attestation, signature, public_key):
    if not all(Path(x).is_file() and not Path(x).is_symlink()
               for x in (attestation, signature, public_key)):
        raise AcceptanceFault("VERIFIER_SIGNATURE_INPUT_UNAVAILABLE")
    if not 0 < Path(signature).stat().st_size < 32768:
        raise AcceptanceFault("VERIFIER_SIGNATURE_INVALID")
    if not 0 < Path(public_key).stat().st_size < 32768:
        raise AcceptanceFault("VERIFIER_PUBLIC_KEY_INVALID")
    try:
        checked = subprocess.run(
            ["openssl", "dgst", "-sha256", "-verify", str(public_key),
             "-signature", str(signature), str(attestation)],
            capture_output=True, timeout=10, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise AcceptanceFault("VERIFIER_SIGNATURE_CHECK_UNAVAILABLE") from None
    if checked.returncode != 0 or b"Verified OK" not in checked.stdout:
        raise AcceptanceFault("VERIFIER_SIGNATURE_NOT_TRUSTED")


def attested_case_bytes(receipt, sid, row, finder, verifier, candidate_digest):
    # Signed JSON binds actual original/oracle/report bytes, not free hashes.
    # Evidence paths are fixed by the verifier and never read from case JSON.
    base = Path(receipt).parent / "cases" / sid
    for key, filename, expected_actor in (
            ("original_attempt_sha256", "original-attempt.json", finder["actor_id"]),
            ("oracle_sha256", "oracle.json", verifier["actor_id"]),
            ("installed_report_sha256", "installed-report.json", verifier["actor_id"])):
        path = base / filename
        if (not path.is_file() or path.is_symlink() or base.is_symlink()
                or path.stat().st_size > 4_194_304):
            raise AcceptanceFault("SCENARIO_EVIDENCE_FILE_MISSING_" + sid)
        data = path.read_bytes()
        if sha(data) != row[key]:
            raise AcceptanceFault("SCENARIO_EVIDENCE_BYTES_MISMATCH_" + sid)
        try:
            content = json.loads(data)
        except ValueError:
            raise AcceptanceFault("SCENARIO_EVIDENCE_JSON_INVALID_" + sid) from None
        if (content.get("scenario_id") != sid or
                content.get("actor_id") != expected_actor):
            raise AcceptanceFault("SCENARIO_EVIDENCE_ACTOR_MISMATCH_" + sid)
        if key != "original_attempt_sha256" and content.get("candidate_sha256") != candidate_digest:
            raise AcceptanceFault("SCENARIO_EVIDENCE_CANDIDATE_MISMATCH_" + sid)


def accept(candidate, release_manifest, receipt, signature, public_key, source,
           required=None):
    if not isinstance(source, str) or not HEX40.fullmatch(source):
        raise AcceptanceFault("RELEASE_SOURCE_INVALID")
    if not required:
        required = MANDATORY
    required = set(required)
    if not required or not all(re.fullmatch(r"SYS-[A-Z]+-\d{2}", x) for x in required):
        raise AcceptanceFault("RELEASE_REQUIRED_SCENARIOS_INVALID")
    zip_path = Path(candidate)
    if not zip_path.is_file() or zip_path.is_symlink():
        raise AcceptanceFault("PRIVATE_CANDIDATE_MISSING")
    data = zip_path.read_bytes()
    digest = sha(data)
    try:
        package = verify_package(zip_path, digest)
    except (InstallError, ValueError, KeyError, OSError, BadZipFile) as exc:
        raise AcceptanceFault("RELEASE_CANDIDATE_INVALID") from exc
    if package["source"] != {"commit": source, "state": "clean"}:
        raise AcceptanceFault("RELEASE_CANDIDATE_SOURCE_MISMATCH")
    manifest = json.loads(Path(release_manifest).read_text())
    if manifest.get("source") != source or manifest.get("files", {}).get("boxweave.zip") != digest:
        raise AcceptanceFault("RELEASE_MANIFEST_DIGEST_MISMATCH")
    check_signature(receipt, signature, public_key)
    evidence = json.loads(Path(receipt).read_text())
    if evidence.get("schema_version") != 1 or evidence.get("decision") != "APPROVED":
        raise AcceptanceFault("INDEPENDENT_RELEASE_APPROVAL_MISSING")
    if evidence.get("source_sha") != source or evidence.get("candidate_sha256") != digest:
        raise AcceptanceFault("VERIFIER_CANDIDATE_IDENTITY_MISMATCH")
    finder = evidence.get("finder")
    verifier = evidence.get("verifier")
    if not isinstance(finder, dict) or not isinstance(verifier, dict):
        raise AcceptanceFault("INDEPENDENT_ACTOR_EVIDENCE_REQUIRED")
    for key in ("actor_id", "box_uuid", "created_at"):
        if not finder.get(key) or not verifier.get(key):
            raise AcceptanceFault("INDEPENDENT_ACTOR_IDENTITY_INCOMPLETE")
    if finder["actor_id"] == verifier["actor_id"] or finder["box_uuid"] == verifier["box_uuid"]:
        raise AcceptanceFault("SAME_ACTOR_OR_SAME_BOX_VERIFIER_REFUSED")
    if not verifier.get("clean_installed_candidate"):
        raise AcceptanceFault("VERIFIER_CLEAN_INSTALL_UNPROVEN")
    if verifier.get("installed_candidate_sha256") != digest:
        raise AcceptanceFault("VERIFIER_INSTALLED_BYTES_MISMATCH")
    scenarios = evidence.get("scenarios")
    if not isinstance(scenarios, dict):
        raise AcceptanceFault("VERIFIER_SCENARIO_MATRIX_MISSING")
    for sid in sorted(required):
        row = scenarios.get(sid)
        if not isinstance(row, dict) or row.get("status") != "PASS":
            raise AcceptanceFault("REQUIRED_SCENARIO_NOT_VERIFIED_" + sid)
        if not all(isinstance(row.get(field), str) and HEX64.fullmatch(row[field])
                   for field in ("original_attempt_sha256", "oracle_sha256",
                                 "installed_report_sha256")):
            raise AcceptanceFault("REQUIRED_SCENARIO_PROOF_MISSING_" + sid)
        if row.get("verifier_actor_id") != verifier["actor_id"]:
            raise AcceptanceFault("SCENARIO_NOT_RUN_BY_VERIFIER_" + sid)
        attested_case_bytes(receipt, sid, row, finder, verifier, digest)
    return {"decision": "APPROVED_FOR_THIS_EXACT_CANDIDATE_ONLY",
            "source_sha": source, "candidate_sha256": digest,
            "attestation_sha256": sha(Path(receipt).read_bytes()),
            "required_scenarios": sorted(required),
            "verifier_actor_id": verifier["actor_id"],
            "public_release_not_yet_verified": True}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--candidate", required=True)
    p.add_argument("--release-manifest", required=True)
    p.add_argument("--attestation", required=True)
    p.add_argument("--signature", required=True)
    p.add_argument("--public-key", required=True)
    p.add_argument("--source", required=True)
    args = p.parse_args(argv)
    try:
        out = accept(args.candidate, args.release_manifest, args.attestation,
                     args.signature, args.public_key, args.source)
        print(json.dumps(out, sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, KeyError) as exc:
        print(json.dumps({"decision": "DENIED", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
