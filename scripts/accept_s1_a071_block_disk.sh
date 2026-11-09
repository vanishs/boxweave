#!/bin/sh
# S1 A-071: native block-device test on a disposable GitHub-hosted Ubuntu VM.
# It may NEVER be run against an installed/shared/production Incus host.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] || [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ]; then
  echo "BLOCKED: only GitHub-hosted disposable VM runners are authorized" >&2
  exit 81
fi
if [ "$(id -u)" -eq 0 ] || ! sudo -n true; then
  echo "BLOCKED: nonroot CI runner with sudo required" >&2
  exit 82
fi
if ! grep -q '^ID=ubuntu$' /etc/os-release || ! grep -Eq '^VERSION_ID="?24\.04"?$' /etc/os-release; then
  echo "BLOCKED: exact Ubuntu 24.04 required" >&2
  exit 83
fi

# No foreign native users or instances may exist on the one-use runner.
if [ -d /var/lib/incus ] || id boxweave >/dev/null 2>&1; then
  echo "BLOCKED: pre-existing native data/service identity on runner" >&2
  exit 84
fi
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends incus incus-client dnsmasq-base e2fsprogs util-linux >/dev/null
sudo systemctl is-active --quiet incus
test -z "$(sudo incus list --all-projects --format csv -c n)"
test "$(sudo incus project list --format csv | wc -l)" -eq 1

work="$(mktemp -d -p /tmp bw-s1-a071-disk.XXXXXX)"
chmod 0711 "$work"
source_dir="$work/mounted-block"
mkdir -p "$source_dir" "$work/ledger"
chmod 0700 "$work/ledger"
diskfile="$work/exclusive-disk.img"
truncate -s 384M "$diskfile"
# It is safe to load the loop module only within this explicitly ephemeral VM.
if [ ! -e /dev/loop-control ]; then sudo modprobe loop; fi
loop="$(sudo losetup --find --show "$diskfile")"
test -n "$loop"
sudo mkfs.ext4 -F -q "$loop"
sudo mount -t ext4 -o nodev,nosuid "$loop" "$source_dir"
test "$(findmnt -n -o FSTYPE --target "$source_dir")" = ext4
test "$(stat -c '%d' "$source_dir")" != "$(stat -c '%d' /)"
test "$(sudo blkid -s TYPE -o value "$loop")" = ext4
printf 'S1_A071_REAL_DISTINCT_VIRTUAL_BLOCK_DEVICE=PASS loop=%s\n' "$loop"
sudo sh -c "printf 's1-test-owned-block-device-v1\n' > '$source_dir/marker'"
marker_before="$(sudo sha256sum "$source_dir/marker" | awk '{print $1}')"

# Bootstrap only this runner. There are no native instances or extra Projects.
if ! sudo incus storage list --format csv | grep -q '^default,.*CREATED'; then
  sudo incus admin init --auto --storage-backend=dir
fi
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo incus storage create bw-s1-a071-block dir source="$source_dir"
sudo incus storage show bw-s1-a071-block | grep -F "source: $source_dir"
sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
uid="$(id -u boxweave)"
project="user-$uid"
sudo runuser -u boxweave -- incus project list --format csv
sudo incus project show "$project" | grep -q 'restricted: "true"'
test "$(sudo runuser -u boxweave -- id -nG)" = incus
if sudo incus profile show default --project "$project" | grep -q '^  root:'; then
  sudo incus profile device set default root pool=bw-s1-a071-block --project "$project"
else
  sudo incus profile device add default root disk path=/ pool=bw-s1-a071-block --project "$project"
fi
sudo incus profile show default --project "$project" | grep -q 'pool: bw-s1-a071-block'
test -z "$(sudo incus list --all-projects --format csv -c n)"

# Build the candidate from the *exact* checked-out commit, not from an old binary.
commit="$(git rev-parse HEAD)"
CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=1.0.23-s1-a071-ghaci -X main.commit=$commit" -o "$work/boxweave" ./cmd/boxweave
chmod 0755 "$work/boxweave"
sudo chown boxweave:incus "$work/ledger"
bin="$work/boxweave"
db="$work/ledger/ledger.sqlite"
sudo runuser -u boxweave -- "$bin" build-info
echo "S1_A071_TESTED_SOURCE_SHA=$commit"
sha256sum "$bin"

# Keep all readback in the actual unprivileged Project; no root MCP worker.
sudo runuser -u boxweave -- "$bin" doctor -db "$db" -project "$project" > "$work/positive.json"
python3 - "$work/positive.json" <<'PY'
import json,sys
n=json.load(open(sys.argv[1]))["native"]
assert n["ready"] and n["selected_storage_pool_ready"]
assert n["selected_storage_separate_mount"] and n["selected_storage_space_checked"]
assert 64*(1<<20) < n["selected_storage_available_bytes"] < 384*(1<<20)
print("S1_A071_NATIVE_BLOCK_POOL_POSITIVE=PASS actual_bytes="+str(n["selected_storage_available_bytes"]))
PY

# Incorrect host-free-space claim: block loop is limited to 384 MiB while /
# has much more. Fill only the owned loop filesystem until <64 MiB remains.
available="$(df -B1 "$source_dir" | awk 'NR==2 {print $4}')"
fill="$((available - 48*1024*1024))"
test "$fill" -gt 0
sudo fallocate -l "$fill" "$source_dir/only-test-fill"
if sudo runuser -u boxweave -- "$bin" doctor -db "$db" -project "$project" > "$work/low-space.json" 2>"$work/low-space.err"; then
  echo "FAIL: doctor advertised ready on exhausted native block storage" >&2
  exit 85
fi
python3 - "$work/low-space.json" <<'PY'
import json,sys
n=json.load(open(sys.argv[1]))["native"]
assert not n["ready"] and n["error_class"]=="SELECTED_STORAGE_SPACE_INSUFFICIENT"
assert 0 <= n["selected_storage_available_bytes"] < 64*(1<<20)
print("S1_A071_REAL_BLOCK_LOW_SPACE_REFUSED=PASS free_bytes="+str(n["selected_storage_available_bytes"]))
PY
sudo rm "$source_dir/only-test-fill"

# Root in a *private* mount namespace can fault only the CI VM's selected
# canonical pool. The global mount, Incus, and foreign resources are untouched.
sudo unshare --mount --propagation private sh -c '
  set -eu
  umount /var/lib/incus/storage-pools/bw-s1-a071-block
  if runuser -u boxweave -- "$1" doctor -db "$2" -project "$3" >"$4" 2>"$5"; then
    echo "FAIL: doctor falsely ready without selected block pool mount" >&2
    exit 86
  fi
' sh "$bin" "$db" "$project" "$work/missing.json" "$work/missing.err"
python3 - "$work/missing.json" <<'PY'
import json,sys
n=json.load(open(sys.argv[1]))["native"]
assert not n["ready"] and n["error_class"]=="SELECTED_STORAGE_BACKING_UNVERIFIED"
print("S1_A071_REAL_BLOCK_MOUNT_ABSENT_REFUSED=PASS")
PY
test "$(findmnt -n -o FSTYPE --target /var/lib/incus/storage-pools/bw-s1-a071-block)" = ext4

# Persistent virtual block source survives real Incus daemon restart; the
# original filesystem marker and project pool mapping are independently read.
sudo systemctl restart incus
sudo systemctl is-active --quiet incus
sudo incus profile show default --project "$project" | grep -q 'pool: bw-s1-a071-block'
test -z "$(sudo incus list --all-projects --format csv -c n)"
test "$(sudo sha256sum "$source_dir/marker" | awk '{print $1}')" = "$marker_before"
test "$(findmnt -n -o FSTYPE --target "$source_dir")" = ext4
test "$(findmnt -n -o FSTYPE --target /var/lib/incus/storage-pools/bw-s1-a071-block)" = ext4
sudo runuser -u boxweave -- "$bin" doctor -db "$db" -project "$project" > "$work/recovered.json"
python3 - "$work/recovered.json" <<'PY'
import json,sys
n=json.load(open(sys.argv[1]))["native"]
assert n["ready"] and n["selected_storage_separate_mount"] and n["selected_storage_space_checked"]
print("S1_A071_PERSISTENT_LOOP_BLOCK_AFTER_NATIVE_RESTART=PASS")
PY

echo "A071_DISTINCT_PERSISTENT_BLOCK_DISK_NATIVE_NEGATIVES_AND_RECOVERY=PASS"
