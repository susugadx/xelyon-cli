package openrouter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	defaultOpenRouterDiagnosticSmokeTimeout         = 120 * time.Second
	defaultOpenRouterDiagnosticSmokeMaxOutputTokens = 64
	openRouterDiagnosticSmokeToolName               = "xelyon_openrouter_doctor_probe"
	openRouterDiagnosticToolSmokeSkipReason         = "OpenRouter function calling payloads are disabled (OPENROUTER_FUNCTION_CALLING=0)"
)

type openRouterDiagnosticSmokeRequest = providerdiag.TextToolSmokeRequest

func runOpenRouterDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultOpenRouterDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultOpenRouterDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := openRouterDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(openRouterAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	result := DiagnosticSmokeResult{Ran: true, Route: report.Route}
	started := time.Now()
	for _, request := range openRouterDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			providerdiag.AddTextToolSmokeRequestResult(
				&result,
				providerdiag.NewSkippedTextToolSmokeRequest(request, report.Route, openRouterDiagnosticToolSmokeSkipReason),
			)
			continue
		}

		requestResult, err := runOpenRouterDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
		providerdiag.AddTextToolSmokeRequestResult(&result, requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func openRouterDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []openRouterDiagnosticSmokeRequest {
	return providerdiag.TextToolSmokeRequests(providerdiag.TextToolSmokeRequestOptions{
		TextSmoke:              options.TextSmoke,
		ToolSmoke:              options.ToolSmoke,
		FunctionCallingEnabled: functionCallingEnabled,
		ProviderSlug:           "openrouter",
		ToolName:               openRouterDiagnosticSmokeToolName,
		ToolExpectedValue:      "openrouter-tool-ok",
	})
}

func runOpenRouterDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request openRouterDiagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newOpenRouterDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	applyOpenRouterDiagnosticToolChoice(provider, request)

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})

	started := time.Now()
	content, err := provider.ChatWithTools(
		requestCtx,
		request.SystemPrompt,
		[]api.Message{{Role: "user", Content: request.UserContent}},
		report.Model,
	)
	elapsed := time.Since(started).Round(time.Millisecond)
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "openrouter", report.Model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:          request.Name,
		Ran:           true,
		ToolPayload:   request.ToolPayload,
		Route:         report.Route,
		Content:       strings.TrimSpace(content),
		Duration:      elapsed.String(),
		UsageObserved: usageObserved,
		Usage:         providerdiag.SmokeUsageFromAPIUsage(usage),
		Cost:          providerdiag.SmokeCostFromEstimate(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !providerdiag.ContentHasToolCall(content, openRouterDiagnosticSmokeToolName) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", openRouterDiagnosticSmokeToolName)
			return result, errors.New(result.Error)
		}
		return result, nil
	}
	if strings.TrimSpace(content) == "" {
		result.Error = fmt.Sprintf("%s smoke response content is empty", request.Name)
		return result, errors.New(result.Error)
	}
	return result, nil
}

func newOpenRouterDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request openRouterDiagnosticSmokeRequest, output io.Writer) context.Context {
	return providerdiag.NewTextToolSmokeRequestContext(
		ctx,
		cfg,
		request,
		providerdiag.NoopDiagnosticToolDefinitions(openRouterDiagnosticSmokeToolName, "OpenRouter"),
		output,
	)
}

func applyOpenRouterDiagnosticToolChoice(provider *Provider, request openRouterDiagnosticSmokeRequest) {
	if request.ToolPayload {
		provider.SetToolChoice(openRouterDiagnosticSmokeToolName)
		return
	}
	provider.ClearToolChoice()
}

func openRouterSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}
