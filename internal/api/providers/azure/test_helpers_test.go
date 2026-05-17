package azure

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func azureTestContext(cfg *config.Config) context.Context {
	var out strings.Builder
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return config.WithContext(ctx, cfg)
}

func hasDiagnosticCheck(report DiagnosticReport, name string, status DiagnosticStatus) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func diagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}
