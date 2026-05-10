package openai

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestShouldStreamResponsesWrapperUsesProviderDiagPolicy(t *testing.T) {
	for _, model := range []string{"", "gpt-5.3-codex", "gpt-5.5-pro", "gpt-5.5-pro-2026-05-01"} {
		if got, want := ShouldStreamResponses(model), providerdiag.ShouldStreamResponsesCatalogModel(model); got != want {
			t.Fatalf("ShouldStreamResponses(%q) = %t, want providerdiag %t", model, got, want)
		}
	}
}
