#!/bin/sh
# One-use GitHub-hosted RC10 residual CLI and partial Peer/Forward fault acceptance. No shared Host effects.
set -eu
umask 077
test "${GITHUB_REPOSITORY:-}" = vanishs/boxweave
test "${GITHUB_REF:-}" = refs/heads/docs/s5b-rc10-closeout-20261121
test "$(git rev-parse HEAD)" = "$GITHUB_SHA"
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
git diff --exit-code eb021d3985c787c517cd7aee3efaf433d38f0393 -- cmd internal go.mod go.sum install.sh uninstall.sh
test "$GITHUB_ACTIONS" = true
test "$RUNNER_ENVIRONMENT" = github-hosted
test "$(id -u)" != 0
sudo -n true
grep -q '^ID=ubuntu$' /etc/os-release
grep -Eq '^VERSION_ID="?24\.04"?$' /etc/os-release
if [ -e /var/lib/incus ] || id boxweave >/dev/null 2>&1; then
 echo S25_LAB_BLOCKED_FOREIGN_NATIVE_STATE >&2; exit 82
fi
work="$(mktemp -d -p /tmp bw-v1-s25-lab.XXXXXX)"
chmod 0711 "$work"
mkdir -m 0700 "$work/staging" "$work/private"
proof="$RUNNER_TEMP/s25-evidence"
mkdir -p "$proof"
echo "S25_LAB_SOURCE_SHA=$(git rev-parse HEAD)"
echo S25_LAB_DISPOSABLE_HOST=PASS

sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl gpg python3 jq sqlite3 nftables conntrack >/dev/null
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL --max-time 35 https://pkgs.zabbly.com/key.asc -o "$work/staging/zabbly.asc"
fp="$(gpg --show-keys --with-colons "$work/staging/zabbly.asc" | awk -F: '$1=="fpr"{print $10;exit}')"
test "$fp" = 4EFC590696CB15B87C73A3AD82CC8797C838DCFD
sudo install -m 0644 "$work/staging/zabbly.asc" /etc/apt/keyrings/s25-zabbly.asc
sudo tee /etc/apt/sources.list.d/s25-zabbly.sources >/dev/null <<'S25_APT_SOURCE'
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: noble
Components: main
Architectures: amd64
Signed-By: /etc/apt/keyrings/s25-zabbly.asc
S25_APT_SOURCE
sudo apt-get update -qq
pkg="$(apt-cache madison incus | awk '/pkgs.zabbly.com/ && $3 ~ /:7[.]6[-.]/ {print $3;exit}')"
test -n "$pkg"
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "incus=$pkg" "incus-client=$pkg" "incus-base=$pkg" >/dev/null
sudo systemctl start incus.service
sudo systemctl is-active --quiet incus.service
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo incus admin init --auto --storage-backend=dir
echo "S25_LAB_INCUS_APT=$pkg"
echo S25_LAB_NATIVE_INCUS_7_6_INITIALIZED=PASS

sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
uid="$(id -u boxweave)"
project="user-$uid"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project list --format csv >/dev/null
test "$(sudo incus project get "$project" restricted)" = true
if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project show default >/dev/null 2>&1; then
 echo S25_FAIL_FOREIGN_PROJECT_ACCESS >&2; exit 83
fi
echo S25_LAB_FOREIGN_NATIVE_PROJECT_DENIED=PASS
sudo incus image copy images:ubuntu/24.04 local: --target-project "$project" --alias boxweave-ubuntu-24.04 >/dev/null

net="incusbr-$uid"
acl="bw-s25-standalone-owned"
sudo incus network show "$net" >/dev/null
# Proxy opt-in is limited to this empty, newly created test Project.
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo incus project set "$project" restricted.devices.proxy=allow
test "$(sudo incus project get "$project" restricted)" = true
# Installed controller is the immutable released artifact, not a test build.
curl -fsSL --max-time 120 'https://github.com/vanishs/boxweave/releases/download/v1.0.0-rc.10/boxweave-v1.0.0-rc.10-linux-amd64' -o "$work/staging/boxweave"
test "$(sha256sum "$work/staging/boxweave" | cut -d' ' -f1)" = f67e769dbfeade39023f655b0a182993bf2459f645d182e52d04bba85093bfa6
go test -c -o "$work/staging/root-fault-fixture" ./internal/install
sudo install -m0755 "$work/staging/root-fault-fixture" /usr/local/bin/s5b-rc10-root-fault-fixture
go build -o "$work/staging/cli-auth-fixture" scripts/s5b_rc10_cli_auth_probe.go
sudo install -m0755 "$work/staging/cli-auth-fixture" /usr/local/bin/s5b-rc10-cli-auth-fixture
CGO_ENABLED=0 go build -tags s4b3_native_ci -o "$work/staging/packet" ./scripts/s4_b3_native_packet_probe.go
chmod 0755 "$work/staging/boxweave" "$work/staging/packet"
binary_sha="$(sha256sum "$work/staging/boxweave" | awk '{print $1}')"
sudo install -m 0755 "$work/staging/boxweave" /usr/local/bin/boxweave-s25-candidate

printf 'source_commit=%s\ncandidate_binary_sha256=%s\nincus_apt=%s\n' "$(git rev-parse HEAD)" "$binary_sha" "$pkg" > "$proof/identity.txt"
echo "S5B_RC10_RESIDUAL_PINNED_PRODUCT_SOURCE=eb021d3985c787c517cd7aee3efaf433d38f0393"
echo "S5B_RC10_RESIDUAL_PUBLISHED_BINARY_SHA256=$binary_sha"
sudo /usr/local/bin/boxweave-s25-candidate native-prepare-plan -project "$project" -pool default -network "$net" -acl "$acl" > "$work/private/plan.json"
sha="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["review_sha256"])' "$work/private/plan.json")"
if sudo /usr/local/bin/boxweave-s25-candidate native-prepare-apply -project "$project" -pool default -network "$net" -acl "$acl" -native-prepare-sha256 0000000000000000000000000000000000000000000000000000000000000000 -approve-local >/dev/null 2>&1; then
 echo S25_FAIL_WRONG_PREP_SHA_ACCEPTED >&2;exit 84
fi
sudo /usr/local/bin/boxweave-s25-candidate native-prepare-apply -project "$project" -pool default -network "$net" -acl "$acl" -native-prepare-sha256 "$sha" -approve-local > "$work/private/applied.json"
sudo /usr/local/bin/boxweave-s25-candidate native-prereq-plan -project "$project" > "$proof/native-prereq.json"
python3 - "$proof/native-prereq.json" <<'S25_PREREQ_ASSERT'
import sys,json
d=json.load(open(sys.argv[1]))
assert d["config_ready"] and len(d["checks"])==8 and all(x["ok"] for x in d["checks"]),d
S25_PREREQ_ASSERT
echo S25_LAB_NATIVE_PREREQ_EIGHT_CHECKS=PASS

sudo /usr/local/bin/boxweave-s25-candidate install-stage -project "$project" -source-binary /usr/local/bin/boxweave-s25-candidate -expect-sha256 "$binary_sha" -approve-stage > "$work/private/stage.json"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config /usr/local/bin/boxweave doctor -project "$project" > "$proof/doctor.json"
python3 - "$proof/doctor.json" <<'S25_DOCTOR_ASSERT'
import json,sys
s=json.load(open(sys.argv[1]))
assert s["ready"] and s["native"]["ready"],s
S25_DOCTOR_ASSERT
for name in admin1 admin2; do
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config /usr/local/bin/boxweave token-create > "$work/private/$name.receipt"
 chmod 0600 "$work/private/$name.receipt"
 # The CLI emits a two-line issuance receipt, while token_from_file
 # requires a single-line fixture. Reject malformed or duplicate records
 # before writing anything; both files remain private to this fresh runner.
 awk '
  NR == 1 && /^id=[^[:space:]]+ expires=[^[:space:]]+$/ { next }
  NR == 2 && /^token=[^[:space:]]+$/ { value=substr($0,7); next }
  { bad=1 }
  END { if (bad || NR != 2 || value == "") exit 1; print value }
 ' "$work/private/$name.receipt" > "$work/private/$name.secret"
 chmod 0600 "$work/private/$name.secret"
done
sudo /usr/local/bin/boxweave install-activate -project "$project" -approve-local > "$work/private/activated.json"
sudo systemctl is-active --quiet boxweave.service
test "$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}')" = 401
echo S25_LAB_LOOPBACK_MCP_UNAUTH_DENIAL=PASS
# Independent HTTP boundary negatives, no issued token material.
for path in /not-mcp /; do
 code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -X POST    "http://127.0.0.1:18650$path" -H 'Content-Type: application/json' -d '{}')"
 test "$code" = 404
done
for field in 'Origin: https://attacker.invalid' 'Host: attacker.invalid'; do
 code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -X POST    http://127.0.0.1:18650/mcp -H "$field"    -H 'Content-Type: application/json' -d '{}')"
 test "$code" = 403
done
code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -X POST  http://127.0.0.1:18650/mcp -H 'Authorization: Bearer definitely-invalid'  -H 'Content-Type: application/json' -d '{}')"
test "$code" = 401
echo S25_HTTP_BAD_HOST_ORIGIN_PATH_AND_TOKEN_DENIAL=PASS


# This is a separate, newly created CI test identity, never a live Box or
# copied user credential. The original frozen candidate remains untouched.
PYTHONPATH=scripts python3 -B scripts/s5b_rc10_residual_native.py "$work" "$project" "$net" "$acl" "$binary_sha"
