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
By default it will ask an existing Ubuntu administrator to authorize a public
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
$ErrorActionPreference = 'Stop'
$ServerHost = '__HOST__'
$ServerPort = __PORT__
$ServerKey = '__SERVER_KEY__'
$McpServer = 'boxweave'
$RemoteMcp = '/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py'
$wsl = (Get-Command wsl.exe -CommandType Application -ErrorAction Stop).Source
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
if ($BootstrapUser -cnotmatch '^[a-z_][a-z0-9_-]{0,30}$') { throw 'Invalid Ubuntu administrator login' }

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
    Write-Host ('Authorizing public key via existing Ubuntu administrator: ' + $BootstrapUser)
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
$sshArgs = @('--distribution',$Distro,'--exec','/usr/bin/ssh',
             '-F','/dev/null','-T',
             '-o','BatchMode=yes','-o','IdentitiesOnly=yes',
             '-o','StrictHostKeyChecking=yes',
             '-o','ConnectTimeout=15','-o',('UserKnownHostsFile=' + $identity.known_hosts),
             '-i',$identity.key,'-p',"$ServerPort",'boxweave@' + $ServerHost,$RemoteMcp)
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
# Preserve all unrelated TOML. For ambiguous MCP structures refuse automation.
if ($oldConfig -match '(?m)^[ \t]*mcp_servers[ \t]*=' -or
    $oldConfig -match '(?m)^[ \t]*mcp_servers\.boxweave\.' -or
    $oldConfig -match '(?m)^[ \t]*\[mcp_servers\.boxweave\.' -or
    $oldConfig -match '(?m)^[ \t]*\[[^\]\r\n]*["'']boxweave["''][^\]\r\n]*\]') {
    throw 'Ambiguous existing MCP TOML; inspect config.toml manually'
}
$wslString = ConvertTo-Json -InputObject $wsl -Compress
$argsString = ConvertTo-Json -InputObject @($sshArgs) -Compress
$entry = "[mcp_servers.boxweave]" + "`r`n" +
         "command = " + $wslString + "`r`n" +
         "args = " + $argsString + "`r`n" +
         "startup_timeout_sec = 30" + "`r`n" +
         "tool_timeout_sec = 180" + "`r`n"
$pattern = '(?ms)^[ \t]*\[mcp_servers\.boxweave\][ \t]*(?:\#[^\r\n]*)?\r?\n.*?(?=^[ \t]*\[|\z)'
$matches = [regex]::Matches($oldConfig, $pattern)
if ($matches.Count -gt 1) { throw 'Multiple BoxWeave MCP TOML sections: manual review required' }
$changed = $false
if ($matches.Count -eq 1) {
    if ($matches[0].Value.Trim() -ne $entry.Trim()) {
        if (-not $Replace) {
            throw 'BoxWeave MCP is already configured. Review it and pass -Replace to update only this entry'
        }
        $matcher = [regex]::new($pattern)
        $newConfig = $matcher.Replace($oldConfig, [Text.RegularExpressions.MatchEvaluator]{ param($match) $entry }, 1)
        $changed = $true
    }
} else {
    $newConfig = $oldConfig.TrimEnd([char]13,[char]10) + "`r`n`r`n" + $entry
    $changed = $true
}
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
Write-Host ('Windows ChatGPT/Codex MCP config: ' + $configPath)
Write-Host 'MCP read verified and BoxWeave configured. Restart ChatGPT/Codex desktop to load the MCP server.'
""";

def ps_quote(value):
    return "'" + value.replace("'", "''") + "'"


def render(state, admin_user):
    host, port = state.get("host"), state.get("port")
    key = state.get("server_key", "")
    if (not isinstance(host, str) or not HOST.fullmatch(host)
        or type(port) is not int or not 1 <= port <= 65535
        or not isinstance(key, str) or not PUB.fullmatch(" ".join(key.split()[:2]))
        or not USER.fullmatch(admin_user)):
        raise ValueError("PUBLIC_CLIENT_IDENTITY_INVALID")
    host_key = " ".join(key.split()[:2])
    return (POWERSHELL.replace("__HOST__", host).replace("__PORT__", str(port))
            .replace("__SERVER_KEY__", host_key).replace("__ADMIN_USER__", admin_user))


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
    parser.add_argument("--admin-user", help="Existing Ubuntu SSH administrator authorized to run sudo online.py enroll")
    parser.add_argument("--output", help="New PowerShell file; defaults to ./boxweave-setup.ps1")
    parser.add_argument("--stdout", action="store_true", help="Emit the public script to stdout instead of a file")
    args = parser.parse_args(argv)
    try:
        if os.geteuid() != 0:
            raise ValueError("ROOT_REQUIRED_FOR_CLIENT_PUBLIC_STATE")
        state = online.current_state()
        if state.get("phase") != "ready":
            raise ValueError("READY_INSTALL_REQUIRED")
        interactive = sys.stdin.isatty()
        select(args.agent, ("codex",), "Choose agent", interactive)
        select(args.transport, ("wsl",), "Choose SSH transport", interactive)
        admin = args.admin_user or os.environ.get("SUDO_USER", "")
        if not USER.fullmatch(admin or ""):
            if not interactive:
                raise ValueError("ADMIN_USER_REQUIRED")
            admin = input("Existing Ubuntu SSH administrator: ").strip()
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
