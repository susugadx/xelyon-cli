package gemini

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseGemini_ServiceTierPolicyUsesRuntimeConfig(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "gemini-key")
	t.Setenv(geminiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierPriority

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Model:        defaultGeminiDiagnosticModel,
		CatalogModel: defaultGeminiDiagnosticModel,
	})
	if report.ServiceTier.ConfiguredTier != config.GeminiServiceTierPriority ||
		report.ServiceTier.RequestBodyTier != config.GeminiServiceTierPriority ||
		report.ServiceTier.PricingFamily != "gemini_priority" ||
		report.ServiceTier.BillingTier != "" {
		t.Fatalf("ServiceTier = %+v, want configured priority request pricing without billing observation", report.ServiceTier)
	}

	serviceTier := requireGeminiServiceTierCheckDetail(t, report,
		"configured=priority",
		"request_body=priority",
		"pricing_family=gemini_priority",
	)
	if strings.Contains(serviceTier.Detail, "billing=") {
		t.Fatalf("service_tier detail = %q, should not include billing before smoke", serviceTier.Detail)
	}
}

func requireGeminiServiceTierCheckDetail(t *testing.T, report DiagnosticReport, fragments ...string) DiagnosticCheck {
	t.Helper()
	check := requireGeminiDiagnosticCheckStatus(t, report, "service_tier", DiagnosticStatusOK)
	for _, fragment := range fragments {
		if !strings.Contains(check.Detail, fragment) {
			t.Fatalf("service_tier detail = %q, want %q", check.Detail, fragment)
		}
	}
	return check
}
