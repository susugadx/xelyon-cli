package api

import (
	"testing"
)

func TestConvertHistoryToInputItems_ThoughtSignatureFirstOnly(t *testing.T) {
	// assistant + ToolCalls(2個) → 最初の InputItem にのみ ThoughtSignature が付くこと
	history := []Message{
		{
			Role:    "assistant",
			Content: "",
			ToolCalls: []OpenAIToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"/a.go"}`,
					},
					ThoughtSignature: "sig-encrypted-abc",
					ThoughtParts: []map[string]any{
						{"thought": true, "text": "thinking"},
					},
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: OpenAIToolCallFunction{
						Name:      "bash",
						Arguments: `{"command":"ls"}`,
					},
					// 2番目の ToolCall にも ThoughtSignature がセットされているが、
					// ConvertHistoryToInputItems は i==0 のみコピーする
					ThoughtSignature: "sig-encrypted-abc",
					ThoughtParts: []map[string]any{
						{"thought": true, "text": "thinking"},
					},
				},
			},
		},
	}

	items := ConvertHistoryToInputItems(history)

	if len(items) != 2 {
		t.Fatalf("Expected 2 InputItems, got %d", len(items))
	}

	// 最初の InputItem に ThoughtSignature が付くこと
	if items[0].ThoughtSignature != "sig-encrypted-abc" {
		t.Errorf("First InputItem should have ThoughtSignature, got %q", items[0].ThoughtSignature)
	}
	if len(items[0].ThoughtParts) != 1 {
		t.Errorf("First InputItem should have 1 ThoughtPart, got %d", len(items[0].ThoughtParts))
	}

	// 2番目の InputItem に ThoughtSignature が付かないこと
	if items[1].ThoughtSignature != "" {
		t.Errorf("Second InputItem should NOT have ThoughtSignature, got %q", items[1].ThoughtSignature)
	}
	if len(items[1].ThoughtParts) != 0 {
		t.Errorf("Second InputItem should NOT have ThoughtParts, got %d", len(items[1].ThoughtParts))
	}
}

func TestConvertHistoryToInputItems_NoThoughtSignature(t *testing.T) {
	// ThoughtSignature なしの ToolCalls → フィールドが空であること
	history := []Message{
		{
			Role: "assistant",
			ToolCalls: []OpenAIToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: OpenAIToolCallFunction{
						Name:      "write_file",
						Arguments: `{"path":"/b.go","content":"hello"}`,
					},
					// ThoughtSignature/ThoughtParts なし
				},
			},
		},
	}

	items := ConvertHistoryToInputItems(history)

	if len(items) != 1 {
		t.Fatalf("Expected 1 InputItem, got %d", len(items))
	}

	if items[0].ThoughtSignature != "" {
		t.Errorf("InputItem should have empty ThoughtSignature, got %q", items[0].ThoughtSignature)
	}
	if items[0].ThoughtParts != nil {
		t.Errorf("InputItem should have nil ThoughtParts, got %v", items[0].ThoughtParts)
	}

	// function_call としての基本フィールドは正しいこと
	if items[0].Type != "function_call" {
		t.Errorf("Expected type=function_call, got %s", items[0].Type)
	}
	if items[0].CallID != "call_1" {
		t.Errorf("Expected CallID=call_1, got %s", items[0].CallID)
	}
	if items[0].Name != "write_file" {
		t.Errorf("Expected Name=write_file, got %s", items[0].Name)
	}
}

func TestConvertHistoryToInputItems_UsesOpenAIResponsesReplayMetadata(t *testing.T) {
	msg := Message{Role: "assistant", Content: "display text"}
	msg.SetOpenAIResponsesInputItems([]InputItem{
		{Type: "reasoning", ID: "rs_1", Summary: []map[string]any{{"text": "summary"}}, EncryptedContent: "encrypted"},
		{Type: "message", Role: "assistant", ID: "msg_1", Content: "provider text"},
	})

	items := ConvertHistoryToInputItems([]Message{msg})

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want replay metadata items", len(items))
	}
	if items[0].Type != "reasoning" || items[0].ID != "rs_1" || items[0].EncryptedContent != "encrypted" {
		t.Fatalf("items[0] = %#v, want reasoning replay item", items[0])
	}
	if items[1].Type != "message" || items[1].Role != "assistant" || items[1].Content != "provider text" {
		t.Fatalf("items[1] = %#v, want provider message replay item", items[1])
	}

	items[0].Summary[0]["text"] = "mutated"
	again := ConvertHistoryToInputItems([]Message{msg})
	if again[0].Summary[0]["text"] != "summary" {
		t.Fatalf("replay metadata was mutable across conversion: %#v", again[0].Summary)
	}
}

func TestConvertHistoryToInputItems_ToolReplayUsesCurrentContent(t *testing.T) {
	msg := Message{Role: "tool", Content: "trimmed output", ToolCallID: "call_1"}
	msg.SetOpenAIResponsesInputItems([]InputItem{
		{Type: "function_call_output", CallID: "call_1", Output: "old raw output"},
	})

	items := ConvertHistoryToInputItems([]Message{msg})

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Output != "trimmed output" {
		t.Fatalf("function_call_output = %q, want current message content", items[0].Output)
	}
}

func TestConvertHistoryToInputItems_ToolReplayNormalizesEmptyOutput(t *testing.T) {
	msg := Message{Role: "tool", Content: "   ", ToolCallID: "call_1"}
	msg.SetOpenAIResponsesInputItems([]InputItem{
		{Type: "function_call_output", CallID: "call_1", Output: "old raw output"},
	})

	items := ConvertHistoryToInputItems([]Message{msg})

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Output != "(no output)" {
		t.Fatalf("function_call_output = %q, want normalized empty output", items[0].Output)
	}
}

func TestReplaceOpenAIResponsesFunctionCallArguments(t *testing.T) {
	msg := Message{Role: "assistant"}
	msg.SetOpenAIResponsesInputItems([]InputItem{
		{Type: "message", Role: "assistant", Content: "I'll edit"},
		{Type: "function_call", CallID: "call_1", Name: "apply_patch", Arguments: `{"patch":"old"}`},
	})

	if !msg.ReplaceOpenAIResponsesFunctionCallArguments("call_1", "apply_patch", `{"patch":"new"}`) {
		t.Fatal("ReplaceOpenAIResponsesFunctionCallArguments() = false, want true")
	}

	items := msg.OpenAIResponsesInputItems()
	if items[1].Arguments != `{"patch":"new"}` {
		t.Fatalf("function_call arguments = %q, want updated", items[1].Arguments)
	}
}

func TestReplaceOpenAIResponsesFunctionCallOutput(t *testing.T) {
	msg := Message{Role: "tool", Content: "old", ToolCallID: "call_1"}
	msg.SetOpenAIResponsesInputItems([]InputItem{
		{Type: "function_call_output", CallID: "call_1", Output: "old"},
	})

	msg.ReplaceOpenAIResponsesFunctionCallOutput("call_1", "new")

	items := msg.OpenAIResponsesInputItems()
	if items[0].Output != "new" {
		t.Fatalf("function_call_output = %q, want updated", items[0].Output)
	}
}

func TestReplaceOpenAIResponsesFunctionCallOutput_NormalizesEmptyOutput(t *testing.T) {
	msg := Message{Role: "tool", Content: "old", ToolCallID: "call_1"}
	msg.SetOpenAIResponsesInputItems([]InputItem{
		{Type: "function_call_output", CallID: "call_1", Output: "old"},
	})

	msg.ReplaceOpenAIResponsesFunctionCallOutput("call_1", "")

	items := msg.OpenAIResponsesInputItems()
	if items[0].Output != "(no output)" {
		t.Fatalf("function_call_output = %q, want normalized empty output", items[0].Output)
	}
}

func TestConvertHistoryToInputItems_ToolResult(t *testing.T) {
	// tool result が function_call_output に正しく変換されること
	history := []Message{
		{Role: "tool", Content: "file contents", ToolCallID: "call_1"},
	}

	items := ConvertHistoryToInputItems(history)

	if len(items) != 1 {
		t.Fatalf("Expected 1 InputItem, got %d", len(items))
	}
	if items[0].Type != "function_call_output" {
		t.Errorf("Expected type=function_call_output, got %s", items[0].Type)
	}
	if items[0].CallID != "call_1" {
		t.Errorf("Expected CallID=call_1, got %s", items[0].CallID)
	}
	if items[0].Output != "file contents" {
		t.Errorf("Expected Output=file contents, got %s", items[0].Output)
	}
}

func TestConvertHistoryToInputItems_EmptyToolOutput(t *testing.T) {
	// 空のツール出力は "(no output)" に補完されること
	history := []Message{
		{Role: "tool", Content: "", ToolCallID: "call_1"},
	}

	items := ConvertHistoryToInputItems(history)

	if len(items) != 1 {
		t.Fatalf("Expected 1 InputItem, got %d", len(items))
	}
	if items[0].Output != "(no output)" {
		t.Errorf("Expected Output=(no output), got %q", items[0].Output)
	}
}
