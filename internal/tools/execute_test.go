package tools

import (
	"bytes"
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
			name: "read_file",
			tc:   &ToolCall{Tool: "read_file", Args: map[string]string{"path": "test.txt"}},
			want: []string{"File: test.txt"},
		},
		{
			name: "write_file",
			tc:   &ToolCall{Tool: "write_file", Args: map[string]string{"path": "test.txt", "content": "hello\nworld"}},
			want: []string{"File: test.txt (2 lines)"},
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
		got := isBashReadOnly(tt.command)
		if got != tt.want {
			t.Errorf("isBashReadOnly(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}

func TestIsWriteToolConsistency(t *testing.T) {
	writeTools := []string{"write_file", "str_replace", "delete_file"}
	for _, tool := range writeTools {
		if !IsWriteTool(tool) {
			t.Errorf("IsWriteTool(%q) = false, want true", tool)
		}
	}

	readTools := []string{"read_file", "list_dir", "web_search"}
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

func TestExecute_UsesUnifiedToolDisplay(t *testing.T) {
	color.NoColor = true

	origTool := DefaultRegistry.GetTool("search_code")
	t.Cleanup(func() {
		restoreRegistryTool("search_code", origTool)
	})
	DefaultRegistry.Register(&testDisplayTool{
		name:   "search_code",
		result: "Found 3 match(es) in 2 file(s)\nstats.go:42: ...\nauto_compress.go:30: ...",
	})

	oldStdout := os.Stdout
	oldColorOutput := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w

	tc := &ToolCall{
		Tool: "search_code",
		Args: map[string]string{
			"pattern": "threshold",
			"path":    "internal/agent/",
		},
	}
	result, change := Execute(tc)

	_ = w.Close()
	os.Stdout = oldStdout
	color.Output = oldColorOutput

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if change != nil {
		t.Fatalf("Execute() returned unexpected change: %+v", change)
	}
	if !strings.Contains(result, "Found 3 match(es) in 2 file(s)") {
		t.Fatalf("Execute() result = %q", result)
	}
	if !strings.Contains(output, "INTERNAL STDOUT") {
		t.Fatalf("Execute() should keep internal stdout in normal mode, got: %q", output)
	}
	if !strings.Contains(output, `🔍 search_code: "threshold" in internal/agent/ → 3 matches, 2 files`) {
		t.Fatalf("Execute() output missing summary line: %q", output)
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
