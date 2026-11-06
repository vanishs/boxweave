"""Non-spawning Agent role handoff with enforced independent verification claims.

Until an actual distinct actor executes a clean candidate verification in an
isolated Box, this only emits MANUAL_ORCHESTRATION_REQUIRED.
"""
from __future__ import annotations

import hashlib
import json
from pathlib import Path

from .engine import AcceptanceFault, now, unique_file


def handoffs(original_case, finder, fixer, candidate_sha256):
    if not isinstance(original_case, dict) or not original_case.get("original_attempt_sha256"):
        raise AcceptanceFault("FINDER_ORIGINAL_REPRO_MISSING")
    if not isinstance(finder, dict) or not finder.get("actor_id") or not finder.get("box_uuid"):
        raise AcceptanceFault("FINDER_ACTOR_IDENTITY_MISSING")
    if not isinstance(fixer, dict) or not fixer.get("actor_id") or not fixer.get("branch"):
        raise AcceptanceFault("FIXER_WORKER_OR_BRANCH_MISSING")
    if finder["actor_id"] == fixer["actor_id"]:
        raise AcceptanceFault("FINDER_AND_FIXER_NOT_DISTINCT")
    if not isinstance(candidate_sha256, str) or len(candidate_sha256) != 64 or             any(c not in "0123456789abcdef" for c in candidate_sha256):
        raise AcceptanceFault("CANDIDATE_BYTES_UNPINNED")
    return {"schema_version": 1, "created_at": now(),
            "workflow": "Coordinator -> Finder -> Fixer -> Verifier",
            "finder": finder, "fixer": fixer, "candidate_sha256": candidate_sha256,
            "original_case": original_case,
            "verifier": {"state": "NOT_ASSIGNED", "must_differ_from_actor": finder["actor_id"],
                         "must_differ_from_box": finder["box_uuid"],
                         "must_install_exact_candidate": candidate_sha256},
            "status": "MANUAL_ORCHESTRATION_REQUIRED_NO_AGENT_SPAWNER",
            "independent_acceptance": False}
