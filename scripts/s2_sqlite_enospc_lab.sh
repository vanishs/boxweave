#!/bin/sh
# Original S2 private SQLite ENOSPC witness. Shared with clean N2.
# This is not an installed-native ACK or power-loss test.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
test "$#" -eq 4
work="$1"; project="$2"; candidate="$3"; probe="$4"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test -x "$candidate" && test -x "$probe"
test ! -e "$work/s2-sqlite-full-loop.img"
test ! -e "$work/sqlite-full"
# A separate ~128MiB ext4 loop device supplies an actual SQLite FULL
# fault: unlike WAL SIGKILL, this NativeAck cannot durably write while
# free space is exhausted. The guest/Incus native effects are never
# re-dispatched; the original is only a controller-local journal fixture.
test -z "$(sudo incus list --all-projects --format csv -c n)"
host_space="$(df -B1 --output=avail / | tail -n1 | tr -d ' ')"
test "$host_space" -gt 8589934592
sqlite_full_backing="$work/s2-sqlite-full-loop.img"
truncate -s 128M "$sqlite_full_backing"
sqlite_full_loop="$(sudo losetup --find --show "$sqlite_full_backing")"
sudo mkfs.ext4 -F -q "$sqlite_full_loop"
sqlite_full_dir="$work/sqlite-full/fsync-ledger"
mkdir -p "$sqlite_full_dir"
# The caller uses umask 077; traverse only this test-specific parent.
chmod 0711 "$work/sqlite-full"
sudo mount -t ext4 -o nodev,nosuid "$sqlite_full_loop" "$sqlite_full_dir"
sudo chown boxweave:incus "$sqlite_full_dir"
sudo chmod 0700 "$sqlite_full_dir"
test "$(sudo stat -Lc '%d' "$sqlite_full_dir")" != "$(stat -Lc '%d' /)"
sqlite_full_db="$sqlite_full_dir/ledger.sqlite"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$candidate" token-create -db "$sqlite_full_db" \
  > "$work/sqlite-full-admin.secret"
chmod 0600 "$work/sqlite-full-admin.secret"
sqlite_full_token="$(sed -n 's/^token=//p' "$work/sqlite-full-admin.secret")"
test -n "$sqlite_full_token"
s2_sqlite_full() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    BW_S2_FSYNC_DB="$sqlite_full_db" BW_S2_FSYNC_TOKEN="$sqlite_full_token" \
    BW_S2_FSYNC_PROJECT="$project" BW_S2_FSYNC_ARM_FILE="$sqlite_full_dir/unused" \
    "$probe" "$1"
}
s2_sqlite_full prepare > "$work/sqlite-full-prepare.txt"
grep -Eq '^S2_FSYNC_ORIGINAL_OPERATION_ID=[0-9a-f]{32}$' "$work/sqlite-full-prepare.txt"
s2_sqlite_full diskfull > "$work/sqlite-full-native-ack.txt"
grep -Fqx 'S2_SQLITE_REAL_SEPARATE_EXT4_DISK_FULL_NATIVE_ACK_REFUSED=PASS' "$work/sqlite-full-native-ack.txt"
sudo test -f "$sqlite_full_dir/.s2-sqlite-diskfull-only-filler"
sudo rm -f -- "$sqlite_full_dir/.s2-sqlite-diskfull-only-filler"
sudo test ! -e "$sqlite_full_dir/.s2-sqlite-diskfull-only-filler"
s2_sqlite_full verify > "$work/sqlite-full-verify.txt"
grep -Fqx "S2_SQLITE_WAL_FSYNC_REAL_KILL_UNKNOWN_NO_REPLAY=PASS" \
  "$work/sqlite-full-verify.txt"
cat "$work/sqlite-full-prepare.txt" "$work/sqlite-full-native-ack.txt" "$work/sqlite-full-verify.txt"
sudo umount "$sqlite_full_dir"
sudo losetup -d "$sqlite_full_loop"
echo "S2_SQLITE_ACTUAL_ISOLATED_EXT4_ENOSPC_RECOVERY_UNKNOWN_NO_REPLAY=PASS"
