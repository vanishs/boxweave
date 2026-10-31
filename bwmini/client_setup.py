"""Public-only client setup generator for alphabetically ordered cross-platform apps.

The previously reviewed Windows Codex/WSL PowerShell path is preserved.
Other profiles emit a self-contained Python client script, executed only on
the intended client OS, never on the Linux BoxWeave server.
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
    from . import online, client_profiles
except ImportError:
    import online
    import client_profiles

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
    [switch]$Replace
)
# The server-side wizard chooses a default. An explicit -Replace:$false at
# execution time can still veto replacement after reviewing the script.
__REPLACE_POLICY__
$ErrorActionPreference = 'Stop'
# EDIT ME: use the client-reachable host/IP and SSH port.
# Keep the independently verified server host-key pin unchanged.
$ServerHost = '__HOST__'
$ServerPort = __PORT__
$ServerKey = '__SERVER_KEY__'
if ($ServerHost -cnotmatch '^[A-Za-z0-9][A-Za-z0-9.-]{0,252}$' -or
    $ServerPort -lt 1 -or $ServerPort -gt 65535) {
    throw 'Invalid BoxWeave server address or SSH port'
}
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
    Write-Host 'Enrollment pending. On the Linux BoxWeave server, run (then rerun with -AlreadyEnrolled):'
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
# The Linux MCP gateway adds its own managed workload public key to create
# requests. This script deliberately does not modify Codex AGENTS files.
Write-Host ('Windows ChatGPT/Codex MCP config: ' + $configPath)
Write-Host 'MCP read verified and BoxWeave registered. Restart ChatGPT/Codex desktop.'

""";

def ps_quote(value):
    return "'" + value.replace("'", "''") + "'"


def render(state, admin_user, replace=False):
    if type(replace) is not bool:
        raise ValueError("CLIENT_REPLACE_POLICY_INVALID")
    host, port = state.get("host"), state.get("port")
    key = state.get("server_key", "")
    if (not isinstance(host, str) or not HOST.fullmatch(host)
        or type(port) is not int or not 1 <= port <= 65535
        or not isinstance(key, str) or not PUB.fullmatch(" ".join(key.split()[:2]))
        or not USER.fullmatch(admin_user)):
        raise ValueError("PUBLIC_CLIENT_IDENTITY_INVALID")
    host_key = " ".join(key.split()[:2])
    return (POWERSHELL.replace("__HOST__", host).replace("__PORT__", str(port))
            .replace("__SERVER_KEY__", host_key)
            .replace("__ADMIN_USER__", admin_user)
            .replace("__REPLACE_POLICY__",
                     "if (-not $PSBoundParameters.ContainsKey('Replace')) { $Replace = $true }"
                     if replace else ""))


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


class ClientCancelled(Exception):
    """The human declined the reviewed client setup change."""


def choose(prompt, options):
    """Interactive, bounded list: [(label, value), ...]. No implicit yes."""
    print("\n" + prompt, file=sys.stderr)
    for number, (label, _) in enumerate(options, 1):
        print(f"  {number}) {label}", file=sys.stderr)
    while True:
        print("Select number: ", end="", file=sys.stderr, flush=True)
        answer = input().strip()
        if answer.isdecimal() and 1 <= int(answer) <= len(options):
            return options[int(answer) - 1][1]
        print("Choose one of the listed numbers.", file=sys.stderr)


def select(name, available, prompt, interactive):
    if name:
        if name not in available:
            raise ValueError("UNSUPPORTED_CLIENT_PROFILE")
        return name
    if not interactive:
        raise ValueError("NONINTERACTIVE_CLIENT_SELECTION_REQUIRED")
    return choose(prompt, [(item, item) for item in available])


def script_header(replace_mcp):
    """PowerShell comments only; safe to copy or redirect as .ps1 source."""
    policy = ("Update only the existing BoxWeave MCP transport after verified read "
              "(backup kept)." if replace_mcp else
              "Preserve conflicting existing BoxWeave MCP transport settings.")
    return ("# ===============================================================\n"
            "# BoxWeave: Windows Codex + WSL SSH client setup\n"
            "# Save this complete script as boxweave-setup.ps1 on the Windows client.\n"
            "# Run in Windows PowerShell: .\\boxweave-setup.ps1 -Distro Ubuntu\n"
            "# Ubuntu is an example WSL distribution name; use the installed distro name.\n"
            "# This is a PowerShell script, not a Linux Bash script.\n"
            "# The script contains public data only. Review it before execution.\n"
            "# Windows MCP policy: " + policy + "\n"
            "# SSH public-key enrollment still needs host administrator approval.\n"
            "# ===============================================================\n\n")


def main(argv=None):
    parser = argparse.ArgumentParser(
        description="BoxWeave reviewed cross-platform MCP client setup generator",
        epilog="Apps are listed alphabetically; generated scripts require review and client-side execution.")
    parser.add_argument("--agent", "--app", dest="app", choices=client_profiles.APPS,
                        help="Target MCP application (alphabetical)")
    parser.add_argument("--os", dest="system", choices=client_profiles.SYSTEMS,
                        help="Operating system where the client app runs")
    parser.add_argument("--transport", choices=client_profiles.TRANSPORTS,
                        help="Native OpenSSH or Windows-to-WSL OpenSSH")
    parser.add_argument("--admin-user",
                        help="Existing Linux SSH administrator authorized to run sudo online.py enroll")
    destination = parser.add_mutually_exclusive_group()
    destination.add_argument("--output", help="Optional NEW setup script; refuses overwriting existing files")
    destination.add_argument("--stdout", action="store_true",
                             help="Print a copyable script to stdout (the default)")
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--replace", dest="mcp_replace", action="store_const", const=True,
                       help="Replace a conflicting BoxWeave MCP entry only after read verification; backup")
    group.add_argument("--no-replace", dest="mcp_replace", action="store_const", const=False,
                       help="Refuse conflicting BoxWeave MCP entries (noninteractive default)")
    parser.set_defaults(mcp_replace=None)
    args = parser.parse_args(argv)
    try:
        state = load_public_state()
        if state.get("phase") != "ready":
            raise ValueError("READY_INSTALL_REQUIRED")
        interactive = sys.stdin.isatty()
        app = select(args.app, client_profiles.APPS, "Choose MCP application (alphabetical)",
                     interactive)
        # A noninteractive old --transport wsl invocation remains Windows by default.
        system = select(args.system or ("windows" if args.transport == "wsl" else None),
                        client_profiles.SYSTEMS, "Choose client operating system", interactive)
        valid_transports = ("native", "wsl") if system == "windows" else ("native",)
        transport = select(args.transport, valid_transports,
                           "Choose SSH transport", interactive)
        if not client_profiles.profile_valid(app, system, transport):
            raise ValueError("UNSUPPORTED_CLIENT_PROFILE")
        admin = args.admin_user or os.environ.get("SUDO_USER", "") or (
            pwd.getpwuid(os.getuid()).pw_name if os.geteuid() != 0 else "")
        if not USER.fullmatch(admin or ""):
            if not interactive:
                raise ValueError("ADMIN_USER_REQUIRED")
            print("Existing Linux SSH administrator: ", end="", file=sys.stderr, flush=True)
            admin = input().strip()
        if not USER.fullmatch(admin):
            raise ValueError("ADMIN_USER_INVALID")
        if args.mcp_replace is not None:
            replace_mcp = args.mcp_replace
        elif interactive:
            replace_mcp = choose(
                "Replace an existing conflicting BoxWeave MCP entry on the client?",
                [("No: preserve the existing registration and stop for review", False),
                 ("Yes: replace only BoxWeave after authenticated MCP read; keep backup", True)])
        else:
            replace_mcp = False

        legacy = app == "codex" and system == "windows" and transport == "wsl"
        if legacy:
            script = script_header(replace_mcp) + render(state, admin, replace=replace_mcp)
            shell = "Windows PowerShell (.ps1)"
            filename = "boxweave-setup.ps1"
        else:
            public = {key: state[key] for key in ("host", "port", "server_key")}
            script = client_profiles.render(public | {
                "app": app, "os": system, "transport": transport,
                "admin_user": admin, "replace": replace_mcp})
            shell = "Python 3.11+ (.py)"
            filename = "boxweave-setup.py"
        if args.output:
            published = publish(args.output, script)
            print(shell + " setup saved to: " + published, file=sys.stderr)
        else:
            print("BoxWeave " + app + " / " + system + " / " + transport +
                  " setup; copy the complete " + shell + " script below.", file=sys.stderr)
            print("Or redirect stdout to " + filename + " and transfer it to the client.", file=sys.stderr)
            print("Generation changed no Linux or client settings.", file=sys.stderr)
            sys.stdout.write(script)
        return 0
    except (ClientCancelled, KeyboardInterrupt, EOFError):
        print(json.dumps({"state": "cancelled", "changed": False}), file=sys.stderr)
        return 130
    except (ValueError, OSError, KeyError) as exc:
        print(json.dumps({"state": "error", "code": str(exc)}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
