package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newTUIAdapterTestAgent(t *testing.T) (*Agent, *bytes.Buffer) {
	t.Helper()

	cfg := newProjectMapDisabledConfig()
	out := &bytes.Buffer{}
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), out, out)

	agent := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	return agent, out
}

func TestTUIConfig_UnknownSubcommand_DoesNotRunInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/config foo") {
		t.Fatal("HandleCommand(/config foo) = false, want true")
	}

	got := out.String()
	if !strings.Contains(got, "/config foo is not available in TUI mode") {
		t.Fatalf("output = %q, want unsupported TUI /config message", got)
	}
	if strings.Contains(got, "Configuration Menu") {
		t.Fatalf("output = %q, should not contain interactive config menu", got)
	}
}

func TestTUIConfig_Show_StillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/config show") {
		t.Fatal("HandleCommand(/config show) = false, want true")
	}

	got := out.String()
	if !strings.Contains(got, "default_provider                    = openai") {
		t.Fatalf("output = %q, want config show output", got)
	}
}

func TestTUIConfig_Model_StillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-old"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-old"})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	agent, out := newTUIAdapterTestAgent(t)
	adapter := NewTUIAdapter(agent, nil)

	if !adapter.HandleCommand("/config model gpt-new") {
		t.Fatal("HandleCommand(/config model gpt-new) = false, want true")
	}

	if agent.CurrentModel != "gpt-new" {
		t.Fatalf("agent.CurrentModel = %q, want %q", agent.CurrentModel, "gpt-new")
	}
	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.DefaultModel != "gpt-new" {
		t.Fatalf("loaded.DefaultModel = %q, want %q", loaded.DefaultModel, "gpt-new")
	}
	if loaded.ProviderModels["openai"].DefaultModel != "gpt-new" {
		t.Fatalf("loaded.ProviderModels[openai].DefaultModel = %q, want %q", loaded.ProviderModels["openai"].DefaultModel, "gpt-new")
	}
	if !strings.Contains(out.String(), "Default model updated to: gpt-new") {
		t.Fatalf("output = %q, want success message", out.String())
	}
}
