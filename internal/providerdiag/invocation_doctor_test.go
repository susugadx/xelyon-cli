package providerdiag

import "testing"

func TestSkippedInvocationSmokeRequest(t *testing.T) {
	request := InvocationSmokeRequest{
		Name:            "thinking",
		ToolPayload:     true,
		ImagePayload:    true,
		ThinkingEnabled: true,
	}

	smoke := NewSkippedInvocationSmokeRequest(request, " route does not support request ")
	if !smoke.Skipped || smoke.Ran || smoke.Name != "thinking" {
		t.Fatalf("skipped smoke request = %+v, want skipped thinking entry", smoke)
	}
	if !smoke.ToolPayload || !smoke.ImagePayload || !smoke.ThinkingEnabled {
		t.Fatalf("skipped smoke request = %+v, want payload flags preserved", smoke)
	}
	if smoke.SkipReason != "route does not support request" {
		t.Fatalf("skip reason = %q, want trimmed reason", smoke.SkipReason)
	}
}

func TestAddInvocationSmokeRequestResult(t *testing.T) {
	result := InvocationSmokeResult{Ran: true}
	text := InvocationSmokeRequestResult{
		Name:          "text",
		Ran:           true,
		Content:       "ok",
		RequestID:     "req_text",
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 10, OutputTokens: 3},
		Cost:          SmokeCost{USD: 0.125},
	}
	AddInvocationSmokeRequestResult(&result, text)

	if len(result.Requests) != 1 || !result.UsageObserved || result.Usage.InputTokens != 10 || result.Cost.USD != 0.125 {
		t.Fatalf("result after text observation = %+v, want request and usage/cost summary", result)
	}

	tool := InvocationSmokeRequestResult{
		Name:          "tool",
		Ran:           true,
		ToolPayload:   true,
		RequestID:     "req_tool",
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 2, OutputTokens: 1},
		Cost:          SmokeCost{PricingUnavailable: true},
	}
	AddInvocationSmokeRequestResult(&result, tool)

	if result.Usage.InputTokens != 12 || result.Usage.OutputTokens != 4 || !result.Cost.PricingUnavailable {
		t.Fatalf("result after tool observation = %+v, want summed usage and unavailable cost", result)
	}
	if !AllInvocationSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllInvocationSmokeRequestsObservedUsage(%+v) = false, want true", result.Requests)
	}

	AddInvocationSmokeRequestResult(&result, InvocationSmokeRequestResult{Name: "image", Ran: true})
	if AllInvocationSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllInvocationSmokeRequestsObservedUsage(%+v) = true, want false for partial usage", result.Requests)
	}

	AddInvocationSmokeRequestResult(&result, NewSkippedInvocationSmokeRequest(InvocationSmokeRequest{Name: "thinking", ThinkingEnabled: true}, "unsupported"))
	if got := len(result.Requests); got != 4 {
		t.Fatalf("request count = %d, want skipped request appended", got)
	}
}
