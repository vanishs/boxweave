#!/bin/sh
# CI-only P1 two actual BoxWeave controllers: unique nonroot service OS users,
# independent auto-restricted Incus Projects/credentials, independent SQLite.
# Refuses shared hosts, inherited Instances, and original UNKNOWN changes.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] ||
   [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then
  echo "S2_P1_DUAL_BLOCKED: dedicated CI only" >&2
  exit 91
fi
work="$1"
project="$2"
socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$socket" = /var/lib/incus/unix.socket.user
test "$project" = "user-$(id -u boxweave)"
test -z "$(sudo incus list --all-projects --format csv -c n)"
peer_user=boxweave-s2peer
sudo useradd --create-home --gid incus --shell /usr/sbin/nologin "$peer_user"
peer_uid="$(id -u "$peer_user")"
peer_project="user-$peer_uid"
test "$peer_project" != "$project"

actor_a() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor_b() {
  sudo runuser -u "$peer_user" -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    HOME="/home/$peer_user" XDG_CONFIG_HOME="/home/$peer_user/.config" "$@"
}
# Establish the second authorized Incus per-UID restricted Project as the
# second service OS identity, never as the first controller or as host admin.
actor_b incus project list --format csv > "$work/dualctrl-peer-projects.txt"
sudo incus project show "$peer_project" > "$work/dualctrl-peer-project.txt"
sudo incus project show "$project" > "$work/dualctrl-first-project.txt"
sudo incus project show "$peer_project" | grep -q 'restricted: "true"'
if actor_a incus project show "$peer_project" > /dev/null 2>&1; then
  echo "S2_P1_DUAL_FAIL: first controller can read peer Project" >&2
  exit 93
fi
if actor_b incus project show "$project" > /dev/null 2>&1; then
  echo "S2_P1_DUAL_FAIL: second controller can read first Project" >&2
  exit 94
fi
echo "S2_P1_TWO_RESTRICTED_OS_USERS_DENY_FOREIGN_INCUS_PROJECT=PASS"

# The second Project gets its own effective profile, bound to the same already
# authorized one-use dedicated quota pool and private-deny test ACL. Do not
# edit daemon-global/default Project networking or widen any privilege.
sudo incus project set "$peer_project" restricted.containers.nesting=allow
sudo incus profile device set default root pool=bw-s2-native-pool --project "$peer_project"
sudo incus profile device set default eth0 --project "$peer_project" \
  security.acls=bw-s2-native-acl \
  security.acls.default.ingress.action=reject \
  security.ipv4_filtering=true \
  security.ipv6_filtering=true \
  security.mac_filtering=true
sudo incus image copy images:ubuntu/24.04 local: --target-project "$peer_project" \
  --alias boxweave-ubuntu-24.04
sudo incus image info boxweave-ubuntu-24.04 --project "$peer_project" > /dev/null

dual_a="$work/dualctrl-a"
dual_b="$work/dualctrl-b"
mkdir -m 0700 "$dual_a" "$dual_b"
sudo chown boxweave:incus "$dual_a"
sudo chown "$peer_user":incus "$dual_b"
adb="$dual_a/ledger.sqlite"
bdb="$dual_b/ledger.sqlite"
actor_a "$work/boxweave" token-create -db "$adb" -role admin > "$work/dualctrl-a.secret"
actor_b "$work/boxweave" token-create -db "$bdb" -role admin > "$work/dualctrl-b.secret"
chmod 0600 "$work/dualctrl-a.secret" "$work/dualctrl-b.secret"
atoken="$(sed -n 's/^token=//p' "$work/dualctrl-a.secret")"
btoken="$(sed -n 's/^token=//p' "$work/dualctrl-b.secret")"
test -n "$atoken" && test -n "$btoken" && test "$atoken" != "$btoken"
a_pid=""
b_pid=""
cleanup() {
  sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $adb" || :
  sudo pkill -TERM -u "$peer_user" -f "^$work/boxweave serve -db $bdb" || :
  if [ -n "$a_pid" ]; then wait "$a_pid" 2>/dev/null || :; fi
  if [ -n "$b_pid" ]; then wait "$b_pid" 2>/dev/null || :; fi
}
trap cleanup EXIT
actor_a "$work/boxweave" serve -db "$adb" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18660 > "$work/dualctrl-a.redacted.log" 2>&1 &
a_pid=$!
actor_b "$work/boxweave" serve -db "$bdb" -incus-socket "$socket" -project "$peer_project" \
  -listen 127.0.0.1:18661 > "$work/dualctrl-b.redacted.log" 2>&1 &
b_pid=$!
for port in 18660 18661; do
  attempt=0
  until [ "$attempt" -ge 45 ]; do
    code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:$port/mcp" 2>/dev/null || :)"
    if [ "$code" = 401 ]; then break; fi
    attempt=$((attempt + 1))
    sleep 1
  done
  if [ "$attempt" -ge 45 ]; then
    echo "S2_P1_DUAL_FAIL: independent service failed on $port" >&2
    exit 95
  fi
done
# Authenticated on its own port, but each token is refused by the OTHER
# controller's independently issued database even before SDK tool dispatch.
for pair in "18660:$btoken" "18661:$atoken"; do
  port="${pair%%:*}"
  token="${pair#*:}"
  code="$(curl -sS -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer $token" \
    -X POST "http://127.0.0.1:$port/mcp" 2>/dev/null)"
  test "$code" = 401
done
echo "S2_P1_TWO_CONTROLLER_CROSS_TOKEN_AUTH_DENIED=PASS"

GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  BW_S2_DUAL_A_URL=http://127.0.0.1:18660/mcp BW_S2_DUAL_A_TOKEN="$atoken" \
  BW_S2_DUAL_B_URL=http://127.0.0.1:18661/mcp BW_S2_DUAL_B_TOKEN="$btoken" \
  BW_S2_DUAL_WITNESS_DIR="$work" "$work/s2-two-controllers-probe" \
  > "$work/dualctrl-probe.log" 2>&1 &
probe_pid=$!
attempt=0
while [ ! -s "$work/dualctrl.ready" ] && [ "$attempt" -lt 160 ]; do
  if ! kill -0 "$probe_pid" 2>/dev/null; then
    cat "$work/dualctrl-probe.log"
    echo "S2_P1_DUAL_FAIL: native two-controller prober exited early" >&2
    exit 96
  fi
  attempt=$((attempt + 1))
  sleep 1
done
if [ ! -s "$work/dualctrl.ready" ]; then
  cat "$work/dualctrl-probe.log"
  echo "S2_P1_DUAL_FAIL: independent witness did not start" >&2
  exit 97
fi
uuid_a="$(sed -n '1p' "$work/dualctrl.ready")"
uuid_b="$(sed -n '2p' "$work/dualctrl.ready")"
test -n "$uuid_a" && test -n "$uuid_b" && test "$uuid_a" != "$uuid_b"
name=bw-s2-dualctrl-owned-110f
test "$(sudo incus config get "$name" volatile.uuid --project "$project")" = "$uuid_a"
test "$(sudo incus config get "$name" volatile.uuid --project "$peer_project")" = "$uuid_b"
test "$(sudo incus list --project "$project" --format csv -c n)" = "$name"
test "$(sudo incus list --project "$peer_project" --format csv -c n)" = "$name"
echo "S2_P1_SECOND_CONTROLLER_INDEPENDENT_ADMIN_SAME_NAME_DISTINCT_UUID=PASS"
touch "$work/dualctrl.release"
if ! wait "$probe_pid"; then
  cat "$work/dualctrl-probe.log"
  echo "S2_P1_DUAL_FAIL: original MCP operations did not independently finish" >&2
  exit 98
fi
grep -Fqx S2_P1_TWO_REAL_OS_IDENTITIES_TWO_CONTROLLER_PROJECTS_SAME_NAME_NO_CROSS_ACCESS=PASS \
  "$work/dualctrl-probe.log"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_TWO_CONTROLLER_INDEPENDENT_NATIVE_AND_MCP_ISOLATION=PASS"
