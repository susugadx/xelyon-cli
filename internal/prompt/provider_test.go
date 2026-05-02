package prompt

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetProviderPrefix_Gemini(t *testing.T) {
	prefix := GetProviderPrefix("gemini")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for gemini")
	}
	checks := []string{
		"## Provider Notes",
		"raw JSON",
	}
	for _, check := range checks {
		if !strings.Contains(prefix, check) {
			t.Errorf("gemini prefix missing %q", check)
		}
	}
}

func TestGetProviderPrefix_GeminiCaseInsensitive(t *testing.T) {
	prefix := GetProviderPrefix("Gemini")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for Gemini (uppercase)")
	}
}

func TestGetProviderPrefix_DeepSeek(t *testing.T) {
	prefix := GetProviderPrefix("deepseek")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for deepseek")
	}
	checks := []string{
		"tool calls for file operations",
		"unused imports",
	}
	for _, check := range checks {
		if !strings.Contains(prefix, check) {
			t.Errorf("deepseek prefix missing %q", check)
		}
	}
}

func TestGetProviderPrefix_DeepSeekCaseInsensitive(t *testing.T) {
	prefix := GetProviderPrefix("DeepSeek")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for DeepSeek (uppercase)")
	}
}

func TestGetProviderPrefix_Claude(t *testing.T) {
	prefix := GetProviderPrefix("claude")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for claude")
	}
	if !strings.Contains(prefix, "dedicated tools") {
		t.Error("claude prefix missing dedicated tools rule")
	}
	if !strings.Contains(prefix, "do not prefix commands") {
		t.Error("claude prefix missing cd prefix rule")
	}
}

func TestGetProviderPrefix_Anthropic(t *testing.T) {
	anthropic := GetProviderPrefix("anthropic")
	claude := GetProviderPrefix("claude")
	if anthropic != claude {
		t.Error("anthropic and claude should return identical prefixes")
	}
}

func TestGetProviderPrefix_AnthropicWithSpaces(t *testing.T) {
	anthropic := GetProviderPrefix("  anthropic  ")
	claude := GetProviderPrefix("claude")
	if anthropic != claude {
		t.Error("spaced anthropic and claude should return identical prefixes")
	}
}

func TestGetProviderPrefix_Bedrock(t *testing.T) {
	bedrock := GetProviderPrefix("bedrock")
	if bedrock != "" {
		t.Error("provider-only bedrock prefix should be empty because Bedrock route depends on model family")
	}
}

func TestBuildProviderSystemPrompt_BedrockClaudeModel(t *testing.T) {
	base := "Header\n## Workflow Rules\nRules"
	result := BuildProviderSystemPromptWithConfig(base, "bedrock", "global.anthropic.claude-sonnet-4-6", config.DefaultConfig())
	if !strings.Contains(result, "### Claude-specific") {
		t.Fatal("Bedrock Claude model should receive Claude-specific provider notes")
	}
}

func TestBuildProviderSystemPrompt_BedrockConverseModelReturnsBase(t *testing.T) {
	base := "Header\n## Workflow Rules\nRules"
	result := BuildProviderSystemPromptWithConfig(base, "bedrock", "amazon.nova-pro-v1:0", config.DefaultConfig())
	if result != base {
		t.Fatalf("Bedrock non-Claude model should not receive Claude provider notes, got %q", result)
	}
}

func TestBuildProviderSystemPrompt_BedrockCatalogClaudeAlias(t *testing.T) {
	base := "Header\n## Workflow Rules\nRules"
	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel: "corp-bedrock-sonnet46",
		CatalogModel: "global.anthropic.claude-sonnet-4-6",
	}

	result := BuildProviderSystemPromptWithConfig(base, "bedrock", "corp-bedrock-sonnet46", cfg)
	if !strings.Contains(result, "### Claude-specific") {
		t.Fatal("Bedrock catalog_model Claude alias should receive Claude-specific provider notes")
	}
}

func TestGetProviderPrefix_Unknown(t *testing.T) {
	prefix := GetProviderPrefix("unknown-provider")
	if prefix != "" {
		t.Errorf("expected empty prefix for unknown provider, got: %q", prefix)
	}
}

func TestGetProviderPrefix_OpenAI(t *testing.T) {
	prefix := GetProviderPrefix("openai")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for openai")
	}
	if !strings.Contains(prefix, "byte corruption") {
		t.Error("openai prefix missing byte corruption rule")
	}
	if strings.Contains(prefix, "briefly state what you are about to do") {
		t.Error("openai prefix should not contain per-tool narration rule (moved to common prompt)")
	}
}

func TestGetProviderPrefix_OpenAICaseInsensitive(t *testing.T) {
	prefix := GetProviderPrefix("OpenAI")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for OpenAI (uppercase)")
	}
}

func TestProviderSpecificRules(t *testing.T) {
	gemini := GetProviderPrefix("gemini")
	deepseek := GetProviderPrefix("deepseek")
	openai := GetProviderPrefix("openai")

	if !strings.Contains(gemini, "raw JSON, not markdown code blocks") {
		t.Error("gemini should contain raw JSON rule")
	}
	if strings.Contains(deepseek, "raw JSON, not markdown code blocks") {
		t.Error("deepseek should not contain gemini-specific raw JSON wording")
	}

	if !strings.Contains(deepseek, "tool calls for file operations") {
		t.Error("deepseek should contain tool call rule")
	}
	if strings.Contains(gemini, "tool calls for file operations") {
		t.Error("gemini should not contain deepseek-specific tool call rule")
	}

	if !strings.Contains(openai, "byte corruption") {
		t.Error("openai should contain byte corruption rule")
	}
	if strings.Contains(gemini, "byte corruption") {
		t.Error("gemini should not contain openai-specific byte corruption rule")
	}
	if strings.Contains(deepseek, "byte corruption") {
		t.Error("deepseek should not contain openai-specific byte corruption rule")
	}
}

func TestBuildProviderSystemPrompt_InsertsBeforeWorkflowRules(t *testing.T) {
	base := "You are XELYON.\n\n## Core Identity\n- test\n\n## Workflow Rules\n- workflow"
	result := BuildProviderSystemPromptWithConfig(base, "gemini", "gemini-3.1-pro-preview-customtools", config.DefaultConfig())

	if strings.HasPrefix(result, "## Provider Notes") {
		t.Error("provider notes should not replace the start of the base prompt")
	}

	idxNotes := strings.Index(result, "## Provider Notes")
	idxWorkflow := strings.Index(result, "## Workflow Rules")
	if idxNotes < 0 {
		t.Fatal("provider notes not inserted")
	}
	if idxWorkflow < 0 {
		t.Fatal("workflow rules header not found")
	}
	if idxNotes > idxWorkflow {
		t.Error("provider notes should appear before workflow rules")
	}
	if !strings.Contains(result, "### Gemini-specific") {
		t.Error("gemini provider notes should be inserted")
	}
}

func TestBuildProviderSystemPrompt_FallbackWhenHeaderMissing(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPromptWithConfig(base, "openai", "gpt-5.2", config.DefaultConfig())

	if !strings.HasPrefix(result, "<!-- PROVIDER_NOTES_START:openai -->") {
		t.Error("when workflow header is missing, provider notes marker block should be prepended")
	}
	if !strings.HasSuffix(result, base) {
		t.Error("fallback behavior should keep the original base prompt")
	}
}

func TestBuildProviderSystemPrompt_EmptyProvider(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPromptWithConfig(base, "", "", config.DefaultConfig())

	if result != base {
		t.Error("empty provider should return unchanged base prompt")
	}
}

func TestBuildProviderSystemPrompt_Anthropic(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	anthropic := BuildProviderSystemPromptWithConfig(base, "anthropic", "claude-sonnet-4-6", config.DefaultConfig())
	claude := BuildProviderSystemPromptWithConfig(base, "claude", "claude-sonnet-4-6", config.DefaultConfig())

	if anthropic != claude {
		t.Error("anthropic and claude should produce identical system prompts")
	}
}

func TestBuildProviderSystemPrompt_OpenRouterReturnsBase(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	result := BuildProviderSystemPromptWithConfig(base, "openrouter", "anthropic/claude-opus-4.6", config.DefaultConfig())

	if result != base {
		t.Error("openrouter with empty prefix should return unchanged base prompt")
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
	if !strings.Contains(claudePrompt, "search_code: code discovery tool") || !strings.Contains(claudePrompt, "read_file: low-level exact-content reader") {
		t.Fatal("claude prompt should keep low-level investigation override guidance when those tools are visible")
	}

	openAIPrompt := GetSystemPromptForProvider("openai", "gpt-5.4")
	if !strings.Contains(openAIPrompt, "### apply_patch (edit tool)") {
		t.Fatal("openai prompt should include apply_patch guidance")
	}
	if strings.Contains(openAIPrompt, "### Legacy edit tools") {
		t.Fatal("openai prompt should not include legacy edit tool guidance")
	}
	if strings.Contains(openAIPrompt, "search_code: code discovery tool") || strings.Contains(openAIPrompt, "read_file: low-level exact-content reader") {
		t.Fatal("openai prompt should not advertise legacy low-level investigation overrides")
	}
	if !strings.Contains(openAIPrompt, "read_file: exact-content reader for edit/apply_patch exact-control override") {
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

func TestBuildProviderSystemPrompt_AzureReusesOpenAINotes(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	result := BuildProviderSystemPromptWithConfig(base, "azure", "azure-gpt-5.4", config.DefaultConfig())
	if !strings.Contains(result, "### OpenAI-specific") {
		t.Fatal("azure provider prompt should reuse OpenAI-specific provider notes")
	}
	displayResult := BuildProviderSystemPromptWithConfig(base, "Azure OpenAI", "azure-gpt-5.4", config.DefaultConfig())
	if !strings.Contains(displayResult, "### OpenAI-specific") {
		t.Fatal("Azure OpenAI display name prompt should reuse OpenAI-specific provider notes")
	}
}

func TestBuildProviderSystemPromptWithConfig_AddsMissingPrefix(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	got := BuildProviderSystemPromptWithConfig(base, "openai", "gpt-5.4", config.DefaultConfig())
	if !strings.Contains(got, "## Provider Notes") {
		t.Fatalf("provider notes should be added when missing:\n%s", got)
	}
	if !strings.Contains(got, "### OpenAI-specific") {
		t.Fatalf("openai-specific notes should be added:\n%s", got)
	}
}

func TestBuildProviderSystemPromptWithConfig_DoesNotDuplicatePrefix(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	wrapped := BuildProviderSystemPromptWithConfig(base, "openai", "gpt-5.4", config.DefaultConfig())
	got := BuildProviderSystemPromptWithConfig(wrapped, "openai", "gpt-5.4", config.DefaultConfig())
	if strings.Count(got, "## Provider Notes") != 1 {
		t.Fatalf("provider notes should not be duplicated:\n%s", got)
	}
}

func TestBuildProviderSystemPromptWithConfig_ReplacesProviderNotesByMarker(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	wrapped := BuildProviderSystemPromptWithConfig(base, "openai", "gpt-5.4", config.DefaultConfig())
	got := BuildProviderSystemPromptWithConfig(wrapped, "gemini", "gemini-3.1-pro-preview-customtools", config.DefaultConfig())
	if strings.Count(got, "## Provider Notes") != 1 {
		t.Fatalf("provider notes should remain a single block:\n%s", got)
	}
	if !strings.Contains(got, "### Gemini-specific") {
		t.Fatalf("gemini provider notes should replace openai notes:\n%s", got)
	}
	if strings.Contains(got, "### OpenAI-specific") {
		t.Fatalf("openai provider notes should be replaced:\n%s", got)
	}
}

func TestBuildProviderSystemPromptWithConfig_EmptyPrefixStripsExistingProviderNotes(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	wrapped := BuildProviderSystemPromptWithConfig(base, "openai", "gpt-5.4", config.DefaultConfig())
	got := BuildProviderSystemPromptWithConfig(wrapped, "openrouter", "openai/gpt-5.4", config.DefaultConfig())
	if strings.Contains(got, "## Provider Notes") {
		t.Fatalf("empty-prefix provider should strip previous provider notes:\n%s", got)
	}
}
