"""Durable once-only gate for scenario actions (not a privileged host executor).

A dispatched UNKNOWN is never retried. The record is written *before* any
external mutator call; loss of the caller leaves a frozen target requiring
independent reconciliation. This is an execution primitive, not a scheduler.
"""
from __future__ import annotations

import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import uuid

from .engine import AcceptanceFault, canonical, now, private_mkdir, unique_file


def immutable_target(target):
    if (not isinstance(target, dict) or
            not re.fullmatch(r"[a-z0-9][a-z0-9-]{2,62}", str(target.get("owner", ""))) or
            not re.fullmatch(r"[0-9a-fA-F-]{36}", str(target.get("provider_uuid", ""))) or
            not str(target.get("created_at", "")).isdigit() or
            target.get("test_owned") is not True):
        raise AcceptanceFault("FOREIGN_OR_UNPINNED_TEST_TARGET")
    return {"owner": target["owner"], "provider_uuid": target["provider_uuid"],
            "created_at": str(target["created_at"]), "test_owned": True}


class OnceOnly:
    def __init__(self, root):
        self.root = private_mkdir(root)

    def reserve(self, request_key, payload, target):
        if not re.fullmatch(r"[0-9a-f]{32}", str(request_key)):
            raise AcceptanceFault("ONCE_ONLY_REQUEST_KEY_INVALID")
        identity = immutable_target(target)
        body_hash = hashlib.sha256(canonical(payload)).hexdigest()
        file = self.root / ("dispatch-" + request_key + ".json")
        lock = self.root / ".dispatch.lock"
        with lock.open("a+b") as fd:
            fcntl.flock(fd, fcntl.LOCK_EX)
            frozen = self.root / ("frozen-" + identity["provider_uuid"] + ".json")
            if frozen.exists() or frozen.is_symlink():
                raise AcceptanceFault("TARGET_FROZEN_UNKNOWN_NO_RETRY")
            if file.exists():
                old = json.loads(file.read_text())
                if old["body_hash"] != body_hash or old["target"] != identity:
                    raise AcceptanceFault("SAME_KEY_DIFFERENT_BODY_OR_TARGET")
                return {"dispatched": False, "state": "ALREADY_RESERVED_NO_RETRY",
                        "original_file": str(file)}
            record = {"schema_version": 1, "created_at": now(), "request_key": request_key,
                      "target": identity, "body_hash": body_hash,
                      "state": "DISPATCH_RESERVATION_MAY_HAVE_REACHED_PROVIDER"}
            unique_file(file, record)
            # Freeze immediately before the potential external mutation;
            # no optimistic second attempt can race or crash through this.
            unique_file(frozen, {"state": "UNKNOWN_UNTIL_INDEPENDENT_RECONCILIATION",
                                 "request_key": request_key, "target": identity})
            return {"dispatched": False, "state": "RESERVED_ONLY_NEEDS_APPROVED_EXECUTOR",
                    "original_file": str(file)}

    def observe(self, provider_uuid):
        path = self.root / ("frozen-" + str(provider_uuid) + ".json")
        if path.exists():
            return json.loads(path.read_text())
        return {"state": "NOT_RESERVED"}
