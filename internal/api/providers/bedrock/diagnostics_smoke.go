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
		if skipReason, ok := bedrockDiagnosticSmokeSkipReason(report, request); ok {
			result.Requests = append(result.Requests, newBedrockDiagnosticSkippedSmokeRequest(request, skipReason))
			continue
		}
		requestResult, requestErr := runBedrockDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report.Model, request, output)
		result.Requests = append(result.Requests, requestResult)
		result.addRequestObservation(requestResult)
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
	requestCfg := config.CloneConfig(cfg)
	requestCfg.Thinking.Enabled = request.ThinkingEnabled
	if request.ThinkingEnabled && strings.TrimSpace(requestCfg.Thinking.Level) == "" {
		requestCfg.Thinking.Level = "low"
	}
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
		Usage:           diagnosticSmokeUsage(usage),
		Cost:            diagnosticSmokeCost(costEstimate),
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

func (r *DiagnosticSmokeResult) addRequestObservation(request DiagnosticSmokeRequestResult) {
	if request.Skipped {
		return
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
	r.Usage = diagnosticSmokeUsage(current)
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

func diagnosticSmokeUsage(usage api.Usage) DiagnosticSmokeUsage {
	return DiagnosticSmokeUsage{
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}

func diagnosticSmokeCost(estimate cost.CostEstimate) DiagnosticSmokeCost {
	return DiagnosticSmokeCost{
		USD:                estimate.Cost,
		PricingUnavailable: estimate.PricingUnavailable,
	}
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
