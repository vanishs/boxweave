#!/bin/sh
# Guided upgrade acceptance on a genuine one-use GitHub-hosted VM.
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
FORWARD_INSTALL_MODE=legacy-exact
work="$(mktemp -d -p /tmp bw-guided-upgrade.XXXXXX)"
chmod 0755 "$work"
echo "GUIDED_UPGRADE_SOURCE=$(git rev-parse HEAD)"
CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags "-X main.version=1.0.0-dev.guided-upgrade -X main.commit=$(git rev-parse HEAD) -buildid=" -o "$work/candidate" ./cmd/boxweave
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
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project list --format csv >/dev/null
test "$(sudo incus project get "$project" restricted)" = true
test "$(sudo runuser -u boxweave -- id -nG)" = incus
proxy_before="$(sudo incus project get "$project" restricted.devices.proxy)"
unit=/etc/systemd/system/boxweave.service
dropin=/etc/systemd/system/boxweave.service.d/99-guided-upgrade-ci.conf
installed=/usr/local/bin/boxweave

backup=/var/lib/boxweave-installer-backup
old_sha="$(sha256sum "$work/legacy" | cut -d' ' -f1)"
sudo "$work/legacy" install-stage -project "$project" -source-binary "$work/legacy" -expect-sha256 "$old_sha" -approve-stage > "$work/stage.json"
sudo install -d -m 0755 "$(dirname "$dropin")"
printf '[Service]\nEnvironment="BOXWEAVE_FORWARD_EXTERNAL_ALLOW="\n' | sudo tee "$dropin" >/dev/null
sudo "$installed" install-activate -project "$project" -approve-local > "$work/active.json"
unit_sha="$(sudo sha256sum "$unit" | cut -d' ' -f1)"
override_sha="$(sudo sha256sum "$dropin" | cut -d' ' -f1)"
# Never print the actual fixture's original bearer.
(
 umask 077
 sudo runuser -u boxweave -- "$installed" token-create -project "$project" > "$work/auth.secret"
)
healthy() {
 sudo systemctl is-active --quiet boxweave
 test "$(sudo sha256sum "$unit" | cut -d' ' -f1)" = "$unit_sha"
 test "$(sudo sha256sum "$dropin" | cut -d' ' -f1)" = "$override_sha"
 test "$(sudo incus project get "$project" restricted.devices.proxy)" = "$proxy_before"
 test -z "$(sudo incus list --all-projects --format csv -c n)"
 PYTHONPATH=scripts python3 - "$work/auth.secret" "$1" <<'PY'
import pathlib,sys
from mcp_testkit import LoopbackMCP
raw=next(l[6:] for l in pathlib.Path(sys.argv[1]).read_text().splitlines() if l.startswith("token="))
with LoopbackMCP("http://127.0.0.1:18650/mcp", token=raw) as client:
    status=client.call("host_status",{}).structured
    assert status["version"] == sys.argv[2], "unexpected authenticated running version"
    assert status["mcp_ready"] and status["native"]["service_ready"]
PY
}
# A real controlling TTY with an automated fixture operator. No root host
# changes happen outside this one-use VM. The pty sends only the exact
# public confirmation or a denial and never prints credential values.
tty_run() {
 python3 - "$@" <<'PY'
import os,pty,select,sys,time,errno
answer=sys.argv[1]
argv=sys.argv[2:]
pid,fd=pty.fork()
if pid==0:
    os.execvp(argv[0],argv)
data=b""
answered=False
deadline=time.monotonic()+180
try:
    while time.monotonic()<deadline:
        ready,_,_=select.select([fd],[],[],1)
        if not ready:
            continue
        try:
            chunk=os.read(fd,65536)
        except OSError as exc:
            if exc.errno==errno.EIO:
                break
            raise
        if not chunk:
            break
        sys.stdout.buffer.write(chunk);sys.stdout.buffer.flush()
        data=(data+chunk)[-10000:]
        if b"to approve this exact plan:" in data and not answered:
            os.write(fd,(answer+"\n").encode());answered=True
    else:
        os.kill(pid,15)
        raise RuntimeError("guided installer fixture timed out")
finally:
    os.close(fd)
_,status=os.waitpid(pid,0)
sys.exit(os.waitstatus_to_exitcode(status))
PY
}
rc11_sha=54d16100e06b543568c4efdcb36cbd4bc07ae9412aeb7f271924f71f0bdce218
# Use the real released RC11 selector, not a forged manifest or fake CLI.
healthy 1.0.0-rc.10
manifest_before="$(sudo sha256sum /var/lib/boxweave/install-manifest.json | cut -d' ' -f1)"
sudo sh install.sh --plan --upgrade --version v1.0.0-rc.11 > "$work/plan.log"
grep -Fxq operation=upgrade "$work/plan.log"
grep -Fxq installed_version=1.0.0-rc.10 "$work/plan.log"
grep -Fxq project="$project" "$work/plan.log"
grep -Fxq binary_sha256="$rc11_sha" "$work/plan.log"
grep -Fxq previous_backup=none "$work/plan.log"
test "$(sudo sha256sum /var/lib/boxweave/install-manifest.json | cut -d' ' -f1)" = "$manifest_before"
sudo test ! -e "$backup"
if tty_run NO sudo sh install.sh --upgrade --version v1.0.0-rc.11 > "$work/declined.log"; then exit 83; fi
grep -Fq 'upgrade not approved' "$work/declined.log"
healthy 1.0.0-rc.10
sudo test ! -e "$backup"
if sudo sh install.sh --upgrade --version v1.0.0-rc.11 </dev/null > "$work/noninteractive.log" 2>&1; then exit 84; fi
grep -Fq 'noninteractive upgrade requires' "$work/noninteractive.log"
echo GUIDED_UPGRADE_PLAN_AND_DECLINE_READ_ONLY=PASS
tty_run "UPGRADE $project" sudo sh install.sh --upgrade --version v1.0.0-rc.11 > "$work/rc11.log"
grep -Fq GUIDED_UPGRADE_COMPLETE "$work/rc11.log"
test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$rc11_sha"
healthy 1.0.0-rc.11
sudo python3 - "$backup/receipt.json" <<'PY'
import json,sys
assert json.load(open(sys.argv[1]))["Previous"]["version"]=="1.0.0-rc.10"
PY
backup_before="$(sudo sha256sum "$backup/receipt.json" | cut -d' ' -f1)"
pid_before="$(sudo systemctl show -p MainPID --value boxweave)"
tty_run NO sudo sh install.sh --upgrade --version v1.0.0-rc.11 > "$work/noop.log"
grep -Fq UPGRADE_ALREADY_CURRENT "$work/noop.log"
test "$(sudo sha256sum "$backup/receipt.json" | cut -d' ' -f1)" = "$backup_before"
test "$(sudo systemctl show -p MainPID --value boxweave)" = "$pid_before"
echo GUIDED_UPGRADE_PUBLISHED_RC10_RC11_NOOP=PASS
# Legacy noninteractive apply must not silently commit the old generation.
host="$(tr -d '\n' < /etc/machine-id | sha256sum | cut -d' ' -f1)"
if sudo sh install.sh --apply --upgrade --project "$project" --confirm-host-sha256 "$host" --local-binary "$work/candidate" --sha256 "$candidate_sha" >"$work/explicit-block.log" 2>&1; then exit 85; fi
test "$(sudo sha256sum "$backup/receipt.json" | cut -d' ' -f1)" = "$backup_before"
healthy 1.0.0-rc.11
sudo sh install.sh --plan --upgrade --local-binary "$work/candidate" --sha256 "$candidate_sha" > "$work/second-plan.log"
grep -Fxq previous_backup_version=1.0.0-rc.10 "$work/second-plan.log"
if tty_run NO sudo sh install.sh --upgrade --local-binary "$work/candidate" --sha256 "$candidate_sha" > "$work/second-declined.log"; then exit 86; fi
test "$(sudo sha256sum "$backup/receipt.json" | cut -d' ' -f1)" = "$backup_before"
# Corrupt only the disposable backup witness, then restore the exact bytes.
sudo cp -p "$backup/receipt.json" "$work/receipt.original"
printf '{}' | sudo tee "$backup/receipt.json" >/dev/null
if tty_run "UPGRADE $project" sudo sh install.sh --upgrade --local-binary "$work/candidate" --sha256 "$candidate_sha" > "$work/corrupt.log" 2>&1; then exit 87; fi
sudo cp -p "$work/receipt.original" "$backup/receipt.json"
test "$(sudo sha256sum "$backup/receipt.json" | cut -d' ' -f1)" = "$backup_before"
healthy 1.0.0-rc.11
echo GUIDED_UPGRADE_BACKUP_NEGATIVES=PASS
tty_run "UPGRADE $project" sudo sh install.sh --upgrade --local-binary "$work/candidate" --sha256 "$candidate_sha" > "$work/candidate.log"
grep -Fq PREVIOUS_BACKUP_COMMITTED "$work/candidate.log"
grep -Fq GUIDED_UPGRADE_COMPLETE "$work/candidate.log"
test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$candidate_sha"
healthy 1.0.0-dev.guided-upgrade
sudo python3 - "$backup/receipt.json" <<'PY'
import json,sys
assert json.load(open(sys.argv[1]))["Previous"]["version"]=="1.0.0-rc.11"
PY
sudo "$installed" rollback -project "$project" -approve-local > "$work/rollback.json"
test "$(sudo sha256sum "$installed" | cut -d' ' -f1)" = "$rc11_sha"
healthy 1.0.0-rc.11
echo GUIDED_UPGRADE_PREVIOUS_COMMIT_NEW_BACKUP_AND_REAL_ROLLBACK=PASS
# Prove existing fresh install policy is still respected by the separate
# normal workflow and no native resources were created by upgrade.
sudo systemctl stop boxweave
sudo test -d "$backup"
rm -f "$work/auth.secret"
echo GUIDED_UPGRADE_DISPOSABLE_ACCEPTANCE_PASS
