package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	deepseekprovider "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newDeepSeekDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deepseek",
		Short: "Diagnose DeepSeek provider configuration",
		Long: `Diagnose DeepSeek provider configuration.

Checks DEEPSEEK_API_KEY, DEEPSEEK_API_URL, provider registration, model/catalog
model resolution, Chat Completions route selection, thinking request config,
function calling settings, and token/cost metadata. DEEPSEEK_API_URL is an
exact Chat Completions endpoint override; the official DeepSeek path ends with
/chat/completions. OpenAI-compatible /v1/chat/completions proxy paths are
allowed but reported as endpoint warnings. Use --smoke to send a minimal live
request. Use --tool-smoke to force a dummy tool call when function calling is
enabled. Use --print-request to print the sanitized smoke request JSON without
sending it.`,
		Args: cobra.NoArgs,
		RunE: runDeepSeekDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorDeepSeekModelFlag, "model", "", "DeepSeek model or configured alias for 'doctor deepseek'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor deepseek' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal DeepSeek text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live DeepSeek smoke request that forces a dummy tool call")
	addDoctorTimeoutFlag(cmd, "deepseek", "")
	addDoctorJSONFlag(cmd, "deepseek")
	addDoctorPrintRequestFlag(cmd, "deepseek")

	return cmd
}

func runDeepSeekDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := deepseekprovider.Diagnose(cmd.Context(), deepseekprovider.DiagnosticOptions{
		Config:       cfg,
		Model:        doctorDeepSeekModelFlag,
		CatalogModel: doctorCatalogModelFlag,
		RunSmoke:     !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag),
		TextSmoke:    doctorSmokeFlag,
		ToolSmoke:    doctorToolSmokeFlag,
		PrintRequest: doctorPrintRequestFlag,
		SmokeTimeout: doctorTimeoutFlag,
	})
	if loadErr != nil {
		report.Checks = append([]deepseekprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     deepseekprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if doctorJSONFlag {
		if err := renderDeepSeekDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderDeepSeekDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("deepseek diagnostics failed")
	}
	return nil
}

func renderDeepSeekDoctorJSON(w io.Writer, report deepseekprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderDeepSeekDoctorText(w io.Writer, report deepseekprovider.DiagnosticReport) {
	fmt.Fprintln(w, "DeepSeek doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "API model: %s\n", report.APIModel)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "Thinking: supported=%t enabled=%t type=%s", report.ThinkingSupported, report.ThinkingEnabled, report.ThinkingType)
	if strings.TrimSpace(report.ReasoningEffort) != "" {
		fmt.Fprintf(w, " reasoning_effort=%s", report.ReasoningEffort)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, deepSeekDoctorCheckLines(report.Checks))

	if report.RequestPreview != nil {
		renderDoctorRequestPreviewSection(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		if len(report.Smoke.Requests) > 1 {
			for _, request := range report.Smoke.Requests {
				renderDeepSeekDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, deepSeekDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              deepSeekDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true})
	}
}

func renderDeepSeekDoctorSmokeRequest(w io.Writer, request deepseekprovider.DiagnosticSmokeRequestResult) {
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
		Usage:              deepSeekDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true, PrintUsageAndCost: true})
}

func deepSeekDoctorCheckLines(checks []deepseekprovider.DiagnosticCheck) []doctorCheckLine {
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

func deepSeekDoctorSmokeUsage(usage deepseekprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
