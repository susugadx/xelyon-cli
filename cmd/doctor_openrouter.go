package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	openrouterprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openrouter"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newOpenRouterDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openrouter",
		Short: "Diagnose OpenRouter provider configuration",
		Long: `Diagnose OpenRouter provider configuration.

Checks OPENROUTER_API_KEY, OPENROUTER_API_URL, provider registration,
model/catalog model resolution, OpenAI-compatible vs Anthropic Skin route
selection, image input support, function calling settings, and token/cost
metadata. Use --smoke to send a minimal live request through the selected
route. Use --tool-smoke to force a dummy diagnostic tool call when function
calling is enabled. Use --print-request to print the sanitized smoke request
JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: runOpenRouterDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorOpenRouterModelFlag, "model", "", "OpenRouter model or configured alias for 'doctor openrouter'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor openrouter' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal OpenRouter text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live OpenRouter smoke request that forces a dummy tool call")
	addDoctorTimeoutFlag(cmd, "openrouter", "")
	addDoctorJSONFlag(cmd, "openrouter")
	addDoctorPrintRequestFlag(cmd, "openrouter")

	return cmd
}

func runOpenRouterDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := openrouterprovider.Diagnose(cmd.Context(), openrouterprovider.DiagnosticOptions{
		Config:       cfg,
		Model:        doctorOpenRouterModelFlag,
		CatalogModel: doctorCatalogModelFlag,
		RunSmoke:     !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag),
		TextSmoke:    doctorSmokeFlag,
		ToolSmoke:    doctorToolSmokeFlag,
		PrintRequest: doctorPrintRequestFlag,
		SmokeTimeout: doctorTimeoutFlag,
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

	if doctorJSONFlag {
		if err := renderOpenRouterDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderOpenRouterDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("openrouter diagnostics failed")
	}
	return nil
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

	if report.RequestPreview != nil {
		fmt.Fprintln(w)
		renderDoctorRequestPreview(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		if len(report.Smoke.Requests) > 1 {
			for _, request := range report.Smoke.Requests {
				renderOpenRouterDoctorSmokeRequest(w, request)
			}
			fmt.Fprintf(w, "Smoke total usage: %s\n", formatDoctorSmokeUsage(openRouterDoctorSmokeUsage(report.Smoke.Usage)))
			fmt.Fprintf(w, "Smoke total cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
			return
		}
		fmt.Fprintf(w, "Smoke route: %s\n", report.Smoke.Route)
		fmt.Fprintf(w, "Smoke duration: %s\n", report.Smoke.Duration)
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		fmt.Fprintf(w, "Smoke usage: %s\n", formatDoctorSmokeUsage(openRouterDoctorSmokeUsage(report.Smoke.Usage)))
		fmt.Fprintf(w, "Smoke cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
	}
}

func renderOpenRouterDoctorSmokeRequest(w io.Writer, request openrouterprovider.DiagnosticSmokeRequestResult) {
	if request.Skipped {
		fmt.Fprintf(w, "Smoke request %s: skipped (%s)\n", request.Name, request.SkipReason)
		return
	}

	status := "ok"
	if strings.TrimSpace(request.Error) != "" {
		status = "fail"
	}
	fmt.Fprintf(
		w,
		"Smoke request %s: %s route=%s duration=%s\n",
		request.Name,
		status,
		request.Route,
		request.Duration,
	)
	if strings.TrimSpace(request.Content) != "" {
		fmt.Fprintf(w, "Smoke content %s: %s\n", request.Name, request.Content)
	}
	fmt.Fprintf(w, "Smoke usage %s: %s\n", request.Name, formatDoctorSmokeUsage(openRouterDoctorSmokeUsage(request.Usage)))
	fmt.Fprintf(w, "Smoke cost estimate %s: %s\n", request.Name, doctorSmokeCostText(request.UsageObserved, request.Cost.PricingUnavailable, request.Cost.USD))
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
	}
}
