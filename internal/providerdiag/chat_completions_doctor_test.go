package providerdiag

import (
	"context"
	"io"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
)

func TestTextToolSmokeRequests(t *testing.T) {
	requests := TextToolSmokeRequests(TextToolSmokeRequestOptions{
		ToolSmoke:              true,
		FunctionCallingEnabled: false,
		ProviderSlug:           "openrouter",
		ToolName:               "xelyon_openrouter_doctor_probe",
		ToolExpectedValue:      "openrouter-tool-ok",
	})

	if len(requests) != 2 {
		t.Fatalf("requests = %#v, want text fallback plus tool request", requests)
	}
	if requests[0].Name != "text" || requests[0].ToolPayload {
		t.Fatalf("text request = %#v, want text payload", requests[0])
	}
	if requests[0].UserContent != "Reply with: xelyon openrouter doctor ok" {
		t.Fatalf("text content = %q, want provider-specific probe", requests[0].UserContent)
	}
	if requests[1].Name != "tool" || !requests[1].ToolPayload {
		t.Fatalf("tool request = %#v, want tool payload", requests[1])
	}
	if requests[1].UserContent != `Call xelyon_openrouter_doctor_probe exactly once with {"value":"openrouter-tool-ok"} and do not answer in prose.` {
		t.Fatalf("tool content = %q, want forced tool probe", requests[1].UserContent)
	}
}

func TestNewChatCompletionsSmokeRequestContextDisablesToolsForText(t *testing.T) {
	ctx := NewChatCompletionsSmokeRequestContext(
		context.Background(),
		config.DefaultConfig(),
		ChatCompletionsSmokeRequest{Name: "text"},
		NoopDiagnosticToolDefinitions("xelyon_test_probe", "Test"),
		io.Discard,
	)

	if !api.IsToolUseDisabled(ctx) {
		t.Fatal("text smoke context should disable tool use")
	}
	if got := api.ToolDefinitionsFromContext(ctx); len(got) != 0 {
		t.Fatalf("tool definitions = %#v, want none for text smoke", got)
	}
}

func TestNewChatCompletionsSmokeRequestContextAddsToolsForToolPayload(t *testing.T) {
	ctx := NewChatCompletionsSmokeRequestContext(
		context.Background(),
		config.DefaultConfig(),
		ChatCompletionsSmokeRequest{Name: "tool", ToolPayload: true},
		NoopDiagnosticToolDefinitions("xelyon_test_probe", "Test"),
		io.Discard,
	)

	if api.IsToolUseDisabled(ctx) {
		t.Fatal("tool smoke context should allow tool use")
	}
	tools := api.ToolDefinitionsFromContext(ctx)
	if len(tools) != 1 || tools[0].Name != "xelyon_test_probe" {
		t.Fatalf("tool definitions = %#v, want diagnostic tool", tools)
	}
}

func TestSmokeUsageAndCostAggregation(t *testing.T) {
	gotUsage := AddSmokeUsage(
		SmokeUsage{InputTokens: 10, OutputTokens: 3, ThinkingTokens: 1, CachedInputTokens: 2, CacheCreationTokens: 1},
		SmokeUsage{InputTokens: 4, OutputTokens: 5, ThinkingTokens: 2, CachedInputTokens: 1, CacheCreationTokens: 3},
	)
	if gotUsage != (SmokeUsage{InputTokens: 14, OutputTokens: 8, ThinkingTokens: 3, CachedInputTokens: 3, CacheCreationTokens: 4}) {
		t.Fatalf("AddSmokeUsage() = %+v, want summed fields", gotUsage)
	}

	gotCost := AddSmokeCost(SmokeCost{USD: 0.125}, SmokeCost{USD: 0.25})
	if gotCost.USD != 0.375 || gotCost.PricingUnavailable {
		t.Fatalf("AddSmokeCost(available) = %+v, want summed available cost", gotCost)
	}
	gotCost = AddSmokeCost(gotCost, SmokeCost{PricingUnavailable: true})
	if !gotCost.PricingUnavailable || gotCost.USD != 0.375 {
		t.Fatalf("AddSmokeCost(unavailable) = %+v, want unavailable preserving known cost", gotCost)
	}

	projectedCost := SmokeCostFromEstimate(cost.CostEstimate{Cost: 0.4, PricingUnavailable: true})
	if projectedCost.USD != 0.4 || !projectedCost.PricingUnavailable {
		t.Fatalf("SmokeCostFromEstimate() = %+v, want projected fields", projectedCost)
	}
}

func TestSkippedChatCompletionsToolEntries(t *testing.T) {
	request := ChatCompletionsSmokeRequest{Name: "tool", ToolPayload: true}

	smoke := NewSkippedChatCompletionsToolSmokeRequest(request, "chat_completions", "Provider function calling payloads are disabled")
	if !smoke.Skipped || smoke.Ran || !smoke.ToolPayload || smoke.Name != "tool" || smoke.Route != "chat_completions" {
		t.Fatalf("skipped smoke request = %+v, want skipped tool entry", smoke)
	}
	if smoke.SkipReason != "Provider function calling payloads are disabled" {
		t.Fatalf("smoke skip reason = %q", smoke.SkipReason)
	}

	preview := NewSkippedChatCompletionsToolPreviewRequest(request, "chat_completions", "Provider function calling payloads are disabled")
	if !preview.Skipped || !preview.ToolPayload || preview.Name != "tool" || preview.Route != "chat_completions" {
		t.Fatalf("skipped preview request = %+v, want skipped tool entry", preview)
	}
	if preview.Method != "" || preview.URL != "" || preview.Body != nil {
		t.Fatalf("skipped preview request = %+v, should not include live request fields", preview)
	}
}

func TestAddChatCompletionsSmokeRequestResult(t *testing.T) {
	result := ChatCompletionsSmokeResult{Ran: true, Route: "chat_completions"}
	text := ChatCompletionsSmokeRequestResult{
		Name:          "text",
		Ran:           true,
		Route:         "chat_completions",
		Content:       "ok",
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 10, OutputTokens: 3},
		Cost:          SmokeCost{USD: 0.125},
	}
	AddChatCompletionsSmokeRequestResult(&result, text)

	if result.Content != "ok" || !result.UsageObserved || result.Usage.InputTokens != 10 || result.Cost.USD != 0.125 {
		t.Fatalf("result after text observation = %+v, want content, usage and cost", result)
	}

	tool := ChatCompletionsSmokeRequestResult{
		Name:          "tool",
		Ran:           true,
		ToolPayload:   true,
		Route:         "chat_completions",
		Content:       `{"tool":"xelyon_test_probe"}`,
		UsageObserved: true,
		Usage:         SmokeUsage{InputTokens: 2, OutputTokens: 1},
		Cost:          SmokeCost{USD: 0.25},
	}
	AddChatCompletionsSmokeRequestResult(&result, tool)

	if !result.ToolPayload || result.Content != "ok" {
		t.Fatalf("result after tool observation = %+v, want tool payload and first content preserved", result)
	}
	if result.Usage.InputTokens != 12 || result.Usage.OutputTokens != 4 || result.Cost.USD != 0.375 {
		t.Fatalf("result after tool observation = %+v, want summed usage and cost", result)
	}
	if !AllChatCompletionsSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllChatCompletionsSmokeRequestsObservedUsage(%+v) = false, want true", result.Requests)
	}

	AddChatCompletionsSmokeRequestResult(&result, ChatCompletionsSmokeRequestResult{Name: "followup", Ran: true})
	if AllChatCompletionsSmokeRequestsObservedUsage(result.Requests) {
		t.Fatalf("AllChatCompletionsSmokeRequestsObservedUsage(%+v) = true, want false for partial usage", result.Requests)
	}
}

func TestContentHasToolCall(t *testing.T) {
	if !ContentHasToolCall(`{"tool":"xelyon_test_probe","args":{}}`, "xelyon_test_probe") {
		t.Fatal("ContentHasToolCall() = false, want true")
	}
	if ContentHasToolCall(`{"tool":"other","args":{}}`, "xelyon_test_probe") {
		t.Fatal("ContentHasToolCall(other) = true, want false")
	}
}
