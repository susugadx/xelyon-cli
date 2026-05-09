package cmd

import (
	"encoding/json"
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
support those request shapes yet.`,
		Args: cobra.NoArgs,
		RunE: runBedrockDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorBedrockModelFlag, "model", "", "Bedrock model ID or configured alias for 'doctor bedrock'")
	cmd.Flags().StringVar(&doctorCatalogModelFlag, "catalog-model", "", "Catalog model for 'doctor bedrock' capability/token/pricing policy")
	cmd.Flags().BoolVar(&doctorSmokeFlag, "smoke", false, "Send a live minimal Bedrock text smoke request")
	cmd.Flags().BoolVar(&doctorToolSmokeFlag, "tool-smoke", false, "Send a live Bedrock smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&doctorBedrockImageSmokeFlag, "image-smoke", false, "Send a live Bedrock image input smoke request")
	cmd.Flags().BoolVar(&doctorBedrockThinkingSmokeFlag, "thinking-smoke", false, "Send a live Bedrock extended-thinking smoke request")
	cmd.Flags().DurationVar(&doctorTimeoutFlag, "timeout", defaultAzureDoctorTimeout, "Timeout for 'doctor bedrock' live smoke requests")
	cmd.Flags().BoolVar(&doctorJSONFlag, "json", false, "Print 'doctor bedrock' diagnostics as JSON")

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
		RunSmoke:      doctorSmokeFlag || doctorToolSmokeFlag || doctorBedrockImageSmokeFlag || doctorBedrockThinkingSmokeFlag,
		TextSmoke:     doctorSmokeFlag,
		ToolSmoke:     doctorToolSmokeFlag,
		ImageSmoke:    doctorBedrockImageSmokeFlag,
		ThinkingSmoke: doctorBedrockThinkingSmokeFlag,
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
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(payload))
	return err
}

func renderBedrockDoctorText(w io.Writer, report bedrockprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Bedrock doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Region: %s\n", report.Region)
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	fmt.Fprintln(w)

	for _, check := range report.Checks {
		fmt.Fprintf(w, "%-4s %s: %s\n", strings.ToUpper(string(check.Status)), check.Name, check.Message)
		if strings.TrimSpace(check.Detail) != "" {
			fmt.Fprintf(w, "     detail: %s\n", check.Detail)
		}
		if strings.TrimSpace(check.Suggestion) != "" {
			fmt.Fprintf(w, "     suggestion: %s\n", check.Suggestion)
		}
	}

	if report.Smoke == nil || !report.Smoke.Ran {
		return
	}
	fmt.Fprintln(w)
	for _, request := range report.Smoke.Requests {
		renderBedrockDoctorSmokeRequest(w, request)
	}
	if len(report.Smoke.Requests) > 1 {
		fmt.Fprintf(
			w,
			"Smoke total usage: input=%d cached=%d output=%d reasoning=%d cache_creation=%d\n",
			report.Smoke.Usage.InputTokens,
			report.Smoke.Usage.CachedInputTokens,
			report.Smoke.Usage.OutputTokens,
			report.Smoke.Usage.ThinkingTokens,
			report.Smoke.Usage.CacheCreationTokens,
		)
		fmt.Fprintf(w, "Smoke total cost estimate: %s\n", bedrockDoctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost))
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
		bedrockDoctorSmokeRequestIDText(request.RequestID),
	)
	if strings.TrimSpace(request.Content) != "" {
		fmt.Fprintf(w, "Smoke content %s: %s\n", request.Name, request.Content)
	}
	fmt.Fprintf(
		w,
		"Smoke usage %s: input=%d cached=%d output=%d reasoning=%d cache_creation=%d\n",
		request.Name,
		request.Usage.InputTokens,
		request.Usage.CachedInputTokens,
		request.Usage.OutputTokens,
		request.Usage.ThinkingTokens,
		request.Usage.CacheCreationTokens,
	)
	fmt.Fprintf(w, "Smoke cost estimate %s: %s\n", request.Name, bedrockDoctorSmokeCostText(request.UsageObserved, request.Cost))
}

func bedrockDoctorSmokeRequestIDText(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return "(not returned)"
	}
	return requestID
}

func bedrockDoctorSmokeCostText(usageObserved bool, smokeCost bedrockprovider.DiagnosticSmokeCost) string {
	if !usageObserved {
		return "N/A (usage unavailable)"
	}
	if smokeCost.PricingUnavailable {
		return "N/A (pricing unavailable)"
	}
	return fmt.Sprintf("$%.8f USD", smokeCost.USD)
}
