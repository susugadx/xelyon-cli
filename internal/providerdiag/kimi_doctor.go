package providerdiag

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// KimiSmokeRequest は Kimi doctor の smoke request descriptor を表す。
type KimiSmokeRequest struct {
	Name             string
	ToolPayload      bool
	ImagePayload     bool
	WebSearchPayload bool
	Route            string
}

// KimiSmokeUsageObservation は Kimi smoke request で観測した usage を表す。
type KimiSmokeUsageObservation struct {
	InputTokens              int     `json:"input_tokens,omitempty"`
	OutputTokens             int     `json:"output_tokens,omitempty"`
	ThinkingTokens           int     `json:"thinking_tokens,omitempty"`
	CachedInputTokens        int     `json:"cached_input_tokens,omitempty"`
	WebSearchCallCount       int     `json:"web_search_call_count,omitempty"`
	WebSearchCallFeeEstimate float64 `json:"web_search_call_fee_estimate,omitempty"`
	SearchResultTotalTokens  int     `json:"search_result_total_tokens,omitempty"`
}

// KimiSmokeRequestResult は Kimi doctor の live smoke request 単位結果を表す。
type KimiSmokeRequestResult struct {
	Name                     string                    `json:"name"`
	Ran                      bool                      `json:"ran"`
	Skipped                  bool                      `json:"skipped,omitempty"`
	SkipReason               string                    `json:"skip_reason,omitempty"`
	ToolPayload              bool                      `json:"tool_payload,omitempty"`
	Content                  string                    `json:"content,omitempty"`
	Duration                 string                    `json:"duration,omitempty"`
	UsageObserved            bool                      `json:"usage_observed"`
	Usage                    KimiSmokeUsageObservation `json:"usage,omitempty"`
	PromptCacheKeyPresent    bool                      `json:"prompt_cache_key_present"`
	PromptCacheKey           string                    `json:"prompt_cache_key,omitempty"`
	ImagePayload             bool                      `json:"image_payload,omitempty"`
	WebSearchPayload         bool                      `json:"web_search_payload,omitempty"`
	WebSearchCallCount       int                       `json:"web_search_call_count,omitempty"`
	WebSearchCallFeeEstimate float64                   `json:"web_search_call_fee_estimate,omitempty"`
	WebSearchUsageObserved   bool                      `json:"web_search_usage_observed,omitempty"`
	SearchResultTotalTokens  int                       `json:"search_result_total_tokens,omitempty"`
	Error                    string                    `json:"error,omitempty"`
}

// KimiSmokeResult は Kimi doctor の live smoke 実行結果を表す。
type KimiSmokeResult struct {
	Ran                      bool                     `json:"ran"`
	ToolPayload              bool                     `json:"tool_payload"`
	ImagePayload             bool                     `json:"image_payload"`
	WebSearchPayload         bool                     `json:"web_search_payload"`
	Content                  string                   `json:"content,omitempty"`
	Duration                 string                   `json:"duration,omitempty"`
	UsageObserved            bool                     `json:"usage_observed"`
	CachedInputTokens        int                      `json:"cached_input_tokens"`
	WebSearchCallCount       int                      `json:"web_search_call_count,omitempty"`
	WebSearchCallFeeEstimate float64                  `json:"web_search_call_fee_estimate,omitempty"`
	WebSearchUsageObserved   bool                     `json:"web_search_usage_observed"`
	SearchResultTotalTokens  int                      `json:"search_result_total_tokens,omitempty"`
	Requests                 []KimiSmokeRequestResult `json:"requests,omitempty"`
}

// KimiRequestPreview は Kimi doctor の request preview を表す。
type KimiRequestPreview struct {
	Requests []KimiRequestPreviewRequest `json:"requests"`
}

// KimiRequestPreviewRequest は Kimi doctor の request preview 単位結果を表す。
type KimiRequestPreviewRequest struct {
	Name             string            `json:"name"`
	Skipped          bool              `json:"skipped,omitempty"`
	SkipReason       string            `json:"skip_reason,omitempty"`
	ToolPayload      bool              `json:"tool_payload,omitempty"`
	ImagePayload     bool              `json:"image_payload,omitempty"`
	WebSearchPayload bool              `json:"web_search_payload,omitempty"`
	Route            string            `json:"route"`
	Method           string            `json:"method,omitempty"`
	URL              string            `json:"url,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Body             any               `json:"body,omitempty"`
}

// NewKimiSmokeRequestResult は request descriptor から Kimi smoke request result の base を構築する。
func NewKimiSmokeRequestResult(request KimiSmokeRequest) KimiSmokeRequestResult {
	return KimiSmokeRequestResult{
		Name:             request.Name,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		WebSearchPayload: request.WebSearchPayload,
	}
}

// NewSkippedKimiSmokeRequest は skipped Kimi smoke entry を構築する。
func NewSkippedKimiSmokeRequest(request KimiSmokeRequest, skipReason string) KimiSmokeRequestResult {
	result := NewKimiSmokeRequestResult(request)
	result.Skipped = true
	result.SkipReason = strings.TrimSpace(skipReason)
	return result
}

// NewKimiPreviewRequest は request descriptor から Kimi request preview の base を構築する。
func NewKimiPreviewRequest(request KimiSmokeRequest) KimiRequestPreviewRequest {
	return KimiRequestPreviewRequest{
		Name:             request.Name,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		WebSearchPayload: request.WebSearchPayload,
		Route:            request.Route,
	}
}

// NewKimiRequestPreview は Kimi doctor の request preview entry を構築する。
func NewKimiRequestPreview(request KimiSmokeRequest, transport RequestPreviewTransport) KimiRequestPreviewRequest {
	result := NewKimiPreviewRequest(request)
	result.Method = transport.Method
	result.URL = transport.URL
	result.Headers = transport.Headers
	result.Body = transport.Body
	return result
}

// NewSkippedKimiPreviewRequest は skipped Kimi preview entry を構築する。
func NewSkippedKimiPreviewRequest(request KimiSmokeRequest, skipReason string) KimiRequestPreviewRequest {
	result := NewKimiPreviewRequest(request)
	result.Skipped = true
	result.SkipReason = strings.TrimSpace(skipReason)
	return result
}

// KimiSmokeUsageFromAPIUsage は api.Usage を Kimi doctor JSON DTO へ投影する。
func KimiSmokeUsageFromAPIUsage(usage api.Usage) KimiSmokeUsageObservation {
	return KimiSmokeUsageObservation{
		InputTokens:              usage.InputTokens,
		OutputTokens:             usage.OutputTokens,
		ThinkingTokens:           usage.ThinkingTokens,
		CachedInputTokens:        usage.CachedInputTokens,
		WebSearchCallCount:       usage.WebSearchCalls,
		WebSearchCallFeeEstimate: usage.StorageCost,
		SearchResultTotalTokens:  usage.WebSearchResultTokens,
	}
}

// WebSearchUsageObserved は Kimi $web_search call fee / call count が観測されたかを返す。
func (u KimiSmokeUsageObservation) WebSearchUsageObserved() bool {
	return u.WebSearchCallCount > 0
}

// AddKimiSmokeRequestResult は request-level smoke 結果を追加し Kimi 固有の観測値を summary に集約する。
func AddKimiSmokeRequestResult(result *KimiSmokeResult, request KimiSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	if strings.TrimSpace(request.Content) != "" {
		result.Content = request.Content
	}
	result.UsageObserved = result.UsageObserved || request.UsageObserved
	result.CachedInputTokens += request.Usage.CachedInputTokens
	result.WebSearchCallCount += request.Usage.WebSearchCallCount
	result.WebSearchCallFeeEstimate += request.Usage.WebSearchCallFeeEstimate
	result.SearchResultTotalTokens += request.Usage.SearchResultTotalTokens
	result.WebSearchUsageObserved = result.WebSearchUsageObserved || request.WebSearchUsageObserved
}
