param(
    [Parameter(Mandatory=$true)][string]$SetupPath,
    [Parameter(Mandatory=$true)][string]$FixtureRoot,
    [ValidateSet('atomic','adapters','original','nested')][string]$Case = 'atomic'
)
$ErrorActionPreference='Stop'
$tokens=$null;$errors=$null
$ast=[Management.Automation.Language.Parser]::ParseFile($SetupPath,[ref]$tokens,[ref]$errors)
if($errors.Count){throw 'Setup failed syntax parsing'}
$names=@('Write-Utf8NoBom','Backup-File','Quote-Toml','Set-TomlSection','Update-CodexToml','Update-Agents')
foreach($definition in $ast.FindAll({param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -in $names},$true)){
 . ([scriptblock]::Create($definition.Extent.Text))
}
$fixtureDir=Join-Path ([IO.Path]::GetFullPath($FixtureRoot)) ('config-fixture-'+[guid]::NewGuid())
$null=New-Item -ItemType Directory -Path $fixtureDir
$encoding=New-Object Text.UTF8Encoding($false)
function Assert-Fixture([bool]$Condition,[string]$Message){if(-not $Condition){throw $Message}}
$checks=0
if($Case -eq 'original'){
 $path=Join-Path $fixtureDir 'existing.toml'
 $tmp=Join-Path $fixtureDir 'replacement.tmp'
 [IO.File]::WriteAllText($path,'original',$encoding)
 [IO.File]::WriteAllText($tmp,'replacement',$encoding)
 $reproduced=$false
 try{[IO.File]::Replace($tmp,$path,$null)}catch{
  $reproduced=$_.Exception.GetBaseException() -is [ArgumentException]
 }
 Assert-Fixture $reproduced 'Expected the legacy null-to-empty-string failure'
 Assert-Fixture ([IO.File]::ReadAllText($path) -eq 'original') 'Original fixture changed'
 [pscustomobject]@{case=$Case;reproduced=$true;powershell=$PSVersionTable.PSVersion.ToString()} | ConvertTo-Json -Compress
 exit 0
}
if($Case -eq 'atomic'){
 $path=Join-Path $fixtureDir 'new config.toml'
 Write-Utf8NoBom -Path $path -Content 'first'
 Assert-Fixture ([IO.File]::ReadAllText($path) -eq 'first') 'New file content differs';$checks++
 Backup-File $path
 Write-Utf8NoBom -Path $path -Content 'second'
 Assert-Fixture ([IO.File]::ReadAllText($path) -eq 'second') 'Existing file replacement failed';$checks++
 $backups=@(Get-ChildItem -LiteralPath $fixtureDir | Where-Object {$_.Name -like '*.backup-*'})
 Assert-Fixture ($backups.Count -eq 1 -and [IO.File]::ReadAllText($backups[0].FullName) -eq 'first') 'Backup content changed';$checks++
 Write-Utf8NoBom -Path $path -Content 'third'
 Assert-Fixture ([IO.File]::ReadAllText($path) -eq 'third') 'Repeat replacement failed';$checks++
 $bytes=[IO.File]::ReadAllBytes($path)
 Assert-Fixture (-not ($bytes.Length -ge 3 -and $bytes[0] -eq 239 -and $bytes[1] -eq 187 -and $bytes[2] -eq 191)) 'Unexpected UTF8 BOM';$checks++
 $leftovers=@(Get-ChildItem -LiteralPath $fixtureDir | Where-Object {$_.Name -like '.boxweave-*'})
 Assert-Fixture ($leftovers.Count -eq 0) 'Staged files were left behind';$checks++
}elseif($Case -eq 'nested'){
 $McpName='boxweave';$Replace=$true
 $config=Join-Path $fixtureDir 'config.toml'
 $nested="[mcp_servers.boxweave.tools.list_containers]`n# Keep approval policy`napproval_mode = 'approve'`n[mcp_servers.boxweave.tools.connect]`napproval_mode = 'approve'`n"
 $other="[mcp_servers.other]`ncommand = 'other'`n"
 $original="[mcp_servers.boxweave]`ncommand = 'old'`nargs = []`n"+$nested+$other
 [IO.File]::WriteAllText($config,$original,$encoding)
 $updated=Update-CodexToml -Path $config -Command 'wsl.exe' -Arguments @('-d','Ubuntu')
 Assert-Fixture ($updated.Contains($nested+$other)) 'Approval tables or other server changed';$checks++
 Backup-File $config
 Write-Utf8NoBom -Path $config -Content $updated
 $backup=@(Get-ChildItem -LiteralPath $fixtureDir | Where-Object {$_.Name -like '*.backup-*'})
 Assert-Fixture ($backup.Count -eq 1 -and [IO.File]::ReadAllText($backup[0].FullName) -eq $original) 'Original backup missing';$checks++
 $Replace=$false
 $repeated=Update-CodexToml -Path $config -Command 'wsl.exe' -Arguments @('-d','Ubuntu')
 Assert-Fixture ($repeated -eq $updated) 'Repeat with approval tables changed configuration';$checks++
 $rejected=$false
 try{Update-CodexToml -Path $config -Command 'different' -Arguments @() | Out-Null}catch{$rejected=$true}
 Assert-Fixture $rejected 'Conflicting transport did not require replacement';$checks++
 $Replace=$true
 [IO.File]::WriteAllText($config,$original+"[mcp_servers.boxweave.env]`nX = 'y'`n",$encoding)
 $rejected=$false
 try{Update-CodexToml -Path $config -Command 'wsl.exe' -Arguments @() | Out-Null}catch{$rejected=$true}
 Assert-Fixture $rejected 'Unsupported nested configuration was accepted';$checks++
 Assert-Fixture ([IO.File]::ReadAllText($config).Contains("X = 'y'")) 'Rejected configuration was changed';$checks++
}else{
 $McpName='boxweave';$Replace=$true
 $config=Join-Path $fixtureDir 'config.toml'
 $original="# Preserve this preference`n[desktop]`nrunCodexInWindowsSubsystemForLinux = true`n[mcp_servers.other]`ncommand = 'other'`n"
 [IO.File]::WriteAllText($config,$original,$encoding)
 $arguments=@('-d','Ubuntu','-e','ssh','-T','-i','/home/fixture/.ssh/boxweave')
 $updated=Update-CodexToml -Path $config -Command 'wsl.exe' -Arguments $arguments
 Backup-File $config
 Write-Utf8NoBom -Path $config -Content $updated
 $saved=[IO.File]::ReadAllText($config)
 Assert-Fixture ($saved.Contains($original.TrimEnd())) 'Unrelated configuration was changed';$checks++
 Assert-Fixture ($saved.Contains('[mcp_servers.boxweave]')) 'MCP table missing';$checks++
 $repeated=Update-CodexToml -Path $config -Command 'wsl.exe' -Arguments $arguments
 Assert-Fixture ($repeated -eq $saved) 'Repeated configuration is not stable';$checks++
 Write-Utf8NoBom -Path $config -Content $repeated
 $rules=Join-Path $fixtureDir 'AGENTS.md'
 [IO.File]::WriteAllText($rules,"User instructions`n",$encoding)
 Update-Agents -Path $rules -Content 'Fixture remote policy'
 Update-Agents -Path $rules -Content 'Fixture remote policy'
 $policy=[IO.File]::ReadAllText($rules)
 Assert-Fixture ($policy.StartsWith('User instructions')) 'User instructions were lost';$checks++
 Assert-Fixture ([regex]::Matches($policy,'<!-- BEGIN BOXWEAVE MANAGED -->').Count -eq 1) 'Duplicate managed rules';$checks++
 Assert-Fixture ($policy.Contains('Fixture remote policy')) 'Generated policy missing';$checks++
}
[pscustomobject]@{case=$Case;checks=$checks;powershell=$PSVersionTable.PSVersion.ToString();result='passed'} | ConvertTo-Json -Compress
