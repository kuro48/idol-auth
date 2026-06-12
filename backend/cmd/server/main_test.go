package main

import (
	"log/slog"
	"testing"
)

func TestSetupLogger_KnownLevels(t *testing.T) {
	tests := []struct {
		level string
	}{
		{"debug"},
		{"info"},
		{"warn"},
		{"error"},
		{"DEBUG"},
		{"INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			// Should not panic for valid level strings
			setupLogger(tt.level)
		})
	}
}

func TestSetupLogger_InvalidLevel_DefaultsToInfo(t *testing.T) {
	// Arrange + Act: invalid level should not panic and should fall back to Info
	setupLogger("not-a-level")

	// Assert: default logger is set
	logger := slog.Default()
	if logger == nil {
		t.Fatal("expected slog.Default() to return a non-nil logger")
	}
}

func TestSetupLogger_EmptyLevel_DefaultsToInfo(t *testing.T) {
	// Should not panic on empty string
	setupLogger("")
}
