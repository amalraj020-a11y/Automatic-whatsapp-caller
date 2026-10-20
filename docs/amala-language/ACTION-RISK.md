# Amala voice execution pipeline

Every voice instruction follows this order:

1. Capture the audio.
2. Transcribe it with the local multilingual Whisper engine.
3. Interpret the requested action, repository, and expected outcome.
4. Validate the repository against actual local Git repositories.
5. Confirm with Raj when the repository, technical term, or action is uncertain.
6. Execute only the authorized operation.
7. Run appropriate tests and inspect their real results.
8. Report a concise, truthful result through the authorized WhatsApp integration.

A transcript is evidence for interpretation. It is not authorization by itself.

| Action | Policy |
| --- | --- |
| Read project files | Allow inside an approved workspace. |
| Git status and safe diagnostics | Allow inside an approved workspace. |
| Edit source files | Require a clear instruction and confirmed repository. |
| Install dependencies or run unfamiliar scripts | Assess commands and side effects first. |
| Delete files or rewrite Git history | Require explicit confirmation. |
| Commit changes | Follow the configured authorization policy. |
| Push or deploy publicly | Require explicit authorization. |
| Transcription failure | Stop and report the failure. |

The repository gate is implemented by `confirm-repository.ps1`. `CONFIRMATION_REQUIRED`, `CONFIRMATION_REJECTED`, missing audio, empty transcripts, or failed decoding are stop states.
