package history

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestSessionAddMessageFromAPI_PreservesFunctionCallingMetadata(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessageFromAPI(api.Message{
		Role:    "assistant",
		Content: "Checking file",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"main.go"}`,
			},
		}},
		ToolCallID: "call_1",
		ToolName:   "read_file",
	}, "test-model")

	if len(session.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(session.Messages))
	}
	msg := session.Messages[0]
	if msg.Role != "assistant" || msg.Content != "Checking file" {
		t.Fatalf("message = %+v", msg)
	}
	if len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("ToolCalls = %+v, want read_file", msg.ToolCalls)
	}
	if msg.ToolCallID != "call_1" || msg.ToolName != "read_file" {
		t.Fatalf("tool metadata = (%q, %q), want (call_1, read_file)", msg.ToolCallID, msg.ToolName)
	}
}

func TestSessionUnsavedMessagesNegativePersistedCountAndHelpers(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	session.persistedCount = -5

	unsaved := session.unsavedMessages()
	if len(unsaved) != 1 {
		t.Fatalf("len(unsavedMessages()) = %d, want 1", len(unsaved))
	}
	if session.persistedCount != 0 {
		t.Fatalf("persistedCount = %d, want 0 after normalization", session.persistedCount)
	}

	session.markPersisted()
	if got := session.unsavedMessages(); got != nil {
		t.Fatalf("unsavedMessages() after markPersisted = %v, want nil", got)
	}

	if got := cloneStringMap(nil); got != nil {
		t.Fatalf("cloneStringMap(nil) = %v, want nil", got)
	}
	cloned := cloneStringMap(map[string]string{"a": "1"})
	cloned["a"] = "2"
	if got := truncateRunes("hello", 0); got != "" {
		t.Fatalf("truncateRunes(max=0) = %q, want empty", got)
	}
}

func TestSessionMarkPersisted_NilReceiver(t *testing.T) {
	var session *Session
	session.markPersisted()
}
