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

// TextToolSmokeRequest は provider doctor の text/tool smoke request 単位を表す。
type TextToolSmokeRequest struct {
	Name         string
	SystemPrompt string
	UserContent  string
	ToolPayload  bool
}

// TextToolSmokeRequestResult は provider doctor の text/tool smoke request 単位結果を表す。
type TextToolSmokeRequestResult struct {
	Name          string     `json:"name"`
	Ran           bool       `json:"ran"`
	Skipped       bool       `json:"skipped,omitempty"`
	SkipReason    string     `json:"skip_reason,omitempty"`
	ToolPayload   bool       `json:"tool_payload"`
	Route         string     `json:"route"`
	Content       string     `json:"content,omitempty"`
	Duration      string     `json:"duration,omitempty"`
	UsageObserved bool       `json:"usage_observed"`
	Usage         SmokeUsage `json:"usage"`
	Cost          SmokeCost  `json:"cost"`
	Error         string     `json:"error,omitempty"`
}

// TextToolSmokeResult は provider doctor の text/tool smoke 実行結果を表す。
type TextToolSmokeResult struct {
	Ran           bool                         `json:"ran"`
	ToolPayload   bool                         `json:"tool_payload"`
	Route         string                       `json:"route"`
	Content       string                       `json:"content,omitempty"`
	Duration      string                       `json:"duration,omitempty"`
	UsageObserved bool                         `json:"usage_observed"`
	Usage         SmokeUsage                   `json:"usage"`
	Cost          SmokeCost                    `json:"cost"`
	Requests      []TextToolSmokeRequestResult `json:"requests,omitempty"`
}

// TextToolRequestPreviewRequest は provider doctor の text/tool preview request 単位結果を表す。
type TextToolRequestPreviewRequest struct {
	Name        string            `json:"name"`
	Skipped     bool              `json:"skipped,omitempty"`
	SkipReason  string            `json:"skip_reason,omitempty"`
	ToolPayload bool              `json:"tool_payload"`
	Route       string            `json:"route"`
	Method      string            `json:"method,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        any               `json:"body,omitempty"`
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
func TextToolSmokeRequests(options TextToolSmokeRequestOptions) []TextToolSmokeRequest {
	textSmoke := options.TextSmoke || !options.ToolSmoke
	if options.ToolSmoke && !options.FunctionCallingEnabled {
		textSmoke = true
	}

	providerSlug := strings.TrimSpace(options.ProviderSlug)
	toolName := strings.TrimSpace(options.ToolName)
	toolExpectedValue := strings.TrimSpace(options.ToolExpectedValue)

	var requests []TextToolSmokeRequest
	if textSmoke {
		requests = append(requests, TextToolSmokeRequest{
			Name:         "text",
			SystemPrompt: "Reply briefly.",
			UserContent:  fmt.Sprintf("Reply with: xelyon %s doctor ok", providerSlug),
		})
	}
	if options.ToolSmoke {
		requests = append(requests, TextToolSmokeRequest{
			Name:         "tool",
			SystemPrompt: "Use the diagnostic tool.",
			UserContent:  fmt.Sprintf(`Call %s exactly once with {"value":"%s"} and do not answer in prose.`, toolName, toolExpectedValue),
			ToolPayload:  true,
		})
	}
	return requests
}

// NewSkippedTextToolSmokeRequest は function calling 無効時の skipped tool smoke entry を構築する。
func NewSkippedTextToolSmokeRequest(request TextToolSmokeRequest, route, skipReason string) TextToolSmokeRequestResult {
	return TextToolSmokeRequestResult{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  strings.TrimSpace(skipReason),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

// NewSkippedTextToolPreviewRequest は function calling 無効時の skipped tool preview entry を構築する。
func NewSkippedTextToolPreviewRequest(request TextToolSmokeRequest, route, skipReason string) TextToolRequestPreviewRequest {
	return TextToolRequestPreviewRequest{
		Name:        request.Name,
		Skipped:     true,
		SkipReason:  strings.TrimSpace(skipReason),
		ToolPayload: request.ToolPayload,
		Route:       route,
	}
}

// NewTextToolPreviewRequest は text/tool request preview entry を構築する。
func NewTextToolPreviewRequest(request TextToolSmokeRequest, route string, transport RequestPreviewTransport) TextToolRequestPreviewRequest {
	return TextToolRequestPreviewRequest{
		Name:        request.Name,
		ToolPayload: request.ToolPayload,
		Route:       route,
		Method:      transport.Method,
		URL:         transport.URL,
		Headers:     transport.Headers,
		Body:        transport.Body,
	}
}

// AddTextToolSmokeRequestResult は request-level smoke 結果を追加し summary に集約する。
func AddTextToolSmokeRequestResult(result *TextToolSmokeResult, request TextToolSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	if request.ToolPayload {
		result.ToolPayload = true
	}
	if result.Route == "" {
		result.Route = request.Route
	}
	if strings.TrimSpace(result.Content) == "" {
		result.Content = request.Content
	}

	result.Usage = AddSmokeUsage(result.Usage, request.Usage)
	result.Cost = AddSmokeCost(result.Cost, request.Cost)
	result.UsageObserved = AllTextToolSmokeRequestsObservedUsage(result.Requests)
}

// AllTextToolSmokeRequestsObservedUsage は skipped 以外の実行済み request すべてで usage が観測されたかを返す。
func AllTextToolSmokeRequestsObservedUsage(requests []TextToolSmokeRequestResult) bool {
	observedAnyRequest := false
	for _, request := range requests {
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

// NewTextToolSmokeRequestContext は doctor smoke/preview 用の runtime context を構築する。
func NewTextToolSmokeRequestContext(
	ctx context.Context,
	cfg *config.Config,
	request TextToolSmokeRequest,
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

// RedactedBearerHeaders は request preview 用の redacted bearer 認証 headers を返す。
func RedactedBearerHeaders() map[string]string {
	headers := JSONHeaders()
	headers["Authorization"] = "Bearer <redacted>"
	return headers
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

// SmokeUsage は doctor smoke で観測した usage を表す。
type SmokeUsage struct {
	InputTokens         int    `json:"input_tokens"`
	OutputTokens        int    `json:"output_tokens"`
	ThinkingTokens      int    `json:"thinking_tokens"`
	CachedInputTokens   int    `json:"cached_input_tokens"`
	CacheCreationTokens int    `json:"cache_creation_tokens"`
	BillingServiceTier  string `json:"billing_service_tier,omitempty"`
}

// SmokeCost は doctor smoke の cost estimate を表す。
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
		BillingServiceTier:  usage.BillingServiceTier,
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
		BillingServiceTier:  usage.BillingServiceTier,
	}
}

// AddSmokeUsage は request-level smoke usage を合算する。
func AddSmokeUsage(current, next SmokeUsage) SmokeUsage {
	usage := APIUsageFromSmokeUsage(current)
	usage.Add(APIUsageFromSmokeUsage(next))
	result := SmokeUsageFromAPIUsage(usage)
	result.BillingServiceTier = mergeSmokeBillingServiceTier(current.BillingServiceTier, next.BillingServiceTier)
	return result
}

func mergeSmokeBillingServiceTier(current, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	switch {
	case current == "":
		return next
	case next == "":
		return current
	case current == next:
		return current
	default:
		return "mixed"
	}
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
