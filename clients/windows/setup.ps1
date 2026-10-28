#requires -version 5.1
<#
BoxWeave client enrollment for native Windows and Windows-native agents with a
WSL SSH transport. All connection parameters arrive through BOXWEAVE_* env
vars set by the generated enrollment script; nothing is base64-packed.

The SSH transport is addressed through a managed "Host boxweave" block in the
user's SSH config (native: %USERPROFILE%\.ssh\config, WSL: the distro's
~/.ssh/config), so agent registrations stay "ssh boxweave /usr/local/bin/boxweave-mcp"
and keep working when the server address changes: edit the block (or the
generated script's EDIT ME header) and rerun.

Switches:
  -Replace                replace a reviewed boxweave MCP registration
  -Diagnose               read-only connection/registration diagnostic
  -ReportPath PATH        write the diagnostic JSON report (with -Diagnose)
  -Timeout SECONDS        diagnostic protocol budget (1-300, default 30)
  -BootstrapIdentityFile  existing authorized SSH identity for enrollment
  -AgentHome PATH         agent configuration directory override
  -ProjectPath PATH       local Cursor workspace for a project rule
#>
[CmdletBinding()]
param(
    [switch]$Diagnose,
    [ValidateRange(1,300)][double]$Timeout = 30,
    [string]$ReportPath = '',
    [string]$ProjectPath = '',
    [switch]$Replace,
    [string]$BootstrapIdentityFile = '',
    [string]$AgentHome = '',
    [string]$McpName = 'boxweave',
    [string]$KeyName = 'boxweave'
)
$ErrorActionPreference = 'Stop'

$Script:McpRemoteCommand = '/usr/local/bin/boxweave-mcp'

function Update-ZCodeJson([string]$Original, [string]$Command, [string[]]$Arguments, [bool]$AllowReplace) {
    $obj=if ($Original.Trim()) { ConvertFrom-Json -InputObject $Original } else { [pscustomobject]@{} }
    if ($obj -isnot [pscustomobject]) { throw 'Invalid ZCode JSON object' }
    if (-not $obj.PSObject.Properties['mcp']) { $obj | Add-Member NoteProperty mcp ([pscustomobject]@{}) }
    if ($obj.mcp -isnot [pscustomobject]) { throw 'Invalid ZCode MCP object' }
    if (-not $obj.mcp.PSObject.Properties['servers']) { $obj.mcp | Add-Member NoteProperty servers ([pscustomobject]@{}) }
    if ($obj.mcp.servers -isnot [pscustomobject]) { throw 'Invalid ZCode MCP servers' }
    if ($obj.PSObject.Properties['features'] -and ($obj.features -isnot [pscustomobject] -or
        ($obj.features.PSObject.Properties['mcp'] -and $obj.features.mcp -cne $true))) { throw 'Enable MCP in ZCode before setup' }
    $old=$obj.mcp.servers.PSObject.Properties['boxweave']
    if ($old -and -not $AllowReplace) {
        $entry=$old.Value
        if ($entry -isnot [pscustomobject] -or $entry.type -cne 'stdio' -or $entry.command -cne $Command -or
            ($entry.args -isnot [array]) -or
            (ConvertTo-Json -InputObject @($entry.args) -Compress) -cne (ConvertTo-Json -InputObject @($Arguments) -Compress) -or
            $entry.timeoutMs -cne 180000 -or
            @($entry.PSObject.Properties.Name | Where-Object {$_ -notin @('type','command','args','timeoutMs')}).Count) {
            throw 'Existing ZCode boxweave entry differs; rerun with -Replace after review'
        }
    }
    $obj.mcp.servers | Add-Member -Force NoteProperty boxweave ([pscustomobject]@{type='stdio';command=$Command;args=@($Arguments);timeoutMs=180000})
    return ConvertTo-Json -InputObject $obj -Depth 100
}
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
    if ($text -match '(?m)^\s*mcp_servers\s*=') { throw 'Inline MCP TOML requires explicit migration' }
    # Codex may add per-tool approval settings. Updating the transport must
    # preserve those tables byte-for-byte, rather than rejecting or removing them.
    $nested = [regex]::Matches($text, ('(?m)^\[mcp_servers\.' + [regex]::Escape($McpName) + '\.([^\]\r\n]+)\]'))
    foreach ($table in $nested) {
        if ($table.Groups[1].Value -notmatch '^tools(?:\.[A-Za-z0-9_-]+)*$') { throw 'Nested BoxWeave MCP settings require explicit migration' }
    }
    $header = '(?ms)^\[mcp_servers\.' + [regex]::Escape($McpName) + '\][^\r\n]*\r?\n.*?(?=^\[|\z)'
    $old = [regex]::Matches($text,$header)
    if ($old.Count -gt 1) { throw 'Duplicate BoxWeave MCP table' }
    if ($text -match [regex]::Escape($McpName) -and $old.Count -eq 0) { throw 'Unsupported existing BoxWeave TOML spelling' }
    if ($old.Count -eq 1 -and -not $Replace) {
        try {
            $extra=@($old[0].Value -split '\r?\n' | Where-Object { $_ -notmatch '^\s*(#|$|\[|command\s*=|args\s*=|startup_timeout_sec\s*=|tool_timeout_sec\s*=)' })
            if ($extra.Count) { throw 'Additional settings need explicit replacement' }
            $c = [regex]::Match($old[0].Value,'(?m)^command\s*=\s*(.+?)\s*$').Groups[1].Value | ConvertFrom-Json
            # Windows PowerShell 5.1 wraps a top-level JSON array in a PSObject
            # with value/Count properties; unwrap it before comparing.
            $aRaw = ConvertFrom-Json -InputObject ([regex]::Match($old[0].Value,'(?m)^args\s*=\s*(.+?)\s*$').Groups[1].Value)
            if ($null -ne $aRaw.PSObject.Properties['value'] -and $null -ne $aRaw.PSObject.Properties['Count'] -and $aRaw -isnot [array]) {
                $a = @($aRaw.value)
            } else {
                $a = @($aRaw)
            }
            if ($c -ne $Command -or (ConvertTo-Json -InputObject $a -Compress) -cne (ConvertTo-Json -InputObject @($Arguments) -Compress)) { throw 'Different entry' }
        } catch { throw 'Existing boxweave entry differs; rerun with -Replace after review' }
    }
    $encoded = ($Arguments | ForEach-Object { Quote-Toml $_ }) -join ', '
    $block = '[mcp_servers.' + $McpName + "]`r`ncommand = " + (Quote-Toml $Command) + "`r`nargs = [$encoded]`r`nstartup_timeout_sec = 30`r`ntool_timeout_sec = 180`r`n"
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
function Update-SshConfigManaged([string]$Path, [string]$Block) {
    # Idempotently place the Host boxweave transport block between markers.
    $begin = '# BEGIN BOXWEAVE MANAGED'
    $end = '# END BOXWEAVE MANAGED'
    $managed = $begin + "`r`n" + $Block.Trim() + "`r`n" + $end
    $original = if (Test-Path -LiteralPath $Path) { [System.IO.File]::ReadAllText($Path) } else { '' }
    $a = [regex]::Matches($original,[regex]::Escape($begin)).Count
    $b = [regex]::Matches($original,[regex]::Escape($end)).Count
    if ($a -ne $b -or $a -gt 1 -or ($a -eq 1 -and $original.IndexOf($end) -lt $original.IndexOf($begin))) { throw 'Malformed BoxWeave ssh_config markers' }
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
function New-SshConfigBlock([string]$HostName, [int]$Port, [string]$User, [string]$IdentityFile, [string]$KnownHostsFile) {
    return ('Host ' + $McpName + "`r`n" +
        '    HostName ' + $HostName + "`r`n" +
        '    Port ' + $Port + "`r`n" +
        '    User ' + $User + "`r`n" +
        '    IdentityFile ' + $IdentityFile + "`r`n" +
        '    IdentitiesOnly yes' + "`r`n" +
        '    IdentityAgent none' + "`r`n" +
        '    BatchMode yes' + "`r`n" +
        '    UserKnownHostsFile ' + $KnownHostsFile + "`r`n" +
        '    StrictHostKeyChecking yes' + "`r`n" +
        '    HostKeyAlgorithms ssh-ed25519' + "`r`n" +
        '    ConnectTimeout 15' + "`r`n" +
        '    ServerAliveInterval 15' + "`r`n" +
        '    ServerAliveCountMax 4')
}
function Set-PinnedHostNative([string]$Path, [string]$Line) {
    $expected=$Line.Trim() -split '\s+'
    if ($expected.Count -ne 3 -or $expected[1] -cne 'ssh-ed25519') { throw 'Invalid expected host pin' }
    if (Test-Path -LiteralPath $Path) {
        # OpenSSH resolves exact, comma-separated and hashed host names.
        $records=@(& ssh-keygen -F $expected[0] -f $Path 2>$null)
        $lookupExit=$LASTEXITCODE
        if ($lookupExit -notin @(0,1)) { throw 'Could not inspect pinned host file' }
        $found=$false
        foreach ($record in $records) {
            $text=([string]$record).Trim()
            if (-not $text -or $text.StartsWith('#')) { continue }
            $parts=$text -split '\s+'
            if ($parts.Count -lt 3 -or $parts[1] -cne $expected[1] -or $parts[2] -cne $expected[2]) {
                throw "Host public key changed! Verify the new Ubuntu fingerprint out-of-band before replacing $Path"
            }
            $found=$true
        }
        if ($found) { return }
        $original=[IO.File]::ReadAllText($Path)
        Backup-File $Path
        Write-Utf8NoBom $Path ($original.TrimEnd()+[Environment]::NewLine+$Line+[Environment]::NewLine)
    } else {
        Write-Utf8NoBom $Path ($Line + [Environment]::NewLine)
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

function Test-BoxWeaveMcp([string]$Command,[string[]]$Arguments,[double]$Timeout=30,[hashtable]$Evidence=@{}) {
    $clock=[Diagnostics.Stopwatch]::StartNew()
    $stage='transport'
    $start=New-Object Diagnostics.ProcessStartInfo
    $start.FileName=$Command
    $start.Arguments=(@($Arguments | ForEach-Object { Quote-ProcessArgument $_ }) -join ' ')
    $start.UseShellExecute=$false; $start.CreateNoWindow=$true
    $start.RedirectStandardInput=$true; $start.RedirectStandardOutput=$true; $start.RedirectStandardError=$true
    $process=New-Object Diagnostics.Process
    $process.StartInfo=$start
    $null=$process.Start()
    $Evidence.process_exit_code=$null
    $stderrTask=$process.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null)
    $pendingBytes=New-Object 'System.Collections.Generic.List[byte]'
    $readBuffer=New-Object byte[] 4096
    function Remaining-Mcp {
        $left=[int]($Timeout*1000-$clock.ElapsedMilliseconds)
        if ($left -le 0) { throw 'DEADLINE_EXCEEDED: MCP timeout' }
        return $left
    }
    function Send-Mcp($Message) {
        $Evidence.write_pending=$true
        $task=$process.StandardInput.WriteLineAsync((ConvertTo-Json -InputObject $Message -Compress -Depth 15))
        if (-not $task.Wait((Remaining-Mcp))) { throw 'DEADLINE_EXCEEDED: MCP timeout' }
        $task=$process.StandardInput.FlushAsync()
        if (-not $task.Wait((Remaining-Mcp))) { throw 'DEADLINE_EXCEEDED: MCP timeout' }
        $Evidence.write_pending=$false
    }
    function Call-Mcp([int]$Id,[string]$Method,$Parameters) {
        Send-Mcp @{jsonrpc='2.0';id=$Id;method=$Method;params=$Parameters}
        for ($budget=0;$budget -lt 256;$budget++) {
            $newline=$pendingBytes.IndexOf([byte]10)
            while ($newline -lt 0) {
                $read=$process.StandardOutput.BaseStream.ReadAsync($readBuffer,0,$readBuffer.Length)
                if (-not $read.Wait((Remaining-Mcp))) { throw 'DEADLINE_EXCEEDED: MCP timeout' }
                $count=$read.Result
                if ($count -eq 0) {
                    $exitDetail='process still running'
                    if ($process.WaitForExit(100)) { $Evidence.process_exit_code=$process.ExitCode;$exitDetail='process exit '+$process.ExitCode }
                    throw ('TRANSPORT_FAILED: MCP stream closed before verification ('+$exitDetail+')')
                }
                $pendingBytes.AddRange([byte[]]$readBuffer[0..($count-1)])
                $Evidence.bytes=[long]$Evidence.bytes+$count
                if ($pendingBytes.Count -gt 2097152 -or $Evidence.bytes -gt 8388608) { throw 'RESPONSE_LIMIT: MCP response limit exceeded' }
                $newline=$pendingBytes.IndexOf([byte]10)
            }
            $line=[Text.Encoding]::UTF8.GetString($pendingBytes.GetRange(0,$newline).ToArray())
            $pendingBytes.RemoveRange(0,$newline+1)
            if (-not $line.Trim()) { continue }
            try { $message=ConvertFrom-Json -InputObject $line -ErrorAction Stop } catch { throw 'PROTOCOL_INVALID: Invalid MCP response' }
            if (-not ($message -is [pscustomobject]) -or $message.jsonrpc -cne '2.0') { throw 'PROTOCOL_INVALID: Invalid MCP envelope' }
            if ($message.PSObject.Properties['method']) {
                if ($message.PSObject.Properties['id']) { Send-Mcp @{jsonrpc='2.0';id=$message.id;error=@{code=-32601;message='Unsupported client request'}} }
                continue
            }
            if ((-not ($message.id -is [int]) -and -not ($message.id -is [long])) -or $message.id -ne $Id) { throw 'PROTOCOL_INVALID: Invalid response ID' }
            $hasResult=$null -ne $message.PSObject.Properties['result']
            $hasError=$null -ne $message.PSObject.Properties['error']
            if ($hasResult -eq $hasError) { throw 'PROTOCOL_INVALID: Invalid MCP result' }
            if ($hasError) { throw 'PROVIDER_CALL_FAILED: MCP protocol error' }
            if (-not ($message.result -is [pscustomobject])) { throw 'PROTOCOL_INVALID: Invalid MCP result' }
            return $message.result
        }
        throw 'RESPONSE_LIMIT: MCP message budget exceeded'
    }
    try {
        $stage='initialize'; $Evidence.stage=$stage
        $hello=Call-Mcp 1 'initialize' @{protocolVersion='2024-11-05';capabilities=@{};clientInfo=@{name='boxweave-client';version='0.4.0'}}
        if ($hello.protocolVersion -cne '2024-11-05') { throw 'PROTOCOL_UNSUPPORTED: Unsupported MCP protocol' }
        if(-not ($hello.capabilities -is [pscustomobject]) -or -not ($hello.capabilities.tools -is [pscustomobject])){throw 'PROTOCOL_INVALID: Tools capability missing'}
        $Evidence.protocol_version=$hello.protocolVersion; $Evidence.initialize='passed'
        Send-Mcp @{jsonrpc='2.0';method='notifications/initialized'}
        $stage='discovery'; $Evidence.stage=$stage
        $parameters=@{}; $seen=New-Object 'System.Collections.Generic.HashSet[string]'
        $id=2; $found=$false
        for($page=0;$page -lt 16;$page++){
            $listing=Call-Mcp $id 'tools/list' $parameters; $id++
            if (-not ($listing.tools -is [array])) { throw 'PROTOCOL_INVALID: Invalid tools list' }
            foreach($tool in $listing.tools){
                if (-not ($tool -is [pscustomobject]) -or -not ($tool.name -is [string])) { throw 'PROTOCOL_INVALID: Invalid tool' }
                if ($tool.name -ceq 'list_containers') { $found=$true }
            }
            $Evidence.tool_pages=$page+1
            if($found){break}
            if(-not $listing.PSObject.Properties['nextCursor']){throw 'TOOL_UNAVAILABLE: Provider list_containers tool unavailable'}
            if(-not ($listing.nextCursor -is [string]) -or -not $listing.nextCursor -or -not $seen.Add($listing.nextCursor)){throw 'PROTOCOL_INVALID: Invalid discovery cursor'}
            $parameters=@{cursor=$listing.nextCursor}
        }
        if(-not $found){throw 'RESPONSE_LIMIT: MCP page limit exceeded'}
        $Evidence.discovery='passed'
        $stage='provider_call'; $Evidence.stage=$stage
        $result=Call-Mcp $id 'tools/call' @{name='list_containers';arguments=@{}}
        if($result.PSObject.Properties['isError'] -and -not ($result.isError -is [bool])){throw 'PROTOCOL_INVALID: Invalid tool result'}
        if($result.isError){throw 'PROVIDER_CALL_FAILED: Read-only provider verification failed'}
        if(-not ($result.content -is [array])){throw 'PROTOCOL_INVALID: Invalid tool content'}
        foreach($block in $result.content){
            if(-not ($block -is [pscustomobject]) -or $block.type -cnotin @('text','image','audio','resource','resource_link')){throw 'PROTOCOL_INVALID: Invalid content block'}
            if($block.type -ceq 'text' -and -not ($block.text -is [string])){throw 'PROTOCOL_INVALID: Invalid text block'}
            if($block.type -cin @('image','audio') -and (-not ($block.data -is [string]) -or -not ($block.mimeType -is [string]))){throw 'PROTOCOL_INVALID: Invalid media block'}
            if($block.type -ceq 'resource' -and -not ($block.resource -is [pscustomobject])){throw 'PROTOCOL_INVALID: Invalid resource block'}
            if($block.type -ceq 'resource_link' -and -not ($block.uri -is [string])){throw 'PROTOCOL_INVALID: Invalid resource link'}
        }
        $Evidence.provider_call='passed'
    } finally {
        # Async writes may still be pending; dispose after bounded tree termination.
        if (-not $Evidence.write_pending) { try {$process.StandardInput.Close()} catch {} }
        if (-not $process.WaitForExit(1000)) {
            try { & taskkill.exe /PID $process.Id /T /F 2>$null | Out-Null } catch {}
            if (-not $process.HasExited) { try {$process.Kill()} catch {} }
            $null=$process.WaitForExit(2000)
        }
        $Evidence.cleanup=if($process.HasExited){'completed'}else{'incomplete'}
        $Evidence.elapsed_ms=$clock.ElapsedMilliseconds
        $process.Dispose()
    }
}

$policyTemplate = @'
## BoxWeave remote development (Containarium)
- Use MCP server boxweave to list/provision containers and its dynamic connect tool to work inside a box.
- Develop, install dependencies, test and deploy INSIDE the assigned project LXC. Inspect existing files/Git state and preserve uncommitted work.
- Inspect existing resources first; reuse a suitable project box. Never modify unrelated or containarium-core-* boxes.
- The Ubuntu MCP endpoint is {ssh_host}:{ssh_port}, user {linux_user}. The transport key is restricted to the MCP launcher, not a general host shell credential.
- For create_container.ssh_keys, explicitly supply this FULL managed Ubuntu-to-container public key:
  {container_public_key}
- Never substitute a client transport key, a fingerprint, or another nesting layer's key. Never read/export private keys or JWTs.
- A create timeout is ambiguous: inspect the original logical name before retrying; never duplicate or force-recreate to bypass a timeout.
- Running does not prove SSH readiness. Use one deadline per phase (normally 600 seconds), poll every 15-30 seconds, and verify connect with whoami && hostname before project work. Report actual stdout and exit codes.
- After IP reuse, allow bounded trusted host-key reconciliation. Never disable host verification or edit known_hosts to bypass a mismatch. Diagnose persistent failures read-only.
- Service activity, HTTP 200/401/403 and CLI exit 0 do not prove authenticated MCP execution. Test initialize/initialized, tools/list and an actual read-only tools/call.
- Keep identities separate at each nested MCP host; direct Incus execution is not evidence of nested authenticated MCP execution.
- Use systemd/Compose only for requested persistent workloads; verify health and application autostart. LXC boot.autostart requires separate evidence.
- Publish ports only when requested. Inspect routes/conflicts and verify actual client connectivity. Passthrough does not provide LAN-only source filtering or TLS.
- Obtain explicit approval for destructive/host-wide changes or public exposure. Never change host SSH/firewall/provider credentials to make a test pass.
'@

function Get-ConnectionFromEnv {
    $hostName=$env:BOXWEAVE_SSH_HOST; $portText=$env:BOXWEAVE_SSH_PORT; $user=$env:BOXWEAVE_SSH_USER
    $hostKey=$env:BOXWEAVE_SERVER_HOST_KEY; $containerKey=$env:BOXWEAVE_CONTAINER_PUBLIC_KEY
    $agent=$env:BOXWEAVE_AGENT; $transport=$env:BOXWEAVE_TRANSPORT; $distro=$env:BOXWEAVE_WSL_DISTRO
    if (-not $hostName -or $hostName -cnotmatch '^[A-Za-z0-9][A-Za-z0-9.:-]{0,252}$') { throw 'Invalid or missing BOXWEAVE_SSH_HOST' }
    if (-not $portText -or $portText -cnotmatch '^[0-9]+$' -or [int]$portText -lt 1 -or [int]$portText -gt 65535) { throw 'Invalid or missing BOXWEAVE_SSH_PORT' }
    if (-not $user -or $user -cnotmatch '^[a-z_][a-z0-9_-]{0,63}$') { throw 'Invalid or missing BOXWEAVE_SSH_USER' }
    if (-not $hostKey -or $hostKey -cnotmatch '^ssh-ed25519 ([A-Za-z0-9+/=]+)(?: [^\r\n]+)?$') { throw 'Invalid or missing BOXWEAVE_SERVER_HOST_KEY' }
    $keyBlob=[Convert]::FromBase64String($Matches[1])
    if ($keyBlob.Length -ne 51 -or [Text.Encoding]::ASCII.GetString($keyBlob,4,11) -cne 'ssh-ed25519') { throw 'Invalid BOXWEAVE_SERVER_HOST_KEY' }
    # The container key is required for enrollment but not for diagnostics;
    # validate it only when present.
    if ($containerKey -and $containerKey -cnotmatch '^(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp256) [A-Za-z0-9+/=]+(?: [^\r\n]+)?$') { throw 'Invalid BOXWEAVE_CONTAINER_PUBLIC_KEY' }
    if (-not $agent -or $agent -notmatch '^(codex|cursor|claude-code|zcode)$') { throw 'Invalid or missing BOXWEAVE_AGENT' }
    # Accept any case: the generated script writes 'Wsl'/'Native' via
    # capitalize(), and hand edits may vary; normalize for downstream -ceq.
    if (-not $transport -or $transport -notmatch '^(Native|WSL)$') { throw 'Invalid or missing BOXWEAVE_TRANSPORT (Native or WSL)' }
    $transport = if ($transport -ieq 'WSL') { 'WSL' } else { 'Native' }
    if ($transport -ceq 'WSL' -and $distro -and $distro -cnotmatch '^[A-Za-z0-9_.-]+$') { throw 'Invalid BOXWEAVE_WSL_DISTRO' }
    return @{Host=$hostName;Port=[int]$portText;User=$user;HostKey=$hostKey;ContainerKey=$containerKey;
             Agent=$agent.ToLowerInvariant();Transport=$transport;Distro=$distro}
}

function Get-PolicyText($conn) {
    return $policyTemplate.Replace('{ssh_host}',$conn.Host).Replace('{ssh_port}',[string]$conn.Port).Replace('{linux_user}',$conn.User).Replace('{container_public_key}',$conn.ContainerKey)
}

function Invoke-BoxWeaveDiagnostic {
    $report=[ordered]@{schema_version=1;operation='client_diagnose';overall='incomplete';connection='unverified';registration='unverified';guidance='unverified';agent_ui='unverified';stages=@();process_exit_code=$null;cleanup='not_started'}
    $clock=[Diagnostics.Stopwatch]::StartNew()
    function Record-Diagnostic($Stage,$Status,$Code){
        $report.stages+=@{id=$Stage;status=$Status;code=$Code;elapsed_ms=$clock.ElapsedMilliseconds}
    }
    function Stop-Diagnostic($Code,$Stage,$Status='incomplete'){
        Record-Diagnostic $Stage $Status $Code
        throw 'Diagnostic stopped'
    }
    function Safe-DiagnosticPath([string]$Path){
        $candidate=[IO.Path]::GetFullPath($Path)
        while($candidate){
            if(Test-Path -LiteralPath $candidate){
                $item=Get-Item -Force -LiteralPath $candidate
                if($item.Attributes -band [IO.FileAttributes]::ReparsePoint){Stop-Diagnostic 'INPUT_INVALID' 'input'}
                if(-not $item.PSIsContainer -and $item.Length -gt 1048576){Stop-Diagnostic 'INPUT_INVALID' 'input'}
            }
            $candidate=Split-Path -Parent $candidate
        }
    }
    function Read-WslDiagnostic([string[]]$Arguments){
        $info=New-Object Diagnostics.ProcessStartInfo
        $info.FileName=$wslExe;$info.Arguments=(@($Arguments | ForEach-Object {Quote-ProcessArgument $_}) -join ' ')
        $info.UseShellExecute=$false;$info.CreateNoWindow=$true;$info.RedirectStandardOutput=$true;$info.RedirectStandardError=$true
        $proc=New-Object Diagnostics.Process;$proc.StartInfo=$info;$null=$proc.Start()
        $null=$proc.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null)
        $bytes=New-Object byte[] 8193;$length=0;$timer=[Diagnostics.Stopwatch]::StartNew()
        try{
            while($length -lt $bytes.Length){
                $left=[int](10000-$timer.ElapsedMilliseconds)
                if($left -le 0){throw 'WSL preflight timeout'}
                $read=$proc.StandardOutput.BaseStream.ReadAsync($bytes,$length,$bytes.Length-$length)
                if(-not $read.Wait($left)){throw 'WSL preflight timeout'}
                if($read.Result -eq 0){break}
                $length+=$read.Result
            }
            if($length -ge $bytes.Length -or -not $proc.WaitForExit(1000) -or $proc.ExitCode -ne 0){throw 'WSL preflight failed'}
            return [Text.Encoding]::UTF8.GetString($bytes,0,$length)
        } finally {
            if(-not $proc.HasExited){try{& taskkill.exe /PID $proc.Id /T /F 2>$null | Out-Null}catch{};if(-not $proc.HasExited){try{$proc.Kill()}catch{}};$null=$proc.WaitForExit(2000)}
            $proc.Dispose()
        }
    }
    try{
        if($Replace -or $BootstrapIdentityFile -or $ProjectPath){Stop-Diagnostic 'INPUT_INVALID' 'input'}
        try{
            if($Timeout -lt 1 -or $Timeout -gt 300){throw 'timeout'}
            $d=Get-ConnectionFromEnv
        } catch {Stop-Diagnostic 'INPUT_INVALID' 'input'}
        Record-Diagnostic 'input' 'passed' 'OK'
        $report.runtime='windows-native';$report.transport=$d.Transport
        $homePath=if($AgentHome){$AgentHome}elseif($d.Agent -ceq 'codex'){if($env:CODEX_HOME){$env:CODEX_HOME}else{Join-Path $env:USERPROFILE '.codex'}}elseif($d.Agent -ceq 'cursor'){Join-Path $env:USERPROFILE '.cursor'}elseif($d.Agent -ceq 'zcode'){Join-Path $env:USERPROFILE '.zcode'}else{if($env:CLAUDE_CONFIG_DIR){$env:CLAUDE_CONFIG_DIR}else{Join-Path $env:USERPROFILE '.claude'}}
        Safe-DiagnosticPath $homePath
        $token=if($d.Port -ne 22 -or $d.Host.Contains(':')){'['+$d.Host+']:'+$d.Port}else{$d.Host}
        $pin=$token+' '+(($d.HostKey -split ' ')[0..1] -join ' ')
        if($d.Transport -ceq 'WSL'){
            $distro=$d.Distro
            if(-not $distro){Stop-Diagnostic 'PROFILE_UNSUPPORTED' 'runtime'}
            $wslExe=(Get-Command wsl.exe -ErrorAction Stop).Source
            $body='set -eu; command -v ssh >/dev/null; test ! -L "$HOME"; test ! -L "$HOME/.ssh"; test ! -L "$HOME/.ssh/boxweave"; test ! -L "$HOME/.ssh/boxweave_known_hosts"; test -f "$HOME/.ssh/boxweave"; test -f "$HOME/.ssh/boxweave_known_hosts"; printf "%s\n" "$HOME"; cat "$HOME/.ssh/boxweave_known_hosts"'
            $encoded=[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($body))
            try{$observed=Read-WslDiagnostic @('-d',$distro,'-e','sh','-c',("printf %s "+$encoded+" | base64 -d | sh"))}
            catch{Stop-Diagnostic 'WSL_PREFLIGHT_FAILED' 'trust'}
            $parts=$observed -split ([string][char]10),2
            if($parts.Count -ne 2 -or $parts[0] -cnotmatch '^/[^\r\n]+$'){Stop-Diagnostic 'INPUT_INVALID' 'runtime'}
            $key=$parts[0]+'/.ssh/boxweave';$known=$parts[0]+'/.ssh/boxweave_known_hosts'
            if($parts[1].Trim() -cne $pin){Stop-Diagnostic 'PIN_MISMATCH' 'trust' 'failed'}
            $transportCommand=$wslExe;$prefix=@('-d',$distro,'-e','ssh');$nullConfig='/dev/null'
        }else{
            $transportCommand=(Get-Command ssh.exe -ErrorAction Stop).Source
            $key=Join-Path $env:USERPROFILE '.ssh\boxweave';$known=Join-Path $env:USERPROFILE '.ssh\boxweave_known_hosts'
            Safe-DiagnosticPath $key;Safe-DiagnosticPath $known
            if(-not (Test-Path -LiteralPath $key -PathType Leaf)){Stop-Diagnostic 'IDENTITY_MISSING' 'trust'}
            if(-not (Test-Path -LiteralPath $known -PathType Leaf)){Stop-Diagnostic 'PIN_MISSING' 'trust'}
            if(([IO.File]::ReadAllText($known)).Trim() -cne $pin){Stop-Diagnostic 'PIN_MISMATCH' 'trust' 'failed'}
            $prefix=@();$nullConfig='NUL'
        }
        Record-Diagnostic 'runtime' 'passed' 'OK';Record-Diagnostic 'trust' 'passed' 'OK'
        $sshArguments=@('-T','-i',$key,'-o','IdentitiesOnly=yes','-o','BatchMode=yes','-o','StrictHostKeyChecking=yes','-o',('UserKnownHostsFile='+$known),'-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15','-o','ServerAliveInterval=15','-o','ServerAliveCountMax=4','-p',[string]$d.Port,($d.User+'@'+$d.Host),$Script:McpRemoteCommand)
        $expectedArguments=@($prefix)+'boxweave'+$Script:McpRemoteCommand
        if($d.Agent -ceq 'claude-code'){
            $report.registration='unsupported';Record-Diagnostic 'registration' 'incomplete' 'REGISTRATION_UNSUPPORTED'
        }else{
            $config=Join-Path $homePath $(if($d.Agent -ceq 'codex'){'config.toml'}elseif($d.Agent -ceq 'zcode'){'cli/config.json'}else{'mcp.json'})
            Safe-DiagnosticPath $config
            if(-not (Test-Path -LiteralPath $config -PathType Leaf)){
                $report.registration='missing';Record-Diagnostic 'registration' 'incomplete' 'REGISTRATION_MISSING'
            }else{
                try{
                    $text=[IO.File]::ReadAllText($config)
                    if($d.Agent -ceq 'codex'){
                        $match=[regex]::Matches($text,'(?ms)^\[mcp_servers\.boxweave\][ \t]*(?:#[^\r\n]*)?\r?\n(.*?)(?=^\[|\z)')
                        if($match.Count -ne 1){throw 'unsupported'}
                        $fields=@{}
                        foreach($line in ($match[0].Groups[1].Value -split '\r?\n')){
                            if($line -match '^\s*(#|$)'){continue}
                            if($line -cnotmatch '^\s*(command|args|startup_timeout_sec|tool_timeout_sec)\s*=\s*(.*?)\s*$'){throw 'unsupported'}
                            $name=$Matches[1];$value=$Matches[2]
                            if($fields.ContainsKey($name)){throw 'duplicate'}
                            $fields[$name]=ConvertFrom-Json -InputObject $value
                        }
                        $entry=[pscustomobject]$fields
                    }else{
                        $parsed=ConvertFrom-Json -InputObject $text
                        if($d.Agent -ceq 'zcode'){
                            $entry=$parsed.mcp.servers.boxweave
                            if($parsed.PSObject.Properties['features'] -and $parsed.features.PSObject.Properties['mcp'] -and $parsed.features.mcp -ne $true){throw 'disabled'}
                            if($entry.type -and $entry.type -cne 'stdio'){throw 'unsupported'}
                            if($entry.PSObject.Properties['enable'] -and $entry.enable -ne $true){throw 'disabled'}
                        }else{$entry=$parsed.mcpServers.boxweave}
                        if(-not ($entry -is [pscustomobject]) -or @($entry.PSObject.Properties.Name | Where-Object {$_ -notin @('command','args','enabled','enable','type','timeoutMs')}).Count){throw 'unsupported'}
                    }
                    $matched=($entry.command -ceq $transportCommand -and ($entry.args -is [array]) -and
                        (ConvertTo-Json -InputObject @($entry.args) -Compress) -ceq (ConvertTo-Json -InputObject @($expectedArguments) -Compress))
                    if($entry.PSObject.Properties['enabled'] -and $entry.enabled -ne $true){$matched=$false}
                    $report.registration=if($matched){'matched'}else{'mismatch'}
                    Record-Diagnostic 'registration' $(if($matched){'passed'}else{'failed'}) $(if($matched){'OK'}else{'REGISTRATION_MISMATCH'})
                }catch{$report.registration='unsupported';Record-Diagnostic 'registration' 'incomplete' 'REGISTRATION_UNSUPPORTED'}
            }
        }
        if($d.Agent -ceq 'codex'){
            $guidance=Join-Path $homePath 'AGENTS.override.md';Safe-DiagnosticPath $guidance
            if(-not (Test-Path -LiteralPath $guidance) -or -not ([IO.File]::ReadAllText($guidance)).Trim()){$guidance=Join-Path $homePath 'AGENTS.md'}
            Safe-DiagnosticPath $guidance
            if((Test-Path -LiteralPath $guidance -PathType Leaf) -and ([IO.File]::ReadAllText($guidance)).Contains('<!-- BEGIN BOXWEAVE MANAGED -->')){$report.guidance='present'}
        }elseif($d.Agent -ceq 'zcode'){
            $guidance=Join-Path $homePath 'AGENTS.md';Safe-DiagnosticPath $guidance
            if((Test-Path -LiteralPath $guidance -PathType Leaf) -and ([IO.File]::ReadAllText($guidance)).Contains('<!-- BEGIN BOXWEAVE MANAGED -->')){$report.guidance='present'}
        }elseif($d.Agent -ceq 'cursor'){$report.guidance='manual'}
        Record-Diagnostic 'guidance' $(if($report.guidance -eq 'present'){'passed'}else{'unverified'}) $(if($report.guidance -eq 'present'){'FILE_PRESENT'}else{'MANUAL_VERIFICATION_REQUIRED'})
        $evidence=@{}
        try{
            Test-BoxWeaveMcp -Command $transportCommand -Arguments (@($prefix)+@('-F',$nullConfig,'-o','IdentityAgent=none','-o',('GlobalKnownHostsFile='+$nullConfig),'-o','UpdateHostKeys=no')+$sshArguments) -Timeout $Timeout -Evidence $evidence
            $report.connection='passed'
        }catch{
            $code='TRANSPORT_FAILED';if($_.Exception.Message -match '^([A-Z_]+):'){$code=$Matches[1]}
            $report.connection='failed'
            Record-Diagnostic $(if($evidence.stage){$evidence.stage}else{'transport'}) 'failed' $code
        }finally{
            foreach($stage in @('initialize','discovery','provider_call')){if($evidence[$stage] -eq 'passed'){Record-Diagnostic $stage 'passed' 'OK'}}
            foreach($field in @('protocol_version','process_exit_code','cleanup','tool_pages')){if($evidence.ContainsKey($field)){$report[$field]=$evidence[$field]}}
        }
    }catch{
        if(-not ($report.stages | Where-Object {$_.status -in @('failed','incomplete')})){Record-Diagnostic 'input' 'incomplete' 'INPUT_UNAVAILABLE'}
    }
    if($report.cleanup -eq 'incomplete'){Record-Diagnostic 'cleanup' 'failed' 'CLEANUP_INCOMPLETE'}
    $statuses=@($report.stages | ForEach-Object {$_.status})
    $report.overall=if('failed' -in $statuses){'failed'}elseif('incomplete' -in $statuses){'incomplete'}else{'passed'}
    $code=if($report.overall -eq 'passed'){0}elseif($report.overall -eq 'failed'){1}else{2}
    if($ReportPath){
        $temporary=$null
        try{
            Safe-DiagnosticPath $ReportPath
            if(Test-Path -LiteralPath $ReportPath){throw 'Report exists'}
            $destination=[IO.Path]::GetFullPath($ReportPath)
            $temporary=Join-Path (Split-Path -Parent $destination) ('.boxweave-report-'+[guid]::NewGuid())
            $stream=[IO.File]::Open($temporary,[IO.FileMode]::CreateNew);$stream.Dispose()
            $acl=New-Object Security.AccessControl.FileSecurity;$acl.SetAccessRuleProtection($true,$false)
            $rule=New-Object Security.AccessControl.FileSystemAccessRule([Security.Principal.WindowsIdentity]::GetCurrent().User,'FullControl','Allow')
            $acl.SetAccessRule($rule)
            if($PSVersionTable.PSEdition -eq 'Desktop'){[IO.File]::SetAccessControl($temporary,$acl)}
            else{[IO.FileSystemAclExtensions]::SetAccessControl((New-Object IO.FileInfo($temporary)),$acl)}
            [IO.File]::WriteAllText($temporary,($report | ConvertTo-Json -Depth 10),(New-Object Text.UTF8Encoding($false)))
            [IO.File]::Move($temporary,$destination)
        }catch{Record-Diagnostic 'report' 'failed' 'REPORT_WRITE_FAILED';$report.overall='failed';$code=1}
        finally{if($temporary -and (Test-Path -LiteralPath $temporary)){Remove-Item -LiteralPath $temporary -Force}}
    }
    $report | ConvertTo-Json -Depth 10
    return $code
}
if ($Diagnose) {
    $result=@(Invoke-BoxWeaveDiagnostic)
    $diagnosticExit=[int]$result[-1]
    if($result.Count -gt 1){$result[0..($result.Count-2)] | Write-Output}
    exit $diagnosticExit
}
if ($ReportPath) { throw '-ReportPath requires -Diagnose' }

$conn = Get-ConnectionFromEnv
if (-not $conn.ContainerKey) { throw 'Missing BOXWEAVE_CONTAINER_PUBLIC_KEY; rerun the generated script with its EDIT ME block intact' }
$SshHost=$conn.Host; $SshPort=$conn.Port; $SshUser=$conn.User
$Mode=$conn.Transport; $Agent=$conn.Agent
$WslDistro=$conn.Distro
$ServerHostKey=($conn.HostKey -split ' ')[1]
$glmProvider=$env:BOXWEAVE_CLAUDE_PROVIDER
$glmModel=$env:BOXWEAVE_GLM_MODEL
if (($glmProvider -or $glmModel) -and $Agent -cne 'claude-code') { throw 'GLM options require Claude Code' }
if ($glmModel -and -not $glmProvider) { throw 'GLM model requires provider' }

if ($Mode -eq 'WSL' -and -not $WslDistro) {
    $distros=@(wsl.exe -l -q | ForEach-Object { ($_ -replace [char]0,'').Trim() } | Where-Object { $_ })
    Assert-Exit 'WSL distribution discovery'
    if ($distros.Count -eq 1) { $WslDistro=$distros[0] }
    elseif ($distros.Count -gt 1) {
        for ($i=0;$i -lt $distros.Count;$i++) { Write-Host ("{0}) {1}" -f ($i+1),$distros[$i]) }
        $answer=Read-Host 'Select WSL distribution number'
        if ($answer -notmatch '^[0-9]+$' -or [int]$answer -lt 1 -or [int]$answer -gt $distros.Count) { throw 'Invalid selection; rerun with BOXWEAVE_WSL_DISTRO set' }
        $WslDistro=$distros[[int]$answer-1]
    } else { throw 'No installed WSL distribution' }
}
$claude = if ($Agent -eq 'claude-code') { (Get-Command claude -ErrorAction Stop).Source } else { $null }
if ($claude -and $AgentHome) { throw 'Set CLAUDE_CONFIG_DIR instead of -AgentHome for Claude Code' }
if ($McpName -notmatch '^[a-zA-Z0-9_-]+$' -or $KeyName -notmatch '^[a-zA-Z0-9_-]+$') {
    throw 'Invalid MCP name or SSH key filename.'
}
if ($Mode -eq 'WSL' -and $WslDistro -cnotmatch '^[a-zA-Z0-9_.-]+$') {
    throw 'WSL mode requires a WSL distribution name (wsl.exe -l -q lists installed distributions).'
}
$ssh = (Get-Command ssh.exe -ErrorAction Stop).Source
$sshKeygen = (Get-Command ssh-keygen.exe -ErrorAction Stop).Source
$wsl = if ($Mode -eq 'WSL') { (Get-Command wsl.exe -ErrorAction Stop).Source } else { $null }

$sshDir = Join-Path $env:USERPROFILE '.ssh'
$codexDir = if ($AgentHome) { $AgentHome } elseif ($Agent -eq 'cursor') { Join-Path $env:USERPROFILE '.cursor' } elseif ($Agent -eq 'zcode') { Join-Path $env:USERPROFILE '.zcode' } elseif ($Agent -eq 'claude-code') { if ($env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR } else { Join-Path $env:USERPROFILE '.claude' } } elseif ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $env:USERPROFILE '.codex' }
foreach ($candidate in @($sshDir,$codexDir)) {
    $ancestor=$candidate
    while ($ancestor) {
        if ((Test-Path -LiteralPath $ancestor) -and ((Get-Item -LiteralPath $ancestor).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing reparse point in setup path' }
        $ancestor=Split-Path -Parent $ancestor
    }
}
if ($BootstrapIdentityFile -and -not (Test-Path -LiteralPath $BootstrapIdentityFile -PathType Leaf)) { throw 'Bootstrap identity file not found' }
New-Item -ItemType Directory -Force $sshDir, $codexDir | Out-Null
# GLM credentials are read only on this client, never from a generated profile.
$glmText=$null
if ($glmProvider) {
    if ($glmProvider -cnotin @('zai','zhipu')) { throw 'Invalid Claude provider' }
    if ($glmModel -and ($glmModel -isnot [string] -or $glmModel -cnotmatch '^[A-Za-z0-9][A-Za-z0-9_.:\[\]-]{0,127}$')) { throw 'Invalid GLM model' }
    if (-not $env:GLM_API_KEY -or -not $env:GLM_API_KEY.Trim() -or $env:GLM_API_KEY -match '[\r\n]') { throw 'Set GLM_API_KEY in the client environment before setup' }
    $glmPath=Join-Path $codexDir 'settings.json'
    if (Test-Path -LiteralPath $glmPath) {
        if ((Get-Item -LiteralPath $glmPath).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing settings reparse point' }
        $glmObj=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($glmPath))
    } else { $glmObj=[pscustomobject]@{} }
    if ($glmObj -isnot [pscustomobject]) { throw 'Invalid Claude settings object' }
    if (-not $glmObj.PSObject.Properties['env']) { $glmObj | Add-Member NoteProperty env ([pscustomobject]@{}) }
    if ($glmObj.env -isnot [pscustomobject]) { throw 'Invalid Claude settings env' }
    $glmUrl=if ($glmProvider -ceq 'zai') { 'https://api.z.ai/api/anthropic' } else { 'https://open.bigmodel.cn/api/anthropic' }
    $glmValues=@{ANTHROPIC_BASE_URL=$glmUrl; ANTHROPIC_AUTH_TOKEN=$env:GLM_API_KEY; API_TIMEOUT_MS='3000000'}
    if ($glmModel) {
        foreach ($field in @('ANTHROPIC_MODEL','ANTHROPIC_DEFAULT_OPUS_MODEL','ANTHROPIC_DEFAULT_SONNET_MODEL','ANTHROPIC_DEFAULT_HAIKU_MODEL')) { $glmValues[$field]=$glmModel }
    }
    $glmObj.env.PSObject.Properties.Remove('ANTHROPIC_API_KEY')
    foreach ($field in $glmValues.Keys) { $glmObj.env | Add-Member -Force NoteProperty $field $glmValues[$field] }
    $glmText=ConvertTo-Json -InputObject $glmObj -Depth 100
}
$nativePrivate = Join-Path $sshDir $KeyName
$nativePublic = "$nativePrivate.pub"
$nativeKnown = Join-Path $sshDir "${KeyName}_known_hosts"
$nativeConfig = Join-Path $sshDir 'config'
$configPath = Join-Path $codexDir 'config.toml'
$agentsPath = Join-Path $codexDir 'AGENTS.override.md'
if (-not (Test-Path -LiteralPath $agentsPath) -or -not ([IO.File]::ReadAllText($agentsPath)).Trim()) { $agentsPath = Join-Path $codexDir 'AGENTS.md' }

if ($Agent -eq 'zcode') {
    $configPath=Join-Path $codexDir 'cli/config.json'
    $agentsPath=Join-Path $codexDir 'AGENTS.md'
    $cliDir=Join-Path $codexDir 'cli'
    if ((Test-Path -LiteralPath $cliDir) -and ((Get-Item -LiteralPath $cliDir).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing ZCode CLI reparse point' }
}
foreach ($candidate in @($nativePrivate,$nativePublic,$nativeKnown,$nativeConfig,$configPath,$agentsPath)) {
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
# Pin SSH server identity using the key from the EDIT ME header,
# never using untrusted ssh-keyscan or StrictHostKeyChecking=no.
# The Windows-side pin is unconditional: the enrollment ssh below runs from
# Windows in both modes; WSL additionally gets its own pin for its transport.
$hostToken = if ($SshPort -eq 22 -and -not $SshHost.Contains(':')) { $SshHost } else { '[{0}]:{1}' -f $SshHost, $SshPort }
$pinnedHostLine = "$hostToken ssh-ed25519 $ServerHostKey"
Set-PinnedHostNative -Path $nativeKnown -Line $pinnedHostLine

$clientPubLine = $nativePubLine
$transportCommand = $ssh
$transportArgs = @('boxweave', $Script:McpRemoteCommand)
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

    # Managed "Host boxweave" block inside the WSL distro's ~/.ssh/config.
    # The block text is built here (values already validated) and passed to the
    # WSL side base64-encoded; awk drops any previous managed region first.
    $wslBlock = ('# BEGIN BOXWEAVE MANAGED' + "`n" +
        'Host ' + $McpName + "`n" +
        '    HostName ' + $SshHost + "`n" +
        '    Port ' + $SshPort + "`n" +
        '    User ' + $SshUser + "`n" +
        '    IdentityFile ' + $wslHome + '/.ssh/' + $KeyName + "`n" +
        '    IdentitiesOnly yes' + "`n" +
        '    IdentityAgent none' + "`n" +
        '    BatchMode yes' + "`n" +
        '    UserKnownHostsFile ' + $wslHome + '/.ssh/' + $KeyName + '_known_hosts' + "`n" +
        '    StrictHostKeyChecking yes' + "`n" +
        '    HostKeyAlgorithms ssh-ed25519' + "`n" +
        '    ConnectTimeout 15' + "`n" +
        '    ServerAliveInterval 15' + "`n" +
        '    ServerAliveCountMax 4' + "`n" +
        '# END BOXWEAVE MANAGED')
    $wslConfigScript = @'
set -eu
umask 077
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
config="$HOME/.ssh/config"
test ! -L "$config"
touch "$config"
chmod 600 "$config"
block=$(printf '%s' "$1" | base64 -d)
tmp="$config.boxweave-tmp"
trap 'rm -f "$tmp"' EXIT
awk 'BEGIN{inb=0} $0=="# BEGIN BOXWEAVE MANAGED"{inb=1;next} $0=="# END BOXWEAVE MANAGED"{inb=0;next} !inb{print}' "$config" > "$tmp"
printf '%s\n' "$block" >> "$tmp"
chmod 600 "$tmp"
mv -f "$tmp" "$config"
'@
    $blockB64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($wslBlock))
    Invoke-WslScript -Script $wslConfigScript -ScriptArguments @($blockB64) -What 'WSL ssh_config managed block'
} else {
    Update-SshConfigManaged -Path $nativeConfig -Block (New-SshConfigBlock -HostName $SshHost -Port $SshPort -User $SshUser -IdentityFile $nativePrivate -KnownHostsFile $nativeKnown)
}

$pendingConfig = ''
if ($Agent -eq 'codex') {
    $pendingConfig = Update-CodexToml -Path $configPath -Command $transportCommand -Arguments $transportArgs
    $null=Update-Agents -Path $agentsPath -Content (Get-PolicyText $conn) -Preview
} elseif ($Agent -eq 'zcode') {
    $original=if (Test-Path -LiteralPath $configPath) { [IO.File]::ReadAllText($configPath) } else { '' }
    $pendingConfig=Update-ZCodeJson -Original $original -Command $transportCommand -Arguments $transportArgs -AllowReplace ([bool]$Replace)
    $null=Update-Agents -Path $agentsPath -Content (Get-PolicyText $conn) -Preview
} elseif ($Agent -eq 'cursor') {
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
$restricted = 'restrict,command="' + $Script:McpRemoteCommand + '" ' + $clientPubLine
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

# Explicit probe options mirror the managed Host block; the probe must not
# depend on the user's SSH config being parsed exactly as we wrote it.
$probeExtra = @('-F','NUL','-o','IdentityAgent=none','-o','GlobalKnownHostsFile=NUL','-o','UpdateHostKeys=no')
if ($Mode -eq 'WSL') {
    $probeArgs = @('-d',$WslDistro,'-e','ssh') + $probeExtra + @('-T','-i',"$wslHome/.ssh/$KeyName",
                   '-o','IdentitiesOnly=yes','-o','BatchMode=yes',
                   '-o','StrictHostKeyChecking=yes','-o',"UserKnownHostsFile=$wslHome/.ssh/${KeyName}_known_hosts",
                   '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15',
                   '-o','ServerAliveInterval=15','-o','ServerAliveCountMax=4',
                   '-p',"$SshPort","$SshUser@$SshHost",$Script:McpRemoteCommand)
    $probeCommand = $wsl
} else {
    $probeArgs = $probeExtra + @('-T','-i',$nativePrivate,
                   '-o','IdentitiesOnly=yes','-o','BatchMode=yes',
                   '-o','StrictHostKeyChecking=yes','-o',"UserKnownHostsFile=$nativeKnown",
                   '-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15',
                   '-o','ServerAliveInterval=15','-o','ServerAliveCountMax=4',
                   '-p',"$SshPort","$SshUser@$SshHost",$Script:McpRemoteCommand)
    $probeCommand = $ssh
}
Test-BoxWeaveMcp -Command $probeCommand -Arguments $probeArgs
$agents = Get-PolicyText $conn
if ($Agent -eq 'claude-code') {
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
    if ($Agent -eq 'zcode') { $null=New-Item -ItemType Directory -Force -Path (Split-Path -Parent $configPath) }
    Backup-File $configPath
    Write-Utf8NoBom $configPath $pendingConfig
    if ($Agent -eq 'cursor') {
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
if ($null -ne $glmText) {
    Backup-File $glmPath
    Write-Utf8NoBom $glmPath $glmText
}
Write-Host 'MCP initialize/tools/list/read-only provider call: passed.'
Write-Host "Agent registration: written (ssh $McpName). Reopen the agent and verify BoxWeave in its MCP UI."
Write-Host "Change the server address later by editing the '$McpName' block in the SSH config; registrations keep working."
if ($Agent -eq 'cursor' -and -not $ProjectPath) {
    Write-Host "Guidance: manual step required. Paste $agentsPath into Cursor Settings > Rules > User Rules."
} else { Write-Host 'Guidance: written; verify loading in a new session.' }
