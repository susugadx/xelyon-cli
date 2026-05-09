package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultAzureDoctorTimeout = 120 * time.Second

var (
	doctorDeploymentFlag           string
	doctorCatalogModelFlag         string
	doctorBedrockModelFlag         string
	doctorKimiModelFlag            string
	doctorSmokeFlag                bool
	doctorToolSmokeFlag            bool
	doctorBedrockImageSmokeFlag    bool
	doctorBedrockThinkingSmokeFlag bool
	doctorKimiImageSmokeFlag       bool
	doctorKimiWebSearchSmokeFlag   bool
	doctorTimeoutFlag              = defaultAzureDoctorTimeout
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
--print-config with --deployment and --catalog-model to print a config YAML
snippet without running diagnostics.`,
		Args: cobra.NoArgs,
		RunE: runAzureDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorDeploymentFlag, "deployment", "", "Azure OpenAI deployment name for 'doctor azure'")
	cmd.Flags().StringVar(&doctorCatalogModelFlag, "catalog-model", "", "Catalog model for 'doctor azure' capability/token policy")
	cmd.Flags().BoolVar(&doctorSmokeFlag, "smoke", false, "Send a live minimal Responses API request for 'doctor azure'")
	cmd.Flags().BoolVar(&doctorToolSmokeFlag, "tool-smoke", false, "Send a live Azure OpenAI smoke request that forces a dummy tool call")
	cmd.Flags().DurationVar(&doctorTimeoutFlag, "timeout", defaultAzureDoctorTimeout, "Timeout for 'doctor azure --smoke'")
	cmd.Flags().BoolVar(&doctorJSONFlag, "json", false, "Print 'doctor azure' diagnostics as JSON")
	cmd.Flags().BoolVar(&doctorPrintConfigFlag, "print-config", false, "Print Azure OpenAI config YAML for the given deployment/catalog model")

	return cmd
}

func runAzureDoctorInvocation(cmd *cobra.Command, args []string) error {
	if doctorPrintConfigFlag {
		if err := renderAzureDoctorConfigSnippet(cmd.OutOrStdout(), azureDoctorConfigSnippetOptions{
			Deployment:   doctorDeploymentFlag,
			CatalogModel: doctorCatalogModelFlag,
			JSON:         doctorJSONFlag,
			Smoke:        doctorSmokeFlag,
			ToolSmoke:    doctorToolSmokeFlag,
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
		Config:       cfg,
		Deployment:   doctorDeploymentFlag,
		CatalogModel: doctorCatalogModelFlag,
		RunSmoke:     doctorSmokeFlag || doctorToolSmokeFlag,
		ToolSmoke:    doctorToolSmokeFlag,
		SmokeTimeout: doctorTimeoutFlag,
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
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(payload))
	return err
}

func renderAzureDoctorText(w io.Writer, report azureprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Azure OpenAI doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
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

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Smoke duration: %s\n", report.Smoke.Duration)
		fmt.Fprintf(w, "Smoke response ID: %s\n", azureDoctorSmokeResponseIDText(report.Smoke.ResponseID))
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		fmt.Fprintf(
			w,
			"Smoke usage: input=%d cached=%d output=%d reasoning=%d cache_creation=%d\n",
			report.Smoke.Usage.InputTokens,
			report.Smoke.Usage.CachedInputTokens,
			report.Smoke.Usage.OutputTokens,
			report.Smoke.Usage.ThinkingTokens,
			report.Smoke.Usage.CacheCreationTokens,
		)
		fmt.Fprintf(w, "Smoke cost estimate: %s\n", azureDoctorSmokeCostText(*report.Smoke))
	}
}

func azureDoctorSmokeResponseIDText(responseID string) string {
	responseID = strings.TrimSpace(responseID)
	if responseID == "" {
		return "(not returned)"
	}
	return responseID
}

func azureDoctorSmokeCostText(smoke azureprovider.DiagnosticSmokeResult) string {
	if !smoke.UsageObserved {
		return "N/A (usage unavailable)"
	}
	if smoke.Cost.PricingUnavailable {
		return "N/A (pricing unavailable)"
	}
	return fmt.Sprintf("$%.8f USD", smoke.Cost.USD)
}
