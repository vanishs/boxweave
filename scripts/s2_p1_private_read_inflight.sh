#!/bin/sh
# One-use GitHub-hosted Incus only: actual HTTP 200 from native Project and
# owned Box GET held until a new read-specific bearer has been revoked.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] ||
   [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
lab="$work/p1-private-read"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
db="$lab/ledger.sqlite"
name=bw-s2-read-revoke-110r
actor() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor "$work/boxweave" token-create -db "$db" -role admin > "$work/private-read-seed.secret"
chmod 0600 "$work/private-read-seed.secret"
seed_token="$(sed -n 's/^token=//p' "$work/private-read-seed.secret")"
test -n "$seed_token"
service_pid=""
proxy_pid=""
cleanup_processes() {
  sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
  if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; fi
  if [ -n "$proxy_pid" ]; then kill "$proxy_pid" 2>/dev/null || :; wait "$proxy_pid" 2>/dev/null || :; fi
}
trap cleanup_processes EXIT
online() {
  attempt=0
  until [ "$attempt" -ge 45 ]; do
    status="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18662/mcp 2>/dev/null || :)"
    if [ "$status" = 401 ]; then return 0; fi
    attempt=$((attempt + 1))
    sleep 1
  done
  echo "S2_P1_PRIVATE_FAIL: candidate service unavailable" >&2
  return 1
}
probe() {
  actor BW_S2_PRIVATE_READ_DIR="$lab" BW_S2_PRIVATE_READ_PROJECT="$project" \
    BW_S2_PRIVATE_READ_SEED_TOKEN="$seed_token" \
    "$work/s2-private-read-probe" "$1"
}

# First create and Exec one real bounded native test-only Box, with no fault
# injector. All subsequent protected reads target this exact original UUID.
actor "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18662 > "$work/private-read-direct.redacted.log" 2>&1 &
service_pid=$!
online
probe seed > "$work/private-read-seed-results.txt"
grep -Fqx S2_P1_PRIVATE_READ_ORIGINAL_NATIVE_BOX_EXEC_SEEDED=PASS "$work/private-read-seed-results.txt"
uuid="$(sudo python3 - "$lab/original.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
assert len(obj["uuid"])==36 and len(obj["exec_id"])==32
print(obj["uuid"])
PY
)"
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
echo "S2_P1_PRIVATE_READ_SEED_ADMIN_UUID_CONFIRMED=PASS"

sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
wait "$service_pid" 2>/dev/null || :
service_pid=""
attempt=0
until [ "$attempt" -ge 30 ]; do
  status="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18662/mcp 2>/dev/null || :)"
  if [ "$status" != 401 ]; then break; fi
  attempt=$((attempt + 1)); sleep 1
done
test "$attempt" -lt 30

# The separate proxy can forward the actual native response; it does not
# fabricate an SDK call, an Incus response or an original Operation.
actor BW_S2_TEST_UID="$(id -u boxweave)" "$work/s2-private-read-proxy" \
  -listen "$lab/proxy.sock" -upstream "$socket" -project "$project" \
  -name "$name" -dir "$lab" > "$work/private-read-proxy.redacted.log" 2>&1 &
proxy_pid=$!
attempt=0
until sudo test -S "$lab/proxy.sock" || [ "$attempt" -ge 35 ]; do
  attempt=$((attempt + 1)); sleep 1
done
sudo test -S "$lab/proxy.sock"
actor "$work/boxweave" serve -db "$db" -incus-socket "$lab/proxy.sock" \
  -project "$project" -listen 127.0.0.1:18662 \
  > "$work/private-read-delayed.redacted.log" 2>&1 &
service_pid=$!
online
probe read > "$work/private-read-inflight-results.txt"
for method in HOST_STATUS BOX_LIST BOX_GET OPERATION_OUTPUT_GET; do
  grep -Fqx "S2_P1_PRIVATE_READ_REVOKED_AFTER_REAL_NATIVE_HTTP_200_$method=PASS" \
    "$work/private-read-inflight-results.txt"
  grep -Fqx "S2_P1_PRIVATE_READ_EXPIRED_AFTER_REAL_NATIVE_HTTP_200_$method=PASS" \
    "$work/private-read-inflight-results.txt"
done
grep -Fqx S2_P1_FOUR_REAL_MCP_NATIVE_PRIVATE_READ_BEARER_REVOKE_NO_LEAK=PASS \
  "$work/private-read-inflight-results.txt"
grep -Fqx S2_P1_FOUR_REAL_MCP_NATIVE_PRIVATE_READ_BEARER_EXPIRY_NO_LEAK=PASS \
  "$work/private-read-inflight-results.txt"
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
echo "S2_P1_PRIVATE_READ_REVOKED_CALLS_LEFT_NATIVE_UUID_UNCHANGED=PASS"

# Cleanup only this acknowledged original UUID via still-valid seeded admin.
probe cleanup > "$work/private-read-cleanup-results.txt"
grep -Fqx S2_P1_PRIVATE_READ_EXACT_FIXTURE_CLEANUP=PASS "$work/private-read-cleanup-results.txt"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_FOUR_REAL_PRIVATE_READ_TOKEN_RACES_NATIVE_POSTREAD_EMPTY=PASS"
