package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestParseListenArgsDefaultsToIdentityMessage(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := parseListenArgs(nil, &stderr)
	if err != nil {
		t.Fatalf("parseListenArgs() error = %v", err)
	}
	if cfg.message != "I am Amala, Raj's AI assistant bot. You can message me on WhatsApp or call my boss if needed." {
		t.Fatalf("identity message = %q", cfg.message)
	}
	if cfg.storePath == "" {
		t.Fatal("default store path is empty")
	}
}

func TestParseListenArgsAllowsLocalStoreAndMessage(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := parseListenArgs([]string{"--store", `C:\private\whatsapp.db`, "--message", "I am Amala."}, &stderr)
	if err != nil {
		t.Fatalf("parseListenArgs() error = %v", err)
	}
	if cfg.storePath != `C:\private\whatsapp.db` || cfg.message != "I am Amala." {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestParseListenArgsRejectsUnexpectedArgument(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseListenArgs([]string{"unexpected"}, &stderr)
	if !errors.Is(err, errUsage) {
		t.Fatalf("parseListenArgs() error = %v, want errUsage", err)
	}
}
