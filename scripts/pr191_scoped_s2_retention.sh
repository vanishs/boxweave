#!/bin/sh
# Installed Scoped retention and the independent private SQLite FULL gate.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
test "$#" -eq 3
project="$1"; receipt="$2"; sha="$3"
test "$project" = "user-$(id -u boxweave)"
n2="$(dirname "$receipt")"
case "$n2" in /tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$receipt" = "$n2/actor.receipt"
test "$(stat -c '%a' "$receipt")" = 600
test "$(sha256sum "$n2/candidate-a" | cut -d' ' -f1)" = "$sha"
test "$(sudo sha256sum /usr/local/bin/boxweave | cut -d' ' -f1)" = "$sha"
test "$(git rev-parse HEAD)" = "${N2_EXACT_HEAD:?}"
test -z "$(sudo incus list --all-projects -c n -f csv)"
sudo systemctl is-active --quiet boxweave.service boxweave-scoped-lifecycle.service boxweave-scoped-guard.service
work="$(mktemp -d -p /tmp bw-v1-s2-ghci.XXXXXX)"
chmod 0711 "$work"
umask 077
db=/var/lib/boxweave/ledger.sqlite
socket=/var/lib/incus/unix.socket.user
name=bw-s2-mcp-quota-110i
# Original independent SQLite test uses only a new, separate filesystem
# and synthetic native ID. No live service or existing ledger is filled.
sh scripts/s2_sqlite_enospc_lab.sh "$work" "$project" /usr/local/bin/boxweave "$n2/s2-sqlite-fsync-probe"
runtime="$(sudo systemctl show boxweave-scoped-guard.service boxweave-scoped-lifecycle.service incus.service -p ActiveEnterTimestampMonotonic -p MainPID)"
secret="$(sed -n 's/^token=//p' "$receipt")"
test -n "$secret"
uuid=""
quota() {
  GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted BW_S2_MCP_QUOTA_URL=http://127.0.0.1:18650/mcp \
    BW_S2_MCP_QUOTA_TOKEN="$secret" BW_S2_MCP_QUOTA_UUID="$uuid" "$n2/s2-mcp-quota-probe" "$1"
}
quota create > "$work/box.txt"
uuid="$(sed -n 's/^S2_MCP_QUOTA_BOX_UUID=//p' "$work/box.txt")"
printf '%s\n' "$uuid" | grep -Eq '^[0-9a-f-]{36}$'
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
test "$(sudo incus list --all-projects -c n -f csv)" = "$name"
spool="/var/lib/incus/containers/${project}_$name/exec-output"
original=""; native=""
gc() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted BW_S2_INSTALLED_SCOPED_GC=1 BW_S2_GC_INSTALLED_WORK="$work" \
    BW_S2_GC_DB="$db" BW_S2_GC_TOKEN="$secret" BW_S2_GC_PROJECT="$project" BW_S2_GC_UUID="$uuid" \
    BW_S2_GC_OPERATION="$original" BW_S2_GC_NATIVE="$native" BW_S2_GC_SOCKET="$socket" "$n2/s2-exec-gc-probe" "$1"
}
read_ids() {
  original="$(sed -n 's/^S2_GC_ORIGINAL_ID=//p' "$1")"
  native="$(sed -n 's/^S2_GC_NATIVE_ID=//p' "$1")"
  printf '%s\n' "$original" | grep -Eq '^[0-9a-f]{32}$'
  printf '%s\n' "$native" | grep -Eq '^[0-9a-f-]{36}$'
  grep -Fxq S2_GC_PUBLIC_FRESH_BOTH_STREAMS=PASS "$1"
  for stream in stdout stderr; do sudo test -f "$spool/exec_$native.$stream"; done
  cat "$1"
}
offline() {
  sudo systemctl stop boxweave.service
  test "$(sudo systemctl show boxweave.service -p ActiveState --value)" = inactive
  test "$(sudo systemctl show boxweave.service -p MainPID --value)" = 0
}
online() {
  sudo systemctl start boxweave.service
  for n in $(seq 1 40); do
    if test "$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 http://127.0.0.1:18650/mcp || :)" = 401; then return 0; fi
    sleep 1
  done
  return 1
}
gc create > "$work/fresh.txt"
read_ids "$work/fresh.txt"
first_original="$original"; first_native="$native"
offline
gc age
gc sweep
for stream in stdout stderr; do sudo test ! -f "$spool/exec_$native.$stream"; done
echo S2_GC_INDEPENDENT_PHYSICAL_TWO_NATIVE_LOGS_REMOVED=PASS
online
gc check
gc create-loss > "$work/loss.txt"
read_ids "$work/loss.txt"
test "$first_original" != "$original" && test "$first_native" != "$native"
offline
gc age
# Exact existing restricted test-only Unix proxy is used only by the
# standalone offline sweep, never configured into the installed controller.
mkdir -m 0700 "$work/ledger"
sudo chown boxweave:incus "$work/ledger"
proxy="$work/ledger/gc-delete-proxy.sock"
marker="$work/ledger/gc-native-delete.txt"
sudo runuser -u boxweave -- env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  "$n2/s2-gc-delete-proxy" -listen "$proxy" -upstream "$socket" -marker "$marker" -project "$project" -native-id "$native" > "$work/proxy.log" 2>&1 &
proxy_pid=$!
for n in $(seq 1 30); do sudo test -S "$proxy" && break; sleep 1; done
sudo test -S "$proxy"
real_socket="$socket"; socket="$proxy"
gc lossy-sweep
socket="$real_socket"
sudo grep -Fxq GC_EXACT_NATIVE_STDOUT_DELETE_ACCEPTED_ACK_SUPPRESSED "$marker"
sudo test ! -e "$marker.unexpected"
sudo test ! -f "$spool/exec_$native.stdout"
sudo test -f "$spool/exec_$native.stderr"
kill "$proxy_pid" 2>/dev/null || :
wait "$proxy_pid" 2>/dev/null || :
# Review only the exact SUCCEEDED Exec's GC receipt; no unknown lifecycle
# original is released. Existing CLI verifies root reviewer and native absence.
BW_S2_INSTALLED_SCOPED_GC=1 BW_S2_GC_CANDIDATE_SHA="$sha" sh scripts/s2_exec_gc_installed_review.sh "$work" "$db" "$project" "$socket" "$original" "$native" "$uuid"
online
gc loss-check
sudo test ! -f "$spool/exec_$native.stdout"
sudo test -f "$spool/exec_$native.stderr"
quota cleanup
test -z "$(sudo incus list --all-projects -c n -f csv)"
test "$runtime" = "$(sudo systemctl show boxweave-scoped-guard.service boxweave-scoped-lifecycle.service incus.service -p ActiveEnterTimestampMonotonic -p MainPID)"
sudo python3 - "$db" "$project" "$uuid" "$first_original" "$first_native" "$original" "$native" <<'CHECK'
import sqlite3,sys
db,project,uuid,first,fn,loss,ln=sys.argv[1:]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True) as c:
    for oid,native in ((first,fn),(loss,ln)):
        assert c.execute("SELECT layer,box_uuid,incarnation,state,native_operation_id FROM operations WHERE id=?", (oid,)).fetchall()==[(project,uuid,uuid,"SUCCEEDED",native)]
        assert c.execute("SELECT exit_code FROM exec_outputs WHERE operation_id=?", (oid,)).fetchall()==[(0,)]
    assert c.execute("SELECT stream,state FROM exec_output_gc WHERE operation_id=? ORDER BY stream",(first,)).fetchall()==[("stderr","EXPIRED"),("stdout","EXPIRED")]
    assert c.execute("SELECT stream,state FROM exec_output_gc WHERE operation_id=? ORDER BY stream",(loss,)).fetchall()==[("stdout","REVIEWED_RELEASED")]
    assert c.execute("SELECT state FROM owned_boxes WHERE project=? AND uuid=?", (project,uuid)).fetchall()==[("DELETED",)]
    assert c.execute("SELECT state FROM network_bindings WHERE project=? AND box_uuid=?", (project,uuid)).fetchall()==[("RELEASED",)]
    assert not c.execute("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')").fetchall()
print("P1B_SCOPED_GC_FIXED_LEDGER_NATIVE_ABSENCE=PASS")
CHECK
