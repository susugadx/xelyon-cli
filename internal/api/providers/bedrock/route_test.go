package bedrock

import "testing"

func TestResolveBedrockRoute(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		catalogModel string
		want         bedrockRoute
	}{
		{
			name:  "claude direct model uses Claude Messages",
			model: "global.anthropic.claude-sonnet-4-6",
			want:  bedrockRouteClaudeMessages,
		},
		{
			name:         "custom alias uses Claude Messages via catalog model",
			model:        "corp-bedrock-sonnet46",
			catalogModel: "global.anthropic.claude-sonnet-4-6",
			want:         bedrockRouteClaudeMessages,
		},
		{
			name:  "nova uses ConverseStream",
			model: "amazon.nova-pro-v1:0",
			want:  bedrockRouteConverseStream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveBedrockRoute(tt.model, tt.catalogModel); got != tt.want {
				t.Fatalf("resolveBedrockRoute(%q, %q) = %q, want %q", tt.model, tt.catalogModel, got, tt.want)
			}
		})
	}
}
