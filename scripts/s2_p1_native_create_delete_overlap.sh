#!/bin/sh
# Real Incus 7.6, exact bound Create UUID while the original is VERIFYING.
# A disposable GDB non-stop breakpoint suspends only the worker thread
# immediately before Verified; all concurrent SDK calls use real MCP.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] || [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
command -v gdb >/dev/null 2>&1 || {
 sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends gdb >/dev/null
}
lab="$work/p1-native-overlap"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
db="$lab/ledger.sqlite"
actor() {
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
   GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
   BW_S2_TEST_UID="$(id -u boxweave)" \
   HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor "$work/boxweave" token-create -db "$db" -role admin > "$work/p1-overlap-seed.secret"
actor "$work/boxweave" token-create -db "$db" -role admin > "$work/p1-overlap-peer.secret"
chmod 0600 "$work/p1-overlap-seed.secret" "$work/p1-overlap-peer.secret"
token="$(sed -n 's/^token=//p' "$work/p1-overlap-seed.secret")"
other="$(sed -n 's/^token=//p' "$work/p1-overlap-peer.secret")"
test -n "$token" && test -n "$other" && test "$token" != "$other"
# Unlike an all-stop GDB breakpoint, non-stop permits the MCP listener and
# other HTTP handler threads to continue while only verified worker is held.
cat > "$work/p1-overlap.gdb" <<EOF
set pagination off
set confirm off
set debuginfod enabled off
set breakpoint pending off
set print thread-events off
set non-stop on
handle SIGURG nostop noprint pass
handle SIGPIPE nostop noprint pass
break github.com/vanishs/boxweave/internal/state/worker.go:79
commands 1
silent
shell /bin/sh -c "if test -e $lab/armed; then touch $lab/held; timeout 155 /bin/sh -c 'until test -e $lab/release; do sleep 0.1; done'; fi"
continue
end
run
EOF
sudo chown boxweave:incus "$work/p1-overlap.gdb"
actor touch "$lab/armed"
debug_pid=""
probe_pid=""
cleanup() {
 sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
 if [ -n "$probe_pid" ]; then kill "$probe_pid" 2>/dev/null || :; wait "$probe_pid" 2>/dev/null || :; fi
 if [ -n "$debug_pid" ]; then kill "$debug_pid" 2>/dev/null || :; wait "$debug_pid" 2>/dev/null || :; fi
}
trap cleanup EXIT
actor gdb -nx -q --batch -x "$work/p1-overlap.gdb" --args \
 "$work/boxweave" serve -db "$db" -incus-socket "$socket" \
 -project "$project" -listen 127.0.0.1:18668 \
 > "$work/p1-overlap-debugger.redacted.log" 2>&1 &
debug_pid=$!
attempt=0
until [ "$attempt" -ge 55 ]; do
 if ! kill -0 "$debug_pid" 2>/dev/null; then
  tail -100 "$work/p1-overlap-debugger.redacted.log"
  echo "S2_P1_OVERLAP_FAIL: native debugger service ended" >&2
  exit 95
 fi
 code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18668/mcp 2>/dev/null || :)"
 if [ "$code" = 401 ]; then break; fi
 attempt=$((attempt+1)); sleep 1
done
test "$attempt" -lt 55
grep -Eq 'Breakpoint 1 at|Breakpoint 1,|Breakpoint 1 \(' "$work/p1-overlap-debugger.redacted.log"
actor BW_S2_OVERLAP_LAB="$lab" BW_S2_OVERLAP_TOKEN="$token" \
 BW_S2_OVERLAP_SECOND_TOKEN="$other" BW_S2_OVERLAP_PROJECT="$project" \
 "$work/s2-p1-native-overlap-probe" > "$work/p1-native-overlap-results.txt" 2>&1 &
probe_pid=$!
attempt=0
while [ "$attempt" -lt 100 ] && ! sudo test -s "$lab/bound.json"; do
 if ! kill -0 "$probe_pid" 2>/dev/null; then
  cat "$work/p1-native-overlap-results.txt"
  tail -80 "$work/p1-overlap-debugger.redacted.log"
  echo "S2_P1_OVERLAP_FAIL: original did not bind UUID in held VERIFYING" >&2
  exit 96
 fi
 attempt=$((attempt+1)); sleep 1
done
sudo test -s "$lab/bound.json"
uuid="$(sudo python3 - "$lab/bound.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert len(v["uuid"])==36 and len(v["original_id"])==32 and len(v["native_id"])>0
print(v["uuid"])
PY
)"
name=bw-s2-p1-overlap-110x
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid"
test "$(sudo incus list "$name" --project "$project" --format csv -c s)" = STOPPED
test "$(sudo incus list --all-projects --format csv -c n)" = "$name"
sudo touch "$lab/native-verified"
echo "S2_P1_NATIVE_CREATE_BOUND_UUID_IN_VERIFYING_INDEPENDENT_NATIVE_WITNESS=PASS"
if ! wait "$probe_pid"; then
 cat "$work/p1-native-overlap-results.txt"
 tail -100 "$work/p1-overlap-debugger.redacted.log"
 echo "S2_P1_OVERLAP_FAIL: source-held MCP Delete admission" >&2
 exit 97
fi
probe_pid=""
grep -Fqx S2_P1_REAL_NATIVE_BOUND_VERIFYING_TWELVE_SDK_DELETE_REFUSED=PASS \
 "$work/p1-native-overlap-results.txt"
grep -Fqx S2_P1_REAL_NATIVE_OVERLAP_RELEASE_VERIFIED_DELETE_SUCCEEDED=PASS \
 "$work/p1-native-overlap-results.txt"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_REAL_NATIVE_BOUND_VERIFYING_CONCURRENT_DELETE_FULL_LIFECYCLE=PASS"
