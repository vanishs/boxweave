#!/bin/sh
# Install policy regression on a genuine one-use GitHub-hosted VM.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "${GITHUB_REPOSITORY:-}" = vanishs/boxweave
test "$(id -u)" != 0
sudo -n true
grep -q '^ID=ubuntu$' /etc/os-release
grep -Eq '^VERSION_ID="?24\.04"?$' /etc/os-release
test ! -e /var/lib/incus
test ! -e /var/lib/boxweave
test ! -e /usr/local/bin/boxweave
test ! -e /etc/systemd/system/boxweave.service
test ! -e /etc/systemd/system/boxweave.service.d
! id boxweave >/dev/null 2>&1
case "${FORWARD_INSTALL_MODE:-}" in fresh|legacy-unset|legacy-exact) ;; *) exit 81;; esac
work="$(mktemp -d -p /tmp bw-forward-install.XXXXXX)"
chmod 0755 "$work"
echo "FORWARD_INSTALL_SOURCE=$(git rev-parse HEAD) MODE=$FORWARD_INSTALL_MODE"
CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags "-X main.version=1.0.0-dev.forward-install -X main.commit=$(git rev-parse HEAD) -buildid=" -o "$work/candidate" ./cmd/boxweave
candidate_sha="$(sha256sum "$work/candidate" | cut -d' ' -f1)"
echo "FORWARD_INSTALL_CANDIDATE_SHA256=$candidate_sha"
if [ "$FORWARD_INSTALL_MODE" != fresh ]; then
  # Independently pinned RC10 amd64 published bytes, with the old unit template.
  curl -fsSL --max-time 120 https://github.com/vanishs/boxweave/releases/download/v1.0.0-rc.10/boxweave-v1.0.0-rc.10-linux-amd64 -o "$work/legacy"
  printf '%s  %s\n' f67e769dbfeade39023f655b0a182993bf2459f645d182e52d04bba85093bfa6 "$work/legacy" | sha256sum -c -
  chmod 0755 "$work/legacy"
fi
sudo env GITHUB_ACTIONS="$GITHUB_ACTIONS" RUNNER_ENVIRONMENT="$RUNNER_ENVIRONMENT" python3 scripts/pr191_ci_apt_sources.py
sudo apt-get -o APT::Update::Error-Mode=any -o Acquire::Retries=2 update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl gpg dnsmasq-base conntrack nftables iproute2 >/dev/null
curl -fsSL --max-time 35 https://pkgs.zabbly.com/key.asc -o "$work/key.asc"
python3 scripts/pr191_ci_apt_sources.py verify-incus-key "$work/key.asc"
sudo install -m 0644 "$work/key.asc" /etc/apt/keyrings/bw-forward-install.asc
echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/bw-forward-install.asc] https://pkgs.zabbly.com/incus/stable noble main' | sudo tee /etc/apt/sources.list.d/bw-forward-install.list >/dev/null
sudo apt-get update -qq
pkg="$(python3 scripts/pr191_ci_apt_sources.py incus-package)"
test -n "$pkg"
if [ "$FORWARD_INSTALL_MODE" = fresh ]; then
  sudo python3 - <<'PYCI'
import os, stat
for path in ("/", "/usr", "/usr/local", "/usr/local/bin"):
    st = os.lstat(path)
    assert stat.S_ISDIR(st.st_mode) and st.st_uid == 0, path
    if path in ("/usr/local", "/usr/local/bin"):
        os.chmod(path, stat.S_IMODE(st.st_mode) & ~0o022)
    assert not os.lstat(path).st_mode & 0o022, path
PYCI
  sudo "$work/candidate" native-plan -incus-package-version "$pkg" > "$work/bootstrap-plan.json"
  bootstrap_review="$(python3 -c 'import json,sys;x=json.load(open(sys.argv[1]));assert x["mode"]=="fresh-install-proposed";print(x["plan_sha256"])' "$work/bootstrap-plan.json")"
  sudo "$work/candidate" native-bootstrap -incus-package-version "$pkg" -native-plan-sha256 "$bootstrap_review" -approve-native-bootstrap > "$work/bootstrap-apply.json"
  python3 -c 'import json,sys;x=json.load(open(sys.argv[1]));assert x["mode"]=="new-daemon-ready" and x["ipv4_address_pool"]>=2045' "$work/bootstrap-apply.json"
else
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "incus=$pkg" "incus-client=$pkg" "incus-base=$pkg" >/dev/null
  sudo systemctl start incus.service
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  sudo incus admin init --auto --storage-backend=dir
  sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
fi
uid="$(id -u boxweave)"
project="user-$uid"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project list --format csv >/dev/null
test "$(sudo incus project get "$project" restricted)" = true
test "$(sudo runuser -u boxweave -- id -nG)" = incus
proxy_before="$(sudo incus project get "$project" restricted.devices.proxy)"
proxy_expected="$proxy_before"
unit=/etc/systemd/system/boxweave.service
dropin=/etc/systemd/system/boxweave.service.d/99-forward-policy-ci.conf
installed=/usr/local/bin/boxweave
policy() {
  # Select only this non-secret variable, never dump the service environment.
  sudo python3 - "$1" <<'PY'
import pathlib, subprocess, sys
pid = int(subprocess.check_output(["systemctl","show","boxweave.service","-p","MainPID","--value"]))
assert pid > 1
values = [p.split(b"=",1)[1].decode("ascii") for p in pathlib.Path(f"/proc/{pid}/environ").read_bytes().split(b"\x00") if p.startswith(b"BOXWEAVE_FORWARD_EXTERNAL_ALLOW=")]
expected = [] if sys.argv[1] == "UNSET" else [sys.argv[1]]
assert values == expected, "unexpected effective Forward policy"
PY
}
healthy() {
  sudo systemctl is-active --quiet boxweave.service
  test "$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}')" = 401
  test "$(ss -H -ltn 'sport = :18650' | awk '{print $4}')" = 127.0.0.1:18650
  test "$(sudo incus project get "$project" restricted.devices.proxy)" = "$proxy_expected"
  test -z "$(sudo incus list --all-projects --format csv -c n)"
}
unit_hash() { sudo sha256sum "$unit" | cut -d' ' -f1; }
manifest_unit() { python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["unit_sha256"])' "$1"; }
if [ "$FORWARD_INSTALL_MODE" = fresh ]; then
  case "$proxy_before" in ""|block) ;; *) exit 84;; esac
  bridge="incusbr-$uid"
  native_preimage() {
    sudo incus query "/1.0/projects/$project" | python3 -c 'import json,sys;x=json.load(sys.stdin);print(json.dumps({k:x[k] for k in ("config","description")},sort_keys=True))'
    sudo incus query "/1.0/profiles/default?project=$project" | python3 -c 'import json,sys;x=json.load(sys.stdin);print(json.dumps({k:x[k] for k in ("config","description","devices")},sort_keys=True))'
  }
  native_preimage > "$work/native-before.jsonl"
  sudo sh install.sh --plan --scoped-peer-fresh --local-binary "$work/candidate" --sha256 "$candidate_sha" > "$work/scoped-plan.txt"
  grep -Fxq "project=$project" "$work/scoped-plan.txt"
  host_review="$(sed -n 's/^host_machine_id_sha256=//p' "$work/scoped-plan.txt")"
  review="$(sed -n 's/^scoped_fresh_review_sha256=//p' "$work/scoped-plan.txt")"
  printf '%s\n' "$host_review" | grep -Ex '[a-f0-9]{64}'
  printf '%s\n' "$review" | grep -Ex '[a-f0-9]{64}'
  test ! -e /var/lib/boxweave-scoped-peer-v2
  test ! -e "$installed"
  if sudo sh install.sh --apply --scoped-peer-fresh --project "$project" --local-binary "$work/candidate" --sha256 "$candidate_sha" --confirm-host-sha256 "$host_review" --scoped-fresh-review-sha256 "$(printf '%064d' 0)" > "$work/stale.log" 2>&1; then
    exit 85
  fi
  grep -Fq 'scoped fresh review digest changed' "$work/stale.log"
  test ! -e /var/lib/boxweave-scoped-peer-v2
  test ! -e "$installed"
  native_preimage > "$work/native-stale.jsonl"
  cmp "$work/native-before.jsonl" "$work/native-stale.jsonl"
  echo FORWARD_INSTALL_SCOPED_STALE_REVIEW_REFUSED_NO_EFFECT=PASS
  python3 scripts/n2_installed_lifecycle.py tty-install-scoped "$project" "$work/candidate" "$candidate_sha"
  sudo "$installed" install-status -project "$project" > "$work/active.json"
  python3 -c 'import json,sys;x=json.load(open(sys.argv[1]));assert x["scoped_peer_mode"]=="SCOPED_PEER_FRESH_V2" and x["service_active"] and not x["restart_required"] and x["binary_sha256"]==sys.argv[2]' "$work/active.json" "$candidate_sha"
  sudo systemctl is-active --quiet boxweave.service boxweave-scoped-guard.service boxweave-scoped-lifecycle.service
  test "$(sudo grep -xc 'Environment=BOXWEAVE_FORWARD_EXTERNAL_ALLOW=all' "$unit")" = 1
  test "$(unit_hash)" = "$(manifest_unit "$work/active.json")"
  proxy_expected=allow
  test "$(sudo incus project get "$project" restricted)" = true
  policy all
  healthy
  echo FORWARD_INSTALL_REVIEWED_NATIVE_PROXY_GRANT=PASS
  echo FORWARD_INSTALL_FRESH_ALL_AND_LOOPBACK_MANAGEMENT=PASS
  old_unit="$(unit_hash)"
  sudo install -d -m 0755 "$(dirname "$dropin")"
  printf '[Service]\nEnvironment="BOXWEAVE_FORWARD_EXTERNAL_ALLOW="\n' | sudo tee "$dropin" >/dev/null
  sudo systemctl daemon-reload
  sudo systemctl restart boxweave.service
  # installer health verifies the actual running binary and HTTP response
  sudo "$installed" install-activate -project "$project" -approve-local > "$work/override.json"
  policy ""
  healthy
  pid_before="$(sudo systemctl show -p MainPID --value boxweave.service)"
  old_override="$(sudo sha256sum "$dropin" | cut -d' ' -f1)"
  old_manifest="$(sudo sha256sum /var/lib/boxweave/install-manifest.json | cut -d' ' -f1)"
  native_preimage > "$work/native-installed.jsonl"
  if sudo "$work/candidate" install-stage -project "$project" -source-binary "$work/candidate" -expect-sha256 "$candidate_sha" -scoped-fresh-review-sha256 "$review" -approve-stage > "$work/repeat-stage.log" 2>&1; then
    exit 87
  fi
  grep -Fq 'unsupported existing or unreadable BoxWeave installation:' "$work/repeat-stage.log"
  test "$(unit_hash)" = "$old_unit"
  test "$(sudo sha256sum "$dropin" | cut -d' ' -f1)" = "$old_override"
  test "$(sudo sha256sum /var/lib/boxweave/install-manifest.json | cut -d' ' -f1)" = "$old_manifest"
  test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$candidate_sha"
  test "$(sudo systemctl show -p MainPID --value boxweave.service)" = "$pid_before"
  native_preimage > "$work/native-repeat.jsonl"
  cmp "$work/native-installed.jsonl" "$work/native-repeat.jsonl"
  policy ""
  healthy
  echo FORWARD_INSTALL_FRESH_EXPLICIT_EMPTY_OVERRIDE_AND_REPEAT_STAGE_REFUSED_NO_EFFECT=PASS
  sudo incus image copy images:ubuntu/24.04 local: --target-project "$project" --alias boxweave-ubuntu-24.04 >/dev/null
  (umask 077; sudo runuser -u boxweave -- "$installed" token-create -project "$project" > "$work/actor.receipt")
  (umask 077; awk 'NR==2 && /^token=[^[:space:]]+$/ {sub(/^token=/,""); print; ok=1} END {if(!ok || NR!=2) exit 1}' "$work/actor.receipt" > "$work/actor.secret")
  sudo systemctl start boxweave.service
  policy ""
  addr="$(sudo incus network get "$bridge" ipv4.address | cut -d/ -f1)"
  test -n "$addr"
  python3 scripts/forward_project_capabilities_native.py "$work" "$project" "$addr" "$candidate_sha"
  test ! -e "$dropin"
  sudo rmdir "$(dirname "$dropin")"
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  # Supported normal retirement must restore the original reviewed native scope.
  sudo sh uninstall.sh --scoped --plan --project "$project" > "$work/retirement-plan.json"
  python3 -c 'import json,sys;x=json.load(open(sys.argv[1]));assert x["ready"] and x["mode"]=="SCOPED_PEER_FRESH_V2"' "$work/retirement-plan.json"
  PYTHONPATH=scripts python3 - "$project" <<'PYRETIRE'
import sys
from n2_installed_lifecycle import tty_run
p=sys.argv[1]
tty_run(["sudo","sh","uninstall.sh","--scoped","--full","--project",p],
        [(("Type REMOVE SCOPED "+p).encode(),"REMOVE SCOPED "+p)])
PYRETIRE
  native_preimage > "$work/native-retired.jsonl"
  cmp "$work/native-before.jsonl" "$work/native-retired.jsonl"
  test "$(sudo incus project get "$project" restricted.devices.proxy)" = "$proxy_before"
  test ! -e "$installed"
  sudo systemctl is-active --quiet incus.service
  echo FORWARD_INSTALL_SCOPED_RETIREMENT_EXACT_NATIVE_PREIMAGE=PASS
else
  old_sha="$(sha256sum "$work/legacy" | cut -d' ' -f1)"
  sudo "$work/legacy" install-stage -project "$project" -source-binary "$work/legacy" -expect-sha256 "$old_sha" -approve-stage > "$work/stage.json"
  if sudo grep -q '^Environment=BOXWEAVE_FORWARD_EXTERNAL_ALLOW=' "$unit"; then exit 82; fi
  old_unit="$(unit_hash)"
  test "$old_unit" = "$(manifest_unit "$work/stage.json")"
  want=UNSET
  if [ "$FORWARD_INSTALL_MODE" = legacy-exact ]; then
    addr="$(sudo incus network get "incusbr-$uid" ipv4.address | cut -d/ -f1)"
    test -n "$addr"
    want="incusbr-$uid/tcp/$addr:23095"
    sudo install -d -m 0755 "$(dirname "$dropin")"
    printf '[Service]\nEnvironment="BOXWEAVE_FORWARD_EXTERNAL_ALLOW=%s"\n' "$want" | sudo tee "$dropin" >/dev/null
    old_override="$(sudo sha256sum "$dropin" | cut -d' ' -f1)"
  fi
  sudo "$installed" install-activate -project "$project" -approve-local > "$work/active.json"
  healthy
  policy "$want"
  # Schema 6 intentionally has no pre-stable migration from published RC10.
  pid_before="$(sudo systemctl show -p MainPID --value boxweave)"
  if sudo "$installed" upgrade -project "$project" -source-binary "$work/candidate" -expect-sha256 "$candidate_sha" -approve-local > "$work/upgrade-denied.log" 2>&1; then
    exit 86
  fi
  grep -Fq 'restricted service preflight unavailable' "$work/upgrade-denied.log"
  test "$(sudo systemctl show -p MainPID --value boxweave)" = "$pid_before"
  test "$(unit_hash)" = "$old_unit"
  test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$old_sha"
  test ! -e /var/lib/boxweave-installer-backup
  if [ "$FORWARD_INSTALL_MODE" = legacy-exact ]; then
    test "$(sudo sha256sum "$dropin" | cut -d' ' -f1)" = "$old_override"
  else
    test ! -e "$dropin"
  fi
  healthy
  policy "$want"
  echo "FORWARD_INSTALL_INCOMPATIBLE_SCHEMA_PRESERVED=$FORWARD_INSTALL_MODE PASS"
fi
if [ "$FORWARD_INSTALL_MODE" != fresh ]; then
  sudo systemctl stop boxweave.service
fi
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo FORWARD_INSTALL_DISPOSABLE_SCOPE_PASS
