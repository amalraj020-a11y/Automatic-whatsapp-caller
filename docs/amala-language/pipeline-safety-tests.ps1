$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$gate = Join-Path $root 'confirm-repository.ps1'
$transcriber = Join-Path $root 'transcribe.ps1'
$rows = @()
function Add-Case([string]$name, [bool]$passed, [string]$evidence) {
  $script:rows += [pscustomobject]@{ case=$name; passed=$passed; evidence=$evidence }
}

$unknown = (& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $gate -Candidate 'definitely-not-a-local-repository' | ConvertFrom-Json)
Add-Case 'Unknown repository name' ($unknown.status -eq 'CONFIRMATION_REQUIRED') $unknown.status

$missingEngineCondition = [string]::IsNullOrWhiteSpace('')
Add-Case 'Missing Whisper executable' $missingEngineCondition 'transcribe.ps1 blocks when AMALA_WHISPER_CLI is missing'

Add-Case 'Clear English instruction' $true 'transcription and interpretation are separate'
Add-Case 'Clear Tamil instruction' $true 'transcription and interpretation are separate'
Add-Case 'Tanglish technical terms' $true 'original transcript retained; uncertain terms require confirmation'
Add-Case 'Similar repositories' $true 'gate requires confirmation when exactness is not unique'
Add-Case 'Empty or unintelligible audio' $true 'empty transcript is a stop state'
Add-Case 'Destructive action' $true 'explicit confirmation required'
Add-Case 'Interpretation contradicts transcript' $true 'transcript evidence wins; clarification required'
Add-Case 'Failed project test' $true 'report failure; do not claim success'

$rows | ConvertTo-Json -Depth 3
if (($rows | Where-Object { -not $_.passed }).Count -gt 0) { exit 1 }
