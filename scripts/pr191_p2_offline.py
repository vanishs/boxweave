#!/usr/bin/env python3
"""Exact-candidate offline lifecycle on a fresh disposable hosted N2 VM.

Never run on a developer or operator Host. The existing N2 bootstrap
owns clean-runner checks. No production hook or synthetic Native receipt.
"""
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import sqlite3
import subprocess
import sys
import time

import uninstall_core as u
from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, pid, token, tty_run
from pr188_scoped_roundtrip import second_install
from pr191_scoped_ack_loss import native_names, QUARANTINE
from pr178_n2_scoped_lifecycle import acls, instance


def run(*args):
    return subprocess.check_output(args, text=True, timeout=140).strip()


def plan(project, action="--offline-plan"):
    p = subprocess.run(["sh", "uninstall.sh", action, "--project", project],
                       capture_output=True, text=True, timeout=140)
    try:
        result = json.loads(p.stdout)
    except ValueError:
        result = None
    return p, result


def unchanged(before):
    assert u.cold_sqlite_files(u.STATE / "ledger.sqlite") == before
    assert not any(os.path.lexists(p) for p in (u.AUDIT, u.SCOPED_DRAIN, u.SCOPED_FENCE, u.RETIRED))


def exact_fixture(project, receipt, digest, empty=True):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    assert project == "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    assert run("git", "rev-parse", "HEAD") == os.environ["N2_EXACT_HEAD"]
    work = Path(receipt).parent
    assert work.parent == Path("/tmp") and work.name.startswith("bw-n2-installed.")
    assert Path(receipt).name == "actor.receipt" and not Path(receipt).stat().st_mode & 0o077
    for path in (u.BINARY, work / "candidate-a"):
        assert hashlib.sha256(path.read_bytes()).hexdigest() == digest
    if empty:
        assert not native_names() and acls() == {QUARANTINE}
    assert not any(os.path.lexists(p) for p in (u.AUDIT, u.SCOPED_DRAIN, u.SCOPED_FENCE, u.RETIRED))
    return work


def seed(project, receipt):
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
        box = op(c, "box_create", {"name": "d4-a", "cpu": 1, "memory_mib": 256,
                 "disk_gib": 2, "image": "local:boxweave-ubuntu-24.04",
                 "start": True, "internet": True}, "p2-r3-offline-create")["box_uuid"]
    assert native_names() == ["d4-a"] and instance(project, "d4-a")["config"]["volatile.uuid"] == box
    return box


def stage_original(project, testbin, box):
    staged = run("runuser", "-u", "boxweave", "--", "env",
                 "GITHUB_ACTIONS="+os.environ["GITHUB_ACTIONS"], "RUNNER_ENVIRONMENT="+os.environ["RUNNER_ENVIRONMENT"],
                 "BOXWEAVE_PR177_D4_LAB=disposable-installed-v1", "D4_TEST_PROJECT="+project,
                 "D4_TEST_MODE=SCOPED_POWER", "D4_TEST_PHASE=stage", str(testbin),
                 "-test.run", "^TestPR177D4InstalledCompletedOriginal$", "-test.v", "-test.timeout", "100s")
    original = re.search(r"PR177_D4_COMPLETED_ORIGINAL=([0-9a-f]{32})", staged).group(1)
    assert instance(project, "d4-a")["status"] == "Stopped"
    run("systemctl", "start", u.UNIT.name)
    for _ in range(30):
        result = json.loads(run(str(u.BINARY), "doctor", "-maintenance", "-project", project))
        matches = [x for x in (result.get("ledger") or {}).get("nonterminal_or_unknown_originals", [])
                   if x["id"] == original]
        if matches and matches[0]["state"] == "UNKNOWN":
            assert matches[0]["box_uuid"] == box and matches[0]["native_receipt"] is True
            assert matches[0]["claim_protection"] == "RETAINED" and matches[0]["recorded_claims"] == 1
            break
        time.sleep(1)
    else:
        raise AssertionError("original UNKNOWN and Native receipt not independently observed")
    run("systemctl", "stop", u.UNIT.name)
    return original


def cold_original(db, original):
    with u.cold_sqlite_copy(db) as copied:
        with closing(sqlite3.connect(copied.as_uri()+"?mode=ro", uri=True)) as conn:
            row = conn.execute("SELECT id,tracking_id,action,state,native_operation_id,box_uuid,incarnation "
                               "FROM operations WHERE id=?", (original,)).fetchone()
            claims = conn.execute("SELECT layer,resource_key FROM operation_resource_claims "
                                  "WHERE operation_id=? ORDER BY layer,resource_key", (original,)).fetchall()
            all_ops = conn.execute("SELECT id,action,state,native_operation_id FROM operations ORDER BY id").fetchall()
            return row, claims, all_ops


def verify_complete(project, incus_pid):
    record = u.scoped_audit(project)
    assert record["status"] == "COMPLETE" and record["scoped_plan"]["offline_entry"] is True
    archive, _ = u.scoped_archive_proof(record)
    u.scoped_postcheck(record["scoped_plan"], archive)
    assert not any(os.path.lexists(p) for p in (u.STATE, u.BINARY, u.UNIT, u.SCOPED_JOURNAL,
                                              u.SCOPED_FENCE, u.SCOPED_DRAIN, u.SCOPED_SOCKET))
    assert not native_names() and pid("incus.service") == incus_pid
    return record, archive


def interrupt_prefix(project, receipt, digest):
    exact_fixture(project, receipt, digest, empty=False)
    assert native_names() == ["d4-a"]
    original = u.scoped_offline_progress

    def step(plan, effect=False):
        progress = original(plan, effect)
        if effect and progress["completed"] == 1:
            raise InterruptedError("P2_R3_TEST_VERIFIED_PREFIX")
        return progress

    u.scoped_offline_progress = step
    try:
        u.main(["--offline-full", "--project", project])
    except InterruptedError as exc:
        assert str(exc) == "P2_R3_TEST_VERIFIED_PREFIX"
        record = u.scoped_audit(project)
        assert record["phase"] == "ARCHIVED" and record["status"] == "STARTED"
        return
    raise AssertionError("verified-prefix interruption not reached")


def interrupt_cleanup(project):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    assert run("git", "rev-parse", "HEAD") == os.environ["N2_EXACT_HEAD"]
    record = u.scoped_audit(project)
    assert record["phase"] == "ARCHIVED" and record["scoped_plan"].get("offline_archive")
    sync = u.scoped_sync_dir

    def after_sync(path):
        sync(path)
        if not u.UNIT.exists() and u.BINARY.exists():
            current = u.scoped_audit(project)
            if current["phase"] == "CLEANUP_STARTED":
                raise InterruptedError("P2_R3_TEST_POSTNATIVE_CLEANUP")

    u.scoped_sync_dir = after_sync
    try:
        u.main(["--offline-resume", "--project", project])
    except InterruptedError as exc:
        assert str(exc) == "P2_R3_TEST_POSTNATIVE_CLEANUP"
        return
    raise AssertionError("post-native cleanup interruption not reached")


def exceptional(project, receipt, digest, work, box):
    original = stage_original(project, work / "p2-offline-original.test", box)
    db = u.STATE / "ledger.sqlite"
    before = u.cold_sqlite_files(db)
    witness = cold_original(db, original)
    row, claims, _ = witness
    assert row[2:4] == ("box_stop", "UNKNOWN") and u.VALID_UUID.fullmatch(row[4])
    assert row[5:] == (box, box) and claims == [(project, "box:d4-a")]
    incus_pid = pid("incus.service")
    p, reviewed = plan(project)
    assert p.returncode == 0 and reviewed and reviewed["ready"], (p.stderr, reviewed)
    assert [x["id"] for x in reviewed["offline_archive"]["originals"]] == [original]
    assert [t["uuid"] for t in reviewed["offline_archive"]["targets"]] == [box]
    unchanged(before)
    print("P2_R3_SCOPED_OFFLINE_EXACT_UNKNOWN_REVIEW=PASS", flush=True)
    # Exact disposable sentinel, not an operator resource.
    run("incus", "network", "acl", "create", "p2-r3-foreign", "--project", "default")
    foreign = run("incus", "query", "/1.0/network-acls/p2-r3-foreign?project=default")
    denied, bad = plan(project)
    assert denied.returncode != 0 and (bad is None or not bad["ready"])
    assert run("incus", "query", "/1.0/network-acls/p2-r3-foreign?project=default") == foreign
    unchanged(before)
    assert instance(project, "d4-a")["config"]["volatile.uuid"] == box
    run("incus", "network", "acl", "delete", "p2-r3-foreign", "--project", "default")
    print("P2_R3_SCOPED_OFFLINE_FOREIGN_REFUSED_NO_EFFECT=PASS", flush=True)
    run("incus", "project", "create", "p2-r3-foreign")
    foreign_project = run("incus", "project", "show", "p2-r3-foreign")
    approval = "REMOVE SCOPED OFFLINE " + project
    tty_run([sys.executable, __file__, "--interrupt-prefix", project, receipt, digest],
            [(("Type " + approval).encode(), approval)])
    record = u.scoped_audit(project)
    archive = Path(record["archive"])
    assert record["phase"] == "ARCHIVED" and u.SCOPED_FENCE.exists()
    assert not native_names()
    assert cold_original(db, original) == witness
    assert u.cold_sqlite_files(db) == before
    p, resume = plan(project, "--offline-resume-plan")
    assert p.returncode == 0 and resume["native_observation"]["completed"] == 1, (p.stderr, resume)
    assert resume["review_sha256"] == record["review_sha256"]
    assert resume == plan(project, "--offline-resume-plan")[1]
    prompt = "RESUME SCOPED OFFLINE " + project
    tty_run([sys.executable, __file__, "--interrupt-cleanup", project],
            [(("Type " + prompt).encode(), prompt)])
    p, cleanup = plan(project, "--offline-resume-plan")
    assert p.returncode == 0 and cleanup["phase"] == "CLEANUP_STARTED"
    assert cleanup["review_sha256"] == record["review_sha256"] and cleanup["archive"] == str(archive)
    assert run("incus", "project", "show", "p2-r3-foreign") == foreign_project
    tty_run(["sh", "uninstall.sh", "--offline-resume", "--project", project],
            [(("Type " + prompt).encode(), prompt)])
    done, final_archive = verify_complete(project, incus_pid)
    assert archive == final_archive
    assert done["result"]["business_operations_replayed"] is False
    assert done["result"]["operation_outcome_changed"] is False
    archived_db = archive / "state-ledger.sqlite"
    assert cold_original(archived_db, original) == witness
    for suffix, proof in before.items():
        target = Path(str(archived_db) + suffix)
        assert (hashlib.sha256(target.read_bytes()).hexdigest() == proof["sha256"]) if proof else not target.exists()
    print("P2_R3_SCOPED_OFFLINE_PREFIX_RESUMED=PASS", flush=True)
    print("P2_R3_SCOPED_OFFLINE_ARCHIVED_UNKNOWN_UNCHANGED=PASS", flush=True)
    print("P2_R3_SCOPED_OFFLINE_NATIVE_RETIRED=PASS", flush=True)
    print("P2_R3_SCOPED_OFFLINE_ORIGINAL", original, row[4], row[1], box, flush=True)
    subprocess.run([sys.executable, "scripts/pr188_h2c_purge.py", project], check=True, timeout=360)
    print("P2_R3_SCOPED_OFFLINE_PURGE=PASS", flush=True)
    assert run("incus", "project", "show", "p2-r3-foreign") == foreign_project
    print("P2_R3_SCOPED_OFFLINE_FOREIGN_UNCHANGED=PASS", flush=True)
    print("P2_R3_SCOPED_OFFLINE_POSTNATIVE_RESUMED=PASS", flush=True)
    # Only the exact empty sentinel created by this disposable job.
    run("incus", "project", "delete", "p2-r3-foreign")
    second_install(project, receipt, str(work / "candidate-a"), digest, (box,), incus_pid)


def decline(project, receipt, digest):
    exact_fixture(project, receipt, digest)
    try:
        u.main(["--offline-full", "--project", project])
    except u.UninstallError as exc:
        assert "OPERATOR_APPROVAL_REQUIRED" in str(exc)
        return
    raise AssertionError("declined offline retirement unexpectedly succeeded")


def missing_ack(project, receipt, digest):
    exact_fixture(project, receipt, digest, empty=False)
    assert native_names() == ["bw-s2-p1-stop-110p"]
    run("systemctl", "stop", u.UNIT.name)
    db = u.STATE / "ledger.sqlite"
    before = u.cold_sqlite_files(db)
    with u.cold_sqlite_copy(db) as copied:
        with closing(sqlite3.connect(copied.as_uri()+"?mode=ro", uri=True)) as conn:
            originals = conn.execute("SELECT id,box_uuid,native_operation_id FROM operations "
                                     "WHERE state='UNKNOWN' AND action='box_stop'").fetchall()
    assert len(originals) == 1 and originals[0][2] == ""
    box = originals[0][1]
    assert instance(project, "bw-s2-p1-stop-110p")["status"] == "Stopped"
    p, bad = plan(project)
    assert p.returncode != 0 and bad and not bad["ready"], (p.stderr, bad)
    assert any("SCOPED_OFFLINE_ARCHIVE_PROOF_UNVERIFIED" in b for b in bad["blockers"])
    unchanged(before)
    assert instance(project, "bw-s2-p1-stop-110p")["config"]["volatile.uuid"] == box
    print("P2_R3_SCOPED_OFFLINE_MISSING_ACK_REFUSED=PASS", flush=True)


def main(project, receipt, digest, mode):
    work = exact_fixture(project, receipt, digest)
    assert mode in ("empty", "settled", "original")
    denied, _ = plan(project)
    assert denied.returncode != 0 and not u.AUDIT.exists() and not u.SCOPED_FENCE.exists()
    print("P2_R3_SCOPED_OFFLINE_ACTIVE_REFUSED=PASS", flush=True)
    box = seed(project, receipt) if mode != "empty" else None
    run("systemctl", "stop", u.UNIT.name)
    if mode == "original":
        exceptional(project, receipt, digest, work, box)
        return
    before = u.cold_sqlite_files(u.STATE / "ledger.sqlite")
    incus_pid = pid("incus.service")
    p, reviewed = plan(project)
    assert p.returncode == 0 and reviewed["ready"] and reviewed["offline_entry"] is True, (p.stderr, reviewed)
    assert reviewed["mode"] == (u.SCOPED_MODE if mode == "empty" else u.DRAIN_MODE)
    unchanged(before)
    approval = "REMOVE SCOPED OFFLINE " + project
    if mode == "empty":
        tty_run([sys.executable, __file__, "--decline", project, receipt, digest],
                [(("Type " + approval).encode(), "DECLINE")])
        unchanged(before)
        print("P2_R3_SCOPED_OFFLINE_DECLINED_NO_EFFECT=PASS", flush=True)
    else:
        assert [x["action"] for x in reviewed["steps"]] == ["box_stop", "box_delete"]
        assert {x["box"] for x in reviewed["steps"]} == {box}
    tty_run(["sh", "uninstall.sh", "--offline-full", "--project", project],
            [(("Type " + approval).encode(), approval)])
    record, archive = verify_complete(project, incus_pid)
    if mode == "settled":
        drain = u.read_json(archive / "drain-receipt.json")
        assert drain["phase"] == "DRAINED" and drain["index"] == len(drain["steps"]) == 2
        assert len({x["id"] for x in drain["completed"]}) == 2
    print("P2_R3_SCOPED_OFFLINE_" + mode.upper() + "_COMPLETE=PASS", flush=True)


if __name__ == "__main__":
    if sys.argv[1] == "--interrupt-prefix":
        interrupt_prefix(*sys.argv[2:])
    elif sys.argv[1] == "--interrupt-cleanup":
        interrupt_cleanup(*sys.argv[2:])
    elif sys.argv[1] == "--decline":
        decline(*sys.argv[2:])
    elif sys.argv[1] == "--missing-ack":
        missing_ack(*sys.argv[2:])
    else:
        main(*sys.argv[1:])
