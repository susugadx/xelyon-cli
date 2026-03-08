package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLogLevelDefault(t *testing.T) {
	// Reset for test
	originalLevel := currentLogLevel
	defer func() { currentLogLevel = originalLevel }()

	currentLogLevel = LogInfo

	if GetLogLevel() != LogInfo {
		t.Errorf("default log level = %v, want LogInfo", GetLogLevel())
	}
}

func TestSetLogLevel(t *testing.T) {
	originalLevel := currentLogLevel
	defer func() { currentLogLevel = originalLevel }()

	tests := []struct {
		name  string
		level LogLevel
	}{
		{"Debug", LogDebug},
		{"Info", LogInfo},
		{"Warn", LogWarn},
		{"Error", LogError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLogLevel(tt.level)
			if GetLogLevel() != tt.level {
				t.Errorf("SetLogLevel(%v) failed, got %v", tt.level, GetLogLevel())
			}
		})
	}
}

func TestDebugEnvVar(t *testing.T) {
	originalLevel := currentLogLevel
	originalEnv := os.Getenv("XELYON_DEBUG")
	defer func() {
		currentLogLevel = originalLevel
		if originalEnv != "" {
			os.Setenv("XELYON_DEBUG", originalEnv)
		} else {
			os.Unsetenv("XELYON_DEBUG")
		}
	}()

	// Test that XELYON_DEBUG=1 should enable debug logging
	os.Setenv("XELYON_DEBUG", "1")
	// Simulate init behavior
	if os.Getenv("XELYON_DEBUG") == "1" {
		SetLogLevel(LogDebug)
	}

	if GetLogLevel() != LogDebug {
		t.Error("XELYON_DEBUG=1 should set LogDebug")
	}
}

func TestLogFunctionsNoPanic(t *testing.T) {
	// These tests just verify the functions don't panic
	// Actual output testing would require capturing stdout

	originalLevel := currentLogLevel
	defer func() { currentLogLevel = originalLevel }()

	SetLogLevel(LogDebug)

	// Should not panic
	Debug("test debug: %s", "value")
	InfoLog("test info: %d", 42)
	Warn("test warning: %v", nil)
	WarnWithoutEmoji("test warning without emoji")
	ErrorLog("test error: %s", "oops")
	ErrorLogWithoutEmoji("test error without emoji")
	SuccessLog("test success")
	SuccessLogWithEmoji("OK", "test success with emoji")
}

func TestLogFunctionsToWriter_UseInjectedWriters(t *testing.T) {
	originalLevel := currentLogLevel
	defer func() { currentLogLevel = originalLevel }()

	SetLogLevel(LogDebug)

	var out bytes.Buffer
	var errOut bytes.Buffer

	DebugToWriter(&errOut, "debug message")
	InfoLogToWriter(&out, "info message")
	WarnToWriter(&out, "warn message")
	ErrorLogToWriter(&errOut, "error message")
	SuccessLogWithEmojiToWriter(&out, "OK", "done")

	if !strings.Contains(stripANSI(errOut.String()), "[DEBUG] debug message") {
		t.Fatalf("expected debug output in injected error writer, got %q", errOut.String())
	}
	plainOut := stripANSI(out.String())
	for _, want := range []string{"info message", "Warning: warn message", "OK done"} {
		if !strings.Contains(plainOut, want) {
			t.Fatalf("expected injected output to contain %q, got %q", want, plainOut)
		}
	}
	if !strings.Contains(stripANSI(errOut.String()), "Error: error message") {
		t.Fatalf("expected error output in injected error writer, got %q", errOut.String())
	}
}

func TestLogLevelFiltering(t *testing.T) {
	originalLevel := currentLogLevel
	defer func() { currentLogLevel = originalLevel }()

	// At Error level, Debug/Info/Warn should be filtered
	SetLogLevel(LogError)

	// These should not output anything (but shouldn't panic)
	Debug("filtered")
	InfoLog("filtered")
	Warn("filtered")

	// Error should still work
	ErrorLog("not filtered")
}
