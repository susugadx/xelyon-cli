package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMultimodalMessage_ToMessage(t *testing.T) {
	mm := MultimodalMessage{
		Role:    "user",
		Content: "Test message",
		Image: &ImageData{
			Path:      "/test/image.png",
			MediaType: "image/png",
			Base64:    "dGVzdA==",
		},
	}

	msg := mm.ToMessage()

	if msg.Role != "user" {
		t.Errorf("ToMessage() Role = %v, want 'user'", msg.Role)
	}

	if msg.Content != "Test message" {
		t.Errorf("ToMessage() Content = %v, want 'Test message'", msg.Content)
	}
	if !msg.HasImage() {
		t.Fatal("ToMessage() should preserve runtime image state")
	}
	image := msg.ImageData()
	if image == nil || image.Base64 != "dGVzdA==" || image.MediaType != "image/png" {
		t.Fatalf("ToMessage() image = %+v, want image/png dGVzdA==", image)
	}
}

func TestMessageImageStateIsRuntimeOnly(t *testing.T) {
	msg := NewUserImageMessage("describe", &ImageData{
		Path:      "/test/image.png",
		MediaType: "image/png",
		Base64:    "dGVzdA==",
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal(Message) error = %v", err)
	}

	payload := string(data)
	if strings.Contains(payload, "dGVzdA==") || strings.Contains(payload, "image/png") || strings.Contains(payload, "image") {
		t.Fatalf("marshaled Message leaked runtime image state: %s", payload)
	}
	if !msg.HasImage() {
		t.Fatal("Message should keep runtime image state after marshal")
	}
}

func TestMultimodalMessage_HasImage_WithImage(t *testing.T) {
	mm := MultimodalMessage{
		Role:    "user",
		Content: "Test",
		Image: &ImageData{
			Base64: "dGVzdA==",
		},
	}

	if !mm.HasImage() {
		t.Error("HasImage() should return true when image has Base64 data")
	}
}

func TestMultimodalMessage_HasImage_NoImage(t *testing.T) {
	mm := MultimodalMessage{
		Role:    "user",
		Content: "Test",
		Image:   nil,
	}

	if mm.HasImage() {
		t.Error("HasImage() should return false when image is nil")
	}
}

func TestMultimodalMessage_HasImage_EmptyBase64(t *testing.T) {
	mm := MultimodalMessage{
		Role:    "user",
		Content: "Test",
		Image: &ImageData{
			Path:      "/test/image.png",
			MediaType: "image/png",
			Base64:    "", // 空のBase64
		},
	}

	if mm.HasImage() {
		t.Error("HasImage() should return false when Base64 is empty")
	}
}

func TestMessageMarshalOmitAnthropicThinkingBlocks(t *testing.T) {
	msg := Message{Role: "assistant", Content: "calling a tool"}
	msg.SetAnthropicContentBlocks([]AnthropicContentBlock{
		{Type: "thinking", Thinking: "provider private thought", Signature: "sig_1"},
		{Type: "tool_use", ID: "toolu_1", Name: "read_file", Input: map[string]any{"path": "README.md"}},
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal(Message) error = %v", err)
	}

	payload := string(data)
	if strings.Contains(payload, "anthropic_content_blocks") || strings.Contains(payload, "anthropic_thinking_blocks") || strings.Contains(payload, "provider private thought") || strings.Contains(payload, "sig_1") {
		t.Fatalf("marshaled Message leaked provider state: %s", payload)
	}

	contentBlocks := msg.AnthropicContentBlocks()
	if len(contentBlocks) != 2 {
		t.Fatalf("len(AnthropicContentBlocks()) = %d, want 2", len(contentBlocks))
	}
	contentBlocks[1].Input["path"] = "mutated"
	if got := msg.AnthropicContentBlocks()[1].Input["path"]; got != "README.md" {
		t.Fatalf("AnthropicContentBlocks() returned mutable input, got %q", got)
	}

	blocks := msg.AnthropicThinkingBlocks()
	if len(blocks) != 1 {
		t.Fatalf("len(AnthropicThinkingBlocks()) = %d, want 1", len(blocks))
	}
	blocks[0].Thinking = "mutated"
	if got := msg.AnthropicThinkingBlocks()[0].Thinking; got != "provider private thought" {
		t.Fatalf("AnthropicThinkingBlocks() returned mutable state, got %q", got)
	}
}

func TestMessageMarshalOmitOpenAIResponsesReplayItems(t *testing.T) {
	msg := Message{Role: "assistant", Content: "visible answer"}
	msg.SetOpenAIResponsesInputItems([]InputItem{
		{Type: "reasoning", ID: "rs_1", EncryptedContent: "encrypted-provider-state"},
		{Type: "message", Role: "assistant", Content: "visible answer"},
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal(Message) error = %v", err)
	}

	payload := string(data)
	if strings.Contains(payload, "openai_responses") ||
		strings.Contains(payload, "encrypted-provider-state") ||
		strings.Contains(payload, "rs_1") {
		t.Fatalf("marshaled Message leaked Responses replay state: %s", payload)
	}

	items := msg.OpenAIResponsesInputItems()
	if len(items) != 2 {
		t.Fatalf("len(OpenAIResponsesInputItems()) = %d, want 2", len(items))
	}
	items[0].EncryptedContent = "mutated"
	if got := msg.OpenAIResponsesInputItems()[0].EncryptedContent; got != "encrypted-provider-state" {
		t.Fatalf("OpenAIResponsesInputItems() returned mutable state, got %q", got)
	}
}
