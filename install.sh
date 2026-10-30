#!/usr/bin/env bash
# Explicit, digest-gated Ubuntu application install/update coordinator.
# Never executes network downloads, invents credentials, or upgrades existing providers.
set -Eeuo pipefail
umask 077
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
usage() {
  cat <<'USAGE'
Usage: bash install.sh plan|apply --app-user NAME --archive FILE --sha256 HEX \
  --config FILE [--prefix ABSOLUTE_PATH] [--distribution ABSOLUTE_FILE]
  [--provision-provider --provider-installer FILE --provider-sha256 HEX \
   --ack-provider-root-changes PROVISION_PROVIDER]

plan: read-only artifact/host checks; no installation.
apply: run digest-verified package installation as an existing non-root app user.
The optional provider provisioning invokes a separately reviewed, pinned LOCAL
upstream installer as root only on a host with no detected Incus/Containarium.
It does not configure an MCP launcher, issue credentials, manage SSH/firewall,
TLS/host keys or auto-migrate legacy data. A reviewed MCP configuration and
working user systemd bus are prerequisites. Rerun apply for app update/repair.
USAGE
}
die() { printf '[boxweave] %s\n' "$*" >&2; exit 2; }
(( $# )) || { usage; exit 2; }
ACTION=$1; shift
case "$ACTION" in plan|apply) ;; --help|-h|help) usage; exit 0;; *) die 'Choose plan or apply';; esac
APP_USER=''; ARCHIVE=''; EXPECTED=''; CONFIG=''; PREFIX=''; DISTRIBUTION="$ROOT/bwmini/distribution.py"
PROVISION=0; PROVIDER_INSTALLER=''; PROVIDER_SHA=''; ACK=''
while (( $# )); do
  case "$1" in
    --app-user|--archive|--sha256|--config|--prefix|--distribution|--provider-installer|--provider-sha256|--ack-provider-root-changes)
      (( $# >= 2 )) || die "Missing value for $1"
      case "$1" in
        --app-user) APP_USER=$2;; --archive) ARCHIVE=$2;; --sha256) EXPECTED=$2;;
        --config) CONFIG=$2;; --prefix) PREFIX=$2;; --distribution) DISTRIBUTION=$2;;
        --provider-installer) PROVIDER_INSTALLER=$2;; --provider-sha256) PROVIDER_SHA=$2;;
        --ack-provider-root-changes) ACK=$2;;
      esac
      shift 2;;
    --provision-provider) PROVISION=1; shift;;
    *) die "Unknown option: $1";;
  esac
done
[[ "$APP_USER" =~ ^[a-z_][a-z0-9_-]*$ && "$APP_USER" != root ]] || die 'Specify a non-root --app-user'
[[ "$EXPECTED" =~ ^[a-f0-9]{64}$ ]] || die 'Expected --sha256 must be lowercase SHA256'
[[ "$ARCHIVE" == /* && "$CONFIG" == /* && "$DISTRIBUTION" == /* ]] || die 'Archive, config and distribution paths must be absolute'
[[ -z "$PREFIX" || "$PREFIX" == /* ]] || die 'Prefix must be absolute'
command -v sha256sum >/dev/null && command -v python3 >/dev/null || die 'sha256sum and python3 are required'
getent passwd "$APP_USER" >/dev/null || die 'Application user must exist before installation'
HOME_DIR=$(getent passwd "$APP_USER" | cut -d: -f6)
UID_APP=$(id -u "$APP_USER")
if [[ -z "$PREFIX" ]]; then PREFIX="$HOME_DIR/boxweave-v2"; fi
[[ "$PREFIX" == "$HOME_DIR"/* && "$PREFIX" != "$HOME_DIR" ]] || die 'Private prefix must be inside the application user home'
[[ -f "$ARCHIVE" && ! -L "$ARCHIVE" && -f "$CONFIG" && ! -L "$CONFIG" && -f "$DISTRIBUTION" && ! -L "$DISTRIBUTION" ]] || die 'Reviewed local archive/config/distribution files required (no symlinks)'
check_hash() {
  local actual
  actual=$(sha256sum -- "$1") || die 'Digest could not be read'
  actual=${actual%% *}
  [[ "$actual" == "$2" ]] || die 'Independent digest mismatch: no execution'
}
check_hash "$ARCHIVE" "$EXPECTED"
if (( PROVISION )); then
  [[ "$ACK" == PROVISION_PROVIDER ]] || die 'Explicit provider root-change acknowledgement required'
  [[ "$PROVIDER_INSTALLER" == /* && "$PROVIDER_SHA" =~ ^[a-f0-9]{64}$ ]] || die 'Pinned absolute provider installer and digest required'
  [[ -f "$PROVIDER_INSTALLER" && ! -L "$PROVIDER_INSTALLER" ]] || die 'Provider installer must be a local regular file'
  check_hash "$PROVIDER_INSTALLER" "$PROVIDER_SHA"
  if command -v incus >/dev/null || command -v containarium >/dev/null || [[ -e /etc/containarium || -e /var/lib/incus ]]; then
    die 'Existing Incus/Provider detected: automatic provisioning or upgrade refused'
  fi
else
  [[ -z "$PROVIDER_INSTALLER" && -z "$PROVIDER_SHA" && -z "$ACK" ]] || die 'Provider options require --provision-provider'
fi
[[ -f /etc/os-release ]] || die 'Cannot identify host OS'
# shellcheck disable=SC1091
. /etc/os-release
[[ "$ID" == ubuntu && "$VERSION_ID" == 24.04 ]] || die 'Only Ubuntu 24.04 host is supported by this entry'
[[ "$(uname -m)" == x86_64 ]] || die 'Release wheel lock requires x86_64'
python3 -c 'import sys;raise SystemExit(0 if sys.version_info[:2]==(3,12) else 2)' || die 'Python 3.12 is required'
command -v systemctl >/dev/null || die 'systemd is required'
if (( ! PROVISION )); then
  command -v incus >/dev/null && command -v containarium >/dev/null || die 'Install/review Incus and Provider before application setup or explicitly provision the host'
fi
python3 - "$CONFIG" <<'PY'
import json, pathlib, sys
try:
    p = pathlib.Path(sys.argv[1]); cfg = json.loads(p.read_text())
    assert isinstance(cfg, dict) and cfg.get('mode') == 'mcp'
    assert isinstance(cfg.get('launcher'), list) and cfg['launcher']
    assert pathlib.Path(cfg['launcher'][0]).is_absolute()
    assert cfg.get('host', '127.0.0.1') == '127.0.0.1'
except (OSError, ValueError, KeyError, AssertionError, TypeError):
    sys.exit('Reviewed non-placeholder MCP config with loopback binding required')
PY
[[ -d "/run/user/$UID_APP" ]] || die 'Application user runtime bus is missing; prepare login/linger separately'
if [[ "$ACTION" == plan ]]; then
  printf '[boxweave] plan checked: user=%s prefix=%s provision_provider=%s; no writes, MCP/TLS/boot not verified\n' "$APP_USER" "$PREFIX" "$PROVISION"
  exit 0
fi
if (( PROVISION )); then
  [[ "$EUID" -eq 0 ]] || die 'Provider provisioning requires root; application itself remains non-root'
  printf '[boxweave] Executing independently reviewed local upstream installer as root; check its own rollback boundary\n' >&2
  bash "$PROVIDER_INSTALLER"
  command -v incus >/dev/null && command -v containarium >/dev/null || die 'Provider provisioning did not establish both CLI prerequisites'
fi
if [[ "$EUID" -eq 0 ]]; then
  command -v runuser >/dev/null || die 'runuser required to launch app as a non-root user'
  exec runuser -u "$APP_USER" -- env XDG_RUNTIME_DIR="/run/user/$UID_APP" python3 "$DISTRIBUTION" install \
    --archive "$ARCHIVE" --sha256 "$EXPECTED" --prefix "$PREFIX" --config "$CONFIG" --activate
fi
[[ "$(id -un)" == "$APP_USER" ]] || die 'Run apply as app user, or use root only to switch to the named existing app user'
exec python3 "$DISTRIBUTION" install --archive "$ARCHIVE" --sha256 "$EXPECTED" \
  --prefix "$PREFIX" --config "$CONFIG" --activate
