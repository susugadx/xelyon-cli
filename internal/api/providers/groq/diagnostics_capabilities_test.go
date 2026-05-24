package groq

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnoseGroq_CapabilitiesDoNotRequireAPIKey(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, "://bad")
	t.Setenv(groqFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel:         "meta-llama/llama-4-scout-17b-16e-instruct",
		Capabilities:         true,
		RequiredCapabilities: []string{"chat_completions", "function_calling"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capability-only diagnostics without API key to pass: %#v", report.Checks)
	}
	if _, ok := groqDiagnosticCheckByName(report, "auth"); ok {
		t.Fatalf("auth check was added for capability-only report: %#v", report.Checks)
	}
	if _, ok := groqDiagnosticCheckByName(report, "endpoint"); ok {
		t.Fatalf("endpoint check was added for capability-only report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.ChatCompletions || !report.Capabilities.FunctionCalling {
		t.Fatalf("Capabilities = %+v, want chat/function enabled", report.Capabilities)
	}
	check, ok := groqDiagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("required_capability check = %#v, %v; want ok", check, ok)
	}
}

func TestDiagnoseGroq_RequiredCapabilityOnlyDoesNotValidateEndpoint(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, "://bad")
	t.Setenv(groqFunctionCallingEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModel:         "meta-llama/llama-4-scout-17b-16e-instruct",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityFunctionCalling},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want required-capability-only diagnostics to skip auth and endpoint checks: %#v", report.Checks)
	}
	for _, name := range []string{"auth", "endpoint"} {
		if _, ok := groqDiagnosticCheckByName(report, name); ok {
			t.Fatalf("%s check was added for required-capability-only report: %#v", name, report.Checks)
		}
	}
	check, ok := groqDiagnosticCheckByName(report, providerdiag.RequiredCapabilityCheckName)
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("required_capability check = %#v, %v; want ok", check, ok)
	}
}

func TestDiagnoseGroq_CapabilitiesIgnoreNonGroqCatalogMetadata(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "corp-groq-model",
		CatalogModel: "gpt-5.4",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want non-Groq catalog to warn without failing capability report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 0 {
		t.Fatalf("capability context = %d known=%t, want unknown for non-Groq catalog", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 0 {
		t.Fatalf("capability max output = %d known=%t, want unknown for non-Groq catalog", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown)
	}
	if report.Capabilities.Pricing.Available {
		t.Fatalf("capability pricing = %+v, want unavailable for non-Groq catalog", report.Capabilities.Pricing)
	}
	capabilityCheck, ok := groqDiagnosticCheckByName(report, "capabilities")
	if !ok ||
		strings.Contains(capabilityCheck.Detail, "context_window=1000000") ||
		strings.Contains(capabilityCheck.Detail, "max_output_tokens=64000") ||
		strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities check = %#v, %v; want no OpenAI catalog metadata", capabilityCheck, ok)
	}
}

func TestDiagnoseGroq_CapabilitiesPreserveMaxOutputOverrideForNonGroqCatalog(t *testing.T) {
	t.Setenv(groqAPIKeyEnv, "")
	t.Setenv(groqAPIURLEnv, "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("groq", config.ProviderModelConfig{
		ModelOverrides: map[string]config.ModelOverride{
			"corp-groq-model": {MaxOutputTokens: 7777},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "corp-groq-model",
		CatalogModel: "gpt-5.4",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want non-Groq catalog to warn without failing capability report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.ContextWindowKnown || report.Capabilities.ContextWindowTokens != 0 {
		t.Fatalf("capability context = %d known=%t, want unknown for non-Groq catalog", report.Capabilities.ContextWindowTokens, report.Capabilities.ContextWindowKnown)
	}
	if !report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 7777 || report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceModelOverrides {
		t.Fatalf("capability max output = %d known=%t source=%q, want explicit override", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown, report.Capabilities.MaxOutputTokensSource)
	}
	if report.Capabilities.Pricing.Available {
		t.Fatalf("capability pricing = %+v, want unavailable for non-Groq catalog", report.Capabilities.Pricing)
	}
	capabilityCheck, ok := groqDiagnosticCheckByName(report, "capabilities")
	if !ok ||
		!strings.Contains(capabilityCheck.Detail, "max_output_tokens=7777 (model_overrides)") ||
		strings.Contains(capabilityCheck.Detail, "context_window=1000000") ||
		strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities check = %#v, %v; want override without OpenAI catalog metadata", capabilityCheck, ok)
	}
	catalogPolicy, ok := groqDiagnosticCheckByName(report, "catalog_policy")
	if !ok ||
		!strings.Contains(catalogPolicy.Detail, "max_output_tokens=7777") ||
		strings.Contains(catalogPolicy.Detail, "context_window=1000000") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy check = %#v, %v; want same override value as capabilities", catalogPolicy, ok)
	}
}
