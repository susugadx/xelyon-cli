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
request. Use --print-request to print the sanitized smoke request JSON without
sending it.`,
		Args: cobra.NoArgs,
		RunE: runOpenAIDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorOpenAIModelFlag, "model", "", "OpenAI model or configured alias for 'doctor openai'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor openai' capability/token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal OpenAI text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live OpenAI smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, "Print resolved OpenAI model capabilities without sending a live request")
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
		Config:         cfg,
		Model:          doctorOpenAIModelFlag,
		CatalogModel:   doctorCatalogModelFlag,
		RunSmoke:       !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorOpenAIRetentionSmokeFlag),
		TextSmoke:      doctorSmokeFlag,
		ToolSmoke:      doctorToolSmokeFlag,
		Capabilities:   doctorCapabilitiesFlag,
		RetentionSmoke: doctorOpenAIRetentionSmokeFlag,
		PrintRequest:   doctorPrintRequestFlag,
		SmokeTimeout:   doctorTimeoutFlag,
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
		fmt.Fprintln(w)
		renderDoctorRequestPreview(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		if len(report.Smoke.Requests) > 1 {
			for _, request := range report.Smoke.Requests {
				renderOpenAIDoctorSmokeRequest(w, request)
			}
			fmt.Fprintf(w, "Smoke total usage: %s\n", formatDoctorSmokeUsage(openAIDoctorSmokeUsage(report.Smoke.Usage)))
			fmt.Fprintf(w, "Smoke total cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
			return
		}
		fmt.Fprintf(w, "Smoke route: %s\n", report.Smoke.Route)
		fmt.Fprintf(w, "Smoke duration: %s\n", report.Smoke.Duration)
		fmt.Fprintf(w, "Smoke response ID: %s\n", doctorOptionalIDText(report.Smoke.ResponseID))
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		fmt.Fprintf(w, "Smoke usage: %s\n", formatDoctorSmokeUsage(openAIDoctorSmokeUsage(report.Smoke.Usage)))
		fmt.Fprintf(w, "Smoke cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
	}
}

func renderOpenAIDoctorSmokeRequest(w io.Writer, request openaiprovider.DiagnosticSmokeRequestResult) {
	if request.Skipped {
		fmt.Fprintf(w, "Smoke request %s: skipped (%s)\n", request.Name, request.SkipReason)
		return
	}

	status := "ok"
	if strings.TrimSpace(request.Error) != "" {
		status = "fail"
	}
	line := fmt.Sprintf(
		"Smoke request %s: %s route=%s duration=%s response_id=%s",
		request.Name,
		status,
		request.Route,
		request.Duration,
		doctorOptionalIDText(request.ResponseID),
	)
	if request.RetentionPayload {
		line += fmt.Sprintf(" previous_response_id=%s", doctorOptionalIDText(request.PreviousResponseID))
	}
	fmt.Fprintln(w, line)
	if strings.TrimSpace(request.Content) != "" {
		fmt.Fprintf(w, "Smoke content %s: %s\n", request.Name, request.Content)
	}
	fmt.Fprintf(w, "Smoke usage %s: %s\n", request.Name, formatDoctorSmokeUsage(openAIDoctorSmokeUsage(request.Usage)))
	fmt.Fprintf(w, "Smoke cost estimate %s: %s\n", request.Name, doctorSmokeCostText(request.UsageObserved, request.Cost.PricingUnavailable, request.Cost.USD))
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
	}
}
