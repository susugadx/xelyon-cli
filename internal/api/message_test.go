package api

import (
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
