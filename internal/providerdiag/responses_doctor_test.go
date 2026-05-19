package providerdiag

import "testing"

func TestSkippedResponsesEntries(t *testing.T) {
	request := ResponsesSmokeRequest{Name: "tool", ToolPayload: true}

	smoke := NewSkippedResponsesSmokeRequest(request, "Provider function calling payloads are disabled")
	if !smoke.Skipped || smoke.Ran || !smoke.ToolPayload || smoke.Name != "tool" {
		t.Fatalf("skipped smoke request = %+v, want skipped tool entry", smoke)
	}
	if smoke.SkipReason != "Provider function calling payloads are disabled" {
		t.Fatalf("smoke skip reason = %q", smoke.SkipReason)
	}

	routedSmoke := NewSkippedRoutedResponsesSmokeRequest(request, "responses_streaming", "Provider function calling payloads are disabled")
	if !routedSmoke.Skipped || routedSmoke.Ran || !routedSmoke.ToolPayload || routedSmoke.Route != "responses_streaming" {
		t.Fatalf("routed skipped smoke request = %+v, want skipped tool entry with route", routedSmoke)
	}

	preview := NewSkippedResponsesPreviewRequest(request, "responses_streaming", "Provider function calling payloads are disabled")
	if !preview.Skipped || !preview.ToolPayload || preview.Name != "tool" || preview.Route != "responses_streaming" {
		t.Fatalf("skipped preview request = %+v, want skipped tool entry", preview)
	}
	if preview.Method != "" || preview.URL != "" || preview.Body != nil {
		t.Fatalf("skipped preview request = %+v, should not include live request fields", preview)
	}

	routedPreview := NewSkippedRoutedResponsesPreviewRequest(request, "responses_streaming", "Provider function calling payloads are disabled")
	if !routedPreview.Skipped || !routedPreview.ToolPayload || routedPreview.Name != "tool" || routedPreview.Route != "responses_streaming" {
		t.Fatalf("routed skipped preview request = %+v, want skipped tool entry with route", routedPreview)
	}
}

func TestResponsesPreviewRequests(t *testing.T) {
	request := ResponsesSmokeRequest{Name: "retention_followup", RetentionPayload: true}
	transport := RequestPreviewTransport{
		Method:             "POST",
		URL:                "https://example.test/v1/responses",
		Headers:            JSONHeaders(),
		PreviousResponseID: " resp_initial ",
		Body:               map[string]any{"store": true},
	}

	preview := NewResponsesPreviewRequest(request, "responses_non_streaming", transport)
	if preview.Name != request.Name || !preview.RetentionPayload || preview.Route != "responses_non_streaming" {
		t.Fatalf("preview request = %+v, want retention request with route", preview)
	}
	if preview.PreviousResponseID != "resp_initial" || preview.Method != "POST" || preview.URL != transport.URL || preview.Body == nil {
		t.Fatalf("preview transport = %+v, want trimmed previous response ID and transport fields", preview)
	}

	routed := NewRoutedResponsesPreviewRequest(request, "responses_streaming", transport)
	if routed.Route != "responses_streaming" || routed.PreviousResponseID != "resp_initial" || routed.Headers["Content-Type"] != "application/json" {
		t.Fatalf("routed preview request = %+v, want route and transport fields", routed)
	}
}

func TestAddResponsesSmokeRequestResult(t *testing.T) {
	result := ResponsesSmokeResult{Ran: true}
	text := ResponsesSmokeRequestResult{
		Name:          "text",
		Ran:           true,
		Content:       "ok",
		ResponseID:    "resp_text",
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 10, OutputTokens: 3},
		Cost:          SmokeCost{USD: 0.125},
	}
	AddResponsesSmokeRequestResult(&result, text)

	if result.Content != "ok" || result.ResponseID != "resp_text" || !result.UsageObserved {
		t.Fatalf("result after text observation = %+v, want content, response id and usage observation", result)
	}

	retention := ResponsesSmokeRequestResult{
		Name:             "retention_followup",
		Ran:              true,
		RetentionPayload: true,
		ResponseID:       "resp_followup",
		UsageObserved:    true,
		Usage:            SmokeUsage{InputTokens: 2, OutputTokens: 1},
		Cost:             SmokeCost{USD: 0.25},
	}
	AddResponsesSmokeRequestResult(&result, retention)

	if !result.RetentionPayload || result.ResponseID != "resp_text" {
		t.Fatalf("result after retention observation = %+v, want retention payload and first response id preserved", result)
	}
	if result.Usage.InputTokens != 12 || result.Usage.OutputTokens != 4 || result.Cost.USD != 0.375 {
		t.Fatalf("result after retention observation = %+v, want summed usage and cost", result)
	}
	if !AllResponsesSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllResponsesSmokeRequestsObservedUsage(%+v) = false, want true", result.Requests)
	}

	AddResponsesSmokeRequestResult(&result, ResponsesSmokeRequestResult{Name: "followup", Ran: true})
	if AllResponsesSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllResponsesSmokeRequestsObservedUsage(%+v) = true, want false for partial usage", result.Requests)
	}
}

func TestAddRoutedResponsesSmokeRequestResult(t *testing.T) {
	result := RoutedResponsesSmokeResult{Ran: true, Route: "responses_streaming"}
	text := RoutedResponsesSmokeRequestResult{
		Name:          "text",
		Ran:           true,
		Route:         "responses_streaming",
		Content:       "ok",
		ResponseID:    "resp_text",
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 10, OutputTokens: 3},
		Cost:          SmokeCost{USD: 0.125},
	}
	AddRoutedResponsesSmokeRequestResult(&result, text)

	if result.Route != "responses_streaming" || result.Content != "ok" || result.ResponseID != "resp_text" {
		t.Fatalf("result after text observation = %+v, want route, content and response id", result)
	}

	tool := RoutedResponsesSmokeRequestResult{
		Name:          "tool",
		Ran:           true,
		ToolPayload:   true,
		Route:         "responses_streaming",
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 2, OutputTokens: 1},
		Cost:          SmokeCost{PricingUnavailable: true},
	}
	AddRoutedResponsesSmokeRequestResult(&result, tool)

	if !result.ToolPayload || result.Content != "ok" {
		t.Fatalf("result after tool observation = %+v, want tool payload and first content preserved", result)
	}
	if result.Usage.InputTokens != 12 || result.Usage.OutputTokens != 4 || !result.Cost.PricingUnavailable {
		t.Fatalf("result after tool observation = %+v, want summed usage and unavailable cost", result)
	}
	if !AllRoutedResponsesSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllRoutedResponsesSmokeRequestsObservedUsage(%+v) = false, want true", result.Requests)
	}
}
