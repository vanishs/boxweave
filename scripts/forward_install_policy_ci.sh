#!/bin/sh
# Install policy regression on a genuine one-use GitHub-hosted VM.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
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
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl gpg dnsmasq-base >/dev/null
curl -fsSL --max-time 35 https://pkgs.zabbly.com/key.asc -o "$work/key.asc"
test "$(gpg --show-keys --with-colons "$work/key.asc" | awk -F: '$1=="fpr"{print $10;exit}')" = 4EFC590696CB15B87C73A3AD82CC8797C838DCFD
sudo install -m 0644 "$work/key.asc" /etc/apt/keyrings/bw-forward-install.asc
echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/bw-forward-install.asc] https://pkgs.zabbly.com/incus/stable noble main' | sudo tee /etc/apt/sources.list.d/bw-forward-install.list >/dev/null
sudo apt-get update -qq
pkg="$(apt-cache madison incus | awk '/pkgs.zabbly.com/ && $3 ~ /:7[.]6[-.]/ {print $3;exit}')"
test -n "$pkg"
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "incus=$pkg" "incus-client=$pkg" "incus-base=$pkg" >/dev/null
sudo systemctl start incus.service
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo incus admin init --auto --storage-backend=dir
sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
uid="$(id -u boxweave)"
project="user-$uid"
sudo runuser -u boxweave -- incus project list --format csv >/dev/null
test "$(sudo incus project get "$project" restricted)" = true
test "$(sudo runuser -u boxweave -- id -nG)" = incus
proxy_before="$(sudo incus project get "$project" restricted.devices.proxy)"
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
  test "$(sudo incus project get "$project" restricted.devices.proxy)" = "$proxy_before"
  test -z "$(sudo incus list --all-projects --format csv -c n)"
}
unit_hash() { sudo sha256sum "$unit" | cut -d' ' -f1; }
manifest_unit() { python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["unit_sha256"])' "$1"; }
if [ "$FORWARD_INSTALL_MODE" = fresh ]; then
  sudo "$work/candidate" install-stage -project "$project" -source-binary "$work/candidate" -expect-sha256 "$candidate_sha" -approve-stage > "$work/stage.json"
  test "$(sudo grep -xc 'Environment=BOXWEAVE_FORWARD_EXTERNAL_ALLOW=all' "$unit")" = 1
  test "$(unit_hash)" = "$(manifest_unit "$work/stage.json")"
  sudo "$installed" install-activate -project "$project" -approve-local > "$work/active.json"
  policy all
  healthy
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
  sudo "$work/candidate" install-stage -project "$project" -source-binary "$work/candidate" -expect-sha256 "$candidate_sha" -approve-stage > "$work/stage-again.json"
  test "$(unit_hash)" = "$old_unit"
  policy ""
  echo FORWARD_INSTALL_FRESH_EXPLICIT_EMPTY_OVERRIDE_AND_REPEAT_STAGE=PASS
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
  sudo "$installed" upgrade -project "$project" -source-binary "$work/candidate" -expect-sha256 "$candidate_sha" -approve-local > "$work/upgraded.json"
  test "$(unit_hash)" = "$old_unit"
  test "$(manifest_unit "$work/upgraded.json")" = "$old_unit"
  test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$candidate_sha"
  healthy
  policy "$want"
  sudo "$installed" rollback -project "$project" -approve-local > "$work/rolled-back.json"
  test "$(unit_hash)" = "$old_unit"
  test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$old_sha"
  if [ "$FORWARD_INSTALL_MODE" = legacy-exact ]; then
    test "$(sudo sha256sum "$dropin" | cut -d' ' -f1)" = "$old_override"
  else
    test ! -e "$dropin"
  fi
  healthy
  policy "$want"
  echo "FORWARD_INSTALL_UPGRADE_ROLLBACK_PRESERVED=$FORWARD_INSTALL_MODE PASS"
fi
sudo systemctl stop boxweave.service
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo FORWARD_INSTALL_DISPOSABLE_SCOPE_PASS
