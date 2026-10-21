#!/usr/bin/env bash
# BoxWeave server installer for Ubuntu. Run with `curl ... | sudo bash`.
set -Eeuo pipefail
umask 077

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
RAW_BASE="https://raw.githubusercontent.com/${BOXWEAVE_REPO}/${BOXWEAVE_REF}"
CONFIG_DIR=/etc/boxweave
BIN=/usr/local/sbin

log() { printf '\n[boxweave] %s\n' "$*"; }
die() { printf '[boxweave] ERROR: %s\n' "$*" >&2; exit 1; }
usage() {
  cat <<EOF
Usage: curl -fsSL ${RAW_BASE}/install.sh | sudo bash
       sudo bash install.sh [--user USER] [--ssh-host HOST] [--ssh-port PORT]
                            [--allow-cidr CIDR] [--api-url URL] [--sync-interval SECONDS]
                            [--repo OWNER/REPO] [--ref REF]
                            [--jwt-secret-file PATH] [--containariumd-binary PATH]
Environment: BOXWEAVE_LINUX_USER, BOXWEAVE_SSH_HOST, BOXWEAVE_SSH_PORT,
             BOXWEAVE_ALLOWED_CIDRS, BOXWEAVE_API_URL, BOXWEAVE_SYNC_INTERVAL,
             BOXWEAVE_JWT_SECRET_FILE, BOXWEAVE_CONTAINARIUMD_BINARY,
             CONTAINARIUM_VERSION, BOXWEAVE_REPO, BOXWEAVE_REF.
EOF
}
while (($#)); do
  case "$1" in
    --user) LINUX_USER="${2:?}"; shift 2;;
    --ssh-host) SSH_HOST="${2:?}"; shift 2;;
    --ssh-port) SSH_PORT="${2:?}"; shift 2;;
    --allow-cidr) ALLOWED_CIDRS="${2:?}"; shift 2;;
    --api-url) API_URL="${2:?}"; shift 2;;
    --jwt-secret-file) JWT_SECRET_FILE="${2:?}"; shift 2;;
    --containariumd-binary) CONTAINARIUMD_BINARY="${2:?}"; shift 2;;
    --sync-interval) SYNC_INTERVAL="${2:?}"; shift 2;;
    --repo) BOXWEAVE_REPO="${2:?}"; shift 2;;
    --ref) BOXWEAVE_REF="${2:?}"; shift 2;;
    --help|-h) usage; exit 0;;
    *) die "Unknown argument: $1";;
  esac
done
RAW_BASE="https://raw.githubusercontent.com/${BOXWEAVE_REPO}/${BOXWEAVE_REF}"
[[ $EUID -eq 0 ]] || die 'This installer requires sudo/root.'
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

log "Configuring Ubuntu user $LINUX_USER ($TARGET_HOME)"
export DEBIAN_FRONTEND=noninteractive
if ! command -v curl >/dev/null || ! command -v ssh-keygen >/dev/null || ! command -v python3 >/dev/null || ! command -v runuser >/dev/null; then
  apt-get update -qq
  apt-get install -y -qq curl ca-certificates openssh-client python3 util-linux
fi

if ! command -v containarium >/dev/null || ! command -v incus >/dev/null; then
  log 'Containarium missing; downloading official installer from containarium.dev'
  tmp_install=$(mktemp)
  trap 'rm -f "${tmp_install:-}"' EXIT
  curl -fsSL --retry 3 https://containarium.dev/install.sh -o "$tmp_install"
  bash "$tmp_install"
  rm -f "$tmp_install"; unset tmp_install
fi
command -v incus >/dev/null || die 'Incus not found after Containarium installation'
command -v containarium >/dev/null || die 'Containarium CLI not found after installation'
if [[ -z "$CONTAINARIUMD_BINARY" ]]; then
  CONTAINARIUMD_BINARY=$(command -v containariumd || true)
fi
[[ -n "$CONTAINARIUMD_BINARY" && -x "$CONTAINARIUMD_BINARY" ]] || die 'containariumd executable not found; use --containariumd-binary PATH' 
[[ "$CONTAINARIUMD_BINARY" == /* && "$JWT_SECRET_FILE" == /* ]] || die 'Provider binary and JWT secret paths must be absolute'
[[ -f "$JWT_SECRET_FILE" ]] || die "Provider JWT secret not found: $JWT_SECRET_FILE"

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
python3 - "$API_URL" <<'PY_VALIDATE'
import sys
from urllib.parse import urlparse
u = urlparse(sys.argv[1])
assert u.scheme in ('http', 'https') and u.hostname and not u.username and not u.password
assert not u.path or u.path == '/'
assert u.scheme != 'http' or u.hostname in ('localhost', '127.0.0.1', '::1'), 'Plaintext HTTP is allowed only for loopback'
PY_VALIDATE

# Manage a host-user SSH key that the Containarium provider's LocateOrGenerate sees.
# Do not silently replace personal keys. Priority: id_ed25519, id_rsa, containarium_ed25519.
SSH_DIR="$TARGET_HOME/.ssh"
install -d -o "$TARGET_UID" -g "$TARGET_GID" -m 0700 "$SSH_DIR"
BOX_PUB=""
for prefix in id_ed25519 id_rsa containarium_ed25519; do
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

# Ensure API service is enabled and locally reachable, without exposing an additional listener.
systemctl enable --now containarium.service >/dev/null || die 'Could not enable/start containarium.service'
log 'Checking local Containarium REST API'
healthy=0
for _ in $(seq 1 90); do
  status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 2 "${API_URL%/}/v1/containers" || true)
  if [[ "$status" == 200 || "$status" == 401 || "$status" == 403 ]]; then healthy=1; break; fi
  sleep 2
done
(( healthy == 1 )) || die "Containarium REST API unavailable on $API_URL"

log 'Installing Containarium platform MCP with upstream release checksum validation'
ARCH=$(dpkg --print-architecture)
case "$ARCH" in amd64|arm64) ;; *) die "Unsupported CPU architecture: $ARCH";; esac
if [[ "$CONTAINARIUM_VERSION" == latest ]]; then
  release_json=$(curl -fsSL --retry 3 'https://api.github.com/repos/footprintai/containarium/releases/latest')
  CONTAINARIUM_VERSION=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])' <<< "$release_json")
fi
[[ "$CONTAINARIUM_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die 'Unsupported Containarium release tag'
TMPDIR_DL=$(mktemp -d)
trap 'rm -rf "${TMPDIR_DL:-}"' EXIT
RELEASE_URL="https://github.com/footprintai/containarium/releases/download/${CONTAINARIUM_VERSION}"
curl -fsSL --retry 3 "$RELEASE_URL/SHA256SUMS.txt" -o "$TMPDIR_DL/SHA256SUMS.txt"
curl -fsSL --retry 3 "$RELEASE_URL/mcp-server-linux-${ARCH}" -o "$TMPDIR_DL/mcp-server-linux-${ARCH}"
(cd "$TMPDIR_DL" && grep -E "^[a-fA-F0-9]{64}[[:space:]]+\\*?mcp-server-linux-${ARCH}$" SHA256SUMS.txt | sha256sum -c -) \
  || die 'Release binary checksum verification failed'
install -d -o root -g root -m 0755 /usr/local/libexec/boxweave
install -o root -g root -m 0755 "$TMPDIR_DL/mcp-server-linux-${ARCH}" /usr/local/libexec/boxweave/containarium-mcp-server
rm -rf "$TMPDIR_DL"; unset TMPDIR_DL

install -d -o root -g root -m 0755 "$CONFIG_DIR" "$CONFIG_DIR/providers"
# JSON configuration avoids substituting untrusted paths into root-executed services.
export BOXWEAVE_CONFIG_USER="$LINUX_USER" BOXWEAVE_CONFIG_ALLOW="$ALLOWED_CIDRS" BOXWEAVE_CONFIG_API="$API_URL"
export BOXWEAVE_CONFIG_SECRET="$JWT_SECRET_FILE" BOXWEAVE_CONFIG_BINARY="$CONTAINARIUMD_BINARY"
python3 - <<'PY'
import json, os, pathlib
out = pathlib.Path('/etc/boxweave/config.json')
allow = [x.strip() for x in os.environ['BOXWEAVE_CONFIG_ALLOW'].split(',') if x.strip()]
data = {'schema_version': 1, 'provider': 'containarium', 'default_locale': 'en-US', 'linux_user': os.environ['BOXWEAVE_CONFIG_USER'], 'allowed_container_cidrs': allow,
        'jwt_secret_file': os.environ['BOXWEAVE_CONFIG_SECRET'],
        'api_url': os.environ['BOXWEAVE_CONFIG_API'],
        'containarium_daemon_binary': os.environ['BOXWEAVE_CONFIG_BINARY']}
out.write_text(json.dumps(data, indent=2) + '\n')
out.chmod(0o600)
PY

log 'Installing trusted SSH host-key sync and automatic token rotation'
for spec in 'server/hostkeys.py:boxweave-hostkeys' 'server/tokens.py:boxweave-token'; do
  source_file="${spec%%:*}"; target_file="${spec##*:}"
  staging=$(mktemp)
  curl -fsSL --retry 3 "$RAW_BASE/$source_file" -o "$staging"
  python3 -m py_compile "$staging" || die "Invalid Python source: $source_file"
  install -o root -g root -m 0755 "$staging" "$BIN/$target_file"
  rm -f "$staging"
done

# The SSH forced-command entry points only to this root-owned executable.
# No user-owned wrapper or symlink is used.
LAUNCHER=/usr/local/bin/boxweave-mcp
# Non-secret, root-owned environment file read by the launcher.
{
  printf 'CONTAINARIUM_SERVER_URL=%q\n' "$API_URL"
  if [[ "$API_URL" == http://* ]]; then
    echo 'CONTAINARIUM_MCP_ALLOW_INSECURE=true'
  fi
} > "$CONFIG_DIR/providers/containarium.env"
chmod 0644 "$CONFIG_DIR/providers/containarium.env"
cat > /usr/local/bin/boxweave-mcp <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
set -a
source /etc/boxweave/providers/containarium.env
set +a
export CONTAINARIUM_JWT_TOKEN_FILE="$HOME/.config/boxweave/containarium-mcp.jwt"
exec /usr/local/libexec/boxweave/containarium-mcp-server
EOF
chmod 0755 /usr/local/bin/boxweave-mcp

cat > /etc/systemd/system/boxweave-hostkeys.service <<'EOF'
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
cat > /etc/systemd/system/boxweave-hostkeys.timer <<'EOF'
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
sed -i "s/__SYNC_INTERVAL__/$SYNC_INTERVAL/" /etc/systemd/system/boxweave-hostkeys.timer
cat > /etc/systemd/system/boxweave-token.service <<'EOF'
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
cat > /etc/systemd/system/boxweave-token.timer <<'EOF'
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

# Refuse to run competing legacy synchronizers by default. This installation
# never removes old services or credentials without an explicit migration step.
for old_timer in containarium-hostkey-sync.timer containarium-token-rotate.timer; do
  if systemctl is-active --quiet "$old_timer"; then
    die "Legacy timer $old_timer is still active. Stop legacy timers before switching to BoxWeave; see docs/MIGRATION.md"
  fi
done

# Token generation must pass before timers start. No token contents are printed.
systemctl daemon-reload
systemctl start boxweave-token.service || die 'Cannot mint MCP JWT. Inspect journalctl -u boxweave-token.service'
systemctl start boxweave-hostkeys.service || die 'Host-key sync failed. Inspect journalctl -u boxweave-hostkeys.service'
systemctl enable --now boxweave-token.timer boxweave-hostkeys.timer >/dev/null

# A trusted snapshot of host's public ED25519 key, to be pinned on Windows.
HOSTKEY_PATH=/etc/ssh/ssh_host_ed25519_key.pub
[[ -f "$HOSTKEY_PATH" ]] || die 'Host ED25519 SSH key not present; ensure OpenSSH server is configured'
HOSTKEY_B64=$(awk '$1=="ssh-ed25519" {print $2; exit}' "$HOSTKEY_PATH")
[[ -n "$HOSTKEY_B64" ]] || die 'Invalid SSH host public key'
BOX_KEY_B64=$(base64 -w0 < "$BOX_PUB")
FINGERPRINT=$(ssh-keygen -lf "$HOSTKEY_PATH" -E sha256 | awk '{print $2}')

# Display a single copy/paste line. Only PUBLIC key data in command, never secrets.
cat <<EOF

================================================================================
 BoxWeave server installation completed.
 Ubuntu account:       ${LINUX_USER}
 Ubuntu SSH endpoint:  ${SSH_HOST}:${SSH_PORT}
 Server SSH ED25519 fingerprint: ${FINGERPRINT}
 Container public key: ${BOX_PUB}

 On your Windows laptop, run ONE of these PowerShell one-liners, under the
 same Windows account that runs your AI coding agent. No manual file editing.

 A) Native Windows OpenSSH (recommended):
 Invoke-WebRequest '${RAW_BASE}/clients/windows/setup.ps1' -OutFile "\$env:TEMP\\boxweave-setup.ps1"; powershell.exe -NoProfile -ExecutionPolicy Bypass -File "\$env:TEMP\\boxweave-setup.ps1" -SshHost '${SSH_HOST}' -SshPort ${SSH_PORT} -SshUser '${LINUX_USER}' -ServerHostKey '${HOSTKEY_B64}' -BoxPublicKeyBase64 '${BOX_KEY_B64}' -McpRemoteCommand '${LAUNCHER}' -Mode Native -Agent Codex -Provider Containarium

 B) WSL OpenSSH (replace DISTRO using wsl.exe -l -q):
 Invoke-WebRequest '${RAW_BASE}/clients/windows/setup.ps1' -OutFile "\$env:TEMP\\boxweave-setup.ps1"; powershell.exe -NoProfile -ExecutionPolicy Bypass -File "\$env:TEMP\\boxweave-setup.ps1" -SshHost '${SSH_HOST}' -SshPort ${SSH_PORT} -SshUser '${LINUX_USER}' -ServerHostKey '${HOSTKEY_B64}' -BoxPublicKeyBase64 '${BOX_KEY_B64}' -McpRemoteCommand '${LAUNCHER}' -Mode WSL -WslDistro 'DISTRO' -Agent Codex -Provider Containarium

 The bootstrap step requires an existing SSH login to ${LINUX_USER}@${SSH_HOST}.
 The new dedicated SSH key only starts MCP; it does not grant a host shell.
 Restart your coding agent after the selected client setup finishes.
================================================================================
EOF
