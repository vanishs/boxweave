#!/usr/bin/env python3
"""Independent fixed-ledger checks for the full installed S2 probe.

Only read the known disposable runner ledger. Never reconcile or remove
unproved originals. Pure oracle tests use private in-memory rows.
"""
import hashlib
import os
from pathlib import Path
import pwd
import re
import sys

from pr178_n2_scoped_lifecycle import acls
from pr191_scoped_ack_loss import guard_identity, native_names
from pr191_scoped_lifecycle_recovery import rows
from n2_installed_lifecycle import pid, token
from mcp_testkit import LoopbackMCP, MCPToolError

MAIN = "bw-s2-ghci-110a"
PEER = "bw-s2-peer-native-110c"
UNITS = ("boxweave.service", "boxweave-scoped-lifecycle.service", "incus.service")
KEYS = (
    ("native-s2-create-once-20261110", "box_create"),
    ("native-s2-update-cpu-memory-01", "box_update"),
    ("native-s2-start-000000001", "box_start"),
    ("native-s2-actor-one-shared-work-exec-01", "box_exec"),
    ("native-s2-actor-two-shared-work-exec-01", "box_exec"),
    ("native-s2-stop-0000000001", "box_stop"),
    ("native-s2-delete-00000001", "box_delete"),
    ("native-s2-reused-name-fresh-uuid-01", "box_create"),
    ("native-s2-delete-new-incarnation-01", "box_delete"),
)


def by_key(read, key):
    return read("SELECT id,tracking_id,actor_id,token_id,layer,action,state,box_uuid,incarnation,native_operation_id "
                "FROM operations WHERE key_digest=? ORDER BY actor_id",
                (hashlib.sha256(key.encode()).hexdigest(),))


def verify_history(read, project, actor, queued, foreign):
    assert re.fullmatch(r"[0-9a-f]{32}", queued) and re.fullmatch(r"[0-9a-f]{32}", foreign)
    assert queued != foreign
    for work, layer, count in (("lab/history-120", project, 132),
                               ("lab/queued-cancel", project, 1),
                               ("lab/foreign-layer", project + "-foreign", 1)):
        records = read("SELECT id,actor_id,token_id,layer,state,native_operation_id "
                       "FROM operations WHERE work_ref=? ORDER BY id", (work,))
        assert len(records) == count and len({x[0] for x in records}) == count
        assert all(x[1:] == (actor, actor, layer, "CANCELLED", "") for x in records)
        if count == 1:
            assert records[0][0] == (queued if work == "lab/queued-cancel" else foreign)
    cancels = by_key(read, "native-s2-queued-cancel-tool-original-01")
    assert len(cancels) == 1 and cancels[0][0] != queued
    assert cancels[0][2:7] == (actor, actor, project, "operation_cancel", "SUCCEEDED")
    assert not cancels[0][9]


def verify_originals(read, project, actor, actor2):
    out = []
    for key, action in KEYS:
        r = by_key(read, key)
        assert len(r) == 1, key
        row = r[0]
        assert row[4:7] == (project, action, "SUCCEEDED") and row[9], key
        assert row[7] == row[8] and row[7], key
        assert row[2] == row[3] == (actor2 if "actor-two" in key else actor), key
        out.append(row)
    assert len({r[0] for r in out}) == len(out)
    assert len({r[1] for r in out}) == len(out)
    assert len({r[9] for r in out}) == len(out)
    assert all(re.fullmatch(r"[0-9a-f]{32}", r[i]) for r in out for i in (0, 1))
    uuid, reused = out[0][7], out[-2][7]
    assert uuid != reused and all(r[7] == uuid for r in out[:-2]) and out[-1][7] == reused
    cross = by_key(read, "native-s2-same-key-two-independent-admin-actors-01")
    assert len(cross) == 2 and {r[2] for r in cross} == {actor, actor2}
    assert all(r[2] == r[3] and r[4:9] == (project, "box_exec", "SUCCEEDED", uuid, uuid) and r[9] for r in cross)
    assert len({r[0] for r in cross}) == len({r[1] for r in cross}) == len({r[9] for r in cross}) == 2
    expected = sorted([(uuid, uuid, "DELETED", out[-3][0]), (reused, reused, "DELETED", out[-1][0])])
    assert read("SELECT uuid,incarnation,state,deleted_by_operation_id FROM owned_boxes WHERE project=? AND name=? ORDER BY uuid",
                (project, MAIN)) == expected
    for u in (uuid, reused):
        assert read("SELECT state FROM network_bindings WHERE project=? AND box_uuid=? AND nic='eth0'", (project, u)) == [("RELEASED",)]
    for r in out + cross:
        if r[5] == "box_exec":
            assert read("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (r[0],)) == [(0,)]
    return out + cross


def verify_public_wan(read, project, actor, uuid, log):
    prefix="native-s2-wan-"
    client=prefix+"curl-available-01"
    online=[prefix+f"online-before-{i:02d}" for i in range(1,5)]
    offline=prefix+"offline-01"
    restored=prefix+"online-after-01"
    keys=[client,*online,offline,restored]
    records={}
    for line in log.splitlines():
        if not line.startswith("S2_NATIVE_PUBLIC_IPV4_ORIGINAL "):
            continue
        _,key,oid,native,tracking,box,exit_text,status=line.split()
        assert key in keys and key not in records
        code=int(exit_text)
        assert re.fullmatch(r"[0-9]{3}|CLIENT|UNREADABLE",status)
        row=by_key(read,key)
        assert len(row)==1
        assert row[0][:10]==(oid,tracking,actor,actor,project,"box_exec","SUCCEEDED",uuid,uuid,native)
        assert box==uuid and native and tracking
        assert read("SELECT exit_code FROM exec_outputs WHERE operation_id=?",(oid,))==[(code,)]
        records[key]=(code,status)
    assert set(records)=={k for k in keys if by_key(read,k)}
    assert client in records and records[client][1]=="CLIENT"
    def positive(key):
        code,status=records[key]
        return code==0 and bool(re.fullmatch(r"2[0-9]{2}",status))
    tested=[k for k in online if k in records]
    assert tested==online[:len(tested)]
    if records[client][0]!=0:
        assert not tested and "S2_NATIVE_EXTERNAL_IPV4_WAN_CURL=UNVERIFIED_MISSING_CLIENT" in log.splitlines()
        success=False
    else:
        assert tested
        success=positive(tested[-1])
        assert not any(positive(k) for k in tested[:-1])
        assert success or len(tested)==len(online)
    if success:
        assert records.get(offline, (0,""))[0]!=0 and records[offline][1]=="000"
        assert restored in records and positive(restored)
        assert "S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS" in log.splitlines()
        assert "P1B_SCOPED_PUBLIC_IPV4_TLS=UNVERIFIED" not in log.splitlines()
        return "PASS"
    assert offline not in records and restored not in records
    assert "S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS" not in log.splitlines()
    assert "P1B_SCOPED_PUBLIC_IPV4_TLS=UNVERIFIED" in log.splitlines()
    return "UNVERIFIED"


def main(project, candidate_sha, phase, *args):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    assert project == "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    binary = Path("/usr/local/bin/boxweave")
    assert hashlib.sha256(binary.read_bytes()).hexdigest() == candidate_sha
    ids = [pid(u) for u in UNITS]
    assert os.path.samefile(f"/proc/{ids[0]}/exe", binary)
    guard = guard_identity()
    if phase == "before":
        assert not args and not native_names()
        assert acls() == {"bw-peer-quarantine"}
        assert rows("SELECT COUNT(*) FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
        assert not rows("SELECT id FROM operations WHERE work_ref IN ('lab/history-120','lab/queued-cancel','lab/foreign-layer','lab/s2')")
        for key, _ in KEYS:
            assert not by_key(rows, key)
        print("|".join(map(str, ids)) + "|" + hashlib.sha256(guard.encode()).hexdigest())
        return
    if phase == "failure":
        assert not args
        for r in rows("SELECT id,tracking_id,action,state,native_operation_id,error_class FROM operations WHERE state='UNKNOWN' ORDER BY id"):
            print("P1B_SCOPED_S2_UNKNOWN", *r)
        return
    if phase == "wan":
        assert len(args)==2
        actor,logpath=args
        original=by_key(rows,KEYS[0][0])
        assert len(original)==1
        result=verify_public_wan(rows,project,actor,original[0][7],Path(logpath).read_text())
        print("P1B_SCOPED_PUBLIC_IPV4_TLS_INDEPENDENT="+result)
        print("P1B_SCOPED_PUBLIC_IPV4_TLS_ORIGINAL_EVIDENCE=PASS")
        return
    if phase == "foreign":
        assert len(args) == 2 and native_names() == [MAIN]
        receipt, uuid = args
        assert re.fullmatch(r"[0-9a-f-]{36}", uuid)
        assert not rows("SELECT uuid FROM owned_boxes WHERE uuid=?", (uuid,))
        before = rows("SELECT id,state,native_operation_id FROM operations ORDER BY id")
        with LoopbackMCP(token=token(receipt), allow_mutations=False, timeout=60) as c:
            try:
                c.call("box_get", {"box_id": uuid})
            except MCPToolError:
                pass
            else:
                raise AssertionError("foreign native UUID exposed through MCP")
            assert not c.call("box_list", {}).structured["boxes"]
        assert rows("SELECT id,state,native_operation_id FROM operations ORDER BY id") == before
        print("P1B_SCOPED_S2_FOREIGN_UUID_READ_DENIED=PASS")
        return
    if phase == "spool":
        assert len(args) == 2
        actor, native_id = args
        r = by_key(rows, "native-s2-dual-stream-over-16m-001")
        assert len(r) == 1 and r[0][2:7] == (actor, actor, project, "box_exec", "SUCCEEDED")
        assert r[0][7] == r[0][8] and r[0][7] and r[0][9] == native_id
        assert rows("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (r[0][0],)) == [(0,)]
        assert rows("SELECT uuid,incarnation,state FROM owned_boxes WHERE project=? AND name=? AND state='OWNED'",
                    (project, MAIN)) == [(r[0][7], r[0][7], "OWNED")]
        print("P1B_SCOPED_S2_SPOOL_ORIGINAL", r[0][0], native_id, r[0][7])
        return
    assert phase == "after" and len(args) == 5
    actor, actor2, queued, foreign, original_runtime = args
    assert original_runtime == "|".join(map(str, ids)) + "|" + hashlib.sha256(guard.encode()).hexdigest()
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')")
    assert rows("SELECT COUNT(*) FROM owned_boxes WHERE project=? AND state='OWNED'", (project,)) == [(0,)]
    assert acls() == {"bw-peer-quarantine"} and native_names() == [MAIN]
    verify_history(rows, project, actor, queued, foreign)
    result = verify_originals(rows, project, actor, actor2)
    for row in result:
        print("P1B_SCOPED_S2_ORIGINAL", row[5], row[0], row[1], row[9], row[7])
    print("P1B_SCOPED_S2_FIXED_LEDGER=PASS")


if __name__ == "__main__":
    main(*sys.argv[1:])
