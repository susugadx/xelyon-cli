package llmcatalog

import "testing"

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
