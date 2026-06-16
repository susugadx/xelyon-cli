package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	claudeprovider "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunClaudeDoctor runs Claude diagnostics and renders the selected output.
func RunClaudeDoctor(ctx context.Context, out io.Writer, options ClaudeOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := claudeprovider.Diagnose(ctx, claudeprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                options.Model,
		CatalogModel:         options.CatalogModel,
		RunSmoke:             !options.PrintRequest && (options.Smoke || options.ToolSmoke || options.ImageSmoke || options.ThinkingSmoke || options.WebSearchSmoke),
		TextSmoke:            options.Smoke,
		ToolSmoke:            options.ToolSmoke,
		ImageSmoke:           options.ImageSmoke,
		ThinkingSmoke:        options.ThinkingSmoke,
		WebSearchSmoke:       options.WebSearchSmoke,
		Capabilities:         options.Capabilities,
		PrintRequest:         options.PrintRequest,
		RequiredCapabilities: options.RequiredCapabilities,
		SmokeTimeout:         options.Timeout,
	})
	if loadErr != nil {
		report.Checks = append([]claudeprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     claudeprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderClaudeDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderClaudeDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("claude diagnostics failed")
	}
	return false, nil
}

func renderClaudeDoctorJSON(w io.Writer, report claudeprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderClaudeDoctorText(w io.Writer, report claudeprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Claude doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "Capabilities: function_calling=%t image_input=%t web_search=%t thinking=%t context_management=%t claude_compaction=%t",
		report.FunctionCallingEnabled,
		report.ImageInputSupported,
		report.WebSearchSupported,
		report.ThinkingEnabled,
		report.ContextManagementEnabled,
		report.ClaudeCompactionSupported,
	)
	if strings.TrimSpace(report.ThinkingType) != "" {
		fmt.Fprintf(w, " thinking_type=%s", report.ThinkingType)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Anthropic version: %s\n", report.AnthropicVersion)
	if len(report.AnthropicBeta) > 0 {
		fmt.Fprintf(w, "Anthropic beta: %s\n", strings.Join(report.AnthropicBeta, ","))
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, claudeDoctorCheckLines(report.Checks))

	if report.Capabilities != nil {
		fmt.Fprintln(w)
		renderDoctorCapabilities(w, report.Capabilities)
	}

	if report.RequestPreview != nil {
		renderDoctorRequestPreviewSection(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		if len(report.Smoke.Requests) > 1 || claudeDoctorSmokeHasSkippedRequest(report.Smoke.Requests) {
			for _, request := range report.Smoke.Requests {
				renderClaudeDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, claudeDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			Error:              claudeDoctorSmokeFirstError(report.Smoke.Requests),
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              claudeDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true, PrintError: true})
	}
}

func claudeDoctorSmokeHasSkippedRequest(requests []claudeprovider.DiagnosticSmokeRequestResult) bool {
	for _, request := range requests {
		if request.Skipped {
			return true
		}
	}
	return false
}

func renderClaudeDoctorSmokeRequest(w io.Writer, request claudeprovider.DiagnosticSmokeRequestResult) {
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
		Usage:              claudeDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true, PrintError: true, PrintUsageAndCost: true})
}

func claudeDoctorSmokeFirstError(requests []claudeprovider.DiagnosticSmokeRequestResult) string {
	for _, request := range requests {
		if errText := strings.TrimSpace(request.Error); errText != "" {
			return errText
		}
	}
	return ""
}

func claudeDoctorCheckLines(checks []claudeprovider.DiagnosticCheck) []doctorCheckLine {
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

func claudeDoctorSmokeUsage(usage claudeprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
