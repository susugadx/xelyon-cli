package history

import (
	"encoding/json"
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

func TestSessionAddMessageFromAPI_RoundTripsReasoningAndFunctionCalling(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessageFromAPI(api.Message{
		Role:             "assistant",
		Content:          "Checking file",
		ReasoningContent: "Need to inspect the file first.",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"main.go"}`,
			},
		}},
	}, "test-model")
	session.AddMessageFromAPI(api.Message{
		Role:       "tool",
		Content:    "package main",
		ToolCallID: "call_1",
		ToolName:   "read_file",
	}, "test-model")

	if len(session.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(session.Messages))
	}
	entry := session.Messages[0]
	if entry.ReasoningContent != "Need to inspect the file first." {
		t.Fatalf("ReasoningContent = %q, want preserved reasoning_content", entry.ReasoningContent)
	}
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal(MessageEntry) error = %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal(entryJSON, &stored); err != nil {
		t.Fatalf("json.Unmarshal(MessageEntry JSON) error = %v", err)
	}
	if stored["reasoning_content"] != "Need to inspect the file first." {
		t.Fatalf("reasoning_content = %#v, want persisted reasoning_content", stored["reasoning_content"])
	}

	restored := session.ToAPIMessages()
	if len(restored) != 2 {
		t.Fatalf("len(ToAPIMessages()) = %d, want 2", len(restored))
	}
	if restored[0].ReasoningContent != "Need to inspect the file first." {
		t.Fatalf("restored ReasoningContent = %q, want preserved reasoning_content", restored[0].ReasoningContent)
	}
	if len(restored[0].ToolCalls) != 1 || restored[0].ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("restored ToolCalls = %+v, want read_file", restored[0].ToolCalls)
	}
	if restored[1].Role != "tool" || restored[1].ToolCallID != "call_1" || restored[1].Content != "package main" {
		t.Fatalf("restored tool message = %+v, want role=tool with tool_call_id/content", restored[1])
	}
}

func TestSessionAddMessageFromAPIWithStoredContent_PreservesMetadataWithContentOverride(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessageFromAPIWithStoredContent(api.Message{
		Role:             "assistant",
		Content:          "raw [COMPACTION]hidden[/COMPACTION] response",
		ReasoningContent: "Need to hide compaction notice from display history.",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"main.go"}`,
			},
		}},
	}, "raw  response", "test-model")

	restored := session.ToAPIMessages()
	if len(restored) != 1 {
		t.Fatalf("len(ToAPIMessages()) = %d, want 1", len(restored))
	}
	if restored[0].Content != "raw  response" {
		t.Fatalf("Content = %q, want stored content override", restored[0].Content)
	}
	if restored[0].ReasoningContent != "Need to hide compaction notice from display history." {
		t.Fatalf("ReasoningContent = %q, want API metadata preserved", restored[0].ReasoningContent)
	}
	if len(restored[0].ToolCalls) != 1 || restored[0].ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("ToolCalls = %+v, want API metadata preserved", restored[0].ToolCalls)
	}
}

func TestSessionToAPIMessages_LegacyEntryWithoutReasoningContent(t *testing.T) {
	var entry MessageEntry
	if err := json.Unmarshal([]byte(`{"role":"assistant","content":"legacy response","model":"test-model"}`), &entry); err != nil {
		t.Fatalf("json.Unmarshal(MessageEntry) error = %v", err)
	}
	session := NewSession("test-model")
	session.Messages = []MessageEntry{entry}

	restored := session.ToAPIMessages()
	if len(restored) != 1 {
		t.Fatalf("len(ToAPIMessages()) = %d, want 1", len(restored))
	}
	if restored[0].ReasoningContent != "" {
		t.Fatalf("ReasoningContent = %q, want empty for legacy entry", restored[0].ReasoningContent)
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
