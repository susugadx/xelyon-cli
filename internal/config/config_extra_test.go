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
