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
# N3 guard prerequisite, not final installed Peer acceptance.
if test "${PR178_SCOPED_N2:-0}" = 1; then
  CGO_ENABLED=1 go test -c -o "$work/n3-direction.test" ./internal/install
  for attempt in 1 2 3; do
    sudo unshare --net -- env BOXWEAVE_PR178_DIRECTION_NETNS=isolated-v1 "$work/n3-direction.test" -test.run '^TestPR178N3NativeDirectionGuard$' -test.v
  done
  echo PR178_N3_GUARD_PREREQUISITE=PASS
fi
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
# D1 exact-head installed read-only observations before workload admission.
d1_pid="$(sudo systemctl show boxweave.service -p MainPID --value)"
for actor in root boxweave; do
  if test "$actor" = root; then
    sudo "$installed" doctor -project "$project" > "$work/d1-$actor.json"
  else
    sudo runuser -u boxweave -- "$installed" doctor -project "$project" > "$work/d1-$actor.json"
  fi
done
python3 - "$work/d1-root.json" "$work/d1-boxweave.json" "${PR178_SCOPED_N2:-0}" <<'D1CHECK'
import json,sys
root,svc=(json.load(open(p)) for p in sys.argv[1:3])
for x in (root,svc):
    d=x["diagnostic"]
    assert x["ready"] and x["native"]["ready"] and x["schema_version"]==6
    assert d["version"]==1 and d["database"]["state"]=="OBSERVED" and d["database"]["epoch_ns"]>0
    assert d["coverage"]=="PARTIAL_D2_DIAGNOSIS" and not d["cross_source_atomic"] and d["mcp"]["reason"]=="LOCAL_HTTP_401_ONLY_AUTHENTICATED_MCP_NOT_CHECKED"
    assert d["installation"]["installed_identity"]=="VERIFIED" and d["installation"]["runtime"]=="VERIFIED"
r,s=(x["diagnostic"]["installation"] for x in (root,svc))
if sys.argv[3]=="1":
    assert r["mode"]=="SCOPED_V2" and r["root_proof"]=="VERIFIED",r
    assert root["diagnostic"]["scoped_ledger"]=={"owned":0,"retired":0}
    assert s["candidate"]=="SCOPED_V2" and s["mode"]=="UNVERIFIED" and s["root_proof"]=="ROOT_AUTHORITY_REQUIRED",s
else:
    assert r["mode"]=="LEGACY" and s["mode"]=="LEGACY",(r,s)
for x in (root,svc):
    d=x["diagnostic"]
    assert d["ledger"]["active_count"]==0 and d["ledger"]["forward_confirmed"]==0,d
    assert d["native_inventory"]["state"]=="OBSERVED" and d["native_inventory_info"]["inventory_scanned"],d
    assert d["forward_devices"]["state"]=="OBSERVED" and d["forward_verified"]==0,d
    assert d["oauth"]["state"]=="OBSERVED" and d["tunnel"]["state"]=="NOT_CHECKED" and d["packet_reachability"]["state"]=="NOT_CHECKED",d
    assert d["forward_policy"]["state"]=="OBSERVED",d
if sys.argv[3]=="1":
    assert root["diagnostic"]["root_policy"]["state"]=="OBSERVED" and root["diagnostic"]["root_policy"]["kernel_guard"]=="OBSERVED",root["diagnostic"]["root_policy"]
    assert root["diagnostic"]["ledger"]["peer_capacity"]["retained_directions"]==0
    assert "address_retirement" not in root and "address_retirement" not in svc
    assert svc["diagnostic"]["root_policy"]["state"]=="ROOT_AUTHORITY_REQUIRED"
else:
    assert root["diagnostic"]["root_policy"]["state"]=="NOT_APPLICABLE"
    assert root["address_retirement"]["pending"]==0
print("PR177_D2_INSTALLED_EMPTY_DIAGNOSTIC_COVERAGE=PASS",flush=True)
print("PR177_D1_INSTALLED_MODE_AND_AUTHORITY=PASS",flush=True)
D1CHECK
test "$d1_pid" = "$(sudo systemctl show boxweave.service -p MainPID --value)"
echo PR177_D1_INSTALLED_NO_SERVICE_RESTART=PASS

(umask 077; sudo runuser -u boxweave -- "$installed" token-create -project "$project" -ttl 30m > "$work/actor.receipt")
if test "${PR178_SCOPED_N2:-0}" = 1; then
  sudo python3 scripts/pr178_n2_scoped_lifecycle.py "$project" "$work/actor.receipt"
  echo PR178_N2_CLEAN_INSTALLED_LIFECYCLE_PASS
  sudo python3 scripts/pr178_n3_scoped_peer.py "$project" "$work/actor.receipt"
  sudo "$installed" doctor -project "$project" > "$work/d2-scoped-active.json"
  python3 - "$work/d2-scoped-active.json" <<'D2ACTIVE'
import json,sys
x=json.load(open(sys.argv[1]));d=x["diagnostic"];p=d["root_policy"];l=d["ledger"]
assert d["installation"]["mode"]=="SCOPED_V2" and p["state"]=="OBSERVED" and p["kernel_guard"]=="OBSERVED",p
assert l["network_observed"] and x["project_capacity"]["owned"]>=4
assert d["scoped_ledger"]["retired"]>=1 and l["peer_capacity"]["active_directions"]>=3
assert p["sequence"]==l["peer_capacity"]["completed_ledger_events"] and p["active_directions"]==l["peer_capacity"]["active_directions"],(p,l["peer_capacity"])
assert "address_retirement" not in x and not any(i["code"]=="LEGACY_IPV4_RETIREMENT_PENDING" for i in d["issues"])
# A changing independent native inventory is never a false healthy count.
if d["native_inventory"]["state"]=="OBSERVED":
    assert d["native_inventory_info"]["owned"]>=4
else:
    assert d["native_inventory"]["state"] in ("UNVERIFIED","PARTIAL")
    if d["native_inventory"]["state"]=="UNVERIFIED": assert "native_inventory_info" not in d
print("PR177_D2_SCOPED_ACTIVE_RETIRED_DIAGNOSTIC=PASS",flush=True)
D2ACTIVE

  sudo python3 scripts/pr178_n4_native.py "$project" "$work/actor.receipt"
  sudo python3 scripts/pr178_n4_concurrent.py "$project" "$work/actor.receipt"
  sudo python3 scripts/pr178_n4_reincarnation.py "$project" "$work/actor.receipt"
  CGO_ENABLED=1 go test -c -o "$work/n4-partial.test" ./internal/install
  chmod 0755 "$work/n4-partial.test"
  # Deliberate incomplete native original is the LAST test, on a fresh
  # disposable GitHub runner only. Preserve its journal and resource locks.
  sudo systemctl stop boxweave.service
  sudo python3 scripts/pr178_n4_partial.py "$project" "$work/n4-partial.test" > "$work/n4-partial.log" || { cat "$work/n4-partial.log"; exit 1; }
  cat "$work/n4-partial.log"
  original="$(sed -n 's/.*PR178_N4_PARTIAL_ORIGINAL=\([0-9a-f]\{32\}\)$/\1/p' "$work/n4-partial.log")"
  test "${#original}" = 32
  (umask 077; sudo runuser -u boxweave -- "$installed" token-create -project "$project" -ttl 15m > "$work/n4-degraded.receipt")
  sudo python3 scripts/pr178_n4_degraded.py "$project" "$work/n4-degraded.receipt" "$original"
  d2_pid="$(sudo systemctl show boxweave.service -p MainPID --value)"
  d2_rc=0
  sudo "$installed" doctor -project "$project" > "$work/d2-scoped-degraded.json" || d2_rc=$?
  test "$d2_rc" -le 1
  python3 - "$work/d2-scoped-degraded.json" "$original" <<'D2DEGRADED'
import json,sys
x=json.load(open(sys.argv[1]));d=x["diagnostic"];p=d["root_policy"];l=d["ledger"]
assert d["mcp"]["state"]=="OBSERVED",d["mcp"]
assert p["state"]=="JOURNAL_INCOMPLETE_OR_UNVERIFIED" and p["kernel_guard"]!="OBSERVED",p
orig=next(v for v in l["active_operations"] if v["original_id"]==sys.argv[2])
assert orig["state"]=="UNKNOWN" and orig["claim_protection"]=="RETAINED" and orig["live_claims"]==5 and orig["recovery_route"]=="MCP_RECOVERY_INSPECTION_REQUIRED",orig
assert "address_retirement" not in x and any(i["code"]=="SCOPED_POLICY_JOURNAL_UNVERIFIED" for i in d["issues"])
print("PR177_D2_SCOPED_DEGRADED_DIAGNOSTIC=PASS",flush=True)
D2DEGRADED
  test "$d2_pid" = "$(sudo systemctl show boxweave.service -p MainPID --value)"

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
