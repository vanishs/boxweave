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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=FRESH_RUNNER_PRECHECK"

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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=INCUS_76_BASE_READY"
test -z "$(sudo incus list --all-projects --format csv -c n)"

. scripts/s2_ext4_quota_pool_lab.sh

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
CGO_ENABLED=1 go build -trimpath -o "$work/s2-bh018-atomic-mcp-probe" ./scripts/s2_bh018_atomic_mcp_probe.go
if [ "${S4_A_RAW_WITNESS:-}" = true ]; then
  CGO_ENABLED=1 go build -trimpath -o "$work/s4-oauth-native-probe" ./scripts/s4_oauth_installed_probe.go
  CGO_ENABLED=1 go build -trimpath -o "$work/s4-native-event-tap" ./scripts/s4_raw_native_event_tap.go
  chmod 0755 "$work/s4-native-event-tap"
  echo "S4_A_NATIVE_STAGE=OAUTH_AND_INCUS_EVENT_PROBES_COMPILED"
fi
chmod 0755 "$work/s2-bh018-atomic-mcp-probe"
CGO_ENABLED=1 go build -trimpath -o "$work/s2-crash-probe" ./scripts/s2_crash_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-queued-cancel-seed" ./scripts/s2_queued_cancel_seed.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-native-guest-quota" ./scripts/s2_native_guest_quota_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-mcp-quota-probe" ./scripts/s2_mcp_quota_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-exec-gc-probe" ./scripts/s2_exec_gc_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-gc-delete-proxy" ./scripts/s2_gc_delete_proxy.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-sqlite-fsync-probe" ./scripts/s2_sqlite_fsync_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p0-native-sqlite-proxy" ./scripts/s2_p0_native_sqlite_proxy.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p0-native-sqlite-probe" ./scripts/s2_p0_native_sqlite_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-ack-loss-probe" ./scripts/s2_ack_loss_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-unknown-create-bind-probe" ./scripts/s2_unknown_create_binding_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-ack-loss-proxy" ./scripts/s2_ack_loss_proxy.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p1-stop-delete-proxy" ./scripts/s2_p1_stop_delete_proxy.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p1-stop-delete-probe" ./scripts/s2_p1_stop_delete_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p1-token-inflight-probe" ./scripts/s2_p1_token_inflight_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-two-controllers-probe" ./scripts/s2_p1_two_controllers_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-private-read-proxy" ./scripts/s2_p1_private_read_delay_proxy.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-private-read-probe" ./scripts/s2_p1_private_read_inflight_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-bound-create-probe" ./scripts/s2_p1_bound_create_receipt_fault_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-malformed-exec-proxy" ./scripts/s2_p2_malformed_exec_proxy.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-malformed-exec-probe" ./scripts/s2_p2_malformed_exec_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-altpool-probe" ./scripts/s2_p2_alternate_pool_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p1-list-debugger-probe" ./scripts/s2_p1_operation_list_boundary_probe.go
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p1-native-overlap-probe" ./scripts/s2_p1_native_create_delete_overlap_probe.go
chmod 0755 "$work/boxweave" "$work/s2-probe" "$work/s2-crash-probe" "$work/s2-queued-cancel-seed" "$work/s2-native-guest-quota" "$work/s2-mcp-quota-probe" "$work/s2-exec-gc-probe" "$work/s2-gc-delete-proxy" "$work/s2-sqlite-fsync-probe" "$work/s2-p0-native-sqlite-proxy" "$work/s2-p0-native-sqlite-probe" "$work/s2-ack-loss-probe" "$work/s2-unknown-create-bind-probe" "$work/s2-ack-loss-proxy" "$work/s2-p1-stop-delete-proxy" "$work/s2-p1-stop-delete-probe" "$work/s2-p1-token-inflight-probe" "$work/s2-two-controllers-probe" "$work/s2-private-read-proxy" "$work/s2-private-read-probe" "$work/s2-bound-create-probe" "$work/s2-malformed-exec-proxy" "$work/s2-malformed-exec-probe" "$work/s2-altpool-probe" "$work/s2-p1-list-debugger-probe" "$work/s2-p1-native-overlap-probe"
sha256sum "$work/boxweave"
echo "S4_A_CANDIDATE_BINARY_SHA256=$(sha256sum "$work/boxweave" | cut -d' ' -f1)"
sudo chown boxweave:incus "$work/ledger"
db="$work/ledger/ledger.sqlite"
socket=/var/lib/incus/unix.socket.user

# Initialize this disposable ledger explicitly, not as a side effect of Doctor.
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" status -db "$db" -incus-socket "$socket" -project "$project" > /dev/null

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
assert n["selected_storage_pool_ready"]
assert n["selected_storage_quota_verified"] is False
assert any("STORAGE_QUOTA_UNVERIFIED" in warning for warning in n["warnings"])
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
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" doctor -db "$db" \
  -incus-socket "$socket" -project "$project" > "$work/noquota-doctor.json"
python3 - "$work/noquota-doctor.json" <<'PYTEST'
import json,sys
with open(sys.argv[1]) as f:
    obj=json.load(f)
native=obj["native"]
assert obj["ready"] is True and native["ready"] is True, native
assert native["selected_storage_quota_verified"] is False, native
assert native["selected_storage_pool_ready"] is True, native
assert any("STORAGE_QUOTA_UNVERIFIED" in x for x in native["warnings"]), native
print("S2_NATIVE_DIR_NO_QUOTA_READY_WITH_EXPLICIT_WARNING=PASS")
PYTEST
sudo incus profile device set default root pool=bw-s2-native-pool --project "$project"
sudo incus storage delete bw-s2-noquota-pool
sudo umount "$work/noquota-mounted"
sudo losetup -d "$noquota_loop"

sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" > "$work/admin.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" > "$work/peer.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" > "$work/admin2.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" -ttl 1s > "$work/expired-peer.secret"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$db" > "$work/revoked-peer.secret"
chmod 0600 "$work/admin.secret" "$work/peer.secret" "$work/admin2.secret" "$work/expired-peer.secret" "$work/revoked-peer.secret"
revoked_peer_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/revoked-peer.secret")"
echo "$revoked_peer_id" | grep -Eq '^[0-9a-f]{32}$'
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-revoke -db "$db" -id "$revoked_peer_id" > "$work/revoked-peer-result.txt"
admin="$(sed -n 's/^token=//p' "$work/admin.secret")"
peer="$(sed -n 's/^token=//p' "$work/peer.secret")"
admin2="$(sed -n 's/^token=//p' "$work/admin2.secret")"
expired_peer="$(sed -n 's/^token=//p' "$work/expired-peer.secret")"
revoked_peer="$(sed -n 's/^token=//p' "$work/revoked-peer.secret")"
admin_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/admin.secret")"
admin2_id="$(sed -n 's/^id=\([^ ]*\).*/\1/p' "$work/admin2.secret")"
test -n "$admin" && test -n "$peer" && test -n "$admin2" && test -n "$expired_peer" && test -n "$revoked_peer"
echo "$admin_id" | grep -Eq '^[0-9a-f]{32}$'
echo "$admin2_id" | grep -Eq '^[0-9a-f]{32}$'
test "$admin_id" != "$admin2_id"

sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" serve -db "$db" \
  -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/service.redacted.log" 2>&1 &
service_pid=$!
trap 'if [ -n "${s4_event_tap_pid:-}" ]; then sudo touch "$work/ledger/s4-native-events-stop" 2>/dev/null || :; kill "$s4_event_tap_pid" 2>/dev/null || :; wait "$s4_event_tap_pid" 2>/dev/null || :; fi; kill "$service_pid" 2>/dev/null || :; wait "$service_pid" 2>/dev/null || :' EXIT
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
if [ "${S5A2_RECOVERY_NATIVE_ONLY:-}" = true ]; then
  # This branch inherits the pristine one-use host/Project/ledger guards.
  # The test probe creates a scoped SQLite fault, not a production setting.
  CGO_ENABLED=1 go build -trimpath -o "$work/s5a2-recovery-probe" ./scripts/s5a2_recovery_probe.go
  chmod 0755 "$work/s5a2-recovery-probe"
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted S5A2_RECOVERY_NATIVE_ONLY=true \
    S5A2_DB="$db" S5A2_TOKEN="$admin" "$work/s5a2-recovery-probe"
  # Stop only the exact process spawned by this one-use fixture before the
  # installed CLI observes the offline clone. Never run a second controller.
  sudo kill -TERM "$service_pid"
  wait "$service_pid" 2>/dev/null || :
  stopped=0
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    if ! curl --max-time 1 -sS -o /dev/null http://127.0.0.1:18650/mcp 2>/dev/null; then
      stopped=1
      break
    fi
    sleep 1
  done
  test "$stopped" = 1
  sh scripts/s5a2_installed_cli_review.sh "$work" "$db" "$project" "$socket"
  exit 0
fi
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=INSTALLED_HTTP_MCP_READY"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=DUAL_SERVE_EXCLUSION_START"

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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=DUAL_SERVE_EXCLUSION_PASS"

# Prove a *shipped* older official Go SDK (v1.6.1, protocol <=2025-11-25)
# negotiates against the current v1.8.0 server. Its independent temp module
# cannot silently use the candidate's newer SDK. No native writes are allowed.
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=LEGACY_OFFICIAL_MCP_SDK_START"
legacy_client_dir="$work/legacy-sdk-client"
mkdir -m 0700 "$legacy_client_dir"
# The tracked probe is build-tagged "ignore"; remove only that tag in
# the isolated client module so go mod tidy also resolves its imports.
sed '1,2d' scripts/s2_legacy_sdk_probe.go > "$legacy_client_dir/main.go"
(
  cd "$legacy_client_dir"
  GOWORK=off go mod init boxweave-s2-legacy-client
  GOWORK=off go get github.com/modelcontextprotocol/go-sdk@v1.6.1
  GOWORK=off go mod tidy
  GOWORK=off CGO_ENABLED=0 go build -trimpath -o "$work/s2-legacy-sdk-probe" .
)
chmod 0755 "$work/s2-legacy-sdk-probe"
GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  BW_S2_LEGACY_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_LEGACY_TOKEN="$admin" "$work/s2-legacy-sdk-probe" > "$work/legacy-sdk-probe.log" 2>&1 || {
    cat "$work/legacy-sdk-probe.log"
    echo "S2_P1_FAIL: old SDK interoperability probe" >&2
    exit 125
  }
grep -Fqx "S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_LEGACY_PROTOCOL_16_TOOLS_NO_NATIVE_EFFECT=PASS" "$work/legacy-sdk-probe.log"
grep -Fqx "S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_NATIVE_CREATE_UPDATE_EXEC_DELETE=PASS" "$work/legacy-sdk-probe.log"
echo "S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_LEGACY_PROTOCOL_16_TOOLS_NO_NATIVE_EFFECT=PASS"
echo "S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_NATIVE_CREATE_UPDATE_EXEC_DELETE=PASS"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=LEGACY_OFFICIAL_MCP_SDK_PASS"

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

# TC-006: a real, separately authorized FOREIGN Project has a stopped
# native instance with the SAME NAME as the one this candidate will create.
# Neither the ordinary restricted user nor the SDK should adopt/read/delete it.
foreign_project=bw-s2-foreign-ci-project
foreign_name=bw-s2-ghci-110a
sudo incus project create "$foreign_project" --config features.images=false
sudo incus profile device add default root disk path=/ pool=bw-s2-native-pool \
  --project "$foreign_project"
sudo incus init --empty "$foreign_name" --project "$foreign_project"
foreign_uuid="$(sudo incus config get "$foreign_name" volatile.uuid --project "$foreign_project")"
echo "$foreign_uuid" | grep -Eq '^[0-9a-f-]{36}$'
test "$(sudo incus list "$foreign_name" --project "$foreign_project" --format csv -c n)" = "$foreign_name"
if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
    -u XDG_CONFIG_HOME HOME=/home/boxweave \
    XDG_CONFIG_HOME=/home/boxweave/.config \
    incus info "$foreign_name" --project "$foreign_project" \
    > "$work/foreign-project-native-denied.txt" 2>&1; then
  echo "S2_NATIVE_FAIL: restricted service accessed foreign Project instance" >&2
  exit 121
fi
echo "S2_P1_SECOND_REAL_PROJECT_SAME_NAME_NATIVE_INSTANCE_DENIED_TO_NONROOT=PASS"

# Physical public WAN routing may not be available to a hosted Guest.
# A separately routed namespace+veth+TLS server gives an explicit, stricter
# SIMULATED-WAN ON/OFF/ON proof, but never counts as public Internet.
sh scripts/s2_p1_simulated_wan_lab.sh setup "$work" "$project"
simulated_wan_ca_b64="$(base64 -w0 "$work/s2-simwan-cert.pem")"
test -n "$simulated_wan_ca_b64"

[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=BASELINE_S2_MCP_PROBE_START"
# The independent official MCP SDK prober temporarily holds one completed
# 16MiB+4-byte dual-stream Exec until this CI VM administrator measures its
# ACTUAL native spool files. This is read-only and never deletes native logs.
# A failure is not an excuse to retry an original native mutation.
( BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" BW_S2_PEER_TOKEN="$peer" \
  BW_S2_ADMIN2_TOKEN="$admin2" BW_S2_ADMIN1_ID="$admin_id" \
  BW_S2_ADMIN2_ID="$admin2_id" BW_S2_OUTPUT_WITNESS_DIR="$work" \
  BW_S2_EXPIRED_TOKEN="$expired_peer" BW_S2_REVOKED_TOKEN="$revoked_peer" \
  BW_S2_FOREIGN_LAYER_OPID="$foreign_layer_id" \
  BW_S2_HISTORY_CURSOR_WITNESS_DIR="$work" \
  BW_S2_SIMULATED_WAN_CA_CERT_B64="$simulated_wan_ca_b64" \
  BW_S2_QUEUED_CANCEL_OPID="$queued_cancel_id" "$work/s2-probe" ) > "$work/native-probe.log" 2>&1 &
probe_pid=$!
# The official SDK client pauses immediately after its first 50 records.
# Commit 12 new durable CANCELLED history rows from a separate restricted
# writer before allowing the original HMAC cursor to page older originals.
attempt=0
while [ ! -s "$work/history-cursor.ready" ] && [ "$attempt" -lt 90 ]; do
  if ! kill -0 "$probe_pid" 2>/dev/null; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ ! -s "$work/history-cursor.ready" ]; then
  echo "S2_NATIVE_FAIL: installed MCP client never reached history cursor handshake" >&2
  cat "$work/native-probe.log"
  exit 108
fi
test "$(cat "$work/history-cursor.ready")" = "first-page-observed"
sudo runuser -u boxweave -- env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  BW_S2_SEED_DB="$db" BW_S2_SEED_PROJECT="$project" \
  BW_S2_SEED_ADMIN_TOKEN="$admin" "$work/s2-queued-cancel-seed" append \
  > "$work/history-append.txt"
grep -Fqx "S2_SQLITE_CONCURRENT_HISTORY_APPEND_12=PASS" "$work/history-append.txt"
touch "$work/history-cursor.release"
echo "S2_NATIVE_REAL_SQLITE_HISTORY_WRITER_DURING_MCP_PAGE=PASS"
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
grep -Fqx S2_P1_SIMULATED_WAN_HTTPS_ON_OFF_ON_GUEST=PASS "$work/native-probe.log"
grep -Fqx S2_P1_NATIVE_GUEST_ORIGIN_READY_TRUE_ACTUAL_SOCKET_AND_MCP=PASS "$work/native-probe.log"
grep -Fqx S2_P1_NATIVE_GUEST_READY_RESTART_CLEARS_STALE_TRUE=PASS "$work/native-probe.log"
sh scripts/s2_p1_simulated_wan_lab.sh verify "$work" "$project"
sh scripts/s2_p1_simulated_wan_lab.sh cleanup "$work" "$project"
# The receiving middleware logs only a hashed actor and coarse method/outcome.
# A deliberately injected hostile _meta test string must never appear in logs.
if grep -Fq "S2_P1_META_REDACTION_SENTINEL_110" "$work/service.redacted.log"; then
  echo "S2_P1_FAIL: untrusted _meta content appeared in service logs" >&2
  exit 119
fi
grep -Fqx "S2_P1_REAL_SDK_HOSTILE_META_AND_SCHEMA_NO_NATIVE_EFFECT=PASS" "$work/native-probe.log"
grep -Fqx "S2_P1_REAL_SDK_LEGACY_TEXT_AND_STRUCTURED_RECEIPT_SAME_ID=PASS" "$work/native-probe.log"
echo "S2_P1_INSTALLED_RECEIVING_MIDDLEWARE_REDACTION_AND_TYPED_NEGATIVES=PASS"

# SDK has now independently created, accessed and destroyed its own
# same-named Box while this foreign original remained stopped throughout.
test "$(sudo incus config get "$foreign_name" volatile.uuid --project "$foreign_project")" = "$foreign_uuid"
test "$(sudo incus list "$foreign_name" --project "$foreign_project" --format csv -c n)" = "$foreign_name"
sudo incus delete "$foreign_name" --project "$foreign_project"
sudo incus project delete "$foreign_project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_TWO_REAL_PROJECTS_SAME_NAME_ISOLATED_FOREIGN_UUID_PRESERVED=PASS"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=BASELINE_S2_MCP_PROBE_PASS"

# BH-018: an additional native end-to-end acceptance through the *installed*
# non-root, restricted-Project BoxWeave service and official Go MCP SDK.
# Real Incus 7.6 exits 126/127 cause native Operation Failure even after
# successful Guest side effects. Only the exact original native terminal and
# its bounded output paths allow the atomic SQLite SUCCEEDED receipt.
# A restart of THIS ONE-USE service/ledger must retain original operation
# identity and output without native redispatch; fresh commands must work.
# The original historical UNKNOWN claims on other hosts are never touched.
test -z "$(sudo incus list --all-projects --format csv -c n)"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=BH018_INSTALLED_MCP_PROBE_START"
bh018_checkpoint="$work/bh018-atomic-original.json"
test ! -e "$bh018_checkpoint"
# S4-A only: start THREE independent read-only CLI observers before the
# first disposable native admission; Incus may expire a final Operation
# immediately. They read one-use SQLite WAL and native daemon over the
# restricted service identity's ordinary Incus CLI connection. No replay.
if [ "${S4_A_RAW_WITNESS:-}" = true ]; then
  test -f scripts/s4_raw_operation_witness.py
  # The CI checkout under /home/runner is not necessarily traversable by
  # the restricted service UID. Copy the exact tracked, test-only observer
  # to our own chmod 0711 disposable staging root; never widen checkout,
  # installed service, Incus socket, or host-global permissions.
  install -m 0755 scripts/s4_raw_operation_witness.py "$work/s4-raw-operation-witness.py"
  cmp -s scripts/s4_raw_operation_witness.py "$work/s4-raw-operation-witness.py"
  if ! sudo runuser -u boxweave -- python3 "$work/s4-raw-operation-witness.py" --help >/dev/null 2>&1; then
    echo "S4_A_RAW_NATIVE_WITNESS_RESTRICTED_EXEC=BLOCKED"
    exit 94
  fi
  echo "S4_A_RAW_NATIVE_WITNESS_RESTRICTED_EXEC=PASS"
  # Subscribe through the official Incus Go SDK BEFORE dispatching a fresh
  # effect. The native daemon can expire completed Operations immediately;
  # the read-only event stream preserves the original final metadata digest.
  # Only the one-use restricted service user and its own Project are used.
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
    -u XDG_CONFIG_HOME HOME=/home/boxweave \
    XDG_CONFIG_HOME=/home/boxweave/.config \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    "$work/s4-native-event-tap" -socket "$socket" -project "$project" \
    -output "$work/ledger/s4-native-events.jsonl" \
    -ready "$work/ledger/s4-native-events-ready" \
    -stop "$work/ledger/s4-native-events-stop" \
    > "$work/s4-native-event-tap.redacted.log" 2>&1 &
  s4_event_tap_pid=$!
  attempt=0
  until sudo test -s "$work/ledger/s4-native-events-ready" || [ "$attempt" -ge 100 ]; do
    if ! kill -0 "$s4_event_tap_pid" 2>/dev/null; then break; fi
    attempt=$((attempt+1))
    sleep 0.1
  done
  if ! sudo grep -Fqx 'PROJECT_OPERATION_SUBSCRIBED' "$work/ledger/s4-native-events-ready" 2>/dev/null; then
    if grep -Fq 'S4_EVENT_TAP_FAIL=PROJECT_OPERATION_SUBSCRIPTION_DENIED' "$work/s4-native-event-tap.redacted.log"; then
      echo "S4_A_RAW_NATIVE_EVENT_SUBSCRIPTION=PROJECT_DENIED"
    else
      echo "S4_A_RAW_NATIVE_EVENT_SUBSCRIPTION=BLOCKED"
    fi
    exit 94
  fi
  echo "S4_A_RAW_NATIVE_EVENT_SUBSCRIPTION=PASS"
  s4_witness_pids=""
  for case in create exit126 exit127; do
    case "$case" in
      create) key="s2-bh018-atomic-native-create-001" ;;
      exit126) key="s2-bh018-original-exit-126-001" ;;
      exit127) key="s2-bh018-original-exit-127-001" ;;
    esac
    digest="$(printf %s "$key" | sha256sum | cut -d ' ' -f 1)"
    sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
      -u XDG_CONFIG_HOME HOME=/home/boxweave \
      XDG_CONFIG_HOME=/home/boxweave/.config \
      python3 "$work/s4-raw-operation-witness.py" \
      --db "$db" --key-digest "$digest" --timeout-seconds 120 \
      --events-jsonl "$work/ledger/s4-native-events.jsonl" \
      > "$work/s4-raw-$case.json" 2> "$work/s4-raw-$case.redacted.err" &
    s4_witness_pids="$s4_witness_pids $!"
  done
  unset key digest
fi
bh018_start_rc=0
( BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" BW_S2_BH018_CHECKPOINT="$bh018_checkpoint" \
  "$work/s2-bh018-atomic-mcp-probe" start ) > "$work/bh018-atomic-start.log" 2>&1 || bh018_start_rc=$?
if [ "$bh018_start_rc" -ne 0 ]; then
  echo "S4_A_NATIVE_STAGE=BH018_INSTALLED_START_FAILED"
  if grep -Eq 'BH018_MCP_FAIL: original [0-9a-f]{32} stopped in UNKNOWN' "$work/bh018-atomic-start.log"; then
    echo 'S4_A_BH018_FAILURE_CLASS=ORIGINAL_TERMINAL_UNKNOWN'
  elif grep -Eq 'BH018_MCP_FAIL: original [0-9a-f]{32} stopped in FAILED' "$work/bh018-atomic-start.log"; then
    echo 'S4_A_BH018_FAILURE_CLASS=ORIGINAL_TERMINAL_FAILED'
  elif grep -Fq 'BH018_MCP_FAIL: MCP box_create rejected' "$work/bh018-atomic-start.log"; then
    echo 'S4_A_BH018_FAILURE_CLASS=CREATE_MCP_REJECTED'
  elif grep -Fq 'BH018_MCP_FAIL: MCP box_exec rejected' "$work/bh018-atomic-start.log"; then
    echo 'S4_A_BH018_FAILURE_CLASS=EXEC_MCP_REJECTED'
  elif grep -Fq 'BH018_MCP_FAIL: MCP operation_get rejected' "$work/bh018-atomic-start.log"; then
    echo 'S4_A_BH018_FAILURE_CLASS=ORIGINAL_READ_REJECTED'
  elif grep -Fq 'BH018_MCP_FAIL: terminal confirmation deadline exceeded' "$work/bh018-atomic-start.log"; then
    echo 'S4_A_BH018_FAILURE_CLASS=TERMINAL_POLL_TIMEOUT'
  elif grep -Fq 'BH018_MCP_FAIL: original guest exit was not retained' "$work/bh018-atomic-start.log"; then
    echo 'S4_A_BH018_FAILURE_CLASS=GUEST_EXIT_UNVERIFIED'
  else
    echo 'S4_A_BH018_FAILURE_CLASS=UNTYPED_CLIENT_FAILURE'
  fi
  # Read the one-use WAL through an independent, read-only local process.
  # Print only immutable original IDs and bounded state/error enums, never
  # the original Guest command/output or bearer credentials.
  sudo python3 - "$db" <<'S4_BH018_FAILED_RECEIPT'
import hashlib
import sqlite3
import sys
db = sys.argv[1]
originals = (
    ("create", "s2-bh018-atomic-native-create-001"),
    ("exit126", "s2-bh018-original-exit-126-001"),
    ("exit127", "s2-bh018-original-exit-127-001"),
)
try:
    with sqlite3.connect("file:" + db + "?mode=ro", uri=True, timeout=3) as conn:
        for label, key in originals:
            digest = hashlib.sha256(key.encode()).hexdigest()
            rows = conn.execute(
                "SELECT id,state,native_operation_id,error_class "
                "FROM operations WHERE key_digest=? LIMIT 2", (digest,)
            ).fetchall()
            if len(rows) == 1:
                oid, status, native_id, error_class = rows[0]
                print("S4_A_BH018_ORIGINAL_CASE=" + label + " BW_ID=" + oid
                      + " STATE=" + status + " NATIVE_ID=" + (native_id or "NONE")
                      + " ERROR_CLASS=" + (error_class or "NONE"))
            elif len(rows) > 1:
                print("S4_A_BH018_ORIGINAL_CASE=" + label + " AMBIGUOUS")
            else:
                print("S4_A_BH018_ORIGINAL_CASE=" + label + " NOT_ADMITTED")
except (sqlite3.Error, OSError):
    print("S4_A_BH018_SQLITE_POSTMORTEM=UNAVAILABLE")
S4_BH018_FAILED_RECEIPT
  exit 95
fi
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=BH018_INSTALLED_START_PASS"
grep -Fqx 'BH018_NATIVE_REAL_HTTP_MCP_EXIT_126_ATOMIC_RECEIPT=PASS' "$work/bh018-atomic-start.log"
grep -Fqx 'BH018_NATIVE_REAL_HTTP_MCP_EXIT_127_ATOMIC_RECEIPT=PASS' "$work/bh018-atomic-start.log"
grep -Fqx 'BH018_NATIVE_REAL_HTTP_MCP_ATOMIC_START=PASS' "$work/bh018-atomic-start.log"
if [ "${S4_A_RAW_WITNESS:-}" = true ]; then
  # Capture witnesses are test-only: do not print bearer tokens, raw stderr,
  # unmanaged operation messages, or assume that a missing object failed.
  for p in $s4_witness_pids; do
    wait "$p" || :
  done
  for case in create exit126 exit127; do
    if ! test -s "$work/s4-raw-$case.json"; then
      if grep -Fq 'sqlite3.OperationalError:' "$work/s4-raw-$case.redacted.err"; then
        echo "S4_A_RAW_WITNESS_${case}=SQLITE_READ_FAILURE"
      elif grep -Eq 'PermissionError:|FileNotFoundError:' "$work/s4-raw-$case.redacted.err"; then
        echo "S4_A_RAW_WITNESS_${case}=FILE_ACCESS_DENIED"
      else
        echo "S4_A_RAW_WITNESS_${case}=UNCLASSIFIED_FAILURE"
      fi
    fi
  done
  python3 - "$work" <<'S4_RAW_ORIGINAL_CHECK'
import json
import pathlib
import sys
root = pathlib.Path(sys.argv[1])
seen = []
for name in ("create", "exit126", "exit127"):
    try:
        with (root / ("s4-raw-" + name + ".json")).open() as stream:
            result = json.load(stream)
    except (OSError, ValueError, TypeError):
        # A crashed observer is not an observed terminal, and no raw error
        # log or original Guest output is exposed to the public CI log.
        print("S4_A_RAW_NATIVE_" + name.upper() + "=WITNESS_UNREADABLE")
        continue
    original = result.get("original_sqlite") or {}
    terminal = result.get("native_terminal") or {}
    if (result.get("observation") == "CAPTURED_TERMINAL"
            and result.get("original_id_matches") is True
            and original.get("project")
            and original.get("native_operation_id") == terminal.get("native_operation_id")
            and terminal.get("status_code") in (200, 400, 401)):
        seen.append((name, original, terminal))
    else:
        reason = result.get("reason")
        allowed = {
            "SQLITE_FRESH_ORIGINAL_NOT_DISCOVERED",
            "SQLITE_ORIGINAL_NOT_FOUND",
            "NO_NATIVE_ORIGINAL_RECEIPT",
            "NATIVE_READ_UNAVAILABLE",
            "NATIVE_NOT_RETAINED_OR_UNAVAILABLE",
            "NATIVE_METADATA_MISSING",
            "NATIVE_ORIGINAL_ID_MISMATCH",
            "NATIVE_STATUS_UNVERIFIED",
            "NATIVE_RESPONSE_INVALID",
            "NATIVE_NOT_TERMINAL",
            "NATIVE_ID_FORMAT_UNVERIFIED",
        }
        reason = reason if reason in allowed else "UNCLASSIFIED"
        print("S4_A_RAW_NATIVE_" + name.upper() + "=NOT_CAPTURED_" + reason)
if not seen:
    print("S4_A_RAW_NATIVE_ORIGINAL_TERMINAL=NOT_CAPTURED")
    # Continue independent OAuth/native tests. Missing retained raw Op is
    # still a failing final acceptance gate, not a reason to omit other proof.
for name, original, native in seen:
    print("S4_A_RAW_NATIVE_CASE_" + name.upper() + "=PASS")
    print("S4_A_NATIVE_RAW_TERMINAL_CASE=" + name
          + " BW_ID=" + original["operation_id"]
          + " NATIVE_ID=" + native["native_operation_id"]
          + " PROJECT=" + original["project"]
          + " BOX_UUID=" + str(original.get("box_uuid") or "PENDING"))
    print("S4_A_NATIVE_RAW_TERMINAL_STATUS=" + str(native["status_code"]))
if seen:
    print("S4_A_RAW_NATIVE_ORIGINAL_TERMINAL=PASS")
if any(native.get("evidence_source") == "INCUS_OPERATION_EVENT_WEBSOCKET"
       for _, _, native in seen):
    print("S4_A_RAW_NATIVE_EVENT_ORIGINAL_TERMINAL=PASS")
else:
    print("S4_A_RAW_NATIVE_EVENT_ORIGINAL_TERMINAL=NOT_CAPTURED")
S4_RAW_ORIGINAL_CHECK
fi
# Emit only scoped, validated identifiers from the disposable client; no token or output bytes.
echo "BH018_MCP_PROJECT=$project"
grep '^BH018_MCP_' "$work/bh018-atomic-start.log"
if [ "${S4_A_RAW_WITNESS:-}" = true ]; then
  S4_A_WORK="$work" S4_A_DB="$db" S4_A_PROJECT="$project" \
    sh scripts/s4_oauth_native_integration.sh

fi
echo "BH018_NATIVE_INSTALLED_MCP_ORIGINAL_126_127_ATOMIC_RECEIPTS=PASS"

sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
# A runuser/sudo wrapper can exit before its child has released the SQLite
# serve lock; wait for the exact owned nonroot service, not merely the wrapper.
attempt=0
while sudo pgrep -u boxweave -f "^$work/boxweave serve" >/dev/null; do
  if [ "$attempt" -ge 30 ]; then
    echo "BH018_INSTALLED_NORMAL_RESTART_OLD_SERVICE_STILL_ACTIVE=BLOCKED" >&2
    exit 92
  fi
  attempt=$((attempt+1))
  sleep 1
done
echo "BH018_INSTALLED_NORMAL_RESTART_OLD_SERVICE_EXITED=PASS"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
  -u XDG_CONFIG_HOME HOME=/home/boxweave \
  XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/bh018-restarted-service.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 40 ]; then
  echo "BH018_INSTALLED_NORMAL_RESTART_NOT_READY=BLOCKED" >&2
  if grep -Fqi 'serve already active for this database' "$work/bh018-restarted-service.redacted.log"; then
    echo "BH018_INSTALLED_NORMAL_RESTART_CLASS=SQLITE_SERVE_LOCK" >&2
  elif grep -Fqi 'address already in use' "$work/bh018-restarted-service.redacted.log"; then
    echo "BH018_INSTALLED_NORMAL_RESTART_CLASS=LISTEN_IN_USE" >&2
  elif grep -Fqi 'permission denied' "$work/bh018-restarted-service.redacted.log"; then
    echo "BH018_INSTALLED_NORMAL_RESTART_CLASS=PERMISSION_DENIED" >&2
  else
    echo "BH018_INSTALLED_NORMAL_RESTART_CLASS=UNCLASSIFIED" >&2
  fi
  exit 93
fi
echo "BH018_INSTALLED_NORMAL_RESTART_HTTP_401=PASS"
# Preserve the one-use client exit status instead of losing it through
# a negated shell condition. Never print raw stderr, Authorization headers,
# transport errors, Guest output or secrets in a public CI log.
bh018_resume_rc=0
( BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" \
  BW_S2_ADMIN_TOKEN="$admin" BW_S2_BH018_CHECKPOINT="$bh018_checkpoint" \
  "$work/s2-bh018-atomic-mcp-probe" resume ) > "$work/bh018-atomic-resume.log" 2>&1 || bh018_resume_rc=$?
if [ "$bh018_resume_rc" -ne 0 ]; then
  echo "BH018_INSTALLED_NORMAL_RESTART_RESUME_EXIT_CODE=$bh018_resume_rc"
  # Emit only fixed, secret-free phase and original-ID witnesses.
  grep '^BH018_MCP_RESUME_PHASE=' "$work/bh018-atomic-resume.log" | tail -n 16 || :
  grep '^BH018_MCP_FRESH_ADMITTED_EXIT_' "$work/bh018-atomic-resume.log" || :
  grep '^BH018_MCP_TYPED_' "$work/bh018-atomic-resume.log" || :
  if grep -Fq 'BH018_MCP_FAIL: MCP box_exec rejected' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=FRESH_EXEC_MCP_REJECTED'
  elif grep -Fq 'BH018_MCP_FAIL: lost immutable BW original operation ID' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=FRESH_RECEIPT_ID_MISSING'
  elif grep -Fq 'BH018_MCP_FAIL: MCP operation_get rejected' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=ORIGINAL_OPERATION_READ_REJECTED'
  elif grep -Fq 'BH018_MCP_FAIL: MCP operation_output_get rejected' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=ORIGINAL_OUTPUT_READ_REJECTED'
  elif grep -Fq 'BH018_MCP_FAIL: unexpected stdout for original' "$work/bh018-atomic-resume.log"; then
    if grep -Fq 'class=DUPLICATE_EFFECT' "$work/bh018-atomic-resume.log"; then
      echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=DUPLICATE_GUEST_EFFECT_MARKER'
    elif grep -Fq 'class=EMPTY' "$work/bh018-atomic-resume.log"; then
      echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=EMPTY_GUEST_EFFECT_MARKER'
    elif grep -Fq 'class=EXTRA_BYTES' "$work/bh018-atomic-resume.log"; then
      echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=EXTRA_GUEST_EFFECT_BYTES'
    else
      echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=WRONG_GUEST_EFFECT_MARKER'
    fi
  elif grep -Fq 'BH018_MCP_FAIL: original guest exit was not retained' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=RECORDED_GUEST_EXIT_MISSING'
  elif grep -Fq 'BH018_MCP_FAIL: new readback lacks native identity' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=FRESH_NATIVE_ID_MISSING'
  elif grep -Fq 'BH018_MCP_FAIL: original receipt ID drift' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=ORIGINAL_RECEIPT_ID_DRIFT'
  elif grep -Fq 'BH018_MCP_FAIL: confused native terminal and guest command_success' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=RECORDED_GUEST_SUCCESS_DRIFT'
  elif grep -Fq 'BH018_MCP_FAIL: base64 output' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=RECORDED_GUEST_OUTPUT_INVALID'
  elif grep -Fq 'BH018_MCP_FAIL: terminal confirmation deadline exceeded' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=TERMINAL_POLL_TIMEOUT'
  elif grep -Fq 'BH018_MCP_FAIL: original ' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=ORIGINAL_TERMINAL_UNVERIFIED'
  elif grep -Fq 'BH018_MCP_FAIL:' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=OTHER_VERIFIER_ASSERTION'
  elif [ "$bh018_resume_rc" -eq 137 ]; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=PROCESS_KILLED_137'
  elif grep -Eq 'fatal error:|runtime: out of memory' "$work/bh018-atomic-resume.log"; then
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=GO_RUNTIME_ABORT'
  else
    echo 'BH018_INSTALLED_NORMAL_RESTART_FAILURE_CLASS=UNTYPED_CLIENT_FAILURE'
  fi
  # Read-only postmortem of the fresh original: never log Guest bytes or tokens.
  fresh_read_id="$(sed -n 's/^BH018_MCP_FRESH_ADMITTED_EXIT_[0-9]*_BW_ID=\([0-9a-f]*\)$/\1/p' "$work/bh018-atomic-resume.log" | tail -n1)"
  if printf '%s' "$fresh_read_id" | grep -Eq '^[0-9a-f]{32}$'; then
    sudo runuser -u boxweave -- python3 - "$db" "$fresh_read_id" <<'S4_BH018_RESTART_INSPECT'
import sqlite3, sys
try:
    with sqlite3.connect("file:" + sys.argv[1] + "?mode=ro", uri=True, timeout=4) as conn:
        row = conn.execute(
            "SELECT state,native_operation_id,error_class,box_uuid FROM operations WHERE id=?",
            (sys.argv[2],)).fetchone()
    if row:
        state, native_id, error, box_uuid = row
        allowed_states = {"ADMITTED", "CLAIMED", "DISPATCHING", "VERIFYING", "SUCCEEDED", "FAILED", "UNKNOWN", "CANCELLED"}
        safe = lambda x: x if isinstance(x, str) and x.isupper() and len(x) <= 100 and all(c.isupper() or c.isdigit() or c == "_" for c in x) else "REDACTED"
        print("BH018_INSTALLED_FAILURE_SQLITE_ORIGINAL=" + sys.argv[2]
              + " STATE=" + (state if state in allowed_states else "UNVERIFIED")
              + " ERROR_CLASS=" + safe(error or "NONE")
              + " NATIVE_ID_PRESENT=" + str(bool(native_id))
              + " BOX_UUID_PRESENT=" + str(bool(box_uuid)))
    else:
        print("BH018_INSTALLED_FAILURE_SQLITE_ORIGINAL=NOT_FOUND")
except (sqlite3.Error, OSError):
    print("BH018_INSTALLED_FAILURE_SQLITE_READ=UNAVAILABLE")
S4_BH018_RESTART_INSPECT
  else
    echo "BH018_INSTALLED_FAILURE_FRESH_ORIGINAL_ID=NOT_CAPTURED"
  fi
  # Independent admin-native Guest observation for a known, disposable,
  # exact UUID. Read only a fixed test marker's line count, no output bytes.
  if grep -Fq 'BH018_MCP_FAIL: unexpected stdout for original' "$work/bh018-atomic-resume.log"; then
    observed_uuid="$(sudo incus config get bw-s2-bh018-atomic-native volatile.uuid --project "$project" 2>/dev/null || :)"
    expected_uuid="$(python3 - "$bh018_checkpoint" <<'S4_BH018_EXPECT_UUID'
import json,sys
try:
    print(json.load(open(sys.argv[1]))["box_id"])
except (OSError,ValueError,KeyError):
    print("")
S4_BH018_EXPECT_UUID
)"
    if [ -n "$expected_uuid" ] && [ "$expected_uuid" = "$observed_uuid" ]; then
      for exit_case in 126 127; do
        if count="$(sudo incus exec bw-s2-bh018-atomic-native --project "$project" -- /bin/sh -c "wc -l < /var/tmp/bh018-atomic-effect-$exit_case" 2>/dev/null)" &&
            printf '%s' "$count" | grep -Eq '^[0-9]{1,8}$'; then
          echo "BH018_INSTALLED_NATIVE_GUEST_MARKER_${exit_case}_LINES=$count"
        else
          echo "BH018_INSTALLED_NATIVE_GUEST_MARKER_${exit_case}_LINES=UNAVAILABLE"
        fi
      done
    else
      echo 'BH018_INSTALLED_NATIVE_GUEST_MARKER_WITNESS=UUID_UNVERIFIED'
    fi
  fi
  echo "BH018_INSTALLED_NORMAL_RESTART_MCP_VERIFIER=BLOCKED" >&2
  exit 96
fi
grep -Fqx 'BH018_NATIVE_REAL_HTTP_MCP_EXIT_126_DEDUPE_READBACK_AFTER_RESTART=PASS' "$work/bh018-atomic-resume.log"
grep -Fqx 'BH018_NATIVE_REAL_HTTP_MCP_EXIT_127_DEDUPE_READBACK_AFTER_RESTART=PASS' "$work/bh018-atomic-resume.log"
grep -Fqx 'BH018_NATIVE_REAL_HTTP_MCP_ATOMIC_RESTART_AND_CLEANUP=PASS' "$work/bh018-atomic-resume.log"
# Preserve original native IDs and new readback/cleanup IDs in sanitized CI evidence.
grep '^BH018_MCP_' "$work/bh018-atomic-resume.log"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "BH018_NATIVE_INSTALLED_MCP_ATOMIC_RESTART_IDEMPOTENCY_AND_CLEANUP=PASS"

if [ "${S4_A_RAW_WITNESS:-}" = true ]; then
  # The pre-dispatch native subscription survives controller restart and
  # records the original Delete even if Incus expires its object immediately.
  digest="$(printf %s s2-bh018-atomic-native-delete-001 | sha256sum | cut -d' ' -f1)"
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
    -u XDG_CONFIG_HOME HOME=/home/boxweave \
    XDG_CONFIG_HOME=/home/boxweave/.config \
    python3 "$work/s4-raw-operation-witness.py" \
    --db "$db" --key-digest "$digest" --timeout-seconds 20 \
    --events-jsonl "$work/ledger/s4-native-events.jsonl" > "$work/s4-raw-delete.json"
  sudo runuser -u boxweave -- python3 - "$db" "$work/s4-raw-delete.json" <<'S4_DELETE_CHECK'
import json,sqlite3,sys
with open(sys.argv[2]) as handle:
    record=json.load(handle)
original=record.get("original_sqlite") or {}
native=record.get("native_terminal") or {}
assert record.get("observation")=="CAPTURED_TERMINAL", "original Delete terminal NOT_CAPTURED"
assert record.get("original_id_matches") is True
assert original["action"]=="box_delete" and original["state"]=="SUCCEEDED"
assert original["native_operation_id"]==native["native_operation_id"]
assert native["status_code"]==200 and native["error_present"] is False
with sqlite3.connect("file:"+sys.argv[1]+"?mode=ro",uri=True) as db:
    tombstone=db.execute("SELECT state,deleted_by_operation_id,deleted_at FROM owned_boxes WHERE uuid=? AND project=?",(original["box_uuid"],original["project"])).fetchone()
    assert tombstone and tombstone[0]=="DELETED" and tombstone[1]==original["operation_id"] and tombstone[2]>0
    assert db.execute("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'",(original["operation_id"],)).fetchone()[0]==1
    assert db.execute("SELECT COUNT(*) FROM operations WHERE action='box_delete' AND box_uuid=?",(original["box_uuid"],)).fetchone()[0]==1
    # Witnesses may have captured a VERIFYING journal before the terminal
    # transaction. Independently re-read exact originals after cleanup.
    from pathlib import Path
    for case,action,guest_exit in (("create","box_create",None),("exit126","box_exec",126),("exit127","box_exec",127)):
        with (Path(sys.argv[2]).parent/("s4-raw-"+case+".json")).open() as handle:
            earlier=json.load(handle)
        prior=earlier.get("original_sqlite") or {}
        terminal=earlier.get("native_terminal") or {}
        assert earlier.get("observation")=="CAPTURED_TERMINAL"
        row=db.execute("SELECT action,state,layer,box_uuid,native_operation_id FROM operations WHERE id=?",(prior["operation_id"],)).fetchone()
        assert row==(action,"SUCCEEDED",original["project"],original["box_uuid"],terminal["native_operation_id"])
        assert prior["native_operation_id"]==terminal["native_operation_id"]
        if guest_exit is not None:
            assert db.execute("SELECT exit_code FROM exec_outputs WHERE operation_id=?",(prior["operation_id"],)).fetchone()==(guest_exit,)
        else:
            assert terminal["status_code"]==200 and terminal["error_present"] is False
print("S4_A_RAW_CREATE_EXEC_FINAL_SQLITE_UUID_EXIT_CORRELATION=PASS")
print("S4_A_RAW_DELETE_BW_ID="+original["operation_id"]+" NATIVE_ID="+native["native_operation_id"]+" BOX_UUID="+original["box_uuid"]+" PROJECT="+original["project"])
print("S4_A_RAW_DELETE_TERMINAL_ATOMIC_TOMBSTONE_SINGLE_ORIGINAL=PASS")
S4_DELETE_CHECK
  # S5A1: compare each newly admitted post-restart readback with its
  # own independent native terminal, never the earlier 126/127 original.
  expected_readback_uuid="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["box_id"])' "$bh018_checkpoint")"
  for exit_case in 126 127; do
    digest="$(printf %s "s2-bh018-independent-readback-${exit_case}-001" | sha256sum | cut -d' ' -f1)"
    sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
      -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
      python3 "$work/s4-raw-operation-witness.py" \
      --db "$db" --key-digest "$digest" --timeout-seconds 20 \
      --events-jsonl "$work/ledger/s4-native-events.jsonl" > "$work/s5a1-raw-readback-$exit_case.json"
    sudo runuser -u boxweave -- python3 - "$db" "$work/s5a1-raw-readback-$exit_case.json" "$expected_readback_uuid" "$exit_case" <<'S5A1_READBACK_CHECK'
import json, sqlite3, sys
with open(sys.argv[2]) as handle:
    witness = json.load(handle)
expected_box = sys.argv[3]
original = witness.get("original_sqlite") or {}
native = witness.get("native_terminal") or {}
assert witness.get("observation") == "CAPTURED_TERMINAL"
assert witness.get("original_id_matches") is True
assert original["action"] == "box_exec" and original["box_uuid"] == expected_box
assert original["state"] == "SUCCEEDED" and original["guest_exit_code"] == 0
assert original["native_operation_id"] == native["native_operation_id"]
assert native["status_code"] == 200 and native["error_present"] is False
with sqlite3.connect("file:" + sys.argv[1] + "?mode=ro", uri=True) as db:
    assert db.execute("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", (original["operation_id"],)).fetchone() == (1,)
print("BH018_S5A1_FRESH_READBACK_" + sys.argv[4] + "=PASS BW_ID=" + original["operation_id"] + " NATIVE_ID=" + native["native_operation_id"])
S5A1_READBACK_CHECK
  done
  # The observer must remain connected through the OAuth refresh/once
  # effect. Stop only the test-owned stream after independent checks.
  sudo runuser -u boxweave -- touch "$work/ledger/s4-native-events-stop"
  if ! wait "$s4_event_tap_pid"; then
    echo "S4_A_RAW_NATIVE_EVENT_STREAM=DISCONNECTED" >&2
    exit 94
  fi
  s4_event_tap_pid=""
  echo "S4_A_RAW_NATIVE_EVENT_STREAM=PASS"
fi


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

# BH-018: kill the actual one-use installed BoxWeave daemon when its
# original Guest Exec terminal SQLite WAL commit begins. Unlike the earlier
# separate SQLite helper SIGKILL and already-completed-output restart, this
# uses the same production FinalizeVerifiedExec path as HTTP MCP box_exec.
# It starts a NEW named fixture: an unrelated UNKNOWN is never reclaimed.
command -v cc >/dev/null
cc -shared -fPIC -O2 -std=gnu11 -Wall -Wextra \
  -o "$work/s2-bh018-service-cut.so" scripts/s2_sqlite_fsync_cut.c -ldl
chmod 0755 "$work/s2-bh018-service-cut.so"
cut_state="$work/bh018-service-cut-original.json"
cut_arm="$work/ledger/bh018-service-cut-arm"
cut_witness="$work/ledger/bh018-service-cut-witness"
test ! -e "$cut_state"
sudo test ! -e "$cut_arm"
sudo test ! -e "$cut_witness"
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  BW_S2_FSYNC_DB_WAL="$db-wal" BW_S2_FSYNC_ARM_FILE="$cut_arm" \
  BW_S2_FSYNC_WITNESS_FILE="$cut_witness" BW_S2_FSYNC_CUT_MODE=before \
  LD_PRELOAD="$work/s2-bh018-service-cut.so" \
  "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/bh018-service-cut-start.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
BW_S2_CRASH_VARIANT=bh018-final-wal-cut BW_S2_CRASH_STATE_FILE="$cut_state" \
  BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" BW_S2_ADMIN_TOKEN="$admin" \
  "$work/s2-crash-probe" start > "$work/bh018-service-cut-original.redacted.log" 2>&1
test -s "$cut_state"
sudo python3 - "$db" "$cut_state" <<'PY_BH018_COMMIT_CUT'
import json,sqlite3,sys
with open(sys.argv[2]) as f: original=json.load(f)
with sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True,timeout=3) as c:
    row=c.execute('SELECT state,native_operation_id FROM operations WHERE id=?',
                  (original['operation_id'],)).fetchone()
assert row == ('VERIFYING', original['native_operation_id']), (row,original)
print('BH018_INSTALLED_ORIGINAL_NATIVE_ACK_BEFORE_COMMIT_CUT=PASS')
print('BH018_INSTALLED_CUT_ORIGINAL_BW_ID='+original['operation_id'])
print('BH018_INSTALLED_CUT_ORIGINAL_NATIVE_ID='+original['native_operation_id'])
PY_BH018_COMMIT_CUT
sudo touch "$cut_arm"
attempt=0
until [ "$attempt" -ge 45 ]; do
  if sudo test -s "$cut_witness"; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
sudo grep -Fqx 'ORIGINAL_SQLITE_WAL_FSYNC_CUT_ACTUALLY_TRIGGERED' "$cut_witness"
wait "$service_pid" 2>/dev/null || :
sudo rm -f -- "$cut_arm"
echo 'BH018_INSTALLED_SERVICE_KILLED_DURING_REAL_EXEC_SQLITE_WAL_COMMIT=PASS'
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/bh018-service-cut-restarted.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 40 ]; then
  echo "BH018_INSTALLED_SERVICE_CUT_RESTART_HTTP_401=BLOCKED" >&2
  if grep -Fqi 'serve already active for this database' "$work/bh018-service-cut-restarted.redacted.log"; then
    echo "BH018_INSTALLED_SERVICE_CUT_RESTART_CLASS=SQLITE_SERVE_LOCK" >&2
  else
    echo "BH018_INSTALLED_SERVICE_CUT_RESTART_CLASS=UNCLASSIFIED" >&2
  fi
  exit 94
fi
echo "BH018_INSTALLED_SERVICE_CUT_RESTART_HTTP_401=PASS"
if ! ( BW_S2_CRASH_VARIANT=bh018-final-wal-cut BW_S2_CRASH_STATE_FILE="$cut_state" \
  BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" BW_S2_ADMIN_TOKEN="$admin" \
  "$work/s2-crash-probe" resume ) >"$work/bh018-service-cut-resume.redacted.log" 2>&1; then
  grep -E '^BH018_|^S2_NATIVE_|S2_CRASH_ACCEPTANCE_FAILED' "$work/bh018-service-cut-resume.redacted.log" | tail -n 15 || :
  echo "BH018_INSTALLED_SERVICE_CUT_RECOVERY_MCP_VERIFIER=BLOCKED" >&2
  exit 95
fi
grep -Fqx 'S2_NATIVE_COMPLETED_OUTPUT_AFTER_SQLITE_RESTART=PASS' "$work/bh018-service-cut-resume.redacted.log"
grep -E '^BH018_INSTALLED_WAL_KILL_ATOMIC_(SUCCEEDED|UNKNOWN)_' "$work/bh018-service-cut-resume.redacted.log"
# An independent sqlite3 read-only oracle must see exactly one of two
# transactions: all three terminal rows committed OR none (UNKNOWN + lock).
sudo python3 - "$db" "$cut_state" <<'PY_BH018_ATOMIC_ORACLE'
import json,sqlite3,sys
db,checkpoint=sys.argv[1:]
with open(checkpoint) as f: original=json.load(f)
oid,native=original["operation_id"],original["native_operation_id"]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True,timeout=3) as conn:
    op=conn.execute("SELECT state,native_operation_id FROM operations WHERE id=?",(oid,)).fetchone()
    outs=conn.execute("SELECT stdout_path,stderr_path,exit_code FROM exec_outputs WHERE operation_id=?",(oid,)).fetchall()
    audit=conn.execute("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'",(oid,)).fetchone()[0]
assert op and op[1]==native, (op,native)
if op[0]=="SUCCEEDED":
    assert len(outs)==1 and outs[0][2]==0 and audit==1,(outs,audit)
    prefix="/1.0/instances/bw-s2-bh018-final-cut/logs/exec-output/exec_"+native
    assert any(outs[0][:2])
    assert outs[0][0] in ("",prefix+".stdout") and outs[0][1] in ("",prefix+".stderr"),outs
    print("BH018_INSTALLED_WAL_KILL_SQLITE_ALL_TERMINAL_ROWS_ATOMIC=PASS")
elif op[0]=="UNKNOWN":
    assert not outs and audit==0,(outs,audit)
    print("BH018_INSTALLED_WAL_KILL_SQLITE_NO_PARTIAL_TERMINAL_ROWS=PASS")
else:
    raise SystemExit("BH018 WAL SIGKILL recovered invalid/split state: "+op[0])
print("BH018_INSTALLED_WAL_KILL_SQLITE_TERMINAL_OR_UNKNOWN_CROSSCHECK=PASS")
PY_BH018_ATOMIC_ORACLE

attempt=0
count=""
until [ "$attempt" -ge 45 ]; do
  count="$(sudo incus exec bw-s2-bh018-final-cut --project "$project" -- \
    /bin/sh -c 'wc -l < /tmp/bw-s2-bh018-final-cut-marker' 2>/dev/null || :)"
  if [ "$count" = 1 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
test "$count" = 1
echo 'BH018_INSTALLED_SERVICE_PRECOMMIT_SIGKILL_GUEST_EFFECT_ONCE=PASS'
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-bh018-final-cut
sudo incus stop bw-s2-bh018-final-cut --project "$project"
sudo incus delete bw-s2-bh018-final-cut --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo 'BH018_INSTALLED_SERVICE_WAL_COMMIT_SIGKILL_UNKNOWN_LOCK_AND_TEARDOWN=PASS'

# BH-018 separate post-fsync/ACK-loss original terminal, in a distinct
# one-use named Box and idempotency scope. No earlier UNKNOWN is touched.
# The test-only interposer SIGKILLs the installed service AFTER the real
# WAL sync returns, before the normal commit caller receives its ACK.
command -v cc >/dev/null
cc -shared -fPIC -O2 -std=gnu11 -Wall -Wextra \
  -o "$work/s2-bh018-service-after-cut.so" scripts/s2_sqlite_fsync_cut.c -ldl
chmod 0755 "$work/s2-bh018-service-after-cut.so"
cut_state="$work/bh018-service-after-cut-original.json"
cut_arm="$work/ledger/bh018-service-after-cut-arm"
cut_witness="$work/ledger/bh018-service-after-cut-witness"
test ! -e "$cut_state"
sudo test ! -e "$cut_arm"
sudo test ! -e "$cut_witness"
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  BW_S2_FSYNC_DB_WAL="$db-wal" BW_S2_FSYNC_ARM_FILE="$cut_arm" \
  BW_S2_FSYNC_WITNESS_FILE="$cut_witness" BW_S2_FSYNC_CUT_MODE=after \
  LD_PRELOAD="$work/s2-bh018-service-after-cut.so" \
  "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/bh018-service-after-cut-start.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
BW_S2_CRASH_VARIANT=bh018-final-wal-cut-after BW_S2_CRASH_STATE_FILE="$cut_state" \
  BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" BW_S2_ADMIN_TOKEN="$admin" \
  "$work/s2-crash-probe" start > "$work/bh018-service-after-cut-original.redacted.log" 2>&1
test -s "$cut_state"
sudo python3 - "$db" "$cut_state" <<'PY_BH018_COMMIT_CUT'
import json,sqlite3,sys
with open(sys.argv[2]) as f: original=json.load(f)
with sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True,timeout=3) as c:
    row=c.execute('SELECT state,native_operation_id FROM operations WHERE id=?',
                  (original['operation_id'],)).fetchone()
assert row == ('VERIFYING', original['native_operation_id']), (row,original)
print('BH018_INSTALLED_ORIGINAL_NATIVE_ACK_BEFORE_POSTSYNC_CUT=PASS')
print('BH018_INSTALLED_POSTSYNC_CUT_ORIGINAL_BW_ID='+original['operation_id'])
print('BH018_INSTALLED_POSTSYNC_CUT_ORIGINAL_NATIVE_ID='+original['native_operation_id'])
PY_BH018_COMMIT_CUT
sudo touch "$cut_arm"
attempt=0
until [ "$attempt" -ge 45 ]; do
  if sudo test -s "$cut_witness"; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
sudo grep -Fqx 'ORIGINAL_SQLITE_WAL_FSYNC_CUT_ACTUALLY_TRIGGERED' "$cut_witness"
wait "$service_pid" 2>/dev/null || :
sudo rm -f -- "$cut_arm"
echo 'BH018_INSTALLED_SERVICE_KILLED_AFTER_REAL_EXEC_SQLITE_WAL_SYNC=PASS'
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/boxweave" serve -db "$db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18650 > "$work/bh018-service-after-cut-restarted.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18650/mcp 2>/dev/null)" = 401 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 40 ]; then
  echo "BH018_INSTALLED_SERVICE_CUT_RESTART_HTTP_401=BLOCKED" >&2
  if grep -Fqi 'serve already active for this database' "$work/bh018-service-after-cut-restarted.redacted.log"; then
    echo "BH018_INSTALLED_SERVICE_CUT_RESTART_CLASS=SQLITE_SERVE_LOCK" >&2
  else
    echo "BH018_INSTALLED_SERVICE_CUT_RESTART_CLASS=UNCLASSIFIED" >&2
  fi
  exit 94
fi
echo "BH018_INSTALLED_SERVICE_CUT_RESTART_HTTP_401=PASS"
# Read-only independent database oracle BEFORE the MCP verifier, so a
# failed original replay can be classified without touching the claim.
sudo python3 - "$db" "$cut_state" <<'PY_BH018_POSTSYNC_DIAG'
import json,sqlite3,sys
with open(sys.argv[2]) as f: original=json.load(f)
with sqlite3.connect("file:"+sys.argv[1]+"?mode=ro",uri=True,timeout=3) as c:
    oid=original["operation_id"]
    row=c.execute("SELECT state,native_operation_id FROM operations WHERE id=?",(oid,)).fetchone()
    outputs=c.execute("SELECT COUNT(*) FROM exec_outputs WHERE operation_id=?",(oid,)).fetchone()[0]
    audits=c.execute("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'",(oid,)).fetchone()[0]
assert row is not None
print("BH018_INSTALLED_POSTSYNC_SQLITE_REOPEN_STATE="+row[0])
print("BH018_INSTALLED_POSTSYNC_SQLITE_NATIVE_MATCH="+str(row[1]==original["native_operation_id"]))
print("BH018_INSTALLED_POSTSYNC_SQLITE_OUTPUT_ROWS="+str(outputs))
print("BH018_INSTALLED_POSTSYNC_SQLITE_TERMINAL_AUDIT_ROWS="+str(audits))
PY_BH018_POSTSYNC_DIAG
if ! ( BW_S2_CRASH_VARIANT=bh018-final-wal-cut-after BW_S2_CRASH_STATE_FILE="$cut_state" \
  BW_S2_MCP_URL="http://127.0.0.1:18650/mcp" BW_S2_ADMIN_TOKEN="$admin" \
  "$work/s2-crash-probe" resume ) >"$work/bh018-service-after-cut-resume.redacted.log" 2>&1; then
  grep -E '^BH018_|^S2_NATIVE_|S2_CRASH_ACCEPTANCE_FAILED' "$work/bh018-service-after-cut-resume.redacted.log" | tail -n 15 || :
  if grep -Fq 'tool box_exec returned error' "$work/bh018-service-after-cut-resume.redacted.log"; then
    echo "BH018_INSTALLED_POSTSYNC_MCP_FAIL_CLASS=BOX_EXEC_TOOL_ERROR" >&2
  elif grep -Fq 'tool operation_get returned error' "$work/bh018-service-after-cut-resume.redacted.log"; then
    echo "BH018_INSTALLED_POSTSYNC_MCP_FAIL_CLASS=OPERATION_GET_TOOL_ERROR" >&2
  elif grep -Fq 'tool operation_output_get returned error' "$work/bh018-service-after-cut-resume.redacted.log"; then
    echo "BH018_INSTALLED_POSTSYNC_MCP_FAIL_CLASS=OUTPUT_GET_TOOL_ERROR" >&2
  elif grep -Fq 'S2_CRASH_ACCEPTANCE_FAILED' "$work/bh018-service-after-cut-resume.redacted.log"; then
    echo "BH018_INSTALLED_POSTSYNC_MCP_FAIL_CLASS=ORIGINAL_ASSERTION" >&2
  else
    echo "BH018_INSTALLED_POSTSYNC_MCP_FAIL_CLASS=UNCLASSIFIED" >&2
  fi
  echo "BH018_INSTALLED_SERVICE_CUT_RECOVERY_MCP_VERIFIER=BLOCKED" >&2
  exit 95
fi
grep -Fqx 'S2_NATIVE_COMPLETED_OUTPUT_AFTER_SQLITE_RESTART=PASS' "$work/bh018-service-after-cut-resume.redacted.log"
grep -E '^BH018_INSTALLED_WAL_KILL_ATOMIC_(SUCCEEDED|UNKNOWN)_' "$work/bh018-service-after-cut-resume.redacted.log"
grep -Fxq 'BH018_INSTALLED_WAL_KILL_ATOMIC_SUCCEEDED_OUTPUT_AND_NEXT_OPERATION=PASS' "$work/bh018-service-after-cut-resume.redacted.log"
echo 'BH018_INSTALLED_POSTSYNC_ACK_LOST_FULL_COMMIT_RESTORED=PASS'
# An independent sqlite3 read-only oracle must see exactly one of two
# transactions: all three terminal rows committed OR none (UNKNOWN + lock).
sudo python3 - "$db" "$cut_state" <<'PY_BH018_ATOMIC_ORACLE'
import json,sqlite3,sys
db,checkpoint=sys.argv[1:]
with open(checkpoint) as f: original=json.load(f)
oid,native=original["operation_id"],original["native_operation_id"]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True,timeout=3) as conn:
    op=conn.execute("SELECT state,native_operation_id FROM operations WHERE id=?",(oid,)).fetchone()
    outs=conn.execute("SELECT stdout_path,stderr_path,exit_code FROM exec_outputs WHERE operation_id=?",(oid,)).fetchall()
    audit=conn.execute("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'",(oid,)).fetchone()[0]
assert op and op[1]==native, (op,native)
if op[0]=="SUCCEEDED":
    assert len(outs)==1 and outs[0][2]==0 and audit==1,(outs,audit)
    prefix="/1.0/instances/bw-s2-bh018-after-cut/logs/exec-output/exec_"+native
    assert any(outs[0][:2])
    assert outs[0][0] in ("",prefix+".stdout") and outs[0][1] in ("",prefix+".stderr"),outs
    print("BH018_INSTALLED_WAL_KILL_SQLITE_ALL_TERMINAL_ROWS_ATOMIC=PASS")
elif op[0]=="UNKNOWN":
    assert not outs and audit==0,(outs,audit)
    print("BH018_INSTALLED_WAL_KILL_SQLITE_NO_PARTIAL_TERMINAL_ROWS=PASS")
else:
    raise SystemExit("BH018 WAL SIGKILL recovered invalid/split state: "+op[0])
print("BH018_INSTALLED_WAL_KILL_SQLITE_TERMINAL_OR_UNKNOWN_CROSSCHECK=PASS")
PY_BH018_ATOMIC_ORACLE

attempt=0
count=""
until [ "$attempt" -ge 45 ]; do
  count="$(sudo incus exec bw-s2-bh018-after-cut --project "$project" -- \
    /bin/sh -c 'wc -l < /tmp/bw-s2-bh018-after-cut-marker' 2>/dev/null || :)"
  if [ "$count" = 1 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
done
test "$count" = 1
echo 'BH018_INSTALLED_SERVICE_POSTSYNC_SIGKILL_GUEST_EFFECT_ONCE=PASS'
test "$(sudo incus list --all-projects --format csv -c n)" = bw-s2-bh018-after-cut
sudo incus stop bw-s2-bh018-after-cut --project "$project"
sudo incus delete bw-s2-bh018-after-cut --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo 'BH018_INSTALLED_SERVICE_AFTER_WAL_FSYNC_ACKLOSS_RECOVERY=PASS'

[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=FOREIGN_SQLITE_MATRIX_START"
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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=FOREIGN_SQLITE_FIXTURES_CREATED"
for case in foreign corrupt legacy ambiguous versioned_foreign; do
  [ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=FOREIGN_SQLITE_NEGATIVE_${case}_START"
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
  [ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=FOREIGN_SQLITE_NEGATIVE_${case}_PASS"
done
echo "S2_NATIVE_SQLITE_FOREIGN_CORRUPT_FAIL_CLOSED=PASS"
echo "S2_NATIVE_SQLITE_OLD_MULTIPLE_VERSION_AND_FOREIGN_TABLES_DENIED=PASS"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_NEW_DISPOSABLE_DB_START"

# Deliberate ACK loss after the REAL Incus server accepts a Create. This
# does not retry or adopt any historical task; it uses a new independent
# SQLite ledger, exact disposable original name and the same non-root Incus
# user/Project. The MITM Unix socket cannot access privileged daemon APIs.
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_PRIOR_SERVICE_TERMINATED"
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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_PROXY_SOCKET_READY"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$work/boxweave" token-create -db "$ack_db" > "$work/ack-admin.secret"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_ONE_USE_CREDENTIAL_READY"
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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_ORIGINAL_CLIENT_PROBE_PASS"
sudo grep -Fqx "UPSTREAM_ACCEPTED_ONCE_AND_ACK_SUPPRESSED" "$ack_marker"
test "$(sudo wc -l "$ack_marker" | awk '{print $1}')" = 1
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_PROXY_NATIVE_ACK_SUPPRESSED_ONCE"
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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_INDEPENDENT_NATIVE_UUID_READY"
# The native daemon may expose volatile.uuid before its asynchronous Create
# has become consistently visible to a separate inventory connection.
# Retry READS ONLY: no second Create, no automatic adoption, no UNKNOWN replay.
# The exact all-Projects singleton remains a non-negotiable safety gate.
attempt=0
inventory=""
inventory_rc=0
while [ "$attempt" -lt 45 ]; do
  inventory_rc=0
  inventory="$(sudo incus list --all-projects --format csv -c n 2>/dev/null)" || inventory_rc=$?
  if [ "$inventory_rc" -eq 0 ] && [ "$inventory" = bw-s2-ack-native-110d ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 45 ]; then
  if [ "$inventory_rc" -ne 0 ]; then
    echo "S4_A_ACK_NATIVE_INVENTORY_CLASS=READ_UNAVAILABLE" >&2
  elif [ -z "$inventory" ]; then
    echo "S4_A_ACK_NATIVE_INVENTORY_CLASS=EXPECTED_INSTANCE_NOT_LISTED" >&2
  else
    echo "S4_A_ACK_NATIVE_INVENTORY_CLASS=UNEXPECTED_OR_AMBIGUOUS_INSTANCES" >&2
  fi
  echo "S4_A_NATIVE_STAGE=ACK_LOSS_INDEPENDENT_INVENTORY_BLOCKED" >&2
  exit 95
fi
test "$(sudo incus config get bw-s2-ack-native-110d volatile.uuid --project "$project")" = "$ack_uuid"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_INDEPENDENT_NATIVE_INVENTORY_PASS"
echo "S2_NATIVE_CREATE_LOST_ACK_EFFECT_EXISTS_INDEPENDENTLY=PASS"
# Genuine accepted-but-unbound native Create is an explicit negative Delete
# boundary: even an independently witnessed UUID cannot authorize adoption.
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_UNBOUND_DELETE_REFUSAL_START"
unbound_rc=0
( BW_S2_ACK_NATIVE_UUID="$ack_uuid" BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" unbound_delete ) \
  > "$work/ack-unbound-delete.redacted.log" 2>&1 || unbound_rc=$?
if [ "$unbound_rc" -ne 0 ]; then
  if grep -Fq 'unbound real native UUID was deleted or adopted' "$work/ack-unbound-delete.redacted.log"; then
    echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=REJECTION_MISSING" >&2
  elif grep -Fq 'ACK-loss original lacked safe UNKNOWN lock' "$work/ack-unbound-delete.redacted.log"; then
    echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=ORIGINAL_UNKNOWN_RECEIPT_DRIFT" >&2
  elif grep -Fq 'original ACK loss never reached terminal safe UNKNOWN' "$work/ack-unbound-delete.redacted.log"; then
    echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=ORIGINAL_UNKNOWN_NOT_READY" >&2
  elif grep -Fq 'original accepted Create receipt missing' "$work/ack-unbound-delete.redacted.log"; then
    echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=ORIGINAL_CHECKPOINT_MISSING" >&2
  elif grep -Fq 'only private disposable MCP listener' "$work/ack-unbound-delete.redacted.log"; then
    echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=CLIENT_ENDPOINT_SCOPE_MISMATCH" >&2
  elif grep -Fq 'tool operation_get refused' "$work/ack-unbound-delete.redacted.log"; then
    echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=READ_REFUSED" >&2
  else
    echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=REDACTED_CLIENT_FAILURE" >&2
  fi
  if sudo incus config get bw-s2-ack-native-110d volatile.uuid --project "$project" 2>/dev/null |
      grep -Fqx "$ack_uuid"; then
    echo "S4_A_UNBOUND_DELETE_NATIVE_POSTREAD=ORIGINAL_UUID_RETAINED"
  else
    echo "S4_A_UNBOUND_DELETE_NATIVE_POSTREAD=UNVERIFIED_OR_MISSING"
  fi
  exit 94
fi
grep -Fqx 'S2_P1_REAL_UNBOUND_CREATE_UUID_DELETE_REFUSED_WITHOUT_REDISPATCH=PASS' "$work/ack-unbound-delete.redacted.log"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_UNBOUND_DELETE_REFUSAL_PASS"
if [ "$(sudo wc -l "$ack_marker" | awk '{print $1}')" != 1 ]; then
  echo "S4_A_UNBOUND_DELETE_FAILURE_CLASS=UNEXPECTED_NATIVE_POST_COUNT" >&2
  exit 95
fi
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_UNBOUND_NATIVE_POST_COUNT_PASS"

# Reopen the SAME unknown journal; do not replay the original Create.
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_REOPEN_SAME_UNKNOWN_START"
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
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_POSTRESTART_CLIENT_START"
resume_ack_rc=0
( BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" resume ) \
  > "$work/ack-resume.redacted.log" 2>&1 || resume_ack_rc=$?
if [ "$resume_ack_rc" -ne 0 ]; then
  echo "S4_A_ACK_RESTART_FAILURE_CLASS=REDACTED_ORIGINAL_REPLAY_ASSERTION" >&2
  exit 94
fi
grep -Fqx 'S2_NATIVE_CREATE_ACK_LOSS_RESTART_NO_REPLAY=PASS' "$work/ack-resume.redacted.log"
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_POSTRESTART_ORIGINAL_PASS"
if [ "$(sudo wc -l "$ack_marker" | awk '{print $1}')" != 1 ]; then
  echo "S4_A_ACK_RESTART_NATIVE_MARKER_COUNT=UNEXPECTED" >&2
  exit 95
fi
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_POSTRESTART_POST_COUNT_PASS"
# This is a second independent native connection; the original daemon
# Create may still be finishing when the original HTTP server reopens.
# Only read native state. Never re-issue a Create or adopt UNKNOWN claims.
attempt=0
after_uuid=""
uuid_read_rc=0
while [ "$attempt" -lt 45 ]; do
  uuid_read_rc=0
  after_uuid="$(sudo incus config get bw-s2-ack-native-110d volatile.uuid --project "$project" 2>/dev/null)" || uuid_read_rc=$?
  if [ "$uuid_read_rc" -eq 0 ] && [ "$after_uuid" = "$ack_uuid" ]; then
    break
  fi
  # A DIFFERENT nonempty UUID is a safety failure, not transient lag.
  if [ -n "$after_uuid" ] && [ "$after_uuid" != "$ack_uuid" ]; then
    echo "S4_A_ACK_RESTART_NATIVE_UUID=IDENTITY_DRIFT" >&2
    exit 96
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 45 ]; then
  echo "S4_A_ACK_RESTART_NATIVE_UUID=UNAVAILABLE" >&2
  exit 96
fi
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_POSTRESTART_NATIVE_UUID_VERIFIED"
attempt=0
inventory=""
inventory_rc=0
while [ "$attempt" -lt 45 ]; do
  inventory_rc=0
  inventory="$(sudo incus list --all-projects --format csv -c n 2>/dev/null)" || inventory_rc=$?
  if [ "$inventory_rc" -eq 0 ] && [ "$inventory" = bw-s2-ack-native-110d ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
if [ "$attempt" -ge 45 ]; then
  if [ "$inventory_rc" -ne 0 ]; then
    echo "S4_A_ACK_RESTART_INVENTORY=READ_UNAVAILABLE" >&2
  elif [ -z "$inventory" ]; then
    echo "S4_A_ACK_RESTART_INVENTORY=EXPECTED_INSTANCE_NOT_LISTED" >&2
  else
    echo "S4_A_ACK_RESTART_INVENTORY=UNEXPECTED_OR_AMBIGUOUS_INSTANCES" >&2
  fi
  exit 96
fi
[ "${S4_A_RAW_WITNESS:-}" != true ] || echo "S4_A_NATIVE_STAGE=ACK_LOSS_POSTRESTART_INDEPENDENT_INVENTORY_PASS"
echo "S2_NATIVE_CREATE_ACK_LOSS_POSTRESTART_NO_SECOND_POST=PASS"

# Only this **one-use CI host administrator**, NOT BoxWeave's original
# UNKNOWN recovery API, deletes the exact known test fixture after native
# effects were independently witnessed. Leave the SQLite record UNKNOWN.
# An ACK was intentionally suppressed immediately after HTTP 202; the native
# Create can still be running when its UUID becomes visible. Inspect Incus
# asynchronous operations read-only and wait for the REAL native Create to
# settle before attempting cleanup. Never retry the original MCP mutation.
echo "S4_A_NATIVE_STAGE=ACK_LOSS_NATIVE_SETTLEMENT_START"
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
echo "S4_A_NATIVE_STAGE=ACK_LOSS_NATIVE_SETTLEMENT_PASS"
if [ "$(sudo incus config get bw-s2-ack-native-110d volatile.uuid --project "$project")" != "$ack_uuid" ]; then
  echo "S4_A_ACK_BIND_UUID=IDENTITY_MISMATCH" >&2
  exit 96
fi
echo "S4_A_NATIVE_STAGE=ACK_LOSS_BIND_UUID_PASS"
# Genuine native Create accepted once, its ACK lost, and the original
# controller's ledger is still UNKNOWN. The staged Create recovery probe
# first REFUSES legacy identity-only binding, then reconciles the precise
# native pin/lease/marker in an isolated SQLite copy. The live ledger remains
# immutable and receives no retry/delete.
ack_original_id="$(python3 - "$ack_checkpoint" <<'PY_SCRIPT'
import json,sys
op=json.load(open(sys.argv[1]))["original_id"]
assert len(op)==32
print(op)
PY_SCRIPT
)"
echo "S4_A_NATIVE_STAGE=ACK_LOSS_ORIGINAL_CHECKPOINT_READ"
ack_native_marker="$(sudo incus config get bw-s2-ack-native-110d user.boxweave.create_operation --project "$project")"
if [ "$ack_native_marker" != "$ack_original_id" ]; then
  if [ -z "$ack_native_marker" ]; then
    echo "S4_A_ACK_BIND_MARKER=MISSING" >&2
  else
    echo "S4_A_ACK_BIND_MARKER=ORIGINAL_MISMATCH" >&2
  fi
  exit 96
fi
echo "S4_A_NATIVE_STAGE=ACK_LOSS_BIND_MARKER_PASS"
echo "S2_NATIVE_CREATE_REAL_LOST_ACK_ORIGINAL_MARKER=PASS"
ack_copy="$work/ledger/unknown-create-bind-copy"
if ! sudo install -d -m 0700 -o boxweave -g incus "$ack_copy"; then
  echo "S4_A_ACK_BIND_COPY=DIRECTORY_UNAVAILABLE" >&2
  exit 96
fi
# The original checkpoint belongs to the runner and is deliberately 0600.
# Give the restricted probe ONLY a new exact byte-copy owned by its UID.
if ! sudo install -m 0600 -o boxweave -g incus "$ack_checkpoint" "$ack_copy/ack-original.json"; then
  echo "S4_A_ACK_BIND_COPY=CHECKPOINT_UNAVAILABLE" >&2
  exit 96
fi
if ! sudo runuser -u boxweave -- python3 - "$ack_db" "$ack_copy/ledger.sqlite" <<'PY_SCRIPT'
import sqlite3,sys
src,dst=sys.argv[1:]
with sqlite3.connect(f"file:{src}?mode=ro",uri=True) as original:
    assert original.execute("PRAGMA integrity_check").fetchone()[0]=="ok"
    with sqlite3.connect(dst) as copy:
        original.backup(copy)
PY_SCRIPT
then
  echo "S4_A_ACK_BIND_COPY=SQLITE_BACKUP_UNAVAILABLE" >&2
  exit 96
fi
sudo chmod 0600 "$ack_copy/ledger.sqlite"
echo "S4_A_NATIVE_STAGE=ACK_LOSS_SQLITE_COPY_READY"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/s2-unknown-create-bind-probe" "$ack_copy/ack-original.json" \
  "$ack_copy/ledger.sqlite" "$project" "$ack_uuid"
echo "S4_A_NATIVE_STAGE=ACK_LOSS_SQLITE_COPY_STAGED_RECOVERY_PASS"
# This original database remains truly unbound UNKNOWN; the modern staged
# reconciliation and its durable original receipt happened ONLY in a copy.
if ! sudo runuser -u boxweave -- python3 - "$ack_db" "$ack_original_id" <<'PY_SCRIPT'
import sqlite3,sys
db,op=sys.argv[1:]
with sqlite3.connect(f"file:{db}?mode=ro",uri=True) as c:
    row=c.execute("SELECT state,box_uuid,incarnation,native_operation_id FROM operations WHERE id=?",(op,)).fetchone()
    owned=c.execute("SELECT COUNT(*) FROM owned_boxes").fetchone()[0]
    valid=row==("UNKNOWN","","","") and owned==0
    print("S4_A_ACK_ORIGINAL_SQLITE_UNBOUND="+("PASS" if valid else "MISMATCH"))
    if not valid:
        print("S4_A_ACK_ORIGINAL_SQLITE_ROW_PRESENT="+str(row is not None)+" OWNED_ROWS="+str(owned))
    sys.exit(0 if valid else 1)
PY_SCRIPT
then
  echo "S4_A_ACK_ORIGINAL_SQLITE_UNCHANGED=FAIL" >&2
  exit 96
fi
echo "S2_NATIVE_CREATE_ORIGINAL_SQLITE_UNCHANGED_BY_STAGED_RECOVERY_COPY=PASS"
echo "S4_A_NATIVE_STAGE=ACK_LOSS_ORIGINAL_SQLITE_UNCHANGED_PASS"
echo "S2_NATIVE_CREATE_LOST_ACK_ORIGINAL_NATIVE_FINISHED_BEFORE_CLEANUP=PASS"
sudo incus delete bw-s2-ack-native-110d --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_NATIVE_CREATE_ACK_LOSS_FIXTURE_ADMIN_TEARDOWN=PASS"
echo "S4_A_NATIVE_STAGE=ACK_LOSS_ADMIN_TEARDOWN_PASS"

# Same isolated native/proxy service, a separate new Box and request keys.
# Advance the real native ETag just between S2's GetInstance and
# UpdateInstance, proving the stale update cannot override newer settings.
BW_S2_ACK_STATE_FILE="$ack_checkpoint" \
  BW_S2_MCP_URL="http://127.0.0.1:18652/mcp" \
  BW_S2_ADMIN_TOKEN="$ack_admin" "$work/s2-ack-loss-probe" etag
echo "S4_A_NATIVE_STAGE=ACK_PROXY_ETAG_PASS"
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
echo "S4_A_NATIVE_STAGE=ACK_PROXY_UPDATE_ACK_PASS"
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
echo "S4_A_NATIVE_STAGE=ACK_PROXY_UPDATE_ACK_RESUME_PASS"
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
echo "S4_A_NATIVE_STAGE=ACK_PROXY_POSTREAD_PASS"
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
echo "S4_A_NATIVE_STAGE=ACK_PROXY_POSTREAD_RESUME_PASS"
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

. scripts/s2_native_guest_quota_witness.sh
sudo incus delete --force "$quota_name" --project "$project"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_NATIVE_GUEST_QUOTA_SPACE_RECOVERED_AND_FIXTURE_REMOVED=PASS"

# FINAL S2 native fault witness must pass the public BoxWeave MCP Tool,
# original SQLite journal, 16MiB output-get and exact guest-exit semantics,
# not merely a separate direct native SDK call. Entirely new Box/ledger.
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
mkdir -m 0700 "$work/mcp-quota-ledger"
sudo chown boxweave:incus "$work/mcp-quota-ledger"
mcp_quota_db="$work/mcp-quota-ledger/ledger.sqlite"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/boxweave" token-create -db "$mcp_quota_db" > "$work/mcp-quota-admin.secret"
chmod 0600 "$work/mcp-quota-admin.secret"
mcp_quota_token="$(sed -n 's/^token=//p' "$work/mcp-quota-admin.secret")"
test -n "$mcp_quota_token"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/boxweave" serve -db "$mcp_quota_db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18656 > "$work/mcp-quota-server.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18656/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
mcp_quota_uuid=""
s2_mcp_quota() {
  BW_S2_MCP_QUOTA_URL="http://127.0.0.1:18656/mcp" \
    BW_S2_MCP_QUOTA_TOKEN="$mcp_quota_token" \
    BW_S2_MCP_QUOTA_UUID="$mcp_quota_uuid" \
    BW_S2_MCP_QUOTA_ORIGINAL_ID="${mcp_original:-}" \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    "$work/s2-mcp-quota-probe" "$1"
}
. scripts/s2_mcp_quota_witness.sh

# 7-day retention real Incus/MCP proof on a new disposable original.
retention_op=""
retention_native=""
s2_gc() {
  GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    BW_S2_GC_DB="$mcp_quota_db" BW_S2_GC_TOKEN="$mcp_quota_token" \
    BW_S2_GC_PROJECT="$project" BW_S2_GC_UUID="$mcp_quota_uuid" \
    BW_S2_GC_OPERATION="$retention_op" BW_S2_GC_NATIVE="$retention_native" \
    BW_S2_GC_SOCKET="$socket" "$work/s2-exec-gc-probe" "$1"
}
s2_gc create > "$work/s2-gc-original.txt"
grep -Fqx "S2_GC_PUBLIC_FRESH_BOTH_STREAMS=PASS" "$work/s2-gc-original.txt"
retention_op="$(sed -n 's/^S2_GC_ORIGINAL_ID=//p' "$work/s2-gc-original.txt")"
retention_native="$(sed -n 's/^S2_GC_NATIVE_ID=//p' "$work/s2-gc-original.txt")"
echo "$retention_op" | grep -Eq '^[0-9a-f]{32}$'
echo "$retention_native" | grep -Eq '^[0-9a-f-]{36}$'
for stream in stdout stderr; do
  sudo test -f "$mcp_spool/exec_$retention_native.$stream"
done
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
s2_gc_nonroot() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    BW_S2_GC_DB="$mcp_quota_db" BW_S2_GC_TOKEN="$mcp_quota_token" \
    BW_S2_GC_PROJECT="$project" BW_S2_GC_UUID="$mcp_quota_uuid" \
    BW_S2_GC_OPERATION="$retention_op" BW_S2_GC_NATIVE="$retention_native" \
    BW_S2_GC_SOCKET="$socket" "$work/s2-exec-gc-probe" "$1"
}
s2_gc_nonroot age > "$work/s2-gc-age.txt"
grep -Fqx "S2_GC_SEVEN_DAYS_NOT_PREMATURELY_EXPIRED=PASS" "$work/s2-gc-age.txt"
s2_gc_nonroot sweep > "$work/s2-gc-sweep.txt"
grep -Fqx "S2_GC_NATIVE_7D_TWO_STREAMS_DELETE_AUDIT_404_NO_REPLAY=PASS" "$work/s2-gc-sweep.txt"
for stream in stdout stderr; do
  sudo test ! -f "$mcp_spool/exec_$retention_native.$stream"
done
echo "S2_GC_INDEPENDENT_PHYSICAL_TWO_NATIVE_LOGS_REMOVED=PASS"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/boxweave" serve -db "$mcp_quota_db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18656 > "$work/mcp-retention-restart.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18656/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
s2_gc check > "$work/s2-gc-public-expired.txt"
grep -Fqx "S2_GC_PUBLIC_ORIGINAL_TWO_STREAMS_EXPIRED=PASS" "$work/s2-gc-public-expired.txt"
echo "S2_GC_APPROVED_SEVEN_DAY_COMPLETE_NATIVE_MCP_CYCLE=PASS"


# New independent original on this exact disposable owned Box. Incus really
# deletes original stdout, but the test proxy suppresses only the HTTP ACK.
# Never replay a native DELETE or silently delete the surviving stderr.
s2_gc create-loss > "$work/s2-gc-loss-original.txt"
grep -Fqx "S2_GC_PUBLIC_FRESH_BOTH_STREAMS=PASS" "$work/s2-gc-loss-original.txt"
retention_op="$(sed -n 's/^S2_GC_ORIGINAL_ID=//p' "$work/s2-gc-loss-original.txt")"
retention_native="$(sed -n 's/^S2_GC_NATIVE_ID=//p' "$work/s2-gc-loss-original.txt")"
echo "$retention_op" | grep -Eq '^[0-9a-f]{32}$'
echo "$retention_native" | grep -Eq '^[0-9a-f-]{36}$'
for stream in stdout stderr; do
  sudo test -f "$mcp_spool/exec_$retention_native.$stream"
done
sudo pkill -TERM -u boxweave -f "^$work/boxweave serve" || :
wait "$service_pid" 2>/dev/null || :
s2_gc_nonroot age > "$work/s2-gc-loss-age.txt"
grep -Fqx "S2_GC_SEVEN_DAYS_NOT_PREMATURELY_EXPIRED=PASS" "$work/s2-gc-loss-age.txt"
gc_socket="$work/ledger/gc-delete-proxy.sock"
gc_marker="$work/ledger/gc-native-delete.txt"
sudo runuser -u boxweave -- env GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  "$work/s2-gc-delete-proxy" -listen "$gc_socket" -upstream "$socket" \
  -marker "$gc_marker" -project "$project" -native-id "$retention_native" \
  > "$work/gc-delete-proxy.redacted.log" 2>&1 &
gc_proxy_pid=$!
attempt=0
until sudo test -S "$gc_socket" || [ "$attempt" -ge 30 ]; do
  attempt=$((attempt+1))
  sleep 1
done
sudo test -S "$gc_socket"
real_socket="$socket"
socket="$gc_socket"
s2_gc_nonroot lossy-sweep > "$work/s2-gc-loss-sweep.txt"
socket="$real_socket"
grep -Fqx "S2_GC_REAL_NATIVE_DELETE_ACK_LOSS_REVIEW_REQUIRED_SIBLING_LOCKED=PASS" "$work/s2-gc-loss-sweep.txt"
sudo grep -Fqx "GC_EXACT_NATIVE_STDOUT_DELETE_ACCEPTED_ACK_SUPPRESSED" "$gc_marker"
sudo test ! -e "$gc_marker.unexpected"
sudo test ! -f "$mcp_spool/exec_$retention_native.stdout"
sudo test -f "$mcp_spool/exec_$retention_native.stderr"
echo "S2_GC_REAL_NATIVE_DELETE_ACK_LOST_ORIGINAL_NOT_REPLAYED=PASS"
kill "$gc_proxy_pid" 2>/dev/null || :
wait "$gc_proxy_pid" 2>/dev/null || :
# Direct-ledger fixture teardown, NOT proof of installed root CLI recovery.
sh scripts/s2_exec_gc_installed_review.sh "$work" "$mcp_quota_db" "$project" "$socket" \
  "$retention_op" "$retention_native" "$mcp_quota_uuid"
s2_gc_nonroot release-test-fixture > "$work/s2-gc-loss-reviewed.txt"
grep -Fqx "S2_GC_DISPOSABLE_LEDGER_REVIEW_RELEASE_NO_NATIVE_RETRY=PASS" "$work/s2-gc-loss-reviewed.txt"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  "$work/boxweave" serve -db "$mcp_quota_db" -incus-socket "$socket" -project "$project" \
  -listen 127.0.0.1:18656 > "$work/mcp-gc-loss-restart.redacted.log" 2>&1 &
service_pid=$!
attempt=0
until [ "$attempt" -ge 40 ]; do
  if [ "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18656/mcp 2>/dev/null)" = 401 ]; then
    break
  fi
  attempt=$((attempt+1))
  sleep 1
done
test "$attempt" -lt 40
s2_gc loss-check > "$work/s2-gc-loss-public.txt"
grep -Fqx "S2_GC_PUBLIC_LOST_ACK_TTL_ELAPSED_NOT_PHYSICAL_EXPIRY=PASS" "$work/s2-gc-loss-public.txt"
echo "S2_GC_LOST_NATIVE_DELETE_ACK_SAME_SQLITE_RESTART_NO_FALSE_EXPIRY=PASS"

s2_mcp_quota cleanup > "$work/mcp-quota-cleanup.txt"
grep -Fqx "S2_MCP_QUOTA_EXACT_BOX_DELETED=PASS" "$work/mcp-quota-cleanup.txt"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_MCP_QUOTA_FINAL_NATIVE_INVENTORY_EMPTY=PASS"

# Disposable test-only LD_PRELOAD wrapper kills the exact SQLite WAL fsync
# (before or just after the real syscall) during NativeAck.Commit on a
# wholly separate, single-actor SQLite database. No native dispatch here:
# the actual Incus post-acceptance ACK-loss proof is a different real gate.
command -v cc >/dev/null
cc -shared -fPIC -O2 -std=gnu11 -Wall -Wextra -o "$work/s2-fsync-cut.so" \
  scripts/s2_sqlite_fsync_cut.c -ldl
chmod 0755 "$work/s2-fsync-cut.so"
for phase in before after; do
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  fsync_dir="$work/fsync-$phase/fsync-ledger"
  mkdir -p "$fsync_dir"
  chmod 0700 "$fsync_dir"
  sudo chown boxweave:incus "$fsync_dir"
  fsync_db="$fsync_dir/ledger.sqlite"
  fsync_arm="$fsync_dir/arm"
  fsync_witness="$fsync_dir/sync-triggered"
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    "$work/boxweave" token-create -db "$fsync_db" \
    > "$work/fsync-$phase-admin.secret"
  chmod 0600 "$work/fsync-$phase-admin.secret"
  fsync_token="$(sed -n 's/^token=//p' "$work/fsync-$phase-admin.secret")"
  test -n "$fsync_token"
  s2_fsync() {
    sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
      GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
      HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
      BW_S2_FSYNC_DB="$fsync_db" BW_S2_FSYNC_TOKEN="$fsync_token" \
      BW_S2_FSYNC_PROJECT="$project" BW_S2_FSYNC_ARM_FILE="$fsync_arm" \
      "$work/s2-sqlite-fsync-probe" "$1"
  }
  s2_fsync prepare > "$work/fsync-$phase-prepare.txt"
  grep -Eq '^S2_FSYNC_ORIGINAL_OPERATION_ID=[0-9a-f]{32}$' "$work/fsync-$phase-prepare.txt"
  test ! -e "$fsync_arm"
  test ! -e "$fsync_witness"
  set +e
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    BW_S2_FSYNC_DB="$fsync_db" BW_S2_FSYNC_DB_WAL="$fsync_db-wal" \
    BW_S2_FSYNC_TOKEN="$fsync_token" BW_S2_FSYNC_PROJECT="$project" \
    BW_S2_FSYNC_ARM_FILE="$fsync_arm" BW_S2_FSYNC_WITNESS_FILE="$fsync_witness" \
    BW_S2_FSYNC_CUT_MODE="$phase" LD_PRELOAD="$work/s2-fsync-cut.so" \
    "$work/s2-sqlite-fsync-probe" ack > "$work/fsync-$phase-kill.redacted.txt" 2>&1
  fsync_exit=$?
  set -e
  test "$fsync_exit" -ne 0
  sudo grep -Fqx 'ORIGINAL_SQLITE_WAL_FSYNC_CUT_ACTUALLY_TRIGGERED' "$fsync_witness"
  s2_fsync verify > "$work/fsync-$phase-verify.txt"
  grep -Fqx "S2_SQLITE_WAL_FSYNC_REAL_KILL_UNKNOWN_NO_REPLAY=PASS" \
    "$work/fsync-$phase-verify.txt"
  if [ "$phase" = before ]; then
    echo "S2_SQLITE_WAL_FSYNC_BEFORE_REAL_KILL_UNKNOWN_NO_REPLAY=PASS"
  else
    echo "S2_SQLITE_WAL_FSYNC_AFTER_REAL_KILL_UNKNOWN_NO_REPLAY=PASS"
  fi
done

sh scripts/s2_sqlite_enospc_lab.sh "$work" "$project" "$work/boxweave" "$work/s2-sqlite-fsync-probe"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S4_A_NATIVE_STAGE=S2_P0_NATIVE_SQLITE_FAULT_START"
sh scripts/s2_p0_native_sqlite_fault.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P0_NATIVE_SQLITE_FAULT_PASS"
echo "S4_A_NATIVE_STAGE=S2_P1_STOP_DELETE_FAULT_START"
sh scripts/s2_p1_stop_delete_fault.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P1_STOP_DELETE_FAULT_PASS"
echo "S4_A_NATIVE_STAGE=S2_P1_TOKEN_INFLIGHT_START"
sh scripts/s2_p1_token_inflight.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P1_TOKEN_INFLIGHT_PASS"
echo "S4_A_NATIVE_STAGE=S2_P1_OPERATION_LIST_BOUNDARY_START"
sh scripts/s2_p1_operation_list_boundary.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P1_OPERATION_LIST_BOUNDARY_PASS"
# This separate test-only Go overlay adds ONE native-verified, pre-atomic
# Create pause in an independently compiled disposable binary, without altering
# the original release candidate used by the complete native suite. Unlike
# prior GDB source-line experiments, its exact unbound precommit boundary is
# verified by independent SQLite + Incus before 12 real MCP Deletes.
# A failure is a real required CI failure, not an invented S2 PASS.
echo "S4_A_NATIVE_STAGE=OVERLAP_START"
sh scripts/s2_p1_native_create_delete_overlap.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=OVERLAP_PASS"
echo "S4_A_NATIVE_STAGE=S2_P1_TWO_CONTROLLERS_START"
sh scripts/s2_p1_two_controllers.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P1_TWO_CONTROLLERS_PASS"
echo "S4_A_NATIVE_STAGE=S2_P1_PRIVATE_READ_INFLIGHT_START"
sh scripts/s2_p1_private_read_inflight.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P1_PRIVATE_READ_INFLIGHT_PASS"
echo "S4_A_NATIVE_STAGE=ATOMIC_CREATE_ROLLBACK_START"
sh scripts/s2_p1_bound_create_receipt_fault.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=ATOMIC_CREATE_ROLLBACK_PASS"
echo "S4_A_NATIVE_STAGE=S2_P2_MALFORMED_EXEC_FAULT_START"
sh scripts/s2_p2_malformed_exec_fault.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P2_MALFORMED_EXEC_FAULT_PASS"
echo "S4_A_NATIVE_STAGE=S2_P2_ALTERNATE_POOL_START"
sh scripts/s2_p2_alternate_pool.sh "$work" "$project" "$socket"
echo "S4_A_NATIVE_STAGE=S2_P2_ALTERNATE_POOL_PASS"
echo "S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS"
# S2's historical isolated RFC2544 198.18.0.2 simulated WAN is
# intentionally run against its ORIGINAL legacy S2 native fixture. The
# S2.5 policy explicitly denies RFC2544 private-style external targets
# and is separately accepted through the exact-source nested installed
# v1 MCP + real IPv4/public and routed simulated IPv6 TLS packet gate.
# No native Guest can be left running while changing these ACL rules.
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo S25_S2_LEGACY_SIMWAN_SEPARATE_FROM_HARDENED_ACL=PASS
# S2.5's *separate* native prerequisite review rejects an empty ACL:
# Internet=true otherwise permits routed access to OTHER native Projects.
# This is a disposable, empty single-use project. Do not backport these
# global-fixture commands into shared/physical-host native acceptance.
if [ "$(printenv S25_NATIVE_PREREQ_PROBE 2>/dev/null || true)" = true ]; then
  owned_net="incusbr-$(id -u boxweave)"
  v4="$(sudo incus network get "$owned_net" ipv4.address | cut -d/ -f1)"
  v6="$(sudo incus network get "$owned_net" ipv6.address | cut -d/ -f1)"
  test -n "$v4"
  # Incus rules are prioritized reject BEFORE allow, regardless of list
  # order. Carve exactly one native gateway /32 and /128 from rejected
  # private ranges; an "allow gateway" overlapping a reject is ineffective.
  python3 - "$work/s25-native-private-cidrs.txt" "$v4" "$v6" <<'S25_GATEWAY_CARVE_OUT'
import ipaddress,sys
blocked4="0.0.0.0/8,10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.0.2.0/24,192.168.0.0/16,198.18.0.0/15,198.51.100.0/24,203.0.113.0/24,224.0.0.0/4,240.0.0.0/4"
blocked6="fc00::/7,::1/128"
def carve(source, gateway):
    out=[]
    target=ipaddress.ip_address(gateway) if gateway not in ("","none") else None
    for raw in source.split(","):
        net=ipaddress.ip_network(raw)
        if target is None or target not in net:
            out.append(net)
        else:
            out.extend(net.address_exclude(ipaddress.ip_network(str(target)+("/32" if target.version==4 else "/128"))))
    for n in out:
        assert target not in n if target is not None else True
    return ",".join(str(x) for x in out)
open(sys.argv[1],"w").write(carve(blocked4,sys.argv[2])+"\n"+carve(blocked6,sys.argv[3])+"\n")
S25_GATEWAY_CARVE_OUT
  denied_v4="$(sed -n '1p' "$work/s25-native-private-cidrs.txt")"
  denied_v6="$(sed -n '2p' "$work/s25-native-private-cidrs.txt")"
  test -n "$denied_v4" && test -n "$denied_v6"
  sudo incus network acl rule add bw-s2-native-acl egress \
    action=reject state=enabled destination="$denied_v4" \
    --description "Deny routed private and reserved IPv4 networks except exact bridge gateway"
  sudo incus network acl rule add bw-s2-native-acl egress \
    action=reject state=enabled destination="$denied_v6" \
    --description "Deny routed ULA IPv6 except gateway, preserve link-local RA"
  # Native readback of the persisted disjoint private rules.
  sudo incus query /1.0/network-acls/bw-s2-native-acl > "$work/s25-native-egress-acl.json"
  python3 - "$work/s25-native-egress-acl.json" "$work/s25-native-private-cidrs.txt" "$v4" "$v6" <<'S25_NATIVE_ACL_V'
import ipaddress,json,sys
a=json.load(open(sys.argv[1]))
rules=a["egress"]
approved=open(sys.argv[2]).read().splitlines()
assert len(rules)==2,rules
assert [x["action"] for x in rules]==["reject","reject"],rules
assert [x["state"] for x in rules]==["enabled","enabled"],rules
assert [x["destination"] for x in rules]==approved,rules
for i,gw in enumerate(sys.argv[3:]):
    if gw in ("","none"):
        continue
    address=ipaddress.ip_address(gw)
    assert all(address not in ipaddress.ip_network(prefix) for prefix in approved[i].split(","))
S25_NATIVE_ACL_V
  echo S25_NATIVE_EXCLUSIVE_ACL_PRIVATE_ROUTED_EGRESS_DENY=PASS
fi
# S2.5 exact hardened-ACL native preflight on the now-EMPTY one-use
# Project. This readback does not count as a live Guest packet witness:
# the separately installed exact-source S2.5 L0/L1/L2 LAB must verify it.
# The older S2 RFC2544 benchmark WAN simulator runs BEFORE hardening, so
# its positive control is not mislabeled as S2.5 private-route evidence.
if [ "${S25_NATIVE_PREREQ_PROBE:-}" = true ]; then
  # Independent native-authority evidence for a failed prerequisite:
  # JSON-only, no tokens, no SQLite, no secret files. This does not
  # substitute for or bypass the subsequent product preflight.
  sudo incus query /1.0/network-acls/bw-s2-native-acl > "$work/s25-native-final-acl.json"
  sudo incus query "/1.0/networks/$owned_net" > "$work/s25-native-final-network.json"
  python3 - "$work/s25-native-final-acl.json" "$work/s25-native-final-network.json" \
    "$work/s25-native-private-cidrs.txt" "$project" <<'S25_FINAL_ROOT_NATIVE_CROSSCHECK'
import hashlib,ipaddress,json,sys,urllib.parse
acl,net=(json.load(open(x)) for x in sys.argv[1:3])
cidrs=open(sys.argv[3]).read().splitlines()
owned=sys.argv[4]
refs=acl.get("used_by",[])
def reference_allowed(ref):
    u=urllib.parse.urlparse(ref)
    return (u.path.startswith("/1.0/profiles/") or u.path.startswith("/1.0/instances/")) and \
        urllib.parse.parse_qs(u.query)=={"project":[owned]} and not u.fragment
ref_ok=bool(refs) and any(r==f"/1.0/profiles/default?project={owned}" for r in refs) and all(reference_allowed(r) for r in refs)
egress=acl.get("egress",[])
cidr_equal=len(egress)==2 and [r.get("destination") for r in egress]==cidrs
expected_descriptions=["Deny routed private and reserved IPv4 networks except exact bridge gateway",
                       "Deny routed ULA IPv6 except gateway, preserve link-local RA"]
rule_shapes= len(egress)==2 and all(
    r.get("action")=="reject" and r.get("state")=="enabled" and
    r.get("description")==expected_descriptions[i] and
    all(r.get(k,"")=="" for k in ("source","protocol","source_port","destination_port","icmp_type","icmp_code"))
    for i,r in enumerate(egress))
print("S25_NATIVE_FINAL_NATIVE_ACL_REFS="+json.dumps(refs))
print("S25_NATIVE_FINAL_NATIVE_ACL_EXCLUSIVE="+str(ref_ok))
print("S25_NATIVE_FINAL_NATIVE_ACL_CIDRS_MATCH_PYTHON="+str(cidr_equal))
print("S25_NATIVE_FINAL_NATIVE_ACL_RULE_SHAPES="+str(rule_shapes))
print("S25_NATIVE_FINAL_ROOT_BRIDGE_ADDRESSES="+json.dumps({
    "ipv4.address":net.get("config",{}).get("ipv4.address"),
    "ipv6.address":net.get("config",{}).get("ipv6.address")}))
print("S25_NATIVE_FINAL_ACL_RULE_CIDR_SHA256="+json.dumps(
    [hashlib.sha256(r.get("destination","").encode()).hexdigest() for r in egress]))
S25_FINAL_ROOT_NATIVE_CROSSCHECK
  sudo "$work/boxweave" native-prereq-plan -project "$project" > "$work/s25-native-prereq-plan.json"
  python3 - "$work/s25-native-prereq-plan.json" <<'PY_S25_PREREQ'
import json,sys
data=json.load(open(sys.argv[1],encoding="utf-8"))
for check in data["checks"]:
    print("S25_NATIVE_PREREQ_%s=%s" % (check["id"].upper(), "PASS" if check["ok"] else "BLOCKED"))
if not data["config_ready"] or not data["packet_witness_required"]:
    raise SystemExit("S25 native operator prerequisite review is BLOCKED")
PY_S25_PREREQ
  s25_sha="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["review_sha256"])' "$work/s25-native-prereq-plan.json")"
  if ! sudo "$work/boxweave" native-prereq-verify -project "$project" \
    -native-prereq-sha256 "$s25_sha" -approve-local > "$work/s25-native-prereq-verified.json"; then
    python3 - "$work/s25-native-prereq-plan.json" "$work/s25-native-prereq-verified.json" <<'PY_S25_DIAGNOSTIC'
import json,sys
before,after=(json.load(open(x,encoding="utf-8")) for x in sys.argv[1:])
print("S25_NATIVE_PREREQ_PLAN_DIGEST="+before["review_sha256"])
print("S25_NATIVE_PREREQ_VERIFY_DIGEST="+after["review_sha256"])
print("S25_NATIVE_PREREQ_PLAN_STATE_DIGEST="+before["native_state_sha256"])
print("S25_NATIVE_PREREQ_VERIFY_STATE_DIGEST="+after["native_state_sha256"])
print("S25_NATIVE_PREREQ_CHANGED_FIELDS="+str(sorted(
    k for k in set(before)|set(after) if before.get(k)!=after.get(k))))
PY_S25_DIAGNOSTIC
    echo "S25_NATIVE_PREREQ_FAIL: refused changed review digest (no native write)" >&2
    exit 93
  fi
  # On the single-use lab, confirm a changed *real* Project config
  # invalidates the exact reviewed SHA. Always restore after the denial.
  sudo incus project set "$project" user.s25-approval-drift=invalidated
  if sudo "$work/boxweave" native-prereq-verify -project "$project" \
    -native-prereq-sha256 "$s25_sha" -approve-local > /dev/null 2>&1; then
    echo "S25_NATIVE_PREREQ_FAIL: native Project drift accepted" >&2
    exit 94
  fi
  sudo incus project unset "$project" user.s25-approval-drift
  echo "S25_NATIVE_PREREQ_NATIVE_PROJECT_DRIFT_DENIED=PASS"
  if sudo "$work/boxweave" native-prereq-verify -project "$project" \
    -native-prereq-sha256 0000000000000000000000000000000000000000000000000000000000000000 -approve-local \
    > /dev/null 2>&1; then
    echo "S25_NATIVE_PREREQ_FAIL: wrong plan digest accepted" >&2
    exit 92
  fi
  echo "S25_NATIVE_PREREQ_CONFIG_AND_DIGEST=PASS"
  echo "S25_NATIVE_PREREQ_PACKET_WITNESS=REQUIRED_SEPARATE"
fi
