#!/usr/bin/env python3
"""One-use exact installed Scoped ACK-loss acceptance.

Each runner owns one original; a failed ACK may leave a quarantined native
Create or an absent native Delete with a retained ACL. Never unlock, adopt,
replay a native effect, or clean an unresolved original.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import socket
import sqlite3
import subprocess
import sys
import time

from mcp_testkit import LoopbackMCP, MCPToolError
from n2_installed_lifecycle import op, pid, token
from pr177_d4_installed_recovery import wait_mcp_listener
from pr178_n2_scoped_lifecycle import acls, binding, incus, instance, verify
from pr178_n4_native import guest, SERVER
from pr191_scoped_lifecycle_recovery import DB, ROOT, rows

NAME = "p1b-first-ack-cut"
TRIGGER = "p1b_only_ci_native_ack_fault"
CTL = "boxweave.service"
QUARANTINE = "bw-peer-quarantine"
ARGS = {"name": NAME, "cpu": 1, "memory_mib": 256, "disk_gib": 2,
        "image": "local:boxweave-ubuntu-24.04", "internet": True, "start": True}


def snapshot(oid):
    return (
        rows("SELECT id,tracking_id,state,native_operation_id,box_uuid,incarnation,updated_at,error_class FROM operations WHERE id=?", (oid,)),
        rows("SELECT layer,resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key", (oid,)),
        rows("SELECT seq,state,safe_detail FROM operation_events WHERE operation_id=? ORDER BY seq", (oid,)),
        rows("SELECT decision,evidence_digest FROM recovery_audit WHERE operation_id=? ORDER BY seq", (oid,)),
    )


def work_snapshot(work):
    return [(oid, snapshot(oid)) for (oid,) in rows("SELECT id FROM operations WHERE work_ref=? ORDER BY id", (work,))]


def native_names():
    return sorted(incus("list", "--all-projects", "-c", "n", "-f", "csv").splitlines())


def native_idle(project):
    return not json.loads(incus("operation", "list", "--project", project, "-f", "json"))


def guard_identity():
    # Guard is an active oneshot, not a running process.
    state = subprocess.check_output(["systemctl", "show", "boxweave-scoped-guard.service",
                                      "-p", "ActiveState", "-p", "ActiveEnterTimestampMonotonic"], text=True)
    assert "ActiveState=active" in state
    return state


def wait_poststate(predicate):
    end = time.monotonic() + 75
    while time.monotonic() < end:
        if predicate():
            return
        time.sleep(.4)
    raise RuntimeError("native poststate deadline; original preserved")


def ack_fault_ddl(action, work, preserve_native=False):
    assert action in ("box_create", "box_delete", "box_update")
    assert work == "p1b/first-ack-" + action
    if preserve_native:
        assert action == "box_update"
        # Roll back only the normal ACK transaction, leaving the
        # receipt-preserving UNKNOWN fallback available.
        return (f"CREATE TRIGGER {TRIGGER} BEFORE INSERT ON operation_events "
                f"WHEN NEW.state='VERIFYING' AND EXISTS(SELECT 1 FROM operations "
                f"WHERE id=NEW.operation_id AND action='{action}' AND work_ref='{work}') "
                "BEGIN SELECT RAISE(ABORT,'P1B_ONE_USE_ACK_EVENT_FAULT'); END")
    # Refuse both the normal ACK and the receipt-preserving fallback.
    return (f"CREATE TRIGGER {TRIGGER} BEFORE UPDATE OF native_operation_id ON operations "
            f"WHEN OLD.action='{action}' AND OLD.work_ref='{work}' "
            "AND OLD.native_operation_id='' AND NEW.native_operation_id!='' "
            "BEGIN SELECT RAISE(ABORT,'P1B_ONE_USE_ACK_DURABILITY_FAULT'); END")


def ack_cut(c, action, args, preserve_native=False):
    work = "p1b/first-ack-" + action
    request = dict(args, idempotency_key="pr191-first-ack-" + action, work_ref=work)
    assert action in ("box_create", "box_delete", "box_update")
    assert not rows("SELECT id FROM operations WHERE work_ref=?", (work,))
    assert not rows("SELECT name FROM sqlite_master WHERE type='trigger' AND name=?", (TRIGGER,))
    with sqlite3.connect(DB + "?mode=rw", uri=True, timeout=15) as db:
        db.execute(ack_fault_ddl(action, work, preserve_native))
    try:
        accepted = c.call(action, request).structured
        oid = accepted["operation_id"]
        final = c.poll_operation(oid, seconds=150)
        assert final["state"] == "UNKNOWN", final
        native_id = final.get("native_operation_id", "")
        if preserve_native:
            assert re.fullmatch(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}", native_id)
        else:
            assert not native_id
        assert final["tracking_id"] == accepted["tracking_id"]
        assert rows("SELECT id FROM operations WHERE work_ref=? AND action=?", (work, action)) == [(oid,)]
        assert not os.path.lexists(ROOT / ("op-" + oid))
        assert snapshot(oid)[1] and not snapshot(oid)[3]
        return final, request
    finally:
        # Only the one-use test trigger is removed. Original and native
        # evidence are retained even if an assertion fails.
        with sqlite3.connect(DB + "?mode=rw", uri=True, timeout=15) as db:
            db.execute(f"DROP TRIGGER {TRIGGER}")


def assert_blocked(c, original, action, request):
    oid = original["operation_id"]
    before = snapshot(oid)
    before_work = work_snapshot(request["work_ref"])
    diag = c.call("operation_recover", {"operation_id": oid, "mode": "inspect"}).structured
    assert diag["diagnosis"]["disposition"] == "BLOCKED", diag
    assert diag["diagnosis"]["reason"] == "SCOPED_LIFECYCLE_ORIGINAL_ACK_UNPROVED", diag
    assert diag["diagnosis"]["claim_protection"] == "RETAINED" and not diag["native_retried"]
    args = {"operation_id": oid, "mode": "reconcile", "idempotency_key": "pr191-first-ack-inspection"}
    first = c.call("operation_recover", args).structured
    rec = first["recovery"]
    assert rec["disposition"] == "BLOCKED" and rec["operation"]["state"] == "FAILED", first
    assert not first["native_retried"] and rec["operation"]["operation_id"] != oid
    dup = c.call("operation_recover", args).structured
    assert dup["recovery"]["operation"]["duplicate"] and not dup["native_retried"]
    assert dup["recovery"]["operation"]["operation_id"] == rec["operation"]["operation_id"]
    # Exact-key replay must be the original receipt, not a second dispatch.
    replay = c.call(action, request).structured
    assert replay["operation_id"] == oid and replay["tracking_id"] == original["tracking_id"]
    assert replay["state"] == "UNKNOWN" and replay["duplicate"], replay
    try:
        c.call(action, dict(request, idempotency_key="pr191-first-ack-fresh-refused"))
    except MCPToolError:
        pass
    else:
        raise AssertionError("fresh mutation admitted against unresolved original")
    assert snapshot(oid) == before
    assert not os.path.lexists(ROOT / ("op-" + oid))
    assert rows("SELECT id FROM operations WHERE work_ref=? AND action=?", (request["work_ref"], action)) == [(oid,)]
    assert work_snapshot(request["work_ref"]) == before_work
    print("P1B_SCOPED_ACK_ORIGINAL_RETAINED", oid, original["tracking_id"],
          rec["operation"]["operation_id"], flush=True)


def assert_update_locked(c, original, request, project, preserve_native):
    oid, uuid = original["operation_id"], original["box_uuid"]
    native_id = original.get("native_operation_id", "")
    before = snapshot(oid)
    before_work = work_snapshot(request["work_ref"])
    assert before[0] == [(oid, original["tracking_id"], "UNKNOWN", native_id,
                          uuid, uuid, before[0][0][6], "NATIVE_ACK_DURABILITY_UNVERIFIED")]
    assert before[1] == [(project, "box:" + NAME)] and not before[3]
    assert not any(row[1] in ("VERIFYING", "SUCCEEDED") for row in before[2])
    assert original["next_action"]["kind"] == "HOST_LOCAL_REVIEW"
    assert not os.path.lexists(ROOT / ("op-" + oid))
    diag = c.call("operation_recover", {"operation_id": oid, "mode": "inspect"}).structured
    assert diag["diagnosis"]["claim_protection"] == "RETAINED" and not diag["native_retried"]
    if not preserve_native:
        assert diag["diagnosis"]["disposition"] == "APPROVAL_REQUIRED"
        assert diag["diagnosis"]["reason"] == "ORIGINAL_NATIVE_ACK_MISSING"
    # Inspect only: a durable receipt may legitimately enable a separate
    # reviewed recovery, but that is not an ACK recovery or native replay.
    replay = c.call("box_update", request).structured
    assert replay["operation_id"] == oid and replay["tracking_id"] == original["tracking_id"]
    assert replay["state"] == "UNKNOWN" and replay["duplicate"]
    for bad in (dict(request, cpu=3),
                dict(request, cpu=3, idempotency_key="pr191-update-ack-fresh-refused")):
        try:
            c.call("box_update", bad)
        except MCPToolError:
            pass
        else:
            raise AssertionError("conflicting Update admitted against unresolved original")
    assert snapshot(oid) == before and work_snapshot(request["work_ref"]) == before_work
    assert rows("SELECT id FROM operations WHERE work_ref=? AND action='box_update'",
                (request["work_ref"],)) == [(oid,)]
    print("P1B_SCOPED_UPDATE_ACK_ORIGINAL_RETAINED", oid, original["tracking_id"],
          native_id or "NO_DURABLE_NATIVE_ID", uuid, flush=True)


def update_case(project, receipt, cut, units, pids, guard, controller):
    preserve_native = cut == "update-receipt"
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
        created = op(c, "box_create", dict(ARGS, start=False), "p1b-update-ack-seed")
        uuid = created["box_uuid"]
        acl = verify(project, NAME, uuid, "true")
        raw = instance(project, NAME)
        assert raw["status"] == "Stopped" and raw["config"]["limits.cpu"] == "1"
        original_nic = raw["expanded_devices"]["eth0"]
        original_binding = binding(uuid)
        original_fences = rows("SELECT * FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,))
        assert len(original_fences) == 2
        original, request = ack_cut(c, "box_update", {"box_id": uuid, "cpu": 2}, preserve_native)
        assert original["box_uuid"] == uuid
        wait_poststate(lambda: native_idle(project) and instance(project, NAME)["config"]["limits.cpu"] == "2")

        def native_proof():
            raw = instance(project, NAME)
            assert native_names() == [NAME] and acls() == {QUARANTINE, acl}
            assert raw["status"] == "Stopped" and raw["config"]["limits.cpu"] == "2"
            assert raw["config"]["user.boxweave.create_operation"] == created["operation_id"]
            assert verify(project, NAME, uuid, "true") == acl
            assert raw["expanded_devices"]["eth0"] == original_nic
            assert binding(uuid) == original_binding
            assert rows("SELECT * FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == original_fences
            assert rows("SELECT uuid,incarnation,state FROM owned_boxes WHERE uuid=?", (uuid,)) == [(uuid, uuid, "OWNED")]
            assert native_idle(project)

        native_proof()
        assert_update_locked(c, original, request, project, preserve_native)
        native_proof()
        before = work_snapshot(request["work_ref"])
    subprocess.run(["systemctl", "restart", CTL], check=True, timeout=60)
    wait_mcp_listener()
    assert pid(CTL) != controller and [pid(u) for u in units] == pids and guard_identity() == guard
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        after = c.operation_get(original["operation_id"])
        assert after["state"] == "UNKNOWN" and after.get("native_operation_id", "") == original.get("native_operation_id", "")
        assert after["tracking_id"] == original["tracking_id"] and work_snapshot(request["work_ref"]) == before
        native_proof()
        assert_update_locked(c, after, request, project, preserve_native)
        native_proof()
    print("P1B_SCOPED_UPDATE_ACK_DURABILITY=PASS", cut, flush=True)


def echo():
    with socket.create_connection(("127.0.0.1", 28286), timeout=3) as sock:
        sock.settimeout(3)
        sock.sendall(b"p1b-ack")
        reply = b""
        while len(reply) < len(b"p1b-ack"):
            part = sock.recv(64)
            assert part
            reply += part
        assert reply == b"p1b-ack"


def assert_forward_absent():
    try:
        with socket.create_connection(("127.0.0.1", 28286), timeout=1):
            pass
    except ConnectionRefusedError:
        return
    raise AssertionError("dependent forward still accepts connections")


def main(project, receipt, candidate_sha, cut):
    assert os.geteuid() == 0 and cut in ("create", "delete", "update", "update-receipt")
    assert os.environ.get("GITHUB_ACTIONS") == "true" and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    assert project == "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    assert hashlib.sha256(Path("/usr/local/bin/boxweave").read_bytes()).hexdigest() == candidate_sha
    assert not native_names() and acls() == {QUARANTINE}
    assert rows("SELECT COUNT(*) FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    units = ("incus.service", "boxweave-scoped-lifecycle.service")
    guard = guard_identity()
    pids = [pid(u) for u in units]
    controller = pid(CTL)
    if cut in ("update", "update-receipt"):
        update_case(project, receipt, cut, units, pids, guard, controller)
        return
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
        if cut == "create":
            original, request = ack_cut(c, "box_create", ARGS)
            assert not original.get("box_uuid")
            wait_poststate(lambda: native_names() == [NAME] and native_idle(project))
            def native_proof():
                raw = instance(project, NAME)
                nic = raw["expanded_devices"]["eth0"]
                assert raw["config"]["user.boxweave.create_operation"] == original["operation_id"]
                assert raw["status"] == "Stopped" and raw["config"]["boot.autostart"] == "false"
                assert nic["security.acls"] == QUARANTINE and nic["security.acls.default.egress.action"] == "reject"
                b = nic["network"]
                expected = {"box:" + NAME, f"create-pin:{b}:acl:{QUARANTINE}",
                            f"create-pin:{b}:mac:{nic['hwaddr']}",
                            f"create-pin:{b}:v4:{nic['ipv4.address']}",
                            f"create-pin:{b}:v6:{nic['ipv6.address']}"}
                assert {key for layer, key in snapshot(original["operation_id"])[1] if layer == project} == expected
                assert not rows("SELECT uuid FROM owned_boxes WHERE name=?", (NAME,))
                assert acls() == {QUARANTINE} and native_names() == [NAME]
                return raw["config"]["volatile.uuid"], tuple(sorted(expected))
            action = "box_create"
        else:
            created = op(c, "box_create", ARGS, "p1b-ack-create")
            uuid = created["box_uuid"]
            acl = verify(project, NAME, uuid, "true")
            guest(project, NAME, "python3", "-c",
                  "import subprocess,sys;subprocess.Popen(['python3','-u','-c',sys.argv[1]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)", SERVER)
            forward = op(c, "port_forward_create", {"box_id": uuid, "protocol": "tcp",
                         "host_bind": "127.0.0.1", "host_port": 28286, "guest_port": 18182},
                         "p1b-ack-forward")["operation_id"]
            echo()
            original, request = ack_cut(c, "box_delete", {"box_id": uuid})
            assert original["box_uuid"] == uuid
            wait_poststate(lambda: not native_names() and native_idle(project))
            def native_proof():
                assert not native_names() and acls() == {QUARANTINE, acl}
                assert rows("SELECT state FROM owned_boxes WHERE uuid=?", (uuid,)) == [("OWNED",)]
                assert binding(uuid)["state"] == "PINNED"
                assert rows("SELECT state,operation_id FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == [
                    ("RESERVED", created["operation_id"]), ("RESERVED", created["operation_id"])]
                assert rows("SELECT state FROM forward_claims WHERE forward_id=?", (forward,)) == [("REVOKED",)]
                children = rows("SELECT o.action,o.state,o.native_operation_id,o.layer FROM operations o JOIN operation_resource_claims c ON c.operation_id=o.id WHERE c.resource_key=? ORDER BY o.action", ("delete-parent:" + original["operation_id"],))
                assert [(a, st, layer) for a, st, nid, layer in children] == [("box_stop", "SUCCEEDED", project), ("port_forward_delete", "SUCCEEDED", project)], children
                assert all(nid for a, st, nid, layer in children), children
                assert_forward_absent()
                return uuid, acl, forward
            action = "box_delete"
        # Observe two independent native/ledger snapshots before and after
        # blocked recovery and across a controller restart. No fake ACK.
        expected = native_proof()
        time.sleep(.4)
        assert native_proof() == expected
        assert_blocked(c, original, action, request)
        assert native_proof() == expected
        before = work_snapshot(request["work_ref"])
    subprocess.run(["systemctl", "restart", CTL], check=True, timeout=60)
    wait_mcp_listener()
    assert pid(CTL) != controller and [pid(u) for u in units] == pids and guard_identity() == guard
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        after = c.operation_get(original["operation_id"])
        assert after["state"] == "UNKNOWN" and not after.get("native_operation_id")
        assert after["tracking_id"] == original["tracking_id"] and work_snapshot(request["work_ref"]) == before
        assert native_proof() == expected
        assert_blocked(c, original, action, request)
        assert native_proof() == expected
    print("P1B_SCOPED_ACK_LOSS=PASS", cut, flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
