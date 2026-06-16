package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunBedrockDoctor runs Bedrock diagnostics and renders the selected output.
func RunBedrockDoctor(ctx context.Context, out io.Writer, options BedrockOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := bedrockprovider.Diagnose(ctx, bedrockprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                options.Model,
		CatalogModel:         options.CatalogModel,
		RunSmoke:             !options.PrintRequest && (options.Smoke || options.ToolSmoke || options.ImageSmoke || options.ThinkingSmoke),
		TextSmoke:            options.Smoke,
		ToolSmoke:            options.ToolSmoke,
		ImageSmoke:           options.ImageSmoke,
		ThinkingSmoke:        options.ThinkingSmoke,
		Capabilities:         options.Capabilities,
		PrintRequest:         options.PrintRequest,
		RequiredCapabilities: options.RequiredCapabilities,
		SmokeTimeout:         options.Timeout,
	})
	if loadErr != nil {
		report.Checks = append([]bedrockprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     bedrockprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderBedrockDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderBedrockDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("bedrock diagnostics failed")
	}
	return false, nil
}

func renderBedrockDoctorJSON(w io.Writer, report bedrockprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderBedrockDoctorText(w io.Writer, report bedrockprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Bedrock doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Region: %s\n", report.Region)
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	fmt.Fprintln(w)

	renderDoctorChecks(w, bedrockDoctorCheckLines(report.Checks))

	if report.Capabilities != nil {
		fmt.Fprintln(w)
		renderDoctorCapabilities(w, report.Capabilities)
	}

	if report.RequestPreview != nil {
		renderDoctorRequestPreviewSection(w, report.RequestPreview)
	}

	if report.Smoke == nil || !report.Smoke.Ran {
		return
	}
	fmt.Fprintln(w)
	for _, request := range report.Smoke.Requests {
		renderBedrockDoctorSmokeRequest(w, request)
	}
	if len(report.Smoke.Requests) > 1 {
		renderDoctorSmokeTotal(w, bedrockDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
	}
}

func renderBedrockDoctorSmokeRequest(w io.Writer, request bedrockprovider.DiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
		Duration:           request.Duration,
		Content:            request.Content,
		Error:              request.Error,
		Skipped:            request.Skipped,
		SkipReason:         request.SkipReason,
		UsageObserved:      request.UsageObserved,
		PricingUnavailable: request.Cost.PricingUnavailable,
		CostUSD:            request.Cost.USD,
		Usage:              bedrockDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{IDLabel: "request_id", IDValue: request.RequestID, PrintUsageAndCost: true})
}

func bedrockDoctorCheckLines(checks []bedrockprovider.DiagnosticCheck) []doctorCheckLine {
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

func bedrockDoctorSmokeUsage(usage bedrockprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
