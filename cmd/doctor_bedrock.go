package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newBedrockDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bedrock",
		Short: "Diagnose AWS Bedrock configuration",
		Long: `Diagnose AWS Bedrock configuration.

Checks AWS region and credentials, provider registration, model/catalog model
resolution, Bedrock route selection, function calling settings, token/cost
metadata, and optional live smoke requests. Use --smoke for a text request,
--tool-smoke for a dummy tool call, --image-smoke for a tiny image request, and
--thinking-smoke for an extended-thinking request. ConverseStream image and
thinking smoke requests are reported as skipped because that route does not
support those request shapes yet. Use --print-request to print sanitized request
JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: runBedrockDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorBedrockModelFlag, "model", "", "Bedrock model ID or configured alias for 'doctor bedrock'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor bedrock' capability/token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal Bedrock text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live Bedrock smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&doctorBedrockImageSmokeFlag, "image-smoke", false, "Send a live Bedrock image input smoke request")
	cmd.Flags().BoolVar(&doctorBedrockThinkingSmokeFlag, "thinking-smoke", false, "Send a live Bedrock extended-thinking smoke request")
	addDoctorTimeoutFlag(cmd, "bedrock", "")
	addDoctorJSONFlag(cmd, "bedrock")
	addDoctorPrintRequestFlag(cmd, "bedrock")

	return cmd
}

func runBedrockDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := bedrockprovider.Diagnose(cmd.Context(), bedrockprovider.DiagnosticOptions{
		Config:        cfg,
		Model:         doctorBedrockModelFlag,
		CatalogModel:  doctorCatalogModelFlag,
		RunSmoke:      !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorBedrockImageSmokeFlag || doctorBedrockThinkingSmokeFlag),
		TextSmoke:     doctorSmokeFlag,
		ToolSmoke:     doctorToolSmokeFlag,
		ImageSmoke:    doctorBedrockImageSmokeFlag,
		ThinkingSmoke: doctorBedrockThinkingSmokeFlag,
		PrintRequest:  doctorPrintRequestFlag,
		SmokeTimeout:  doctorTimeoutFlag,
	})
	if loadErr != nil {
		report.Checks = append([]bedrockprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     bedrockprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if doctorJSONFlag {
		if err := renderBedrockDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderBedrockDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("bedrock diagnostics failed")
	}
	return nil
}

func renderBedrockDoctorJSON(w io.Writer, report bedrockprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderBedrockDoctorText(w io.Writer, report bedrockprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Bedrock doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Region: %s\n", report.Region)
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	fmt.Fprintln(w)

	renderDoctorChecks(w, bedrockDoctorCheckLines(report.Checks))

	if report.RequestPreview != nil {
		fmt.Fprintln(w)
		renderDoctorRequestPreview(w, report.RequestPreview)
	}

	if report.Smoke == nil || !report.Smoke.Ran {
		return
	}
	fmt.Fprintln(w)
	for _, request := range report.Smoke.Requests {
		renderBedrockDoctorSmokeRequest(w, request)
	}
	if len(report.Smoke.Requests) > 1 {
		fmt.Fprintf(w, "Smoke total usage: %s\n", formatDoctorSmokeUsage(bedrockDoctorSmokeUsage(report.Smoke.Usage)))
		fmt.Fprintf(w, "Smoke total cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
	}
}

func renderBedrockDoctorSmokeRequest(w io.Writer, request bedrockprovider.DiagnosticSmokeRequestResult) {
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
		"Smoke request %s: %s duration=%s request_id=%s\n",
		request.Name,
		status,
		request.Duration,
		doctorOptionalIDText(request.RequestID),
	)
	if strings.TrimSpace(request.Content) != "" {
		fmt.Fprintf(w, "Smoke content %s: %s\n", request.Name, request.Content)
	}
	fmt.Fprintf(w, "Smoke usage %s: %s\n", request.Name, formatDoctorSmokeUsage(bedrockDoctorSmokeUsage(request.Usage)))
	fmt.Fprintf(w, "Smoke cost estimate %s: %s\n", request.Name, doctorSmokeCostText(request.UsageObserved, request.Cost.PricingUnavailable, request.Cost.USD))
}

func bedrockDoctorCheckLines(checks []bedrockprovider.DiagnosticCheck) []doctorCheckLine {
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

func bedrockDoctorSmokeUsage(usage bedrockprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}
