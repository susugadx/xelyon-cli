package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestToInt(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int
	}{
		{name: "int", value: int(3), want: 3},
		{name: "int64", value: int64(4), want: 4},
		{name: "float64", value: float64(5), want: 5},
		{name: "unsupported", value: "6", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toInt(tt.value); got != tt.want {
				t.Fatalf("toInt(%T(%v)) = %d, want %d", tt.value, tt.value, got, tt.want)
			}
		})
	}
}

func TestYAMLNodeHelpers(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(`
general:
  ui_language: ja
provider_models:
  openai:
    default_model: gpt-5.4
`), &doc); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	parent := findYAMLMappingValue(doc.Content[0], "general")
	if parent == nil {
		t.Fatal("findYAMLMappingValue() should return general mapping")
	}
	child := findYAMLMappingValue(parent, "ui_language")
	if child == nil || child.Value != "ja" {
		t.Fatalf("child value = %#v, want ja", child)
	}

	if !setNestedYAMLValueToNull(&doc, "general", "ui_language") {
		t.Fatal("setNestedYAMLValueToNull() should update existing nested key")
	}
	child = findYAMLMappingValue(parent, "ui_language")
	if child == nil || child.Tag != "!!null" || child.Value != "null" {
		t.Fatalf("nested child after nulling = %#v, want null scalar", child)
	}

	if !removeTopLevelYAMLKey(&doc, "provider_models") {
		t.Fatal("removeTopLevelYAMLKey() should remove provider_models")
	}
	if findYAMLMappingValue(doc.Content[0], "provider_models") != nil {
		t.Fatal("provider_models should be removed")
	}
}

func TestMarshalConfigYAML_NilLSPServersAndProviderModelsRemoval(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LSP.Servers = nil
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateAbsent, nil)
	cfg.ProviderModels = nil

	data, err := marshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("marshalConfigYAML() error = %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "servers: null") {
		t.Fatalf("marshalConfigYAML() should preserve nil lsp.servers, got:\n%s", content)
	}
	if strings.Contains(content, "\nprovider_models:") {
		t.Fatalf("marshalConfigYAML() should omit provider_models when absent, got:\n%s", content)
	}
}
