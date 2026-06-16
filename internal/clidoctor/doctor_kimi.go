package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	kimiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// RunKimiDoctor runs Kimi diagnostics and renders the selected output.
func RunKimiDoctor(ctx context.Context, out io.Writer, options KimiOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	model := ""
	if options.ModelChanged {
		model = options.Model
	}
	report := kimiprovider.Diagnose(ctx, kimiprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                model,
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
		report.Checks = append([]kimiprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     kimiprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderKimiDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderKimiDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("kimi diagnostics failed")
	}
	return false, nil
}

func renderKimiDoctorJSON(w io.Writer, report kimiprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderKimiDoctorText(w io.Writer, report kimiprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Kimi doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, kimiDoctorCheckLines(report.Checks))

	if report.Capabilities != nil {
		fmt.Fprintln(w)
		renderDoctorCapabilities(w, report.Capabilities)
	}

	if report.RequestPreview != nil {
		renderDoctorRequestPreviewSection(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Smoke duration: %s\n", report.Smoke.Duration)
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		for _, request := range report.Smoke.Requests {
			renderKimiDoctorSmokeRequest(w, request)
		}
		fmt.Fprintf(w, "Cached input tokens observed: %d\n", report.Smoke.CachedInputTokens)
		if report.Smoke.WebSearchPayload {
			fmt.Fprintf(w, "Web search call count: %d\n", report.Smoke.WebSearchCallCount)
			fmt.Fprintf(w, "Web search call fee estimate: $%.4f USD\n", report.Smoke.WebSearchCallFeeEstimate)
			fmt.Fprintf(w, "Web search usage observed: %t\n", report.Smoke.WebSearchUsageObserved)
			if report.Smoke.SearchResultTotalTokens > 0 {
				fmt.Fprintf(w, "Search result total tokens observed: %d\n", report.Smoke.SearchResultTotalTokens)
			}
			fmt.Fprintln(w, "Note: Kimi $web_search call fee is separate from token cost; search result tokens are included in the next prompt_tokens response and are not added again.")
		}
	}
}

func renderKimiDoctorSmokeRequest(w io.Writer, request kimiprovider.DiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:       request.Name,
		Duration:   request.Duration,
		Error:      request.Error,
		Skipped:    request.Skipped,
		SkipReason: request.SkipReason,
	}, doctorSmokeRequestRenderOptions{OmitContent: true, PrintError: true})
}

func kimiDoctorCheckLines(checks []kimiprovider.DiagnosticCheck) []doctorCheckLine {
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
