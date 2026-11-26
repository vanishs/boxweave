#!/usr/bin/env python3
"""Reuse the original S2 official-SDK Exec crash witness on installed Scoped.

One-use hosted CI only. Never restart Incus or the Root helper, never
admin-delete or unlock the uncertain original, even on fixture failure.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import subprocess
import sys
import tempfile
import time

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import pid, token
from pr177_d4_installed_recovery import wait_mcp_listener
from pr178_n2_scoped_lifecycle import acls, incus, instance
from pr191_scoped_ack_loss import guard_identity, native_names
from pr191_scoped_lifecycle_recovery import rows

CTL = "boxweave.service"
BIN = Path("/usr/local/bin/boxweave")
DB = "/var/lib/boxweave/ledger.sqlite"
NAME = "bw-s2-crash-110b"


def property_value(unit, prop):
    return subprocess.check_output(["systemctl", "show", unit, "-p", prop, "--value"],
                                   text=True, timeout=10).strip()


def main(project, receipt, candidate_sha, probe):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    account = pwd.getpwnam("boxweave")
    assert project == "user-" + str(account.pw_uid)
    assert hashlib.sha256(BIN.read_bytes()).hexdigest() == candidate_sha
    assert not property_value(CTL, "DropInPaths")
    assert not native_names() and acls() == {"bw-peer-quarantine"}
    assert rows("SELECT COUNT(*) FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    assert Path(probe).is_file()
    guard = guard_identity()
    units = ("incus.service", "boxweave-scoped-lifecycle.service")
    pids = [pid(u) for u in units]
    controller = pid(CTL)
    assert os.path.samefile(f"/proc/{controller}/exe", BIN)
    assert Path(f"/proc/{controller}").stat().st_uid == account.pw_uid
    secret = token(receipt)
    with LoopbackMCP(token=secret, allow_mutations=False, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
    state = Path(tempfile.mkdtemp(prefix="bw-v1-s2-ghci.", dir="/tmp")) / "original.json"
    env = dict(os.environ, BW_S2_CRASH_STATE_FILE=str(state),
               BW_S2_MCP_URL="http://127.0.0.1:18650/mcp", BW_S2_ADMIN_TOKEN=secret)
    env.pop("BW_S2_CRASH_VARIANT", None)
    subprocess.run([probe, "start"], env=env, check=True, timeout=250)
    original = json.loads(state.read_text())
    oid, native_id, uuid = (original[k] for k in ("operation_id", "native_operation_id", "box_uuid"))
    assert native_names() == [NAME]
    assert instance(project, NAME)["config"]["volatile.uuid"] == uuid
    # Second serve must refuse the SAME live ledger before startup recovery,
    # on an alternate loopback listener and the exact Scoped mode.
    other = subprocess.run(["timeout", "--signal=TERM", "8s", "runuser", "-u", "boxweave", "--",
                            str(BIN), "serve", "-db", DB, "-incus-socket",
                            "/var/lib/incus/unix.socket.user", "-project", project,
                            "-listen", "127.0.0.1:18655", "-scoped-peer-fresh"],
                           capture_output=True, text=True, timeout=12)
    assert other.returncode != 0 and "serve already active for this database" in other.stderr + other.stdout
    original_row = rows("SELECT state,native_operation_id,tracking_id,box_uuid FROM operations WHERE id=?", (oid,))
    assert len(original_row) == 1 and original_row[0][:2] == ("VERIFYING", native_id)
    assert original_row[0][3] == uuid
    tracking = original_row[0][2]
    print("S2_NATIVE_DUPLICATE_SERVE_DURING_REAL_EXEC_NO_RECOVERY=PASS", flush=True)
    assert pid(CTL) == controller and os.path.samefile(f"/proc/{controller}/exe", BIN)
    subprocess.run(["systemctl", "kill", "--kill-who=main", "--signal=SIGKILL", CTL],
                   check=True, timeout=10)
    deadline = time.monotonic() + 45
    while time.monotonic() < deadline:
        current = int(property_value(CTL, "MainPID") or "0")
        if current > 1 and current != controller:
            break
        time.sleep(.2)
    else:
        raise AssertionError("controller did not restart")
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    wait_mcp_listener()
    assert os.path.samefile(f"/proc/{current}/exe", BIN)
    print("S2_NATIVE_EXACT_SERVICE_PROCESS_KILLED=PASS", oid, native_id, controller, current, flush=True)
    subprocess.run([probe, "resume"], env=env, check=True, timeout=250)
    assert rows("SELECT state,native_operation_id,tracking_id,box_uuid FROM operations WHERE id=?", (oid,)) == [
        ("UNKNOWN", native_id, tracking, uuid)]
    assert rows("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", (oid,))[0][0] > 0
    assert rows("SELECT count(*) FROM operations WHERE id=? AND state='SUCCEEDED'",
                (original["recorded_operation_id"],)) == [(1,)]
    deadline = time.monotonic() + 45
    while time.monotonic() < deadline:
        try:
            count = incus("exec", NAME, "--project", project, "--", "/bin/sh", "-c",
                          "wc -l < /tmp/bw-s2-crash-marker").strip()
            if count == "1":
                break
        except subprocess.CalledProcessError:
            pass
        time.sleep(.5)
    else:
        raise AssertionError("original guest effect not observed exactly once")
    assert instance(project, NAME)["config"]["volatile.uuid"] == uuid
    assert native_names() == [NAME] and [pid(u) for u in units] == pids and guard_identity() == guard
    print("S2_NATIVE_CRASH_ORIGINAL_GUEST_EFFECT_EXACTLY_ONCE=PASS", flush=True)
    print("P1B_SCOPED_S2_EXEC_CRASH=PASS", oid, native_id, tracking, uuid, flush=True)
    # This disposable runner ends with the original UNKNOWN and native Box.
    # Do not administratively remove it or manufacture a teardown PASS.


if __name__ == "__main__":
    main(*sys.argv[1:])
