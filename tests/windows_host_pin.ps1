param([string]$SetupPath)
$ErrorActionPreference='Stop'
$tokens=$null;$errors=$null
$ast=[System.Management.Automation.Language.Parser]::ParseFile($SetupPath,[ref]$tokens,[ref]$errors)
if($errors.Count){throw 'Setup parse failed'}
foreach($name in @('Write-Utf8NoBom','Backup-File','Set-PinnedHostNative')){
    $node=$ast.Find({param($n) $n -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $n.Name -eq $name},$true)
    Invoke-Expression $node.Extent.Text
}
$root=Join-Path ([IO.Path]::GetTempPath()) ('boxweave-pin-'+[guid]::NewGuid())
$null=New-Item -ItemType Directory $root
try{
    $path=Join-Path $root 'known_hosts'
    $key='AAAAC3NzaC1lZDI1NTE5AAAAILaKi9l2Up9S11oPEd9q0uNvDTM/XRnjS2uGoKiyAOfp'
    $line="192.0.2.1 ssh-ed25519 $key"
    Set-PinnedHostNative $path $line
    $multi=$line+[Environment]::NewLine+[Environment]::NewLine+"[192.0.2.2]:2222 ssh-ed25519 $key"+[Environment]::NewLine
    [IO.File]::WriteAllText($path,$multi)
    Set-PinnedHostNative $path $line
    if([IO.File]::ReadAllText($path) -cne $multi){throw 'Matching multi-host file was changed'}
    Set-PinnedHostNative $path ("[192.0.2.2]:2222 ssh-ed25519 $key")
    Set-PinnedHostNative $path ("192.0.2.3 ssh-ed25519 $key")
    if(-not ([IO.File]::ReadAllText($path)).Contains('192.0.2.1')){throw 'Other host lost'}
    if(-not (Get-ChildItem $root -Filter 'known_hosts.backup-*')){throw 'Missing backup'}
    $before=[IO.File]::ReadAllText($path)
    $rejected=$false
    try{Set-PinnedHostNative $path '192.0.2.1 ssh-ed25519 AAAACHANGED'}catch{$rejected=$true}
    if(-not $rejected -or [IO.File]::ReadAllText($path) -cne $before){throw 'Changed key was accepted'}
    & ssh-keygen -H -f $path 1>$null 2>$null
    if($LASTEXITCODE -ne 0){throw 'Hash fixture failed'}
    $before=[IO.File]::ReadAllText($path)
    Set-PinnedHostNative $path $line
    if([IO.File]::ReadAllText($path) -cne $before){throw 'Hashed pin modified'}
    Write-Output 'Host pin regression cases passed'
}finally{Remove-Item -LiteralPath $root -Recurse -Force}
