package tuiagent

import (
	"testing"

	agentpkg "github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestTUIAdapter_WrapperStateAccessors(t *testing.T) {
	disableColors(t)

	agent := &agentpkg.Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "claude-sonnet-4-6",
		Stats:             &agentpkg.SessionStats{},
		Runtime:           agentpkg.NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
	}

	adapter := NewTUIAdapter(agent, nil)

	if got, want := adapter.GetStatusLine(), agent.FormatStatusLine(); got != want {
		t.Fatalf("GetStatusLine() = %q, want %q", got, want)
	}
	if adapter.IsProcessing() {
		t.Fatal("IsProcessing() = true, want false")
	}
	adapter.processing.Store(true)
	if !adapter.IsProcessing() {
		t.Fatal("IsProcessing() = false, want true")
	}
	if got := adapter.GetProviderName(); got != "claude" {
		t.Fatalf("GetProviderName() = %q, want %q", got, "claude")
	}
	if got := adapter.GetProviderConfigKey(); got != "anthropic" {
		t.Fatalf("GetProviderConfigKey() = %q, want %q", got, "anthropic")
	}
}

func TestTUIAdapter_StatusSnapshotLabelsPlanModeState(t *testing.T) {
	disableColors(t)

	agent := &agentpkg.Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-5.4",
		Stats:           &agentpkg.SessionStats{},
		Runtime:         agentpkg.NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
		PlanModeEnabled: false,
	}
	adapter := NewTUIAdapter(agent, nil)

	if got := adapter.StatusSnapshot().Mode; got != "Plan: OFF" {
		t.Fatalf("StatusSnapshot().Mode = %q, want Plan: OFF", got)
	}

	agent.PlanModeEnabled = true
	if got := adapter.StatusSnapshot().Mode; got != "Plan: ON" {
		t.Fatalf("StatusSnapshot().Mode = %q, want Plan: ON", got)
	}
}

func TestTUIAdapter_CancelAndCleanup(t *testing.T) {
	agent := &agentpkg.Agent{}
	adapter := NewTUIAdapter(agent, nil)

	adapter.Cancel()
	adapter.Cleanup()
}

func TestTUIAdapter_LoadConfigForEditReturnsClone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultModel = "gpt-before"
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	adapter := NewTUIAdapter(&agentpkg.Agent{}, nil)
	loaded, err := adapter.LoadConfigForEdit()
	if err != nil {
		t.Fatalf("LoadConfigForEdit() error = %v", err)
	}

	loaded.DefaultModel = "gpt-edited"
	reloaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if reloaded.DefaultModel != "gpt-before" {
		t.Fatalf("LoadConfigForEdit() should return clone, file model = %q", reloaded.DefaultModel)
	}
}

func TestTUIAdapter_SaveAndSyncConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-old"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-old"})

	runtime := agentpkg.NewAgentRuntimeWithConfig(cfg)
	agent := agentpkg.NewAgentWithRuntime("gpt-old", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	adapter := NewTUIAdapter(agent, nil)
	next := config.CloneConfig(cfg)
	next.DefaultModel = "gpt-new"
	next.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-new"})

	if err := adapter.SaveAndSyncConfig(next); err != nil {
		t.Fatalf("SaveAndSyncConfig() error = %v", err)
	}

	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.DefaultModel != "gpt-new" {
		t.Fatalf("saved DefaultModel = %q, want %q", loaded.DefaultModel, "gpt-new")
	}
	if agent.Runtime.Config.DefaultModel != "gpt-new" {
		t.Fatalf("runtime DefaultModel = %q, want %q", agent.Runtime.Config.DefaultModel, "gpt-new")
	}
	if agent.CurrentModel != "gpt-new" {
		t.Fatalf("agent.CurrentModel = %q, want %q", agent.CurrentModel, "gpt-new")
	}
}
