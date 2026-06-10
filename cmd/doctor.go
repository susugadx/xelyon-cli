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
	doctorDeploymentFlag                       string
	doctorCatalogModelFlag                     string
	doctorBedrockModelFlag                     string
	doctorClaudeModelFlag                      string
	doctorDeepSeekModelFlag                    string
	doctorGeminiModelFlag                      string
	doctorGroqModelFlag                        string
	doctorKimiModelFlag                        string
	doctorOllamaModelFlag                      string
	doctorOpenAIModelFlag                      string
	doctorOpenAISubscriptionModelFlag          string
	doctorOpenRouterModelFlag                  string
	doctorSmokeFlag                            bool
	doctorToolSmokeFlag                        bool
	doctorCapabilitiesFlag                     bool
	doctorRequiredCapabilityFlags              []string
	doctorAzureRetentionSmokeFlag              bool
	doctorOpenAIRetentionSmokeFlag             bool
	doctorOpenAISubscriptionRetentionSmokeFlag bool
	doctorOpenAISubscriptionCacheSmokeFlag     bool
	doctorOpenAISubscriptionCompactSmokeFlag   bool
	doctorOpenAISubscriptionThinkingSmokeFlag  bool
	doctorBedrockImageSmokeFlag                bool
	doctorBedrockThinkingSmokeFlag             bool
	doctorClaudeImageSmokeFlag                 bool
	doctorClaudeThinkingSmokeFlag              bool
	doctorClaudeWebSearchSmokeFlag             bool
	doctorGeminiImageSmokeFlag                 bool
	doctorGeminiWebSearchSmokeFlag             bool
	doctorKimiImageSmokeFlag                   bool
	doctorKimiWebSearchSmokeFlag               bool
	doctorTimeoutFlag                          = defaultDoctorTimeout
	doctorJSONFlag                             bool
	doctorPrintConfigFlag                      bool
	doctorPrintRequestFlag                     bool
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
	cmd.AddCommand(newClaudeDoctorCommand())
	cmd.AddCommand(newDeepSeekDoctorCommand())
	cmd.AddCommand(newGeminiDoctorCommand())
	cmd.AddCommand(newGroqDoctorCommand())
	cmd.AddCommand(newKimiDoctorCommand())
	cmd.AddCommand(newOllamaDoctorCommand())
	cmd.AddCommand(newOpenAIDoctorCommand())
	cmd.AddCommand(newOpenAISubscriptionDoctorCommand())
	cmd.AddCommand(newOpenRouterDoctorCommand())
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
--retention-smoke to verify a previous_response_id chain. Use --capabilities
to print resolved model/deployment capabilities without sending a live request.
AZURE_OPENAI_BASE_URL is a resource v1 base URL; resource root and /openai
normalize to /openai/v1, and request preview / live smoke use
<normalized_base_url>/responses. Non-standard paths are treated as intentional
proxy base URLs and reported as warnings.
Use --print-request to print the sanitized smoke request JSON without sending
it. Use --require-capability to fail when a resolved local capability is
missing. Use --print-config with --deployment and --catalog-model to print a
config YAML snippet without running diagnostics.`,
		Args: cobra.NoArgs,
		RunE: runAzureDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorDeploymentFlag, "deployment", "", "Azure OpenAI deployment name for 'doctor azure'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor azure' capability/token policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal Responses API request for 'doctor azure'")
	addDoctorToolSmokeFlag(cmd, "Send a live Azure OpenAI smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, "Print resolved Azure OpenAI deployment capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd)
	cmd.Flags().BoolVar(&doctorAzureRetentionSmokeFlag, "retention-smoke", false, "Send live Azure OpenAI Responses API requests that verify previous_response_id retention")
	addDoctorTimeoutFlag(cmd, "azure", "Timeout for 'doctor azure --smoke'")
	addDoctorJSONFlag(cmd, "azure")
	addDoctorPrintRequestFlag(cmd, "azure")
	cmd.Flags().BoolVar(&doctorPrintConfigFlag, "print-config", false, "Print Azure OpenAI config YAML for the given deployment/catalog model")

	return cmd
}

func runAzureDoctorInvocation(cmd *cobra.Command, args []string) error {
	if doctorPrintConfigFlag {
		if err := renderAzureDoctorConfigSnippet(cmd.OutOrStdout(), azureDoctorConfigSnippetOptions{
			Deployment:           doctorDeploymentFlag,
			CatalogModel:         doctorCatalogModelFlag,
			JSON:                 doctorJSONFlag,
			Smoke:                doctorSmokeFlag,
			ToolSmoke:            doctorToolSmokeFlag,
			Capabilities:         doctorCapabilitiesFlag,
			RequiredCapabilities: doctorRequiredCapabilityFlags,
			RetentionSmoke:       doctorAzureRetentionSmokeFlag,
			PrintRequest:         doctorPrintRequestFlag,
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
		Config:               cfg,
		Deployment:           doctorDeploymentFlag,
		CatalogModel:         doctorCatalogModelFlag,
		RunSmoke:             !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorAzureRetentionSmokeFlag),
		TextSmoke:            doctorSmokeFlag,
		ToolSmoke:            doctorToolSmokeFlag,
		Capabilities:         doctorCapabilitiesFlag,
		RequiredCapabilities: doctorRequiredCapabilityFlags,
		RetentionSmoke:       doctorAzureRetentionSmokeFlag,
		PrintRequest:         doctorPrintRequestFlag,
		SmokeTimeout:         doctorTimeoutFlag,
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
	if strings.TrimSpace(report.Route) != "" {
		fmt.Fprintf(w, "Route: %s\n", report.Route)
	}
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintln(w)

	renderDoctorChecks(w, azureDoctorCheckLines(report.Checks))

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
				renderAzureDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, azureDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Duration:           report.Smoke.Duration,
			ResponseID:         report.Smoke.ResponseID,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              azureDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeResponseID: true})
	}
}

func renderAzureDoctorSmokeRequest(w io.Writer, request azureprovider.DiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
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
		Usage:              azureDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{
		IDLabel:                   "response_id",
		IDValue:                   request.ResponseID,
		IncludePreviousResponseID: true,
		PrintUsageAndCost:         true,
	})
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
		BillingServiceTier:  usage.BillingServiceTier,
	}
}
