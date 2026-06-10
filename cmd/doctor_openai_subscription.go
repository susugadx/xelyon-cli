package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func newOpenAISubscriptionDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "openai-subscription",
		Aliases: []string{"openai_subscription", "chatgpt", "codex-subscription"},
		Short:   "Diagnose OpenAI Subscription provider configuration",
		Long: `Diagnose OpenAI Subscription provider configuration.

Checks local ChatGPT/Codex OAuth auth status, provider registration, model
allowlist, endpoint/originator, billing/cost display, and v2 Responses runtime
policy. The default command is local-only: it does not refresh tokens and does
not send network requests. Use --smoke for a live text request, --tool-smoke
for function_call/function_call_output continuation, --cache-smoke for
prompt_cache_key request-shape verification, --retention-smoke to confirm
full-payload fallback policy, --thinking-smoke for the selected thinking level,
and --compact-smoke to classify optional Compact API support. Use
--print-request to print a sanitized structural request preview without
sending it.`,
		Args: cobra.NoArgs,
		RunE: runOpenAISubscriptionDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorOpenAISubscriptionModelFlag, "model", "", "OpenAI Subscription model for 'doctor openai-subscription'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor openai-subscription' capability/token policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal OpenAI Subscription text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live OpenAI Subscription tool-loop smoke request")
	addDoctorCapabilitiesFlag(cmd, "Print resolved OpenAI Subscription capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd)
	cmd.Flags().BoolVar(&doctorOpenAISubscriptionRetentionSmokeFlag, "retention-smoke", false, "Verify OpenAI Subscription full-payload retention policy")
	cmd.Flags().BoolVar(&doctorOpenAISubscriptionCacheSmokeFlag, "cache-smoke", false, "Verify OpenAI Subscription prompt_cache_key request shape")
	cmd.Flags().BoolVar(&doctorOpenAISubscriptionCompactSmokeFlag, "compact-smoke", false, "Classify optional OpenAI Subscription Compact API support")
	cmd.Flags().BoolVar(&doctorOpenAISubscriptionThinkingSmokeFlag, "thinking-smoke", false, "Verify OpenAI Subscription reasoning.effort request shape for the configured thinking level")
	addDoctorTimeoutFlag(cmd, "openai-subscription", "")
	addDoctorJSONFlag(cmd, "openai-subscription")
	addDoctorPrintRequestFlag(cmd, "openai-subscription")

	return cmd
}

func runOpenAISubscriptionDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := openaisubscription.DiagnoseOpenAISubscription(cmd.Context(), openaisubscription.SubscriptionDiagnosticOptions{
		Config:               cfg,
		Model:                doctorOpenAISubscriptionModelFlag,
		CatalogModel:         doctorCatalogModelFlag,
		RunSmoke:             !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorOpenAISubscriptionRetentionSmokeFlag || doctorOpenAISubscriptionCacheSmokeFlag || doctorOpenAISubscriptionCompactSmokeFlag || doctorOpenAISubscriptionThinkingSmokeFlag),
		TextSmoke:            doctorSmokeFlag,
		ToolSmoke:            doctorToolSmokeFlag,
		RetentionSmoke:       doctorOpenAISubscriptionRetentionSmokeFlag,
		CacheSmoke:           doctorOpenAISubscriptionCacheSmokeFlag,
		CompactSmoke:         doctorOpenAISubscriptionCompactSmokeFlag,
		ThinkingSmoke:        doctorOpenAISubscriptionThinkingSmokeFlag,
		Capabilities:         doctorCapabilitiesFlag,
		RequiredCapabilities: doctorRequiredCapabilityFlags,
		PrintRequest:         doctorPrintRequestFlag,
		SmokeTimeout:         doctorTimeoutFlag,
		SmokeOutput:          cmd.OutOrStdout(),
	})
	if loadErr != nil {
		report.Checks = append([]openaisubscription.DiagnosticCheck{{
			Name:       "config",
			Status:     openaisubscription.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if doctorJSONFlag {
		if err := renderOpenAISubscriptionDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderOpenAISubscriptionDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("openai_subscription diagnostics failed")
	}
	return nil
}

func renderOpenAISubscriptionDoctorJSON(w io.Writer, report openaisubscription.SubscriptionDiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderOpenAISubscriptionDoctorText(w io.Writer, report openaisubscription.SubscriptionDiagnosticReport) {
	fmt.Fprintln(w, "OpenAI Subscription doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Endpoint: %s\n", report.Endpoint)
	fmt.Fprintf(w, "Billing: %s\n", report.Billing)
	fmt.Fprintf(w, "API cost: %s\n", report.APICost)
	fmt.Fprintf(w, "Runtime mode: %s\n", report.RuntimeMode)
	fmt.Fprintf(w, "Auth: %s\n", report.AuthState)
	if strings.TrimSpace(report.Account) != "" {
		fmt.Fprintf(w, "Account: %s\n", report.Account)
	}
	fmt.Fprintf(w, "Originator: %s\n", report.Originator)
	fmt.Fprintln(w, "Responses compatibility:")
	fmt.Fprintf(w, "  prompt_cache_key: %s\n", report.PromptCacheKey)
	fmt.Fprintf(w, "  prompt_cache_retention: %s\n", report.PromptCacheRetention)
	fmt.Fprintf(w, "  store: %s\n", report.Store)
	fmt.Fprintf(w, "  previous_response_id: %s\n", report.PreviousResponseID)
	fmt.Fprintf(w, "  context_management: %s\n", report.ContextManagement)
	fmt.Fprintf(w, "  tool_call: %s\n", subscriptionDoctorBoolStatus(report.FunctionCalling))
	fmt.Fprintln(w)

	renderDoctorChecks(w, openAISubscriptionDoctorCheckLines(report.Checks))

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
				renderOpenAISubscriptionDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, openAISubscriptionDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              openAISubscriptionDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true})
	}
}

func renderOpenAISubscriptionDoctorSmokeRequest(w io.Writer, request openaisubscription.SubscriptionDiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
		Route:              request.Route,
		Duration:           request.Duration,
		Content:            request.Content,
		Error:              request.Error,
		Skipped:            request.Skipped,
		SkipReason:         request.SkipReason,
		RetentionPayload:   request.RetentionPayload,
		UsageObserved:      request.UsageObserved,
		PricingUnavailable: request.Cost.PricingUnavailable,
		CostUSD:            request.Cost.USD,
		Usage:              openAISubscriptionDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{
		IncludeRoute:      true,
		PrintError:        true,
		PrintUsageAndCost: true,
	})
}

func openAISubscriptionDoctorCheckLines(checks []openaisubscription.DiagnosticCheck) []doctorCheckLine {
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

func openAISubscriptionDoctorSmokeUsage(usage providerdiag.SmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}

func subscriptionDoctorBoolStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
