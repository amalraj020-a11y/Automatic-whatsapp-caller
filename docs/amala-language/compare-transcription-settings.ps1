param(
  [string]$AudioDir = (Join-Path $PSScriptRoot 'audio-tests'),
  [string]$OutputDir = (Join-Path $PSScriptRoot 'comparisons')
)
$ErrorActionPreference = 'Stop'
$cli=$env:AMALA_WHISPER_CLI; $model=$env:AMALA_WHISPER_MODEL
if(-not $cli -or -not (Test-Path -LiteralPath $cli -PathType Leaf)){throw 'AMALA_WHISPER_CLI must point to whisper-cli.exe.'}
if(-not $model -or -not (Test-Path -LiteralPath $model -PathType Leaf) -or $model -match '\.en\.bin$'){throw 'AMALA_WHISPER_MODEL must be multilingual.'}
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
$prompt='Preserve only spoken technical terms, including login, backend, finish, test, result, WhatsApp, filenames, Git commands, and URLs. Do not add terms from this prompt.'
$cases=@(
  @{name='auto'; lang='auto'; prompt=$null},
  @{name='auto_technical_prompt'; lang='auto'; prompt=$prompt}
)
foreach($file in @('tamil_only.mp3.m4a','english_only.mp3.m4a','tanglish.mp3.m4a')){
  $input=Join-Path $AudioDir $file; if(-not(Test-Path -LiteralPath $input)){throw "Missing $file"}
  $base=[IO.Path]::GetFileNameWithoutExtension([IO.Path]::GetFileNameWithoutExtension($file)); $wav=Join-Path $env:TEMP ("amala-compare-$([guid]::NewGuid().ToString('N')).wav")
  try{
    & ffmpeg -nostdin -hide_banner -loglevel error -y -i $input -ar 16000 -ac 1 -c:a pcm_s16le $wav 2>$null
    if($LASTEXITCODE -ne 0){throw "ffmpeg failed for $file"}
    foreach($case in $cases){
      $threadCount = if($env:AMALA_WHISPER_THREADS){$env:AMALA_WHISPER_THREADS}else{'16'}
      $args=@('-m',$model,'-f',$wav,'-l',$case.lang,'-nt','-np','-t',$threadCount); if($case.prompt){$args+=@('--prompt',$case.prompt)}
      $text=((& $cli @args 2>$null) -join [Environment]::NewLine).Trim(); $status=if($LASTEXITCODE -eq 0 -and $text){'TRANSCRIBED'}else{'TRANSCRIPTION_FAILED'}
      [pscustomobject]@{file=$file;setting=$case.name;language=$case.lang;status=$status;transcript=$text} | ConvertTo-Json -Compress | Add-Content -LiteralPath (Join-Path $OutputDir "$base.jsonl") -Encoding UTF8
    }
  }finally{Remove-Item -LiteralPath $wav -Force -ErrorAction SilentlyContinue}
}
Get-ChildItem -LiteralPath $OutputDir -Filter '*.jsonl' | Select-Object -ExpandProperty FullName
