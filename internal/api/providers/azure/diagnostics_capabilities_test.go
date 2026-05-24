package azure

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnosticCapabilitiesFromSnapshot(t *testing.T) {
	snapshot := providerdiag.CapabilitySnapshot{
		RequestModel:       "corp-codex-deployment",
		CatalogModel:       "gpt-5.3-codex",
		Route:              DiagnosticRouteResponsesStreaming,
		RouteReason:        "deployment=corp-codex-deployment uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming",
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

	got := diagnosticCapabilitiesFromSnapshot(snapshot)
	if got.Deployment != snapshot.RequestModel || got.CatalogModel != snapshot.CatalogModel || got.Route != snapshot.Route || got.RouteReason != snapshot.RouteReason {
		t.Fatalf("route/model projection = %+v, want snapshot values", got)
	}
	if !got.ResponsesAPI || !got.ResponsesStreaming || !got.FunctionCalling || !got.ImageInput {
		t.Fatalf("feature projection = %+v, want enabled features", got)
	}
	if !got.Retention.PreviousResponseID || !got.Retention.SessionPersistence {
		t.Fatalf("retention projection = %+v, want previous_response_id and session persistence", got.Retention)
	}
	if !got.ServerCompaction.RequestPayload || got.ServerCompaction.CompactThreshold != 272000 || !got.ServerCompaction.SkipLocalAutoCompression {
		t.Fatalf("server compaction projection = %+v, want snapshot values", got.ServerCompaction)
	}
	if got.ContextWindowTokens != 400000 || !got.ContextWindowKnown || got.MaxOutputTokens != 128000 || !got.MaxOutputTokensKnown || got.MaxOutputTokensSource != providerdiag.MaxOutputSourceCatalog || got.MaxOutputRuntimeFallback != 0 {
		t.Fatalf("catalog projection = %+v, want context and max output snapshot values", got)
	}
	if !got.Pricing.Available || got.Pricing.Detail != "pricing=input $1.75/M cached $0.175/M output $14.00/M" {
		t.Fatalf("pricing projection = %+v, want formatted pricing detail", got.Pricing)
	}
}

func TestDiagnosticCapabilitiesFromSnapshotKeepsAzureRuntimeFallback(t *testing.T) {
	got := diagnosticCapabilitiesFromSnapshot(providerdiag.CapabilitySnapshot{
		RequestModel: "corp-gpt52-pro-deployment",
		CatalogModel: "gpt-5.2-pro",
		MaxOutput: providerdiag.MaxOutputPolicy{
			Source:          providerdiag.MaxOutputSourceMissing,
			RuntimeFallback: 16384,
		},
	})

	if got.MaxOutputTokens != 16384 || got.MaxOutputTokensKnown || got.MaxOutputTokensSource != providerdiag.MaxOutputSourceRuntimeFallback || got.MaxOutputRuntimeFallback != 16384 {
		t.Fatalf("max output projection = %+v, want Azure runtime fallback fields", got)
	}
}

func TestDiagnose_CapabilitiesDoNotRequireAzureEndpointOrAuth(t *testing.T) {
	t.Setenv(baseURLEnv, "")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-codex-deployment",
		CatalogModel: "gpt-5.3-codex",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capabilities without endpoint/auth to succeed: %#v", report.Checks)
	}
	if _, ok := diagnosticCheckByName(report, "base_url"); ok {
		t.Fatalf("base_url check was added for capabilities-only report: %#v", report.Checks)
	}
	if _, ok := diagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for capabilities-only report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capabilities")
	}
	capabilities := report.Capabilities
	if !capabilities.ResponsesAPI || !capabilities.ResponsesStreaming {
		t.Fatalf("route capabilities = %+v, want Responses streaming", capabilities)
	}
	if !capabilities.FunctionCalling || !capabilities.ImageInput || !capabilities.Thinking {
		t.Fatalf("tool/image/thinking capabilities = %+v, want enabled", capabilities)
	}
	if !capabilities.Retention.PreviousResponseID || !capabilities.Retention.SessionPersistence {
		t.Fatalf("retention capabilities = %+v, want previous_response_id and session persistence", capabilities.Retention)
	}
	if !capabilities.ServerCompaction.Enabled || !capabilities.ServerCompaction.RequestPayload || capabilities.ServerCompaction.CompactThreshold <= 0 {
		t.Fatalf("server compaction capabilities = %+v, want request payload with compact_threshold", capabilities.ServerCompaction)
	}
	if capabilities.ContextWindowTokens != 400000 || !capabilities.ContextWindowKnown {
		t.Fatalf("context capability = %+v, want gpt-5.3-codex context window", capabilities)
	}
	if capabilities.MaxOutputTokens != 128000 || !capabilities.MaxOutputTokensKnown || capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceCatalog {
		t.Fatalf("max output capability = %+v, want catalog max output", capabilities)
	}
	if !capabilities.Pricing.Available {
		t.Fatalf("pricing capability = %+v, want available pricing", capabilities.Pricing)
	}
	if !hasDiagnosticCheck(report, "capabilities", DiagnosticStatusOK) {
		t.Fatalf("missing capabilities OK check: %#v", report.Checks)
	}
}

func TestDiagnose_CapabilitiesDoNotReportRetentionWhenRouteUnresolved(t *testing.T) {
	t.Setenv(baseURLEnv, "")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       &config.Config{},
		Capabilities: true,
	})
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capabilities")
	}
	if report.Capabilities.ResponsesAPI {
		t.Fatalf("ResponsesAPI = true, want false when deployment route is unresolved: %+v", report.Capabilities)
	}
	if report.Capabilities.Thinking {
		t.Fatalf("Thinking = true, want false when deployment route is unresolved: %+v", report.Capabilities)
	}
	if report.Capabilities.Retention.PreviousResponseID || report.Capabilities.Retention.SessionPersistence {
		t.Fatalf("retention capabilities = %+v, want no previous_response_id or session persistence without a resolved route", report.Capabilities.Retention)
	}
}

func TestDiagnose_RequiredThinkingFollowsResponsesReasoningConfig(t *testing.T) {
	t.Setenv(baseURLEnv, "")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")

	disabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Deployment:           "corp-gpt54-deployment",
		CatalogModel:         "gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if !disabled.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure when Responses reasoning config is omitted: %#v", disabled.Checks)
	}
	check, ok := diagnosticCheckByName(disabled, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail || !strings.Contains(check.Detail, "thinking=missing") {
		t.Fatalf("required_capability check = %#v, %v; want missing thinking", check, ok)
	}

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	enabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Deployment:           "corp-gpt54-deployment",
		CatalogModel:         "gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if enabled.HasFailures() {
		t.Fatalf("HasFailures() = true, want thinking required capability to pass when Responses reasoning config is enabled: %#v", enabled.Checks)
	}
	check, ok = diagnosticCheckByName(enabled, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK || !strings.Contains(check.Detail, "thinking=ok") {
		t.Fatalf("required_capability check = %#v, %v; want ok thinking", check, ok)
	}
}

func TestDiagnose_RequiredCapabilitiesDoNotRequireAzureEndpointOrAuth(t *testing.T) {
	t.Setenv(baseURLEnv, "")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Deployment:           "corp-codex-deployment",
		CatalogModel:         "gpt-5.3-codex",
		RequiredCapabilities: []string{"responses_api", "thinking", "previous_response_id", "server_compaction"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want local required capability check to pass without endpoint/auth: %#v", report.Checks)
	}
	if _, ok := diagnosticCheckByName(report, "base_url"); ok {
		t.Fatalf("base_url check was added for required capability report: %#v", report.Checks)
	}
	if _, ok := diagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for required capability report: %#v", report.Checks)
	}
	if report.Capabilities != nil {
		t.Fatalf("Capabilities = %#v, want nil without --capabilities", report.Capabilities)
	}
	check, ok := diagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("required_capability check = %#v, %v; want ok", check, ok)
	}
	for _, want := range []string{"responses_api=ok", "thinking=ok", "previous_response_id=ok", "server_compaction=ok"} {
		if !strings.Contains(check.Detail, want) {
			t.Fatalf("required_capability detail = %q, want %q", check.Detail, want)
		}
	}
}

func TestDiagnose_RequiredCapabilityFailsWhenMissing(t *testing.T) {
	t.Setenv(baseURLEnv, "")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Deployment:           "corp-gpt55-pro-deployment",
		CatalogModel:         "gpt-5.5-pro",
		RequiredCapabilities: []string{"responses_streaming"},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing required capability failure: %#v", report.Checks)
	}
	if _, ok := diagnosticCheckByName(report, "base_url"); ok {
		t.Fatalf("base_url check was added for required capability report: %#v", report.Checks)
	}
	if _, ok := diagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for required capability report: %#v", report.Checks)
	}
	check, ok := diagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("required_capability check = %#v, %v; want fail", check, ok)
	}
	if !strings.Contains(check.Detail, "responses_streaming=missing") {
		t.Fatalf("required_capability detail = %q, want missing streaming", check.Detail)
	}
}

func TestDiagnose_RequiredCapabilityStreamingUnknownWithoutResolvedCatalogModel(t *testing.T) {
	t.Setenv(baseURLEnv, "")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Deployment:           "corp-gpt55-pro-deployment",
		RequiredCapabilities: []string{"responses_streaming"},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want unresolved catalog required capability failure: %#v", report.Checks)
	}
	if report.CatalogModelSource != diagnosticCatalogModelSourceDeploymentFallback {
		t.Fatalf("CatalogModelSource = %q, want deployment fallback", report.CatalogModelSource)
	}
	if _, ok := diagnosticCheckByName(report, "base_url"); ok {
		t.Fatalf("base_url check was added for required capability report: %#v", report.Checks)
	}
	if _, ok := diagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for required capability report: %#v", report.Checks)
	}
	check, ok := diagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
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

func TestDiagnose_RequiredCapabilityStreamingPassesForKnownCatalogModelDeploymentFallback(t *testing.T) {
	t.Setenv(baseURLEnv, "")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Deployment:           "gpt-5.4",
		RequiredCapabilities: []string{"responses_streaming"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want known fallback catalog model to pass required capability: %#v", report.Checks)
	}
	if report.CatalogModelSource != diagnosticCatalogModelSourceDeploymentFallback {
		t.Fatalf("CatalogModelSource = %q, want deployment fallback", report.CatalogModelSource)
	}
	if report.Route != DiagnosticRouteResponsesStreaming {
		t.Fatalf("Route = %q, want responses streaming", report.Route)
	}
	check, ok := diagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("required_capability check = %#v, %v; want ok", check, ok)
	}
	if !strings.Contains(check.Detail, "responses_streaming=ok") {
		t.Fatalf("required_capability detail = %q, want ok streaming", check.Detail)
	}
}
