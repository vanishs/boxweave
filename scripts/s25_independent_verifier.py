#!/usr/bin/env python3
"""Independent read-only S2.5 V-tier verifier, not imported by the Agent.

Uses only Incus CLI readbacks, local immutable SQLite receipts, installed
binary digests and guest-private nested state. It makes NO MCP mutation and
cannot obtain Agent tokens. Run on one-use CI host only.
"""
import hashlib
import json
import os
import sqlite3
import subprocess
import sys
import time
from pathlib import Path

def require(condition, reason):
    if not condition:
        raise SystemExit("S25_V_FAIL: " + reason)

def cli(*args):
    result = subprocess.run(args, text=True, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, timeout=35, check=False)
    if result.returncode:
        raise RuntimeError("native read-only CLI failed (%d): %s" %
                           (result.returncode, result.stderr[:300]))
    return result.stdout.strip()

def native(*args):
    return cli("sudo", "incus", *args)

def as_json(text):
    return json.loads(text)

def inner(*args):
    return native("exec", "bw-s25-mcp-l1", "--project", os.environ["BW_S25_PROJECT"],
                  "--", *args)

def host_read(kind, project, name, uuid):
    obj = as_json(native("list", "--project", project, "--format", "json"))
    entries = [x for x in obj if x.get("name") == name]
    if kind == "present":
        require(len(entries) == 1, "exact native Box absent/duplicated: " + name)
        ins = entries[0]
        require(ins.get("status") in ("Running", "Ready"), "native Box not running")
        require(ins.get("config", {}).get("volatile.uuid") == uuid,
                "native UUID differs from MCP accepted original")
        return ins
    require(len(entries) == 0, "deleted Box still exists: " + name)
    return None

def read_ledger(db, operation_id, uuid, expected_agent):
    # The separate verifier has host-local administrative read privilege;
    # the authenticated MCP Agent never accesses native SQLite directly.
    code = (
        "import sqlite3,sys;"
        "c=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True,timeout=3);"
        "v=c.execute('SELECT state,box_uuid,agent_ref FROM operations WHERE id=?',"
        "(sys.argv[2],)).fetchall();"
        "assert v==[('SUCCEEDED',sys.argv[3],sys.argv[4])],v"
    )
    cli("sudo","python3","-c",code,str(db),operation_id,uuid,expected_agent)


def sha256(data):
    return hashlib.sha256(data).hexdigest()

def inspect_l1():
    manifest = as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    project = os.environ["BW_S25_PROJECT"]
    name = manifest["name"]
    uuid = manifest.get("uuid")
    require(uuid and len(uuid) >= 32 and name == "bw-s25-mcp-l1",
            "original parent checkpoint missing or unrecognized")
    inst = host_read("present", project, name, uuid)
    require(inst.get("expanded_config", {}).get("security.nesting") == "true",
            "L0's MCP-created L1 not actually nesting-enabled")
    devices = inst.get("expanded_devices", {})
    nic = devices.get("eth0", {})
    require(nic.get("security.acls.default.ingress.action") == "reject" and
            nic.get("security.acls.default.egress.action") == "allow" and
            nic.get("security.acls") and
            nic.get("security.ipv4_filtering") == "true" and
            nic.get("security.ipv6_filtering") == "true" and
            nic.get("security.mac_filtering") == "true",
            "L1 effective NIC lost default deny or antispof")
    read_ledger(os.environ["BW_S25_OUTER_DB"], manifest["create_op"],
                uuid, "agent-outer-alpha")
    print("S25_V_L0_MCP_CREATED_L1_NATIVE_UUID_SQLITE_ACL=PASS")
    return manifest

def inspect_l2():
    inspect_l1()
    data = as_json(inner("cat", "/root/s25-inner-checkpoint.json"))
    require(data.get("name") == "bw-s25-mcp-l2" and data.get("uuid"),
            "inner independent Agent checkpoint missing L2")
    innerproj = inner("sh", "-c", "printf user-; id -u boxweave")
    child_list = as_json(inner("incus", "list", "--project",
                               innerproj, "--format", "json"))
    instances = [x for x in child_list if x.get("name") == data["name"]]
    require(len(instances) == 1, "actual nested Incus L2 absent")
    inst = instances[0]
    require(inst.get("config", {}).get("volatile.uuid") == data["uuid"],
            "L1 SDK MCP L2 UUID not same native instance")
    require(inst.get("status") in ("Running", "Ready"), "actual L2 not running")
    guest_sha = inner("sha256sum", "/usr/local/bin/boxweave").split()[0]
    require(guest_sha == os.environ["BW_S25_BINARY_SHA256"],
            "nested running v1 executable differs from L0 exact SHA")
    require(inner("systemctl", "is-active", "boxweave.service") == "active",
            "nested installed v1 service not active")
    # A read-only SQL query inside the nested guest, no token material.
    code = ("import sqlite3,sys;"
            "c=sqlite3.connect('file:/var/lib/boxweave/ledger.sqlite?mode=ro',uri=True);"
            "v=c.execute('SELECT state,box_uuid FROM operations WHERE id=?',(sys.argv[1],)).fetchone();"
            "assert v==('SUCCEEDED',sys.argv[2]),v")
    inner("python3", "-c", code, data["create_op"], data["uuid"])
    # All lifecycle Operations are independently reconciled with the same
    # original native UUID and the actor that actually submitted them.
    for key in ("update_op","restart_stop_op","restart_start_op"):
        require(data.get(key), "inner MCP L2 lifecycle original missing: "+key)
        inner("python3","-c",code,data[key],data["uuid"])
    # The real MCP-created L2 (internet=true) must have an explicit
    # per-instance egress permit, but still default-deny incoming and
    # enforce anti-spoof. The safe default Profile retains implicit reject.
    nic=inst.get("expanded_devices",{}).get("eth0",{})
    require(nic.get("security.acls.default.ingress.action")=="reject" and
            nic.get("security.acls.default.egress.action")=="allow" and
            nic.get("security.ipv4_filtering")=="true" and
            nic.get("security.ipv6_filtering")=="true" and
            nic.get("security.mac_filtering")=="true" and
            nic.get("security.acls"),
            "MCP requested Internet but native effective NIC unsafe")
    uid=innerproj.split("-")[-1]
    gw4=inner("incus","network","get","incusbr-"+uid,"ipv4.address").split("/")[0]
    gw6=inner("incus","network","get","incusbr-"+uid,"ipv6.address").split("/")[0]
    require(gw4 and gw6, "inner native bridge address unavailable")
    # Native Incus Running does not imply Ubuntu has finished DHCPv4
    # or IPv6 RA/route convergence after an actual Guest restart.
    # Witness the positive packets with an explicit bounded deadline.
    # Never silently downgrade an absent IPv6 route to a PASS.
    for family, gateway in (("-4", gw4), ("-6", gw6)):
        deadline=time.monotonic()+70
        last_error=""
        while True:
            try:
                inner("incus","exec","bw-s25-mcp-l2","--project",innerproj,
                      "--","ping",family,"-c","1","-W","2",gateway)
                break
            except (RuntimeError, subprocess.TimeoutExpired) as exc:
                last_error=str(exc)[:300]
                if time.monotonic()>=deadline:
                    # Safe, non-mutating guest network facts for the red
                    # artifact, without confusing L1 bridge with L2 route.
                    try:
                        routes=inner("incus","exec","bw-s25-mcp-l2",
                                     "--project",innerproj,
                                     "--","ip",family,"route","show")
                    except Exception as route_exc:
                        routes=str(route_exc)[:200]
                    raise RuntimeError("S25_V_NATIVE_GATEWAY_UNREADY "+
                                       family+" "+gateway+"; route="+routes[:400]+
                                       "; last="+last_error)
                time.sleep(2)
        print("S25_V_L2_NATIVE_GATEWAY_"+("IPV4" if family=="-4" else "IPV6")+"=PASS")
    print("S25_V_MCP_L2_NATIVE_IPV4_IPV6_GATEWAY_EGRESS=PASS")
    # This gateway witness does NOT prove direct upstream WAN on a GitHub
    # runner. The native WAN field remains explicitly BLOCKED separately.
    require(inst.get("config",{}).get("limits.cpu")=="2",
            "same native L2 UUID did not retain revised CPU limit")
    print("S25_V_L1_MCP_L2_UPDATE_STOP_START_SAME_NATIVE_UUID=PASS")
    print("S25_V_L1_MCP_CREATED_L2_NATIVE_UUID_SQLITE_BINARY=PASS")


def inspect_simwan6(inner_layer=False, cleaned=False):
    """Read-only V-tier I/N crosswalk for real routed HTTPS, not mock/public WAN.

    The simulator runs in a different Linux network namespace from the owned
    Incus bridge, and this verifier never has the Agent's MCP tokens.
    """
    if inner_layer:
        data = as_json(inner("cat", "/root/s25-inner-checkpoint.json"))
        project = inner("sh", "-c", "printf user-; id -u boxweave")
        name = "bw-s25-mcp-l2"
        instances = as_json(inner("incus", "list", "--project", project, "--format", "json"))
        cert_path = "/root/s25simwan6/simwan6.crt"
        log = inner("cat", "/root/s25simwan6/server.log")
        namespace = inner("ip", "netns", "list")
    else:
        data = as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
        project = os.environ["BW_S25_PROJECT"]
        name = "bw-s25-mcp-l1"
        instances = as_json(native("list", "--project", project, "--format", "json"))
        cert_path = os.environ["BW_S25_SIMWAN6_CERT_FILE"]
        log = cli("sudo", "cat", str(Path(os.environ["BW_S25_SIMWAN6_WORKDIR"], "server.log")))
        namespace = cli("sudo", "ip", "netns", "list")
    require("s25simwan6" in namespace, "separate routed upstream namespace absent")
    require("SIM_WAN6_REMOTE=" in log, "upstream HTTPS accepted no packets")
    require(data.get("wan6_peer_uuid") and data.get("wan6_peer_create_op"),
            "original SDK-created WAN sibling missing")
    require(data.get("uuid") and data.get("name") == name, "original primary Box missing")
    by_name = {x["name"]: x for x in instances}
    peer_name = "bw-s25-wan6-peer"
    if cleaned:
        if inner_layer:
            require(name not in by_name, "nested original L2 still exists after cleanup")
        else:
            require(name in by_name, "outer L1 missing before original cleanup")
            require(by_name[name].get("config", {}).get("volatile.uuid") == data["uuid"],
                    "outer L1 changed UUID during simulated-WAN peer cleanup")
        require(peer_name not in by_name,
                "MCP-original WAN peer still exists after cleanup")
        for field in ("wan6_peer_stop_op", "wan6_peer_delete_op"):
            require(data.get(field), "missing cleanup original operation "+field)
            if inner_layer:
                sql_op = ("import sqlite3,sys;"
                          "c=sqlite3.connect('file:/var/lib/boxweave/ledger.sqlite?mode=ro',uri=True);"
                          "v=c.execute('SELECT state,box_uuid FROM operations WHERE id=?',(sys.argv[1],)).fetchone();"
                          "assert v==('SUCCEEDED',sys.argv[2]),v")
                inner("python3", "-c", sql_op, data[field], data["wan6_peer_uuid"])
            else:
                read_ledger(os.environ["BW_S25_OUTER_DB"], data[field],
                            data["wan6_peer_uuid"], "agent-s25-simwan6")
        print("S25_V_SIMWAN6_"+("INNER" if inner_layer else "OUTER")+
              "_ORIGINAL_MCP_PEER_DELETED=PASS")
        return
    require(name in by_name, "main Box missing from independent native readback")
    main = by_name[name]
    require(main.get("config", {}).get("volatile.uuid") == data["uuid"],
            "original SDK primary native UUID mismatch")
    require(peer_name in by_name, "second SDK-created sibling absent")
    peer = by_name[peer_name]
    require(peer.get("status") in ("Running", "Ready"), "sibling not Running")
    require(peer.get("config", {}).get("volatile.uuid") == data["wan6_peer_uuid"],
            "original WAN sibling native UUID differs")
    require(main.get("status") in ("Running", "Ready"), "primary not Running")
    for inst, enabled in ((main, True), (peer, False)):
        nic = inst.get("expanded_devices", {}).get("eth0", {})
        require(nic.get("security.acls") and
                nic.get("security.acls.default.ingress.action") == "reject" and
                nic.get("security.ipv4_filtering") == "true" and
                nic.get("security.ipv6_filtering") == "true" and
                nic.get("security.mac_filtering") == "true",
                "missing actual per-NIC native IPv4/IPv6/MAC safety boundary")
        if enabled:
            require(nic.get("security.acls.default.egress.action") == "allow",
                    "MCP Internet=on lacked native permission")
        else:
            require(nic.get("security.acls.default.egress.action") != "allow",
                    "MCP Internet=off still permits native egress")
    cases = [
        ("wan6_peer_create_op", data["wan6_peer_uuid"]),
        ("wan6_main_off_op", data["uuid"]),
        ("wan6_main_on_op", data["uuid"]),
        ("wan6_peer_on_op", data["wan6_peer_uuid"]),
        ("wan6_peer_off_op", data["wan6_peer_uuid"]),
    ]
    for prefix, uuid in (("main_initial_on", data["uuid"]),
                         ("main_off", data["uuid"]), ("main_on", data["uuid"]),
                         ("main_final_on", data["uuid"]),
                         ("peer_initial_off", data["wan6_peer_uuid"]),
                         ("peer_on", data["wan6_peer_uuid"]),
                         ("peer_off", data["wan6_peer_uuid"])):
        cases.append(("wan6_"+prefix+"_probe_op",uuid))
    for field, uuid in cases:
        require(data.get(field), "missing original SDK operation "+field)
        if inner_layer:
            sql_op = ("import sqlite3,sys;"
                      "c=sqlite3.connect('file:/var/lib/boxweave/ledger.sqlite?mode=ro',uri=True);"
                      "v=c.execute('SELECT state,box_uuid,agent_ref FROM operations WHERE id=?',(sys.argv[1],)).fetchone();"
                      "assert v==('SUCCEEDED',sys.argv[2],'agent-s25-simwan6'),v")
            inner("python3", "-c", sql_op, data[field], uuid)
        else:
            read_ledger(os.environ["BW_S25_OUTER_DB"], data[field],
                        uuid, "agent-s25-simwan6")
    url = "https://[2001:db8:25:2026::2]:18479/witness"
    def call_guest(guest):
        args = ["incus", "exec", guest, "--project", project, "--", "curl",
                "-g", "-6", "--noproxy", "*",
                "--cacert", "/tmp/s25wan6-ca.crt", "--connect-timeout", "3",
                "--max-time", "8", "-fsS", url]
        if inner_layer:
            args = ["sudo","incus","exec", "bw-s25-mcp-l1","--project",
                    os.environ["BW_S25_PROJECT"],"--"] + args
        else:
            args = ["sudo"] + args
        return subprocess.run(args, text=True, stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE, timeout=22, check=False)
    pos = call_guest(name)
    require(pos.returncode == 0 and pos.stdout.strip() ==
            "S25_SIMULATED_IPV6_WAN_HTTPS_OK",
            "independent actual guest TLS/witness positive failed: "+pos.stderr[:240])
    neg = call_guest(peer_name)
    require(neg.returncode != 0 and
            "S25_SIMULATED_IPV6_WAN_HTTPS_OK" not in neg.stdout,
            "unpermitted peer accessed external routed IPv6 WAN")
    # Positive TLS must terminate in the upstream namespace, not on the
    # private guest bridge/gateway. Original agent never reads native state.
    if inner_layer:
        require("2001:db8:25:2026::2" in inner("ip","-n","s25simwan6","-6","addr","show"),
                "inner HTTPS responder not in its own routed namespace")
    else:
        require("2001:db8:25:2026::2" in cli("sudo","ip","-n","s25simwan6","-6","addr","show"),
                "outer HTTPS responder not in separately routed namespace")
    print("S25_V_SIMWAN6_"+("INNER" if inner_layer else "OUTER")+
          "_SDK_ORIGINALS_NATIVE_NIC_ROUTED_TLS_PEER_DENIAL=PASS")

def inspect_agents():
    inspect_l1()
    m=as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    require(m.get("peer_uuid") and m.get("peer_create_op") and m.get("peer_delete_op"),
            "second Agent's independent create/cleanup receipt absent")
    require(m.get("agent_a_exec") and m.get("agent_b_exec") and
            m["agent_a_exec"]!=m["agent_b_exec"],
            "two Agent native operation identities missing/collapsed")
    host_read("absent",os.environ["BW_S25_PROJECT"],"bw-s25-mcp-peer",m["peer_uuid"])
    read_ledger(os.environ["BW_S25_OUTER_DB"],m["peer_create_op"],
                m["peer_uuid"],"agent-s25-beta")
    read_ledger(os.environ["BW_S25_OUTER_DB"],m["peer_delete_op"],
                m["peer_uuid"],"agent-s25-beta")
    read_ledger(os.environ["BW_S25_OUTER_DB"],m["agent_a_exec"],
                m["uuid"],"agent-s25-alpha")
    read_ledger(os.environ["BW_S25_OUTER_DB"],m["agent_b_exec"],
                m["peer_uuid"],"agent-s25-beta")
    print("S25_V_TWO_AGENT_SEPARATE_NATIVE_ORIGINALS_AND_CLEANUP=PASS")

def inspect_quiet_operation():
    inspect_l1()
    m=as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    require(m.get("quiet_op"), "lost external quiet Operation checkpoint")
    read_ledger(os.environ["BW_S25_OUTER_DB"],m["quiet_op"],m["uuid"],
                "agent-outer-alpha")
    print("S25_V_QUIET_ORIGINAL_SQLITE_RECONCILED=PASS")


def inspect_git_workflow():
    inspect_l1()
    m=as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    for k in ("git_setup_op","git_failure_op","git_repair_op"):
        require(m.get(k) and len(m[k])==32,
                "missing independently verifiable original Git Operation "+k)
        read_ledger(os.environ["BW_S25_OUTER_DB"],m[k],m["uuid"],
                    "agent-outer-alpha")
    script = (
       "set -eu; cd /root/s25-git-loop; "
       "test \"$(git rev-list --count HEAD)\" = 2; "
       "test -z \"$(git status --porcelain)\"; "
       "git show HEAD~1:calc.py | grep -q 'return value + value + 1'; "
       "grep -q 'return value + value$' calc.py; "
       "python3 -B -m unittest -q test_calc; "
       "echo S25_V_GUEST_GIT_TWO_COMMITS_AND_REPAIR=PASS"
    )
    witness=inner("sh","-c",script)
    require("S25_V_GUEST_GIT_TWO_COMMITS_AND_REPAIR=PASS" in witness,
            "independent Guest Git/source/test repair failed")
    print("S25_V_MCP_ORIGINAL_GIT_FAIL_AND_REPAIR_OPS=PASS")


def nested_cleaned():
    inspect_l1()
    checkpoint = as_json(inner("cat", "/root/s25-inner-checkpoint.json"))
    require(checkpoint.get("delete_op") and checkpoint.get("uuid"),
            "inner original MCP delete receipt missing")
    innerproj = inner("sh","-c","printf user-; id -u boxweave")
    child_list = as_json(inner("incus","list","--project",innerproj,"--format","json"))
    require(not any(x.get("name")=="bw-s25-mcp-l2" for x in child_list),
            "inner MCP-deleted L2 still exists")
    code = ("import sqlite3,sys;"
            "c=sqlite3.connect('file:/var/lib/boxweave/ledger.sqlite?mode=ro',uri=True);"
            "v=c.execute('SELECT state,box_uuid FROM operations WHERE id=?',(sys.argv[1],)).fetchone();"
            "assert v==('SUCCEEDED',sys.argv[2]),v")
    inner("python3","-c",code,checkpoint["delete_op"],checkpoint["uuid"])
    print("S25_V_INNER_ORIGINAL_MCP_L2_DELETE_NATIVE_ABSENCE=PASS")


def verify_round2_offline_rollback():
    """Independent no-token post-mortem on exact installed rollback binary.

    Not a substitute for in-flight lost native ACK/SQLite kill injection:
    verifies safe local operator fallback with full original MCP history.
    """
    m = as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    project = os.environ["BW_S25_PROJECT"]
    require(m.get("create_op") and m.get("delete_op") and m.get("uuid"),
            "original rollback checkpoint lacks actual L0 lifecycle")
    expected_sha = os.environ["BW_S25_BINARY_SHA256"]
    installed = cli("sudo", "sha256sum", "/usr/local/bin/boxweave").split()[0]
    require(installed == expected_sha,
            "offline operator rollback did not restore exact originally accepted binary")
    status = as_json(cli("sudo", "/usr/local/bin/boxweave",
                         "install-status", "-project", project))
    require(status.get("binary_sha256") == expected_sha and
            status.get("build_commit") == os.environ.get("GITHUB_SHA") and
            status.get("service_active") and
            status.get("restart_required") is False,
            "rollback installer status inconsistent with original candidate")
    require(cli("sudo", "systemctl", "is-active", "boxweave.service") == "active",
            "BoxWeave service was not recovered after root-local rollback")
    # After externally admitted lifecycle cleanup, NO native Box is adopted,
    # re-created, or removed by an implicit installer fallback.
    boxes = as_json(native("list", "--project", project, "--format", "json"))
    require(boxes == [], "native Box leaked or recreated after rollback")
    original = [
        ("create_op", "uuid", "agent-outer-alpha"),
        ("delete_op", "uuid", "agent-s25-cleanup"),
        ("peer_delete_op", "peer_uuid", "agent-s25-beta"),
        ("quiet_op", "uuid", "agent-outer-alpha"),
        ("wan6_peer_delete_op", "wan6_peer_uuid", "agent-s25-simwan6"),
        ("agent_a_exec", "uuid", "agent-s25-alpha"),
        ("agent_b_exec", "peer_uuid", "agent-s25-beta"),
    ]
    for op, uuid, actor in original:
        require(m.get(op) and m.get(uuid), "original fault receipt missing "+op)
        read_ledger(os.environ["BW_S25_OUTER_DB"], m[op], m[uuid], actor)
    integrity = (
        "import sqlite3,sys;"
        "d=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True,timeout=3);"
        "assert d.execute('PRAGMA integrity_check').fetchone()==('ok',)"
    )
    cli("sudo", "python3", "-c", integrity, os.environ["BW_S25_OUTER_DB"])
    print("S25_V_R2_ORIGINAL_OPS_EXACT_BINARY_SQLITE_NATIVE_ROLLBACK=PASS")

def verify_r12_lost_reply():
    m = as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    require(m.get("r12_lost_reply_exec_op") and m.get("uuid"), "lost ACK original key/UUID missing")
    require(m.get("r12_lost_key") == "s25-r12-lost-mcp-reply-exec", "wrong lost response original key")
    require(m.get("r12_lost_work") == "lab/s25/r12-lost-reply", "wrong original work ref")
    read_ledger(os.environ["BW_S25_OUTER_DB"], m["r12_lost_reply_exec_op"],
                m["uuid"], "agent-s25-acklost")
    # The independently read original UUID must still exist unchanged after
    # the old Agent's response was lost and original token revoked.
    host_read("present", os.environ["BW_S25_PROJECT"], m["name"], m["uuid"])
    verify = (
        "import sqlite3,sys;"
        "c=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True);"
        "v=list(c.execute('SELECT id,state,agent_ref,work_ref FROM operations "
        "WHERE resource_key=? AND action=? AND work_ref=?',"
        "(sys.argv[2], 'box_exec', sys.argv[3])));"
        "assert v==[(sys.argv[4],'SUCCEEDED','agent-s25-acklost',sys.argv[3])],v"
    )
    cli("sudo","python3","-c",verify,os.environ["BW_S25_OUTER_DB"],
        "box:"+m["name"],m["r12_lost_work"],m["r12_lost_reply_exec_op"])
    print("S25_V_R12_LOST_ACK_EXACT_ONE_NATIVE_EXEC_ORIGINAL=PASS")


def verify_r12_foreign():
    """Independent outer Verifier repeats actual packet tests inside the nested native daemon."""
    m = as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    require(m.get("uuid"), "outer nested L1 exact original missing")
    project = os.environ["BW_S25_PROJECT"]
    # Outer witness: exact parent vs inner Incus socket inode must differ.
    outer_id = cli("sudo", "stat", "-c", "%d:%i", "/var/lib/incus/unix.socket.user")
    inner_id = cli("sudo", "incus", "exec", "bw-s25-mcp-l1", "--project",
                   project, "--", "stat", "-c", "%d:%i", "/var/lib/incus/unix.socket.user")
    require(outer_id.strip() != inner_id.strip(),
            "nested Incus socket unexpectedly same inode as parent")
    data = as_json(cli("sudo", "incus", "exec", "bw-s25-mcp-l1", "--project",
                       project, "--", "cat", "/root/s25-r12-foreign-state.json"))
    require(data.get("state") == "VERIFIED_SETUP", "foreign fixture not verified")
    assert data["evidence"]["socket_scope"]["no_parent_socket_bind_mount"]
    assert data["evidence"]["socket_scope"]["foreign_project_denied_to_service"]
    output = cli("sudo", "incus", "exec", "bw-s25-mcp-l1", "--project",
                 project, "--", "env", "BW_S25_R12_DISPOSABLE=1", "python3",
                 "/root/s25_r12_foreign_l1.py", "verify")
    require("S25_V_R12_L1_FOREIGN_PROJECT_PACKET_AND_PARENT_SOCKET=PASS" in output,
            "independent foreign IPv4/IPv6 packet re-test did not PASS")
    print("S25_V_R12_L1_FOREIGN_PROJECT_PACKET_AND_PARENT_SOCKET=PASS")


def verify_r12_foreign_clean():
    """Post-cleanup readback from a separate outer actor, not from cleanup command."""
    project = os.environ["BW_S25_PROJECT"]
    data = as_json(cli("sudo", "incus", "exec", "bw-s25-mcp-l1", "--project",
                       project, "--", "cat", "/root/s25-r12-foreign-state.json"))
    require(data.get("state") == "CLEANED", "foreign fixture cleanup not committed")
    names = cli("sudo", "incus", "exec", "bw-s25-mcp-l1", "--project",
                project, "--", "incus", "list", "--project", "default",
                "-f", "csv", "-c", "n")
    require(not names.strip(), "foreign native Guest leaked")
    owned_id = cli("sudo", "incus", "exec", "bw-s25-mcp-l1", "--project",
                   project, "--", "incus", "config", "get", "bw-s25-mcp-l2",
                   "volatile.uuid", "--project", "user-1001")
    require(owned_id.strip() == data["evidence"]["owned_uuid"],
            "foreign fixture cleanup changed original nested UUID")
    print("S25_V_R12_L1_FOREIGN_NATIVE_UUID_CLEANUP=PASS")


def cleaned():
    m = as_json(Path(os.environ["BW_S25_CHECKPOINT"]).read_text())
    require(m.get("delete_op") and m.get("uuid"),
            "missing original authenticated L0 delete receipt")
    host_read("absent", os.environ["BW_S25_PROJECT"], m["name"], m["uuid"])
    read_ledger(os.environ["BW_S25_OUTER_DB"], m["delete_op"], m["uuid"],
                "agent-s25-cleanup")
    print("S25_V_EXACT_NATIVE_L0_CLEANUP_ORIGINAL_SQLITE=PASS")

if __name__ == "__main__":
    require(os.environ.get("GITHUB_ACTIONS") == "true" and
            os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted",
            "one-use GitHub LAB only")
    require(len(sys.argv) == 2, "mode required")
    if sys.argv[1] == "l1":
        inspect_l1()
    elif sys.argv[1] == "l2":
        inspect_l2()
    elif sys.argv[1] == "wan6outer":
        inspect_simwan6(False)
    elif sys.argv[1] == "wan6outerclean":
        inspect_simwan6(False, cleaned=True)
    elif sys.argv[1] == "wan6inner":
        inspect_simwan6(True)
    elif sys.argv[1] == "wan6innerclean":
        inspect_simwan6(True, cleaned=True)
    elif sys.argv[1] == "agents":
        inspect_agents()
    elif sys.argv[1] == "git":
        inspect_git_workflow()
    elif sys.argv[1] == "quiet":
        inspect_quiet_operation()
    elif sys.argv[1] == "l2clean":
        nested_cleaned()
    elif sys.argv[1] == "round2-postrollback":
        verify_round2_offline_rollback()
    elif sys.argv[1] == "r12-lost-reply":
        verify_r12_lost_reply()
    elif sys.argv[1] == "r12-foreign":
        verify_r12_foreign()
    elif sys.argv[1] == "r12-foreign-clean":
        verify_r12_foreign_clean()
    elif sys.argv[1] == "clean":
        cleaned()
    else:
        raise SystemExit("unknown independent verifier mode")
