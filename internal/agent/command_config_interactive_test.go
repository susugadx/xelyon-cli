package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestRunInteractiveConfig_ScalarAndStructMapFlows(t *testing.T) {
	t.Run("scalar update saves value", func(t *testing.T) {
		withConfigCommandHooks(t)

		cfg := newProjectMapDisabledConfig()
		categories := []config.ConfigCategory{{Name: "provider"}}
		field := &config.ConfigField{Path: "default_model", FieldType: config.FieldTypeString}
		menu := &scriptedConfigMenu{
			runResults: []configMenuRunResult{
				{category: &categories[0]},
				{category: nil, err: errors.New("cancel")},
			},
			showResults: []configMenuShowResult{
				{field: field},
				{field: nil, err: errors.New("back")},
			},
			editResults: []configMenuEditResult{
				{value: "gpt-next", changed: true},
			},
		}

		buildConfigRegistryForCommand = func(cfg *config.Config) []config.ConfigCategory {
			return categories
		}
		newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *uiruntime.Runtime) configCommandMenu {
			return menu
		}
		setFieldValueForCommand = func(cfg *config.Config, path string, value interface{}) error {
			cfg.DefaultModel = value.(string)
			return nil
		}

		var saved int
		saveConfigForCommand = func(cfg *config.Config) error {
			saved++
			return nil
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(cfg, &out)
		runInteractiveConfig(agent, cfg)

		if cfg.DefaultModel != "gpt-next" {
			t.Fatalf("DefaultModel = %q, want %q", cfg.DefaultModel, "gpt-next")
		}
		if saved != 1 {
			t.Fatalf("save count = %d, want 1", saved)
		}
		if !strings.Contains(out.String(), "✓ Saved: default_model = gpt-next") {
			t.Fatalf("output = %q, want saved message", out.String())
		}
	})

	t.Run("struct map save error is reported", func(t *testing.T) {
		withConfigCommandHooks(t)

		cfg := newProjectMapDisabledConfig()
		categories := []config.ConfigCategory{{Name: "provider"}}
		field := &config.ConfigField{Path: "provider_models", FieldType: config.FieldTypeStructMap}
		menu := &scriptedConfigMenu{
			runResults: []configMenuRunResult{
				{category: &categories[0]},
				{category: nil, err: errors.New("cancel")},
			},
			showResults: []configMenuShowResult{
				{field: field},
				{field: nil, err: errors.New("back")},
			},
			editResults: []configMenuEditResult{
				{value: nil, changed: true},
			},
		}

		buildConfigRegistryForCommand = func(cfg *config.Config) []config.ConfigCategory {
			return categories
		}
		newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *uiruntime.Runtime) configCommandMenu {
			return menu
		}
		saveConfigForCommand = func(cfg *config.Config) error {
			return errors.New("save failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(cfg, &out)
		runInteractiveConfig(agent, cfg)

		if !strings.Contains(out.String(), "Error saving: save failed") {
			t.Fatalf("output = %q, want struct-map save error", out.String())
		}
	})
}
