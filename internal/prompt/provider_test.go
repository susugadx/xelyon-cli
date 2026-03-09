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
		"read broadly first",
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
		"read broadly first",
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
	checks := []string{
		"Use search_code for file search",
		"Use read_file for file contents",
		"parallel tool calls",
	}
	for _, check := range checks {
		if !strings.Contains(prefix, check) {
			t.Errorf("claude prefix missing %q", check)
		}
	}
}

func TestGetProviderPrefix_Anthropic(t *testing.T) {
	anthropic := GetProviderPrefix("anthropic")
	claude := GetProviderPrefix("claude")
	if anthropic == "" {
		t.Fatal("expected non-empty prefix for anthropic")
	}
	if anthropic != claude {
		t.Error("anthropic and claude should return identical prefixes")
	}
}

func TestGetProviderPrefix_Bedrock(t *testing.T) {
	bedrock := GetProviderPrefix("bedrock")
	claude := GetProviderPrefix("claude")
	if bedrock == "" {
		t.Fatal("expected non-empty prefix for bedrock")
	}
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
	checks := []string{
		"proceed without asking for confirmation",
		"root cause",
		"targeted str_replace edits",
		"orphaned dependencies",
	}
	for _, check := range checks {
		if !strings.Contains(prefix, check) {
			t.Errorf("openai prefix missing %q", check)
		}
	}
}

func TestGetProviderPrefix_OpenAICaseInsensitive(t *testing.T) {
	prefix := GetProviderPrefix("OpenAI")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for OpenAI (uppercase)")
	}
}

func TestCommonExecutionBlock(t *testing.T) {
	gemini := GetProviderPrefix("gemini")
	deepseek := GetProviderPrefix("deepseek")
	openai := GetProviderPrefix("openai")

	commonChecks := []string{
		"### Shared Notes",
		"prefer search_code -> str_replace(line-range)",
		"read broadly first",
		"Wait for build/test output",
		"Project Context",
	}
	for _, check := range commonChecks {
		if !strings.Contains(gemini, check) {
			t.Errorf("gemini prefix missing common note: %s", check)
		}
		if !strings.Contains(deepseek, check) {
			t.Errorf("deepseek prefix missing common note: %s", check)
		}
		if !strings.Contains(openai, check) {
			t.Errorf("openai prefix missing common note: %s", check)
		}
	}
}

func TestCommonBoostBlock(t *testing.T) {
	providers := []string{"gemini", "deepseek", "groq", "openai", "claude", "openrouter", "ollama"}
	boostChecks := []string{
		"### Quality Boost",
		"surrounding code",
		"extra read is cheaper",
		"self-review changed call sites",
	}
	for _, provider := range providers {
		prefix := GetProviderPrefix(provider)
		for _, check := range boostChecks {
			if !strings.Contains(prefix, check) {
				t.Errorf("%s prefix missing boost rule: %s", provider, check)
			}
		}
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

	openaiOnlyChecks := []string{
		"proceed without asking for confirmation",
		"root cause",
		"targeted str_replace edits",
		"orphaned dependencies",
	}
	for _, check := range openaiOnlyChecks {
		if !strings.Contains(openai, check) {
			t.Errorf("openai should contain rule: %s", check)
		}
		if strings.Contains(gemini, check) {
			t.Errorf("gemini should not contain openai-specific rule: %s", check)
		}
		if strings.Contains(deepseek, check) {
			t.Errorf("deepseek should not contain openai-specific rule: %s", check)
		}
	}
}

func TestBuildProviderSystemPrompt_InsertsBeforeWorkflowRules(t *testing.T) {
	base := "You are XELYON.\n\n## Core Identity\n- test\n\n## Workflow Rules\n- workflow"
	result := BuildProviderSystemPrompt(base, "gemini", "gemini-3.1-pro-preview-customtools")

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
	result := BuildProviderSystemPrompt(base, "openai", "gpt-5.2")

	if !strings.HasPrefix(result, "## Provider Notes") {
		t.Error("when workflow header is missing, provider notes should be prepended")
	}
	if !strings.HasSuffix(result, base) {
		t.Error("fallback behavior should keep the original base prompt")
	}
}

func TestBuildProviderSystemPrompt_EmptyProvider(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "", "")

	if result != base {
		t.Error("empty provider should return unchanged base prompt")
	}
}

func TestBuildProviderSystemPrompt_Anthropic(t *testing.T) {
	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	anthropic := BuildProviderSystemPrompt(base, "anthropic", "claude-sonnet-4-6")
	claude := BuildProviderSystemPrompt(base, "claude", "claude-sonnet-4-6")

	if anthropic != claude {
		t.Error("anthropic and claude should produce identical system prompts")
	}
}

func TestBuildProviderSystemPrompt_ClaudeOpusAddsCacheMargin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	sonnet := BuildProviderSystemPromptWithConfig(base, "claude", "claude-sonnet-4-6", cfg)
	opus := BuildProviderSystemPromptWithConfig(base, "claude", "claude-opus-4-6", cfg)

	if strings.Contains(sonnet, "### Stable Working Reference") {
		t.Fatal("sonnet prompt should not include the Opus cache margin block")
	}
	if !strings.Contains(opus, "### Stable Working Reference") {
		t.Fatal("opus prompt should include the cache margin block")
	}
	if len(opus)-len(sonnet) < 2500 {
		t.Fatalf("expected opus cache margin to add substantial stable context, delta=%d", len(opus)-len(sonnet))
	}
}

func TestBuildProviderSystemPrompt_OpenRouterAnthropicOpusAddsCacheMargin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	result := BuildProviderSystemPromptWithConfig(base, "openrouter", "anthropic/claude-opus-4.6", cfg)

	if !strings.Contains(result, "### Stable Working Reference") {
		t.Fatal("openrouter anthropic opus prompt should include the cache margin block")
	}
}

func TestBuildProviderSystemPrompt_CacheMarginDisabledWhenPromptCacheOff(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = false

	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	result := BuildProviderSystemPromptWithConfig(base, "claude", "claude-opus-4-6", cfg)

	if strings.Contains(result, "### Stable Working Reference") {
		t.Fatal("cache margin block should not be included when prompt cache is disabled")
	}
}

func TestBuildProviderSystemPromptWithConfig_UsesInjectedConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	base := "You are XELYON.\n\n## Workflow Rules\n- workflow"
	result := BuildProviderSystemPromptWithConfig(base, "claude", "claude-opus-4-6", cfg)

	if !strings.Contains(result, "### Stable Working Reference") {
		t.Fatal("injected config should enable cache margin block")
	}
}
