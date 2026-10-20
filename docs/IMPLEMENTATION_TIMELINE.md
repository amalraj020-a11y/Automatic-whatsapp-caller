# Amala calling project: request and implementation timeline

This is a privacy-redacted record of the user requests and engineering outcomes for this project, prepared on 2026-10-20. Personal phone numbers, linked-device identifiers, credentials, raw audio, and runtime packet logs are intentionally omitted.

## 2026-10-19 — language and safety requirements

- Raj requested Tamil script, English, and naturally mixed Tamil-English (Tanglish), including Romanized Tamil, with technical terms, repository names, filenames, commands, URLs, and programming terminology preserved accurately.
- The request preferred local multilingual Whisper for speech recognition and local open-source Tamil speech synthesis where a suitable voice exists. It explicitly cautioned that transcription support does not prove Tamil TTS quality.
- The user requested tests for Tamil-only, English-only, and mixed-language recordings; license, hardware, accuracy, and language-gap documentation; and no paid speech API dependencies where feasible.
- The supplied sample instruction was: “Amala, inniku Multimax project-la login page finish pannu, tests run pannu, result WhatsApp-la send pannu.” The project name must be verified against local repositories before edits.
- The user later stressed that speech must pass capture, transcription, interpretation, repository validation, confirmation, authorized execution, verification, and reporting. Destructive work, pushes, and deployment require explicit authorization.
- The user wanted a separate WhatsApp calling device/number for Amala and repeatedly reported QR-linking and call setup issues. The linked-device store now exists locally; its identifier and contents are not included here.

## 2026-10-19 — local transcription results

Three real recordings were tested locally (Tamil 9.7 s, English 10.2 s, Tanglish 14.3 s). Their audio is not included in this repository.

The saved model-generated outputs were:

- Tamil: `அமலா, இன்னுக்கு மல்டிமாக்சில் என்ன வேலை முடித்துது என்று சொல்லு, நாளைக்கு என்ன வேலையெல்லாம் செய்யவேண்டும் என்பதையும் எனக்குத் தெரிவி.`
- English: `Amala, check today's completed task, identify any errors and send me the summary of the remaining work on WhatsApp and tomorrow WhatsApp plan.`
- Tanglish recording: `அம்மலா இன்றைக்கு மல்டிமாக்ச் பிராஜக்கில் லோகின் பேச் பினிஷ் பண்ணு தெஸ் பண்ணு ரிச்சல்து வாட்சாப்பில் செய்துவிடும் அன்று இன்றைக்கு என்னவெல்லாம் ஓக் நடந்தது அப்படியின்றாது என்று எனக்கு சொல்லும்.`

These are model outputs, not verified reference transcripts. The Tanglish result contains recognition errors and writes English terms in Tamil script; it does not establish the exact Latin spelling of the repository name. The technical-prompt comparison lost spoken Tanglish content, so the original unprompted transcript was retained. No word error rate was computed.

## 2026-10-20 — voice call reports and testing

- Raj reported that Amala's greeting played, but she did not respond after he spoke. He requested a female voice.
- The client was changed to wait for WhatsApp's explicit call-accepted event before greeting, to use the installed Microsoft Zira female English voice, to disable Qwen's hidden reasoning text for spoken replies, and to lower the VAD threshold. The voice is English; Tamil speech output is not claimed.
- At approximately 14:05, an attempted call was not answered by WhatsApp and timed out before the greeting.
- At approximately 14:19, another call was accepted; the greeting played. Raj said he spoke, but Amala did not reply. The call ended around 14:22.
- That call reported zero decoded frames, zero VAD speech onsets, and zero completed clips. This locates the failure before Whisper and local reasoning: no decoded caller audio reached the application's listener.
- After that test, the relay reader was changed to close its DataChannel on call cancellation (unblocking its receive loop), relay packet-kind counters were added, and raw audio payloads and call keys were removed from diagnostics.
- The updated binary builds successfully, but the relay cleanup/counters have not yet been verified in another live call. No claim is made that the microphone path is fixed.
- Two command-package tests still fail because their assertions expect POSIX path and file-permission behavior on Windows. These failures remain visible and are not reported as a passing suite.

## Cost, privacy, and provider record

- Local Whisper and local Ollama were used for speech processing and spoken reply generation. No Groq transcription call and no OpenAI API call were made in these tests.
- The current phone `talk` prototype uses local Ollama; the requested Groq-primary reasoning provider is not yet integrated into this calling loop.
- User audio, the WhatsApp database, phone number, call identifiers, model weights, binaries, and per-call logs are excluded from Git tracking.

## 2026-10-20 — inbound audio and reply generation verified in a live call

- At approximately 14:33, an accepted call delivered incoming RTP to the application. The relay decoded audio, detected a speech-level frame, and captured a 4.3-second utterance.
- Local Whisper produced the exact output `So, what were the hopes you have done today?` This is the model output, not a verified reference transcript.
- Local Ollama generated: `I hope to have made a difference today. I'm looking forward to hearing what you think.` The reply was synthesized and queued. Raj reported hearing the greeting and an interim “give me a minute to think” message; whether he heard the final synthesized reply is unconfirmed.
- While speech processing was underway, later clips were dropped because the queue was full. To improve responsiveness, the source now waits for 500 ms of silence before ending an utterance, holds up to eight queued utterances, and asks Ollama for a single short spoken sentence with a smaller output limit. These newest changes have not yet been verified in another live call.
- The build compiles. The Windows test suite is not fully green: two existing tests assume POSIX path and file-mode behavior. No claim of a full passing test suite is made.

### Follow-up live call and transcript/echo safeguards (2026-10-20, approximately 18:03–18:06 IST)

- The test call was accepted, and the greeting WAV finished in the local sender. The caller ended the call approximately three minutes later.
- The privacy-safe relay summary showed `rtp_packets=0`, `decoded_audio_frames=0`, and zero captured speech clips. No speech reached Whisper on this attempt, so no spoken reply was generated. This locates this specific failure before transcription and reasoning; the log cannot determine whether the handset sent no microphone audio or the relay failed to deliver it.
- Added VAD gating so Amala’s own greeting/reply playback is not reprocessed as a new caller utterance. VAD remains disabled while a captured turn is transcribed/reasoned/spoken, then reopens for the next turn. Empty and known Whisper non-speech markers (including `[BLANK_AUDIO]`) are rejected; Amala asks the caller to repeat instead of asking Ollama to invent a response.
- Focused `TestUsableTranscript` regression tests pass. The updated binary builds successfully. These fixes still need a call with verified inbound speech to validate end-to-end reply playback.
- Added a one-time spoken troubleshooting prompt after 12 seconds without inbound media, so the call does not remain silent after the greeting when no remote audio reaches Amala.

### End-to-end speech exchange on the updated call build (2026-10-20, approximately 18:26 IST)

- The retry received RTP from the relay and decoded inbound audio. Whisper transcribed the first captured turn as `What were the works done to do? Hello. Hello.`; local Ollama replied `Hello. I'm Amala. What were the works done to do?`
- The next captured turn was transcribed as `Whatever the works you have done today.`; Ollama replied `I have done my best today.` The local outbound player completed playback for both replies. Remote handset audibility cannot be proven from sender-side logs alone.
- A further utterance was captured after the second reply, demonstrating that the live call continued beyond one exchange. This call disproves the earlier assumption that the receiver always gets zero RTP: relay delivery was intermittent between calls.
- Fixed a callback reuse bug found during this call: an earlier reply's completion callback could fire when the following interim acknowledgement ended, reopening VAD while reasoning was still in progress. The latest build clears the callback for interim audio; this final callback fix was built and the focused transcript tests passed after the observed exchange.
