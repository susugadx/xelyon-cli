package claude

import (
	"fmt"
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
	// user 3個以下: stableOffset=3 → len(userIndices) <= 3 → BPなし
	messages := []AnthropicMessage{
		{Role: "user", Content: []AnthropicContentBlock{{Type: "tool_result", ToolUseID: "t1", Content: "big result"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply1"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "second"}}},
		{Role: "assistant", Content: []AnthropicContentBlock{{Type: "text", Text: "reply2"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "third"}}},
	}

	SetMessageCacheBreakpoints(messages)

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

	SetMessageCacheBreakpoints(messages)

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

	SetMessageCacheBreakpoints(messages)

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

	SetMessageCacheBreakpoints(messages)

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
	SetMessageCacheBreakpoints(messages)

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

	SetMessageCacheBreakpoints(messages)

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

	SetMessageCacheBreakpoints(messages)

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
