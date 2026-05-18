package kimi

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func runKimiDiagnosticSmokeRequest(ctx context.Context, cfg *config.Config, provider *Provider, model string, request kimiDiagnosticSmokeRequest, output io.Writer) (DiagnosticSmokeRequestResult, error) {
	requestCfg := config.CloneConfig(cfg)
	requestCfg.Thinking.Enabled = request.Thinking
	requestCtx := newKimiDiagnosticSmokeRequestContext(ctx, requestCfg, provider, request, output)

	var usage api.Usage
	endpointUsageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		endpointUsageObserved = endpointUsageObserved || observed.HasTokenObservation()
	})

	started := time.Now()
	var promptCacheKey string
	var content string
	var err error
	if request.ImagePayload {
		image := kimiDiagnosticImage()
		built, buildErr := provider.buildImageChatCompletionsRequest(requestCtx, request.SystemPrompt, nil, request.UserContent, image, model)
		if buildErr != nil {
			err = buildErr
		} else {
			promptCacheKey = built.PromptCacheKey
			content, err = provider.ChatWithImage(requestCtx, request.SystemPrompt, nil, request.UserContent, image, model)
		}
	} else if request.WebSearchPayload {
		built := buildKimiWebSearchRequest(requestCtx, initialKimiWebSearchMessages(request.UserContent), model, "kimi")
		promptCacheKey = built.PromptCacheKey
		content, err = provider.webSearch(requestCtx, request.UserContent, model, "kimi")
	} else {
		history := []api.Message{{Role: "user", Content: request.UserContent}}
		built := provider.buildChatCompletionsRequest(
			requestCtx,
			request.SystemPrompt,
			history,
			model,
		)
		promptCacheKey = built.PromptCacheKey
		content, err = provider.ChatWithTools(
			requestCtx,
			request.SystemPrompt,
			history,
			model,
		)
	}
	elapsed := time.Since(started).Round(time.Millisecond)
	if err == nil && !request.ToolPayload && strings.TrimSpace(content) == "" {
		err = fmt.Errorf("%s smoke response content is empty", request.Name)
	}

	usageObservation := providerdiag.KimiSmokeUsageFromAPIUsage(usage)
	result := providerdiag.NewKimiSmokeRequestResult(request.kimiSmokeRequest())
	result.Ran = true
	result.Content = strings.TrimSpace(content)
	result.Duration = elapsed.String()
	result.UsageObserved = endpointUsageObserved
	result.Usage = usageObservation
	result.PromptCacheKeyPresent = strings.TrimSpace(promptCacheKey) != ""
	result.PromptCacheKey = promptCacheKey
	result.WebSearchCallCount = usageObservation.WebSearchCallCount
	result.WebSearchCallFeeEstimate = usageObservation.WebSearchCallFeeEstimate
	result.WebSearchUsageObserved = usageObservation.WebSearchUsageObserved()
	result.SearchResultTotalTokens = usageObservation.SearchResultTotalTokens
	if err != nil {
		result.Error = err.Error()
	}
	return result, err
}

func newKimiDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, provider *Provider, request kimiDiagnosticSmokeRequest, output io.Writer) context.Context {
	requestCtx := newKimiDiagnosticContext(ctx, cfg, request.Thinking, output)
	if request.SessionID != "" {
		requestCtx = api.WithPromptCacheScope(requestCtx, api.PromptCacheScope{SessionID: request.SessionID})
	}
	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, diagnosticSmokeToolDefinitions())
		provider.SetToolChoice(diagnosticSmokeToolName)
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		provider.ClearToolChoice()
	}
	provider.SetMCPTools(nil)
	provider.setDiagnosticFunctionCalling(request.ToolPayload)
	return requestCtx
}

func kimiDiagnosticImage() *api.ImageData {
	return &api.ImageData{
		MediaType: "image/png",
		Base64:    kimiDiagnosticPNGBase64,
	}
}

const kimiDiagnosticPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="

func (p *Provider) setDiagnosticFunctionCalling(enabled bool) {
	if p == nil {
		return
	}
	p.functionCalling = &enabled
}

func newKimiDiagnosticContext(ctx context.Context, cfg *config.Config, thinking bool, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCfg := config.CloneConfig(cfg)
	requestCfg.Thinking.Enabled = thinking
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = config.WithContext(requestCtx, requestCfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	return requestCtx
}

const diagnosticSmokeToolName = "xelyon_kimi_doctor_probe"

func diagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        diagnosticSmokeToolName,
		Description: "No-op diagnostic probe used to verify Kimi tool calling.",
		Parameters: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"value"},
		},
	}}
}

func diagnosticSmokeContentHasToolCall(content string) bool {
	return strings.Contains(content, `"tool":"`+diagnosticSmokeToolName+`"`)
}
