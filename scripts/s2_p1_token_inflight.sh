#!/bin/sh
# One-use CI only: real installed HTTP MCP bearer revocation/expiry while a
# private operation_get is waiting. No native mutations or historic UNKNOWN.
set -eu
if [ "$GITHUB_ACTIONS" != true ] || [ "$RUNNER_ENVIRONMENT" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"
project="$2"
socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
lab="$work/p1-token-inflight"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
db="$lab/ledger.sqlite"
actor() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor "$work/boxweave" token-create -db "$db" -role admin > "$work/p1-token-bootstrap.secret"
chmod 0600 "$work/p1-token-bootstrap.secret"
service_pid=""
cleanup() {
  sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
  if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; fi
}
trap cleanup EXIT
actor "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18659 > "$work/p1-token-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18659/mcp 2>/dev/null || :)"
  if [ "$code" = 401 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
actor BW_S2_AUTH_LAB="$lab" BW_S2_AUTH_PROJECT="$project" \
  "$work/s2-p1-token-inflight-probe" > "$work/p1-token-results.txt"
grep -Fqx S2_P1_INSTALLED_MCP_REVOKED_DURING_READ_NO_LEAK=PASS "$work/p1-token-results.txt"
grep -Fqx S2_P1_INSTALLED_MCP_EXPIRED_DURING_READ_NO_LEAK=PASS "$work/p1-token-results.txt"
grep -Fqx S2_P1_INSTALLED_INFLIGHT_BEARER_REVOCATION_AND_EXPIRY=PASS "$work/p1-token-results.txt"
grep -Fqx S2_P1_REAL_INSTALLED_MCP_STATUS_CHANGED_DURING_SIGNED_CURSOR=PASS "$work/p1-token-results.txt"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_REAL_INSTALLED_TOKEN_RACE_ISOLATED_NO_NATIVE_EFFECT=PASS"
