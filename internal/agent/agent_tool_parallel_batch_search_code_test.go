package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteToolCallsWithParallel_SearchCodeBatchPreservesPatternObservation(t *testing.T) {
	dir := t.TempDir()
	alphaPath := filepath.Join(dir, "alpha.go")
	betaPath := filepath.Join(dir, "beta.go")
	if err := os.WriteFile(alphaPath, []byte("package main\n\nfunc alpha_token() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(betaPath, []byte("package main\n\nfunc beta_token() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	agent := NewAgent("test-model", &mockProvider{name: "test"}, false)
	agent.registry().ClearExcludedTools()
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}
	toolCalls := []*tools.ToolCall{
		{
			ID:      "alpha",
			Tool:    "search_code",
			Args:    map[string]string{"pattern": "alpha_token", "path": dir, "mode": "literal"},
			RawArgs: map[string]any{"pattern": "alpha_token", "path": dir, "mode": "literal"},
		},
		{
			ID:      "beta",
			Tool:    "search_code",
			Args:    map[string]string{"pattern": "beta_token", "path": dir, "mode": "literal"},
			RawArgs: map[string]any{"pattern": "beta_token", "path": dir, "mode": "literal"},
		},
	}

	observed := make(map[string]*tools.RuntimeObservation)
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		observed[tc.Args["pattern"]] = result.Observation
	}
	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	assertObservationHasResolvedPath(t, observed["alpha_token"], alphaPath)
	assertObservationHasResolvedPath(t, observed["beta_token"], betaPath)
}

func assertObservationHasResolvedPath(t *testing.T, observation *tools.RuntimeObservation, want string) {
	t.Helper()
	if observation == nil {
		t.Fatalf("Observation = nil, want path %q", want)
	}
	for _, file := range observation.TouchedFiles {
		if file.ResolvedPath == want {
			return
		}
	}
	t.Fatalf("TouchedFiles = %#v, want resolved path %q", observation.TouchedFiles, want)
}
