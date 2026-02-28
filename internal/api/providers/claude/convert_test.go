package claude

import (
	"fmt"
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

func TestSetMessageCacheBreakpoints_ShortConversation(t *testing.T) {
	// user 4個以下: BP#4 だけ設定、BP#3 は設定しない
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "first"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply1"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "second"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply2"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "third"}}},
	}

	SetMessageCacheBreakpoints(messages)

	// BP#4: 最後のuser (index 4) にcache_control
	if messages[4].Content[0].CacheControl == nil {
		t.Error("expected cache_control on last user message (BP#4)")
	}
	// BP#3: user が stableOffset(4) 以下なので設定されない
	if messages[0].Content[0].CacheControl != nil {
		t.Error("first user should not have cache_control (short conversation)")
	}
	if messages[2].Content[0].CacheControl != nil {
		t.Error("second user should not have cache_control (short conversation)")
	}
	// assistant にはcache_controlなし
	if messages[1].Content[0].CacheControl != nil {
		t.Error("assistant message should not have cache_control")
	}
}

func TestSetMessageCacheBreakpoints_LongConversation(t *testing.T) {
	// user 8個: BP#4 = user[7], BP#3 = user[3]（末尾から5番目）
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

	SetMessageCacheBreakpoints(messages)

	// user インデックス: 0, 2, 4, 6, 8, 10, 12, 14
	// BP#4: user[7] = index 14
	if messages[14].Content[0].CacheControl == nil {
		t.Error("expected cache_control on last user (index 14, BP#4)")
	}
	// BP#3: user[7-4] = user[3] = index 6
	if messages[6].Content[0].CacheControl == nil {
		t.Error("expected cache_control on stable boundary user (index 6, BP#3)")
	}
	// その他の user にはcache_controlなし
	for _, idx := range []int{0, 2, 4, 8, 10, 12} {
		if messages[idx].Content[0].CacheControl != nil {
			t.Errorf("user at index %d should not have cache_control", idx)
		}
	}
}

func TestSetMessageCacheBreakpoints_SingleUser(t *testing.T) {
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "only one"}}},
	}

	SetMessageCacheBreakpoints(messages)

	// BP#4 のみ設定
	if messages[0].Content[0].CacheControl == nil {
		t.Error("expected cache_control on the only user message (BP#4)")
	}
}

func TestSetMessageCacheBreakpoints_NoUser(t *testing.T) {
	messages := []AnthropicMessage{
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "response"}}},
	}

	// panic しないこと
	SetMessageCacheBreakpoints(messages)

	if messages[0].Content[0].CacheControl != nil {
		t.Error("assistant message should not have cache_control")
	}
}

func TestSetMessageCacheBreakpoints_ToolResultAsUser(t *testing.T) {
	// tool_result は Anthropic 形式で role:"user" としてマージされる
	var messages []AnthropicMessage
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			messages = append(messages, AnthropicMessage{
				Role: "user",
				Content: []AnthropicContentBlock{
					{Type: "tool_result", ToolUseID: fmt.Sprintf("t%d", i), Content: "result"},
				},
			})
		} else {
			messages = append(messages, AnthropicMessage{
				Role: "user",
				Content: []AnthropicContentBlock{
					{Type: "text", Text: fmt.Sprintf("msg%d", i)},
				},
			})
		}
		messages = append(messages, AnthropicMessage{
			Role:    "assistant",
			Content: []AnthropicContentBlock{{Type: "text", Text: "reply"}},
		})
	}

	SetMessageCacheBreakpoints(messages)

	// user は 6個 → BP#3 = user[6-1-4] = user[1] = index 2
	// BP#4 = user[5] = index 10
	if messages[10].Content[0].CacheControl == nil {
		t.Error("expected BP#4 on last user")
	}
	if messages[2].Content[0].CacheControl == nil {
		t.Error("expected BP#3 on stable boundary user (tool_result)")
	}
}

func TestSetMessageCacheBreakpoints_MultiBlockUser(t *testing.T) {
	// 複数コンテンツブロックを持つ user - 最後のブロックに cache_control
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{
			{Type: "tool_result", ToolUseID: "t1", Content: "result1"},
			{Type: "tool_result", ToolUseID: "t2", Content: "result2"},
		}},
	}

	SetMessageCacheBreakpoints(messages)

	// 最後のコンテンツブロックのみ cache_control
	if messages[0].Content[0].CacheControl != nil {
		t.Error("first content block should not have cache_control")
	}
	if messages[0].Content[1].CacheControl == nil {
		t.Error("expected cache_control on last content block of user message")
	}
}

func TestSetMessageCacheBreakpoints_StablePrefixConsistency(t *testing.T) {
	// ターンが追加されても BP#3 の位置が安定していることを確認
	makeMessages := func(userCount int) []AnthropicMessage {
		var msgs []AnthropicMessage
		for i := 0; i < userCount; i++ {
			msgs = append(msgs,
				AnthropicMessage{Role: "user", Content: []AnthropicContentBlock{
					{Type: "text", Text: fmt.Sprintf("user%d", i)},
				}},
				AnthropicMessage{Role: "assistant", Content: []AnthropicContentBlock{
					{Type: "text", Text: fmt.Sprintf("reply%d", i)},
				}},
			)
		}
		return msgs
	}

	// 6ターン: BP#3 = user[1] = index 2
	msgs6 := makeMessages(6)
	SetMessageCacheBreakpoints(msgs6)
	bp3at6 := -1
	for i, m := range msgs6 {
		if m.Role == "user" && i != len(msgs6)-2 && len(m.Content) > 0 && m.Content[len(m.Content)-1].CacheControl != nil {
			bp3at6 = i
		}
	}

	// 7ターン: BP#3 = user[2] = index 4
	msgs7 := makeMessages(7)
	SetMessageCacheBreakpoints(msgs7)
	bp3at7 := -1
	for i, m := range msgs7 {
		if m.Role == "user" && i != len(msgs7)-2 && len(m.Content) > 0 && m.Content[len(m.Content)-1].CacheControl != nil {
			bp3at7 = i
		}
	}

	// BP#3 は 1ターン分だけ進む（安定している）
	if bp3at7-bp3at6 != 2 { // 2 = user+assistant の1ターン分
		t.Errorf("BP#3 should advance by exactly 1 turn (2 messages), but moved from %d to %d", bp3at6, bp3at7)
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
