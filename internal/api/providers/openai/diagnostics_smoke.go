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
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	defaultOpenAIDiagnosticSmokeTimeout       = 120 * time.Second
	defaultOpenAIDiagnosticSmokeMaxOutputToks = 64
	openAIDiagnosticSmokeToolName             = "xelyon_openai_doctor_probe"
)

type openAIDiagnosticSmokeRequest = providerdiag.ResponsesSmokeRequest

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

	baseSmokeCfg := openAIDiagnosticConfigWithModelPolicy(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	baseSmokeCfg.Responses.Store = false
	baseSmokeCfg.Responses.PersistResponseID = false
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
			providerdiag.AddRoutedResponsesSmokeRequestResult(
				&result,
				providerdiag.NewSkippedRoutedResponsesSmokeRequest(
					request,
					report.Route,
					"OpenAI function calling payloads are disabled (OPENAI_FUNCTION_CALLING=0)",
				),
			)
			continue
		}

		requestCfg := openAIDiagnosticSmokeRequestConfig(baseSmokeCfg, request)
		requestResult, err := runOpenAIDiagnosticSmokeRequest(smokeCtx, requestCfg, provider, report, request, output)
		providerdiag.AddRoutedResponsesSmokeRequestResult(&result, requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func openAIDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []openAIDiagnosticSmokeRequest {
	textSmoke := options.TextSmoke || (!options.ToolSmoke && !options.RetentionSmoke)
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
	if options.RetentionSmoke {
		requests = append(requests,
			openAIDiagnosticSmokeRequest{
				Name:             "retention_initial",
				SystemPrompt:     "Reply briefly.",
				UserContent:      "Reply with: xelyon openai retention initial ok",
				RetentionPayload: true,
			},
			openAIDiagnosticSmokeRequest{
				Name:             "retention_followup",
				SystemPrompt:     "Reply briefly.",
				UserContent:      "Reply with: xelyon openai retention followup ok",
				RetentionPayload: true,
			},
		)
	}
	return requests
}

func openAIDiagnosticSmokeRequestConfig(base *config.Config, request openAIDiagnosticSmokeRequest) *config.Config {
	cfg := config.CloneConfig(base)
	if request.RetentionPayload {
		cfg.Responses.Store = true
	}
	return cfg
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

	observedRequests, restoreObserver := observeOpenAIDiagnosticResponsesRequests(provider)
	defer restoreObserver()

	started := time.Now()
	content, responseID, err := runOpenAIDiagnosticProviderSmokeRequest(
		requestCtx,
		provider,
		report.Route,
		request.SystemPrompt,
		[]api.Message{{Role: "user", Content: request.UserContent}},
		report.Model,
		request.RetentionPayload,
	)
	elapsed := time.Since(started).Round(time.Millisecond)
	if !request.RetentionPayload {
		provider.ClearResponseID()
	}
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "openai", report.Model, usage)
	observed := observedRequests()
	previousResponseID := openAIDiagnosticObservedPreviousResponseID(observed)

	result := DiagnosticSmokeRequestResult{
		Name:               request.Name,
		Ran:                true,
		ToolPayload:        request.ToolPayload,
		RetentionPayload:   request.RetentionPayload,
		Route:              report.Route,
		Content:            strings.TrimSpace(content),
		ResponseID:         strings.TrimSpace(responseID),
		PreviousResponseID: previousResponseID,
		Duration:           elapsed.String(),
		UsageObserved:      usageObserved,
		Usage:              providerdiag.SmokeUsageFromAPIUsage(usage),
		Cost:               providerdiag.SmokeCostFromEstimate(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.RetentionPayload {
		if err := validateOpenAIDiagnosticRetentionSmokeRequest(request, observed, result); err != nil {
			result.Error = err.Error()
			return result, err
		}
		if strings.TrimSpace(content) == "" {
			result.Error = fmt.Sprintf("%s smoke response content is empty", request.Name)
			return result, errors.New(result.Error)
		}
		return result, nil
	}
	if request.ToolPayload {
		if !providerdiag.ContentHasToolCall(content, openAIDiagnosticSmokeToolName) {
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

func observeOpenAIDiagnosticResponsesRequests(provider *Provider) (func() []ResponsesRequest, func()) {
	if provider == nil {
		return func() []ResponsesRequest { return nil }, func() {}
	}
	previousObserver := provider.responsesRequestObserver
	observed := make([]ResponsesRequest, 0, 1)
	provider.responsesRequestObserver = func(request ResponsesRequest) {
		observed = append(observed, request)
		if previousObserver != nil {
			previousObserver(request)
		}
	}
	snapshot := func() []ResponsesRequest {
		return append([]ResponsesRequest(nil), observed...)
	}
	restore := func() {
		provider.responsesRequestObserver = previousObserver
	}
	return snapshot, restore
}

func openAIDiagnosticObservedPreviousResponseID(requests []ResponsesRequest) string {
	if len(requests) == 0 {
		return ""
	}
	return strings.TrimSpace(requests[0].PreviousResponseID)
}

func validateOpenAIDiagnosticRetentionSmokeRequest(request openAIDiagnosticSmokeRequest, observed []ResponsesRequest, result DiagnosticSmokeRequestResult) error {
	if len(observed) == 0 {
		return fmt.Errorf("%s smoke did not build a Responses request", request.Name)
	}
	if !observed[0].Store {
		return fmt.Errorf("%s smoke request did not set responses.store=true", request.Name)
	}
	if request.Name != "retention_followup" {
		return nil
	}
	if strings.TrimSpace(result.PreviousResponseID) == "" {
		return fmt.Errorf("retention followup did not send previous_response_id")
	}
	if len(observed) > 1 {
		return fmt.Errorf("retention followup retried without proving previous_response_id was accepted")
	}
	return nil
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
	retentionPayload bool,
) (string, string, error) {
	if provider == nil {
		return "", "", fmt.Errorf("openai diagnostic smoke provider is nil")
	}
	if route == DiagnosticRouteChatCompletions {
		content, err := provider.chatWithCompletions(ctx, systemPrompt, history, model)
		return content, "", err
	}
	if retentionPayload {
		content, err := provider.chatWithResponses(ctx, systemPrompt, history, model)
		return content, provider.GetResponseID(), err
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
