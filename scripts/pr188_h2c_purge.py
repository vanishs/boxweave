#!/usr/bin/env python3
"""Real completed retirement only; interrupt the production purge after fsync.

Executed only in the existing clean hosted installed-lifecycle job.
No manufactured COMPLETE receipt, original ledger edit or native cleanup.
"""
import os
from pathlib import Path
import subprocess
import sys

from n2_installed_lifecycle import pid, tty_run
import uninstall_core as u


def interrupt(project, boundary):
    real_sync = u.scoped_sync_dir

    def after_sync(path):
        real_sync(path)
        if not u.PURGE_FENCE.exists():
            return
        receipt = u.read_json(u.PURGE_FENCE)
        if receipt["schema"] != 2:
            raise AssertionError("new exact prefix receipt required")
        if u.purge_prefix(receipt["plan"]) == boundary:
            raise InterruptedError("PR188_H2C_TEST_INTERRUPTION")

    u.scoped_sync_dir = after_sync
    try:
        u.main(["--purge", "--project", project])
    except InterruptedError as exc:
        if str(exc) != "PR188_H2C_TEST_INTERRUPTION":
            raise
        assert u.PURGE_FENCE.exists()
        return
    raise AssertionError("purge interruption boundary not reached")


def main(project):
    assert os.geteuid() == 0 and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    before = pid("incus.service")
    record = u.read_json(u.AUDIT)
    assert record["status"] == "COMPLETE"
    receipt = u.purge_review(project)
    assert receipt["schema"] == 2
    plan = receipt["plan"]
    order = u.purge_order(plan)
    assert u.purge_prefix(plan) == 0
    approval = "PURGE " + project
    first_dir = len(plan["files"])
    boundaries = sorted({1, first_dir, len(order)})
    assert len(boundaries) == 3
    for boundary in boundaries:
        tty_run([sys.executable, __file__, "--interrupt", project, str(boundary)],
                [(("Type "+approval).encode(), approval)])
        assert u.purge_review(project) == receipt
        assert u.purge_review(project) == receipt
        assert u.purge_prefix(plan) == boundary
        assert pid("incus.service") == before
        print("PR188_H2C_PURGE_PREFIX=" + str(boundary), flush=True)
    tty_run(["sh", "uninstall.sh", "--purge", "--project", project],
            [(("Type "+approval).encode(), approval)])
    assert not any(os.path.lexists(p) for p in (u.AUDIT,u.RETIRED,u.PURGE_FENCE))
    assert not any(os.path.lexists(p) for p in order)
    assert pid("incus.service") == before
    assert u.purge_native_empty(project) == plan["native_postflight_sha256"]
    print("PR188_H2C_PURGE_INTERRUPTED_RESUMED=PASS", flush=True)


if __name__ == "__main__":
    if sys.argv[1] == "--interrupt":
        assert os.geteuid() == 0 and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
        interrupt(sys.argv[2], int(sys.argv[3]))
    else:
        main(sys.argv[1])
