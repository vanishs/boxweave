#!/usr/bin/env python3
"""Exact-head disposable installed Scoped final-commit fault acceptance.

The test-owned SQLite trigger refuses only a named fixture terminal update.
No production switch, alternate ledger, native replay, or fake receipt.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import sqlite3
import subprocess
import sys

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import pid, token
from pr178_n2_scoped_lifecycle import acls, binding, incus, instance, verify

DB = "file:/var/lib/boxweave/ledger.sqlite"
ROOT = Path("/var/lib/boxweave-scoped-peer-v2")
NAME = "p1b-recovery-final-cut"
TRIGGER = "p1b_only_ci_final_commit_fault"


def rows(sql, params=()):
    with sqlite3.connect(DB + "?mode=ro", uri=True) as db:
        return db.execute(sql, params).fetchall()


def journal(oid, kind, native_id):
    assert len(oid) == 32 and all(c in "0123456789abcdef" for c in oid)
    names = ["intent.json", "verified.json"]
    if kind == "CREATE":
        names.insert(1, "acl-verified.json")
    raw = [(ROOT / ("op-" + oid) / n).read_bytes() for n in names]
    data = [json.loads(b) for b in raw]
    last = data[-1]
    for row, status in zip(data, ["DISPATCH_UNKNOWN", "ACL_VERIFIED", "VERIFIED"]
                           if kind == "CREATE" else ["DISPATCH_UNKNOWN", "VERIFIED"]):
        assert row == dict(last, Status=status), row
    assert last["Kind"] == kind and last["OriginalID"] == oid and last["NativeID"] == native_id
    return last, [hashlib.sha256(b).hexdigest() for b in raw]


def cut(c, action, args, key):
    work = "p1b/" + key
    assert action in ("box_create", "box_delete")
    assert not rows("SELECT id FROM operations WHERE work_ref=?", (work,))
    assert not rows("SELECT name FROM sqlite_master WHERE type='trigger' AND name=?", (TRIGGER,))
    # In this exact single-use CI VM only. Root journal and native dispatch
    # run normally; the original ledger's last commit must fail atomically.
    with sqlite3.connect(DB + "?mode=rw", uri=True, timeout=15) as db:
        db.execute(f"CREATE TRIGGER {TRIGGER} BEFORE UPDATE OF state ON operations "
                   f"WHEN OLD.action='{action}' AND OLD.work_ref='{work}' "
                   "AND OLD.state='VERIFYING' AND NEW.state='SUCCEEDED' "
                   "BEGIN SELECT RAISE(ABORT,'P1B_ONE_USE_FINAL_COMMIT_FAULT'); END")
    try:
        rec = c.call(action, dict(args, idempotency_key="pr191-" + key,
                                  work_ref=work)).structured
        oid = rec["operation_id"]
        after = c.poll_operation(oid, seconds=150)
        assert after["state"] == "UNKNOWN" and after.get("native_operation_id"), after
        assert rows("SELECT id,state,native_operation_id FROM operations WHERE work_ref=?", (work,)) == [
            (oid, "UNKNOWN", after["native_operation_id"])]
        assert rows("SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=?", (oid,))[0][0] > 0
        return after
    finally:
        # Remove only this exact test-owned trigger; retain original and
        # all native effects on any failure, never unlock or replay.
        with sqlite3.connect(DB + "?mode=rw", uri=True, timeout=15) as db:
            db.execute(f"DROP TRIGGER {TRIGGER}")


def settle(c, original, kind, uuid, key):
    oid = original["operation_id"]
    native_id = original["native_operation_id"]
    rec, before = journal(oid, kind, native_id)
    assert rec["UUID"] == uuid
    inspect = c.call("operation_recover", {"operation_id": oid, "mode": "inspect"}).structured
    if inspect["diagnosis"]["disposition"] != "SAFE_RECOVER":
        # Diagnostic evidence from the exact disposable Root service only.
        # Never emit arbitrary journal messages, credentials or native output.
        log = subprocess.run(["journalctl", "-u", "boxweave-scoped-lifecycle.service",
                              "--no-pager", "-o", "cat", "-n", "100"],
                             capture_output=True, text=True, timeout=15, check=True).stdout
        prefix = "Root original " + oid + " refused or uncertain: "
        matches = [line.split(prefix, 1)[1] for line in log.splitlines() if prefix in line]
        classes = (
            ("original native Create not proven complete", "ORIGINAL_NATIVE_TERMINAL"),
            ("native Project not quiescent", "PROJECT_NATIVE_BUSY"),
            ("completed original lifecycle journal unavailable", "COMPLETED_JOURNAL"),
            ("scoped Create original differs from completed journal", "ORIGINAL_JOURNAL_MISMATCH"),
            ("scoped native Create identity or stopped/boot preimage changed", "CREATE_SHAPE"),
            ("scoped Create NIC or disk shape changed", "NIC_DISK_SHAPE"),
            ("completed Create ACL unverified", "CREATE_ACL"),
            ("foreign or ambiguous native static lease", "LEASE_CONFLICT"),
            ("missing native static leases", "LEASE_MISSING"),
            ("scoped native inventory differs from ledger", "NATIVE_INVENTORY"),
            ("resource locked", "LEDGER_RESOURCE_LOCKED"),
            ("scoped Delete absence unverified", "DELETE_ABSENCE"),
            ("scoped Delete ACL absence unverified", "DELETE_ACL"),
        )
        detail = next((label for fragment, label in classes
                       if matches and fragment in matches[-1]),
                      "UNCLASSIFIED_ROOT_REFUSAL" if matches else "NO_ROOT_REFUSAL_LOG")
        print("P1B_SCOPED_RECOVERY_ROOT_REFUSAL", oid, detail, flush=True)
    assert inspect["diagnosis"]["disposition"] == "SAFE_RECOVER", inspect
    assert inspect["diagnosis"]["reason"] == "SCOPED_LIFECYCLE_COMPLETED_NO_REPLAY", inspect
    assert inspect["diagnosis"]["claim_protection"] == "RETAINED" and not inspect["native_retried"]
    args = {"operation_id": oid, "mode": "reconcile", "idempotency_key": "pr191-" + key}
    first = c.call("operation_recover", args).structured
    result = first["recovery"]
    rid = result["operation"]["operation_id"]
    assert result["disposition"] == "SETTLED" and result["operation"]["state"] == "SUCCEEDED"
    assert rid != oid and not first["native_retried"], first
    again = c.call("operation_recover", args).structured
    assert again["recovery"]["operation"]["operation_id"] == rid
    assert again["recovery"]["operation"]["duplicate"] and not again["native_retried"]
    final = c.operation_get(oid)
    assert final["state"] == "SUCCEEDED" and final["box_uuid"] == uuid
    assert final["tracking_id"] == original["tracking_id"] and final["native_operation_id"] == native_id
    assert journal(oid, kind, native_id) == (rec, before)
    print("P1B_SCOPED_" + kind + "_FINAL_COMMIT_RECOVERY=PASS", oid, native_id, rid, uuid, flush=True)
    return rec


def main(project, receipt, candidate_sha):
    assert os.geteuid() == 0
    assert os.environ.get("GITHUB_ACTIONS") == "true" and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    assert project == "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    assert hashlib.sha256(Path("/usr/local/bin/boxweave").read_bytes()).hexdigest() == candidate_sha
    assert not incus("list", "--all-projects", "-c", "n", "-f", "csv").strip()
    assert acls() == {"bw-peer-quarantine"}
    assert rows("SELECT COUNT(*) FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    controller, root = pid("boxweave.service"), pid("boxweave-scoped-lifecycle.service")
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        hs = c.host_status()
        assert hs["ready"] and hs["peer_mode"] == "SCOPED_V2" and hs["native"]["project"] == project
        create = cut(c, "box_create", {"name": NAME, "cpu": 1, "memory_mib": 256,
                                       "disk_gib": 2, "image": "local:boxweave-ubuntu-24.04",
                                       "internet": True, "start": True}, "scoped-create-final-cut")
        assert not create.get("box_uuid"), create
        assert not rows("SELECT uuid FROM owned_boxes WHERE name=?", (NAME,))
        before = instance(project, NAME)
        uuid = before["config"]["volatile.uuid"]
        rec, _ = journal(create["operation_id"], "CREATE", create["native_operation_id"])
        assert rec["UUID"] == uuid and before["status"] == "Running"
        assert before["config"]["user.boxweave.create_operation"] == create["operation_id"]
        assert acls() == {"bw-peer-quarantine", rec["Binding"]["ACL"]}
        settle(c, create, "CREATE", uuid, "scoped-create-final-reconcile")
        acl = verify(project, NAME, uuid, "true")
        assert instance(project, NAME)["config"]["volatile.uuid"] == uuid
        assert instance(project, NAME)["status"] == "Running"
        delete = cut(c, "box_delete", {"box_id": uuid}, "scoped-delete-final-cut")
        assert delete["box_uuid"] == uuid and binding(uuid)["state"] == "PINNED"
        assert rows("SELECT state FROM owned_boxes WHERE uuid=?", (uuid,)) == [("OWNED",)]
        assert not incus("list", "--all-projects", "-c", "n", "-f", "csv").strip()
        assert acl not in acls()
        settle(c, delete, "DELETE", uuid, "scoped-delete-final-reconcile")
        assert binding(uuid)["state"] == "RELEASED"
        assert rows("SELECT state,operation_id FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == [
            ("RELEASED", delete["operation_id"]), ("RELEASED", delete["operation_id"])]
        assert rows("SELECT state FROM owned_boxes WHERE uuid=?", (uuid,)) == [("DELETED",)]
        assert acls() == {"bw-peer-quarantine"}
        assert not incus("list", "--all-projects", "-c", "n", "-f", "csv").strip()
        assert pid("boxweave.service") == controller and pid("boxweave-scoped-lifecycle.service") == root
        assert c.host_status()["ready"]
    print("P1B_SCOPED_FINAL_COMMIT_NATIVE_RECOVERY=PASS", flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
