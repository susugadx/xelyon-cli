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
			model: "global.anthropic.claude-sonnet-4-6-v1",
			want:  BedrockModelFamilyClaude,
		},
		{
			name:         "claude catalog alias",
			model:        "corp-bedrock-sonnet46",
			catalogModel: "global.anthropic.claude-sonnet-4-6-v1",
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
		{model: "global.anthropic.claude-sonnet-4-6-v1", want: true},
		{model: "amazon.nova-pro-v1:0", want: true},
		{model: "us.amazon.nova-pro-v1:0", want: true},
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
