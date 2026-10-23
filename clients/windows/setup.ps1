#requires -version 5.1
<#
Configure BoxWeave with the Containarium provider for a supported Windows coding agent.
Mode Native: ssh.exe runs on Windows.
Mode WSL:    wsl.exe launches OpenSSH inside the selected WSL distribution.
Both modes preserve the native Windows agent runtime; WSL is only the SSH transport.
#>
[CmdletBinding()]
param(
    [string]$DataBase64 = '',
    [string]$ProjectPath = '',
    [switch]$Replace,
    [string]$SshHost = '',
    [ValidateRange(1,65535)][int]$SshPort = 22,
    [string]$SshUser = '',
    # Existing SSH login identity for key-only hosts; omit to use password login.
    [string]$BootstrapIdentityFile = '',
    [string]$ServerHostKey = '',
    [string]$BoxPublicKeyBase64 = '',
    [string]$McpRemoteCommand = '/usr/local/bin/boxweave-mcp',
    [ValidateSet('Native','WSL')][string]$Mode = 'Native',
    [string]$WslDistro = '',
    [ValidateSet('Codex','Cursor','Claude-Code')][string]$Agent = 'Codex',
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
    if ((Test-Path -LiteralPath $Path) -and ((Get-Item -LiteralPath $Path).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing configuration reparse point' }
    $tmp = Join-Path (Split-Path -Parent $Path) ('.boxweave-' + [guid]::NewGuid())
    try {
        [IO.File]::WriteAllText($tmp, $Content, (New-Object System.Text.UTF8Encoding($false)))
        if (Test-Path -LiteralPath $Path) { [IO.File]::Replace($tmp,$Path,[System.Management.Automation.Language.NullString]::Value) }
        else { [IO.File]::Move($tmp,$Path) }
    } finally { if (Test-Path -LiteralPath $tmp) { Remove-Item -LiteralPath $tmp -Force } }
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
    if ($text -match '(?m)^\s*mcp_servers\s*=' -or $text -match ('(?m)^\[mcp_servers\.' + [regex]::Escape($McpName) + '\.')) { throw 'Inline/nested MCP TOML requires explicit migration' }
    $header = '(?ms)^\[mcp_servers\.' + [regex]::Escape($McpName) + '\][^\r\n]*\r?\n.*?(?=^\[|\z)'
    $old = [regex]::Matches($text,$header)
    if ($old.Count -gt 1) { throw 'Duplicate BoxWeave MCP table' }
    if ($text -match [regex]::Escape($McpName) -and $old.Count -eq 0) { throw 'Unsupported existing BoxWeave TOML spelling' }
    if ($old.Count -eq 1 -and -not $Replace) {
        try {
            $extra=@($old[0].Value -split '\r?\n' | Where-Object { $_ -notmatch '^\s*(#|$|\[|command\s*=|args\s*=|startup_timeout_sec\s*=)' })
            if ($extra.Count) { throw 'Additional settings need explicit replacement' }
            $c = [regex]::Match($old[0].Value,'(?m)^command\s*=\s*(.+?)\s*$').Groups[1].Value | ConvertFrom-Json
            $a = @([regex]::Match($old[0].Value,'(?m)^args\s*=\s*(.+?)\s*$').Groups[1].Value | ConvertFrom-Json)
            if ($c -ne $Command -or (ConvertTo-Json -InputObject $a -Compress) -cne (ConvertTo-Json -InputObject @($Arguments) -Compress)) { throw 'Different entry' }
        } catch { throw 'Existing boxweave entry differs; rerun with -Replace after review' }
    }
    $encoded = ($Arguments | ForEach-Object { Quote-Toml $_ }) -join ', '
    $block = '[mcp_servers.' + $McpName + "]`r`ncommand = " + (Quote-Toml $Command) + "`r`nargs = [$encoded]`r`nstartup_timeout_sec = 30`r`n"
    $text = Set-TomlSection -Source $text -Header "mcp_servers.$McpName" -Replacement $block
    return $text
}
function Update-Agents([string]$Path, [string]$Content, [switch]$Preview) {
    $begin = '<!-- BEGIN BOXWEAVE MANAGED -->'
    $end = '<!-- END BOXWEAVE MANAGED -->'
    $managed = $begin + "`r`n" + $Content.Trim() + "`r`n" + $end
    $original = if (Test-Path -LiteralPath $Path) { [System.IO.File]::ReadAllText($Path) } else { '' }
    $a = [regex]::Matches($original,[regex]::Escape($begin)).Count
    $b = [regex]::Matches($original,[regex]::Escape($end)).Count
    if ($a -ne $b -or $a -gt 1 -or ($a -eq 1 -and $original.IndexOf($end) -lt $original.IndexOf($begin))) { throw 'Malformed BoxWeave instruction markers' }
    $pattern = '(?s)' + [regex]::Escape($begin) + '.*?' + [regex]::Escape($end)
    $match = [regex]::Match($original, $pattern)
    if ($match.Success) {
        $updated = $original.Substring(0, $match.Index) + $managed + $original.Substring($match.Index + $match.Length)
    } else {
        $updated = $original.TrimEnd() + "`r`n`r`n" + $managed + "`r`n"
    }
    if ($Preview) { return $updated }
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

function Quote-ProcessArgument([string]$Value) {
    # WSL parses switch tokens from the raw command line; quoting every token can
    # turn -d/-e into a shell command. Quote only empty, whitespace or quoted values.
    if ($Value.Length -gt 0 -and $Value -notmatch '[\s"]') { return $Value }
    return '"' + [regex]::Replace([regex]::Replace($Value, '(\\*)"', '$1$1\"'), '(\\+)$', '$1$1') + '"'
}
function ConvertTo-WslScriptCommand([string]$Script, [string[]]$ScriptArguments = @()) {
    # Do not pass quoted/multiline shell source through legacy native argv serialization.
    $normalized = $Script.Replace("`r`n", "`n").Replace("`r", "`n")
    $encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($normalized))
    $command = "printf '%s' '$encoded' | base64 -d | sh -s --"
    foreach ($value in $ScriptArguments) {
        # Callers pass only validated key names and Base64 data, never arbitrary shell text.
        if ($value -notmatch '^[A-Za-z0-9_./+=:-]+$') {
            throw 'Invalid WSL script argument; expected a nonempty ASCII token'
        }
        $command += " '$value'"
    }
    return $command
}
function Invoke-WslScript([string]$Script, [string[]]$ScriptArguments = @(), [string]$What = 'WSL script') {
    $command = ConvertTo-WslScriptCommand -Script $Script -ScriptArguments $ScriptArguments
    & $wsl -d $WslDistro -e sh -c $command
    Assert-Exit $What
}

function Test-BoxWeaveMcp([string]$Command,[string[]]$Arguments) {
    $start=New-Object Diagnostics.ProcessStartInfo
    $start.FileName=$Command
    $start.Arguments=(($Arguments | ForEach-Object { Quote-ProcessArgument $_ }) -join ' ')
    $start.UseShellExecute=$false; $start.CreateNoWindow=$true
    $start.RedirectStandardInput=$true; $start.RedirectStandardOutput=$true; $start.RedirectStandardError=$true
    $process=New-Object Diagnostics.Process
    $process.StartInfo=$start
    $null=$process.Start()
    $stderrTask=$process.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null)
    $pendingBytes=New-Object 'System.Collections.Generic.List[byte]'
    $readBuffer=New-Object byte[] 4096
    $deadline=[DateTime]::UtcNow.AddSeconds(30)
    function Send-Mcp($Message) {
        $process.StandardInput.WriteLine((ConvertTo-Json -InputObject $Message -Compress -Depth 15))
        $process.StandardInput.Flush()
    }
    function Call-Mcp([int]$Id,[string]$Method,$Parameters) {
        Send-Mcp @{jsonrpc='2.0';id=$Id;method=$Method;params=$Parameters}
        for ($budget=0;$budget -lt 256;$budget++) {
            $remaining=[int]($deadline-[DateTime]::UtcNow).TotalMilliseconds
            if ($remaining -le 0) { throw 'MCP handshake/tool discovery timeout' }
            $newline=$pendingBytes.IndexOf([byte]10)
            while ($newline -lt 0) {
                $remaining=[int]($deadline-[DateTime]::UtcNow).TotalMilliseconds
                if ($remaining -le 0) { throw 'MCP handshake/tool discovery timeout' }
                $read=$process.StandardOutput.BaseStream.ReadAsync($readBuffer,0,$readBuffer.Length)
                if (-not $read.Wait($remaining)) { throw 'MCP handshake/tool discovery timeout' }
                $count=$read.Result
                if ($count -eq 0) {
                    $exitDetail = 'process still running'
                    if ($process.WaitForExit(500)) { $exitDetail = 'process exit ' + $process.ExitCode }
                    throw ('MCP stream closed before verification (' + $exitDetail + ')')
                }
                $pendingBytes.AddRange([byte[]]$readBuffer[0..($count-1)])
                if ($pendingBytes.Count -gt 2097152) { throw 'MCP response limit exceeded' }
                $newline=$pendingBytes.IndexOf([byte]10)
            }
            $line=[Text.Encoding]::UTF8.GetString($pendingBytes.GetRange(0,$newline).ToArray())
            $pendingBytes.RemoveRange(0,$newline+1)
            if (-not $line.Trim()) { continue }
            $message=ConvertFrom-Json -InputObject $line
            if ($message.id -eq $Id -and -not $message.method) {
                if ($message.PSObject.Properties['error']) { throw 'MCP protocol error' }
                return $message.result
            }
            if ($message.method -and $message.PSObject.Properties['id']) {
                Send-Mcp @{jsonrpc='2.0';id=$message.id;error=@{code=-32601;message='Unsupported client request'}}
            }
        }
        throw 'MCP message budget exceeded'
    }
    try {
        $hello=Call-Mcp 1 'initialize' @{protocolVersion='2024-11-05';capabilities=@{};clientInfo=@{name='boxweave-client';version='0.2.0'}}
        if ($hello.protocolVersion -ne '2024-11-05') { throw 'Unsupported MCP protocol' }
        Send-Mcp @{jsonrpc='2.0';method='notifications/initialized'}
        $listing=Call-Mcp 2 'tools/list' @{}
        if (-not ($listing.tools | Where-Object { $_.name -eq 'list_containers' })) { throw 'Provider list_containers tool unavailable' }
        $result=Call-Mcp 3 'tools/call' @{name='list_containers';arguments=@{}}
        if ($result.isError) { throw 'Read-only provider verification failed' }
    } finally {
        $process.StandardInput.Close()
        if (-not $process.WaitForExit(2000)) {
            & taskkill.exe /PID $process.Id /T /F 2>$null | Out-Null
            $null=$process.WaitForExit(2000)
        }
        $process.Dispose()
    }
}

if ($DataBase64) {
    $d = ConvertFrom-Json -InputObject ([Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($DataBase64)))
    if ($d.os -ne 'windows' -or $d.runtime -ne 'native' -or $d.transport -notin @('native','wsl') -or $d.agent -notin @('codex','cursor','claude-code')) { throw 'Invalid Windows client profile' }
    if ($d.ssh_port -lt 1 -or $d.ssh_port -gt 65535) { throw 'Invalid SSH port' }
    if ($d.setup_options) {
        $opts=$d.setup_options
        if (-not $BootstrapIdentityFile -and $opts.bootstrap_identity) { $BootstrapIdentityFile=$opts.bootstrap_identity }
        if (-not $AgentHome -and $opts.agent_home) { $AgentHome=$opts.agent_home }
        if (-not $ProjectPath -and $opts.project_path) { $ProjectPath=$opts.project_path }
        if (-not $WslDistro -and $opts.wsl_distro) { $WslDistro=$opts.wsl_distro }
        if ($opts.replace -eq $true) { $Replace=$true }
    }
    if (-not $d.policy -or $d.policy.Length -gt 20000) { throw 'Invalid guidance text' }
    $SshHost=$d.ssh_host; $SshPort=$d.ssh_port; $SshUser=$d.linux_user
    $Mode=$d.transport; $Agent=$d.agent; $McpRemoteCommand=$d.launcher
    if ($d.server_public_key -notmatch '^ssh-ed25519 ([A-Za-z0-9+/=]+)(?: [^\r\n]+)?$') { throw 'Invalid server key' }
    $ServerHostKey=$Matches[1]
    $BoxPublicKeyBase64=[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($d.container_public_key))
}
if ($Mode -eq 'WSL' -and -not $WslDistro) {
    $distros=@(wsl.exe -l -q | ForEach-Object { ($_ -replace [char]0,'').Trim() } | Where-Object { $_ })
    Assert-Exit 'WSL distribution discovery'
    if ($distros.Count -eq 1) { $WslDistro=$distros[0] }
    elseif ($distros.Count -gt 1) {
        for ($i=0;$i -lt $distros.Count;$i++) { Write-Host ("{0}) {1}" -f ($i+1),$distros[$i]) }
        $answer=Read-Host 'Select WSL distribution number'
        if ($answer -notmatch '^[0-9]+$' -or [int]$answer -lt 1 -or [int]$answer -gt $distros.Count) { throw 'Invalid selection; rerun with -WslDistro NAME' }
        $WslDistro=$distros[[int]$answer-1]
    } else { throw 'No installed WSL distribution' }
}
$claude = if ($Agent -eq 'Claude-Code') { (Get-Command claude -ErrorAction Stop).Source } else { $null }
if ($claude -and $AgentHome) { throw 'Set CLAUDE_CONFIG_DIR instead of -AgentHome for Claude Code' }
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
$codexDir = if ($AgentHome) { $AgentHome } elseif ($Agent -eq 'Cursor') { Join-Path $env:USERPROFILE '.cursor' } elseif ($Agent -eq 'Claude-Code') { if ($env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR } else { Join-Path $env:USERPROFILE '.claude' } } elseif ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $env:USERPROFILE '.codex' }
foreach ($candidate in @($sshDir,$codexDir)) {
    $ancestor=$candidate
    while ($ancestor) {
        if ((Test-Path -LiteralPath $ancestor) -and ((Get-Item -LiteralPath $ancestor).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing reparse point in setup path' }
        $ancestor=Split-Path -Parent $ancestor
    }
}
if ($BootstrapIdentityFile -and -not (Test-Path -LiteralPath $BootstrapIdentityFile -PathType Leaf)) { throw 'Bootstrap identity file not found' }
New-Item -ItemType Directory -Force $sshDir, $codexDir | Out-Null
$nativePrivate = Join-Path $sshDir $KeyName
$nativePublic = "$nativePrivate.pub"
$nativeKnown = Join-Path $sshDir "${KeyName}_known_hosts"
$configPath = Join-Path $codexDir 'config.toml'
$agentsPath = Join-Path $codexDir 'AGENTS.override.md'
if (-not (Test-Path -LiteralPath $agentsPath) -or -not ([IO.File]::ReadAllText($agentsPath)).Trim()) { $agentsPath = Join-Path $codexDir 'AGENTS.md' }

foreach ($candidate in @($nativePrivate,$nativePublic,$nativeKnown,$configPath,$agentsPath)) {
    if ((Test-Path -LiteralPath $candidate) -and ((Get-Item -LiteralPath $candidate).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing identity/configuration reparse point' }
}
$nativePubLine = ''
if ($Mode -eq 'Native') {
if ((Test-Path -LiteralPath $nativePrivate) -xor (Test-Path -LiteralPath $nativePublic)) {
    throw 'A Windows keypair is incomplete; refusing to overwrite the remaining file.'
}
if (-not (Test-Path -LiteralPath $nativePrivate)) {
    Write-Host '[boxweave] Generating a dedicated private/public keypair on Windows.'
    # Explicit Windows argv quoting preserves the empty passphrase on both 5.1 and 7+.
    $keyStart=New-Object Diagnostics.ProcessStartInfo
    $keyStart.FileName=$sshKeygen
    $keyStart.UseShellExecute=$false; $keyStart.CreateNoWindow=$true
    $keyStart.Arguments=(@('-q','-t','ed25519','-f',$nativePrivate,'-N','','-C','boxweave') | ForEach-Object { Quote-ProcessArgument $_ }) -join ' '
    $keyProcess=[Diagnostics.Process]::Start($keyStart)
    try {
        if (-not $keyProcess.WaitForExit(30000)) { $keyProcess.Kill(); throw 'SSH key generation timeout' }
        if ($keyProcess.ExitCode -ne 0) { throw ('SSH key generation failed (exit '+$keyProcess.ExitCode+')') }
    } finally { $keyProcess.Dispose() }
}
& $sshKeygen -lf $nativePublic -E sha256 | Out-Host
Assert-Exit 'Windows SSH public key validation'
$nativePubLine = [System.IO.File]::ReadAllText($nativePublic).Trim()
if ($nativePubLine -notmatch '^ssh-ed25519\s+[a-zA-Z0-9+/=]+') { throw 'Invalid Windows SSH public key.' }

}
# Pin SSH server identity using the key delivered by the Ubuntu console,
# never using untrusted ssh-keyscan or StrictHostKeyChecking=no.
$hostToken = if ($SshPort -eq 22 -and -not $SshHost.Contains(':')) { $SshHost } else { '[{0}]:{1}' -f $SshHost, $SshPort }
$pinnedHostLine = "$hostToken ssh-ed25519 $ServerHostKey"
Set-PinnedHostNative -Path $nativeKnown -Line $pinnedHostLine

$clientPubLine = $nativePubLine
$transportCommand = $ssh
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
test ! -L "$key" && test ! -L "$key.pub"
if [ -e "$key" ] && [ ! -e "$key.pub" ] || [ -e "$key.pub" ] && [ ! -e "$key" ]; then
    echo 'Incomplete WSL SSH keypair; refusing overwrite' >&2
    exit 2
fi
if [ ! -e "$key" ]; then
    ssh-keygen -q -t ed25519 -f "$key" -N '' -C 'boxweave-wsl'
fi
chmod 600 "$key"
ssh-keygen -lf "$key.pub" -E sha256 >/dev/null
cat "$key.pub"
'@
    Write-Host "[boxweave] Generating/reusing key in WSL distro: $WslDistro"
    $wslLines = Invoke-WslScript -Script $wslScript -ScriptArguments @($KeyName) -What 'WSL key generation'
    $clientPubLine = ($wslLines | Where-Object { $_ -match '^ssh-ed25519\s+' } | Select-Object -Last 1).Trim()
    if (-not $clientPubLine) { throw 'WSL did not return a valid SSH public key.' }

    $wslHomeLines = Invoke-WslScript -Script 'printf "%s\n" "$HOME"' -What 'WSL home lookup'
    $wslHome = ($wslHomeLines | Select-Object -Last 1).Trim()
    if ($wslHome -notmatch '^/[^\r\n]+$') { throw 'Invalid WSL home path' }

    $wslPinScript = @'
set -eu
umask 077
path="$HOME/.ssh/$1"
test ! -L "$path"
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
    Invoke-WslScript -Script $wslPinScript -ScriptArguments @(("${KeyName}_known_hosts"), $pinB64) -What 'WSL host pinning'
    $linuxKey = "$wslHome/.ssh/$KeyName"
    $linuxKnown = "$wslHome/.ssh/${KeyName}_known_hosts"
    $transportCommand = $wsl
    $transportArgs = @('-d', $WslDistro, '-e', 'ssh', '-T', '-i', $linuxKey,
                       '-o', 'IdentitiesOnly=yes', '-o', 'BatchMode=yes',
                       '-o', 'StrictHostKeyChecking=yes', '-o', "UserKnownHostsFile=$linuxKnown",
                       '-o', 'HostKeyAlgorithms=ssh-ed25519', '-o', 'ConnectTimeout=15', '-p', "$SshPort",
                       "$SshUser@$SshHost", $McpRemoteCommand)
} else {
    $transportArgs = @('-T', '-i', $nativePrivate,
                       '-o', 'IdentitiesOnly=yes', '-o', 'BatchMode=yes',
                       '-o', 'StrictHostKeyChecking=yes', '-o', "UserKnownHostsFile=$nativeKnown",
                       '-o', 'HostKeyAlgorithms=ssh-ed25519', '-o', 'ConnectTimeout=15', '-p', "$SshPort",
                       "$SshUser@$SshHost", $McpRemoteCommand)
}

$pendingConfig = ''
if ($Agent -eq 'Codex') {
    $pendingConfig = Update-CodexToml -Path $configPath -Command $transportCommand -Arguments $transportArgs
    if ($DataBase64) { $null=Update-Agents -Path $agentsPath -Content $d.policy -Preview }
} elseif ($Agent -eq 'Cursor') {
    $configPath=Join-Path $codexDir 'mcp.json'
    $obj=if (Test-Path -LiteralPath $configPath) { ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($configPath)) } else { New-Object PSObject }
    if ($null -eq $obj -or $obj -is [array] -or $obj -is [string] -or $obj -is [ValueType]) { throw 'Invalid Cursor JSON object' }
    if (-not $obj.PSObject.Properties['mcpServers']) { $obj | Add-Member -NotePropertyName mcpServers -NotePropertyValue (New-Object PSObject) }
    $servers=$obj.mcpServers
    if ($null -eq $servers -or $servers -is [array] -or $servers -is [string] -or $servers -is [ValueType]) { throw 'Invalid Cursor MCP object' }
    $old=$servers.PSObject.Properties[$McpName]
    if ($old -and -not $Replace -and ($old.Value.command -ne $transportCommand -or (ConvertTo-Json -InputObject @($old.Value.args) -Compress) -cne (ConvertTo-Json -InputObject @($transportArgs) -Compress))) { throw 'Existing boxweave entry differs; rerun with -Replace after review' }
    $servers | Add-Member -NotePropertyName $McpName -NotePropertyValue ([pscustomobject]@{command=$transportCommand;args=@($transportArgs)}) -Force
    $pendingConfig=ConvertTo-Json -InputObject $obj -Depth 100
    if ($ProjectPath -and -not (Test-Path -LiteralPath $ProjectPath -PathType Container)) { throw 'Cursor project path must be an existing local directory' }
} else {
    & $claude mcp get $McpName 1>$null 2>$null
    $existingClaude=($LASTEXITCODE -eq 0)
    if ($existingClaude -and -not $Replace) { throw 'Existing Claude boxweave entry; rerun with -Replace after review' }
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
test ! -L "$HOME/.ssh/authorized_keys"
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
                 '-o', 'HostKeyAlgorithms=ssh-ed25519', '-o', 'ConnectTimeout=15', '-p', "$SshPort")
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

$agents = @"
## BoxWeave remote development environment (Containarium provider)

- Use MCP server **$McpName** (BoxWeave with the Containarium provider) for provisioning and managing LXC containers. Use its dynamic ``connect`` tool inside an existing box, not a separate per-box MCP registration.
- Ubuntu SSH endpoint: **${SshHost}:${SshPort}**, Unix user: **${SshUser}**. SSH transport mode: **$Mode**. The dedicated transport key is restricted by OpenSSH ``restrict,command=``; it is NOT a general Ubuntu shell credential.
- The managed Ubuntu-to-container public SSH key is exactly:

  $boxPub

- When creating a container that requires ``connect``, pass that COMPLETE public-key line via ``create_container.ssh_keys``. Never substitute the Windows/WSL transport key. Do not assume a public-key fingerprint is enough.
- Reuse a suitable existing project container; otherwise provision an isolated box. Query box state before connecting and verify actual results afterward.
- A timed-out create request can still succeed. Query ``list_containers``/``get_container`` with the original logical name before any new create. Do NOT duplicate, rename, force-delete or overwrite a partially provisioned resource.
- Running does not prove SSH provisioning is complete. Allow package/key setup to finish, retry ``connect`` within the deadline, and verify ``whoami && hostname`` before work. Diagnose persistent failures with read-only tools, including ``debug_container`` when supported.
- Unless the user specifies another limit, use one 600-second deadline per provisioning, installation or readiness phase, polling about every 15-30 seconds. Do not reset the allowance on each retry. Preserve background job logs and actual exit codes; transport timeouts or disconnects do not establish completion. The installer's ``--api-ready-timeout`` bounds only API waiting, not total installation time.
- An active Containarium daemon may still be waiting for core containers/PostgreSQL before the API listens. Inspect progress and wait rather than restarting or deleting infrastructure. HTTP 200/401/403 proves reachability only; authenticated operation requires real MCP calls.
- Use ``sudo boxweave status`` / ``sudo boxweave doctor --api`` when configuration permissions require it. Exit 0 is an observation: inspect service states, required timers, installer exit code and ``/etc/boxweave/update-in-progress``. A successful oneshot can be inactive/dead; check Result=success and ExecMainStatus=0, with its timer active/waiting and enabled. Only reset failed state for actually failed units.
- After recreation, ``knownhosts: key mismatch`` can indicate IP reuse; wait for the host-side trusted Incus synchronization and retry within a bounded window. Never disable SSH verification or edit Ubuntu ``known_hosts`` from Codex.
- For a requested nested smoke test, use ordinary Ubuntu LXC -> inner BoxWeave MCP -> disposable workload LXC -> Python hello world. Do not make special images, Incus upgrades, cold backups or security audits prerequisites. Reuse compatible Incus; report security capabilities separately when unverified.
- Each nested MCP host has its own managed Ubuntu-to-container public identity; discover and verify that layer's FULL public key instead of assuming the outer key above applies. Never read/export private keys or JWT contents. Start the inner managed launcher as its configured non-root user and use an MCP client with initialize/initialized, tool discovery and tools/call. Direct Incus execution is not proof of inner MCP operation; verify identity and stdout/exit code through inner ``connect``.
- Develop, test and deploy applications INSIDE each project's LXC. Prefer non-root container user and use systemd/Compose for persistence if appropriate. Verify health, reboot persistence and requested LAN access.
- Use Containarium port passthrough only when explicitly required. Never assume it provides LAN-only source-IP enforcement; protect sensitive services with authentication/network restrictions.
- Never modify ``containarium-core-*`` infrastructure boxes, unrelated projects, host SSH settings, root access, or platform credentials. Obtain permission for destructive operations.
- The MCP token may have admin privileges; use only requested tools and scopes, never print secrets or private keys.
"@
if ($DataBase64) {
    if (-not $d.policy -or $d.policy.Length -gt 20000) { throw 'Invalid guidance text' }
    $agents=$d.policy
}
Test-BoxWeaveMcp -Command $transportCommand -Arguments $transportArgs
if ($Agent -eq 'Claude-Code') {
    if ($existingClaude -and $Replace) {
        & $claude mcp remove --scope user $McpName 1>$null 2>$null
        Assert-Exit 'Claude MCP replacement'
    }
    & $claude mcp add --scope user --transport stdio $McpName -- $transportCommand @transportArgs 1>$null 2>$null
    Assert-Exit 'Claude MCP registration'
    $rulesDir=Join-Path $codexDir 'rules'
    $null=New-Item -ItemType Directory -Force -Path $rulesDir
    $agentsPath=Join-Path $rulesDir 'boxweave.md'
    Backup-File $agentsPath
    Write-Utf8NoBom $agentsPath $agents
} else {
    Backup-File $configPath
    Write-Utf8NoBom $configPath $pendingConfig
    if ($Agent -eq 'Cursor') {
        $agentsPath=Join-Path $codexDir 'boxweave-user-rules.md'
        Backup-File $agentsPath
        Write-Utf8NoBom $agentsPath $agents
        if ($ProjectPath) {
            $rulesDir=Join-Path $ProjectPath '.cursor/rules'
            $null=New-Item -ItemType Directory -Force -Path $rulesDir
            $projectRule=Join-Path $rulesDir 'boxweave.mdc'
            Backup-File $projectRule
            Write-Utf8NoBom $projectRule ("---`nalwaysApply: true`n---`n`n"+$agents)
        }
    } else { Update-Agents -Path $agentsPath -Content $agents }
}
Write-Host 'MCP initialize/tools/list/read-only provider call: passed.'
Write-Host "Agent registration: written for $Agent. Reopen the agent and verify BoxWeave in its MCP UI."
if ($Agent -eq 'Cursor' -and -not $ProjectPath) {
    Write-Host "Guidance: manual step required. Paste $agentsPath into Cursor Settings > Rules > User Rules."
} else { Write-Host 'Guidance: written; verify loading in a new session.' }
