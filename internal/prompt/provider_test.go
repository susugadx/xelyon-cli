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
		"briefly state what you are about to do",
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

func TestGetProviderPrefix_Bedrock(t *testing.T) {
	bedrock := GetProviderPrefix("bedrock")
	claude := GetProviderPrefix("claude")
	if bedrock != claude {
		t.Error("bedrock and claude should return identical prefixes")
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
	if !strings.Contains(prefix, "briefly state what you are about to do") {
		t.Error("openai prefix missing narration rule")
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

	if !strings.HasPrefix(result, "## Provider Notes") {
		t.Error("when workflow header is missing, provider notes should be prepended")
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
