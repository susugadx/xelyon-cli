package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	kimiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultKimiDoctorModel = "kimi-k2.6"

func newKimiDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kimi",
		Short: "Diagnose Kimi native provider configuration",
		Long: `Diagnose Kimi native provider configuration.

Checks MOONSHOT_API_KEY, KIMI_API_URL, provider registration, model config,
image capability, unsupported native features, and prompt_cache_key request
shape. Use --smoke to send live Kimi Chat Completions requests, --image-smoke
to send one tiny image request, --tool-smoke to include a dummy tool call,
or --web-search-smoke to verify the built-in $web_search route.`,
		Args: cobra.NoArgs,
		RunE: runKimiDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorKimiModelFlag, "model", "", fmt.Sprintf("Kimi model for 'doctor kimi' (default: config/XELYON_MODEL, fallback %s)", defaultKimiDoctorModel))
	cmd.Flags().BoolVar(&doctorSmokeFlag, "smoke", false, "Send live minimal Kimi Chat Completions smoke requests")
	cmd.Flags().BoolVar(&doctorKimiImageSmokeFlag, "image-smoke", false, "Send one live Kimi image input smoke request")
	cmd.Flags().BoolVar(&doctorToolSmokeFlag, "tool-smoke", false, "Send a live Kimi smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&doctorKimiWebSearchSmokeFlag, "web-search-smoke", false, "Send a live Kimi built-in $web_search smoke request")
	cmd.Flags().DurationVar(&doctorTimeoutFlag, "timeout", defaultAzureDoctorTimeout, "Timeout for 'doctor kimi' live smoke requests")
	cmd.Flags().BoolVar(&doctorJSONFlag, "json", false, "Print 'doctor kimi' diagnostics as JSON")

	return cmd
}

func runKimiDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := kimiprovider.Diagnose(cmd.Context(), kimiprovider.DiagnosticOptions{
		Config:         cfg,
		Model:          kimiDoctorExplicitModel(cmd),
		RunSmoke:       doctorSmokeFlag || doctorToolSmokeFlag || doctorKimiImageSmokeFlag || doctorKimiWebSearchSmokeFlag,
		TextSmoke:      doctorSmokeFlag,
		ToolSmoke:      doctorToolSmokeFlag,
		ImageSmoke:     doctorKimiImageSmokeFlag,
		WebSearchSmoke: doctorKimiWebSearchSmokeFlag,
		SmokeTimeout:   doctorTimeoutFlag,
	})
	if loadErr != nil {
		report.Checks = append([]kimiprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     kimiprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if doctorJSONFlag {
		if err := renderKimiDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderKimiDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("kimi diagnostics failed")
	}
	return nil
}

func kimiDoctorExplicitModel(cmd *cobra.Command) string {
	if cmd == nil || !cmd.Flags().Changed("model") {
		return ""
	}
	return doctorKimiModelFlag
}

func renderKimiDoctorJSON(w io.Writer, report kimiprovider.DiagnosticReport) error {
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(payload))
	return err
}

func renderKimiDoctorText(w io.Writer, report kimiprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Kimi doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
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
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		fmt.Fprintf(w, "Cached input tokens observed: %d\n", report.Smoke.CachedInputTokens)
		if report.Smoke.WebSearchPayload {
			fmt.Fprintf(w, "Web search call count: %d\n", report.Smoke.WebSearchCallCount)
			fmt.Fprintf(w, "Web search call fee estimate: $%.4f USD\n", report.Smoke.WebSearchCallFeeEstimate)
			fmt.Fprintf(w, "Web search usage observed: %t\n", report.Smoke.WebSearchUsageObserved)
			if report.Smoke.SearchResultTotalTokens > 0 {
				fmt.Fprintf(w, "Search result total tokens observed: %d\n", report.Smoke.SearchResultTotalTokens)
			}
			fmt.Fprintln(w, "Note: Kimi $web_search call fee is separate from token cost; search result tokens are included in the next prompt_tokens response and are not added again.")
		}
	}
}
