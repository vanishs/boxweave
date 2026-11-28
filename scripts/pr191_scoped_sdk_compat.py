#!/usr/bin/env python3
"""Run the unchanged pinned old-SDK probe against the installed Scoped service.

Only the clean hosted N2 fixture is supported. All normal lifecycle effects
are submitted by the original official SDK client. The independent witness
reads the fixed ledger and native inventory; it never replays or cleans up a
failed original.
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

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import pid, token
from pr178_n2_scoped_lifecycle import acls, instance
from pr177_d4_installed_recovery import wait_mcp_listener
from pr191_scoped_ack_loss import guard_identity, native_names
from pr191_scoped_lifecycle_recovery import rows

BIN = Path("/usr/local/bin/boxweave")
NAME = "bw-s2-legacy-compat-110e"
ACTIONS = ("create", "update", "start", "exec", "stop", "delete")


def original_for_key(read, key):
    digest = hashlib.sha256(key.encode()).hexdigest()
    found = read("SELECT id,tracking_id,actor_id,token_id,layer,action,state,"
                 "box_uuid,incarnation,native_operation_id FROM operations WHERE key_digest=?",
                 (digest,))
    assert len(found) <= 1, key
    return found


def originals(read):
    result = []
    for action in ACTIONS:
        result.extend(original_for_key(read, "s2-old-sdk-positive-" + action))
    return result


def verify_originals(read, project):
    result = originals(read)
    assert len(result) == len(ACTIONS)
    identity = result[0][7]
    assert identity and all(r[7] == r[8] == identity for r in result)
    assert all(r[2:5] == result[0][2:5] for r in result)
    assert all(result[0][2:5])
    assert [r[5] for r in result] == ["box_" + a for a in ACTIONS]
    assert all(r[6] == "SUCCEEDED" for r in result)
    assert all(re.fullmatch(r"[0-9a-f]{32}", r[i]) for r in result for i in (0, 1))
    assert len({r[0] for r in result}) == len(ACTIONS)
    assert len({r[1] for r in result}) == len(ACTIONS)
    assert all(r[9] for r in result) and len({r[9] for r in result}) == len(ACTIONS)
    assert read("SELECT uuid,incarnation,state,deleted_by_operation_id FROM owned_boxes "
                "WHERE project=? AND name=?", (project, NAME)) == [
                    (identity, identity, "DELETED", result[-1][0])]
    assert read("SELECT state FROM network_bindings WHERE project=? AND box_uuid=? AND nic='eth0'",
                (project, identity)) == [("RELEASED",)]
    assert read("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (result[3][0],)) == [(0,)]
    return result


OUTPUT_KEYS = (
    ("pr191-sdk-output-create-01", "box_create", None),
    ("native-s2-exec-nonzero-001", "box_exec", 7),
    ("native-s2-noisy-paginated-exec-001", "box_exec", 0),
    ("native-s2-dual-stream-over-16m-001", "box_exec", 0),
    ("native-s2-delayed-time-bounded-exec-01", "box_exec", 0),
    ("native-s2-quiet-exec-zero-001", "box_exec", 0),
    ("native-s2-continuous-output-timeout-01", "box_exec", 124),
    ("native-s2-cancel-only-read-poll-001", "box_exec", 0),
    ("pr191-sdk-output-stop-01", "box_stop", None),
    ("pr191-sdk-output-delete-01", "box_delete", None),
)


def verify_current_outputs(read, project):
    result = []
    for key, action, exit_code in OUTPUT_KEYS:
        found = original_for_key(read, key)
        assert len(found) == 1, key
        row = found[0]
        assert row[5:7] == (action, "SUCCEEDED"), key
        assert row[9], key
        if exit_code is not None:
            assert read("SELECT exit_code FROM exec_outputs WHERE operation_id=?",
                        (row[0],)) == [(exit_code,)], key
        result.append(row)
    uuid = result[0][7]
    assert uuid and all(r[7] == r[8] == uuid and r[2:5] == result[0][2:5] for r in result)
    assert all(result[0][2:5])
    assert all(re.fullmatch(r"[0-9a-f]{32}", r[i]) for r in result for i in (0, 1))
    assert len({r[0] for r in result}) == len(result)
    assert len({r[1] for r in result}) == len(result)
    assert len({r[9] for r in result}) == len(result)
    cancel = original_for_key(read, "native-s2-exec-late-cancel-01")
    assert len(cancel) == 1 and cancel[0][5:7] == ("operation_cancel", "FAILED")
    assert cancel[0][0] not in {r[0] for r in result}
    assert read("SELECT uuid,incarnation,state,deleted_by_operation_id FROM owned_boxes "
                "WHERE project=? AND name=?", (project, "bw-s2-scoped-sdk-output")) == [
                    (uuid, uuid, "DELETED", result[-1][0])]
    assert read("SELECT state FROM network_bindings WHERE project=? AND box_uuid=? AND nic='eth0'",
                (project, uuid)) == [("RELEASED",)]
    return result


SPECIAL_NAME = "bw-s2-bh018-atomic-native"
SPECIAL_KEYS = (
    ("s2-bh018-atomic-native-create-001", "box_create", None),
    ("s2-bh018-original-exit-126-001", "box_exec", 126),
    ("s2-bh018-original-exit-127-001", "box_exec", 127),
    ("s2-bh018-independent-readback-126-001", "box_exec", 0),
    ("s2-bh018-independent-readback-127-001", "box_exec", 0),
    ("s2-bh018-atomic-native-stop-001", "box_stop", None),
    ("s2-bh018-atomic-native-delete-001", "box_delete", None),
)


def verify_specials(read, project, checkpoint, complete):
    result = []
    for key, action, code in SPECIAL_KEYS[:7 if complete else 3]:
        found = original_for_key(read, key)
        assert len(found) == 1, key
        r = found[0]
        assert r[5:9] == (action, "SUCCEEDED", checkpoint["box_id"], checkpoint["box_id"]), key
        assert r[4] == project and r[2] and r[3] and r[9], key
        assert all(re.fullmatch(r"[0-9a-f]{32}", r[i]) for i in (0, 1)), key
        if code is not None:
            assert read("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (r[0],)) == [(code,)], key
        result.append(r)
    assert all(r[2:5] == result[0][2:5] for r in result)
    assert all(len({r[i] for r in result}) == len(result) for i in (0, 1, 9))
    assert result[0][0] == checkpoint["create_id"]
    assert [(r[0], r[9], code) for r, code in zip(result[1:3], (126, 127))] == [
        (c["original_id"], c["native_id"], c["exit"]) for c in checkpoint["specials"]]
    if complete:
        assert read("SELECT uuid,incarnation,state,deleted_by_operation_id FROM owned_boxes "
                    "WHERE project=? AND name=?", (project, SPECIAL_NAME)) == [
                        (checkpoint["box_id"], checkpoint["box_id"], "DELETED", result[-1][0])]
        assert read("SELECT state FROM network_bindings WHERE project=? AND box_uuid=? AND nic='eth0'",
                    (project, checkpoint["box_id"])) == [("RELEASED",)]
    return result


OAUTH_KEY = "s4-oauth-native-one-effect-001"
OAUTH_MARKERS = (
    "S4_A_REAL_INSTALLED_OAUTH_REFRESH_GRANT_REPLAY_ONCE=PASS",
    "S4_A_INSTALLED_OAUTH_REVOKED_GRANT_MANUAL_NO_ADMISSION=PASS",
    "S4_A_INDEPENDENT_GUEST_ONE_EFFECT=PASS",
)


def oauth_rows(read):
    return read("SELECT id,tracking_id,actor_id,token_id,layer,action,state,"
                "box_uuid,incarnation,native_operation_id FROM operations WHERE key_digest=? ORDER BY id",
                (hashlib.sha256(OAUTH_KEY.encode()).hexdigest(),))


def verify_oauth(read, project, uuid, primary):
    records = oauth_rows(read)
    assert len(records) == 3
    assert all(r[4:9] == (project, "box_exec", "SUCCEEDED", uuid, uuid) for r in records)
    assert all(r[2] and r[3] and r[9] for r in records)
    assert all(len({r[i] for r in records}) == 3 for i in (0, 1, 2, 3, 9))
    assert all(re.fullmatch(r"[0-9a-f]{32}", r[i]) for r in records for i in (0, 1))
    assert sum(r[2] == r[3] for r in records) == 1
    original = [r for r in records if (r[0], r[9]) == primary]
    assert len(original) == 1 and original[0][2] != original[0][3]
    for r in records:
        assert read("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (r[0],)) == [(0,)]
    return records


def main(project, receipt, candidate_sha, probe, current_probe, special_probe, oauth_probe):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    assert project == "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    assert hashlib.sha256(BIN.read_bytes()).hexdigest() == candidate_sha
    assert Path(probe).is_file()
    assert not originals(rows)
    assert not native_names() and acls() == {"bw-peer-quarantine"}
    assert rows("SELECT count(*) FROM operations WHERE state IN "
                "('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    units = ("boxweave.service", "boxweave-scoped-lifecycle.service", "incus.service")
    pids = [pid(u) for u in units]
    assert os.path.samefile(f"/proc/{pids[0]}/exe", BIN)
    guard = guard_identity()
    secret = token(receipt)
    with LoopbackMCP(token=secret, allow_mutations=False, timeout=60) as c:
        assert c.host_status()["peer_mode"] == "SCOPED_V2"
    env = dict(os.environ, BW_S2_LEGACY_MCP_URL="http://127.0.0.1:18650/mcp",
               BW_S2_LEGACY_TOKEN=secret)
    subprocess.run([probe], env=env, check=True, timeout=330)
    result = verify_originals(rows, project)
    assert not native_names() and acls() == {"bw-peer-quarantine"}
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    assert rows("SELECT count(*) FROM operations WHERE state IN "
                "('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    for r in result:
        print("P1B_SCOPED_OLD_SDK_ORIGINAL", r[5], r[0], r[9], r[1], r[7], flush=True)
    print("P1B_SCOPED_OLD_SDK_FIXED_LEDGER_NATIVE_ABSENCE=PASS", flush=True)
    assert Path(current_probe).is_file()
    for key, _, _ in OUTPUT_KEYS:
        assert not original_for_key(rows, key)
    assert not original_for_key(rows, "native-s2-exec-late-cancel-01")
    env = dict(os.environ, BW_S2_MCP_URL="http://127.0.0.1:18650/mcp",
               BW_S2_ADMIN_TOKEN=secret, BW_S2_INSTALLED_EXEC_OUTPUT_ONLY="1")
    env.pop("BW_S2_OUTPUT_WITNESS_DIR", None)
    subprocess.run([current_probe], env=env, check=True, timeout=510)
    result = verify_current_outputs(rows, project)
    assert not native_names() and acls() == {"bw-peer-quarantine"}
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    assert rows("SELECT count(*) FROM operations WHERE state IN "
                "('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    for r in result:
        print("P1B_SCOPED_CURRENT_SDK_ORIGINAL", r[5], r[0], r[9], r[1], r[7], flush=True)
    print("P1B_SCOPED_CURRENT_SDK_OUTPUT_FIXED_LEDGER=PASS", flush=True)

    # Reuse the official SDK special-exit probe after both SDK checks.
    # Keep the private checkpoint even on failure; never replay UNKNOWN.
    assert Path(special_probe).is_file()
    assert all(not original_for_key(rows, key) for key, _, _ in SPECIAL_KEYS)
    state = Path(tempfile.mkdtemp(prefix="bw-bh018-scoped.")) / "original.json"
    env = dict(os.environ, BW_S2_MCP_URL="http://127.0.0.1:18650/mcp",
               BW_S2_ADMIN_TOKEN=secret, BW_S2_BH018_CHECKPOINT=str(state))
    subprocess.run([special_probe, "start"], env=env, check=True, timeout=730)
    checkpoint = json.loads(state.read_text())
    before = verify_specials(rows, project, checkpoint, False)
    assert native_names() == [SPECIAL_NAME]
    assert instance(project, SPECIAL_NAME)["config"]["volatile.uuid"] == checkpoint["box_id"]
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    assert rows("SELECT count(*) FROM operations WHERE state IN "
                "('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    # Same exact owned guest and controller; no new public listener.
    assert Path(oauth_probe).is_file() and not oauth_rows(rows)
    count = rows("SELECT count(*) FROM operations")[0][0]
    with (state.parent / "oauth.private.log").open("x") as log:
        os.fchmod(log.fileno(), 0o600)
        subprocess.run(["runuser", "-u", "boxweave", "--", "env",
                        "-u", "INCUS_CONF", "-u", "INCUS_CONFIG_DIR",
                        "HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config",
                        "INCUS_SOCKET=/var/lib/incus/unix.socket.user",
                        "GITHUB_ACTIONS=true", "RUNNER_ENVIRONMENT=github-hosted",
                        "BW_S4_SCOPED_OAUTH=1", oauth_probe,
                        "-db", "/var/lib/boxweave/ledger.sqlite", "-project", project,
                        "-box-id", checkpoint["box_id"],
                        "-endpoint", "http://127.0.0.1:18650/mcp"],
                       stdout=log, stderr=subprocess.STDOUT, check=True, timeout=390)
    lines = (state.parent / "oauth.private.log").read_text().splitlines()
    for marker in OAUTH_MARKERS:
        assert lines.count(marker) == 1, marker
    pattern = r"S4_A_OAUTH_NATIVE_ORIGINAL_BW_ID=([0-9a-f]{32}) NATIVE_ID=([0-9a-f-]{36}) BOX_UUID=([0-9a-f-]{36})"
    matches = [m for line in lines if (m := re.fullmatch(pattern, line))]
    assert len(matches) == 1 and matches[0][3] == checkpoint["box_id"]
    primary = matches[0].group(1, 2)
    oauth_originals = verify_oauth(rows, project, checkpoint["box_id"], primary)
    assert rows("SELECT count(*) FROM operations") == [(count + 3,)]
    assert verify_specials(rows, project, checkpoint, False) == before
    assert native_names() == [SPECIAL_NAME]
    assert instance(project, SPECIAL_NAME)["config"]["volatile.uuid"] == checkpoint["box_id"]
    assert [pid(u) for u in units] == pids and guard_identity() == guard
    for marker in OAUTH_MARKERS:
        print(marker, flush=True)
    print("P1B_SCOPED_OAUTH_FIXED_LEDGER=PASS", flush=True)
    subprocess.run(["systemctl", "restart", units[0]], check=True, timeout=60)
    wait_mcp_listener()
    controller = pid(units[0])
    assert controller != pids[0] and os.path.samefile(f"/proc/{controller}/exe", BIN)
    assert [pid(u) for u in units[1:]] == pids[1:] and guard_identity() == guard
    assert verify_specials(rows, project, checkpoint, False) == before
    subprocess.run([special_probe, "resume"], env=env, check=True, timeout=730)
    after = verify_specials(rows, project, checkpoint, True)
    assert after[:3] == before
    assert verify_oauth(rows, project, checkpoint["box_id"], primary) == oauth_originals
    assert not native_names() and acls() == {"bw-peer-quarantine"}
    assert [pid(u) for u in units] == [controller, *pids[1:]] and guard_identity() == guard
    assert rows("SELECT count(*) FROM operations WHERE state IN "
                "('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')") == [(0,)]
    print("P1B_SCOPED_SPECIAL_EXEC_RESTART_FIXED_LEDGER=PASS", flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
