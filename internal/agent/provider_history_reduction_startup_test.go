package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestInitializeProjectInstructionsSyncsExperimentalProviderHistoryReductionMode(t *testing.T) {
	unsetProviderHistoryReductionEnv(t)
	dir := t.TempDir()
	writeProviderHistoryReductionProjectConfig(t, dir, "dry_run")
	agent := newProviderHistoryReductionStartupTestAgent(t, dir)

	if err := initializeProjectInstructions(agent, projectInstructionApplyOptions{injectProjectMap: false}); err != nil {
		t.Fatalf("initializeProjectInstructions() error = %v", err)
	}
	assertProviderHistoryReductionRuntimeMode(t, agent.Runtime, ProviderHistoryReductionDryRun, true)
}

func TestInitializeProjectInstructionsSyncsProviderHistoryReductionEnvWithoutProjectConfig(t *testing.T) {
	t.Setenv(config.ProviderHistoryReductionEnvVar, "apply")
	agent := newProviderHistoryReductionStartupTestAgent(t, t.TempDir())

	if err := initializeProjectInstructions(agent, projectInstructionApplyOptions{injectProjectMap: false}); err != nil {
		t.Fatalf("initializeProjectInstructions() error = %v", err)
	}
	assertProviderHistoryReductionRuntimeMode(t, agent.Runtime, ProviderHistoryReductionApply, true)
}

func TestInitializeProjectInstructionsReturnsInvalidExperimentalProviderHistoryReductionMode(t *testing.T) {
	unsetProviderHistoryReductionEnv(t)
	dir := t.TempDir()
	writeProviderHistoryReductionProjectConfig(t, dir, "x")
	agent := newProviderHistoryReductionStartupTestAgent(t, dir)

	err := initializeProjectInstructions(agent, projectInstructionApplyOptions{injectProjectMap: false})
	if err == nil {
		t.Fatal("initializeProjectInstructions() error = nil, want invalid mode error")
	}
	assertInvalidProviderHistoryReductionModeError(t, err.Error())
}

func TestRunHeadlessWithConfigReturnsConfigErrorForInvalidProviderHistoryReductionEnv(t *testing.T) {
	t.Setenv(config.ProviderHistoryReductionEnvVar, "x")
	t.Setenv("HOME", t.TempDir())

	result := RunHeadlessWithConfig(context.Background(), "query", "test-model", &mockProvider{name: "openai"}, newProjectMapDisabledConfig())
	if result.Status != "error" {
		t.Fatalf("RunHeadlessWithConfig() status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != "config_error" {
		t.Fatalf("RunHeadlessWithConfig() error = %#v, want config_error", result.Error)
	}
	assertInvalidProviderHistoryReductionModeError(t, result.Error.Message)
}

func newProviderHistoryReductionStartupTestAgent(t *testing.T, cwd string) *Agent {
	t.Helper()
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.InvocationCWD = cwd
	agent := NewAgentWithRuntime("test-model", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)
	return agent
}

func writeProviderHistoryReductionProjectConfig(t *testing.T, dir, mode string) {
	t.Helper()
	data := []byte("experimental:\n  provider_history_reduction:\n    mode: " + mode + "\n")
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertProviderHistoryReductionRuntimeMode(t *testing.T, runtime *AgentRuntime, wantMode ProviderHistoryReductionMode, wantSet bool) {
	t.Helper()
	if runtime.Options.ProviderHistoryReductionMode != wantMode || runtime.Options.ProviderHistoryReductionModeSet != wantSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want (%v, %v)", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet, wantMode, wantSet)
	}
}

func assertInvalidProviderHistoryReductionModeError(t *testing.T, got string) {
	t.Helper()
	want := `invalid provider history reduction mode "x" (expected: off, dry_run, apply, auto)`
	if !strings.Contains(got, want) {
		t.Fatalf("error = %q, want containing %q", got, want)
	}
}
