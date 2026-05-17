package openrouter

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOpenRouter_MissingAPIKeyFails(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "")
	t.Setenv(openRouterAPIURLEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want missing API key failure: %#v", report.Checks)
	}
	requireOpenRouterDiagnosticCheckStatus(t, report, "auth", DiagnosticStatusFail)
}
