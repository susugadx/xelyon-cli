package openai

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestOpenAIDiagnosticCapabilitiesFromSnapshot(t *testing.T) {
	snapshot := providerdiag.CapabilitySnapshot{
		RequestModel:       "corp-openai-deployment",
		CatalogModel:       "gpt-5.3-codex",
		Route:              DiagnosticRouteResponsesStreaming,
		RouteReason:        "model=corp-openai-deployment uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming",
		ResponsesAPI:       true,
		ResponsesStreaming: true,
		FunctionCalling:    true,
		ImageInput:         providerdiag.KnownCapabilityAvailability(true),
		Retention:          providerdiag.NewRetentionSnapshot(true, true, true),
		ServerCompaction: providerdiag.ServerCompactionSnapshot{
			Enabled:                  true,
			RequestPayload:           true,
			CompactThreshold:         272000,
			LocalFallback:            true,
			SkipLocalAutoCompression: true,
			Detail:                   "context_management.compaction would be sent with previous_response_id",
		},
		ContextWindowTokens: 400000,
		ContextWindowKnown:  true,
		MaxOutput: providerdiag.MaxOutputPolicy{
			Tokens:    128000,
			Source:    providerdiag.MaxOutputSourceCatalog,
			Available: true,
		},
		Pricing: cost.PricingInfo{
			InputCostPerM:       1.75,
			CachedInputCostPerM: 0.175,
			OutputCostPerM:      14,
		},
	}

	got := openAIDiagnosticCapabilitiesFromSnapshot(snapshot)
	if got.Model != snapshot.RequestModel || got.CatalogModel != snapshot.CatalogModel || got.Route != snapshot.Route || got.RouteReason != snapshot.RouteReason {
		t.Fatalf("route/model projection = %+v, want snapshot values", got)
	}
	if !got.ResponsesAPI || !got.ResponsesStreaming || got.ChatCompletions {
		t.Fatalf("route capability projection = %+v, want Responses streaming only", got)
	}
	if !got.FunctionCalling || !got.ImageInput || !got.Retention.PreviousResponseID || !got.Retention.SessionPersistence {
		t.Fatalf("feature projection = %+v retention=%+v, want enabled features", got, got.Retention)
	}
	if !got.ServerCompaction.RequestPayload || got.ServerCompaction.CompactThreshold != 272000 || !got.ServerCompaction.SkipLocalAutoCompression {
		t.Fatalf("server compaction projection = %+v, want snapshot values", got.ServerCompaction)
	}
	if got.ContextWindowTokens != 400000 || !got.ContextWindowKnown || got.MaxOutputTokens != 128000 || !got.MaxOutputTokensKnown || got.MaxOutputTokensSource != providerdiag.MaxOutputSourceCatalog {
		t.Fatalf("catalog projection = %+v, want context and max output snapshot values", got)
	}
	if !got.Pricing.Available || got.Pricing.Detail != "pricing=input $1.75/M cached $0.175/M output $14.00/M" {
		t.Fatalf("pricing projection = %+v, want formatted pricing detail", got.Pricing)
	}
}

func TestDiagnoseOpenAI_CapabilitiesDoNotRequireAPIKey(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "://bad")
	t.Setenv(openAIResponsesURLEnv, "://bad")
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-openai-deployment",
		CatalogModel: "gpt-5.4",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capabilities without API key to succeed: %#v", report.Checks)
	}
	if _, ok := openAIDiagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for capabilities-only report: %#v", report.Checks)
	}
	for _, checkName := range []string{"api_url", "api_url_path", "responses_url", "responses_url_path"} {
		if _, ok := openAIDiagnosticCheckByName(report, checkName); ok {
			t.Fatalf("%s check was added for capabilities-only report: %#v", checkName, report.Checks)
		}
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capabilities")
	}
	capabilities := report.Capabilities
	if !capabilities.ResponsesAPI || !capabilities.ResponsesStreaming || capabilities.ChatCompletions {
		t.Fatalf("route capabilities = %+v, want Responses streaming only", capabilities)
	}
	if !capabilities.FunctionCalling || !capabilities.ImageInput || !capabilities.WebSearch {
		t.Fatalf("tool/image/web_search capabilities = %+v, want enabled", capabilities)
	}
	if capabilities.Thinking {
		t.Fatalf("Thinking = true, want false when Responses request reasoning config is not enabled: %+v", capabilities)
	}
	if !capabilities.Retention.PreviousResponseID || !capabilities.Retention.SessionPersistence {
		t.Fatalf("retention capabilities = %+v, want previous_response_id and session persistence", capabilities.Retention)
	}
	if !capabilities.ServerCompaction.Enabled || !capabilities.ServerCompaction.RequestPayload || capabilities.ServerCompaction.CompactThreshold <= 0 {
		t.Fatalf("server compaction capabilities = %+v, want request payload with compact_threshold", capabilities.ServerCompaction)
	}
	if capabilities.ContextWindowTokens != 1000000 || !capabilities.ContextWindowKnown {
		t.Fatalf("context capability = %+v, want gpt-5.4 context window", capabilities)
	}
	if capabilities.MaxOutputTokens != 16384 || !capabilities.MaxOutputTokensKnown || capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceProviderDefault {
		t.Fatalf("max output capability = %+v, want provider default max output", capabilities)
	}
	if !capabilities.Pricing.Available {
		t.Fatalf("pricing capability = %+v, want available pricing", capabilities.Pricing)
	}
	if !hasOpenAIDiagnosticCheck(report, "capabilities", DiagnosticStatusOK) {
		t.Fatalf("missing capabilities OK check: %#v", report.Checks)
	}
}

func TestDiagnoseOpenAI_CapabilitiesShowChatCompletionsLimitations(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-4",
		CatalogModel: "gpt-4",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capabilities without API key to succeed: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capabilities")
	}
	if report.Capabilities.ResponsesAPI || !report.Capabilities.ChatCompletions {
		t.Fatalf("route capabilities = %+v, want Chat Completions route", report.Capabilities)
	}
	if report.Capabilities.WebSearch {
		t.Fatalf("WebSearch = true, want false for Chat Completions route")
	}
	if report.Capabilities.Thinking {
		t.Fatalf("Thinking = true, want false for Chat Completions route")
	}
	if report.Capabilities.Retention.PreviousResponseID {
		t.Fatalf("retention capabilities = %+v, want no previous_response_id on Chat Completions", report.Capabilities.Retention)
	}
	if report.Capabilities.Retention.SessionPersistence {
		t.Fatalf("retention capabilities = %+v, want no session persistence on Chat Completions", report.Capabilities.Retention)
	}
	if report.Capabilities.ServerCompaction.RequestPayload {
		t.Fatalf("server compaction capabilities = %+v, want no request payload on Chat Completions", report.Capabilities.ServerCompaction)
	}
}

func TestDiagnoseOpenAI_RequiredWebSearchFollowsResponsesRuntimeGate(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")

	chatCompletions := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "gpt-4",
		CatalogModel:         "gpt-4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityWebSearch},
	})
	if !chatCompletions.HasFailures() {
		t.Fatalf("HasFailures() = false, want web_search required capability failure: %#v", chatCompletions.Checks)
	}
	if _, ok := openAIDiagnosticCheckByName(chatCompletions, "auth"); ok {
		t.Fatalf("auth check was added for required capability report: %#v", chatCompletions.Checks)
	}
	check, ok := openAIDiagnosticCheckByName(chatCompletions, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("required_capability check = %#v, %v; want fail", check, ok)
	}
	if !strings.Contains(check.Detail, "web_search=missing") {
		t.Fatalf("required_capability detail = %q, want missing web_search", check.Detail)
	}

	responses := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-openai-deployment",
		CatalogModel:         "gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityWebSearch},
	})
	if responses.HasFailures() {
		t.Fatalf("HasFailures() = true, want web_search required capability to pass for Responses route: %#v", responses.Checks)
	}
	check, ok = openAIDiagnosticCheckByName(responses, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("required_capability check = %#v, %v; want ok", check, ok)
	}
	if !strings.Contains(check.Detail, "web_search=ok") {
		t.Fatalf("required_capability detail = %q, want ok web_search", check.Detail)
	}
}

func TestDiagnoseOpenAI_RequiredCapabilitiesDoNotRequireAPIKey(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "://bad")
	t.Setenv(openAIResponsesURLEnv, "://bad")
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                "corp-openai-deployment",
		CatalogModel:         "gpt-5.4",
		RequiredCapabilities: []string{"responses_api", "thinking", "previous_response_id", "server_compaction"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want local required capability check to pass without API key: %#v", report.Checks)
	}
	if _, ok := openAIDiagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for required capability report: %#v", report.Checks)
	}
	for _, checkName := range []string{"api_url", "api_url_path", "responses_url", "responses_url_path"} {
		if _, ok := openAIDiagnosticCheckByName(report, checkName); ok {
			t.Fatalf("%s check was added for required capability report: %#v", checkName, report.Checks)
		}
	}
	if report.Capabilities != nil {
		t.Fatalf("Capabilities = %#v, want nil without --capabilities", report.Capabilities)
	}
	check, ok := openAIDiagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("required_capability check = %#v, %v; want ok", check, ok)
	}
	for _, want := range []string{"responses_api=ok", "thinking=ok", "previous_response_id=ok", "server_compaction=ok"} {
		if !strings.Contains(check.Detail, want) {
			t.Fatalf("required_capability detail = %q, want %q", check.Detail, want)
		}
	}
}

func TestDiagnoseOpenAI_RequiredThinkingFollowsResponsesRuntimeGate(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")

	chatCompletions := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "gpt-4",
		CatalogModel:         "gpt-4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if !chatCompletions.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure on Chat Completions route: %#v", chatCompletions.Checks)
	}
	check, ok := openAIDiagnosticCheckByName(chatCompletions, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail || !strings.Contains(check.Detail, "thinking=missing") {
		t.Fatalf("required_capability check = %#v, %v; want missing thinking", check, ok)
	}

	responsesDisabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-openai-deployment",
		CatalogModel:         "gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if !responsesDisabled.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure when Responses reasoning config is omitted: %#v", responsesDisabled.Checks)
	}
	check, ok = openAIDiagnosticCheckByName(responsesDisabled, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail || !strings.Contains(check.Detail, "thinking=missing") {
		t.Fatalf("required_capability check = %#v, %v; want missing thinking for disabled Responses reasoning", check, ok)
	}

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	responses := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                "corp-openai-deployment",
		CatalogModel:         "gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if responses.HasFailures() {
		t.Fatalf("HasFailures() = true, want thinking required capability to pass when Responses reasoning config is enabled: %#v", responses.Checks)
	}
	check, ok = openAIDiagnosticCheckByName(responses, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK || !strings.Contains(check.Detail, "thinking=ok") {
		t.Fatalf("required_capability check = %#v, %v; want ok thinking", check, ok)
	}

	codex := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-openai-codex",
		CatalogModel:         "gpt-5.3-codex",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if codex.HasFailures() {
		t.Fatalf("HasFailures() = true, want Codex Responses reasoning fallback to satisfy thinking: %#v", codex.Checks)
	}
	check, ok = openAIDiagnosticCheckByName(codex, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK || !strings.Contains(check.Detail, "thinking=ok") {
		t.Fatalf("required_capability check = %#v, %v; want ok thinking for Codex fallback", check, ok)
	}
}

func TestDiagnoseOpenAI_RequiredCapabilityFailsWhenMissing(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "gpt-5.5-pro",
		CatalogModel:         "gpt-5.5-pro",
		RequiredCapabilities: []string{"responses_streaming"},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing required capability failure: %#v", report.Checks)
	}
	if _, ok := openAIDiagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for required capability report: %#v", report.Checks)
	}
	check, ok := openAIDiagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("required_capability check = %#v, %v; want fail", check, ok)
	}
	if !strings.Contains(check.Detail, "responses_streaming=missing") {
		t.Fatalf("required_capability detail = %q, want missing streaming", check.Detail)
	}
}

func TestDiagnoseOpenAI_RequiredCapabilityStreamingUnknownForCustomResponsesModelWithoutCatalogModel(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(openAIAPIURLEnv, "")
	t.Setenv(openAIResponsesURLEnv, "")

	cfg := config.DefaultConfig()
	cfg.OpenAI.ResponsesAPIModels = append(cfg.OpenAI.ResponsesAPIModels, "corp-gpt55-pro-alias")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                "corp-gpt55-pro-alias",
		RequiredCapabilities: []string{"responses_streaming"},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want unknown required capability failure: %#v", report.Checks)
	}
	if _, ok := openAIDiagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for required capability report: %#v", report.Checks)
	}
	modelCheck, ok := openAIDiagnosticCheckByName(report, "model")
	if !ok || modelCheck.Status != DiagnosticStatusWarn {
		t.Fatalf("model check = %#v, %v; want unresolved catalog warning", modelCheck, ok)
	}
	check, ok := openAIDiagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("required_capability check = %#v, %v; want fail", check, ok)
	}
	if !strings.Contains(check.Detail, "responses_streaming=unknown") {
		t.Fatalf("required_capability detail = %q, want unknown streaming", check.Detail)
	}
	if !strings.Contains(check.Suggestion, "--catalog-model") {
		t.Fatalf("required_capability suggestion = %q, want catalog model guidance", check.Suggestion)
	}
}
