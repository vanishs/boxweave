#!/bin/sh
# Single-use real Incus Stop/Delete accepted-202 lost-ACK fault.
# Do not operate an existing/foreign Incus, retry the original effect, or
# interfere with historical UNKNOWN. Never installed on user hosts.
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
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted "$@"
}
online() {
  attempt=0
  until [ "$attempt" -ge 40 ]; do
    code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18658/mcp 2>/dev/null || :)"
    if [ "$code" = 401 ]; then return 0; fi
    attempt=$((attempt+1)); sleep 1
  done
  echo "S2_P1_FAIL: unique MCP listener unavailable" >&2
  return 1
}
stop_service() {
  sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
  if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; service_pid=""; fi
  attempt=0
  until [ "$attempt" -ge 30 ]; do
    code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18658/mcp 2>/dev/null || :)"
    if [ "$code" != 401 ]; then return 0; fi
    attempt=$((attempt+1)); sleep 1
  done
  echo "S2_P1_FAIL: original MCP service still active" >&2
  return 1
}
for phase in stop delete; do
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  lab="$work/p1-$phase"; mkdir -m 0700 "$lab"; sudo chown boxweave:incus "$lab"
  db="$lab/ledger.sqlite"
  actor "$work/boxweave" token-create -db "$db" -role admin > "$work/p1-$phase-token.secret"
  chmod 0600 "$work/p1-$phase-token.secret"
  token="$(sed -n 's/^token=//p' "$work/p1-$phase-token.secret")"; test -n "$token"
  p1_probe() {
    actor BW_S2_P1_DIR="$lab" BW_S2_P1_PROJECT="$project" BW_S2_P1_TOKEN="$token" \
      "$work/s2-p1-stop-delete-probe" "$1"
  }
  actor "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
    -listen 127.0.0.1:18658 > "$work/p1-$phase-seed.redacted.log" 2>&1 &
  service_pid=$!; online
  p1_probe seed > "$work/p1-$phase-seed.txt"
  grep -Fqx S2_P1_STOP_DELETE_NATIVE_OWNED_REAL_UUID_SEEDED=PASS "$work/p1-$phase-seed.txt"
  name="bw-s2-p1-$phase-110p"
  uuid="$(sudo python3 - "$lab/original.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert len(v["box_id"])==36 and len(v["create_id"])==32
print(v["box_id"])
PY
)"
  test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
  initial="$(sudo incus list "$name" --project "$project" --format csv -c s)"
  if [ "$phase" = stop ]; then test "$initial" = RUNNING; phase_label=STOP
  else test "$initial" = STOPPED; phase_label=DELETE; fi
  stop_service
  marker="$lab/native-accepted.txt"
  proxy_socket="$lab/proxy.sock"
  actor "$work/s2-p1-stop-delete-proxy" -listen "$proxy_socket" \
    -upstream "$socket" -project "$project" -phase "$phase" -marker "$marker" \
    > "$work/p1-$phase-proxy.redacted.log" 2>&1 &
  proxy_pid=$!
  attempt=0
  until sudo test -S "$proxy_socket" || [ "$attempt" -ge 30 ]; do
    attempt=$((attempt+1)); sleep 1
  done
  sudo test -S "$proxy_socket"
  actor "$work/boxweave" serve -db "$db" -incus-socket "$proxy_socket" -project "$project" \
    -listen 127.0.0.1:18658 > "$work/p1-$phase-ack-loss.redacted.log" 2>&1 &
  service_pid=$!; online
  p1_probe mutate > "$work/p1-$phase-admitted.txt"
  grep -Fqx S2_P1_STOP_DELETE_ORIGINAL_MUTATION_ADMITTED=PASS "$work/p1-$phase-admitted.txt"
  attempt=0
  until sudo test -f "$marker" || [ "$attempt" -ge 45 ]; do
    attempt=$((attempt+1)); sleep 1
  done
  sudo grep -Eq "^S2_P1_NATIVE_"$phase_label"_202_ACCEPTED_ID=[0-9a-f-]{36}$" "$marker"
  attempt=0
  if [ "$phase" = stop ]; then
    until [ "$(sudo incus list "$name" --project "$project" --format csv -c s)" = STOPPED ] ||
      [ "$attempt" -ge 90 ]; do attempt=$((attempt+1)); sleep 1; done
    test "$attempt" -lt 90
    test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
  else
    until [ -z "$(sudo incus list "$name" --project "$project" --format csv -c n)" ] ||
      [ "$attempt" -ge 90 ]; do attempt=$((attempt+1)); sleep 1; done
    test "$attempt" -lt 90
    if sudo incus info "$name" --project "$project" > "$work/p1-delete-gone.txt" 2>&1; then exit 97; fi
  fi
  p1_probe poll > "$work/p1-$phase-before-restart.txt"
  grep -Fqx "S2_P1_NATIVE_"$phase_label"_LOST_ACK_BEFORE_RESTART_UNKNOWN=PASS" \
    "$work/p1-$phase-before-restart.txt"
  sudo test ! -e "$marker.unexpected"
  echo "S2_P1_REAL_NATIVE_"$phase_label"_202_EFFECT_ACK_LOST=PASS"
  stop_service
  actor "$work/boxweave" serve -db "$db" -incus-socket "$proxy_socket" -project "$project" \
    -listen 127.0.0.1:18658 > "$work/p1-$phase-restart.redacted.log" 2>&1 &
  service_pid=$!; online
  p1_probe resume > "$work/p1-$phase-after-restart.txt"
  grep -Fqx "S2_P1_NATIVE_"$phase_label"_LOST_ACK_RESTART_UNKNOWN_NO_REPLAY=PASS" \
    "$work/p1-$phase-after-restart.txt"
  sudo test ! -e "$marker.unexpected"
  stop_service
  if [ "$phase" = stop ]; then
    test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
    sudo incus delete "$name" --project "$project"
  fi
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  sudo pkill -TERM -u boxweave -f "^$work/s2-p1-stop-delete-proxy" || :
  wait "$proxy_pid" 2>/dev/null || :
  proxy_pid=""
  echo "S2_P1_REAL_NATIVE_"$phase_label"_202_LOST_ACK_SAFE_RESTART=PASS"
done
echo "S2_P1_REAL_STOP_AND_DELETE_NATIVE_202_ACK_LOSS_NO_REPLAY=PASS"
