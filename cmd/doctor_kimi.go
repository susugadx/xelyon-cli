package cmd

import (
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

Checks MOONSHOT_API_KEY, KIMI_API_URL, provider registration, model/catalog
config, Chat Completions route, image capability, unsupported native features,
and prompt_cache_key request shape. Use --smoke to send live Kimi Chat
Completions requests, --image-smoke to send one tiny image request,
--tool-smoke to include a dummy tool call, --web-search-smoke to verify the
built-in $web_search route, or --print-request to preview sanitized request
JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: runKimiDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorKimiModelFlag, "model", "", fmt.Sprintf("Kimi model for 'doctor kimi' (default: config/XELYON_MODEL, fallback %s)", defaultKimiDoctorModel))
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor kimi' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send live minimal Kimi Chat Completions smoke requests")
	cmd.Flags().BoolVar(&doctorKimiImageSmokeFlag, "image-smoke", false, "Send one live Kimi image input smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live Kimi smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&doctorKimiWebSearchSmokeFlag, "web-search-smoke", false, "Send a live Kimi built-in $web_search smoke request")
	addDoctorTimeoutFlag(cmd, "kimi", "")
	addDoctorJSONFlag(cmd, "kimi")
	addDoctorPrintRequestFlag(cmd, "kimi")

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
		CatalogModel:   doctorCatalogModelFlag,
		RunSmoke:       !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag || doctorKimiImageSmokeFlag || doctorKimiWebSearchSmokeFlag),
		TextSmoke:      doctorSmokeFlag,
		ToolSmoke:      doctorToolSmokeFlag,
		ImageSmoke:     doctorKimiImageSmokeFlag,
		WebSearchSmoke: doctorKimiWebSearchSmokeFlag,
		PrintRequest:   doctorPrintRequestFlag,
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
	return renderDoctorJSON(w, report)
}

func renderKimiDoctorText(w io.Writer, report kimiprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Kimi doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, kimiDoctorCheckLines(report.Checks))

	if report.RequestPreview != nil {
		fmt.Fprintln(w)
		renderDoctorRequestPreview(w, report.RequestPreview)
	}

	if report.Smoke != nil && report.Smoke.Ran {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Smoke duration: %s\n", report.Smoke.Duration)
		if strings.TrimSpace(report.Smoke.Content) != "" {
			fmt.Fprintf(w, "Smoke content: %s\n", report.Smoke.Content)
		}
		for _, request := range report.Smoke.Requests {
			if request.Skipped {
				fmt.Fprintf(w, "Smoke request %s: skipped (%s)\n", request.Name, request.SkipReason)
				continue
			}
			status := "ok"
			if strings.TrimSpace(request.Error) != "" {
				status = "fail"
			}
			fmt.Fprintf(w, "Smoke request %s: %s duration=%s\n", request.Name, status, request.Duration)
			if strings.TrimSpace(request.Error) != "" {
				fmt.Fprintf(w, "Smoke error %s: %s\n", request.Name, request.Error)
			}
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

func kimiDoctorCheckLines(checks []kimiprovider.DiagnosticCheck) []doctorCheckLine {
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
