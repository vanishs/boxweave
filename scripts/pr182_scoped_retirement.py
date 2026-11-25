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
from pr188_scoped_roundtrip import network, second_install


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
        # The live Root executor requires exact ACL exclusivity. A foreign
        # ACL is a valid preservation witness after owned native effects.
        run("incus", "network", "acl", "create", "pr182-foreign", "--project", "default")
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


def main(project, receipt, candidate=None, digest=None):
    assert os.geteuid() == 0
    # Normal full retirement previews the exact owned fixture without effects.
    with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(receipt),
                     allow_mutations=True, timeout=60) as c:
        uuid = op(c, "box_create", {"name": "scoped-retirement-live", "cpu": 1,
                  "memory_mib": 256, "disk_gib": 2, "image": "local:boxweave-ubuntu-24.04",
                  "internet": True, "start": True}, "pr182-live")["box_uuid"]
        second = op(c, "box_create", {"name": "scoped-retirement-peer", "cpu": 1, "memory_mib": 256, "disk_gib": 2, "image": "local:boxweave-ubuntu-24.04", "internet": True, "start": True}, "pr188-peer-fixture")["box_uuid"]
        network(c, uuid, second, "roundtrip-first")
        before = pid("boxweave.service")
        p, plan = check_plan(project)
        assert p.returncode == 0 and plan["ready"] and plan["mode"] == u.DRAIN_MODE, (p.stderr, plan)
        assert [x["action"] for x in plan["steps"]] == ["box_peer_set", "port_forward_delete", "box_stop", "box_delete", "box_stop", "box_delete"]
        assert {x["box"] for x in plan["steps"]} == {uuid, second}
        assert pid("boxweave.service") == before and not u.AUDIT.exists() and not u.SCOPED_FENCE.exists()
        print("PR182_SCOPED_LIVE_PLAN=PASS", flush=True)
        # Test-only harmless unit-specific override, never inherited policy.
        if os.environ.get("GITHUB_ACTIONS") == "true" and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted":
            from pr188_scoped_roundtrip import http_status
            folder = Path("/etc/systemd/system/boxweave.service.d")
            fixture = folder / "91-boxweave-ci-retirement-review.conf"
            folder.mkdir(mode=0o755)
            try:
                with fixture.open("x") as out:
                    out.write("[Service]\nEnvironment=BOXWEAVE_CI_REVIEW_FIXTURE=1\n")
                run("systemctl", "daemon-reload")
                denied, reviewed = check_plan(project)
                assert denied.returncode != 0 and not reviewed["ready"]
                assert any("EFFECTIVE_DROPIN_UNREVIEWED" in b for b in reviewed["blockers"])
                assert pid("boxweave.service")==before and http_status(18082)==200
                assert not u.SCOPED_DRAIN.exists() and not u.AUDIT.exists()
                print("PR188_UNKNOWN_EFFECTIVE_DROPIN_NO_EFFECT=PASS",flush=True)
            finally:
                fixture.unlink(missing_ok=True)
                folder.rmdir()
                run("systemctl", "daemon-reload")
            assert pid("boxweave.service")==before

    # Add unrelated native resources. Never remove them.
    run("incus", "project", "create", "pr182-foreign")
    run("incus", "network", "acl", "create", "pr182-foreign", "--project", "default")
    foreign_project = run("incus", "query", "/1.0/projects/pr182-foreign")
    foreign_acl = run("incus", "query", "/1.0/network-acls/pr182-foreign?project=default")
    blocked, bad = check_plan(project)
    assert blocked.returncode != 0 and not bad["ready"], bad
    assert "DRAIN_ROOT_ACL_INVENTORY_NOT_EXCLUSIVE" in bad["capture"]["blockers"], bad
    assert run("incus", "query", "/1.0/network-acls/pr182-foreign?project=default") == foreign_acl
    assert pid("boxweave.service") == before and not u.SCOPED_DRAIN.exists() and not u.AUDIT.exists()
    print("PR188_H2A_FOREIGN_ACL_PRE_DISPATCH_REFUSED=PASS", flush=True)
    # Remove only this exact disposable fixture; never delete foreign ACLs
    # from the uninstaller. Recreate it after native drain in interrupted().
    run("incus", "network", "acl", "delete", "pr182-foreign", "--project", "default")
    incus_pid = pid("incus.service")
    p, plan = check_plan(project)
    assert p.returncode == 0 and plan["ready"] and len(plan["capture"]["scope"]["retired"]) >= 3, (p.stderr, plan)
    print("PR182_SCOPED_RETIRED_HISTORY=PASS", flush=True)
    tty_run([sys.executable, __file__, "--interrupt", project],
            [(("Type REMOVE SCOPED " + project).encode(), "REMOVE SCOPED " + project)])
    foreign = [foreign_project, run("incus", "query", "/1.0/network-acls/pr182-foreign?project=default")]
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
    drain = json.loads((archive / "drain-receipt.json").read_text())
    assert drain["phase"] == "DRAINED" and drain["index"] == len(drain["steps"]) == 6
    assert not u.SCOPED_DRAIN.exists()
    print("PR188_H2A_DRAIN_COMPLETE=PASS", flush=True)
    print("PR182_SCOPED_COMPLETE=PASS", flush=True)
    # Explicit data erasure is separate from verified full retirement.
    preview = json.loads(run("sh", "uninstall.sh", "--purge-plan", "--project", project))
    assert preview["ready"] and preview["irreversible"] and str(archive / "state-ledger.sqlite") in preview["files"]
    assert archive.exists() and u.AUDIT.exists()
    subprocess.run(["env", "RUNNER_ENVIRONMENT=github-hosted", sys.executable,
                    str(Path(__file__).with_name("pr188_h2c_purge.py")), project], check=True)
    assert not any(os.path.lexists(p) for p in (u.RETIRED, u.AUDIT, u.PURGE_FENCE))
    assert pid("incus.service") == incus_pid
    assert foreign == [run("incus", "query", "/1.0/projects/pr182-foreign"),
                       run("incus", "query", "/1.0/network-acls/pr182-foreign?project=default")]
    print("RC14_SCOPED_PURGE=PASS", flush=True)
    if candidate is not None:
        assert digest and os.environ.get("GITHUB_ACTIONS")=="true" and os.environ.get("RUNNER_ENVIRONMENT")=="github-hosted"
        # The unchanged witnesses above are exact resources created by
        # this fixture, not unrelated resources of an operator. Remove
        # only them before asking the fresh exclusive plan to run again.
        run("incus", "network", "acl", "delete", "pr182-foreign", "--project", "default")
        run("incus", "project", "delete", "pr182-foreign")
        second_install(project, receipt, candidate, digest, (uuid, second), incus_pid)



if __name__ == "__main__":
    if sys.argv[1] == "--interrupt":
        interrupted(sys.argv[2])
    else:
        main(*sys.argv[1:])
