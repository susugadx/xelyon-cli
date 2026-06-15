package openrouter

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnoseOpenRouter_CapabilitiesDoNotRequireAPIKey(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "://bad")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-openrouter-gpt",
		CatalogModel:         "openai/gpt-5.4",
		Capabilities:         true,
		RequiredCapabilities: []string{"chat_completions", "function_calling"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capability-only diagnostics without API key to pass: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckAbsent(t, report, "auth")
	requireOpenRouterDiagnosticCheckAbsent(t, report, "endpoint")
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.ChatCompletions || !report.Capabilities.FunctionCalling {
		t.Fatalf("Capabilities = %+v, want chat/function enabled", report.Capabilities)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
}

func TestDiagnoseOpenRouter_RequiredCapabilityOnlyDoesNotValidateEndpoint(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "://bad")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-openrouter-gpt",
		CatalogModel:         "openai/gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityFunctionCalling},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want required-capability-only diagnostics to skip auth and endpoint checks: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckAbsent(t, report, "auth")
	requireOpenRouterDiagnosticCheckAbsent(t, report, "endpoint")
	requireOpenRouterDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
}

func TestDiagnoseOpenRouter_RequiredImageInputUsesTrustedModelCapability(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "://bad")

	deepSeek := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "deepseek/deepseek-v4-flash",
		CatalogModel:         "deepseek/deepseek-v4-flash",
		Capabilities:         true,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput},
	})
	if !deepSeek.HasFailures() {
		t.Fatalf("HasFailures() = false, want non-image OpenRouter model to fail image_input requirement: %#v", deepSeek.Checks)
	}
	if deepSeek.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if deepSeek.Capabilities.ImageInput || !deepSeek.Capabilities.ImageInputKnown {
		t.Fatalf("OpenRouter DeepSeek image capability = %t known=%t, want known missing", deepSeek.Capabilities.ImageInput, deepSeek.Capabilities.ImageInputKnown)
	}
	check := requireOpenRouterDiagnosticCheckStatus(t, deepSeek, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "image_input=missing") {
		t.Fatalf("required_capability detail = %q, want missing image_input", check.Detail)
	}

	openAI := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-openrouter-gpt",
		CatalogModel:         "openai/gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput},
	})
	if openAI.HasFailures() {
		t.Fatalf("HasFailures() = true, want trusted OpenRouter OpenAI vision catalog to pass image_input requirement: %#v", openAI.Checks)
	}
	check = requireOpenRouterDiagnosticCheckStatus(t, openAI, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
	if !strings.Contains(check.Detail, "image_input=ok") {
		t.Fatalf("required_capability detail = %q, want ok image_input", check.Detail)
	}
}

func TestDiagnoseOpenRouter_RequiredImageInputUnknownWithoutTrustedCatalog(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "://bad")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-openrouter-model",
		CatalogModel:         "vendor/model",
		Capabilities:         true,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want untrusted image_input capability to fail: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.ImageInputKnown {
		t.Fatalf("OpenRouter image capability = %t known=%t, want unknown without trusted catalog", report.Capabilities.ImageInput, report.Capabilities.ImageInputKnown)
	}
	check := requireOpenRouterDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "image_input=unknown") {
		t.Fatalf("required_capability detail = %q, want unknown image_input", check.Detail)
	}
}

func TestDiagnoseOpenRouter_RequiredImageInputRejectsCrossOwnerCatalogMetadata(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "://bad")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "anthropic/claude-future-prod",
		CatalogModel:         "openai/gpt-5.4",
		Capabilities:         true,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want cross-owner catalog_model to fail image_input requirement: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if got := report.Capabilities.CatalogModel; got != "" {
		t.Fatalf("capability catalog_model = %q, want no trusted policy catalog", got)
	}
	if report.Capabilities.ImageInput || report.Capabilities.ImageInputKnown {
		t.Fatalf("OpenRouter cross-owner image capability = %t known=%t, want unknown", report.Capabilities.ImageInput, report.Capabilities.ImageInputKnown)
	}
	check := requireOpenRouterDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "image_input=unknown") {
		t.Fatalf("required_capability detail = %q, want unknown image_input", check.Detail)
	}
	capabilityCheck := requireOpenRouterDiagnosticCheckStatus(t, report, "capabilities", DiagnosticStatusOK)
	if !strings.Contains(capabilityCheck.Detail, "image_input=unknown") ||
		strings.Contains(capabilityCheck.Detail, "context_window=1000000") ||
		strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities detail = %q, want unknown image_input without untrusted OpenAI metadata", capabilityCheck.Detail)
	}
}

func TestDiagnoseOpenRouter_CapabilitiesUseTrustedPolicyCatalogForRoutedModel(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-sonnet-4.6",
		CatalogModel: "openai/gpt-5.4",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want mismatched catalog_model to warn without failing capability report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if got, want := report.Capabilities.CatalogModel, "anthropic/claude-sonnet-4.6"; got != want {
		t.Fatalf("capability catalog_model = %q, want trusted policy catalog %q", got, want)
	}
	if !report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 1000000 {
		t.Fatalf("capability context = %d known=%t, want Claude metadata", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if !report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 64000 {
		t.Fatalf("capability max output = %d known=%t, want Claude metadata", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown)
	}
	if !report.Capabilities.Pricing.Available || report.Capabilities.Pricing.InputCostPerM != 3.00 {
		t.Fatalf("capability pricing = %+v, want Claude pricing", report.Capabilities.Pricing)
	}
	capabilityCheck := requireOpenRouterDiagnosticCheckStatus(t, report, "capabilities", DiagnosticStatusOK)
	if strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities detail = %q, want no requested OpenAI catalog metadata", capabilityCheck.Detail)
	}
}

func TestDiagnoseOpenRouter_CapabilitiesRejectCrossOwnerCatalogMetadata(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "anthropic/claude-future-prod",
		CatalogModel: "openai/gpt-5.4",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want cross-owner catalog_model to warn without failing capability report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if got := report.Capabilities.CatalogModel; got != "" {
		t.Fatalf("capability catalog_model = %q, want no trusted policy catalog", got)
	}
	if report.Capabilities.ImageInputKnown {
		t.Fatalf("capability image_input = %t known=%t, want unknown for untrusted catalog", report.Capabilities.ImageInput, report.Capabilities.ImageInputKnown)
	}
	if report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 0 {
		t.Fatalf("capability context = %d known=%t, want unknown for untrusted catalog", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 0 {
		t.Fatalf("capability max output = %d known=%t, want unknown for untrusted catalog", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown)
	}
	if report.Capabilities.Pricing.Available {
		t.Fatalf("capability pricing = %+v, want unavailable for untrusted catalog", report.Capabilities.Pricing)
	}
	capabilityCheck := requireOpenRouterDiagnosticCheckStatus(t, report, "capabilities", DiagnosticStatusOK)
	if strings.Contains(capabilityCheck.Detail, "context_window=1000000") ||
		strings.Contains(capabilityCheck.Detail, "max_output_tokens=64000") ||
		strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities detail = %q, want no untrusted OpenAI catalog metadata", capabilityCheck.Detail)
	}
}

func TestDiagnoseOpenRouter_CapabilitiesPreserveMaxOutputOverrideForUntrustedCatalog(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
		ModelOverrides: map[string]config.ModelOverride{
			"anthropic/claude-future-prod": {MaxOutputTokens: 7777},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "anthropic/claude-future-prod",
		CatalogModel: "openai/gpt-5.4",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want cross-owner catalog_model to warn without failing capability report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 0 {
		t.Fatalf("capability context = %d known=%t, want unknown for untrusted catalog", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if !report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 7777 || report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceModelOverrides {
		t.Fatalf("capability max output = %d known=%t source=%q, want explicit override", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown, report.Capabilities.MaxOutputTokensSource)
	}
	if report.Capabilities.Pricing.Available {
		t.Fatalf("capability pricing = %+v, want unavailable for untrusted catalog", report.Capabilities.Pricing)
	}
	capabilityCheck := requireOpenRouterDiagnosticCheckStatus(t, report, "capabilities", DiagnosticStatusOK)
	if !strings.Contains(capabilityCheck.Detail, "max_output_tokens=7777 (model_overrides)") ||
		strings.Contains(capabilityCheck.Detail, "context_window=1000000") ||
		strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities detail = %q, want override without untrusted OpenAI catalog metadata", capabilityCheck.Detail)
	}
	catalogPolicy := requireOpenRouterDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "max_output_tokens=7777") ||
		strings.Contains(catalogPolicy.Detail, "context_window=1000000") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy detail = %q, want same override value as capabilities", catalogPolicy.Detail)
	}
}
