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

	kimiPolicy := KimiCatalogPolicy(cfg, "kimi-k2.6", "kimi-k2.6")
	if got, want := kimiPolicy.KimiDetail(), "catalog_model=kimi-k2.6, context_window=256000, max_output_tokens=32768, pricing=input $0.95/M cached $0.160/M output $4.00/M"; got != want {
		t.Fatalf("KimiDetail() = %q, want %q", got, want)
	}

	claudePolicy := ClaudeCatalogPolicy(cfg, "claude-sonnet-4-6", "claude-sonnet-4-6")
	if got, want := claudePolicy.ClaudeDetail(), "catalog_model=claude-sonnet-4-6, context_window=200000, max_output_tokens=64000, pricing=input $3.00/M cached $0.300/M output $15.00/M"; got != want {
		t.Fatalf("ClaudeDetail() = %q, want %q", got, want)
	}

	openRouterPolicy := OpenRouterCatalogPolicy(cfg, "openai/gpt-5.4", "openai/gpt-5.4")
	if got, want := openRouterPolicy.OpenRouterDetail(), "catalog_model=openai/gpt-5.4, context_window=1000000, max_output_tokens=64000, pricing=input $2.50/M cached $0.250/M output $15.00/M"; got != want {
		t.Fatalf("OpenRouterDetail() = %q, want %q", got, want)
	}

	ollamaPolicy := OllamaCatalogPolicy(cfg, "qwen2.5-coder:7b", "qwen2.5-coder:7b")
	if got, want := ollamaPolicy.OllamaDetail(), "catalog_model=qwen2.5-coder:7b, context_window=32768, max_output_tokens=4096, pricing=input $0.00/M cached $0.000/M output $0.00/M"; got != want {
		t.Fatalf("OllamaDetail() = %q, want %q", got, want)
	}
}

func TestMaxOutputPolicyPreservesOpenAIAndAzureFallbackDifference(t *testing.T) {
	cfg := config.DefaultConfig()

	openAIPolicy := OpenAICatalogPolicy(cfg, "gpt-5.2-pro", "gpt-5.2-pro")
	if !openAIPolicy.MaxOutput.Available || openAIPolicy.MaxOutput.Source != MaxOutputSourceProviderDefault || openAIPolicy.MaxOutput.Tokens != 16384 {
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

func TestCatalogPolicyIgnoresCrossProviderGlobalMetadata(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		model        string
		catalogModel string
		policy       func(*config.Config, string, string) CatalogPolicy
		maxOutput    func(*config.Config, string, string) MaxOutputPolicy
		detail       func(CatalogPolicy) string
		unwanted     []string
		wantDetail   string
	}{
		{
			name:         "gemini",
			providerName: "Gemini",
			model:        "corp-gemini-model",
			catalogModel: "gpt-5.5",
			policy:       GeminiCatalogPolicy,
			maxOutput:    GeminiMaxOutputPolicy,
			detail:       CatalogPolicy.GeminiDetail,
			unwanted:     []string{"128000", "1050000", "gpt-5.5-pro"},
			wantDetail:   "catalog_model=gpt-5.5, context_window=unknown, max_output_tokens=unknown, pricing=unavailable",
		},
		{
			name:         "kimi",
			providerName: "Kimi",
			model:        "corp-kimi-model",
			catalogModel: "gpt-5.5",
			policy:       KimiCatalogPolicy,
			maxOutput:    KimiMaxOutputPolicy,
			detail:       CatalogPolicy.KimiDetail,
			unwanted:     []string{"128000", "1050000", "gpt-5.5-pro"},
			wantDetail:   "catalog_model=gpt-5.5, context_window=unknown, max_output_tokens=unknown, pricing=unavailable",
		},
		{
			name:         "claude",
			providerName: "Claude",
			model:        "corp-claude-model",
			catalogModel: "gpt-5.5",
			policy:       ClaudeCatalogPolicy,
			maxOutput:    ClaudeMaxOutputPolicy,
			detail:       CatalogPolicy.ClaudeDetail,
			unwanted:     []string{"128000", "1050000", "gpt-5.5-pro"},
			wantDetail:   "catalog_model=gpt-5.5, context_window=unknown, max_output_tokens=unknown, pricing=unavailable",
		},
		{
			name:         "deepseek",
			providerName: "DeepSeek",
			model:        "corp-deepseek-model",
			catalogModel: "gpt-5.4",
			policy:       DeepSeekCatalogPolicy,
			maxOutput:    DeepSeekMaxOutputPolicy,
			detail:       CatalogPolicy.DeepSeekDetail,
			unwanted:     []string{"1000000", "64000", "pricing=input $2.50/M"},
			wantDetail:   "catalog_model=gpt-5.4, context_window=unknown, max_output_tokens=unknown, pricing=unavailable",
		},
		{
			name:         "groq",
			providerName: "Groq",
			model:        "corp-groq-model",
			catalogModel: "gpt-5.4",
			policy:       GroqCatalogPolicy,
			maxOutput:    GroqMaxOutputPolicy,
			detail:       CatalogPolicy.GroqDetail,
			unwanted:     []string{"1000000", "64000", "pricing=input $2.50/M"},
			wantDetail:   "catalog_model=gpt-5.4, context_window=unknown, max_output_tokens=unknown, pricing=unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			policy := tt.policy(cfg, tt.model, tt.catalogModel)
			if policy.ContextWindowKnown || policy.ContextWindowTokens != 0 {
				t.Fatalf("%s context window = %d known=%t, want unknown for cross-provider catalog", tt.providerName, policy.ContextWindowTokens, policy.ContextWindowKnown)
			}
			if policy.MaxOutput.Available || policy.MaxOutput.Tokens != 0 || policy.MaxOutput.Source != MaxOutputSourceMissing {
				t.Fatalf("%s max output = %+v, want missing for cross-provider catalog", tt.providerName, policy.MaxOutput)
			}
			if !policy.Pricing.PricingUnavailable {
				t.Fatalf("%s pricing = %+v, want unavailable for cross-provider catalog", tt.providerName, policy.Pricing)
			}

			detail := tt.detail(policy)
			for _, unwanted := range tt.unwanted {
				if strings.Contains(detail, unwanted) {
					t.Fatalf("%s detail = %q, should not contain cross-provider metadata %q", tt.providerName, detail, unwanted)
				}
			}
			if detail != tt.wantDetail {
				t.Fatalf("%s detail = %q, want %q", tt.providerName, detail, tt.wantDetail)
			}

			maxOutput := tt.maxOutput(cfg, tt.model, tt.catalogModel)
			if maxOutput.Available || maxOutput.Source != MaxOutputSourceMissing {
				t.Fatalf("%s max output policy = %+v, want missing", tt.providerName, maxOutput)
			}
		})
	}
}

func TestCatalogPolicyPreservesMaxOutputOverrideForUntrustedCatalog(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		model        string
		catalogModel string
		policy       func(*config.Config, string, string) CatalogPolicy
		maxOutput    func(*config.Config, string, string) MaxOutputPolicy
		detail       func(CatalogPolicy) string
	}{
		{
			name:         "gemini",
			provider:     "gemini",
			model:        "corp-gemini-model",
			catalogModel: "gpt-5.5",
			policy:       GeminiCatalogPolicy,
			maxOutput:    GeminiMaxOutputPolicy,
			detail:       CatalogPolicy.GeminiDetail,
		},
		{
			name:         "kimi",
			provider:     "kimi",
			model:        "corp-kimi-model",
			catalogModel: "gpt-5.5",
			policy:       KimiCatalogPolicy,
			maxOutput:    KimiMaxOutputPolicy,
			detail:       CatalogPolicy.KimiDetail,
		},
		{
			name:         "claude",
			provider:     "claude",
			model:        "corp-claude-model",
			catalogModel: "gpt-5.5",
			policy:       ClaudeCatalogPolicy,
			maxOutput:    ClaudeMaxOutputPolicy,
			detail:       CatalogPolicy.ClaudeDetail,
		},
		{
			name:         "deepseek",
			provider:     "deepseek",
			model:        "corp-deepseek-model",
			catalogModel: "gpt-5.4",
			policy:       DeepSeekCatalogPolicy,
			maxOutput:    DeepSeekMaxOutputPolicy,
			detail:       CatalogPolicy.DeepSeekDetail,
		},
		{
			name:         "groq",
			provider:     "groq",
			model:        "corp-groq-model",
			catalogModel: "gpt-5.4",
			policy:       GroqCatalogPolicy,
			maxOutput:    GroqMaxOutputPolicy,
			detail:       CatalogPolicy.GroqDetail,
		},
		{
			name:         "openrouter",
			provider:     "openrouter",
			model:        "anthropic/claude-future-prod",
			catalogModel: "",
			policy:       OpenRouterCatalogPolicy,
			maxOutput:    OpenRouterMaxOutputPolicy,
			detail:       CatalogPolicy.OpenRouterDetail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SetProviderModelConfig(tt.provider, config.ProviderModelConfig{
				ModelOverrides: map[string]config.ModelOverride{
					tt.model: {MaxOutputTokens: 7777},
				},
			})

			policy := tt.policy(cfg, tt.model, tt.catalogModel)
			if policy.ContextWindowKnown || policy.ContextWindowTokens != 0 {
				t.Fatalf("context window = %d known=%t, want unknown for untrusted catalog", policy.ContextWindowTokens, policy.ContextWindowKnown)
			}
			if !policy.MaxOutput.Available || policy.MaxOutput.Tokens != 7777 || policy.MaxOutput.Source != MaxOutputSourceModelOverrides {
				t.Fatalf("catalog policy max output = %+v, want explicit override", policy.MaxOutput)
			}
			if !policy.Pricing.PricingUnavailable {
				t.Fatalf("pricing = %+v, want unavailable for untrusted catalog", policy.Pricing)
			}
			detail := tt.detail(policy)
			if !strings.Contains(detail, "max_output_tokens=7777") ||
				strings.Contains(detail, "context_window=1000000") ||
				strings.Contains(detail, "pricing=input $2.50/M") {
				t.Fatalf("policy detail = %q, want override without untrusted catalog metadata", detail)
			}

			maxOutput := tt.maxOutput(cfg, tt.model, tt.catalogModel)
			if !maxOutput.Available || maxOutput.Tokens != 7777 || maxOutput.Source != MaxOutputSourceModelOverrides {
				t.Fatalf("max output policy = %+v, want explicit override", maxOutput)
			}
		})
	}
}

func TestSimpleProviderCatalogPolicyMatchesGroqWrapper(t *testing.T) {
	cfg := config.DefaultConfig()
	model := "meta-llama/llama-4-scout-17b-16e-instruct"

	got := SimpleProviderCatalogPolicy(cfg, "groq", model, model)
	want := GroqCatalogPolicy(cfg, model, model)
	if got != want {
		t.Fatalf("SimpleProviderCatalogPolicy(groq) = %#v, want wrapper %#v", got, want)
	}
}

func TestRuntimeMaxOutputTokensStaysSeparateFromUntrustedCatalogPolicy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
		DefaultModel: "corp-openrouter-model",
		CatalogModel: "openai/gpt-5.5",
	})
	policyCfg := ProviderDiagnosticPolicyConfig(cfg, ProviderDiagnosticPolicyConfigOptions{
		Provider:     "openrouter",
		Model:        "corp-openrouter-model",
		CatalogModel: "bogus",
	})

	policy := OpenRouterCatalogPolicy(policyCfg, "corp-openrouter-model", "")
	if policy.MaxOutput.Available || policy.MaxOutput.Tokens != 0 {
		t.Fatalf("catalog policy max output = %+v, want unknown without trusted catalog", policy.MaxOutput)
	}
	if got := RuntimeMaxOutputTokens(policyCfg, "openrouter", "corp-openrouter-model"); got != 64000 {
		t.Fatalf("RuntimeMaxOutputTokens() = %d, want provider runtime fallback", got)
	}
}

func TestBedrockCatalogPolicyUsesProviderdiagTrustBoundary(t *testing.T) {
	cfg := config.DefaultConfig()

	policy := BedrockCatalogPolicy(cfg, "amazon.nova-pro-v1:0", "gpt-5.4", MaxOutputPolicy{
		Tokens:    5000,
		Source:    MaxOutputSourceCatalog,
		Available: true,
	})
	if policy.ContextWindowKnown || policy.ContextWindowTokens != 0 {
		t.Fatalf("Bedrock context window = %d known=%t, want unknown for non-Bedrock catalog", policy.ContextWindowTokens, policy.ContextWindowKnown)
	}
	if !policy.MaxOutput.Available || policy.MaxOutput.Tokens != 5000 || policy.MaxOutput.Source != MaxOutputSourceCatalog {
		t.Fatalf("Bedrock max output = %+v, want provider-supplied request-model fallback", policy.MaxOutput)
	}
	if !policy.Pricing.PricingUnavailable {
		t.Fatalf("Bedrock pricing = %+v, want unavailable for non-Bedrock catalog", policy.Pricing)
	}

	trusted := BedrockCatalogPolicy(cfg, "global.anthropic.claude-sonnet-4-6", "global.anthropic.claude-sonnet-4-6", MaxOutputPolicy{
		Tokens:    64000,
		Source:    MaxOutputSourceCatalog,
		Available: true,
	})
	if !trusted.ContextWindowKnown || trusted.ContextWindowTokens == 0 {
		t.Fatalf("trusted Bedrock context window = %d known=%t, want known metadata", trusted.ContextWindowTokens, trusted.ContextWindowKnown)
	}
	if trusted.Pricing.PricingUnavailable {
		t.Fatalf("trusted Bedrock pricing = %+v, want pricing metadata", trusted.Pricing)
	}
}

func TestBedrockMaxOutputPoliciesKeepRuntimeSourcesInProviderdiag(t *testing.T) {
	cfg := config.DefaultConfig()

	untrusted := BedrockUntrustedCatalogMaxOutputPolicy(cfg, "amazon.nova-pro-v1:0")
	if !untrusted.Available || untrusted.Tokens != 5000 || untrusted.Source != MaxOutputSourceCatalog {
		t.Fatalf("BedrockUntrustedCatalogMaxOutputPolicy() = %+v, want request-model catalog fallback", untrusted)
	}

	cfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel:    "anthropic.claude-custom-v1:0",
		MaxOutputTokens: 9999,
	})
	claude := BedrockClaudeMaxOutputPolicy(cfg, "anthropic.claude-custom-v1:0")
	if !claude.Available || claude.Tokens != 9999 || claude.Source != MaxOutputSourceProviderDefault {
		t.Fatalf("BedrockClaudeMaxOutputPolicy() = %+v, want provider default", claude)
	}

	cfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel:    "global.anthropic.claude-sonnet-4-6",
		CatalogModel:    "gpt-5.5",
		MaxOutputTokens: 9999,
	})
	claudeCatalog := BedrockClaudeMaxOutputPolicy(cfg, "global.anthropic.claude-sonnet-4-6")
	if !claudeCatalog.Available || claudeCatalog.Tokens != 64000 || claudeCatalog.Source != MaxOutputSourceCatalog {
		t.Fatalf("BedrockClaudeMaxOutputPolicy(cross-provider catalog) = %+v, want Bedrock request-model catalog max output", claudeCatalog)
	}

	cfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		ModelOverrides: map[string]config.ModelOverride{
			"anthropic.claude-custom-v1:0": {MaxOutputTokens: 2048},
		},
	})
	override := BedrockClaudeMaxOutputPolicy(cfg, "anthropic.claude-custom-v1:0")
	if !override.Available || override.Tokens != 2048 || override.Source != MaxOutputSourceModelOverrides {
		t.Fatalf("BedrockClaudeMaxOutputPolicy(override) = %+v, want model override", override)
	}
}

func TestOllamaCatalogPolicyIgnoresNonOllamaGlobalMetadata(t *testing.T) {
	cfg := config.DefaultConfig()

	policy := OllamaCatalogPolicy(cfg, "corp-ollama-model", "gpt-5.5")
	if policy.ContextWindowKnown || policy.ContextWindowTokens != 0 {
		t.Fatalf("Ollama context window = %d known=%t, want unknown for non-Ollama catalog", policy.ContextWindowTokens, policy.ContextWindowKnown)
	}
	if !policy.MaxOutput.Available || policy.MaxOutput.Tokens != 4096 || policy.MaxOutput.Source != MaxOutputSourceProviderDefault {
		t.Fatalf("Ollama max output = %+v, want provider default for non-Ollama catalog", policy.MaxOutput)
	}
	if policy.Pricing.PricingUnavailable {
		t.Fatalf("Ollama pricing = %+v, want local zero pricing", policy.Pricing)
	}
	detail := policy.OllamaDetail()
	for _, unwanted := range []string{"128000", "1050000", "gpt-5.5-pro"} {
		if strings.Contains(detail, unwanted) {
			t.Fatalf("OllamaDetail() = %q, should not contain non-Ollama metadata %q", detail, unwanted)
		}
	}
	if got, want := detail, "catalog_model=gpt-5.5, context_window=unknown, max_output_tokens=4096, pricing=input $0.00/M cached $0.000/M output $0.00/M"; got != want {
		t.Fatalf("OllamaDetail() = %q, want %q", got, want)
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
