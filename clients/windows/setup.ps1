#requires -version 5.1
<#
Configure BoxWeave with the Containarium provider for a supported Windows coding agent.
Mode Native: ssh.exe runs on Windows.
Mode WSL:    wsl.exe launches OpenSSH inside the selected WSL distribution.
Both modes preserve the native Windows agent runtime; WSL is only the SSH transport.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$SshHost,
    [ValidateRange(1,65535)][int]$SshPort = 22,
    [Parameter(Mandatory=$true)][string]$SshUser,
    # Existing SSH login identity for key-only hosts; omit to use password login.
    [string]$BootstrapIdentityFile = '',
    [Parameter(Mandatory=$true)][string]$ServerHostKey,
    [Parameter(Mandatory=$true)][string]$BoxPublicKeyBase64,
    [Parameter(Mandatory=$true)][string]$McpRemoteCommand,
    [ValidateSet('Native','WSL')][string]$Mode = 'Native',
    [string]$WslDistro = '',
    [ValidateSet('Codex')][string]$Agent = 'Codex',
    [ValidateSet('Containarium')][string]$Provider = 'Containarium',
    [string]$AgentHome = '',
    [string]$McpName = 'boxweave',
    [string]$KeyName = 'boxweave'
)
$ErrorActionPreference = 'Stop'

function Assert-Exit([string]$What) {
    if ($LASTEXITCODE -ne 0) { throw "$What failed (exit code $LASTEXITCODE)" }
}
function Write-Utf8NoBom([string]$Path, [string]$Content) {
    [System.IO.File]::WriteAllText($Path, $Content, (New-Object System.Text.UTF8Encoding($false)))
}
function Backup-File([string]$Path) {
    if (Test-Path -LiteralPath $Path) {
        $stamp = Get-Date -Format 'yyyyMMdd-HHmmss-fff'
        Copy-Item -LiteralPath $Path -Destination "$Path.backup-$stamp" -ErrorAction Stop
    }
}
function Quote-Toml([string]$Value) {
    # JSON double-quoted escaping is a compatible TOML basic-string subset.
    return ConvertTo-Json -InputObject $Value -Compress
}
function Set-TomlSection([string]$Source, [string]$Header, [string]$Replacement) {
    $pattern = '(?ms)^\[' + [regex]::Escape($Header) + '\][^\r\n]*\r?\n.*?(?=^\[|\z)'
    $match = [regex]::Match($Source, $pattern)
    if ($match.Success) {
        return $Source.Substring(0, $match.Index) + $Replacement + "`r`n" + $Source.Substring($match.Index + $match.Length)
    }
    return $Source.TrimEnd() + "`r`n`r`n" + $Replacement + "`r`n"
}
function Update-CodexToml([string]$Path, [string]$Command, [string[]]$Arguments) {
    $text = if (Test-Path -LiteralPath $Path) { [System.IO.File]::ReadAllText($Path) } else { '' }
    $desktop = [regex]::Match($text, '(?ms)^\[desktop\][^\r\n]*\r?\n.*?(?=^\[|\z)')
    if ($desktop.Success) {
        $body = [regex]::Replace($desktop.Value, '(?m)^\s*runCodexInWindowsSubsystemForLinux\s*=\s*[^\r\n]*\r?\n?', '')
        $replacement = $body.TrimEnd() + "`r`nrunCodexInWindowsSubsystemForLinux = false`r`n"
        $text = $text.Substring(0, $desktop.Index) + $replacement + $text.Substring($desktop.Index + $desktop.Length)
    } else {
        $text = $text.TrimEnd() + "`r`n`r`n[desktop]`r`nrunCodexInWindowsSubsystemForLinux = false`r`n"
    }
    $encoded = ($Arguments | ForEach-Object { Quote-Toml $_ }) -join ', '
    $block = '[mcp_servers.' + $McpName + "]`r`ncommand = " + (Quote-Toml $Command) + "`r`nargs = [$encoded]`r`nstartup_timeout_sec = 30`r`n"
    $text = Set-TomlSection -Source $text -Header "mcp_servers.$McpName" -Replacement $block
    Backup-File $Path
    Write-Utf8NoBom $Path $text
}
function Update-Agents([string]$Path, [string]$Content) {
    $begin = '<!-- BEGIN BOXWEAVE MANAGED -->'
    $end = '<!-- END BOXWEAVE MANAGED -->'
    $managed = $begin + "`r`n" + $Content.Trim() + "`r`n" + $end
    $original = if (Test-Path -LiteralPath $Path) { [System.IO.File]::ReadAllText($Path) } else { '' }
    $pattern = '(?s)' + [regex]::Escape($begin) + '.*?' + [regex]::Escape($end)
    $match = [regex]::Match($original, $pattern)
    if ($match.Success) {
        $updated = $original.Substring(0, $match.Index) + $managed + $original.Substring($match.Index + $match.Length)
    } else {
        $updated = $original.TrimEnd() + "`r`n`r`n" + $managed + "`r`n"
    }
    Backup-File $Path
    Write-Utf8NoBom $Path $updated
}
function Set-PinnedHostNative([string]$Path, [string]$Line) {
    if (Test-Path -LiteralPath $Path) {
        if ([System.IO.File]::ReadAllText($Path).Trim() -ne $Line) {
            throw "Host public key changed! Verify the new Ubuntu fingerprint out-of-band before replacing $Path"
        }
    } else {
        Write-Utf8NoBom $Path ($Line + "`n")
    }
}

if ($SshHost -notmatch '^[A-Za-z0-9.:-]+$' -or $SshUser -notmatch '^[a-z_][a-z0-9_-]*$') {
    throw 'Invalid SSH host or Unix user.'
}
if ($McpName -notmatch '^[a-zA-Z0-9_-]+$' -or $KeyName -notmatch '^[a-zA-Z0-9_-]+$') {
    throw 'Invalid MCP name or SSH key filename.'
}
if ($McpRemoteCommand -notmatch '^/[a-zA-Z0-9_./-]+$') { throw 'Invalid remote MCP command path.' }
if ($ServerHostKey -notmatch '^[a-zA-Z0-9+/]+={0,2}$') { throw 'Invalid server SSH public key.' }
if ($Mode -eq 'WSL' -and $WslDistro -notmatch '^[a-zA-Z0-9_.-]+$') {
    throw 'WSL mode requires -WslDistro NAME (wsl.exe -l -q lists installed distributions).'
}
$boxPub = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($BoxPublicKeyBase64)).Trim()
if ($boxPub -notmatch '^(ssh-ed25519|ssh-rsa|ecdsa-sha2-[a-z0-9]+)\s+[a-zA-Z0-9+/=]+(?:\s+[^\r\n]*)?$') {
    throw 'Invalid Ubuntu-to-container SSH public key.'
}
$ssh = (Get-Command ssh.exe -ErrorAction Stop).Source
$sshKeygen = (Get-Command ssh-keygen.exe -ErrorAction Stop).Source
$wsl = if ($Mode -eq 'WSL') { (Get-Command wsl.exe -ErrorAction Stop).Source } else { $null }

$sshDir = Join-Path $env:USERPROFILE '.ssh'
$codexDir = if ($AgentHome) { $AgentHome } elseif ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $env:USERPROFILE '.codex' }
New-Item -ItemType Directory -Force $sshDir, $codexDir | Out-Null
$nativePrivate = Join-Path $sshDir $KeyName
$nativePublic = "$nativePrivate.pub"
$nativeKnown = Join-Path $sshDir "${KeyName}_known_hosts"
$configPath = Join-Path $codexDir 'config.toml'
$agentsPath = Join-Path $codexDir 'AGENTS.md'

if ((Test-Path -LiteralPath $nativePrivate) -xor (Test-Path -LiteralPath $nativePublic)) {
    throw 'A Windows keypair is incomplete; refusing to overwrite the remaining file.'
}
if (-not (Test-Path -LiteralPath $nativePrivate)) {
    Write-Host '[boxweave] Generating a dedicated private/public keypair on Windows.'
    & $sshKeygen -q -t ed25519 -f $nativePrivate -N '""' -C 'boxweave'
    Assert-Exit 'Windows ssh-keygen'
}
& $sshKeygen -lf $nativePublic -E sha256 | Out-Host
Assert-Exit 'Windows SSH public key validation'
$nativePubLine = [System.IO.File]::ReadAllText($nativePublic).Trim()
if ($nativePubLine -notmatch '^ssh-ed25519\s+[a-zA-Z0-9+/=]+') { throw 'Invalid Windows SSH public key.' }

# Pin SSH server identity using the key delivered by the Ubuntu console,
# never using untrusted ssh-keyscan or StrictHostKeyChecking=no.
$hostToken = if ($SshPort -eq 22) { $SshHost } else { '[{0}]:{1}' -f $SshHost, $SshPort }
$pinnedHostLine = "$hostToken ssh-ed25519 $ServerHostKey"
Set-PinnedHostNative -Path $nativeKnown -Line $pinnedHostLine

$clientPubLine = $nativePubLine
$transportCommand = 'ssh.exe'
$transportArgs = @()
if ($Mode -eq 'WSL') {
    $wslScript = @'
set -eu
umask 077
name="$1"
sshdir="$HOME/.ssh"
mkdir -p "$sshdir"
chmod 700 "$sshdir"
key="$sshdir/$name"
if [ -e "$key" ] && [ ! -e "$key.pub" ] || [ -e "$key.pub" ] && [ ! -e "$key" ]; then
    echo 'Incomplete WSL SSH keypair; refusing overwrite' >&2
    exit 2
fi
if [ ! -e "$key" ]; then
    ssh-keygen -q -t ed25519 -f "$key" -N '' -C 'boxweave-wsl'
fi
chmod 600 "$key"
cat "$key.pub"
'@
    Write-Host "[boxweave] Generating/reusing key in WSL distro: $WslDistro"
    $wslLines = & $wsl -d $WslDistro -e sh -c $wslScript sh $KeyName
    Assert-Exit 'WSL key generation'
    $clientPubLine = ($wslLines | Where-Object { $_ -match '^ssh-ed25519\s+' } | Select-Object -Last 1).Trim()
    if (-not $clientPubLine) { throw 'WSL did not return a valid SSH public key.' }

    $wslHomeLines = & $wsl -d $WslDistro -e sh -c 'printf "%s\n" "$HOME"'
    Assert-Exit 'WSL home lookup'
    $wslHome = ($wslHomeLines | Select-Object -Last 1).Trim()
    if ($wslHome -notmatch '^/[^\r\n]+$') { throw 'Invalid WSL home path' }

    $wslPinScript = @'
set -eu
umask 077
path="$HOME/.ssh/$1"
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
expected=$(printf '%s' "$2" | base64 -d)
if [ -e "$path" ] && [ "$(cat "$path")" != "$expected" ]; then
    echo 'WSL SSH pinned host key differs; verify identity before changing it' >&2
    exit 1
fi
printf '%s\n' "$expected" > "$path"
chmod 600 "$path"
'@
    $pinB64 = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes($pinnedHostLine))
    & $wsl -d $WslDistro -e sh -c $wslPinScript sh "${KeyName}_known_hosts" $pinB64
    Assert-Exit 'WSL host pinning'
    $linuxKey = "$wslHome/.ssh/$KeyName"
    $linuxKnown = "$wslHome/.ssh/${KeyName}_known_hosts"
    $transportCommand = 'wsl.exe'
    $transportArgs = @('-d', $WslDistro, '-e', 'ssh', '-T', '-i', $linuxKey,
                       '-o', 'IdentitiesOnly=yes', '-o', 'BatchMode=yes',
                       '-o', 'StrictHostKeyChecking=yes', '-o', "UserKnownHostsFile=$linuxKnown",
                       '-o', 'HostKeyAlgorithms=ssh-ed25519', '-p', "$SshPort",
                       "$SshUser@$SshHost", $McpRemoteCommand)
} else {
    $transportArgs = @('-T', '-i', $nativePrivate,
                       '-o', 'IdentitiesOnly=yes', '-o', 'BatchMode=yes',
                       '-o', 'StrictHostKeyChecking=yes', '-o', "UserKnownHostsFile=$nativeKnown",
                       '-o', 'HostKeyAlgorithms=ssh-ed25519', '-p', "$SshPort",
                       "$SshUser@$SshHost", $McpRemoteCommand)
}

# Register ONLY the active transport's public key under a restricted sshd key option.
$restricted = 'restrict,command="' + $McpRemoteCommand + '" ' + $clientPubLine
$keyData = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes($restricted))
$keyMaterial = ($clientPubLine -split '\s+')[1]
$enrollmentScript = @'
set -eu
umask 077
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
touch "$HOME/.ssh/authorized_keys"
chmod 600 "$HOME/.ssh/authorized_keys"
line=$(printf '%s' '__KEYDATA__' | base64 -d)
tmp=$(mktemp "$HOME/.ssh/.authorized_keys.boxweave.XXXXXX")
trap 'rm -f "$tmp"' EXIT
awk -v key='__KEYMATERIAL__' '{ keep=1; for(i=1;i<=NF;i++) if($i==key) keep=0; if(keep) print $0 }' "$HOME/.ssh/authorized_keys" > "$tmp"
printf '%s\n' "$line" >> "$tmp"
chmod 600 "$tmp"
mv -f "$tmp" "$HOME/.ssh/authorized_keys"
'@
$enrollmentScript = $enrollmentScript.Replace('__KEYDATA__', $keyData).Replace('__KEYMATERIAL__', $keyMaterial)
$sshAuthArgs = @('-o', 'StrictHostKeyChecking=yes', '-o', "UserKnownHostsFile=$nativeKnown",
                 '-o', 'HostKeyAlgorithms=ssh-ed25519', '-p', "$SshPort")
# Do not let ssh-agent offer unrelated identities during first-time enrollment:
# sshd may disconnect with "Too many authentication failures" before a password prompt.
# For SSH-key-only servers, explicitly select the already-authorized bootstrap key.
if ($BootstrapIdentityFile) {
    if (-not (Test-Path -LiteralPath $BootstrapIdentityFile -PathType Leaf)) {
        throw "Bootstrap SSH identity file does not exist: $BootstrapIdentityFile"
    }
    $bootstrapKey = (Resolve-Path -LiteralPath $BootstrapIdentityFile).ProviderPath
    $sshAuthArgs += @('-o', 'IdentitiesOnly=yes', '-o', 'IdentityAgent=none',
                      '-o', 'PreferredAuthentications=publickey', '-i', $bootstrapKey)
    Write-Host '[boxweave] Enrolling with the explicitly selected bootstrap SSH identity.'
} else {
    $sshAuthArgs += @('-o', 'PubkeyAuthentication=no',
                      '-o', 'PreferredAuthentications=keyboard-interactive,password')
    Write-Host '[boxweave] Enrolling with password authentication (SSH agent keys disabled).'
    Write-Host '         On key-only servers, rerun with -BootstrapIdentityFile PATH.'
}
Write-Host '[boxweave] Authorizing the selected transport key on Ubuntu using your EXISTING SSH login.'
$scriptB64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($enrollmentScript))
$remoteEnrollmentCommand = "printf '%s' '$scriptB64' | base64 -d | sh"
& $ssh @sshAuthArgs -T "$SshUser@$SshHost" $remoteEnrollmentCommand
Assert-Exit 'Restricted SSH key enrollment'

Update-CodexToml -Path $configPath -Command $transportCommand -Arguments $transportArgs
$agents = @"
## BoxWeave remote development environment (Containarium provider)

- Use MCP server **$McpName** (BoxWeave with the Containarium provider) for provisioning and managing LXC containers. Use its dynamic `connect` tool inside an existing box, not a separate per-box MCP registration.
- Ubuntu SSH endpoint: **${SshHost}:${SshPort}**, Unix user: **${SshUser}**. SSH transport mode: **$Mode**. The dedicated transport key is restricted by OpenSSH `restrict,command=`; it is NOT a general Ubuntu shell credential.
- The managed Ubuntu-to-container public SSH key is exactly:

  $boxPub

- When creating a container that requires `connect`, pass that COMPLETE public-key line via `create_container.ssh_keys`. Never substitute the Windows/WSL transport key. Do not assume a public-key fingerprint is enough.
- Reuse a suitable existing project container; otherwise provision an isolated box. Query box state before connecting and verify actual results afterward.
- If container creation times out, first query `list_containers`/`get_container` before retrying. Do NOT create duplicates, force-delete or overwrite resources.
- After recreation, `knownhosts: key mismatch` can indicate IP reuse; wait for the host-side trusted Incus synchronization and retry within a bounded window. Never disable SSH verification or edit Ubuntu `known_hosts` from Codex.
- Develop, test and deploy applications INSIDE each project's LXC. Prefer non-root container user and use systemd/Compose for persistence if appropriate. Verify health, reboot persistence and requested LAN access.
- Use Containarium port passthrough only when explicitly required. Never assume it provides LAN-only source-IP enforcement; protect sensitive services with authentication/network restrictions.
- Never modify `containarium-core-*` infrastructure boxes, unrelated projects, host SSH settings, root access, or platform credentials. Obtain permission for destructive operations.
- The MCP token may have admin privileges; use only requested tools and scopes, never print secrets or private keys.
"@
Update-Agents -Path $agentsPath -Content $agents

Write-Host ''
Write-Host '===================================================================='
Write-Host 'BoxWeave client configuration complete.'
Write-Host "Mode:           $Mode"
Write-Host "Windows key:    $nativePrivate (private key retained on Windows)"
if ($Mode -eq 'WSL') { Write-Host "WSL key:        $linuxKey (private key retained inside $WslDistro)" }
Write-Host "Codex TOML:     $configPath"
Write-Host "Global rules:   $agentsPath"
Write-Host 'Restart Windows Codex Desktop; its runtime stays Windows-native.'
Write-Host 'Test list_containers and connect through the BoxWeave MCP registration.'
Write-Host '===================================================================='
