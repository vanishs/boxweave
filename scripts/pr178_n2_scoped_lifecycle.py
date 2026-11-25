#!/usr/bin/env python3
"""Installed scoped N2 acceptance; run only inside a disposable clean lab."""
import json
import sqlite3
import subprocess
import sys

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, token
from pr188_h4a_installed import busy

DB = "file:/var/lib/boxweave/ledger.sqlite?mode=ro"


def incus(*args):
    return subprocess.check_output(["incus", *args], text=True)


def binding(uuid):
    with sqlite3.connect(DB, uri=True) as db:
        db.row_factory = sqlite3.Row
        row = db.execute("SELECT * FROM network_bindings WHERE box_uuid=?", (uuid,)).fetchone()
        assert row is not None, uuid
        return dict(row)


def instance(project, name):
    return json.loads(incus("query", "/1.0/instances/" + name + "?project=" + project))


def acls():
    return {x["name"] for x in json.loads(incus("network", "acl", "list", "--format=json"))}


def verify(project, name, uuid, boot):
    row = binding(uuid)
    assert row["state"] == "PINNED", row
    assert row["acl"].startswith("bw-peer-") and row["acl"] != "bw-peer-quarantine", row
    raw = instance(project, name)
    assert raw["config"]["volatile.uuid"] == uuid
    assert raw["config"]["boot.autostart"] == boot
    assert raw["expanded_devices"]["eth0"]["security.acls"] == row["acl"]
    return row["acl"]


def main(project, receipt):
    guard = "boxweave-scoped-guard.service"
    for prop in ("Requires", "After"):
        deps = subprocess.check_output(["systemctl", "show", "-p", prop, "--value", "incus.service"], text=True)
        assert guard in deps.split()
    subprocess.run(["/usr/local/bin/boxweave", "native-scoped-guard-boot"], check=True)
    print("PR178_N2_BOOT_ORDER=PASS", flush=True)
    with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(receipt),
                     allow_mutations=True, timeout=60) as c:
        assert c.host_status()["ready"]
        args = {"cpu": 1, "memory_mib": 256, "disk_gib": 2,
                "image": "local:boxweave-ubuntu-24.04", "internet": True, "start": True}
        a = op(c, "box_create", dict(args, name="scoped-alpha"), "scoped-a")["box_uuid"]
        b = op(c, "box_create", dict(args, name="scoped-beta", **{"boot.autostart": False}), "scoped-b")["box_uuid"]
        aa = verify(project, "scoped-alpha", a, "true")
        ba = verify(project, "scoped-beta", b, "false")
        assert aa != ba and {aa, ba, "bw-peer-quarantine"} == acls()
        print("PR178_N2_MCP_SCOPED_CREATE=PASS", flush=True)
        busy(c, a, "SCOPED_V2")
        op(c, "box_stop", {"box_id": a}, "scoped-a-stop")
        op(c, "box_delete", {"box_id": a}, "scoped-a-delete")
        assert aa not in acls()
        assert instance(project, "scoped-beta")["status"] == "Running"
        assert verify(project, "scoped-beta", b, "false") == ba
        assert acls() == {ba, "bw-peer-quarantine"}
        print("PR178_N2_MCP_SCOPED_DELETE=PASS", flush=True)
        op(c, "box_stop", {"box_id": b}, "scoped-b-stop")
        op(c, "box_delete", {"box_id": b}, "scoped-b-delete")
        assert acls() == {"bw-peer-quarantine"}
        assert not incus("list", "--all-projects", "-c", "n", "-f", "csv").strip()
        old = [binding(a), binding(b)]
        d = op(c, "box_create", dict(args, name="scoped-after-delete", start=False,
                                     **{"boot.autostart": False}), "scoped-history-create")["box_uuid"]
        da = verify(project, "scoped-after-delete", d, "false")
        assert da not in {aa, ba} and instance(project, "scoped-after-delete")["status"] == "Stopped"
        for prior in old:
            assert binding(d)["ipv4"] != prior["ipv4"] and binding(d)["ipv6"] != prior["ipv6"]
        op(c, "box_delete", {"box_id": d}, "scoped-history-delete")
        assert acls() == {"bw-peer-quarantine"}
        print("PR178_N2_MCP_LIFECYCLE=PASS", flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
