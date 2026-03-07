package history

import (
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestNewSession(t *testing.T) {
	model := "test-model"
	session := NewSession(model)

	if session == nil {
		t.Fatal("NewSession returned nil")
	}

	if session.Model != model {
		t.Errorf("Expected model %s, got %s", model, session.Model)
	}

	if session.ID == "" {
		t.Error("Expected non-empty session ID")
	}

	if len(session.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(session.Messages))
	}

	if session.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}

	if session.LastModified.IsZero() {
		t.Error("Expected non-zero last modified time")
	}
}

func TestSession_AddMessage(t *testing.T) {
	session := NewSession("test-model")
	initialModified := session.LastModified

	time.Sleep(10 * time.Millisecond) // Ensure time difference

	session.AddMessage("user", "Hello", "test-model")

	if len(session.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(session.Messages))
	}

	msg := session.Messages[0]
	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", msg.Role)
	}

	if msg.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", msg.Content)
	}

	if msg.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", msg.Model)
	}

	if msg.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	if !session.LastModified.After(initialModified) {
		t.Error("Expected LastModified to be updated")
	}
}

func TestSession_AddMultipleMessages(t *testing.T) {
	session := NewSession("test-model")

	session.AddMessage("user", "Message 1", "test-model")
	session.AddMessage("assistant", "Response 1", "test-model")
	session.AddMessage("user", "Message 2", "test-model")

	if len(session.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(session.Messages))
	}

	// Check order
	if session.Messages[0].Content != "Message 1" {
		t.Error("First message should be 'Message 1'")
	}
	if session.Messages[1].Content != "Response 1" {
		t.Error("Second message should be 'Response 1'")
	}
	if session.Messages[2].Content != "Message 2" {
		t.Error("Third message should be 'Message 2'")
	}
}

func TestSession_ToAPIMessages(t *testing.T) {
	session := NewSession("test-model")

	session.AddMessage("user", "Hello", "test-model")
	session.AddMessage("assistant", "Hi there", "test-model")

	apiMsgs := session.ToAPIMessages()

	if len(apiMsgs) != 2 {
		t.Fatalf("Expected 2 API messages, got %d", len(apiMsgs))
	}

	// Check first message
	if apiMsgs[0].Role != "user" {
		t.Errorf("Expected first role 'user', got '%s'", apiMsgs[0].Role)
	}
	if apiMsgs[0].Content != "Hello" {
		t.Errorf("Expected first content 'Hello', got '%s'", apiMsgs[0].Content)
	}

	// Check second message
	if apiMsgs[1].Role != "assistant" {
		t.Errorf("Expected second role 'assistant', got '%s'", apiMsgs[1].Role)
	}
	if apiMsgs[1].Content != "Hi there" {
		t.Errorf("Expected second content 'Hi there', got '%s'", apiMsgs[1].Content)
	}

	// Verify type
	_ = apiMsgs
}

func TestSession_ToAPIMessages_Empty(t *testing.T) {
	session := NewSession("test-model")

	apiMsgs := session.ToAPIMessages()

	if len(apiMsgs) != 0 {
		t.Errorf("Expected empty API messages, got %d", len(apiMsgs))
	}
}

func TestAddMessageFromAPI_ToolCallsRoundTrip(t *testing.T) {
	// assistant + ToolCalls（ThoughtSignature付き）を保存→復元→ToolCalls が欠落しないこと
	session := NewSession("gemini-3-pro")

	msg := api.Message{
		Role:    "assistant",
		Content: "I'll read the file.",
		ToolCalls: []api.OpenAIToolCall{
			{
				ID:   "call_abc123",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"/src/main.go"}`,
				},
				ThoughtSignature: "encrypted-sig-data-xyz",
				ThoughtParts: []map[string]any{
					{"thought": true, "text": "thinking about reading"},
					{"thought_signature": "sig-only"},
				},
			},
		},
	}

	session.AddMessageFromAPI(msg, "gemini-3-pro")

	// 復元
	apiMsgs := session.ToAPIMessages()
	if len(apiMsgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(apiMsgs))
	}

	restored := apiMsgs[0]
	if restored.Role != "assistant" {
		t.Errorf("Expected role=assistant, got %s", restored.Role)
	}
	if restored.Content != "I'll read the file." {
		t.Errorf("Expected content preserved, got %s", restored.Content)
	}
	if len(restored.ToolCalls) != 1 {
		t.Fatalf("Expected 1 ToolCall, got %d", len(restored.ToolCalls))
	}
	tc := restored.ToolCalls[0]
	if tc.ID != "call_abc123" {
		t.Errorf("Expected ToolCall.ID=call_abc123, got %s", tc.ID)
	}
	if tc.Function.Name != "read_file" {
		t.Errorf("Expected Function.Name=read_file, got %s", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"path":"/src/main.go"}` {
		t.Errorf("Expected Function.Arguments preserved, got %s", tc.Function.Arguments)
	}
	if tc.ThoughtSignature != "encrypted-sig-data-xyz" {
		t.Errorf("Expected ThoughtSignature preserved, got %s", tc.ThoughtSignature)
	}
	if len(tc.ThoughtParts) != 2 {
		t.Errorf("Expected 2 ThoughtParts, got %d", len(tc.ThoughtParts))
	}
}

func TestAddMessageFromAPI_ToolResultRoundTrip(t *testing.T) {
	// tool result（ToolCallID, ToolName付き）を保存→復元→全フィールド一致すること
	session := NewSession("gemini-3-pro")

	msg := api.Message{
		Role:       "tool",
		Content:    "file contents here",
		ToolCallID: "call_abc123",
		ToolName:   "read_file",
	}

	session.AddMessageFromAPI(msg, "gemini-3-pro")

	apiMsgs := session.ToAPIMessages()
	if len(apiMsgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(apiMsgs))
	}

	restored := apiMsgs[0]
	if restored.Role != "tool" {
		t.Errorf("Expected role=tool, got %s", restored.Role)
	}
	if restored.Content != "file contents here" {
		t.Errorf("Expected content preserved, got %s", restored.Content)
	}
	if restored.ToolCallID != "call_abc123" {
		t.Errorf("Expected ToolCallID=call_abc123, got %s", restored.ToolCallID)
	}
	if restored.ToolName != "read_file" {
		t.Errorf("Expected ToolName=read_file, got %s", restored.ToolName)
	}
}

func TestSession_ResponseID(t *testing.T) {
	session := NewSession("gpt-5.2-codex")

	// 初期状態: 空
	if session.ResponseID != "" {
		t.Errorf("ResponseID = %q, want empty", session.ResponseID)
	}

	// 設定
	session.ResponseID = "resp_abc123"
	if session.ResponseID != "resp_abc123" {
		t.Errorf("ResponseID = %q, want 'resp_abc123'", session.ResponseID)
	}
}

func TestAddMessageFromAPI_PlainMessageRegression(t *testing.T) {
	// 通常メッセージ（FCなし）は従来通り動くこと（回帰テスト）
	session := NewSession("test-model")

	// AddMessage（既存メソッド）で追加
	session.AddMessage("user", "Hello", "test-model")

	// AddMessageFromAPI で FC なしメッセージを追加
	session.AddMessageFromAPI(api.Message{
		Role:    "assistant",
		Content: "Hi there!",
	}, "test-model")

	apiMsgs := session.ToAPIMessages()
	if len(apiMsgs) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(apiMsgs))
	}

	// user メッセージ（AddMessage 経由）
	if apiMsgs[0].Role != "user" || apiMsgs[0].Content != "Hello" {
		t.Errorf("First message mismatch: %+v", apiMsgs[0])
	}
	if len(apiMsgs[0].ToolCalls) != 0 {
		t.Errorf("Plain user message should have no ToolCalls, got %d", len(apiMsgs[0].ToolCalls))
	}

	// assistant メッセージ（AddMessageFromAPI 経由、FC なし）
	if apiMsgs[1].Role != "assistant" || apiMsgs[1].Content != "Hi there!" {
		t.Errorf("Second message mismatch: %+v", apiMsgs[1])
	}
	if len(apiMsgs[1].ToolCalls) != 0 {
		t.Errorf("Plain assistant message should have no ToolCalls, got %d", len(apiMsgs[1].ToolCalls))
	}
	if apiMsgs[1].ToolCallID != "" {
		t.Errorf("Plain assistant message should have empty ToolCallID, got %s", apiMsgs[1].ToolCallID)
	}
}
