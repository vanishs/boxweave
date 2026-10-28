#requires -version 5.1
<#
Observation-only connection probe. Uses an operator-reviewed public descriptor.
No enrollment, registration repair, key generation or trust-file modification.
#>
param([Parameter(Mandatory=$true)][string]$Descriptor, [ValidateRange(1,120)][int]$Timeout=30)
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$ErrorActionPreference='Stop'
$code='INPUT_INVALID'
$process=$null
$deadline=[DateTime]::UtcNow.AddSeconds($Timeout)
function Remaining {
  $n=[int]($deadline-[DateTime]::UtcNow).TotalMilliseconds
  if($n -le 0){$script:code='DEADLINE_EXCEEDED';throw 'deadline'}
  return $n
}
function QuoteArg([string]$Value) {
  if($Value -match '["\r\n]' -or $Value.EndsWith('\')){throw 'unsupported argument'}
  return '"'+$Value+'"'
}
function SendMessage($Value) {
  $task=$process.StandardInput.WriteLineAsync(($Value|ConvertTo-Json -Depth 12 -Compress))
  if(-not $task.Wait((Remaining))){$script:code='DEADLINE_EXCEEDED';throw 'write deadline'}
  if(-not $process.StandardInput.FlushAsync().Wait((Remaining))){$script:code='DEADLINE_EXCEEDED';throw 'flush deadline'}
}
$nextId=0
$totalChars=0
function ValidateReadOnlyResult($Result) {
  if($Result.isError -or -not ($Result.content -is [array]) -or $Result.content.Count -eq 0){throw 'tool failed'}
  foreach($block in $Result.content){
    if($block.type -ne 'text' -or -not ($block.text -is [string]) -or [string]::IsNullOrWhiteSpace($block.text)){throw 'unsupported tool result'}
  }
}
function CallMcp([string]$Method,$Params) {
  $script:nextId++
  $id=$nextId
  SendMessage @{jsonrpc='2.0';id=$id;method=$Method;params=$Params}
  for($message=0;$message -lt 256;$message++){
    $line=New-Object Text.StringBuilder
    while($true){
      $buffer=New-Object char[] 1
      $read=$process.StandardOutput.ReadAsync($buffer,0,1)
      if(-not $read.Wait((Remaining))){$script:code='DEADLINE_EXCEEDED';throw 'read deadline'}
      if($read.Result -eq 0){$script:code='TRANSPORT_FAILED';throw 'closed stream'}
      $script:totalChars++
      if($line.Length -ge 1048576 -or $script:totalChars -gt 4194304){$script:code='RESPONSE_LIMIT';throw 'output limit'}
      if($buffer[0] -eq [char]10){break}
      [void]$line.Append($buffer[0])
    }
    $script:code='PROTOCOL_INVALID'
    $reply=$line.ToString()|ConvertFrom-Json
    if($reply.jsonrpc -ne '2.0'){throw 'protocol'}
    if($reply.PSObject.Properties.Name -contains 'method'){
      if($reply.PSObject.Properties.Name -contains 'id'){
        SendMessage @{jsonrpc='2.0';id=$reply.id;error=@{code=-32601;message='Unsupported client request'}}
      }
      continue
    }
    if(($reply.id -isnot [int] -and $reply.id -isnot [long]) -or $reply.id -ne $id -or ($reply.PSObject.Properties.Name -contains 'error') -or
       -not ($reply.PSObject.Properties.Name -contains 'result')){throw 'invalid response'}
    return $reply.result
  }
  $script:code='RESPONSE_LIMIT';throw 'message limit'
}
try {
  $file=Get-Item -LiteralPath $Descriptor
  if($file.Length -gt 16384 -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'descriptor'}
  $d=[IO.File]::ReadAllText($file.FullName)|ConvertFrom-Json
  if($d.host -notmatch '^[a-zA-Z0-9][a-zA-Z0-9.-]{0,252}$' -or
     $d.user -notmatch '^[a-z_][a-z0-9_-]{0,63}$' -or
     $d.port -lt 1 -or $d.port -gt 65535 -or
     $d.server_key -notmatch '^ssh-ed25519 [A-Za-z0-9+/]+={0,3}$'){throw 'profile'}
  $code='IDENTITY_MISSING'
  $identity=Get-Item -LiteralPath $d.identity_file
  if($identity.PSIsContainer -or ($identity.Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'identity'}
  $code='PIN_MISSING'
  $known=Get-Item -LiteralPath $d.known_hosts
  if($known.Length -gt 1048576 -or ($known.Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'trust file'}
  $endpoint=if($d.port -eq 22){$d.host}else{'['+$d.host+']:'+$d.port}
  $pins=@([IO.File]::ReadAllLines($known.FullName)|Where-Object {
    $parts=$_ -split '\s+'
    $parts.Count -ge 3 -and ($parts[0] -split ',') -contains $endpoint
  })
  if($pins.Count -eq 0){throw 'pin absent'}
  $code='PIN_MISMATCH'
  $matched=@($pins|Where-Object { $p=$_ -split '\s+'; ($p[1]+' '+$p[2]) -ceq $d.server_key })
  if($matched.Count -eq 0){throw 'pin mismatch'}
  $code='RUNTIME_MISSING'
  $ssh=(Get-Command ssh.exe -ErrorAction Stop).Source
  $sshArgs=@('-F','NUL','-o','BatchMode=yes','-o','StrictHostKeyChecking=yes','-o','IdentitiesOnly=yes',
    '-o','ConnectTimeout=15','-o',('UserKnownHostsFile='+$known.FullName),'-i',$identity.FullName,
    '-p',[string]$d.port,($d.user+'@'+$d.host),'/usr/local/bin/boxweave-mcp')
  $start=New-Object Diagnostics.ProcessStartInfo
  $start.FileName=$ssh
  $start.Arguments=($sshArgs|ForEach-Object {QuoteArg $_}) -join ' '
  $start.UseShellExecute=$false
  $start.CreateNoWindow=$true
  $start.RedirectStandardInput=$true
  $start.RedirectStandardOutput=$true
  $start.RedirectStandardError=$true
  $start.StandardOutputEncoding=New-Object Text.UTF8Encoding($false,$true)
  $process=New-Object Diagnostics.Process
  $process.StartInfo=$start
  $code='TRANSPORT_FAILED'
  [void]$process.Start()
  # Discard diagnostic bytes without retaining or printing potentially sensitive stderr.
  $drain=$process.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null)
  $r=CallMcp 'initialize' @{protocolVersion='2024-11-05';capabilities=@{};clientInfo=@{name='boxweave-native-connection';version='0.2.1'}}
  $code='PROTOCOL_UNSUPPORTED'
  if($r.protocolVersion -ne '2024-11-05'){throw 'unsupported protocol'}
  $code='PROTOCOL_INVALID'
  if($null -eq $r.capabilities -or $null -eq $r.capabilities.tools){throw 'capability shape'}
  SendMessage @{jsonrpc='2.0';method='notifications/initialized'}
  $seen=@{};$params=@{};$found=$false
  for($page=0;$page -lt 16;$page++){
    $r=CallMcp 'tools/list' $params
    $code='PROTOCOL_INVALID'
    if(-not ($r.tools -is [array])){throw 'tools shape'}
    if(@($r.tools|Where-Object {$_.name -eq 'list_containers'}).Count){$found=$true;break}
    if(-not $r.nextCursor){break}
    if($seen.ContainsKey([string]$r.nextCursor)){throw 'cursor repeat'}
    $seen[[string]$r.nextCursor]=$true;$params=@{cursor=$r.nextCursor}
  }
  $code='TOOL_UNAVAILABLE'
  if(-not $found){throw 'tool absent'}
  $r=CallMcp 'tools/call' @{name='list_containers';arguments=@{}}
  $code='PROVIDER_CALL_FAILED'
  ValidateReadOnlyResult $r
  $code='OK'
  @{connection='passed';overall='incomplete';registration='unverified';agent_ui='unverified';code=$code;protocol='2024-11-05';remote_cleanup='unverified'}|ConvertTo-Json -Compress
  $exitCode=0
} catch {
  @{connection='not_passed';overall='incomplete';code=$code;registration='unverified';agent_ui='unverified'}|ConvertTo-Json -Compress
  $exitCode=2
} finally {
  if($process){
    try{if(-not $process.HasExited){$process.Kill();[void]$process.WaitForExit(2000)}}catch{}
    $process.Dispose()
  }
}
exit $exitCode
