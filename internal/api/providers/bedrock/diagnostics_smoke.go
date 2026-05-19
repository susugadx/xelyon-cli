package bedrock

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
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const defaultBedrockDiagnosticSmokeTimeout = 120 * time.Second

func runBedrockDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions, requestPlan bedrockDiagnosticRequestPlan) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultBedrockDiagnosticSmokeTimeout
	}
	maxOutputTokens := bedrockDiagnosticRequestMaxOutputTokens(options)

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := bedrockDiagnosticSmokeConfig(cfg, report, maxOutputTokens)

	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider, err := newBedrockDiagnosticSmokeProvider(smokeCfg, report.Region, options)
	if err != nil {
		return DiagnosticSmokeResult{Ran: true}, err
	}
	result := DiagnosticSmokeResult{Ran: true}
	for _, request := range requestPlan.Requests {
		if skipReason, ok := bedrockDiagnosticRequestSkipReason(report, request); ok {
			providerdiag.AddInvocationSmokeRequestResult(&result, newBedrockDiagnosticSkippedSmokeRequest(request, skipReason))
			continue
		}
		requestResult, requestErr := runBedrockDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report.Model, request, output)
		providerdiag.AddInvocationSmokeRequestResult(&result, requestResult)
		if requestErr != nil {
			return result, requestErr
		}
	}
	return result, nil
}

func bedrockDiagnosticSmokeConfig(cfg *config.Config, report DiagnosticReport, maxOutputTokens int) *config.Config {
	smokeCfg := config.CloneConfig(cfg)
	catalogModel := strings.TrimSpace(report.CatalogModel)
	if catalogModel == "" {
		catalogModel = report.Model
	}
	smokeCfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel: report.Model,
		CatalogModel: catalogModel,
		ModelOverrides: map[string]config.ModelOverride{
			report.Model: {
				CatalogModel:    catalogModel,
				MaxOutputTokens: maxOutputTokens,
			},
		},
	})
	smokeCfg.PromptCache.Enabled = false
	smokeCfg.Compression.ClaudeCompaction = false
	return smokeCfg
}

func newBedrockDiagnosticSmokeProvider(cfg *config.Config, region string, options DiagnosticOptions) (*Provider, error) {
	if options.invokeClient != nil || options.converseClient != nil {
		return &Provider{
			client:         options.invokeClient,
			converseClient: options.converseClient,
			region:         region,
			runtimeConfig:  cfg,
		}, nil
	}
	provider, err := New()
	if err != nil {
		return nil, err
	}
	provider.SetRuntimeConfig(cfg)
	return provider, nil
}

func runBedrockDiagnosticSmokeRequest(ctx context.Context, cfg *config.Config, provider *Provider, model string, request bedrockDiagnosticSmokeRequest, output io.Writer) (DiagnosticSmokeRequestResult, error) {
	requestCfg := bedrockDiagnosticRequestConfig(cfg, request)
	requestCtx := newBedrockDiagnosticSmokeRequestContext(ctx, requestCfg, request, output)

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})
	provider.SetMCPTools(nil)

	started := time.Now()
	var content string
	var err error
	if request.ImagePayload {
		content, err = provider.ChatWithImage(requestCtx, request.SystemPrompt, nil, request.UserContent, bedrockDiagnosticImage(), model)
	} else {
		content, err = provider.ChatWithTools(requestCtx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, model)
	}
	elapsed := time.Since(started).Round(time.Millisecond)
	requestID := provider.lastBedrockRequestID()
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(requestCfg, "bedrock", model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:            request.Name,
		Ran:             true,
		ToolPayload:     request.ToolPayload,
		ImagePayload:    request.ImagePayload,
		ThinkingEnabled: request.ThinkingEnabled,
		Content:         strings.TrimSpace(content),
		RequestID:       requestID,
		Duration:        elapsed.String(),
		UsageObserved:   usageObserved,
		Usage:           providerdiag.SmokeUsageFromAPIUsage(usage),
		Cost:            providerdiag.SmokeCostFromEstimate(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !diagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s tool call", diagnosticSmokeToolName)
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

func newBedrockDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request bedrockDiagnosticSmokeRequest, output io.Writer) context.Context {
	requestCtx := newBedrockDiagnosticContext(ctx, cfg, output)
	if request.ToolPayload {
		return api.WithToolDefinitions(requestCtx, diagnosticSmokeToolDefinitions())
	}
	requestCtx = api.WithToolDefinitions(requestCtx, nil)
	return api.WithToolUseDisabled(requestCtx)
}

func newBedrockDiagnosticContext(ctx context.Context, cfg *config.Config, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = config.WithContext(requestCtx, cfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	return requestCtx
}

func bedrockDiagnosticImage() *api.ImageData {
	const redPNG16x16 = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAAFklEQVR42mP4z8BAEmIY1TCqYfhqAACQ+f8B8u7oVwAAAABJRU5ErkJggg=="
	return &api.ImageData{
		Path:      "bedrock-doctor-red-16x16.png",
		MediaType: "image/png",
		Base64:    redPNG16x16,
		Size:      79,
	}
}

const diagnosticSmokeToolName = "xelyon_bedrock_doctor_probe"

func diagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        diagnosticSmokeToolName,
		Description: "No-op diagnostic probe used to verify Bedrock tool calling.",
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
