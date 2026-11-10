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

# CI only: install an exact upstream 7.6 signed package on the pristine
# disposable VM. Ubuntu 24.04's archive ships 6.0.0, whose bridged NIC does
# not accept the required ACL options. Never silently relax the S2 safety
# precondition or enable unreviewed unsigned package sources.
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  ca-certificates curl gpg dnsmasq-base e2fsprogs util-linux >/dev/null
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL --max-time 30 https://pkgs.zabbly.com/key.asc -o "$work/zabbly-key.asc"
fingerprint="$(gpg --show-keys --with-colons "$work/zabbly-key.asc" | awk -F: '$1=="fpr" {print $10; exit}')"
test "$fingerprint" = 4EFC590696CB15B87C73A3AD82CC8797C838DCFD
echo "S2_NATIVE_SIGNED_PACKAGE_KEY_FINGERPRINT_VERIFIED=PASS"
sudo install -m 0644 "$work/zabbly-key.asc" /etc/apt/keyrings/boxweave-s2-zabbly.asc
sudo tee /etc/apt/sources.list.d/boxweave-s2-zabbly.sources >/dev/null <<'EOF'
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: noble
Components: main
Architectures: amd64
Signed-By: /etc/apt/keyrings/boxweave-s2-zabbly.asc
EOF
sudo apt-get update -qq
native_pkg="$(apt-cache madison incus | awk '/pkgs.zabbly.com/ && $3 ~ /:7[.]6[-.]/ { print $3; exit }')"
if [ -z "$native_pkg" ]; then
  echo "S2_NATIVE_BLOCKED: exact signed upstream 7.6 is unavailable in CI apt" >&2
  apt-cache policy incus
  exit 85
fi
echo "S2_NATIVE_SIGNED_PACKAGE_VERSION=$native_pkg"
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  "incus=$native_pkg" "incus-client=$native_pkg" "incus-base=$native_pkg" >/dev/null
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
test "$(sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config id -nG)" = incus
echo "S2_NATIVE_SERVICE_INCUS_XDG_CONFIG=/home/boxweave/.config"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project list --format csv
sudo incus project show "$project" | grep -q 'restricted: "true"'

# All edits are performed only on the new, empty single-use Project. Do not
# edit global/default instances or attach ACLs to any foreign NIC.
# The installed Ubuntu Incus rejects the optional NIC egress override; leave
# its stricter native default reject intact rather than widen access.
sudo incus profile device set default root pool=bw-s2-native-pool --project "$project"
echo "S2_NATIVE_INCUS_VERSION"
sudo incus version
echo "S2_NATIVE_PROFILE_BEFORE_ACL"
sudo incus profile show default --project "$project"
echo "S2_NATIVE_NETWORKS"
sudo incus network list
sudo incus network acl create bw-s2-native-acl
sudo incus profile device set default eth0 --project "$project" \
  security.acls=bw-s2-native-acl \
  security.acls.default.ingress.action=reject \
  security.ipv4_filtering=true \
  security.ipv6_filtering=true \
  security.mac_filtering=true

# Prepare the allowlisted immutable Ubuntu image alias for native Create.
# No arbitrary image URL/alias is accepted from MCP clients.
sudo incus image copy images:ubuntu/24.04 local: --target-project "$project" \
  --alias boxweave-ubuntu-24.04
echo "S2_NATIVE_IMAGE_ALIAS_LIST_ADMIN"
sudo incus image alias list --project "$project"
echo "S2_NATIVE_IMAGE_LIST_ADMIN"
sudo incus image list --project "$project"
echo "S2_NATIVE_IMAGE_ALIAS_LIST_RESTRICTED"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus image alias list --format csv
if ! sudo incus image info boxweave-ubuntu-24.04 --project "$project"; then
  echo "S2_NATIVE_IMAGE_ALIAS_NOT_VISIBLE_IN_OWNED_PROJECT" >&2
  exit 86
fi
echo "S2_NATIVE_OWNED_UBUNTU_ALIAS_VERIFIED=PASS"

CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=1.0.24-s2-native-ci -X main.commit=$(git rev-parse HEAD)" \
  -o "$work/boxweave" ./cmd/boxweave
CGO_ENABLED=1 go build -trimpath -o "$work/s2-probe" ./scripts/s2_native_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-crash-probe" ./scripts/s2_crash_probe.go
chmod 0755 "$work/boxweave" "$work/s2-probe" "$work/s2-crash-probe"
sha256sum "$work/boxweave"
sudo chown boxweave:incus "$work/ledger"
db="$work/ledger/ledger.sqlite"
socket=/var/lib/incus/unix.socket.user

# No native instance mutation until this real non-root OS + Project +
# selected block pool + default NIC preflight is actually ready.
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" doctor -db "$db" \
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

sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -role admin > "$work/admin.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -role viewer > "$work/viewer.secret"
chmod 0600 "$work/admin.secret" "$work/viewer.secret"
admin="$(sed -n 's/^token=//p' "$work/admin.secret")"
viewer="$(sed -n 's/^token=//p' "$work/viewer.secret")"
test -n "$admin" && test -n "$viewer"

sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$db" \
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

# Independently kill the exact disposable BoxWeave service WHILE a native
# guest Exec (whose original native ID is already durably acknowledged) runs.
# Never restart the native effect; revive ONLY the same SQLite-backed MCP
# process. Distinct earlier UNKNOWN tasks on any shared host are untouched.
crash_state="$work/original-crash.json"
BW_S2_CRASH_STATE_FILE="$crash_state" \
  BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" "$work/s2-crash-probe" start
test -s "$crash_state"
if ! sudo pkill -KILL -u boxweave -f "^$work/boxweave serve"; then
  echo "S2_NATIVE_FAIL: exact disposable service process not found for kill" >&2
  exit 88
fi
wait "$service_pid" 2>/dev/null || :
echo "S2_NATIVE_EXACT_SERVICE_PROCESS_KILLED=PASS"

# Reopen THE SAME journal. The host's native Incus daemon is not restarted,
# touched, or reconciled by a privileged repair.
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$db" \
  -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/restarted-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
BW_S2_CRASH_STATE_FILE="$crash_state" \
  BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" "$work/s2-crash-probe" resume

# SECOND ORACLE: The original guest process must have produced exactly ONE
# marker. A successful operation_get is never enough to prove that. This is
# a separate administrator-only read of the single-use Incus test daemon.
crash_uuid="$(python3 - "$crash_state" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["box_uuid"])
PY
)"
native_uuid="$(sudo incus config get bw-s2-crash-110b volatile.uuid --project "$project")"
test -n "$native_uuid" && test "$native_uuid" = "$crash_uuid"
attempt=0
marker_count=""
until [ "$attempt" -ge 45 ]; do
  marker_count="$(sudo incus exec bw-s2-crash-110b --project "$project" -- /bin/sh -c 'wc -l < /tmp/bw-s2-crash-marker' 2>/dev/null || :)"
  if [ "$marker_count" = 1 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$marker_count" = 1
echo "S2_NATIVE_CRASH_ORIGINAL_GUEST_EFFECT_EXACTLY_ONCE=PASS"

# The ONE-USE runner owns this exact test instance. Administrator-only test
# fixture teardown after independent evidence is NOT BoxWeave's remote
# UNKNOWN unlock and does not change this original journal's UNKNOWN state.
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-crash-110b
sudo incus stop bw-s2-crash-110b --project "$project"
sudo incus delete bw-s2-crash-110b --project "$project"
echo "S2_NATIVE_DISPOSABLE_CRASH_FIXTURE_ADMIN_TEARDOWN=PASS"

# Separate administrator's read-only inventory is a SECOND native oracle:
# after exact delete no guest from this run remains in any Project.
if [ -n "$(sudo incus list --all-projects --format csv -c n)" ]; then
  echo "S2_NATIVE_FAIL: original Box still exists after claimed delete" >&2
  sudo incus list --all-projects
  exit 87
fi
echo "S2_NATIVE_INDEPENDENT_INCUS_POSTREAD_EMPTY=PASS"
echo "S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS"
