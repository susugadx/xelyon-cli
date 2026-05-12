package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteToolCallsWithParallel_ReadFileBatchPreservesObservation(t *testing.T) {
	dir := t.TempDir()
	alphaPath := filepath.Join(dir, "alpha.go")
	betaPath := filepath.Join(dir, "beta.go")
	if err := os.WriteFile(alphaPath, []byte("package main\n\nconst alpha = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(betaPath, []byte("package main\n\nconst beta = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	agent := NewAgent("test-model", &mockProvider{name: "test"}, false)
	agent.registry().ClearExcludedTools()
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}
	toolCalls := []*tools.ToolCall{
		{
			ID:      "alpha",
			Tool:    "read_file",
			Args:    map[string]string{"path": "alpha.go"},
			RawArgs: map[string]any{"path": "alpha.go"},
		},
		{
			ID:      "beta",
			Tool:    "read_file",
			Args:    map[string]string{"path": "beta.go"},
			RawArgs: map[string]any{"path": "beta.go"},
		},
	}

	observed := make(map[string]*tools.RuntimeObservation)
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		observed[tc.ID] = result.Observation
	}
	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	assertObservationHasResolvedPath(t, observed["alpha"], alphaPath)
	assertObservationHasResolvedPath(t, observed["beta"], betaPath)
}
