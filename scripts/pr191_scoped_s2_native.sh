#!/bin/sh
# Full original S2 MCP/packet probe on the installed Scoped service.
# Only the clean N2 runner invokes this. Do not replay or clean UNKNOWN.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
test "$#" -eq 5
project="$1"; receipt="$2"; candidate_sha="$3"; probe="$4"; seed="$5"
test "$project" = "user-$(id -u boxweave)"
test "$(sudo sha256sum /usr/local/bin/boxweave | cut -d' ' -f1)" = "$candidate_sha"
test -x "$probe" && test -x "$seed"
test "$(stat -c '%a' "$receipt")" = 600
n2_work="$(dirname "$receipt")"
case "$n2_work" in /tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$receipt" = "$n2_work/actor.receipt"
test "$probe" = "$n2_work/s2-native-sdk-probe"
test "$seed" = "$n2_work/s2-queued-cancel-seed"
test "$(sha256sum "$n2_work/candidate-a" | cut -d' ' -f1)" = "$candidate_sha"
test "$(git rev-parse HEAD)" = "${N2_EXACT_HEAD:?}"
db=/var/lib/boxweave/ledger.sqlite
runtime="$(sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_scoped_s2_native.py "$project" "$candidate_sha" before)"
sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_scoped_schema_refusal.py "$project" "$candidate_sha"
work="$(mktemp -d -p /tmp bw-v1-s2-ghci.XXXXXX)"
chmod 0711 "$work"
umask 077
installed=/usr/local/bin/boxweave
admin="$(sed -n 's/^token=//p' "$receipt")"
admin_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$receipt")"
test -n "$admin"
for role in peer admin2 revoked expired; do
  ttl=30m
  test "$role" != expired || ttl=1s
  sudo runuser -u boxweave -- "$installed" token-create -project "$project" -ttl "$ttl" > "$work/$role.secret"
done
peer="$(sed -n 's/^token=//p' "$work/peer.secret")"
admin2="$(sed -n 's/^token=//p' "$work/admin2.secret")"
admin2_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/admin2.secret")"
revoked="$(sed -n 's/^token=//p' "$work/revoked.secret")"
revoked_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/revoked.secret")"
expired="$(sed -n 's/^token=//p' "$work/expired.secret")"
sudo runuser -u boxweave -- "$installed" token-revoke -project "$project" -id "$revoked_id" >/dev/null
test "$admin_id" != "$admin2_id"
for id in "$admin_id" "$admin2_id" "$revoked_id"; do
  printf '%s\n' "$id" | grep -Eq '^[0-9a-f]{32}$'
done
seed_history() {
  sudo runuser -u boxweave -- env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    BW_S2_INSTALLED_SCOPED_NATIVE=1 BW_S2_SEED_DB="$db" BW_S2_SEED_PROJECT="$project" \
    BW_S2_SEED_ADMIN_TOKEN="$admin" "$seed" "$@"
}
seed_history > "$work/queued-cancel-id.txt"
queued="$(sed -n '1p' "$work/queued-cancel-id.txt")"
foreign="$(sed -n '2p' "$work/queued-cancel-id.txt")"
test "$(wc -l < "$work/queued-cancel-id.txt")" = 2
for id in "$queued" "$foreign"; do
  printf '%s\n' "$id" | grep -Eq '^[0-9a-f]{32}$'
done
test "$queued" != "$foreign"
test -z "$(sudo incus list --all-projects -c n -f csv)"

# Scoped Root lifecycle deliberately refuses ANY foreign native inventory.
# The foreign fixture is checked after positive lifecycle, never by changing
# the global inventory guard. Earlier c843166 failure is retained evidence.
(umask 022; sh scripts/s2_p1_simulated_wan_lab.sh setup "$work" "$project")
ca="$(base64 -w0 "$work/s2-simwan-cert.pem")"
test -n "$ca"
# Host direct TLS is a separate environment control, never Guest PASS.
host_status=UNVERIFIED
if host_code="$(curl -q -4 --noproxy '*' --proto '=https' --proto-redir '=https' -L -fsS --connect-timeout 4 --max-time 8 -o /dev/null -w '%{http_code}' https://example.com)" && printf '%s\n' "$host_code" | grep -Ex '2[0-9]{2}' >/dev/null; then
  host_status=PASS
fi
echo "P1B_SCOPED_PUBLIC_IPV4_TLS_HOST_CONTROL=$host_status"
start_log="$(date --iso-8601=seconds)"
BW_S2_INSTALLED_SCOPED_NATIVE=1 BW_S2_MCP_URL=http://127.0.0.1:18650/mcp \
  BW_S2_ADMIN_TOKEN="$admin" BW_S2_PEER_TOKEN="$peer" BW_S2_ADMIN2_TOKEN="$admin2" \
  BW_S2_ADMIN1_ID="$admin_id" BW_S2_ADMIN2_ID="$admin2_id" \
  BW_S2_EXPIRED_TOKEN="$expired" BW_S2_REVOKED_TOKEN="$revoked" \
  BW_S2_FOREIGN_LAYER_OPID="$foreign" BW_S2_QUEUED_CANCEL_OPID="$queued" \
  BW_S2_SIMULATED_WAN_CA_CERT_B64="$ca" BW_S2_HISTORY_CURSOR_WITNESS_DIR="$work" \
  BW_S2_WAN_WITNESS_DIR="$work" BW_S2_OUTPUT_WITNESS_DIR="$work" "$probe" > "$work/probe.log" 2>&1 &
probe_pid=$!
# Do not restart or clean a failed native original. The VM itself is one-use.
wait_marker() {
  marker="$1"; attempts="$2"
  while test ! -s "$work/$marker" && test "$attempts" -gt 0; do
    kill -0 "$probe_pid" 2>/dev/null || break
    attempts=$((attempts-1))
    sleep 1
  done
  if test ! -s "$work/$marker"; then
    cat "$work/probe.log"
    sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_scoped_s2_native.py "$project" "$candidate_sha" failure || :
    echo "Original probe did not reach $marker" >&2
    exit 1
  fi
}
wait_marker history-cursor.ready 90
test "$(cat "$work/history-cursor.ready")" = first-page-observed
seed_history append > "$work/history-append.txt"
grep -Fxq S2_SQLITE_CONCURRENT_HISTORY_APPEND_12=PASS "$work/history-append.txt"
touch "$work/history-cursor.release"
echo S2_NATIVE_REAL_SQLITE_HISTORY_WRITER_DURING_MCP_PAGE=PASS
# Read-only snapshots bracket the same fixed guest HTTPS attempts.
# The exact pinned NIC/UUID must be independently observed. Counters are
# diagnostic, not a WAN positive or proof of cause; no policy is changed.
wan_snapshot() {
  phase="$1"
  wait_marker "wan-$phase.ready" 450
  read -r wan_uuid wan_ip < "$work/wan-$phase.ready"
  bridge="$(sudo incus profile device get default eth0 network --project "$project")"
  printf '%s\n' "$bridge" | grep -Eq '^[a-zA-Z0-9_.-]{1,15}$'
  sudo incus query "/1.0/instances/bw-s2-ghci-110a?project=$project" |
    python3 -c 'import ipaddress,json,sys,uuid; x=json.load(sys.stdin); u,a,b=sys.argv[1:]; assert str(uuid.UUID(u))==u and x["config"]["volatile.uuid"]==u; assert x["status"]=="Running" and str(ipaddress.IPv4Address(a))==a and x["expanded_devices"]["eth0"]["ipv4.address"]==a and x["expanded_devices"]["eth0"]["network"]==b' "$wan_uuid" "$wan_ip" "$bridge"
  nat="$(sudo incus network get "$bridge" ipv4.nat --project default)"
  forwarding="$(sysctl -n net.ipv4.ip_forward)"
  case "$nat" in true|false) ;; *) nat=UNVERIFIED ;; esac
  case "$forwarding" in 0|1) ;; *) forwarding=UNVERIFIED ;; esac
  echo "P1B_SCOPED_WAN_HOST_PATH $phase $wan_uuid nat=$nat forwarding=$forwarding"
  # Emit only base-chain policy/counters, not the full host ruleset.
  if sudo timeout 8 iptables -w 3 -nvx -L FORWARD > "$work/wan-$phase.forward" 2>/dev/null; then
    awk 'NR==1 && $1=="Chain" && $2=="FORWARD" && ($4=="ACCEPT" || $4=="DROP") && $5~/^[0-9]+$/ {print "P1B_SCOPED_WAN_HOST_FORWARD " phase " " $4 " " $5; ok=1} END{if(!ok) print "P1B_SCOPED_WAN_HOST_FORWARD " phase " UNVERIFIED"}' phase="$phase" "$work/wan-$phase.forward"
  else
    echo "P1B_SCOPED_WAN_HOST_FORWARD $phase UNVERIFIED"
  fi
  # Observe only the exact guest source and TCP/443; emit aggregate counts.
  if sudo timeout 8 conntrack -L -f ipv4 -p tcp -s "$wan_ip" --dport 443 > "$work/wan-$phase.conntrack" 2>/dev/null; then
    awk '$1=="tcp"{n++; if(/\[UNREPLIED\]/)u++; if(/ESTABLISHED/)e++} END{printf "P1B_SCOPED_WAN_HOST_CONNTRACK %s %d %d %d\n",phase,n,u,e}' phase="$phase" "$work/wan-$phase.conntrack"
  else
    echo "P1B_SCOPED_WAN_HOST_CONNTRACK $phase UNVERIFIED"
  fi
  sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_wan_host_observe.py "$phase"
  touch "$work/wan-$phase.release"
}
wan_snapshot before
wan_snapshot after
wait_marker output-witness.ready 450
native_id="$(cat "$work/output-witness.ready")"
printf '%s\n' "$native_id" | grep -Eq '^[0-9a-f-]{36}$'
# Read only the exact completed original's two native spool files. N2
# default pool need not be a separately mounted block/dir quota fixture.
# No allocation or EDQUOT inference here; original physical quota gate retained.
spool="/var/lib/incus/containers/${project}_bw-s2-ghci-110a/exec-output"
sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_scoped_s2_native.py "$project" "$candidate_sha" spool "$admin_id" "$native_id"
physical=UNVERIFIED
if sudo test -f "$spool/exec_$native_id.stdout" && sudo test -f "$spool/exec_$native_id.stderr"; then
  out_bytes="$(sudo stat -c %s "$spool/exec_$native_id.stdout")"
  err_bytes="$(sudo stat -c %s "$spool/exec_$native_id.stderr")"
  echo "P1B_SCOPED_S2_NATIVE_SPOOL_BYTES $native_id $out_bytes $err_bytes"
  test "$out_bytes" = 16777220 && test "$err_bytes" = 16777220
  physical=PASS
fi
echo "P1B_SCOPED_S2_PHYSICAL_SPOOL=$physical"
touch "$work/output-witness.release"
if ! wait "$probe_pid"; then
  cat "$work/probe.log"
  exit 1
fi
cat "$work/probe.log"
grep -Fxq P1B_SCOPED_S2_NATIVE_PROBE=PASS "$work/probe.log"
sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_scoped_s2_native.py "$project" "$candidate_sha" wan "$admin_id" "$work/probe.log"
sh scripts/s2_p1_simulated_wan_lab.sh verify-scoped "$work" "$project"
sh scripts/s2_p1_simulated_wan_lab.sh cleanup "$work" "$project"
sudo journalctl --unit boxweave.service --since "$start_log" --no-pager > "$work/service.redacted.log"
if grep -Fq S2_P1_META_REDACTION_SENTINEL_110 "$work/service.redacted.log"; then
  echo "Untrusted meta appeared in service log" >&2
  exit 1
fi
grep -Fxq S2_P1_REAL_SDK_HOSTILE_META_AND_SCHEMA_NO_NATIVE_EFFECT=PASS "$work/probe.log"
grep -Fxq S2_P1_REAL_SDK_LEGACY_TEXT_AND_STRUCTURED_RECEIPT_SAME_ID=PASS "$work/probe.log"
echo P1B_SCOPED_S2_REDACTION=PASS
test -z "$(sudo incus list --all-projects -c n -f csv)"
# Retain the original independent same-name native negative. A newly
# created, empty test Project has no shared NIC and no existing resources.
foreign_project=bw-s2-foreign-ci-project
foreign_name=bw-s2-ghci-110a
pool="$(sudo incus profile device get default root pool --project "$project")"
test -n "$pool"
sudo incus project create "$foreign_project" --config features.images=false
sudo incus profile device add default root disk path=/ pool="$pool" --project "$foreign_project"
sudo incus init --empty "$foreign_name" --project "$foreign_project"
foreign_uuid="$(sudo incus config get "$foreign_name" volatile.uuid --project "$foreign_project")"
printf '%s\n' "$foreign_uuid" | grep -Eq '^[0-9a-f-]{36}$'
sudo incus config show "$foreign_name" --project "$foreign_project" > "$work/foreign-before"
if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus info "$foreign_name" --project "$foreign_project" > "$work/foreign-denied" 2>&1; then
  echo "Restricted service unexpectedly accessed the isolated foreign fixture" >&2
  exit 1
fi
echo S2_P1_SECOND_REAL_PROJECT_SAME_NAME_NATIVE_INSTANCE_DENIED_TO_NONROOT=PASS
sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_scoped_s2_native.py "$project" "$candidate_sha" foreign "$receipt" "$foreign_uuid"
test "$(sudo incus config get "$foreign_name" volatile.uuid --project "$foreign_project")" = "$foreign_uuid"
sudo incus config show "$foreign_name" --project "$foreign_project" > "$work/foreign-after"
cmp "$work/foreign-before" "$work/foreign-after"
sudo incus query "/1.0/instances/$foreign_name?project=$foreign_project" | python3 -c 'import json,sys;x=json.load(sys.stdin);assert x["status"]=="Stopped" and x["config"]["volatile.uuid"]==sys.argv[1]' "$foreign_uuid"
echo P1B_SCOPED_S2_NATIVE_FOREIGN_UNCHANGED=PASS
sudo env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr191_scoped_s2_native.py "$project" "$candidate_sha" after "$admin_id" "$admin2_id" "$queued" "$foreign" "$runtime"
# Leave the one-use native negative intact for runner retirement. No
# administrative deletion to manufacture an empty inventory verdict.
