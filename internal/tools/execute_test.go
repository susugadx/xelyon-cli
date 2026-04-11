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

func TestPreviewToolCall(t *testing.T) {
	tests := []struct {
		name string
		tc   *ToolCall
		want []string
	}{
		{
			name: "gather_context",
			tc:   &ToolCall{Tool: "gather_context", Args: map[string]string{"query": "Agent", "path": "internal/agent", "file_filter": "go"}},
			want: []string{"Query: Agent", "Path: internal/agent", "File filter: go"},
		},
		{
			name: "read_file",
			tc:   &ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["test.txt"]`}},
			want: []string{"File: test.txt"},
		},
		{
			name: "write_file",
			tc:   &ToolCall{Tool: "write_file", Args: map[string]string{"path": "test.txt", "content": "hello\nworld"}},
			want: []string{"File: test.txt (2 lines)"},
		},
		{
			name: "apply_patch",
			tc:   &ToolCall{Tool: "apply_patch", Args: map[string]string{"patch": "*** Begin Patch\n*** Add File: test.txt\n+hello\n*** End Patch"}},
			want: []string{"Patch: 4 lines"},
		},
		{
			name: "bash",
			tc:   &ToolCall{Tool: "bash", Args: map[string]string{"command": "ls -la"}},
			want: []string{"Command: ls -la"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color.NoColor = true
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PreviewToolCall(tt.tc)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			for _, w := range tt.want {
				if !strings.Contains(output, w) {
					t.Errorf("PreviewToolCall() output missing %q, got: %q", w, output)
				}
			}
		})
	}
}

func TestIsBashReadOnly(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		// read-only コマンド
		{"ls -la", true},
		{"cat main.go", true},
		{"grep -r TODO .", true},
		{"git status", true},
		{"git log --oneline -5", true},
		{"git diff HEAD~1", true},
		{"go test ./...", true},
		{"pwd", true},
		{"echo hello", true},
		{"find . -name '*.go'", true},
		{"head -20 main.go", true},
		{"tree", true},
		{"env", true},

		// 空コマンド
		{"", true},
		{"  ", true},

		// unsafe コマンド（パイプ）
		{"cat main.go | grep TODO", false},
		{"ls -la | wc -l", false},

		// unsafe コマンド（リダイレクト）
		{"echo hello > file.txt", false},
		{"cat a >> b", false},

		// unsafe コマンド（連結）
		{"go fmt ./... && go build", false},
		{"ls; rm foo", false},

		// unsafe コマンド（不明なコマンド）
		{"make build", false},
		{"go build -o xelyon", false},
		{"rm -rf dist", false},
		{"npm install", false},
		{"go mod tidy", false},

		// プレフィックス誤マッチ防止（lsxyz は ls ではない）
		{"lsxyz", false},
		{"grepall foo", false},
	}
	for _, tt := range tests {
		got := IsReadOnlyBashCommand(tt.command)
		if got != tt.want {
			t.Errorf("IsReadOnlyBashCommand(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}

func TestIsWriteToolConsistency(t *testing.T) {
	writeTools := []string{"apply_patch", "write_file", "str_replace", "delete_file"}
	for _, tool := range writeTools {
		if !IsWriteTool(tool) {
			t.Errorf("IsWriteTool(%q) = false, want true", tool)
		}
	}

	readTools := []string{"gather_context", "read_file", "list_dir", "web_search"}
	for _, tool := range readTools {
		if IsWriteTool(tool) {
			t.Errorf("IsWriteTool(%q) = true, want false", tool)
		}
	}
}

type testDisplayTool struct {
	name   string
	result string
}

func restoreRegistryTool(name string, orig Tool) {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	if orig != nil {
		DefaultRegistry.tools[name] = orig
		return
	}
	delete(DefaultRegistry.tools, name)
}

func (t *testDisplayTool) Name() string {
	return t.name
}

func (t *testDisplayTool) Description() string {
	return "test tool"
}

func (t *testDisplayTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testDisplayTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	execCtx.Output().Println("INTERNAL STDOUT")
	return t.result, nil, nil
}

func TestExecuteWithContext_UsesUnifiedToolDisplay(t *testing.T) {
	color.NoColor = true

	origTool := DefaultRegistry.GetTool("search_code")
	t.Cleanup(func() {
		restoreRegistryTool("search_code", origTool)
	})
	DefaultRegistry.Register(&testDisplayTool{
		name:   "search_code",
		result: "Found 3 match(es) in 2 file(s)\nstats.go:42: ...\nauto_compress.go:30: ...",
	})

	var output bytes.Buffer

	tc := &ToolCall{
		Tool: "search_code",
		Args: map[string]string{
			"pattern": "threshold",
			"path":    "internal/agent/",
		},
	}
	result, change := ExecuteWithContext(ExecutionContext{
		Stdout: &output,
		Stderr: &output,
	}, tc)

	if change != nil {
		t.Fatalf("ExecuteWithContext() returned unexpected change: %+v", change)
	}
	if !strings.Contains(result, "Found 3 match(es) in 2 file(s)") {
		t.Fatalf("ExecuteWithContext() result = %q", result)
	}
	if !strings.Contains(output.String(), "INTERNAL STDOUT") {
		t.Fatalf("ExecuteWithContext() should keep internal stdout in normal mode, got: %q", output.String())
	}
	if !strings.Contains(output.String(), `🔍 search_code: "threshold" in internal/agent/ → 3 matches, 2 files`) {
		t.Fatalf("ExecuteWithContext() output missing summary line: %q", output.String())
	}
}

type testQuietTool struct {
	name   string
	result string
}

func (t *testQuietTool) Name() string {
	return t.name
}

func (t *testQuietTool) Description() string {
	return "quiet test tool"
}

func (t *testQuietTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testQuietTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	out := execCtx.Output()
	out.Green.Printf("QUIET COLOR OUTPUT\n")
	out.Printf("QUIET STDOUT OUTPUT\n")
	return t.result, nil, nil
}

type testOverlapQuietTool struct {
	started chan string
	release map[string]chan struct{}
}

func (t *testOverlapQuietTool) Name() string {
	return "overlap_quiet_test"
}

func (t *testOverlapQuietTool) Description() string {
	return "overlap quiet test tool"
}

func (t *testOverlapQuietTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testOverlapQuietTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	id := args["id"]
	t.started <- id
	<-t.release[id]
	execCtx.Output().Printf("OVERLAP STDOUT %s\n", id)
	return "result " + id, nil, nil
}

type testContextIsolationTool struct {
	started chan string
	release chan struct{}
}

type testCancelledTool struct {
	ran bool
}

func (t *testContextIsolationTool) Name() string {
	return "context_isolation_test"
}

func (t *testContextIsolationTool) Description() string {
	return "context isolation test tool"
}

func (t *testContextIsolationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testContextIsolationTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	t.started <- execCtx.ProviderName
	<-t.release
	value := execCtx.ProviderName + "/" + execCtx.Model
	execCtx.Output().Printf("CTX %s\n", value)
	return value, nil, nil
}

func (t *testCancelledTool) Name() string {
	return "cancelled_test"
}

func (t *testCancelledTool) Description() string {
	return "cancelled test tool"
}

func (t *testCancelledTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testCancelledTool) Run(_ ExecutionContext, _ map[string]string) (string, *FileChange, error) {
	t.ran = true
	return "should not run", nil, nil
}

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
