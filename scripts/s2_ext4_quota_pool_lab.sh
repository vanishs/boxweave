# Shared original S2 quota fixture; no production entry point.
# Source from the guarded one-use hosted runner only.
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
case "${work:?}" in /tmp/bw-v1-s2-ghci.*|/tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$(realpath "$work")" = "$work"
test "$(stat -c '%u' "$work")" = "$(id -u)"

test ! -e /var/lib/boxweave
test ! -e /usr/local/bin/boxweave
test ! -e "$work/exclusive-block.img"
test ! -e "$work/mounted"
test -z "$(sudo incus list --all-projects --format csv -c n)"
# A real single-use mounted ext4 loop block device (NOT a host-root
# free-space proxy) is prerequisite to the restricted SDK native preflight.
if [ ! -e /dev/loop-control ]; then
  sudo modprobe loop
fi
# Loop backing is sparse on the disposable runner root: require real host
# headroom before allocating any guest/recorded data inside the 4GiB mount.
host_available_before="$(df -B1 --output=avail / | tail -n1 | tr -d ' ')"
test "$host_available_before" -gt 8589934592
echo "S2_NATIVE_DISPOSABLE_HOST_ROOT_FREE_SPACE_GUARD=PASS"
truncate -s 4G "$work/exclusive-block.img"
loop="$(sudo losetup --find --show "$work/exclusive-block.img")"
test -n "$loop"
# ext4 supports native Incus dir-volume project quotas only when the
# filesystem project/quota features and project-quota mount are enabled.
# A plain ext4 dir pool silently skips the root-disk size quota in Incus 7.6.
sudo mkfs.ext4 -F -q -O quota,project "$loop"
mkdir -m 0755 "$work/mounted"
sudo mount -t ext4 -o nodev,nosuid,prjquota "$loop" "$work/mounted"
sudo tune2fs -l "$loop" | grep -E '^Filesystem features:' | grep -qw project
findmnt -n -o OPTIONS --target "$work/mounted" | grep -Eq '(^|,)prjquota(,|$)'
echo "S2_NATIVE_EXT4_PROJECT_QUOTA_BACKING_CONFIGURED=PASS"
sudo mkdir -m 0755 "$work/mounted/native-pool"
sudo incus storage create bw-s2-native-pool dir source="$work/mounted/native-pool"
test "$(findmnt -n -o FSTYPE --target "$work/mounted")" = ext4
