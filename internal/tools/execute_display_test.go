package tools

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

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

func TestExecuteUnpublishedWithContext_DoesNotPublishWrapperResult(t *testing.T) {
	color.NoColor = true

	origTool := DefaultRegistry.GetTool("search_code")
	t.Cleanup(func() {
		restoreRegistryTool("search_code", origTool)
	})
	DefaultRegistry.Register(&testDisplayTool{
		name:   "search_code",
		result: "Found 3 match(es) in 2 file(s)",
	})

	var output bytes.Buffer
	callbacks := 0
	tc := &ToolCall{
		Tool: "search_code",
		Args: map[string]string{
			"pattern": "threshold",
			"path":    "internal/agent/",
		},
	}
	execCtx := ExecutionContext{
		Stdout: &output,
		Stderr: &output,
		ToolResultCallback: func(ToolResultInfo) {
			callbacks++
		},
	}
	execResult := ExecuteUnpublishedWithContext(execCtx, tc)

	if execResult.Change != nil {
		t.Fatalf("ExecuteUnpublishedWithContext() returned unexpected change: %+v", execResult.Change)
	}
	if !strings.Contains(execResult.Result, "Found 3 match(es)") {
		t.Fatalf("ExecuteUnpublishedWithContext() result = %q", execResult.Result)
	}
	if callbacks != 0 {
		t.Fatalf("ExecuteUnpublishedWithContext() callbacks = %d, want 0", callbacks)
	}
	if !strings.Contains(output.String(), "INTERNAL STDOUT") {
		t.Fatalf("ExecuteUnpublishedWithContext() should keep internal stdout, got: %q", output.String())
	}
}
