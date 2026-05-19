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

func TestInvocationPreviewRequests(t *testing.T) {
	request := InvocationSmokeRequest{
		Name:            "image",
		ImagePayload:    true,
		ThinkingEnabled: true,
	}
	transport := RequestPreviewTransport{
		Method:  "POST",
		URL:     "https://bedrock-runtime.us-east-1.amazonaws.com/model/example/invoke-with-response-stream",
		Headers: RedactedSigV4Headers(),
		Body:    map[string]any{"anthropic_version": "bedrock-2023-05-31"},
	}

	preview := NewInvocationPreviewRequest(request, "claude_messages", "invoke_model_with_response_stream", "anthropic.claude-3-5-sonnet", transport)
	if preview.Name != "image" || !preview.ImagePayload || !preview.ThinkingEnabled {
		t.Fatalf("preview request = %+v, want descriptor fields", preview)
	}
	if preview.Route != "claude_messages" || preview.Operation != "invoke_model_with_response_stream" || preview.ModelID != "anthropic.claude-3-5-sonnet" {
		t.Fatalf("preview request = %+v, want invocation route fields", preview)
	}
	if preview.Method != transport.Method || preview.URL != transport.URL || preview.Headers["Authorization"] != "<redacted: AWS SigV4>" || preview.Body == nil {
		t.Fatalf("preview request = %+v, want transport fields", preview)
	}

	skipped := NewSkippedInvocationPreviewRequest(request, "converse_stream", " unsupported ")
	if !skipped.Skipped || skipped.SkipReason != "unsupported" || skipped.Route != "converse_stream" {
		t.Fatalf("skipped preview request = %+v, want trimmed skipped preview", skipped)
	}
	if skipped.Method != "" || skipped.URL != "" || skipped.Body != nil {
		t.Fatalf("skipped preview request = %+v, should not include live request fields", skipped)
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
