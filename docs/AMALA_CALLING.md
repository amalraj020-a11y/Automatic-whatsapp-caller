# Amala WhatsApp calling integration

This repository contains the MeowCaller code plus the Amala `talk` command for an interactive WhatsApp voice call.

## Call flow

1. Connect with an already-linked WhatsApp companion device.
2. Place an outgoing WhatsApp call and wait for WhatsApp's explicit accept event.
3. Synthesize a short greeting with Windows SAPI using the installed Microsoft Zira female English voice.
4. Receive decoded mono audio frames from MeowCaller, detect utterances locally, and write temporary 16 kHz WAV clips.
5. Transcribe clips with local multilingual `whisper.cpp` and the small multilingual model.
6. Ask local Ollama for a short reply, then synthesize and queue it into the call.

The outbound greeting's local playback completion does not prove what the remote handset heard. The current voice is English; this does not provide Tamil speech synthesis. Tamil and Tanglish speech output remain unverified.

## Setup

- Windows, Go, PowerShell, FFmpeg, `whisper-cli.exe`, and a multilingual Whisper model.
- A linked-device WhatsApp SQLite store. Keep it outside the repository.
- Local Ollama with a model such as `qwen3:1.7b`.
- Microsoft SAPI voice token `TTS_MS_EN-US_ZIRA_11.0` (female English voice).

Set local environment variables (do not commit values):

```powershell
$env:AMALA_WHISPER_CLI = 'C:\path\to\whisper-cli.exe'
$env:AMALA_WHISPER_MODEL = 'C:\path\to\ggml-small.bin'
$env:AMALA_WHISPER_THREADS = '8'
$env:MEOW_LOG_LEVEL = 'info'
```

Start Ollama locally, then run:

```powershell
go run ./cmd/meowcaller talk --store 'C:\path\to\linked-device.db' --transcriber 'C:\path\to\transcribe.ps1' --model qwen3:1.7b '+<country-code><number>'
```

Use a test handset and consent to recording/transcription before testing. Do not commit phone numbers, linked-device stores, recordings, generated speech, Whisper models, or logs.

## Current safety and limitations

- Speech-to-text runs locally. The call command does not call Groq or OpenAI.
- The call command currently uses local Ollama for reply generation; Groq is not wired into this prototype.
- Audio is temporarily written for transcription and removed after processing.
- Unknown or ambiguous repository names must be confirmed before project edits; see `amala-language/confirm-repository.ps1` and `amala-language/ACTION-RISK.md`.
- The exact local Whisper transcripts and their limitations are recorded in `IMPLEMENTATION_TIMELINE.md`; no WER is claimed because there are no verified reference transcripts.
- The last accepted call delivered a greeting, but the call process received zero decoded audio frames. The relay receive/shutdown diagnostics have since been improved, but the new build has not yet been verified in a live call.
- Windows-specific store-path and POSIX file-mode tests currently fail under Windows semantics. Build succeeds; these test failures are not marked as passing.

## Relevant files

- `cmd/meowcaller/talk.go`: local STT, local reply loop, VAD, female SAPI TTS.
- `livecall.go`, `engine.go`, `engine_media.go`: call lifecycle, answer event, bidirectional media, privacy-safe relay counters, and cancellation cleanup.
- `docs/amala-language/`: Tamil/English/Tanglish policy, repository confirmation gate, offline validator, comparison and safety scripts.
- `IMPLEMENTATION_TIMELINE.md`: privacy-redacted user-request and test timeline.

### Latest live-call verification (2026-10-20)

At approximately 14:33, an accepted call received and decoded inbound audio. Local Whisper transcribed a 4.3-second clip as `So, what were the hopes you have done today?`; local Ollama generated and queued a reply. Raj confirmed hearing the greeting and an interim “give me a minute to think” message. Final reply audibility is not confirmed. Later utterances were dropped while the queue was occupied, so the source was adjusted to use a 500 ms silence window, a larger bounded queue, and shorter generated replies. These adjustments need another live call to verify. The build compiles; two tests still fail due to POSIX assumptions on Windows.

### Follow-up live call and safeguards (2026-10-20)

A call was accepted and the greeting finished locally, but the relay summary recorded zero incoming RTP packets and zero decoded audio frames. The caller ended the call without an inbound utterance reaching Whisper, so no answer could be generated. The specific cause is before transcription; available evidence does not distinguish handset microphone transmission from relay delivery. Current source suppresses VAD during Amala playback/processing and rejects Whisper non-speech markers such as `[BLANK_AUDIO]`, preventing self-echo queueing and fabricated reasoning from empty audio. Focused transcript regression tests pass and the binary builds; an end-to-end reply test still requires a call that delivers inbound speech.

If no inbound media is decoded within 12 seconds of the greeting, Amala now speaks a short microphone/audio-path troubleshooting prompt. This reports that no voice audio reached her rather than claiming she heard speech. The focused build/test verification was rerun after this change.

### Live speech exchange (2026-10-20)

The next retry received and decoded inbound RTP. Local Whisper transcribed `What were the works done to do? Hello. Hello.` and then `Whatever the works you have done today.` Ollama generated replies for both, and the local outbound player finished each reply. Sender-side playback completion is not proof the handset played the audio. Another utterance arrived after a reply, so the call stayed interactive. Relay delivery varied between consecutive calls: the previous attempt had zero RTP, while this attempt received it. The latest source also clears a stale playback callback that could reopen listening when an interim acknowledgement ended; the updated build and focused tests pass.
