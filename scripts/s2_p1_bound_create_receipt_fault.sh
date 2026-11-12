#!/bin/sh
# Real disposable native Create; test-only SQLite trigger fails the LAST
# SUCCEEDED receipt after Incus 202 and exact UUID binding. Do not install any
# test trigger on retained/shared databases or touch historical UNKNOWNs.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] ||
   [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"
project="$2"
socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
lab="$work/p1-bound-receipt"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
db="$lab/ledger.sqlite"
actor() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor "$work/boxweave" token-create -db "$db" -role admin > "$work/bound-receipt-admin.secret"
chmod 0600 "$work/bound-receipt-admin.secret"
token="$(sed -n 's/^token=//p' "$work/bound-receipt-admin.secret")"
test -n "$token"
service_pid=""
cleanup() {
  sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
  if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; fi
}
trap cleanup EXIT
actor "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18663 > "$work/bound-receipt-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  status="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18663/mcp 2>/dev/null || :)"
  if [ "$status" = 401 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
actor BW_S2_BOUND_LAB="$lab" BW_S2_BOUND_TOKEN="$token" \
  BW_S2_BOUND_PROJECT="$project" "$work/s2-bound-create-probe" \
  > "$work/bound-receipt-result.txt"
grep -Fqx S2_P1_REAL_NATIVE_CREATE_BOUND_UUID_FINAL_RECEIPT_UNKNOWN_NO_REDISPATCH=PASS \
  "$work/bound-receipt-result.txt"
grep -Fqx S2_P1_REAL_BOUND_UNKNOWN_DIFFERENT_KEY_CREATE_DELETE_REFUSED=PASS \
  "$work/bound-receipt-result.txt"

name=bw-s2-p1-bound-final-110u
uuid="$(sudo python3 - "$lab/original.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert len(v["box_id"])==36 and len(v["original_id"])==32 and v["native_id"]
assert v["name"]=="bw-s2-p1-bound-final-110u"
print(v["box_id"])
PY
)"
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
test "$(sudo incus list "$name" --project "$project" --format csv -c s)" = STOPPED
test "$(sudo incus list --all-projects --format csv -c n)" = "$name"
echo "S2_P1_BOUND_UNKNOWN_INDEPENDENT_NATIVE_UUID_AND_STOPPED_STATE=PASS"
# The separate CI administrator may destroy this exact artificial test-only
# native Box after proof. Never silently unlock/adopt/retry its UNKNOWN journal.
sudo incus delete "$name" --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_BOUND_UNKNOWN_DISPOSABLE_NATIVE_FIXTURE_ONLY_CLEANUP=PASS"
