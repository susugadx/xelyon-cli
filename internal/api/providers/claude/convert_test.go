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
	// user 5個: stableCount = 5-3 = 2, bpInterval=3 → 2 < 3 → BPなし
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "first"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply1"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "second"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply2"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "third"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply3"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "fourth"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply4"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "fifth"}}},
	}

	SetMessageCacheBreakpoints(messages)

	// 全 user に CacheControl なし
	for _, idx := range []int{0, 2, 4, 6, 8} {
		if messages[idx].Content[0].CacheControl != nil {
			t.Errorf("user at index %d should not have cache_control (short conversation)", idx)
		}
	}
}

func TestSetMessageCacheBreakpoints_LongConversation(t *testing.T) {
	// user 10個: stableCount = 10-3 = 7, roundedDown = (7/3)*3 = 6, BP#3 = user[5]
	var messages []AnthropicMessage
	for i := 0; i < 10; i++ {
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

	// user インデックス: 0,2,4,6,8,10,12,14,16,18
	// BP#3 = user[5] = index 10
	if messages[10].Content[0].CacheControl == nil {
		t.Error("expected cache_control on user[5] (index 10, BP#3)")
	}
	// 最後の user (index 18) には CacheControl なし（BP#4 削除）
	if messages[18].Content[0].CacheControl != nil {
		t.Error("last user should NOT have cache_control (no BP#4)")
	}
	// その他の user にも CacheControl なし
	for _, idx := range []int{0, 2, 4, 6, 8, 12, 14, 16} {
		if messages[idx].Content[0].CacheControl != nil {
			t.Errorf("user at index %d should not have cache_control", idx)
		}
	}
}

func TestSetMessageCacheBreakpoints_IntervalStability(t *testing.T) {
	// bpInterval=3 の切り捨てにより、3ターンの間 BP#3 が同じ位置に留まることを検証
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

	findBP3 := func(msgs []AnthropicMessage) int {
		for i, m := range msgs {
			if m.Role == "user" && len(m.Content) > 0 && m.Content[len(m.Content)-1].CacheControl != nil {
				return i
			}
		}
		return -1
	}

	applyAndFind := func(userCount int) int {
		msgs := makeMessages(userCount)
		SetMessageCacheBreakpoints(msgs)
		return findBP3(msgs)
	}

	// user 6: stableCount=3, roundedDown=3, BP#3=user[2]=index 4
	bp3at6 := applyAndFind(6)
	// user 7: stableCount=4, roundedDown=3, BP#3=user[2]=index 4 ← 同じ！
	bp3at7 := applyAndFind(7)
	// user 8: stableCount=5, roundedDown=3, BP#3=user[2]=index 4 ← まだ同じ！
	bp3at8 := applyAndFind(8)

	if bp3at6 != bp3at7 || bp3at7 != bp3at8 {
		t.Errorf("BP#3 should stay stable for 3 turns: user6=%d, user7=%d, user8=%d",
			bp3at6, bp3at7, bp3at8)
	}

	// user 9: stableCount=6, roundedDown=6, BP#3=user[5]=index 10 ← ここで更新
	bp3at9 := applyAndFind(9)
	if bp3at9 == bp3at8 {
		t.Errorf("BP#3 should advance at user 9, but stayed at %d", bp3at9)
	}
	if bp3at9 != 10 { // user[5] = index 10
		t.Errorf("BP#3 at user 9 should be index 10, got %d", bp3at9)
	}
}

func TestSetMessageCacheBreakpoints_NoBP4(t *testing.T) {
	// 任意の長さで、最後のuserメッセージに CacheControl がないことを検証
	for _, userCount := range []int{6, 8, 10, 15} {
		var messages []AnthropicMessage
		for i := 0; i < userCount; i++ {
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

		// 最後の user メッセージ
		lastUserIdx := len(messages) - 2
		if messages[lastUserIdx].Content[0].CacheControl != nil {
			t.Errorf("userCount=%d: last user (index %d) should NOT have cache_control",
				userCount, lastUserIdx)
		}
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

func TestSetMessageCacheBreakpoints_SingleUser(t *testing.T) {
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "only one"}}},
	}

	// panic しないこと + BPなし（stableCount = 1-3 = -2 < 3）
	SetMessageCacheBreakpoints(messages)

	if messages[0].Content[0].CacheControl != nil {
		t.Error("single user should not have cache_control")
	}
}

func TestSetMessageCacheBreakpoints_MultiBlockUser(t *testing.T) {
	// 十分な長さで BP#3 が設定される状況で、最後のブロックに cache_control が付くことを検証
	var messages []AnthropicMessage
	// user[0]: multi-block
	messages = append(messages, AnthropicMessage{
		Role: "user",
		Content: []AnthropicContentBlock{
			{Type: "tool_result", ToolUseID: "t1", Content: "result1"},
			{Type: "tool_result", ToolUseID: "t2", Content: "result2"},
		},
	})
	// 残りの user+assistant を追加（合計 user 7個、stableCount=4, roundedDown=3, BP#3=user[2]）
	for i := 1; i < 7; i++ {
		messages = append(messages,
			AnthropicMessage{Role: "assistant", Content: []AnthropicContentBlock{
				{Type: "text", Text: "reply"},
			}},
			AnthropicMessage{Role: "user", Content: []AnthropicContentBlock{
				{Type: "text", Text: fmt.Sprintf("msg%d", i)},
			}},
		)
	}

	SetMessageCacheBreakpoints(messages)

	// BP#3 = user[2] = index 4（3番目のuser）
	// user[0]=index 0 (multi-block), user[1]=index 2, user[2]=index 4
	if messages[4].Content[0].CacheControl == nil {
		t.Error("expected cache_control on BP#3 user")
	}

	// user[0] の multi-block は BP ではないので CacheControl なし
	if messages[0].Content[0].CacheControl != nil {
		t.Error("first block of user[0] should not have cache_control")
	}
	if messages[0].Content[1].CacheControl != nil {
		t.Error("second block of user[0] should not have cache_control")
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
