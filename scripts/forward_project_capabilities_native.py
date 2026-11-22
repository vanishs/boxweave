"""One-use installed MCP HTTP proxy capability acceptance."""
import http.client
import ipaddress
import os
from pathlib import Path
import re
import subprocess
import sys
import time

from mcp_testkit import LoopbackMCP, MCPToolError, token_from_file

assert os.environ.get("GITHUB_ACTIONS") == "true"
assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
assert os.environ.get("GITHUB_REPOSITORY") == "vanishs/boxweave"
assert os.environ.get("FORWARD_INSTALL_MODE") == "fresh"
assert os.geteuid() != 0
work, project, host, digest = sys.argv[1:]
work = Path(work)
assert str(work).startswith("/tmp/bw-forward-install.")
assert re.fullmatch(r"user-[0-9]+", project)
assert re.fullmatch(r"[0-9a-f]{64}", digest)
assert ipaddress.IPv4Address(host).is_private
actor = token_from_file(work / "actor.secret")
unit = "boxweave.service"
dropin = "/etc/systemd/system/boxweave.service.d/99-forward-policy-ci.conf"
name = "bw-forward-native-http"
port = 23195
body = b"BoxWeave HTTP test OK" + bytes([10])

def command(*args):
    return subprocess.check_output(args, text=True, timeout=30).strip()

def status(m, mode, proxy="allow"):
    h = m.host_status()
    assert h["ready"] and h["commit"] == os.environ["GITHUB_SHA"]
    assert h["binary_sha256"] == digest
    n = h["native"]
    assert n["project"] == project and n["project_permissions"]["restricted.devices.proxy"] == proxy, n
    assert n["external_forward_policy"] == mode, n

def connect(mode, proxy="allow"):
    # Read-only connection establishment retry; never retry a mutation.
    for _ in range(30):
        try:
            m = LoopbackMCP(token=actor, timeout=12, allow_mutations=True)
            m.connect()
            status(m, mode, proxy)
            return m
        except (OSError, RuntimeError, AssertionError):
            time.sleep(1)
    raise AssertionError("installed MCP not ready")

def mutate(m, tool, key, want="SUCCEEDED", **args):
    receipt = m.call(tool, dict(args, idempotency_key="project-capability-" + key,
                                work_ref="project-capability-native-ci")).structured
    oid = receipt["operation_id"]
    assert re.fullmatch(r"[0-9a-f]{32}", oid)
    op = m.poll_operation(oid, seconds=150)
    assert op["state"] == want, (tool, oid, op)
    print("FORWARD_CAPABILITY_ORIGINAL=" + key + ":" + oid, flush=True)
    return op

def forward_list(m, uuid):
    return m.call("port_forward_list", {"box_id": uuid}).structured["forwards"]

def listener_rows():
    # Incus/Go may represent a wildcard listener as a dual-stack socket.
    # Do not filter by AF_INET: ss can render the local address as "*".
    return command("ss", "-H", "-ltn", f"sport = :{port}").splitlines()

def wait_listener(present):
    for _ in range(40):
        rows = listener_rows()
        if not present and not rows:
            return
        if present and rows:
            endpoints = [row.split()[3] for row in rows]
            wildcard = {f"0.0.0.0:{port}", f"*:{port}", f"[::]:{port}", f":::{port}"}
            if all(endpoint in wildcard for endpoint in endpoints):
                print("FORWARD_INSTALL_WILDCARD_LISTENER=" + ",".join(endpoints), flush=True)
                return
        time.sleep(0.25)
    raise AssertionError(f"listener present={present}: {listener_rows()!r}")

def http_check(address):
    conn = http.client.HTTPConnection(address, port, timeout=5)
    try:
        conn.request("GET", "/")
        response = conn.getresponse()
        assert response.status == 200 and response.read() == body
    finally:
        conn.close()

m = connect("disabled")
try:
    created = mutate(m, "box_create", "create", name=name, cpu=1, memory_mib=512,
                     disk_gib=2, start=True, internet=False)
    uuid = created["box_uuid"]
    assert command("sudo", "incus", "--project", project, "config", "get", name, "volatile.uuid") == uuid
    mutate(m, "box_wait_ready", "ready", box_id=uuid, checks=["running", "exec", "ipv4"], timeout_seconds=120)
    setup = mutate(m, "box_exec", "http", box_id=uuid, argv=["/bin/sh", "-ec",
        "install -d -m 755 /srv/bw-http; printf 'BoxWeave HTTP test OK\\n' > /srv/bw-http/index.html; "
        "systemd-run --unit=bw-http --property=User=nobody --property=NoNewPrivileges=yes "
        "/usr/bin/python3 -m http.server 8080 --bind 127.0.0.1 --directory /srv/bw-http; "
        "sleep 1; systemctl is-active --quiet bw-http"], timeout_ms=15000)
    assert m.operation_output(setup["operation_id"])["exit_code"] == 0
    args = {"box_id": uuid, "guest_port": 8080, "host_port": port,
            "host_bind": "0.0.0.0", "protocol": "tcp", "confirm_external_exposure": True}
    try:
        m.call("port_forward_create", dict(args, idempotency_key="project-capability-disabled"))
    except MCPToolError:
        pass
    else:
        raise AssertionError("disabled service policy admitted external bind")
    assert not forward_list(m, uuid)
    print("FORWARD_INSTALL_EXTERNAL_POLICY_DENIAL=PASS", flush=True)
finally:
    m.close()

# Restore the original fresh unit policy by removing only this harness-owned
# override on the disposable CI host. This is not an installer upgrade effect.
command("sudo", "rm", dropin)
command("sudo", "systemctl", "daemon-reload")
command("sudo", "systemctl", "restart", unit)
m = connect("all")
try:
    assert not listener_rows(), "test port already occupied"
    original = mutate(m, "port_forward_create", "wildcard", **args)
    assert original["native_operation_id"]
    rows = forward_list(m, uuid)
    assert len(rows) == 1 and rows[0]["state"] == "CONFIRMED" and rows[0]["native_verified"]
    row = rows[0]
    assert row["forward_id"] == original["operation_id"] and row["host_bind"] == "0.0.0.0"
    assert command("sudo", "incus", "--project", project, "config", "device", "get", name,
                   row["native_device"], "listen") == f"tcp:0.0.0.0:{port}"
    wait_listener(True)
    http_check("127.0.0.1")
    http_check(host)
    print("FORWARD_INSTALL_WILDCARD_HTTP_PACKET=PASS", flush=True)
    mutate(m, "port_forward_delete", "remove", forward_id=row["forward_id"])
    assert not forward_list(m, uuid)
    wait_listener(False)
    # Native policy denial is separate from the service policy. With no
    # active forwards, temporarily tighten the exact owned test Project.
    command("sudo", "incus", "project", "set", project, "restricted.devices.proxy=block")
    status(m, "all", "block")
    denied = mutate(m, "port_forward_create", "proxy-blocked", want="FAILED",
                    box_id=uuid, guest_port=8080, host_port=port, host_bind="127.0.0.1", protocol="tcp")
    assert denied["error_class"] == "PROJECT_PROXY_DENIED" and not denied.get("native_operation_id")
    assert not forward_list(m, uuid)
    command("sudo", "incus", "project", "set", project, "restricted.devices.proxy=allow")
    status(m, "all")
    print("FORWARD_INSTALL_PROJECT_PROXY_DENIAL=PASS", flush=True)
    mutate(m, "box_stop", "stop", box_id=uuid)
    mutate(m, "box_delete", "delete", box_id=uuid)
    assert not command("sudo", "incus", "--project", project, "list", "-c", "n", "-f", "csv")
finally:
    m.close()
