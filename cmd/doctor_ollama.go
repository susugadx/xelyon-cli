package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	ollamaprovider "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newOllamaDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ollama",
		Short: "Diagnose Ollama provider configuration",
		Long: `Diagnose Ollama provider configuration.

Checks OLLAMA_BASE_URL, provider registration, model/catalog model resolution,
installed local model availability, Ollama /api/chat route selection, function
calling settings, and token/cost metadata. Use --smoke to send a minimal live
local request. Use --tool-smoke to force a dummy tool call when function
calling is enabled. Use --print-request to print the sanitized smoke request
JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: runOllamaDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorOllamaModelFlag, "model", "", "Ollama model or configured alias for 'doctor ollama'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor ollama' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal Ollama text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live Ollama smoke request that forces a dummy tool call")
	addDoctorTimeoutFlag(cmd, "ollama", "")
	addDoctorJSONFlag(cmd, "ollama")
	addDoctorPrintRequestFlag(cmd, "ollama")

	return cmd
}

func runOllamaDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := ollamaprovider.Diagnose(cmd.Context(), ollamaprovider.DiagnosticOptions{
		Config:       cfg,
		Model:        doctorOllamaModelFlag,
		CatalogModel: doctorCatalogModelFlag,
		RunSmoke:     !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag),
		TextSmoke:    doctorSmokeFlag,
		ToolSmoke:    doctorToolSmokeFlag,
		PrintRequest: doctorPrintRequestFlag,
		SmokeTimeout: doctorTimeoutFlag,
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

	if doctorJSONFlag {
		if err := renderOllamaDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderOllamaDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("ollama diagnostics failed")
	}
	return nil
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
	}
}
