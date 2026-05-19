package cmd

import (
	"bytes"
	"testing"
)

type doctorJSONSmokeReport struct {
	Smoke doctorJSONSmokeResult `json:"smoke"`
}

type doctorJSONSmokeResult struct {
	Ran                      bool                     `json:"ran"`
	Route                    string                   `json:"route"`
	ResponseID               string                   `json:"response_id"`
	RequestID                string                   `json:"request_id"`
	Duration                 string                   `json:"duration"`
	RetentionPayload         bool                     `json:"retention_payload"`
	UsageObserved            bool                     `json:"usage_observed"`
	WebSearchCallCount       int                      `json:"web_search_call_count"`
	WebSearchCallFeeEstimate float64                  `json:"web_search_call_fee_estimate"`
	WebSearchUsageObserved   bool                     `json:"web_search_usage_observed"`
	CachedInputTokens        int                      `json:"cached_input_tokens"`
	SearchResultTotalTokens  int                      `json:"search_result_total_tokens"`
	Usage                    doctorJSONSmokeUsage     `json:"usage"`
	Cost                     doctorJSONSmokeCost      `json:"cost"`
	Requests                 []doctorJSONSmokeRequest `json:"requests"`
}

type doctorJSONSmokeUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	ThinkingTokens      int `json:"thinking_tokens"`
	CachedInputTokens   int `json:"cached_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
}

type doctorJSONSmokeCost struct {
	USD                float64 `json:"usd"`
	PricingUnavailable bool    `json:"pricing_unavailable"`
}

type doctorJSONSmokeRequest struct {
	Name                     string               `json:"name"`
	Ran                      bool                 `json:"ran"`
	Skipped                  bool                 `json:"skipped"`
	SkipReason               string               `json:"skip_reason"`
	ToolPayload              bool                 `json:"tool_payload"`
	ImagePayload             bool                 `json:"image_payload"`
	WebSearchPayload         bool                 `json:"web_search_payload"`
	RetentionPayload         bool                 `json:"retention_payload"`
	ThinkingPayload          bool                 `json:"thinking_payload"`
	Route                    string               `json:"route"`
	ResponseID               string               `json:"response_id"`
	RequestID                string               `json:"request_id"`
	PreviousResponseID       string               `json:"previous_response_id"`
	UsageObserved            bool                 `json:"usage_observed"`
	Usage                    doctorJSONSmokeUsage `json:"usage"`
	Cost                     doctorJSONSmokeCost  `json:"cost"`
	WebSearchCallCount       int                  `json:"web_search_call_count"`
	WebSearchCallFeeEstimate float64              `json:"web_search_call_fee_estimate"`
	WebSearchUsageObserved   bool                 `json:"web_search_usage_observed"`
	SearchResultTotalTokens  int                  `json:"search_result_total_tokens"`
}

func unmarshalDoctorJSONSmoke(t *testing.T, out *bytes.Buffer) doctorJSONSmokeResult {
	t.Helper()
	return unmarshalDoctorJSON[doctorJSONSmokeReport](t, out).Smoke
}

func requireDoctorJSONSmokeUsage(t *testing.T, got, want doctorJSONSmokeUsage) {
	t.Helper()
	if got != want {
		t.Fatalf("smoke usage = %+v, want %+v", got, want)
	}
}

func requireDoctorJSONSmokeCost(t *testing.T, got doctorJSONSmokeCost, wantUSD float64, wantPricingUnavailable bool) {
	t.Helper()
	if got.USD != wantUSD || got.PricingUnavailable != wantPricingUnavailable {
		t.Fatalf("smoke cost = %+v, want usd=%v pricing_unavailable=%t", got, wantUSD, wantPricingUnavailable)
	}
}

func requireDoctorJSONSmokeRequestCount(t *testing.T, smoke doctorJSONSmokeResult, want int) {
	t.Helper()
	if len(smoke.Requests) != want {
		t.Fatalf("smoke requests = %+v, want %d requests", smoke.Requests, want)
	}
}

func requireDoctorJSONSmokeRequestAt(t *testing.T, smoke doctorJSONSmokeResult, index int, name string) doctorJSONSmokeRequest {
	t.Helper()
	if index >= len(smoke.Requests) {
		t.Fatalf("missing smoke request index %d in %+v", index, smoke.Requests)
	}
	request := smoke.Requests[index]
	if request.Name != name {
		t.Fatalf("smoke request[%d] = %+v, want name=%q", index, request, name)
	}
	return request
}

func requireDoctorSmokeTextUnavailableTotalCost(t *testing.T, output string) {
	t.Helper()
	requireDoctorContractTextContainsAll(t, output, []string{"Smoke total cost estimate: N/A (usage unavailable)"})
	requireDoctorContractTextOmitsAll(t, output, []string{"Smoke total cost estimate: $"})
}
