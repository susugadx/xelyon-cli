package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestStructMapEditorProviderModels_AllowsClaudeSiblingWhenAnthropicExists(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
	})

	var out bytes.Buffer
	runtime := NewRuntime(strings.NewReader("a\nclaude\nclaude-custom\ns\n"), &out, &out)
	editor := NewStructMapEditorWithRuntime("provider_models", config.FieldTypeStructMap, runtime)

	changed, err := editor.Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}

	providerModels := cfg.ProviderModelsForEdit()
	if _, ok := providerModels["anthropic"]; !ok {
		t.Fatal("provider_models.anthropic should remain configured")
	}
	if got := providerModels["claude"].DefaultModel; got != "claude-custom" {
		t.Fatalf("provider_models.claude.default_model = %q, want %q", got, "claude-custom")
	}
	if strings.Contains(out.String(), "Provider already configured") {
		t.Fatalf("unexpected duplicate warning: %q", out.String())
	}
}

func TestStructMapEditorProviderModels_AllowsAnthropicSiblingWhenClaudeExists(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"claude": {DefaultModel: "claude-custom"},
	})

	var out bytes.Buffer
	runtime := NewRuntime(strings.NewReader("a\nanthropic\nanthropic-custom\ns\n"), &out, &out)
	editor := NewStructMapEditorWithRuntime("provider_models", config.FieldTypeStructMap, runtime)

	changed, err := editor.Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}

	providerModels := cfg.ProviderModelsForEdit()
	if _, ok := providerModels["claude"]; !ok {
		t.Fatal("provider_models.claude should remain configured")
	}
	if got := providerModels["anthropic"].DefaultModel; got != "anthropic-custom" {
		t.Fatalf("provider_models.anthropic.default_model = %q, want %q", got, "anthropic-custom")
	}
	if strings.Contains(out.String(), "Provider already configured") {
		t.Fatalf("unexpected duplicate warning: %q", out.String())
	}
}
