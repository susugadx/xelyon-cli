package api

import (
	"testing"
)

func TestConvertHistoryToInputItems(t *testing.T) {
	tests := []struct {
		name    string
		history []Message
		want    int // expected number of items
	}{
		{
			name:    "empty history",
			history: []Message{},
			want:    0,
		},
		{
			name: "single message",
			history: []Message{
				{Role: "user", Content: "Hello"},
			},
			want: 1,
		},
		{
			name: "multiple messages",
			history: []Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
				{Role: "user", Content: "How are you?"},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertHistoryToInputItems(tt.history)
			if len(got) != tt.want {
				t.Errorf("ConvertHistoryToInputItems() returned %d items, want %d", len(got), tt.want)
			}

			// Verify content is preserved
			for i, msg := range tt.history {
				if got[i].Role != msg.Role {
					t.Errorf("Item[%d].Role = %q, want %q", i, got[i].Role, msg.Role)
				}
				if got[i].Content != msg.Content {
					t.Errorf("Item[%d].Content = %q, want %q", i, got[i].Content, msg.Content)
				}
				if got[i].Type != "message" {
					t.Errorf("Item[%d].Type = %q, want 'message'", i, got[i].Type)
				}
			}
		})
	}
}

func TestOpenAIProvider_SupportsCompact(t *testing.T) {
	p := NewOpenAIProvider("test-key")

	if !p.SupportsCompact() {
		t.Error("SupportsCompact() = false, want true")
	}
}
