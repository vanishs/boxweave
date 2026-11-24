#!/bin/sh
# Exact-head clean-VM N2 installer lifecycle. Never run on a shared Host.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
sudo -n true
grep -q '^ID=ubuntu$' /etc/os-release
grep -Eq '^VERSION_ID="?24\.04"?$' /etc/os-release
test ! -e /var/lib/incus
test ! -e /var/lib/boxweave
test ! -e /var/lib/boxweave-native-prereq-v1.json
test ! -e /usr/local/bin/boxweave
test ! -e /etc/systemd/system/boxweave.service
test ! -e /etc/systemd/system/boxweave-ipv4-retirement.service
! id boxweave >/dev/null 2>&1
work="$(mktemp -d -p /tmp bw-n2-installed.XXXXXX)"
chmod 0755 "$work"
commit="$(git rev-parse HEAD)"
test "$commit" = "${N2_EXACT_HEAD:?}"
echo "N2_INSTALL_SOURCE=$commit"
for version in a b; do
  CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags "-X main.version=1.0.0-dev.n2-$version -X main.commit=$commit -buildid=" -o "$work/candidate-$version" ./cmd/boxweave
done
first="$(sha256sum "$work/candidate-a" | cut -d' ' -f1)"
next="$(sha256sum "$work/candidate-b" | cut -d' ' -f1)"
test "$first" != "$next"
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl gpg dnsmasq-base conntrack nftables iproute2 >/dev/null
curl -fsSL --max-time 35 https://pkgs.zabbly.com/key.asc -o "$work/key.asc"
test "$(gpg --show-keys --with-colons "$work/key.asc" | awk -F: '$1=="fpr"{print $10;exit}')" = 4EFC590696CB15B87C73A3AD82CC8797C838DCFD
sudo install -m 0644 "$work/key.asc" /etc/apt/keyrings/bw-n2-install.asc
echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/bw-n2-install.asc] https://pkgs.zabbly.com/incus/stable noble main' | sudo tee /etc/apt/sources.list.d/bw-n2-install.list >/dev/null
sudo apt-get update -qq
pkg="$(apt-cache madison incus | awk '/pkgs.zabbly.com/ && $3 ~ /:7[.]6[-.]/ {print $3;exit}')"
test -n "$pkg"
sudo "$work/candidate-a" native-plan -incus-package-version "$pkg" > "$work/bootstrap-plan.json"
digest="$(python3 -c 'import json,sys;x=json.load(open(sys.argv[1]));assert x["mode"]=="fresh-install-proposed";print(x["plan_sha256"])' "$work/bootstrap-plan.json")"
sudo "$work/candidate-a" native-bootstrap -incus-package-version "$pkg" -native-plan-sha256 "$digest" -approve-native-bootstrap > "$work/bootstrap-apply.json"
python3 - "$work/bootstrap-apply.json" <<'PY'
import ipaddress,json,sys
x=json.load(open(sys.argv[1]))
assert x["mode"]=="new-daemon-ready",x
p=ipaddress.ip_interface(x["ipv4_bridge_prefix"])
assert p.network.prefixlen<=21 and x["ipv4_address_pool"]>=2045,x
print("N2_CLEAN_BOOTSTRAP_ADDRESS_POOL=PASS",p,x["ipv4_address_pool"],flush=True)
PY
uid="$(id -u boxweave)"
project="user-$uid"
bridge="incusbr-$uid"
test "$(sudo incus project get "$project" restricted)" = true
test -z "$(sudo incus list --all-projects -c n -f csv)"
if test "${PR178_SCOPED_N2:-0}" = 1; then
  python3 scripts/n2_installed_lifecycle.py tty-install-scoped "$project" "$work/candidate-a" "$first"
  installed=/usr/local/bin/boxweave
  sudo systemctl is-active --quiet boxweave.service boxweave-scoped-guard.service boxweave-scoped-lifecycle.service
  sudo "$installed" install-status -project "$project" > "$work/scoped-active.json"
  python3 - "$work/scoped-active.json" <<'CHECK'
import json,sys
x=json.load(open(sys.argv[1]))
assert x["scoped_peer_mode"]=="SCOPED_PEER_FRESH_V2" and x["service_active"] and not x["restart_required"],x
CHECK
  echo PR178_N2_INSTALL=PASS
  if sudo "$installed" upgrade -project "$project" -source-binary "$work/candidate-b" -expect-sha256 "$next" -approve-local; then
    echo "scoped upgrade unexpectedly accepted" >&2
    exit 1
  fi
  test ! -e /var/lib/boxweave-installer-backup
  echo PR178_N2_FRESH_ONLY=PASS
else
  python3 scripts/n2_installed_lifecycle.py tty-install "$project" "$work/candidate-a" "$first"
fi
installed=/usr/local/bin/boxweave
if test "${PR178_SCOPED_N2:-0}" != 1; then
sudo systemctl is-active --quiet boxweave.service
sudo systemctl is-active --quiet boxweave-ipv4-retirement.service
test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$first"
sudo "$installed" install-status -project "$project" > "$work/active.json"
python3 - "$work/active.json" <<'PY'
import json,sys
x=json.load(open(sys.argv[1]))
assert x["service_active"] and not x["restart_required"] and len(x["retirement_unit_sha256"])==64
print("N2_INSTALLED_HELPER_ACTIVATION=PASS",flush=True)
PY
fi
sudo incus image copy images:ubuntu/24.04 local: --target-project "$project" --alias boxweave-ubuntu-24.04 >/dev/null
(umask 077; sudo runuser -u boxweave -- "$installed" token-create -project "$project" -ttl 30m > "$work/actor.receipt")
if test "${PR178_SCOPED_N2:-0}" = 1; then
  sudo python3 scripts/pr178_n2_scoped_lifecycle.py "$project" "$work/actor.receipt"
  echo PR178_N2_CLEAN_INSTALLED_LIFECYCLE_PASS
  exit 0
fi
sudo python3 scripts/n2_installed_lifecycle.py mcp-retry "$project" "$work/actor.receipt"
test -z "$(sudo incus list --all-projects -c n -f csv)"
unit_hash="$(sudo sha256sum /etc/systemd/system/boxweave-ipv4-retirement.service | cut -d' ' -f1)"
sudo "$installed" upgrade -project "$project" -source-binary "$work/candidate-b" -expect-sha256 "$next" -approve-local > "$work/upgrade.json"
test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$next"
sudo systemctl is-active --quiet boxweave-ipv4-retirement.service
test "$(sudo sha256sum /etc/systemd/system/boxweave-ipv4-retirement.service | cut -d' ' -f1)" = "$unit_hash"
sudo "$installed" rollback -project "$project" -approve-local > "$work/rollback.json"
test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$first"
sudo systemctl is-active --quiet boxweave.service
sudo systemctl is-active --quiet boxweave-ipv4-retirement.service
test "$(sudo sha256sum /etc/systemd/system/boxweave-ipv4-retirement.service | cut -d' ' -f1)" = "$unit_hash"
sudo "$installed" install-commit -project "$project" -approve-local >/dev/null
test ! -e /var/lib/boxweave-installer-backup
sudo python3 scripts/n2_installed_lifecycle.py mcp-read "$project" "$work/actor.receipt"
echo N2_SCHEMA6_UPGRADE_ROLLBACK=PASS
python3 scripts/n2_installed_lifecycle.py tty-uninstall "$project"
test ! -e /var/lib/boxweave
test ! -e /usr/local/bin/boxweave
test ! -e /etc/systemd/system/boxweave.service
test ! -e /etc/systemd/system/boxweave-ipv4-retirement.service
test ! -e /run/boxweave-ipv4-retirement.sock
test ! -e /var/lib/boxweave-native-prereq-v1.json
sudo systemctl is-active --quiet incus.service
test -z "$(sudo incus list --all-projects -c n -f csv)"
sudo python3 - <<'PY'
import json,pathlib
x=json.load(open("/var/lib/boxweave-uninstall-v1-audit.json"))
assert x["status"]=="COMPLETE",x["status"]
result=x["result"]
assert result["outcome"]=="COMPLETE"
archive=pathlib.Path(result["retirement_archive"])
assert (archive/"boxweave-ipv4-retirement.service").is_file()
print("N2_FULL_UNINSTALL_PRESERVES_ARCHIVE=PASS",flush=True)
PY
rm -f "$work/actor.receipt"
echo N2_CLEAN_INSTALLED_LIFECYCLE_PASS
