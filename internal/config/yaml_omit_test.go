package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYamlMarshalIncludesAllSections(t *testing.T) {
	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)

	sections := []string{
		"thinking:", "tool_confirm:", "streaming:",
		"web_search:", "diff:", "output:", "general:", "compression:",
		"review:",
		"loop_detection:", "api_retry:", "prompt_cache:",
		"responses:", "paste:", "bash:", "git_stage:",
		"sub_agent:",
		"lsp:", "openai:",
	}

	for _, s := range sections {
		if !strings.Contains(out, s) {
			t.Errorf("section %q is missing from yaml output", s)
		}
	}
}

func TestApplyDefaults_MaxOutputTokensMerge(t *testing.T) {
	// YAML に max_output_tokens がない場合でも、applyDefaults でデフォルト値が適用されること
	yamlData := `
provider_models:
  claude:
    default_model: claude-sonnet-4-5-20250514
  bedrock:
    default_model: global.anthropic.claude-opus-4-5-20251101-v1:0
  gemini:
    default_model: gemini-3.1-pro-preview
`
	cfg := DefaultConfig()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatal(err)
	}

	// Unmarshal 直後は map が上書きされ、MaxOutputTokens が 0 になっている
	if cfg.ProviderModels["claude"].MaxOutputTokens != 0 {
		t.Error("before applyDefaults, MaxOutputTokens should be 0")
	}

	applyDefaults(cfg)

	// applyDefaults 後はデフォルト値が復元されている
	defaults := DefaultConfig()
	tests := []struct {
		provider string
		expected int
	}{
		{"claude", defaults.ProviderModels["claude"].MaxOutputTokens},
		{"bedrock", defaults.ProviderModels["bedrock"].MaxOutputTokens},
		{"gemini", defaults.ProviderModels["gemini"].MaxOutputTokens},
	}
	for _, tt := range tests {
		got := cfg.ProviderModels[tt.provider].MaxOutputTokens
		if got != tt.expected {
			t.Errorf("ProviderModels[%q].MaxOutputTokens = %d, want %d", tt.provider, got, tt.expected)
		}
	}
}

func TestApplyDefaults_MaxOutputTokensPreserved(t *testing.T) {
	// YAML で明示的に max_output_tokens を設定した場合は上書きされないこと
	yamlData := `
provider_models:
  claude:
    default_model: claude-sonnet-4-5-20250514
    max_output_tokens: 99999
`
	cfg := DefaultConfig()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatal(err)
	}

	applyDefaults(cfg)

	if cfg.ProviderModels["claude"].MaxOutputTokens != 99999 {
		t.Errorf("explicitly set MaxOutputTokens should be preserved, got %d", cfg.ProviderModels["claude"].MaxOutputTokens)
	}
}

func TestYamlRoundTripPreservesFalse(t *testing.T) {
	// false に設定して Save → Load しても false のままであること
	cfg := DefaultConfig()
	cfg.Thinking.Enabled = false

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.Thinking.Enabled {
		t.Error("Thinking.Enabled should be false after round-trip")
	}
}
