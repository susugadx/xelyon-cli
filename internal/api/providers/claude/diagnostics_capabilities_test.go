package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnoseClaude_CapabilitiesDoNotRequireAPIKey(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "://bad", "")
	t.Setenv(claudeFunctionCallEnv, "1")
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                defaultClaudeModel,
		CatalogModel:         defaultClaudeModel,
		Capabilities:         true,
		RequiredCapabilities: []string{"function_calling", "image_input", "web_search", "thinking"},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capability-only diagnostics without API key to pass: %#v", report.Checks)
	}
	requireNoClaudeDiagnosticChecks(t, report, "auth", "endpoint")
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.FunctionCalling || !report.Capabilities.ImageInput || !report.Capabilities.WebSearch {
		t.Fatalf("Capabilities = %+v, want function/image/web_search enabled", report.Capabilities)
	}
	if !report.Capabilities.Thinking || !report.Capabilities.ThinkingKnown {
		t.Fatalf("Capabilities thinking = %t known=%t, want enabled request config", report.Capabilities.Thinking, report.Capabilities.ThinkingKnown)
	}
	requireClaudeDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
}

func TestDiagnoseClaude_RequiredThinkingFollowsRequestConfig(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "://bad", "")

	disabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                defaultClaudeModel,
		CatalogModel:         defaultClaudeModel,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if !disabled.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure when request config disables thinking: %#v", disabled.Checks)
	}
	check := requireClaudeDiagnosticCheckStatus(t, disabled, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "thinking=missing") {
		t.Fatalf("required_capability detail = %q, want missing thinking", check.Detail)
	}

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	enabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                defaultClaudeModel,
		CatalogModel:         defaultClaudeModel,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if enabled.HasFailures() {
		t.Fatalf("HasFailures() = true, want thinking required capability to pass when request config enables thinking: %#v", enabled.Checks)
	}
	check = requireClaudeDiagnosticCheckStatus(t, enabled, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
	if !strings.Contains(check.Detail, "thinking=ok") {
		t.Fatalf("required_capability detail = %q, want ok thinking", check.Detail)
	}

	alwaysOn := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "claude-fable-5",
		CatalogModel:         "claude-fable-5",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if alwaysOn.HasFailures() {
		t.Fatalf("HasFailures() = true, want Fable 5 model policy to satisfy thinking even when request config disables thinking: %#v", alwaysOn.Checks)
	}
	check = requireClaudeDiagnosticCheckStatus(t, alwaysOn, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
	if !strings.Contains(check.Detail, "thinking=ok") || alwaysOn.ThinkingType != "adaptive" {
		t.Fatalf("Fable 5 required_capability detail = %q thinking_type=%q, want adaptive thinking ok", check.Detail, alwaysOn.ThinkingType)
	}
}

func TestDiagnoseClaude_RequiredWebSearchRequiresTrustedClaudeCatalog(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "://bad", "")
	t.Setenv(claudeFunctionCallEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-claude-future",
		CatalogModel:         "gpt-5.4",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityWebSearch},
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want web_search required capability failure for untrusted catalog: %#v", report.Checks)
	}
	check := requireClaudeDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "web_search=unknown") {
		t.Fatalf("required_capability detail = %q, want unknown web_search", check.Detail)
	}
	requireNoClaudeDiagnosticChecks(t, report, "auth", "endpoint")

	trusted := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "corp-claude-alias",
		CatalogModel:         defaultClaudeModel,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityWebSearch},
	})
	if trusted.HasFailures() {
		t.Fatalf("HasFailures() = true, want trusted Claude catalog to satisfy web_search: %#v", trusted.Checks)
	}
	check = requireClaudeDiagnosticCheckStatus(t, trusted, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
	if !strings.Contains(check.Detail, "web_search=ok") {
		t.Fatalf("required_capability detail = %q, want ok web_search", check.Detail)
	}
}

func TestDiagnoseClaude_RequiredCapabilityOnlyDoesNotValidateEndpoint(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "://bad", "")
	t.Setenv(claudeFunctionCallEnv, "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                defaultClaudeModel,
		CatalogModel:         defaultClaudeModel,
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityImageInput},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want required-capability-only diagnostics to skip auth and endpoint checks: %#v", report.Checks)
	}
	requireNoClaudeDiagnosticChecks(t, report, "auth", "endpoint")
	requireClaudeDiagnosticCheckStatus(t, report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK)
}

func TestDiagnoseClaude_CatalogPolicyMatchesCapabilityOverrideForUntrustedCatalog(t *testing.T) {
	setClaudeDiagnosticTestEnv(t, "://bad", "")
	t.Setenv(claudeFunctionCallEnv, "1")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		ModelOverrides: map[string]config.ModelOverride{
			"corp-claude-model": {MaxOutputTokens: 7777},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        "corp-claude-model",
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
	capabilityCheck := requireClaudeDiagnosticCheckStatus(t, report, "capabilities", DiagnosticStatusOK)
	if !strings.Contains(capabilityCheck.Detail, "max_output_tokens=7777 (model_overrides)") ||
		strings.Contains(capabilityCheck.Detail, "context_window=1050000") ||
		strings.Contains(capabilityCheck.Detail, "pricing=input $2.50/M") {
		t.Fatalf("capabilities detail = %q, want override without OpenAI catalog metadata", capabilityCheck.Detail)
	}
	catalogPolicy := requireClaudeDiagnosticCheckStatus(t, report, "catalog_policy", DiagnosticStatusWarn)
	if !strings.Contains(catalogPolicy.Detail, "max_output_tokens=7777") ||
		strings.Contains(catalogPolicy.Detail, "context_window=1050000") ||
		strings.Contains(catalogPolicy.Detail, "pricing=input $2.50/M") {
		t.Fatalf("catalog_policy detail = %q, want same override value as capabilities", catalogPolicy.Detail)
	}
}
