package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOpenRouter_DefaultModelUsesAnthropicMessagesRoute(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})

	if report.Provider != "openrouter" {
		t.Fatalf("Provider = %q, want openrouter", report.Provider)
	}
	if report.Model != "anthropic/claude-sonnet-4.6" {
		t.Fatalf("Model = %q, want default Claude model", report.Model)
	}
	if report.CatalogModel != "anthropic/claude-sonnet-4.6" {
		t.Fatalf("CatalogModel = %q, want default catalog model", report.CatalogModel)
	}
	if report.Route != DiagnosticRouteAnthropicMessages {
		t.Fatalf("Route = %q, want anthropic_messages", report.Route)
	}
	if !strings.HasSuffix(report.APIURL, "/v1/messages") {
		t.Fatalf("APIURL = %q, want Anthropic Skin messages endpoint", report.APIURL)
	}
	if report.UpstreamProvider != "anthropic" || report.UpstreamModel != "claude-sonnet-4.6" {
		t.Fatalf("upstream = %s/%s, want anthropic/claude-sonnet-4.6", report.UpstreamProvider, report.UpstreamModel)
	}
	if report.MaxOutputTokens != 64000 || report.ContextWindowTokens != 200000 {
		t.Fatalf("token policy = max %d context %d, want Claude metadata", report.MaxOutputTokens, report.ContextWindowTokens)
	}
	for _, name := range []string{"auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input"} {
		requireOpenRouterDiagnosticCheckStatus(t, report, name, DiagnosticStatusOK)
	}
}

func TestDiagnoseOpenRouter_OpenAIModelUsesChatCompletionsRoute(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-openrouter-gpt",
		CatalogModel: "openai/gpt-5.4",
	})

	if report.Model != "corp-openrouter-gpt" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "openai/gpt-5.4" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want chat_completions", report.Route)
	}
	if report.UpstreamProvider != "openai" || report.UpstreamModel != "gpt-5.4" {
		t.Fatalf("upstream = %s/%s, want openai/gpt-5.4", report.UpstreamProvider, report.UpstreamModel)
	}
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusOK)
	if !strings.Contains(catalogPolicy.Detail, "context_window=1000000") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=64000") ||
		!strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy detail = %q, want OpenRouter delegated policy detail", catalogPolicy.Detail)
	}
}

func TestDiagnoseOpenRouter_AliasCatalogClaudeWarnsRouteUsesRequestModel(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
		DefaultModel: "corp-claude-alias",
		CatalogModel: "anthropic/claude-sonnet-4.6",
	})

	report := Diagnose(context.Background(), DiagnosticOptions{Config: cfg})
	if report.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want alias to follow request model and use chat_completions", report.Route)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "route", DiagnosticStatusWarn)
}

func TestDiagnoseOpenRouter_NonOpenRouterCatalogModelDoesNotUseGlobalMetadata(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-openrouter-model",
		CatalogModel: "gpt-5.5",
	})

	if report.CatalogModel != "gpt-5.5" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit value in report", report.CatalogModel, report.CatalogModelSource)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want non-OpenRouter catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 128000 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI catalog max output ignored", report.MaxOutputTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "context_window=unknown") ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=unknown") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want no OpenAI token metadata", catalogPolicy.Detail)
	}
}

func TestDiagnoseOpenRouter_InvalidExplicitCatalogClearsConfiguredCatalogMetadata(t *testing.T) {
	var received openaicompat.ChatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOpenRouterChatCompletionsSSE(w,
			`{"choices":[{"delta":{"content":"fallback ok"}}]}`,
			`{"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":2}}`,
		)
	}))
	defer server.Close()

	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, server.URL)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
		DefaultModel: "corp-openrouter-model",
		CatalogModel: "openai/gpt-5.5",
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		CatalogModel: "bogus",
		RunSmoke:     true,
		TextSmoke:    true,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if report.Model != "corp-openrouter-model" || report.CatalogModel != "bogus" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("model/catalog = %q/%q (%s), want configured model with explicit invalid catalog", report.Model, report.CatalogModel, report.CatalogModelSource)
	}
	if received.Model != "corp-openrouter-model" {
		t.Fatalf("smoke model = %q, want configured request model", received.Model)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want invalid catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens != 64000 {
		t.Fatalf("MaxOutputTokens = %d, want OpenRouter provider fallback without stale catalog metadata", report.MaxOutputTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "catalog_model=bogus") ||
		strings.Contains(catalogPolicy.Detail, "1050000") ||
		strings.Contains(catalogPolicy.Detail, "128000") {
		t.Fatalf("catalog_policy detail = %q, want invalid catalog only and no stale OpenAI metadata", catalogPolicy.Detail)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "cost", DiagnosticStatusWarn)
	if report.Smoke == nil || !report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD != 0 {
		t.Fatalf("Smoke cost = %#v, want pricing unavailable without stale configured catalog", report.Smoke)
	}
}

func TestDiagnoseOpenRouter_RoutedModelWarnsMismatchedCatalogModel(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "openai/gpt-5.4",
	})

	if report.CatalogModel != "openai/gpt-5.4" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit value in report", report.CatalogModel, report.CatalogModelSource)
	}
	if report.UpstreamProvider != "anthropic" || report.UpstreamModel != "claude-sonnet-4.6" {
		t.Fatalf("upstream = %s/%s, want actual routed request model", report.UpstreamProvider, report.UpstreamModel)
	}
	if report.ContextWindowTokens != 200000 || report.MaxOutputTokens != 64000 {
		t.Fatalf("token policy = max %d context %d, want request model metadata", report.MaxOutputTokens, report.ContextWindowTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "requested_catalog_model=openai/gpt-5.4") ||
		!strings.Contains(catalogPolicy.Detail, "policy_catalog_model=anthropic/claude-sonnet-4.6") ||
		!strings.Contains(catalogPolicy.Detail, "context_window=200000") ||
		!strings.Contains(catalogPolicy.Detail, "pricing=input $3.00/M") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy detail = %q, want request model policy and no OpenAI pricing", catalogPolicy.Detail)
	}
}

func TestDiagnoseOpenRouter_RoutedUnknownModelRejectsCrossOwnerCatalogModel(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "sk-or-test")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-future-prod",
		CatalogModel: "openai/gpt-5.4",
	})

	if report.UpstreamProvider != "anthropic" || report.UpstreamModel != "claude-future-prod" {
		t.Fatalf("upstream = %s/%s, want request model owner", report.UpstreamProvider, report.UpstreamModel)
	}
	if report.ContextWindowTokens != 0 {
		t.Fatalf("ContextWindowTokens = %d, want cross-owner catalog metadata ignored", report.ContextWindowTokens)
	}
	if report.MaxOutputTokens == 32768 {
		t.Fatalf("MaxOutputTokens = %d, want OpenAI catalog max output ignored", report.MaxOutputTokens)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_model", DiagnosticStatusWarn)
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "model owner=anthropic but catalog_model owner=openai") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") ||
		strings.Contains(catalogPolicy.Detail, "context_window=1000000") {
		t.Fatalf("catalog_policy detail = %q, want cross-owner catalog ignored", catalogPolicy.Detail)
	}
}
