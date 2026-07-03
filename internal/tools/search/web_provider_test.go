package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveSearchProvider(t *testing.T) {
	tests := []struct {
		name              string
		configProvider    string
		mainProvider      string
		mainProviderOwner string
		want              string
	}{
		// config設定あり → そちらを使用
		{name: "config gemini wins", configProvider: "gemini", mainProvider: "deepseek", want: "gemini"},
		{name: "config openai wins", configProvider: "openai", mainProvider: "deepseek", want: "openai"},
		{name: "config claude wins", configProvider: "claude", mainProvider: "openai", want: "claude"},
		{name: "config kimi wins", configProvider: "kimi", mainProvider: "deepseek", want: "kimi"},
		{name: "config moonshot wins", configProvider: "moonshot", mainProvider: "deepseek", want: "moonshot"},
		{name: "config openai_subscription wins", configProvider: "openai_subscription", mainProvider: "deepseek", want: "openai_subscription"},
		{name: "config chatgpt alias canonicalizes", configProvider: "chatgpt", mainProvider: "deepseek", want: "openai_subscription"},
		{name: "config dashed subscription alias canonicalizes", configProvider: "openai-subscription", mainProvider: "deepseek", want: "openai_subscription"},
		{name: "config codex subscription alias canonicalizes", configProvider: "codex-subscription", mainProvider: "deepseek", want: "openai_subscription"},

		// config未設定 → exact owner key を優先
		{name: "session owner anthropic wins over canonical claude runtime", configProvider: "", mainProvider: "claude", mainProviderOwner: "anthropic", want: "anthropic"},
		{name: "session owner moonshot wins over canonical kimi runtime", configProvider: "", mainProvider: "kimi", mainProviderOwner: "moonshot", want: "moonshot"},
		{name: "session owner chatgpt canonicalizes before registry", configProvider: "", mainProvider: "openai_subscription", mainProviderOwner: "chatgpt", want: "openai_subscription"},

		// config未設定 → main provider
		{name: "fallback to openai main provider", configProvider: "", mainProvider: "openai", want: "openai"},
		{name: "fallback to openai_subscription main provider", configProvider: "", mainProvider: "openai_subscription", want: "openai_subscription"},
		{name: "fallback to chatgpt main provider alias canonicalizes", configProvider: "", mainProvider: "chatgpt", want: "openai_subscription"},
		{name: "fallback to gemini main provider", configProvider: "", mainProvider: "gemini", want: "gemini"},
		{name: "fallback to claude main provider", configProvider: "", mainProvider: "claude", want: "claude"},
		{name: "fallback to anthropic main provider", configProvider: "", mainProvider: "anthropic", want: "anthropic"},
		{name: "fallback to kimi main provider", configProvider: "", mainProvider: "kimi", want: "kimi"},
		{name: "fallback to moonshot main provider", configProvider: "", mainProvider: "moonshot", want: "moonshot"},

		// config未設定 + ネイティブ非対応 → 空文字
		{name: "deepseek unsupported", configProvider: "", mainProvider: "deepseek", want: ""},
		{name: "openrouter unsupported", configProvider: "", mainProvider: "openrouter", want: ""},
		{name: "groq unsupported", configProvider: "", mainProvider: "groq", want: ""},
		{name: "ollama unsupported", configProvider: "", mainProvider: "ollama", want: ""},

		// 無効なconfig設定 → メインにフォールバック
		{name: "invalid config falls back to openai", configProvider: "invalid", mainProvider: "openai", want: "openai"},
		{name: "invalid config falls back to empty", configProvider: "invalid", mainProvider: "deepseek", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{WebSearch: config.WebSearchConfig{Provider: tt.configProvider}}
			got := resolveSearchProvider(cfg, tt.mainProvider, tt.mainProviderOwner)
			if got != tt.want {
				t.Errorf("resolveSearchProvider(config=%q, main=%q, owner=%q) = %q, want %q", tt.configProvider, tt.mainProvider, tt.mainProviderOwner, got, tt.want)
			}
		})
	}
}

func TestWebSearchProviderError(t *testing.T) {
	msg := webSearchProviderError()
	if !strings.Contains(msg, "web_search.provider") {
		t.Fatal("error message should mention web_search.provider")
	}
	if !strings.Contains(msg, "gemini") {
		t.Fatal("error message should mention gemini as an option")
	}
}
