package kimi

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnoseKimi_CapabilitiesUseCatalogThinkingSupport(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "")
	t.Setenv(kimiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                "corp-kimi-model",
		CatalogModel:         "kimi-k2.6",
		Capabilities:         true,
		RequiredCapabilities: []string{"chat_completions", "image_input", "web_search", "thinking"},
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want capability-only diagnostics without API key to pass: %#v", report.Checks)
	}
	if hasKimiDiagnosticCheckName(report, "auth") {
		t.Fatalf("auth check was added for capability-only report: %#v", report.Checks)
	}
	for _, name := range []string{"api_url", "api_url_path"} {
		if hasKimiDiagnosticCheckName(report, name) {
			t.Fatalf("%s check was added for capability-only report: %#v", name, report.Checks)
		}
	}
	if report.Capabilities == nil {
		t.Fatal("Capabilities = nil, want resolved capability DTO")
	}
	if !report.Capabilities.ChatCompletions || !report.Capabilities.ImageInput || !report.Capabilities.WebSearch || !report.Capabilities.Thinking {
		t.Fatalf("Capabilities = %+v, want chat/image/web_search/thinking enabled", report.Capabilities)
	}
	if !hasKimiDiagnosticCheck(report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK) {
		t.Fatalf("required_capability check missing or not ok: %#v", report.Checks)
	}
}

func TestDiagnoseKimi_RequiredThinkingFollowsRequestPolicy(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "")
	t.Setenv(kimiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	disabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "kimi-k2.6",
		CatalogModel:         "kimi-k2.6",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if !disabled.HasFailures() {
		t.Fatalf("HasFailures() = false, want thinking required capability failure when request policy disables thinking: %#v", disabled.Checks)
	}
	if !hasKimiDiagnosticCheck(disabled, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusFail) {
		t.Fatalf("required_capability check missing or not fail: %#v", disabled.Checks)
	}

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	enabled := Diagnose(context.Background(), DiagnosticOptions{
		Config:               cfg,
		Model:                "kimi-k2.6",
		CatalogModel:         "kimi-k2.6",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if enabled.HasFailures() {
		t.Fatalf("HasFailures() = true, want thinking required capability to pass when request policy enables thinking: %#v", enabled.Checks)
	}
	if !hasKimiDiagnosticCheck(enabled, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK) {
		t.Fatalf("required_capability check missing or not ok: %#v", enabled.Checks)
	}

	forced := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "kimi-k2-thinking",
		CatalogModel:         "kimi-k2-thinking",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
	})
	if forced.HasFailures() {
		t.Fatalf("HasFailures() = true, want forced thinking model to satisfy thinking without request toggle: %#v", forced.Checks)
	}
	if !hasKimiDiagnosticCheck(forced, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK) {
		t.Fatalf("required_capability check missing or not ok for forced thinking model: %#v", forced.Checks)
	}

	for _, tt := range []struct {
		name         string
		model        string
		catalogModel string
	}{
		{name: "direct k2.7 code", model: "kimi-k2.7-code", catalogModel: "kimi-k2.7-code"},
		{name: "catalog k2.7 code", model: "corp-kimi-code", catalogModel: "kimi-k2.7-code"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			report := Diagnose(context.Background(), DiagnosticOptions{
				Config:               config.DefaultConfig(),
				Model:                tt.model,
				CatalogModel:         tt.catalogModel,
				RequiredCapabilities: []string{providerdiag.RequiredCapabilityThinking},
			})
			if report.HasFailures() {
				t.Fatalf("HasFailures() = true, want K2.7 Code to satisfy thinking without request toggle: %#v", report.Checks)
			}
			if !hasKimiDiagnosticCheck(report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK) {
				t.Fatalf("required_capability check missing or not ok for K2.7 Code: %#v", report.Checks)
			}
		})
	}
}

func TestDiagnoseKimi_RequiredCapabilityOnlyDoesNotValidateEndpoint(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "")
	t.Setenv(kimiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:               config.DefaultConfig(),
		Model:                "kimi-k2.6",
		CatalogModel:         "kimi-k2.6",
		RequiredCapabilities: []string{providerdiag.RequiredCapabilityChatCompletions},
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want required-capability-only diagnostics to skip auth and endpoint checks: %#v", report.Checks)
	}
	for _, name := range []string{"auth", "api_url", "api_url_path"} {
		if hasKimiDiagnosticCheckName(report, name) {
			t.Fatalf("%s check was added for required-capability-only report: %#v", name, report.Checks)
		}
	}
	if !hasKimiDiagnosticCheck(report, providerdiag.RequiredCapabilityCheckName, DiagnosticStatusOK) {
		t.Fatalf("required_capability check missing or not ok: %#v", report.Checks)
	}
}
