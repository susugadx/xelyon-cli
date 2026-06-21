package prompt

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetProviderPrefix_DefaultEmptyForProviderFamilies(t *testing.T) {
	tests := []string{
		"gemini",
		"Gemini",
		"deepseek",
		"DeepSeek",
		"groq",
		"claude",
		"anthropic",
		"  anthropic  ",
		"bedrock",
		"openai",
		"OpenAI",
		"azure",
		"Azure OpenAI",
		"openrouter",
		"ollama",
		"unknown-provider",
	}

	for _, provider := range tests {
		t.Run(provider, func(t *testing.T) {
			if got := GetProviderPrefix(provider); got != "" {
				t.Fatalf("GetProviderPrefix(%q) = %q, want empty provider notes", provider, got)
			}
		})
	}
}

func TestBuildProviderSystemPrompt_DefaultEmptyForProviderFamilies(t *testing.T) {
	base := "Header\n## Workflow Rules\nRules"
	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel: "corp-bedrock-sonnet46",
		CatalogModel: "global.anthropic.claude-sonnet-4-6",
	}

	tests := []struct {
		name     string
		provider string
		model    string
	}{
		{name: "gemini", provider: "gemini", model: "gemini-3.1-pro-preview-customtools"},
		{name: "deepseek", provider: "deepseek", model: "deepseek-chat"},
		{name: "groq", provider: "groq", model: "meta-llama/llama-4-scout-17b-16e-instruct"},
		{name: "claude", provider: "claude", model: "claude-sonnet-4-6"},
		{name: "anthropic alias", provider: "anthropic", model: "claude-sonnet-4-6"},
		{name: "openai", provider: "openai", model: "gpt-5.4"},
		{name: "azure", provider: "azure", model: "azure-gpt-5.4"},
		{name: "azure display name", provider: "Azure OpenAI", model: "azure-gpt-5.4"},
		{name: "openrouter", provider: "openrouter", model: "anthropic/claude-opus-4.6"},
		{name: "bedrock claude route", provider: "bedrock", model: "global.anthropic.claude-sonnet-4-6"},
		{name: "bedrock catalog claude alias", provider: "bedrock", model: "corp-bedrock-sonnet46"},
		{name: "bedrock converse route", provider: "bedrock", model: "amazon.nova-pro-v1:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildProviderSystemPromptWithConfig(base, tt.provider, tt.model, cfg)
			if got != base {
				t.Fatalf("BuildProviderSystemPromptWithConfig(%q, %q) should not add provider notes, got:\n%s", tt.provider, tt.model, got)
			}
		})
	}
}

func TestBuildProviderSystemPrompt_EmptyProvider(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPromptWithConfig(base, "", "", config.DefaultConfig())

	if result != base {
		t.Error("empty provider should return unchanged base prompt")
	}
}

func TestGetSystemPromptForProvider_UsesProviderResolvedMode(t *testing.T) {
	claudePrompt := GetSystemPromptForProvider("claude", "claude-sonnet-4-6")
	if !strings.Contains(claudePrompt, "### Legacy edit tools") {
		t.Fatal("claude prompt should include legacy edit tool guidance")
	}
	if strings.Contains(claudePrompt, "### apply_patch (edit tool)") {
		t.Fatal("claude prompt should not include apply_patch guidance")
	}
	if !strings.Contains(claudePrompt, "search_code: low-level exact-search tool") || !strings.Contains(claudePrompt, "read_file: low-level exact-content override") {
		t.Fatal("claude prompt should keep low-level investigation override guidance when those tools are visible")
	}
	if strings.Contains(claudePrompt, "## Provider Notes") {
		t.Fatalf("claude prompt should not include provider notes:\n%s", claudePrompt)
	}

	openAIPrompt := GetSystemPromptForProvider("openai", "gpt-5.4")
	if !strings.Contains(openAIPrompt, "### apply_patch (edit tool)") {
		t.Fatal("openai prompt should include apply_patch guidance")
	}
	if strings.Contains(openAIPrompt, "### Legacy edit tools") {
		t.Fatal("openai prompt should not include legacy edit tool guidance")
	}
	if strings.Contains(openAIPrompt, "search_code: low-level exact-search tool") || strings.Contains(openAIPrompt, "read_file: low-level exact-content override") {
		t.Fatal("openai prompt should not advertise legacy low-level investigation overrides")
	}
	if !strings.Contains(openAIPrompt, "read_file: exact-content override for known files or ranges when edit/apply_patch needs precise context") {
		t.Fatal("openai prompt should keep read_file exact-control guidance when it stays visible")
	}

	azurePrompt := GetSystemPromptForProvider("azure", "azure-gpt-5.4")
	if !strings.Contains(azurePrompt, "### apply_patch (edit tool)") {
		t.Fatal("azure prompt should include apply_patch guidance")
	}
	azureDisplayPrompt := GetSystemPromptForProvider("Azure OpenAI", "azure-gpt-5.4")
	if !strings.Contains(azureDisplayPrompt, "### apply_patch (edit tool)") {
		t.Fatal("Azure OpenAI display name prompt should include apply_patch guidance")
	}
}

func TestBuildProviderSystemPromptWithConfig_StripsMarkedProviderNotesWhenDefaultEmpty(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	stale := "You are XELYON.\n\n" +
		"<!-- PROVIDER_NOTES_START:deepseek -->\n" +
		legacyGeneratedProviderNotesSections[1] + "\n" +
		"<!-- PROVIDER_NOTES_END -->\n\n" +
		"## Workflow Rules\n- workflow"

	got := BuildProviderSystemPromptWithConfig(stale, "gemini", "gemini-3.1-pro-preview-customtools", config.DefaultConfig())
	if got != base {
		t.Fatalf("stale marker-based provider notes should be stripped, got:\n%s", got)
	}
}

func TestBuildProviderSystemPromptWithConfig_StripsLegacyGeneratedProviderNotes(t *testing.T) {
	for _, section := range legacyGeneratedProviderNotesSections {
		t.Run(strings.Split(section, "\n")[1], func(t *testing.T) {
			legacy := "You are XELYON.\n\n" + section + "\n\n## Workflow Rules\n- workflow"
			got := BuildProviderSystemPromptWithConfig(legacy, "openrouter", "openai/gpt-5.4", config.DefaultConfig())

			if strings.Contains(got, "## Provider Notes") {
				t.Fatalf("legacy generated provider notes should be stripped:\n%s", got)
			}
		})
	}
}

func TestBuildProviderSystemPromptWithConfig_PreservesCustomProviderSpecificNotesSection(t *testing.T) {
	custom := "You are XELYON.\n\n## Provider Notes\n### OpenAI-specific\n- Keep this team-specific adapter note\n\n## Workflow Rules\n- workflow"
	got := BuildProviderSystemPromptWithConfig(custom, "openrouter", "openai/gpt-5.4", config.DefaultConfig())

	if !strings.Contains(got, "Keep this team-specific adapter note") {
		t.Fatalf("custom provider-specific notes should be preserved:\n%s", got)
	}
}

func TestBuildProviderSystemPromptWithConfig_PreservesCustomProviderNotesSection(t *testing.T) {
	custom := "You are XELYON.\n\n## Provider Notes\n### Team-specific\n- Keep this custom rule\n\n## Workflow Rules\n- workflow"
	got := BuildProviderSystemPromptWithConfig(custom, "openrouter", "openai/gpt-5.4", config.DefaultConfig())

	if !strings.Contains(got, "### Team-specific") {
		t.Fatalf("custom provider notes should be preserved for empty-prefix provider:\n%s", got)
	}
	if strings.Count(got, "## Provider Notes") != 1 {
		t.Fatalf("custom provider notes count should stay unchanged:\n%s", got)
	}
}
