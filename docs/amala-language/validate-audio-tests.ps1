param(
  [string]$AudioDir = (Join-Path $PSScriptRoot 'audio-tests'),
  [switch]$Transcribe
)
$ErrorActionPreference = 'Stop'
$files = @('tamil_only.mp3.m4a','english_only.mp3.m4a','tanglish.mp3.m4a')
$rows = foreach ($name in $files) {
  $path = Join-Path $AudioDir $name
  if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
    [pscustomobject]@{ file=$name; readable=$false; duration=''; status='MISSING'; transcript='' }
    continue
  }
  $duration = (& ffprobe -v error -show_entries format=duration -of csv=p=0 -- $path 2>$null).Trim()
  $readable = $LASTEXITCODE -eq 0 -and $duration -as [double] -gt 0
  $transcript = ''
  $status = if ($readable) { 'VALID_AUDIO_ONLY' } else { 'INVALID_AUDIO' }
  if ($Transcribe -and $readable) {
    try {
      $transcript = (& (Join-Path $PSScriptRoot 'transcribe.ps1') -InputPath $path 2>&1 | Out-String).Trim()
      $status = if ($LASTEXITCODE -eq 0) { 'TRANSCRIBED' } else { 'TRANSCRIPTION_FAILED' }
    } catch { $status = 'TRANSCRIPTION_BLOCKED'; $transcript = $_.Exception.Message }
  }
  [pscustomobject]@{ file=$name; readable=$readable; duration=$duration; status=$status; transcript=$transcript }
}
$rows | ConvertTo-Json -Depth 3
