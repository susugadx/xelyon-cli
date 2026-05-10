package openai

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
	defaultOpenAIDiagnosticSmokeTimeout       = 120 * time.Second
	defaultOpenAIDiagnosticSmokeMaxOutputToks = 64
	openAIDiagnosticSmokeToolName             = "xelyon_openai_doctor_probe"
)

type openAIDiagnosticSmokeRequest struct {
	Name         string
	SystemPrompt string
	UserContent  string
	ToolPayload  bool
}

func runOpenAIDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultOpenAIDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultOpenAIDiagnosticSmokeMaxOutputToks
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := openAIDiagnosticConfigWithModelPolicy(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	smokeCfg.Responses.Store = false
	smokeCfg.Responses.PersistResponseID = false
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(openAIAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()
	defer provider.ClearResponseID()
	result := DiagnosticSmokeResult{Ran: true, Route: report.Route}
	started := time.Now()
	for _, request := range openAIDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			result.Requests = append(result.Requests, DiagnosticSmokeRequestResult{
				Name:        request.Name,
				Skipped:     true,
				SkipReason:  "OpenAI function calling payloads are disabled (OPENAI_FUNCTION_CALLING=0)",
				ToolPayload: true,
				Route:       report.Route,
			})
			continue
		}

		requestResult, err := runOpenAIDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
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

func openAIDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []openAIDiagnosticSmokeRequest {
	textSmoke := options.TextSmoke || !options.ToolSmoke
	if options.ToolSmoke && !functionCallingEnabled {
		textSmoke = true
	}

	var requests []openAIDiagnosticSmokeRequest
	if textSmoke {
		requests = append(requests, openAIDiagnosticSmokeRequest{
			Name:         "text",
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon openai doctor ok",
		})
	}
	if options.ToolSmoke {
		requests = append(requests, openAIDiagnosticSmokeRequest{
			Name:         "tool",
			SystemPrompt: "Use the diagnostic tool.",
			UserContent:  `Call xelyon_openai_doctor_probe exactly once with {"value":"openai-tool-ok"} and do not answer in prose.`,
			ToolPayload:  true,
		})
	}
	return requests
}

func runOpenAIDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request openAIDiagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newOpenAIDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	if request.ToolPayload {
		provider.SetToolChoice(openAIDiagnosticSmokeToolName)
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
	content, responseID, err := runOpenAIDiagnosticProviderSmokeRequest(
		requestCtx,
		provider,
		report.Route,
		request.SystemPrompt,
		[]api.Message{{Role: "user", Content: request.UserContent}},
		report.Model,
	)
	elapsed := time.Since(started).Round(time.Millisecond)
	provider.ClearResponseID()
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "openai", report.Model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:          request.Name,
		Ran:           true,
		ToolPayload:   request.ToolPayload,
		Route:         report.Route,
		Content:       strings.TrimSpace(content),
		ResponseID:    strings.TrimSpace(responseID),
		Duration:      elapsed.String(),
		UsageObserved: usageObserved,
		Usage:         openAIDiagnosticSmokeUsage(usage),
		Cost:          openAIDiagnosticSmokeCost(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !openAIDiagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", openAIDiagnosticSmokeToolName)
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

func newOpenAIDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request openAIDiagnosticSmokeRequest, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, openAIDiagnosticSmokeToolDefinitions())
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	}
	return config.WithContext(requestCtx, cfg)
}

func runOpenAIDiagnosticProviderSmokeRequest(
	ctx context.Context,
	provider *Provider,
	route string,
	systemPrompt string,
	history []api.Message,
	model string,
) (string, string, error) {
	if provider == nil {
		return "", "", fmt.Errorf("openai diagnostic smoke provider is nil")
	}
	if route == DiagnosticRouteChatCompletions {
		content, err := provider.chatWithCompletions(ctx, systemPrompt, history, model)
		return content, "", err
	}
	provider.responsesLocalAutoCompressSkip = false
	return provider.runResponsesRequest(ctx, responsesRequestRunOptions{
		URL: resolveResponsesAPIURL(),
		BuildRequest: func() ResponsesRequest {
			return provider.buildChatResponsesRequest(ctx, systemPrompt, history, model)
		},
		DebugName:   "OpenAI",
		Debug:       os.Getenv("XELYON_DEBUG_OPENAI") == "1",
		DebugWriter: api.ErrorWriterFromContext(ctx),
	})
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
	if strings.TrimSpace(r.ResponseID) == "" {
		r.ResponseID = request.ResponseID
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
	r.Usage = openAIDiagnosticSmokeUsage(current)
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

func openAIDiagnosticSmokeUsage(usage api.Usage) DiagnosticSmokeUsage {
	return DiagnosticSmokeUsage{
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}

func openAIDiagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return DiagnosticSmokeCost{
		USD:                estimate.Cost,
		PricingUnavailable: estimate.PricingUnavailable,
	}
}

func openAIDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        openAIDiagnosticSmokeToolName,
		Description: "No-op diagnostic probe used to verify OpenAI tool calling.",
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

func openAIDiagnosticSmokeContentHasToolCall(content string) bool {
	return strings.Contains(content, `"tool":"`+openAIDiagnosticSmokeToolName+`"`)
}
