#requires -version 5.1
param([Parameter(Mandatory=$true)][string]$Descriptor,[Parameter(Mandatory=$true)][string]$PayloadDirectory,[Parameter(Mandatory=$true)][string]$AgentHome,[string]$CodexExecutable)
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$ErrorActionPreference='Stop';$code='PACKAGE_INVALID';$cache=$null;$connection='unverified';$registration='unverified';$status=2
function Digest([byte[]]$Bytes){$h=[Security.Cryptography.SHA256]::Create();try{return ([BitConverter]::ToString($h.ComputeHash($Bytes))).Replace('-','').ToLowerInvariant()}finally{$h.Dispose()}}
function CheckPath([string]$Path){$p=[IO.Path]::GetFullPath($Path);while($p){if((Test-Path -LiteralPath $p) -and ((Get-Item -LiteralPath $p).Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'reparse path'};$p=[IO.Path]::GetDirectoryName($p)}}
function Quote([string]$Value){if($Value -match '["\r\n]' -or $Value.EndsWith('\')){throw 'unsupported argument'};return '"'+$Value+'"'}
function Program([string]$Exe,[string[]]$Arguments,[string]$ProfileDirectory){
  $info=New-Object Diagnostics.ProcessStartInfo;$info.FileName=$Exe;$info.Arguments=($Arguments|ForEach-Object {Quote $_}) -join ' '
  $info.UseShellExecute=$false;$info.CreateNoWindow=$true;$info.RedirectStandardOutput=$true;$info.RedirectStandardError=$true
  $info.StandardOutputEncoding=New-Object Text.UTF8Encoding($false,$true)
  if($ProfileDirectory){$info.EnvironmentVariables['CODEX_HOME']=$ProfileDirectory;$info.EnvironmentVariables.Remove('CODEX_SQLITE_HOME');$info.WorkingDirectory=$ProfileDirectory}
  $p=New-Object Diagnostics.Process;$p.StartInfo=$info
  try{[void]$p.Start();$out=$p.StandardOutput.ReadToEndAsync();$err=$p.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null);if(-not $p.WaitForExit(120000)){$p.Kill();throw 'deadline'};$text=$out.Result;if($text.Length -gt 4194304){throw 'output limit'};return @{exit=$p.ExitCode;output=$text}}finally{$p.Dispose()}
}
try {
  CheckPath $Descriptor;CheckPath $PayloadDirectory;CheckPath $AgentHome
  if($AgentHome -notmatch '^[A-Za-z]:[\\/]' -or -not(Test-Path -LiteralPath $AgentHome -PathType Container)){throw 'explicit existing home required'}
  $file=Get-Item -LiteralPath $Descriptor;if($file.Length -gt 32768){throw 'descriptor limit'}
  $d=[IO.File]::ReadAllText($file.FullName)|ConvertFrom-Json
  if(($d.schema -isnot [int] -and $d.schema -isnot [long]) -or $d.schema -ne 1 -or $d.profile -ne 'codex/windows/native' -or $d.source_id -ne ('sha256:'+$d.payloads.'register-native.ps1')){throw 'profile'}
  $path=Join-Path $PayloadDirectory 'diagnose-connection.ps1';CheckPath $path
  $file=Get-Item -LiteralPath $path;if($file.Length -gt 131072){throw 'probe limit'}
  $bytes=[IO.File]::ReadAllBytes($file.FullName)
  if((Digest $bytes) -ne $d.payloads.'diagnose-connection.ps1'){$code='PAYLOAD_TAMPERED';throw 'probe hash'}
  $code='RUNTIME_UNSUPPORTED'
  if($CodexExecutable){if($CodexExecutable -notmatch '^[A-Za-z]:[\\/]'){throw 'absolute executable required'};CheckPath $CodexExecutable;$item=Get-Item -LiteralPath $CodexExecutable;if($item.PSIsContainer -or $item.Extension -ine '.exe'){throw 'executable required'};$codex=$item.FullName}else{$codex=(Get-Command codex.exe -CommandType Application -ErrorAction Stop).Source}
  $version=Program $codex @('--version') ''
  if($version.exit -ne 0 -or $version.output.Trim() -ne $d.codex_version){throw 'unreviewed CLI version'}
  $cache=Join-Path ([IO.Path]::GetTempPath()) ('boxweave-diagnose-'+[Guid]::NewGuid().ToString('N'));[void][IO.Directory]::CreateDirectory($cache)
  [IO.File]::WriteAllBytes((Join-Path $cache 'diagnose-connection.ps1'),$bytes)
  $public=Join-Path $cache 'connection.json';[IO.File]::WriteAllText($public,($d.connection|ConvertTo-Json -Depth 8),[Text.UTF8Encoding]::new($false))
  $probe=Program (Get-Command powershell.exe).Source @('-NoProfile','-ExecutionPolicy','Bypass','-File',(Join-Path $cache 'diagnose-connection.ps1'),'-Descriptor',$public,'-Timeout','90') ''
  $result=$probe.output|ConvertFrom-Json;$connection=$result.connection;$connectionCode=$result.code;$code='CONFIG_UNSUPPORTED'
  $read=Program $codex @('mcp','list','--json') ([IO.Path]::GetFullPath($AgentHome))
  if($read.exit -ne 0 -or -not $read.output.TrimStart().StartsWith('[')){throw 'configuration inspection failed'}
  $entries=@(($read.output|ConvertFrom-Json)|Where-Object {$_.name -eq 'boxweave'})
  if($entries.Count -eq 0){$registration='missing'}else{
    if($entries.Count -ne 1){throw 'ambiguous registration'}
    $entry=$entries[0];$c=$d.connection
    $expected=@('-F','NUL','-o','BatchMode=yes','-o','StrictHostKeyChecking=yes','-o','IdentitiesOnly=yes','-o','ConnectTimeout=15','-o',('UserKnownHostsFile='+[IO.Path]::GetFullPath($c.known_hosts)),'-i',[IO.Path]::GetFullPath($c.identity_file),'-p',[string]$c.port,($c.user+'@'+$c.host),'/usr/local/bin/boxweave-mcp')
    $same=$entry.transport.command -ceq (Get-Command ssh.exe).Source
    $same=$same -and ((ConvertTo-Json -InputObject @($entry.transport.args) -Compress) -ceq (ConvertTo-Json -InputObject $expected -Compress))
    $same=$same -and (-not $entry.transport.env) -and (-not $entry.transport.cwd)
    $registration=if(-not $same){'mismatch'}elseif(-not $entry.enabled){'disabled'}else{'matching'}
  }
  if($probe.exit -eq 0 -and $connection -eq 'passed' -and $registration -eq 'matching'){$code='OK';$status=0}
  elseif($connection -eq 'passed'){$code='REGISTRATION_'+$registration.ToUpperInvariant()}
  else{$code=$connectionCode}
}catch{}
finally{
  if($cache -and (Test-Path -LiteralPath $cache)){
    $resolved=[IO.Path]::GetFullPath($cache);$parent=[IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([char]'\')+'\'
    if($resolved.StartsWith($parent,[StringComparison]::OrdinalIgnoreCase) -and [IO.Path]::GetFileName($resolved) -match '^boxweave-diagnose-[a-f0-9]{32}$'){Remove-Item -LiteralPath $resolved -Recurse -Force}
  }
}
@{code=$code;connection=$connection;registration=$registration;agent_ui='unverified';overall='incomplete';remote_cleanup='unverified';observation_only=$true}|ConvertTo-Json -Compress
exit $status
