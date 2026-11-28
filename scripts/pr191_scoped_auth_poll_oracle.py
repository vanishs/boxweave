#!/usr/bin/env python3
"""Read-only independent witness for the installed long-poll/cursor fixtures."""
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import sys

from n2_installed_lifecycle import pid
from pr178_n2_scoped_lifecycle import acls
from pr191_scoped_ack_loss import guard_identity, native_names, native_idle, wait_poststate
from pr191_scoped_lifecycle_recovery import rows
from pr191_scoped_update_transport import command

UNITS = ("boxweave.service", "boxweave-scoped-lifecycle.service", "incus.service")
GET = "lab/inflight-credential"
CURSOR = "lab/p1-installed-cursor"
KEYS = {(GET, mode): "s2-p1-inflight-" + mode + "-original-key"
        for mode in ("revoke", "expire")}
KEYS.update({(CURSOR, "p1-cursor"): "s2-p1-installed-cursor-%02d-original"})
PASS = ("S2_P1_INSTALLED_MCP_REVOKED_DURING_READ_NO_LEAK",
        "S2_P1_INSTALLED_MCP_EXPIRED_DURING_READ_NO_LEAK",
        "S2_P1_REAL_INSTALLED_MCP_STATUS_CHANGED_DURING_SIGNED_CURSOR",
        "S2_P1_INSTALLED_INFLIGHT_BEARER_REVOCATION_AND_EXPIRY",
        "P1B_SCOPED_CURSOR_EXACT_FIVE_ORIGINALS")


def digest(key):
    return hashlib.sha256(key.encode()).hexdigest()


def snapshot(read):
    # No safe_intent, credential bytes, private output or host paths.
    ops = read("SELECT id,tracking_id,layer,actor_id,token_id,action,work_ref,agent_ref,"
               "key_digest,state,native_operation_id,box_uuid,incarnation,error_class,updated_at "
               "FROM operations ORDER BY id")
    events = read("SELECT operation_id,seq,state FROM operation_events ORDER BY seq")
    claims = read("SELECT operation_id,layer,resource_key FROM operation_resource_claims ORDER BY operation_id,resource_key")
    outputs = read("SELECT operation_id,exit_code,observed_at FROM exec_outputs ORDER BY operation_id")
    audit = read("SELECT operation_id,seq,decision,evidence_digest FROM recovery_audit ORDER BY seq")
    return dict(ops=ops, events=events, claims=claims, outputs=outputs, audit=audit)


def verify(before, after, project, log):
    for marker in PASS:
        assert log.count(marker + "=PASS") == 1, marker
    new = [r for r in after["ops"] if r[0] not in {o[0] for o in before["ops"]}]
    ids = {r[0] for r in new}
    assert len(ids) == len(new) == 7
    for k, old in before.items():
        assert [r for r in after[k] if r[0] not in ids] == old, k
    assert not any(r[0] in ids for k in ("claims", "outputs", "audit") for r in after[k])
    expected = {digest(KEYS[(GET, mode)]) for mode in ("revoke", "expire")}
    expected |= {digest(KEYS[(CURSOR, "p1-cursor")] % i) for i in range(5)}
    assert {r[8] for r in new} == expected
    get_actors = set()
    cursor_actors = set()
    witness = {}
    for line in log:
        if line.startswith("P1B_SCOPED_GET_AUTH_ORIGINAL "):
            _, mode, oid, tracking = line.split()
            assert mode not in witness and mode in ("revoke", "expire")
            witness[mode] = (oid, tracking)
    assert set(witness) == {"revoke", "expire"}
    for r in new:
        oid, tracking, layer, actor, credential, action, work, agent, key, state, native, uuid, inc, err, _ = r
        assert all(re.fullmatch("[0-9a-f]{32}", v) for v in (oid, tracking, actor))
        assert layer == project and actor == credential and action == "box_exec"
        assert (state, native, uuid, inc, err) == ("CANCELLED", "", "", "", "CANCELLED_BEFORE_DISPATCH")
        assert [e[2] for e in after["events"] if e[0] == oid] == ["ADMITTED", "CANCELLED"]
        if work == GET:
            assert agent in witness and key == digest(KEYS[(GET, agent)])
            assert witness[agent] == (oid, tracking)
            get_actors.add(actor)
        else:
            assert work == CURSOR and agent == "p1-cursor"
            assert key in {digest(KEYS[(CURSOR, agent)] % i) for i in range(5)}
            cursor_actors.add(actor)
    assert len(get_actors) == 2 and len(cursor_actors) == 1 and not get_actors & cursor_actors


def main(project, receipt, candidate_sha, phase, *args):
    assert os.geteuid() == 0 and os.environ.get("GITHUB_ACTIONS") == "true"
    assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    assert project == "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    receipt = Path(receipt)
    work = receipt.parent
    assert work.parent == Path("/tmp") and work.name.startswith("bw-n2-installed.")
    assert receipt.name == "actor.receipt" and not receipt.stat().st_mode & 0o077
    assert command("git", "rev-parse", "HEAD") == os.environ["N2_EXACT_HEAD"]
    assert hashlib.sha256(Path("/usr/local/bin/boxweave").read_bytes()).hexdigest() == candidate_sha
    assert hashlib.sha256((work / "candidate-a").read_bytes()).hexdigest() == candidate_sha
    assert not native_names() and acls() == {"bw-peer-quarantine"}
    wait_poststate(lambda: native_idle(project))
    assert not rows("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','OBSERVING','UNKNOWN')")
    snap = snapshot(rows)
    runtime = [pid(u) for u in UNITS] + [guard_identity()]
    if phase == "before":
        assert not args
        assert not any(r[6] in (GET, CURSOR) for r in snap["ops"])
        print(json.dumps(dict(snapshot=snap, runtime=runtime)))
    else:
        assert phase == "after" and len(args) == 2
        prior = json.loads(Path(args[0]).read_text())
        assert prior["runtime"] == runtime
        # Normalize SQLite tuples to the serialized checkpoint representation.
        verify(prior["snapshot"], json.loads(json.dumps(snap)), project,
               Path(args[1]).read_text().splitlines())
        print("P1B_SCOPED_AUTH_POLL_CURSOR_FIXED_LEDGER_NO_NATIVE_EFFECT=PASS", flush=True)


if __name__ == "__main__":
    main(*sys.argv[1:])
