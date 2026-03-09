package api

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetMaxOutputTokens(t *testing.T) {
	cfg := &config.Config{
		ProviderModels: map[string]config.ProviderModelConfig{
			"deepseek": {
				MaxOutputTokens: 16384,
				ModelOverrides: map[string]config.ModelOverride{
					"user-model": {MaxOutputTokens: 9999},
				},
			},
		},
	}

	tests := []struct {
		name     string
		provider string
		model    string
		expected int
	}{
		{
			name:     "User override has highest priority",
			provider: "deepseek",
			model:    "user-model",
			expected: 9999,
		},
		{
			name:     "Known model map has second priority",
			provider: "deepseek",
			model:    "deepseek-chat",
			expected: 8192,
		},
		{
			name:     "Provider default is fallback",
			provider: "deepseek",
			model:    "unknown-model",
			expected: 16384,
		},
		{
			name:     "Works even if provider config is missing",
			provider: "unknown-provider",
			model:    "claude-sonnet-4-6",
			expected: 64000,
		},
		{
			name:     "Returns 0 for completely unknown",
			provider: "unknown-provider",
			model:    "totally-unknown",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := config.WithContext(context.Background(), cfg)
			got := GetMaxOutputTokens(ctx, tt.provider, tt.model)
			if got != tt.expected {
				t.Errorf("GetMaxOutputTokens() = %v, want %v", got, tt.expected)
			}
		})
	}
}
