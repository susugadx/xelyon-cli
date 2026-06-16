package history

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestSession_ToAPIMessages_SkipsToolExecutionEntries(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	session.AddToolExecution("wait_agent", map[string]string{"ids": `["sub-001"]`}, "done", true, "test-model")
	session.AddMessage("assistant", "world", "test-model")

	msgs := session.ToAPIMessages()
	if len(msgs) != 2 {
		t.Fatalf("len(ToAPIMessages()) = %d, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Fatalf("unexpected roles: %#v", msgs)
	}
}

func TestSession_AddMessageFromAPI_RoundTripsProviderMetadata(t *testing.T) {
	session := NewSession("test-model")
	msg := api.Message{
		Role:    "assistant",
		Content: "I'll inspect it.",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   "toolu_01XYZ",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"README.md"}`,
			},
		}},
	}
	msg.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "need the README", Signature: "sig_1"},
		{Type: "tool_use", ID: "toolu_01XYZ", Name: "read_file", Input: map[string]any{"path": "README.md"}},
	})

	session.AddMessageFromAPI(msg, "claude-test")
	if len(session.Messages) != 1 {
		t.Fatalf("len(session.Messages) = %d, want 1", len(session.Messages))
	}
	entry := session.Messages[0]
	if entry.ProviderMetadata == nil {
		t.Fatal("ProviderMetadata = nil, want persisted provider metadata")
	}
	if len(entry.ProviderMetadata.AnthropicContentBlocks) != 2 {
		t.Fatalf("len(ProviderMetadata.AnthropicContentBlocks) = %d, want 2", len(entry.ProviderMetadata.AnthropicContentBlocks))
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal(MessageEntry) error = %v", err)
	}
	if !strings.Contains(string(entryJSON), "provider_metadata") || !strings.Contains(string(entryJSON), "anthropic_content_blocks") {
		t.Fatalf("MessageEntry JSON = %s, want provider metadata persisted", string(entryJSON))
	}

	restored := session.ToAPIMessages()
	if len(restored) != 1 {
		t.Fatalf("len(ToAPIMessages()) = %d, want 1", len(restored))
	}
	blocks := restored[0].AnthropicThinkingBlocks()
	if len(blocks) != 1 {
		t.Fatalf("len(restored AnthropicThinkingBlocks) = %d, want 1", len(blocks))
	}
	if blocks[0].Thinking != "need the README" || blocks[0].Signature != "sig_1" {
		t.Fatalf("restored thinking block = %#v, want preserved thinking/signature", blocks[0])
	}
	contentBlocks := restored[0].AnthropicContentBlocks()
	if len(contentBlocks) != 2 || contentBlocks[1].Type != "tool_use" || contentBlocks[1].ID != "toolu_01XYZ" {
		t.Fatalf("restored content blocks = %#v, want ordered thinking/tool_use blocks", contentBlocks)
	}

	requestJSON, err := json.Marshal(restored[0])
	if err != nil {
		t.Fatalf("json.Marshal(api.Message) error = %v", err)
	}
	if strings.Contains(string(requestJSON), "anthropic_content_blocks") || strings.Contains(string(requestJSON), "anthropic_thinking_blocks") || strings.Contains(string(requestJSON), "need the README") {
		t.Fatalf("api.Message JSON leaked provider metadata: %s", string(requestJSON))
	}
}

func TestSession_TruncateMessages(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	session.AddMessage("assistant", "world", "test-model")
	session.markPersisted()
	session.AddMessage("user", "next", "test-model")

	if !session.TruncateMessages(1) {
		t.Fatal("TruncateMessages() = false, want true")
	}
	if len(session.Messages) != 1 {
		t.Fatalf("len(session.Messages) = %d, want 1", len(session.Messages))
	}
	if session.persistedCount != 1 {
		t.Fatalf("persistedCount = %d, want 1", session.persistedCount)
	}
}

func TestSession_ResetConversation(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	session.CompactedItems = []CompactedItem{{Type: "compacted", Data: "compressed"}}
	session.IsCompactedMode = true
	session.ResponseID = "resp_123"
	session.markPersisted()

	session.ResetConversation()

	if len(session.Messages) != 0 {
		t.Fatalf("len(session.Messages) = %d, want 0", len(session.Messages))
	}
	if len(session.CompactedItems) != 0 {
		t.Fatalf("len(session.CompactedItems) = %d, want 0", len(session.CompactedItems))
	}
	if session.IsCompactedMode {
		t.Fatal("IsCompactedMode = true, want false")
	}
	if session.ResponseID != "" {
		t.Fatalf("ResponseID = %q, want empty", session.ResponseID)
	}
	if session.persistedCount != 0 {
		t.Fatalf("persistedCount = %d, want 0", session.persistedCount)
	}
}

func TestNewSession_GeneratesUniqueIDs(t *testing.T) {
	const sessionCount = 64
	seen := make(map[string]struct{}, sessionCount)

	for i := 0; i < sessionCount; i++ {
		session := NewSession("test-model")
		if session.ID == "" {
			t.Fatal("session.ID = empty, want non-empty")
		}
		if _, exists := seen[session.ID]; exists {
			t.Fatalf("duplicate session ID generated: %q", session.ID)
		}
		seen[session.ID] = struct{}{}
	}
}

func TestNewSessionWithWorkingDir_NormalizesExplicitWorkingDir(t *testing.T) {
	root := t.TempDir()
	workingDir := filepath.Join(root, "repo", "child", "..")

	session := NewSessionWithWorkingDir("test-model", workingDir)

	if got, want := session.WorkingDir, filepath.Join(root, "repo"); got != want {
		t.Fatalf("WorkingDir = %q, want %q", got, want)
	}
}
