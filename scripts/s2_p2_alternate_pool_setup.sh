# Shared exact two-mount S2 alternate-pool setup, before Scoped fresh review.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
case "${work:?}" in /tmp/bw-v1-s2-ghci.*) ;; /tmp/bw-n2-installed.*)
  test "${P1B_S2_ALTPOOL:-0}" = 1
  test "${PR178_SCOPED_N2:-0}" = 1
  test "$(git rev-parse HEAD)" = "${N2_EXACT_HEAD:?}"
  test ! -e /var/lib/boxweave
  test ! -e /usr/local/bin/boxweave
  ;; *) exit 92 ;; esac
test "$(realpath "$work")" = "$work"
test "$(stat -c '%u' "$work")" = "$(id -u)"
test "$project" = "user-$(id -u boxweave)"
test -z "$(sudo incus list --all-projects --format csv -c n)"
test "$(sudo incus profile device get default root pool --project "$project")" = bw-s2-native-pool
test "$(sudo incus storage get bw-s2-native-pool source)" = "$work/mounted/native-pool"
test "$(findmnt -n -o TARGET --target "$work/mounted")" = "$work/mounted"
test "$(findmnt -n -o FSTYPE --mountpoint "$work/mounted")" = ext4
findmnt -n -o OPTIONS --mountpoint "$work/mounted" | grep -Eq '(^|,)prjquota(,|$)'
test "$(df -B1 --output=avail / | tail -n1 | tr -d ' ')" -gt 3221225472
alt="$work/p2-alt-block"
mkdir -m 0755 "$alt"
truncate -s 12G "$alt/alternative-disk.img"
alt_loop="$(sudo losetup --find --show "$alt/alternative-disk.img")"
test -n "$alt_loop"
sudo mkfs.ext4 -F -q -O quota,project "$alt_loop"
sudo mkdir -m 0755 "$alt/mounted"
sudo mount -t ext4 -o nodev,nosuid,prjquota "$alt_loop" "$alt/mounted"
sudo tune2fs -l "$alt_loop" | grep -E '^Filesystem features:' | grep -qw project
findmnt -n -o OPTIONS --target "$alt/mounted" | grep -Eq '(^|,)prjquota(,|$)'
test "$(stat -c '%d' "$alt/mounted")" != "$(stat -c '%d' "$work/mounted")"
test "$(stat -c '%d' "$alt/mounted")" != "$(stat -c '%d' /)"
test "$(findmnt -n -o SOURCE --mountpoint "$alt/mounted")" = "$alt_loop"
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
