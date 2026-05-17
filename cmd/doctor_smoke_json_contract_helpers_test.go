package cmd

import (
	"strings"
	"testing"
)

func doctorSmokeJSONUsage(input, cached, output, reasoning, cacheCreation int) *doctorJSONSmokeUsage {
	return &doctorJSONSmokeUsage{
		InputTokens:         input,
		CachedInputTokens:   cached,
		OutputTokens:        output,
		ThinkingTokens:      reasoning,
		CacheCreationTokens: cacheCreation,
	}
}

func doctorSmokeJSONCost(usd float64, pricingUnavailable bool) *doctorJSONSmokeCost {
	return &doctorJSONSmokeCost{USD: usd, PricingUnavailable: pricingUnavailable}
}

func requireDoctorSmokeJSONContract(t *testing.T, smoke doctorJSONSmokeResult, want doctorSmokeJSONContract) {
	t.Helper()
	if !smoke.Ran {
		t.Fatalf("smoke.ran = false, want true: %+v", smoke)
	}
	if smoke.Route != want.route {
		t.Fatalf("smoke.route = %q, want %q", smoke.Route, want.route)
	}
	if smoke.ResponseID != want.responseID {
		t.Fatalf("smoke.response_id = %q, want %q", smoke.ResponseID, want.responseID)
	}
	if smoke.RequestID != want.requestID {
		t.Fatalf("smoke.request_id = %q, want %q", smoke.RequestID, want.requestID)
	}
	if smoke.Duration != want.duration {
		t.Fatalf("smoke.duration = %q, want %q", smoke.Duration, want.duration)
	}
	if smoke.UsageObserved != want.usageObserved {
		t.Fatalf("smoke.usage_observed = %t, want %t", smoke.UsageObserved, want.usageObserved)
	}
	if want.usage != nil {
		requireDoctorJSONSmokeUsage(t, smoke.Usage, *want.usage)
	}
	if want.cost != nil {
		requireDoctorJSONSmokeCost(t, smoke.Cost, want.cost.USD, want.cost.PricingUnavailable)
	}
	if smoke.CachedInputTokens != want.cachedInputTokens {
		t.Fatalf("smoke.cached_input_tokens = %d, want %d", smoke.CachedInputTokens, want.cachedInputTokens)
	}
	if smoke.WebSearchCallCount != want.webSearchCallCount || smoke.WebSearchCallFeeEstimate != want.webSearchCallFeeEstimate || smoke.WebSearchUsageObserved != want.webSearchUsageObserved || smoke.SearchResultTotalTokens != want.searchResultTotalTokens {
		t.Fatalf("smoke web search observation = calls:%d fee:%v observed:%t search_tokens:%d, want calls:%d fee:%v observed:%t search_tokens:%d",
			smoke.WebSearchCallCount,
			smoke.WebSearchCallFeeEstimate,
			smoke.WebSearchUsageObserved,
			smoke.SearchResultTotalTokens,
			want.webSearchCallCount,
			want.webSearchCallFeeEstimate,
			want.webSearchUsageObserved,
			want.searchResultTotalTokens,
		)
	}
	requireDoctorJSONSmokeRequestCount(t, smoke, len(want.requests))
	for i, request := range want.requests {
		requireDoctorSmokeJSONRequestContract(t, requireDoctorJSONSmokeRequestAt(t, smoke, i, request.name), request)
	}
}

func requireDoctorSmokeJSONRequestContract(t *testing.T, got doctorJSONSmokeRequest, want doctorSmokeJSONRequestContract) {
	t.Helper()
	if got.Ran != want.ran || got.Skipped != want.skipped {
		t.Fatalf("smoke request %s ran/skipped = %t/%t, want %t/%t: %+v", want.name, got.Ran, got.Skipped, want.ran, want.skipped, got)
	}
	if want.skipReasonContains != "" && !strings.Contains(got.SkipReason, want.skipReasonContains) {
		t.Fatalf("smoke request %s skip_reason = %q, want substring %q", want.name, got.SkipReason, want.skipReasonContains)
	}
	if got.ToolPayload != want.toolPayload ||
		got.ImagePayload != want.imagePayload ||
		got.WebSearchPayload != want.webSearchPayload ||
		got.RetentionPayload != want.retentionPayload ||
		got.ThinkingPayload != want.thinkingPayload {
		t.Fatalf("smoke request %s payload flags = tool:%t image:%t web:%t retention:%t thinking:%t, want tool:%t image:%t web:%t retention:%t thinking:%t",
			want.name,
			got.ToolPayload,
			got.ImagePayload,
			got.WebSearchPayload,
			got.RetentionPayload,
			got.ThinkingPayload,
			want.toolPayload,
			want.imagePayload,
			want.webSearchPayload,
			want.retentionPayload,
			want.thinkingPayload,
		)
	}
	if got.Route != want.route {
		t.Fatalf("smoke request %s route = %q, want %q", want.name, got.Route, want.route)
	}
	if got.ResponseID != want.responseID || got.RequestID != want.requestID || got.PreviousResponseID != want.previousResponseID {
		t.Fatalf("smoke request %s ids = response:%q request:%q previous:%q, want response:%q request:%q previous:%q",
			want.name,
			got.ResponseID,
			got.RequestID,
			got.PreviousResponseID,
			want.responseID,
			want.requestID,
			want.previousResponseID,
		)
	}
	if got.UsageObserved != want.usageObserved {
		t.Fatalf("smoke request %s usage_observed = %t, want %t", want.name, got.UsageObserved, want.usageObserved)
	}
	if want.usage != nil {
		requireDoctorJSONSmokeUsage(t, got.Usage, *want.usage)
	}
	if want.cost != nil {
		requireDoctorJSONSmokeCost(t, got.Cost, want.cost.USD, want.cost.PricingUnavailable)
	}
	if got.WebSearchCallCount != want.webSearchCallCount || got.WebSearchCallFeeEstimate != want.webSearchCallFeeEstimate || got.WebSearchUsageObserved != want.webSearchUsageObserved || got.SearchResultTotalTokens != want.searchResultTotalTokens {
		t.Fatalf(
			"smoke request %s web search observation = calls:%d fee:%v observed:%t search_tokens:%d, want calls:%d fee:%v observed:%t search_tokens:%d",
			want.name,
			got.WebSearchCallCount,
			got.WebSearchCallFeeEstimate,
			got.WebSearchUsageObserved,
			got.SearchResultTotalTokens,
			want.webSearchCallCount,
			want.webSearchCallFeeEstimate,
			want.webSearchUsageObserved,
			want.searchResultTotalTokens,
		)
	}
}
