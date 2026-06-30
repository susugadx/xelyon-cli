package claude

import (
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
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

func TestConvertToAnthropicMessages_ImageUserMessageKeepsToolSequence(t *testing.T) {
	history := []api.Message{
		api.NewUserImageMessage("inspect", &api.ImageData{MediaType: "image/png", Base64: "aW1hZ2U="}),
		{
			Role: "assistant",
			ToolCalls: []api.OpenAIToolCall{{
				ID:       "toolu_01ABC",
				Type:     "function",
				Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"README.md"}`},
			}},
		},
		{Role: "tool", ToolCallID: "toolu_01ABC", Content: "README contents"},
	}

	result := ConvertToAnthropicMessages(history)

	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want image user + assistant tool_use + user tool_result", len(result))
	}
	if result[0].Role != "user" || len(result[0].Content) != 2 {
		t.Fatalf("result[0] = %#v, want image+text user message", result[0])
	}
	if result[0].Content[0].Type != "image" || result[0].Content[0].Source == nil || result[0].Content[0].Source.Data != "aW1hZ2U=" {
		t.Fatalf("image block = %#v, want base64 image", result[0].Content[0])
	}
	if result[0].Content[1].Type != "text" || result[0].Content[1].Text != "inspect" {
		t.Fatalf("text block = %#v, want inspect text", result[0].Content[1])
	}
	if result[1].Role != "assistant" || result[1].Content[0].Type != "tool_use" || result[1].Content[0].ID != "toolu_01ABC" {
		t.Fatalf("result[1] = %#v, want assistant tool_use", result[1])
	}
	if result[2].Role != "user" || result[2].Content[0].Type != "tool_result" || result[2].Content[0].ToolUseID != "toolu_01ABC" {
		t.Fatalf("result[2] = %#v, want user tool_result", result[2])
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

func TestConvertToAnthropicMessages_WithThinkingBlocks(t *testing.T) {
	assistantMessage := api.Message{
		Role: "assistant",
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
	}
	assistantMessage.SetAnthropicThinkingBlocks([]api.AnthropicThinkingBlock{
		{Type: "thinking", Thinking: "need the file", Signature: "sig_1"},
	})

	history := []api.Message{
		{Role: "user", Content: "Read a file"},
		assistantMessage,
	}

	result := ConvertToAnthropicMessagesWithThinking(history, true)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	assistantMsg := result[1]
	if len(assistantMsg.Content) != 2 {
		t.Fatalf("assistant content length = %d, want thinking + tool_use", len(assistantMsg.Content))
	}
	if assistantMsg.Content[0].Type != "thinking" || assistantMsg.Content[0].Thinking != "need the file" || assistantMsg.Content[0].Signature != "sig_1" {
		t.Fatalf("thinking block = %#v, want preserved thinking block", assistantMsg.Content[0])
	}
	if assistantMsg.Content[1].Type != "tool_use" {
		t.Fatalf("Content[1].Type = %q, want tool_use", assistantMsg.Content[1].Type)
	}

	withoutThinking := ConvertToAnthropicMessagesWithThinking(history, false)
	if len(withoutThinking[1].Content) != 1 || withoutThinking[1].Content[0].Type != "tool_use" {
		t.Fatalf("thinking disabled content = %#v, want only tool_use", withoutThinking[1].Content)
	}

	defaultResult := ConvertToAnthropicMessages(history)
	if len(defaultResult[1].Content) != 1 || defaultResult[1].Content[0].Type != "tool_use" {
		t.Fatalf("default conversion content = %#v, want thinking omitted unless explicitly enabled", defaultResult[1].Content)
	}
}

func TestConvertToAnthropicMessages_WithOrderedContentBlocks(t *testing.T) {
	assistantMessage := api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{
			{
				ID:   "toolu_01A",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"a.txt"}`,
				},
			},
			{
				ID:   "toolu_01B",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"b.txt"}`,
				},
			},
		},
	}
	assistantMessage.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "need a", Signature: "sig_a"},
		{Type: "tool_use", ID: "toolu_01A", Name: "read_file", Input: map[string]any{"path": "a.txt"}},
		{Type: "thinking", Thinking: "need b", Signature: "sig_b"},
		{Type: "tool_use", ID: "toolu_01B", Name: "read_file", Input: map[string]any{"path": "b.txt"}},
	})

	history := []api.Message{
		{Role: "user", Content: "Read both files"},
		assistantMessage,
	}

	result := ConvertToAnthropicMessagesWithThinking(history, true)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	content := result[1].Content
	if len(content) != 4 {
		t.Fatalf("assistant content length = %d, want original thinking/tool sequence", len(content))
	}
	wantTypes := []string{"thinking", "tool_use", "thinking", "tool_use"}
	for i, want := range wantTypes {
		if content[i].Type != want {
			t.Fatalf("content[%d].Type = %q, want %q; content=%#v", i, content[i].Type, want, content)
		}
	}
	if content[1].ID != "toolu_01A" || content[1].Input["path"] != "a.txt" {
		t.Fatalf("content[1] = %#v, want first tool_use preserved", content[1])
	}
	if content[2].Thinking != "need b" || content[2].Signature != "sig_b" {
		t.Fatalf("content[2] = %#v, want second thinking preserved between tool calls", content[2])
	}
	if content[3].ID != "toolu_01B" || content[3].Input["path"] != "b.txt" {
		t.Fatalf("content[3] = %#v, want second tool_use preserved", content[3])
	}

	defaultResult := ConvertToAnthropicMessages(history)
	if len(defaultResult[1].Content) != 2 {
		t.Fatalf("default conversion content = %#v, want only generic tool_use blocks", defaultResult[1].Content)
	}
	for _, block := range defaultResult[1].Content {
		if block.Type == "thinking" {
			t.Fatalf("default conversion leaked thinking block: %#v", defaultResult[1].Content)
		}
	}
}

func TestConvertToAnthropicMessages_IncompleteOrderedContentBlocksFallsBackToToolCalls(t *testing.T) {
	assistantMessage := api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{
			{
				ID:   "toolu_01A",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"a.txt"}`,
				},
			},
			{
				ID:   "toolu_01B",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"b.txt"}`,
				},
			},
		},
	}
	assistantMessage.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "need a", Signature: "sig_a"},
		{Type: "tool_use", ID: "toolu_01A", Name: "read_file", Input: map[string]any{"path": "a.txt"}},
	})

	result := ConvertToAnthropicMessagesWithThinking([]api.Message{
		{Role: "user", Content: "Read both files"},
		assistantMessage,
	}, true)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	content := result[1].Content
	if len(content) != 2 {
		t.Fatalf("assistant content = %#v, want fallback tool_use blocks only", content)
	}
	for i, block := range content {
		if block.Type != "tool_use" {
			t.Fatalf("content[%d].Type = %q, want tool_use; content=%#v", i, block.Type, content)
		}
	}
	if content[0].ID != "toolu_01A" || content[1].ID != "toolu_01B" {
		t.Fatalf("fallback tool_use IDs = %#v, want both tool calls preserved", content)
	}
}

func TestConvertToAnthropicMessages_ExtraOrderedToolUseFallsBackToToolCalls(t *testing.T) {
	assistantMessage := api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{
			{
				ID:   "toolu_01A",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"a.txt"}`,
				},
			},
		},
	}
	assistantMessage.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "need a", Signature: "sig_a"},
		{Type: "tool_use", ID: "toolu_01A", Name: "read_file", Input: map[string]any{"path": "a.txt"}},
		{Type: "tool_use", ID: "toolu_extra", Name: "read_file", Input: map[string]any{"path": "extra.txt"}},
	})

	result := ConvertToAnthropicMessagesWithThinking([]api.Message{
		{Role: "user", Content: "Read a file"},
		assistantMessage,
	}, true)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	content := result[1].Content
	if len(content) != 1 || content[0].Type != "tool_use" || content[0].ID != "toolu_01A" {
		t.Fatalf("assistant content = %#v, want only recorded tool call", content)
	}
}

func TestConvertToAnthropicMessages_LegacyThinkingBlocksSkipAmbiguousMultiToolReplay(t *testing.T) {
	assistantMessage := api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{
			{
				ID:   "toolu_01A",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"a.txt"}`,
				},
			},
			{
				ID:   "toolu_01B",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"b.txt"}`,
				},
			},
		},
	}
	assistantMessage.SetAnthropicThinkingBlocks([]api.AnthropicThinkingBlock{
		{Type: "thinking", Thinking: "need a", Signature: "sig_a"},
		{Type: "thinking", Thinking: "need b", Signature: "sig_b"},
	})

	result := ConvertToAnthropicMessagesWithThinking([]api.Message{
		{Role: "user", Content: "Read both files"},
		assistantMessage,
	}, true)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	content := result[1].Content
	if len(content) != 2 {
		t.Fatalf("assistant content = %#v, want ambiguous legacy thinking omitted", content)
	}
	for i, block := range content {
		if block.Type != "tool_use" {
			t.Fatalf("content[%d].Type = %q, want tool_use; content=%#v", i, block.Type, content)
		}
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

func TestSetMessageCacheBreakpoints_ShortConversation(t *testing.T) {
	// user 3個以下: stableOffset=3 → len(userIndices) <= 3 → BPなし
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t1", Content: "big result"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply1"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "second"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply2"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "third"}}},
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	// 全ブロックに CacheControl なし
	for i, msg := range messages {
		for j, block := range msg.Content {
			if block.CacheControl != nil {
				t.Errorf("messages[%d].Content[%d] should not have cache_control (short conversation)", i, j)
			}
		}
	}
}

func TestSetMessageCacheBreakpoints_DynamicToolResult(t *testing.T) {
	// user 7個: tool_result を含むメッセージのうち、最大2つの content に BP を設定
	// user[0]: tool_result 5000 chars (大)
	// user[1]: tool_result 200 chars (小)
	// user[2]: tool_result 8000 chars (最大)
	// user[3]: text only
	// user[4..6]: 最新3ターン（対象外）
	largeContent := string(make([]byte, 5000))
	smallContent := "small result"
	hugeContent := string(make([]byte, 8000))

	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t1", Content: largeContent}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply1"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t2", Content: smallContent}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply2"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t3", Content: hugeContent}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply3"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "text only"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply4"}}},
		// 最新3ターン（対象外）
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t4", Content: string(make([]byte, 9999))}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply5"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "recent1"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply6"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "recent2"}}},
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	// BP#3: hugeContent (8000) at messages[4]
	if messages[4].Content[0].CacheControl == nil {
		t.Error("expected cache_control on largest tool_result (messages[4])")
	}
	// BP#4: largeContent (5000) at messages[0]
	if messages[0].Content[0].CacheControl == nil {
		t.Error("expected cache_control on second largest tool_result (messages[0])")
	}

	// smallContent は対象外
	if messages[2].Content[0].CacheControl != nil {
		t.Error("small tool_result should not have cache_control")
	}
	// text only は対象外
	if messages[6].Content[0].CacheControl != nil {
		t.Error("text-only message should not have cache_control")
	}
	// 最新3ターン内の tool_result は対象外
	if messages[8].Content[0].CacheControl != nil {
		t.Error("recent tool_result should not have cache_control")
	}
}

func TestSetMessageCacheBreakpoints_ExcludesRecentTurns(t *testing.T) {
	// 全 tool_result が最新3ターン内 → BP なし
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "old"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		// 最新3ターン
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t1", Content: string(make([]byte, 5000))}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t2", Content: string(make([]byte, 8000))}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "latest"}}},
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	for i, msg := range messages {
		for j, block := range msg.Content {
			if block.CacheControl != nil {
				t.Errorf("messages[%d].Content[%d] should not have cache_control (all in recent turns)", i, j)
			}
		}
	}
}

func TestSetMessageCacheBreakpoints_SingleToolResult(t *testing.T) {
	// 安定区間に tool_result が1つだけ → BP 1つだけ設定
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t1", Content: "only result"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg1"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg2"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg3"}}},
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	if messages[0].Content[0].CacheControl == nil {
		t.Error("expected cache_control on the only tool_result")
	}

	// 他のメッセージには cache_control なし
	for _, idx := range []int{2, 4, 6} {
		if messages[idx].Content[0].CacheControl != nil {
			t.Errorf("messages[%d] should not have cache_control", idx)
		}
	}
}

func TestSetMessageCacheBreakpoints_NoUser(t *testing.T) {
	messages := []AnthropicMessage{
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "response"}}},
	}

	// panic しないこと
	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	if messages[0].Content[0].CacheControl != nil {
		t.Error("assistant message should not have cache_control")
	}
}

func TestSetMessageCacheBreakpoints_NoToolResults(t *testing.T) {
	// tool_result がない会話 → BP なし
	var messages []AnthropicMessage
	for i := 0; i < 8; i++ {
		messages = append(messages,
			AnthropicMessage{Role: "user", Content: []AnthropicContentBlock{
				{Type: "text", Text: fmt.Sprintf("user%d", i)},
			}},
			AnthropicMessage{Role: "assistant", Content: []AnthropicContentBlock{
				{Type: "text", Text: fmt.Sprintf("reply%d", i)},
			}},
		)
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	for i, msg := range messages {
		for j, block := range msg.Content {
			if block.CacheControl != nil {
				t.Errorf("messages[%d].Content[%d] should not have cache_control (no tool_results)", i, j)
			}
		}
	}
}

func TestSetMessageCacheBreakpoints_MultiBlockUser(t *testing.T) {
	// 1つの user メッセージに複数の tool_result → 最大のブロックに BP
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{
			{Type: "tool_result", ToolUseID: "t1", Content: "short"},
			{Type: "tool_result", ToolUseID: "t2", Content: string(make([]byte, 3000))},
		}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg"}}},
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	// 大きい tool_result (block[1]) に BP
	if messages[0].Content[1].CacheControl == nil {
		t.Error("expected cache_control on larger tool_result block")
	}
	// 小さい tool_result (block[0]) には BP なし（上位2つに入らない場合）
	// ただし候補が2つしかないため、short にも BP が付く
	if messages[0].Content[0].CacheControl == nil {
		t.Error("expected cache_control on second tool_result (only 2 candidates)")
	}
}

func TestSetMessageCacheBreakpoints_UsesEstimatedTokens(t *testing.T) {
	largeBytesLowTokens := strings.Repeat("abcd ", 40) // 200 bytes, lower token density
	smallerBytesHighTokens := strings.Repeat("x", 120) // 120 bytes, higher token density

	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t1", Content: largeBytesLowTokens}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t2", Content: smallerBytesHighTokens}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg1"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg2"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg3"}}},
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), true)

	if messages[2].Content[0].CacheControl == nil {
		t.Fatal("expected token-dense tool_result to receive cache_control")
	}
}

func TestSetMessageCacheBreakpoints_Disabled(t *testing.T) {
	// prompt_cache.enabled=false → cache_control を設定しない
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = false

	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t1", Content: string(make([]byte, 5000))}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "msg"}}},
	}

	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, cfg, false)

	if messages[0].Content[0].CacheControl != nil {
		t.Error("expected no cache_control when prompt cache disabled")
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

func TestConvertToAnthropicMessages_EmptyContentFallback(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: ""},
	}

	result := ConvertToAnthropicMessages(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Content[0].Text != "(empty)" {
		t.Errorf("expected empty content fallback to '(empty)', got %q", result[0].Content[0].Text)
	}
}
