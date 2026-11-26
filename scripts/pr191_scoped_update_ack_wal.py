#!/usr/bin/env python3
"""One-use installed Scoped Update NativeAck WAL cut, with real native 202.

Reuse the original P0 restricted native proxy and official SDK, and the
existing exact transaction marker interposer. No runtime code change,
alternate installed ledger, native cleanup or release of an UNKNOWN.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import secrets
import subprocess
import sys
import tempfile
import time

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import pid, token
from pr177_d4_installed_recovery import wait_mcp_listener
from pr178_n2_scoped_lifecycle import acls, binding, instance, verify
from pr191_scoped_ack_loss import (CTL, QUARANTINE, guard_identity,
                                   native_idle, native_names, snapshot,
                                   wait_poststate)
from pr191_scoped_lifecycle_recovery import ROOT, rows
from pr191_scoped_update_transport import (BIN, DB, SOCKET, command,
                                          restricted, supervised, wait_port)
from pr191_scoped_wal_cut import (ARM, WITNESS, LIB, TABLE, TRIGGER,
                                  cold_ledger_rows, marker_setup)

WORK = "lab/p0-native-sqlite"
NAMES = {"before": "bw-s2-p0-before-110j", "after": "bw-s2-p0-after-110k"}


def ack_proof(query, original, actual, nonce, phase):
    oid, tracking, uuid = (original[k] for k in ("update_id", "tracking_id", "box_id"))
    observed = query("SELECT id,tracking_id,state,native_operation_id,box_uuid,incarnation "
                     "FROM operations WHERE work_ref=? AND action='box_update'", (WORK,))
    assert len(observed) == 1
    row = observed[0]
    assert row[:2] == (oid, tracking) and row[4:] == (uuid, uuid)
    committed = row[2] == "VERIFYING"
    assert row[2:4] == (("VERIFYING", actual) if committed else ("DISPATCHING", ""))
    assert phase != "after" or committed, "post-sync native ACK not durable"
    events = query("SELECT state,safe_detail FROM operation_events WHERE operation_id=? ORDER BY seq", (oid,))
    assert [v[0] for v in events] == ["ADMITTED", "DISPATCHING"] + (["VERIFYING"] if committed else [])
    assert query("SELECT nonce,original FROM " + TABLE) == ([(nonce, oid)] if committed else [])
    assert query("PRAGMA integrity_check") == [("ok",)]
    return row[2], row[3]


def main(project, receipt, candidate_sha, phase):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted" and phase in NAMES
    account = pwd.getpwnam("boxweave")
    assert project == "user-" + str(account.pw_uid)
    work = Path(receipt).parent
    assert work.parent == Path("/tmp") and work.name.startswith("bw-n2-installed.")
    assert Path(receipt).name == "actor.receipt" and Path(receipt).stat().st_mode & 0o077 == 0
    assert command("git", "rev-parse", "HEAD") == os.environ["N2_EXACT_HEAD"]
    assert hashlib.sha256(BIN.read_bytes()).hexdigest() == candidate_sha
    assert hashlib.sha256((work / "candidate-a").read_bytes()).hexdigest() == candidate_sha
    assert not any(os.path.lexists(p) for p in (ARM, WITNESS, LIB))
    assert not native_names() and acls() == {QUARANTINE}
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING')")
    assert not rows("SELECT id FROM operations WHERE work_ref=?", (WORK,))
    units = ("incus.service", "boxweave-scoped-lifecycle.service")
    pids = [pid(u) for u in units]
    guard = guard_identity()
    unit_before = Path("/etc/systemd/system/boxweave.service").read_bytes()
    lab = Path(tempfile.mkdtemp(prefix="bw-v1-s2-ghci.", dir="/tmp"))
    lab.chmod(0o711)
    fixture = lab / ("p0-" + phase)
    fixture.mkdir(mode=0o700)
    os.chown(fixture, account.pw_uid, account.pw_gid)
    saved = fixture / "original.json"
    credential = token(receipt)
    def sdk(mode, installed):
        env = dict(os.environ, BW_S2_P0_SCOPED="1", BW_S2_P0_DIR=str(fixture),
                   BW_S2_P0_PROJECT=project, BW_S2_P0_TOKEN=credential,
                   BW_S2_P0_INSTALLED_LISTENER="1" if installed else "0")
        subprocess.run(restricted(str(work / "s2-p0-native-sqlite-probe"), mode),
                       env=env, check=True, timeout=210)
    with LoopbackMCP(token=credential, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
    sdk("seed", True)
    cp = json.loads(saved.read_text())
    uuid, create = cp["box_id"], cp["create_id"]
    assert re.fullmatch(r"[0-9a-f-]{36}", uuid) and not cp["update_id"]
    name = NAMES[phase]
    acl = verify(project, name, uuid, "true")
    raw = instance(project, name)
    assert raw["status"] == "Stopped" and raw["config"]["limits.cpu"] == "1"
    nic, pin = raw["expanded_devices"]["eth0"], binding(uuid)
    fences = rows("SELECT * FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,))
    assert len(fences) == 2
    prior = {oid: snapshot(oid) for (oid,) in rows("SELECT id FROM operations")}
    previous = pid(CTL)
    def native_proof():
        raw = instance(project, name)
        assert raw["status"] == "Stopped" and raw["config"]["limits.cpu"] == "2"
        assert raw["config"]["volatile.uuid"] == uuid
        assert raw["config"]["user.boxweave.create_operation"] == create
        assert raw["expanded_devices"]["eth0"] == nic and verify(project, name, uuid, "true") == acl
        assert binding(uuid) == pin and native_names() == [name] and acls() == {QUARANTINE, acl}
        assert rows("SELECT * FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == fences
        assert rows("SELECT state FROM owned_boxes WHERE uuid=?", (uuid,)) == [("OWNED",)]
        assert [pid(u) for u in units] == pids and guard_identity() == guard
        assert all(snapshot(oid) == v for oid, v in prior.items())
    command("systemctl", "stop", CTL)
    assert command("systemctl", "show", CTL, "-p", "MainPID", "--value") == "0"
    wait_port(18650, False)
    assert not command("systemctl", "show", CTL, "-p", "DropInPaths", "--value")
    subprocess.run(["cc", "-shared", "-fPIC", "-O2", "-std=gnu11", "-Wall", "-Wextra", "-Werror",
                    "-o", str(LIB), "scripts/pr191_scoped_wal_cut.c", "-ldl"], check=True, timeout=30)
    LIB.chmod(0o755)
    nonce = secrets.token_hex(16)
    proxy, marker, arm = (fixture / n for n in ("proxy.sock", "native-202.txt", "arm"))
    with supervised(restricted(str(work / "s2-p0-native-sqlite-proxy"), "-listen", str(proxy),
                    "-upstream", SOCKET, "-project", project, "-name", name,
                    "-arm", str(arm), "-marker", str(marker)), lab / "proxy.log") as prox:
        for _ in range(60):
            assert prox.poll() is None
            if proxy.is_socket():
                break
            time.sleep(.5)
        assert proxy.is_socket() and not marker.exists() and not arm.exists()
        with supervised(restricted("LD_PRELOAD=" + str(LIB), "BW_P1B_WAL_NONCE=" + nonce,
                        "BW_P1B_WAL_PHASE=" + phase, str(BIN), "serve", "-db", DB,
                        "-project", project, "-incus-socket", str(proxy),
                        "-listen", "127.0.0.1:18657", "-scoped-peer-fresh"),
                        lab / "controller.log") as ctl:
            wait_port(18657, True, ctl)
            with LoopbackMCP("http://127.0.0.1:18657/mcp", token=credential, timeout=60) as c:
                assert c.host_status()["peer_mode"] == "SCOPED_V2"
            children = command("ps", "--ppid", str(ctl.pid), "-o", "pid=").split()
            assert len(children) == 1
            controller = int(children[0])
            assert os.path.samefile("/proc/" + str(controller) + "/exe", BIN)
            assert str(LIB) in Path("/proc/" + str(controller) + "/maps").read_text()
            page = marker_setup("box_update", nonce, WORK)
            with ARM.open("x") as f:
                f.write(page)
            ARM.chmod(0o644)
            sdk("apply", False)
            cp = json.loads(saved.read_text())
            assert cp["create_id"] == create and cp["box_id"] == uuid
            assert re.fullmatch(r"[0-9a-f]{32}", cp["update_id"])
            assert re.fullmatch(r"[0-9a-f]{32}", cp["tracking_id"])
            assert ctl.wait(timeout=180) in (-9, 137)
            assert WITNESS.read_text() == f"P1B_EXACT_WAL_COMMIT_SYNC_CUT {phase} {controller}\n"
            assert not Path("/proc/" + str(controller)).exists()
        wait_port(18657, False)
        match = re.fullmatch(r"NATIVE_REAL_UPDATE_202_ACCEPTED=([0-9a-f-]{36})\n", marker.read_text())
        assert match and arm.read_text() == "arm-on-real-native-202\n"
        actual = match[1]
        assert not (fixture / "unexpected-native-dispatch.txt").exists()
        wait_poststate(lambda: native_idle(project) and instance(project, name)["config"]["limits.cpu"] == "2")
        native_proof()
        with cold_ledger_rows(Path(DB)) as cold:
            durable = ack_proof(cold, cp, actual, nonce, phase)
        live = ack_proof(rows, cp, actual, nonce, phase)
        assert live == durable, "live/cold ACK visibility differs; preserve original"
        oid = cp["update_id"]
        assert snapshot(oid)[1] == [(project, "box:" + name)] and not snapshot(oid)[3]
        assert not os.path.lexists(ROOT / ("op-" + oid))
        print("P1B_SCOPED_UPDATE_NATIVE_ACK_WAL_CUT=PASS", phase, oid, actual, live[0], flush=True)
        # Remove only the exact test-owned schema objects, after proof.
        ARM.unlink()
        script = """import sqlite3,sys
db=sqlite3.connect('file:/var/lib/boxweave/ledger.sqlite?mode=rw',uri=True,timeout=10)
nonce,oid,state=sys.argv[1:]
assert db.execute('SELECT nonce,original FROM p1b_test_wal_marker').fetchall() == ([(nonce,oid)] if state=='VERIFYING' else [])
with db:
    db.execute('DROP TRIGGER p1b_test_wal_terminal')
    db.execute('DROP TABLE p1b_test_wal_marker')
db.close()
"""
        subprocess.run(restricted("python3", "-c", script, nonce, oid, live[0]), check=True, timeout=30)
        assert not rows("SELECT name FROM sqlite_master WHERE name IN (?,?)", (TABLE, TRIGGER))
        marker_before = marker.read_bytes()
    LIB.unlink()
    command("systemctl", "start", CTL)
    wait_mcp_listener()
    assert pid(CTL) != previous and Path("/etc/systemd/system/boxweave.service").read_bytes() == unit_before
    assert rows("PRAGMA integrity_check") == [("ok",)]
    before = snapshot(oid)
    assert before[0][0][:6] == (oid, cp["tracking_id"], "UNKNOWN", live[1], uuid, uuid)
    assert before[1] == [(project, "box:" + name)] and not before[3]
    assert [e[1] for e in before[2]].count("UNKNOWN") == 1
    assert not any(e[1] == "SUCCEEDED" for e in before[2])
    sdk("resume", True)
    assert snapshot(oid) == before
    assert rows("SELECT id FROM operations WHERE work_ref=?", (WORK,)) == [(oid,)]
    assert rows("SELECT id FROM operations ORDER BY id") == sorted((i,) for i in list(prior) + [oid])
    assert marker.read_bytes() == marker_before and not (fixture / "unexpected-native-dispatch.txt").exists()
    native_proof()
    print("P1B_SCOPED_UPDATE_NATIVE_ACK_WAL_RESTART_RETAINED=PASS", phase, oid, cp["tracking_id"], uuid, flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
