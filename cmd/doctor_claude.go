package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	claudeprovider "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newClaudeDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Diagnose Claude provider configuration",
		Long: `Diagnose Claude provider configuration.

Checks ANTHROPIC_API_KEY, ANTHROPIC_API_URL, provider registration, model/catalog
model resolution, Anthropic Messages route, function calling, image input,
thinking request config, context management, Claude compaction, native web
search, and token/cost metadata. ANTHROPIC_API_URL is an exact Messages
endpoint override; the official path ends with /v1/messages, and other paths
are treated as intentional proxy endpoints. Use --smoke to send a live text
request, --tool-smoke to force a dummy tool call, --image-smoke to send one
tiny image request, --thinking-smoke to send one thinking request,
--web-search-smoke to verify native Claude web search, or --print-request to
print sanitized request JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: runClaudeDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorClaudeModelFlag, "model", "", "Claude model or configured alias for 'doctor claude'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor claude' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal Claude text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live Claude smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&doctorClaudeImageSmokeFlag, "image-smoke", false, "Send one live Claude image input smoke request")
	cmd.Flags().BoolVar(&doctorClaudeThinkingSmokeFlag, "thinking-smoke", false, "Send one live Claude thinking request smoke")
	cmd.Flags().BoolVar(&doctorClaudeWebSearchSmokeFlag, "web-search-smoke", false, "Send one live Claude native web search smoke request")
	addDoctorTimeoutFlag(cmd, "claude", "")
	addDoctorJSONFlag(cmd, "claude")
	addDoctorPrintRequestFlag(cmd, "claude")

	return cmd
}

func runClaudeDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := claudeprovider.Diagnose(cmd.Context(), claudeprovider.DiagnosticOptions{
		Config:         cfg,
		Model:          doctorClaudeModelFlag,
		CatalogModel:   doctorCatalogModelFlag,
		RunSmoke:       !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorClaudeImageSmokeFlag || doctorClaudeThinkingSmokeFlag || doctorClaudeWebSearchSmokeFlag),
		TextSmoke:      doctorSmokeFlag,
		ToolSmoke:      doctorToolSmokeFlag,
		ImageSmoke:     doctorClaudeImageSmokeFlag,
		ThinkingSmoke:  doctorClaudeThinkingSmokeFlag,
		WebSearchSmoke: doctorClaudeWebSearchSmokeFlag,
		PrintRequest:   doctorPrintRequestFlag,
		SmokeTimeout:   doctorTimeoutFlag,
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

	if doctorJSONFlag {
		if err := renderClaudeDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderClaudeDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("claude diagnostics failed")
	}
	return nil
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
