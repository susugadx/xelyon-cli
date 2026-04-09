package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_FirstRunProviderModelsEditorStartsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := cfg.ProviderModelsForEdit(); len(got) != 0 {
		t.Fatalf("ProviderModelsForEdit() = %#v, want empty on first-run config", got)
	}
}

func TestDeleteProviderModelConfig_LastOverrideRestoresAbsentSection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.refreshEffectiveProviderModels()

	cfg.SetProviderModelConfig("openai", ProviderModelConfig{DefaultModel: "gpt-custom"})
	cfg.DeleteProviderModelConfig("openai")

	if cfg.ProviderModelsForSave() != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after deleting last override from absent section", cfg.ProviderModelsForSave())
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "provider_models:") {
		t.Fatalf("saved config should not contain provider_models after deleting last override, got:\n%s", content)
	}

	reloaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := reloaded.ResolveModelForProvider("openai"); got != want {
		t.Fatalf("ResolveModelForProvider(openai) = %q, want provider default %q", got, want)
	}
}

func TestDeleteProviderModelConfig_LastOverridePreservesExplicitEmptySection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
default_model: deepseek-chat
provider_models: {}
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	cfg.SetProviderModelConfig("openai", ProviderModelConfig{DefaultModel: "gpt-custom"})
	cfg.DeleteProviderModelConfig("openai")

	got := cfg.ProviderModelsForSave()
	if got == nil || len(got) != 0 {
		t.Fatalf("ProviderModelsForSave() = %#v, want explicit empty provider_models map", got)
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider_models: {}") {
		t.Fatalf("saved config should preserve explicit empty provider_models after add/delete, got:\n%s", content)
	}
}

func TestDeleteProviderModelConfig_LastLoadedOverrideRestoresAbsentSection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
default_model: deepseek-chat
provider_models:
  openai:
    default_model: gpt-custom
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	cfg.DeleteProviderModelConfig("openai")

	if got := cfg.ProviderModelsForSave(); got != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after deleting last loaded override", got)
	}

	if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateAbsent)
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	if strings.Contains(string(data), "provider_models:") {
		t.Fatalf("saved config should not contain provider_models after deleting last loaded override, got:\n%s", string(data))
	}
}

func TestProviderModelsStateTransitionMatrix(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Config)
		mutate    func(*Config)
		wantState providerModelSectionState
		verify    func(*testing.T, *Config)
	}{
		{
			name: "absent plus add then delete restores absent",
			setup: func(cfg *Config) {
				cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
				cfg.refreshEffectiveProviderModels()
			},
			mutate: func(cfg *Config) {
				cfg.SetProviderModelConfig("openai", ProviderModelConfig{DefaultModel: "gpt-new"})
				cfg.DeleteProviderModelConfig("openai")
			},
			wantState: providerModelSectionStateAbsent,
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				if got := cfg.ProviderModelsForSave(); got != nil {
					t.Fatalf("ProviderModelsForSave() = %#v, want nil", got)
				}
			},
		},
		{
			name: "explicit empty plus add then delete preserves explicit empty",
			setup: func(cfg *Config) {
				cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
				cfg.refreshEffectiveProviderModels()
			},
			mutate: func(cfg *Config) {
				cfg.SetProviderModelConfig("openai", ProviderModelConfig{DefaultModel: "gpt-new"})
				cfg.DeleteProviderModelConfig("openai")
			},
			wantState: providerModelSectionStateExplicitEmpty,
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				got := cfg.ProviderModelsForSave()
				if got == nil || len(got) != 0 {
					t.Fatalf("ProviderModelsForSave() = %#v, want explicit empty map", got)
				}
			},
		},
		{
			name: "explicit entries delete last entry restores absent",
			setup: func(cfg *Config) {
				cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
					"openai": {DefaultModel: "gpt-old"},
				})
				cfg.refreshEffectiveProviderModels()
			},
			mutate: func(cfg *Config) {
				cfg.DeleteProviderModelConfig("openai")
			},
			wantState: providerModelSectionStateAbsent,
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				if got := cfg.ProviderModelsForSave(); got != nil {
					t.Fatalf("ProviderModelsForSave() = %#v, want nil", got)
				}
			},
		},
		{
			name: "explicit empty add then clear default model preserves explicit empty",
			setup: func(cfg *Config) {
				cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
				cfg.refreshEffectiveProviderModels()
			},
			mutate: func(cfg *Config) {
				cfg.SetProviderModelConfig("openai", ProviderModelConfig{DefaultModel: "gpt-new", MaxOutputTokens: 999})
				cfg.ClearProviderDefaultModelOverride("openai")
			},
			wantState: providerModelSectionStateExplicitEntriesPreserveEmpty,
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				got := cfg.ProviderModelsForSave()
				if got == nil {
					t.Fatal("ProviderModelsForSave() should keep explicit entry with non-default fields")
				}
				if got["openai"].DefaultModel != "" {
					t.Fatalf("ProviderModelsForSave()[openai].DefaultModel = %q, want empty", got["openai"].DefaultModel)
				}
				if got["openai"].MaxOutputTokens != 999 {
					t.Fatalf("ProviderModelsForSave()[openai].MaxOutputTokens = %d, want %d", got["openai"].MaxOutputTokens, 999)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setup(cfg)
			tt.mutate(cfg)

			if got := cfg.providerModelSectionState(); got != tt.wantState {
				t.Fatalf("providerModelSectionState() = %v, want %v", got, tt.wantState)
			}
			tt.verify(t, cfg)
		})
	}
}
