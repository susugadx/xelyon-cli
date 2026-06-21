package uiruntime

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDefaultLogLevel_Info(t *testing.T) {
	t.Setenv("XELYON_DEBUG", "")

	if got := defaultLogLevel(); got != LogInfo {
		t.Fatalf("defaultLogLevel() = %v, want %v", got, LogInfo)
	}
}

func TestDefaultLogLevel_DebugEnv(t *testing.T) {
	t.Setenv("XELYON_DEBUG", "1")

	if got := defaultLogLevel(); got != LogDebug {
		t.Fatalf("defaultLogLevel() = %v, want %v", got, LogDebug)
	}
}

func TestLogFunctionsNoPanic(t *testing.T) {
	t.Setenv("XELYON_DEBUG", "1")

	DebugToWriter(io.Discard, "test debug: %s", "value")
	InfoLogToWriter(io.Discard, "test info: %d", 42)
	WarnToWriter(io.Discard, "test warning: %v", nil)
	WarnWithoutEmojiToWriter(io.Discard, "test warning without emoji")
	ErrorLogToWriter(io.Discard, "test error: %s", "oops")
	ErrorLogWithoutEmojiToWriter(io.Discard, "test error without emoji")
	SuccessLogToWriter(io.Discard, "test success")
	SuccessLogWithEmojiToWriter(io.Discard, "OK", "test success with emoji")
}

func TestLogFunctionsToWriter_UseInjectedWriters(t *testing.T) {
	t.Setenv("XELYON_DEBUG", "1")

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

func TestRuntimeLogLevelOverridesDefault(t *testing.T) {
	t.Setenv("XELYON_DEBUG", "")

	var out bytes.Buffer
	var errOut bytes.Buffer
	rt := NewRuntime(nil, &out, &errOut)

	if rt.LogLevel() != LogInfo {
		t.Fatalf("runtime default log level = %v, want %v", rt.LogLevel(), LogInfo)
	}

	rt.SetLogLevel(LogDebug)
	if rt.LogLevel() != LogDebug {
		t.Fatalf("runtime LogLevel = %v, want %v", rt.LogLevel(), LogDebug)
	}

	DebugWithRuntime(rt, "rt-debug")
	if !strings.Contains(stripANSI(errOut.String()), "[DEBUG] rt-debug") {
		t.Fatalf("expected debug output via runtime, got %q", errOut.String())
	}

	var globalErr bytes.Buffer
	DebugToWriter(&globalErr, "global-debug")
	if globalErr.Len() != 0 {
		t.Fatalf("expected no output from DebugToWriter at default info level, got %q", globalErr.String())
	}
}

func TestRuntimeLogLevelFiltering(t *testing.T) {
	t.Setenv("XELYON_DEBUG", "1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	rt := NewRuntime(nil, &out, &errOut)
	rt.SetLogLevel(LogError)

	DebugWithRuntime(rt, "filtered-debug")
	InfoLogWithRuntime(rt, "filtered-info")
	WarnWithRuntime(rt, "filtered-warn")

	if out.Len() != 0 || errOut.Len() != 0 {
		t.Fatalf("expected no output at Error level, out=%q, err=%q", out.String(), errOut.String())
	}

	ErrorLogWithRuntime(rt, "visible-error")
	if !strings.Contains(stripANSI(errOut.String()), "Error: visible-error") {
		t.Fatalf("expected error output, got %q", errOut.String())
	}
}
