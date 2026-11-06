"""One-attempt private install and honest installed inspection in an approved Box.

Developer-only: this worker has no signing identity and cannot approve a release.
The private root-owned grant must be provisioned outside candidate workflows.
Incomplete scenario drivers remain BLOCKED; exit zero never substitutes for them.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import sys

from .engine import AcceptanceFault, private_mkdir, unique_file
from .private_installer import check_owned_lab, run as install, CONFIRM
from .installed_verifier import inspect
from .workflow_gate import asset_index, ASSETS

WORKER_OUTPUT = Path("/var/lib/boxweave-reacceptance/original-worker")


def run(assets, source, grant, output, *, workflow_run_id=None, workflow_run_attempt=None):
    if os.geteuid() != 0:
        raise AcceptanceFault("VERIFIER_APPROVED_TEST_BOX_ROOT_REQUIRED")
    if (type(workflow_run_id) is not int or not 0 < workflow_run_id < 2**63 or
            type(workflow_run_attempt) is not int or not 0 < workflow_run_attempt < 2**63):
        raise AcceptanceFault("VERIFIER_WORKFLOW_ACTOR_IDENTITY_REQUIRED")
    identity = check_owned_lab(grant)
    source_dir = Path(assets)
    index = asset_index(source_dir)
    manifest = json.loads((source_dir / "release.json").read_text())
    if manifest.get("source") != source:
        raise AcceptanceFault("VERIFIER_SOURCE_MISMATCH")
    out = Path(output)
    if out != WORKER_OUTPUT or out.exists() or out.is_symlink():
        raise AcceptanceFault("VERIFIER_ORIGINAL_RUN_EXISTS_NO_REPLAY")
    # The controlled root-owned parent is part of the install trust boundary.
    for parent in (out.parent, *out.parent.parents):
        st = parent.lstat()
        if parent.is_symlink() or st.st_uid != 0 or st.st_mode & 0o022:
            raise AcceptanceFault("VERIFIER_OUTPUT_PARENT_UNTRUSTED")
    private_mkdir(out)
    unique_file(out / "worker-intent.json", {
        "state": "ORIGINAL_WORKER_NO_REPLAY", "source_sha": source,
        "candidate_sha256": index["boxweave.zip"],
        "outer_uuid": identity["outer_uuid"],
        "outer_created_at": identity["outer_created_at"]})
    stage = out / "candidate"
    private_mkdir(stage)
    for name in sorted(ASSETS):
        # asset_index rejects symlinks and unexpected files before copying;
        # re-read digest here refuses source substitution during staging.
        data = (source_dir / name).read_bytes()
        if hashlib.sha256(data).hexdigest() != index[name]:
            raise AcceptanceFault("VERIFIER_STAGING_SOURCE_CHANGED")
        with (stage / name).open("xb") as stream:
            stream.write(data)
        (stage / name).chmod(0o600)
    asset_index(stage)
    install(stage, manifest["version"], index["release.json"], source,
            out / "install", apply=True, grant=grant, confirm=CONFIRM)
    report = inspect(stage / "boxweave.zip", stage / "release.json", source,
                     out / "installed")
    # Full stateful drivers are deliberately not inferred from doctor/CLI reads.
    report["worker_actor_id"] = f"github-verifier-{workflow_run_id}-{workflow_run_attempt}"
    report["outer_uuid"] = identity["outer_uuid"]
    report["release_gate_approved"] = False
    unique_file(out / "worker-report.json", report)
    return report


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--assets", required=True)
    p.add_argument("--source", required=True)
    p.add_argument("--grant", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--public-report", required=True)
    p.add_argument("--workflow-run-id", type=int, required=True)
    p.add_argument("--workflow-run-attempt", type=int, required=True)
    args = p.parse_args(argv)
    try:
        report = run(args.assets, args.source, args.grant, args.out,
                     workflow_run_id=args.workflow_run_id,
                     workflow_run_attempt=args.workflow_run_attempt)
        # Export only the deliberately non-secret installed summary. Original
        # root evidence and UNKNOWN scenes remain in the approved test Box.
        with Path(args.public_report).open("x") as stream:
            json.dump(report, stream, sort_keys=True, indent=2)
        print(json.dumps({"state": "PARTIAL", "release_gate_approved": False,
                          "reason": "FULL_INDEPENDENT_SCENARIO_DRIVERS_INCOMPLETE"}))
        return 2
    except (AcceptanceFault, OSError, ValueError, KeyError) as exc:
        print(json.dumps({"state": "BLOCKED_OR_UNKNOWN", "reason": str(exc),
                          "release_gate_approved": False}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
