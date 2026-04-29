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
