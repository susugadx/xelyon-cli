package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	openaiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newOpenAIDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openai",
		Short: "Diagnose OpenAI provider configuration",
		Long: `Diagnose OpenAI provider configuration.

Checks OPENAI_API_KEY, OpenAI Chat Completions and Responses endpoints,
provider registration, model/catalog model resolution, route selection,
function calling settings, token/cost metadata, and Responses API retention
settings. Use --smoke to send a minimal live request. Use --tool-smoke to
force a dummy tool call when function calling is enabled. Use
--retention-smoke to verify a Responses API previous_response_id chain. Use
--capabilities to print resolved model capabilities without sending a live
request. Use --require-capability to fail when a resolved local capability is
missing. Use --print-request to print the sanitized smoke request JSON without
sending it.`,
		Args: cobra.NoArgs,
		RunE: runOpenAIDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorOpenAIModelFlag, "model", "", "OpenAI model or configured alias for 'doctor openai'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor openai' capability/token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal OpenAI text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live OpenAI smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, "Print resolved OpenAI model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd)
	cmd.Flags().BoolVar(&doctorOpenAIRetentionSmokeFlag, "retention-smoke", false, "Send live OpenAI Responses API requests that verify previous_response_id retention")
	addDoctorTimeoutFlag(cmd, "openai", "")
	addDoctorJSONFlag(cmd, "openai")
	addDoctorPrintRequestFlag(cmd, "openai")

	return cmd
}

func runOpenAIDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := openaiprovider.Diagnose(cmd.Context(), openaiprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                doctorOpenAIModelFlag,
		CatalogModel:         doctorCatalogModelFlag,
		RunSmoke:             !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorOpenAIRetentionSmokeFlag),
		TextSmoke:            doctorSmokeFlag,
		ToolSmoke:            doctorToolSmokeFlag,
		Capabilities:         doctorCapabilitiesFlag,
		RequiredCapabilities: doctorRequiredCapabilityFlags,
		RetentionSmoke:       doctorOpenAIRetentionSmokeFlag,
		PrintRequest:         doctorPrintRequestFlag,
		SmokeTimeout:         doctorTimeoutFlag,
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

	if doctorJSONFlag {
		if err := renderOpenAIDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderOpenAIDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("openai diagnostics failed")
	}
	return nil
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
