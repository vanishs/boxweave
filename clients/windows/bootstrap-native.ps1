#requires -version 5.1
<# Trusted entry point: the descriptor must arrive through an operator-authorized channel. #>
param([Parameter(Mandatory=$true)][string]$Descriptor,[Parameter(Mandatory=$true)][string]$PayloadDirectory,[Parameter(Mandatory=$true)][string]$AgentHome,[string]$CodexExecutable,[switch]$Replace)
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$ErrorActionPreference='Stop';$cache=$null;$code='PACKAGE_INVALID';$exitCode=2
function Digest([byte[]]$Bytes){$h=[Security.Cryptography.SHA256]::Create();try{return ([BitConverter]::ToString($h.ComputeHash($Bytes))).Replace('-','').ToLowerInvariant()}finally{$h.Dispose()}}
function CheckPath([string]$Path){$p=[IO.Path]::GetFullPath($Path);while($p){if((Test-Path -LiteralPath $p) -and ((Get-Item -LiteralPath $p).Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'reparse path'};$parent=[IO.Path]::GetDirectoryName($p);if($parent -eq $p){break};$p=$parent}}
try{
  CheckPath $Descriptor;CheckPath $PayloadDirectory
  $file=Get-Item -LiteralPath $Descriptor;if($file.Length -gt 32768){throw 'descriptor limit'}
  $descriptorBytes=[IO.File]::ReadAllBytes($file.FullName);$d=[Text.Encoding]::UTF8.GetString($descriptorBytes)|ConvertFrom-Json
  if(($d.schema -isnot [int] -and $d.schema -isnot [long]) -or $d.schema -ne 1 -or $d.profile -ne 'codex/windows/native' -or $d.source_id -ne ('sha256:'+$d.payloads.'register-native.ps1')){throw 'profile'}
  $names=@('register-native.ps1','diagnose-connection.ps1')
  if(@($d.payloads.PSObject.Properties.Name).Count -ne 2){throw 'unexpected payload'}
  $verified=@{}
  foreach($name in $names){
    $digest=$d.payloads.$name;if($digest -notmatch '^[a-f0-9]{64}$'){throw 'digest missing'}
    $path=Join-Path $PayloadDirectory $name;CheckPath $path;$item=Get-Item -LiteralPath $path
    if($item.PSIsContainer -or $item.Length -gt 131072){throw 'payload limit'}
    $bytes=[IO.File]::ReadAllBytes($item.FullName)
    if((Digest $bytes) -ne $digest){$code='PAYLOAD_TAMPERED';throw 'digest mismatch'}
    $verified[$name]=$bytes
  }
  $tempRoot=[IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([char]'\')
  $cache=Join-Path $tempRoot ('boxweave-native-'+[Guid]::NewGuid().ToString('N'));[void][IO.Directory]::CreateDirectory($cache)
  foreach($name in $names){[IO.File]::WriteAllBytes((Join-Path $cache $name),$verified[$name])}
  $copy=Join-Path $cache 'descriptor.json';[IO.File]::WriteAllBytes($copy,$descriptorBytes)
  $arguments=@('-NoProfile','-ExecutionPolicy','Bypass','-File',(Join-Path $cache 'register-native.ps1'),'-Descriptor',$copy,'-AgentHome',$AgentHome)
  if($CodexExecutable){$arguments+=@('-CodexExecutable',$CodexExecutable)}
  if($Replace){$arguments+='-Replace'}
  & powershell.exe @arguments
  $exitCode=$LASTEXITCODE
}catch{@{code=$code;registration='not_completed';overall='incomplete'}|ConvertTo-Json -Compress}
finally{
  if($cache -and (Test-Path -LiteralPath $cache)){
    $resolved=[IO.Path]::GetFullPath($cache);$boundary=[IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([char]'\')+'\'
    if($resolved.StartsWith($boundary,[StringComparison]::OrdinalIgnoreCase) -and [IO.Path]::GetFileName($resolved) -match '^boxweave-native-[a-f0-9]{32}$'){Remove-Item -LiteralPath $resolved -Recurse -Force}
  }
}
exit $exitCode
