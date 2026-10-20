param([Parameter(Mandatory=$true)][string]$InputPath)
$ErrorActionPreference = 'Stop'
$whisper = $env:AMALA_WHISPER_CLI
$model = $env:AMALA_WHISPER_MODEL
$threads = if ($env:AMALA_WHISPER_THREADS) { [int]$env:AMALA_WHISPER_THREADS } else { 16 }
if (-not $whisper -or -not (Test-Path -LiteralPath $whisper -PathType Leaf)) { throw 'AMALA_WHISPER_CLI must point to whisper-cli.exe.' }
if (-not $model -or -not (Test-Path -LiteralPath $model -PathType Leaf) -or $model -match '\.en\.bin$') { throw 'AMALA_WHISPER_MODEL must point to a multilingual ggml model.' }
if (-not (Test-Path -LiteralPath $InputPath -PathType Leaf)) { throw 'Audio attachment was not found.' }
$scratch = Join-Path ([System.IO.Path]::GetTempPath()) ('amala-stt-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $scratch | Out-Null
try {
  $wav = Join-Path $scratch 'input.wav'
  $savedPreference = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  & ffmpeg -nostdin -hide_banner -loglevel error -y -i $InputPath -ar 16000 -ac 1 -c:a pcm_s16le $wav 2>$null
  $ffmpegExit = $LASTEXITCODE
  if ($ffmpegExit -ne 0) { $ErrorActionPreference = $savedPreference; throw 'ffmpeg could not decode the audio.' }
  $lines = & $whisper -m $model -f $wav -l auto -nt -np -t $threads 2>$null
  $whisperExit = $LASTEXITCODE
  $ErrorActionPreference = $savedPreference
  if ($whisperExit -ne 0) { throw 'Local Whisper transcription failed.' }
  $transcript = ($lines -join [Environment]::NewLine).Trim()
  if ([string]::IsNullOrWhiteSpace($transcript)) { throw 'No speech was recognized.' }
  Write-Output $transcript
} finally {
  Remove-Item -LiteralPath $scratch -Recurse -Force -ErrorAction SilentlyContinue
}
