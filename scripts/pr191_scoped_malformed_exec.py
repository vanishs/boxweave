#!/usr/bin/env python3
"""Real malformed terminal Exec on the exact installed Scoped ledger.

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
from pr178_n2_scoped_lifecycle import acls, binding, instance, incus
from pr191_scoped_ack_loss import (CTL, QUARANTINE, guard_identity, native_idle,
                                   native_names, snapshot, wait_poststate, work_snapshot)
from pr191_scoped_lifecycle_recovery import ROOT, rows
from pr191_scoped_update_transport import (BIN, DB, SOCKET, command, http_status,
                                          restricted, supervised, wait_port)

NAME = "bw-s2-p2-malformed-exec-110v"
WORK = "lab/p2-malformed-return"
KEY = "s2-p2-malformed-exec-original"
COMPETING = "s2-p2-malformed-native-competing-effect"
UNITS = ("incus.service", "boxweave-scoped-lifecycle.service")


def digest(s):
    return hashlib.sha256(s.encode()).hexdigest()


def check_original(read, project, actor, cp, accepted, malformed, duplicate):
    oid, native, tracking, u = (cp[k] for k in ("exec_id", "native_id", "exec_tracking", "box_id"))
    assert re.fullmatch(r"[0-9a-f-]{36}", native) and re.fullmatch(r"[0-9a-f-]{36}", u)
    assert re.fullmatch(r"[0-9a-f]{32}", oid) and re.fullmatch(r"[0-9a-f]{32}", tracking)
    assert accepted == "REAL_EXEC_202_NATIVE_ID="+native+"\n"
    assert malformed == "MALFORMED_FINAL_NATIVE_ID="+native+"\n" and not duplicate
    assert read("SELECT id,tracking_id,actor_id,token_id,layer,action,state,native_operation_id,"
                "box_uuid,incarnation,error_class FROM operations WHERE key_digest=?", (digest(KEY),)) == [
                    (oid,tracking,actor,actor,project,"box_exec","UNKNOWN",native,u,u,
                     "EXEC_TERMINAL_EVIDENCE_UNVERIFIED")]
    assert read("SELECT id FROM operations WHERE work_ref=?",(WORK,))==[(oid,)]
    assert read("SELECT layer,resource_key FROM operation_resource_claims WHERE operation_id=?",(oid,))==[
        (project,"box:"+NAME)]
    events=[r[0] for r in read("SELECT state FROM operation_events WHERE operation_id=? ORDER BY seq",(oid,))]
    assert events[0]=="ADMITTED" and events[-1]=="UNKNOWN" and events.count("VERIFYING")==1
    assert not any(s in ("SUCCEEDED","FAILED","CANCELLED") for s in events)
    assert not read("SELECT id FROM operations WHERE key_digest=?",(digest(COMPETING),))
    assert not read("SELECT operation_id FROM exec_outputs WHERE operation_id=?",(oid,))
    assert not read("SELECT operation_id FROM recovery_audit WHERE operation_id=?",(oid,))


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
    assert http_status(18650)==401 and http_status(18665) is None
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
    lab=parent/"p2-malformed-exec"
    lab.mkdir(mode=0o700)
    os.chown(lab,account.pw_uid,account.pw_gid)
    proxy=lab/"proxy.sock"
    env=("BW_S2_MALFORM_SCOPED=1","BW_S2_MALFORM_LAB="+str(lab),
         "BW_S2_MALFORM_PROJECT="+project,"BW_S2_MALFORM_TOKEN="+credential)
    def probe(mode):
        with (work/f"scoped-malform-{mode}.log").open("xb") as output:
            subprocess.run(restricted(*env,str(work/"s2-malformed-exec-probe"),mode),
                           stdout=output,stderr=subprocess.STDOUT,check=True,timeout=220)
        print((work/f"scoped-malform-{mode}.log").read_text(),end="",flush=True)

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
    for _ in range(45):
        ready=subprocess.run(["incus","exec",NAME,"--project",project,"--","/bin/true"],
                             capture_output=True,timeout=12)
        if ready.returncode==0:
            break
        time.sleep(1)
    else:
        raise AssertionError("positive real guest readiness not observed")
    print("S2_P2_GUEST_REAL_EXEC_READY_POSITIVE_CONTROL=PASS",flush=True)
    # The original root filesystem stability check is isolated in this exact
    # newly owned guest. It does not touch the later original effect witness.
    precheck="/root/bw-s2-p2-root-mount-precheck-110v"
    incus("exec",NAME,"--project",project,"--","/bin/sh","-c",
          "test ! -e "+precheck+" && printf 'root-fs-check-110v\\n' > "+precheck)
    assert incus("exec",NAME,"--project",project,"--","cat",precheck).strip()=="root-fs-check-110v"
    incus("exec",NAME,"--project",project,"--","rm",precheck)
    print("S2_P2_GUEST_ROOT_WITNESS_STABLE_ACROSS_SEPARATE_EXEC=PASS",flush=True)
    wait_poststate(lambda:native_idle(project))
    def native_proof():
        assert incus("exec",NAME,"--project",project,"--","cat",
                     "/root/bw-s2-malformed-exec-once-110v")=="native-once-110v\n"
        wait_poststate(lambda:native_idle(project))
        current=instance(project,NAME)
        assert current["config"]["volatile.uuid"]==u and current["status"]=="Running"
        assert current["expanded_devices"]["eth0"]==raw["expanded_devices"]["eth0"]
        assert binding(u)==pin and native_names()==[NAME] and acls()=={QUARANTINE,pin["acl"]}
        assert [pid(unit) for unit in UNITS]==pids and guard_identity()==guard

    command("systemctl","stop",CTL)
    assert command("systemctl","show",CTL,"-p","MainPID","--value")=="0"
    wait_port(18650,False)
    accepted=lab/"native-accepted.txt"
    malformed=lab/"native-malformed.txt"
    duplicate=lab/"native-duplicate.txt"
    with supervised(restricted("BW_S2_TEST_UID="+str(account.pw_uid),str(work/"s2-malformed-exec-proxy"),
                        "-listen",str(proxy),"-upstream",SOCKET,"-project",project,
                        "-name",NAME,"-lab",str(lab)),parent/"proxy.log") as pc:
        for _ in range(60):
            assert pc.poll() is None
            if proxy.is_socket():
                break
            time.sleep(.5)
        assert proxy.is_socket()
        with supervised(restricted(str(BIN),"serve","-db",DB,"-incus-socket",str(proxy),
                                   "-project",project,"-listen","127.0.0.1:18665",
                                   "-scoped-peer-fresh"),parent/"controller.log") as ctl:
            wait_port(18665,True,ctl)
            with LoopbackMCP("http://127.0.0.1:18665/mcp",token=credential,timeout=60) as c:
                assert c.host_status()["peer_mode"]=="SCOPED_V2"
            probe("malform")
            cp=json.loads((lab/"original.json").read_text())
            assert all(cp[k]==v for k,v in seed.items() if k not in ("exec_id","native_id"))
            check_original(rows,project,actor,cp,accepted.read_text(),malformed.read_text(),duplicate.exists())
            assert not os.path.lexists(ROOT/("op-"+cp["exec_id"]))
            native_proof()
            original=snapshot(cp["exec_id"])
            before_work=work_snapshot(WORK)
            evidence=(accepted.read_bytes(),malformed.read_bytes(),(lab/"original.json").read_bytes())
            print("S2_P2_REAL_GUEST_MUTATION_EXACTLY_ONCE_AFTER_MALFORMED_RESPONSE=PASS",flush=True)
        wait_port(18665,False)
    command("systemctl","start",CTL)
    wait_port(18650,True)
    assert pid(CTL)!=ctl_pid and Path("/etc/systemd/system/boxweave.service").read_bytes()==service_file
    probe("resume")
    check_original(rows,project,actor,cp,accepted.read_text(),malformed.read_text(),duplicate.exists())
    assert snapshot(cp["exec_id"])==original and work_snapshot(WORK)==before_work
    assert evidence==(accepted.read_bytes(),malformed.read_bytes(),(lab/"original.json").read_bytes())
    native_proof()
    after=rows("SELECT id,state,native_operation_id,tracking_id,updated_at FROM operations ORDER BY id")
    new={seed["create_id"],cp["exec_id"]}
    assert [r for r in after if r[0] not in new]==prior
    assert rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','OBSERVING','UNKNOWN')")==[(cp["exec_id"],)]
    print("P1B_SCOPED_MALFORMED_EXEC_ORIGINAL",cp["exec_id"],cp["native_id"],cp["exec_tracking"],u,flush=True)
    # No native Delete, no removal of UNKNOWN, no false cleanup PASS.
    print("P1B_SCOPED_MALFORMED_EXEC_FIXED_LEDGER_RESTART_RETAINED_UNKNOWN=PASS",flush=True)


if __name__=="__main__":
    main(*sys.argv[1:])
