package providerdiag

import "testing"

func TestMultimodalRequestBases(t *testing.T) {
	request := MultimodalSmokeRequest{
		Name:             "thinking",
		ToolPayload:      true,
		ImagePayload:     true,
		ThinkingPayload:  true,
		WebSearchPayload: true,
		Route:            "claude_messages",
	}

	smoke := NewSkippedMultimodalSmokeRequest(request, " disabled ")
	if !smoke.Skipped || smoke.Ran || smoke.SkipReason != "disabled" {
		t.Fatalf("skipped smoke request = %+v, want skipped with trimmed reason", smoke)
	}
	if !smoke.ToolPayload || !smoke.ImagePayload || !smoke.ThinkingPayload || !smoke.WebSearchPayload || smoke.Route != "claude_messages" {
		t.Fatalf("skipped smoke request = %+v, want payload flags and route preserved", smoke)
	}

	preview := NewSkippedMultimodalPreviewRequest(request, " disabled ")
	if !preview.Skipped || preview.SkipReason != "disabled" || preview.Route != "claude_messages" {
		t.Fatalf("skipped preview request = %+v, want skipped with route", preview)
	}
	if preview.Method != "" || preview.URL != "" || preview.Body != nil {
		t.Fatalf("skipped preview request = %+v, should not include live request fields", preview)
	}
}

func TestAddMultimodalSmokeRequestResult(t *testing.T) {
	result := MultimodalSmokeResult{Ran: true}
	text := MultimodalSmokeRequestResult{
		Name:          "text",
		Ran:           true,
		Route:         "stream",
		Content:       "ok",
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 10, OutputTokens: 3},
		Cost:          SmokeCost{USD: 0.125},
	}
	AddMultimodalSmokeRequestResult(&result, text)

	if result.Route != "stream" || result.Content != "ok" || !result.UsageObserved {
		t.Fatalf("result after text observation = %+v, want route/content/usage", result)
	}

	web := MultimodalSmokeRequestResult{
		Name:             "web_search",
		Ran:              true,
		WebSearchPayload: true,
		Route:            "generate_content",
		UsageObserved:    true,
		Usage:            SmokeUsage{InputTokens: 2, OutputTokens: 1},
		Cost:             SmokeCost{PricingUnavailable: true},
	}
	AddMultimodalSmokeRequestResult(&result, web)

	if !result.WebSearchPayload || result.Route != "mixed" || result.Content != "ok" {
		t.Fatalf("result after web observation = %+v, want web flag, mixed route and first content", result)
	}
	if result.Usage.InputTokens != 12 || result.Usage.OutputTokens != 4 || !result.Cost.PricingUnavailable {
		t.Fatalf("result after web observation = %+v, want summed usage and unavailable cost", result)
	}
	if !AllMultimodalSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllMultimodalSmokeRequestsObservedUsage(%+v) = false, want true", result.Requests)
	}

	AddMultimodalSmokeRequestResult(&result, MultimodalSmokeRequestResult{Name: "image", Ran: true, ImagePayload: true, Route: "stream"})
	if AllMultimodalSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllMultimodalSmokeRequestsObservedUsage(%+v) = true, want false for partial usage", result.Requests)
	}
}

func TestAddThinkingMultimodalSmokeRequestResult(t *testing.T) {
	result := ThinkingMultimodalSmokeResult{Ran: true}
	thinking := MultimodalSmokeRequestResult{
		Name:            "thinking",
		Ran:             true,
		ThinkingPayload: true,
		Route:           "claude_messages",
		UsageObserved:   true,
		Usage:           SmokeUsage{InputTokens: 1},
	}
	AddThinkingMultimodalSmokeRequestResult(&result, thinking)

	if !result.ThinkingPayload || result.Route != "claude_messages" || !result.UsageObserved {
		t.Fatalf("result after thinking observation = %+v, want thinking summary", result)
	}

	AddThinkingMultimodalSmokeRequestResult(
		&result,
		NewSkippedMultimodalSmokeRequest(MultimodalSmokeRequest{Name: "tool", ToolPayload: true, Route: "claude_messages"}, "disabled"),
	)
	if !result.UsageObserved || len(result.Requests) != 2 {
		t.Fatalf("result after skipped request = %+v, want skipped ignored by summary but retained", result)
	}
}
