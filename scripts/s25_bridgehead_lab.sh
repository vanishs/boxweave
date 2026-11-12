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
cp scripts/s25_simwan6.py "$work/staging/s25_simwan6.py"
chmod 0755 "$work/staging/boxweave" "$work/staging/s25-lab-agent"
binary_sha="$(sha256sum "$work/staging/boxweave" | awk '{print $1}')"
agent_sha="$(sha256sum "$work/staging/s25-lab-agent" | awk '{print $1}')"
setup_sha="$(sha256sum "$work/staging/s25_inner_setup.sh" | awk '{print $1}')"
simwan_sha="$(sha256sum "$work/staging/s25_simwan6.py" | awk '{print $1}')"
sudo install -m 0755 "$work/staging/boxweave" /usr/local/bin/boxweave-s25-candidate

printf 'source_commit=%s\ncandidate_binary_sha256=%s\nagent_binary_sha256=%s\ninner_setup_sha256=%s\nincus_apt=%s\n' "$(git rev-parse HEAD)" "$binary_sha" "$agent_sha" "$setup_sha" "$pkg" > "$proof/identity.txt"
printf 'simwan6_helper_sha256=%s\n' "$simwan_sha" >> "$proof/identity.txt"

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

# Read-only binary staging on the disposable private Incus bridge only;
# credentials and SQLite are in a different 0700 directory.
gateway="$(sudo incus network get "$net" ipv4.address | cut -d/ -f1)"
test -n "$gateway"
port=18465
proxy_port=18466
python3 -m http.server "$port" --bind "$gateway" --directory "$work/staging" > "$work/private/stage-http.log" 2>&1 &
stage_pid=$!
# LAB-only bridge relay allows only explicitly listed public HTTPS hosts.
python3 scripts/s25_package_proxy.py "$gateway" "$proxy_port" > "$work/private/proxy.log" 2>&1 &
proxy_pid=$!
trap 'if [ -f "$work/private/outer-checkpoint.json" ]; then cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json" || :; fi; if [ -e "$work/private/simwan6/simwan6.json" ]; then sudo env BW_S25_SIMWAN_LAB=1 python3 scripts/s25_simwan6.py stop --workdir "$work/private/simwan6" || :; fi; kill "$stage_pid" "$proxy_pid" 2>/dev/null || :; wait "$stage_pid" "$proxy_pid" 2>/dev/null || :' EXIT
i=0
while [ "$i" -lt 30 ]; do
 if curl -fsSI --connect-timeout 1 --max-time 2 "http://$gateway:$port/boxweave" >/dev/null 2>&1;then break;fi
 i=$((i+1));sleep 1
done
test "$i" -lt 30
echo S25_LAB_PRIVATE_BRIDGE_SHA_BOUND_ARTIFACTS_READY=PASS
i=0
while [ "$i" -lt 15 ]; do
 if curl -4 -fsSI --proxy "http://$gateway:$proxy_port" --connect-timeout 2    --max-time 8 https://pkgs.zabbly.com/key.asc >/dev/null 2>&1; then break; fi
 i=$((i+1));sleep 1
done
test "$i" -lt 15
if curl -fsS -I --proxy "http://$gateway:$proxy_port" --noproxy ''   --max-time 4 https://127.0.0.1/ >/dev/null 2>&1; then
 echo S25_LAB_FAIL_PACKAGE_PROXY_ALLOWED_LOOPBACK >&2;exit 87
fi
echo S25_LAB_SCOPED_PACKAGE_PROXY_HTTPS_AND_DENIAL=PASS

sudo env BW_S25_SIMWAN_LAB=1 python3 scripts/s25_simwan6.py start --bridge "$net" --workdir "$work/private/simwan6"
sudo cp "$work/private/simwan6/simwan6.crt" "$work/staging/s25_simwan6.crt"
sudo chown "$(id -u):$(id -g)" "$work/staging/s25_simwan6.crt"
chmod 0644 "$work/staging/s25_simwan6.crt"
echo S25_L0_SIMWAN6_TLS_NAMESPACE_READY=PASS

export BW_S25_URL=http://127.0.0.1:18650/mcp
export BW_S25_PROJECT="$project"
export BW_S25_OUTER_DB=/var/lib/boxweave/ledger.sqlite
export BW_S25_ADMIN1_FILE="$work/private/admin1.secret"
export BW_S25_ADMIN2_FILE="$work/private/admin2.secret"
export BW_S25_VIEWER_FILE="$work/private/viewer.secret"
export BW_S25_CHECKPOINT="$work/private/outer-checkpoint.json"
export BW_S25_STAGE_URL="http://$gateway:$port"
export BW_S25_PACKAGE_PROXY_URL="http://$gateway:$proxy_port"
export BW_S25_SETUP_SHA256="$setup_sha"
export BW_S25_AGENT_SHA256="$agent_sha"
export BW_S25_BINARY_SHA256="$binary_sha"
export BW_S25_SIMWAN6_HELPER_SHA="$simwan_sha"
export BW_S25_SIMWAN6_URL='https://[2001:db8:25:2026::2]:18479/witness'
export BW_S25_SIMWAN6_CERT_FILE="$work/staging/s25_simwan6.crt"
export BW_S25_SIMWAN6_WORKDIR="$work/private/simwan6"
"$work/staging/s25-lab-agent" admit
"$work/staging/s25-lab-agent" recover
python3 scripts/s25_independent_verifier.py l1
"$work/staging/s25-lab-agent" wan6
python3 scripts/s25_independent_verifier.py wan6outer
"$work/staging/s25-lab-agent" wan6-clean
python3 scripts/s25_independent_verifier.py wan6outerclean
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
"$work/staging/s25-lab-agent" double-agent
python3 scripts/s25_independent_verifier.py agents
"$work/staging/s25-lab-agent" quiet-long-poll
python3 scripts/s25_independent_verifier.py quiet
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
# Installer/runtime durability requires *same SQLite* surviving actual
# systemd service restart, independently recovered by separate SDK process.
sudo systemctl restart boxweave.service
sudo systemctl is-active --quiet boxweave.service
# Type=simple systemd ACTIVE alone proves a process was started; the MCP
# loopback socket may still be unbound. Re-read the actual auth boundary
# before dispatching any Agent request after this deliberate restart.
restart_ready=no
restart_attempt=0
while [ "$restart_attempt" -lt 40 ]; do
 sudo systemctl is-active --quiet boxweave.service
 code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST   http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}' || :)"
 if [ "$code" = 401 ]; then restart_ready=yes; break; fi
 restart_attempt=$((restart_attempt+1))
 sleep 1
done
if [ "$restart_ready" != yes ]; then
 echo S25_RESTART_HTTP_AUTH_BOUNDARY_UNREADY=BLOCKED >&2
 exit 89
fi
echo S25_RESTART_AUTHENTIC_HTTP_READY=PASS
"$work/staging/s25-lab-agent" post-restart
python3 scripts/s25_independent_verifier.py agents
echo S25_ONE_USE_CONTROLLER_RESTART_NATIVE_SQLITE_RECONCILED=PASS
# Round 2 fault: kill the exact disposable L0 BoxWeave systemd main process,
# NOT Incus and not a shared/physical host. Original L1 and peer lifecycle
# receipts must remain immutable and be independently recovered after the
# actual process has been replaced. Active systemd alone is not HTTP ready.
old_main="$(sudo systemctl show boxweave.service -p MainPID --value)"
test "$old_main" -gt 1
sudo systemctl kill --kill-whom=main --signal=SIGKILL boxweave.service
kill_recovered=no
attempt=0
while [ "$attempt" -lt 40 ]; do
 new_main="$(sudo systemctl show boxweave.service -p MainPID --value)"
 state="$(sudo systemctl show boxweave.service -p ActiveState --value)"
 code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}' || :)"
 if [ "$new_main" -gt 1 ] && [ "$new_main" != "$old_main" ] && [ "$state" = active ] && [ "$code" = 401 ]; then
  kill_recovered=yes; break
 fi
 attempt=$((attempt+1))
 sleep 1
done
test "$kill_recovered" = yes
echo S25_R2_REAL_SERVICE_SIGKILL_DISTINCT_PID_HTTP_AUTH_RECOVERY=PASS
"$work/staging/s25-lab-agent" post-restart
python3 scripts/s25_independent_verifier.py agents
echo S25_R2_SIGKILL_ORIGINAL_TWO_AGENT_SQLITE_AND_NATIVE_NO_REPLAY=PASS
# Token revocation is a direct dedicated-user local CLI action; the old
# HTTP bearer is tested by a separate process but never logged/exported.
viewer_id="$(sed -n 's/^id=\([0-9a-f]*\) .*/\1/p' "$work/private/viewer.secret")"
test "$viewer_id" != ""
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config  /usr/local/bin/boxweave token-revoke -id "$viewer_id" >/dev/null
python3 - "$work/private/viewer.secret" <<'S25_REVOKED_HTTP'
import sys,urllib.request,urllib.error
token=next(x[6:].strip() for x in open(sys.argv[1]) if x.startswith("token="))
req=urllib.request.Request("http://127.0.0.1:18650/mcp",data=b"{}",
  headers={"Authorization":"Bearer "+token,"Content-Type":"application/json"},method="POST")
try:
  res=urllib.request.urlopen(req,timeout=8)
except urllib.error.HTTPError as exc:
  assert exc.code==401,exc.code
else:
  raise AssertionError("revoked bearer unexpectedly returned HTTP "+str(res.status))
print("S25_REVOKED_VIEWER_TOKEN_HTTP_DENIED=PASS")
S25_REVOKED_HTTP
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config  /usr/local/bin/boxweave token-create -role viewer > "$work/private/viewer.rotated"
chmod 0600 "$work/private/viewer.rotated"
export BW_S25_VIEWER_FILE="$work/private/viewer.rotated"
"$work/staging/s25-lab-agent" rotated-viewer
"$work/staging/s25-lab-agent" bootstrap
python3 scripts/s25_independent_verifier.py wan6inner
# Evidence is captured *before* any original SDK cleanup. These outputs are
# public test CA/packet metadata, exact native NIC and original IDs only.
# No Token, private key, SQLite DB or unbounded Guest output is exported.
sudo cat "$work/private/simwan6/server.log" > "$proof/simwan6-l0-upstream.log"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- cat /root/s25simwan6/server.log > "$proof/simwan6-l1-upstream.log"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- cat /root/s25-inner-checkpoint.json > "$proof/simwan6-l1-originals.json"
sudo ip -s link show s25s6host > "$proof/simwan6-l0-veth-packets.txt"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- ip -s link show s25s6host > "$proof/simwan6-l1-veth-packets.txt"
sudo incus config show bw-s25-mcp-l1 --project "$project" --expanded > "$proof/simwan6-l0-native-nic.yaml"
inner_uid="$(sudo incus exec bw-s25-mcp-l1 --project "$project" -- id -u boxweave)"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- incus config show bw-s25-mcp-l2 --project "user-$inner_uid" --expanded > "$proof/simwan6-l1-native-nic.yaml"
grep -q 'SIM_WAN6_REMOTE=' "$proof/simwan6-l0-upstream.log"
grep -q 'SIM_WAN6_REMOTE=' "$proof/simwan6-l1-upstream.log"
echo S25_SIMWAN6_BOTH_LAYER_RAW_PACKET_AND_NATIVE_EVIDENCE_ARCHIVED=PASS
python3 scripts/s25_independent_verifier.py l2
# This marker is emitted only once independent native L2 UUID, CPU and
# original Update/Stop/Start receipts were all verified, not from
# truncated Guest stdout inside the outer L1 bootstrap Exec.
echo S25_L1_MCP_L2_UPDATE_STOP_START_SAME_UUID=PASS
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
echo S25_TWO_LAYER_V1_MCP_CREATE_EXEC=PASS
"$work/staging/s25-lab-agent" guest-git
python3 scripts/s25_independent_verifier.py git
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
"$work/staging/s25-lab-agent" inner-clean
python3 scripts/s25_independent_verifier.py wan6innerclean
python3 scripts/s25_independent_verifier.py l2clean
"$work/staging/s25-lab-agent" cleanup
python3 scripts/s25_independent_verifier.py clean
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
test -z "$(sudo incus list --project "$project" --format csv -c n)"
# Round 2 independent operator rollback: run only *after* original SDK
# cleanup. The upgrade canary uses identical source with only a distinct
# build label; always verify actual install digest and preserve original
# SQLite receipt history. No quota stress and no native guest writes.
mkdir -m 0755 "$work/canary-readable"
CGO_ENABLED=1 go build -trimpath \
  -ldflags "-X main.version=1.0.25-hostgo-r2-canary -X main.commit=$(git rev-parse HEAD)" \
  -o "$work/canary-readable/boxweave-r2-canary" ./cmd/boxweave
canary_sha="$(sha256sum "$work/canary-readable/boxweave-r2-canary" | awk '{print $1}')"
test "$canary_sha" != "$binary_sha"
test ! -e /var/lib/boxweave-installer-backup
if sudo /usr/local/bin/boxweave rollback -project "$project" -approve-local > "$work/private/unbacked-rollback.json" 2>&1; then
 echo S25_R2_FAIL_UNBACKED_ROLLBACK_ACCEPTED >&2; exit 95
fi
if sudo /usr/local/bin/boxweave upgrade -project "$project" \
  -source-binary "$work/canary-readable/boxweave-r2-canary" \
  -expect-sha256 0000000000000000000000000000000000000000000000000000000000000000 \
  -approve-local > "$work/private/wrong-upgrade.json" 2>&1; then
 echo S25_R2_FAIL_WRONG_UPGRADE_SHA_ACCEPTED >&2; exit 95
fi
test "$(sha256sum /usr/local/bin/boxweave | awk '{print $1}')" = "$binary_sha"
sudo systemctl is-active --quiet boxweave.service
echo S25_R2_UNBACKED_ROLLBACK_AND_BAD_UPGRADE_DIGEST_REFUSED=PASS
sudo /usr/local/bin/boxweave upgrade -project "$project" \
  -source-binary "$work/canary-readable/boxweave-r2-canary" \
  -expect-sha256 "$canary_sha" -approve-local > "$work/private/upgraded.json"
test "$(sha256sum /usr/local/bin/boxweave | awk '{print $1}')" = "$canary_sha"
sudo systemctl is-active --quiet boxweave.service
sudo test -d /var/lib/boxweave-installer-backup
echo S25_R2_EXACT_SOURCE_CANARY_UPGRADE_WITH_LEDGER_BACKUP=PASS
sudo systemctl stop boxweave.service
test "$(sudo systemctl is-active boxweave.service)" = inactive
code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp || :)"
test "$code" != 401
echo S25_R2_OPERATOR_FALLBACK_WHILE_MCP_UNAVAILABLE=PASS
sudo /usr/local/bin/boxweave rollback -project "$project" -approve-local > "$work/private/rollback.json"
test "$(sha256sum /usr/local/bin/boxweave | awk '{print $1}')" = "$binary_sha"
rollback_ready=no
attempt=0
while [ "$attempt" -lt 40 ]; do
 sudo systemctl is-active --quiet boxweave.service
 code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp || :)"
 if [ "$code" = 401 ]; then rollback_ready=yes; break; fi
 attempt=$((attempt+1));sleep 1
done
test "$rollback_ready" = yes
echo S25_R2_OPERATOR_ROOT_LOCAL_EXACT_BINARY_ROLLBACK=PASS
"$work/staging/s25-lab-agent" round2-postrollback
python3 scripts/s25_independent_verifier.py round2-postrollback
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
printf 'round2_original_binary_sha256=%s\nround2_canary_binary_sha256=%s\n' "$binary_sha" "$canary_sha" > "$proof/round2-binary-identities.txt"
echo S25_R2_ORIGINAL_SQLITE_NATIVE_CLEAN_AFTER_ROLLBACK=PASS
echo S25_TARGETED_MCP_DUAL_AGENT_RECOVERY_VERIFIER=PASS
echo S25_ALL_BH_AND_TC_FORMAL_SIGNOFF=NOT_YET
