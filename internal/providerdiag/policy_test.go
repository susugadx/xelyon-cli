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

	groqPolicy := GroqCatalogPolicy(cfg, "meta-llama/llama-4-scout-17b-16e-instruct", "meta-llama/llama-4-scout-17b-16e-instruct")
	if got, want := groqPolicy.GroqDetail(), "catalog_model=meta-llama/llama-4-scout-17b-16e-instruct, context_window=131072, max_output_tokens=8192, pricing=input $0.11/M cached $0.110/M output $0.34/M"; got != want {
		t.Fatalf("GroqDetail() = %q, want %q", got, want)
	}

	deepSeekPolicy := DeepSeekCatalogPolicy(cfg, "deepseek-v4-flash", "deepseek-v4-flash")
	if got, want := deepSeekPolicy.DeepSeekDetail(), "catalog_model=deepseek-v4-flash, context_window=1000000, max_output_tokens=384000, pricing=input $0.14/M cached $0.003/M output $0.28/M"; got != want {
		t.Fatalf("DeepSeekDetail() = %q, want %q", got, want)
	}

	geminiPolicy := GeminiCatalogPolicy(cfg, "gemini-3.1-pro-preview-customtools", "gemini-3.1-pro-preview-customtools")
	if got, want := geminiPolicy.GeminiDetail(), "catalog_model=gemini-3.1-pro-preview-customtools, context_window=1000000, max_output_tokens=65536, pricing=input $2.00/M cached $0.200/M output $12.00/M"; got != want {
		t.Fatalf("GeminiDetail() = %q, want %q", got, want)
	}

	openRouterPolicy := OpenRouterCatalogPolicy(cfg, "openai/gpt-5.4", "openai/gpt-5.4")
	if got, want := openRouterPolicy.OpenRouterDetail(), "catalog_model=openai/gpt-5.4, context_window=1000000, max_output_tokens=64000, pricing=input $2.50/M cached $0.250/M output $15.00/M"; got != want {
		t.Fatalf("OpenRouterDetail() = %q, want %q", got, want)
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

func TestEvaluateRequiredCapabilities(t *testing.T) {
	snapshot := CapabilitySnapshot{
		ResponsesAPI:                   true,
		ResponsesStreaming:             false,
		ResponsesStreamingAvailability: KnownCapabilityAvailability(false),
		FunctionCalling:                true,
		ImageInput:                     true,
		Retention:                      NewRetentionSnapshot(true, true, true),
		ServerCompaction: ServerCompactionSnapshot{
			RequestPayload: true,
		},
	}

	check := EvaluateRequiredCapabilities(snapshot, []string{
		"responses-api",
		"function_calling",
		"responses_api",
		"responses_streaming",
		"unknown_capability",
	})
	if check.Satisfied() {
		t.Fatalf("Satisfied() = true, want false for missing/unknown capability: %+v", check.Results)
	}
	if !check.HasUnknown() {
		t.Fatalf("HasUnknown() = false, want true: %+v", check.Results)
	}
	if got, want := check.Detail(), "responses_api=ok, function_calling=ok, responses_streaming=missing, unknown_capability=unknown"; got != want {
		t.Fatalf("Detail() = %q, want %q", got, want)
	}
}

func TestEvaluateRequiredCapabilitiesSatisfied(t *testing.T) {
	snapshot := CapabilitySnapshot{
		ResponsesAPI:                   true,
		ResponsesStreaming:             true,
		ResponsesStreamingAvailability: KnownCapabilityAvailability(true),
		ChatCompletions:                true,
		FunctionCalling:                true,
		ImageInput:                     true,
		Retention:                      NewRetentionSnapshot(true, true, true),
		ServerCompaction: ServerCompactionSnapshot{
			RequestPayload: true,
		},
	}

	check := EvaluateRequiredCapabilities(snapshot, SupportedRequiredCapabilities())
	if !check.Satisfied() {
		t.Fatalf("Satisfied() = false, want true: detail=%q results=%+v", check.Detail(), check.Results)
	}
}

func TestEvaluateRequiredCapabilitiesUnknownAvailability(t *testing.T) {
	snapshot := CapabilitySnapshot{
		ResponsesAPI:                   true,
		ResponsesStreaming:             true,
		ResponsesStreamingAvailability: UnknownCapabilityAvailability(),
	}

	check := EvaluateRequiredCapabilities(snapshot, []string{RequiredCapabilityResponsesStreaming})
	if check.Satisfied() {
		t.Fatalf("Satisfied() = true, want false for unknown availability: %+v", check.Results)
	}
	if !check.HasUnknownAvailability() {
		t.Fatalf("HasUnknownAvailability() = false, want true: %+v", check.Results)
	}
	if got, want := check.Detail(), "responses_streaming=unknown"; got != want {
		t.Fatalf("Detail() = %q, want %q", got, want)
	}
}

func TestResponsesStreamingCapabilityAvailability(t *testing.T) {
	knownPolicy := CatalogPolicy{ContextWindowKnown: true}
	unknownPolicy := CatalogPolicy{ContextWindowKnown: false}

	for _, tt := range []struct {
		name               string
		responsesStreaming bool
		policy             CatalogPolicy
		want               CapabilityAvailability
	}{
		{
			name:               "streaming route with known catalog",
			responsesStreaming: true,
			policy:             knownPolicy,
			want:               KnownCapabilityAvailability(true),
		},
		{
			name:               "streaming route with unknown catalog",
			responsesStreaming: true,
			policy:             unknownPolicy,
			want:               UnknownCapabilityAvailability(),
		},
		{
			name:               "non streaming route with unknown catalog",
			responsesStreaming: false,
			policy:             unknownPolicy,
			want:               KnownCapabilityAvailability(false),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := ResponsesStreamingCapabilityAvailability(tt.responsesStreaming, tt.policy)
			if got != tt.want {
				t.Fatalf("ResponsesStreamingCapabilityAvailability() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRequiredCapabilityFailureSuggestion(t *testing.T) {
	missing := RequiredCapabilityCheck{Results: []RequiredCapabilityResult{{
		Name:   RequiredCapabilityResponsesStreaming,
		Status: RequiredCapabilityStatusMissing,
	}}}
	if got, want := RequiredCapabilityFailureSuggestion(missing, "model/configuration", ""), "Choose a model/configuration that provides the missing capability, or remove --require-capability"; got != want {
		t.Fatalf("RequiredCapabilityFailureSuggestion() = %q, want %q", got, want)
	}

	unknown := RequiredCapabilityCheck{Results: []RequiredCapabilityResult{{
		Name:   "unknown_capability",
		Status: RequiredCapabilityStatusUnknownName,
	}}}
	if got, want := RequiredCapabilityFailureSuggestion(unknown, "model/configuration", ""), "Use one of: "+SupportedRequiredCapabilitiesText(); got != want {
		t.Fatalf("RequiredCapabilityFailureSuggestion() = %q, want %q", got, want)
	}

	unknownAvailability := RequiredCapabilityCheck{Results: []RequiredCapabilityResult{{
		Name:   RequiredCapabilityResponsesStreaming,
		Status: RequiredCapabilityStatusUnknownAvailability,
	}}}
	if got, want := RequiredCapabilityFailureSuggestion(unknownAvailability, "model/configuration", "Set --catalog-model"), "Set --catalog-model"; got != want {
		t.Fatalf("RequiredCapabilityFailureSuggestion() = %q, want %q", got, want)
	}
}
