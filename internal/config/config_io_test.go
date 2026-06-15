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

	agentsPath := filepath.Join(tmpDir, ".xelyon", "AGENTS.md")
	info, err := os.Stat(agentsPath)
	if err != nil {
		t.Fatalf("LoadConfig() did not create AGENTS.md: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("AGENTS.md size = %d, want empty file", info.Size())
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
	if cfg.ProviderHistoryReduction.Mode != ProviderHistoryReductionModeDryRun {
		t.Errorf("ProviderHistoryReduction.Mode should default to dry_run, got %q", cfg.ProviderHistoryReduction.Mode)
	}
	if !cfg.ProviderHistoryReduction.RehydrateContext {
		t.Error("ProviderHistoryReduction.RehydrateContext should default to true")
	}
	if cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode != ProviderHistoryRawOutputArtifactsModeDryRun {
		t.Errorf("ProviderHistoryReduction.RawOutputArtifacts.Mode should default to dry_run, got %q", cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode)
	}
}

func TestLoadConfig_ProviderHistoryReductionExplicitFalse(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configYAML := "provider_history_reduction:\n  mode: apply\n  rehydrate_context: false\n"
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ProviderHistoryReduction.Mode != ProviderHistoryReductionModeApply {
		t.Fatalf("ProviderHistoryReduction.Mode = %q, want apply", cfg.ProviderHistoryReduction.Mode)
	}
	if cfg.ProviderHistoryReduction.RehydrateContext {
		t.Fatal("ProviderHistoryReduction.RehydrateContext = true, want explicit false")
	}
	if cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode != ProviderHistoryRawOutputArtifactsModeDryRun {
		t.Fatalf("RawOutputArtifacts.Mode = %q, want default dry_run", cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode)
	}
}

func TestLoadConfig_ProviderHistoryRawOutputArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	rawOutputRoot := filepath.Join(tmpDir, "rawoutputs")

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configYAML := strings.Join([]string{
		"provider_history_reduction:",
		"  mode: apply",
		"  rehydrate_context: true",
		"  raw_output_artifacts:",
		"    mode: apply",
		"    root: " + rawOutputRoot,
		"    max_artifact_bytes: 1048576",
		"    session_quota_bytes: 2097152",
		"    chunk_bytes: 524288",
		"    active_context_budget_tokens: 2048",
		"    active_context_budget_max_tokens: 4096",
		"    retention: session",
		"",
	}, "\n")
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	raw := cfg.ProviderHistoryReduction.RawOutputArtifacts
	if raw.Mode != ProviderHistoryRawOutputArtifactsModeApply ||
		raw.Root != rawOutputRoot ||
		raw.MaxArtifactBytes != 1048576 ||
		raw.SessionQuotaBytes != 2097152 ||
		raw.ChunkBytes != 524288 ||
		raw.ActiveContextBudgetTokens != 2048 ||
		raw.ActiveContextBudgetMaxTokens != 4096 ||
		raw.Retention != ProviderHistoryRawOutputArtifactsRetentionSession {
		t.Fatalf("RawOutputArtifacts = %#v, want explicit values", raw)
	}
}

func TestLoadConfig_ProviderHistoryRawOutputArtifactsRejectsRelativeRoot(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("provider_history_reduction:\n  raw_output_artifacts:\n    root: relative/rawoutputs\n"), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want invalid raw_output_artifacts.root error")
	}
	want := "provider_history_reduction.raw_output_artifacts.root"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("LoadConfig() error = %q, want containing %q", err.Error(), want)
	}
}

func TestLoadConfig_ProviderHistoryRawOutputArtifactsRejectsInvalidBudget(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("provider_history_reduction:\n  raw_output_artifacts:\n    max_artifact_bytes: 0\n"), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want invalid raw_output_artifacts budget error")
	}
	want := "provider_history_reduction.raw_output_artifacts.max_artifact_bytes"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("LoadConfig() error = %q, want containing %q", err.Error(), want)
	}
}

func TestLoadConfig_ProviderHistoryReductionRejectsStableAuto(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("provider_history_reduction:\n  mode: auto\n"), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want stable auto error")
	}
	want := `invalid provider history reduction mode "auto" (expected: off, dry_run, apply)`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("LoadConfig() error = %q, want containing %q", err.Error(), want)
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

	agentsPath := filepath.Join(tmpDir, ".xelyon", "AGENTS.md")
	info, err := os.Stat(agentsPath)
	if err != nil {
		t.Fatalf("SaveConfig() did not create AGENTS.md: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("AGENTS.md size = %d, want empty file", info.Size())
	}
}

func TestSaveConfig_DoesNotOverwriteGlobalAgentInstructions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	agentsPath := filepath.Join(tmpDir, ".xelyon", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(agentsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	const existing = "# existing global guidance\n"
	if err := os.WriteFile(agentsPath, []byte(existing), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := SaveConfig(DefaultConfig()); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != existing {
		t.Fatalf("AGENTS.md was overwritten:\n%s", string(data))
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
