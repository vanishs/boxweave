# Sourced only before fresh Scoped review on a one-use hosted VM.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "${P1B_S2_NOQUOTA:-}" = 1
test "${PR178_SCOPED_N2:-}" = 1
test "$(id -u)" != 0
case "${work:?}" in /tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$(realpath "$work")" = "$work"
test "$(stat -c '%u' "$work")" = "$(id -u)"
test "$(git rev-parse HEAD)" = "${N2_EXACT_HEAD:?}"
test "$project" = "user-$(id -u boxweave)"
test ! -e /var/lib/boxweave
test ! -e /usr/local/bin/boxweave
test ! -e "$work/noquota-only.img"
test ! -e "$work/noquota-mounted"
test -z "$(sudo incus list --all-projects --format csv -c n)"
if [ ! -e /dev/loop-control ]; then sudo modprobe loop; fi
test "$(df -B1 --output=avail / | tail -n1 | tr -d ' ')" -gt 8589934592
# 4GiB for the same installed common H3/H4 smoke, no host root quota test.
truncate -s 4G "$work/noquota-only.img"
noquota_loop="$(sudo losetup --find --show "$work/noquota-only.img")"
test -n "$noquota_loop"
sudo mkfs.ext4 -F -q -O ^quota,^project "$noquota_loop"
mkdir -m 0755 "$work/noquota-mounted"
sudo mount -t ext4 -o nodev,nosuid "$noquota_loop" "$work/noquota-mounted"
test "$(findmnt -n -o TARGET --target "$work/noquota-mounted")" = "$work/noquota-mounted"
test "$(findmnt -n -o FSTYPE --target "$work/noquota-mounted")" = ext4
test "$(stat -Lc '%d' "$work/noquota-mounted")" != "$(stat -Lc '%d' /)"
features="$(sudo tune2fs -l "$noquota_loop" | sed -n 's/^Filesystem features:[[:space:]]*//p')"
test -n "$features"
if printf '%s\n' "$features" | grep -Ew 'project|quota'; then exit 88; fi
if findmnt -n -o OPTIONS --mountpoint "$work/noquota-mounted" | grep -Eq '(^|,)(prjquota|pquota|usrquota|grpquota)(,|$)'; then exit 88; fi
sudo mkdir -m 0755 "$work/noquota-mounted/native-pool"
sudo incus storage create bw-s2-noquota-pool dir source="$work/noquota-mounted/native-pool"
sudo incus profile device set default root pool=bw-s2-noquota-pool --project "$project"
test "$(sudo incus profile device get default root pool --project "$project")" = bw-s2-noquota-pool
echo P1B_SCOPED_NOQUOTA_SEPARATE_EXT4_NO_PROJECT_QUOTA=PASS
