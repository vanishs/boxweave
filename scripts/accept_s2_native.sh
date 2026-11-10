#!/bin/sh
# Real S2 native black-box candidate acceptance on one-use GitHub-hosted VM.
# Refuses non-CI, pre-existing/shared Incus or installed service identities.
set -eu

if [ "${GITHUB_ACTIONS:-}" != true ] || [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ]; then
  echo "S2_NATIVE_BLOCKED: single-use GH-hosted VM only" >&2
  exit 81
fi
if [ "$(id -u)" -eq 0 ] || ! sudo -n true; then
  echo "S2_NATIVE_BLOCKED: must run as nonroot CI runner with sudo" >&2
  exit 82
fi
if ! grep -q '^ID=ubuntu$' /etc/os-release || ! grep -Eq '^VERSION_ID="?24\.04"?$' /etc/os-release; then
  echo "S2_NATIVE_BLOCKED: requires Ubuntu 24.04 VM" >&2
  exit 83
fi
if [ -e /var/lib/incus ] || id boxweave >/dev/null 2>&1; then
  echo "S2_NATIVE_BLOCKED: existing Incus state/user; never adopt" >&2
  exit 84
fi

work="$(mktemp -d -p /tmp bw-v1-s2-ghci.XXXXXX)"
chmod 0711 "$work"
mkdir -m 0700 "$work/ledger"
echo "S2_NATIVE_CANDIDATE_SOURCE_SHA=$(git rev-parse HEAD)"

sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends incus incus-client dnsmasq-base e2fsprogs util-linux >/dev/null
sudo systemctl start incus.service
sudo systemctl is-active --quiet incus.service
test -z "$(sudo incus list --all-projects --format csv -c n)"
test "$(sudo incus project list --format csv | wc -l)" -eq 1
sudo incus admin init --auto --storage-backend=dir
test -z "$(sudo incus list --all-projects --format csv -c n)"

# A real single-use mounted ext4 loop block device (NOT a host-root
# free-space proxy) is prerequisite to the restricted SDK native preflight.
if [ ! -e /dev/loop-control ]; then
  sudo modprobe loop
fi
truncate -s 4G "$work/exclusive-block.img"
loop="$(sudo losetup --find --show "$work/exclusive-block.img")"
test -n "$loop"
sudo mkfs.ext4 -F -q "$loop"
mkdir -m 0755 "$work/mounted"
sudo mount -t ext4 -o nodev,nosuid "$loop" "$work/mounted"
sudo mkdir -m 0755 "$work/mounted/native-pool"
sudo incus storage create bw-s2-native-pool dir source="$work/mounted/native-pool"
test "$(findmnt -n -o FSTYPE --target "$work/mounted")" = ext4

sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
uid="$(id -u boxweave)"
project="user-$uid"
test "$(sudo runuser -u boxweave -- id -nG)" = incus
sudo runuser -u boxweave -- incus project list --format csv
sudo incus project show "$project" | grep -q 'restricted: "true"'

# All edits are performed only on the new, empty single-use Project. Do not
# edit global/default instances or attach ACLs to any foreign NIC.
sudo incus profile device set default root pool=bw-s2-native-pool --project "$project"
sudo incus network acl create bw-s2-native-acl
sudo incus profile device set default eth0 --project "$project" \
  security.acls=bw-s2-native-acl \
  security.acls.default.ingress.action=reject \
  security.acls.default.egress.action=allow \
  security.ipv4_filtering=true \
  security.ipv6_filtering=true \
  security.mac_filtering=true

# Prepare the allowlisted immutable Ubuntu image alias for native Create.
# No arbitrary image URL/alias is accepted from MCP clients.
sudo incus image copy images:ubuntu/24.04 local: --project "$project" \
  --alias boxweave-ubuntu-24.04
sudo incus image alias list --project "$project" --format csv | grep -q boxweave-ubuntu-24.04

CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=1.0.24-s2-native-ci -X main.commit=$(git rev-parse HEAD)" \
  -o "$work/boxweave" ./cmd/boxweave
CGO_ENABLED=1 go build -trimpath -o "$work/s2-probe" ./scripts/s2_native_probe.go
chmod 0755 "$work/boxweave" "$work/s2-probe"
sha256sum "$work/boxweave"
sudo chown boxweave:incus "$work/ledger"
db="$work/ledger/ledger.sqlite"
socket=/var/lib/incus/unix.socket.user

# No native instance mutation until this real non-root OS + Project +
# selected block pool + default NIC preflight is actually ready.
sudo runuser -u boxweave -- "$work/boxweave" doctor -db "$db" \
  -incus-socket "$socket" -project "$project" > "$work/preflight.json"
python3 - "$work/preflight.json" <<'PY'
import json, sys
status = json.load(open(sys.argv[1]))
n = status["native"]
assert status["ready"] and n["ready"]
assert n["selected_storage_separate_mount"] and n["selected_storage_space_checked"]
assert n["default_nic_configured"]
print("S2_NATIVE_PRECONDITION_EXACT_NONROOT_BLOCK_POOL=PASS")
PY

sudo runuser -u boxweave -- "$work/boxweave" token-create -db "$db" -role admin > "$work/admin.secret"
sudo runuser -u boxweave -- "$work/boxweave" token-create -db "$db" -role viewer > "$work/viewer.secret"
chmod 0600 "$work/admin.secret" "$work/viewer.secret"
admin="$(sed -n 's/^token=//p' "$work/admin.secret")"
viewer="$(sed -n 's/^token=//p' "$work/viewer.secret")"
test -n "$admin" && test -n "$viewer"

sudo runuser -u boxweave -- "$work/boxweave" serve -db "$db" \
  -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/service.redacted.log" 2>&1 &
service_pid=$!
trap 'kill "$service_pid" 2>/dev/null || :; wait "$service_pid" 2>/dev/null || :' EXIT
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40

# The independent original native Agent's own official MCP Go SDK client
# reads the exact installed candidate's bytes, not in-process Go mocks.
BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" BW_S2_VIEWER_TOKEN="$viewer" \
  "$work/s2-probe"

# Separate administrator's read-only inventory is a SECOND native oracle:
# after exact delete no guest from this run remains in any Project.
if [ -n "$(sudo incus list --all-projects --format csv -c n)" ]; then
  echo "S2_NATIVE_FAIL: original Box still exists after claimed delete" >&2
  sudo incus list --all-projects
  exit 87
fi
echo "S2_NATIVE_INDEPENDENT_INCUS_POSTREAD_EMPTY=PASS"
echo "S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS"
