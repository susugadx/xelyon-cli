package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestHandleConfigCommand_LoadErrorAndModelSaveError(t *testing.T) {
	t.Run("load error is reported", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return nil, errors.New("broken config")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleConfigCommand(agent, []string{"show"}) {
			t.Fatal("handleConfigCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Failed to load config") {
			t.Fatalf("output = %q, want load error", out.String())
		}
	})

	t.Run("model save error is reported", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return newProjectMapDisabledConfig(), nil
		}
		saveConfigForCommand = func(cfg *config.Config) error {
			return errors.New("save failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleConfigCommand(agent, []string{"model", "next-model"}) {
			t.Fatal("handleConfigCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Failed to save config") {
			t.Fatalf("output = %q, want save error", out.String())
		}
	})
}

func TestHandleConfigCommand_DelegatesInteractiveMode(t *testing.T) {
	withConfigCommandHooks(t)

	cfg := newProjectMapDisabledConfig()
	loadConfigForCommand = func() (*config.Config, error) {
		return cfg, nil
	}

	menu := &scriptedConfigMenu{
		runResults: []configMenuRunResult{{category: nil, err: errors.New("cancel")}},
	}
	var factoryCalls int
	newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *uiruntime.Runtime) configCommandMenu {
		factoryCalls++
		return menu
	}

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(cfg, &out)
	if !handleConfigCommand(agent, nil) {
		t.Fatal("handleConfigCommand() = false, want true")
	}
	if factoryCalls != 1 {
		t.Fatalf("config menu factory calls = %d, want 1", factoryCalls)
	}
}

func TestIsNonInteractiveConfigSubcommand(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"show"}, true},
		{[]string{"show", "extra"}, false},
		{[]string{"model"}, false},
		{[]string{"model", "gpt-5"}, true},
		{[]string{"other"}, false},
	}

	for _, tt := range tests {
		if got := isNonInteractiveConfigSubcommand(tt.args); got != tt.want {
			t.Fatalf("isNonInteractiveConfigSubcommand(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestHandleConfigCommand_Model_UpdatesProviderModel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "gpt-original",
	})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-original",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         NewAgentRuntimeWithConfig(cfg),
	}

	handleConfigCommand(a, []string{"model", "gpt-new"})

	if a.CurrentModel != "gpt-new" {
		t.Errorf("Agent.CurrentModel = %s, want gpt-new", a.CurrentModel)
	}

	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.DefaultModel != "gpt-new" {
		t.Errorf("Config.DefaultModel = %s, want gpt-new", loadedCfg.DefaultModel)
	}

	pm, ok := loadedCfg.ProviderModels["openai"]
	if !ok {
		t.Fatal("ProviderModels['openai'] missing")
	}
	if pm.DefaultModel != "gpt-new" {
		t.Errorf("ProviderModels['openai'].DefaultModel = %s, want gpt-new", pm.DefaultModel)
	}
}
