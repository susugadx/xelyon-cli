package agent

import (
	"testing"

	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
)

func phase5DResponsesInputItems(t *testing.T, input any) []openairesponses.InputItem {
	t.Helper()
	items, ok := input.([]openairesponses.InputItem)
	if !ok {
		t.Fatalf("responses input = %#v, want []InputItem", input)
	}
	return items
}

func phase5DAssertFunctionCallOutput(t *testing.T, item openairesponses.InputItem, callID, output string) {
	t.Helper()
	if item.Type != "function_call_output" || item.CallID != callID || item.Output != output {
		t.Fatalf("function_call_output = %#v, want call_id=%q output=%q", item, callID, output)
	}
}

func phase5DFindResponsesFunctionOutput(t *testing.T, items []openairesponses.InputItem, callID string) *openairesponses.InputItem {
	t.Helper()
	for i := range items {
		if items[i].Type == "function_call_output" && items[i].CallID == callID {
			return &items[i]
		}
	}
	return nil
}
