package providerdiag

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ChatCompletionsSmokeRequest は OpenAI-compatible doctor smoke の request 単位を表す。
type ChatCompletionsSmokeRequest struct {
	Name         string
	SystemPrompt string
	UserContent  string
	ToolPayload  bool
}

// TextToolSmokeRequestOptions は text/tool smoke の request plan 入力を表す。
type TextToolSmokeRequestOptions struct {
	TextSmoke              bool
	ToolSmoke              bool
	FunctionCallingEnabled bool
	ProviderSlug           string
	ToolName               string
	ToolExpectedValue      string
}

// TextToolSmokeRequests は text/tool smoke の request plan を構築する。
func TextToolSmokeRequests(options TextToolSmokeRequestOptions) []ChatCompletionsSmokeRequest {
	textSmoke := options.TextSmoke || !options.ToolSmoke
	if options.ToolSmoke && !options.FunctionCallingEnabled {
		textSmoke = true
	}

	providerSlug := strings.TrimSpace(options.ProviderSlug)
	toolName := strings.TrimSpace(options.ToolName)
	toolExpectedValue := strings.TrimSpace(options.ToolExpectedValue)

	var requests []ChatCompletionsSmokeRequest
	if textSmoke {
		requests = append(requests, ChatCompletionsSmokeRequest{
			Name:         "text",
			SystemPrompt: "Reply briefly.",
			UserContent:  fmt.Sprintf("Reply with: xelyon %s doctor ok", providerSlug),
		})
	}
	if options.ToolSmoke {
		requests = append(requests, ChatCompletionsSmokeRequest{
			Name:         "tool",
			SystemPrompt: "Use the diagnostic tool.",
			UserContent:  fmt.Sprintf(`Call %s exactly once with {"value":"%s"} and do not answer in prose.`, toolName, toolExpectedValue),
			ToolPayload:  true,
		})
	}
	return requests
}

// NewChatCompletionsSmokeRequestContext は doctor smoke/preview 用の runtime context を構築する。
func NewChatCompletionsSmokeRequestContext(
	ctx context.Context,
	cfg *config.Config,
	request ChatCompletionsSmokeRequest,
	toolDefinitions []api.ToolDefinition,
	output io.Writer,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, toolDefinitions)
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	}
	return config.WithContext(requestCtx, cfg)
}

// RedactedBearerHeaders は request preview 用の認証済み Chat Completions headers を返す。
func RedactedBearerHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer <redacted>",
	}
}

// NoopDiagnosticToolDefinitions は doctor tool smoke 用の no-op tool 定義を返す。
func NoopDiagnosticToolDefinitions(toolName, providerLabel string) []api.ToolDefinition {
	return []api.ToolDefinition{{
		Name:        strings.TrimSpace(toolName),
		Description: fmt.Sprintf("No-op diagnostic probe used to verify %s tool calling.", strings.TrimSpace(providerLabel)),
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

// ContentHasToolCall は internal tool-call JSON に toolName が含まれるかを返す。
func ContentHasToolCall(content, toolName string) bool {
	return strings.Contains(content, `"tool":"`+strings.TrimSpace(toolName)+`"`)
}

// SmokeUsage は Chat Completions doctor smoke で観測した usage を表す。
type SmokeUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	ThinkingTokens      int `json:"thinking_tokens"`
	CachedInputTokens   int `json:"cached_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
}

// SmokeCost は Chat Completions doctor smoke の cost estimate を表す。
type SmokeCost struct {
	USD                float64 `json:"usd"`
	PricingUnavailable bool    `json:"pricing_unavailable"`
}

// SmokeUsageFromAPIUsage は api.Usage を doctor JSON DTO へ投影する。
func SmokeUsageFromAPIUsage(usage api.Usage) SmokeUsage {
	return SmokeUsage{
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}

// APIUsageFromSmokeUsage は doctor JSON DTO を api.Usage に戻す。
func APIUsageFromSmokeUsage(usage SmokeUsage) api.Usage {
	return api.Usage{
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		ThinkingTokens:      usage.ThinkingTokens,
		CachedInputTokens:   usage.CachedInputTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
	}
}

// AddSmokeUsage は request-level smoke usage を合算する。
func AddSmokeUsage(current, next SmokeUsage) SmokeUsage {
	usage := APIUsageFromSmokeUsage(current)
	usage.Add(APIUsageFromSmokeUsage(next))
	return SmokeUsageFromAPIUsage(usage)
}

// SmokeCostFromEstimate は cost estimate を doctor JSON DTO へ投影する。
func SmokeCostFromEstimate(estimate cost.CostEstimate) SmokeCost {
	return SmokeCost{
		USD:                estimate.Cost,
		PricingUnavailable: estimate.PricingUnavailable,
	}
}

// AddSmokeCost は request-level smoke cost を合算する。
func AddSmokeCost(current, next SmokeCost) SmokeCost {
	if next.PricingUnavailable {
		current.PricingUnavailable = true
	} else {
		current.USD += next.USD
	}
	return current
}
