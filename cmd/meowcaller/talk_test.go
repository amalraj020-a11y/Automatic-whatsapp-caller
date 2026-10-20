package main

import "testing"

func TestUsableTranscriptRejectsWhisperNonSpeechMarkers(t *testing.T) {
	for _, transcript := range []string{"", "  ", "[BLANK_AUDIO]", "[inaudible]", "[SILENCE]", "[SIDE CONVERSATION]", "[MUSIC]"} {
		if usableTranscript(transcript) {
			t.Errorf("usableTranscript(%q) = true, want false", transcript)
		}
	}
}

func TestUsableTranscriptKeepsRecognizedSpeechVerbatim(t *testing.T) {
	for _, transcript := range []string{"login page finish pannu", "அமலா, என்ன வேலை?", "I said [unclear] then continue"} {
		if !usableTranscript(transcript) {
			t.Errorf("usableTranscript(%q) = false, want true", transcript)
		}
	}
}
