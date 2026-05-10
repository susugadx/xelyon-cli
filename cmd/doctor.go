package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
	"github.com/susugadx/xelyon-cli/internal/config"
)

var (
	doctorDeploymentFlag           string
	doctorCatalogModelFlag         string
	doctorBedrockModelFlag         string
	doctorKimiModelFlag            string
	doctorOpenAIModelFlag          string
	doctorSmokeFlag                bool
	doctorToolSmokeFlag            bool
	doctorAzureRetentionSmokeFlag  bool
	doctorOpenAIRetentionSmokeFlag bool
	doctorBedrockImageSmokeFlag    bool
	doctorBedrockThinkingSmokeFlag bool
	doctorKimiImageSmokeFlag       bool
	doctorKimiWebSearchSmokeFlag   bool
	doctorTimeoutFlag              = defaultDoctorTimeout
	doctorJSONFlag                 bool
	doctorPrintConfigFlag          bool
)

func newDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run provider configuration diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newAzureDoctorCommand())
	cmd.AddCommand(newBedrockDoctorCommand())
	cmd.AddCommand(newKimiDoctorCommand())
	cmd.AddCommand(newOpenAIDoctorCommand())
	return cmd
}

func newAzureDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: "Diagnose Azure OpenAI configuration",
		Long: `Diagnose Azure OpenAI configuration.

Checks base URL, authentication, deployment resolution, catalog model,
function calling settings, and Responses API retention settings. Use --smoke
to send a minimal live Responses API request. Use --tool-smoke to force a
dummy tool call and verify function calling support for the deployment. Use
--retention-smoke to verify a previous_response_id chain. Use
--print-config with --deployment and --catalog-model to print a config YAML
snippet without running diagnostics.`,
		Args: cobra.NoArgs,
		RunE: runAzureDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorDeploymentFlag, "deployment", "", "Azure OpenAI deployment name for 'doctor azure'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor azure' capability/token policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal Responses API request for 'doctor azure'")
	addDoctorToolSmokeFlag(cmd, "Send a live Azure OpenAI smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&doctorAzureRetentionSmokeFlag, "retention-smoke", false, "Send live Azure OpenAI Responses API requests that verify previous_response_id retention")
	addDoctorTimeoutFlag(cmd, "azure", "Timeout for 'doctor azure --smoke'")
	addDoctorJSONFlag(cmd, "azure")
	cmd.Flags().BoolVar(&doctorPrintConfigFlag, "print-config", false, "Print Azure OpenAI config YAML for the given deployment/catalog model")

	return cmd
}

func runAzureDoctorInvocation(cmd *cobra.Command, args []string) error {
	if doctorPrintConfigFlag {
		if err := renderAzureDoctorConfigSnippet(cmd.OutOrStdout(), azureDoctorConfigSnippetOptions{
			Deployment:     doctorDeploymentFlag,
			CatalogModel:   doctorCatalogModelFlag,
			JSON:           doctorJSONFlag,
			Smoke:          doctorSmokeFlag,
			ToolSmoke:      doctorToolSmokeFlag,
			RetentionSmoke: doctorAzureRetentionSmokeFlag,
		}); err != nil {
			cmd.SilenceUsage = true
			return err
		}
		return nil
	}

	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := azureprovider.Diagnose(cmd.Context(), azureprovider.DiagnosticOptions{
		Config:         cfg,
		Deployment:     doctorDeploymentFlag,
		CatalogModel:   doctorCatalogModelFlag,
		RunSmoke:       doctorSmokeFlag || doctorToolSmokeFlag || doctorAzureRetentionSmokeFlag,
		TextSmoke:      doctorSmokeFlag,
		ToolSmoke:      doctorToolSmokeFlag,
		RetentionSmoke: doctorAzureRetentionSmokeFlag,
		SmokeTimeout:   doctorTimeoutFlag,
	})
	if loadErr != nil {
		report.Checks = append([]azureprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     azureprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if doctorJSONFlag {
		if err := renderAzureDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderAzureDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("azure OpenAI diagnostics failed")
	}
	return nil
}

func renderAzureDoctorJSON(w io.Writer, report azureprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderAzureDoctorText(w io.Writer, report azureprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Azure OpenAI doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintln(w)

	renderDoctorChecks(w, azureDoctorCheckLines(report.Checks))

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		if len(report.Smoke.Requests) > 1 {
			for _, request := range report.Smoke.Requests {
				renderAzureDoctorSmokeRequest(w, request)
			}
			fmt.Fprintf(w, "Smoke total usage: %s\n", formatDoctorSmokeUsage(azureDoctorSmokeUsage(report.Smoke.Usage)))
			fmt.Fprintf(w, "Smoke total cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
			return
		}
		fmt.Fprintf(w, "Smoke duration: %s\n", report.Smoke.Duration)
		fmt.Fprintf(w, "Smoke response ID: %s\n", doctorOptionalIDText(report.Smoke.ResponseID))
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		fmt.Fprintf(w, "Smoke usage: %s\n", formatDoctorSmokeUsage(azureDoctorSmokeUsage(report.Smoke.Usage)))
		fmt.Fprintf(w, "Smoke cost estimate: %s\n", doctorSmokeCostText(report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD))
	}
}

func renderAzureDoctorSmokeRequest(w io.Writer, request azureprovider.DiagnosticSmokeRequestResult) {
	if request.Skipped {
		fmt.Fprintf(w, "Smoke request %s: skipped (%s)\n", request.Name, request.SkipReason)
		return
	}

	status := "ok"
	if strings.TrimSpace(request.Error) != "" {
		status = "fail"
	}
	line := fmt.Sprintf(
		"Smoke request %s: %s duration=%s response_id=%s",
		request.Name,
		status,
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
	fmt.Fprintf(w, "Smoke usage %s: %s\n", request.Name, formatDoctorSmokeUsage(azureDoctorSmokeUsage(request.Usage)))
	fmt.Fprintf(w, "Smoke cost estimate %s: %s\n", request.Name, doctorSmokeCostText(request.UsageObserved, request.Cost.PricingUnavailable, request.Cost.USD))
}

func azureDoctorCheckLines(checks []azureprovider.DiagnosticCheck) []doctorCheckLine {
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

func azureDoctorSmokeUsage(usage azureprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}
