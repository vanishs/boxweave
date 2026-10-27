#!/usr/bin/env bash
# BoxWeave server installer for Ubuntu. Run with `curl ... | sudo bash`.
set -Eeuo pipefail
umask 077

REPO_FROM_ENV=${BOXWEAVE_REPO+x}
REF_FROM_ENV=${BOXWEAVE_REF+x}
MCP_FROM_ENV=${CONTAINARIUM_VERSION+x}
BOXWEAVE_REPO="${BOXWEAVE_REPO:-vanishs/boxweave}"
BOXWEAVE_REF="${BOXWEAVE_REF:-main}"
CONTAINARIUM_VERSION="${CONTAINARIUM_VERSION:-latest}"
CONTAINARIUMD_BINARY="${BOXWEAVE_CONTAINARIUMD_BINARY:-}"
JWT_SECRET_FILE="${BOXWEAVE_JWT_SECRET_FILE:-/etc/containarium/jwt.secret}"
LINUX_USER="${BOXWEAVE_LINUX_USER:-${SUDO_USER:-}}"
SSH_HOST="${BOXWEAVE_SSH_HOST:-}"
SSH_PORT="${BOXWEAVE_SSH_PORT:-}"
ALLOWED_CIDRS="${BOXWEAVE_ALLOWED_CIDRS:-}"
API_URL="${BOXWEAVE_API_URL:-http://127.0.0.1:8080}"
SYNC_INTERVAL="${BOXWEAVE_SYNC_INTERVAL:-15}"
API_READY_TIMEOUT="${BOXWEAVE_API_READY_TIMEOUT:-900}"
RAW_BASE="https://raw.githubusercontent.com/${BOXWEAVE_REPO}/${BOXWEAVE_REF}"
CONFIG_DIR=/etc/boxweave
BIN=/usr/local/sbin
INSTALL_PHASE='initialization'
TMP_INSTALL=''
TMPDIR_DL=''
STAGING_FILE=''
BACKUP_DIR=''
TIMERS_PAUSED=0
SAVED_BOX_PUB=''
LOCAL_SOURCE_DIR=''
WEB_MODE="${BOXWEAVE_WEB_MODE:-}"
WEB_HOST="${BOXWEAVE_WEB_HOST:-}"
WEB_PORT="${BOXWEAVE_WEB_PORT:-}"
WEB_BACKEND_PORT="${BOXWEAVE_WEB_BACKEND_PORT:-}"
declare -A EXPLICIT=()
for option in 'LINUX_USER:BOXWEAVE_LINUX_USER' 'SSH_HOST:BOXWEAVE_SSH_HOST' 'SSH_PORT:BOXWEAVE_SSH_PORT' \
              'ALLOWED_CIDRS:BOXWEAVE_ALLOWED_CIDRS' 'API_URL:BOXWEAVE_API_URL' 'SYNC_INTERVAL:BOXWEAVE_SYNC_INTERVAL' \
              'API_READY_TIMEOUT:BOXWEAVE_API_READY_TIMEOUT' 'JWT_SECRET_FILE:BOXWEAVE_JWT_SECRET_FILE' \
              'CONTAINARIUMD_BINARY:BOXWEAVE_CONTAINARIUMD_BINARY' 'BOXWEAVE_REPO:BOXWEAVE_REPO' 'BOXWEAVE_REF:BOXWEAVE_REF'; do
  setting=${option%%:*}; variable=${option#*:}
  [[ ! -v "$variable" ]] || EXPLICIT[$setting]=1
done
# REPO/REF already have defaults above; distinguish their original environment.
unset 'EXPLICIT[BOXWEAVE_REPO]' 'EXPLICIT[BOXWEAVE_REF]'
[[ -z "$REPO_FROM_ENV" ]] || EXPLICIT[BOXWEAVE_REPO]=1
[[ -z "$REF_FROM_ENV" ]] || EXPLICIT[BOXWEAVE_REF]=1
[[ -z "$MCP_FROM_ENV" ]] || EXPLICIT[CONTAINARIUM_VERSION]=1
cleanup() {
  [[ -z "$TMP_INSTALL" ]] || rm -f -- "$TMP_INSTALL"
  [[ -z "$TMPDIR_DL" ]] || rm -rf -- "$TMPDIR_DL"
  [[ -z "$STAGING_FILE" ]] || rm -f -- "$STAGING_FILE"
  if (( TIMERS_PAUSED )); then
    if systemctl stop boxweave-token.timer boxweave-hostkeys.timer >/dev/null 2>&1; then
      printf '[boxweave] Managed timers remain stopped after an interrupted update. Rerun this installer to finish repair. Backup: %s\n' "$BACKUP_DIR" >&2
    else
      printf '[boxweave] Update incomplete; could not stop every managed timer. Inspect their state before retrying. Backup: %s\n' "$BACKUP_DIR" >&2
    fi
  fi
}
report_failure() {
  local code=$?
  printf '[boxweave] FAILED in phase "%s" (exit %s). The installer can be rerun after the cause is fixed.\n' "$INSTALL_PHASE" "$code" >&2
  exit "$code"
}
trap cleanup EXIT
trap report_failure ERR

log() { printf '\n[boxweave] %s\n' "$*"; }
die() { printf '[boxweave] ERROR: %s\n' "$*" >&2; exit 1; }
usage() {
  cat <<EOF
Usage: curl -fsSL ${RAW_BASE}/install.sh | sudo bash
       sudo bash install.sh [--user USER] [--ssh-host HOST] [--ssh-port PORT]
                            [--allow-cidr CIDR] [--api-url URL] [--sync-interval SECONDS]
                            [--api-ready-timeout SECONDS]
                            [--web-mode https|http] [--web-host PRIVATE_IPV4]
                            [--web-port 18443] [--web-backend-port 18444]
                            [--source-dir REVIEWED_CHECKOUT]
                            [--repo OWNER/REPO] [--ref REF]
                            [--jwt-secret-file PATH] [--containariumd-binary PATH]
Environment: BOXWEAVE_LINUX_USER, BOXWEAVE_SSH_HOST, BOXWEAVE_SSH_PORT,
             BOXWEAVE_ALLOWED_CIDRS, BOXWEAVE_API_URL, BOXWEAVE_SYNC_INTERVAL,
             BOXWEAVE_JWT_SECRET_FILE, BOXWEAVE_CONTAINARIUMD_BINARY,
             BOXWEAVE_API_READY_TIMEOUT,
             CONTAINARIUM_VERSION, BOXWEAVE_REPO, BOXWEAVE_REF.
Rerun to update/repair BoxWeave. Saved settings are preserved unless explicitly
overridden. Existing Containarium/Incus packages are reused, not upgraded.
EOF
}
while (($#)); do
  case "$1" in
    --user) LINUX_USER="${2:?}"; EXPLICIT[LINUX_USER]=1; shift 2;;
    --ssh-host) SSH_HOST="${2:?}"; EXPLICIT[SSH_HOST]=1; shift 2;;
    --ssh-port) SSH_PORT="${2:?}"; EXPLICIT[SSH_PORT]=1; shift 2;;
    --allow-cidr) ALLOWED_CIDRS="${2:?}"; EXPLICIT[ALLOWED_CIDRS]=1; shift 2;;
    --api-url) API_URL="${2:?}"; EXPLICIT[API_URL]=1; shift 2;;
    --jwt-secret-file) JWT_SECRET_FILE="${2:?}"; EXPLICIT[JWT_SECRET_FILE]=1; shift 2;;
    --containariumd-binary) CONTAINARIUMD_BINARY="${2:?}"; EXPLICIT[CONTAINARIUMD_BINARY]=1; shift 2;;
    --sync-interval) SYNC_INTERVAL="${2:?}"; EXPLICIT[SYNC_INTERVAL]=1; shift 2;;
    --api-ready-timeout) API_READY_TIMEOUT="${2:?}"; EXPLICIT[API_READY_TIMEOUT]=1; shift 2;;
    --repo) BOXWEAVE_REPO="${2:?}"; EXPLICIT[BOXWEAVE_REPO]=1; shift 2;;
    --ref) BOXWEAVE_REF="${2:?}"; EXPLICIT[BOXWEAVE_REF]=1; shift 2;;
    --source-dir) LOCAL_SOURCE_DIR="${2:?}"; shift 2;;
    --web-mode) WEB_MODE="${2:?}"; shift 2;;
    --web-host) WEB_HOST="${2:?}"; shift 2;;
    --web-port) WEB_PORT="${2:?}"; shift 2;;
    --web-backend-port) WEB_BACKEND_PORT="${2:?}"; shift 2;;
    --help|-h) usage; exit 0;;
    *) die "Unknown argument: $1";;
  esac
done
RAW_BASE="https://raw.githubusercontent.com/${BOXWEAVE_REPO}/${BOXWEAVE_REF}"
[[ $EUID -eq 0 ]] || die 'This installer requires sudo/root.'
command -v flock >/dev/null || die 'flock (util-linux) is required'
install -d -o root -g root -m 0700 /run/boxweave-install
exec 9>/run/boxweave-install/installer.lock
flock -n 9 || die 'Another BoxWeave installation/update is running'

# Existing settings win over automatic defaults; explicit flags/environment win
# over saved settings. Never execute shell text from the JSON configuration.
load_existing_settings() {
  [[ -e "$CONFIG_DIR/config.json" ]] || return 0
  command -v python3 >/dev/null || die 'python3 is needed to preserve the existing configuration'
  STAGING_FILE=$(mktemp)
  python3 - "$CONFIG_DIR/config.json" >"$STAGING_FILE" <<'PY_SETTINGS'
import json, pathlib, sys
try:
    data = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding='utf-8'))
    if not isinstance(data, dict) or data.get('provider') != 'containarium' or data.get('schema_version') != 1:
        raise ValueError('Unsupported configuration')
    mapping = {'linux_user': 'LINUX_USER', 'api_url': 'API_URL', 'jwt_secret_file': 'JWT_SECRET_FILE',
               'containarium_daemon_binary': 'CONTAINARIUMD_BINARY', 'ssh_host': 'SSH_HOST',
               'ssh_port': 'SSH_PORT', 'sync_interval': 'SYNC_INTERVAL', 'api_ready_timeout': 'API_READY_TIMEOUT',
               'source_repo': 'BOXWEAVE_REPO', 'source_ref': 'BOXWEAVE_REF',
               'mcp_release': 'CONTAINARIUM_VERSION', 'container_public_key_file': 'SAVED_BOX_PUB'}
    for key, name in mapping.items():
        if key in data:
            value = data[key]
            if type(value) not in (str, int) or '\0' in str(value):
                raise ValueError('Invalid setting')
            sys.stdout.buffer.write((name + '\0' + str(value) + '\0').encode())
    if 'allowed_container_cidrs' in data:
        value = data['allowed_container_cidrs']
        if not isinstance(value, list) or not all(isinstance(x, str) and '\0' not in x for x in value):
            raise ValueError('Invalid CIDRs')
        sys.stdout.buffer.write(('ALLOWED_CIDRS\0' + ','.join(value) + '\0').encode())
except (OSError, ValueError, TypeError):
    sys.exit('Existing BoxWeave configuration is invalid or unsupported; it has not been replaced.')
PY_SETTINGS
  local key value saved_sync=0
  while IFS= read -r -d '' key && IFS= read -r -d '' value; do
    [[ "$key" != SYNC_INTERVAL ]] || saved_sync=1
    [[ -v "EXPLICIT[$key]" ]] || printf -v "$key" '%s' "$value"
  done <"$STAGING_FILE"
  rm -f "$STAGING_FILE"; STAGING_FILE=''
  # Earlier installers stored the interval only in the timer unit.
  if [[ ! -v 'EXPLICIT[SYNC_INTERVAL]' ]] && (( ! saved_sync )); then
    value=$(sed -nE 's/^OnUnitInactiveSec=([0-9]+)s$/\1/p' /etc/systemd/system/boxweave-hostkeys.timer 2>/dev/null || true)
    [[ -z "$value" ]] || SYNC_INTERVAL="$value"
  fi
}
load_existing_settings
if [[ -f "$CONFIG_DIR/update-in-progress" ]]; then
  log 'An earlier managed-file update did not finish; rerunning will repair and verify it.'
fi
RAW_BASE="https://raw.githubusercontent.com/${BOXWEAVE_REPO}/${BOXWEAVE_REF}"
[[ "$BOXWEAVE_REPO" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die 'Invalid repository name'
[[ "$BOXWEAVE_REF" =~ ^[A-Za-z0-9_.-]+$ ]] || die 'Invalid repository ref'
[[ -n "$LINUX_USER" ]] || die 'Cannot detect the non-root target user. Rerun with --user USER.'
[[ "$LINUX_USER" =~ ^[a-z_][a-z0-9_-]*$ && "$LINUX_USER" != root ]] || die 'Invalid non-root username'
getent passwd "$LINUX_USER" >/dev/null || die "User $LINUX_USER not found"
TARGET_HOME=$(getent passwd "$LINUX_USER" | cut -d: -f6)
TARGET_UID=$(id -u "$LINUX_USER")
TARGET_GID=$(id -g "$LINUX_USER")
[[ "$TARGET_HOME" =~ ^/[A-Za-z0-9_./-]+$ ]] || die 'Unsupported home path characters'
[[ -f /etc/os-release ]] || die 'Only Ubuntu Linux is supported'
. /etc/os-release
[[ "${ID:-}" == ubuntu ]] || die 'Only Ubuntu is supported by this installer'
command -v systemctl >/dev/null || die 'systemd is required'
command -v apt-get >/dev/null || die 'apt is required'
if command -v systemd-detect-virt >/dev/null; then
  container_type=$(systemd-detect-virt --container 2>/dev/null || true)
  if [[ -n "$container_type" && "$container_type" != none ]]; then
    log "Container environment: $container_type. Existing Incus is reused; BoxWeave does not require an Incus upgrade just for nesting."
  fi
fi

[[ "$API_READY_TIMEOUT" =~ ^[0-9]+$ ]] && ((API_READY_TIMEOUT >= 30 && API_READY_TIMEOUT <= 3600)) || die 'Invalid API ready timeout (30–3600 seconds)'

log "Configuring Ubuntu user $LINUX_USER ($TARGET_HOME)"
export DEBIAN_FRONTEND=noninteractive
if ! command -v curl >/dev/null || ! command -v ssh-keygen >/dev/null || ! command -v python3 >/dev/null || ! command -v runuser >/dev/null; then
  apt-get update -qq
  apt-get install -y -qq curl ca-certificates openssh-client python3 util-linux
fi
[[ "$SYNC_INTERVAL" =~ ^[0-9]+$ ]] && ((SYNC_INTERVAL >= 5 && SYNC_INTERVAL <= 3600)) || die 'Invalid sync interval (5–3600 seconds)'
python3 - "$API_URL" "$ALLOWED_CIDRS" <<'PY_PREFLIGHT'
import ipaddress, sys
from urllib.parse import urlsplit
try:
    u = urlsplit(sys.argv[1])
    if (u.scheme not in ('http', 'https') or not u.hostname or u.username or u.password
            or u.path not in ('', '/') or u.query or u.fragment
            or (u.scheme == 'http' and u.hostname not in ('localhost', '127.0.0.1', '::1'))):
        raise ValueError('Invalid API origin')
    u.port  # Validate the port even when not explicitly supplied.
    for value in filter(None, (x.strip() for x in sys.argv[2].split(','))):
        net = ipaddress.ip_network(value, strict=False)
        if net.version != 4 or not net.is_private or net.is_loopback or net.is_link_local:
            raise ValueError('Invalid CIDR')
except ValueError:
    sys.exit('Invalid API URL or allowed CIDRs; installed BoxWeave files have not been changed.')
PY_PREFLIGHT

# Recover permissions from an earlier interrupted installation without
# disabling APT signatures or trusting an unknown replacement key.
repair_zabbly_apt_permissions() {
  local keyring=/usr/share/keyrings/zabbly-incus.gpg
  local sourcelist=/etc/apt/sources.list.d/zabbly-incus-stable.list
  local preferences=/etc/apt/preferences.d/zabbly-incus
  if [[ -f "$keyring" ]]; then
    local fingerprint
    fingerprint=$(gpg --show-keys --with-colons "$keyring" 2>/dev/null | awk -F: '$1=="fpr" {print $10; exit}' || true)
    [[ "$fingerprint" == '4EFC590696CB15B87C73A3AD82CC8797C838DCFD' ]] ||       die "Unexpected Zabbly APT signing key at $keyring; inspect it manually"
    chmod 0644 "$keyring"
  fi
  if [[ -f "$sourcelist" ]] && grep -Fq 'https://pkgs.zabbly.com/incus/stable' "$sourcelist"; then
    chmod 0644 "$sourcelist"
  fi
  if [[ -f "$preferences" ]] && grep -Fq 'Pin: origin pkgs.zabbly.com' "$preferences"; then
    chmod 0644 "$preferences"
  fi
}

# A sufficiently recent Incus binary may be present without a storage pool.
# The upstream installer treats this as installed and may skip initialization.
# Only initialize on a host with zero existing containers and no pool.
ensure_incus_initialized() {
  command -v incus >/dev/null || die 'Incus executable not found'
  incus --force-local info >/dev/null || die 'Local Incus daemon is not reachable'
  local pools instances
  pools=$(incus --force-local storage list --format json) || die 'Could not list Incus storage pools'
  if ! python3 -c 'import json,sys; sys.exit(0 if json.load(sys.stdin) else 1)' <<< "$pools"; then
    instances=$(incus --force-local list --all-projects --format json) || die 'Could not list Incus containers across projects'
    python3 -c 'import json,sys; sys.exit(0 if not json.load(sys.stdin) else 1)' <<< "$instances" ||       die 'Incus has containers but no storage pool; manual review required'
    log 'Incus has no storage pool; initializing default network and storage on this empty host'
    incus --force-local admin init --auto || die 'Incus default initialization failed'
  fi
  pools=$(incus --force-local storage list --format json) || die 'Could not recheck Incus storage pools'
  python3 -c 'import json,sys; sys.exit(0 if json.load(sys.stdin) else 1)' <<< "$pools" ||     die 'Incus has no storage pool after initialization'
  if incus --force-local storage list --format json | python3 -c 'import json,sys; sys.exit(0 if any(p.get("driver") == "dir" for p in json.load(sys.stdin)) else 1)'; then
    log 'WARNING: Incus uses the dir storage driver. ZFS or btrfs is recommended for stronger per-container storage isolation.'
  fi
}

# Refuse to run competing legacy synchronizers by default. This installation
# never removes old services or credentials without an explicit migration step.
for old_timer in containarium-hostkey-sync.timer containarium-token-rotate.timer; do
  if systemctl is-active --quiet "$old_timer"; then
    die "Legacy timer $old_timer is still active. Stop legacy timers before switching to BoxWeave; see docs/MIGRATION.md"
  fi
done

INSTALL_PHASE='MCP download and checksum verification'
log 'Staging all BoxWeave components and the checksum-verified MCP backend before replacing installed files'
MCP_RELEASE="$CONTAINARIUM_VERSION"
ARCH=$(dpkg --print-architecture)
export ARCH
case "$ARCH" in amd64|arm64) ;; *) die "Unsupported CPU architecture: $ARCH";; esac
if [[ "$CONTAINARIUM_VERSION" == latest ]]; then
  release_json=$(curl -fsSL --retry 3 'https://api.github.com/repos/footprintai/containarium/releases/latest')
  CONTAINARIUM_VERSION=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])' <<< "$release_json")
fi
[[ "$CONTAINARIUM_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die 'Unsupported Containarium release tag'
TMPDIR_DL=$(mktemp -d)
# Resolve a mutable ref once, so all fetched components belong to one commit.
if [[ "$BOXWEAVE_REF" =~ ^[a-f0-9]{40}$ ]]; then
  SOURCE_COMMIT="$BOXWEAVE_REF"
else
  commit_json=$(curl -fsSL --retry 3 "https://api.github.com/repos/$BOXWEAVE_REPO/commits/$BOXWEAVE_REF")
  SOURCE_COMMIT=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["sha"])' <<< "$commit_json")
fi
[[ "$SOURCE_COMMIT" =~ ^[a-f0-9]{40}$ ]] || die 'Could not resolve a source commit'
RAW_BASE="https://raw.githubusercontent.com/${BOXWEAVE_REPO}/${SOURCE_COMMIT}"
curl -fsSL --retry 3 "$RAW_BASE/install.sh" -o "$TMPDIR_DL/reference-install.sh"
stage_source() {
  local relative=$1
  if [[ -n "$LOCAL_SOURCE_DIR" ]]; then
    python3 - "$LOCAL_SOURCE_DIR" "$relative" "$TMPDIR_DL/$relative" <<'PY_LOCAL_SOURCE'
from pathlib import Path
import shutil, sys
root=Path(sys.argv[1]).resolve(strict=True)
p=root/sys.argv[2]
if not root.is_dir() or p.is_symlink() or not p.resolve(strict=True).is_relative_to(root):
    sys.exit('Invalid local source file')
shutil.copyfile(p,sys.argv[3])
PY_LOCAL_SOURCE
  else
    curl -fsSL --retry 3 "$RAW_BASE/$relative" -o "$TMPDIR_DL/$relative"
  fi
}
export BOXWEAVE_LOCAL_SOURCE_STAGED="${LOCAL_SOURCE_DIR:+true}"
for source_file in server/cli.py server/hostkeys.py server/tokens.py server/jobs.py server/mcp_client.py server/smoke.py server/provenance.py server/clients.py clients/windows/setup.ps1 clients/posix/setup.py VERSION server/web_install.py server/web_token.py server/web_entry.py server/webui/__init__.py server/webui/store.py server/webui/tenants.py server/webui/access.py server/webui/graph.py server/webui/provider.py server/webui/app.py server/webui/command.py server/webui/requirements.txt server/webui/requirements.lock web/package.json web/package-lock.json web/tsconfig.json web/vite.config.ts web/index.html web/src/main.tsx web/src/tenants.tsx web/src/access.tsx web/src/model.ts web/src/style.css; do
  mkdir -p "$TMPDIR_DL/$(dirname "$source_file")"
  stage_source "$source_file"
done
python3 - "$TMPDIR_DL" <<'PY_SOURCE'
import ast, pathlib, re, sys
p = pathlib.Path(sys.argv[1])
for name in ('cli.py', 'hostkeys.py', 'tokens.py', 'jobs.py', 'mcp_client.py', 'smoke.py', 'provenance.py', 'clients.py'):
    ast.parse((p / 'server' / name).read_text(encoding='utf-8'))
ast.parse((p / 'clients/posix/setup.py').read_text(encoding='utf-8'))
for source in (p / 'server').rglob('*.py'): ast.parse(source.read_text(encoding='utf-8'))
if not re.fullmatch(r'[0-9A-Za-z.+-]{1,64}', (p / 'VERSION').read_text().strip()):
    sys.exit('Invalid BoxWeave version file')
PY_SOURCE
RELEASE_URL="https://github.com/footprintai/containarium/releases/download/${CONTAINARIUM_VERSION}"
curl -fsSL --retry 3 "$RELEASE_URL/SHA256SUMS.txt" -o "$TMPDIR_DL/SHA256SUMS.txt"
curl -fsSL --retry 3 "$RELEASE_URL/mcp-server-linux-${ARCH}" -o "$TMPDIR_DL/mcp-server-linux-${ARCH}"
(cd "$TMPDIR_DL" && grep -E "^[a-fA-F0-9]{64}[[:space:]]+\\*?mcp-server-linux-${ARCH}$" SHA256SUMS.txt | sha256sum -c -) \
  || die 'Release binary checksum verification failed'

INSTALL_PHASE='provider installation'
repair_zabbly_apt_permissions
if ! command -v containarium >/dev/null || ! command -v incus >/dev/null; then
  log 'Containarium missing; downloading official installer from containarium.dev'
  TMP_INSTALL=$(mktemp)
  curl -fsSL --retry 3 https://containarium.dev/install.sh -o "$TMP_INSTALL"
  # Containarium installs APT keyrings and source lists. APT's _apt user must
  # read these files; do not propagate BoxWeave's private-file umask 077.
  (umask 022; bash "$TMP_INSTALL")
  rm -f "$TMP_INSTALL"; TMP_INSTALL=''
  repair_zabbly_apt_permissions
fi
command -v incus >/dev/null || die 'Incus not found after Containarium installation'
command -v containarium >/dev/null || die 'Containarium CLI not found after installation'
INSTALL_PHASE='Incus validation'
ensure_incus_initialized
if [[ -z "$CONTAINARIUMD_BINARY" ]]; then
  CONTAINARIUMD_BINARY=$(command -v containariumd || true)
fi
[[ -n "$CONTAINARIUMD_BINARY" && -x "$CONTAINARIUMD_BINARY" ]] || die 'containariumd executable not found; use --containariumd-binary PATH' 
[[ "$CONTAINARIUMD_BINARY" == /* && "$JWT_SECRET_FILE" == /* ]] || die 'Provider binary and JWT secret paths must be absolute'
[[ -f "$JWT_SECRET_FILE" ]] || die "Provider JWT secret not found: $JWT_SECRET_FILE"

INSTALL_PHASE='SSH transport preparation'
# Windows SSH requires an OpenSSH server on Ubuntu. Preserve any existing sshd
# configuration; only install the distribution package when absent.
if [[ ! -x /usr/sbin/sshd ]]; then
  log 'Installing Ubuntu OpenSSH server for the Windows-to-MCP transport'
  apt-get update -qq
  apt-get install -y -qq openssh-server
fi
if [[ ! -f /etc/ssh/ssh_host_ed25519_key.pub ]]; then
  /usr/bin/ssh-keygen -A
fi
/usr/sbin/sshd -t || die 'OpenSSH server configuration is invalid'
systemctl enable --now ssh.service >/dev/null || die 'Unable to start ssh.service'

# Detect target SSH address independently of Containarium/LXC bridge.
if [[ -z "$SSH_HOST" ]]; then
  SSH_HOST=$(ip -4 route get 1.1.1.1 2>/dev/null | sed -nE 's/.* src ([0-9.]+).*/\1/p' | head -n1 || true)
  if [[ -z "$SSH_HOST" ]]; then
    SSH_HOST=$(hostname -I | awk '{print $1}' || true)
  fi
fi
[[ -n "$SSH_HOST" && "$SSH_HOST" =~ ^[a-zA-Z0-9.:-]+$ ]] || die 'Could not detect SSH host; rerun with --ssh-host HOST'
if [[ -z "$SSH_PORT" ]]; then
  if command -v sshd >/dev/null; then
    SSH_PORT=$(sshd -T 2>/dev/null | awk '$1=="port" {print $2; exit}' || true)
  elif [[ -x /usr/sbin/sshd ]]; then
    SSH_PORT=$(/usr/sbin/sshd -T 2>/dev/null | awk '$1=="port" {print $2; exit}' || true)
  fi
  SSH_PORT=${SSH_PORT:-22}
fi
[[ "$SSH_PORT" =~ ^[0-9]+$ ]] && ((SSH_PORT >= 1 && SSH_PORT <= 65535)) || die 'Invalid SSH port'
[[ "$SYNC_INTERVAL" =~ ^[0-9]+$ ]] && ((SYNC_INTERVAL >= 5 && SYNC_INTERVAL <= 3600)) || die 'Invalid sync interval (5–3600 seconds)'

INSTALL_PHASE='provider SSH identity preparation'
# Manage a host-user SSH key that the Containarium provider's LocateOrGenerate sees.
# Do not silently replace personal keys. Priority: id_ed25519, id_rsa, containarium_ed25519.
SSH_DIR="$TARGET_HOME/.ssh"
install -d -o "$TARGET_UID" -g "$TARGET_GID" -m 0700 "$SSH_DIR"
BOX_PUB=""
prefixes=(id_ed25519 id_rsa containarium_ed25519)
if [[ -n "$SAVED_BOX_PUB" ]]; then
  if [[ "$SAVED_BOX_PUB" == "$SSH_DIR/id_ed25519.pub" || "$SAVED_BOX_PUB" == "$SSH_DIR/id_rsa.pub" || "$SAVED_BOX_PUB" == "$SSH_DIR/containarium_ed25519.pub" ]]; then
    prefixes=("$(basename "${SAVED_BOX_PUB%.pub}")")
    [[ -f "$SAVED_BOX_PUB" && -f "${SAVED_BOX_PUB%.pub}" ]] || die 'Saved container SSH key pair is incomplete; repair it instead of replacing its identity'
  elif [[ ! -v 'EXPLICIT[LINUX_USER]' ]]; then
    die 'Saved container SSH key path does not match the target user'
  fi
fi
for prefix in "${prefixes[@]}"; do
  if [[ -e "$SSH_DIR/$prefix" || -e "$SSH_DIR/$prefix.pub" ]]; then
    [[ -f "$SSH_DIR/$prefix.pub" && -f "$SSH_DIR/$prefix" ]] || die "Incomplete existing SSH key pair: $SSH_DIR/$prefix"
    BOX_PUB="$SSH_DIR/$prefix.pub"; break
  fi
done
if [[ -z "$BOX_PUB" ]]; then
  for prefix in id_ed25519 id_rsa containarium_ed25519; do
    [[ ! -e "$SSH_DIR/$prefix" && ! -e "$SSH_DIR/$prefix.pub" ]] || die 'Incomplete existing SSH key pair; repair it before continuing'
  done
  log 'Generating an Ubuntu-to-container keypair as the target user'
  runuser -u "$LINUX_USER" -- ssh-keygen -q -t ed25519 \
    -f "$SSH_DIR/containarium_ed25519" -N '' -C 'containarium-mcp'
  BOX_PUB="$SSH_DIR/containarium_ed25519.pub"
fi
ssh-keygen -lf "$BOX_PUB" -E sha256 >/dev/null || die "Invalid SSH public key: $BOX_PUB"
PRIVATE_BOX_KEY="${BOX_PUB%.pub}"
runuser -u "$LINUX_USER" -- ssh-keygen -y -P "" -f "$PRIVATE_BOX_KEY" >/dev/null 2>&1 || die "MCP key cannot be opened without a passphrase: $PRIVATE_BOX_KEY"

# Core containers may take several minutes to download and start for the
# first time. Do not mistake their initialization delay for a failed daemon.
INSTALL_PHASE='Containarium REST API readiness'
systemctl enable --now containarium.service >/dev/null || die 'Could not enable/start containarium.service'
log "Waiting up to ${API_READY_TIMEOUT}s for Containarium REST API at $API_URL"
healthy=0
start_time=$SECONDS
last_report=$SECONDS
while (( SECONDS - start_time < API_READY_TIMEOUT )); do
  remaining=$((API_READY_TIMEOUT - (SECONDS - start_time)))
  probe_timeout=$((remaining < 4 ? remaining : 4))
  status=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time "$probe_timeout" "${API_URL%/}/v1/containers" || true)
  if [[ "$status" == 200 || "$status" == 401 || "$status" == 403 ]]; then
    healthy=1
    break
  fi
  remaining=$((API_READY_TIMEOUT - (SECONDS - start_time)))
  (( remaining > 0 )) || break
  probe_timeout=$((remaining < 5 ? remaining : 5))
  daemon_state=$(timeout "${probe_timeout}s" systemctl show containarium.service -p ActiveState --value) || die 'Could not query Containarium service state'
  if [[ "$daemon_state" != active && "$daemon_state" != activating && "$daemon_state" != reloading ]]; then
    die "Containarium service is ${daemon_state:-unknown} during first boot. Review: sudo journalctl -u containarium -n 80 --no-pager"
  fi
  if (( SECONDS - last_report >= 30 )); then
    log "Containarium is running; REST API not ready yet ($((SECONDS - start_time))s elapsed, last HTTP status: ${status:-unavailable}). Core containers may still be initializing."
    last_report=$SECONDS
  fi
  remaining=$((API_READY_TIMEOUT - (SECONDS - start_time)))
  (( remaining > 0 )) || break
  sleep "$((remaining < 5 ? remaining : 5))"
done
(( healthy == 1 )) || die "Containarium REST API did not become ready within ${API_READY_TIMEOUT}s. Check 'sudo incus list' and 'sudo journalctl -u containarium -n 80 --no-pager' before rerunning."
log "Containarium REST API responded with HTTP $status"

INSTALL_PHASE='mandatory web staging and production build'
web_options=()
[[ -z "$WEB_MODE" ]] || web_options+=(--mode "$WEB_MODE")
[[ -z "$WEB_HOST" ]] || web_options+=(--host "$WEB_HOST")
[[ -z "$WEB_PORT" ]] || web_options+=(--port "$WEB_PORT")
[[ -z "$WEB_BACKEND_PORT" ]] || web_options+=(--backend-port "$WEB_BACKEND_PORT")
python3 "$TMPDIR_DL/server/web_install.py" prepare "$TMPDIR_DL" "${web_options[@]}"

INSTALL_PHASE='BoxWeave configuration staging'

install -d -o root -g root -m 0755 "$CONFIG_DIR" "$CONFIG_DIR/providers"
# JSON configuration avoids substituting untrusted paths into root-executed services.
export BOXWEAVE_CONFIG_USER="$LINUX_USER" BOXWEAVE_CONFIG_ALLOW="$ALLOWED_CIDRS" BOXWEAVE_CONFIG_API="$API_URL"
export BOXWEAVE_CONFIG_SECRET="$JWT_SECRET_FILE" BOXWEAVE_CONFIG_BINARY="$CONTAINARIUMD_BINARY"
export BOXWEAVE_CONFIG_HOST="$SSH_HOST" BOXWEAVE_CONFIG_PORT="$SSH_PORT" BOXWEAVE_CONFIG_SYNC="$SYNC_INTERVAL"
export BOXWEAVE_CONFIG_READY="$API_READY_TIMEOUT" BOXWEAVE_CONFIG_REPO="$BOXWEAVE_REPO" BOXWEAVE_CONFIG_REF="$BOXWEAVE_REF"
export BOXWEAVE_CONFIG_MCP="$CONTAINARIUM_VERSION"
export BOXWEAVE_CONFIG_RELEASE="$MCP_RELEASE" BOXWEAVE_CONFIG_PUBLIC_KEY="$BOX_PUB"
python3 - "$CONFIG_DIR/config.json" "$TMPDIR_DL/config.json" <<'PY'
import json, os, pathlib, ipaddress, sys
old = pathlib.Path(sys.argv[1])
out = pathlib.Path(sys.argv[2])
allow = [x.strip() for x in os.environ['BOXWEAVE_CONFIG_ALLOW'].split(',') if x.strip()]
for cidr in allow:
    net = ipaddress.ip_network(cidr, strict=False)
    if net.version != 4 or not net.is_private or net.is_loopback or net.is_link_local:
        sys.exit('Host-key synchronization requires private IPv4 CIDRs')
data = json.loads(old.read_text(encoding='utf-8')) if old.exists() else {}
data.setdefault('default_locale', 'en-US')
data.update({'schema_version': 1, 'provider': 'containarium', 'linux_user': os.environ['BOXWEAVE_CONFIG_USER'], 'allowed_container_cidrs': allow,
        'jwt_secret_file': os.environ['BOXWEAVE_CONFIG_SECRET'],
        'api_url': os.environ['BOXWEAVE_CONFIG_API'],
        'containarium_daemon_binary': os.environ['BOXWEAVE_CONFIG_BINARY'],
        'ssh_host': os.environ['BOXWEAVE_CONFIG_HOST'], 'ssh_port': int(os.environ['BOXWEAVE_CONFIG_PORT']),
        'sync_interval': int(os.environ['BOXWEAVE_CONFIG_SYNC']), 'api_ready_timeout': int(os.environ['BOXWEAVE_CONFIG_READY']),
        'source_repo': os.environ['BOXWEAVE_CONFIG_REPO'], 'source_ref': os.environ['BOXWEAVE_CONFIG_REF'],
        'mcp_version': os.environ['BOXWEAVE_CONFIG_MCP'], 'mcp_release': os.environ['BOXWEAVE_CONFIG_RELEASE'],
        'container_public_key_file': os.environ['BOXWEAVE_CONFIG_PUBLIC_KEY']})
out.write_text(json.dumps(data, indent=2) + '\n')
out.chmod(0o600)
PY

python3 "$TMPDIR_DL/server/clients.py" publish "$TMPDIR_DL/config.json" "$TMPDIR_DL/client.json" "$TMPDIR_DL" "$SOURCE_COMMIT" /etc/ssh/ssh_host_ed25519_key.pub

python3 "$TMPDIR_DL/server/provenance.py" build --directory "$TMPDIR_DL" \
  --installer "${BASH_SOURCE[0]:-/dev/stdin}" --repo "$BOXWEAVE_REPO" --ref "$BOXWEAVE_REF" \
  --commit "$SOURCE_COMMIT" --backend "$CONTAINARIUM_VERSION" --daemon "$CONTAINARIUMD_BINARY"

INSTALL_PHASE='BoxWeave service installation'
log 'Preparing BoxWeave launchers and service units'
cat >"$TMPDIR_DL/boxweave" <<'CLI'
#!/usr/bin/env bash
set -euo pipefail
exec /usr/bin/python3 /usr/local/libexec/boxweave/cli.py "$@"
CLI

# The SSH forced-command entry points only to this root-owned executable.
# No user-owned wrapper or symlink is used.
LAUNCHER=/usr/local/bin/boxweave-mcp
# Non-secret, root-owned environment file read by the launcher.
{
  printf 'CONTAINARIUM_SERVER_URL=%q\n' "$API_URL"
  if [[ "$API_URL" == http://* ]]; then
    echo 'CONTAINARIUM_MCP_ALLOW_INSECURE=true'
  fi
} > "$TMPDIR_DL/containarium.env"
chmod 0644 "$TMPDIR_DL/containarium.env"
cat > "$TMPDIR_DL/boxweave-mcp" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
set -a
source /etc/boxweave/providers/containarium.env
set +a
export CONTAINARIUM_JWT_TOKEN_FILE="$HOME/.config/boxweave/containarium-mcp.jwt"
exec /usr/local/libexec/boxweave/containarium-mcp-server
EOF
chmod 0755 "$TMPDIR_DL/boxweave-mcp"

cat > "$TMPDIR_DL/boxweave-hostkeys.service" <<'EOF'
[Unit]
Description=Verify and sync Containarium LXC SSH host keys
After=containarium.service
[Service]
Type=oneshot
User=root
ExecStart=/usr/local/sbin/boxweave-hostkeys --apply
TimeoutStartSec=180
UMask=0077
EOF
cat > "$TMPDIR_DL/boxweave-hostkeys.timer" <<'EOF'
[Unit]
Description=Periodic verified Containarium SSH host-key sync
[Timer]
OnBootSec=30s
OnUnitInactiveSec=__SYNC_INTERVAL__s
AccuracySec=1s
Unit=boxweave-hostkeys.service
[Install]
WantedBy=timers.target
EOF
sed -i "s/__SYNC_INTERVAL__/$SYNC_INTERVAL/" "$TMPDIR_DL/boxweave-hostkeys.timer"
cat > "$TMPDIR_DL/boxweave-token.service" <<'EOF'
[Unit]
Description=Rotate Containarium MCP JWT
After=containarium.service
[Service]
Type=oneshot
User=root
ExecStart=/usr/local/sbin/boxweave-token
TimeoutStartSec=60
UMask=0077
EOF
cat > "$TMPDIR_DL/boxweave-token.timer" <<'EOF'
[Unit]
Description=Rotate Containarium MCP JWT daily
[Timer]
OnBootSec=5min
OnUnitInactiveSec=1d
Persistent=true
Unit=boxweave-token.service
[Install]
WantedBy=timers.target
EOF

INSTALL_PHASE='BoxWeave service verification'
# Publish only complete, validated files. Renaming on the destination filesystem
# avoids truncating active MCP executables and SSH launchers (ETXTBSY).
atomic_install() {
  local source=$1 destination=$2 mode=$3
  [[ ! -L "$destination" ]] || die "Refusing managed-path symlink: $destination"
  install -d -o root -g root -m 0755 "$(dirname "$destination")"
  STAGING_FILE=$(mktemp "$(dirname "$destination")/.boxweave-update.XXXXXX")
  install -o root -g root -m "$mode" "$source" "$STAGING_FILE"
  mv -fT -- "$STAGING_FILE" "$destination"
  STAGING_FILE=''
}

publish_components() {
  local item source destination mode unit load
  local files=(
    'mcp-server-linux-ARCH:/usr/local/libexec/boxweave/containarium-mcp-server:0755'
    'server/cli.py:/usr/local/libexec/boxweave/cli.py:0644'
    'server/clients.py:/usr/local/libexec/boxweave/clients.py:0644'
    'client.json:/etc/boxweave/client.json:0644'
    'server/jobs.py:/usr/local/libexec/boxweave/jobs.py:0644'
    'server/mcp_client.py:/usr/local/libexec/boxweave/mcp_client.py:0644'
    'server/smoke.py:/usr/local/libexec/boxweave/smoke.py:0644'
    'server/provenance.py:/usr/local/libexec/boxweave/provenance.py:0644'
    'VERSION:/usr/local/libexec/boxweave/VERSION:0644'
    'boxweave:/usr/local/bin/boxweave:0755'
    'server/hostkeys.py:/usr/local/sbin/boxweave-hostkeys:0755'
    'server/tokens.py:/usr/local/sbin/boxweave-token:0755'
    'config.json:/etc/boxweave/config.json:0600'
    'install-manifest.json:/etc/boxweave/install-manifest.json:0600'
    'containarium.env:/etc/boxweave/providers/containarium.env:0644'
    'boxweave-mcp:/usr/local/bin/boxweave-mcp:0755'
    'boxweave-hostkeys.service:/etc/systemd/system/boxweave-hostkeys.service:0644'
    'boxweave-hostkeys.timer:/etc/systemd/system/boxweave-hostkeys.timer:0644'
    'boxweave-token.service:/etc/systemd/system/boxweave-token.service:0644'
    'boxweave-token.timer:/etc/systemd/system/boxweave-token.timer:0644'
  )
  install -d -o root -g root -m 0700 /var/backups/boxweave
  BACKUP_DIR=$(mktemp -d /var/backups/boxweave/update-XXXXXXXX)
  for item in "${files[@]}"; do
    destination=${item#*:}; destination=${destination%:*}
    [[ ! -L "$destination" && ( ! -e "$destination" || -f "$destination" ) ]] || die "Invalid managed path: $destination"
    if [[ -f "$destination" ]]; then
      mkdir -p "$BACKUP_DIR/files$(dirname "$destination")"
      cp -p -- "$destination" "$BACKUP_DIR/files$destination"
    else
      printf '%s\n' "$destination" >>"$BACKUP_DIR/previously-absent.txt"
    fi
  done
  log "Managed-file backup: $BACKUP_DIR (credentials and container data are not copied)"
  # Check all managed units before stopping any; preserve intentional masks.
  local units=(boxweave-hostkeys.timer boxweave-token.timer boxweave-hostkeys.service boxweave-token.service)
  for unit in "${units[@]}"; do
    load=$(systemctl show "$unit" -p LoadState --value || true)
    [[ "$load" == loaded || "$load" == not-found || "$load" == bad-setting || "$load" == error ]] || die "Managed unit $unit is ${load:-unknown}; inspect/unmask it before rerunning"
  done
  printf '%s\n' "$BACKUP_DIR" >"$CONFIG_DIR/update-in-progress"
  TIMERS_PAUSED=1
  for unit in "${units[@]}"; do
    load=$(systemctl show "$unit" -p LoadState --value || true)
    if [[ "$load" != not-found ]]; then
      if ! systemctl stop "$unit"; then
        # A broken unit can reject a stop job even when already inactive.
        load=$(systemctl show "$unit" -p ActiveState --value || true)
        [[ "$load" == inactive || "$load" == failed ]] || die "Could not quiesce $unit; no further files were published"
      fi
    fi
  done
  for item in "${files[@]}"; do
    source=${item%%:*}; source=${source/ARCH/$ARCH}
    destination=${item#*:}; mode=${destination##*:}; destination=${destination%:*}
    atomic_install "$TMPDIR_DL/$source" "$destination" "$mode"
  done
}
publish_components
# Token generation must pass before timers start. No token contents are printed.
systemctl daemon-reload
/usr/bin/python3 /usr/local/libexec/boxweave/cli.py --version
for unit in boxweave-token.service boxweave-hostkeys.service; do
  # Fresh oneshot units may be unloaded even after daemon-reload. reset-failed
  # fails for them; only clear an actual failed state before retrying a service.
  if systemctl is-failed --quiet "$unit"; then
    systemctl reset-failed "$unit"
  fi
done
systemctl start boxweave-token.service || die 'Cannot mint MCP JWT. Inspect journalctl -u boxweave-token.service'
systemctl start boxweave-hostkeys.service || die 'Host-key sync failed. Inspect journalctl -u boxweave-hostkeys.service'
systemctl enable boxweave-token.timer boxweave-hostkeys.timer >/dev/null
systemctl restart boxweave-token.timer boxweave-hostkeys.timer
systemctl is-active --quiet boxweave-token.timer boxweave-hostkeys.timer || die 'Managed timers did not become active'
TIMERS_PAUSED=0
INSTALL_PHASE='mandatory independent web installation'
python3 "$TMPDIR_DL/server/web_install.py" publish "$TMPDIR_DL"
/usr/bin/python3 /usr/local/libexec/boxweave/provenance.py complete "$CONFIG_DIR/install-manifest.json"
rm -f "$CONFIG_DIR/update-in-progress"
TIMERS_PAUSED=0

cat <<EOF

====================================================================
 BoxWeave server installation complete.

 To configure your coding agent, run:
   boxweave client

 For non-interactive usage and client options:
   boxweave client --help

 Choose Windows or macOS, then Codex, Cursor, Claude Code or ZCode.
 Run the generated setup command on the client computer.
 Existing clients normally need no reenrollment for an unchanged endpoint.
 Reconnect the agent's MCP session after a server update.
====================================================================
EOF
