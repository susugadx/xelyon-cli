package openairesponses

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestModelIdentity_CatalogNameDefaultsToRequestName(t *testing.T) {
	identity := NewModelIdentity("corp-deployment", "")

	if got := identity.RequestName(); got != "corp-deployment" {
		t.Fatalf("RequestName() = %q, want corp-deployment", got)
	}
	if got := identity.CatalogName(); got != "corp-deployment" {
		t.Fatalf("CatalogName() = %q, want request model fallback", got)
	}
}

func TestBuildChatRequest_UsesPreviousResponseIDForTrailingToolOutputs(t *testing.T) {
	req := BuildChatRequest(ChatRequestOptions{
		Base: BaseRequestOptions{
			Model:           NewModelIdentity("gpt-5.4", ""),
			MaxOutputTokens: 1000,
			Stream:          true,
			Store:           true,
		},
		SystemPrompt:       "system",
		PreviousResponseID: "resp_123",
		History: []api.Message{
			{Role: "assistant", Content: "calling tool"},
			{Role: "tool", ToolCallID: "call_1", Content: "tool output"},
		},
	})

	if req.PreviousResponseID != "resp_123" {
		t.Fatalf("PreviousResponseID = %q, want resp_123", req.PreviousResponseID)
	}
	outputs, ok := req.Input.([]InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []InputItem", req.Input)
	}
	if len(outputs) != 1 || outputs[0].Type != "function_call_output" || outputs[0].CallID != "call_1" {
		t.Fatalf("Input = %#v, want trailing function_call_output", outputs)
	}
}

func TestBuildChatRequest_IncludesCompactedInputWithoutPreviousResponseID(t *testing.T) {
	req := BuildChatRequest(ChatRequestOptions{
		Base: BaseRequestOptions{
			Model: NewModelIdentity("gpt-5.4", ""),
			Store: false,
		},
		SystemPrompt: "system",
		CompactedInput: []api.InputItem{
			{Type: "compacted", Data: "compact-data"},
		},
		History: []api.Message{
			{Role: "user", Content: "next turn"},
		},
	})

	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty", req.PreviousResponseID)
	}
	input, ok := req.Input.([]InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []InputItem", req.Input)
	}
	if len(input) != 3 {
		t.Fatalf("len(Input) = %d, want developer + compacted + current history", len(input))
	}
	if input[0].Role != "developer" {
		t.Fatalf("Input[0] = %#v, want developer", input[0])
	}
	if input[1].Type != "compacted" || input[1].Data != "compact-data" {
		t.Fatalf("Input[1] = %#v, want compacted item", input[1])
	}
	if input[2].Role != "user" || input[2].Content != "next turn" {
		t.Fatalf("Input[2] = %#v, want current user history", input[2])
	}
}

func TestBuildImageRequest_IncludesDeveloperHistoryAndImage(t *testing.T) {
	req := BuildImageRequest(ImageRequestOptions{
		Base: BaseRequestOptions{
			Model:  NewModelIdentity("gpt-5.4", ""),
			Stream: true,
			Store:  true,
		},
		SystemPrompt: "system",
		History:      []api.Message{{Role: "user", Content: "before"}},
		UserMessage:  "what is this?",
		Image: &api.ImageData{
			Base64:    "abc123",
			MediaType: "image/png",
		},
	})

	input, ok := req.Input.([]InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []InputItem", req.Input)
	}
	if len(input) != 3 {
		t.Fatalf("len(Input) = %d, want 3", len(input))
	}
	if input[0].Role != "developer" || input[0].Content != "system" {
		t.Fatalf("developer message = %#v, want system prompt", input[0])
	}
	parts, ok := input[2].Content.([]InputContentPart)
	if !ok || len(parts) != 2 {
		t.Fatalf("image content = %#v, want two content parts", input[2].Content)
	}
	if parts[0].Type != "input_image" || parts[0].ImageURL != "data:image/png;base64,abc123" {
		t.Fatalf("image part = %#v, want data URL image", parts[0])
	}
}

func TestBuildImageRequest_IncludesCompactedInput(t *testing.T) {
	req := BuildImageRequest(ImageRequestOptions{
		Base: BaseRequestOptions{
			Model: NewModelIdentity("gpt-5.4", ""),
			Store: false,
		},
		SystemPrompt:   "system",
		CompactedInput: []api.InputItem{{Type: "compacted", Data: "compact-data"}},
		UserMessage:    "what is this?",
		Image: &api.ImageData{
			Base64:    "abc123",
			MediaType: "image/png",
		},
	})

	input, ok := req.Input.([]InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []InputItem", req.Input)
	}
	if len(input) != 3 {
		t.Fatalf("len(Input) = %d, want developer + compacted + image", len(input))
	}
	if input[1].Type != "compacted" || input[1].Data != "compact-data" {
		t.Fatalf("Input[1] = %#v, want compacted item", input[1])
	}
	parts, ok := input[2].Content.([]InputContentPart)
	if !ok || len(parts) != 2 {
		t.Fatalf("image content = %#v, want two content parts", input[2].Content)
	}
}

func TestBuildFunctionToolChoice_UsesResponsesAPIShape(t *testing.T) {
	toolName := "read_file"
	choice, ok := BuildFunctionToolChoice(&toolName).(map[string]interface{})
	if !ok {
		t.Fatalf("BuildFunctionToolChoice() type = %T, want map[string]interface{}", choice)
	}
	if choice["type"] != "function" {
		t.Fatalf("tool_choice.type = %v, want function", choice["type"])
	}
	if choice["name"] != "read_file" {
		t.Fatalf("tool_choice.name = %v, want read_file", choice["name"])
	}
	if _, ok := choice["function"]; ok {
		t.Fatalf("Responses API tool_choice must not use chat-completions function wrapper: %#v", choice)
	}
}
