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
calling is enabled. Use --capabilities or --require-capability to verify
resolved local capabilities without sending a live request. Use --print-request
to print the sanitized smoke request JSON without sending it. OPENROUTER_API_URL
must be a Chat Completions endpoint or compatible proxy path; Anthropic Skin
/v1/messages is derived by the provider and should not be configured directly.`,
		Args: cobra.NoArgs,
		RunE: runOpenRouterDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorOpenRouterModelFlag, "model", "", "OpenRouter model or configured alias for 'doctor openrouter'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor openrouter' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal OpenRouter text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live OpenRouter smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, "Print resolved OpenRouter model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd)
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
		Config:               cfg,
		Model:                doctorOpenRouterModelFlag,
		CatalogModel:         doctorCatalogModelFlag,
		RunSmoke:             !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag),
		TextSmoke:            doctorSmokeFlag,
		ToolSmoke:            doctorToolSmokeFlag,
		Capabilities:         doctorCapabilitiesFlag,
		PrintRequest:         doctorPrintRequestFlag,
		RequiredCapabilities: doctorRequiredCapabilityFlags,
		SmokeTimeout:         doctorTimeoutFlag,
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
	}
}
