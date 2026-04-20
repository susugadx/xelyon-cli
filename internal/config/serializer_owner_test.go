package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMarshalConfigYAMLSerializer_PreservesExplicitEmptyProviderModels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
	cfg.refreshEffectiveProviderModels()

	data, err := marshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("marshalConfigYAML() error = %v", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	providerModels := findYAMLMappingValue(doc.Content[0], "provider_models")
	if providerModels == nil {
		t.Fatal("provider_models should exist for explicit empty section")
	}
	if providerModels.Kind != yaml.MappingNode {
		t.Fatalf("provider_models.Kind = %v, want MappingNode", providerModels.Kind)
	}
	if got := len(providerModels.Content); got != 0 {
		t.Fatalf("len(provider_models.Content) = %d, want 0 for explicit empty map", got)
	}
}

func TestMarshalConfigYAMLSerializer_UsesOnlyRawProviderModelEntries(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {
			DefaultModel: "gpt-4.1-mini",
		},
	})
	cfg.refreshEffectiveProviderModels()

	data, err := marshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("marshalConfigYAML() error = %v", err)
	}

	var decoded struct {
		ProviderModels map[string]ProviderModelConfig `yaml:"provider_models"`
	}
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if got := len(decoded.ProviderModels); got != 1 {
		t.Fatalf("len(provider_models) = %d, want 1 raw entry", got)
	}
	openai, ok := decoded.ProviderModels["openai"]
	if !ok {
		t.Fatal(`provider_models["openai"] should exist`)
	}
	if openai.DefaultModel != "gpt-4.1-mini" {
		t.Fatalf("provider_models[openai].default_model = %q, want %q", openai.DefaultModel, "gpt-4.1-mini")
	}
	if _, hasDefaultDeepseek := decoded.ProviderModels["deepseek"]; hasDefaultDeepseek {
		t.Fatal(`provider_models should not contain effective default entry "deepseek"`)
	}
}

func TestSaveConfigSerializer_WritesHeaderAndPatchedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.LSP.Servers = nil
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.refreshEffectiveProviderModels()

	if err := saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, configDir, configFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	content := string(data)

	if !strings.HasPrefix(content, "# XELYON CLI 設定\n") {
		t.Fatalf("saved content should start with header, got:\n%s", content)
	}
	if !strings.Contains(content, "servers: null") {
		t.Fatalf("saved content should preserve nil lsp.servers, got:\n%s", content)
	}
	if strings.Contains(content, "\nprovider_models:") {
		t.Fatalf("saved content should omit provider_models when section is absent, got:\n%s", content)
	}
}
