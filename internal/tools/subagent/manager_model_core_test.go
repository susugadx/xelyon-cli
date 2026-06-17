package subagent

import "testing"

// TestInferSubAgentModel は provider ごとの既定サブエージェントモデルを確認します。
func TestInferSubAgentModel(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{provider: "openai", want: "gpt-5.4-mini"},
		{provider: "claude", want: "claude-haiku-4-5-20251001"},
		{provider: "anthropic", want: "claude-haiku-4-5-20251001"},
		{provider: "gemini", want: "gemini-3.1-flash-lite"},
		{provider: "deepseek", want: "deepseek-v4-flash"},
		{provider: "groq", want: "llama-3.3-70b-versatile"},
		{provider: "openrouter", want: "openai/gpt-5.4-mini"},
		{provider: "azure", want: ""},
		{provider: "unknown", want: ""},
	}

	for _, tt := range tests {
		if got := inferSubAgentModel(nil, tt.provider); got != tt.want {
			t.Errorf("inferSubAgentModel(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

// TestNormalizeSubAgentModel はプレースホルダ文字列を未設定として扱うことを確認します。
func TestNormalizeSubAgentModel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "   ", want: ""},
		{input: "sub_agent.default_model", want: ""},
		{input: "SUB_AGENT.DEFAULT_MODEL", want: ""},
		{input: "gpt-5.4-mini", want: "gpt-5.4-mini"},
	}

	for _, tt := range tests {
		if got := normalizeSubAgentModel(tt.input); got != tt.want {
			t.Errorf("normalizeSubAgentModel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
