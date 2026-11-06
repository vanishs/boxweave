"""Carry real independent Box evidence to the official release workflow.

This developer-only reader neither executes tests nor signs their results.
The authorized repository workflow authenticates the submission; original
Agent/Box observations remain the basis for independent acceptance claims.
"""
from __future__ import annotations

import argparse
import base64
import copy
import io
import json
from pathlib import Path
import stat
import sys
import zipfile

from bwmini.distribution import InstallError, verify as verify_package
from .engine import AcceptanceFault
from .release_gate import (FULL_SCENARIOS, HEX40, json_object, sha,
                           validate_attestation, validate_release_assets)
from .signing_preflight import prepare as preflight
from .workflow_gate import asset_index

MAX_ENCODED = 55_000
MAX_FILE = 524_288
MAX_TOTAL = 4_194_304
MAX_ZIP = MAX_TOTAL + 131_072
SOLID_MEMBER = "box-evidence.zip"
ROOT_FILES = frozenset(("release-gate.json", "original-receipt.json",
                       "submitted-receipt.json", "submission.json", "recording.json"))
CASE_FILES = frozenset(("original-attempt.json", "oracle.json", "installed-report.json",
                       "capability.json", "capability-response.json"))


def require(condition, reason):
    if not condition:
        raise AcceptanceFault(reason)


def allowed(name):
    if name in ROOT_FILES:
        return True
    parts = name.split("/")
    return (len(parts) == 3 and parts[0] == "cases" and
            parts[1] in FULL_SCENARIOS and parts[2] in CASE_FILES)


def object_bytes(data, reason):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            require(key not in result, reason + "_DUPLICATE_KEY")
            result[key] = value
        return result
    try:
        value = json.loads(data, object_pairs_hook=unique)
    except (ValueError, UnicodeError):
        raise AcceptanceFault(reason) from None
    require(isinstance(value, dict), reason)
    return value


def encoded_object(value):
    return (json.dumps(value, sort_keys=True, separators=(",", ":")) + "\n").encode()


def directory_files(root):
    root = Path(root)
    require(root.is_dir() and not root.is_symlink(), "BOX_EVIDENCE_DIRECTORY_INVALID")
    files = {}
    for path in root.rglob("*"):
        require(not path.is_symlink(), "BOX_EVIDENCE_SYMLINK_REFUSED")
        if path.is_dir():
            parts = path.relative_to(root).parts
            require(parts == ("cases",) or
                    (len(parts) == 2 and parts[0] == "cases" and parts[1] in FULL_SCENARIOS),
                    "BOX_EVIDENCE_EXTRA_DIRECTORY")
            continue
        name = path.relative_to(root).as_posix()
        st = path.lstat()
        require(allowed(name) and stat.S_ISREG(st.st_mode) and st.st_nlink == 1,
                "BOX_EVIDENCE_EXTRA_OR_UNSAFE_FILE")
        require(0 < st.st_size <= MAX_FILE, "BOX_EVIDENCE_FILE_TOO_LARGE")
        data = path.read_bytes()
        require(len(data) == st.st_size, "BOX_EVIDENCE_FILE_CHANGED")
        object_bytes(data, "BOX_EVIDENCE_JSON_INVALID")
        files[name] = data
        require(sum(map(len, files.values())) <= MAX_TOTAL, "BOX_EVIDENCE_TOTAL_TOO_LARGE")
    require("release-gate.json" in files, "BOX_EVIDENCE_RECEIPT_MISSING")
    return files


def new_output(path):
    path = Path(path)
    require(not path.exists() and not path.is_symlink(), "BOX_ACCEPTANCE_OUTPUT_EXISTS")
    require(not any(parent.is_symlink() for parent in path.absolute().parents),
            "BOX_ACCEPTANCE_OUTPUT_PARENT_SYMLINK")
    return path


def bind_hash(value, key, digest):
    if key in value:
        require(value[key] == digest, "BOX_EVIDENCE_SUPPLIED_DIGEST_MISMATCH_" + key)
    value[key] = digest


def candidate(assets, source):
    require(isinstance(source, str) and HEX40.fullmatch(source), "BOX_ACCEPTANCE_SOURCE_INVALID")
    index = asset_index(assets)
    root = Path(assets)
    package = verify_package(root / "boxweave.zip", index["boxweave.zip"])
    manifest = json_object(root / "release.json", "BOX_ACCEPTANCE_MANIFEST_INVALID")
    require(package.get("source") == {"commit": source, "state": "clean"} and
            manifest.get("source") == source, "BOX_ACCEPTANCE_CANDIDATE_SOURCE_MISMATCH")
    validate_release_assets(root / "release.json", manifest, package, root / "boxweave.zip")
    return index


def pack(assets, evidence, source, candidate_run_id, output):
    """Package existing real observations; never manufacture missing tests."""
    require(type(candidate_run_id) is int and 0 < candidate_run_id < 2**63,
            "BOX_ACCEPTANCE_CANDIDATE_RUN_INVALID")
    files = directory_files(evidence)
    require(not (set(files) & (ROOT_FILES - {"release-gate.json"})),
            "BOX_ACCEPTANCE_ALREADY_PACKAGED_REFUSED")
    original = files["release-gate.json"]
    report = copy.deepcopy(object_bytes(original, "BOX_ACCEPTANCE_REPORT_INVALID"))
    index = candidate(assets, source)
    require(report.get("source_sha") == source and
            report.get("candidate_sha256") == index["boxweave.zip"],
            "BOX_ACCEPTANCE_REPORT_CANDIDATE_MISMATCH")
    bind_hash(report, "release_manifest_sha256", index["release.json"])
    if "release_assets_sha256" in report:
        require(report["release_assets_sha256"] == index, "BOX_ACCEPTANCE_ALL_ASSETS_MISMATCH")
    report["release_assets_sha256"] = index
    require(isinstance(report.get("scenarios"), dict), "BOX_ACCEPTANCE_SCENARIO_MATRIX_INVALID")
    for sid, row in report["scenarios"].items():
        require(sid in FULL_SCENARIOS and isinstance(row, dict), "BOX_ACCEPTANCE_SCENARIO_INVALID")
        if row.get("status") == "PASS":
            for key, name in (("original_attempt_sha256", "original-attempt.json"),
                              ("oracle_sha256", "oracle.json"),
                              ("installed_report_sha256", "installed-report.json")):
                path = "cases/" + sid + "/" + name
                require(path in files, "BOX_ACCEPTANCE_ORIGINAL_FILE_MISSING_" + sid)
                bind_hash(row, key, sha(files[path]))
        elif row.get("status") == "BLOCKED" and row.get("applicability") == "unsupported":
            base = "cases/" + sid + "/"
            require(base + "capability.json" in files and base + "capability-response.json" in files,
                    "BOX_ACCEPTANCE_CAPABILITY_FILE_MISSING_" + sid)
            capability = object_bytes(files[base + "capability.json"], "BOX_CAPABILITY_INVALID")
            # Do not rewrite native observations to repair a mismatched hash.
            require(capability.get("observed_response_sha256") == sha(files[base + "capability-response.json"]),
                    "BOX_ACCEPTANCE_CAPABILITY_RESPONSE_MISMATCH_" + sid)
            bind_hash(row, "capability_evidence_sha256", sha(files[base + "capability.json"]))
    # The eventual evidence-recording run does not exist yet. Its provenance
    # is stamped by that workflow, never by the caller's original report.
    report.pop("workflow", None)
    validate_attestation(Path(evidence) / "release-gate.json", report, source,
                         index["boxweave.zip"], index["release.json"])
    submitted = encoded_object(report)
    files["release-gate.json"] = submitted
    files["original-receipt.json"] = original
    files["submitted-receipt.json"] = submitted
    files["submission.json"] = encoded_object({"schema_version": 1,
        "candidate_run_id": candidate_run_id,
        "files_sha256": {name: sha(data) for name, data in files.items()
                         if name != "release-gate.json"}})
    # A single fixed outer member provides solid compression across the
    # canonical inner files. Only one fixed nesting level is accepted; this
    # preserves exact original bytes without exceeding workflow input limits.
    inner = io.BytesIO()
    with zipfile.ZipFile(inner, "w", compression=zipfile.ZIP_STORED) as archive:
        for name, data in sorted(files.items()):
            info = zipfile.ZipInfo(name)
            info.external_attr = 0o100600 << 16
            archive.writestr(info, data)
    require(len(inner.getvalue()) <= MAX_ZIP, "BOX_ACCEPTANCE_EXPANDED_ARCHIVE_TOO_LARGE")
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        archive.writestr(SOLID_MEMBER, inner.getvalue())
    encoded = base64.b64encode(buffer.getvalue()).decode("ascii")
    require(len(encoded) <= MAX_ENCODED, "BOX_ACCEPTANCE_BUNDLE_TOO_LARGE_USE_SMALLER_SANITIZED_EVIDENCE")
    payload = {"ref": "main", "inputs": {"candidate_run_id": str(candidate_run_id),
                                          "evidence_bundle_b64": encoded}}
    data = encoded_object(payload)
    require(len(data) <= 65_535, "BOX_ACCEPTANCE_WORKFLOW_INPUT_TOO_LARGE")
    with new_output(output).open("xb") as stream:
        stream.write(data)
    return {"decision": "BOX_ACCEPTANCE_SUBMISSION_PACKAGED", "files": len(files),
            "payload_sha256": sha(data), "signature_produced": False, "release_authorized": False,
            "tests_executed_by_this_command": False}


def unpack(encoded, output):
    require(isinstance(encoded, str) and len(encoded) <= MAX_ENCODED + 1 and
            0 < len(encoded.strip()) <= MAX_ENCODED,
            "BOX_ACCEPTANCE_ENCODED_SIZE_INVALID")
    try:
        data = base64.b64decode(encoded.strip(), validate=True)
    except (ValueError, UnicodeError):
        raise AcceptanceFault("BOX_ACCEPTANCE_BASE64_INVALID") from None
    out = new_output(output)
    with zipfile.ZipFile(io.BytesIO(data)) as envelope:
        outer = envelope.infolist()
        if len(outer) == 1 and outer[0].filename == SOLID_MEMBER:
            item = outer[0]
            require(0 < item.file_size <= MAX_ZIP and not item.flag_bits & 1 and
                    item.compress_type in (zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED) and
                    stat.S_IFMT(item.external_attr >> 16) in (0, stat.S_IFREG),
                    "BOX_ACCEPTANCE_OUTER_ARCHIVE_UNSAFE")
            data = envelope.read(item)
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        items = archive.infolist()
        require(0 < len(items) <= len(ROOT_FILES) + len(FULL_SCENARIOS) * len(CASE_FILES),
                "BOX_ACCEPTANCE_MEMBER_COUNT_INVALID")
        names = [item.filename for item in items]
        require(len(names) == len(set(names)) and all(allowed(name) for name in names) and
                "recording.json" not in names, "BOX_ACCEPTANCE_UNSAFE_OR_DUPLICATE_MEMBER")
        require(sum(item.file_size for item in items) <= MAX_TOTAL and all(
                0 < item.file_size <= MAX_FILE and not item.is_dir() and not item.flag_bits & 1 and
                item.compress_type in (zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED) and
                stat.S_IFMT(item.external_attr >> 16) in (0, stat.S_IFREG) for item in items),
                "BOX_ACCEPTANCE_UNSAFE_OR_OVERSIZED_MEMBER")
        files = {item.filename: archive.read(item) for item in items}
    for content in files.values():
        object_bytes(content, "BOX_ACCEPTANCE_ARCHIVE_JSON_INVALID")
    require({"release-gate.json", "original-receipt.json", "submitted-receipt.json", "submission.json"} <= set(files),
            "BOX_ACCEPTANCE_SUBMISSION_FILES_MISSING")
    verify_submission(files)
    out.mkdir(mode=0o700)
    for name, content in files.items():
        path = out / name
        path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        with path.open("xb") as stream:
            stream.write(content)
        path.chmod(0o600)
    return {"decision": "BOX_ACCEPTANCE_BUNDLE_EXTRACTED", "files": len(files),
            "tests_executed_by_this_command": False, "release_authorized": False}


def verify_submission(files):
    require("submission.json" in files and "original-receipt.json" in files and
            "submitted-receipt.json" in files, "BOX_ACCEPTANCE_ORIGINAL_SUBMISSION_MISSING")
    manifest = object_bytes(files["submission.json"], "BOX_ACCEPTANCE_SUBMISSION_INVALID")
    expected = {name: sha(data) for name, data in files.items()
                if name not in ("release-gate.json", "submission.json", "recording.json")}
    require(manifest.get("schema_version") == 1 and
            type(manifest.get("candidate_run_id")) is int and 0 < manifest["candidate_run_id"] < 2**63 and
            manifest.get("files_sha256") == expected,
            "BOX_ACCEPTANCE_SUBMITTED_BYTES_CHANGED")
    submitted = object_bytes(files["submitted-receipt.json"], "BOX_ACCEPTANCE_SUBMITTED_REPORT_INVALID")
    # The transport manifest authenticates no observations by itself. A caller
    # can construct it without running pack(), so independently rederive the
    # narrowly allowed metadata additions from the preserved original report.
    original = object_bytes(files["original-receipt.json"], "BOX_ACCEPTANCE_ORIGINAL_REPORT_INVALID")
    derived = copy.deepcopy(original)
    derived.pop("workflow", None)
    require("workflow" not in submitted, "BOX_ACCEPTANCE_SUBMITTED_WORKFLOW_NOT_AUTHORITATIVE")
    for field in ("release_manifest_sha256", "release_assets_sha256"):
        if field not in derived:
            require(field in submitted, "BOX_ACCEPTANCE_DERIVED_BINDING_MISSING")
            derived[field] = copy.deepcopy(submitted[field])
    require(isinstance(derived.get("scenarios"), dict), "BOX_ACCEPTANCE_ORIGINAL_SCENARIOS_INVALID")
    for sid, row in derived["scenarios"].items():
        require(sid in FULL_SCENARIOS and isinstance(row, dict), "BOX_ACCEPTANCE_ORIGINAL_SCENARIO_INVALID")
        if row.get("status") == "PASS":
            bindings = (("original_attempt_sha256", "original-attempt.json"),
                        ("oracle_sha256", "oracle.json"),
                        ("installed_report_sha256", "installed-report.json"))
        elif row.get("status") == "BLOCKED" and row.get("applicability") == "unsupported":
            bindings = (("capability_evidence_sha256", "capability.json"),)
        else:
            bindings = ()
        for field, filename in bindings:
            name = "cases/" + sid + "/" + filename
            require(name in files, "BOX_ACCEPTANCE_ORIGINAL_FILE_MISSING_" + sid)
            bind_hash(row, field, sha(files[name]))
    # Canonical JSON equality also preserves bool/int and numeric value types.
    require(encoded_object(derived) == encoded_object(submitted),
            "BOX_ACCEPTANCE_ORIGINAL_OBSERVATIONS_CHANGED")
    current = object_bytes(files["release-gate.json"], "BOX_ACCEPTANCE_REPORT_INVALID")
    current.pop("workflow", None)
    require(encoded_object(current) == encoded_object(submitted),
            "BOX_ACCEPTANCE_OBSERVATIONS_CHANGED_DURING_RECORDING")
    return manifest


def stamp_context(assets, evidence, source, context):
    files = directory_files(evidence)
    manifest = verify_submission(files)
    require("recording.json" not in files, "BOX_ACCEPTANCE_ALREADY_RECORDED")
    require(isinstance(context, dict) and manifest.get("candidate_run_id") == context.get("candidate_run_id"),
            "BOX_ACCEPTANCE_SUBMISSION_RUN_MISMATCH")
    report = object_bytes(files["release-gate.json"], "BOX_ACCEPTANCE_REPORT_INVALID")
    report["workflow"] = context
    index = candidate(assets, source)
    require(report.get("release_assets_sha256") == index, "BOX_ACCEPTANCE_ALL_ASSETS_MISMATCH")
    validate_attestation(Path(evidence) / "release-gate.json", report, source,
                         index["boxweave.zip"], index["release.json"], expected_context=context)
    root = Path(evidence)
    recording = {"schema_version": 1, "workflow": context,
                 "original_receipt_sha256": sha(files["original-receipt.json"]),
                 "submitted_receipt_sha256": sha(files["submitted-receipt.json"]),
                 "recorded_receipt_sha256": sha(encoded_object(report)),
                 "trust_model": "authorized_repository_workflow_records_external_agent_box_evidence",
                 "tests_executed_by_this_command": False}
    with (root / "recording.json").open("xb") as stream:
        stream.write(encoded_object(recording))
    temp = root / ".recorded-receipt.tmp"
    with temp.open("xb") as stream:
        stream.write(encoded_object(report))
    temp.replace(root / "release-gate.json")
    return check(assets, evidence, source, context)


def check(assets, evidence, source, context):
    files = directory_files(evidence)
    verify_submission(files)
    require("recording.json" in files, "BOX_ACCEPTANCE_WORKFLOW_RECORDING_MISSING")
    recorded = object_bytes(files["recording.json"], "BOX_ACCEPTANCE_RECORDING_INVALID")
    require(recorded.get("workflow") == context and
            recorded.get("original_receipt_sha256") == sha(files["original-receipt.json"]) and
            recorded.get("submitted_receipt_sha256") == sha(files["submitted-receipt.json"]) and
            recorded.get("recorded_receipt_sha256") == sha(files["release-gate.json"]),
            "BOX_ACCEPTANCE_RECORDING_BYTES_MISMATCH")
    result = preflight(assets, Path(evidence) / "release-gate.json", source, context)
    return {"decision": "BOX_ACCEPTANCE_EVIDENCE_VALIDATED", "source_sha": source,
            "candidate_sha256": result["candidate_sha256"],
            "attestation_sha256": result["attestation_sha256"],
            "release_assets_sha256": result["release_assets_sha256"],
            "signature_produced": False, "release_authorized": False, "tests_executed_by_this_command": False,
            "trust_model": "authorized_repository_workflow_records_external_agent_box_evidence"}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    p = commands.add_parser("pack", help="Package existing sanitized real Box evidence")
    p.add_argument("--assets", required=True)
    p.add_argument("--evidence", required=True)
    p.add_argument("--source", required=True)
    p.add_argument("--candidate-run-id", required=True, type=int)
    p.add_argument("--out", required=True)
    p = commands.add_parser("unpack", help="Read only base64 evidence from stdin")
    p.add_argument("--out", required=True)
    for name in ("stamp-context", "check"):
        p = commands.add_parser(name)
        p.add_argument("--assets", required=True)
        p.add_argument("--evidence", required=True)
        p.add_argument("--source", required=True)
        p.add_argument("--context", required=True)
    args = parser.parse_args(argv)
    try:
        if args.command == "pack":
            result = pack(args.assets, args.evidence, args.source, args.candidate_run_id, args.out)
        elif args.command == "unpack":
            result = unpack(sys.stdin.read(MAX_ENCODED + 2), args.out)
        else:
            context = json_object(args.context, "BOX_ACCEPTANCE_CONTEXT_INVALID")
            function = stamp_context if args.command == "stamp-context" else check
            result = function(args.assets, args.evidence, args.source, context)
        print(json.dumps(result, sort_keys=True))
        return 0
    except (AcceptanceFault, InstallError, OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile) as exc:
        print(json.dumps({"decision": "DENIED", "reason": str(exc),
                          "signature_produced": False, "release_authorized": False, "tests_executed_by_this_command": False}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
