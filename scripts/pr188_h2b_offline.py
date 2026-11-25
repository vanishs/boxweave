#!/usr/bin/env python3
"""Clean disposable installed Legacy acceptance; retain the original UNKNOWN."""
import hashlib
import json
import os
from pathlib import Path
import re
import sqlite3
import subprocess
import sys
import time

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, token, tty_run, pid
from pr177_d4_installed_recovery import run
import uninstall_core as u


def digest(p):
    return hashlib.sha256(Path(p).read_bytes()).hexdigest()


def main(project, receipt, testbin):
    assert os.geteuid() == 0 and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(receipt), allow_mutations=True, timeout=60) as c:
        box = op(c, "box_create", {"name": "d4-a", "cpu": 1, "memory_mib": 256,
                 "disk_gib": 2, "image": "local:boxweave-ubuntu-24.04",
                 "start": True, "internet": True}, "h2b-create")["box_uuid"]
    run(["systemctl", "stop", u.UNIT.name])
    staged = subprocess.check_output(["runuser", "-u", "boxweave", "--", "env",
        "BOXWEAVE_PR177_D4_LAB=disposable-installed-v1", "D4_TEST_PROJECT="+project,
        "D4_TEST_MODE=LEGACY", "D4_TEST_PHASE=stage", testbin,
        "-test.run", "^TestPR177D4InstalledCompletedOriginal$", "-test.v", "-test.timeout", "100s"],
        text=True, timeout=100)
    original = re.search(r"PR177_D4_COMPLETED_ORIGINAL=([0-9a-f]{32})", staged).group(1)
    run(["systemctl", "start", u.UNIT.name])
    # Read private snapshots rather than leaving root SQLite sidecars.
    for _ in range(30):
        observed = subprocess.run([str(u.BINARY), "doctor", "-maintenance", "-project", project],
                                  capture_output=True, text=True, timeout=60)
        report = json.loads(observed.stdout)
        if any(x["id"] == original and x["state"] == "UNKNOWN"
               for x in (report.get("ledger") or {}).get("nonterminal_or_unknown_originals", [])):
            break
        time.sleep(1)
    else:
        raise AssertionError("original UNKNOWN not independently observed")
    run(["systemctl", "stop", u.UNIT.name])
    run(["systemctl", "stop", u.RETIREMENT_UNIT.name])
    db = u.STATE / "ledger.sqlite"
    before = u.cold_sqlite_files(db)
    assert before[""] is not None and before["-journal"] is None
    run(["incus", "project", "create", "h2b-foreign"])
    foreign = subprocess.check_output(["incus", "project", "show", "h2b-foreign"], text=True)
    incus_pid = pid("incus.service")
    plan = u.offline_plan(project)
    assert plan["ready"], plan
    assert len(plan["unresolved_originals"]) == 1 and plan["unresolved_originals"][0]["id"] == original, plan
    assert plan["unresolved_originals"][0]["box_uuid"] == box
    assert plan["unresolved_originals"][0]["state"] == "UNKNOWN"
    assert u.cold_sqlite_files(db) == before and not u.OFFLINE_FENCE.exists() and not u.AUDIT.exists()
    print("PR188_H2B_EXACT_UNKNOWN_REVIEW=PASS", flush=True)
    approval = "RETIRE OFFLINE " + project
    tty_run([sys.executable, __file__, "--interrupt-cleanup", project],
            [(("Type "+approval).encode(), approval)])
    preview = u.offline_resume_preview(project)
    assert preview["phase"] == "CLEANUP_STARTED" and preview["status"] == "STARTED"
    assert preview == u.offline_resume_preview(project)
    assert not u.UNIT.exists() and u.BINARY.exists() and u.OFFLINE_FENCE.exists()
    resume = "RESUME OFFLINE " + project
    tty_run(["sh", "uninstall.sh", "--offline-resume", "--project", project],
            [(("Type "+resume).encode(), resume)])
    assert u.offline_resume_preview(project)["status"] == "COMPLETE"
    print("PR188_H2C_OFFLINE_POSTNATIVE_RESUMED=PASS", flush=True)
    audit = json.loads(u.AUDIT.read_text())
    assert audit["status"] == "COMPLETE", audit
    archive = Path(audit["result"]["retirement_archive"]) / "state-ledger.sqlite"
    for suffix, proof in before.items():
        path = Path(str(archive)+suffix)
        assert (digest(path) == proof["sha256"]) if proof else not path.exists()
    with u.cold_sqlite_copy(archive) as snapshot:
        conn = sqlite3.connect(snapshot.as_uri()+"?mode=ro", uri=True)
        try:
            assert conn.execute("select state from operations where id=?", (original,)).fetchone() == ("UNKNOWN",)
            assert conn.execute("select count(*) from operation_resource_claims where operation_id=?", (original,)).fetchone()[0] == 1
        finally:
            conn.close()
    print("PR188_H2B_ARCHIVED_UNKNOWN_UNCHANGED=PASS", flush=True)
    assert not u.native_instances(project, guided=True) and not u.OFFLINE_FENCE.exists()
    assert not u.STATE.exists() and not u.BINARY.exists() and not u.NATIVE.exists()
    assert pid("incus.service") == incus_pid
    print("PR188_H2B_NATIVE_RETIRED=PASS", flush=True)
    assert subprocess.check_output(["incus","project","show","h2b-foreign"], text=True) == foreign
    print("PR188_H2B_FOREIGN_UNCHANGED=PASS", flush=True)
    subprocess.run([sys.executable, str(Path(__file__).with_name("pr188_h2c_purge.py")), project], check=True)
    assert subprocess.check_output(["incus","project","show","h2b-foreign"], text=True) == foreign
    print("PR188_H2C_OFFLINE_PURGE=PASS", flush=True)


def interrupt_cleanup(project):
    real_sync = u.scoped_sync_dir
    def sync(path):
        real_sync(path)
        if not u.UNIT.exists() and u.BINARY.exists() and u.AUDIT.exists():
            record = u.read_json(u.AUDIT)
            if record.get("phase") == "CLEANUP_STARTED":
                raise InterruptedError("PR188_H2C_TEST_CLEANUP_INTERRUPT")
    u.scoped_sync_dir = sync
    try:
        u.main(["--offline-full", "--project", project])
    except InterruptedError as exc:
        if str(exc) != "PR188_H2C_TEST_CLEANUP_INTERRUPT":
            raise
        return
    raise AssertionError("post-native interruption not reached")


if __name__ == "__main__":
    if sys.argv[1] == "--interrupt-cleanup":
        assert os.geteuid() == 0 and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
        interrupt_cleanup(sys.argv[2])
    else:
        main(*sys.argv[1:])
