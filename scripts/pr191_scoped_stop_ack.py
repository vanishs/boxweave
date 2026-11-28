#!/usr/bin/env python3
"""Real lost Stop acknowledgement on the exact installed Scoped ledger.

Only the original test-only restricted transport is used while the normal
controller is stopped. Preserve the unresolved original and native guest.
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
from pr191_scoped_ack_loss import (CTL, QUARANTINE, guard_identity, native_idle,
                                   native_names, snapshot, wait_poststate, work_snapshot)
from pr191_scoped_lifecycle_recovery import ROOT, rows
from pr191_scoped_update_transport import (BIN, DB, SOCKET, command, http_status,
                                          restricted, supervised, wait_port)

NAME = "bw-s2-p1-stop-110p"
WORK = "lab/p1-native-ack-loss"
KEY = "s2-p1-stop-native-202-ack-loss-original"
COMPETING = "s2-p1-stop-new-key-must-not-dispatch"
UNITS = ("incus.service", "boxweave-scoped-lifecycle.service")


def digest(s):
    return hashlib.sha256(s.encode()).hexdigest()


def check_original(read, project, actor, cp, accepted, duplicate):
    oid, tracking, u = (cp[k] for k in ("action_id", "action_tracking", "box_id"))
    match = re.fullmatch(r"S2_P1_NATIVE_STOP_202_ACCEPTED_ID=([0-9a-f-]{36})\n", accepted)
    assert match and not duplicate
    native = match[1]
    assert native != cp["create_native"] and re.fullmatch(r"[0-9a-f-]{36}", u)
    assert re.fullmatch(r"[0-9a-f]{32}", oid) and re.fullmatch(r"[0-9a-f]{32}", tracking)
    assert read("SELECT id,tracking_id,actor_id,token_id,layer,action,state,native_operation_id,"
                "box_uuid,incarnation,error_class FROM operations WHERE key_digest=?", (digest(KEY),)) == [
                    (oid,tracking,actor,actor,project,"box_stop","UNKNOWN","",u,u,
                     "NATIVE_OUTCOME_AMBIGUOUS")]
    assert read("SELECT id FROM operations WHERE work_ref=?",(WORK,))==[(oid,)]
    assert read("SELECT layer,resource_key FROM operation_resource_claims WHERE operation_id=?",(oid,))==[
        (project,"box:"+NAME)]
    events=[r[0] for r in read("SELECT state FROM operation_events WHERE operation_id=? ORDER BY seq",(oid,))]
    assert events == ["ADMITTED","DISPATCHING","UNKNOWN"]
    assert not read("SELECT id FROM operations WHERE native_operation_id=? OR key_digest=?",(native,digest(COMPETING)))
    assert not read("SELECT operation_id FROM recovery_audit WHERE operation_id=?",(oid,))
    return native


def main(project, receipt, candidate_sha):
    assert os.geteuid()==0 and os.environ.get("GITHUB_ACTIONS")=="true"
    assert os.environ.get("RUNNER_ENVIRONMENT")=="github-hosted"
    account=pwd.getpwnam("boxweave")
    assert project=="user-"+str(account.pw_uid)
    receipt=Path(receipt)
    work=receipt.parent
    assert work.parent==Path("/tmp") and work.name.startswith("bw-n2-installed.")
    assert receipt.name=="actor.receipt" and not receipt.stat().st_mode&0o077
    assert command("git","rev-parse","HEAD")==os.environ["N2_EXACT_HEAD"]
    assert hashlib.sha256(BIN.read_bytes()).hexdigest()==candidate_sha
    assert hashlib.sha256((work/"candidate-a").read_bytes()).hexdigest()==candidate_sha
    assert not native_names() and acls()=={QUARANTINE}
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','OBSERVING','UNKNOWN')")
    assert not rows("SELECT id FROM operations WHERE work_ref=? OR key_digest IN (?,?)",
                    (WORK,digest(KEY),digest(COMPETING)))
    assert http_status(18650)==401 and http_status(18658) is None
    ctl_pid=pid(CTL)
    assert os.path.samefile(f"/proc/{ctl_pid}/exe", BIN)
    pids=[pid(unit) for unit in UNITS]
    guard=guard_identity()
    service_file=Path("/etc/systemd/system/boxweave.service").read_bytes()
    prior=rows("SELECT id,state,native_operation_id,tracking_id,updated_at FROM operations ORDER BY id")
    actor=next(line.split()[0][3:] for line in receipt.read_text().splitlines() if line.startswith("id="))
    assert re.fullmatch(r"[0-9a-f]{32}",actor)
    credential=token(receipt)
    with LoopbackMCP(token=credential,timeout=60) as c:
        assert c.host_status()["peer_mode"]=="SCOPED_V2"
    parent=Path(tempfile.mkdtemp(prefix="bw-v1-s2-ghci.",dir="/tmp"))
    parent.chmod(0o711)
    lab=parent/"p1-stop"
    lab.mkdir(mode=0o700)
    os.chown(lab,account.pw_uid,account.pw_gid)
    proxy=lab/"proxy.sock"
    env=("BW_S2_P1_SCOPED=1","BW_S2_P1_DIR="+str(lab),
         "BW_S2_P1_PROJECT="+project,"BW_S2_P1_TOKEN="+credential)
    def probe(mode):
        with (work/f"scoped-stop-{mode}.log").open("xb") as output:
            subprocess.run(restricted(*env,str(work/"s2-stop-delete-probe"),mode),
                           stdout=output,stderr=subprocess.STDOUT,check=True,timeout=220)
        print((work/f"scoped-stop-{mode}.log").read_text(),end="",flush=True)

    probe("seed")
    seed=json.loads((lab/"original.json").read_text())
    u=seed["box_id"]
    assert rows("SELECT id,native_operation_id,tracking_id,actor_id,layer,action,state,box_uuid,incarnation "
                "FROM operations WHERE id=?",(seed["create_id"],))==[
        (seed["create_id"],seed["create_native"],seed["create_tracking"],actor,project,"box_create","SUCCEEDED",u,u)]
    assert native_names()==[NAME]
    raw=instance(project,NAME)
    pin=binding(u)
    assert raw["config"]["volatile.uuid"]==u and raw["status"]=="Running"
    assert raw["config"]["user.boxweave.create_operation"]==seed["create_id"]
    assert pin["state"]=="PINNED" and raw["expanded_devices"]["eth0"]["security.acls"]==pin["acl"]
    assert acls()=={QUARANTINE,pin["acl"]}
    wait_poststate(lambda:native_idle(project))

    def native_proof():
        wait_poststate(lambda:instance(project,NAME)["status"]=="Stopped" and native_idle(project))
        current=instance(project,NAME)
        assert current["config"]["volatile.uuid"]==u
        assert current["config"]["user.boxweave.create_operation"]==seed["create_id"]
        assert current["config"].get("boot.autostart")==raw["config"].get("boot.autostart")
        assert current["expanded_devices"]["eth0"]==raw["expanded_devices"]["eth0"]
        assert rows("SELECT uuid,incarnation,state FROM owned_boxes WHERE uuid=?", (u,))==[(u,u,"OWNED")]
        assert binding(u)==pin and native_names()==[NAME] and acls()=={QUARANTINE,pin["acl"]}
        assert [pid(unit) for unit in UNITS]==pids and guard_identity()==guard

    command("systemctl","stop",CTL)
    assert command("systemctl","show",CTL,"-p","MainPID","--value")=="0"
    wait_port(18650,False)
    accepted=lab/"native-accepted.txt"
    duplicate=lab/"native-accepted.txt.unexpected"
    with supervised(restricted(str(work/"s2-stop-delete-proxy"),
                        "-listen",str(proxy),"-upstream",SOCKET,"-project",project,
                        "-phase","stop","-marker",str(accepted)),parent/"proxy.log") as pc:
        for _ in range(60):
            assert pc.poll() is None
            if proxy.is_socket():
                break
            time.sleep(.5)
        assert proxy.is_socket()
        with supervised(restricted(str(BIN),"serve","-db",DB,"-incus-socket",str(proxy),
                                   "-project",project,"-listen","127.0.0.1:18658",
                                   "-scoped-peer-fresh"),parent/"controller.log") as ctl:
            wait_port(18658,True,ctl)
            with LoopbackMCP("http://127.0.0.1:18658/mcp",token=credential,timeout=60) as c:
                assert c.host_status()["peer_mode"]=="SCOPED_V2"
            probe("mutate")
            probe("poll")
            cp=json.loads((lab/"original.json").read_text())
            assert all(cp[k]==v for k,v in seed.items() if k not in ("action_id",))
            native=check_original(rows,project,actor,cp,accepted.read_text(),duplicate.exists())
            assert not os.path.lexists(ROOT/("op-"+cp["action_id"]))
            native_proof()
            original=snapshot(cp["action_id"])
            before_work=work_snapshot(WORK)
            evidence=(accepted.read_bytes(),(lab/"original.json").read_bytes())
            print("P1B_SCOPED_STOP_202_NATIVE_STOPPED_EXACT_UUID=PASS",flush=True)
        wait_port(18658,False)
    command("systemctl","start",CTL)
    wait_port(18650,True)
    assert pid(CTL)!=ctl_pid and Path("/etc/systemd/system/boxweave.service").read_bytes()==service_file
    probe("resume")
    native=check_original(rows,project,actor,cp,accepted.read_text(),duplicate.exists())
    assert snapshot(cp["action_id"])==original and work_snapshot(WORK)==before_work
    assert evidence==(accepted.read_bytes(),(lab/"original.json").read_bytes())
    native_proof()
    after=rows("SELECT id,state,native_operation_id,tracking_id,updated_at FROM operations ORDER BY id")
    new={seed["create_id"],cp["action_id"]}
    assert [r for r in after if r[0] not in new]==prior
    assert rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','OBSERVING','UNKNOWN')")==[(cp["action_id"],)]
    print("P1B_SCOPED_STOP_ACK_ORIGINAL",cp["action_id"],native,cp["action_tracking"],u,flush=True)
    # No native Delete, no removal of UNKNOWN, no false cleanup PASS.
    print("P1B_SCOPED_STOP_ACK_FIXED_LEDGER_RESTART_RETAINED_UNKNOWN=PASS",flush=True)


if __name__=="__main__":
    main(*sys.argv[1:])
