#requires -version 5.1
param([Parameter(Mandatory=$true)][string]$ConnectionDescriptor,[Parameter(Mandatory=$true)][string]$TestRoot,[string]$CodexExecutable)
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$payload=Join-Path $root 'clients/windows'
$bootstrap=Join-Path $payload 'bootstrap-native.ps1'
$taskRoot=[IO.Path]::GetFullPath($TestRoot)
if(Test-Path -LiteralPath $taskRoot){throw 'Use a fresh test root'}
[void][IO.Directory]::CreateDirectory($taskRoot)
function Assert($Condition,[string]$Message){if(-not $Condition){throw $Message}}
foreach($name in @('bootstrap-native.ps1','register-native.ps1','diagnose-connection.ps1')){
  $tokens=$null;$errors=$null;[void][Management.Automation.Language.Parser]::ParseFile((Join-Path $payload $name),[ref]$tokens,[ref]$errors)
  Assert ($errors.Count -eq 0) ('Parse failed: '+$name)
}
$connection=[IO.File]::ReadAllText((Get-Item -LiteralPath $ConnectionDescriptor).FullName)|ConvertFrom-Json
$descriptor=Join-Path $taskRoot 'trusted.json'
$hashes=@{};foreach($name in @('register-native.ps1','diagnose-connection.ps1')){$hashes[$name]=(Get-FileHash -LiteralPath (Join-Path $payload $name) -Algorithm SHA256).Hash.ToLowerInvariant()}
$codexPath=if($CodexExecutable){$CodexExecutable}else{(Get-Command codex.exe -CommandType Application -ErrorAction Stop).Source}
$version=(& $codexPath --version).Trim()
$d=@{schema=1;profile='codex/windows/native';source_id=('sha256:'+$hashes['register-native.ps1']);codex_version=$version;payloads=$hashes;connection=$connection}
[IO.File]::WriteAllText($descriptor,($d|ConvertTo-Json -Depth 12),[Text.UTF8Encoding]::new($false))
function Run([string]$ProfileDirectory,[switch]$Replace,[string]$Package=$payload,[string]$Manifest=$descriptor,[string]$Executable=$codexPath){
  $args=@('-NoProfile','-ExecutionPolicy','Bypass','-File',$bootstrap,'-Descriptor',$Manifest,'-PayloadDirectory',$Package,'-AgentHome',$ProfileDirectory)
  if($Executable){$args+=@('-CodexExecutable',$Executable)}
  if($Replace){$args+='-Replace'}
  $raw=& powershell.exe @args;$code=$LASTEXITCODE
  try{$result=($raw -join "`n")|ConvertFrom-Json}catch{throw ('No structured receipt, exit '+$code)}
  return @{exit=$code;receipt=$result}
}
# Existing personal config is observed only as a digest, never edited or printed.
$personal=Join-Path $env:USERPROFILE '.codex/config.toml'
$personalHash=if(Test-Path -LiteralPath $personal){(Get-FileHash -LiteralPath $personal).Hash}else{$null}
$nativeHome=Join-Path $taskRoot ('native '+[char]0x6d4b+[char]0x8bd5+' home')
# Model double-click's ordinary shell: Codex is absent from PATH, Windows tools remain.
$savedPath=$env:PATH
try {
  $env:PATH="$env:WINDIR\System32;$env:WINDIR\System32\WindowsPowerShell\v1.0;$env:WINDIR\System32\OpenSSH"
  Assert (-not(Get-Command codex.exe -CommandType Application -ErrorAction SilentlyContinue)) 'Test PATH still contains Codex'
  $noRuntime=Run (Join-Path $taskRoot 'missing-runtime') -Executable ''
  Assert ($noRuntime.exit -eq 2 -and $noRuntime.receipt.code -eq 'RUNTIME_UNSUPPORTED') 'Missing PATH runtime did not fail safely'
  Assert (-not(Test-Path -LiteralPath (Join-Path $taskRoot 'missing-runtime'))) 'Missing runtime created Agent home'
  $relative=Run (Join-Path $taskRoot 'relative-runtime') -Executable 'codex.exe'
  Assert ($relative.exit -eq 2 -and $relative.receipt.code -eq 'RUNTIME_UNSUPPORTED') 'Relative executable accepted'
  Assert (-not(Test-Path -LiteralPath (Join-Path $taskRoot 'relative-runtime'))) 'Invalid runtime created Agent home'
  $first=Run $nativeHome
} finally {$env:PATH=$savedPath}
Assert ($first.exit -eq 0 -and $first.receipt.registration -eq 'verified_cli' -and $first.receipt.connection -eq 'passed') ('First registration failed: '+$first.receipt.code)
$config=Join-Path $nativeHome 'config.toml';$firstHash=(Get-FileHash -LiteralPath $config).Hash
$again=Run $nativeHome;Assert ($again.exit -eq 0 -and -not $again.receipt.changed) ('Idempotence failed: '+$again.receipt.code)
Assert ((Get-FileHash -LiteralPath $config).Hash -eq $firstHash) 'Repeat changed config'
$conflictHome=Join-Path $taskRoot 'conflict';[void][IO.Directory]::CreateDirectory($conflictHome)
$old=@'
model = "synthetic-model" # unrelated
[mcp_servers.other]
command = "other-command"
[mcp_servers.boxweave]
command = "old-command"
args = ["old"]
startup_timeout_sec = 45
[mcp_servers.boxweave.tools.connect]
approval_mode = "prompt"
'@
$oldPath=Join-Path $conflictHome 'config.toml';[IO.File]::WriteAllText($oldPath,$old,[Text.UTF8Encoding]::new($false));$oldHash=(Get-FileHash -LiteralPath $oldPath).Hash
$refused=Run $conflictHome;Assert ($refused.exit -eq 2 -and $refused.receipt.code -eq 'REGISTRATION_CONFLICT') ('Conflict not refused: '+$refused.receipt.code)
Assert ((Get-FileHash -LiteralPath $oldPath).Hash -eq $oldHash) 'Declined replacement modified config'
$replaced=Run $conflictHome -Replace;Assert ($replaced.exit -eq 0 -and $replaced.receipt.changed) ('Replacement failed: '+$replaced.receipt.code)
$new=[IO.File]::ReadAllText($oldPath)
foreach($keep in @('model = "synthetic-model" # unrelated','command = "other-command"','startup_timeout_sec = 45','[mcp_servers.boxweave.tools.connect]','approval_mode = "prompt"')){Assert ($new.Contains($keep)) ('Lost unrelated config: '+$keep)}
Assert ((Get-FileHash -LiteralPath $replaced.receipt.backup).Hash -eq $oldHash) 'Backup differs from original'
$inlineHome=Join-Path $taskRoot 'inline';[void][IO.Directory]::CreateDirectory($inlineHome);$inline=Join-Path $inlineHome 'config.toml'
[IO.File]::WriteAllText($inline,'mcp_servers = { boxweave = { command = "old-command" } }',[Text.UTF8Encoding]::new($false));$inlineHash=(Get-FileHash -LiteralPath $inline).Hash
$unsupported=Run $inlineHome -Replace;Assert ($unsupported.exit -eq 2 -and $unsupported.receipt.code -eq 'MANUAL_MIGRATION_REQUIRED') ('Inline form not refused: '+$unsupported.receipt.code)
Assert ((Get-FileHash -LiteralPath $inline).Hash -eq $inlineHash) 'Unsupported form modified'
$badPackage=Join-Path $taskRoot 'tampered';[void][IO.Directory]::CreateDirectory($badPackage)
foreach($name in $hashes.Keys){Copy-Item -LiteralPath (Join-Path $payload $name) -Destination (Join-Path $badPackage $name)}
$marker=Join-Path $taskRoot 'untrusted-executed'
Add-Content -LiteralPath (Join-Path $badPackage 'register-native.ps1') -Value ('Set-Content -LiteralPath '+"'"+$marker+"' -Value 'bad'")
$tampered=Run (Join-Path $taskRoot 'must-not-exist') -Package $badPackage
Assert ($tampered.exit -eq 2 -and $tampered.receipt.code -eq 'PAYLOAD_TAMPERED') 'Tampered payload accepted'
Assert (-not(Test-Path -LiteralPath $marker)) 'Untrusted payload executed'
Assert (-not(Test-Path -LiteralPath (Join-Path $taskRoot 'must-not-exist'))) 'Tamper created Agent home'
# A case-only change in a base64 pin is a different key, not a case-insensitive match.
$wrongConnection=$connection|ConvertTo-Json|ConvertFrom-Json
$parts=$wrongConnection.server_key -split ' '
$wrongConnection.server_key=$parts[0]+' '+$parts[1].ToUpperInvariant()
$wrongPin=Join-Path $taskRoot 'wrong-pin.json';[IO.File]::WriteAllText($wrongPin,($wrongConnection|ConvertTo-Json),[Text.UTF8Encoding]::new($false))
$pinRaw=& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $payload 'diagnose-connection.ps1') -Descriptor $wrongPin
$pinExit=$LASTEXITCODE;$pin=$pinRaw|ConvertFrom-Json
Assert ($pinExit -eq 2 -and $pin.code -eq 'PIN_MISMATCH') 'Case-only pin change was accepted'
$badSchema=Join-Path $taskRoot 'bad-schema.json';$bad=$d.Clone();$bad.schema=$true
[IO.File]::WriteAllText($badSchema,($bad|ConvertTo-Json -Depth 12),[Text.UTF8Encoding]::new($false))
$schemaResult=Run (Join-Path $taskRoot 'bad-schema-home') -Manifest $badSchema
Assert ($schemaResult.exit -eq 2 -and $schemaResult.receipt.code -eq 'PACKAGE_INVALID') 'Boolean schema accepted'
# Exercise the result validator without starting SSH: malformed/empty cannot pass.
$ast=[Management.Automation.Language.Parser]::ParseFile((Join-Path $payload 'diagnose-connection.ps1'),[ref]$null,[ref]$null)
$validator=$ast.Find({param($a) $a -is [Management.Automation.Language.FunctionDefinitionAst] -and $a.Name -eq 'ValidateReadOnlyResult'},$true)
. ([ScriptBlock]::Create($validator.Extent.Text))
$rejected=0
foreach($value in @(@{content=@()},@{content=@(@{type='text';text=''})},@{isError=$true;content=@(@{type='text';text='denied'})})){
  try{ValidateReadOnlyResult $value}catch{$rejected++}
}
Assert ($rejected -eq 3) 'Malformed read-only results passed'
ValidateReadOnlyResult @{content=@(@{type='text';text='No containers found.'})}
if($personalHash){Assert ((Get-FileHash -LiteralPath $personal).Hash -eq $personalHash) 'Personal configuration changed'}
@{result='passed';codex=$version;profile='codex/windows/native';scope='isolated existing-identity registration';checks=@('PowerShell5.1 parse','trusted payload verification','actual native MCP read','Codex CLI recognition','Unicode and space AgentHome','idempotence','declined conflict preserved','replacement preserves approvals/settings','backup','inline form rejected','tamper blocks execution','malformed read-only results rejected','case-sensitive server pin','boolean schema rejected','personal config unchanged');agent_ui='unverified';test_root=$taskRoot}|ConvertTo-Json -Depth 8 -Compress
