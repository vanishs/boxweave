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
cleanup() {
 sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
 if [ -n "$service_pid" ]; then wait "$service_pid" 2>/dev/null || :; fi
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
  BW_S2_ALT_TOKEN="$token" "$work/s2-altpool-probe" > "$work/p2-altpool-results.txt"
grep -Fqx S2_P2_ALTERNATE_SELECTED_PROFILE_POOL_DEFAULT_TEN_GIB=PASS "$work/p2-altpool-results.txt"
grep -Fqx S2_P2_ALTERNATE_POOL_NO_EXPLICIT_SELECTION_NO_RESIDUAL_BOX=PASS "$work/p2-altpool-results.txt"
name=bw-s2-alternate-pool-110w
uuid="$(sudo python3 - "$lab/original.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert len(v["box_id"])==36 and len(v["create_id"])==32
print(v["box_id"])
PY
)"
# Native independent readback after the original Box was deleted:
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
