package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	geminiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunGeminiDoctor は Gemini の診断を実行し、選択された形式で出力する。
func RunGeminiDoctor(ctx context.Context, out io.Writer, options GeminiOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := geminiprovider.Diagnose(ctx, geminiprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                options.Model,
		CatalogModel:         options.CatalogModel,
		RunSmoke:             !options.PrintRequest && (options.Smoke || options.ToolSmoke || options.ImageSmoke || options.WebSearchSmoke),
		TextSmoke:            options.Smoke,
		ToolSmoke:            options.ToolSmoke,
		ImageSmoke:           options.ImageSmoke,
		WebSearchSmoke:       options.WebSearchSmoke,
		Capabilities:         options.Capabilities,
		PrintRequest:         options.PrintRequest,
		RequiredCapabilities: options.RequiredCapabilities,
		SmokeTimeout:         options.Timeout,
	})
	if loadErr != nil {
		report.Checks = append([]geminiprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     geminiprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderGeminiDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderGeminiDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("gemini diagnostics failed")
	}
	return false, nil
}

func renderGeminiDoctorJSON(w io.Writer, report geminiprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderGeminiDoctorText(w io.Writer, report geminiprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Gemini doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "Capabilities: function_calling=%t image_input=%t web_search=%t context_caching=%t thinking=%t\n",
		report.FunctionCallingEnabled,
		report.ImageInputSupported,
		report.WebSearchSupported,
		report.ContextCachingEnabled,
		report.ThinkingEnabled,
	)
	if strings.TrimSpace(report.ServiceTier.ConfiguredTier) != "" {
		fmt.Fprintf(w, "Service tier: %s\n", report.ServiceTier.Detail())
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, geminiDoctorCheckLines(report.Checks))

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
				renderGeminiDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, geminiDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			Error:              geminiDoctorSmokeFirstError(report.Smoke.Requests),
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              geminiDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true, PrintError: true})
	}
}

func renderGeminiDoctorSmokeRequest(w io.Writer, request geminiprovider.DiagnosticSmokeRequestResult) {
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
		Usage:              geminiDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true, PrintError: true, PrintUsageAndCost: true})
}

func geminiDoctorSmokeFirstError(requests []geminiprovider.DiagnosticSmokeRequestResult) string {
	for _, request := range requests {
		if errText := strings.TrimSpace(request.Error); errText != "" {
			return errText
		}
	}
	return ""
}

func geminiDoctorCheckLines(checks []geminiprovider.DiagnosticCheck) []doctorCheckLine {
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

func geminiDoctorSmokeUsage(usage geminiprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
