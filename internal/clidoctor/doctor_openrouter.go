package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	openrouterprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openrouter"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunOpenRouterDoctor は OpenRouter の診断を実行し、選択された形式で出力する。
func RunOpenRouterDoctor(ctx context.Context, out io.Writer, options OpenRouterOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := openrouterprovider.Diagnose(ctx, openrouterprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                options.Model,
		CatalogModel:         options.CatalogModel,
		RunSmoke:             !options.PrintRequest && (options.Smoke || options.ToolSmoke),
		TextSmoke:            options.Smoke,
		ToolSmoke:            options.ToolSmoke,
		Capabilities:         options.Capabilities,
		PrintRequest:         options.PrintRequest,
		RequiredCapabilities: options.RequiredCapabilities,
		SmokeTimeout:         options.Timeout,
	})
	if loadErr != nil {
		report.Checks = append([]openrouterprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     openrouterprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderOpenRouterDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderOpenRouterDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("openrouter diagnostics failed")
	}
	return false, nil
}

func renderOpenRouterDoctorJSON(w io.Writer, report openrouterprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderOpenRouterDoctorText(w io.Writer, report openrouterprovider.DiagnosticReport) {
	fmt.Fprintln(w, "OpenRouter doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	if strings.TrimSpace(report.UpstreamProvider) != "" || strings.TrimSpace(report.UpstreamModel) != "" {
		fmt.Fprintf(w, "Upstream model: %s/%s\n", report.UpstreamProvider, report.UpstreamModel)
	}
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, openRouterDoctorCheckLines(report.Checks))

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
				renderOpenRouterDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, openRouterDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              openRouterDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true})
	}
}

func renderOpenRouterDoctorSmokeRequest(w io.Writer, request openrouterprovider.DiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
		Route:              request.Route,
		Duration:           request.Duration,
		Content:            request.Content,
		Error:              request.Error,
		Skipped:            request.Skipped,
		SkipReason:         request.SkipReason,
		UsageObserved:      request.UsageObserved,
		PricingUnavailable: request.Cost.PricingUnavailable,
		CostUSD:            request.Cost.USD,
		Usage:              openRouterDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true, PrintUsageAndCost: true})
}

func openRouterDoctorCheckLines(checks []openrouterprovider.DiagnosticCheck) []doctorCheckLine {
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

func openRouterDoctorSmokeUsage(usage openrouterprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
