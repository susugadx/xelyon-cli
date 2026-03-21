package api

import "testing"

// mockReasoningProvider は ReasoningContentProvider を実装するモック
type mockReasoningProvider struct {
	mockProvider
	reasoning string
}

func (m *mockReasoningProvider) LastReasoningContent() string {
	return m.reasoning
}

func TestGetReasoningContent(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		want     string
	}{
		{
			name: "provider with reasoning content",
			provider: &mockReasoningProvider{
				mockProvider: mockProvider{name: "test"},
				reasoning:    "Let me think step by step...",
			},
			want: "Let me think step by step...",
		},
		{
			name: "provider with empty reasoning content",
			provider: &mockReasoningProvider{
				mockProvider: mockProvider{name: "test"},
				reasoning:    "",
			},
			want: "",
		},
		{
			name:     "provider without reasoning support",
			provider: &mockProvider{name: "basic"},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetReasoningContent(tt.provider)
			if got != tt.want {
				t.Errorf("GetReasoningContent() = %q, want %q", got, tt.want)
			}
		})
	}
}
