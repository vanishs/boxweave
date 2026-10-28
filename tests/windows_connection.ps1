#requires -version 5.1
$ErrorActionPreference='Stop'
$probe=Join-Path $PSScriptRoot '../clients/windows/diagnose-connection.ps1'
$tokens=$null;$errors=$null
[void][Management.Automation.Language.Parser]::ParseFile((Resolve-Path $probe),[ref]$tokens,[ref]$errors)
if($errors.Count){throw 'PowerShell syntax failed'}
$testDir=Join-Path ([IO.Path]::GetTempPath()) ('bw-native-'+[Guid]::NewGuid().ToString('N'))
[void][IO.Directory]::CreateDirectory($testDir)
try {
  $descriptor=Join-Path $testDir 'profile.json'
  $identity=Join-Path $testDir 'synthetic-identity'
  $known=Join-Path $testDir 'known_hosts'
  $d=@{host='reviewed-server.example';port=22;user='reviewed_user';
       server_key='ssh-ed25519 QUFBQQ==';identity_file=$identity;known_hosts=$known}
  function Check([string]$Expected) {
    $d|ConvertTo-Json|Set-Content -LiteralPath $descriptor -Encoding UTF8
    $output=& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $probe -Descriptor $descriptor -Timeout 2
    if($LASTEXITCODE -ne 2){throw 'Expected incomplete preflight exit'}
    if(($output|ConvertFrom-Json).code -ne $Expected){throw ('Wrong category: '+$output)}
  }
  Check 'IDENTITY_MISSING'
  [IO.File]::WriteAllText($identity,'synthetic fixture, not a private key')
  [IO.File]::WriteAllText($known,'reviewed-server.example ssh-ed25519 QkJCQg==')
  Check 'PIN_MISMATCH'
  $d.host='invalid host'
  Check 'INPUT_INVALID'
  'PASS: PowerShell AST and 3 no-network preflight cases; real SSH/registration unverified'
} finally {
  foreach($file in @($descriptor,$identity,$known)){if(Test-Path -LiteralPath $file){Remove-Item -LiteralPath $file}}
  Remove-Item -LiteralPath $testDir
}
