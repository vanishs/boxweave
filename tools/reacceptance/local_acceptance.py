"""Read-only developer-side checks for Agent-managed installed Box evidence.

No GitHub workflow, signing key, special account, runner, or release action.
A successful consistency check cannot authenticate a remote human/Agent claim.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
from zipfile import BadZipFile

from bwmini.distribution import InstallError, verify as verify_package
from .engine import AcceptanceFault
from .release_gate import (
    HEX40, json_object, sha, validate_attestation, validate_release_assets,
)
from .workflow_gate import asset_index


def check(assets, evidence, source):
    if not isinstance(source, str) or HEX40.fullmatch(source) is None:
        raise AcceptanceFault("LOCAL_SOURCE_INVALID")
    root = Path(assets)
    index = asset_index(root)
    package = verify_package(root / "boxweave.zip", index["boxweave.zip"])
    manifest = json_object(root / "release.json", "LOCAL_RELEASE_MANIFEST_INVALID")
    if (package.get("source") != {"commit": source, "state": "clean"} or
            manifest.get("source") != source):
        raise AcceptanceFault("LOCAL_CANDIDATE_SOURCE_MISMATCH")
    validate_release_assets(root / "release.json", manifest, package, root / "boxweave.zip")
    report = json_object(evidence, "LOCAL_EVIDENCE_INVALID")
    if report.get("release_assets_sha256") != index:
        raise AcceptanceFault("LOCAL_ALL_RELEASE_ASSETS_MISMATCH")
    validate_attestation(evidence, report, source, index["boxweave.zip"],
                         index["release.json"])
    return {
        "decision": "LOCAL_EVIDENCE_CONSISTENT_NOT_RELEASE_AUTHORIZATION",
        "source_sha": source,
        "candidate_sha256": index["boxweave.zip"],
        "release_assets_sha256": index,
        "evidence_sha256": sha(Path(evidence).read_bytes()),
        "remote_execution_authenticated": False,
        "independent_box_truth_verified_by_this_command": False,
        "release_authorized": False,
        "published": False,
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--assets", required=True)
    parser.add_argument("--evidence", required=True)
    parser.add_argument("--source", required=True)
    args = parser.parse_args(argv)
    try:
        result = check(args.assets, args.evidence, args.source)
        print(json.dumps(result, sort_keys=True))
        return 0
    except (AcceptanceFault, InstallError, OSError, ValueError, TypeError, KeyError,
            BadZipFile) as exc:
        print(json.dumps({"decision": "DENIED", "reason": str(exc),
                          "release_authorized": False, "published": False}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
