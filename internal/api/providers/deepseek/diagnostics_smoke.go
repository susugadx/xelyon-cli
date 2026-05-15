package deepseek

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
	defaultDeepSeekDiagnosticSmokeTimeout         = 120 * time.Second
	defaultDeepSeekDiagnosticSmokeMaxOutputTokens = 64
	deepSeekDiagnosticSmokeToolName               = "xelyon_deepseek_doctor_probe"
)

type deepSeekDiagnosticSmokeRequest = providerdiag.ChatCompletionsSmokeRequest

func runDeepSeekDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultDeepSeekDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultDeepSeekDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := deepSeekDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(deepSeekAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	result := DiagnosticSmokeResult{Ran: true, Route: report.Route}
	started := time.Now()
	for _, request := range deepSeekDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			result.Requests = append(result.Requests, newDeepSeekDiagnosticSkippedToolSmokeRequest(request, report.Route))
			continue
		}

		requestResult, err := runDeepSeekDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
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

func deepSeekDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []deepSeekDiagnosticSmokeRequest {
	return providerdiag.TextToolSmokeRequests(providerdiag.TextToolSmokeRequestOptions{
		TextSmoke:              options.TextSmoke,
		ToolSmoke:              options.ToolSmoke,
		FunctionCallingEnabled: functionCallingEnabled,
		ProviderSlug:           "deepseek",
		ToolName:               deepSeekDiagnosticSmokeToolName,
		ToolExpectedValue:      "deepseek-tool-ok",
	})
}

func newDeepSeekDiagnosticSkippedToolSmokeRequest(request deepSeekDiagnosticSmokeRequest, route string) DiagnosticSmokeRequestResult {
	return DiagnosticSmokeRequestResult{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  deepSeekDiagnosticDisabledToolSkipReason(),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

func newDeepSeekDiagnosticSkippedToolPreviewRequest(request deepSeekDiagnosticSmokeRequest, route string) DiagnosticRequestPreviewRequest {
	return DiagnosticRequestPreviewRequest{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  deepSeekDiagnosticDisabledToolSkipReason(),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

func deepSeekDiagnosticDisabledToolSkipReason() string {
	return fmt.Sprintf("DeepSeek function calling payloads are disabled (%s=0)", deepSeekFunctionCallingEnv)
}

func runDeepSeekDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request deepSeekDiagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newDeepSeekDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	if request.ToolPayload {
		provider.SetToolChoice(deepSeekDiagnosticSmokeToolName)
	} else {
		provider.ClearToolChoice()
	}

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
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "deepseek", report.Model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:          request.Name,
		Ran:           true,
		ToolPayload:   request.ToolPayload,
		Route:         report.Route,
		Content:       strings.TrimSpace(content),
		Duration:      elapsed.String(),
		UsageObserved: usageObserved,
		Usage:         deepSeekDiagnosticSmokeUsage(usage),
		Cost:          deepSeekDiagnosticSmokeCost(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !deepSeekDiagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", deepSeekDiagnosticSmokeToolName)
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

func newDeepSeekDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request deepSeekDiagnosticSmokeRequest, output io.Writer) context.Context {
	return providerdiag.NewChatCompletionsSmokeRequestContext(ctx, cfg, request, deepSeekDiagnosticSmokeToolDefinitions(), output)
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

func deepSeekSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}

func deepSeekDiagnosticSmokeUsage(usage api.Usage) DiagnosticSmokeUsage {
	return providerdiag.SmokeUsageFromAPIUsage(usage)
}

func deepSeekDiagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return providerdiag.SmokeCostFromEstimate(estimate)
}

func deepSeekDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return providerdiag.NoopDiagnosticToolDefinitions(deepSeekDiagnosticSmokeToolName, "DeepSeek")
}

func deepSeekDiagnosticSmokeContentHasToolCall(content string) bool {
	return providerdiag.ContentHasToolCall(content, deepSeekDiagnosticSmokeToolName)
}
