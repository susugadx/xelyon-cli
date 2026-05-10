package providerdiag

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestCatalogPolicyDetailsPreserveProviderFormatting(t *testing.T) {
	cfg := config.DefaultConfig()

	openAIPolicy := OpenAICatalogPolicy(cfg, "gpt-5.3-codex", "gpt-5.3-codex")
	if got, want := openAIPolicy.OpenAIDetail(), "catalog_model=gpt-5.3-codex, context_window=400000, max_output_tokens=128000, pricing=input $1.75/M cached $0.175/M output $14.00/M"; got != want {
		t.Fatalf("OpenAIDetail() = %q, want %q", got, want)
	}

	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-codex-deployment",
		CatalogModel: "gpt-5.3-codex",
		ModelOverrides: map[string]config.ModelOverride{
			"corp-codex-deployment": {
				CatalogModel: "gpt-5.3-codex",
			},
		},
	})
	azurePolicy := AzureCatalogPolicy(cfg, "corp-codex-deployment", "gpt-5.3-codex")
	if got, want := azurePolicy.AzureDetail(), "catalog_model=gpt-5.3-codex, context_window=400000, max_output_tokens=128000 (catalog), responses_streaming=true, pricing=input $1.75/M cached $0.175/M output $14.00/M"; got != want {
		t.Fatalf("AzureDetail() = %q, want %q", got, want)
	}
}

func TestMaxOutputPolicyPreservesOpenAIAndAzureFallbackDifference(t *testing.T) {
	cfg := config.DefaultConfig()

	openAIPolicy := OpenAICatalogPolicy(cfg, "gpt-5.2-pro", "gpt-5.2-pro")
	if !openAIPolicy.MaxOutput.Available || openAIPolicy.MaxOutput.Source != "provider_default" || openAIPolicy.MaxOutput.Tokens != 16384 {
		t.Fatalf("OpenAI max output = %+v, want provider default available", openAIPolicy.MaxOutput)
	}

	azurePolicy := AzureCatalogPolicy(cfg, "corp-gpt52-pro-deployment", "gpt-5.2-pro")
	if azurePolicy.MaxOutput.Available || azurePolicy.MaxOutput.RuntimeFallback != 16384 {
		t.Fatalf("Azure max output = %+v, want missing metadata with runtime fallback", azurePolicy.MaxOutput)
	}
	if !strings.Contains(azurePolicy.AzureDetail(), "max_output_tokens=missing (runtime_fallback=16384)") {
		t.Fatalf("AzureDetail() = %q, want runtime fallback detail", azurePolicy.AzureDetail())
	}
}

func TestShouldStreamResponsesCatalogModelAndReason(t *testing.T) {
	for _, tt := range []struct {
		model string
		want  bool
	}{
		{model: "", want: true},
		{model: "gpt-5.3-codex", want: true},
		{model: "gpt-5.5-pro", want: false},
		{model: "gpt-5.5-pro-2026-05-01", want: false},
	} {
		if got := ShouldStreamResponsesCatalogModel(tt.model); got != tt.want {
			t.Fatalf("ShouldStreamResponsesCatalogModel(%q) = %t, want %t", tt.model, got, tt.want)
		}
	}

	if got, want := ResponsesStreamingReason("", true), "catalog_model is not resolved; Responses streaming defaults to enabled"; got != want {
		t.Fatalf("ResponsesStreamingReason(empty) = %q, want %q", got, want)
	}
	if got, want := ResponsesStreamingReason("gpt-5.5-pro", false), "catalog_model=gpt-5.5-pro disables Responses streaming"; got != want {
		t.Fatalf("ResponsesStreamingReason(disabled) = %q, want %q", got, want)
	}
}

func TestRouteDecisionReasonString(t *testing.T) {
	decision := RouteDecision{
		Route:   "responses_streaming",
		Reasons: []string{" deployment=corp uses Responses API ", "", "catalog_model=gpt-5.3-codex supports Responses streaming"},
	}
	if got, want := decision.ReasonString(), "deployment=corp uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming"; got != want {
		t.Fatalf("ReasonString() = %q, want %q", got, want)
	}
}
