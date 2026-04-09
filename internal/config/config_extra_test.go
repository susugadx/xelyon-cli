package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCloneConfig(t *testing.T) {
	original := DefaultConfig()
	original.DefaultProvider = "claude"
	original.DefaultModel = "claude-sonnet-4-6"
	original.CommandAliases["x"] = "exit"
	original.SetProviderModelConfig("claude", ProviderModelConfig{DefaultModel: "claude-custom"})

	cloned := CloneConfig(original)

	// クローンが nil でないこと
	if cloned == nil {
		t.Fatal("CloneConfig() returned nil")
	}

	// クローンの値が元と一致すること
	if cloned.DefaultProvider != "claude" {
		t.Errorf("cloned.DefaultProvider = %q, want %q", cloned.DefaultProvider, "claude")
	}
	if cloned.DefaultModel != "claude-sonnet-4-6" {
		t.Errorf("cloned.DefaultModel = %q, want %q", cloned.DefaultModel, "claude-sonnet-4-6")
	}

	// クローンを変更しても元が変わらないこと（ディープコピー検証）
	cloned.DefaultProvider = "openai"
	cloned.DefaultModel = "gpt-5.2"
	cloned.CommandAliases["x"] = "changed"

	if original.DefaultProvider != "claude" {
		t.Errorf("original.DefaultProvider changed to %q after modifying clone", original.DefaultProvider)
	}
	if original.DefaultModel != "claude-sonnet-4-6" {
		t.Errorf("original.DefaultModel changed to %q after modifying clone", original.DefaultModel)
	}
	if original.CommandAliases["x"] != "exit" {
		t.Errorf("original.CommandAliases[\"x\"] changed to %q after modifying clone", original.CommandAliases["x"])
	}

	// ProviderModels のマップもディープコピーされること
	cloned.ProviderModels["deepseek"] = ProviderModelConfig{DefaultModel: "changed-model"}
	if original.ProviderModels["deepseek"].DefaultModel == "changed-model" {
		t.Error("original.ProviderModels was modified after changing clone")
	}
	if got := cloned.ProviderModelsForSave()["claude"].DefaultModel; got != "claude-custom" {
		t.Fatalf("cloned raw provider model = %q, want %q", got, "claude-custom")
	}
}

func TestCloneConfig_Nil(t *testing.T) {
	cloned := CloneConfig(nil)

	if cloned == nil {
		t.Fatal("CloneConfig(nil) returned nil, want DefaultConfig()")
	}

	// nil 入力時はデフォルト設定が返ること
	if cloned.DefaultProvider != "deepseek" {
		t.Errorf("CloneConfig(nil).DefaultProvider = %q, want %q", cloned.DefaultProvider, "deepseek")
	}
	if cloned.DefaultModel != "deepseek-chat" {
		t.Errorf("CloneConfig(nil).DefaultModel = %q, want %q", cloned.DefaultModel, "deepseek-chat")
	}
}

func TestMigrateOldKeys_NewKeyPriority(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		checkFn func(t *testing.T, cfg *Config)
	}{
		{
			name: "旧キーのみ → 新キーへ移行",
			yaml: `
general:
  language: en
`,
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.General.UILanguage != "en" {
					t.Errorf("UILanguage = %q, want %q", cfg.General.UILanguage, "en")
				}
			},
		},
		{
			name: "新キーのみ → そのまま保持",
			yaml: `
general:
  ui_language: ja
`,
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.General.UILanguage != "ja" {
					t.Errorf("UILanguage = %q, want %q", cfg.General.UILanguage, "ja")
				}
			},
		},
		{
			name: "新旧両方 → 新キー優先",
			yaml: `
general:
  ui_language: auto
  language: en
`,
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.General.UILanguage != "auto" {
					t.Errorf("UILanguage = %q, want %q (新キー優先)", cfg.General.UILanguage, "auto")
				}
			},
		},
		{
			name: "compression 旧キーのみ → 移行",
			yaml: `
compression:
  auto_compress: false
  threshold_percent: 60
`,
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Compression.Enabled != false {
					t.Errorf("Enabled = %v, want false", cfg.Compression.Enabled)
				}
				if cfg.Compression.TriggerPercent != 60 {
					t.Errorf("TriggerPercent = %d, want 60", cfg.Compression.TriggerPercent)
				}
			},
		},
		{
			name: "compression 新旧両方 → 新キー優先",
			yaml: `
compression:
  enabled: true
  auto_compress: false
  trigger_percent: 70
  threshold_percent: 50
`,
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Compression.Enabled != true {
					t.Errorf("Enabled = %v, want true (新キー優先)", cfg.Compression.Enabled)
				}
				if cfg.Compression.TriggerPercent != 70 {
					t.Errorf("TriggerPercent = %d, want 70 (新キー優先)", cfg.Compression.TriggerPercent)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if err := yaml.Unmarshal([]byte(tt.yaml), cfg); err != nil {
				t.Fatalf("yaml.Unmarshal failed: %v", err)
			}
			migrateOldKeys([]byte(tt.yaml), cfg)
			tt.checkFn(t, cfg)
		})
	}
}

func TestLoadConfigWithValidation(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		// 有効な設定ファイルを作成
		configDir := filepath.Join(tmpDir, ".xelyon")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		validCfg := DefaultConfig()
		data, err := yaml.Marshal(validCfg)
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, data, 0600); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		cfg, result, err := LoadConfigWithValidation()
		if err != nil {
			t.Fatalf("LoadConfigWithValidation() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("LoadConfigWithValidation() returned nil config")
		}
		if !result.Valid {
			t.Errorf("Expected valid result, got issues: %+v", result.Issues)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".xelyon")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		invalidYAML := "default_provider: openai\n  broken: - [\n"
		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		_, _, err := LoadConfigWithValidation()
		if err == nil {
			t.Error("LoadConfigWithValidation() expected error for invalid YAML, got nil")
		}
	})

	t.Run("config with validation issues", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".xelyon")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		// 不正なプロバイダーを含む設定
		badCfg := DefaultConfig()
		badCfg.DefaultProvider = "invalid_provider"
		data, err := yaml.Marshal(badCfg)
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, data, 0600); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		cfg, result, err := LoadConfigWithValidation()
		if err != nil {
			t.Fatalf("LoadConfigWithValidation() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("LoadConfigWithValidation() returned nil config even with validation issues")
		}
		if result.Valid {
			t.Error("Expected invalid result for bad provider")
		}
		if len(result.Issues) == 0 {
			t.Error("Expected validation issues for bad provider")
		}
	})
}
