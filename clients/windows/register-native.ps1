#requires -version 5.1
param([Parameter(Mandatory=$true)][string]$Descriptor,[Parameter(Mandatory=$true)][string]$AgentHome,[string]$CodexExecutable,[switch]$Replace)
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$ErrorActionPreference='Stop'
$probeCode='unverified';$code='INPUT_INVALID';$stage=$null;$mutex=$null;$locked=$false;$exitCode=2
$deadline=[DateTime]::UtcNow.AddSeconds(300)
function CheckPath([string]$Path){
  $p=[IO.Path]::GetFullPath($Path)
  while($p){if(Test-Path -LiteralPath $p){if((Get-Item -LiteralPath $p).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'reparse path'}};$parent=[IO.Path]::GetDirectoryName($p);if($parent -eq $p){break};$p=$parent}
}
function Quote([string]$Value){if($Value -match '["\r\n]' -or $Value.EndsWith('\')){throw 'unsupported argument'};return '"'+$Value+'"'}
function Digest([byte[]]$Bytes){$h=[Security.Cryptography.SHA256]::Create();try{return ([BitConverter]::ToString($h.ComputeHash($Bytes))).Replace('-','').ToLowerInvariant()}finally{$h.Dispose()}}
function Program([string]$Exe,[string[]]$Arguments,[string]$ProfileDirectory){
  $remaining=[int]($deadline-[DateTime]::UtcNow).TotalMilliseconds
  if($remaining -le 0){$script:code='DEADLINE_EXCEEDED';throw 'deadline'}
  $info=New-Object Diagnostics.ProcessStartInfo;$info.FileName=$Exe;$info.Arguments=($Arguments|ForEach-Object {Quote $_}) -join ' '
  $info.StandardOutputEncoding=New-Object Text.UTF8Encoding($false,$true)
  $info.UseShellExecute=$false;$info.CreateNoWindow=$true;$info.RedirectStandardOutput=$true;$info.RedirectStandardError=$true
  if($ProfileDirectory){$info.EnvironmentVariables['CODEX_HOME']=$ProfileDirectory;$info.WorkingDirectory=$ProfileDirectory}
  $p=New-Object Diagnostics.Process;$p.StartInfo=$info
  try{[void]$p.Start();$read=$p.StandardOutput.ReadToEndAsync();$drain=$p.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null)
    if(-not $p.WaitForExit([Math]::Min(120000,$remaining))){$script:code='DEADLINE_EXCEEDED';$p.Kill();throw 'process deadline'}
    $output=$read.Result;if($output.Length -gt 4194304){throw 'output limit'}
    return @{exit=$p.ExitCode;output=$output}
  }finally{$p.Dispose()}
}
function ValidateConfig([string]$ProfileDirectory){
  $r=Program $script:codex @('mcp','list','--json') $ProfileDirectory
  if($r.exit -ne 0){$script:code='CONFIG_UNSUPPORTED';throw 'config validation'}
  $parsed=ConvertFrom-Json -InputObject $r.output
  return @($parsed)
}
try {
  CheckPath $Descriptor;$file=Get-Item -LiteralPath $Descriptor;if($file.Length -gt 32768){throw 'descriptor limit'}
  $d=[IO.File]::ReadAllText($file.FullName)|ConvertFrom-Json
  if(($d.schema -isnot [int] -and $d.schema -isnot [long]) -or $d.schema -ne 1 -or $d.profile -ne 'codex/windows/native' -or $d.source_id -ne ('sha256:'+$d.payloads.'register-native.ps1')){throw 'profile'}
  if((Digest ([IO.File]::ReadAllBytes($PSCommandPath))) -ne $d.payloads.'register-native.ps1'){throw 'unverified payload'}
  $code='HOME_UNSUPPORTED'
  if($AgentHome -notmatch '^[A-Za-z]:[\\/]'){throw 'explicit local AgentHome required'}
  $homePath=[IO.Path]::GetFullPath($AgentHome).TrimEnd([char]'\',[char]'/')
  if($homePath.Length -le 3){throw 'root AgentHome refused'};CheckPath $homePath
  $code='RUNTIME_UNSUPPORTED'
  # Resolve in the caller's actual environment; app-integrated PATH is not a desktop PATH.
  # An explicit executable is operator-selected, never silently discovered in app internals.
  if($CodexExecutable){
    if($CodexExecutable -notmatch '^[A-Za-z]:[\\/]'){throw 'absolute local CodexExecutable required'}
    CheckPath $CodexExecutable
    $item=Get-Item -LiteralPath $CodexExecutable -ErrorAction Stop
    if($item.PSIsContainer -or $item.Extension -ine '.exe'){throw 'CodexExecutable must be an executable file'}
    $codex=$item.FullName
  }else{
    $codex=(Get-Command codex.exe -CommandType Application -ErrorAction Stop).Source
  }
  $v=Program $codex @('--version') ''
  if($v.exit -ne 0 -or $v.output.Trim() -ne $d.codex_version){throw 'unreviewed CLI version'}
  $connection=$d.connection
  foreach($item in @($connection.identity_file,$connection.known_hosts)){if($item -notmatch '^[A-Za-z]:[\\/]'){throw 'absolute local connection path required'};CheckPath $item}
  $code='TRUST_OR_CONNECTION_FAILED'
  # Reuse the independently verified observation-only payload before any config publication.
  $probeFile=Join-Path $PSScriptRoot 'connection.json'
  [IO.File]::WriteAllText($probeFile,($connection|ConvertTo-Json -Depth 8),[Text.UTF8Encoding]::new($false))
  $probe=Program (Get-Command powershell.exe).Source @('-NoProfile','-ExecutionPolicy','Bypass','-File',(Join-Path $PSScriptRoot 'diagnose-connection.ps1'),'-Descriptor',$probeFile,'-Timeout','90') ''
  $probeResult=$probe.output|ConvertFrom-Json;$probeCode=$probeResult.code
  if($probe.exit -ne 0 -or $probeResult.connection -ne 'passed'){throw 'probe failed'}
  $code='STATE_UNAVAILABLE'
  if(-not(Test-Path -LiteralPath $homePath)){[void][IO.Directory]::CreateDirectory($homePath)}
  if(-not(Get-Item -LiteralPath $homePath).PSIsContainer){throw 'home not directory'}
  $mutex=New-Object Threading.Mutex($false,('Local\BoxWeaveNative-'+(Digest ([Text.Encoding]::UTF8.GetBytes($homePath.ToLowerInvariant()))).Substring(0,32)))
  $locked=$mutex.WaitOne(0);if(-not $locked){$code='CONFIG_BUSY';throw 'another enrollment'}
  $config=Join-Path $homePath 'config.toml';CheckPath $config
  $original='';$originalBytes=[byte[]]@();$existed=Test-Path -LiteralPath $config
  if($existed){$f=Get-Item -LiteralPath $config;if($f.Length -gt 1048576 -or $f.PSIsContainer){throw 'config limit'};$originalBytes=[IO.File]::ReadAllBytes($config);$original=[Text.Encoding]::UTF8.GetString($originalBytes).TrimStart([char]0xfeff)}
  $stage=Join-Path $homePath ('.boxweave-stage-'+[Guid]::NewGuid().ToString('N'));[void][IO.Directory]::CreateDirectory($stage)
  $stageConfig=Join-Path $stage 'config.toml';[IO.File]::WriteAllText($stageConfig,$original,[Text.UTF8Encoding]::new($false))
  $servers=ValidateConfig $stage;$existing=@($servers|Where-Object {$_.name -eq 'boxweave'})
  $ssh=(Get-Command ssh.exe -ErrorAction Stop).Source
  $sshArgs=@('-F','NUL','-o','BatchMode=yes','-o','StrictHostKeyChecking=yes','-o','IdentitiesOnly=yes','-o','ConnectTimeout=15','-o','ServerAliveInterval=15','-o','ServerAliveCountMax=4','-o',('UserKnownHostsFile='+[IO.Path]::GetFullPath($connection.known_hosts)),'-i',[IO.Path]::GetFullPath($connection.identity_file),'-p',[string]$connection.port,($connection.user+'@'+$connection.host),'/usr/local/bin/boxweave-mcp')
  $commandLine='command = '+(ConvertTo-Json -InputObject $ssh -Compress)
  $argsLine='args = '+(ConvertTo-Json -InputObject $sshArgs -Compress)
  $lines=$original -split '\r?\n';$mainCount=0;$quoted=$false
  foreach($line in $lines){if($line -match '^\s*\[\s*mcp_servers\.boxweave\s*\]\s*(#.*)?$'){$mainCount++};if($line -match '^\s*\[.*boxweave' -and ($line.Contains('"') -or $line.Contains("'"))){$quoted=$true}}
  if($quoted -or $mainCount -gt 1 -or ($existing.Count -gt 0 -and $mainCount -ne 1)){$code='MANUAL_MIGRATION_REQUIRED';throw 'unsupported table form'}
  $out=New-Object 'Collections.Generic.List[string]';$inMain=$false;$skip=$false
  $transportKeys='command|args|url|env|env_vars|cwd|experimental_environment|bearer_token_env_var|http_headers|env_http_headers|http_headers_helper'
  foreach($line in $lines){
    if($line -match '^\s*\['){
      $inMain=$line -match '^\s*\[\s*mcp_servers\.boxweave\s*\]\s*(#.*)?$'
      $skip=$line -match '^\s*\[\s*mcp_servers\.boxweave\.(env|http_headers|env_http_headers|oauth)\s*\]'
      if($skip){continue};$out.Add($line)
      if($inMain){$out.Add($commandLine);$out.Add($argsLine)};continue
    }
    if($skip){continue}
    if($inMain -and $line -match ('^\s*('+ $transportKeys +')\s*=(.*)$')){
      $value=$Matches[2].Trim()
      if($value.StartsWith('"""') -or $value.StartsWith("'''")){$code='MANUAL_MIGRATION_REQUIRED';throw 'multiline transport'}
      continue
    }
    $out.Add($line)
  }
  if($mainCount -eq 0){$out.Add('');$out.Add('[mcp_servers.boxweave]');$out.Add($commandLine);$out.Add($argsLine)}
  $newline=if($original.Contains("`r`n")){"`r`n"}else{"`n"}
  $candidate=($out -join $newline).TrimEnd([char]13,[char]10)+$newline
  $unchanged=($original -eq $candidate)
  if($existing.Count -gt 0 -and -not $unchanged -and -not $Replace){$code='REGISTRATION_CONFLICT';throw 'replacement not selected'}
  [IO.File]::WriteAllText($stageConfig,$candidate,[Text.UTF8Encoding]::new($false));$newServers=ValidateConfig $stage
  $registered=@($newServers|Where-Object {$_.name -eq 'boxweave'})
  if($registered.Count -ne 1 -or $registered[0].transport.command -ne $ssh -or (ConvertTo-Json -InputObject @($registered[0].transport.args) -Compress) -ne (ConvertTo-Json -InputObject $sshArgs -Compress)){throw 'registration mismatch'}
  if($existed -ne (Test-Path -LiteralPath $config) -or ($existed -and (Digest ([IO.File]::ReadAllBytes($config))) -ne (Digest $originalBytes))){$code='CONFIG_CHANGED';throw 'concurrent external edit'}
  $backup=$null
  if(-not $unchanged){
    if($existed){$backup=$config+'.boxweave-'+[Guid]::NewGuid().ToString('N')+'.bak';[IO.File]::Replace($stageConfig,$config,$backup)}else{[IO.File]::Move($stageConfig,$config)}
  }
  @{code='OK';registration='verified_cli';enabled=$registered[0].enabled;connection='passed';agent_ui='unverified';overall='incomplete';changed=(-not $unchanged);source_id=$d.source_id;backup=$backup;remote_cleanup='unverified'}|ConvertTo-Json -Compress
  $exitCode=0
}catch{@{code=$code;registration='not_completed';probe_code=$probeCode;overall='incomplete';agent_ui='unverified'}|ConvertTo-Json -Compress}
finally{
  if($stage -and (Test-Path -LiteralPath $stage)){$resolved=[IO.Path]::GetFullPath($stage);$parent=[IO.Path]::GetFullPath($homePath).TrimEnd([char]'\')+'\';if($resolved.StartsWith($parent,[StringComparison]::OrdinalIgnoreCase) -and [IO.Path]::GetFileName($resolved) -match '^\.boxweave-stage-[a-f0-9]{32}$'){Remove-Item -LiteralPath $resolved -Recurse -Force}}
  if($locked){$mutex.ReleaseMutex()};if($mutex){$mutex.Dispose()}
}
exit $exitCode
