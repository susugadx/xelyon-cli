package history

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func newStorageLoadTestStorage(t *testing.T) *Storage {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	return storage
}

func TestStorage_Load_MetadataOnlySession(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.CompactedItems = []CompactedItem{{Type: "compacted", Data: "summary"}}
	session.IsCompactedMode = true
	session.ResponseID = "resp_123"
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Messages) != 0 {
		t.Fatalf("len(loaded.Messages) = %d, want 0 for metadata-only session", len(loaded.Messages))
	}
	if !loaded.IsCompactedMode {
		t.Fatal("loaded.IsCompactedMode = false, want true for persisted compacted state")
	}
	if loaded.ResponseID != "resp_123" {
		t.Fatalf("loaded.ResponseID = %q, want resp_123", loaded.ResponseID)
	}
	if len(loaded.CompactedItems) != 1 || loaded.CompactedItems[0].Data != "summary" {
		t.Fatalf("loaded.CompactedItems = %#v, want persisted compacted state", loaded.CompactedItems)
	}
}

func TestStorage_Load_ResponsePromptFingerprint(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("gpt-5")
	session.ApplyResponseContextWithFingerprint("resp_123", "gpt-5", "openai", "openai", "fingerprint-1")
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.ResponsePromptFingerprint != "fingerprint-1" {
		t.Fatalf("loaded.ResponsePromptFingerprint = %q, want fingerprint-1", loaded.ResponsePromptFingerprint)
	}
}

func TestStorage_Load_CompactedStatePreservesFullInputItemShape(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	items := []CompactedItem{
		{
			Type:             "function_call",
			CallID:           "call_1",
			Name:             "read_file",
			Arguments:        `{"path":"README.md"}`,
			ThoughtSignature: "sig_1",
			ThoughtParts:     []map[string]any{{"type": "thought", "text": "inspect README"}},
		},
		{
			Type:   "function_call_output",
			CallID: "call_1",
			Output: "README contents",
		},
		{
			Type:    "message",
			Role:    "assistant",
			Content: "done",
			ID:      "msg_1",
			Status:  "completed",
		},
	}
	session := NewSession("test-model")
	session.SetCompactedState(items, true)
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !reflect.DeepEqual(loaded.CompactedItems, items) {
		t.Fatalf("loaded.CompactedItems = %#v, want %#v", loaded.CompactedItems, items)
	}
}

func TestStorage_Load_MissingHistoryFileForCompactedStateReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.SetCompactedState([]CompactedItem{{Type: "compacted", Data: "summary"}}, true)
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	if err := os.Remove(storage.sessionPath(session.ID)); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err := storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want missing compacted state error")
	}
	if !strings.Contains(err.Error(), "metadata expects 0 messages, 1 compacted items") {
		t.Fatalf("Load() error = %v, want compacted state detail", err)
	}
}

func TestStorage_Load_CorruptedCompactedStateReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.SetCompactedState([]CompactedItem{{Type: "compacted", Data: "summary"}}, true)
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	if err := os.WriteFile(storage.sessionPath(session.ID), []byte("CORRUPTED COMPACTED STATE\n"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want corrupted compacted state error")
	}
	if !strings.Contains(err.Error(), "metadata expects 1 compacted items, loaded 0") {
		t.Fatalf("Load() error = %v, want compacted state mismatch", err)
	}
}

func TestStorage_SaveLoad_RestoresProviderMetadata(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("claude-test")
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
		{Type: "redacted_thinking", Data: "opaque"},
	})
	msg.SetOpenAIResponsesInputItems([]api.InputItem{
		{Type: "reasoning", ID: "rs_1", EncryptedContent: "encrypted-replay-state"},
		{Type: "function_call", CallID: "toolu_01XYZ", Name: "read_file", Arguments: `{"path":"README.md"}`},
	})
	session.AddMessageFromAPI(msg, "claude-test")

	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	rawJSON := string(raw)
	if !strings.Contains(rawJSON, "provider_metadata") ||
		!strings.Contains(rawJSON, "anthropic_content_blocks") ||
		!strings.Contains(rawJSON, "openai_responses_items") {
		t.Fatalf("stored JSONL = %s, want provider metadata persisted", rawJSON)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	restored := loaded.ToAPIMessages()
	if len(restored) != 1 {
		t.Fatalf("len(ToAPIMessages()) = %d, want 1", len(restored))
	}
	blocks := restored[0].AnthropicThinkingBlocks()
	if len(blocks) != 2 {
		t.Fatalf("len(restored AnthropicThinkingBlocks) = %d, want 2", len(blocks))
	}
	if blocks[0].Type != "thinking" || blocks[0].Thinking != "need the README" || blocks[0].Signature != "sig_1" {
		t.Fatalf("restored thinking block = %#v, want preserved thinking/signature", blocks[0])
	}
	if blocks[1].Type != "redacted_thinking" || blocks[1].Data != "opaque" {
		t.Fatalf("restored redacted thinking block = %#v, want preserved data", blocks[1])
	}
	contentBlocks := restored[0].AnthropicContentBlocks()
	if len(contentBlocks) != 3 {
		t.Fatalf("len(restored AnthropicContentBlocks) = %d, want 3", len(contentBlocks))
	}
	if contentBlocks[1].Type != "tool_use" || contentBlocks[1].ID != "toolu_01XYZ" {
		t.Fatalf("restored content blocks = %#v, want ordered thinking/tool_use/redacted blocks", contentBlocks)
	}
	openAIItems := restored[0].OpenAIResponsesInputItems()
	if len(openAIItems) != 2 {
		t.Fatalf("len(restored OpenAIResponsesInputItems) = %d, want 2", len(openAIItems))
	}
	if openAIItems[0].Type != "reasoning" || openAIItems[0].EncryptedContent != "encrypted-replay-state" {
		t.Fatalf("restored OpenAI reasoning replay item = %#v, want encrypted replay state", openAIItems[0])
	}
	if openAIItems[1].Type != "function_call" || openAIItems[1].CallID != "toolu_01XYZ" {
		t.Fatalf("restored OpenAI function_call item = %#v, want call toolu_01XYZ", openAIItems[1])
	}
}

func TestStorage_Load_MissingHistoryFileForConversationSessionReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := os.Remove(storage.sessionPath(session.ID)); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err := storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want missing history file error")
	}
	if !strings.Contains(err.Error(), "metadata expects 1 messages") {
		t.Fatalf("Load() error = %v, want missing history detail", err)
	}
}

func TestStorage_Load_TruncatedConversationHistoryReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Hello", "test-model")
	session.AddMessage("assistant", "Hi there", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}
	if err := os.WriteFile(storage.sessionPath(session.ID), []byte(lines[0]+"\n"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want inconsistency error")
	}
	if !strings.Contains(err.Error(), "metadata expects 2 messages, loaded 1") {
		t.Fatalf("Load() error = %v, want count mismatch detail", err)
	}
}

func TestStorage_Load_LegacyPlaintextHistoryAfterEncryptionToggle(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XELYON_ENCRYPT_HISTORY", "")

	plaintextStorage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "Plaintext message", "test-model")
	if err := plaintextStorage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	t.Setenv("XELYON_ENCRYPT_HISTORY", "1")
	encryptedStorage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage with encryption failed: %v", err)
	}

	loaded, err := encryptedStorage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed after encryption toggle: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("len(loaded.Messages) = %d, want 1", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Plaintext message" {
		t.Fatalf("loaded.Messages[0].Content = %q, want %q", loaded.Messages[0].Content, "Plaintext message")
	}
}

func TestStorage_Load_CorruptedData(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Valid message", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	filePath := storage.sessionPath(session.ID)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	if _, err := f.WriteString("CORRUPTED LINE\n"); err != nil {
		t.Fatalf("Failed to write corrupted line: %v", err)
	}
	_ = f.Close()

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load should not fail on corrupted data: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("Expected 1 valid message (corrupted line skipped), got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Valid message" {
		t.Errorf("Expected 'Valid message', got '%s'", loaded.Messages[0].Content)
	}
}

func TestStorage_Load_CorruptedConversationRowWithToolAuditReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Valid message", "test-model")
	session.AddMessage("assistant", "Assistant reply", "test-model")
	session.AddToolExecution("read_file", map[string]string{"path": "foo.go"}, "ok", true, "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(lines))
	}
	lines[1] = "CORRUPTED ASSISTANT LINE"
	if err := os.WriteFile(storage.sessionPath(session.ID), []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want inconsistency error")
	}
	if !strings.Contains(err.Error(), "metadata expects 2 messages, loaded 1") {
		t.Fatalf("Load() error = %v, want conversation count mismatch", err)
	}
}

func TestStorage_Load_LongConversationLine(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	longContent := strings.Repeat("0123456789abcdef", 5*1024) // 80KiB
	session.AddMessage("assistant", longContent, "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed for long line: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("len(loaded.Messages) = %d, want 1", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != longContent {
		t.Fatal("loaded long message content mismatch")
	}
}
