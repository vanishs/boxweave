#!/usr/bin/env python3
"""One-use fixed-ledger Scoped private-read acceptance.

The seed and cleanup use normal installed MCP, with the original SDK.
The held-response transport is read-only and supervised only while the
normal installed controller is stopped. No Root helper or guard change.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import subprocess
import sys
import tempfile
import time

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import pid, token
from pr178_n2_scoped_lifecycle import acls, binding, instance
from pr191_scoped_ack_loss import (CTL, QUARANTINE, guard_identity,
                                   native_idle, native_names, snapshot,
                                   wait_poststate)
from pr191_scoped_lifecycle_recovery import rows
from pr191_scoped_update_transport import (BIN, DB, SOCKET, command,
                                          http_status, restricted,
                                          supervised, wait_port)

NAME = "bw-s2-read-revoke-110r"


def main(project, receipt, candidate_sha):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    account = pwd.getpwnam("boxweave")
    assert project == "user-" + str(account.pw_uid)
    receipt = Path(receipt)
    work = receipt.parent
    assert work.parent == Path("/tmp") and work.name.startswith("bw-n2-installed.")
    assert receipt.name == "actor.receipt" and receipt.stat().st_mode & 0o077 == 0
    assert command("git", "rev-parse", "HEAD") == os.environ["N2_EXACT_HEAD"]
    assert hashlib.sha256(BIN.read_bytes()).hexdigest() == candidate_sha
    assert hashlib.sha256((work / "candidate-a").read_bytes()).hexdigest() == candidate_sha
    assert not native_names() and acls() == {QUARANTINE}
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')")
    list_rows = rows("SELECT id,work_ref,tracking_id,state,native_operation_id FROM operations WHERE work_ref LIKE 'lab/p1-debugger-list-%' ORDER BY work_ref")
    assert len(list_rows) == 2
    assert [x[1] for x in list_rows] == ["lab/p1-debugger-list-expire", "lab/p1-debugger-list-revoke"]
    assert all(x[3:] == ("CANCELLED", "") for x in list_rows)
    list_before = [(x[0], snapshot(x[0])) for x in list_rows]
    units = ("incus.service", "boxweave-scoped-lifecycle.service")
    pids = [pid(u) for u in units]
    guard = guard_identity()
    unit_before = Path("/etc/systemd/system/boxweave.service").read_bytes()
    assert http_status(18650) == 401 and http_status(18662) is None
    lab = Path(tempfile.mkdtemp(prefix="bw-v1-s2-ghci.", dir="/tmp"))
    lab.chmod(0o711)
    control = lab / "p1-private-read"
    control.mkdir(mode=0o700)
    os.chown(control, account.pw_uid, account.pw_gid)
    proxy = control / "proxy.sock"
    credential = token(receipt)
    env = ("BW_S2_INSTALLED_SCOPED_AUTH=1", "BW_S2_TEST_UID="+str(account.pw_uid),
           "BW_S2_PRIVATE_READ_DIR="+str(control), "BW_S2_PRIVATE_READ_PROJECT="+project)

    def probe(action):
        subprocess.run(restricted(*env, "BW_S2_PRIVATE_READ_SEED_TOKEN="+credential,
                                  str(work / "s2-private-read-probe"), action),
                       check=True, timeout=275)

    probe("seed")
    cp = json.loads((control / "original.json").read_text())
    uuid, oid = cp["uuid"], cp["exec_id"]
    assert re.fullmatch(r"[0-9a-f-]{36}", uuid) and re.fullmatch(r"[0-9a-f]{32}", oid)
    original = rows("SELECT layer,box_uuid,incarnation,state,native_operation_id FROM operations WHERE id=? AND action='box_exec'", (oid,))
    assert len(original) == 1 and original[0][:4] == (project,uuid,uuid,"SUCCEEDED")
    native_id = original[0][4]
    assert re.fullmatch(r"[0-9a-f-]{36}", native_id)
    assert rows("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (oid,)) == [(0,)]
    pin = binding(uuid)
    assert pin["state"] == "PINNED"
    raw = instance(project, NAME)
    assert raw["config"]["volatile.uuid"] == uuid and raw["status"] == "Running"
    assert raw["expanded_devices"]["eth0"]["security.acls"] == pin["acl"]
    # A completed native receipt can precede Incus removal of its operation
    # from the live inventory. Never redispatch the seed; poll only the
    # native observer and fail on persistent operations or inventory drift.
    assert native_names() == [NAME], "unexpected native seed inventory"
    wait_poststate(lambda: native_idle(project))
    assert native_names() == [NAME] and native_idle(project), "seed did not quiesce"
    assert instance(project, NAME)["config"]["volatile.uuid"] == uuid
    assert binding(uuid) == pin
    print("P1B_SCOPED_PRIVATE_READ_SEED_IDLE=PASS", flush=True)
    assert acls() == {QUARANTINE, pin["acl"]}
    before = snapshot(oid)
    all_before = rows("SELECT id,state,native_operation_id,updated_at FROM operations ORDER BY id")

    previous = pid(CTL)
    command("systemctl", "stop", CTL)
    assert command("systemctl", "show", CTL, "-p", "MainPID", "--value") == "0"
    wait_port(18650, False)
    with supervised(restricted(*env, str(work / "s2-private-read-proxy"),
                    "-listen", str(proxy), "-upstream", SOCKET, "-project", project,
                    "-name", NAME, "-dir", str(control)), lab / "proxy.log") as pc:
        for _ in range(60):
            assert pc.poll() is None
            if proxy.is_socket():
                break
            time.sleep(.5)
        assert proxy.is_socket()
        with supervised(restricted(str(BIN), "serve", "-db", DB, "-incus-socket", str(proxy),
                        "-project", project, "-listen", "127.0.0.1:18662",
                        "-scoped-peer-fresh"), lab / "controller.log") as ctl:
            wait_port(18662, True, ctl)
            with LoopbackMCP("http://127.0.0.1:18662/mcp", token=credential, timeout=60) as c:
                assert c.host_status()["peer_mode"] == "SCOPED_V2"
            probe("read")
            probe("stall")
            assert snapshot(oid) == before
            assert rows("SELECT id,state,native_operation_id,updated_at FROM operations ORDER BY id") == all_before
            assert binding(uuid) == pin and native_names() == [NAME] and native_idle(project)
            assert instance(project, NAME)["config"]["volatile.uuid"] == uuid
            assert instance(project, NAME)["expanded_devices"]["eth0"] == raw["expanded_devices"]["eth0"]
            assert "SCOPED_PRIVATE_READ_UNEXPECTED_MUTATION_REFUSED" not in (lab / "proxy.log").read_text()
            print("S2_P2_STALLED_NATIVE_OUTPUT_CANCEL_NO_NATIVE_CHANGE=PASS", flush=True)
        wait_port(18662, False)
    command("systemctl", "start", CTL)
    wait_port(18650, True)
    assert pid(CTL) != previous
    assert Path("/etc/systemd/system/boxweave.service").read_bytes() == unit_before
    assert snapshot(oid) == before and binding(uuid) == pin
    assert rows("SELECT id,state,native_operation_id,updated_at FROM operations ORDER BY id") == all_before
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    print("P1B_SCOPED_PRIVATE_READ_ORIGINAL", oid, original[0][4], uuid, flush=True)
    probe("cleanup")
    assert not native_names() and acls() == {QUARANTINE}
    assert snapshot(oid) == before
    assert binding(uuid)["state"] == "RELEASED"
    assert rows("SELECT state FROM owned_boxes WHERE project=? AND uuid=?", (project,uuid)) == [("DELETED",)]
    assert all(snapshot(x) == saved for x,saved in list_before)
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')")
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    print("P1B_SCOPED_AUTH_FIXED_LEDGER_NATIVE_ABSENCE=PASS", flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
