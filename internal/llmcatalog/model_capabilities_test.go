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
			name:         "azure delegates image support to openai catalog model",
			provider:     "azure",
			model:        "gpt-5.4",
			wantKnown:    true,
			wantSupports: true,
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
		{
			name:      "unknown provider",
			provider:  "custom-provider",
			model:     "gpt-5.4",
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
			name:         "gemini hidden exact catalog model",
			provider:     "gemini",
			model:        "gemini-3.1-pro-preview",
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
			name:         "openrouter provider-prefixed model has known unsupported native web search",
			provider:     "openrouter",
			model:        "openai/gpt-5.4",
			wantKnown:    true,
			wantSupports: false,
		},
		{
			name:         "bedrock has no native web search",
			provider:     "bedrock",
			model:        "global.anthropic.claude-sonnet-4-6",
			wantKnown:    true,
			wantSupports: false,
		},
		{
			name:      "unknown provider",
			provider:  "custom-provider",
			model:     "gpt-5.4",
			wantKnown: false,
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

func TestGeminiFunctionCallingSupport(t *testing.T) {
	tests := []struct {
		model       string
		known       bool
		supported   bool
		replacement string
	}{
		{model: "gemini-3.5-flash", known: true, supported: true},
		{model: "models/gemini-3.1-flash-lite", known: true, supported: true},
		{model: "gemini-3.1-pro-preview-customtools", known: true, supported: true},
		{model: "gemini-2.5-flash-latest", known: true, supported: true},
		{model: "gemini-2.0-flash-001", known: true, supported: true},
		{model: "gemini-2.0-flash-lite", known: true, supported: false, replacement: "gemini-3.1-flash-lite"},
		{model: "models/gemini-2.0-flash-lite", known: true, supported: false, replacement: "gemini-3.1-flash-lite"},
		{model: "gemini-2.0-flash-lite-001", known: true, supported: false, replacement: "gemini-3.1-flash-lite"},
		{model: "corp-gemini-model", known: false, supported: false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := GeminiFunctionCallingSupport(tt.model)
			if got.Known != tt.known || got.Supported != tt.supported || got.Replacement != tt.replacement {
				t.Fatalf("GeminiFunctionCallingSupport(%q) = %#v, want known=%t supported=%t replacement=%q", tt.model, got, tt.known, tt.supported, tt.replacement)
			}
		})
	}
}
