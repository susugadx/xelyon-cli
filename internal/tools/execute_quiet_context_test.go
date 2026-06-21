package tools

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteQuiet_SuppressesInternalStdout(t *testing.T) {
	color.NoColor = true

	origTool := DefaultRegistry.GetTool("quiet_test")
	t.Cleanup(func() {
		restoreRegistryTool("quiet_test", origTool)
	})
	DefaultRegistry.Register(&testQuietTool{
		name:   "quiet_test",
		result: "quiet result",
	})

	oldStdout := os.Stdout
	oldColorOutput := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w

	tc := &ToolCall{
		Tool: "quiet_test",
		Args: map[string]string{},
	}
	result, change := ExecuteQuietWithContext(ExecutionContext{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}, tc)

	_ = w.Close()
	os.Stdout = oldStdout
	color.Output = oldColorOutput

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if change != nil {
		t.Fatalf("ExecuteQuiet() returned unexpected change: %+v", change)
	}
	if result != "quiet result" {
		t.Fatalf("ExecuteQuiet() result = %q, want %q", result, "quiet result")
	}
	if output != "" {
		t.Fatalf("ExecuteQuiet() should not write to stdout, got: %q", output)
	}
}

func TestExecuteQuiet_ParallelOverlapKeepsStdoutSuppressed(t *testing.T) {
	color.NoColor = true

	origTool := DefaultRegistry.GetTool("overlap_quiet_test")
	t.Cleanup(func() {
		restoreRegistryTool("overlap_quiet_test", origTool)
	})
	started := make(chan string, 2)
	release := map[string]chan struct{}{
		"first":  make(chan struct{}),
		"second": make(chan struct{}),
	}
	DefaultRegistry.Register(&testOverlapQuietTool{
		started: started,
		release: release,
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	type execResult struct {
		result string
		change *FileChange
	}

	results := make(chan execResult, 2)
	run := func(id string) {
		result, change := ExecuteQuietWithContext(ExecutionContext{
			Stdout: io.Discard,
			Stderr: io.Discard,
		}, &ToolCall{
			Tool: "overlap_quiet_test",
			Args: map[string]string{"id": id},
		})
		results <- execResult{result: result, change: change}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		run("first")
	}()
	go func() {
		defer wg.Done()
		run("second")
	}()

	startedIDs := map[string]bool{}
	for len(startedIDs) < 2 {
		select {
		case id := <-started:
			startedIDs[id] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for overlapping ExecuteQuiet calls to start")
		}
	}

	close(release["first"])

	select {
	case firstResult := <-results:
		if firstResult.change != nil {
			t.Fatalf("first ExecuteQuiet() returned unexpected change: %+v", firstResult.change)
		}
		if firstResult.result != "result first" {
			t.Fatalf("first ExecuteQuiet() result = %q, want %q", firstResult.result, "result first")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first ExecuteQuiet() result")
	}

	if !common.IsQuietMode() {
		t.Fatal("quiet mode should remain enabled while another ExecuteQuiet() is still running")
	}

	close(release["second"])

	select {
	case secondResult := <-results:
		if secondResult.change != nil {
			t.Fatalf("second ExecuteQuiet() returned unexpected change: %+v", secondResult.change)
		}
		if secondResult.result != "result second" {
			t.Fatalf("second ExecuteQuiet() result = %q, want %q", secondResult.result, "result second")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second ExecuteQuiet() result")
	}

	wg.Wait()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if output != "" {
		t.Fatalf("overlapping ExecuteQuiet() calls should not write to stdout, got: %q", output)
	}
	if common.IsQuietMode() {
		t.Fatal("quiet mode should be disabled after all ExecuteQuiet() calls complete")
	}
}

func TestExecuteWithContext_ConcurrentContextsRemainIsolated(t *testing.T) {
	origTool := DefaultRegistry.GetTool("context_isolation_test")
	t.Cleanup(func() {
		restoreRegistryTool("context_isolation_test", origTool)
	})

	started := make(chan string, 2)
	release := make(chan struct{})
	DefaultRegistry.Register(&testContextIsolationTool{
		started: started,
		release: release,
	})

	type execResult struct {
		result string
		output string
	}

	ctxs := []ExecutionContext{
		{ProviderName: "provider-A", Model: "model-A"},
		{ProviderName: "provider-B", Model: "model-B"},
	}

	resultsCh := make(chan execResult, len(ctxs))
	for _, execCtx := range ctxs {
		execCtx := execCtx
		go func() {
			var buf bytes.Buffer
			execCtx.Stdout = &buf
			execCtx.Stderr = &buf

			result, _ := ExecuteWithContext(execCtx, &ToolCall{
				Tool: "context_isolation_test",
				Args: map[string]string{},
			})
			resultsCh <- execResult{result: result, output: buf.String()}
		}()
	}

	startedProviders := map[string]bool{}
	for len(startedProviders) < len(ctxs) {
		select {
		case provider := <-started:
			startedProviders[provider] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for ExecuteWithContext calls to start")
		}
	}
	close(release)

	got := map[string]string{}
	for i := 0; i < len(ctxs); i++ {
		select {
		case result := <-resultsCh:
			got[result.result] = result.output
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for ExecuteWithContext result")
		}
	}

	for _, execCtx := range ctxs {
		key := execCtx.ProviderName + "/" + execCtx.Model
		output, ok := got[key]
		if !ok {
			t.Fatalf("missing result for %s", key)
		}
		if !strings.Contains(output, "CTX "+key) {
			t.Fatalf("output for %s missing its own context: %q", key, output)
		}
		for _, other := range ctxs {
			otherKey := other.ProviderName + "/" + other.Model
			if otherKey != key && strings.Contains(output, otherKey) {
				t.Fatalf("output for %s leaked other context %s: %q", key, otherKey, output)
			}
		}
	}
}

func TestExecuteWithContext_SkipsExecutionWhenContextCanceled(t *testing.T) {
	origTool := DefaultRegistry.GetTool("cancelled_test")
	t.Cleanup(func() {
		restoreRegistryTool("cancelled_test", origTool)
	})

	tool := &testCancelledTool{}
	DefaultRegistry.Register(tool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, change := ExecuteWithContext(ExecutionContext{
		Context: ctx,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	}, &ToolCall{
		Tool: "cancelled_test",
		Args: map[string]string{},
	})

	if change != nil {
		t.Fatalf("ExecuteWithContext() returned unexpected change: %+v", change)
	}
	if result != "Error: context cancelled" {
		t.Fatalf("ExecuteWithContext() result = %q, want %q", result, "Error: context cancelled")
	}
	if tool.ran {
		t.Fatal("tool should not run after context cancellation")
	}
}
