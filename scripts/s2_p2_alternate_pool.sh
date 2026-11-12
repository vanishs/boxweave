#!/bin/sh
# A SECOND independently mounted ext4/quota Incus dir pool on one-use CI.
# The restricted Project's default Profile is temporarily changed only
# while no Box exists. No public root_pool selection is supported.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] || [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
  [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
test "$(sudo incus profile device get default root pool --project "$project")" = bw-s2-native-pool
test "$(df -B1 --output=avail / | tail -n1 | tr -d ' ')" -gt 3221225472
alt="$work/p2-alt-block"
mkdir -m 0755 "$alt"
truncate -s 12G "$alt/alternative-disk.img"
loop="$(sudo losetup --find --show "$alt/alternative-disk.img")"
test -n "$loop"
sudo mkfs.ext4 -F -q -O quota,project "$loop"
sudo mkdir -m 0755 "$alt/mounted"
sudo mount -t ext4 -o nodev,nosuid,prjquota "$loop" "$alt/mounted"
sudo tune2fs -l "$loop" | grep -E '^Filesystem features:' | grep -qw project
findmnt -n -o OPTIONS --target "$alt/mounted" | grep -Eq '(^|,)prjquota(,|$)'
test "$(stat -c '%d' "$alt/mounted")" != "$(stat -c '%d' "$work/mounted")"
sudo mkdir -m 0755 "$alt/mounted/storage"
pool=bw-s2-p2-alternate-pool
sudo incus storage create "$pool" dir source="$alt/mounted/storage"
test "$(sudo incus storage get "$pool" source)" = "$alt/mounted/storage"
# Profile switch is scoped to ONLY this temporary restricted Project, and
# occurs with verified empty inventory. No foreign Project is inspected.
sudo incus profile device set default root pool="$pool" --project "$project"
test "$(sudo incus profile device get default root pool --project "$project")" = "$pool"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P2_ALTERNATE_MOUNTED_EXT4_QUOTA_PROFILE_SELECTED=PASS"

lab="$work/p2-alt-pool"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
db="$lab/ledger.sqlite"
actor() {
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor "$work/boxweave" token-create -db "$db" -role admin > "$work/p2-alt-token.secret"
chmod 0600 "$work/p2-alt-token.secret"
token="$(sed -n 's/^token=//p' "$work/p2-alt-token.secret")"
test -n "$token"
service_pid=""
probe_pid=""
cleanup() {
 sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
 if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; fi
 if [ -n "$probe_pid" ]; then
  kill "$probe_pid" 2>/dev/null || :
  wait "$probe_pid" 2>/dev/null || :
 fi
}
trap cleanup EXIT
actor "$work/boxweave" serve -db "$db" -incus-socket "$socket" \
  -project "$project" -listen 127.0.0.1:18666 \
  > "$work/p2-alternate-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 45 ]; do
 status="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18666/mcp 2>/dev/null || :)"
 if [ "$status" = 401 ]; then break; fi
 attempt=$((attempt+1)); sleep 1
done
test "$attempt" -lt 45
actor BW_S2_ALT_LAB="$lab" BW_S2_ALT_PROJECT="$project" \
  BW_S2_ALT_TOKEN="$token" "$work/s2-altpool-probe" > "$work/p2-altpool-results.txt" 2>&1 &
probe_pid=$!
attempt=0
while [ "$attempt" -lt 100 ] && ! sudo test -s "$lab/live.ready"; do
 if ! kill -0 "$probe_pid" 2>/dev/null; then
  cat "$work/p2-altpool-results.txt"
  echo "S2_P2_ALT_POOL_FAIL: SDK probe ended before live native witness" >&2
  exit 95
 fi
 attempt=$((attempt + 1))
 sleep 1
done
if ! sudo test -s "$lab/live.ready"; then
 cat "$work/p2-altpool-results.txt"
 echo "S2_P2_ALT_POOL_FAIL: live exact UUID never exposed" >&2
 exit 96
fi
name=bw-s2-alternate-pool-110w
# Independent administrator reads the actual Incus expanded device while
# the Box remains alive, before the MCP client is permitted to delete it.
sudo incus list --project "$project" --format json > "$work/p2-alt-live-native.json"
sudo python3 - "$work/p2-alt-live-native.json" "$lab/original.json" "$pool" "$name" <<'PY'
import json,sys
items=json.load(open(sys.argv[1]))
original=json.load(open(sys.argv[2]))
pool,name=sys.argv[3:]
assert len(items)==1 and items[0]["name"]==name, items
inst=items[0]
assert inst["config"]["volatile.uuid"]==original["box_id"], "native UUID differs"
root=[dev for dev in inst["expanded_devices"].values()
      if dev.get("type")=="disk" and dev.get("path")=="/"]
assert len(root)==1 and root[0].get("pool")==pool, root
assert root[0].get("size")=="10GiB", root
assert inst["status"]=="Stopped", inst
print("S2_P2_ALTERNATE_POOL_LIVE_EXPANDED_ROOT_MATCHES_UUID_POOL_QUOTA=PASS")
PY
uuid="$(sudo python3 - "$lab/original.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert len(v["box_id"])==36 and len(v["create_id"])==32
print(v["box_id"])
PY
)"
sudo touch "$lab/native-verified.release"
if ! wait "$probe_pid"; then
 cat "$work/p2-altpool-results.txt"
 echo "S2_P2_ALT_POOL_FAIL: exact native MCP removal failed" >&2
 exit 97
fi
probe_pid=""
grep -Fqx S2_P2_ALTERNATE_SELECTED_PROFILE_POOL_DEFAULT_TEN_GIB=PASS "$work/p2-altpool-results.txt"
grep -Fqx S2_P2_ALTERNATE_POOL_NO_EXPLICIT_SELECTION_NO_RESIDUAL_BOX=PASS "$work/p2-altpool-results.txt"
# Native independent readback after the verified original was deleted:
# no residual instance may remain in either the alternate or foreign pool.
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P2_ALTERNATE_POOL_NATIVE_UUID_JOURNAL_WITNESS=$uuid"

sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
wait "$service_pid" 2>/dev/null || :
service_pid=""
sudo incus profile device set default root pool=bw-s2-native-pool --project "$project"
test "$(sudo incus profile device get default root pool --project "$project")" = bw-s2-native-pool
sudo incus storage delete "$pool"
sudo umount "$alt/mounted"
sudo losetup -d "$loop"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P2_ALTERNATE_ROOT_POOL_RESTORED_NO_RESIDUAL_STORAGE=PASS"
