package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func resetDefaultRuntimeLogLevelForTest(t *testing.T, level LogLevel, set bool) {
	t.Helper()
	defaultRuntime.mu.Lock()
	originalLevel := defaultRuntime.logLevel
	originalSet := defaultRuntime.logLevelSet
	defaultRuntime.logLevel = level
	defaultRuntime.logLevelSet = set
	defaultRuntime.mu.Unlock()
	t.Cleanup(func() {
		defaultRuntime.mu.Lock()
		defaultRuntime.logLevel = originalLevel
		defaultRuntime.logLevelSet = originalSet
		defaultRuntime.mu.Unlock()
	})
}

func TestLogLevelDefault(t *testing.T) {
	originalEnv := os.Getenv("XELYON_DEBUG")
	t.Cleanup(func() {
		if originalEnv != "" {
			os.Setenv("XELYON_DEBUG", originalEnv)
		} else {
			os.Unsetenv("XELYON_DEBUG")
		}
	})
	os.Unsetenv("XELYON_DEBUG")
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)

	if GetLogLevel() != LogInfo {
		t.Errorf("default log level = %v, want LogInfo", GetLogLevel())
	}
}

func TestSetLogLevel(t *testing.T) {
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)

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
	originalEnv := os.Getenv("XELYON_DEBUG")
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)
	t.Cleanup(func() {
		if originalEnv != "" {
			os.Setenv("XELYON_DEBUG", originalEnv)
		} else {
			os.Unsetenv("XELYON_DEBUG")
		}
	})

	// Test that XELYON_DEBUG=1 should enable debug logging
	os.Setenv("XELYON_DEBUG", "1")
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)

	if GetLogLevel() != LogDebug {
		t.Error("XELYON_DEBUG=1 should set LogDebug")
	}
}

func TestLogFunctionsNoPanic(t *testing.T) {
	// These tests just verify the functions don't panic
	// Actual output testing would require capturing stdout

	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)
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
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)
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
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)
	// At Error level, Debug/Info/Warn should be filtered
	SetLogLevel(LogError)

	// These should not output anything (but shouldn't panic)
	Debug("filtered")
	InfoLog("filtered")
	Warn("filtered")

	// Error should still work
	ErrorLog("not filtered")
}

func TestRuntimeLogLevel(t *testing.T) {
	// global を Info にする
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)
	SetLogLevel(LogInfo)

	var out bytes.Buffer
	var errOut bytes.Buffer
	rt := NewRuntime(nil, &out, &errOut)

	// 未設定の場合は global に fallback
	if rt.LogLevel() != LogInfo {
		t.Fatalf("runtime without logLevelSet should fallback to global, got %v", rt.LogLevel())
	}

	// runtime に Debug を設定
	rt.SetLogLevel(LogDebug)
	if rt.LogLevel() != LogDebug {
		t.Fatalf("runtime LogLevel = %v, want LogDebug", rt.LogLevel())
	}

	// global は変わっていない
	if GetLogLevel() != LogInfo {
		t.Fatalf("global level should stay LogInfo, got %v", GetLogLevel())
	}

	// WithRuntime 系が runtime のレベルを参照する
	DebugWithRuntime(rt, "rt-debug")
	if !strings.Contains(stripANSI(errOut.String()), "[DEBUG] rt-debug") {
		t.Fatalf("expected debug output via runtime, got %q", errOut.String())
	}

	// global は Info なので Debug は出ない
	var globalErr bytes.Buffer
	DebugToWriter(&globalErr, "global-debug")
	if globalErr.Len() != 0 {
		t.Fatalf("expected no output from global DebugToWriter at Info level, got %q", globalErr.String())
	}
}

func TestRuntimeLogLevelFiltering(t *testing.T) {
	resetDefaultRuntimeLogLevelForTest(t, LogInfo, false)
	SetLogLevel(LogDebug)

	var out bytes.Buffer
	var errOut bytes.Buffer
	rt := NewRuntime(nil, &out, &errOut)
	rt.SetLogLevel(LogError)

	// runtime が Error レベル → Debug/Info/Warn は出力されない
	DebugWithRuntime(rt, "filtered-debug")
	InfoLogWithRuntime(rt, "filtered-info")
	WarnWithRuntime(rt, "filtered-warn")

	if out.Len() != 0 || errOut.Len() != 0 {
		t.Fatalf("expected no output at Error level, out=%q, err=%q", out.String(), errOut.String())
	}

	// Error は出力される
	ErrorLogWithRuntime(rt, "visible-error")
	if !strings.Contains(stripANSI(errOut.String()), "Error: visible-error") {
		t.Fatalf("expected error output, got %q", errOut.String())
	}
}
