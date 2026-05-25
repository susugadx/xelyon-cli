package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
)

func TestSlashSuggestions_ShowProviderRuntimeCandidates(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		providerCandidates: []providerpicker.ProviderCandidate{
			{Key: "openai", Label: "OpenAI", Current: true, CredentialStatus: providerpicker.ProviderCredentialConfigured},
			{Key: "claude", Label: "Claude", CredentialStatus: providerpicker.ProviderCredentialMissingKey},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/provider o")

	if !m.slashSuggestions.visible() {
		t.Fatal("provider argument suggestions should be visible")
	}
	if got := len(m.slashSuggestions.suggestions); got != 1 {
		t.Fatalf("provider suggestions len = %d, want 1", got)
	}
	suggestion := m.slashSuggestions.suggestions[0]
	if suggestion.InsertText != "/provider openai" || suggestion.Category != commandcatalog.CommandCategoryModel {
		t.Fatalf("provider suggestion = %#v", suggestion)
	}
	if suggestion.CategoryLabel != "provider" {
		t.Fatalf("provider category label = %q, want provider", suggestion.CategoryLabel)
	}
	if !suggestion.HideCategory {
		t.Fatal("provider argument suggestion should hide repeated category column")
	}
	if got := suggestion.CategoryDisplayLabel(); got != "" {
		t.Fatalf("provider display category = %q, want empty", got)
	}
	rendered := stripANSI(m.chromeCache)
	if !strings.Contains(rendered, "OpenAI · current · configured") || strings.Contains(rendered, "/provider openai") {
		t.Fatalf("provider suggestion render missing display category/description:\n%s", rendered)
	}
	if detail := m.selectedSlashSuggestionDetailText(); !strings.Contains(detail, "Switch to OpenAI") {
		t.Fatalf("selected detail = %q, want provider detail", detail)
	}
}

func TestSlashSuggestions_ShowModelRuntimeCandidates(t *testing.T) {
	agent := &stubAgent{
		statusLine:        "ready",
		providerConfigKey: "openai",
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"openai": {
				{Name: "gpt-5.4", Current: true},
				{Name: "gpt-5.4-mini", Default: true},
				{Name: "Custom model...", Custom: true},
			},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/model gpt-5.4-m")

	if !m.slashSuggestions.visible() {
		t.Fatal("model argument suggestions should be visible")
	}
	if got := len(m.slashSuggestions.suggestions); got != 1 {
		t.Fatalf("model suggestions len = %d, want 1", got)
	}
	suggestion := m.slashSuggestions.suggestions[0]
	if suggestion.InsertText != "/model gpt-5.4-mini" {
		t.Fatalf("model suggestion insert = %q, want /model gpt-5.4-mini", suggestion.InsertText)
	}
	if suggestion.CategoryLabel != "model" {
		t.Fatalf("model category label = %q, want model", suggestion.CategoryLabel)
	}
	if !suggestion.HideCategory {
		t.Fatal("model argument suggestion should hide repeated category column")
	}
	if got := suggestion.CategoryDisplayLabel(); got != "" {
		t.Fatalf("model display category = %q, want empty", got)
	}
	if strings.Contains(stripANSI(m.chromeCache), "Custom model") {
		t.Fatalf("custom model candidate should not be inserted as slash argument:\n%s", stripANSI(m.chromeCache))
	}
}

func TestSlashSuggestions_ShowGemini35FlashModelCandidate(t *testing.T) {
	agent := &stubAgent{
		statusLine:        "ready",
		providerConfigKey: "gemini",
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"gemini": {
				{Name: "gemini-3.5-flash"},
				{Name: "gemini-2.5-flash", Current: true},
				{Name: "Custom model...", Custom: true},
			},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/model gemini-3.5")

	if !m.slashSuggestions.visible() {
		t.Fatal("gemini model argument suggestions should be visible")
	}
	if got := len(m.slashSuggestions.suggestions); got != 1 {
		t.Fatalf("gemini model suggestions len = %d, want 1", got)
	}
	if got := m.slashSuggestions.suggestions[0].InsertText; got != "/model gemini-3.5-flash" {
		t.Fatalf("gemini model suggestion insert = %q, want /model gemini-3.5-flash", got)
	}
}

func TestSlashSuggestions_ShowGemini31FlashLiteModelCandidate(t *testing.T) {
	agent := &stubAgent{
		statusLine:        "ready",
		providerConfigKey: "gemini",
		modelCandidates: map[string][]providerpicker.ModelCandidate{
			"gemini": {
				{Name: "gemini-3.5-flash"},
				{Name: "gemini-3.1-flash-lite"},
				{Name: "Custom model...", Custom: true},
			},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/model gemini-3.1-f")

	if !m.slashSuggestions.visible() {
		t.Fatal("gemini model argument suggestions should be visible")
	}
	if got := len(m.slashSuggestions.suggestions); got != 1 {
		t.Fatalf("gemini model suggestions len = %d, want 1", got)
	}
	if got := m.slashSuggestions.suggestions[0].InsertText; got != "/model gemini-3.1-flash-lite" {
		t.Fatalf("gemini model suggestion insert = %q, want /model gemini-3.1-flash-lite", got)
	}
}
