package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	ollamaprovider "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunOllamaDoctor runs Ollama diagnostics and renders the selected output.
func RunOllamaDoctor(ctx context.Context, out io.Writer, options OllamaOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := ollamaprovider.Diagnose(ctx, ollamaprovider.DiagnosticOptions{
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
		report.Checks = append([]ollamaprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     ollamaprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderOllamaDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderOllamaDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("ollama diagnostics failed")
	}
	return false, nil
}

func renderOllamaDoctorJSON(w io.Writer, report ollamaprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderOllamaDoctorText(w io.Writer, report ollamaprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Ollama doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, ollamaDoctorCheckLines(report.Checks))

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
				renderOllamaDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, ollamaDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              ollamaDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true})
	}
}

func renderOllamaDoctorSmokeRequest(w io.Writer, request ollamaprovider.DiagnosticSmokeRequestResult) {
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
		Usage:              ollamaDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true, PrintUsageAndCost: true})
}

func ollamaDoctorCheckLines(checks []ollamaprovider.DiagnosticCheck) []doctorCheckLine {
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

func ollamaDoctorSmokeUsage(usage ollamaprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
