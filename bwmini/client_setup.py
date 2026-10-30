"""Source-bound, public-only client enrollment generator for Windows Codex + WSL SSH.

Both terminal wizard and non-interactive invocation share validate/render/publish.
The generated script requires an existing, reviewed SSH administrator login
for the root-authorized BoxWeave public-key enrollment. No signer, private key,
or Provider credential is ever exported.
"""
import argparse
import json
import os
from pathlib import Path
import pwd
import re
import stat
import sys

PUBLIC_DESCRIPTOR = Path("/usr/local/lib/boxweave/default/client-public.json")


def load_public_state(path=None):
    """Only root-published, non-secret metadata; never read the private install marker."""
    path = PUBLIC_DESCRIPTOR if path is None else Path(path)
    meta = path.lstat()
    if (not stat.S_ISREG(meta.st_mode) or meta.st_uid != 0
            or meta.st_mode & 0o022 or meta.st_size > 16384):
        raise ValueError("PUBLIC_DESCRIPTOR_UNTRUSTED")
    for parent in path.parents:
        info = parent.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
            raise ValueError("PUBLIC_DESCRIPTOR_PARENT_UNTRUSTED")
    state = json.loads(path.read_text(encoding="utf-8"))
    if (not isinstance(state, dict) or state.get("schema_version") != 1
            or state.get("phase") != "ready"):
        raise ValueError("READY_INSTALL_REQUIRED")
    return state


try:
    from . import online
except ImportError:
    import online

HOST = re.compile(r"[A-Za-z0-9][A-Za-z0-9.-]{0,252}\Z")
USER = re.compile(r"[a-z_][a-z0-9_-]{0,30}\Z")
PUB = re.compile(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}\Z")

# PowerShell runs on Windows; ssh-keygen, SSH private key and MCP probe run in WSL.
# The *only* server mutation is a sudo-authorized call to online.py enroll.
POWERSHELL = r"""#requires -version 5.1
<#
BoxWeave Codex/Windows + WSL SSH setup. Generated from the reviewed server release.
Read this script before executing. No secrets are embedded or downloaded.
By default it will ask an existing Linux administrator to authorize a public
SSH key using sudo online.py enroll. Use -NoEnroll to do that independently.
#>
[CmdletBinding()]
param(
    [string]$Distro = '',
    [string]$BootstrapUser = '__ADMIN_USER__',
    [ValidateSet('password','publickey')][string]$AdminAuthentication = 'password',
    [string]$BootstrapIdentity = '',
    [switch]$NoEnroll,
    [switch]$AlreadyEnrolled,
    [switch]$Replace,
    [switch]$WslGuidance
)
$ErrorActionPreference = 'Stop'
$ServerHost = '__HOST__'
$ServerPort = __PORT__
$ServerKey = '__SERVER_KEY__'
$ManagedWorkloadPublicKey = '__MANAGED_PUBLIC_KEY__'
$McpServer = 'boxweave'
$RemoteMcp = '/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py'
$wsl = Join-Path $env:SystemRoot 'System32\wsl.exe'
if (-not (Test-Path -LiteralPath $wsl -PathType Leaf)) {
    throw ('Windows WSL executable not found: ' + $wsl)
}
if (-not $Distro) {
    $available = @(& $wsl --list --quiet | ForEach-Object { ($_ -replace [char]0, '').Trim() } | Where-Object { $_ })
    if ($LASTEXITCODE -ne 0 -or $available.Count -eq 0) { throw 'No WSL distribution is available' }
    if ($available.Count -eq 1) { $Distro = $available[0] }
    else {
        for ($i = 0; $i -lt $available.Count; $i++) { Write-Host ('{0}) {1}' -f ($i + 1), $available[$i]) }
        $choice = Read-Host 'Select WSL distribution number'
        if ($choice -notmatch '^[0-9]+$' -or [int]$choice -lt 1 -or [int]$choice -gt $available.Count) {
            throw 'Invalid WSL distribution selection'
        }
        $Distro = $available[[int]$choice - 1]
    }
}
if ($Distro -cnotmatch '^[A-Za-z0-9_.-]{1,80}$') { throw 'Invalid WSL distribution name' }
if ($BootstrapUser -cnotmatch '^[a-z_][a-z0-9_-]{0,30}$') { throw 'Invalid Linux administrator login' }

# The Python program below is always executed INSIDE the selected WSL distro.
# Its only writes are under ~/.config/boxweave in that distro. Existing pins
# and keypairs are never silently overwritten.
$identityScript = @'
import json, os, pathlib, re, stat, subprocess, sys
host, port, hostkey = sys.argv[1:]
assert re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9.-]{0,252}', host)
assert port.isdigit() and 1 <= int(port) <= 65535
assert re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/]+={0,2}', hostkey)
home = pathlib.Path.home()
assert home.is_dir() and not home.is_symlink() and home.stat().st_uid == os.getuid()
config = home / '.config'
assert not config.is_symlink()
config.mkdir(mode=0o700, exist_ok=True)
assert config.stat().st_uid == os.getuid()
root = config / 'boxweave'
assert not root.is_symlink()
root.mkdir(mode=0o700, exist_ok=True)
assert root.stat().st_uid == os.getuid() and not (root.stat().st_mode & 0o077)
key = root / 'id_ed25519'
pub = root / 'id_ed25519.pub'
pin = root / 'known_hosts'
assert all(not p.is_symlink() for p in (key,pub,pin))
assert key.exists() == pub.exists(), 'Partial keypair refused'
if not key.exists():
    subprocess.run(['/usr/bin/ssh-keygen','-q','-t','ed25519','-N','','-C','boxweave-codex-wsl','-f',str(key)],
                   check=True, timeout=30)
assert key.stat().st_uid == os.getuid() and stat.S_IMODE(key.stat().st_mode) == 0o600
assert pub.stat().st_uid == os.getuid()
public = ' '.join(pub.read_text().strip().split()[:2])
assert re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/]+={0,2}', public)
token = host if int(port) == 22 else '[' + host + ']:' + port
expected = token + ' ' + hostkey + '\n'
if pin.exists():
    assert pin.read_text() == expected, 'SSH host identity changed: refuse to repin'
else:
    fd = os.open(pin, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as stream: stream.write(expected)
assert pin.stat().st_uid == os.getuid() and stat.S_IMODE(pin.stat().st_mode) == 0o600
print(json.dumps({'key':str(key),'known_hosts':str(pin),'public_key':public}))
'@
$raw = & $wsl --distribution $Distro --exec /usr/bin/python3 -c $identityScript $ServerHost "$ServerPort" $ServerKey
if ($LASTEXITCODE -ne 0) { throw 'WSL identity or host-key pin verification failed' }
$identity = $raw | ConvertFrom-Json
if (-not $identity.public_key -or -not $identity.key -or -not $identity.known_hosts) {
    throw 'WSL identity result invalid'
}
Write-Host ('WSL identity ready in distribution: ' + $Distro)

# -NoEnroll supports two-person administration: display public key and exit
# before writing Codex configuration. The host key is still pinned.
if ($NoEnroll -and $AlreadyEnrolled) { throw 'Cannot combine -NoEnroll and -AlreadyEnrolled' }
if ($NoEnroll) {
    Write-Host 'Enrollment pending. On the Ubuntu BoxWeave server, run (then rerun with -AlreadyEnrolled):'
    Write-Host ("sudo python3 /usr/local/lib/boxweave/default/online.py enroll --public-key '" + $identity.public_key + "'")
    exit 0
}
if (-not $AlreadyEnrolled) {
    $remote = "sudo /usr/bin/python3 /usr/local/lib/boxweave/default/online.py enroll --public-key '" + $identity.public_key + "'"
    $authArgs = @('-F','/dev/null','-tt',
                  '-o','StrictHostKeyChecking=yes','-o','IdentitiesOnly=yes',
                  '-o','IdentityAgent=none','-o','ConnectTimeout=15',
                  '-o',('UserKnownHostsFile=' + $identity.known_hosts))
    if ($AdminAuthentication -eq 'password') {
        if ($BootstrapIdentity) { throw 'Password login cannot use -BootstrapIdentity' }
        $authArgs += @('-o','PubkeyAuthentication=no',
                       '-o','PreferredAuthentications=password,keyboard-interactive')
    } else {
        if (-not $BootstrapIdentity -or -not $BootstrapIdentity.StartsWith('/')) {
            throw 'Public-key administrator auth needs -BootstrapIdentity ABSOLUTE_WSL_PATH'
        }
        $authArgs += @('-o','PreferredAuthentications=publickey',
                       '-i',$BootstrapIdentity)
    }
    $authArgs += @('-p',"$ServerPort",("$BootstrapUser@$ServerHost"),$remote)
    Write-Host ('Authorizing public key via existing Linux administrator: ' + $BootstrapUser)
    & $wsl --distribution $Distro --exec /usr/bin/ssh @authArgs
    if ($LASTEXITCODE -ne 0) {
        throw 'Public-key enrollment not confirmed; no Codex MCP entry was modified'
    }
}

# Real, read-only MCP initialize -> tools/list -> list_containers verification.
$probeScript = @'
import json, subprocess, sys
host,port,key,pin = sys.argv[1:]
ssh=['/usr/bin/ssh','-F','/dev/null','-T','-o','BatchMode=yes',
     '-o','IdentitiesOnly=yes','-o','StrictHostKeyChecking=yes',
     '-o','ConnectTimeout=15','-o','UserKnownHostsFile='+pin,
     '-i',key,'-p',port,'boxweave@'+host,
     '/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py']
commands=[
 {'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'boxweave-codex-wsl','version':'1.0'}}},
 {'jsonrpc':'2.0','method':'notifications/initialized'},
 {'jsonrpc':'2.0','id':2,'method':'tools/list','params':{}},
 {'jsonrpc':'2.0','id':3,'method':'tools/call','params':{'name':'list_containers','arguments':{}}}]
result=subprocess.run(ssh,input=''.join(json.dumps(x)+'\n' for x in commands),
                      capture_output=True,text=True,timeout=90)
assert result.returncode == 0, 'SSH/MCP command failed'
messages=[json.loads(line) for line in result.stdout.splitlines() if line.strip()]
by_id={m.get('id'):m for m in messages if isinstance(m,dict) and 'id' in m}
assert by_id[1].get('result',{}).get('protocolVersion') == '2024-11-05', 'initialize failed'
assert any(t.get('name')=='list_containers' for t in by_id[2].get('result',{}).get('tools',[])), 'list tool missing'
read=by_id[3].get('result',{})
assert not read.get('isError') and read.get('content'), 'read failed'
print('MCP initialize/tools/list/list_containers: passed')
'@
& $wsl --distribution $Distro --exec /usr/bin/python3 -c $probeScript $ServerHost "$ServerPort" $identity.key $identity.known_hosts
if ($LASTEXITCODE -ne 0) {
    throw 'MCP read verification failed; no Codex MCP entry was modified'
}

# ChatGPT/Codex Windows desktop reads %USERPROFILE%\.codex\config.toml.
# No external codex CLI or WindowsApps binary execution is required.
# PowerShell array expressions require a single scalar SSH destination.
# Do not use string concatenation as bare tokens in @(...).
$sshTarget = 'boxweave@' + $ServerHost
$sshArgs = @('--distribution',$Distro,'--exec','/usr/bin/ssh',
             '-F','/dev/null','-T',
             '-o','BatchMode=yes','-o','IdentitiesOnly=yes',
             '-o','StrictHostKeyChecking=yes',
             '-o','ConnectTimeout=15','-o',('UserKnownHostsFile=' + $identity.known_hosts),
             '-i',$identity.key,'-p',"$ServerPort",$sshTarget,$RemoteMcp)
$homeDir = if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $env:USERPROFILE '.codex' }
if (-not [IO.Path]::IsPathRooted($homeDir)) { throw 'Codex home must be an absolute Windows path' }
$homeDir = [IO.Path]::GetFullPath($homeDir)
if (-not (Test-Path -LiteralPath $homeDir)) {
    [void][IO.Directory]::CreateDirectory($homeDir)
}
if (-not (Test-Path -LiteralPath $homeDir -PathType Container) -or
    ((Get-Item -LiteralPath $homeDir).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    throw 'Codex home path is unsafe'
}
$configPath = Join-Path $homeDir 'config.toml'
if ((Test-Path -LiteralPath $configPath) -and
    ((Get-Item -LiteralPath $configPath).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    throw 'Codex configuration symlink/reparse point refused'
}
$oldConfig = if (Test-Path -LiteralPath $configPath -PathType Leaf) {
    [IO.File]::ReadAllText($configPath, [Text.Encoding]::UTF8)
} else { '' }
# Preserve unrelated TOML, including legitimate nested BoxWeave tables
# such as [mcp_servers.boxweave.env] and [mcp_servers.boxweave.oauth].
# Only the top-level transport fields are candidates for replacement.
function Merge-BoxWeaveToml([string]$Original, [string]$CommandJson,
                            [string]$ArgsJson, [bool]$AllowReplace) {
    if ($Original -match '(?m)^[ \t]*mcp_servers[ \t]*=' -or
        $Original -match '(?m)^[ \t]*mcp_servers\.boxweave\.' -or
        $Original -match '(?m)^[ \t]*\[[^\]\r\n]*["'']boxweave["''][^\]\r\n]*\]') {
        throw 'Ambiguous existing MCP TOML; inspect config.toml manually'
    }
    $desired = [ordered]@{
        command = $CommandJson
        args = $ArgsJson
        startup_timeout_sec = '30'
        tool_timeout_sec = '180'
    }
    $rootPattern = '(?ms)^[ \t]*\[mcp_servers\.boxweave\][ \t]*(?:\#[^\r\n]*)?\r?\n.*?(?=^[ \t]*\[|\z)'
    $roots = [regex]::Matches($Original, $rootPattern)
    if ($roots.Count -gt 1) { throw 'Duplicate BoxWeave MCP root table; manual review required' }
    if ($roots.Count -eq 0) {
        if ($Original -match '(?m)^[ \t]*\[mcp_servers\.boxweave\.') {
            throw 'Nested BoxWeave MCP table without explicit parent: manual review required'
        }
        $newRoot = '[mcp_servers.boxweave]' + "`r`n"
        foreach ($key in $desired.Keys) {
            $newRoot += $key + ' = ' + $desired[$key] + "`r`n"
        }
        return $Original.TrimEnd([char]13,[char]10) + "`r`n`r`n" + $newRoot
    }
    $root = $roots[0].Value
    $lines = @($root -split '\r?\n')
    $remaining = New-Object 'System.Collections.Generic.List[string]'
    $found = @{}
    $alreadyMatching = $true
    for ($i = 1; $i -lt $lines.Count; $i++) {
        $line = $lines[$i]
        $kv = [regex]::Match($line, '^[ \t]*([A-Za-z_][A-Za-z0-9_]*)[ \t]*=[ \t]*(.*)$')
        if (-not $kv.Success -or -not $desired.Contains($kv.Groups[1].Value)) {
            $remaining.Add($line)
            continue
        }
        $key = $kv.Groups[1].Value
        if ($found.ContainsKey($key)) { throw 'Duplicate BoxWeave transport field: manual review required' }
        $found[$key] = $true
        $value = $kv.Groups[2].Value.Trim()
        if (($key -eq 'args' -and ($value -notmatch '^\[.*\](?:[ \t]*#.*)?$')) -or
            ($key -eq 'command' -and $value -match '^(?:"{3}|''{3})')) {
            throw 'Multiline BoxWeave transport field: manual review required'
        }
        if ($line.Trim() -cne ($key + ' = ' + $desired[$key])) {
            $alreadyMatching = $false
        }
    }
    foreach ($key in $desired.Keys) {
        if (-not $found.ContainsKey($key)) { $alreadyMatching = $false }
    }
    if ($alreadyMatching) { return $Original }
    if (-not $AllowReplace) {
        throw 'BoxWeave MCP is already configured. Review it and pass -Replace to update only transport fields'
    }
    $newRoot = $lines[0] + "`r`n"
    if ($remaining.Count -gt 0) {
        $keptText = ($remaining -join "`r`n").TrimEnd([char]13,[char]10)
        if ($keptText) { $newRoot += $keptText + "`r`n" }
    }
    foreach ($key in $desired.Keys) {
        $newRoot += $key + ' = ' + $desired[$key] + "`r`n"
    }
    $matcher = [regex]::new($rootPattern)
    return $matcher.Replace($Original, [Text.RegularExpressions.MatchEvaluator]{
        param($match)
        $newRoot
    }, 1)
}
$wslString = ConvertTo-Json -InputObject $wsl -Compress
$argsString = ConvertTo-Json -InputObject @($sshArgs) -Compress
$newConfig = Merge-BoxWeaveToml $oldConfig $wslString $argsString ([bool]$Replace)
$changed = $newConfig -cne $oldConfig
if ($changed) {
    $temp = Join-Path $homeDir ('.boxweave-config-' + [guid]::NewGuid().ToString('N') + '.tmp')
    try {
        [IO.File]::WriteAllText($temp, $newConfig, [Text.UTF8Encoding]::new($false))
        if (Test-Path -LiteralPath $configPath) {
            $backup = $configPath + '.boxweave-' + [guid]::NewGuid().ToString('N') + '.bak'
            [IO.File]::Replace($temp, $configPath, $backup)
            Write-Host ('Previous Codex configuration backed up: ' + $backup)
        } else {
            [IO.File]::Move($temp, $configPath)
        }
    } finally {
        if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Force }
    }
}
# Restore archive/0.5.1's public workload-key guidance. Client transport
# identity must NEVER be used for a new workload's ssh_keys.
$guidance = @'
<!-- BEGIN BOXWEAVE MANAGED -->
## BoxWeave managed box SSH identity
- The BoxWeave MCP server at __HOST__:__PORT__ connects to workload boxes with its own separate server-managed SSH key.
- When calling this server's create_container, include the following FULL public key in ssh_keys (not merely a fingerprint):
  __MANAGED_PUBLIC_KEY__
- You may include other explicitly authorized public keys, but must retain the key above for this server's connect tool.
- Do NOT use your Windows/WSL-to-BoxWeave transport public key or a key from another nested MCP host instead. Never read, request, copy or disclose private keys, JWTs, or host trust secrets.
- Reuse and inspect existing boxes before creating. After creation, Running is not SSH readiness; use get_container, then connect to verify whoami and hostname before running commands. A failed connect does not justify duplicate creation.
- If this MCP server reports MANAGED_PUBLIC_KEY_REQUIRED or MANAGED_IDENTITY_UNAVAILABLE, stop and reconcile the correct public identity; do not bypass host-key verification.
<!-- END BOXWEAVE MANAGED -->
'@
function Merge-BoxWeaveGuidance([string]$Original, [string]$Block) {
    $begin = '<!-- BEGIN BOXWEAVE MANAGED -->'
    $end = '<!-- END BOXWEAVE MANAGED -->'
    $startCount = [regex]::Matches($Original, [regex]::Escape($begin)).Count
    $endCount = [regex]::Matches($Original, [regex]::Escape($end)).Count
    if ($startCount -ne $endCount -or $startCount -gt 1) {
        throw 'Ambiguous BoxWeave AGENTS marker: manual review required'
    }
    if ($startCount -eq 0) {
        if (-not $Original) { return $Block + "`r`n" }
        return $Original.TrimEnd([char]13,[char]10) + "`r`n`r`n" + $Block + "`r`n"
    }
    $pattern = '(?s)' + [regex]::Escape($begin) + '.*?' + [regex]::Escape($end)
    return ([regex]::new($pattern)).Replace($Original,
        [Text.RegularExpressions.MatchEvaluator]{ param($match) $Block }, 1)
}
$guidancePath = Join-Path $homeDir 'AGENTS.md'
$overridePath = Join-Path $homeDir 'AGENTS.override.md'
foreach ($candidate in @($guidancePath,$overridePath)) {
    if ((Test-Path -LiteralPath $candidate) -and
        ((Get-Item -LiteralPath $candidate).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
        throw 'Codex AGENTS symlink/reparse point refused'
    }
}
if ((Test-Path -LiteralPath $overridePath -PathType Leaf) -and
    [IO.File]::ReadAllText($overridePath).Trim()) {
    $guidancePath = $overridePath
}
# Do not silently leave two different managed policies in the active and
# inactive AGENTS files.
$otherGuidance = if ($guidancePath -eq $overridePath) {
    Join-Path $homeDir 'AGENTS.md'
} else { $overridePath }
if ((Test-Path -LiteralPath $otherGuidance -PathType Leaf) -and
    [IO.File]::ReadAllText($otherGuidance).Contains('<!-- BEGIN BOXWEAVE MANAGED -->')) {
    throw 'Existing BoxWeave guidance in another AGENTS file: manual review required'
}
$oldGuidance = if (Test-Path -LiteralPath $guidancePath -PathType Leaf) {
    [IO.File]::ReadAllText($guidancePath, [Text.Encoding]::UTF8)
} else { '' }
$newGuidance = Merge-BoxWeaveGuidance $oldGuidance $guidance
if ($newGuidance -cne $oldGuidance) {
    $temporary = Join-Path $homeDir ('.boxweave-guidance-' + [guid]::NewGuid().ToString('N') + '.tmp')
    try {
        [IO.File]::WriteAllText($temporary, $newGuidance, [Text.UTF8Encoding]::new($false))
        if (Test-Path -LiteralPath $guidancePath -PathType Leaf) {
            $backup = $guidancePath + '.boxweave-' + [guid]::NewGuid().ToString('N') + '.bak'
            [IO.File]::Replace($temporary, $guidancePath, $backup)
            Write-Host ('Previous Codex guidance backed up: ' + $backup)
        } else {
            [IO.File]::Move($temporary, $guidancePath)
        }
    } finally {
        if (Test-Path -LiteralPath $temporary) { Remove-Item -LiteralPath $temporary -Force }
    }
}
Write-Host ('Windows Codex managed box guidance: ' + $guidancePath)

# Optional: the Codex WSL Agent has a distinct Linux home/instruction scope.
# The current Windows-generated MCP registration remains independently managed.
if ($WslGuidance) {
    $policyBase64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($guidance))
    $installWslGuidance = @'
import base64, os, pathlib, re, stat, sys, tempfile, uuid
policy = base64.b64decode(sys.argv[1], validate=True).decode("utf-8")
begin, end = "<!-- BEGIN BOXWEAVE MANAGED -->", "<!-- END BOXWEAVE MANAGED -->"
assert policy.count(begin) == policy.count(end) == 1
home = pathlib.Path.home()
assert home.is_dir() and not home.is_symlink() and home.stat().st_uid == os.getuid()
root = home / ".codex"
assert not root.is_symlink()
root.mkdir(mode=0o700, exist_ok=True)
assert root.is_dir() and root.stat().st_uid == os.getuid()
regular = root / "AGENTS.md"
override = root / "AGENTS.override.md"
for p in (regular, override):
    if p.is_symlink() or (p.exists() and (not p.is_file() or p.stat().st_uid != os.getuid())):
        raise ValueError("UNSAFE_WSL_GUIDANCE_PATH")
active = override if override.is_file() and override.read_text(encoding="utf-8").strip() else regular
other = regular if active == override else override
if other.is_file() and begin in other.read_text(encoding="utf-8"):
    raise ValueError("DUPLICATE_WSL_GUIDANCE_LOCATION")
before = active.read_text(encoding="utf-8") if active.exists() else ""
if before.count(begin) != before.count(end) or before.count(begin) > 1:
    raise ValueError("AMBIGUOUS_WSL_GUIDANCE_MARKERS")
if begin in before:
    after = re.sub(re.escape(begin) + r".*?" + re.escape(end),
                   lambda match: policy, before, count=1, flags=re.S)
else:
    after = before.rstrip("\r\n") + ("\n\n" if before else "") + policy + "\n"
if after != before:
    fd, stage = tempfile.mkstemp(prefix=".boxweave-guidance-", dir=root)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            stream.write(after); stream.flush(); os.fsync(stream.fileno())
        if active.exists():
            backup = root / (active.name + ".boxweave-" + uuid.uuid4().hex + ".bak")
            with backup.open("xb") as dest:
                dest.write(before.encode("utf-8")); dest.flush(); os.fsync(dest.fileno())
            os.chmod(backup, 0o600)
        os.replace(stage, active)
    finally:
        pathlib.Path(stage).unlink(missing_ok=True)
print("WSL Codex managed box guidance: " + str(active))
'@
    & $wsl --distribution $Distro --exec /usr/bin/python3 -c $installWslGuidance $policyBase64
    if ($LASTEXITCODE -ne 0) {
        throw 'WSL Codex guidance failed; Windows registration/guidance may have been saved'
    }
}
Write-Host ('Windows ChatGPT/Codex MCP config: ' + $configPath)
Write-Host 'MCP read verified and BoxWeave configured with managed workload public key guidance. Restart ChatGPT/Codex desktop.'

""";

def ps_quote(value):
    return "'" + value.replace("'", "''") + "'"


def render(state, admin_user):
    host, port = state.get("host"), state.get("port")
    key = state.get("server_key", "")
    managed_key = state.get("managed_public_key", "")
    if (not isinstance(host, str) or not HOST.fullmatch(host)
        or type(port) is not int or not 1 <= port <= 65535
        or not isinstance(key, str) or not PUB.fullmatch(" ".join(key.split()[:2]))
        or not USER.fullmatch(admin_user)):
        raise ValueError("PUBLIC_CLIENT_IDENTITY_INVALID")
    try:
        managed_key = online.public_key(managed_key)
    except (ValueError, online.Failure, TypeError):
        raise ValueError("MANAGED_WORKLOAD_PUBLIC_KEY_INVALID") from None
    host_key = " ".join(key.split()[:2])
    return (POWERSHELL.replace("__HOST__", host).replace("__PORT__", str(port))
            .replace("__SERVER_KEY__", host_key).replace("__MANAGED_PUBLIC_KEY__", managed_key)
            .replace("__ADMIN_USER__", admin_user))


def publish(path, script):
    dest = Path(path).absolute()
    if (dest.is_symlink() or dest.exists() or dest.parent.is_symlink()
        or not dest.parent.is_dir()):
        raise ValueError("CLIENT_OUTPUT_MUST_BE_NEW_REGULAR_FILE")
    # Never write to an untrusted or world-writable output directory.
    parent = dest.parent.stat()
    if parent.st_mode & 0o002:
        raise ValueError("CLIENT_OUTPUT_PARENT_WORLD_WRITABLE")
    fd = os.open(dest, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="\r\n") as stream:
            stream.write(script)
        sudo_user = os.environ.get("SUDO_USER", "")
        if sudo_user not in ("", "root"):
            identity = pwd.getpwnam(sudo_user)
            # Only transfer ownership to the human when writing under their home.
            if dest.is_relative_to(Path(identity.pw_dir).resolve()):
                os.chown(dest, identity.pw_uid, identity.pw_gid)
    except BaseException:
        dest.unlink(missing_ok=True)
        raise
    return str(dest)


def select(name, available, prompt, interactive):
    if name:
        if name not in available:
            raise ValueError("UNSUPPORTED_CLIENT_PROFILE")
        return name
    if not interactive:
        raise ValueError("NONINTERACTIVE_CLIENT_SELECTION_REQUIRED")
    print(prompt, file=sys.stderr)
    for i, choice in enumerate(available, 1):
        print(f"  {i}) {choice}", file=sys.stderr)
    value = input("Select number: ").strip()
    if not value.isdecimal() or not 1 <= int(value) <= len(available):
        raise ValueError("CLIENT_SELECTION_CANCELLED")
    return available[int(value) - 1]


def main(argv=None):
    parser = argparse.ArgumentParser(description="BoxWeave reviewed Windows Codex/WSL setup script generator")
    parser.add_argument("--agent", choices=("codex",))
    parser.add_argument("--transport", choices=("wsl",))
    parser.add_argument("--admin-user", help="Existing Linux SSH administrator authorized to run sudo online.py enroll")
    parser.add_argument("--output", help="New PowerShell file; defaults to ./boxweave-setup.ps1")
    parser.add_argument("--stdout", action="store_true", help="Emit the public script to stdout instead of a file")
    args = parser.parse_args(argv)
    try:
        state = load_public_state()
        if state.get("phase") != "ready":
            raise ValueError("READY_INSTALL_REQUIRED")
        interactive = sys.stdin.isatty()
        select(args.agent, ("codex",), "Choose agent", interactive)
        select(args.transport, ("wsl",), "Choose SSH transport", interactive)
        # Ordinary users may generate public profiles without sudo. The login
        # used later to enroll a key is an existing administrator, not the
        # dedicated boxweave application account.
        admin = args.admin_user or os.environ.get("SUDO_USER", "") or (pwd.getpwuid(os.getuid()).pw_name if os.geteuid() != 0 else "")
        if not USER.fullmatch(admin or ""):
            if not interactive:
                raise ValueError("ADMIN_USER_REQUIRED")
            admin = input("Existing Linux SSH administrator: ").strip()
        if not USER.fullmatch(admin):
            raise ValueError("ADMIN_USER_INVALID")
        script = render(state, admin)
        if args.stdout:
            sys.stdout.write(script)
        else:
            target = args.output or str(Path.cwd() / "boxweave-setup.ps1")
            published = publish(target, script)
            print(json.dumps({"state":"generated","profile":"codex/windows/wsl",
                              "file":published,"registration":"not_attempted",
                              "next":"Copy to Windows; review; execute in PowerShell. Use -Distro NAME for multiple WSL distributions."}))
        return 0
    except (ValueError, OSError, EOFError) as exc:
        print(json.dumps({"state":"error","code":str(exc)}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
