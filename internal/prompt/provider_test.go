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
	if !strings.Contains(prefix, "read_file BEFORE str_replace") {
		t.Error("gemini prefix should contain read_file rule")
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
	if !strings.Contains(prefix, "read_file BEFORE str_replace") {
		t.Error("deepseek prefix should contain read_file rule")
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
	if prefix != "" {
		t.Errorf("expected empty prefix for claude, got: %q", prefix)
	}
}

func TestGetProviderPrefix_Unknown(t *testing.T) {
	prefix := GetProviderPrefix("unknown-provider")
	if prefix != "" {
		t.Errorf("expected empty prefix for unknown provider, got: %q", prefix)
	}
}

func TestCommonRulesBlock(t *testing.T) {
	// 共通ルールが Gemini と DeepSeek の両方に含まれていることを確認
	gemini := GetProviderPrefix("gemini")
	deepseek := GetProviderPrefix("deepseek")

	commonChecks := []string{
		"read_file BEFORE str_replace",
		"search_code",
		"WAIT for output",
		"Follow project rules in Project Context",
		"grep_replace",
	}
	for _, check := range commonChecks {
		if !strings.Contains(gemini, check) {
			t.Errorf("gemini prefix missing common rule: %s", check)
		}
		if !strings.Contains(deepseek, check) {
			t.Errorf("deepseek prefix missing common rule: %s", check)
		}
	}
}

func TestProviderSpecificRules(t *testing.T) {
	gemini := GetProviderPrefix("gemini")
	deepseek := GetProviderPrefix("deepseek")

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
	if !strings.Contains(result, "read_file BEFORE str_replace") {
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
	if !strings.Contains(result, "read_file BEFORE str_replace") {
		t.Error("deepseek result should contain read_file rule")
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

	if result != base {
		t.Errorf("claude result should be unchanged base prompt, got diff length: %d vs %d", len(result), len(base))
	}
}

func TestBuildProviderSystemPrompt_EmptyProvider(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "")

	if result != base {
		t.Error("empty provider should return unchanged base prompt")
	}
}
