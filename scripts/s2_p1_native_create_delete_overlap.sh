#!/bin/sh
# Real native Incus 7.6, actual UUID-bound VERIFYING Create and 12 parallel
# authentic SDK Delete requests. Only the worker source of THIS acceptance
# binary is overlaid with a deterministic post-commit pause. Release/product
# worker.go and its binary remain unchanged, separately native-accepted.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] ||
   [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
lab="$work/p1-native-overlap"
test ! -e "$lab"
mkdir -m 0700 "$lab"
# Build a disposable test-only binary using a Go source overlay while
# still owning this private workspace. No committed code or release binary
# is edited. Fail closed if the exact post-verify anchor has changed.
python3 scripts/s2_p1_build_overlap_overlay.py \
  "$PWD/internal/state/worker.go" "$lab"
CGO_ENABLED=1 go build -trimpath -overlay="$lab/go-overlay.json" \
  -ldflags "-X main.version=1.0.24-s2-overlap-overlay -X main.commit=$(git rev-parse HEAD)" \
  -o "$work/s2-p1-overlap-service" ./cmd/boxweave
chmod 0755 "$work/s2-p1-overlap-service"
sha256sum "$work/s2-p1-overlap-service"
if [ -n "$(git status --porcelain)" ]; then
  echo "S2_NATIVE_OVERLAP_TEST_HARNESS_FAIL: checkout was contaminated by untracked test files" >&2
  git status --porcelain >&2
  exit 93
fi
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
service_pid=""
probe_pid=""
cleanup() {
 sudo pkill -TERM -u boxweave -f "^$work/s2-p1-overlap-service serve -db $db" || :
 if [ -n "$probe_pid" ]; then kill "$probe_pid" 2>/dev/null || :; wait "$probe_pid" 2>/dev/null || :; fi
 if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; fi
}
trap cleanup EXIT
actor BW_S2_OVERLAP_ACTIVATE=one-disposable-original \
 BW_S2_OVERLAP_LAB="$lab" \
 "$work/s2-p1-overlap-service" serve -db "$db" -incus-socket "$socket" \
 -project "$project" -listen 127.0.0.1:18668 \
 > "$work/p1-overlap-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 45 ]; do
 if ! kill -0 "$service_pid" 2>/dev/null; then
  tail -60 "$work/p1-overlap-service.redacted.log"
  echo "S2_P1_FAIL: test-only source overlay service exited early" >&2; exit 95
 fi
 code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18668/mcp 2>/dev/null || :)"
 if [ "$code" = 401 ]; then break; fi
 attempt=$((attempt+1)); sleep 1
done
test "$attempt" -lt 45
echo "S2_P1_NATIVE_OVERLAY_TEST_ONLY_SERVICE_MCP_AUTH_READY=PASS"
actor BW_S2_OVERLAP_LAB="$lab" BW_S2_OVERLAP_TOKEN="$token" \
 BW_S2_OVERLAP_SECOND_TOKEN="$other" BW_S2_OVERLAP_PROJECT="$project" \
 "$work/s2-p1-native-overlap-probe" > "$work/p1-native-overlap-results.txt" 2>&1 &
probe_pid=$!
attempt=0
while [ "$attempt" -lt 90 ] && ! sudo test -s "$lab/bound.json"; do
 if ! kill -0 "$probe_pid" 2>/dev/null; then
  cat "$work/p1-native-overlap-results.txt"
  echo "S2_P1_FAIL: verifier stopped before exact bound SQLite witness" >&2; exit 96
 fi
 attempt=$((attempt+1)); sleep 1
done
sudo test -s "$lab/held"
sudo test -s "$lab/bound.json"
sudo grep -Fqx POST_REGISTER_CREATED_BOX_COMMIT_VERIFIED_EFFECT_RETURNED "$lab/held"
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
echo "S2_P1_NATIVE_UUID_BOUND_VERIFYING_INDEPENDENT_INCUS_CONFIRM=PASS"
if ! wait "$probe_pid"; then
 cat "$work/p1-native-overlap-results.txt"
 tail -70 "$work/p1-overlap-service.redacted.log"
 echo "S2_P1_FAIL: original in-flight MCP lock/verified recovery failed" >&2; exit 97
fi
probe_pid=""
grep -Fqx S2_P1_REAL_NATIVE_BOUND_VERIFYING_TWELVE_SDK_DELETE_REFUSED=PASS \
 "$work/p1-native-overlap-results.txt"
grep -Fqx S2_P1_REAL_NATIVE_OVERLAP_RELEASE_VERIFIED_DELETE_SUCCEEDED=PASS \
 "$work/p1-native-overlap-results.txt"
# Both positive claims were independently checked above, but the SDK probe
# stdout was deliberately written to a private fixture results file. Emit
# only the verified, non-secret PASS markers so the separate P3 audit can
# require them from the public CI witness transcript.
echo "S2_P1_REAL_NATIVE_BOUND_VERIFYING_TWELVE_SDK_DELETE_REFUSED=PASS"
echo "S2_P1_REAL_NATIVE_OVERLAP_RELEASE_VERIFIED_DELETE_SUCCEEDED=PASS"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_REAL_NATIVE_BOUND_VERIFYING_CONCURRENT_DELETE_FULL_LIFECYCLE=PASS"
