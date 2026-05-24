package llmcatalog

import "testing"

func TestKnownImageInputSupport(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		model        string
		wantKnown    bool
		wantSupports bool
	}{
		{
			name:         "openai vision model",
			provider:     "openai",
			model:        "gpt-5.4",
			wantKnown:    true,
			wantSupports: true,
		},
		{
			name:         "openai known text reasoning model",
			provider:     "openai",
			model:        "o3-mini",
			wantKnown:    true,
			wantSupports: false,
		},
		{
			name:      "openai unknown model",
			provider:  "openai",
			model:     "gpt-next-custom",
			wantKnown: false,
		},
		{
			name:         "openrouter delegated openai vision model",
			provider:     "openrouter",
			model:        "openai/gpt-5.4",
			wantKnown:    true,
			wantSupports: true,
		},
		{
			name:         "openrouter delegated deepseek model",
			provider:     "openrouter",
			model:        "deepseek/deepseek-v4-flash",
			wantKnown:    true,
			wantSupports: false,
		},
		{
			name:      "openrouter unknown routed owner",
			provider:  "openrouter",
			model:     "vendor/model",
			wantKnown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supports, known := KnownImageInputSupport(tt.provider, tt.model)
			if known != tt.wantKnown || supports != tt.wantSupports {
				t.Fatalf("KnownImageInputSupport(%q, %q) = supports:%t known:%t, want supports:%t known:%t", tt.provider, tt.model, supports, known, tt.wantSupports, tt.wantKnown)
			}
		})
	}
}

func TestKnownWebSearchSupport(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		model        string
		wantKnown    bool
		wantSupports bool
	}{
		{
			name:         "claude listed model",
			provider:     "claude",
			model:        "claude-sonnet-4-6",
			wantKnown:    true,
			wantSupports: true,
		},
		{
			name:      "claude unverified alias",
			provider:  "claude",
			model:     "corp-claude-future",
			wantKnown: false,
		},
		{
			name:         "gemini listed model",
			provider:     "gemini",
			model:        "gemini-3.1-pro-preview-customtools",
			wantKnown:    true,
			wantSupports: true,
		},
		{
			name:      "gemini legacy pricing-only model",
			provider:  "gemini",
			model:     "gemini-pro",
			wantKnown: false,
		},
		{
			name:         "openai responses model",
			provider:     "openai",
			model:        "gpt-5.4",
			wantKnown:    true,
			wantSupports: true,
		},
		{
			name:         "bedrock has no native web search",
			provider:     "bedrock",
			model:        "global.anthropic.claude-sonnet-4-6",
			wantKnown:    true,
			wantSupports: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supports, known := KnownWebSearchSupport(tt.provider, tt.model)
			if known != tt.wantKnown || supports != tt.wantSupports {
				t.Fatalf("KnownWebSearchSupport(%q, %q) = supports:%t known:%t, want supports:%t known:%t", tt.provider, tt.model, supports, known, tt.wantSupports, tt.wantKnown)
			}
		})
	}
}
