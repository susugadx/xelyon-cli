package agent

import (
	"context"
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

func TestInitializeProjectInstructionsAppliesFinalChecksAndProviderHistoryReductionTogether(t *testing.T) {
	unsetProviderHistoryReductionEnv(t)
	dir := t.TempDir()
	writeProviderHistoryReductionProjectConfigWithFinalChecks(t, dir, "apply", "make verify")
	agent := newProviderHistoryReductionStartupTestAgent(t, dir)

	if err := initializeProjectInstructions(agent, projectInstructionApplyOptions{injectProjectMap: false}); err != nil {
		t.Fatalf("initializeProjectInstructions() error = %v", err)
	}
	assertProviderHistoryReductionRuntimeMode(t, agent.Runtime, ProviderHistoryReductionApply, true)
	assertRuntimeFinalChecks(t, agent, []string{"make verify"}, 30)
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

func TestInitializeProjectInstructionsKeepsFinalChecksOnInvalidExperimentalProviderHistoryReductionMode(t *testing.T) {
	unsetProviderHistoryReductionEnv(t)
	dir := t.TempDir()
	writeProviderHistoryReductionProjectConfigWithFinalChecks(t, dir, "x", "make verify")
	agent := newProviderHistoryReductionStartupTestAgent(t, dir)
	agent.cfg().FinalChecks = config.FinalChecksConfig{Commands: []string{"existing verify"}, Timeout: 99}

	err := initializeProjectInstructions(agent, projectInstructionApplyOptions{injectProjectMap: false})
	if err == nil {
		t.Fatal("initializeProjectInstructions() error = nil, want invalid mode error")
	}
	assertInvalidProviderHistoryReductionModeError(t, err.Error())
	assertRuntimeFinalChecks(t, agent, []string{"existing verify"}, 99)
}

func TestInitializeProjectInstructionsKeepsFinalChecksOnInvalidProviderHistoryReductionEnv(t *testing.T) {
	t.Setenv(config.ProviderHistoryReductionEnvVar, "x")
	dir := t.TempDir()
	writeProviderHistoryReductionProjectConfigWithFinalChecks(t, dir, "dry_run", "make verify")
	agent := newProviderHistoryReductionStartupTestAgent(t, dir)
	agent.cfg().FinalChecks = config.FinalChecksConfig{Commands: []string{"existing verify"}, Timeout: 99}

	err := initializeProjectInstructions(agent, projectInstructionApplyOptions{injectProjectMap: false})
	if err == nil {
		t.Fatal("initializeProjectInstructions() error = nil, want invalid env error")
	}
	assertInvalidProviderHistoryReductionModeError(t, err.Error())
	assertRuntimeFinalChecks(t, agent, []string{"existing verify"}, 99)
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
