#!/bin/sh
# Reuse the original S2 physical quota/recorded-output test on the exact
# clean Scoped installation, with an independently selected test pool.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
test "$#" -eq 3
project="$1"; receipt="$2"; sha="$3"
work="$(dirname "$receipt")"
case "$work" in /tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$receipt" = "$work/actor.receipt"
test "$(stat -c '%a' "$receipt")" = 600
test "$(git rev-parse HEAD)" = "${N2_EXACT_HEAD:?}"
test "$project" = "user-$(id -u boxweave)"
test "$(sha256sum "$work/candidate-a" | cut -d' ' -f1)" = "$sha"
test "$(sudo sha256sum /usr/local/bin/boxweave | cut -d' ' -f1)" = "$sha"
sudo systemctl is-active --quiet boxweave.service boxweave-scoped-lifecycle.service boxweave-scoped-guard.service
runtime="$(sudo systemctl show boxweave.service boxweave-scoped-guard.service boxweave-scoped-lifecycle.service incus.service -p ActiveEnterTimestampMonotonic -p MainPID)"
mcp_quota_db=/var/lib/boxweave/ledger.sqlite
sudo python3 - "$mcp_quota_db" <<'CHECK'
import sqlite3,sys
with sqlite3.connect("file:"+sys.argv[1]+"?mode=ro",uri=True) as c:
    assert not c.execute("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')").fetchall()
CHECK
mcp_quota_token="$(sed -n 's/^token=//p' "$receipt")"
test -n "$mcp_quota_token"
mcp_quota_uuid=""
s2_mcp_quota() {
  BW_S2_MCP_QUOTA_URL="http://127.0.0.1:18650/mcp" \
    BW_S2_MCP_QUOTA_TOKEN="$mcp_quota_token" \
    BW_S2_MCP_QUOTA_UUID="$mcp_quota_uuid" \
    BW_S2_MCP_QUOTA_ORIGINAL_ID="${mcp_original:-}" \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    "$work/s2-mcp-quota-probe" "$1"
}
umask 077
. scripts/s2_mcp_quota_witness.sh
# Reused original public SDK assertions include the actual quota guest exit
# and bounded archive fallback. Verify original ledger and exact ownership
# independently before permitted ordinary MCP cleanup.
cat "$work/mcp-quota-flood.txt"
sudo python3 - "$mcp_quota_db" "$project" "$mcp_quota_uuid" "$mcp_original" "$mcp_native" <<'CHECK'
import sqlite3,sys
db,project,uuid,oid,native=sys.argv[1:]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True) as c:
    assert c.execute("SELECT layer,box_uuid,incarnation,state,native_operation_id,action FROM operations WHERE id=?", (oid,)).fetchall()==[(project,uuid,uuid,"SUCCEEDED",native,"box_exec")]
    assert c.execute("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (oid,)).fetchall()==[(1,)]
    assert c.execute("SELECT state FROM owned_boxes WHERE project=? AND uuid=?", (project,uuid)).fetchall()==[("OWNED",)]
    assert c.execute("SELECT state FROM network_bindings WHERE project=? AND box_uuid=?", (project,uuid)).fetchall()==[("PINNED",)]
    assert not c.execute("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')").fetchall()
CHECK
s2_mcp_quota cleanup
test -z "$(sudo incus list --all-projects -c n -f csv)"
test "$runtime" = "$(sudo systemctl show boxweave.service boxweave-scoped-guard.service boxweave-scoped-lifecycle.service incus.service -p ActiveEnterTimestampMonotonic -p MainPID)"
sudo python3 - "$mcp_quota_db" "$project" "$mcp_quota_uuid" "$mcp_original" "$mcp_native" <<'CHECK'
import sqlite3,sys
db,project,uuid,oid,native=sys.argv[1:]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True) as c:
    assert c.execute("SELECT layer,box_uuid,incarnation,state,native_operation_id FROM operations WHERE id=?", (oid,)).fetchall()==[(project,uuid,uuid,"SUCCEEDED",native)]
    assert c.execute("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (oid,)).fetchall()==[(1,)]
    assert c.execute("SELECT state FROM owned_boxes WHERE project=? AND uuid=?", (project,uuid)).fetchall()==[("DELETED",)]
    assert c.execute("SELECT state FROM network_bindings WHERE project=? AND box_uuid=?", (project,uuid)).fetchall()==[("RELEASED",)]
    assert not c.execute("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')").fetchall()
print("P1B_SCOPED_QUOTA_FIXED_LEDGER_AND_NATIVE_ABSENCE=PASS")
CHECK

# Second independent fixture: direct native SDK recording, MCP ownership.
. scripts/pr191_scoped_s2_direct_quota.sh
