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
# Disposable hosted VM only: the root observer requires a protected parent chain.
sudo python3 - <<'PYCI'
import os, stat
for path in ("/", "/usr", "/usr/local", "/usr/local/bin"):
    st = os.lstat(path)
    assert stat.S_ISDIR(st.st_mode) and st.st_uid == 0, path
    if path in ("/usr/local", "/usr/local/bin"):
        os.chmod(path, stat.S_IMODE(st.st_mode) & ~0o022)
    assert not os.lstat(path).st_mode & 0o022, path
    print("N2_ROOT_PARENT_PROTECTED", path, oct(stat.S_IMODE(os.lstat(path).st_mode)), flush=True)
PYCI
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
  sudo sh install.sh --plan --scoped-peer-fresh --local-binary "$work/candidate-a" --sha256 "$first" > "$work/scoped-plan.txt"
  grep -Eq '^scoped_fresh_review_sha256=[a-f0-9]{64}$' "$work/scoped-plan.txt"
  grep -Fxq "project=$project" "$work/scoped-plan.txt"
  test ! -e /var/lib/boxweave-scoped-peer-v2
  test ! -e /usr/local/bin/boxweave
  echo INSTALL_SCOPED_READONLY_PLAN=PASS
  host_review="$(sed -n 's/^host_machine_id_sha256=//p' "$work/scoped-plan.txt")"
  test -n "$host_review"
  stale_review="$(printf '%064d' 0)"
  if sudo sh install.sh --apply --scoped-peer-fresh --project "$project" --local-binary "$work/candidate-a" --sha256 "$first" --confirm-host-sha256 "$host_review" --scoped-fresh-review-sha256 "$stale_review" > "$work/scoped-stale.log" 2>&1; then
    echo "stale scoped plan unexpectedly accepted" >&2
    exit 1
  fi
  grep -Fq "scoped fresh review digest changed" "$work/scoped-stale.log"
  test ! -e /var/lib/boxweave-scoped-peer-v2
  test ! -e /usr/local/bin/boxweave
  echo INSTALL_SCOPED_STALE_PLAN_REFUSED=PASS
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
  if sudo sh uninstall.sh --plan --scope full --project "$project" > "$work/scoped-uninstall.log" 2>&1; then
    echo "scoped uninstall unexpectedly accepted" >&2
    exit 1
  fi
  grep -Fq "scoped v2 retirement requires" "$work/scoped-uninstall.log"
  if sudo sh install.sh --plan --upgrade --local-binary "$work/candidate-b" --sha256 "$next" > "$work/scoped-upgrade.log" 2>&1; then
    echo "scoped wrapper upgrade unexpectedly accepted" >&2
    exit 1
  fi
  grep -Fq "scoped installation evidence exists" "$work/scoped-upgrade.log"
  sudo systemctl is-active --quiet boxweave.service boxweave-scoped-guard.service boxweave-scoped-lifecycle.service
  test ! -e /var/lib/boxweave-installer-backup
  echo INSTALL_SCOPED_WRAPPERS_REFUSE_LEGACY=PASS
  echo PR178_N2_FRESH_ONLY=PASS
else
  sudo incus project show "$project" > "$work/guided-project-before"
  sudo incus profile show default --project "$project" > "$work/guided-profile-before"
  sudo sh install.sh --plan --guided-fresh-plan --local-binary "$work/candidate-a" --sha256 "$first" > "$work/guided-plan.txt"
  grep -Fq 'GUIDED_FRESH_PLAN_ONLY:' "$work/guided-plan.txt"
  grep -Fq 'Reviewed native preparation:' "$work/guided-plan.txt"
  test ! -e /var/lib/boxweave
  test ! -e /var/lib/boxweave-native-prereq-v1.json
  test ! -e /usr/local/bin/boxweave
  sudo incus project show "$project" > "$work/guided-project-after"
  sudo incus profile show default --project "$project" > "$work/guided-profile-after"
  cmp "$work/guided-project-before" "$work/guided-project-after"
  cmp "$work/guided-profile-before" "$work/guided-profile-after"
  echo I2_GUIDED_LEGACY_PREVIEW_NO_EFFECT=PASS
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
# H3A: real installed CLI with a private disposable DB, no live auth change.
h3a_pid="$(sudo systemctl show boxweave.service -p MainPID --value)"
sudo runuser -u boxweave -- "$installed" oauth-status -project "$project" > "$work/h3a-before.json"
sudo python3 scripts/pr188_h3a_oauth.py --binary "$installed" --project "$project"
sudo runuser -u boxweave -- "$installed" oauth-status -project "$project" > "$work/h3a-after.json"
cmp "$work/h3a-before.json" "$work/h3a-after.json"
test "$(sudo systemctl show boxweave.service -p MainPID --value)" = "$h3a_pid"
echo PR188_H3A_INSTALLED_NO_EFFECT=PASS
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
    assert d["native_inventory"]["state"]=="OBSERVED" and d["native_inventory_info"]["entire_project_inventory_scanned"],d
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
d3_absent="d1-$(printf '%064d' 0)"
d3_rc=0
sudo "$installed" recover --plan --issue "$d3_absent" -project "$project" > "$work/d3-absent.json" || d3_rc=$?
test "$d3_rc" = 1
python3 - "$work/d3-absent.json" <<'D3ABSENT'
import json,sys
p=json.load(open(sys.argv[1]))
assert p["status"]=="BLOCKED_UNPROVEN" and p["reason"]=="ISSUE_NOT_PRESENT_OR_OUTSIDE_COVERAGE"
assert not p["can_repair"] and not p["apply_available"] and "review_sha256" not in p and not p["native_retried"],p
print("PR177_D3_INSTALLED_ABSENT_ISSUE_NO_EFFECT=PASS",flush=True)
D3ABSENT
test "$d1_pid" = "$(sudo systemctl show boxweave.service -p MainPID --value)"
# H1A uses the installed exact-head observer, without requiring controller runtime.
sudo python3 scripts/pr188_h1a_observe.py "$project" active-empty
sudo python3 scripts/pr188_h1b_plan.py "$project" active-empty
sudo systemctl stop boxweave.service
sudo python3 scripts/pr188_h1a_observe.py "$project" offline-empty
sudo python3 scripts/pr188_h1b_plan.py "$project" offline-empty --negative-fixtures
sudo systemctl start boxweave.service
sudo systemctl is-active --quiet boxweave.service


(umask 077; sudo runuser -u boxweave -- "$installed" token-create -project "$project" -ttl 30m > "$work/actor.receipt")
sudo python3 scripts/pr188_h3b_installed.py "$project" "$work/actor.receipt" "${PR178_SCOPED_N2:-0}"
sudo python3 scripts/pr188_h4a_installed.py "$project" "$work/actor.receipt" "${PR178_SCOPED_N2:-0}"
if test "${PR188_OFFLINE_RETIREMENT:-0}" = 1; then
  CGO_ENABLED=1 go test -c -o "$work/h2b-original.test" ./internal/install
  chmod 0755 "$work/h2b-original.test"
  sudo env RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr188_h2b_offline.py "$project" "$work/actor.receipt" "$work/h2b-original.test"
  exit 0
fi
if test "${PR178_SCOPED_N2:-0}" = 1; then
  sudo python3 scripts/pr178_n2_scoped_lifecycle.py "$project" "$work/actor.receipt"
  echo PR178_N2_CLEAN_INSTALLED_LIFECYCLE_PASS
  if test "${PR182_SCOPED_RETIREMENT:-0}" = 1; then
    sudo env GITHUB_ACTIONS="$GITHUB_ACTIONS" RUNNER_ENVIRONMENT="$RUNNER_ENVIRONMENT" python3 scripts/pr182_scoped_retirement.py "$project" "$work/actor.receipt" "$work/candidate-a" "$first"
    exit 0
  fi
  sudo env GITHUB_ACTIONS="$GITHUB_ACTIONS" RUNNER_ENVIRONMENT="$RUNNER_ENVIRONMENT" python3 scripts/pr191_scoped_lifecycle_recovery.py "$project" "$work/actor.receipt" "$first"
  sudo python3 scripts/pr178_n3_scoped_peer.py "$project" "$work/actor.receipt"
  sudo python3 scripts/pr188_h1a_observe.py "$project" live
  sudo python3 scripts/pr188_h1b_plan.py "$project" live --negative-fixtures
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
  sudo python3 scripts/pr177_d4_installed_recovery.py "$project" "$work/actor.receipt" "$work/n4-partial.test" SCOPED_V2
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
  d3_issue="$(python3 -c 'import json,sys;d=json.load(open(sys.argv[1]))["diagnostic"];print(next(i["id"] for i in d["issues"] if i["code"]=="PROJECT_PEER_BARRIER" and i.get("original_id")==sys.argv[2]))' "$work/d2-scoped-degraded.json" "$original")"
  d3_rc=0
  sudo "$installed" recover --plan --issue "$d3_issue" -project "$project" > "$work/d3-scoped-degraded.json" || d3_rc=$?
  test "$d3_rc" = 1
  python3 - "$work/d3-scoped-degraded.json" "$original" <<'D3DEGRADED'
import json,sys
p=json.load(open(sys.argv[1]))
assert p["status"]=="BLOCKED_UNPROVEN" and p["original_id"]==sys.argv[2] and p["original_state"]=="UNKNOWN",p
assert p["claim_count"]==5 and not p["can_repair"] and not p["apply_available"] and not p["native_retried"] and "review_sha256" not in p,p
assert p["mcp_inspect"]["arguments"]=={"operation_id":sys.argv[2],"mode":"inspect"}
print("PR177_D3_INSTALLED_PARTIAL_ORIGINAL_PLAN_BLOCKED=PASS",flush=True)
D3DEGRADED
  d4_rc=0
  sudo "$installed" recover --apply --issue "$d3_issue" -project "$project" --approve-local --reviewer d4-ci --recovery-key d4-partial-blocked-key --confirm-review-sha256 "$(printf '%064d' 0)" < "$work/d3-scoped-degraded.json" || d4_rc=$?
  test "$d4_rc" = 1
  echo PR177_D4_INSTALLED_PARTIAL_APPLY_REFUSED=PASS
  # Existing authenticated degraded acceptance independently rechecks original
  # and exactly five claims; planning must not create a recovery receipt.
  sudo python3 scripts/pr178_n4_degraded.py "$project" "$work/n4-degraded.receipt" "$original"
  test "$d2_pid" = "$(sudo systemctl show boxweave.service -p MainPID --value)"

  exit 0
fi
sudo python3 scripts/n2_installed_lifecycle.py mcp-retry "$project" "$work/actor.receipt"
CGO_ENABLED=1 go test -c -o "$work/d4-legacy.test" ./internal/install
chmod 0755 "$work/d4-legacy.test"
sudo python3 scripts/pr177_d4_installed_recovery.py "$project" "$work/actor.receipt" "$work/d4-legacy.test" LEGACY

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
# Reproduce the historical third-party Project drift only in this clean CI VM.
# Change a harmless reviewed description, not a security setting. Restore it
# exactly before the existing full retirement acceptance.
test -z "$(sudo incus list --all-projects -c n -f csv)"
sudo sh uninstall.sh --plan --guided-full-plan --scope full --project "$project" > "$work/retirement-precheck.json"
python3 - "$work/retirement-precheck.json" <<'PRECHECK'
import json,sys
x=json.load(open(sys.argv[1]))
assert x["ready"] and x["native_retirement_precheck"]["ready"],x
assert x["native_retirement_precheck"]["reason"]=="OBSERVED"
PRECHECK
pid_before="$(sudo systemctl show -p MainPID --value boxweave.service)"
desc_before="$(sudo incus project get "$project" description --property)"
restore_desc() { sudo incus project set "$project" "description=$desc_before" --property; }
trap restore_desc EXIT
sudo incus project set "$project" 'description=BoxWeave isolated CI drift fixture' --property
if sudo sh uninstall.sh --plan --guided-full-plan --scope full --project "$project" > "$work/retirement-drift.log" 2>&1; then
  echo "native drift was accepted" >&2
  exit 1
fi
grep -Fq 'NATIVE_CONFIG_DRIFT' "$work/retirement-drift.log"
test "$(sudo systemctl show -p MainPID --value boxweave.service)" = "$pid_before"
sudo systemctl is-active --quiet boxweave.service
test ! -e /var/lib/boxweave-uninstall-v1-audit.json
restore_desc
trap - EXIT
test "$(sudo incus project get "$project" description --property)" = "$desc_before"
echo I3_LEGACY_DRIFT_REFUSED_BEFORE_RETIREMENT=PASS
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
# Same Host, original Incus daemon and restricted Project retained. Erase
# the reviewed completed Legacy archive, then install a fresh Scoped ledger.
sudo env RUNNER_ENVIRONMENT=github-hosted python3 scripts/pr188_h2c_purge.py "$project"
test ! -e /var/lib/boxweave-uninstall-v1-audit.json
test ! -e /var/lib/boxweave-retired
test ! -e /var/lib/boxweave-purge-v1.json
echo RC14_LEGACY_PURGE=PASS
incus_pid="$(sudo systemctl show incus.service -p MainPID --value)"
sudo sh install.sh --plan --scoped-peer-fresh --local-binary "$work/candidate-a" --sha256 "$first" > "$work/scoped-reinstall-plan.txt"
host="$(sed -n 's/^host_machine_id_sha256=//p' "$work/scoped-reinstall-plan.txt")"
review="$(sed -n 's/^scoped_fresh_review_sha256=//p' "$work/scoped-reinstall-plan.txt")"
sudo sh install.sh --apply --scoped-peer-fresh --project "$project" --local-binary "$work/candidate-a" --sha256 "$first" --confirm-host-sha256 "$host" --scoped-fresh-review-sha256 "$review"
sudo "$installed" install-status -project "$project" > "$work/reinstall-status.json"
sudo python3 - "$work/reinstall-status.json" <<'REINSTALL'
import json, sqlite3, sys
x=json.load(open(sys.argv[1]))
assert x["scoped_peer_mode"]=="SCOPED_PEER_FRESH_V2" and x["service_active"],x
db=sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True)
for table in ("owned_boxes","operations","peer_grants","forward_claims","network_bindings","network_address_fences"):
    assert db.execute("SELECT count(*) FROM "+table).fetchone()[0]==0,table
print("RC14_FRESH_LEDGER=PASS",flush=True)
REINSTALL
test "$incus_pid" = "$(sudo systemctl show incus.service -p MainPID --value)"
echo RC14_LEGACY_TO_SCOPED_REINSTALL=PASS
echo N2_CLEAN_INSTALLED_LIFECYCLE_PASS
