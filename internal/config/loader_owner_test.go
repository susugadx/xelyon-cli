package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeLoaderOwnerConfigFile(t *testing.T, home, content string) {
	t.Helper()

	configDirPath := filepath.Join(home, configDir)
	if err := os.MkdirAll(configDirPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	configPath := filepath.Join(configDirPath, configFile)
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}

func TestLoadConfigLoader_PipelineOrderAndDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	writeLoaderOwnerConfigFile(t, tmpDir, `
general:
  ui_language: ja
  language: en
tool_confirm:
  auto_approve_medium: true
compression:
  auto_compress: false
  trigger_percent: 70
  threshold_percent: 60
lsp:
  enabled: false
  servers: {}
`)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.General.UILanguage != "ja" {
		t.Fatalf("General.UILanguage = %q, want %q (new key should win)", cfg.General.UILanguage, "ja")
	}
	if cfg.Execution.Mode != string(ExecutionTrusted) {
		t.Fatalf("Execution.Mode = %q, want %q", cfg.Execution.Mode, string(ExecutionTrusted))
	}
	if cfg.Compression.Enabled {
		t.Fatal("Compression.Enabled = true, want false from legacy auto_compress migration")
	}
	if cfg.Compression.TriggerPercent != 70 {
		t.Fatalf("Compression.TriggerPercent = %d, want %d (new key should win)", cfg.Compression.TriggerPercent, 70)
	}
	if cfg.APIRetry.Count != 3 {
		t.Fatalf("APIRetry.Count = %d, want %d (default should be applied)", cfg.APIRetry.Count, 3)
	}
	if cfg.LSP.Enabled {
		t.Fatal("LSP.Enabled = true, want false")
	}
	if cfg.LSP.Servers == nil {
		t.Fatal("LSP.Servers = nil, want explicit empty map")
	}
	if got := len(cfg.LSP.Servers); got != 0 {
		t.Fatalf("len(LSP.Servers) = %d, want 0", got)
	}
	if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateAbsent)
	}
}

func TestLoadConfigLoader_LSPDefaultsWhenSectionMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	writeLoaderOwnerConfigFile(t, tmpDir, "default_provider: openai\n")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if !cfg.LSP.Enabled {
		t.Fatal("LSP.Enabled = false, want true default")
	}
	if cfg.LSP.Servers == nil {
		t.Fatal("LSP.Servers = nil, want default server map")
	}
	if _, ok := cfg.LSP.Servers["go"]; !ok {
		t.Fatal(`LSP.Servers["go"] is missing, want default entry`)
	}
}

func TestLoadConfigLoader_LSPSectionWithoutServersKeepsSiblingFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	writeLoaderOwnerConfigFile(t, tmpDir, `
lsp:
  enabled: false
  skip_install_prompt: true
`)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.LSP.Enabled {
		t.Fatal("LSP.Enabled = true, want false")
	}
	if !cfg.LSP.SkipInstallPrompt {
		t.Fatal("LSP.SkipInstallPrompt = false, want true")
	}
	if cfg.LSP.Servers == nil {
		t.Fatal("LSP.Servers = nil, want default server map")
	}
	if _, ok := cfg.LSP.Servers["go"]; !ok {
		t.Fatal(`LSP.Servers["go"] is missing, want default entry`)
	}
}

func TestLoaderYAMLKeyHelpers_InvalidYAMLReturnsFalse(t *testing.T) {
	invalidYAML := []byte("lsp: [")

	if yamlHasKey(invalidYAML, "lsp") {
		t.Fatal("yamlHasKey() = true, want false for invalid yaml")
	}
	if yamlHasNestedKey(invalidYAML, "lsp", "servers") {
		t.Fatal("yamlHasNestedKey() = true, want false for invalid yaml")
	}
}
