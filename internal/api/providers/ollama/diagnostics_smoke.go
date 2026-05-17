package ollama

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	defaultOllamaDiagnosticSmokeTimeout         = 120 * time.Second
	defaultOllamaDiagnosticSmokeMaxOutputTokens = 64
	ollamaDiagnosticSmokeToolName               = "xelyon_ollama_doctor_probe"
)

type ollamaDiagnosticSmokeRequest = providerdiag.TextToolSmokeRequest

func runOllamaDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultOllamaDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultOllamaDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := ollamaDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(report.APIURL)
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	result := DiagnosticSmokeResult{Ran: true, Route: report.Route}
	started := time.Now()
	for _, request := range ollamaDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			result.Requests = append(result.Requests, newOllamaDiagnosticSkippedToolSmokeRequest(request, report.Route))
			continue
		}

		requestResult, err := runOllamaDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
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

func ollamaDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []ollamaDiagnosticSmokeRequest {
	return providerdiag.TextToolSmokeRequests(providerdiag.TextToolSmokeRequestOptions{
		TextSmoke:              options.TextSmoke,
		ToolSmoke:              options.ToolSmoke,
		FunctionCallingEnabled: functionCallingEnabled,
		ProviderSlug:           "ollama",
		ToolName:               ollamaDiagnosticSmokeToolName,
		ToolExpectedValue:      "ollama-tool-ok",
	})
}

func newOllamaDiagnosticSkippedToolSmokeRequest(request ollamaDiagnosticSmokeRequest, route string) DiagnosticSmokeRequestResult {
	return DiagnosticSmokeRequestResult{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  ollamaDiagnosticDisabledToolSkipReason(),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

func newOllamaDiagnosticSkippedToolPreviewRequest(request ollamaDiagnosticSmokeRequest, route string) DiagnosticRequestPreviewRequest {
	return DiagnosticRequestPreviewRequest{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  ollamaDiagnosticDisabledToolSkipReason(),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

func ollamaDiagnosticDisabledToolSkipReason() string {
	return "Ollama function calling payloads are disabled (OLLAMA_FUNCTION_CALLING=0)"
}

func runOllamaDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request ollamaDiagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newOllamaDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	if request.ToolPayload {
		provider.SetToolChoice(ollamaDiagnosticSmokeToolName)
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
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "ollama", report.Model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:          request.Name,
		Ran:           true,
		ToolPayload:   request.ToolPayload,
		Route:         report.Route,
		Content:       strings.TrimSpace(content),
		Duration:      elapsed.String(),
		UsageObserved: usageObserved,
		Usage:         ollamaDiagnosticSmokeUsage(usage),
		Cost:          ollamaDiagnosticSmokeCost(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !ollamaDiagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", ollamaDiagnosticSmokeToolName)
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

func newOllamaDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request ollamaDiagnosticSmokeRequest, output io.Writer) context.Context {
	return providerdiag.NewTextToolSmokeRequestContext(ctx, cfg, request, ollamaDiagnosticSmokeToolDefinitions(), output)
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

func ollamaSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}

func ollamaDiagnosticSmokeUsage(usage api.Usage) DiagnosticSmokeUsage {
	return providerdiag.SmokeUsageFromAPIUsage(usage)
}

func ollamaDiagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return providerdiag.SmokeCostFromEstimate(estimate)
}

func ollamaDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return providerdiag.NoopDiagnosticToolDefinitions(ollamaDiagnosticSmokeToolName, "Ollama")
}

func ollamaDiagnosticSmokeContentHasToolCall(content string) bool {
	return providerdiag.ContentHasToolCall(content, ollamaDiagnosticSmokeToolName)
}
