"""One-use hosted native acceptance of ordinary public Create, no seeded rows."""
import hashlib
import json
import os
import pathlib
import re
import socket
import subprocess
import sys
import time

from mcp_testkit import LoopbackMCP, token_from_file

assert os.environ.get("GITHUB_ACTIONS") == "true"
assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
assert os.environ.get("GITHUB_REPOSITORY") == "vanishs/boxweave"
assert os.environ.get("GITHUB_REF") == "refs/heads/feat/v1-s5b-acceptance-20261120"
assert os.geteuid() != 0
work, project, bridge, acl, binary_sha = sys.argv[1:]
work = pathlib.Path(work)
assert str(work).startswith("/tmp/bw-v1-s25-lab.")
assert re.fullmatch(r"user-[0-9]+", project)
assert re.fullmatch(r"[a-zA-Z0-9_-]+", bridge)
assert re.fullmatch(r"[a-zA-Z0-9_-]+", acl)
assert re.fullmatch(r"[0-9a-f]{64}", binary_sha)
bw = "/usr/local/bin/boxweave"
db = "/var/lib/boxweave/ledger.sqlite"
root_unit = "boxweave-native-peer-root.service"
unit_path = "/etc/systemd/system/" + root_unit
names = {k: "s5b-prod-peer-" + k for k in "abc"}
boxes = {}
pins = {}
ops = {}
actor1 = token_from_file(work / "private/admin1.secret")
actor2 = token_from_file(work / "private/admin2.secret")

def run(args, timeout=90):
    return subprocess.check_output(args, text=True, timeout=timeout).strip()

def native(*args):
    return run(["sudo", "incus", "--project", project, *args])

def service(*args):
    return run(["sudo", "runuser", "-u", "boxweave", "--",
                "env", "-u", "INCUS_CONF", "-u", "INCUS_CONFIG_DIR", "-u", "INCUS_DIR",
                "-u", "XDG_CONFIG_HOME", "HOME=/home/boxweave",
                "XDG_CONFIG_HOME=/home/boxweave/.config", bw, *args], timeout=240)

def sql(query):
    return run(["sudo", "sqlite3", "-readonly", db, query])

def save(key, result):
    assert key not in ops
    ops[key] = result
    path = work / "private/peer-production-checkpoint.json"
    temp = path.with_suffix(".tmp")
    with temp.open("w") as f:
        json.dump({"boxes": boxes, "pins": pins, "operations": ops}, f, sort_keys=True)
        f.flush()
        os.fsync(f.fileno())
    os.replace(temp, path)

def admit(m, tool, key, **args):
    result = m.call(tool, dict(args, idempotency_key="s5b-prod-peer-" + key,
                               work_ref="s5b-peer-production-ci",
                               agent_ref="s5b-native-ci")).structured
    oid = result["operation_id"]
    assert re.fullmatch(r"[0-9a-f]{32}", oid)
    assert not result.get("duplicate"), key
    save(key, oid)
    print("S5B_PROD_PEER_ORIGINAL=" + key + ":" + oid, flush=True)
    return oid

def wait(m, oid, want="SUCCEEDED", native_required=False, seconds=150):
    result = m.poll_operation(oid, seconds=seconds)
    assert result["operation_id"] == oid and result["state"] == want, (
        oid, result["state"], result.get("error_class"))
    if native_required:
        assert re.fullmatch(r"[0-9a-f-]{36}", result["native_operation_id"])
    return result

def open_client(actor, mutations=False):
    return LoopbackMCP(token=actor, timeout=15, allow_mutations=mutations)

def status(m):
    h = m.host_status()
    assert h["version"] == "1.0.0-rc.10-dev-peer-ci"
    assert h["commit"] == os.environ["GITHUB_SHA"]
    assert h["binary_sha256"] == binary_sha and h["ready"]

def root_apply(m, key, allow):
    original = admit(m, "box_peer_set", key, source_box_id=boxes["a"],
                     target_box_id=boxes["b"], allow=allow)
    assert m.operation_get(original)["state"] == "ADMITTED"
    if subprocess.run(["sudo", "test", "-e", unit_path]).returncode == 0:
        # Only the previous one-use unit on this disposable runner, not a
        # user Host. Preserve its exact reviewed text and root journal.
        subprocess.run(["sudo", "systemctl", "stop", root_unit], check=True)
        subprocess.run(["sudo", "systemctl", "disable", root_unit], check=True)
        subprocess.run(["sudo", "mv", unit_path,
                        str(work / "private" / ("unit-before-" + key))], check=True)
        subprocess.run(["sudo", "systemctl", "daemon-reload"], check=True)
    common = ["-db", db, "-project", project, "-operation-id", original]
    service("native-peer-root-stage", *common)
    assert sql("select state from operations where id='" + original + "'") == "DISPATCHING"
    scope = common + ["-network", bridge, "-acl", acl]
    plan = json.loads(run(["sudo", bw, "native-peer-root-plan", *scope]))
    digest = plan["plan_sha256"]
    assert re.fullmatch(r"[0-9a-f]{64}", digest)
    run(["sudo", bw, "native-peer-root-unit-stage", *scope,
         "-peer-plan-sha256", digest, "-approve-local"])
    run(["sudo", "systemctl", "daemon-reload"])
    run(["sudo", "systemctl", "enable", "--now", root_unit])
    for _ in range(40):
        if (subprocess.run(["sudo", "systemctl", "is-active", "--quiet", root_unit]).returncode == 0
                and subprocess.run(["sudo", "test", "-S",
                                    "/run/boxweave-native-peer-root.sock"]).returncode == 0):
            break
        time.sleep(1)
    else:
        raise AssertionError("PRS not ready")
    service("native-peer-root-execute", *common)
    wait(m, original)
    journal = "/var/lib/boxweave-native-peer-conntrack-v1/" + original + ".json"
    receipt = json.loads(run(["sudo", "cat", journal]))
    assert receipt["status"] == "COMPLETE_VERIFIED"
    assert sql("select status||':'||desired from peer_grants where source_uuid='"
               + boxes["a"] + "' and target_uuid='" + boxes["b"] + "'") == (
                   "CONFIRMED:1" if allow else "REVOKED:0")
    print("S5B_PROD_PEER_ROOT_" + key.upper() + "=PASS", flush=True)
    return journal, run(["sudo", "sha256sum", journal]).split()[0]

def guest_probe(source, proto, dest, expected):
    result = subprocess.run(["sudo", "incus", "--project", project, "exec",
                             names[source], "--", "/usr/local/bin/s5b-packet",
                             "probe", proto, dest], timeout=12, capture_output=True)
    assert (result.returncode == 0) == expected, (source, proto, expected)

def packet_matrix(allowed):
    for k in names:
        assert native("config", "get", names[k], "volatile.uuid") == boxes[k]
        assert native("list", names[k], "-c", "s", "-f", "csv") == "RUNNING"
        for attempt in range(30):
            if subprocess.run(["sudo", "incus", "--project", project, "exec", names[k], "--",
                               "systemctl", "is-active", "--quiet", "s5b-packet.service"],
                              capture_output=True).returncode == 0:
                break
            time.sleep(1)
        else:
            raise AssertionError("echo server not ready")
        for p in ("tcp", "udp"):
            guest_probe(k, p, "127.0.0.1", True)
    for p in ("tcp", "udp"):
        for family in ("ipv4", "ipv6"):
            guest_probe("a", p, pins["b"][family], allowed)
            guest_probe("b", p, pins["a"][family], False)
            guest_probe("c", p, pins["b"][family], False)
    print("S5B_PROD_PEER_DUALSTACK_TCP_UDP_ALLOWED=" + str(allowed), flush=True)

def forward_echo(proto, port):
    kind = socket.SOCK_STREAM if proto == "tcp" else socket.SOCK_DGRAM
    with socket.socket(socket.AF_INET, kind) as conn:
        conn.settimeout(5)
        conn.connect(("127.0.0.1", port))
        conn.sendall(b"S5B")
        reply = conn.recv(3)
        assert reply == b"S5B", proto

def observe_forwards(m):
    rows = m.call("port_forward_list", {"box_id": boxes["b"]}).structured["forwards"]
    assert len(rows) == 2
    for row in rows:
        p = row["protocol"]
        assert p in ("tcp", "udp") and row["state"] == "CONFIRMED" and row["native_verified"]
        assert row["operation_id"] == ops["forward-" + p]
        assert row["box_id"] == boxes["b"]
        assert native("config", "device", "get", names["b"], row["native_device"], "type") == "proxy"
        forward_echo(p, 23191 if p == "tcp" else 23192)
    print("S5B_PROD_PEER_ACTIVE_FORWARD_TCP_UDP=PASS", flush=True)

# Empty standard installed ledger and exact normal Create -- no native CLI
# creation, no SQLite write, no fixture helper inserts.
assert not native("list", "-c", "n", "-f", "csv")
for table in ("owned_boxes", "network_bindings", "network_address_fences", "peer_grants"):
    assert sql("select count(*) from " + table) == "0"
with open_client(actor1, True) as m:
    status(m)
    for k in names:
        oid = admit(m, "box_create", "create-" + k, name=names[k], cpu=1,
                    memory_mib=768, disk_gib=2, start=True, internet=False)
        result = wait(m, oid, native_required=True)
        uuid = result["box_uuid"]
        assert re.fullmatch(r"[0-9a-f-]{36}", uuid)
        boxes[k] = uuid
        assert native("config", "get", names[k], "volatile.uuid") == uuid
        fields = sql("select bridge,acl,mac,ipv4,ipv6 from network_bindings where box_uuid='"
                     + uuid + "' and incarnation='" + uuid + "' and state='PINNED'").split("|")
        assert len(fields) == 5 and fields[:2] == [bridge, acl], k
        pins[k] = dict(zip(("bridge", "acl", "mac", "ipv4", "ipv6"), fields))
        for family in ("ipv4", "ipv6"):
            assert native("config", "device", "get", names[k], "eth0", family + ".address") == pins[k][family]
        assert sql("select count(*) from network_address_fences where box_uuid='" + uuid
                   + "' and incarnation='" + uuid + "' and state='RESERVED'") == "2"
        ready = admit(m, "box_wait_ready", "ready-" + k, box_id=uuid,
                      checks=["running", "exec"], timeout_seconds=120)
        wait(m, ready)
    assert len(set(boxes.values())) == 3
    assert len({p["ipv4"] for p in pins.values()}) == 3
    assert len({p["ipv6"] for p in pins.values()}) == 3
    assert len({p["mac"] for p in pins.values()}) == 3
    assert sql("select count(*) from network_bindings where state='PINNED'") == "3"
    assert sql("select count(*) from network_address_fences where state='RESERVED'") == "6"
    assert sql("select count(*) from operation_resource_claims where resource_key like 'create-pin:%'") == "0"
    print("S5B_PROD_MCP_CREATE_AUTOMATIC_PINNED_THREE_BOXES=PASS", flush=True)
    audit = json.loads(run(["sudo", bw, "native-peer-audit", "-db", db,
                            "-project", project, "-network", bridge, "-acl", acl]))
    assert audit["owned_boxes"] == 3
    print("S5B_PROD_INDEPENDENT_ROOT_SCOPE_AUDIT=PASS", flush=True)
    packet_unit = work / "private/s5b-packet.service"
    packet_unit.write_text("[Unit]\nDescription=Disposable packet echo probe\nAfter=network.target\n"
                           "[Service]\nExecStart=/usr/local/bin/s5b-packet server\n"
                           "Restart=on-failure\n[Install]\nWantedBy=multi-user.target\n")
    for k in names:
        native("file", "push", str(work / "staging/packet"), names[k] + "/usr/local/bin/s5b-packet")
        native("exec", names[k], "--", "chmod", "0755", "/usr/local/bin/s5b-packet")
        native("file", "push", str(packet_unit), names[k] + "/etc/systemd/system/s5b-packet.service")
        native("exec", names[k], "--", "systemctl", "daemon-reload")
        native("exec", names[k], "--", "systemctl", "enable", "--now", "s5b-packet.service")
        for attempt in range(40):
            if all(pins[k][family] in native("exec", names[k], "--", "ip",
                                            "-o", "addr", "show", "eth0")
                   for family in ("ipv4", "ipv6")):
                break
            time.sleep(1)
        else:
            raise AssertionError("actual guest static address not assigned")
    packet_matrix(False)
    root_apply(m, "grant", True)
    packet_matrix(True)
    root_apply(m, "revoke", False)
    packet_matrix(False)
    journal, before_hash = root_apply(m, "regrant", True)
    packet_matrix(True)
    print("S5B_PROD_REAL_CREATE_TO_PEER_GRANT_REVOKE_REGRANT=PASS", flush=True)
    # Both confirmed peer and two real Forward claims are now active.
    for proto, port, guest in (("tcp", 23191, 19091), ("udp", 23192, 19092)):
        oid = admit(m, "port_forward_create", "forward-" + proto, box_id=boxes["b"],
                    protocol=proto, host_port=port, guest_port=guest, host_bind="127.0.0.1")
        wait(m, oid, native_required=True)
    observe_forwards(m)
    packet_matrix(True)
    # A real unrelated original in flight while confirmed Peer+Forward exist.
    original = admit(m, "box_exec", "pending", box_id=boxes["c"],
                     argv=["/bin/sh", "-c", "sleep 40; printf s5b-native-pending"],
                     timeout_ms=60000)
    for _ in range(40):
        observed = m.operation_get(original)
        if observed["state"] in ("DISPATCHING", "VERIFYING") and observed.get("native_operation_id"):
            break
        time.sleep(0.2)
    else:
        raise AssertionError("no real in-flight native original")
    native_original = observed["native_operation_id"]
    pid = run(["sudo", "systemctl", "show", "boxweave.service", "-p", "MainPID", "--value"])
    assert int(pid) > 1
    run(["sudo", "systemctl", "kill", "--kill-whom=main", "--signal=SIGKILL", "boxweave.service"])

def reconnect_read(label):
    for _ in range(45):
        try:
            with open_client(actor2) as check:
                status(check)
            break
        except Exception:
            time.sleep(1)
    else:
        raise AssertionError("reconnect unavailable")
    with open_client(actor2) as m:
        status(m)
        observed = m.operation_get(original)
        assert observed["state"] == "UNKNOWN" and observed["native_operation_id"] == native_original
        assert sql("select count(*) from operation_resource_claims where operation_id='" + original + "'") != "0"
        assert sql("select count(*) from operations where id='" + original + "'") == "1"
        assert sql("select state from operations where id='" + ops["regrant"] + "'") == "SUCCEEDED"
        assert run(["sudo", "sha256sum", journal]).split()[0] == before_hash
        assert sql("select status||':'||desired from peer_grants where source_uuid='"
                   + boxes["a"] + "' and target_uuid='" + boxes["b"] + "'") == "CONFIRMED:1"
        packet_matrix(True)
        observe_forwards(m)
    print("S5B_PROD_COMBINED_" + label + "=PASS", flush=True)

reconnect_read("CONTROLLER_CRASH")
assert run(["sudo", "systemctl", "show", "boxweave.service", "-p", "MainPID", "--value"]) != pid
# Completed PRS original is restarted, NEVER dispatched again.
run(["sudo", "systemctl", "restart", "incus.service"])
run(["sudo", "systemctl", "start", "boxweave.service"])
run(["sudo", "systemctl", "restart", root_unit])
reconnect_read("INCUS_PRS_RESTART_COMPLETED_ORIGINAL")
print("S5B_PROD_COMBINED_ACTIVE_PEER_FORWARD_PENDING_EXEC_RESTART=PASS", flush=True)
# Incomplete PRS journal crash recovery is a separate still-open gate.
