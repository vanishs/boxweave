#!/bin/sh
# One-use GitHub-hosted S2.5 LAB. No shared Incus/production effects.
set -eu
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
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl gpg python3 >/dev/null
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
CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=1.0.25-s25lab -X main.commit=$(git rev-parse HEAD)" -o "$work/staging/boxweave" ./cmd/boxweave
CGO_ENABLED=1 go build -trimpath -o "$work/staging/s25-lab-agent" ./scripts/s25_lab_agent.go
cp scripts/s25_inner_setup.sh "$work/staging/s25_inner_setup.sh"
chmod 0755 "$work/staging/boxweave" "$work/staging/s25-lab-agent"
binary_sha="$(sha256sum "$work/staging/boxweave" | awk '{print $1}')"
agent_sha="$(sha256sum "$work/staging/s25-lab-agent" | awk '{print $1}')"
setup_sha="$(sha256sum "$work/staging/s25_inner_setup.sh" | awk '{print $1}')"
sudo install -m 0755 "$work/staging/boxweave" /usr/local/bin/boxweave-s25-candidate
printf 'source_commit=%s\ncandidate_binary_sha256=%s\nagent_binary_sha256=%s\ninner_setup_sha256=%s\nincus_apt=%s\n' "$(git rev-parse HEAD)" "$binary_sha" "$agent_sha" "$setup_sha" "$pkg" > "$proof/identity.txt"

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
for spec in "admin1:admin" "admin2:admin" "viewer:viewer"; do
 name="$(printf '%s\n' "$spec" | cut -d: -f1)"
 role="$(printf '%s\n' "$spec" | cut -d: -f2)"
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config /usr/local/bin/boxweave token-create -role "$role" > "$work/private/$name.secret"
 chmod 0600 "$work/private/$name.secret"
done
sudo /usr/local/bin/boxweave install-activate -project "$project" -approve-local > "$work/private/activated.json"
sudo systemctl is-active --quiet boxweave.service
test "$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}')" = 401
echo S25_LAB_LOOPBACK_MCP_UNAUTH_DENIAL=PASS

# Read-only binary staging on the disposable private Incus bridge only;
# credentials and SQLite are in a different 0700 directory.
gateway="$(sudo incus network get "$net" ipv4.address | cut -d/ -f1)"
test -n "$gateway"
port=18465
python3 -m http.server "$port" --bind "$gateway" --directory "$work/staging" > "$work/private/stage-http.log" 2>&1 &
stage_pid=$!
trap 'kill "$stage_pid" 2>/dev/null || :; wait "$stage_pid" 2>/dev/null || :' EXIT
i=0
while [ "$i" -lt 30 ]; do
 if curl -fsSI --connect-timeout 1 --max-time 2 "http://$gateway:$port/boxweave" >/dev/null 2>&1;then break;fi
 i=$((i+1));sleep 1
done
test "$i" -lt 30
echo S25_LAB_PRIVATE_BRIDGE_SHA_BOUND_ARTIFACTS_READY=PASS

export BW_S25_URL=http://127.0.0.1:18650/mcp
export BW_S25_PROJECT="$project"
export BW_S25_OUTER_DB=/var/lib/boxweave/ledger.sqlite
export BW_S25_ADMIN1_FILE="$work/private/admin1.secret"
export BW_S25_ADMIN2_FILE="$work/private/admin2.secret"
export BW_S25_VIEWER_FILE="$work/private/viewer.secret"
export BW_S25_CHECKPOINT="$work/private/outer-checkpoint.json"
export BW_S25_STAGE_URL="http://$gateway:$port"
export BW_S25_SETUP_SHA256="$setup_sha"
export BW_S25_AGENT_SHA256="$agent_sha"
export BW_S25_BINARY_SHA256="$binary_sha"
"$work/staging/s25-lab-agent" admit
"$work/staging/s25-lab-agent" recover
python3 scripts/s25_independent_verifier.py l1
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
"$work/staging/s25-lab-agent" double-agent
python3 scripts/s25_independent_verifier.py agents
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
"$work/staging/s25-lab-agent" bootstrap
python3 scripts/s25_independent_verifier.py l2
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
echo S25_TWO_LAYER_V1_MCP_CREATE_EXEC=PASS
"$work/staging/s25-lab-agent" inner-clean
python3 scripts/s25_independent_verifier.py l2clean
"$work/staging/s25-lab-agent" cleanup
python3 scripts/s25_independent_verifier.py clean
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
test -z "$(sudo incus list --project "$project" --format csv -c n)"
echo S25_TARGETED_MCP_DUAL_AGENT_RECOVERY_VERIFIER=PASS
echo S25_ALL_BH_AND_TC_FORMAL_SIGNOFF=NOT_YET
