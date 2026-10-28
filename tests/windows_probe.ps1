param([string]$SetupPath,[string]$Python,[string]$Peer)
$ErrorActionPreference='Stop'
$tokens=$null;$errors=$null
$ast=[Management.Automation.Language.Parser]::ParseFile($SetupPath,[ref]$tokens,[ref]$errors)
if($errors.Count){throw 'Syntax error'}
foreach($definition in $ast.FindAll({param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -in @('Quote-ProcessArgument','Test-BoxWeaveMcp')},$true)){
 . ([scriptblock]::Create($definition.Extent.Text))
}
$evidence=@{};$accepted=$false;$code=$null
try{Test-BoxWeaveMcp -Command $Python -Arguments @($Peer) -Timeout 3 -Evidence $evidence;$accepted=$true}
catch{if($_.Exception.Message -match '^([A-Z_]+):'){$code=$Matches[1]}else{$code='UNCLASSIFIED'}}
[pscustomobject]@{accepted=$accepted;code=$code;cleanup=$evidence.cleanup;tool_pages=$evidence.tool_pages} | ConvertTo-Json -Compress
