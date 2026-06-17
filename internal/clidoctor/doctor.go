package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunAzureDoctor は Azure OpenAI の診断を実行し、選択された形式で出力する。
func RunAzureDoctor(ctx context.Context, out io.Writer, options AzureOptions) (bool, error) {
	if options.PrintConfig {
		if err := renderAzureDoctorConfigSnippet(out, azureDoctorConfigSnippetOptions{
			Deployment:           options.Deployment,
			CatalogModel:         options.CatalogModel,
			JSON:                 options.JSON,
			Smoke:                options.Smoke,
			ToolSmoke:            options.ToolSmoke,
			Capabilities:         options.Capabilities,
			RequiredCapabilities: options.RequiredCapabilities,
			RetentionSmoke:       options.RetentionSmoke,
			PrintRequest:         options.PrintRequest,
		}); err != nil {
			return true, err
		}
		return false, nil
	}

	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := azureprovider.Diagnose(ctx, azureprovider.DiagnosticOptions{
		Config:               cfg,
		Deployment:           options.Deployment,
		CatalogModel:         options.CatalogModel,
		RunSmoke:             !options.PrintRequest && (options.Smoke || options.ToolSmoke || options.RetentionSmoke),
		TextSmoke:            options.Smoke,
		ToolSmoke:            options.ToolSmoke,
		Capabilities:         options.Capabilities,
		RequiredCapabilities: options.RequiredCapabilities,
		RetentionSmoke:       options.RetentionSmoke,
		PrintRequest:         options.PrintRequest,
		SmokeTimeout:         options.Timeout,
	})
	if loadErr != nil {
		report.Checks = append([]azureprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     azureprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderAzureDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderAzureDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("azure OpenAI diagnostics failed")
	}
	return false, nil
}

func renderAzureDoctorJSON(w io.Writer, report azureprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderAzureDoctorText(w io.Writer, report azureprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Azure OpenAI doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	if strings.TrimSpace(report.Route) != "" {
		fmt.Fprintf(w, "Route: %s\n", report.Route)
	}
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintln(w)

	renderDoctorChecks(w, azureDoctorCheckLines(report.Checks))

	if report.Capabilities != nil {
		fmt.Fprintln(w)
		renderDoctorCapabilities(w, report.Capabilities)
	}

	if report.RequestPreview != nil {
		renderDoctorRequestPreviewSection(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		if len(report.Smoke.Requests) > 1 {
			for _, request := range report.Smoke.Requests {
				renderAzureDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, azureDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Duration:           report.Smoke.Duration,
			ResponseID:         report.Smoke.ResponseID,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              azureDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeResponseID: true})
	}
}

func renderAzureDoctorSmokeRequest(w io.Writer, request azureprovider.DiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
		Duration:           request.Duration,
		Content:            request.Content,
		Error:              request.Error,
		PreviousResponseID: request.PreviousResponseID,
		Skipped:            request.Skipped,
		SkipReason:         request.SkipReason,
		RetentionPayload:   request.RetentionPayload,
		UsageObserved:      request.UsageObserved,
		PricingUnavailable: request.Cost.PricingUnavailable,
		CostUSD:            request.Cost.USD,
		Usage:              azureDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{
		IDLabel:                   "response_id",
		IDValue:                   request.ResponseID,
		IncludePreviousResponseID: true,
		PrintUsageAndCost:         true,
	})
}

func azureDoctorCheckLines(checks []azureprovider.DiagnosticCheck) []doctorCheckLine {
	lines := make([]doctorCheckLine, 0, len(checks))
	for _, check := range checks {
		lines = append(lines, doctorCheckLine{
			Status:     string(check.Status),
			Name:       check.Name,
			Message:    check.Message,
			Detail:     check.Detail,
			Suggestion: check.Suggestion,
		})
	}
	return lines
}

func azureDoctorSmokeUsage(usage azureprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
