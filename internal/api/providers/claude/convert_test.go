package claude

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestConvertToAnthropicMessages_EmptyHistory(t *testing.T) {
	result := ConvertToAnthropicMessages(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}

	result = ConvertToAnthropicMessages([]api.Message{})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestConvertToAnthropicMessages_NormalUserAssistant(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result := ConvertToAnthropicMessages(history)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	// user message
	if result[0].Role != "user" {
		t.Errorf("result[0].Role = %q, want 'user'", result[0].Role)
	}
	if len(result[0].Content) != 1 {
		t.Fatalf("result[0].Content length = %d, want 1", len(result[0].Content))
	}
	if result[0].Content[0].Type != "text" {
		t.Errorf("result[0].Content[0].Type = %q, want 'text'", result[0].Content[0].Type)
	}
	if result[0].Content[0].Text != "Hello" {
		t.Errorf("result[0].Content[0].Text = %q, want 'Hello'", result[0].Content[0].Text)
	}

	// assistant message
	if result[1].Role != "assistant" {
		t.Errorf("result[1].Role = %q, want 'assistant'", result[1].Role)
	}
	if result[1].Content[0].Text != "Hi there!" {
		t.Errorf("result[1].Content[0].Text = %q, want 'Hi there!'", result[1].Content[0].Text)
	}
}

func TestConvertToAnthropicMessages_ToolRoleConversion(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "Read a file"},
		{Role: "assistant", Content: "I'll read that file.",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "toolu_01ABC",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"/test.txt"}`,
					},
				},
			},
		},
		{Role: "tool", ToolCallID: "toolu_01ABC", Content: "file contents here"},
	}

	result := ConvertToAnthropicMessages(history)

	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}

	// tool message should become user + tool_result
	toolMsg := result[2]
	if toolMsg.Role != "user" {
		t.Errorf("tool message role = %q, want 'user'", toolMsg.Role)
	}
	if len(toolMsg.Content) != 1 {
		t.Fatalf("tool message content length = %d, want 1", len(toolMsg.Content))
	}
	block := toolMsg.Content[0]
	if block.Type != "tool_result" {
		t.Errorf("block.Type = %q, want 'tool_result'", block.Type)
	}
	if block.ToolUseID != "toolu_01ABC" {
		t.Errorf("block.ToolUseID = %q, want 'toolu_01ABC'", block.ToolUseID)
	}
	if block.Content != "file contents here" {
		t.Errorf("block.Content = %q, want 'file contents here'", block.Content)
	}
}

func TestConvertToAnthropicMessages_AssistantWithToolCalls(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "Read a file"},
		{Role: "assistant", Content: "I'll read that file.",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "toolu_01XYZ",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"/readme.md"}`,
					},
				},
			},
		},
	}

	result := ConvertToAnthropicMessages(history)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	assistantMsg := result[1]
	if assistantMsg.Role != "assistant" {
		t.Errorf("assistant message role = %q, want 'assistant'", assistantMsg.Role)
	}

	// Should have text + tool_use blocks
	if len(assistantMsg.Content) != 2 {
		t.Fatalf("assistant content length = %d, want 2", len(assistantMsg.Content))
	}

	// text block
	if assistantMsg.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %q, want 'text'", assistantMsg.Content[0].Type)
	}
	if assistantMsg.Content[0].Text != "I'll read that file." {
		t.Errorf("Content[0].Text = %q, want 'I'll read that file.'", assistantMsg.Content[0].Text)
	}

	// tool_use block
	toolBlock := assistantMsg.Content[1]
	if toolBlock.Type != "tool_use" {
		t.Errorf("Content[1].Type = %q, want 'tool_use'", toolBlock.Type)
	}
	if toolBlock.ID != "toolu_01XYZ" {
		t.Errorf("Content[1].ID = %q, want 'toolu_01XYZ'", toolBlock.ID)
	}
	if toolBlock.Name != "read_file" {
		t.Errorf("Content[1].Name = %q, want 'read_file'", toolBlock.Name)
	}
	if toolBlock.Input["path"] != "/readme.md" {
		t.Errorf("Content[1].Input[path] = %v, want '/readme.md'", toolBlock.Input["path"])
	}
}

func TestConvertToAnthropicMessages_AssistantEmptyContentWithToolCalls(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "Read a file"},
		{Role: "assistant", Content: "",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "toolu_01DEF",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "bash",
						Arguments: `{"command":"ls"}`,
					},
				},
			},
		},
	}

	result := ConvertToAnthropicMessages(history)

	assistantMsg := result[1]

	// Should only have tool_use block (no text block since Content is empty)
	if len(assistantMsg.Content) != 1 {
		t.Fatalf("assistant content length = %d, want 1 (no text block for empty content)", len(assistantMsg.Content))
	}

	if assistantMsg.Content[0].Type != "tool_use" {
		t.Errorf("Content[0].Type = %q, want 'tool_use'", assistantMsg.Content[0].Type)
	}
	if assistantMsg.Content[0].Input["command"] != "ls" {
		t.Errorf("Content[0].Input[command] = %v, want 'ls'", assistantMsg.Content[0].Input["command"])
	}
}

func TestConvertToAnthropicMessages_ConsecutiveSameRoleMerge(t *testing.T) {
	// Simulate: assistant calls tool, then two tool results come back
	// Both tool results (role:"tool") become role:"user", so they should merge
	history := []api.Message{
		{Role: "user", Content: "Do two things"},
		{Role: "assistant", Content: "",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "toolu_01A",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"/a.txt"}`,
					},
				},
				{
					ID:   "toolu_01B",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"/b.txt"}`,
					},
				},
			},
		},
		{Role: "tool", ToolCallID: "toolu_01A", Content: "content of a"},
		{Role: "tool", ToolCallID: "toolu_01B", Content: "content of b"},
	}

	result := ConvertToAnthropicMessages(history)

	// user, assistant, user (merged from two tool results)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages (tool results merged), got %d", len(result))
	}

	mergedUser := result[2]
	if mergedUser.Role != "user" {
		t.Errorf("merged message role = %q, want 'user'", mergedUser.Role)
	}
	if len(mergedUser.Content) != 2 {
		t.Fatalf("merged user content length = %d, want 2", len(mergedUser.Content))
	}

	// First tool_result
	if mergedUser.Content[0].Type != "tool_result" {
		t.Errorf("Content[0].Type = %q, want 'tool_result'", mergedUser.Content[0].Type)
	}
	if mergedUser.Content[0].ToolUseID != "toolu_01A" {
		t.Errorf("Content[0].ToolUseID = %q, want 'toolu_01A'", mergedUser.Content[0].ToolUseID)
	}

	// Second tool_result
	if mergedUser.Content[1].Type != "tool_result" {
		t.Errorf("Content[1].Type = %q, want 'tool_result'", mergedUser.Content[1].Type)
	}
	if mergedUser.Content[1].ToolUseID != "toolu_01B" {
		t.Errorf("Content[1].ToolUseID = %q, want 'toolu_01B'", mergedUser.Content[1].ToolUseID)
	}
}

func TestConvertToAnthropicMessages_InvalidArgumentsJSON(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "test"},
		{Role: "assistant", Content: "",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "toolu_01BAD",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "some_tool",
						Arguments: "invalid json{{{",
					},
				},
			},
		},
	}

	result := ConvertToAnthropicMessages(history)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	toolBlock := result[1].Content[0]
	if toolBlock.Type != "tool_use" {
		t.Errorf("Content[0].Type = %q, want 'tool_use'", toolBlock.Type)
	}
	// Should have empty input on parse failure
	if len(toolBlock.Input) != 0 {
		t.Errorf("Content[0].Input should be empty on invalid JSON, got %v", toolBlock.Input)
	}
}

func TestConvertToAnthropicMessages_MultipleToolCalls(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "Do multiple things"},
		{Role: "assistant", Content: "I'll help with that.",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "toolu_01A",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"/a.txt"}`,
					},
				},
				{
					ID:   "toolu_01B",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "bash",
						Arguments: `{"command":"echo hello"}`,
					},
				},
			},
		},
	}

	result := ConvertToAnthropicMessages(history)

	assistantMsg := result[1]
	// text + 2 tool_use blocks
	if len(assistantMsg.Content) != 3 {
		t.Fatalf("assistant content length = %d, want 3", len(assistantMsg.Content))
	}

	if assistantMsg.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %q, want 'text'", assistantMsg.Content[0].Type)
	}
	if assistantMsg.Content[1].Type != "tool_use" {
		t.Errorf("Content[1].Type = %q, want 'tool_use'", assistantMsg.Content[1].Type)
	}
	if assistantMsg.Content[2].Type != "tool_use" {
		t.Errorf("Content[2].Type = %q, want 'tool_use'", assistantMsg.Content[2].Type)
	}
}
