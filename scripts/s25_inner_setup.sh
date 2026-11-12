#!/bin/sh
# Runs only via box_exec inside the MCP-created unprivileged L1, never with
# a parent Incus socket. All binary/script bytes are SHA256-bound.
set -eu
export DEBIAN_FRONTEND=noninteractive
umask 077
stage="$1"; binary_sha="$2"; agent_sha="$3"
test "$(id -u)" = 0
test ! -e /var/lib/incus
test ! -e /var/lib/boxweave
. /etc/os-release
test "$ID" = ubuntu && test "$VERSION_ID" = 24.04
echo S25_INNER_GUEST_START=PASS

curl -fsSL --retry 2 --max-time 90 "$stage/boxweave" -o /tmp/s25-v1
echo "$binary_sha  /tmp/s25-v1" | sha256sum -c -
curl -fsSL --retry 2 --max-time 90 "$stage/s25-lab-agent" -o /tmp/s25-agent
echo "$agent_sha  /tmp/s25-agent" | sha256sum -c -
install -m 0755 /tmp/s25-v1 /usr/local/bin/boxweave-s25-candidate
install -m 0755 /tmp/s25-agent /usr/local/bin/s25-lab-agent
echo S25_INNER_EXACT_BINARY_TRANSFER=PASS

apt-get update -qq
apt-get install -y --no-install-recommends ca-certificates curl gpg >/dev/null
curl -fsSL --retry 2 --max-time 40 https://pkgs.zabbly.com/key.asc -o /tmp/s25-zabbly.asc
fingerprint="$(gpg --show-keys --with-colons /tmp/s25-zabbly.asc | awk -F: '$1=="fpr"{print $10;exit}')"
test "$fingerprint" = 4EFC590696CB15B87C73A3AD82CC8797C838DCFD
install -d -m 0755 /etc/apt/keyrings
install -m 0644 /tmp/s25-zabbly.asc /etc/apt/keyrings/s25-zabbly.asc
cat > /etc/apt/sources.list.d/s25-zabbly.sources <<'S25_SIGNED_APT'
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: noble
Components: main
Architectures: amd64
Signed-By: /etc/apt/keyrings/s25-zabbly.asc
S25_SIGNED_APT
apt-get update -qq
pkg="$(apt-cache madison incus | awk '/pkgs.zabbly.com/ && $3 ~ /:7[.]6[-.]/ {print $3;exit}')"
test -n "$pkg"
apt-get install -y --no-install-recommends "incus=$pkg" "incus-client=$pkg" "incus-base=$pkg" >/dev/null
systemctl start incus.service
systemctl is-active --quiet incus.service
incus admin init --auto --storage-backend=dir
test -z "$(incus list --all-projects --format csv -c n)"
echo S25_INNER_PINNED_INCUS_DAEMON=PASS

useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
uid="$(id -u boxweave)"
project="user-$uid"
runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  incus project list --format csv >/dev/null
test "$(incus project get "$project" restricted)" = true
if runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  incus project show default >/dev/null 2>&1; then
 echo S25_FAIL_FOREIGN_INNER_PROJECT_VISIBLE >&2
 exit 51
fi
echo S25_INNER_FOREIGN_PROJECT_DENIED=PASS
incus image copy images:ubuntu/24.04 local: --target-project "$project" \
  --alias boxweave-ubuntu-24.04 >/dev/null
net="incusbr-$uid"
incus network show "$net" >/dev/null

target_acl="bw-s25-inner-owned"
candidate=/usr/local/bin/boxweave-s25-candidate
"$candidate" native-prepare-plan -project "$project" -pool default \
  -network "$net" -acl "$target_acl" > /root/s25-native-plan.json
sha="$(python3 -c 'import json;print(json.load(open("/root/s25-native-plan.json"))["review_sha256"])')"
test -n "$sha"
if "$candidate" native-prepare-apply -project "$project" -pool default \
  -network "$net" -acl "$target_acl" \
  -native-prepare-sha256 "0000000000000000000000000000000000000000000000000000000000000000" \
  -approve-local >/dev/null 2>&1; then
 echo S25_FAIL_WRONG_NATIVE_PREP_SHA_ACCEPTED >&2
 exit 53
fi
test ! -e /var/lib/boxweave-native-prereq-v1.json
"$candidate" native-prepare-apply -project "$project" -pool default \
  -network "$net" -acl "$target_acl" \
  -native-prepare-sha256 "$sha" -approve-local > /root/s25-native-applied.json
"$candidate" native-prereq-plan -project "$project" > /root/s25-prereq.json
python3 - <<'S25_INNER_NATIVE_ASSERT'
import json
s=json.load(open("/root/s25-prereq.json"))
assert s["config_ready"] and len(s["checks"])==8 and all(x["ok"] for x in s["checks"]),s
assert any("DIR_QUOTA_UNVERIFIED" in x for x in s.get("warnings",[])),s
S25_INNER_NATIVE_ASSERT
echo S25_INNER_APPROVED_NATIVE_PREP_8_OF_8=PASS

hash="$(sha256sum "$candidate" | awk '{print $1}')"
test "$hash" = "$binary_sha"
"$candidate" install-stage -project "$project" \
  -source-binary "$candidate" -expect-sha256 "$hash" -approve-stage > /root/s25-stage.json
runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  /usr/local/bin/boxweave doctor -project "$project" > /root/s25-doctor.json
python3 - <<'S25_INNER_DOCTOR_ASSERT'
import json
s=json.load(open("/root/s25-doctor.json"))
assert s["ready"] and s["native"]["ready"],s
S25_INNER_DOCTOR_ASSERT
for spec in "admin1:admin" "admin2:admin" "viewer:viewer"; do
 name="$(printf '%s\n' "$spec" | cut -d: -f1)"
 role="$(printf '%s\n' "$spec" | cut -d: -f2)"
 runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  /usr/local/bin/boxweave token-create -role "$role" > "/root/s25-$name.secret"
 chmod 0600 "/root/s25-$name.secret"
done
"$candidate" install-activate -project "$project" -approve-local > /root/s25-activate.json
systemctl is-active --quiet boxweave.service
curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}' | grep -qx 401
echo S25_INNER_EXACT_V1_SERVICE_LOOPBACK=PASS

export BW_S25_ALLOWED_NESTED=1 BW_S25_INNER=1
export BW_S25_URL=http://127.0.0.1:18650/mcp
export BW_S25_ADMIN1_FILE=/root/s25-admin1.secret
export BW_S25_ADMIN2_FILE=/root/s25-admin2.secret
export BW_S25_VIEWER_FILE=/root/s25-viewer.secret
export BW_S25_CHECKPOINT=/root/s25-inner-checkpoint.json
/usr/local/bin/s25-lab-agent admit
/usr/local/bin/s25-lab-agent recover
/usr/local/bin/s25-lab-agent nested-exec
echo S25_INNER_READY=PASS
