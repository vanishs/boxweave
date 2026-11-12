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
    print("S25_V_L1_MCP_CREATED_L2_NATIVE_UUID_SQLITE_BINARY=PASS")

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
    elif sys.argv[1] == "agents":
        inspect_agents()
    elif sys.argv[1] == "l2clean":
        nested_cleaned()
    elif sys.argv[1] == "clean":
        cleaned()
    else:
        raise SystemExit("unknown independent verifier mode")
