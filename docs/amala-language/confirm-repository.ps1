param(
  [Parameter(Mandatory=$true)][string]$Candidate,
  [string[]]$SearchRoot = @((Join-Path $env:USERPROFILE '.openclaw\workspace'),(Join-Path $env:USERPROFILE 'source')),
  [string]$ConfirmedRepository
)
$ErrorActionPreference = 'Stop'
function Normalize([string]$s) { (($s.ToLowerInvariant() -replace '[^\p{L}\p{Nd}]','')) }
$candidateKey = Normalize $Candidate
$repos = foreach ($root in $SearchRoot) {
  if (-not (Test-Path -LiteralPath $root -PathType Container)) { continue }
  Get-ChildItem -LiteralPath $root -Directory -Force -Recurse -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -eq '.git' -and $_.FullName -notmatch '\\(node_modules|\.git)\\' } |
    ForEach-Object { Get-Item -LiteralPath $_.Parent.FullName }
}
$repos = @($repos | Sort-Object FullName -Unique)
$exact = @($repos | Where-Object { (Normalize $_.Name) -eq $candidateKey })
$status = 'CONFIRMATION_REQUIRED'
$reason = 'No exact repository-name match was found.'
if ($exact.Count -eq 1) { $status = 'EXACT_MATCH_FOUND'; $reason = 'Exact name match; still require confirmation before editing.' }
if ($exact.Count -gt 1) { $reason = 'Multiple exact repository-name matches were found.' }
if ($ConfirmedRepository) {
  $selected = @($repos | Where-Object { $_.FullName -eq $ConfirmedRepository })
  if ($selected.Count -eq 1 -and $exact.Count -eq 1 -and $selected[0].FullName -eq $exact[0].FullName) {
    $status = 'CONFIRMED'; $reason = 'User supplied the exact path after the exact-name check.'
  } else { $status = 'CONFIRMATION_REJECTED'; $reason = 'The confirmed path is not the single exact match.' }
}
$repoNames = @($repos | ForEach-Object { $_.FullName })
$exactNames = @($exact | ForEach-Object { $_.FullName })
[pscustomobject]@{ status=$status; candidate=$Candidate; reason=$reason; repositories=$repoNames; exactMatches=$exactNames } | ConvertTo-Json -Depth 4
