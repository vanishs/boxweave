#requires -version 5.1
<# Publish this folder to a NEW GitHub repository using your own gh credentials. #>
[CmdletBinding()]
param([string]$Repository = 'vanishs/boxweave')
$ErrorActionPreference = 'Stop'
if ($Repository -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') { throw 'Use OWNER/REPO (example: vanishs/boxweave)' }
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $root
try {
    Get-Command gh -ErrorAction Stop | Out-Null
    Get-Command git -ErrorAction Stop | Out-Null
    & gh auth status | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Run gh auth login first.' }
    & gh repo view $Repository 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) { throw "Repository $Repository already exists; refusing overwrite." }
    if (Test-Path -LiteralPath '.git') { throw 'Existing .git detected; refusing to rewrite history.' }
    & git init -b main
    if ($LASTEXITCODE -ne 0) { throw 'git init failed' }
    & git add README.md LICENSE install.sh publish.sh publish.ps1 .gitignore clients server agents providers docs tests .github
    if ($LASTEXITCODE -ne 0) { throw 'git add failed' }
    & git commit -m 'Initial release: BoxWeave'
    if ($LASTEXITCODE -ne 0) { throw 'git commit failed; configure git user.name and user.email first.' }
    & gh repo create $Repository --public --source=. --remote=origin --push
    if ($LASTEXITCODE -ne 0) { throw 'GitHub repo creation/push failed.' }
    Write-Host "Created https://github.com/$Repository"
    Write-Host "Ubuntu installer https://raw.githubusercontent.com/$Repository/main/install.sh"
} finally {
    Pop-Location
}
