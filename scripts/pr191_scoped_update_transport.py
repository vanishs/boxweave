#!/usr/bin/env python3
"""One-use clean installed Scoped native Update transport acceptance.

Reuse the original restricted S2 fault proxy and official SDK. The test
controller is the exact installed executable and existing ledger, with only
its private test transport and loopback port changed while the real unit is
stopped. Root guard/helper and Incus retain their fixed identities. Preserve
all UNKNOWN originals, never use a second key to dispatch or clean them.
"""
from contextlib import contextmanager
import hashlib
import http.client
import json
import os
from pathlib import Path
import pwd
import re
import signal
import subprocess
import sys
import tempfile
import time

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, pid, token
from pr177_d4_installed_recovery import wait_mcp_listener
from pr178_n2_scoped_lifecycle import acls, binding, instance, verify
from pr191_scoped_ack_loss import (ARGS, CTL, QUARANTINE, guard_identity,
                                   native_idle, native_names, snapshot,
                                   wait_poststate, work_snapshot)
from pr191_scoped_lifecycle_recovery import ROOT, rows

BIN = Path("/usr/local/bin/boxweave")
DB = "/var/lib/boxweave/ledger.sqlite"
SOCKET = "/var/lib/incus/unix.socket.user"
FIXTURES = {
    "etag": "bw-s2-etag-native-110e",
    "ack": "bw-s2-update-ack-native-110f",
    "postread": "bw-s2-update-postread-native-110g",
}


def command(*args):
    return subprocess.check_output(args, text=True, timeout=60).strip()


def http_status(port):
    conn = http.client.HTTPConnection("127.0.0.1", port, timeout=2)
    try:
        conn.request("POST", "/mcp", body="{}", headers={"Content-Type": "application/json"})
        reply = conn.getresponse()
        reply.read(4096)
        return reply.status
    except (OSError, http.client.HTTPException):
        return None
    finally:
        conn.close()


def wait_port(port, ready, child=None):
    for _ in range(60):
        if child is not None:
            assert child.poll() is None, "supervised test process exited"
        seen = http_status(port)
        if (ready and seen == 401) or (not ready and seen is None):
            return
        time.sleep(.5)
    raise AssertionError("test listener state unverified")


def restricted(*args):
    return ["runuser", "-u", "boxweave", "--", "env", "-u", "INCUS_CONF",
            "-u", "INCUS_CONFIG_DIR", "-u", "XDG_CONFIG_HOME",
            "HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config",
            "GITHUB_ACTIONS=true", "RUNNER_ENVIRONMENT=github-hosted", *args]


@contextmanager
def supervised(argv, log):
    with log.open("xb") as output:
        child = subprocess.Popen(argv, stdout=output, stderr=subprocess.STDOUT,
                                 start_new_session=True)
        try:
            yield child
        finally:
            # This is only the newly created private test process group, not
            # a PID search, a shared daemon, or an original native Operation.
            if child.poll() is None:
                os.killpg(child.pid, signal.SIGTERM)
                child.wait(timeout=30)


def legacy_unknown_refusals(project, oid, uuid):
    # Only a fresh installed Scoped UNKNOWN fixture. The historical generic
    # claim-release path is deliberately unsupported here. No valid challenge
    # is obtained, and no mutation is replayed.
    original = snapshot(oid)
    assert original[0][0][2]=="UNKNOWN" and original[1] and not original[3]
    history = rows("SELECT id,state,tracking_id,updated_at FROM operations ORDER BY id")
    base = [str(BIN),"recover","-db",DB,"-project",project,
            "-incus-socket",SOCKET,"-service-user","boxweave",
            "-operation",oid,"-reviewer","p1b-scoped-refusal","-approve-local"]
    for action in ("inspect","challenge","release"):
        args = base+["-action",action]
        if action=="release":
            args += ["-challenge","0"*64,"-confirm-uuid",uuid,
                     "-reason","Obsolete route negative fixture only.","-acknowledge-uncertain-native"]
        result=subprocess.run(args,capture_output=True,text=True,timeout=40)
        assert result.returncode!=0 and "scoped original requires the reviewed recovery route" in result.stderr, action
        assert snapshot(oid)==original
        assert rows("SELECT id,state,tracking_id,updated_at FROM operations ORDER BY id")==history
    print("P1B_SCOPED_LEGACY_UNKNOWN_RELEASE_REFUSED_ORIGINAL_RETAINED=PASS",
          oid,uuid,flush=True)


def main(project, receipt, candidate_sha):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    account = pwd.getpwnam("boxweave")
    assert project == "user-" + str(account.pw_uid)
    receipt = Path(receipt)
    work = receipt.parent
    assert work.parent == Path("/tmp") and work.name.startswith("bw-n2-installed.")
    assert receipt.name == "actor.receipt" and receipt.stat().st_mode & 0o077 == 0
    assert command("git", "rev-parse", "HEAD") == os.environ["N2_EXACT_HEAD"]
    assert hashlib.sha256(BIN.read_bytes()).hexdigest() == candidate_sha
    assert hashlib.sha256((work / "candidate-a").read_bytes()).hexdigest() == candidate_sha
    assert not native_names() and acls() == {QUARANTINE}
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')")
    assert http_status(18650) == 401 and http_status(18652) is None
    units = ("incus.service", "boxweave-scoped-lifecycle.service")
    pids = [pid(u) for u in units]
    guard = guard_identity()
    unit_before = Path("/etc/systemd/system/boxweave.service").read_bytes()
    fixtures = {}
    credential = token(receipt)
    # All normal Create operations complete BEFORE introducing the fault
    # transport. The fault proxy never handles a Create, Delete or Peer.
    with LoopbackMCP(token=credential, allow_mutations=True, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
        for mode, name in FIXTURES.items():
            created = op(c, "box_create", dict(ARGS, name=name, start=False),
                         "scoped-native-transport-seed-" + mode)
            uuid = created["box_uuid"]
            acl = verify(project, name, uuid, "true")
            raw = instance(project, name)
            assert raw["status"] == "Stopped" and raw["config"]["limits.cpu"] == "1"
            fixtures[mode] = (name, uuid, acl, created["operation_id"],
                              raw["expanded_devices"]["eth0"], binding(uuid),
                              rows("SELECT * FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)))
    assert native_names() == sorted(FIXTURES.values())
    assert acls() == {QUARANTINE} | {v[2] for v in fixtures.values()}
    assert all(len(v[-1]) == 2 for v in fixtures.values())
    retained = {}

    def native_proof(mode):
        name, uuid, acl, create, nic, pin, fences = fixtures[mode]
        raw = instance(project, name)
        assert raw["config"]["volatile.uuid"] == uuid
        assert raw["config"]["user.boxweave.create_operation"] == create
        assert raw["status"] == "Stopped"
        assert raw["config"]["limits.cpu"] == ("1" if mode == "etag" else "2")
        if mode == "etag":
            assert raw["config"]["user.s2-etag-race"] == "native-proxy-race-witness"
        assert raw["expanded_devices"]["eth0"] == nic
        assert verify(project, name, uuid, "true") == acl
        assert binding(uuid) == pin
        assert rows("SELECT * FROM network_address_fences WHERE box_uuid=? ORDER BY family", (uuid,)) == fences
        assert rows("SELECT uuid,incarnation,state FROM owned_boxes WHERE uuid=?", (uuid,)) == [(uuid, uuid, "OWNED")]
        assert native_names() == sorted(FIXTURES.values()) and native_idle(project)
        assert acls() == {QUARANTINE} | {v[2] for v in fixtures.values()}
        assert [pid(u) for u in units] == pids and guard_identity() == guard

    def original_proof(mode, cp, marker):
        name, uuid, *_ = fixtures[mode]
        oid = cp["OriginalID"]
        tracking = cp["TrackingID"]
        assert cp["BoxID"] == uuid and cp["Mode"] == mode
        assert re.fullmatch(r"[0-9a-f]{32}", oid) and re.fullmatch(r"[0-9a-f]{32}", tracking)
        marks = marker.read_text().splitlines()
        if mode == "etag":
            assert marks == ["ETAG_RACE_NATIVE_CHANGED_BEFORE_STALE_PUT", "ETAG_RACE_ORIGINAL_STALE_PUT_REFUSED"], marks
            native_id = ""
        else:
            assert re.fullmatch(r"SCOPED_UPDATE_NATIVE_ID=[0-9a-f-]{36}", marks[0]), marks
            actual = marks[0].split("=", 1)[1]
            if mode == "ack":
                assert marks[1:] == ["UPDATE_ACK_NATIVE_ACCEPTED_ONCE_AND_ACK_SUPPRESSED"], marks
                native_id = ""
            else:
                assert marks[1:] == ["POSTREAD_NATIVE_UPDATE_202_ACK_DELIVERED", "POSTREAD_AFTER_NATIVE_ACK_INSTANCE_GET_REFUSED_ONCE"], marks
                native_id = actual
            print("P1B_SCOPED_UPDATE_TRANSPORT_UPSTREAM", mode, oid, actual, flush=True)
        assert cp["NativeID"] == native_id
        snap = snapshot(oid)
        cls = "POSTCONDITION_UNVERIFIED" if mode == "postread" else "NATIVE_OUTCOME_AMBIGUOUS"
        assert snap[0] == [(oid, tracking, "UNKNOWN", native_id, uuid, uuid, snap[0][0][6], cls)]
        assert snap[1] == [(project, "box:" + name)] and not snap[3]
        assert not any(r[1] == "SUCCEEDED" for r in snap[2])
        assert sum(r[1] == "VERIFYING" for r in snap[2]) == (1 if mode == "postread" else 0)
        assert not os.path.lexists(ROOT / ("op-" + oid))
        assert rows("SELECT id FROM operations WHERE work_ref=? AND action='box_update'", ("p1b/scoped-update-transport-"+mode,)) == [(oid,)]
        return snap

    for mode, name in FIXTURES.items():
        uuid = fixtures[mode][1]
        lab = Path(tempfile.mkdtemp(prefix="bw-v1-s2-ghci.", dir="/tmp"))
        lab.chmod(0o711)
        ledger = lab / "ledger"
        ledger.mkdir(mode=0o700)
        os.chown(ledger, account.pw_uid, account.pw_gid)
        proxy = ledger / "ack-proxy.sock"
        marker = ledger / "native-ack.txt"
        marker.touch(mode=0o600, exist_ok=False)
        os.chown(marker, account.pw_uid, account.pw_gid)
        checkpoint = lab / "scoped-update-original.json"
        previous = pid(CTL)
        command("systemctl", "stop", CTL)
        assert command("systemctl", "show", CTL, "-p", "MainPID", "--value") == "0"
        wait_port(18650, False)
        assert http_status(18652) is None
        env = dict(os.environ, BW_S2_SCOPED_UPDATE="1", BW_S2_SCOPED_UPDATE_MODE=mode,
                   BW_S2_SCOPED_UPDATE_UUID=uuid, BW_S2_SCOPED_UPDATE_CHECKPOINT=str(checkpoint),
                   BW_S2_MCP_URL="http://127.0.0.1:18652/mcp", BW_S2_ADMIN_TOKEN=credential)
        def sdk(action):
            subprocess.run([str(work / "s2-ack-loss-probe"), action], env=env,
                           check=True, timeout=210)
        with supervised(restricted(str(work / "s2-ack-loss-proxy"), "-listen", str(proxy),
                        "-upstream", SOCKET, "-marker", str(marker), "-project", project,
                        "-scoped-update-fixture", name, "-scoped-update-uuid", uuid),
                        lab / "proxy.log") as proxy_child:
            for _ in range(60):
                assert proxy_child.poll() is None
                if proxy.is_socket():
                    break
                time.sleep(.5)
            assert proxy.is_socket()
            # Existing installed DB, exact installed binary, same restricted
            # account. No alternate root helper or journal, no service file edit.
            with supervised(restricted(str(BIN), "serve", "-db", DB, "-project", project,
                            "-incus-socket", str(proxy), "-listen", "127.0.0.1:18652",
                            "-scoped-peer-fresh"), lab / "controller.log") as ctl:
                wait_port(18652, True, ctl)
                with LoopbackMCP("http://127.0.0.1:18652/mcp", token=credential, timeout=60) as c:
                    assert c.host_status()["peer_mode"] == "SCOPED_V2"
                sdk("scoped_update")
                cp = json.loads(checkpoint.read_text())
                wait_poststate(lambda: native_idle(project) and instance(project, name)["config"]["limits.cpu"] == ("1" if mode == "etag" else "2"))
                native_proof(mode)
                before = original_proof(mode, cp, marker)
                before_work = work_snapshot("p1b/scoped-update-transport-" + mode)
                marks = marker.read_bytes()
            wait_port(18652, False)
        # Restore the unchanged actual installed unit, then query/replay
        # ONLY the saved original and expected denied keys using a new SDK.
        command("systemctl", "start", CTL)
        wait_mcp_listener()
        assert pid(CTL) != previous
        assert Path("/etc/systemd/system/boxweave.service").read_bytes() == unit_before
        env["BW_S2_MCP_URL"] = "http://127.0.0.1:18650/mcp"
        sdk("scoped_resume")
        assert original_proof(mode, cp, marker) == before
        assert work_snapshot("p1b/scoped-update-transport-"+mode) == before_work
        assert marker.read_bytes() == marks
        native_proof(mode)
        if mode=="postread":
            legacy_unknown_refusals(project,cp["OriginalID"],uuid)
            assert original_proof(mode,cp,marker)==before
            native_proof(mode)
        retained[mode] = (cp["OriginalID"], before)
        for prior, (oid, saved) in retained.items():
            assert snapshot(oid) == saved
            native_proof(prior)
        print("P1B_SCOPED_UPDATE_TRANSPORT=PASS", mode, flush=True)
    # No administrative cleanup, no unlock, no new native operation against
    # any unresolved original. The whole one-use runner retains evidence.
    assert len(retained) == 3


if __name__ == "__main__":
    main(*sys.argv[1:])
