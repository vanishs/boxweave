"""Actual published RC10 manual Create/Delete recovery leaves, hosted disposable only."""
import json
import os
import pathlib
import re
import subprocess
import sys

from mcp_testkit import LoopbackMCP, MCPToolError, token_from_file

assert os.environ.get("GITHUB_ACTIONS") == "true"
assert os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
assert os.environ.get("GITHUB_REPOSITORY") == "vanishs/boxweave"
assert os.environ.get("GITHUB_REF") == "refs/heads/docs/s5b-rc10-closeout-20261121"
assert os.geteuid() != 0
work, project, bridge, acl, binary_sha = sys.argv[1:]
work = pathlib.Path(work)
assert str(work).startswith("/tmp/bw-v1-s25-lab.")
assert re.fullmatch(r"user-[0-9]+", project)
assert binary_sha == "f67e769dbfeade39023f655b0a182993bf2459f645d182e52d04bba85093bfa6"
bw = "/usr/local/bin/boxweave"
db = "/var/lib/boxweave/ledger.sqlite"
actor = token_from_file(work / "private/admin1.secret")

def run(args, timeout=90):
    return subprocess.check_output(args, text=True, timeout=timeout).strip()

def native(*args):
    return run(["sudo", "incus", "--project", project, *args])

def sql(query):
    # Only this fresh disposable CI ledger; read unless installing/removing
    # the existing narrowly labeled test-only finalization fault pattern.
    return run(["sudo", "sqlite3", db, query])

def original(m, action, key, want, **args):
    args.update(idempotency_key="rc10-manual-cli-" + key,
                work_ref="rc10/manual-cli-fault" if want == "UNKNOWN" else "rc10/manual-cli-positive")
    r = m.call(action, args).structured
    oid = r["operation_id"]
    assert re.fullmatch(r"[0-9a-f]{32}", oid) and not r.get("duplicate")
    result = m.poll_operation(oid, seconds=120)
    assert result["state"] == want and result["operation_id"] == oid
    print("S5B_RC10_SPECIALIZED_ORIGINAL=" + action + ":" + oid + ":" +
          str(result.get("native_operation_id", "")) + ":" + want, flush=True)
    return result

def recover(oid, leaf, *args, success=True):
    cp = subprocess.run(["sudo", bw, "recover", "-project", project,
                         "-operation", oid, "-reviewer", "rc10-ci-reviewer",
                         "-action", leaf, "-approve-local", *args],
                        capture_output=True, text=True, timeout=40)
    if (cp.returncode == 0) != success:
        # Report only fixed diagnostic categories, never raw process output.
        if leaf.endswith("-inspect"):
            categories = {
                "installed identity unverified": "INSTALLED_IDENTITY_UNVERIFIED",
                "restricted recovery worker failed": "RESTRICTED_WORKER_FAILED",
                "permission denied": "PERMISSION_DENIED",
                "native restricted preflight": "NATIVE_PREFLIGHT_FAILED",
                "not found": "NOT_FOUND",
                "invalid state": "INVALID_STATE",
                "unreviewable original Delete": "UNREVIEWABLE_DELETE",
                "exact installed SQLite path": "INSTALLED_PATH_MISMATCH",
                "manual UNKNOWN review requires": "LOCAL_APPROVAL_REQUIRED",
                "original native Create identity not proven": "CREATE_IDENTITY_UNPROVEN",
                "native identity changed": "NATIVE_IDENTITY_CHANGED",
                "context deadline exceeded": "DEADLINE_EXCEEDED",
                "database is locked": "DATABASE_LOCKED",
                "unexpected sqlite trigger/view": "FOREIGN_SCHEMA_REFUSED",
                "invalid operation transition": "INVALID_TRANSITION",
                "native operation or postcondition not independently verified": "NATIVE_UNVERIFIED",
                "unknown action": "UNKNOWN_ACTION",
            }
            message = cp.stderr.lower()
            matches = [label for phrase, label in categories.items() if phrase.lower() in message]
            print("S5B_RC10_INSPECT_ERROR_CLASSES=" + ",".join(matches or ["UNCLASSIFIED"]), flush=True)
        raise AssertionError(("unexpected CLI result", leaf, cp.returncode))
    return json.loads(cp.stdout) if success else None

def audit_challenge(oid, leaf):
    value = recover(oid, leaf)
    assert value["operation_id"] == oid and value["expires_in_seconds"] == 300
    assert re.fullmatch(r"[0-9a-f]{64}", value["challenge"])
    return value

assert not native("list", "-c", "n", "-f", "csv")
assert sql("SELECT count(*) FROM operations") == "0"
assert sql("SELECT count(*) FROM owned_boxes") == "0"
with LoopbackMCP(token=actor, timeout=15, allow_mutations=True) as m:
    h = m.host_status()
    assert h["version"] == "1.0.0-rc.10" and h["commit"] == "eb021d3985c787c517cd7aee3efaf433d38f0393"
    assert h["binary_sha256"] == binary_sha and h["ready"]
    name = "s5b-rc10-cli-delete"
    created = original(m, "box_create", "delete-precondition", "SUCCEEDED",
                       name=name, cpu=1, memory_mib=512, disk_gib=2, start=False, internet=False)
    uuid = created["box_uuid"]
    assert re.fullmatch(r"[0-9a-f-]{36}", uuid)
    assert native("config", "get", name, "volatile.uuid") == uuid
    try:
        # Same production finalization fault pattern as S5A2. Do not change
        # receipt/native ID, ownership or any existing audit row.
        sql("CREATE TRIGGER s5b_rc10_only_cli_fault BEFORE UPDATE OF state ON operations "
            "WHEN OLD.state IN ('VERIFYING','DISPATCHING') AND NEW.state='SUCCEEDED' "
            "AND OLD.work_ref='rc10/manual-cli-fault' BEGIN "
            "SELECT RAISE(ABORT,'S5B_RC10_ONE_USE_FINALIZATION_FAULT'); END")
        op = original(m, "box_delete", "delete-fault", "UNKNOWN", box_id=uuid)
        oid = op["operation_id"]
        native_id = op["native_operation_id"]
        assert re.fullmatch(r"[0-9a-f-]{36}", native_id)
        assert not native("list", name, "-c", "n", "-f", "csv")
        # The installed recovery CLI opens a new Store and validates its
        # exact schema. Remove ONLY the temporary CI fault fixture before
        # opening the recovery CLI, as in the existing S5A2 native test.
        sql("DROP TRIGGER s5b_rc10_only_cli_fault")
        assert sql("SELECT count(*) FROM sqlite_master WHERE type IN ('trigger','view')") == "0"
        assert m.operation_get(oid)["state"] == "UNKNOWN"
        print("S5B_RC10_DELETE_FIXTURE_SCHEMA_RESTORED=PASS", flush=True)
        inspected = recover(oid, "delete-inspect")
        assert inspected["native"]["class"] == "ORIGINAL_TERMINAL_DELETE_VERIFIED"
        assert inspected["native"]["native_operation_id"] == native_id
        challenge = audit_challenge(oid, "delete-challenge")["challenge"]
        # Exact wrong-UUID refusal before positive confirm; no native retry.
        args = ["-challenge", challenge, "-confirm-uuid", uuid,
                "-reason", "independently verified native Delete and absence in disposable CI",
                "-acknowledge-uncertain-native"]
        wrong = list(args)
        wrong[3] = "00000000-0000-0000-0000-000000000000"
        recover(oid, "delete-confirm", *wrong, success=False)
        settled = recover(oid, "delete-confirm", *args)
        assert settled["operation_id"] == oid and settled["state"] == "SUCCEEDED"
        assert settled["native_retried"] is False
        assert sql("SELECT count(*) FROM operation_events WHERE operation_id='" + oid + "' AND state='UNKNOWN'") == "1"
        assert sql("SELECT state||':'||deleted_by_operation_id FROM owned_boxes WHERE uuid='" + uuid + "'") == "DELETED:" + oid
        assert m.operation_get(oid)["native_operation_id"] == native_id
        recover(oid, "delete-confirm", *args, success=False)
        print("S5B_RC10_CLI_DELETE_INSPECT_CHALLENGE_CONFIRM=PASS original=" + oid, flush=True)

        name = "s5b-rc10-cli-create"
        # Same production finalization fault pattern as S5A2. Do not change
        # receipt/native ID, ownership or any existing audit row.
        sql("CREATE TRIGGER s5b_rc10_only_cli_fault BEFORE UPDATE OF state ON operations "
            "WHEN OLD.state IN ('VERIFYING','DISPATCHING') AND NEW.state='SUCCEEDED' "
            "AND OLD.work_ref='rc10/manual-cli-fault' BEGIN "
            "SELECT RAISE(ABORT,'S5B_RC10_ONE_USE_FINALIZATION_FAULT'); END")
        op = original(m, "box_create", "create-fault", "UNKNOWN",
                      name=name, cpu=1, memory_mib=512, disk_gib=2, start=False, internet=False)
        oid = op["operation_id"]
        assert not op.get("box_uuid")
        uuid = native("config", "get", name, "volatile.uuid")
        assert re.fullmatch(r"[0-9a-f-]{36}", uuid)
        assert native("config", "get", name, "user.boxweave.create_operation") == oid
        sql("DROP TRIGGER s5b_rc10_only_cli_fault")
        assert sql("SELECT count(*) FROM sqlite_master WHERE type IN ('trigger','view')") == "0"
        assert m.operation_get(oid)["state"] == "UNKNOWN"
        print("S5B_RC10_CREATE_FIXTURE_SCHEMA_RESTORED=PASS", flush=True)
        inspected = recover(oid, "create-inspect")
        assert inspected["native_state"].lower() == "stopped"
        c = audit_challenge(oid, "create-challenge")
        assert c["native_uuid"] == uuid
        args = ["-challenge", c["challenge"], "-confirm-uuid", uuid,
                "-reason", "independently verified exact original native Create marker in disposable CI",
                "-acknowledge-uncertain-native"]
        wrong = list(args)
        wrong[3] = "00000000-0000-0000-0000-000000000000"
        recover(oid, "create-bind", *wrong, success=False)
        bound = recover(oid, "create-bind", *args)
        assert bound["operation_id"] == oid and bound["native_uuid"] == uuid
        assert bound["original_state"] == "UNKNOWN" and bound["claim_released"] is False
        assert bound["native_retried"] is False
        assert m.operation_get(oid)["state"] == "UNKNOWN"
        assert sql("SELECT count(*) FROM recovery_audit WHERE operation_id='" + oid + "' AND decision='RELEASED'") == "0"
        assert native("config", "get", name, "volatile.uuid") == uuid
        assert native("list", name, "-c", "s", "-f", "csv") == "STOPPED"
        try:
            m.call("box_start", {"box_id": uuid, "idempotency_key": "rc10-must-deny-held-create"})
        except MCPToolError:
            pass
        else:
            raise AssertionError("identity binding released original claim")
        recover(oid, "create-bind", *args, success=False)
        print("S5B_RC10_CLI_CREATE_INSPECT_CHALLENGE_BIND=PASS original=" + oid, flush=True)
    finally:
        sql("DROP TRIGGER IF EXISTS s5b_rc10_only_cli_fault")
# The ephemeral CI VM is teardown; preserve original UNKNOWN and exact
# native fixture until job finishes. No release or arbitrary cleanup.
print("S5B_RC10_SPECIALIZED_RECOVERY_CLI_SIX_LEAVES=PASS", flush=True)
