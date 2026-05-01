package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadConfig_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig() returned nil")
	}
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider = %v, want deepseek", cfg.DefaultProvider)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("LoadConfig() did not create config file")
	}
}

func TestLoadConfig_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	customConfig := &Config{
		DefaultProvider: "openai",
		DefaultModel:    "gpt-4",
		ProviderModels: map[string]ProviderModelConfig{
			"openai": {DefaultModel: "gpt-4"},
		},
		LoopDetection: LoopDetectionConfig{Threshold: 5},
	}

	data, err := yaml.Marshal(customConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.DefaultProvider != "openai" {
		t.Errorf("DefaultProvider = %v, want openai", cfg.DefaultProvider)
	}
	if cfg.DefaultModel != "gpt-4" {
		t.Errorf("DefaultModel = %v, want gpt-4", cfg.DefaultModel)
	}
	if cfg.LoopDetection.Threshold != 5 {
		t.Errorf("LoopDetection.Threshold = %v, want 5", cfg.LoopDetection.Threshold)
	}
}

func TestLoadConfig_Partial(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	partialYAML := "default_provider: claude\n"

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(partialYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.DefaultProvider != "claude" {
		t.Errorf("DefaultProvider = %v, want claude", cfg.DefaultProvider)
	}
	if cfg.ProviderModels == nil {
		t.Error("ProviderModels should be populated with defaults")
	}
	if cfg.LoopDetection.Threshold != 3 {
		t.Errorf("LoopDetection.Threshold should default to 3, got %d", cfg.LoopDetection.Threshold)
	}
	if cfg.APIRetry.Count != 3 {
		t.Errorf("APIRetry.Count should default to 3, got %d", cfg.APIRetry.Count)
	}
	if cfg.Compression.Enabled != true {
		t.Errorf("Compression.Enabled should default to true, got %v", cfg.Compression.Enabled)
	}
	if cfg.Compression.TriggerPercent != 80 {
		t.Errorf("Compression.TriggerPercent should default to 80, got %d", cfg.Compression.TriggerPercent)
	}
	if cfg.Compression.KeepRecent != 20 {
		t.Errorf("Compression.KeepRecent should default to 20, got %d", cfg.Compression.KeepRecent)
	}
	if cfg.Compression.TokenThreshold != 0 {
		t.Errorf("Compression.TokenThreshold should default to 0, got %d", cfg.Compression.TokenThreshold)
	}
	if cfg.Compression.Model != "" {
		t.Errorf("Compression.Model should default to empty string, got %q", cfg.Compression.Model)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	invalidYAML := "default_provider: openai\n  invalid: - [\n"

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestSaveConfig_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.DefaultProvider = "gemini"

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create config file")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "gemini") {
		t.Error("SaveConfig() did not save custom provider")
	}
	if !strings.Contains(content, "# XELYON CLI 設定") {
		t.Error("SaveConfig() should include header comment")
	}
}

func TestSaveConfig_Permissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0600)
	if mode != expectedMode {
		t.Errorf("SaveConfig() file mode = %v, want %v", mode, expectedMode)
	}
}

func TestSaveConfig_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatal(".xelyon directory should not exist yet")
	}

	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create .xelyon directory")
	}

	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0755)
	if mode != expectedMode {
		t.Errorf("SaveConfig() directory mode = %v, want %v", mode, expectedMode)
	}
}
