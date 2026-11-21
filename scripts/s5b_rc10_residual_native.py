"""One-use RC10 residual CLI + active Forward/partial PRS journal fault; no owned-row seeding."""
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
assert os.environ.get("GITHUB_REF") == "refs/heads/docs/s5b-rc10-closeout-20261121"
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
    assert h["version"] == "1.0.0-rc.10"
    assert h["commit"] == "eb021d3985c787c517cd7aee3efaf433d38f0393"
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
    # Resource claims are durable original history, not disposable lock rows.
    # Verify every retained reservation belongs to its exact successful Create,
    # and that there are no pending/uncertain create-pin claims in this clean lab.
    for k in names:
        pin = pins[k]
        prefix = "create-pin:" + bridge + ":"
        expected = {prefix + "v4:" + pin["ipv4"],
                    prefix + "v6:" + pin["ipv6"],
                    prefix + "mac:" + pin["mac"],
                    prefix + "acl:" + acl}
        actual = sql(
            "select c.resource_key from operation_resource_claims c "
            "join operations o on o.id=c.operation_id where c.operation_id='"
            + ops["create-" + k] + "' and c.layer='" + project
            + "' and o.state='SUCCEEDED' and c.resource_key like 'create-pin:%'"
        ).splitlines()
        assert len(actual) == 4 and set(actual) == expected, k
    assert sql("select count(*) from operation_resource_claims where resource_key like 'create-pin:%'") == "12"
    assert sql("select count(*) from operation_resource_claims c join operations o on o.id=c.operation_id "
               "where c.resource_key like 'create-pin:%' and "
               "o.state in ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == "0"
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
    # Only targeted B1 leaves; no repeated full SDK, migration or retirement matrix.
    cli_ok = True
    auth = subprocess.run(["sudo", "runuser", "-u", "boxweave", "--", "env",
                           "GITHUB_ACTIONS=true", "RUNNER_ENVIRONMENT=github-hosted",
                           "GITHUB_REPOSITORY=vanishs/boxweave",
                           "GITHUB_REF=refs/heads/docs/s5b-rc10-closeout-20261121",
                           "/usr/local/bin/s5b-rc10-cli-auth-fixture", "-project", project],
                          timeout=120, check=False)
    if auth.returncode != 0:
        print("S5B_RC10_CLI_AUTH=FAIL", flush=True)
        cli_ok = False
    root_apply(m, "grant", True)
    root_apply(m, "revoke", False)
    # Purge-plan is READ-ONLY and requires a completely stopped, exact
    # pair with explicit bidirectional DROP, not just default deny.
    # This is only a dedicated empty-before-create hosted fixture.
    for k in names:
        assert native("config", "get", names[k], "volatile.uuid") == boxes[k]
        native("config", "set", names[k], "boot.autostart=false")
        stop = admit(m, "box_stop", "purge-stop-" + k, box_id=boxes[k])
        wait(m, stop, native_required=True)
    acl_before = run(["sudo", "incus", "network", "acl", "show", acl])
    plan = subprocess.run(["sudo", bw, "native-peer-purge-plan", "-db", db,
                           "-project", project, "-network", bridge, "-acl", acl,
                           "-peer-source-uuid", boxes["a"], "-peer-target-uuid", boxes["b"]],
                          timeout=90, capture_output=True, text=True)
    if plan.returncode == 0:
        reviewed = json.loads(plan.stdout)
        assert re.fullmatch(r"[0-9a-f]{64}", reviewed["plan_sha256"])
        assert reviewed["mutation_authorized"] is False
        print("S5B_RC10_CLI_NATIVE_PEER_PURGE_PLAN=PASS", flush=True)
    else:
        print("S5B_RC10_CLI_NATIVE_PEER_PURGE_PLAN=FAIL", flush=True)
        cli_ok = False
    assert run(["sudo", "incus", "network", "acl", "show", acl]) == acl_before
    for k in names:
        assert native("list", names[k], "-c", "s", "-f", "csv") == "STOPPED"
        start = admit(m, "box_start", "purge-start-" + k, box_id=boxes[k])
        wait(m, start, native_required=True)
        ready = admit(m, "box_wait_ready", "purge-ready-" + k, box_id=boxes[k], timeout_seconds=120)
        wait(m, ready)
    root_apply(m, "regrant", True)
    packet_matrix(True)
    # Both confirmed peer and two real Forward claims are now active.
    for proto, port, guest in (("tcp", 23191, 19091), ("udp", 23192, 19092)):
        oid = admit(m, "port_forward_create", "forward-" + proto, box_id=boxes["b"],
                    protocol=proto, host_port=port, guest_port=guest, host_bind="127.0.0.1")
        wait(m, oid, native_required=True)
    observe_forwards(m)
    packet_matrix(True)
    # The existing compiled private test seam cuts after a REAL ACL PUT.
    # It is NOT an added product fault endpoint or a SIGKILL of the
    # production PRS PID. The installed PRS/controller/Incus are then
    # actually restarted with the incomplete original and active Forward
    # records intact.
    before_forwards = m.call("port_forward_list", {"box_id": boxes["b"]}).structured["forwards"]
    original = admit(m, "box_peer_set", "partial-with-forward",
                     source_box_id=boxes["c"], target_box_id=boxes["b"], allow=True)
    assert m.operation_get(original)["state"] == "ADMITTED"
    run(["sudo", "systemctl", "stop", root_unit])
    run(["sudo", "systemctl", "disable", root_unit])
    run(["sudo", "mv", unit_path, str(work / "private/unit-before-partial")])
    run(["sudo", "systemctl", "daemon-reload"])
    common = ["-db", db, "-project", project, "-operation-id", original]
    service("native-peer-root-stage", *common)
    assert sql("select state from operations where id='" + original + "'") == "DISPATCHING"
    scope_args = common + ["-network", bridge, "-acl", acl]
    approved = json.loads(run(["sudo", bw, "native-peer-root-plan", *scope_args]))["plan_sha256"]
    assert re.fullmatch(r"[0-9a-f]{64}", approved)
    run(["sudo", bw, "native-peer-root-unit-stage", *scope_args,
         "-peer-plan-sha256", approved, "-approve-local"])
    run(["sudo", "systemctl", "daemon-reload"])
    run(["sudo", "systemctl", "enable", "--now", root_unit])
    scope = work / "private/partial-scope.json"
    scope.write_text(json.dumps({"Project": project, "Bridge": bridge, "ACL": acl,
        "Boxes": [{"Name": names[k], "UUID": boxes[k], "MAC": pins[k]["mac"],
                   "IPv4": pins[k]["ipv4"], "IPv6": pins[k]["ipv6"]} for k in names]}))
    root_fault = run(["sudo", "env", "GITHUB_ACTIONS=true",
                      "BOXWEAVE_S4B3_ONE_USE=20261118-disposable-root-grant",
                      "BOXWEAVE_S4B3_SCOPE=" + str(scope), "BOXWEAVE_S4B3_DB=" + db,
                      "BOXWEAVE_S4B3_ORIGINAL=" + original, "BOXWEAVE_S4B3_SCENARIO=after_acl_fault",
                      "/usr/local/bin/s5b-rc10-root-fault-fixture", "-test.v",
                      "-test.run", "^TestS4B3NativeInstalledRootGrantActualEffect$"], timeout=240)
    assert "S4_B3_REAL_ACL_PUT_UNKNOWN_ROOT_JOURNAL_ALL_OWNED_NICS_STOPPED=PASS" in root_fault
    print("S5B_RC10_ACTIVE_FORWARD_REAL_ACL_PUT_FAULT=PASS original=" + original, flush=True)
    journal = "/var/lib/boxweave-native-peer-conntrack-v1/" + original + ".json"
    receipt = json.loads(run(["sudo", "cat", journal]))
    assert receipt["status"] == "DISPATCH_UNKNOWN"
    before_hash = run(["sudo", "sha256sum", journal]).split()[0]
    assert sql("select state from operations where id='" + original + "'") == "DISPATCHING"

# Preserve the genuine original and let the installed Controller startup
# recover its own durable DISPATCHING row. No direct ledger updates.
run(["sudo", "systemctl", "stop", root_unit])
run(["sudo", "systemctl", "stop", "boxweave.service"])
run(["sudo", "systemctl", "restart", "incus.service"])
run(["sudo", "systemctl", "start", "boxweave.service"])
run(["sudo", "systemctl", "start", root_unit])
for _ in range(45):
    try:
        with open_client(actor2) as m:
            status(m)
        if subprocess.run(["sudo", "test", "-S", "/run/boxweave-native-peer-root.sock"]).returncode == 0:
            break
    except Exception:
        pass
    time.sleep(1)
else:
    raise AssertionError("controller and PRS did not reconnect")

def blocked_forward(proto, port):
    kind = socket.SOCK_STREAM if proto == "tcp" else socket.SOCK_DGRAM
    with socket.socket(socket.AF_INET, kind) as s:
        s.settimeout(3)
        try:
            s.connect(("127.0.0.1", port))
            s.sendall(b"S5B")
            reply = s.recv(3)
        except OSError:
            return
        assert reply == b"", "quarantined guest still replied"

with open_client(actor2, True) as m:
    status(m)
    assert m.operation_get(original)["state"] == "UNKNOWN"
    assert sql("select count(*) from operation_resource_claims where operation_id='" + original + "'") != "0"
    assert sql("select count(*) from peer_grants where operation_id='" + original + "' and status='CONFIRMED'") == "0"
    assert run(["sudo", "sha256sum", journal]).split()[0] == before_hash
    for k in names:
        assert native("config", "get", names[k], "volatile.uuid") == boxes[k]
        assert native("list", names[k], "-c", "s", "-f", "csv") == "STOPPED"
        assert native("config", "get", names[k], "boot.autostart") == "false"
    rows = m.call("port_forward_list", {"box_id": boxes["b"]}).structured["forwards"]
    assert len(rows) == len(before_forwards) == 2
    for row in rows:
        before = next(x for x in before_forwards if x["forward_id"] == row["forward_id"])
        for field in ("operation_id", "forward_id", "box_id", "incarnation", "native_device",
                      "host_bind", "host_port", "guest_port", "protocol", "state"):
            assert row[field] == before[field], field
        assert row["state"] == "CONFIRMED" and row["native_verified"]
        assert native("config", "device", "get", names["b"], row["native_device"], "type") == "proxy"
        blocked_forward(row["protocol"], int(row["host_port"]))
    diag = m.call("operation_recover", {"operation_id": original, "mode": "inspect"}).structured
    assert diag["diagnosis"]["disposition"] == "BLOCKED" and diag["native_retried"] is False
    result = m.call("operation_recover", {"operation_id": original, "mode": "reconcile",
                    "idempotency_key": "s5b-rc10-residual-blocked-recovery"}).structured
    assert result["recovery"]["disposition"] == "BLOCKED" and result["native_retried"] is False
    assert m.operation_get(original)["state"] == "UNKNOWN"
    assert run(["sudo", "sha256sum", journal]).split()[0] == before_hash
    assert sql("select count(*) from operations where id='" + original + "'") == "1"
    # The earlier confirmed forward claim must not imply packet availability
    # while the whole Project is protected. No Start, release or new apply.
    print("S5B_RC10_PARTIAL_PRS_FORWARD_RESTART_FENCE_NO_REPLAY=PASS original=" + original, flush=True)
if not cli_ok:
    raise AssertionError("one or more independent residual CLI checks failed")
