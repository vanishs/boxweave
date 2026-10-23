param(
    [Parameter(Mandatory=$true)][string]$SetupPath,
    [Parameter(Mandatory=$true)][string]$FixturePath,
    [ValidateSet('Legacy','Standard')][string]$Mode = 'Legacy',
    [string]$Case = 'pin'
)
$ErrorActionPreference = 'Stop'
$PSNativeCommandArgumentPassing = $Mode
$PSNativeCommandUseErrorActionPreference = $false
$fixture = ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($FixturePath))
$wsl = $fixture.shim
$WslDistro = 'Fixture Distro'
$tokens = $null; $errors = $null
$ast = [Management.Automation.Language.Parser]::ParseFile($SetupPath, [ref]$tokens, [ref]$errors)
if ($errors.Count) { throw 'Setup script failed syntax parsing' }
$names = @('Assert-Exit','ConvertTo-WslScriptCommand','Invoke-WslScript')
$definitions = $ast.FindAll({
    param($node)
    $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -in $names
}, $true)
foreach ($definition in $definitions) {
    . ([scriptblock]::Create($definition.Extent.Text))
}
# Load only shell fixture bodies; never execute the setup script's top-level enrollment.
$source = [IO.File]::ReadAllText($SetupPath)
$pinScript = [regex]::Match($source, "(?ms)^\s*\`$wslPinScript = @'\r?\n(.*?)\r?\n'@").Groups[1].Value
$keyScript = [regex]::Match($source, "(?ms)^\s*\`$wslScript = @'\r?\n(.*?)\r?\n'@").Groups[1].Value
if (-not $pinScript -or -not $keyScript) { throw 'Shell fixture body missing' }
try {
    switch ($Case) {
        'original' {
            # This intentionally reproduces the pre-fix call with embedded quotes.
            & $wsl -d $WslDistro -e sh -c $pinScript sh 'boxweave_known_hosts' $fixture.pin_base64
            exit $LASTEXITCODE
        }
        'pin' {
            Invoke-WslScript -Script $pinScript -ScriptArguments @('boxweave_known_hosts',$fixture.pin_base64) -What 'WSL host pinning'
            Write-Output 'pin passed'
        }
        'key' {
            Invoke-WslScript -Script $keyScript -ScriptArguments @('boxweave') -What 'WSL key generation'
        }
        'home' {
            Invoke-WslScript -Script 'printf "%s\n" "$HOME"' -What 'WSL home lookup'
        }
        'body' {
            Invoke-WslScript -Script $fixture.body -ScriptArguments @($fixture.arguments) -What 'WSL fixture'
        }
        'command' {
            ConvertTo-WslScriptCommand -Script $fixture.body -ScriptArguments @($fixture.arguments)
        }
        default { throw 'Unknown fixture case' }
    }
} catch {
    [Console]::Error.WriteLine($_.Exception.Message)
    exit 1
}
