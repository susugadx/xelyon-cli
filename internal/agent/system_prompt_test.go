package agent

import (
	"strings"
	"testing"
)

func TestGetProviderPrefix_Gemini(t *testing.T) {
	prefix := getProviderPrefix("gemini")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for gemini")
	}
	if !strings.Contains(prefix, "ALWAYS read_file BEFORE str_replace") {
		t.Error("gemini prefix should contain read_file rule")
	}
}

func TestGetProviderPrefix_GeminiCaseInsensitive(t *testing.T) {
	// providerPrefixes はキーが小文字なので、getProviderPrefix 内で ToLower している
	prefix := getProviderPrefix("Gemini")
	if prefix == "" {
		t.Fatal("expected non-empty prefix for Gemini (uppercase)")
	}
}

func TestGetProviderPrefix_Claude(t *testing.T) {
	prefix := getProviderPrefix("claude")
	if prefix != "" {
		t.Errorf("expected empty prefix for claude, got: %q", prefix)
	}
}

func TestGetProviderPrefix_Unknown(t *testing.T) {
	prefix := getProviderPrefix("unknown-provider")
	if prefix != "" {
		t.Errorf("expected empty prefix for unknown provider, got: %q", prefix)
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
	if !strings.Contains(result, "NEVER guess file contents") {
		t.Error("gemini result should contain the critical rule")
	}
}

func TestBuildProviderSystemPrompt_Claude(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "claude")

	if result != base {
		t.Errorf("claude result should be unchanged base prompt, got diff length: %d vs %d", len(result), len(base))
	}
}

func TestBuildProviderSystemPrompt_DeepSeek(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "deepseek")

	if result != base {
		t.Errorf("deepseek result should be unchanged base prompt")
	}
}

func TestBuildProviderSystemPrompt_EmptyProvider(t *testing.T) {
	base := "You are XELYON, an autonomous AI coding agent."
	result := BuildProviderSystemPrompt(base, "")

	if result != base {
		t.Error("empty provider should return unchanged base prompt")
	}
}
