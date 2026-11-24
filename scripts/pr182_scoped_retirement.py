#!/usr/bin/env python3
"""Exact installed retirement on a disposable CI VM; never on operator Hosts."""
import json
import os
from pathlib import Path
import subprocess
import sys

import uninstall_core as u
from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, pid, token, tty_run


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def check_plan(project):
    p = subprocess.run(["sh", "uninstall.sh", "--scoped", "--plan", "--project", project],
                       capture_output=True, text=True)
    try:
        plan = json.loads(p.stdout)
    except ValueError as exc:
        raise AssertionError(f"Scoped plan did not return JSON (exit={p.returncode}): {p.stderr.strip()}") from exc
    return p, plan


def interrupted(project):
    # This test-only interruption is after the durable native restore receipt.
    # The real operator approval and all production checks still execute.
    def interrupt(plan):
        assert u.scoped_audit(project)["phase"] == "NATIVE_RESTORED"
        raise RuntimeError("PR182_TEST_INTERRUPTION")
    u.scoped_detach_guard = interrupt
    try:
        u.main(["--scoped", "--full", "--project", project])
    except RuntimeError as exc:
        if str(exc) != "PR182_TEST_INTERRUPTION":
            raise
        assert u.scoped_audit(project)["status"] == "STARTED"
        return
    raise AssertionError("interruption not reached")


def main(project, receipt):
    assert os.geteuid() == 0
    # Full retirement refuses even stopped owned Boxes. Delete only through MCP.
    with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(receipt),
                     allow_mutations=True, timeout=60) as c:
        uuid = op(c, "box_create", {"name": "scoped-retirement-live", "cpu": 1,
                  "memory_mib": 256, "disk_gib": 2, "image": "local:boxweave-ubuntu-24.04",
                  "internet": True, "start": False}, "pr182-live")["box_uuid"]
        before = pid("boxweave.service")
        p, plan = check_plan(project)
        assert p.returncode == 2 and not plan["ready"] and any("SCOPED_LIVE_BOXES" in b for b in plan["blockers"]), (p.stderr, plan)
        assert pid("boxweave.service") == before and not u.AUDIT.exists() and not u.SCOPED_FENCE.exists()
        print("PR182_SCOPED_LIVE_REFUSED=PASS", flush=True)
        op(c, "box_delete", {"box_id": uuid}, "pr182-delete")
    # Add unrelated native resources only after MCP cleanup. Never remove them.
    run("incus", "project", "create", "pr182-foreign")
    run("incus", "network", "acl", "create", "pr182-foreign", "--project", "default")
    foreign = [run("incus", "query", "/1.0/projects/pr182-foreign"),
               run("incus", "query", "/1.0/network-acls/pr182-foreign?project=default")]
    incus_pid = pid("incus.service")
    p, plan = check_plan(project)
    assert p.returncode == 0 and plan["ready"] and plan["scoped_native"]["retired"] >= 4, (p.stderr, plan)
    print("PR182_SCOPED_RETIRED_HISTORY=PASS", flush=True)
    tty_run([sys.executable, __file__, "--interrupt", project],
            [(("Type REMOVE SCOPED " + project).encode(), "REMOVE SCOPED " + project)])
    record = u.scoped_audit(project)
    assert record["phase"] == "NATIVE_RESTORED" and u.SCOPED_FENCE.exists()
    archive = Path(record["archive"])
    assert (archive / "scoped-journal" / "retirement-preimage.json").is_file()
    resume = json.loads(run("sh", "uninstall.sh", "--scoped", "--resume-plan", "--project", project))
    assert resume["phase"] == "NATIVE_RESTORED" and resume["review_sha256"] == record["review_sha256"]
    tty_run(["sh", "uninstall.sh", "--scoped", "--resume", "--project", project],
            [(("Type RESUME SCOPED " + project).encode(), "RESUME SCOPED " + project)])
    print("PR182_SCOPED_INTERRUPTED_RESUMED=PASS", flush=True)
    record = u.scoped_audit(project)
    assert record["status"] == "COMPLETE" and record["result"]["outcome"] == "COMPLETE"
    assert record["result"]["approved_review_sha256"] == resume["review_sha256"]
    u.scoped_postcheck(record["scoped_plan"], archive)
    print("PR182_SCOPED_NATIVE_RESTORED=PASS", flush=True)
    assert pid("incus.service") == incus_pid
    assert foreign == [run("incus", "query", "/1.0/projects/pr182-foreign"),
                       run("incus", "query", "/1.0/network-acls/pr182-foreign?project=default")]
    print("PR182_SCOPED_FOREIGN_UNCHANGED=PASS", flush=True)
    for path in (u.STATE, u.BINARY, u.UNIT, u.SCOPED_JOURNAL, u.SCOPED_GUARD_UNIT,
                 u.SCOPED_LIFECYCLE_UNIT, u.SCOPED_INCUS_DROPIN, u.SCOPED_SOCKET, u.SCOPED_FENCE):
        assert not os.path.lexists(path), path
    assert (archive / "scoped-original").is_dir()
    print("PR182_SCOPED_COMPLETE=PASS", flush=True)


if __name__ == "__main__":
    if sys.argv[1] == "--interrupt":
        interrupted(sys.argv[2])
    else:
        main(*sys.argv[1:])
