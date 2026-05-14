package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	geminiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newGeminiDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gemini",
		Short: "Diagnose Gemini provider configuration",
		Long: `Diagnose Gemini provider configuration.

Checks GEMINI_API_KEY, GEMINI_API_URL, provider registration, model/catalog
model resolution, native Gemini request routes, function calling, image input,
thinking, context caching, web search, and token/cost metadata. Use --smoke to
send a live text request, --tool-smoke to force a dummy tool call,
--image-smoke to send one tiny image request, --web-search-smoke to verify
native Gemini web search, or --print-request to print sanitized request JSON
without sending it.`,
		Args: cobra.NoArgs,
		RunE: runGeminiDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorGeminiModelFlag, "model", "", "Gemini model or configured alias for 'doctor gemini'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor gemini' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal Gemini text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live Gemini smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&doctorGeminiImageSmokeFlag, "image-smoke", false, "Send one live Gemini image input smoke request")
	cmd.Flags().BoolVar(&doctorGeminiWebSearchSmokeFlag, "web-search-smoke", false, "Send one live Gemini native web search smoke request")
	addDoctorTimeoutFlag(cmd, "gemini", "")
	addDoctorJSONFlag(cmd, "gemini")
	addDoctorPrintRequestFlag(cmd, "gemini")

	return cmd
}

func runGeminiDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := geminiprovider.Diagnose(cmd.Context(), geminiprovider.DiagnosticOptions{
		Config:         cfg,
		Model:          doctorGeminiModelFlag,
		CatalogModel:   doctorCatalogModelFlag,
		RunSmoke:       !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorGeminiImageSmokeFlag || doctorGeminiWebSearchSmokeFlag),
		TextSmoke:      doctorSmokeFlag,
		ToolSmoke:      doctorToolSmokeFlag,
		ImageSmoke:     doctorGeminiImageSmokeFlag,
		WebSearchSmoke: doctorGeminiWebSearchSmokeFlag,
		PrintRequest:   doctorPrintRequestFlag,
		SmokeTimeout:   doctorTimeoutFlag,
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

	if doctorJSONFlag {
		if err := renderGeminiDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderGeminiDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("gemini diagnostics failed")
	}
	return nil
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
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, geminiDoctorCheckLines(report.Checks))

	if report.RequestPreview != nil {
		fmt.Fprintln(w)
		renderDoctorRequestPreview(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		if len(report.Smoke.Requests) > 1 {
			for _, request := range report.Smoke.Requests {
				renderGeminiDoctorSmokeRequest(w, request)
			}
			fmt.Fprintf(w, "Smoke total usage: %s\n", formatDoctorSmokeUsage(geminiDoctorSmokeUsage(report.Smoke.Usage)))
			fmt.Fprintf(w, "Smoke total cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
			return
		}
		fmt.Fprintf(w, "Smoke route: %s\n", report.Smoke.Route)
		fmt.Fprintf(w, "Smoke duration: %s\n", report.Smoke.Duration)
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		if errText := geminiDoctorSmokeFirstError(report.Smoke.Requests); errText != "" {
			fmt.Fprintf(w, "Smoke error: %s\n", errText)
		}
		fmt.Fprintf(w, "Smoke usage: %s\n", formatDoctorSmokeUsage(geminiDoctorSmokeUsage(report.Smoke.Usage)))
		fmt.Fprintf(w, "Smoke cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
	}
}

func renderGeminiDoctorSmokeRequest(w io.Writer, request geminiprovider.DiagnosticSmokeRequestResult) {
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
	if strings.TrimSpace(request.Error) != "" {
		fmt.Fprintf(w, "Smoke error %s: %s\n", request.Name, request.Error)
	}
	fmt.Fprintf(w, "Smoke usage %s: %s\n", request.Name, formatDoctorSmokeUsage(geminiDoctorSmokeUsage(request.Usage)))
	fmt.Fprintf(w, "Smoke cost estimate %s: %s\n", request.Name, doctorSmokeCostText(request.UsageObserved, request.Cost.PricingUnavailable, request.Cost.USD))
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
	}
}
