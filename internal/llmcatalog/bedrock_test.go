package llmcatalog

import "testing"

func TestBedrockModelFamilyFor(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		catalogModel string
		want         BedrockModelFamily
	}{
		{
			name:  "claude direct model",
			model: "global.anthropic.claude-sonnet-4-6",
			want:  BedrockModelFamilyClaude,
		},
		{
			name:         "claude catalog alias",
			model:        "corp-bedrock-sonnet46",
			catalogModel: "global.anthropic.claude-sonnet-4-6",
			want:         BedrockModelFamilyClaude,
		},
		{
			name:  "nova model",
			model: "amazon.nova-pro-v1:0",
			want:  BedrockModelFamilyConverse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BedrockModelFamilyFor(tt.model, tt.catalogModel); got != tt.want {
				t.Fatalf("BedrockModelFamilyFor(%q, %q) = %q, want %q", tt.model, tt.catalogModel, got, tt.want)
			}
		})
	}
}

func TestIsBedrockModelID(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "global.anthropic.claude-sonnet-4-6", want: true},
		{model: "us.anthropic.claude-sonnet-4-6", want: true},
		{model: "eu.anthropic.claude-sonnet-4-6", want: true},
		{model: "au.anthropic.claude-sonnet-4-6", want: true},
		{model: "amazon.nova-pro-v1:0", want: true},
		{model: "us.amazon.nova-pro-v1:0", want: true},
		{model: "meta.llama3-3-70b-instruct-v1:0", want: true},
		{model: "us.mistral.pixtral-large-2502-v1:0", want: true},
		{model: "us.writer.palmyra-x5-v1:0", want: true},
		{model: "us.deepseek.r1-v1:0", want: true},
		{model: "google.gemma-3-4b-it", want: true},
		{model: "moonshotai.kimi-k2.5", want: true},
		{model: "moonshotai.kimi-k2-thinking", want: true},
		{model: "qwen.qwen3-coder-480b-a35b-instruct-v1:0", want: true},
		{model: "minimax.minimax-m2", want: true},
		{model: "moonshot.kimi-k2-thinking", want: false},
		{model: "anthropic/claude-sonnet-4.6", want: false},
		{model: "gpt-5.4", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsBedrockModelID(tt.model); got != tt.want {
				t.Fatalf("IsBedrockModelID(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestIsBedrockClaudeModel_AvoidsSubstringOnlyMatch(t *testing.T) {
	if IsBedrockClaudeModel("amazon.nova-notclaude-v1:0") {
		t.Fatal("IsBedrockClaudeModel() should not treat arbitrary claude substring as Claude family")
	}
}
