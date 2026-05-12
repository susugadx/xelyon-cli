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
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	defaultGroqDiagnosticSmokeTimeout         = 120 * time.Second
	defaultGroqDiagnosticSmokeMaxOutputTokens = 64
	groqDiagnosticSmokeToolName               = "xelyon_groq_doctor_probe"
)

type groqDiagnosticSmokeRequest struct {
	Name         string
	SystemPrompt string
	UserContent  string
	ToolPayload  bool
}

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
			result.Requests = append(result.Requests, DiagnosticSmokeRequestResult{
				Name:        request.Name,
				Skipped:     true,
				SkipReason:  "Groq function calling payloads are disabled (GROQ_FUNCTION_CALLING=0)",
				ToolPayload: true,
				Route:       report.Route,
			})
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
	textSmoke := options.TextSmoke || !options.ToolSmoke
	if options.ToolSmoke && !functionCallingEnabled {
		textSmoke = true
	}

	var requests []groqDiagnosticSmokeRequest
	if textSmoke {
		requests = append(requests, groqDiagnosticSmokeRequest{
			Name:         "text",
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon groq doctor ok",
		})
	}
	if options.ToolSmoke {
		requests = append(requests, groqDiagnosticSmokeRequest{
			Name:         "tool",
			SystemPrompt: "Use the diagnostic tool.",
			UserContent:  `Call xelyon_groq_doctor_probe exactly once with {"value":"groq-tool-ok"} and do not answer in prose.`,
			ToolPayload:  true,
		})
	}
	return requests
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
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, groqDiagnosticSmokeToolDefinitions())
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	}
	return config.WithContext(requestCtx, cfg)
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

	var usage api.Usage
	usage.InputTokens = request.Usage.InputTokens
	usage.OutputTokens = request.Usage.OutputTokens
	usage.ThinkingTokens = request.Usage.ThinkingTokens
	usage.CachedInputTokens = request.Usage.CachedInputTokens
	usage.CacheCreationTokens = request.Usage.CacheCreationTokens

	var current api.Usage
	current.InputTokens = r.Usage.InputTokens
	current.OutputTokens = r.Usage.OutputTokens
	current.ThinkingTokens = r.Usage.ThinkingTokens
	current.CachedInputTokens = r.Usage.CachedInputTokens
	current.CacheCreationTokens = r.Usage.CacheCreationTokens
	current.Add(usage)
	r.Usage = groqDiagnosticSmokeUsage(current)
	if request.Cost.PricingUnavailable {
		r.Cost.PricingUnavailable = true
	} else {
		r.Cost.USD += request.Cost.USD
	}
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
	return DiagnosticSmokeUsage{
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}

func groqDiagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return DiagnosticSmokeCost{
		USD:                estimate.Cost,
		PricingUnavailable: estimate.PricingUnavailable,
	}
}

func groqDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        groqDiagnosticSmokeToolName,
		Description: "No-op diagnostic probe used to verify Groq tool calling.",
		Parameters: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
			"required": []string{"value"},
		},
	}}
}

func groqDiagnosticSmokeContentHasToolCall(content string) bool {
	return strings.Contains(content, `"tool":"`+groqDiagnosticSmokeToolName+`"`)
}
