package agent

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRunHeadlessWithConfig_UsesFunctionCallingHistoryForToolLoop(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := testSubDir(t)
	testFile := fmt.Sprintf("%s/probe.txt", dir)
	if err := os.WriteFile(testFile, []byte("hello from headless\n"), 0644); err != nil {
		t.Fatal(err)
	}

	provider := &headlessHistoryProbeProvider{
		responses: []string{
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, testFile),
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "Read the probe file", "test-model", provider, newProjectMapDisabledConfig())
	if result.Status != "success" {
		t.Fatalf("RunHeadlessWithConfig() status = %q, want success", result.Status)
	}
	if len(provider.histories) != 2 {
		t.Fatalf("provider histories = %d, want 2", len(provider.histories))
	}

	secondHistory := provider.histories[1]
	if len(secondHistory) != 3 {
		t.Fatalf("second history length = %d, want 3", len(secondHistory))
	}
	if secondHistory[0].Role != "user" {
		t.Fatalf("history[0].Role = %q, want user", secondHistory[0].Role)
	}
	if secondHistory[1].Role != "assistant" {
		t.Fatalf("history[1].Role = %q, want assistant", secondHistory[1].Role)
	}
	if len(secondHistory[1].ToolCalls) != 1 {
		t.Fatalf("history[1].ToolCalls length = %d, want 1", len(secondHistory[1].ToolCalls))
	}

	toolCall := secondHistory[1].ToolCalls[0]
	if toolCall.ID == "" {
		t.Fatal("history[1].ToolCalls[0].ID is empty, want rescue tool_call_id")
	}
	if toolCall.Function.Name != "gather_context" {
		t.Errorf("history[1].ToolCalls[0].Function.Name = %q, want gather_context", toolCall.Function.Name)
	}

	if secondHistory[2].Role != "tool" {
		t.Fatalf("history[2].Role = %q, want tool", secondHistory[2].Role)
	}
	if secondHistory[2].ToolCallID != toolCall.ID {
		t.Errorf("history[2].ToolCallID = %q, want %q", secondHistory[2].ToolCallID, toolCall.ID)
	}
	if secondHistory[2].ToolName != "gather_context" {
		t.Errorf("history[2].ToolName = %q, want gather_context", secondHistory[2].ToolName)
	}
	if !strings.Contains(secondHistory[2].Content, "hello from headless") {
		t.Errorf("history[2].Content = %q, want gather_context output", secondHistory[2].Content)
	}
}

func TestRunHeadless_CallsCleanup(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockProvider{name: "test"}
	_ = RunHeadlessWithConfig(context.Background(), "hello", "test-model", provider, newProjectMapDisabledConfig())

	if called.Load() != 1 {
		t.Errorf("Cleanup was called %d times, want 1", called.Load())
	}
}

func TestRunHeadless_CallsCleanupOnError(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockErrorProvider{}
	result := RunHeadlessWithConfig(context.Background(), "hello", "test-model", provider, newProjectMapDisabledConfig())

	if result.Status != "error" {
		t.Errorf("Expected error status, got %s", result.Status)
	}
	if called.Load() != 1 {
		t.Errorf("Cleanup was called %d times on error path, want 1", called.Load())
	}
}

func TestRunHeadless_RepeatedInvocations(t *testing.T) {
	var called atomic.Int32
	cleanupHook = func() { called.Add(1) }
	defer func() { cleanupHook = nil }()

	provider := &mockProvider{name: "test"}
	for i := 0; i < 5; i++ {
		_ = RunHeadlessWithConfig(context.Background(), "hello", "test-model", provider, newProjectMapDisabledConfig())
	}

	if called.Load() != 5 {
		t.Errorf("Cleanup was called %d times for 5 invocations, want 5", called.Load())
	}
}

func TestRunHeadless_NoLeakOnRepeatedInvocations(t *testing.T) {
	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	const iterations = 20
	provider := &mockProvider{name: "mock"}

	runtime.GC()
	baseGoroutines := runtime.NumGoroutine()

	for i := 0; i < iterations; i++ {
		res := RunHeadlessWithConfig(context.Background(), "test query", "mock-model", provider, newProjectMapDisabledConfig())
		if res.Status != "success" {
			t.Fatalf("iteration %d: RunHeadless failed: %v", i, res.Error)
		}
	}

	runtime.GC()
	finalGoroutines := runtime.NumGoroutine()

	if cleanupCount.Load() != int32(iterations) {
		t.Fatalf("Cleanup call count mismatch: got %d, want %d", cleanupCount.Load(), iterations)
	}

	if leaked := finalGoroutines - baseGoroutines; leaked > 5 {
		t.Errorf("possible goroutine leak: base=%d, final=%d, leaked=%d", baseGoroutines, finalGoroutines, leaked)
	}
}
