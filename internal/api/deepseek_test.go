package api

import (
	"testing"
)

func TestNewDeepSeekProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := NewDeepSeekProvider(apiKey)

	if provider == nil {
		t.Fatal("NewDeepSeekProvider() returned nil")
	}
}

func TestDeepSeekProvider_Name(t *testing.T) {
	provider := NewDeepSeekProvider("test-key")

	name := provider.Name()
	if name != "DeepSeek" {
		t.Errorf("Name() = %v, want 'DeepSeek'", name)
	}
}

func TestDeepSeekProvider_SupportsImages(t *testing.T) {
	provider := NewDeepSeekProvider("test-key")

	supports := provider.SupportsImages()
	if supports {
		t.Error("SupportsImages() = true, want false (DeepSeek does not support images)")
	}
}

func TestGetActualModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "deepseek-chat (default)",
			model: "deepseek-chat",
			want:  "deepseek-chat",
		},
		{
			name:  "deepseek-coder",
			model: "deepseek-coder",
			want:  "deepseek-coder",
		},
		{
			name:  "deepseek-reasoner",
			model: "deepseek-reasoner",
			want:  "deepseek-reasoner",
		},
		{
			name:  "empty string",
			model: "",
			want:  "deepseek-chat", // デフォルト
		},
		{
			name:  "unknown model",
			model: "unknown-model",
			want:  "deepseek-chat", // デフォルト
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getActualModel(tt.model)
			if got != tt.want {
				t.Errorf("getActualModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
