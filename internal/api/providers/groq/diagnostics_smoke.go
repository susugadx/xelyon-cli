package groq

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
	defaultGroqDiagnosticSmokeTimeout         = 120 * time.Second
	defaultGroqDiagnosticSmokeMaxOutputTokens = 64
	groqDiagnosticSmokeToolName               = "xelyon_groq_doctor_probe"
)

type groqDiagnosticSmokeRequest = providerdiag.ChatCompletionsSmokeRequest

func runGroqDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultGroqDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultGroqDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := groqDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(groqAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	result := DiagnosticSmokeResult{Ran: true, Route: report.Route}
	started := time.Now()
	for _, request := range groqDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			result.Requests = append(result.Requests, newGroqDiagnosticSkippedToolSmokeRequest(request, report.Route))
			continue
		}

		requestResult, err := runGroqDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
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

func groqDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []groqDiagnosticSmokeRequest {
	return providerdiag.TextToolSmokeRequests(providerdiag.TextToolSmokeRequestOptions{
		TextSmoke:              options.TextSmoke,
		ToolSmoke:              options.ToolSmoke,
		FunctionCallingEnabled: functionCallingEnabled,
		ProviderSlug:           "groq",
		ToolName:               groqDiagnosticSmokeToolName,
		ToolExpectedValue:      "groq-tool-ok",
	})
}

func newGroqDiagnosticSkippedToolSmokeRequest(request groqDiagnosticSmokeRequest, route string) DiagnosticSmokeRequestResult {
	return DiagnosticSmokeRequestResult{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  groqDiagnosticDisabledToolSkipReason(),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

func newGroqDiagnosticSkippedToolPreviewRequest(request groqDiagnosticSmokeRequest, route string) DiagnosticRequestPreviewRequest {
	return DiagnosticRequestPreviewRequest{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  groqDiagnosticDisabledToolSkipReason(),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

func groqDiagnosticDisabledToolSkipReason() string {
	return "Groq function calling payloads are disabled (GROQ_FUNCTION_CALLING=0)"
}

func runGroqDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request groqDiagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newGroqDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	if request.ToolPayload {
		provider.SetToolChoice(groqDiagnosticSmokeToolName)
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
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "groq", report.Model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:          request.Name,
		Ran:           true,
		ToolPayload:   request.ToolPayload,
		Route:         report.Route,
		Content:       strings.TrimSpace(content),
		Duration:      elapsed.String(),
		UsageObserved: usageObserved,
		Usage:         groqDiagnosticSmokeUsage(usage),
		Cost:          groqDiagnosticSmokeCost(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !groqDiagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", groqDiagnosticSmokeToolName)
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

func newGroqDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request groqDiagnosticSmokeRequest, output io.Writer) context.Context {
	return providerdiag.NewChatCompletionsSmokeRequestContext(ctx, cfg, request, groqDiagnosticSmokeToolDefinitions(), output)
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

func groqSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}

func groqDiagnosticSmokeUsage(usage api.Usage) DiagnosticSmokeUsage {
	return providerdiag.SmokeUsageFromAPIUsage(usage)
}

func groqDiagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return providerdiag.SmokeCostFromEstimate(estimate)
}

func groqDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return providerdiag.NoopDiagnosticToolDefinitions(groqDiagnosticSmokeToolName, "Groq")
}

func groqDiagnosticSmokeContentHasToolCall(content string) bool {
	return providerdiag.ContentHasToolCall(content, groqDiagnosticSmokeToolName)
}
