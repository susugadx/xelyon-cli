package api

import (
	"reflect"
	"testing"
)

func TestCloneMessages_EmptyReturnsNil(t *testing.T) {
	if got := CloneMessages(nil); got != nil {
		t.Fatalf("CloneMessages(nil) = %#v, want nil", got)
	}
	if got := CloneMessages([]Message{}); got != nil {
		t.Fatalf("CloneMessages(empty) = %#v, want nil", got)
	}
}

func TestCloneMessages_DefensiveCopiesNestedState(t *testing.T) {
	original := []Message{
		{
			Role:    "assistant",
			Content: "calling a tool",
			ToolCalls: []OpenAIToolCall{{
				ID:               "call_1",
				Type:             "function",
				ThoughtSignature: "sig_1",
				ThoughtParts: []map[string]any{{
					"type":   "thought",
					"nested": map[string]any{"items": []any{"keep"}},
				}},
				Function: OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"README.md"}`},
			}},
		},
		{
			Role:    "assistant",
			Content: "legacy thinking",
		},
	}
	original[0].SetAnthropicContentBlocks([]AnthropicContentBlock{{
		Type: "tool_use",
		ID:   "toolu_1",
		Name: "read_file",
		Input: map[string]any{
			"path":   "README.md",
			"nested": map[string]any{"keep": []any{"value"}},
		},
	}})
	original[1].SetAnthropicThinkingBlocks([]AnthropicThinkingBlock{{
		Type:      "thinking",
		Thinking:  "provider private thought",
		Signature: "sig_legacy",
	}})

	cloned := CloneMessages(original)
	if !reflect.DeepEqual(cloned, original) {
		t.Fatalf("CloneMessages() = %#v, want %#v", cloned, original)
	}

	cloned[0].Content = "mutated content"
	cloned[0].ToolCalls[0].ID = "mutated_call"
	cloned[0].ToolCalls[0].Function.Name = "mutated_tool"
	cloned[0].ToolCalls[0].ThoughtParts[0]["type"] = "mutated"
	cloned[0].ToolCalls[0].ThoughtParts[0]["nested"].(map[string]any)["items"].([]any)[0] = "mutated"
	cloned[0].providerState.anthropicContentBlocks[0].Input["path"] = "mutated.go"
	cloned[0].providerState.anthropicContentBlocks[0].Input["nested"].(map[string]any)["keep"].([]any)[0] = "mutated"
	cloned[1].providerState.anthropicThinkingBlocks[0].Thinking = "mutated thinking"

	if original[0].Content != "calling a tool" {
		t.Fatalf("original Content mutated to %q", original[0].Content)
	}
	if original[0].ToolCalls[0].ID != "call_1" {
		t.Fatalf("original ToolCalls[0].ID = %q, want call_1", original[0].ToolCalls[0].ID)
	}
	if original[0].ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("original ToolCalls[0].Function.Name = %q, want read_file", original[0].ToolCalls[0].Function.Name)
	}
	if got := original[0].ToolCalls[0].ThoughtParts[0]["type"]; got != "thought" {
		t.Fatalf("original ThoughtParts type = %q, want thought", got)
	}
	if got := original[0].ToolCalls[0].ThoughtParts[0]["nested"].(map[string]any)["items"].([]any)[0]; got != "keep" {
		t.Fatalf("original nested ThoughtParts item = %q, want keep", got)
	}
	if got := original[0].providerState.anthropicContentBlocks[0].Input["path"]; got != "README.md" {
		t.Fatalf("original AnthropicContentBlock Input path = %q, want README.md", got)
	}
	if got := original[0].providerState.anthropicContentBlocks[0].Input["nested"].(map[string]any)["keep"].([]any)[0]; got != "value" {
		t.Fatalf("original nested AnthropicContentBlock input = %q, want value", got)
	}
	if got := original[1].providerState.anthropicThinkingBlocks[0].Thinking; got != "provider private thought" {
		t.Fatalf("original AnthropicThinkingBlocks Thinking = %q, want provider private thought", got)
	}
}
