package prompt

import (
	"strings"
	"testing"
)

func TestGetProviderPrefix_Gemini(t *testing.T) {
	prefix := GetProviderPrefix("gemini")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for gemini")
	}
	if !strings.Contains(prefix, "search_code → str_replace(line-range) is PREFERRED") {
		t.Error("gemini prefix should contain search_code → str_replace(line-range) rule")
	}
	if !strings.Contains(prefix, "search_code") {
		t.Error("gemini prefix should contain search_code rule for reference checking")
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
	if !strings.Contains(prefix, "search_code → str_replace(line-range) is PREFERRED") {
		t.Error("deepseek prefix should contain search_code → str_replace(line-range) rule")
	}
	if !strings.Contains(prefix, "search_code") {
		t.Error("deepseek prefix should contain search_code rule")
	}
	if !strings.Contains(prefix, "ALWAYS use tool calls for file operations") {
		t.Error("deepseek prefix should contain tool calls rule")
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
	if !strings.Contains(prefix, "search_code → str_replace(line-range) is PREFERRED") {
		t.Error("claude prefix should contain search_code → str_replace(line-range) rule")
	}
	if !strings.Contains(prefix, "File search → search_code, NOT bash (grep/rg/find)") {
		t.Error("claude prefix should contain file search rule")
	}
	if !strings.Contains(prefix, "File reading → read_file, NOT bash (cat/head/tail/sed)") {
		t.Error("claude prefix should contain file reading rule")
	}
}

func TestGetProviderPrefix_Anthropic(t *testing.T) {
	// "anthropic" は "claude" のエイリアス — 同一プレフィックスが返ること
	anthropic := GetProviderPrefix("anthropic")
	claude := GetProviderPrefix("claude")
	if anthropic == "" {
		t.Fatal("expected non-empty prefix for anthropic")
	}
	if anthropic != claude {
		t.Error("anthropic and claude should return identical prefixes")
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
	if !strings.Contains(prefix, "search_code → str_replace(line-range) is PREFERRED") {
		t.Error("openai prefix should contain common search_code rule")
	}
	if !strings.Contains(prefix, "execute immediately WITHOUT asking for confirmation") {
		t.Error("openai prefix should contain no-confirmation rule")
	}
	if !strings.Contains(prefix, "NEVER delete existing logic to fix a compile error") {
		t.Error("openai prefix should contain no-delete-on-error rule")
	}
	if !strings.Contains(prefix, "fix ONLY the broken line/call") {
		t.Error("openai prefix should contain targeted-fix rule")
	}
	if !strings.Contains(prefix, "mixed Japanese/JSON/backticks") {
		t.Error("openai prefix should contain str_replace safety rule")
	}
}

func TestGetProviderPrefix_OpenAICaseInsensitive(t *testing.T) {
	prefix := GetProviderPrefix("OpenAI")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for OpenAI (uppercase)")
	}
}

func TestCommonRulesBlock(t *testing.T) {
	// 共通ルールが全プロバイダーに含まれていることを確認
	gemini := GetProviderPrefix("gemini")
	deepseek := GetProviderPrefix("deepseek")
	openai := GetProviderPrefix("openai")

	commonChecks := []string{
		"search_code → str_replace(line-range) is PREFERRED",
		"search_code",
		"WAIT for output",
		"Follow project rules in Project Context",
		"str_replace batch mode",
	}
	for _, check := range commonChecks {
		if !strings.Contains(gemini, check) {
			t.Errorf("gemini prefix missing common rule: %s", check)
		}
		if !strings.Contains(deepseek, check) {
			t.Errorf("deepseek prefix missing common rule: %s", check)
		}
		if !strings.Contains(openai, check) {
			t.Errorf("openai prefix missing common rule: %s", check)
		}
	}
}

func TestProviderSpecificRules(t *testing.T) {
	gemini := GetProviderPrefix("gemini")
	deepseek := GetProviderPrefix("deepseek")
	openai := GetProviderPrefix("openai")

	// Gemini 固有: マークダウンコードブロック禁止
	if !strings.Contains(gemini, "NOT inside markdown code blocks") {
		t.Error("gemini should contain code block rule")
	}
	if strings.Contains(deepseek, "NOT inside markdown code blocks") {
		t.Error("deepseek should NOT contain gemini-specific code block rule")
	}

	// DeepSeek 固有: ツールコール使用の強制
	if !strings.Contains(deepseek, "ALWAYS use tool calls for file operations") {
		t.Error("deepseek should contain tool calls rule")
	}
	if strings.Contains(gemini, "ALWAYS use tool calls for file operations") {
		t.Error("gemini should NOT contain deepseek-specific tool calls rule")
	}

	// DeepSeek 固有: エラー完全修正・unused import 即時削除
	deepseekOnlyChecks := []string{
		"Fix ALL errors completely",
		"NEVER leave errors with excuses",
		"unused imports appear, remove them IMMEDIATELY",
	}
	for _, check := range deepseekOnlyChecks {
		if !strings.Contains(deepseek, check) {
			t.Errorf("deepseek should contain rule: %s", check)
		}
		if strings.Contains(gemini, check) {
			t.Errorf("gemini should NOT contain deepseek-specific rule: %s", check)
		}
	}

	// OpenAI 固有: 確認不要・ロジック削除禁止・ターゲット修正
	openaiOnlyChecks := []string{
		"execute immediately WITHOUT asking for confirmation",
		"NEVER delete existing logic to fix a compile error",
		"fix ONLY the broken line/call",
		"mixed Japanese/JSON/backticks",
	}
	for _, check := range openaiOnlyChecks {
		if !strings.Contains(openai, check) {
			t.Errorf("openai should contain rule: %s", check)
		}
		if strings.Contains(gemini, check) {
			t.Errorf("gemini should NOT contain openai-specific rule: %s", check)
		}
		if strings.Contains(deepseek, check) {
			t.Errorf("deepseek should NOT contain openai-specific rule: %s", check)
		}
	}
}

func TestBuildProviderSystemPrompt_Gemini(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "gemini")

	if !strings.HasPrefix(result, "## ") {
		t.Error("gemini result should start with prefix header")
	}
	if !strings.HasSuffix(result, base) {
		t.Error("gemini result should end with base prompt")
	}
	if !strings.Contains(result, "search_code → str_replace(line-range) is PREFERRED") {
		t.Error("gemini result should contain the critical rule")
	}
	if !strings.Contains(result, "WAIT for output") {
		t.Error("gemini result should contain verification wait rule")
	}
	if !strings.Contains(result, "NOT inside markdown code blocks") {
		t.Error("gemini result should contain code block rule")
	}
	if !strings.Contains(result, "search_code") {
		t.Error("gemini result should contain search_code rule")
	}
}

func TestBuildProviderSystemPrompt_DeepSeek(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "deepseek")

	if !strings.HasPrefix(result, "## ") {
		t.Error("deepseek result should start with prefix header")
	}
	if !strings.HasSuffix(result, base) {
		t.Error("deepseek result should end with base prompt")
	}
	if !strings.Contains(result, "search_code → str_replace(line-range) is PREFERRED") {
		t.Error("deepseek result should contain search_code → str_replace(line-range) rule")
	}
	if !strings.Contains(result, "ALWAYS use tool calls for file operations") {
		t.Error("deepseek result should contain tool calls rule")
	}
	if !strings.Contains(result, "Fix ALL errors completely") {
		t.Error("deepseek result should contain error fix rule")
	}
	if !strings.Contains(result, "unused imports appear, remove them IMMEDIATELY") {
		t.Error("deepseek result should contain unused import rule")
	}
}

func TestBuildProviderSystemPrompt_Claude(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "claude")

	if !strings.HasPrefix(result, "## ") {
		t.Error("claude result should start with prefix header")
	}
	if !strings.HasSuffix(result, base) {
		t.Error("claude result should end with base prompt")
	}
	if !strings.Contains(result, "search_code → str_replace(line-range) is PREFERRED") {
		t.Error("claude result should contain search_code → str_replace(line-range) rule")
	}
	if !strings.Contains(result, "File search → search_code, NOT bash (grep/rg/find)") {
		t.Error("claude result should contain file search rule")
	}
	if !strings.Contains(result, "File reading → read_file, NOT bash (cat/head/tail/sed)") {
		t.Error("claude result should contain file reading rule")
	}
}

func TestBuildProviderSystemPrompt_OpenAI(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "openai")

	if !strings.HasPrefix(result, "## ") {
		t.Error("openai result should start with prefix header")
	}
	if !strings.HasSuffix(result, base) {
		t.Error("openai result should end with base prompt")
	}
	if !strings.Contains(result, "execute immediately WITHOUT asking for confirmation") {
		t.Error("openai result should contain no-confirmation rule")
	}
	if !strings.Contains(result, "NEVER delete existing logic to fix a compile error") {
		t.Error("openai result should contain no-delete-on-error rule")
	}
}

func TestBuildProviderSystemPrompt_EmptyProvider(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "")

	if result != base {
		t.Error("empty provider should return unchanged base prompt")
	}
}

func TestBuildProviderSystemPrompt_Anthropic(t *testing.T) {
	// "anthropic" は "claude" のエイリアス — 同一結果が返ること
	base := "You are XELYON, an autonomous AI coding agent."
	anthropic := BuildProviderSystemPrompt(base, "anthropic")
	claude := BuildProviderSystemPrompt(base, "claude")

	if anthropic != claude {
		t.Error("anthropic and claude should produce identical system prompts")
	}
}
