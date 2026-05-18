package providerdiag

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestKimiSmokeRequestBuilders(t *testing.T) {
	request := KimiSmokeRequest{
		Name:             "web_search_smoke",
		ToolPayload:      true,
		ImagePayload:     true,
		WebSearchPayload: true,
		Route:            "chat_completions_web_search",
	}

	base := NewKimiSmokeRequestResult(request)
	if base.Name != request.Name || !base.ToolPayload || !base.ImagePayload || !base.WebSearchPayload {
		t.Fatalf("base smoke request = %+v, want descriptor flags", base)
	}

	skipped := NewSkippedKimiSmokeRequest(request, " KIMI_FUNCTION_CALLING=0 ")
	if !skipped.Skipped || skipped.SkipReason != "KIMI_FUNCTION_CALLING=0" || !skipped.ToolPayload {
		t.Fatalf("skipped smoke request = %+v, want trimmed skipped tool request", skipped)
	}

	preview := NewKimiPreviewRequest(request)
	if preview.Name != request.Name || preview.Route != request.Route || !preview.ToolPayload || !preview.ImagePayload || !preview.WebSearchPayload {
		t.Fatalf("preview request = %+v, want descriptor flags and route", preview)
	}

	skippedPreview := NewSkippedKimiPreviewRequest(request, " disabled ")
	if !skippedPreview.Skipped || skippedPreview.SkipReason != "disabled" || skippedPreview.Route != request.Route {
		t.Fatalf("skipped preview request = %+v, want trimmed skipped preview with route", skippedPreview)
	}
}

func TestKimiSmokeUsageFromAPIUsage(t *testing.T) {
	got := KimiSmokeUsageFromAPIUsage(api.Usage{
		InputTokens:           10,
		OutputTokens:          4,
		ThinkingTokens:        2,
		CachedInputTokens:     3,
		WebSearchCalls:        2,
		StorageCost:           0.01,
		WebSearchResultTokens: 55,
	})
	want := KimiSmokeUsageObservation{
		InputTokens:              10,
		OutputTokens:             4,
		ThinkingTokens:           2,
		CachedInputTokens:        3,
		WebSearchCallCount:       2,
		WebSearchCallFeeEstimate: 0.01,
		SearchResultTotalTokens:  55,
	}
	if got != want {
		t.Fatalf("KimiSmokeUsageFromAPIUsage() = %+v, want %+v", got, want)
	}
	if !got.WebSearchUsageObserved() {
		t.Fatalf("WebSearchUsageObserved() = false, want true for observed call count")
	}
	if (KimiSmokeUsageObservation{}).WebSearchUsageObserved() {
		t.Fatalf("WebSearchUsageObserved() = true, want false without call count")
	}
}

func TestAddKimiSmokeRequestResultAggregatesKimiObservations(t *testing.T) {
	result := KimiSmokeResult{Ran: true, Content: "first"}

	AddKimiSmokeRequestResult(&result, KimiSmokeRequestResult{
		Name:          "text",
		Ran:           true,
		Content:       "text ok",
		UsageObserved: true,
		Usage: KimiSmokeUsageObservation{
			CachedInputTokens: 2,
		},
	})
	AddKimiSmokeRequestResult(&result, KimiSmokeRequestResult{
		Name:             "web_search_smoke",
		Ran:              true,
		WebSearchPayload: true,
		Content:          "web ok",
		Usage: KimiSmokeUsageObservation{
			CachedInputTokens:        5,
			WebSearchCallCount:       2,
			WebSearchCallFeeEstimate: 0.01,
			SearchResultTotalTokens:  55,
		},
		WebSearchCallCount:       2,
		WebSearchCallFeeEstimate: 0.01,
		WebSearchUsageObserved:   true,
		SearchResultTotalTokens:  55,
	})
	AddKimiSmokeRequestResult(&result, NewSkippedKimiSmokeRequest(KimiSmokeRequest{
		Name:        "tool_smoke",
		ToolPayload: true,
	}, "disabled"))

	if len(result.Requests) != 3 {
		t.Fatalf("requests = %+v, want 3 entries", result.Requests)
	}
	if result.Content != "web ok" {
		t.Fatalf("content = %q, want last non-empty request content", result.Content)
	}
	if !result.UsageObserved {
		t.Fatalf("UsageObserved = false, want true when any Kimi endpoint usage was observed")
	}
	if result.CachedInputTokens != 7 {
		t.Fatalf("CachedInputTokens = %d, want 7", result.CachedInputTokens)
	}
	if result.WebSearchCallCount != 2 || result.WebSearchCallFeeEstimate != 0.01 || !result.WebSearchUsageObserved || result.SearchResultTotalTokens != 55 {
		t.Fatalf("web search aggregate = calls:%d fee:%f observed:%t tokens:%d, want 2/0.01/true/55",
			result.WebSearchCallCount,
			result.WebSearchCallFeeEstimate,
			result.WebSearchUsageObserved,
			result.SearchResultTotalTokens,
		)
	}
	if result.ToolPayload {
		t.Fatalf("ToolPayload = true, want skipped request ignored by summary")
	}
}
