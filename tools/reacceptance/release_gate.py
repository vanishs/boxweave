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
import uuid
from zipfile import BadZipFile, ZipFile

from bwmini.online import RELEASE_FILES

from .engine import AcceptanceFault
from bwmini.distribution import InstallError, verify as verify_package

HEX40 = re.compile(r"^[0-9a-f]{40}$")
HEX64 = re.compile(r"^[0-9a-f]{64}$")

# A signature authenticates an actor; it does not turn a partial graph into a
# completed scenario. These named nodes are the minimum prepublication graph.
REQUIRED_NODES = {
    "SYS-LIFE-01": frozenset(("create", "cli_provider_read", "resize",
        "native_limits", "cli_stop", "mcp_start", "guest_exec", "delete", "no_residue")),
    "SYS-JOB-01": frozenset(("submit", "second_actor_dedup", "body_conflict",
        "original_receipt", "native_effect_once")),
    "SYS-UNKNOWN-01": frozenset(("dispatch", "receipt_loss", "original_read",
        "native_read", "no_replay", "frozen_identity")),
    "SYS-SELFTEST-01": frozenset(("original_start", "inflight_barrier",
        "second_actor_refused", "original_cleanup", "subsequent_run", "native_no_residue")),
    "SYS-UPGRADE-01": frozenset(("baseline_install", "candidate_upgrade",
        "service_restart", "cli_web_mcp_provider_read", "persisted_marker", "rollback_failure")),
    "SYS-RELEASE-01": frozenset(("failed_candidate_denied", "exact_candidate_installed",
        "independent_repro", "digest_bound_approval")),
    "SYS-REPORT-01": frozenset(("false_success_rejected", "tampered_log_rejected",
        "dirty_source_rejected")),
    "SYS-LIFE-02": frozenset(("owned_create", "two_actor_barrier", "conflicting_mutations",
        "stale_uuid_denied", "replacement_protected", "native_effect_count", "unknown_frozen")),
    "SYS-COORD-01": frozenset(("isolated_protection_enabled", "two_actor_cross_gateway",
        "inflight_claim", "winner_effect", "competitor_conflict", "native_denied_no_effect",
        "unrelated_box_progress")),
    "SYS-UPSTREAM-01": frozenset(("running_daemon_identity", "cli_mcp_identities",
        "baseline_difference", "immutable_upstream_ref", "pinned_route_contract",
        "live_mcp_rest_native_comparison")),
    "SYS-COMPOSE-01": frozenset(("owned_compose_file", "mcp_discovery",
        "native_capability", "handler_route_comparison", "native_stack_state", "no_surprise_resources")),
    "SYS-AUTH-01": frozenset(("anonymous_read_denied", "anonymous_write_denied", "login",
        "csrf_denied", "origin_denied", "authorized_effect", "logout", "stale_session_denied",
        "native_denied_no_effect")),
    "SYS-SSH-01": frozenset(("authorized_client_registration", "connect_exec", "native_guest_marker",
        "host_hop_loss", "bounded_recovery", "transport_identity_comparison", "native_attempt_count")),
    "SYS-REST-01": frozenset(("mcp_mutation", "rest_cli_native_read", "reverse_interface_order",
        "missing_identity_denied", "stale_incarnation_denied", "native_state_agreement")),
    "SYS-WEB-PLAN-01": frozenset(("two_browser_actors", "plan_revision", "competing_confirmations",
        "worker_receipt", "native_exactly_one_effect", "cli_mcp_read_agreement")),
    "SYS-NEST-01": frozenset(("outer_owned_host", "distinct_inner_workload", "inner_mcp_exec",
        "parent_stop_start", "persisted_guest_marker", "inner_cleanup", "parent_cleanup", "native_no_orphans")),
    "SYS-NET-01": frozenset(("owned_box", "supported_exposure", "anonymous_reachability",
        "authorized_reachability", "native_route_state", "revoke", "native_no_exposure", "cleanup")),
    "SYS-SEC-01": frozenset(("disabled_module_read", "unauthorized_mutation_denied",
        "invalid_policy_denied", "isolated_opt_in", "native_findings", "rollback", "denied_no_effect")),
    "SYS-BACKUP-01": frozenset(("exact_capability", "owned_backup", "restore", "reopen",
        "native_restored_marker", "restored_identity_timestamps", "rollback", "owned_cleanup")),
    "SYS-DIFF-01": frozenset(("older_release_identity", "candidate_identity", "actual_upstream_identity",
        "changed_supported_path", "both_original_attempts", "same_independent_oracle", "behavior_comparison")),
}
FULL_SCENARIOS = frozenset(REQUIRED_NODES)
MANDATORY = frozenset(("SYS-LIFE-01", "SYS-JOB-01", "SYS-UNKNOWN-01",
                       "SYS-SELFTEST-01", "SYS-UPGRADE-01", "SYS-RELEASE-01",
                       "SYS-REPORT-01"))


def json_object(path, reason, limit=4_194_304):
    path = Path(path)
    if not path.is_file() or path.is_symlink() or not 0 < path.stat().st_size <= limit:
        raise AcceptanceFault(reason)
    def unique_pairs(pairs):
        value = {}
        for key, item in pairs:
            if key in value:
                raise AcceptanceFault(reason + "_DUPLICATE_KEY")
            value[key] = item
        return value
    try:
        result = json.loads(path.read_bytes(), object_pairs_hook=unique_pairs)
    except (ValueError, UnicodeError):
        raise AcceptanceFault(reason) from None
    if not isinstance(result, dict):
        raise AcceptanceFault(reason)
    return result


def validate_release_assets(manifest_file, manifest, package, candidate):
    files = manifest.get("files")
    if (manifest.get("format") != 1 or manifest.get("version") != package.get("version") or
            not isinstance(files, dict) or set(files) != set(RELEASE_FILES)):
        raise AcceptanceFault("RELEASE_MANIFEST_MEMBERS_INVALID")
    root = Path(manifest_file).parent
    with ZipFile(candidate) as archive:
        for name, expected in files.items():
            path = root / name
            if (not isinstance(expected, str) or not HEX64.fullmatch(expected) or
                    not path.is_file() or path.is_symlink() or
                    path.stat().st_size > 180_000_000):
                raise AcceptanceFault("RELEASE_ASSET_INVALID_" + name)
            data = path.read_bytes()
            if sha(data) != expected:
                raise AcceptanceFault("RELEASE_ASSET_DIGEST_MISMATCH_" + name)
            if name != "boxweave.zip":
                try:
                    embedded = archive.read("bwmini/" + name)
                except KeyError:
                    raise AcceptanceFault("RELEASE_HELPER_NOT_IN_CANDIDATE_" + name) from None
                if data != embedded:
                    raise AcceptanceFault("RELEASE_HELPER_CANDIDATE_MISMATCH_" + name)


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
                or base.parent.is_symlink()
                or path.stat().st_size > 4_194_304):
            raise AcceptanceFault("SCENARIO_EVIDENCE_FILE_MISSING_" + sid)
        data = path.read_bytes()
        if sha(data) != row[key]:
            raise AcceptanceFault("SCENARIO_EVIDENCE_BYTES_MISMATCH_" + sid)
        content = json_object(path, "SCENARIO_EVIDENCE_JSON_INVALID_" + sid)
        if (content.get("schema_version") != 1 or
                content.get("scenario_id") != sid or
                content.get("actor_id") != expected_actor):
            raise AcceptanceFault("SCENARIO_EVIDENCE_ACTOR_MISMATCH_" + sid)
        if key != "original_attempt_sha256":
            if content.get("candidate_sha256") != candidate_digest:
                raise AcceptanceFault("SCENARIO_EVIDENCE_CANDIDATE_MISMATCH_" + sid)
            if content.get("status") != "PASS" or content.get("proof_kind") != "installed_blackbox":
                raise AcceptanceFault("SCENARIO_NOT_INSTALLED_PASS_" + sid)
        if key == "original_attempt_sha256":
            process = content.get("process")
            if (not isinstance(process, dict) or
                    process.get("execution") not in ("completed", "UNKNOWN") or
                    not all(isinstance(process.get(field), str) and HEX64.fullmatch(process[field])
                            for field in ("stdout_sha256", "stderr_sha256")) or
                    (process["execution"] == "completed" and type(process.get("exit_code")) is not int)):
                raise AcceptanceFault("ORIGINAL_PROCESS_EVIDENCE_MISSING_" + sid)
        elif key == "oracle_sha256":
            if (content.get("independent_native") is not True or
                    content.get("observed_expected_effect") is not True):
                raise AcceptanceFault("INDEPENDENT_EFFECT_ORACLE_MISSING_" + sid)
        else:
            nodes = content.get("nodes")
            if (content.get("full_graph_complete") is not True or
                    not isinstance(nodes, dict) or
                    not REQUIRED_NODES[sid].issubset(nodes) or
                    any(value != "PASS" for value in nodes.values()) or
                    content.get("unresolved") != []):
                raise AcceptanceFault("SCENARIO_GRAPH_INCOMPLETE_" + sid)


def validate_unsupported(receipt, sid, row, verifier, digest):
    """Accept explicit runtime capability absence, never a failed applicable test."""
    if (row.get("applicability") != "unsupported" or
            not isinstance(row.get("reason"), str) or not 20 <= len(row["reason"]) <= 1024 or
            not isinstance(row.get("capability_evidence_sha256"), str) or
            not HEX64.fullmatch(row["capability_evidence_sha256"])):
        raise AcceptanceFault("UNSUPPORTED_CAPABILITY_EVIDENCE_REQUIRED_" + sid)
    path = Path(receipt).parent / "cases" / sid / "capability.json"
    if path.parent.is_symlink() or path.parent.parent.is_symlink():
        raise AcceptanceFault("UNSUPPORTED_CAPABILITY_PATH_INVALID_" + sid)
    proof = json_object(path, "UNSUPPORTED_CAPABILITY_PROOF_INVALID_" + sid)
    if sha(path.read_bytes()) != row["capability_evidence_sha256"]:
        raise AcceptanceFault("UNSUPPORTED_CAPABILITY_BYTES_MISMATCH_" + sid)
    if (proof.get("schema_version") != 1 or proof.get("scenario_id") != sid or
            proof.get("actor_id") != verifier["actor_id"] or
            proof.get("candidate_sha256") != digest or
            proof.get("result") != "UNSUPPORTED" or proof.get("independent_native") is not True or
            proof.get("channel") not in ("native_cli", "native_rest", "native_mcp") or
            not isinstance(proof.get("capability"), str) or
            not 3 <= len(proof["capability"]) <= 256 or
            not isinstance(proof.get("observed_response_sha256"), str) or
            not HEX64.fullmatch(proof["observed_response_sha256"])):
        raise AcceptanceFault("UNSUPPORTED_CAPABILITY_NOT_OBSERVED_" + sid)
    response_path = path.parent / "capability-response.json"
    response = json_object(response_path, "UNSUPPORTED_NATIVE_RESPONSE_MISSING_" + sid)
    if sha(response_path.read_bytes()) != proof["observed_response_sha256"]:
        raise AcceptanceFault("UNSUPPORTED_NATIVE_RESPONSE_BYTES_MISMATCH_" + sid)
    if (response.get("execution") != "completed" or type(response.get("exit_code")) is not int or
            not all(isinstance(response.get(field), str) and HEX64.fullmatch(response[field])
                    for field in ("stdout_sha256", "stderr_sha256"))):
        raise AcceptanceFault("UNSUPPORTED_NATIVE_PROCESS_INCOMPLETE_" + sid)
    for channel in ("stdout", "stderr"):
        value = response.get(channel)
        if (not isinstance(value, str) or
                sha(value.encode("utf-8")) != response[channel + "_sha256"]):
            raise AcceptanceFault("UNSUPPORTED_NATIVE_OUTPUT_BYTES_MISMATCH_" + sid)


def validate_attestation(receipt, evidence, source, digest, manifest_digest,
                         *, expected_context=None):
    """Validate semantic completeness before signing and after authentication.

    This does not establish trust on its own. Callers must authenticate a
    protected producer and preserve its original evidence independently.
    """
    if evidence.get("schema_version") != 1 or evidence.get("decision") != "APPROVED":
        raise AcceptanceFault("INDEPENDENT_RELEASE_APPROVAL_MISSING")
    if evidence.get("source_sha") != source or evidence.get("candidate_sha256") != digest:
        raise AcceptanceFault("VERIFIER_CANDIDATE_IDENTITY_MISMATCH")
    if evidence.get("release_manifest_sha256") != manifest_digest:
        raise AcceptanceFault("VERIFIER_RELEASE_MANIFEST_MISMATCH")
    finder = evidence.get("finder")
    verifier = evidence.get("verifier")
    if not isinstance(finder, dict) or not isinstance(verifier, dict):
        raise AcceptanceFault("INDEPENDENT_ACTOR_EVIDENCE_REQUIRED")
    for identity in (finder, verifier):
        for key in ("actor_id", "box_uuid", "created_at"):
            value = identity.get(key)
            if not isinstance(value, str) or not 0 < len(value) <= 128:
                raise AcceptanceFault("INDEPENDENT_ACTOR_IDENTITY_INCOMPLETE")
        if not re.fullmatch(r"[A-Za-z0-9_.:-]{1,128}", identity["actor_id"]):
            raise AcceptanceFault("INDEPENDENT_ACTOR_IDENTITY_INVALID")
        try:
            if str(uuid.UUID(identity["box_uuid"])) != identity["box_uuid"]:
                raise ValueError("noncanonical UUID")
        except ValueError:
            raise AcceptanceFault("INDEPENDENT_BOX_UUID_INVALID") from None
    if finder["actor_id"] == verifier["actor_id"] or finder["box_uuid"] == verifier["box_uuid"]:
        raise AcceptanceFault("SAME_ACTOR_OR_SAME_BOX_VERIFIER_REFUSED")
    if verifier.get("clean_installed_candidate") is not True:
        raise AcceptanceFault("VERIFIER_CLEAN_INSTALL_UNPROVEN")
    if verifier.get("installed_candidate_sha256") != digest:
        raise AcceptanceFault("VERIFIER_INSTALLED_BYTES_MISMATCH")
    scenarios = evidence.get("scenarios")
    if not isinstance(scenarios, dict):
        raise AcceptanceFault("VERIFIER_SCENARIO_MATRIX_MISSING")
    if set(scenarios) != FULL_SCENARIOS:
        raise AcceptanceFault("FULL_SCENARIO_ACCOUNTING_REQUIRED")
    for sid in sorted(FULL_SCENARIOS):
        row = scenarios.get(sid)
        if not isinstance(row, dict):
            raise AcceptanceFault("REQUIRED_SCENARIO_NOT_VERIFIED_" + sid)
        if sid not in MANDATORY and row.get("status") == "BLOCKED":
            validate_unsupported(receipt, sid, row, verifier, digest)
            continue
        if row.get("status") != "PASS":
            raise AcceptanceFault("REQUIRED_SCENARIO_NOT_VERIFIED_" + sid)
        if not all(isinstance(row.get(field), str) and HEX64.fullmatch(row[field])
                   for field in ("original_attempt_sha256", "oracle_sha256",
                                 "installed_report_sha256")):
            raise AcceptanceFault("REQUIRED_SCENARIO_PROOF_MISSING_" + sid)
        if row.get("verifier_actor_id") != verifier["actor_id"]:
            raise AcceptanceFault("SCENARIO_NOT_RUN_BY_VERIFIER_" + sid)
        attested_case_bytes(receipt, sid, row, finder, verifier, digest)
    if expected_context is not None:
        numeric_keys = {"candidate_run_id", "candidate_run_attempt", "candidate_artifact_id",
                        "verifier_run_id", "verifier_run_attempt"}
        keys = numeric_keys | {"candidate_artifact_digest"}
        if (not isinstance(expected_context, dict) or set(expected_context) != keys or
                not all(type(expected_context[key]) is int and
                        0 < expected_context[key] < 2 ** 63 for key in numeric_keys) or
                not isinstance(expected_context["candidate_artifact_digest"], str) or
                re.fullmatch(r"sha256:[0-9a-f]{64}",
                             expected_context["candidate_artifact_digest"]) is None):
            raise AcceptanceFault("VERIFIER_EXPECTED_WORKFLOW_CONTEXT_INVALID")
        workflow = evidence.get("workflow")
        if (not isinstance(workflow, dict) or
                not all(type(workflow.get(key)) is int for key in numeric_keys) or
                any(workflow.get(key) != expected_context[key] for key in keys)):
            raise AcceptanceFault("VERIFIER_WORKFLOW_CONTEXT_MISMATCH")
    return evidence


def accept(candidate, release_manifest, receipt, signature, public_key, source,
           required=None, *, expected_context=None):
    if not isinstance(source, str) or not HEX40.fullmatch(source):
        raise AcceptanceFault("RELEASE_SOURCE_INVALID")
    if required is not None and set(required) != MANDATORY:
        raise AcceptanceFault("RELEASE_REQUIRED_SCENARIOS_CANNOT_BE_WEAKENED")
    required = MANDATORY
    zip_path = Path(candidate)
    if not zip_path.is_file() or zip_path.is_symlink():
        raise AcceptanceFault("PRIVATE_CANDIDATE_MISSING")
    if zip_path.stat().st_size > 180_000_000:
        raise AcceptanceFault("PRIVATE_CANDIDATE_TOO_LARGE")
    data = zip_path.read_bytes()
    digest = sha(data)
    try:
        package = verify_package(zip_path, digest)
    except (InstallError, ValueError, KeyError, OSError, BadZipFile) as exc:
        raise AcceptanceFault("RELEASE_CANDIDATE_INVALID") from exc
    if package["source"] != {"commit": source, "state": "clean"}:
        raise AcceptanceFault("RELEASE_CANDIDATE_SOURCE_MISMATCH")
    manifest = json_object(release_manifest, "RELEASE_MANIFEST_INVALID")
    if manifest.get("source") != source or manifest.get("files", {}).get("boxweave.zip") != digest:
        raise AcceptanceFault("RELEASE_MANIFEST_DIGEST_MISMATCH")
    validate_release_assets(release_manifest, manifest, package, zip_path)
    check_signature(receipt, signature, public_key)
    evidence = json_object(receipt, "VERIFIER_ATTESTATION_INVALID")
    validate_attestation(receipt, evidence, source, digest,
                         sha(Path(release_manifest).read_bytes()), expected_context=expected_context)
    verifier = evidence["verifier"]
    unsupported = sorted(sid for sid, row in evidence["scenarios"].items()
                         if row.get("status") == "BLOCKED")
    return {"decision": "APPROVED_FOR_THIS_EXACT_CANDIDATE_ONLY",
            "source_sha": source, "candidate_sha256": digest,
            "attestation_sha256": sha(Path(receipt).read_bytes()),
            "release_manifest_sha256": sha(Path(release_manifest).read_bytes()),
            "required_scenarios": sorted(required),
            "scope": ("SUPPORTED_SCENARIOS_ACCEPTED_UNSUPPORTED_FEATURES_EXCLUDED" if unsupported
                      else "FULL_CANONICAL_SCENARIO_MATRIX"),
            "unsupported_scenarios": unsupported,
            "full_system_acceptance": not unsupported,
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
    p.add_argument("--expected-context", help="Trusted GitHub run/artifact metadata JSON")
    args = p.parse_args(argv)
    try:
        out = accept(args.candidate, args.release_manifest, args.attestation,
                     args.signature, args.public_key, args.source,
                     expected_context=(json_object(args.expected_context, "WORKFLOW_CONTEXT_INVALID")
                                       if args.expected_context else None))
        print(json.dumps(out, sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, KeyError) as exc:
        print(json.dumps({"decision": "DENIED", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
