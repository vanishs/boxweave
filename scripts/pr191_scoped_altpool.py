#!/usr/bin/env python3
"""Original S2 alternate-pool SDK on the exact installed Scoped ledger."""
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import subprocess
import sys
import time

from n2_installed_lifecycle import pid, token
from pr178_n2_scoped_lifecycle import acls, instance, binding
from pr191_scoped_ack_loss import guard_identity, native_names, native_idle
from pr191_scoped_lifecycle_recovery import rows
from pr191_scoped_update_transport import command, restricted, supervised, wait_poststate

NAME = "bw-s2-alternate-pool-110w"
POOL = "bw-s2-p2-alternate-pool"
BASE_POOL = "bw-s2-native-pool"
KEYS = {"create": "s2-altpool-omitted-size-original",
        "delete": "s2-altpool-original-delete"}
UNITS = ("boxweave.service", "boxweave-scoped-lifecycle.service", "incus.service")


def check_original(read, project, actor, cp, kind):
    u = cp["box_id"]
    oid, native, tracking = (cp[kind + "_" + k] for k in ("id", "native", "tracking"))
    assert re.fullmatch(r"[0-9a-f-]{36}", u) and re.fullmatch(r"[0-9a-f-]{36}", native)
    assert re.fullmatch(r"[0-9a-f]{32}", oid) and re.fullmatch(r"[0-9a-f]{32}", tracking)
    digest = hashlib.sha256(KEYS[kind].encode()).hexdigest()
    assert read("SELECT id,native_operation_id,tracking_id,layer,actor_id,token_id,action,state,box_uuid,incarnation "
                "FROM operations WHERE key_digest=?", (digest,)) == [
                    (oid,native,tracking,project,actor,actor,"box_"+kind,"SUCCEEDED",u,u)]
    return oid


def check_absence(read, project, actor, cp):
    u=cp["box_id"]
    create=check_original(read,project,actor,cp,"create")
    delete=check_original(read,project,actor,cp,"delete")
    assert create!=delete and cp["create_native"]!=cp["delete_native"]
    assert cp["create_tracking"]!=cp["delete_tracking"]
    assert read("SELECT id FROM operations WHERE box_uuid=? ORDER BY id",(u,))==[(i,) for i in sorted((create,delete))]
    assert read("SELECT uuid,incarnation,state,deleted_by_operation_id FROM owned_boxes WHERE project=? AND name=?",
                (project,NAME))==[(u,u,"DELETED",delete)]
    assert read("SELECT state FROM network_bindings WHERE project=? AND box_uuid=?",(project,u))==[("RELEASED",)]
    assert not read("SELECT id FROM operations WHERE key_digest=?",
                    (hashlib.sha256(b"s2-altpool-unapproved-selector").hexdigest(),))
    assert not read("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')")


def volume_inventory(project, pool):
    items=json.loads(command("incus","storage","volume","list",pool,"--project",project,"--format=json"))
    return sorted((v["type"],v["name"]) for v in items if v["type"]!="image")


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
    assert hashlib.sha256(Path("/usr/local/bin/boxweave").read_bytes()).hexdigest()==candidate_sha
    assert hashlib.sha256((work/"candidate-a").read_bytes()).hexdigest()==candidate_sha
    assert command("incus","profile","device","get","default","root","pool","--project",project)==POOL
    mounts=[work/"mounted",work/"p2-alt-block/mounted"]
    assert len({p.stat().st_dev for p in mounts+[Path("/")]})==3
    for pool,mount,subdir in zip((BASE_POOL,POOL),mounts,("native-pool","storage")):
        assert command("incus","storage","get",pool,"source")==str(mount/subdir)
        assert command("findmnt","-n","-o","FSTYPE","--mountpoint",str(mount))=="ext4"
        assert "prjquota" in command("findmnt","-n","-o","OPTIONS","--mountpoint",str(mount)).split(",")
    assert not native_names() and acls()=={"bw-peer-quarantine"}
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')")
    initial_volumes={p:volume_inventory(project,p) for p in (BASE_POOL,POOL)}
    assert not any(initial_volumes.values()),initial_volumes
    identities=[pid(u) for u in UNITS]
    guard=guard_identity()
    actor=next(line.split()[0][3:] for line in receipt.read_text().splitlines() if line.startswith("id="))
    assert re.fullmatch(r"[0-9a-f]{32}",actor)
    lab=work/"p2-alt-pool"
    lab.mkdir(mode=0o700)
    os.chown(lab,account.pw_uid,account.pw_gid)
    with supervised(restricted("BW_S2_ALT_SCOPED=1","BW_S2_ALT_LAB="+str(lab),
                               "BW_S2_ALT_PROJECT="+project,"BW_S2_ALT_TOKEN="+token(receipt),
                               str(work/"s2-altpool-probe")),work/"scoped-altpool.log") as child:
        for _ in range(200):
            assert child.poll() is None
            if (lab/"live.ready").is_file():
                break
            time.sleep(.25)
        else:
            raise AssertionError("original SDK live witness not reached")
        cp=json.loads((lab/"original.json").read_text())
        u=cp["box_id"]
        create=check_original(rows,project,actor,cp,"create")
        assert native_names()==[NAME]
        raw=instance(project,NAME)
        assert raw["config"]["volatile.uuid"]==u and raw["config"]["user.boxweave.create_operation"]==create
        root=[d for d in raw["expanded_devices"].values() if d.get("type")=="disk" and d.get("path")=="/"]
        assert len(root)==1 and root[0]["pool"]==POOL and root[0]["size"]=="10GiB"
        assert raw["status"]=="Stopped"
        pin=binding(u)
        assert pin["project"]==project and pin["incarnation"]==u and pin["state"]=="PINNED"
        assert raw["expanded_devices"]["eth0"]["security.acls"]==pin["acl"]
        assert acls()=={"bw-peer-quarantine",pin["acl"]}
        assert volume_inventory(project,POOL)==[("container",NAME)]
        assert volume_inventory(project,BASE_POOL)==initial_volumes[BASE_POOL]
        print("S2_P2_ALTERNATE_POOL_LIVE_EXPANDED_ROOT_MATCHES_UUID_POOL_QUOTA=PASS",flush=True)
        (lab/"native-verified.release").touch(exist_ok=False)
        assert child.wait(timeout=120)==0,"SDK deletion failed; preserve original"
    log=(work/"scoped-altpool.log").read_text().splitlines()
    for marker in ("S2_P2_ALTERNATE_SELECTED_PROFILE_POOL_DEFAULT_TEN_GIB=PASS",
                   "S2_P2_ALTERNATE_POOL_NO_EXPLICIT_SELECTION_NO_RESIDUAL_BOX=PASS"):
        assert marker in log
        print(marker,flush=True)
    final=json.loads((lab/"final.json").read_text())
    assert all(final[k]==v for k,v in cp.items())
    wait_poststate(lambda: native_idle(project))
    check_absence(rows,project,actor,final)
    assert not native_names() and acls()=={"bw-peer-quarantine"}
    assert {p:volume_inventory(project,p) for p in initial_volumes}==initial_volumes
    assert command("incus","profile","device","get","default","root","pool","--project",project)==POOL
    assert [pid(u) for u in UNITS]==identities and guard_identity()==guard
    print("P1B_SCOPED_ALTPOOL_ORIGINALS",u,create,cp["create_native"],cp["create_tracking"],
          final["delete_id"],final["delete_native"],final["delete_tracking"],flush=True)
    # The reviewed pool remains selected until disposable runner retirement.
    # Do not restore a different profile or unmount an installed live pool.
    print("P1B_SCOPED_ALTPOOL_FIXED_LEDGER_AND_NO_RESIDUAL_VOLUME=PASS",flush=True)


if __name__=="__main__":
    main(*sys.argv[1:])
