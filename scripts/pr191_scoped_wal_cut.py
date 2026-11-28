#!/usr/bin/env python3
"""One-use installed Scoped lifecycle and Exec WAL sync process-cut acceptance.

Only the exact CI installation is instrumented. A transaction-local marker
is written by the original terminal event, with no production hook or
alternate ledger. The Root service, Incus and guard are never restarted.
"""
from contextlib import closing, contextmanager
import base64
import hashlib
import os
from pathlib import Path
import pwd
import re
import secrets
import shutil
import sqlite3
import subprocess
import sys
import tempfile
import time

from mcp_testkit import LoopbackMCP, MCPToolError
from n2_installed_lifecycle import op, pid, token
from pr177_d4_installed_recovery import wait_mcp_listener
from pr178_n2_scoped_lifecycle import acls, incus, instance, verify
from pr191_scoped_ack_loss import guard_identity, native_idle, native_names, wait_poststate
from pr191_scoped_lifecycle_recovery import journal, rows, settle

CTL = "boxweave.service"
BIN = Path("/usr/local/bin/boxweave")
LIB = Path("/usr/local/lib/boxweave-p1b-wal-cut.so")
DROP = Path("/etc/systemd/system/boxweave.service.d")
CONF = DROP / "90-pr191-one-use-wal-cut.conf"
ARM = Path("/var/lib/boxweave/p1b-wal-cut-arm")
WITNESS = Path("/var/lib/boxweave/p1b-wal-cut-witness")
NAME = "p1b-wal-original"
WORK = "p1b/installed-wal-cut"
TABLE = "p1b_test_wal_marker"
TRIGGER = "p1b_test_wal_terminal"
QUARANTINE = "bw-peer-quarantine"
EXEC_MARKER = "/tmp/pr191-wal-exec-original"
EXEC_OUT = b"p1b-wal-stdout"
EXEC_ERR = b"p1b-wal-stderr"
EXEC_ARGV = ["/bin/sh", "-c",
             "printf 'once\\n' >> " + EXEC_MARKER +
             "; printf p1b-wal-stdout; printf p1b-wal-stderr >&2"]
ARGS = {"name": NAME, "cpu": 1, "memory_mib": 256, "disk_gib": 2,
        "image": "local:boxweave-ubuntu-24.04", "internet": True, "start": True}


def systemctl(*args):
    return subprocess.check_output(["systemctl", *args], text=True, timeout=60).strip()


def marker_setup(action, nonce, work=WORK):
    # Install the test-only DDL AFTER clean startup, as in the existing
    # ACK/final-commit fixtures. Never relax the product schema validator.
    detail = {"box_create": "ORIGINAL_CREATE_UUID_ATOMIC",
              "box_delete": "SCOPED_DELETE_ACL_ABSENT_ATOMIC",
              "box_exec": "VERIFIED_NATIVE_EXEC_TERMINAL",
              "box_update": ""}[action]
    state = "VERIFYING" if action == "box_update" else "SUCCEEDED"
    assert work == ("lab/p0-native-sqlite" if action == "box_update" else WORK)
    script = """import sqlite3, sys
db = sqlite3.connect('file:/var/lib/boxweave/ledger.sqlite?mode=rw', uri=True, timeout=10)
action, detail, work, nonce, state = sys.argv[1:]
assert state in ("VERIFYING", "SUCCEEDED")
assert db.execute('PRAGMA journal_mode').fetchone() == ('wal',)
assert not db.execute("SELECT name FROM sqlite_master WHERE name IN ('p1b_test_wal_marker','p1b_test_wal_terminal')").fetchall()
assert not db.execute("SELECT id FROM operations WHERE work_ref=?", (work,)).fetchall()
assert all(v.isascii() and all(c.isalnum() or c in '_/-' for c in v) for v in sys.argv[1:])
with db:
    db.execute('CREATE TABLE p1b_test_wal_marker (nonce TEXT NOT NULL, original TEXT NOT NULL)')
    db.execute("CREATE TRIGGER p1b_test_wal_terminal AFTER INSERT ON operation_events "
               "WHEN NEW.state='" + state + "' AND NEW.safe_detail='" + detail +
               "' AND EXISTS(SELECT 1 FROM operations WHERE id=NEW.operation_id AND action='" + action +
               "' AND work_ref='" + work + "') BEGIN INSERT INTO p1b_test_wal_marker VALUES ('" +
               nonce + "', NEW.operation_id); END")
assert db.execute('PRAGMA wal_checkpoint(TRUNCATE)').fetchone() == (0, 0, 0)
print(db.execute("SELECT rootpage FROM sqlite_master WHERE name='p1b_test_wal_marker'").fetchone()[0])
db.close()
"""
    page = subprocess.check_output(["runuser", "-u", "boxweave", "--", "python3", "-c", script,
                                    action, detail, work, nonce, state], text=True, timeout=30).strip()
    assert page.isdecimal() and int(page) > 0
    return page


def assert_poststate(project, action, oid, uuid, acl):
    if action == "box_create":
        assert native_names() == [NAME]
        raw = instance(project, NAME)
        assert raw["config"]["volatile.uuid"] == uuid
        assert raw["config"]["user.boxweave.create_operation"] == oid
        assert raw["status"] == "Running" and raw["config"]["boot.autostart"] == "true"
        assert raw["expanded_devices"]["eth0"]["security.acls"] == acl
        assert acls() == {QUARANTINE, acl}
    elif action == "box_exec":
        assert native_names() == [NAME] and acls() == {QUARANTINE, acl}
        assert verify(project, NAME, uuid, "true") == acl
        assert instance(project, NAME)["status"] == "Running"
        assert incus("exec", NAME, "--project", project, "--", "cat", EXEC_MARKER) == "once\n"
        wait_poststate(lambda: native_idle(project))
    else:
        assert not native_names() and acls() == {QUARANTINE}
    assert native_idle(project)


@contextmanager
def cold_ledger_rows(source):
    """Observe a stable private database/WAL copy without its shared index.

    Only call with an independently quiescent writer. The copy is never
    installed, exported, or used to recover the original.
    """
    source = Path(source)
    wal = Path(str(source) + "-wal")
    paths = (source, wal)
    assert all(p.is_file() and not p.is_symlink() for p in paths)

    def digests():
        return tuple(hashlib.sha256(p.read_bytes()).digest() for p in paths)

    before = digests()
    with tempfile.TemporaryDirectory(prefix="p1b-cold-ledger-") as tmp:
        target = Path(tmp) / "ledger.sqlite"
        copies = (target, Path(str(target) + "-wal"))
        for original, copy in zip(paths, copies):
            shutil.copyfile(original, copy)
            copy.chmod(0o600)
        assert before == digests() == tuple(hashlib.sha256(p.read_bytes()).digest() for p in copies)
        with closing(sqlite3.connect(target)) as db:
            db.execute("PRAGMA query_only=ON")
            yield lambda sql, params=(): db.execute(sql, params).fetchall()
        assert before == digests()


def assert_ledger(action, state, oid, uuid, acl, nonce=None, query=rows, native_id=None):
    terminal = state == "SUCCEEDED"
    assert query("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", (oid,)) == [(int(terminal),)]
    if nonce is not None:
        assert query("SELECT nonce,original FROM " + TABLE) == ([(nonce, oid)] if terminal else [])
    else:
        assert not query("SELECT name FROM sqlite_master WHERE name IN (?,?)", (TABLE, TRIGGER))
    if action == "box_create":
        assert query("SELECT state FROM owned_boxes WHERE uuid=?", (uuid,)) == ([("OWNED",)] if terminal else [])
        assert query("SELECT state,acl FROM network_bindings WHERE box_uuid=?", (uuid,)) == ([("PINNED", acl)] if terminal else [])
        assert query("SELECT state,operation_id FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == ([("RESERVED", oid)] * 2 if terminal else [])
        if not terminal:
            assert query("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", (oid,)) == [(5,)]
    elif action == "box_exec":
        assert native_id
        assert query("SELECT state FROM owned_boxes WHERE uuid=?", (uuid,)) == [("OWNED",)]
        assert query("SELECT state,acl FROM network_bindings WHERE box_uuid=?", (uuid,)) == [("PINNED", acl)]
        assert query("SELECT state FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == [("RESERVED",)] * 2
        prefix = "/1.0/instances/" + NAME + "/logs/exec-output/exec_" + native_id
        output = query("SELECT stdout_path,stderr_path,exit_code FROM exec_outputs WHERE operation_id=?", (oid,))
        assert output == ([(prefix + ".stdout", prefix + ".stderr", 0)] if terminal else [])
        if not terminal:
            assert query("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", (oid,))[0][0] > 0
    else:
        assert query("SELECT state FROM owned_boxes WHERE uuid=?", (uuid,)) == [("DELETED" if terminal else "OWNED",)]
        assert query("SELECT state FROM network_bindings WHERE box_uuid=?", (uuid,)) == [("RELEASED" if terminal else "PINNED",)]
        fences = query("SELECT state,operation_id FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,))
        assert len(fences) == 2 and all(f[0] == ("RELEASED" if terminal else "RESERVED") for f in fences)
        if terminal:
            assert all(f[1] == oid for f in fences)
        else:
            assert query("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", (oid,))[0][0] > 0


def refused(c, tool, args):
    try:
        c.call(tool, args)
    except MCPToolError:
        return
    raise AssertionError("unsafe or inconsistent request unexpectedly accepted: " + tool)


def verify_exec_restart(c, final, request):
    oid = final["operation_id"]
    uuid = final["box_uuid"]
    # An exact-key replay was already checked by the caller. Changed intent
    # must be rejected even when the terminal transaction was committed.
    refused(c, "box_exec", dict(request, argv=["/bin/true"]))
    if final["state"] == "UNKNOWN":
        for stream in ("stdout", "stderr"):
            refused(c, "operation_output_get", {"operation_id": oid, "stream": stream})
        refused(c, "box_exec", dict(request, idempotency_key="p1b-wal-exec-fresh-refused"))
        assert c.operation_get(oid)["state"] == "UNKNOWN"
    else:
        assert final["state"] == "SUCCEEDED"
        for stream, expected in (("stdout", EXEC_OUT), ("stderr", EXEC_ERR)):
            value = c.operation_output(oid, stream=stream)
            assert base64.b64decode(value["content_base64"], validate=True) == expected
            assert value["operation_id"] == oid and value["exit_code"] == 0
            assert value["command_success"] is True and value["ended"] is True
        fresh = op(c, "box_exec", {"box_id": uuid, "argv": ["/bin/true"]}, "p1b-wal-exec-fresh")
        assert fresh["operation_id"] != oid and fresh["state"] == "SUCCEEDED"
    assert c.host_status()["peer_mode"] == "SCOPED_V2"


def main(project, receipt, candidate_sha, case):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    kind, phase = case.split("-")
    assert kind in ("create", "delete", "exec") and phase in ("before", "after")
    assert project == "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    assert hashlib.sha256(BIN.read_bytes()).hexdigest() == candidate_sha
    assert not any(os.path.lexists(p) for p in (DROP, LIB, ARM, WITNESS))
    assert not systemctl("show", CTL, "-p", "DropInPaths", "--value")
    assert not native_names() and acls() == {QUARANTINE}
    assert rows("SELECT COUNT(*) FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    action = "box_" + kind
    units = ("incus.service", "boxweave-scoped-lifecycle.service")
    pids = [pid(u) for u in units]
    guard = guard_identity()
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
        if kind in ("delete", "exec"):
            uuid = op(c, "box_create", ARGS, "p1b-wal-seed")["box_uuid"]
            acl = verify(project, NAME, uuid, "true")
            if kind == "delete":
                op(c, "box_stop", {"box_id": uuid}, "p1b-wal-stop")
                assert instance(project, NAME)["status"] == "Stopped"
                args = {"box_id": uuid}
            else:
                assert instance(project, NAME)["status"] == "Running"
                args = {"box_id": uuid, "argv": EXEC_ARGV}
        else:
            uuid, acl, args = None, None, ARGS
    request = dict(args, idempotency_key="pr191-wal-" + case, work_ref=WORK)
    nonce = secrets.token_hex(16)
    systemctl("stop", CTL)
    subprocess.run(["cc", "-shared", "-fPIC", "-O2", "-std=gnu11", "-Wall", "-Wextra", "-Werror",
                    "-o", str(LIB), "scripts/pr191_scoped_wal_cut.c", "-ldl"], check=True, timeout=30)
    LIB.chmod(0o755)
    assert LIB.stat().st_uid == 0
    DROP.mkdir(mode=0o755)
    with CONF.open("x") as f:
        f.write("[Service]\nRestart=no\n"
                "Environment=GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted\n"
                f"Environment=LD_PRELOAD={LIB} BW_P1B_WAL_NONCE={nonce} BW_P1B_WAL_PHASE={phase}\n")
    systemctl("daemon-reload")
    systemctl("start", CTL)
    wait_mcp_listener()
    controller = pid(CTL)
    assert os.path.samefile("/proc/" + str(controller) + "/exe", BIN)
    assert str(LIB) in Path("/proc/" + str(controller) + "/maps").read_text()
    page = marker_setup(action, nonce)
    with ARM.open("x") as f:
        f.write(page)
    ARM.chmod(0o644)
    # The admitted operation is asynchronous. Never reissue an ambiguous
    # admission; any missing receipt is a failure and original is retained.
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        accepted = c.call(action, request).structured
    oid = accepted["operation_id"]
    assert re.fullmatch("[0-9a-f]{32}", oid)
    deadline = time.monotonic() + 180
    while time.monotonic() < deadline and not WITNESS.exists():
        time.sleep(.2)
    assert WITNESS.read_text() == f"P1B_EXACT_WAL_COMMIT_SYNC_CUT {phase} {controller}\n"
    deadline = time.monotonic() + 10
    while time.monotonic() < deadline and systemctl("show", CTL, "-p", "MainPID", "--value") != "0":
        time.sleep(.1)
    assert systemctl("show", CTL, "-p", "MainPID", "--value") == "0"
    assert systemctl("show", CTL, "-p", "ExecMainCode", "--value") == "2"
    assert systemctl("show", CTL, "-p", "ExecMainStatus", "--value") == "9"
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    print("P1B_SCOPED_WAL_EXACT_CONTROLLER_CUT=PASS", case, oid, controller, flush=True)
    # Independent read-only oracle before restart. SIGKILL before fsync is
    # not power loss: SQLite may recover a complete commit or none of it.
    assert rows("PRAGMA integrity_check") == [("ok",)]
    original = rows("SELECT id,tracking_id,state,native_operation_id FROM operations WHERE work_ref=? AND action=?", (WORK, action))
    assert len(original) == 1 and original[0][:2] == (oid, accepted["tracking_id"])
    _, _, pre, native_id = original[0]
    assert pre in ("VERIFYING", "SUCCEEDED") and native_id
    if kind != "exec":
        rec, hashes = journal(oid, kind.upper(), native_id)
        if kind == "create":
            uuid, acl = rec["UUID"], rec["Binding"]["ACL"]
        else:
            assert rec["UUID"] == uuid and rec["Binding"]["ACL"] == acl
    else:
        assert rows("SELECT box_uuid,incarnation FROM operations WHERE id=?", (oid,)) == [(uuid, uuid)]
    wait_poststate(lambda: native_idle(project))
    assert_poststate(project, action, oid, uuid, acl)
    assert_ledger(action, pre, oid, uuid, acl, nonce, native_id=native_id)
    if phase == "after":
        # A sync is not the WAL-index publication. Independently rebuild the
        # index from copied bytes to distinguish durable data from live visibility.
        # No installed ledger writes, no recovery of the copy.
        with cold_ledger_rows(Path("/var/lib/boxweave/ledger.sqlite")) as cold:
            assert cold("PRAGMA integrity_check") == [("ok",)]
            durable = cold("SELECT id,tracking_id,state,native_operation_id FROM operations WHERE work_ref=? AND action=?", (WORK, action))
            assert len(durable) == 1 and durable[0][:2] == original[0][:2] and durable[0][3] == native_id
            disk_state = durable[0][2]
            assert disk_state in ("VERIFYING", "SUCCEEDED")
            assert_ledger(action, disk_state, oid, uuid, acl, nonce, query=cold, native_id=native_id)
            print("P1B_SCOPED_WAL_VISIBILITY", case, "live=" + pre, "cold=" + disk_state, oid, flush=True)
            assert disk_state == "SUCCEEDED", "post-sync WAL lacks complete durable commit"
        # Do not write test DDL or restart over divergent live/cold evidence.
        # The existing post-sync acceptance remains strict.
        assert pre == "SUCCEEDED", "live/cold visibility differs; original retained"
    print("P1B_SCOPED_WAL_SQLITE_NATIVE_CROSSCHECK=PASS", case, pre, oid, native_id, uuid, flush=True)
    # The schema validator must still refuse unexpected tables/triggers.
    # The controller is dead. Preserve marker proof above and remove ONLY
    # the exact test-owned objects before starting the original clean unit.
    ARM.unlink()
    script = """import sqlite3, sys
db = sqlite3.connect('file:/var/lib/boxweave/ledger.sqlite?mode=rw', uri=True, timeout=10)
nonce, oid, state = sys.argv[1:]
assert db.execute('SELECT nonce,original FROM p1b_test_wal_marker').fetchall() == ([(nonce, oid)] if state == 'SUCCEEDED' else [])
with db:
    db.execute('DROP TRIGGER p1b_test_wal_terminal')
    db.execute('DROP TABLE p1b_test_wal_marker')
db.close()
"""
    subprocess.run(["runuser", "-u", "boxweave", "--", "python3", "-c", script, nonce, oid, pre],
                   check=True, timeout=30)
    assert_ledger(action, pre, oid, uuid, acl, native_id=native_id)
    # No native or original-ledger data cleanup.
    CONF.unlink()
    DROP.rmdir()
    LIB.unlink()
    systemctl("daemon-reload")
    systemctl("start", CTL)
    wait_mcp_listener()
    assert pid(CTL) != controller and [pid(u) for u in units] == pids and guard_identity() == guard
    assert not systemctl("show", CTL, "-p", "DropInPaths", "--value")
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        final = c.operation_get(oid)
        assert final["tracking_id"] == accepted["tracking_id"] and final["native_operation_id"] == native_id
        assert final["state"] == ("UNKNOWN" if pre == "VERIFYING" else "SUCCEEDED")
        assert_ledger(action, final["state"], oid, uuid, acl, native_id=native_id)
        replay = c.call(action, request).structured
        assert replay["operation_id"] == oid and replay["tracking_id"] == final["tracking_id"] and replay["duplicate"]
        assert replay["state"] == final["state"]
        if kind == "exec":
            # Generic Exec has no completed Scoped lifecycle journal. Do not
            # falsely settle an UNKNOWN; retain the original and resource lock.
            verify_exec_restart(c, final, request)
            assert c.operation_get(oid)["state"] == final["state"]
            assert_ledger(action, final["state"], oid, uuid, acl, native_id=native_id)
            print("P1B_SCOPED_EXEC_WAL_ATOMIC=PASS", case, final["state"], oid, native_id, flush=True)
        else:
            if final["state"] == "UNKNOWN":
                # Recovery is separately journaled and never redispatches the effect.
                settle(c, final, kind.upper(), uuid, "p1b-wal-" + case + "-reconcile")
            assert c.operation_get(oid)["state"] == "SUCCEEDED"
            assert_ledger(action, "SUCCEEDED", oid, uuid, acl)
            assert journal(oid, kind.upper(), native_id) == (rec, hashes)
        assert_poststate(project, action, oid, uuid, acl)
        assert rows("SELECT id FROM operations WHERE work_ref=? AND action=?", (WORK, action)) == [(oid,)]
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
    print("P1B_SCOPED_INSTALLED_WAL_CUT=PASS", case, flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
