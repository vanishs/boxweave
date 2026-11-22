#!/bin/sh
# Narrow operation_list boundary diagnostic on one-use GitHub-hosted VM.
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
echo "LIST_BOUNDARY_CANDIDATE_SOURCE_SHA=$(git rev-parse HEAD)"
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


# This read-only fixture never creates a native instance or grants new access.
sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
uid="$(id -u boxweave)"
project="user-$uid"
socket=/var/lib/incus/unix.socket.user
actor() {
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
actor incus project list --format csv >/dev/null
test "$(sudo incus project get "$project" restricted)" = true
test "$(actor id -nG)" = incus
if actor incus project show default >/dev/null 2>&1; then exit 90; fi
CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=1.0.24-s2-native-ci -X main.commit=$(git rev-parse HEAD)" -o "$work/boxweave" ./cmd/boxweave
CGO_ENABLED=1 go build -trimpath -o "$work/s2-p1-list-debugger-probe" ./scripts/s2_p1_operation_list_boundary_probe.go
sha256sum "$work/boxweave"
# Two fresh ledgers and independent debugger runs; no replay of a failing case.
sh scripts/s2_p1_operation_list_boundary.sh "$work" "$project" "$socket"
second="$(mktemp -d -p /tmp bw-v1-s2-ghci.XXXXXX)"
chmod 0711 "$second"
cp "$work/boxweave" "$work/s2-p1-list-debugger-probe" "$second/"
cmp "$work/boxweave" "$second/boxweave"
sh scripts/s2_p1_operation_list_boundary.sh "$second" "$project" "$socket"
echo LIST_BOUNDARY_TWO_INDEPENDENT_RUNS_PASS
