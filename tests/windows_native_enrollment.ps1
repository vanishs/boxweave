#requires -version 5.1
param([Parameter(Mandatory=$true)][string]$Descriptor,[Parameter(Mandatory=$true)][string]$AgentHome,[Parameter(Mandatory=$true)][string]$CodexExecutable,[Parameter(Mandatory=$true)][string]$TestRoot)
$ErrorActionPreference='Stop';[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$root=Split-Path -Parent $PSScriptRoot;$payload=Join-Path $root 'clients/windows'
if(Test-Path -LiteralPath $TestRoot){throw 'Use fresh TestRoot'}
[void][IO.Directory]::CreateDirectory($TestRoot)
function Assert($Condition,[string]$Message){if(-not $Condition){throw $Message}}
foreach($file in @('prepare-identity.ps1','diagnose-native.ps1')){
  $tokens=$null;$errors=$null
  [void][Management.Automation.Language.Parser]::ParseFile((Join-Path $payload $file),[ref]$tokens,[ref]$errors)
  Assert ($errors.Count -eq 0) ('Parse: '+$file)
}
$public=[IO.File]::ReadAllText($Descriptor)|ConvertFrom-Json
$config=Join-Path $AgentHome 'config.toml'
$configHash=(Get-FileHash -LiteralPath $config).Hash
$trustHash=(Get-FileHash -LiteralPath $public.connection.known_hosts).Hash
$personal=Join-Path $env:USERPROFILE '.codex/config.toml'
$personalHash=if(Test-Path -LiteralPath $personal){(Get-FileHash -LiteralPath $personal).Hash}else{$null}
function Diagnose([string]$ProfileDirectory=$AgentHome,[string]$Manifest=$Descriptor,[string]$Package=$payload){
  $raw=& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $payload 'diagnose-native.ps1') -Descriptor $Manifest -PayloadDirectory $Package -AgentHome $ProfileDirectory -CodexExecutable $CodexExecutable
  $exit=$LASTEXITCODE
  return @{exit=$exit;receipt=(($raw -join [Environment]::NewLine)|ConvertFrom-Json)}
}
$first=Diagnose
Assert ($first.exit -eq 0 -and $first.receipt.connection -eq 'passed' -and $first.receipt.registration -eq 'matching') ('Matching diagnosis: '+($first.receipt|ConvertTo-Json -Compress))
Assert ($first.receipt.overall -eq 'incomplete' -and $first.receipt.agent_ui -eq 'unverified') 'Diagnosis fabricated GUI readiness'
foreach($kind in @('missing','disabled','mismatch')){
  $profile=Join-Path $TestRoot $kind;[void][IO.Directory]::CreateDirectory($profile)
  if($kind -ne 'missing'){
    $text=[IO.File]::ReadAllText($config)
    if($kind -eq 'disabled'){$text+=[Environment]::NewLine+'enabled = false'+[Environment]::NewLine}else{
      $oldPort='"'+$public.connection.port+'"'
      $otherPort=if($public.connection.port -eq 65535){65534}else{$public.connection.port+1}
      $text=$text.Replace($oldPort,('"'+$otherPort+'"'))
    }
    [IO.File]::WriteAllText((Join-Path $profile 'config.toml'),$text,[Text.UTF8Encoding]::new($false))
  }
  $before=if($kind -ne 'missing'){(Get-FileHash -LiteralPath (Join-Path $profile 'config.toml')).Hash}else{$null}
  $result=Diagnose -ProfileDirectory $profile
  Assert ($result.exit -eq 2 -and $result.receipt.connection -eq 'passed' -and $result.receipt.registration -eq $kind) ('Registration classification: '+$kind+' '+$result.receipt.code)
  if($before){Assert ((Get-FileHash -LiteralPath (Join-Path $profile 'config.toml')).Hash -eq $before) 'Diagnosis repaired config'}
}
$malformed=Join-Path $TestRoot 'malformed';[void][IO.Directory]::CreateDirectory($malformed)
[IO.File]::WriteAllText((Join-Path $malformed 'config.toml'),'[invalid toml',[Text.UTF8Encoding]::new($false))
$invalid=Diagnose -ProfileDirectory $malformed
Assert ($invalid.exit -eq 2 -and $invalid.receipt.registration -eq 'unverified' -and $invalid.receipt.code -eq 'CONFIG_UNSUPPORTED') 'Malformed config mislabeled as missing registration'
$tampered=Join-Path $TestRoot 'tampered';[void][IO.Directory]::CreateDirectory($tampered)
Copy-Item -LiteralPath (Join-Path $payload 'diagnose-connection.ps1') -Destination $tampered
Add-Content -LiteralPath (Join-Path $tampered 'diagnose-connection.ps1') -Value "throw 'must not execute'"
$bad=Diagnose -Package $tampered
Assert ($bad.exit -eq 2 -and $bad.receipt.code -eq 'PAYLOAD_TAMPERED') 'Tampered probe executed'
$wrong=Get-Content -LiteralPath $Descriptor -Raw|ConvertFrom-Json
$parts=$wrong.connection.server_key -split ' '
$wrong.connection.server_key=$parts[0]+' '+$parts[1].ToUpperInvariant()
$wrongPath=Join-Path $TestRoot 'wrong-pin.json'
[IO.File]::WriteAllText($wrongPath,($wrong|ConvertTo-Json -Depth 12),[Text.UTF8Encoding]::new($false))
$pin=Diagnose -Manifest $wrongPath
Assert ($pin.exit -eq 2 -and $pin.receipt.connection -ne 'passed' -and $pin.receipt.code -eq 'PIN_MISMATCH') 'Wrong pin accepted'
$keyDir=Join-Path $TestRoot ('key '+[char]0x6d4b+[char]0x8bd5)
$raw=& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $payload 'prepare-identity.ps1') -KeyDirectory $keyDir
Assert ($LASTEXITCODE -eq 0) 'Fresh key prepare failed'
$prepared=($raw -join [Environment]::NewLine)|ConvertFrom-Json
Assert ($prepared.remote_enrollment -eq 'pending') 'Preparation claimed remote enrollment'
Assert ((Get-Acl -LiteralPath $keyDir).AreAccessRulesProtected) 'Key directory ACL inherits unrelated access'
$allowed=@([Security.Principal.WindowsIdentity]::GetCurrent().User.Value,'S-1-5-18')
$rules=(Get-Acl -LiteralPath $prepared.identity_file).GetAccessRules($true,$true,[Security.Principal.SecurityIdentifier])
foreach($rule in $rules){Assert ($allowed -contains $rule.IdentityReference.Value) 'Unexpected private-key ACL principal'}
$publicHash=(Get-FileHash -LiteralPath $prepared.public_key_file).Hash
$metadata=Get-Item -LiteralPath $prepared.identity_file
$again=& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $payload 'prepare-identity.ps1') -KeyDirectory $keyDir
Assert ($LASTEXITCODE -eq 2) 'Existing directory overwritten'
Assert ((Get-FileHash -LiteralPath $prepared.public_key_file).Hash -eq $publicHash) 'Existing public key changed'
Assert ((Get-Item -LiteralPath $prepared.identity_file).Length -eq $metadata.Length) 'Existing key metadata changed'
Assert ((Get-FileHash -LiteralPath $config).Hash -eq $configHash) 'Enrolled home modified by diagnosis'
Assert ((Get-FileHash -LiteralPath $public.connection.known_hosts).Hash -eq $trustHash) 'Trust modified by diagnosis'
if($personalHash){Assert ((Get-FileHash -LiteralPath $personal).Hash -eq $personalHash) 'Personal config changed'}
@{result='passed';checks=@('AST','actual strict MCP + selected CLI diagnosis','missing/disabled/mismatch distinct from connection','GUI remains unverified','probe tamper blocked','pin mismatch blocked','fresh private ACL key preparation','existing key directory refused','enrolled/personal config and trust unchanged');inference='not_run';private_key_read='not_performed'}|ConvertTo-Json -Depth 5 -Compress
