#requires -version 5.1
param([Parameter(Mandatory=$true)][string]$KeyDirectory)
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$ErrorActionPreference='Stop';$code='INPUT_INVALID';$process=$null
try {
  if($KeyDirectory -notmatch '^[A-Za-z]:[\\/]' -or $KeyDirectory -match '["\r\n]'){throw 'absolute local path required'}
  $folder=[IO.Path]::GetFullPath($KeyDirectory).TrimEnd([char]'\',[char]'/')
  $parent=[IO.Path]::GetDirectoryName($folder)
  if(-not $parent -or -not(Test-Path -LiteralPath $parent -PathType Container) -or (Test-Path -LiteralPath $folder)){throw 'fresh directory with existing parent required'}
  $walk=$parent
  while($walk){if((Get-Item -LiteralPath $walk).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'reparse path'};$walk=[IO.Path]::GetDirectoryName($walk)}
  $code='PERMISSIONS_UNAVAILABLE'
  [void][IO.Directory]::CreateDirectory($folder)
  $sid=[Security.Principal.WindowsIdentity]::GetCurrent().User
  $acl=New-Object Security.AccessControl.DirectorySecurity
  $acl.SetAccessRuleProtection($true,$false);$acl.SetOwner($sid)
  foreach($principal in @($sid,(New-Object Security.Principal.SecurityIdentifier('S-1-5-18')))){
    $rule=New-Object Security.AccessControl.FileSystemAccessRule($principal,'FullControl','ContainerInherit,ObjectInherit','None','Allow')
    [void]$acl.AddAccessRule($rule)
  }
  Set-Acl -LiteralPath $folder -AclObject $acl
  $actual=Get-Acl -LiteralPath $folder
  if(-not $actual.AreAccessRulesProtected){throw 'ACL not protected'}
  $code='RUNTIME_UNAVAILABLE'
  $keygen=(Get-Command ssh-keygen.exe -CommandType Application -ErrorAction Stop).Source
  $key=Join-Path $folder 'id_ed25519'
  $start=New-Object Diagnostics.ProcessStartInfo
  $start.FileName=$keygen;$start.Arguments='-q -t ed25519 -N "" -C boxweave-v2-native -f "'+$key+'"'
  $start.UseShellExecute=$false;$start.CreateNoWindow=$true
  $start.RedirectStandardOutput=$true;$start.RedirectStandardError=$true
  $process=New-Object Diagnostics.Process;$process.StartInfo=$start
  [void]$process.Start()
  $out=$process.StandardOutput.BaseStream.CopyToAsync([IO.Stream]::Null)
  $err=$process.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null)
  $code='KEYGEN_FAILED'
  if(-not $process.WaitForExit(30000)){$process.Kill();throw 'deadline'}
  if($process.ExitCode -ne 0 -or -not(Test-Path -LiteralPath $key -PathType Leaf)){throw 'key generation'}
  $code='PERMISSIONS_UNAVAILABLE'
  # OpenSSH may apply its own file ACL; explicitly constrain only the new key file.
  $icacls=(Get-Command icacls.exe -CommandType Application -ErrorAction Stop).Source
  $allowed=@($sid.Value,'S-1-5-18')
  $aclArgs=@($key,'/inheritance:r','/grant:r',('*'+$sid.Value+':F'),'*S-1-5-18:F')
  $existing=(Get-Acl -LiteralPath $key).GetAccessRules($true,$true,[Security.Principal.SecurityIdentifier])
  foreach($other in @($existing|ForEach-Object {$_.IdentityReference.Value}|Sort-Object -Unique)){
    if($allowed -notcontains $other){$aclArgs+=@('/remove',('*'+$other))}
  }
  & $icacls @aclArgs *> $null
  if($LASTEXITCODE -ne 0){throw 'key ACL publication failed'}
  $fileRules=(Get-Acl -LiteralPath $key).GetAccessRules($true,$true,[Security.Principal.SecurityIdentifier])
  foreach($rule in $fileRules){if(@($sid.Value,'S-1-5-18') -notcontains $rule.IdentityReference.Value){throw 'unexpected key ACL'}}
  $publicFile=$key+'.pub'
  $public=[IO.File]::ReadAllText($publicFile).Trim()
  if($public -notmatch '^ssh-ed25519 [A-Za-z0-9+/]+={0,2} boxweave-v2-native$'){throw 'public key format'}
  @{code='OK';identity_file=$key;public_key_file=$publicFile;remote_enrollment='pending';connection='unverified';overall='incomplete'}|ConvertTo-Json -Compress
  $status=0
}catch{@{code=$code;remote_enrollment='not_completed';overall='incomplete'}|ConvertTo-Json -Compress;$status=2}
finally{if($process){$process.Dispose()}}
exit $status
