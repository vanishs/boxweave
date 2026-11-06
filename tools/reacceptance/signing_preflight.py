"""Developer-only pre-sign validation; never reads keys or produces signatures.

Protected Verifier jobs may call this after real installed scenario execution.
Eligibility means structurally complete evidence for exact bytes, not trusted
release approval: the producer identity and original observations must still be
independently authenticated, and publication rechecks the resulting signature.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
from zipfile import BadZipFile

from bwmini.distribution import InstallError, verify as verify_package
from .engine import AcceptanceFault
from .release_gate import (HEX40, json_object, sha, validate_attestation,
                           validate_release_assets)
from .workflow_gate import asset_index


def prepare(assets, attestation, source, expected_context):
    if not isinstance(source, str) or HEX40.fullmatch(source) is None:
        raise AcceptanceFault("SIGNING_SOURCE_INVALID")
    if expected_context is None:
        raise AcceptanceFault("SIGNING_WORKFLOW_CONTEXT_REQUIRED")
    index = asset_index(assets)
    root = Path(assets)
    manifest_path = root / "release.json"
    candidate = root / "boxweave.zip"
    manifest = json_object(manifest_path, "SIGNING_RELEASE_MANIFEST_INVALID")
    package = verify_package(candidate, index["boxweave.zip"])
    if (package.get("source") != {"commit": source, "state": "clean"} or
            manifest.get("source") != source):
        raise AcceptanceFault("SIGNING_CANDIDATE_SOURCE_MISMATCH")
    validate_release_assets(manifest_path, manifest, package, candidate)
    evidence = json_object(attestation, "SIGNING_ATTESTATION_INVALID")
    if evidence.get("release_assets_sha256") != index:
        raise AcceptanceFault("SIGNING_ALL_ASSET_BINDING_MISMATCH")
    validate_attestation(attestation, evidence, source, index["boxweave.zip"],
                         index["release.json"], expected_context=expected_context)
    return {"decision": "ELIGIBLE_FOR_PROTECTED_SIGNING_ONLY",
            "source_sha": source, "candidate_sha256": index["boxweave.zip"],
            "attestation_sha256": sha(Path(attestation).read_bytes()),
            "release_assets_sha256": index,
            "signature_produced": False, "release_authorized": False}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--assets", required=True)
    parser.add_argument("--attestation", required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--expected-context", required=True)
    args = parser.parse_args(argv)
    try:
        result = prepare(args.assets, args.attestation, args.source,
                         json_object(args.expected_context, "SIGNING_CONTEXT_INVALID"))
        print(json.dumps(result, sort_keys=True))
        return 0
    except (AcceptanceFault, InstallError, BadZipFile, OSError, ValueError,
            TypeError, KeyError) as exc:
        print(json.dumps({"decision": "DENIED", "reason": str(exc),
                          "signature_produced": False, "release_authorized": False}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
