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
from pr177_d4_installed_recovery import db_state, run
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
    for _ in range(30):
        if db_state(original)[0][0] == "UNKNOWN":
            break
        time.sleep(1)
    assert db_state(original)[0][0] == "UNKNOWN"
    run(["systemctl", "stop", u.UNIT.name])
    run(["systemctl", "stop", u.RETIREMENT_UNIT.name])
    db = u.STATE / "ledger.sqlite"
    assert not any(Path(str(db)+s).exists() for s in ("-wal","-shm","-journal"))
    before = digest(db)
    run(["incus", "project", "create", "h2b-foreign"])
    foreign = subprocess.check_output(["incus", "project", "show", "h2b-foreign"], text=True)
    incus_pid = pid("incus.service")
    plan = u.offline_plan(project)
    assert plan["ready"], plan
    assert len(plan["unresolved_originals"]) == 1 and plan["unresolved_originals"][0]["id"] == original, plan
    assert plan["unresolved_originals"][0]["box_uuid"] == box
    assert plan["unresolved_originals"][0]["state"] == "UNKNOWN"
    assert digest(db) == before and not u.OFFLINE_FENCE.exists() and not u.AUDIT.exists()
    print("PR188_H2B_EXACT_UNKNOWN_REVIEW=PASS", flush=True)
    approval = "RETIRE OFFLINE " + project
    tty_run(["sh", "uninstall.sh", "--offline-full", "--project", project],
            [(("Type "+approval).encode(), approval)])
    audit = json.loads(u.AUDIT.read_text())
    assert audit["status"] == "COMPLETE", audit
    archive = Path(audit["result"]["retirement_archive"]) / "state-ledger.sqlite"
    assert digest(archive) == before
    with sqlite3.connect(archive.as_uri()+"?mode=ro&immutable=1", uri=True) as conn:
        assert conn.execute("select state from operations where id=?", (original,)).fetchone() == ("UNKNOWN",)
        assert conn.execute("select count(*) from operation_resource_claims where operation_id=?", (original,)).fetchone()[0] == 1
    print("PR188_H2B_ARCHIVED_UNKNOWN_UNCHANGED=PASS", flush=True)
    assert not u.native_instances(project, guided=True) and not u.OFFLINE_FENCE.exists()
    assert not u.STATE.exists() and not u.BINARY.exists() and not u.NATIVE.exists()
    assert pid("incus.service") == incus_pid
    print("PR188_H2B_NATIVE_RETIRED=PASS", flush=True)
    assert subprocess.check_output(["incus","project","show","h2b-foreign"], text=True) == foreign
    print("PR188_H2B_FOREIGN_UNCHANGED=PASS", flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
