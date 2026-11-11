#!/bin/sh
# P0: real Incus HTTP 202 accepted Update and its own SQLite NativeAck WAL
# fsync killed before/after syscall. Disposable GitHub-hosted CI ONLY.
set -eu
if [ "$GITHUB_ACTIONS" != true ] || [ "$RUNNER_ENVIRONMENT" != github-hosted ] ||
  [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92;; esac
test "$socket" = /var/lib/incus/unix.socket.user
test "$project" = "user-$(id -u boxweave)"
test -z "$(sudo incus list --all-projects --format csv -c n)"
service_pid=""; proxy_pid=""
cleanup() {
  if [ -n "$service_pid" ]; then kill "$service_pid" 2>/dev/null || :; wait "$service_pid" 2>/dev/null || :; fi
  if [ -n "$proxy_pid" ]; then kill "$proxy_pid" 2>/dev/null || :; wait "$proxy_pid" 2>/dev/null || :; fi
}
trap cleanup EXIT
actor() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
online() {
  attempt=0
  until [ "$attempt" -ge 40 ]; do
    if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18657/mcp 2>/dev/null)" = 401 ]; then return 0; fi
    attempt=$((attempt+1)); sleep 1
  done
  return 1
}
# A runuser supervisor PID is not proof that its child listener exited.
# Kill ONLY the exact disposable nonroot BoxWeave and independently prove
# the previous scoped HTTP listener closed before changing the native socket.
stop_service() {
  sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
  if [ -n "$service_pid" ]; then
    wait "$service_pid" 2>/dev/null || :
    service_pid=""
  fi
  attempt=0
  until [ "$attempt" -ge 30 ]; do
    code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18657/mcp 2>/dev/null || :)"
    if [ "$code" != 401 ]; then return 0; fi
    attempt=$((attempt+1))
    sleep 1
  done
  echo "S2_P0_FAIL: previous one-use MCP listener still active" >&2
  return 1
}
for phase in before after; do
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  lab="$work/p0-$phase"
  mkdir -m 0700 "$lab"; sudo chown boxweave:incus "$lab"
  db="$lab/ledger.sqlite"
  actor "$work/boxweave" token-create -db "$db" -role admin > "$work/p0-$phase-token.secret"
  token="$(sed -n 's/^token=//p' "$work/p0-$phase-token.secret")"; test -n "$token"
  p0_probe() {
    actor BW_S2_P0_DIR="$lab" BW_S2_P0_TOKEN="$token" BW_S2_P0_PROJECT="$project" \
      "$work/s2-p0-native-sqlite-probe" "$1"
  }
  actor "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
    -listen 127.0.0.1:18657 > "$work/p0-$phase-seed.redacted.log" 2>&1 &
  service_pid=$!; online
  p0_probe seed > "$work/p0-$phase-seed.txt"
  grep -Fqx S2_P0_COMBINED_NATIVE_VERIFIED_OWNED_FIXTURE=PASS "$work/p0-$phase-seed.txt"
  name=bw-s2-p0-before-110j; if [ "$phase" = after ]; then name=bw-s2-p0-after-110k; fi
  uuid="$(sudo python3 - "$lab/original.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert len(v["box_id"]) == 36
print(v["box_id"])
PY
)"
  test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
  test "$(sudo incus config get "$name" limits.cpu --project "$project")" = 1
  stop_service
  proxy="$lab/proxy.sock"; marker="$lab/native-202.txt"; arm="$lab/arm"; witness="$lab/witness"
  actor "$work/s2-p0-native-sqlite-proxy" -listen "$proxy" -upstream "$socket" \
    -project "$project" -name "$name" -arm "$arm" -marker "$marker" \
    > "$work/p0-$phase-proxy.redacted.log" 2>&1 &
  proxy_pid=$!; attempt=0
  until sudo test -S "$proxy" || [ "$attempt" -ge 30 ]; do attempt=$((attempt+1)); sleep 1; done
  sudo test -S "$proxy"; sudo test ! -e "$arm"
  actor BW_S2_FSYNC_DB_WAL="$db-wal" BW_S2_FSYNC_ARM_FILE="$arm" \
    BW_S2_FSYNC_WITNESS_FILE="$witness" BW_S2_FSYNC_CUT_MODE="$phase" \
    LD_PRELOAD="$work/s2-fsync-cut.so" "$work/boxweave" serve -db "$db" \
    -incus-socket "$proxy" -project "$project" \
    -listen 127.0.0.1:18657 > "$work/p0-$phase-cut.redacted.log" 2>&1 &
  service_pid=$!; online
  p0_probe apply > "$work/p0-$phase-admitted.txt"
  grep -Fqx S2_P0_COMBINED_ORIGINAL_NATIVE_UPDATE_ADMITTED=PASS "$work/p0-$phase-admitted.txt"
  attempt=0
  until sudo test -f "$witness" || [ "$attempt" -ge 40 ]; do attempt=$((attempt+1)); sleep 1; done
  if ! sudo test -f "$witness"; then
    echo "S2_P0_FAIL: no exact SQLite WAL fd sync-cut witness" >&2
    if sudo test -f "$marker"; then
      echo "S2_P0_DIAGNOSTIC: upstream native HTTP 202 marker PRESENT" >&2
    else
      echo "S2_P0_DIAGNOSTIC: upstream native HTTP 202 marker ABSENT" >&2
    fi
    if sudo test -f "$arm"; then
      echo "S2_P0_DIAGNOSTIC: post-acceptance WAL-cut arm PRESENT" >&2
    else
      echo "S2_P0_DIAGNOSTIC: post-acceptance WAL-cut arm ABSENT" >&2
    fi
    # No raw native HTTP content or tokens: only a compiled marker emitted
    # by the restricted one-use proxy for this exact disposable Update.
    grep -F "S2_P0_SCOPED_NATIVE_UPDATE_HTTP_STATUS=" "$work/p0-$phase-proxy.redacted.log" >&2 || :
    exit 96
  fi
  sudo grep -Fqx ORIGINAL_SQLITE_WAL_FSYNC_CUT_ACTUALLY_TRIGGERED "$witness"
  sudo grep -Eq '^NATIVE_REAL_UPDATE_202_ACCEPTED=[0-9a-f-]{36}$' "$marker"
  set +e; wait "$service_pid"; rc=$?; set -e; test "$rc" -ne 0; service_pid=""
  attempt=0
  until [ "$(sudo incus config get "$name" limits.cpu --project "$project")" = 2 ] || [ "$attempt" -ge 80 ]; do
    attempt=$((attempt+1)); sleep 1
  done
  test "$attempt" -lt 80
  test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
  sudo test ! -e "$lab/unexpected-native-dispatch.txt"
  echo S2_P0_REAL_NATIVE_EFFECT_AFTER_ACTUAL_SQLITE_SYNC_KILL=PASS
  actor "$work/boxweave" serve -db "$db" -incus-socket "$proxy" -project "$project" \
    -listen 127.0.0.1:18657 > "$work/p0-$phase-restart.redacted.log" 2>&1 &
  service_pid=$!; online
  p0_probe resume > "$work/p0-$phase-replay.txt"
  grep -Fqx S2_P0_REAL_NATIVE_SQLITE_CRASH_UNKNOWN_REPLAY_AND_CONCURRENT_READS=PASS "$work/p0-$phase-replay.txt"
  # Independent read-only SQLite connection and separately witnessed native
  # operation ID must agree with the original public MCP receipt. No native
  # operation is inferred from an uncommitted SQLite ACK.
  sudo python3 - "$db" "$lab/original.json" "$marker" "$phase" "$uuid" <<'PY'
import json,re,sqlite3,sys
db,path,marker,phase,uuid=sys.argv[1:]
saved=json.load(open(path))
native=open(marker).read().strip()
assert re.fullmatch(r"NATIVE_REAL_UPDATE_202_ACCEPTED=[0-9a-f-]{36}",native)
native=native.split("=",1)[1]
with sqlite3.connect(f"file:{db}?mode=ro",uri=True) as conn:
    assert conn.execute("PRAGMA integrity_check").fetchone()[0]=="ok"
    op=conn.execute("SELECT state,native_operation_id,tracking_id,box_uuid,work_ref,agent_ref FROM operations WHERE id=?",(saved["update_id"],)).fetchone()
    assert op and op[0]=="UNKNOWN" and op[1] in ("",native)
    assert op[2]==saved["tracking_id"] and op[3]==uuid
    assert op[4]=="lab/p0-native-sqlite" and op[5]=="p0-combined-verifier"
    if phase=="after": assert op[1]==native,("post-fsync native ID missing",op)
    events=[r[0] for r in conn.execute("SELECT state FROM operation_events WHERE operation_id=? ORDER BY seq",(saved["update_id"],))]
    assert events.count("UNKNOWN")==1,events
print("S2_P0_INDEPENDENT_SQLITE_NATIVE_MCP_RECEIPT_CROSSCHECK=PASS")
PY
  sudo test ! -e "$lab/unexpected-native-dispatch.txt"
  stop_service
  sh scripts/s2_p0_unknown_review.sh "$work" "$lab" "$project" "$socket" "$uuid"
  # The independent CI admin tears down ONLY this exact observed disposable UUID.
  sudo incus delete --force "$name" --project "$project"
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  kill "$proxy_pid" 2>/dev/null || :; wait "$proxy_pid" 2>/dev/null || :; proxy_pid=""
  printf 'S2_P0_REAL_NATIVE_202_SQLITE_WAL_%s_SAFE_RECOVERY=PASS\n' "$phase"
done
echo S2_P0_NATIVE_202_COMBINED_WITH_SQLITE_COMMIT_FAULT_BOTH_PHASES=PASS
