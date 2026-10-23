param([string]$SetupPath,[string]$FixtureRoot,[string]$Case='matched')
$ErrorActionPreference='Stop'
$tokens=$null;$errors=$null
$ast=[Management.Automation.Language.Parser]::ParseFile($SetupPath,[ref]$tokens,[ref]$errors)
if($errors.Count){throw 'Syntax error'}
foreach($definition in $ast.FindAll({param($n) $n -is [Management.Automation.Language.FunctionDefinitionAst] -and $n.Name -in @('Invoke-BoxWeaveDiagnostic','Update-CodexToml','Quote-Toml','Set-TomlSection')},$true)){
 . ([scriptblock]::Create($definition.Extent.Text))
}
$originalProfile=$env:USERPROFILE
try{
 $env:USERPROFILE=$FixtureRoot
 $null=New-Item -ItemType Directory -Path (Join-Path $FixtureRoot '.ssh')
 $AgentHome=Join-Path $FixtureRoot '.codex';$null=New-Item -ItemType Directory -Path $AgentHome
 $key=Join-Path $FixtureRoot '.ssh\boxweave'
 $known=Join-Path $FixtureRoot '.ssh\boxweave_known_hosts'
 $public='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIoY7yK5+VYtbzXBFgHguKKWJFTx98DSeDtmDTfS1zA fixture'
 [IO.File]::WriteAllText($key,'OPAQUE FIXTURE NEVER OPEN')
 [IO.File]::WriteAllText($known,'fixture.invalid '+(($public -split ' ')[0..1] -join ' '))
 $ssh=(Get-Command ssh.exe -ErrorAction Stop).Source
 $argsExpected=@('-T','-i',$key,'-o','IdentitiesOnly=yes','-o','BatchMode=yes','-o','StrictHostKeyChecking=yes','-o',('UserKnownHostsFile='+$known),'-o','HostKeyAlgorithms=ssh-ed25519','-o','ConnectTimeout=15','-p','22','fixture@fixture.invalid','/usr/local/bin/boxweave-mcp')
 $Replace=$false;$BootstrapIdentityFile='';$ProjectPath='';$McpName='boxweave';$ReportPath='';$Timeout=1;$WslDistro=''
 $config=Join-Path $AgentHome 'config.toml'
 [IO.File]::WriteAllText($config,(Update-CodexToml -Path $config -Command $ssh -Arguments $argsExpected))
 $data=@{os='windows';runtime='native';transport='native';agent='codex';ssh_host='fixture.invalid';ssh_port=22;linux_user='fixture';launcher='/usr/local/bin/boxweave-mcp';server_public_key=$public}
 if($Case -eq 'missing'){Remove-Item -LiteralPath $config}
 if($Case -eq 'pin'){[IO.File]::WriteAllText($known,'wrong')}
 if($Case -eq 'invalid'){$data=@{}}
 if($Case -eq 'replace'){$Replace=$true}
 if($Case -in @('report','existing_report')){$ReportPath=Join-Path $FixtureRoot 'report.json'}
 if($Case -eq 'existing_report'){[IO.File]::WriteAllText($ReportPath,'Existing report must survive')}
 $DataBase64=[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes(($data | ConvertTo-Json -Compress)))
 $script:calls=0
 function Test-BoxWeaveMcp($Command,$Arguments,$Timeout,$Evidence){
  $script:calls++
  if($Arguments -notcontains 'NUL' -or $Arguments -notcontains 'IdentityAgent=none'){throw 'Unsafe transport'}
  $Evidence.initialize='passed';$Evidence.discovery='passed';$Evidence.provider_call='passed';$Evidence.protocol_version='2024-11-05';$Evidence.cleanup='completed'
 }
 $beforeKey=[IO.File]::ReadAllText($key);$beforePin=[IO.File]::ReadAllText($known)
 $result=@(Invoke-BoxWeaveDiagnostic)
 $code=[int]$result[-1];$report=ConvertFrom-Json -InputObject $result[0]
 $expectedCode=if($Case -in @('invalid','replace','missing')){2}elseif($Case -in @('pin','existing_report')){1}else{0}
 if($code -ne $expectedCode){throw ('Unexpected code '+$code+' report '+$result[0])}
 if($Case -in @('pin','invalid','replace') -and $script:calls -ne 0){throw 'Protocol invoked before preflight'}
 if($Case -in @('matched','missing','report') -and $report.connection -ne 'passed'){throw 'Connection result lost'}
 if($beforeKey -cne [IO.File]::ReadAllText($key) -or $beforePin -cne [IO.File]::ReadAllText($known)){throw 'Identity/trust modified'}
 if($Case -eq 'report' -and -not (Test-Path -LiteralPath $ReportPath)){throw 'Export missing'}
 if($Case -eq 'existing_report' -and [IO.File]::ReadAllText($ReportPath) -cne 'Existing report must survive'){throw 'Report overwritten'}
 if($result[0].Contains($FixtureRoot)){throw 'Home path leaked'}
 [pscustomobject]@{case=$Case;exit_code=$code;result='passed';powershell=$PSVersionTable.PSVersion.ToString()} | ConvertTo-Json -Compress
}finally{$env:USERPROFILE=$originalProfile}
