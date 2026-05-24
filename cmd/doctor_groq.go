package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	groqprovider "github.com/susugadx/xelyon-cli/internal/api/providers/groq"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newGroqDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groq",
		Short: "Diagnose Groq provider configuration",
		Long: `Diagnose Groq provider configuration.

Checks GROQ_API_KEY, GROQ_API_URL, provider registration, model/catalog model
resolution, Chat Completions route selection, function calling settings, and
token/cost metadata. GROQ_API_URL is an exact Chat Completions endpoint
override; the official Groq path ends with /openai/v1/chat/completions.
OpenAI-compatible /v1/chat/completions proxy paths are allowed but reported as
endpoint warnings. Use --smoke to send a minimal live request. Use --tool-smoke
to force a dummy tool call when function calling is enabled. Use --capabilities
or --require-capability to verify resolved local capabilities without sending a
live request. Use --print-request to print the sanitized smoke request JSON
without sending it.`,
		Args: cobra.NoArgs,
		RunE: runGroqDoctorInvocation,
	}

	cmd.Flags().StringVar(&doctorGroqModelFlag, "model", "", "Groq model or configured alias for 'doctor groq'")
	addDoctorCatalogModelFlag(cmd, "Catalog model for 'doctor groq' token/pricing policy")
	addDoctorSmokeFlag(cmd, "Send a live minimal Groq text smoke request")
	addDoctorToolSmokeFlag(cmd, "Send a live Groq smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, "Print resolved Groq model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd)
	addDoctorTimeoutFlag(cmd, "groq", "")
	addDoctorJSONFlag(cmd, "groq")
	addDoctorPrintRequestFlag(cmd, "groq")

	return cmd
}

func runGroqDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := groqprovider.Diagnose(cmd.Context(), groqprovider.DiagnosticOptions{
		Config:               cfg,
		Model:                doctorGroqModelFlag,
		CatalogModel:         doctorCatalogModelFlag,
		RunSmoke:             !doctorPrintRequestFlag && (doctorSmokeFlag || doctorToolSmokeFlag),
		TextSmoke:            doctorSmokeFlag,
		ToolSmoke:            doctorToolSmokeFlag,
		Capabilities:         doctorCapabilitiesFlag,
		PrintRequest:         doctorPrintRequestFlag,
		RequiredCapabilities: doctorRequiredCapabilityFlags,
		SmokeTimeout:         doctorTimeoutFlag,
	})
	if loadErr != nil {
		report.Checks = append([]groqprovider.DiagnosticCheck{{
			Name:       "config",
			Status:     groqprovider.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if doctorJSONFlag {
		if err := renderGroqDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderGroqDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("groq diagnostics failed")
	}
	return nil
}

func renderGroqDoctorJSON(w io.Writer, report groqprovider.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderGroqDoctorText(w io.Writer, report groqprovider.DiagnosticReport) {
	fmt.Fprintln(w, "Groq doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Route: %s\n", report.Route)
	if strings.TrimSpace(report.RouteReason) != "" {
		fmt.Fprintf(w, "Route reason: %s\n", report.RouteReason)
	}
	fmt.Fprintf(w, "API URL: %s\n", report.APIURL)
	fmt.Fprintln(w)

	renderDoctorChecks(w, groqDoctorCheckLines(report.Checks))

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
				renderGroqDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, groqDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              groqDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true})
	}
}

func renderGroqDoctorSmokeRequest(w io.Writer, request groqprovider.DiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
		Route:              request.Route,
		Duration:           request.Duration,
		Content:            request.Content,
		Error:              request.Error,
		Skipped:            request.Skipped,
		SkipReason:         request.SkipReason,
		UsageObserved:      request.UsageObserved,
		PricingUnavailable: request.Cost.PricingUnavailable,
		CostUSD:            request.Cost.USD,
		Usage:              groqDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true, PrintUsageAndCost: true})
}

func groqDoctorCheckLines(checks []groqprovider.DiagnosticCheck) []doctorCheckLine {
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

func groqDoctorSmokeUsage(usage groqprovider.DiagnosticSmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}
