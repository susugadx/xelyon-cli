package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	openaiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunOpenAIDoctor runs OpenAI diagnostics and renders the selected output.
func RunOpenAIDoctor(ctx context.Context, out io.Writer, options OpenAIOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := openaiprovider.Diagnose(ctx, openaiprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                options.Model,
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
		report.Checks = append([]openaiprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     openaiprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderOpenAIDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderOpenAIDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("openai diagnostics failed")
	}
	return false, nil
}

func renderOpenAIDoctorJSON(w io.Writer, report openaiprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderOpenAIDoctorText(w io.Writer, report openaiprovider.DiagnosticReport) {
	fmt.Fprintln(w, "OpenAI doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintf(w, "Responses URL: %s\n", report.ResponsesURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, openAIDoctorCheckLines(report.Checks))

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
				renderOpenAIDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, openAIDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			ResponseID:         report.Smoke.ResponseID,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              openAIDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true, IncludeResponseID: true})
	}
}

func renderOpenAIDoctorSmokeRequest(w io.Writer, request openaiprovider.DiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
		Route:              request.Route,
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
		Usage:              openAIDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{
		IncludeRoute:              true,
		IDLabel:                   "response_id",
		IDValue:                   request.ResponseID,
		IncludePreviousResponseID: true,
		PrintUsageAndCost:         true,
	})
}

func openAIDoctorCheckLines(checks []openaiprovider.DiagnosticCheck) []doctorCheckLine {
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

func openAIDoctorSmokeUsage(usage openaiprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
