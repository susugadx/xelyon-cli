package agent

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestSaveAndSyncProjectConfigRefreshesProjectPromptWhenProjectMapDisabled(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	projectDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	oldBlock := prompt.BuildProjectConfigBlock([]string{"old rule"}, []string{"old context"})
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: prompt.InjectProjectConfigBlock("base prompt", oldBlock),
		CurrentModel: "deepseek-chat",
	}

	pc := &config.ProjectConfig{
		Context:  "new context",
		Rules:    []string{"new rule"},
		FilePath: filepath.Join(projectDir, "xelyon.yaml"),
	}
	if err := agent.SaveAndSyncProjectConfig(pc); err != nil {
		t.Fatalf("SaveAndSyncProjectConfig() error = %v", err)
	}

	if !strings.Contains(agent.SystemPrompt, "new context") || !strings.Contains(agent.SystemPrompt, "new rule") {
		t.Fatalf("SystemPrompt did not refresh project config:\n%s", agent.SystemPrompt)
	}
	if strings.Contains(agent.SystemPrompt, "old context") || strings.Contains(agent.SystemPrompt, "old rule") {
		t.Fatalf("SystemPrompt kept stale project config:\n%s", agent.SystemPrompt)
	}
}

func TestSaveAndSyncProjectConfigRefreshesProjectMapIgnorePatterns(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cfg := config.DefaultConfig()
	cfg.ProjectMap.Enabled = true
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	projectDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "ignored"), 0o755); err != nil {
		t.Fatalf("MkdirAll(ignored) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "ignored", "skip.go"), []byte("package ignored\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(skip.go) error = %v", err)
	}

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "deepseek-chat",
	}
	agent.refreshProjectPrompt("")
	if agent.projectMap == nil {
		t.Fatal("expected initial project map")
	}
	if !testProjectMapHasFile(agent, "ignored/skip.go") {
		t.Fatal("test setup expected ignored/skip.go in initial project map")
	}

	pc := &config.ProjectConfig{
		Context:  "ctx",
		Ignore:   config.ProjectIgnoreConfig{Patterns: []string{"ignored/**"}},
		FilePath: filepath.Join(projectDir, "xelyon.yaml"),
	}
	if err := agent.SaveAndSyncProjectConfig(pc); err != nil {
		t.Fatalf("SaveAndSyncProjectConfig() error = %v", err)
	}

	if !strings.Contains(agent.projectMapIgnoreKey, "ignored/**") {
		t.Fatalf("projectMapIgnoreKey = %q, want saved ignore pattern", agent.projectMapIgnoreKey)
	}
	if testProjectMapHasFile(agent, "ignored/skip.go") {
		t.Fatal("project map should refresh with saved project ignore patterns")
	}
	if !strings.Contains(agent.SystemPrompt, "ctx") {
		t.Fatalf("SystemPrompt did not refresh project context:\n%s", agent.SystemPrompt)
	}
}

func TestSaveAndSyncProjectConfigKeepsFinalChecksOnInvalidProviderHistoryReductionEnv(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv(config.ProviderHistoryReductionEnvVar, "x")
	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	projectDir := t.TempDir()
	agent := newProjectConfigSyncTestAgent(t, cfg)
	agent.cfg().FinalChecks = config.FinalChecksConfig{
		Commands: []string{"existing verify"},
		Timeout:  99,
	}

	pc := &config.ProjectConfig{
		Context: "new context",
		FinalChecks: &config.FinalChecksConfig{
			Commands: []string{"project verify"},
			Timeout:  30,
		},
		FilePath: filepath.Join(projectDir, "xelyon.yaml"),
	}
	err := agent.SaveAndSyncProjectConfig(pc)
	if err == nil {
		t.Fatal("SaveAndSyncProjectConfig() error = nil, want invalid provider history reduction env error")
	}
	assertInvalidProviderHistoryReductionModeError(t, err.Error())
	assertRuntimeFinalChecks(t, agent, []string{"existing verify"}, 99)
	if strings.Contains(agent.SystemPrompt, "new context") {
		t.Fatalf("SystemPrompt refreshed after sync error:\n%s", agent.SystemPrompt)
	}
}

func TestSaveAndSyncProjectConfigKeepsRuntimeOnInvalidProviderHistoryRehydrateContextEnv(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv(config.ProviderHistoryRehydrateContextEnvVar, "maybe")
	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	projectDir := t.TempDir()
	agent := newProjectConfigSyncTestAgent(t, cfg)
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionApply
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = true
	agent.cfg().FinalChecks = config.FinalChecksConfig{
		Commands: []string{"existing verify"},
		Timeout:  99,
	}

	pc := &config.ProjectConfig{
		Context: "new context",
		Experimental: config.ProjectExperimentalConfig{
			ProviderHistoryReduction: config.ProjectProviderHistoryReductionConfig{
				Mode:             config.ProjectProviderHistoryReductionModeDryRun,
				RehydrateContext: false,
			},
		},
		FinalChecks: &config.FinalChecksConfig{
			Commands: []string{"project verify"},
			Timeout:  30,
		},
		FilePath: filepath.Join(projectDir, "xelyon.yaml"),
	}
	err := agent.SaveAndSyncProjectConfig(pc)
	if err == nil {
		t.Fatal("SaveAndSyncProjectConfig() error = nil, want invalid provider history rehydrate_context env error")
	}
	want := `invalid provider history rehydrate_context "maybe" (expected: 1, true, 0, false)`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
	assertProviderHistoryReductionRuntimeMode(t, agent.Runtime, ProviderHistoryReductionApply, true)
	if !agent.Runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext changed to false after sync error")
	}
	assertRuntimeFinalChecks(t, agent, []string{"existing verify"}, 99)
	if strings.Contains(agent.SystemPrompt, "new context") {
		t.Fatalf("SystemPrompt refreshed after sync error:\n%s", agent.SystemPrompt)
	}
}

func TestSaveAndSyncProjectConfigKeepsProviderHistoryModeWhenFinalChecksFallbackFails(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	configDir := filepath.Join(tmpHome, ".xelyon")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("final_checks: ["), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}

	projectDir := t.TempDir()
	cfg := newProjectMapDisabledConfig()
	agent := newProjectConfigSyncTestAgent(t, cfg)
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionApply
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = false
	agent.cfg().FinalChecks = config.FinalChecksConfig{
		Commands: []string{"existing verify"},
		Timeout:  99,
	}

	pc := &config.ProjectConfig{
		Context: "new context",
		Experimental: config.ProjectExperimentalConfig{
			ProviderHistoryReduction: config.ProjectProviderHistoryReductionConfig{
				Mode:             config.ProjectProviderHistoryReductionModeDryRun,
				RehydrateContext: true,
			},
		},
		FilePath: filepath.Join(projectDir, "xelyon.yaml"),
	}
	err := agent.SaveAndSyncProjectConfig(pc)
	if err == nil {
		t.Fatal("SaveAndSyncProjectConfig() error = nil, want invalid global config error")
	}
	assertProviderHistoryReductionRuntimeMode(t, agent.Runtime, ProviderHistoryReductionApply, true)
	if agent.Runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext changed to true after sync error")
	}
	assertRuntimeFinalChecks(t, agent, []string{"existing verify"}, 99)
	if strings.Contains(agent.SystemPrompt, "new context") {
		t.Fatalf("SystemPrompt refreshed after sync error:\n%s", agent.SystemPrompt)
	}
}

func newProjectConfigSyncTestAgent(t *testing.T, cfg *config.Config) *Agent {
	t.Helper()
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	return &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "deepseek-chat",
	}
}
