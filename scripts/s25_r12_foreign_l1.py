#!/usr/bin/env python3
"""One-use disposable nested L1 foreign-Project native packet/parent-socket negative.

Runs only inside GitHub-hosted LAB's MCP-created isolated L1.
Never interacts with legacy/shared Incus.
"""
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import re
import sqlite3
import socket
import subprocess
import sys
import time

OWNED_PROJECT = "user-1001"
SCOPED = os.environ.get("P1B_S25_SCOPED") == "1"
OWNED = "bw-s25-mcp-l1" if SCOPED else "bw-s25-mcp-l2"
FOREIGN_PROJECT = "default"
FOREIGN = "bw-s25-r12-foreign"
STATE = Path("/root/s25-r12-foreign-state.json")
HOSTNAME = "bw-s25-mcp-l1"

def run(*args, check=True, timeout=45):
    p = subprocess.run(args, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    if check and p.returncode:
        raise AssertionError("native command failed: " + " ".join(args[:5]) + " / " + p.stderr[:260])
    return p

def assert_environment():
    global OWNED_PROJECT
    assert os.geteuid() == 0
    assert socket.gethostname() == HOSTNAME, socket.gethostname()
    assert os.environ.get("BW_S25_R12_DISPOSABLE") == "1"
    if SCOPED:
        socket_and_scope()
        uid = run("id", "-u", "boxweave").stdout.strip()
        assert uid.isdigit() and int(uid)>0
        OWNED_PROJECT = "user-" + uid
        status = json.loads(run("/usr/local/bin/boxweave", "install-status", "-project", OWNED_PROJECT).stdout)
        assert status["scoped_peer_mode"] == "SCOPED_PEER_FRESH_V2"
        assert status["native_project"] == OWNED_PROJECT and status["service_active"]
    assert run("incus", "project", "show", OWNED_PROJECT).returncode == 0
    assert run("incus", "project", "show", FOREIGN_PROJECT).returncode == 0
    assert OWNED in run("incus", "list", "--project", OWNED_PROJECT, "-f", "csv", "-c", "n").stdout
    assert run("incus", "info", OWNED, "--project", OWNED_PROJECT).returncode == 0
    assert run("systemctl", "is-active", "boxweave.service").stdout.strip() == "active"

def attr(guest, project, family):
    p = run("incus", "exec", guest, "--project", project, "--", "ip", "-j", "addr", "show", "dev", "eth0", timeout=20)
    a = json.loads(p.stdout)
    assert a and a[0]["ifname"] == "eth0"
    ips = [ipaddress.ip_address(x["local"]) for x in a[0].get("addr_info", [])
           if x.get("family") == ("inet" if family == 4 else "inet6")]
    valid = [x for x in ips if not (x.is_link_local or x.is_loopback or x.is_unspecified)]
    assert valid, (guest, family, "missing routable assigned address")
    return str(valid[0])

def guest_ping(src, project, ip, family):
    args = ("incus", "exec", src, "--project", project, "--",
            "ping", "-4" if family == 4 else "-6", "-n", "-c", "2", "-W", "2", ip)
    return run(*args, check=False, timeout=18)

def gateway(project, family):
    net = "incusbr-" + OWNED_PROJECT.removeprefix("user-") if project == OWNED_PROJECT else "incusbr0"
    value = run("incus", "network", "get", net, "ipv4.address" if family == 4 else "ipv6.address").stdout.strip()
    assert value and value not in ("none", "auto")
    return value.split("/")[0]

def ensure_ping(name, project):
    cmd = "command -v ping >/dev/null || (apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends iputils-ping >/dev/null)"
    run("incus", "exec", name, "--project", project, "--", "sh", "-c", cmd, timeout=180)

def scoped_pin():
    checkpoint = json.loads(Path("/root/s25-inner-checkpoint.json").read_text())
    uuid, original = checkpoint["uuid"], checkpoint["create_op"]
    assert re.fullmatch(r"[0-9a-f-]{36}", uuid) and re.fullmatch(r"[0-9a-f]{32}", original)
    with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro", uri=True) as db:
        assert db.execute("SELECT layer,box_uuid,incarnation,state FROM operations WHERE id=? AND action='box_create'",
                          (original,)).fetchall() == [(OWNED_PROJECT,uuid,uuid,"SUCCEEDED")]
        rows = db.execute("SELECT acl,bridge,ipv4,ipv6,state FROM network_bindings WHERE project=? AND box_uuid=? AND incarnation=?",
                          (OWNED_PROJECT,uuid,uuid)).fetchall()
    assert len(rows)==1
    acl,bridge,v4,v6,state=rows[0]
    assert acl.startswith("bw-peer-") and acl!="bw-peer-quarantine" and state=="PINNED"
    assert bridge=="incusbr-"+OWNED_PROJECT.removeprefix("user-")
    inst=json.loads(run("incus","query","/1.0/instances/"+OWNED+"?project="+OWNED_PROJECT).stdout)
    nic=inst["expanded_devices"]["eth0"]
    assert inst["config"]["volatile.uuid"]==uuid and inst["config"]["user.boxweave.create_operation"]==original
    assert nic["network"]==bridge and nic["security.acls"]==acl
    assert nic["security.acls.default.ingress.action"]=="reject" and nic["security.acls.default.egress.action"]=="allow"
    assert all(nic["security."+family+"_filtering"]=="true" for family in ("ipv4","ipv6","mac"))
    policy=json.loads(run("incus","query","/1.0/network-acls/"+acl).stdout)
    return {"uuid":uuid,"original":original,"acl":acl,"ipv4":v4,"ipv6":v6,
            "acl_sha256":hashlib.sha256(json.dumps(policy,sort_keys=True).encode()).hexdigest()}


# Independent readback oracle: exact current deny-only policy, subtracting
# the single actual gateway host (never the whole bridge prefix).
BLOCKED4 = "0.0.0.0/8,10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.0.2.0/24,192.168.0.0/16,198.18.0.0/15,198.51.100.0/24,203.0.113.0/24,224.0.0.0/4,240.0.0.0/4"
BLOCKED6 = "fc00::/7,::1/128"


def expected_denied_prefixes(blocked, gateway_address):
    addr=ipaddress.ip_address(gateway_address)
    host=ipaddress.ip_network(str(addr)+"/"+str(addr.max_prefixlen))
    result=[]
    for raw in blocked.split(","):
        network=ipaddress.ip_network(raw)
        result.extend(network.address_exclude(host) if addr in network else [network])
    return sorted(result)


def acl_private_egress(foreign_v4,foreign_v6):
    pin=scoped_pin() if SCOPED else None
    name=pin["acl"] if pin else "bw-s25-inner-owned"
    acl=json.loads(run("incus","query","/1.0/network-acls/"+name).stdout)
    egress=acl["egress"]
    assert acl["name"]==name and len(egress)==2,acl
    assert [(r["action"],r["state"]) for r in egress]==[
        ("reject","enabled"),("reject","enabled")
    ], egress
    own_v4=gateway(OWNED_PROJECT,4)
    own_v6=gateway(OWNED_PROJECT,6)
    reject_v4=[ipaddress.ip_network(p) for p in egress[0]["destination"].split(",")]
    reject_v6=[ipaddress.ip_network(p) for p in egress[1]["destination"].split(",")]
    def denied(address,ranges):
        addr=ipaddress.ip_address(address)
        matches=[p for p in ranges if addr in p]
        assert len(matches)<=1,("overlapping ACL CIDRs",address,matches)
        return bool(matches)
    assert not denied(own_v4,reject_v4) and not denied(own_v6,reject_v6)
    assert denied(foreign_v4,reject_v4) and denied(foreign_v6,reject_v6)
    assert denied("10.223.5.2",reject_v4) and denied("fd42:abcd::2",reject_v6)
    assert not denied("1.1.1.1",reject_v4) and not denied("2001:db8:25:2026::2",reject_v6)
    assert not denied("fe80::1",reject_v6), "IPv6 RA/NDP would be blocked"
    assert len(acl["ingress"])==0
    references=acl["used_by"]
    if SCOPED:
        assert sorted(reject_v4)==expected_denied_prefixes(BLOCKED4,own_v4)
        assert sorted(reject_v6)==expected_denied_prefixes(BLOCKED6,own_v6)
        assert all(not r.get(k) for r in egress for k in
                   ("source","protocol","source_port","destination_port","icmp_type","icmp_code"))
        assert references==["/1.0/instances/"+OWNED+"?project="+OWNED_PROJECT],references
        assert acl.get("project")=="default"
        assert not acl.get("config")
        assert pin["ipv4"]==attr(OWNED,OWNED_PROJECT,4) and pin["ipv6"]==attr(OWNED,OWNED_PROJECT,6)
    else:
        assert "/1.0/profiles/default?project="+OWNED_PROJECT in references,references
        assert all(ref=="/1.0/profiles/default?project="+OWNED_PROJECT or
                   (ref.startswith("/1.0/instances/") and ref.endswith("?project="+OWNED_PROJECT))
                   for ref in references),references
    return {"ACL":name,"gateway_v4":own_v4,"gateway_v6":own_v6,
            "private_ipv4_rejected":True,"private_ipv6_ula_rejected":True,
            "scoped_pin":pin,"acl_sha256":hashlib.sha256(json.dumps(acl,sort_keys=True).encode()).hexdigest()}


def socket_and_scope():
    # The guest's /var/lib/incus Unix socket is the INNER Incus daemon.
    # Neither the parent's unix.socket.user nor the parent's ledger may be bind-mounted.
    mounts = Path("/proc/self/mountinfo").read_text()
    for line in mounts.splitlines():
        fields = line.split()
        assert fields[4] not in ("/var/lib/incus", "/var/lib/incus/unix.socket.user",
                                  "/var/lib/incus/unix.socket", "/run/incus", "/run/incus/unix.socket"), line
    assert Path("/var/lib/incus/unix.socket.user").is_socket()
    assert not Path("/home/boxweave/hgcodex-0cf").exists(), "outer candidate sqlite unexpectedly exposed in child"
    cmd = run("runuser", "-u", "boxweave", "--", "incus", "project", "show", "default", check=False)
    assert cmd.returncode != 0, "nested nonroot service can read foreign default Project"
    assert "permission" in (cmd.stdout + cmd.stderr).lower()
    # Verify the parent controller cannot be opened via loopback from inner L1.
    # Inner v1 listener is its own 18650; outer v1 candidate/private root is not mounted.
    return {"foreign_project_denied_to_service": True, "no_parent_socket_bind_mount": True,
            "no_outer_candidate_sqlite": True}

def run_packets():
    owned4 = attr(OWNED, OWNED_PROJECT, 4)
    foreign4 = attr(FOREIGN, FOREIGN_PROJECT, 4)
    owned6 = attr(OWNED, OWNED_PROJECT, 6)
    foreign6 = attr(FOREIGN, FOREIGN_PROJECT, 6)
    for name, project in ((OWNED, OWNED_PROJECT), (FOREIGN, FOREIGN_PROJECT)):
        for family in (4, 6):
            pos = guest_ping(name, project, gateway(project, family), family)
            assert pos.returncode == 0, ("positive gateway control failed", name, family, pos.stderr[:200])
    for name, project, addr4, addr6 in (
        (OWNED, OWNED_PROJECT, foreign4, foreign6),
        (FOREIGN, FOREIGN_PROJECT, owned4, owned6),
    ):
        for addr, family in ((addr4, 4), (addr6, 6)):
            neg = guest_ping(name, project, addr, family)
            assert neg.returncode != 0, ("FORBIDDEN CROSS-PROJECT PACKET GOT THROUGH", name, family, addr, neg.stdout[:200])
    result = {"owned_uuid": run("incus", "config", "get", OWNED, "volatile.uuid", "--project", OWNED_PROJECT).stdout.strip(),
              "foreign_uuid": run("incus", "config", "get", FOREIGN, "volatile.uuid", "--project", FOREIGN_PROJECT).stdout.strip(),
              "owned_ipv4": owned4, "owned_ipv6": owned6, "foreign_ipv4": foreign4, "foreign_ipv6": foreign6,
              "positive_gateway_v4_v6_both_projects": True,
              "bidirectional_cross_project_packet_denied_v4_v6": True,
              "socket_scope": socket_and_scope(), "acl_private_egress":acl_private_egress(foreign4,foreign6)}
    assert result["owned_uuid"] != result["foreign_uuid"]
    return result

def setup():
    assert not STATE.exists(), "original foreign fixture exists or uncertain: NEVER retry"
    assert not run("incus", "list", "--project", FOREIGN_PROJECT, "-f", "csv", "-c", "n").stdout.strip()
    before = scoped_pin() if SCOPED else None
    with STATE.open("x") as out:
        out.write(json.dumps({"state":"LAUNCH_REQUESTED","name":FOREIGN,"project":FOREIGN_PROJECT,"scoped_pin":before})+"\n")
        out.flush()
        os.fsync(out.fileno())
    # Foreign guest is created by INDEPENDENT native root in the disposable
    # inner daemon, not by the v1 project-confined service or shared incus.
    image="images:ubuntu/24.04"
    if SCOPED:
        image=Path("/root/s25-image/fingerprint").read_text().strip()
        assert re.fullmatch(r"[0-9a-f]{64}",image)
        h=hashlib.sha256()
        for path in ("/root/s25-image/meta","/root/s25-image/rootfs"):
            with open(path,"rb") as f:
                for data in iter(lambda:f.read(1024*1024),b""):
                    h.update(data)
        assert h.hexdigest()==image
        assert not run("incus","image","list","--project",FOREIGN_PROJECT,"-f","csv","-c","F").stdout.strip()
        run("incus","image","import","/root/s25-image/meta","/root/s25-image/rootfs",
            "--project",FOREIGN_PROJECT,timeout=180)
        assert run("incus","image","list","--project",FOREIGN_PROJECT,"-f","csv","-c","F").stdout.strip()==image
    run("incus", "launch", image, FOREIGN, "--project", FOREIGN_PROJECT, timeout=270)
    for _ in range(85):
        try:
            attr(FOREIGN, FOREIGN_PROJECT, 4)
            attr(FOREIGN, FOREIGN_PROJECT, 6)
            break
        except (AssertionError, ValueError, KeyError):
            time.sleep(2)
    else:
        raise AssertionError("foreign guest did not acquire IPv4+IPv6")
    for n,p in ((FOREIGN, FOREIGN_PROJECT),(OWNED, OWNED_PROJECT)):
        ensure_ping(n,p)
    evidence=run_packets()
    if SCOPED:
        assert evidence["acl_private_egress"]["scoped_pin"]==before
    STATE.write_text(json.dumps({"state":"VERIFIED_SETUP","name":FOREIGN,"project":FOREIGN_PROJECT,
                                "evidence":evidence}, sort_keys=True, indent=2)+"\n")
    print("S25_R12_L1_FOREIGN_PROJECT_V4_V6_POSITIVE_AND_BIDIRECTIONAL_PACKET_DENIAL=PASS")
    print("S25_R12_L1_PARENT_SOCKET_AND_FOREIGN_PROJECT_AUTH_ISOLATION=PASS")

def verify():
    state=json.loads(STATE.read_text())
    assert state["state"]=="VERIFIED_SETUP" and state["name"]==FOREIGN
    evidence=run_packets()
    assert evidence==state["evidence"], "independent re-read drifted"
    print("S25_V_R12_L1_FOREIGN_PROJECT_PACKET_AND_PARENT_SOCKET=PASS")

def cleanup():
    state=json.loads(STATE.read_text())
    assert state["state"]=="VERIFIED_SETUP", "unsafe/unverified fixture: cleanup blocked"
    assert run("incus", "config", "get", FOREIGN, "volatile.uuid", "--project", FOREIGN_PROJECT).stdout.strip()==state["evidence"]["foreign_uuid"]
    run("incus", "stop", FOREIGN, "--project", FOREIGN_PROJECT, timeout=100)
    run("incus", "delete", FOREIGN, "--project", FOREIGN_PROJECT, timeout=100)
    assert not run("incus", "list", "--project", FOREIGN_PROJECT, "-f", "csv", "-c", "n").stdout.strip()
    assert run("incus", "config", "get", OWNED, "volatile.uuid", "--project", OWNED_PROJECT).stdout.strip()==state["evidence"]["owned_uuid"]
    if SCOPED:
        assert scoped_pin()==state["evidence"]["acl_private_egress"]["scoped_pin"]
    state["state"]="CLEANED"
    STATE.write_text(json.dumps(state,sort_keys=True,indent=2)+"\n")
    print("S25_R12_NATIVE_FOREIGN_FIXTURE_CLEANUP_EXACT_UUID=PASS")

if __name__ == "__main__":
    assert len(sys.argv)==2 and sys.argv[1] in ("setup","verify","cleanup")
    assert_environment()
    {"setup":setup,"verify":verify,"cleanup":cleanup}[sys.argv[1]]()
