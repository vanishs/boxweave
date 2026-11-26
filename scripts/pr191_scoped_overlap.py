#!/usr/bin/env python3
"""One-use installed Scoped pre-atomic Create/Delete overlap witness.

Reuse the original source overlay and official SDK. Only the temporary
controller has a pause; the installed binary, restricted UID, fixed ledger,
Root helper and guard are unchanged. No shared-host execution.
"""
import hashlib
import json
import os
from pathlib import Path
import pwd
import subprocess
import sys
import time

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import pid, token
from pr178_n2_scoped_lifecycle import acls, binding, instance
from pr191_scoped_ack_loss import (CTL, QUARANTINE, guard_identity, native_idle,
                                   native_names, snapshot, wait_poststate)
from pr191_scoped_lifecycle_recovery import journal, rows
from pr191_scoped_update_transport import (BIN, DB, SOCKET, command, http_status,
                                          restricted, supervised, wait_port)

NAME = "bw-s2-p1-overlap-110x"
UNITS = ("incus.service", "boxweave-scoped-lifecycle.service")


def history():
    return rows("SELECT id,state,native_operation_id,tracking_id,updated_at FROM operations ORDER BY id")


def held_proof(read, project, actor, oid, native, uuid, pin):
    assert read("SELECT state,box_uuid,incarnation,native_operation_id,actor_id,layer FROM operations WHERE id=?", (oid,)) == [
        ("VERIFYING", "", "", native, actor, project)]
    for table in ("owned_boxes", "network_bindings", "network_address_fences"):
        column = "uuid" if table == "owned_boxes" else "box_uuid"
        assert not read("SELECT "+column+" FROM "+table+" WHERE "+column+"=?", (uuid,))
    prefix = "create-pin:"+pin["Bridge"]+":"
    claims = {"box:"+NAME, prefix+"v4:"+pin["IPv4"], prefix+"v6:"+pin["IPv6"],
              prefix+"mac:"+pin["MAC"], prefix+"acl:"+QUARANTINE}
    assert set(read("SELECT layer,resource_key FROM operation_resource_claims WHERE operation_id=?", (oid,))) == {
        (project, k) for k in claims}
    assert not read("SELECT id FROM operations WHERE box_uuid=? OR incarnation=?", (uuid,uuid))
    assert not read("SELECT state FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", (oid,))


def final_proof(read, project, actor, oid, native, tracking, uuid, acl, delete=None):
    assert read("SELECT state,box_uuid,incarnation,native_operation_id,tracking_id,actor_id,layer FROM operations WHERE id=?", (oid,)) == [
        ("SUCCEEDED", uuid, uuid, native, tracking, actor, project)]
    assert read("SELECT state,safe_detail FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", (oid,)) == [
        ("SUCCEEDED", "ORIGINAL_CREATE_UUID_ATOMIC")]
    assert read("SELECT project,name,incarnation,state FROM owned_boxes WHERE uuid=?", (uuid,)) == [
        (project, NAME, uuid, "DELETED" if delete else "OWNED")]
    assert read("SELECT state,acl FROM network_bindings WHERE box_uuid=?", (uuid,)) == [
        ("RELEASED" if delete else "PINNED", acl)]
    assert read("SELECT state,operation_id FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == [
        ("RELEASED" if delete else "RESERVED", delete or oid)]*2
    if delete:
        assert read("SELECT deleted_by_operation_id FROM owned_boxes WHERE uuid=?", (uuid,)) == [(delete,)]
        assert read("SELECT state,safe_detail FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", (delete,)) == [
            ("SUCCEEDED", "SCOPED_DELETE_ACL_ABSENT_ATOMIC")]


def marker(child, path):
    end = time.monotonic()+85
    while time.monotonic()<end:
        assert child.poll() is None, "SDK exited before "+path.name
        if path.is_file():
            try:
                return json.loads(path.read_text())
            except json.JSONDecodeError:
                # The SDK exclusively creates, then writes and syncs the marker.
                pass
        time.sleep(.2)
    raise AssertionError("missing SDK witness "+path.name)


def release(path):
    with path.open("x") as f:
        f.write("independently verified\n")


def main(project, receipt, candidate_sha, lab, overlay_sha):
    assert os.geteuid()==0 and os.environ.get("GITHUB_ACTIONS")=="true"
    assert os.environ.get("RUNNER_ENVIRONMENT")=="github-hosted"
    account=pwd.getpwnam("boxweave")
    assert project=="user-"+str(account.pw_uid)
    receipt=Path(receipt)
    work=receipt.parent
    lab=Path(lab)
    overlay=work/"s2-scoped-overlap-service"
    assert work.parent==Path("/tmp") and work.name.startswith("bw-n2-installed.")
    assert lab.parent.parent==Path("/tmp") and lab.parent.name.startswith("bw-v1-s2-ghci.")
    assert lab.name=="p1-native-overlap" and not lab.is_symlink()
    assert receipt.name=="actor.receipt" and not receipt.stat().st_mode&0o077
    assert command("git","rev-parse","HEAD")==os.environ["N2_EXACT_HEAD"]
    assert hashlib.sha256(BIN.read_bytes()).hexdigest()==candidate_sha
    assert hashlib.sha256((work/"candidate-a").read_bytes()).hexdigest()==candidate_sha
    assert hashlib.sha256(overlay.read_bytes()).hexdigest()==overlay_sha!=candidate_sha
    assert not native_names() and acls()=={QUARANTINE}
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','OBSERVING','UNKNOWN')")
    assert http_status(18650)==401 and http_status(18668) is None
    before=history()
    control=pid(CTL)
    assert os.path.samefile(f"/proc/{control}/exe",BIN)
    service=Path("/etc/systemd/system/boxweave.service").read_bytes()
    pids=[pid(u) for u in UNITS]
    guard=guard_identity()
    actor=next(line.split()[0][3:] for line in receipt.read_text().splitlines() if line.startswith("id="))
    credential=token(receipt)
    second_receipt=work/"overlap-second.receipt"
    with second_receipt.open("x") as f:
        os.fchmod(f.fileno(),0o600)
        subprocess.run(restricted(str(BIN),"token-create","-project",project,"-ttl","15m"),
                       stdout=f,check=True,timeout=30)
    other=token(second_receipt)
    assert other!=credential
    with LoopbackMCP(token=credential,timeout=60) as c:
        assert c.host_status()["peer_mode"]=="SCOPED_V2"
    os.chown(lab,account.pw_uid,account.pw_gid)
    env=("BW_S2_OVERLAP_SCOPED=1","BW_S2_OVERLAP_ACTIVATE=one-disposable-original",
         "BW_S2_OVERLAP_LAB="+str(lab),"BW_S2_TEST_UID="+str(account.pw_uid))
    command("systemctl","stop",CTL)
    assert command("systemctl","show",CTL,"-p","MainPID","--value")=="0"
    wait_port(18650,False)
    with supervised(restricted(*env,str(overlay),"serve","-db",DB,"-incus-socket",SOCKET,
                    "-project",project,"-listen","127.0.0.1:18668","-scoped-peer-fresh"),
                    work/"overlap-controller.log") as ctl:
        wait_port(18668,True,ctl)
        with LoopbackMCP("http://127.0.0.1:18668/mcp",token=credential,timeout=60) as c:
            assert c.host_status()["peer_mode"]=="SCOPED_V2"
        print("S2_P1_NATIVE_OVERLAY_TEST_ONLY_SERVICE_MCP_AUTH_READY=PASS",flush=True)
        with supervised(restricted(*env,"BW_S2_OVERLAP_PROJECT="+project,
                        "BW_S2_OVERLAP_TOKEN="+credential,"BW_S2_OVERLAP_SECOND_TOKEN="+other,
                        str(work/"s2-scoped-overlap-sdk")),work/"overlap-sdk.log") as sdk:
            cp=marker(sdk,lab/"bound.json")
            held=json.loads((lab/"held").read_text())
            assert held==dict(cp,phase="NATIVE_VERIFIED_BEFORE_ATOMIC_CREATE",project=project)
            oid,native,uuid=(cp[k] for k in ("original_id","native_id","uuid"))
            rec,digests=journal(oid,"CREATE",native)
            assert rec["UUID"]==uuid and rec["Project"]==project and rec["Name"]==NAME
            pin=rec["Binding"]
            acl=pin["ACL"]
            raw=instance(project,NAME)
            assert raw["config"]["volatile.uuid"]==uuid and raw["config"]["user.boxweave.create_operation"]==oid
            assert raw["status"]=="Stopped" and raw["expanded_devices"]["eth0"]["security.acls"]==acl
            assert native_names()==[NAME] and acls()=={QUARANTINE,acl}
            held_proof(rows,project,actor,oid,native,uuid,pin)
            saved=snapshot(oid)
            tracking=saved[0][0][1]
            assert [r for r in history() if r[0]!=oid]==before
            print("S4_A_NATIVE_UUID_PREATOMIC_INDEPENDENT_INCUS_CONFIRM=PASS",flush=True)
            release(lab/"native-verified")
            assert marker(sdk,lab/"denied")==cp
            assert snapshot(oid)==saved and [r for r in history() if r[0]!=oid]==before
            assert journal(oid,"CREATE",native)==(rec,digests)
            assert instance(project,NAME)==raw and acls()=={QUARANTINE,acl}
            wait_poststate(lambda:native_idle(project))
            held_proof(rows,project,actor,oid,native,uuid,pin)
            release(lab/"denials-verified")
            assert marker(sdk,lab/"created")==cp
            final_proof(rows,project,actor,oid,native,tracking,uuid,acl)
            assert instance(project,NAME)["config"]["volatile.uuid"]==uuid and binding(uuid)["acl"]==acl
            release(lab/"create-verified")
            assert sdk.wait(timeout=150)==0
        result=json.loads((lab/"completed.json").read_text())
        did,dnative,dtracking=(result[k] for k in ("delete_id","delete_native","delete_tracking"))
        assert result["create_id"]==oid and result["create_native"]==native and result["create_tracking"]==tracking and result["uuid"]==uuid
        assert did!=oid and dnative!=native and dtracking!=tracking
        assert rows("SELECT state,box_uuid,incarnation,native_operation_id,tracking_id,actor_id,layer FROM operations WHERE id=?",(did,))==[
            ("SUCCEEDED",uuid,uuid,dnative,dtracking,actor,project)]
        drec,_=journal(did,"DELETE",dnative)
        assert drec["UUID"]==uuid and drec["Binding"]["ACL"]==acl
        final_proof(rows,project,actor,oid,native,tracking,uuid,acl,did)
        assert [r for r in history() if r[0] not in (oid,did)]==before
        assert not native_names() and acls()=={QUARANTINE}
        wait_poststate(lambda:native_idle(project))
    wait_port(18668,False)
    command("systemctl","start",CTL)
    wait_port(18650,True)
    assert pid(CTL)!=control and Path("/etc/systemd/system/boxweave.service").read_bytes()==service
    assert [pid(u) for u in UNITS]==pids and guard_identity()==guard
    assert hashlib.sha256(BIN.read_bytes()).hexdigest()==candidate_sha
    final_proof(rows,project,actor,oid,native,tracking,uuid,acl,did)
    assert not native_names() and acls()=={QUARANTINE}
    print((work/"overlap-sdk.log").read_text(),end="",flush=True)
    print("P1B_SCOPED_PREATOMIC_OVERLAP_ORIGINALS",oid,native,tracking,did,dnative,dtracking,uuid,flush=True)
    print("S4_A_REAL_NATIVE_PREATOMIC_CONCURRENT_DELETE_FULL_LIFECYCLE=PASS",flush=True)
    print("P1B_SCOPED_PREATOMIC_OVERLAP_FIXED_LEDGER=PASS",flush=True)


if __name__=="__main__":
    main(*sys.argv[1:])
