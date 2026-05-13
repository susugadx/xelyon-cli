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

type openRouterDiagnosticSmokeRequest = providerdiag.ChatCompletionsSmokeRequest

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
			result.Requests = append(result.Requests, newOpenRouterDiagnosticSkippedToolSmokeRequest(request.Name, report.Route))
			continue
		}

		requestResult, err := runOpenRouterDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
		result.Requests = append(result.Requests, requestResult)
		result.addRequestObservation(requestResult)
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

func newOpenRouterDiagnosticSkippedToolSmokeRequest(name, route string) DiagnosticSmokeRequestResult {
	return DiagnosticSmokeRequestResult{
		Name:        name,
		Skipped:     true,
		SkipReason:  openRouterDiagnosticToolSmokeSkipReason,
		ToolPayload: true,
		Route:       route,
	}
}

func newOpenRouterDiagnosticSkippedToolPreviewRequest(name, route string) DiagnosticRequestPreviewRequest {
	return DiagnosticRequestPreviewRequest{
		Name:        name,
		Skipped:     true,
		SkipReason:  openRouterDiagnosticToolSmokeSkipReason,
		ToolPayload: true,
		Route:       route,
	}
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
		Usage:         openRouterDiagnosticSmokeUsage(usage),
		Cost:          openRouterDiagnosticSmokeCost(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !openRouterDiagnosticSmokeContentHasToolCall(content) {
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
	return providerdiag.NewChatCompletionsSmokeRequestContext(ctx, cfg, request, openRouterDiagnosticSmokeToolDefinitions(), output)
}

func applyOpenRouterDiagnosticToolChoice(provider *Provider, request openRouterDiagnosticSmokeRequest) {
	if request.ToolPayload {
		provider.SetToolChoice(openRouterDiagnosticSmokeToolName)
		return
	}
	provider.ClearToolChoice()
}

func (r *DiagnosticSmokeResult) addRequestObservation(request DiagnosticSmokeRequestResult) {
	if request.Skipped {
		return
	}
	if request.ToolPayload {
		r.ToolPayload = true
	}
	if r.Route == "" {
		r.Route = request.Route
	}
	if strings.TrimSpace(r.Content) == "" {
		r.Content = request.Content
	}

	r.Usage = providerdiag.AddSmokeUsage(r.Usage, request.Usage)
	r.Cost = providerdiag.AddSmokeCost(r.Cost, request.Cost)
	r.UsageObserved = r.allRanRequestsObservedUsage()
}

func (r *DiagnosticSmokeResult) allRanRequestsObservedUsage() bool {
	observedAnyRequest := false
	for _, request := range r.Requests {
		if request.Skipped || !request.Ran {
			continue
		}
		observedAnyRequest = true
		if !request.UsageObserved {
			return false
		}
	}
	return observedAnyRequest
}

func openRouterSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}

func openRouterDiagnosticSmokeUsage(usage api.Usage) DiagnosticSmokeUsage {
	return providerdiag.SmokeUsageFromAPIUsage(usage)
}

func openRouterDiagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return providerdiag.SmokeCostFromEstimate(estimate)
}

func openRouterDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return providerdiag.NoopDiagnosticToolDefinitions(openRouterDiagnosticSmokeToolName, "OpenRouter")
}

func openRouterDiagnosticSmokeContentHasToolCall(content string) bool {
	return providerdiag.ContentHasToolCall(content, openRouterDiagnosticSmokeToolName)
}
