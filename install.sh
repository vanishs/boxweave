#!/usr/bin/env bash
# Explicit, digest-gated Linux application install/update coordinator.
# Offline application coordinator; explicit bootstrap can initialize scoped credentials.
set -Eeuo pipefail
umask 077
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
usage() {
  cat <<'USAGE'
Usage: sudo bash install.sh [--provider-url URL --provider-secret-file ABSOLUTE_FILE --provider-cli ABSOLUTE_FILE]
  Streamed install/update/repair. Reuse an existing Containarium + Incus
  without changing its binaries, service, storage or signing secret.
  Existing Provider defaults: http://127.0.0.1:8080,
  /etc/containarium/jwt.secret, and containariumd on PATH.
  A standalone preinstalled Incus storage pool is also preserved.
  Custom Provider settings require the options above.
  Recommended: Ubuntu >= 24.04. Linux x86_64, Python >= 3.12 and systemd required.
  No distribution-name restriction. Host dependencies are checked directly.
  Missing packages use APT only when apt-get and a compatible repository
  suite are available; otherwise prepare the Provider/dependencies manually.
Usage: bash install.sh plan|apply --app-user NAME --archive FILE --sha256 HEX \
  --config FILE --distribution-sha256 HEX [--prefix ABSOLUTE_PATH] [--distribution ABSOLUTE_FILE]
  [--provision-provider --provider-installer FILE --provider-sha256 HEX \
   --ack-provider-root-changes PROVISION_PROVIDER]

bootstrap-plan|bootstrap --bootstrap-sha256 HEX [bootstrap options]:
  Prepare a dedicated app user, independent scoped MCP launcher and credential timer
  on an already reviewed Provider. See docs/HOST_BOOTSTRAP.md.

plan: read-only artifact/host checks; no installation.
apply: run digest-verified archive installation as an existing non-root app user.
--package is also accepted as an alias for --archive.
The optional provider provisioning invokes a separately reviewed, pinned LOCAL
upstream installer as root only on a host with no detected Incus/Containarium.
It does not configure an MCP launcher, issue credentials, manage SSH/firewall,
TLS/host keys or auto-migrate legacy data. A reviewed MCP configuration and
working user systemd bus are prerequisites. Rerun apply for app update/repair.
USAGE
}
die() { printf '[boxweave] %s\n' "$*" >&2; exit 2; }
if (( $# == 0 )) || [[ "${1:-}" == --provider-url || "${1:-}" == --provider-secret-file || "${1:-}" == --provider-cli ]]; then
  streamed_options=()
  while (( $# )); do
    case "$1" in
      --provider-url|--provider-secret-file|--provider-cli)
        (( $# >= 2 )) || die "Missing value for $1"
        streamed_options+=("$1" "$2"); shift 2;;
      *) die "Unknown streamed installer option: $1";;
    esac
  done
  [[ "$EUID" -eq 0 ]] || die 'DEFAULT_INSTALL_REQUIRES_ROOT: run the reviewed installer with sudo bash'
  command -v python3 >/dev/null || die 'Python 3 is required'
  python3 - "0.17.12" "${streamed_options[@]}" <<'PY_DEFAULT_RELEASE'
import hashlib,json,os,pathlib,re,subprocess,sys,tempfile,urllib.request
version=sys.argv[1]
api="https://api.github.com/repos/vanishs/boxweave"
prefix="https://github.com/vanishs/boxweave/releases/download/v"+version+"/"
def get(url,limit=1048576):
    request=urllib.request.Request(url,headers={"User-Agent":"BoxWeave-streamed-installer","Accept":"application/vnd.github+json"})
    with urllib.request.urlopen(request,timeout=60) as response:data=response.read(limit+1)
    if len(data)>limit:raise ValueError("release metadata limit")
    return data
try:
    release=json.loads(get(api+"/releases/tags/v"+version))
    assert release.get("tag_name")=="v"+version and not release.get("draft") and not release.get("prerelease")
    assets={a["name"]:a for a in release["assets"]}
    entry=assets["release.json"];expected=entry.get("digest","")
    assert re.fullmatch("sha256:[a-f0-9]{64}",expected)
    assert entry["browser_download_url"]==prefix+"release.json"
    metadata=get(entry["browser_download_url"])
    assert hashlib.sha256(metadata).hexdigest()==expected[7:]
    manifest=json.loads(metadata)
    assert manifest["format"]==1 and manifest["version"]==version and re.fullmatch("[a-f0-9]{40}",manifest["source"])
    reference=json.loads(get(api+"/git/ref/tags/v"+version))
    assert reference["object"]["type"]=="commit" and reference["object"]["sha"]==manifest["source"]
    expected=manifest["files"]["online.py"]
    assert re.fullmatch("[a-f0-9]{64}",expected)
    payload=get(prefix+"online.py",262144)
    assert hashlib.sha256(payload).hexdigest()==expected
    with tempfile.TemporaryDirectory(prefix="boxweave-release-") as folder:
        stage=pathlib.Path(folder);stage.chmod(0o700)
        (stage/"release.json").write_bytes(metadata)
        helper=stage/"online.py";helper.write_bytes(payload);helper.chmod(0o600)
        code=subprocess.run([sys.executable,str(helper),"install","--stage",folder,"--version",version,*sys.argv[2:]],stdin=subprocess.DEVNULL).returncode
        raise SystemExit(code)
except (OSError,ValueError,KeyError,AssertionError) as error:
    print("[boxweave] DEFAULT_RELEASE_NOT_READY_OR_INVALID: matching published release assets could not be verified; no unverified installer was executed.",file=sys.stderr)
    raise SystemExit(2)
PY_DEFAULT_RELEASE
  exit $?
fi
(( $# )) || { usage; exit 2; }
ACTION=$1; shift
if [[ "$ACTION" == bootstrap || "$ACTION" == bootstrap-plan ]]; then
  [[ "${1:-}" == --bootstrap-sha256 && "${2:-}" =~ ^[a-f0-9]{64}$ ]] || die 'Bootstrap requires --bootstrap-sha256 HEX as its first option'
  BOOTSTRAP_SHA=$2; shift 2
  bootstrap_stage=$(mktemp -d)
  trap 'rm -f -- "$bootstrap_stage/bootstrap.py"; rmdir -- "$bootstrap_stage"' EXIT
  cp -- "$ROOT/bwmini/bootstrap.py" "$bootstrap_stage/bootstrap.py"
  actual=$(sha256sum -- "$bootstrap_stage/bootstrap.py"); actual=${actual%% *}
  [[ "$actual" == "$BOOTSTRAP_SHA" ]] || die 'Independent bootstrap digest mismatch: no execution'
  mode=apply; [[ "$ACTION" != bootstrap-plan ]] || mode=plan
  python3 "$bootstrap_stage/bootstrap.py" "$mode" "$@"
  exit $?
fi
case "$ACTION" in plan|apply) ;; --help|-h|help) usage; exit 0;; *) die 'Choose plan or apply';; esac
APP_USER=''; ARCHIVE=''; EXPECTED=''; CONFIG=''; PREFIX=''; DISTRIBUTION="$ROOT/bwmini/distribution.py"; DIST_SHA=''
PROVISION=0; PROVIDER_INSTALLER=''; PROVIDER_SHA=''; ACK=''
while (( $# )); do
  case "$1" in
    --app-user|--package|--archive|--sha256|--config|--prefix|--distribution|--distribution-sha256|--provider-installer|--provider-sha256|--ack-provider-root-changes)
      (( $# >= 2 )) || die "Missing value for $1"
      case "$1" in
        --app-user) APP_USER=$2;; --package|--archive) ARCHIVE=$2;; --sha256) EXPECTED=$2;;
        --config) CONFIG=$2;; --prefix) PREFIX=$2;; --distribution) DISTRIBUTION=$2;;
        --distribution-sha256) DIST_SHA=$2;;
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
[[ "$DIST_SHA" =~ ^[a-f0-9]{64}$ ]] || die 'Expected --distribution-sha256 must be lowercase SHA256'
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
check_hash "$DISTRIBUTION" "$DIST_SHA"
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
[[ "$(uname -s)" == Linux ]] || die 'Linux host required'
# The reviewed provider installer is responsible for its own OS prerequisites.
# Never reject a host based on an /etc/os-release distribution name.
[[ "$(uname -m)" == x86_64 ]] || die 'Release wheel lock requires x86_64'
python3 -c 'import sys;raise SystemExit(0 if sys.version_info[:2]>=(3,12) else 2)' || die 'Python >= 3.12 is required'
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
  printf '[boxweave] plan checked: user=%s prefix=%s provision_provider=%s; no writes, MCP/TLS/boot not verified
' "$APP_USER" "$PREFIX" "$PROVISION"
  exit 0
fi
if (( PROVISION )); then
  [[ "$EUID" -eq 0 ]] || die 'Provider provisioning requires root; application itself remains non-root'
  printf '[boxweave] Executing independently reviewed local upstream installer as root; check its own rollback boundary
' >&2
  [[ "$(stat -c '%u' -- "$PROVIDER_INSTALLER")" == 0 ]] || die 'Privileged Provider installer must be root-owned'
  check_hash "$PROVIDER_INSTALLER" "$PROVIDER_SHA"
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
