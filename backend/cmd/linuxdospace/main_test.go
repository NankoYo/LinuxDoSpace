package main

import (
	"testing"
	"time"

	"linuxdospace/backend/internal/config"
)

// TestEffectiveHTTPWriteTimeoutDisablesStreamingBreakage verifies that the
// backend never keeps a finite server-level WriteTimeout while long-lived token
// streams are exposed on the same HTTP server.
func TestEffectiveHTTPWriteTimeoutDisablesStreamingBreakage(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{
			WriteTimeout: 15 * time.Second,
		},
	}

	got := effectiveHTTPWriteTimeout(cfg)
	if got != 0 {
		t.Fatalf("expected write timeout to be disabled for streaming endpoints, got %s", got)
	}
}

// TestEffectiveHTTPWriteTimeoutKeepsZero confirms that an already-disabled
// timeout stays disabled instead of being rewritten into another value.
func TestEffectiveHTTPWriteTimeoutKeepsZero(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{
			WriteTimeout: 0,
		},
	}

	got := effectiveHTTPWriteTimeout(cfg)
	if got != 0 {
		t.Fatalf("expected zero write timeout to stay zero, got %s", got)
	}
}
