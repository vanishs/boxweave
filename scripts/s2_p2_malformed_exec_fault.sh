#!/bin/sh
# Real native recorded Guest Exec, test-only malformed final Incus HTTP GET.
# Dedicated GitHub-hosted VM only. Original UNKNOWN is never unlocked/retried.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] ||
   [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
lab="$work/p2-malformed-exec"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
db="$lab/ledger.sqlite"
actor() {
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
   GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
   HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor "$work/boxweave" token-create -db "$db" -role admin > "$work/p2-malformed-admin.secret"
chmod 0600 "$work/p2-malformed-admin.secret"
token="$(sed -n 's/^token=//p' "$work/p2-malformed-admin.secret")"
test -n "$token"
service_pid=""; proxy_pid=""
cleanup() {
 sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
 if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; fi
 if [ -n "$proxy_pid" ]; then kill "$proxy_pid" 2>/dev/null || :; wait "$proxy_pid" 2>/dev/null || :; fi
}
trap cleanup EXIT
ready() {
 attempt=0
 until [ "$attempt" -ge 40 ]; do
  code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18665/mcp 2>/dev/null || :)"
  if [ "$code" = 401 ]; then return 0; fi
  attempt=$((attempt+1)); sleep 1
 done
 echo "S2_P2_FAIL: private installed MCP not ready" >&2
 return 1
}
probe() {
 actor BW_S2_MALFORM_LAB="$lab" BW_S2_MALFORM_PROJECT="$project" \
   BW_S2_MALFORM_TOKEN="$token" "$work/s2-malformed-exec-probe" "$1"
}
name=bw-s2-p2-malformed-exec-110v
actor "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
 -listen 127.0.0.1:18665 > "$work/p2-malformed-seed.redacted.log" 2>&1 &
service_pid=$!
ready
probe seed > "$work/p2-malformed-seed-results.txt"
grep -Fqx S2_P2_MALFORMED_NATIVE_EXEC_OWNED_RUNNING_SEEDED=PASS "$work/p2-malformed-seed-results.txt"
uuid="$(sudo python3 - "$lab/original.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert len(v["box_id"])==36 and len(v["create_id"])==32
print(v["box_id"])
PY
)"
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
# Positive readiness is independently proved by a real guest command,
# never inferred merely from the Incus Running state.
attempt=0
until sudo timeout 8 incus exec "$name" --project "$project" -- /bin/true >/dev/null 2>&1 ||
      [ "$attempt" -ge 45 ]; do
  attempt=$((attempt+1)); sleep 1
done
test "$attempt" -lt 45
echo "S2_P2_GUEST_REAL_EXEC_READY_POSITIVE_CONTROL=PASS"
# A successful early /bin/true does not imply cloud-init/systemd has
# finished all private /tmp setup. The durable *effect* witness therefore
# lives in this one-use Guest's persistent /root, never its transient /tmp.
# First prove that separate native exec sessions see the same root data.
sudo incus exec "$name" --project "$project" -- /bin/sh -c  'printf "root-fs-check-110v\n" > /root/bw-s2-p2-root-mount-precheck-110v'
test "$(sudo incus exec "$name" --project "$project" --  cat /root/bw-s2-p2-root-mount-precheck-110v)" = root-fs-check-110v
sudo incus exec "$name" --project "$project" --  rm /root/bw-s2-p2-root-mount-precheck-110v
echo "S2_P2_GUEST_ROOT_WITNESS_STABLE_ACROSS_SEPARATE_EXEC=PASS"
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
wait "$service_pid" 2>/dev/null || :
service_pid=""
attempt=0
until [ "$attempt" -ge 30 ]; do
 code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18665/mcp 2>/dev/null || :)"
 if [ "$code" != 401 ]; then break; fi
 attempt=$((attempt+1)); sleep 1
done
test "$attempt" -lt 30
actor BW_S2_TEST_UID="$(id -u boxweave)" "$work/s2-malformed-exec-proxy" \
 -listen "$lab/proxy.sock" -upstream "$socket" -project "$project" \
 -name "$name" -lab "$lab" > "$work/p2-malformed-proxy.redacted.log" 2>&1 &
proxy_pid=$!
attempt=0
until sudo test -S "$lab/proxy.sock" || [ "$attempt" -ge 40 ]; do
 attempt=$((attempt+1)); sleep 1
done
sudo test -S "$lab/proxy.sock"
start_service() {
 actor "$work/boxweave" serve -db "$db" -incus-socket "$lab/proxy.sock" \
 -project "$project" -listen 127.0.0.1:18665 \
 > "$work/p2-malformed-active.redacted.log" 2>&1 &
 service_pid=$!
 ready
}
start_service
probe malform > "$work/p2-malformed-original-results.txt"
grep -Fqx S2_P2_REAL_INCUS_MALFORMED_FINAL_EXEC_RETURN_UNKNOWN_NO_RETRY=PASS \
 "$work/p2-malformed-original-results.txt"
sudo grep -Eq '^REAL_EXEC_202_NATIVE_ID=[0-9a-f-]{36}$' "$lab/native-accepted.txt"
sudo grep -Eq '^MALFORMED_FINAL_NATIVE_ID=[0-9a-f-]{36}$' "$lab/native-malformed.txt"
sudo test ! -e "$lab/native-duplicate.txt"
native_id="$(sudo sed -n 's/^REAL_EXEC_202_NATIVE_ID=//p' "$lab/native-accepted.txt")"
replied_id="$(sudo sed -n 's/^MALFORMED_FINAL_NATIVE_ID=//p' "$lab/native-malformed.txt")"
test "$native_id" = "$replied_id"
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
# The actual native Guest command appended one line. An unobserved second
# dispatch would leave two lines even if the MCP receipt appeared harmless.
count="$(sudo incus exec "$name" --project "$project" -- /bin/sh -c \
 'wc -l < /root/bw-s2-malformed-exec-once-110v')"
test "$(printf '%s' "$count" | tr -d '[:space:]')" = 1
echo "S2_P2_REAL_GUEST_MUTATION_EXACTLY_ONCE_AFTER_MALFORMED_RESPONSE=PASS"
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
wait "$service_pid" 2>/dev/null || :
service_pid=""
start_service
probe resume > "$work/p2-malformed-restart-results.txt"
grep -Fqx S2_P2_REAL_MALFORMED_EXEC_SAME_SQLITE_RESTART_NO_REDISPATCH=PASS \
 "$work/p2-malformed-restart-results.txt"
sudo test ! -e "$lab/native-duplicate.txt"
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
test "$(sudo incus list --all-projects --format csv -c n)" = "$name"
# Only independently confirmed disposable fixture is removed, not its
# unresolved original journal, which retains UNKNOWN and native ID.
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
wait "$service_pid" 2>/dev/null || :
service_pid=""
sudo incus delete --force "$name" --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P2_REAL_MALFORMED_EXEC_NATIVE_202_UNKNOWN_RESTART_CLEANUP=PASS"
