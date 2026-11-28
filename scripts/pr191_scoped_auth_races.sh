#!/bin/sh
# Reuse the original post-query and held native-read probes only on a
# pristine N2 runner with a verified fixed Scoped installation.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
test "$#" -eq 3
project="$1"; receipt="$2"; sha="$3"
n2="$(dirname "$receipt")"
case "$n2" in /tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$receipt" = "$n2/actor.receipt"
test "$(stat -c '%a' "$receipt")" = 600
test "$project" = "user-$(id -u boxweave)"
test "$(git rev-parse HEAD)" = "${N2_EXACT_HEAD:?}"
test "$(sha256sum "$n2/candidate-a" | cut -d' ' -f1)" = "$sha"
test "$(sudo sha256sum /usr/local/bin/boxweave | cut -d' ' -f1)" = "$sha"
test -z "$(sudo incus list --all-projects -c n -f csv)"
sudo systemctl is-active --quiet boxweave.service boxweave-scoped-lifecycle.service boxweave-scoped-guard.service
runtime="$(sudo systemctl show incus.service boxweave-scoped-lifecycle.service boxweave-scoped-guard.service -p MainPID -p ActiveEnterTimestampMonotonic)"
work="$(mktemp -d -p /tmp bw-v1-s2-ghci.XXXXXX)"
chmod 0711 "$work"
cp "$n2/s2-p1-list-debugger-probe" "$work/s2-p1-list-debugger-probe"
sudo python3 - <<'CHECK'
import sqlite3
with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as c:
    assert not c.execute("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')").fetchall()
    assert not c.execute("SELECT id FROM operations WHERE work_ref LIKE 'lab/p1-debugger-list-%'").fetchall()
CHECK
sudo systemctl stop boxweave.service
test "$(sudo systemctl show boxweave.service -p MainPID --value)" = 0
BW_S2_INSTALLED_SCOPED_AUTH=1 BW_S2_AUTH_CANDIDATE_SHA="$sha" \
  sh scripts/s2_p1_operation_list_boundary.sh "$work" "$project" /var/lib/incus/unix.socket.user
sudo systemctl start boxweave.service
for n in $(seq 1 40); do
  if test "$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 http://127.0.0.1:18650/mcp || :)" = 401; then break; fi
  sleep 1
done
test "$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 http://127.0.0.1:18650/mcp)" = 401
sudo env GITHUB_ACTIONS="$GITHUB_ACTIONS" RUNNER_ENVIRONMENT="$RUNNER_ENVIRONMENT" N2_EXACT_HEAD="$N2_EXACT_HEAD" \
  python3 scripts/pr191_scoped_private_reads.py "$project" "$receipt" "$sha"
# Reuse the original long-poll and signed-cursor SDK against the unchanged
# installed controller. No native fixtures, alternate listener or new DB.
CGO_ENABLED=1 go build -trimpath -o "$work/s2-token-inflight-probe" ./scripts/s2_p1_token_inflight_probe.go
lab="$work/p1-token-inflight"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
umask 077
oracle() {
  sudo env GITHUB_ACTIONS="$GITHUB_ACTIONS" RUNNER_ENVIRONMENT="$RUNNER_ENVIRONMENT" N2_EXACT_HEAD="$N2_EXACT_HEAD" \
    python3 scripts/pr191_scoped_auth_poll_oracle.py "$project" "$receipt" "$sha" "$@"
}
oracle before > "$work/auth-poll-before.json"
sudo runuser -u boxweave -- env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  BW_S2_INSTALLED_SCOPED_AUTH=1 BW_S2_AUTH_LAB="$lab" BW_S2_AUTH_PROJECT="$project" \
  "$work/s2-token-inflight-probe" > "$work/auth-poll-results.txt"
oracle after "$work/auth-poll-before.json" "$work/auth-poll-results.txt"
cat "$work/auth-poll-results.txt"
test "$runtime" = "$(sudo systemctl show incus.service boxweave-scoped-lifecycle.service boxweave-scoped-guard.service -p MainPID -p ActiveEnterTimestampMonotonic)"
