package gemini

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnoseGemini_CapabilitiesDoNotRequireAPIKey(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                defaultGeminiDiagnosticModel,
		CatalogModel:         defaultGeminiDiagnosticModel,
		Capabilities:         true,
		RequiredCapabilities: []string{"function_calling", "image_input", "web_search", "thinking"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capability-only diagnostics without API key to pass: %#v", report.Checks)
	}
	requireNoGeminiDiagnosticChecks(t, report, "auth", "endpoint")
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.FunctionCalling || !report.Capabilities.ImageInput || !report.Capabilities.WebSearch {
		t.Fatalf("Capabilities = %+v, want function/image/web_search enabled", report.Capabilities)
	}
	if report.ThinkingEnabled {
		t.Fatalf("ThinkingEnabled = true, want default config to keep current toggle disabled")
	}
	if !report.Capabilities.Thinking || !report.Capabilities.ThinkingKnown {
		t.Fatalf("Capabilities thinking = %t known=%t, want request support independent of current toggle", report.Capabilities.Thinking, report.Capabilities.ThinkingKnown)
	}
	requireGeminiDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
}

func TestDiagnoseGemini_RequiredCapabilityOnlyDoesNotValidateEndpoint(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                defaultGeminiDiagnosticModel,
		CatalogModel:         defaultGeminiDiagnosticModel,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want required-capability-only diagnostics to skip auth and endpoint checks: %#v", report.Checks)
	}
	requireNoGeminiDiagnosticChecks(t, report, "auth", "endpoint")
	requireGeminiDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
}

func TestDiagnoseGemini_RequiredModelGatedCapabilitiesUnknownForUnverifiedModel(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "gemini-pro",
		CatalogModel:         "gemini-pro",
		Capabilities:         true,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput, providerdiag.RequiredCapabilityWebSearch},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want unverified model-gated capabilities to fail: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if report.Capabilities.ImageInputKnown || report.Capabilities.WebSearchKnown {
		t.Fatalf("Capabilities = %+v, want image/web availability unknown for gemini-pro", report.Capabilities)
	}
	capabilityCheck := requireGeminiDiagnosticCheckStatus(t, report, "capabilities", DiagnosticStatusOK)
	if !strings.Contains(capabilityCheck.Detail, "image_input=unknown") ||
		!strings.Contains(capabilityCheck.Detail, "web_search=unknown") {
		t.Fatalf("capabilities detail = %q, want unknown image/web availability", capabilityCheck.Detail)
	}
	required := requireGeminiDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(required.Detail, "image_input=unknown") ||
		!strings.Contains(required.Detail, "web_search=unknown") {
		t.Fatalf("required_capability detail = %q, want unknown image/web availability", required.Detail)
	}
}

func TestDiagnoseGemini_RequiredModelGatedCapabilitiesUseHiddenTrustedCatalogModel(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-gemini-alias",
		CatalogModel:         "gemini-3.1-pro-preview",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput, providerdiag.RequiredCapabilityWebSearch},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want trusted catalog model to satisfy model-gated capabilities: %#v", report.Checks)
	}
	required := requireGeminiDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
	if !strings.Contains(required.Detail, "image_input=ok") ||
		!strings.Contains(required.Detail, "web_search=ok") {
		t.Fatalf("required_capability detail = %q, want image/web ok", required.Detail)
	}
}

func TestDiagnoseGemini_CatalogPolicyMatchesCapabilityOverrideForUntrustedCatalog(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
		ModelOverrides: map[string]config.ModelOverride{
			"corp-gemini-model": {MaxOutputTokens: 7777},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "corp-gemini-model",
		CatalogModel: "gpt-5.5",
		Capabilities: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want untrusted catalog to warn without failing capability report: %#v", report.Checks)
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.MaxOutputTokensKnown || report.Capabilities.MaxOutputTokens != 7777 || report.Capabilities.MaxOutputTokensSource != providerdiag.MaxOutputSourceModelOverrides {
		t.Fatalf("capability max output = %d known=%t source=%q, want explicit override", report.Capabilities.MaxOutputTokens, report.Capabilities.MaxOutputTokensKnown, report.Capabilities.MaxOutputTokensSource)
	}
	capabilityCheck := requireGeminiDiagnosticCheckStatus(t, report, "capabilities", DiagnosticStatusOK)
	if !strings.Contains(capabilityCheck.Detail, "max_output_tokens=7777 (model_overrides)") ||
		strings.Contains(capabilityCheck.Detail, "context_window=1050000") ||
		strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities detail = %q, want override without OpenAI catalog metadata", capabilityCheck.Detail)
	}
	catalogPolicy := requireGeminiDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "max_output_tokens=7777") ||
		strings.Contains(catalogPolicy.Detail, "context_window=1050000") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy detail = %q, want same override value as capabilities", catalogPolicy.Detail)
	}
}

func TestDiagnoseGemini_RequiredThinkingFailsForModelWithoutThinkingRequestSupport(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "gemini-2.0-flash-exp",
		CatalogModel:         "gemini-2.0-flash-exp",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})

	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure: %#v", report.Checks)
	}
	check := requireGeminiDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "thinking=missing") {
		t.Fatalf("required_capability detail = %q, want missing thinking", check.Detail)
	}
}

func TestDiagnoseGemini_RequiredThinkingFollowsGemini25RequestConfig(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	disabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "gemini-2.5-flash",
		CatalogModel:         "gemini-2.5-flash",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if !disabled.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure when Gemini 2.5 request config omits thinking: %#v", disabled.Checks)
	}
	check := requireGeminiDiagnosticCheckStatus(t, disabled, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "thinking=missing") {
		t.Fatalf("required_capability detail = %q, want missing thinking", check.Detail)
	}

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	enabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                "gemini-2.5-flash",
		CatalogModel:         "gemini-2.5-flash",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if enabled.HasFailures() {
		t.Fatalf("HasFailures() = true, want thinking required capability to pass when Gemini 2.5 request config sends thinking: %#v", enabled.Checks)
	}
	check = requireGeminiDiagnosticCheckStatus(t, enabled, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
	if !strings.Contains(check.Detail, "thinking=ok") {
		t.Fatalf("required_capability detail = %q, want ok thinking", check.Detail)
	}
}
