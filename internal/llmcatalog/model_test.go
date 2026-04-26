package llmcatalog

import "testing"

func TestGPT55CatalogLimits(t *testing.T) {
	tests := []struct {
		model       string
		wantMaxOut  int
		wantContext int
	}{
		{model: "gpt-5.5", wantMaxOut: 128000, wantContext: 1050000},
		{model: "gpt-5.5-pro", wantMaxOut: 128000, wantContext: 1050000},
		{model: "gpt-5.5-2026-04-23", wantMaxOut: 128000, wantContext: 1050000},
		{model: "gpt-5.5-pro-2026-04-23", wantMaxOut: 128000, wantContext: 1050000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			gotMaxOut, ok := KnownMaxOutputTokens(tt.model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", tt.model)
			}
			if gotMaxOut != tt.wantMaxOut {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want %d", tt.model, gotMaxOut, tt.wantMaxOut)
			}
			if gotContext := ModelContextLimit(tt.model); gotContext != tt.wantContext {
				t.Fatalf("ModelContextLimit(%q) = %d, want %d", tt.model, gotContext, tt.wantContext)
			}
		})
	}
}

func TestGPT55ResponsesAPIModel(t *testing.T) {
	tests := []string{"gpt-5.5", "gpt-5.5-pro"}
	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			if !IsOpenAIResponsesModel(model, nil) {
				t.Fatalf("IsOpenAIResponsesModel(%q) = false, want true", model)
			}
		})
	}
}
