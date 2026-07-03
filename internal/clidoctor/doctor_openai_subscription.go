package clidoctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// RunOpenAISubscriptionDoctor は OpenAI Subscription の診断を実行し、選択された形式で出力する。
func RunOpenAISubscriptionDoctor(ctx context.Context, out io.Writer, options OpenAISubscriptionOptions) (bool, error) {
	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	smokeOutput := options.SmokeOutput
	if smokeOutput == nil {
		smokeOutput = out
	}
	report := openaisubscription.DiagnoseOpenAISubscription(ctx, openaisubscription.SubscriptionDiagnosticOptions{
		Config:               cfg,
		Model:                options.Model,
		CatalogModel:         options.CatalogModel,
		RunSmoke:             !options.PrintRequest && (options.Smoke || options.ToolSmoke || options.RetentionSmoke || options.CacheSmoke || options.CompactSmoke || options.ThinkingSmoke || options.WebSearchSmoke),
		TextSmoke:            options.Smoke,
		ToolSmoke:            options.ToolSmoke,
		RetentionSmoke:       options.RetentionSmoke,
		CacheSmoke:           options.CacheSmoke,
		CompactSmoke:         options.CompactSmoke,
		ThinkingSmoke:        options.ThinkingSmoke,
		WebSearchSmoke:       options.WebSearchSmoke,
		Capabilities:         options.Capabilities,
		RequiredCapabilities: options.RequiredCapabilities,
		PrintRequest:         options.PrintRequest,
		SmokeTimeout:         options.Timeout,
		SmokeOutput:          smokeOutput,
	})
	if loadErr != nil {
		report.Checks = append([]openaisubscription.DiagnosticCheck{{
			Name:       "config",
			Status:     openaisubscription.DiagnosticStatusWarn,
			Message:    "failed to load config; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if options.JSON {
		if err := renderOpenAISubscriptionDoctorJSON(out, report); err != nil {
			return false, err
		}
	} else {
		renderOpenAISubscriptionDoctorText(out, report)
	}

	if report.HasFailures() {
		return true, fmt.Errorf("openai_subscription diagnostics failed")
	}
	return false, nil
}

func renderOpenAISubscriptionDoctorJSON(w io.Writer, report openaisubscription.SubscriptionDiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderOpenAISubscriptionDoctorText(w io.Writer, report openaisubscription.SubscriptionDiagnosticReport) {
	fmt.Fprintln(w, "OpenAI Subscription doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Model: %s (%s)\n", report.Model, report.ModelSource)
	fmt.Fprintf(w, "Catalog model: %s (%s)\n", report.CatalogModel, report.CatalogModelSource)
	fmt.Fprintf(w, "Endpoint: %s\n", report.Endpoint)
	fmt.Fprintf(w, "Billing: %s\n", report.Billing)
	fmt.Fprintf(w, "API cost: %s\n", report.APICost)
	fmt.Fprintf(w, "Runtime mode: %s\n", report.RuntimeMode)
	fmt.Fprintf(w, "Auth: %s\n", report.AuthState)
	if strings.TrimSpace(report.Account) != "" {
		fmt.Fprintf(w, "Account: %s\n", report.Account)
	}
	fmt.Fprintf(w, "Originator: %s\n", report.Originator)
	fmt.Fprintln(w, "Responses compatibility:")
	fmt.Fprintf(w, "  prompt_cache_key: %s\n", report.PromptCacheKey)
	fmt.Fprintf(w, "  prompt_cache_retention: %s\n", report.PromptCacheRetention)
	fmt.Fprintf(w, "  store: %s\n", report.Store)
	fmt.Fprintf(w, "  previous_response_id: %s\n", report.PreviousResponseID)
	fmt.Fprintf(w, "  context_management: %s\n", report.ContextManagement)
	fmt.Fprintf(w, "  tool_call: %s\n", subscriptionDoctorBoolStatus(report.FunctionCalling))
	fmt.Fprintln(w)

	renderDoctorChecks(w, openAISubscriptionDoctorCheckLines(report.Checks))

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
				renderOpenAISubscriptionDoctorSmokeRequest(w, request)
			}
			renderDoctorSmokeTotal(w, openAISubscriptionDoctorSmokeUsage(report.Smoke.Usage), report.Smoke.UsageObserved, report.Smoke.Cost.PricingUnavailable, report.Smoke.Cost.USD)
			return
		}
		renderDoctorSmokeSummary(w, doctorSmokeSummaryLine{
			Route:              report.Smoke.Route,
			Duration:           report.Smoke.Duration,
			Content:            report.Smoke.Content,
			UsageObserved:      report.Smoke.UsageObserved,
			PricingUnavailable: report.Smoke.Cost.PricingUnavailable,
			CostUSD:            report.Smoke.Cost.USD,
			Usage:              openAISubscriptionDoctorSmokeUsage(report.Smoke.Usage),
		}, doctorSmokeSummaryRenderOptions{IncludeRoute: true})
	}
}

func renderOpenAISubscriptionDoctorSmokeRequest(w io.Writer, request openaisubscription.SubscriptionDiagnosticSmokeRequestResult) {
	renderDoctorSmokeRequestLine(w, doctorSmokeRequestLine{
		Name:               request.Name,
		Route:              request.Route,
		Duration:           request.Duration,
		Content:            request.Content,
		Error:              request.Error,
		Skipped:            request.Skipped,
		SkipReason:         request.SkipReason,
		RetentionPayload:   request.RetentionPayload,
		UsageObserved:      request.UsageObserved,
		PricingUnavailable: request.Cost.PricingUnavailable,
		CostUSD:            request.Cost.USD,
		Usage:              openAISubscriptionDoctorSmokeUsage(request.Usage),
	}, doctorSmokeRequestRenderOptions{
		IncludeRoute:      true,
		PrintError:        true,
		PrintUsageAndCost: true,
	})
}

func openAISubscriptionDoctorCheckLines(checks []openaisubscription.DiagnosticCheck) []doctorCheckLine {
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

func openAISubscriptionDoctorSmokeUsage(usage providerdiag.SmokeUsage) doctorSmokeUsageLine {
	return doctorSmokeUsageLine{
		InputTokens:         usage.InputTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		BillingServiceTier:  usage.BillingServiceTier,
	}
}

func subscriptionDoctorBoolStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
