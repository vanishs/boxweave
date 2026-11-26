#!/usr/bin/env python3
"""Run the unchanged pinned old-SDK probe against the installed Scoped service.

Only the clean hosted N2 fixture is supported. All normal lifecycle effects
are submitted by the original official SDK client. The independent witness
reads the fixed ledger and native inventory; it never replays or cleans up a
failed original.
"""
import hashlib
import os
from pathlib import Path
import pwd
import re
import subprocess
import sys

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import pid, token
from pr178_n2_scoped_lifecycle import acls
from pr191_scoped_ack_loss import guard_identity, native_names
from pr191_scoped_lifecycle_recovery import rows

BIN = Path("/usr/local/bin/boxweave")
NAME = "bw-s2-legacy-compat-110e"
ACTIONS = ("create", "update", "start", "exec", "stop", "delete")


def originals(read):
    result = []
    for action in ACTIONS:
        key = hashlib.sha256(("s2-old-sdk-positive-" + action).encode()).hexdigest()
        found = read("SELECT id,tracking_id,actor_id,token_id,layer,action,state,"
                     "box_uuid,incarnation,native_operation_id FROM operations WHERE key_digest=?",
                     (key,))
        assert len(found) <= 1, action
        result.extend(found)
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


def main(project, receipt, candidate_sha, probe):
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


if __name__ == "__main__":
    main(*sys.argv[1:])
