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

sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
uid="$(id -u boxweave)"
project="user-$uid"
test "$(sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config id -nG)" = incus
echo "S2_NATIVE_SERVICE_INCUS_XDG_CONFIG=/home/boxweave/.config"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project list --format csv
if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project show default > "$work/foreign-project-denied.txt" 2>&1; then
  echo "S2_NATIVE_FAIL: nonroot BoxWeave service can inspect the foreign default Project" >&2
  exit 90
fi
echo "S2_NATIVE_RESTRICTED_USER_FOREIGN_PROJECT_DENIED=PASS"
sudo incus project show "$project" | grep -q 'restricted: "true"'
# Only this one-use restricted Project is allowed to opt in to nested
# unprivileged containers. Native instance security.nesting stays false
# unless a separately authorized Box field explicitly requests true.
sudo incus project set "$project" restricted.containers.nesting=allow
sudo incus project get "$project" restricted.containers.nesting | grep -qx allow
echo "S2_NATIVE_SCOPED_NESTING_CAPABILITY_CONFIGURED=PASS"

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

# Capability probe only on this one-use, admin-owned Project; preserve the
# existing fail-closed runtime default after the experiment. Accepting a
# config value is not proof of internet packet routing, peer isolation or a
# supported BoxWeave internet toggle. It is merely tagged SDK fit evidence.
if sudo incus profile device set default eth0 --project "$project" \
  security.acls.default.egress.action=allow > "$work/egress-setting.txt" 2>&1; then
  observed_egress="$(sudo incus profile device get default eth0 security.acls.default.egress.action --project "$project")"
  test "$observed_egress" = allow
  sudo incus profile device unset default eth0 security.acls.default.egress.action --project "$project"
  echo "S2_NATIVE_BRIDGE_NIC_EGRESS_ALLOW_CONFIG_SUPPORTED=PASS"
else
  echo "S2_NATIVE_BRIDGE_NIC_EGRESS_ALLOW_CONFIG=UNSUPPORTED"
fi

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
CGO_ENABLED=1 go build -trimpath -o "$work/s2-queued-cancel-seed" ./scripts/s2_queued_cancel_seed.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-native-guest-quota" ./scripts/s2_native_guest_quota_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-ack-loss-probe" ./scripts/s2_ack_loss_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-ack-loss-proxy" ./scripts/s2_ack_loss_proxy.go
chmod 0755 "$work/boxweave" "$work/s2-probe" "$work/s2-crash-probe" "$work/s2-queued-cancel-seed" "$work/s2-native-guest-quota" "$work/s2-ack-loss-probe" "$work/s2-ack-loss-proxy"
sha256sum "$work/boxweave"
sudo chown boxweave:incus "$work/ledger"
db="$work/ledger/ledger.sqlite"
socket=/var/lib/incus/unix.socket.user

# Negative control: the same real Incus daemon may accept a NIC with
# ingress default allow. Our own doctor MUST fail closed before native
# Create/Start when that setting is weaker than the owned ACL contract.
# This is a one-use VM/profile only, never a shared daemon modification.
sudo incus profile device set default eth0 --project "$project" \
  security.acls.default.ingress.action=allow
if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" doctor -db "$db" \
  -incus-socket "$socket" -project "$project" > "$work/unsafe-preflight.json"; then
  echo "S2_NATIVE_FAIL: unsafe ingress accepted as ready" >&2
  exit 89
fi
python3 - "$work/unsafe-preflight.json" <<'PY'
import json,sys
out=json.load(open(sys.argv[1]))
n=out["native"]
assert not out["ready"] and not n["ready"]
assert n["error_class"] == "DEFAULT_NIC_NOT_CONFIGURED", n
print("S2_NATIVE_UNSAFE_INGRESS_DOCTOR_DENIED=PASS")
PY
sudo incus profile device set default eth0 --project "$project" \
  security.acls.default.ingress.action=reject

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

# An independent disposable ext4 mount without project quota must NOT pass
# the candidate native doctor, even though its backing device is separate
# from the CI runner's root. Restore the original eligible pool afterward.
truncate -s 512M "$work/noquota-only.img"
noquota_loop="$(sudo losetup --find --show "$work/noquota-only.img")"
sudo mkfs.ext4 -F -q "$noquota_loop"
mkdir -m 0755 "$work/noquota-mounted"
sudo mount -t ext4 -o nodev,nosuid "$noquota_loop" "$work/noquota-mounted"
sudo mkdir -m 0755 "$work/noquota-mounted/native-pool"
sudo incus storage create bw-s2-noquota-pool dir source="$work/noquota-mounted/native-pool"
sudo incus profile device set default root pool=bw-s2-noquota-pool --project "$project"
if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" doctor -db "$db" \
  -incus-socket "$socket" -project "$project" > "$work/noquota-doctor.json"; then
  echo "S2_NATIVE_FAIL: separately mounted no-quota dir pool reported ready" >&2
  exit 107
fi
python3 - "$work/noquota-doctor.json" <<'PY'
import json,sys
with open(sys.argv[1]) as f:
    obj=json.load(f)
native=obj["native"]
assert obj["ready"] is False and native["ready"] is False, native
assert native["error_class"] == "SELECTED_STORAGE_PROJECT_QUOTA_UNVERIFIED", native
assert native["selected_storage_separate_mount"] is True, native
print("S2_NATIVE_DIR_SEPARATE_MOUNT_WITHOUT_PROJECT_QUOTA_FAILS_CLOSED=PASS")
PY
sudo incus profile device set default root pool=bw-s2-native-pool --project "$project"
sudo incus storage delete bw-s2-noquota-pool
sudo umount "$work/noquota-mounted"
sudo losetup -d "$noquota_loop"

sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -role admin > "$work/admin.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -role viewer > "$work/viewer.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -role admin > "$work/admin2.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -role viewer -ttl 1s > "$work/expired-viewer.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -role viewer > "$work/revoked-viewer.secret"
chmod 0600 "$work/admin.secret" "$work/viewer.secret" "$work/admin2.secret" "$work/expired-viewer.secret" "$work/revoked-viewer.secret"
revoked_viewer_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/revoked-viewer.secret")"
echo "$revoked_viewer_id" | grep -Eq '^[0-9a-f]{32}$'
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-revoke -db "$db" -id "$revoked_viewer_id" > "$work/revoked-viewer-result.txt"
admin="$(sed -n 's/^token=//p' "$work/admin.secret")"
viewer="$(sed -n 's/^token=//p' "$work/viewer.secret")"
admin2="$(sed -n 's/^token=//p' "$work/admin2.secret")"
expired_viewer="$(sed -n 's/^token=//p' "$work/expired-viewer.secret")"
revoked_viewer="$(sed -n 's/^token=//p' "$work/revoked-viewer.secret")"
admin_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/admin.secret")"
admin2_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/admin2.secret")"
test -n "$admin" && test -n "$viewer" && test -n "$admin2" && test -n "$expired_viewer" && test -n "$revoked_viewer"
echo "$admin_id" | grep -Eq '^[0-9a-f]{32}$'
echo "$admin2_id" | grep -Eq '^[0-9a-f]{32}$'
test "$admin_id" != "$admin2_id"

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

# A second serve on the identical SQLite must be rejected BEFORE it may
# run Recover() on the original controller's still-live operations.
# Read-only local status must still be available from a separate process.
if timeout --signal=TERM 8s sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$db" \
  -incus-socket "$socket" -project "$project" -listen 127.0.0.1:18655 > "$work/duplicate-serve.refusal" 2>&1; then
  echo "S2_NATIVE_FAIL: second serve acquired already-owned SQLite" >&2
  exit 98
fi
grep -Fq "serve already active for this database" "$work/duplicate-serve.refusal"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" status -db "$db" \
  -incus-socket "$socket" -project "$project" > "$work/concurrent-status.json"
test -s "$work/concurrent-status.json"
test "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp)" = 401
echo "S2_NATIVE_ACTIVE_SERVER_LOCAL_DB_MUTEX_AND_OBSERVER=PASS"

# Seed a single original ADMITTED operation with no native dispatch under
# the SAME restricted service actor/database. The official SDK prober then
# exercises operation_cancel as an actual MCP mutation, proving no Incus side
# effect or fabricated native ID and immutable same-key cancellation replay.
sudo runuser -u boxweave -- env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  BW_S2_SEED_DB="$db" BW_S2_SEED_PROJECT="$project" \
  BW_S2_SEED_ADMIN_TOKEN="$admin" "$work/s2-queued-cancel-seed" > "$work/queued-cancel-id.txt"
queued_cancel_id="$(sed -n '1p' "$work/queued-cancel-id.txt")"
foreign_layer_id="$(sed -n '2p' "$work/queued-cancel-id.txt")"
test "$(wc -l < "$work/queued-cancel-id.txt")" -eq 2
echo "$queued_cancel_id" | grep -Eq '^[0-9a-f]{32}$'
echo "$foreign_layer_id" | grep -Eq '^[0-9a-f]{32}$'
test "$queued_cancel_id" != "$foreign_layer_id"
test -z "$(sudo incus list --all-projects --format csv -c n)"

# The independent official MCP SDK prober temporarily holds one completed
# 16MiB+4-byte dual-stream Exec until this CI VM administrator measures its
# ACTUAL native spool files. This is read-only and never deletes native logs.
# A failure is not an excuse to retry an original native mutation.
( BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" BW_S2_VIEWER_TOKEN="$viewer" \
  BW_S2_ADMIN2_TOKEN="$admin2" BW_S2_ADMIN1_ID="$admin_id" \
  BW_S2_ADMIN2_ID="$admin2_id" BW_S2_OUTPUT_WITNESS_DIR="$work" \
  BW_S2_EXPIRED_TOKEN="$expired_viewer" BW_S2_REVOKED_TOKEN="$revoked_viewer" \
  BW_S2_FOREIGN_LAYER_OPID="$foreign_layer_id" \
  BW_S2_QUEUED_CANCEL_OPID="$queued_cancel_id" "$work/s2-probe" ) > "$work/native-probe.log" 2>&1 &
probe_pid=$!
attempt=0
while [ ! -s "$work/output-witness.ready" ] && [ "$attempt" -lt 200 ]; do
  if ! kill -0 "$probe_pid" 2>/dev/null; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ ! -s "$work/output-witness.ready" ]; then
  echo "S2_NATIVE_FAIL: no original native Exec output witness checkpoint" >&2
  cat "$work/native-probe.log"
  exit 103
fi
recorded_native_id="$(cat "$work/output-witness.ready")"
echo "$recorded_native_id" | grep -Eq '^[0-9a-f-]{36}$'
# Restrict the witness to Incus's own container state tree, not arbitrary
# host files. Only the exact native Operation ID from the accepted guest
# Exec can match both recorded output files.
# Witness only the exact temporary instance spool directory described by
# the pinned Incus SDK. Never follow rootfs symlinks or traverse guest files.
# A missing path is UNVERIFIED, not evidence of an absent native effect;
# continue the existing core native acceptance without claiming disk PASS.
native_instance_dir="$(printf '/var/lib/incus/containers/%s_bw-s2-ghci-110a' "$project")"
native_spool_dir="$native_instance_dir/exec-output"
disk_witness=UNVERIFIED
if sudo test -d "$native_spool_dir"; then
  recorded_stdout="$native_spool_dir/exec_$recorded_native_id.stdout"
  recorded_stderr="$native_spool_dir/exec_$recorded_native_id.stderr"
  if sudo test -f "$recorded_stdout" && sudo test -f "$recorded_stderr"; then
    actual_stdout_bytes="$(sudo stat -c '%s' "$recorded_stdout")"
    actual_stderr_bytes="$(sudo stat -c '%s' "$recorded_stderr")"
    # The Incus path is a protected root-owned symlink to the selected
    # storage pool. Mount target strings differ across bind mounts; compare
    # actual filesystem device IDs, not path strings or API read limits.
    actual_stdout_mount="$(sudo findmnt -n -o TARGET --target "$recorded_stdout" || :)"
    selected_pool_mount="$(sudo findmnt -n -o TARGET --target "$work/mounted/native-pool" || :)"
    actual_spool_device="$(sudo stat -Lc '%d' "$recorded_stdout")"
    selected_pool_device="$(sudo stat -Lc '%d' "$work/mounted/native-pool")"
    host_root_device="$(sudo stat -Lc '%d' /)"
    echo "S2_NATIVE_EXEC_SPOOL_REAL_STDOUT_BYTES=$actual_stdout_bytes"
    echo "S2_NATIVE_EXEC_SPOOL_REAL_STDERR_BYTES=$actual_stderr_bytes"
    echo "S2_NATIVE_EXEC_SPOOL_PHYSICAL_MOUNT=$actual_stdout_mount"
    echo "S2_NATIVE_EXEC_SELECTED_POOL_MOUNT=$selected_pool_mount"
    if [ "$actual_stdout_bytes" -eq 16777220 ] &&
        [ "$actual_stderr_bytes" -eq 16777220 ] &&
        [ -n "$actual_stdout_mount" ] && [ -n "$selected_pool_mount" ] &&
        [ "$actual_spool_device" = "$selected_pool_device" ] &&
        [ "$actual_spool_device" != "$host_root_device" ]; then
      disk_witness=PASS
      echo "S2_NATIVE_EXEC_SPOOL_ON_SELECTED_BLOCK_POOL=OBSERVED"
    else
      echo "S2_NATIVE_EXEC_SPOOL_BACKING_NOT_PROVED=UNVERIFIED"
    fi
  fi
fi
if [ "$disk_witness" = PASS ]; then
  echo "S2_NATIVE_EXEC_SPOOL_PHYSICAL_FILE_SIZE_WITNESSED=PASS"
  # Prove the current Incus dir root-disk size=3GiB restriction covers its
  # sibling native Exec spool. This test writes only on a newly authorized,
  # isolated 4GiB loop-mounted CI pool. Require enough free capacity to
  # distinguish the kernel's EDQUOT from physical ENOSPC.
  # No recorded Exec file is changed; this unique disposable fixture is
  # removed before releasing the still-live official MCP SDK client.
  quota_test="$native_spool_dir/.boxweave-s2-disposable-quota-test"
  test ! -e "$quota_test"
  quota_free_before="$(df -B1 --output=avail "$work/mounted" | tail -n1 | tr -d ' ')"
  test "$quota_free_before" -gt 2800000000
  # Incus's instance pathname is a symlink; lsattr on that link yields
  # EOPNOTSUPP. Inspect the real spool directory inode's inherited project ID.
  sudo lsattr -p -d "$native_spool_dir"
  test "$(sudo incus config device get bw-s2-ghci-110a root size --project "$project")" = "3GiB"
  echo "S2_NATIVE_DIR_VOLUME_QUOTA_DIAGNOSTIC"
  sudo incus storage volume info bw-s2-native-pool container/bw-s2-ghci-110a --project "$project" || :
  sudo incus config show bw-s2-ghci-110a --expanded --project "$project" | grep -E 'pool:|size:|path:' || :
  sudo findmnt -n -o FSTYPE,OPTIONS --target "$recorded_stdout" || :
  # Root with CAP_SYS_RESOURCE may bypass quota even if guest/native writes
  # cannot. Drop that specific capability for the independent probe.
  if sudo timeout 40s setpriv --bounding-set=-sys_resource --nnp -- fallocate -l 2684354560 "$quota_test" > "$work/quota-fallocate.stderr" 2>&1; then
    sudo rm -f "$quota_test"
    echo "S2_NATIVE_FAIL: Incus size quota allowed >2GiB native Exec sibling allocation" >&2
    exit 105
  fi
  if ! grep -Fq 'Disk quota exceeded' "$work/quota-fallocate.stderr"; then
    sudo rm -f "$quota_test"
    echo "S2_NATIVE_FAIL: expected kernel EDQUOT from native Exec spool quota, not other failure" >&2
    cat "$work/quota-fallocate.stderr" >&2
    exit 106
  fi
  sudo rm -f "$quota_test"
  test ! -e "$quota_test"
  echo "S2_NATIVE_DIR_PROJECT_QUOTA_APPLIES_TO_EXEC_SPOOL=PASS"
else
  echo "S2_NATIVE_EXEC_SPOOL_PHYSICAL_FILE_SIZE_WITNESSED=UNVERIFIED"
  sudo ls -ld "$native_instance_dir" "$native_spool_dir" 2>&1 || :
  sudo find "$native_instance_dir" -maxdepth 2 -type f -name 'exec_*' -printf '%p bytes=%s\n' | head -n 12 || :
  sudo incus query "/1.0/instances/bw-s2-ghci-110a/logs/exec-output?project=$project" || :
fi
touch "$work/output-witness.release"
if ! wait "$probe_pid"; then
  # Dedicated one-use CI VM only. Safe native metadata diagnosis after the
  # failed original operation; do not repeat native creates or expose tokens.
  cat "$work/native-probe.log"
  echo "S2_NATIVE_DIAG_ORIGINAL_CREATE_FINAL_STATE"
  sudo incus list --all-projects --format yaml || :
  sudo incus config show bw-s2-ghci-110a --expanded --project "$project" || :
  native_diagnose_id="$(sed -n 's/.*native_operation_id:\([0-9a-f-]*\).*/\1/p' "$work/native-probe.log" | head -n1)"
  if [ -n "$native_diagnose_id" ]; then
    sudo incus operation show "$native_diagnose_id" --project "$project" || :
  fi
  exit 92
fi
cat "$work/native-probe.log"

# Independently kill the exact disposable BoxWeave service WHILE a native
# guest Exec (whose original native ID is already durably acknowledged) runs.
# Never restart the native effect; revive ONLY the same SQLite-backed MCP
# process. Distinct earlier UNKNOWN tasks on any shared host are untouched.
crash_state="$work/original-crash.json"
if ! ( BW_S2_CRASH_STATE_FILE="$crash_state" \
  BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" "$work/s2-crash-probe" start ) > "$work/crash-start.log" 2>&1; then
  # Postmortem is read-only, on the exact disposable CI Project. Never
  # reissue a possibly completed native create, nor touch a shared daemon.
  cat "$work/crash-start.log"
  echo "S2_NATIVE_DIAG_FAILED_CRASH_FIXTURE_CREATE"
  sudo incus list --all-projects --format yaml || :
  sudo incus config show bw-s2-crash-110b --expanded --project "$project" || :
  sudo incus info bw-s2-crash-110b --project "$project" || :
  original_id="$(sed -n 's/.*native_operation_id:\([0-9a-f-]*\).*/\1/p' "$work/crash-start.log" | head -n1)"
  if [ -n "$original_id" ]; then
    sudo incus operation show "$original_id" --project "$project" || :
  fi
  exit 93
fi
cat "$work/crash-start.log"
test -s "$crash_state"

# The first server now has an ACTUAL in-flight guest Exec with a persisted
# native operation ID. Try a second serve on its SAME SQLite while it runs.
# The second process must fail BEFORE startup Recover can rewrite that work.
if timeout --signal=TERM 8s sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$db" \
  -incus-socket "$socket" -project "$project" -listen 127.0.0.1:18655 > "$work/inflight-duplicate-serve.refusal" 2>&1; then
  echo "S2_NATIVE_FAIL: concurrent serve adopted live original guest Exec" >&2
  exit 99
fi
grep -Fq "serve already active for this database" "$work/inflight-duplicate-serve.refusal"
sudo python3 - "$db" "$crash_state" <<'PY'
import json,sqlite3,sys
db,original_path=sys.argv[1:]
with open(original_path) as f:
    original=json.load(f)
original_id=original["operation_id"]
conn=sqlite3.connect("file:"+db+"?mode=ro",uri=True,timeout=3)
row=conn.execute("SELECT state FROM operations WHERE id=?",(original_id,)).fetchone()
conn.close()
if not row or row[0] not in ("VERIFYING","SUCCEEDED"):
    raise SystemExit("a second serve rewrote a live original operation")
PY
echo "S2_NATIVE_DUPLICATE_SERVE_DURING_REAL_EXEC_NO_RECOVERY=PASS"

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

# The installed candidate must refuse arbitrary/foreign SQLite histories
# BEFORE altering any byte or issuing an Incus mutation. A one-use lab DB
# is intentionally constructed independently from the running journal.
foreign_db="$work/ledger/foreign.sqlite"
corrupt_db="$work/ledger/corrupt.sqlite"
legacy_db="$work/ledger/legacy.sqlite"
ambiguous_db="$work/ledger/ambiguous.sqlite"
versioned_foreign_db="$work/ledger/versioned-foreign.sqlite"
sudo runuser -u boxweave -- python3 - "$foreign_db" "$corrupt_db" "$legacy_db" "$ambiguous_db" "$versioned_foreign_db" <<'PY'
import os,sqlite3,sys
foreign,corrupt,legacy,ambiguous,versioned_foreign=sys.argv[1:]
conn=sqlite3.connect(foreign)
conn.execute("CREATE TABLE foreign_marker (proof TEXT NOT NULL)")
conn.execute("INSERT INTO foreign_marker VALUES ('do-not-adopt')")
conn.commit()
conn.close()
with open(corrupt,"wb") as f:
    f.write(b"not a SQLite database; must remain exactly unchanged")
for path,version,extra_row in ((legacy,0,False),(ambiguous,1,True),(versioned_foreign,1,False)):
    conn=sqlite3.connect(path)
    conn.execute("CREATE TABLE schema_version(version INTEGER NOT NULL)")
    conn.execute("INSERT INTO schema_version VALUES (?)",(version,))
    if extra_row:
        conn.execute("INSERT INTO schema_version VALUES (1)")
    if path==versioned_foreign:
        conn.execute("CREATE TABLE outsider (proof TEXT)")
    conn.commit()
    conn.close()
for path in sys.argv[1:]:
    os.chmod(path,0o600)
PY
for case in foreign corrupt legacy ambiguous versioned_foreign; do
  case "$case" in
    foreign) f="$foreign_db" ;;
    corrupt) f="$corrupt_db" ;;
    legacy) f="$legacy_db" ;;
    ambiguous) f="$ambiguous_db" ;;
    versioned_foreign) f="$versioned_foreign_db" ;;
  esac
  before="$(sudo sha256sum "$f" | cut -d' ' -f1)"
  if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" doctor -db "$f" \
      -incus-socket "$socket" -project "$project" > "$work/sqlite-$case.stderr" 2>&1; then
    echo "S2_NATIVE_FAIL: adopted unauthorized $case SQLite layout" >&2
    exit 94
  fi
  after="$(sudo sha256sum "$f" | cut -d' ' -f1)"
  if [ "$before" != "$after" ]; then
    echo "S2_NATIVE_FAIL: foreign SQLite bytes modified during refusal: $case" >&2
    exit 95
  fi
done
echo "S2_NATIVE_SQLITE_FOREIGN_CORRUPT_FAIL_CLOSED=PASS"
echo "S2_NATIVE_SQLITE_OLD_MULTIPLE_VERSION_AND_FOREIGN_TABLES_DENIED=PASS"

# Deliberate ACK loss after the REAL Incus server accepts a Create. This
# does not retry or adopt any historical task; it uses a new independent
# SQLite ledger, exact disposable original name and the same non-root Incus
# user/Project. The MITM Unix socket cannot access privileged daemon APIs.
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
ack_proxy="$work/ledger/ack-proxy.sock"
ack_marker="$work/ledger/ack-trap.txt"
ack_db="$work/ledger/ack.sqlite"
ack_checkpoint="$work/ack-original.json"
sudo runuser -u boxweave -- env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  "$work/s2-ack-loss-proxy" -listen "$ack_proxy" -upstream "$socket" \
  -marker "$ack_marker" -project "$project" > "$work/ack-proxy.log" 2>&1 &
proxy_pid=$!
trap 'kill "$service_pid" 2>/dev/null || :; wait "$service_pid" 2>/dev/null || :; kill "$proxy_pid" 2>/dev/null || :; wait "$proxy_pid" 2>/dev/null || :' EXIT
attempt=0
until sudo test -S "$ack_proxy" || [ "$attempt" -ge 30 ]; do
  attempt=$((attempt+1))
  sleep 1
done
sudo test -S "$ack_proxy"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$ack_db" -role admin > "$work/ack-admin.secret"
chmod 0600 "$work/ack-admin.secret"
ack_admin="$(sed -n 's/^token=//p' "$work/ack-admin.secret")"
test -n "$ack_admin"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$ack_db" \
  -incus-socket "$ack_proxy" -project "$project" \
  -listen 127.0.0.1:18652 > "$work/ack-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18652/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" start
sudo grep -Fqx "UPSTREAM_ACCEPTED_ONCE_AND_ACK_SUPPRESSED" "$ack_marker"
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 1
echo "S2_NATIVE_CREATE_REAL_INCUS_ACCEPTED_ACK_DROPPED=PASS"

# Second oracle: independent daemon administrator verifies exactly one
# native Box of the original name actually exists. The MCP journal remains
# UNKNOWN and cannot adopt it merely by this read.
attempt=0
ack_uuid=""
until [ "$attempt" -ge 50 ]; do
  ack_uuid="$(sudo incus config get bw-s2-ack-native-110d volatile.uuid --project "$project" 2>/dev/null || :)"
  if [ -n "$ack_uuid" ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
test -n "$ack_uuid"
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-ack-native-110d
echo "S2_NATIVE_CREATE_LOST_ACK_EFFECT_EXISTS_INDEPENDENTLY=PASS"

# Reopen the SAME unknown journal; do not replay the original Create.
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$ack_db" \
  -incus-socket "$ack_proxy" -project "$project" \
  -listen 127.0.0.1:18652 > "$work/ack-restarted-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18652/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" resume
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 1
test "$(sudo incus config get bw-s2-ack-native-110d volatile.uuid --project "$project")" = "$ack_uuid"
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-ack-native-110d
echo "S2_NATIVE_CREATE_ACK_LOSS_POSTRESTART_NO_SECOND_POST=PASS"

# Only this **one-use CI host administrator**, NOT BoxWeave's original
# UNKNOWN recovery API, deletes the exact known test fixture after native
# effects were independently witnessed. Leave the SQLite record UNKNOWN.
# An ACK was intentionally suppressed immediately after HTTP 202; the native
# Create can still be running when its UUID becomes visible. Inspect Incus
# asynchronous operations read-only and wait for the REAL native Create to
# settle before attempting cleanup. Never retry the original MCP mutation.
attempt=0
while [ "$attempt" -lt 120 ]; do
  if sudo incus operation list --project "$project" --format json |
      python3 -c 'import json,sys; ops=json.load(sys.stdin); sys.exit(0 if len(ops)==0 else 1)'; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 120 ]; then
  echo "S2_NATIVE_FAIL: lost-ACK native create still running; cannot safely teardown" >&2
  sudo incus operation list --project "$project" --format yaml || :
  sudo incus info bw-s2-ack-native-110d --project "$project" || :
  exit 96
fi
test "$(sudo incus config get bw-s2-ack-native-110d volatile.uuid --project "$project")" = "$ack_uuid"
echo "S2_NATIVE_CREATE_LOST_ACK_ORIGINAL_NATIVE_FINISHED_BEFORE_CLEANUP=PASS"
sudo incus delete bw-s2-ack-native-110d --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_NATIVE_CREATE_ACK_LOSS_FIXTURE_ADMIN_TEARDOWN=PASS"

# Same isolated native/proxy service, a separate new Box and request keys.
# Advance the real native ETag just between S2's GetInstance and
# UpdateInstance, proving the stale update cannot override newer settings.
BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" etag
sudo grep -Fqx "ETAG_FIXTURE_CREATE_FORWARDED" "$ack_marker"
sudo grep -Fqx "ETAG_RACE_NATIVE_CHANGED_BEFORE_STALE_PUT" "$ack_marker"
sudo grep -Fqx "ETAG_RACE_ORIGINAL_STALE_PUT_REFUSED" "$ack_marker"
if sudo grep -q 'UNEXPECTED_SECOND' "$ack_marker"; then
  echo "S2_NATIVE_FAIL: stale ETag conflict caused a forbidden second native dispatch" >&2
  exit 97
fi
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 4
test "$(sudo incus config get bw-s2-etag-native-110e user.s2-etag-race --project "$project")" = "native-proxy-race-witness"
test "$(sudo incus config get bw-s2-etag-native-110e limits.cpu --project "$project")" = 1
etag_uuid="$(sudo incus config get bw-s2-etag-native-110e volatile.uuid --project "$project")"
test -n "$etag_uuid"
echo "S2_NATIVE_STALE_ETAG_UPDATE_REJECTED_ORIGINAL_NATIVE_UNCHANGED=PASS"

# This is independent CI admin teardown, not a BoxWeave recovery. Verify
# no outstanding native work and only the exact disposable fixture exists.
attempt=0
while [ "$attempt" -lt 60 ]; do
  if sudo incus operation list --project "$project" --format json |
      python3 -c 'import json,sys; sys.exit(0 if len(json.load(sys.stdin))==0 else 1)'; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 60
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-etag-native-110e
sudo incus delete bw-s2-etag-native-110e --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_NATIVE_STALE_ETAG_FIXTURE_ADMIN_TEARDOWN=PASS"

# A distinct newly created, UUID-registered Box receives an actual native
# Update (If-Match, HTTP 202), but the test proxy drops only the reply after
# Incus acceptance. Unlike the stale ETag 412 test, the mutation must really
# change CPU. Only the one-use CI administrator performs independent readback
# and eventually removes this disposable fixture. No original UNKNOWN replay.
update_ack_checkpoint="$work/update-ack-original.json"
BW_S2_UPDATE_ACK_STATE_FILE="$update_ack_checkpoint" \
  BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" update_ack
sudo grep -Fqx "UPDATE_ACK_FIXTURE_CREATE_FORWARDED" "$ack_marker"
sudo grep -Fqx "UPDATE_ACK_NATIVE_ACCEPTED_ONCE_AND_ACK_SUPPRESSED" "$ack_marker"
if sudo grep -q 'UNEXPECTED_SECOND' "$ack_marker"; then
  echo "S2_NATIVE_FAIL: accepted Update ACK loss caused a forbidden second dispatch" >&2
  exit 98
fi
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 6
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-update-ack-native-110f
update_ack_uuid="$(sudo incus config get bw-s2-update-ack-native-110f volatile.uuid --project "$project")"
test -n "$update_ack_uuid"
attempt=0
until [ "$attempt" -ge 90 ]; do
  if [ "$(sudo incus config get bw-s2-update-ack-native-110f limits.cpu --project "$project" 2>/dev/null || :)" = 2 ] &&
      sudo incus operation list --project "$project" --format json |
        python3 -c 'import json,sys; sys.exit(0 if len(json.load(sys.stdin))==0 else 1)'; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 90 ]; then
  echo "S2_NATIVE_FAIL: native accepted Update did not settle with CPU=2" >&2
  exit 99
fi
test "$(sudo incus config get bw-s2-update-ack-native-110f volatile.uuid --project "$project")" = "$update_ack_uuid"
echo "S2_NATIVE_UPDATE_ACCEPTED_LOST_ACK_REAL_NATIVE_READBACK=PASS"

# Restart with the exact same SQLite and original unknown Update. Every
# replay is read-only: neither the old accepted request nor any new key may
# generate another native PUT for this name.
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$ack_db" \
  -incus-socket "$ack_proxy" -project "$project" \
  -listen 127.0.0.1:18652 > "$work/update-ack-restarted-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18652/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
BW_S2_UPDATE_ACK_STATE_FILE="$update_ack_checkpoint" \
  BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" update_ack_resume
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 6
if sudo grep -q 'UNEXPECTED_SECOND' "$ack_marker"; then
  echo "S2_NATIVE_FAIL: Update retransmitted after same-SQLite restart" >&2
  exit 100
fi
test "$(sudo incus config get bw-s2-update-ack-native-110f volatile.uuid --project "$project")" = "$update_ack_uuid"
test "$(sudo incus config get bw-s2-update-ack-native-110f limits.cpu --project "$project")" = 2
sudo incus delete bw-s2-update-ack-native-110f --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_NATIVE_UPDATE_ACCEPTED_LOST_ACK_FIXTURE_ADMIN_TEARDOWN=PASS"

# ACK DELIVERED but the first exact native postcondition GET is refused by
# the one-use restricted socket proxy AFTER the genuine 202. BoxWeave must
# preserve the originally acknowledged native operation ID and UNKNOWN lock.
postread_checkpoint="$work/postread-original.json"
BW_S2_POSTREAD_STATE_FILE="$postread_checkpoint" \
  BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" postread
for event in POSTREAD_FIXTURE_CREATE_FORWARDED POSTREAD_NATIVE_UPDATE_202_ACK_DELIVERED POSTREAD_AFTER_NATIVE_ACK_INSTANCE_GET_REFUSED_ONCE; do
  sudo grep -Fqx "$event" "$ack_marker"
done
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 9
if sudo grep -q 'UNEXPECTED_SECOND' "$ack_marker"; then
  echo "S2_NATIVE_FAIL: postread fault retransmitted Update" >&2
  exit 101
fi
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-update-postread-native-110g
postread_uuid="$(sudo incus config get bw-s2-update-postread-native-110g volatile.uuid --project "$project")"
test -n "$postread_uuid"
attempt=0
until [ "$attempt" -ge 90 ]; do
  if [ "$(sudo incus config get bw-s2-update-postread-native-110g limits.cpu --project "$project" 2>/dev/null || :)" = 2 ] &&
      sudo incus operation list --project "$project" --format json |
        python3 -c 'import json,sys; sys.exit(0 if len(json.load(sys.stdin))==0 else 1)'; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 90
echo "S2_NATIVE_UPDATE_RECEIVED_ACK_LOST_POSTREAD_NATIVE_EFFECT_CONFIRMED=PASS"

sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$ack_db" \
  -incus-socket "$ack_proxy" -project "$project" \
  -listen 127.0.0.1:18652 > "$work/postread-restarted-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18652/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
BW_S2_POSTREAD_STATE_FILE="$postread_checkpoint" \
  BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" postread_resume
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 9
test "$(sudo incus config get bw-s2-update-postread-native-110g volatile.uuid --project "$project")" = "$postread_uuid"
test "$(sudo incus config get bw-s2-update-postread-native-110g limits.cpu --project "$project")" = 2
if sudo grep -q 'UNEXPECTED_SECOND' "$ack_marker"; then
  echo "S2_NATIVE_FAIL: postread uncertainty bypassed original lock" >&2
  exit 102
fi
sudo incus delete bw-s2-update-postread-native-110g --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_NATIVE_UPDATE_POSTREAD_UNKNOWN_RESTART_AND_TEARDOWN=PASS"

# Independent actual guest recorded-output quota exhaustion, on the same
# specially created 4GiB ext4 Project Quota mount. Never fill host root.
# This is a SECOND, unique, untracked CI-only Incus fixture, created only
# after all main MCP/SQLite/native replay tests have finished and cleaned.
test -z "$(sudo incus list --all-projects --format csv -c n)"
quota_name="bw-s2-quota-native-110h"
quota_uuid=""
s2_quota_native() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    BW_S2_QUOTA_PROJECT="$project" BW_S2_QUOTA_SOCKET="$socket" \
    BW_S2_QUOTA_UUID="$quota_uuid" "$work/s2-native-guest-quota" "$1"
}
s2_quota_native create > "$work/quota-guest-create.txt"
quota_uuid="$(sed -n 's/^S2_NATIVE_QUOTA_GUEST_UUID=//p' "$work/quota-guest-create.txt")"
echo "$quota_uuid" | grep -Eq '^[0-9a-f-]{36}$'
test "$(sudo incus config get "$quota_name" volatile.uuid --project "$project")" = "$quota_uuid"

# Wait for read-only guest shell readiness, never retry a native mutation.
ready=0
for attempt in $(seq 1 40); do
  if sudo incus exec "$quota_name" --project "$project" -- /bin/true >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
test "$ready" -eq 1
s2_quota_native seed > "$work/quota-guest-seed.txt"
grep -Fqx "S2_NATIVE_QUOTA_SPOOL_SEEDED=PASS" "$work/quota-guest-seed.txt"

native_quota_spool="$(printf '/var/lib/incus/containers/%s_%s/exec-output' "$project" "$quota_name")"
sudo test -d "$native_quota_spool"
test "$(sudo stat -Lc '%d' "$native_quota_spool")" = "$(sudo stat -Lc '%d' "$work/mounted/native-pool")"
test "$(sudo stat -Lc '%d' "$native_quota_spool")" != "$(sudo stat -Lc '%d' /)"
sudo lsattr -p -d "$native_quota_spool"
quota_state_url="/1.0/storage-pools/bw-s2-native-pool/volumes/container/$quota_name/state?project=$project"
sudo incus query "$quota_state_url" > "$work/quota-guest-before.json"
quota_values="$(python3 - "$work/quota-guest-before.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
v=v.get("metadata",v)["usage"]
print(int(v["used"]),int(v["total"]))
PY
)"
quota_used="$(printf '%s\n' "$quota_values" | awk '{print $1}')"
quota_limit="$(printf '%s\n' "$quota_values" | awk '{print $2}')"
test "$quota_limit" -eq 2147483648
test "$quota_used" -gt 10485760
quota_headroom=50331648
quota_fill_bytes=$((quota_limit - quota_used - quota_headroom))
test "$quota_fill_bytes" -gt 1073741824
physical_available="$(df -B1 --output=avail "$work/mounted" | tail -n1 | tr -d ' ')"
test "$physical_available" -gt $((quota_fill_bytes + 268435456))
quota_filler="$native_quota_spool/.boxweave-s2-guest-quota-filler"
sudo test ! -e "$quota_filler"
sudo timeout 30s setpriv --bounding-set=-sys_resource --nnp -- \
  fallocate -l "$quota_fill_bytes" "$quota_filler"
sudo incus query "$quota_state_url" > "$work/quota-guest-filled.json"
quota_used_filled="$(python3 - "$work/quota-guest-filled.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)"
test "$quota_used_filled" -gt $((quota_limit - 100663296))
test "$quota_used_filled" -le "$quota_limit"
echo "S2_NATIVE_GUEST_EXEC_QUOTA_NEAR_LIMIT_PRECONDITION=PASS"

# This original Incus operation is dispatched ONCE by the test-only native
# SDK helper. It tries to record 128MiB with only ~48MiB of quota left.
s2_quota_native flood > "$work/quota-guest-flood.txt"
quota_exec_id="$(sed -n 's/^S2_NATIVE_QUOTA_OP_ID=//p' "$work/quota-guest-flood.txt")"
echo "$quota_exec_id" | grep -Eq '^[0-9a-f-]{36}$'
grep -Fq 'S2_NATIVE_QUOTA_OP_STATUS=' "$work/quota-guest-flood.txt"
quota_stdout="$(printf '%s/exec_%s.stdout' "$native_quota_spool" "$quota_exec_id")"
sudo test -f "$quota_stdout"
quota_stdout_bytes="$(sudo stat -c '%s' "$quota_stdout")"
echo "S2_NATIVE_GUEST_QUOTA_PARTIAL_STDOUT_BYTES=$quota_stdout_bytes"
test "$quota_stdout_bytes" -gt 0
test "$quota_stdout_bytes" -lt 134217728
sudo incus query "$quota_state_url" > "$work/quota-guest-after-output.json"
quota_used_output="$(python3 - "$work/quota-guest-after-output.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)"
test "$quota_used_output" -le "$quota_limit"
echo "S2_NATIVE_REAL_GUEST_RECORDED_EXEC_QUOTA_PARTIAL_OUTPUT=PASS"

# Remove only the unique quota-test filler, NEVER original native
# stdout/stderr or an unresolved task. Independently confirm recovered space.
sudo rm -f -- "$quota_filler"
sudo test ! -e "$quota_filler"
sudo incus query "$quota_state_url" > "$work/quota-guest-reclaimed.json"
quota_used_reclaimed="$(python3 - "$work/quota-guest-reclaimed.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)"
test "$quota_used_reclaimed" -lt $((quota_used_output - quota_fill_bytes / 2))
test "$(sudo incus config get "$quota_name" volatile.uuid --project "$project")" = "$quota_uuid"
sudo incus delete --force "$quota_name" --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_NATIVE_GUEST_QUOTA_SPACE_RECOVERED_AND_FIXTURE_REMOVED=PASS"
echo "S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS"
