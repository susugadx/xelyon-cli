package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// --- requestCacheMode tests ---

func TestRequestCacheMode_AllModes(t *testing.T) {
	tests := []struct {
		name  string
		usage api.Usage
		want  string
	}{
		{
			name: "read + create",
			usage: api.Usage{
				InputTokens:         200,
				CachedInputTokens:   80,
				CacheCreationTokens: 40,
			},
			want: "read + create",
		},
		{
			name: "read only",
			usage: api.Usage{
				InputTokens:       200,
				CachedInputTokens: 80,
			},
			want: "read",
		},
		{
			name: "create only",
			usage: api.Usage{
				InputTokens:         200,
				CacheCreationTokens: 40,
			},
			want: "create",
		},
		{
			name:  "none",
			usage: api.Usage{InputTokens: 200},
			want:  "none",
		},
		{
			name:  "zero usage",
			usage: api.Usage{},
			want:  "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requestCacheMode(tt.usage)
			if got != tt.want {
				t.Errorf("requestCacheMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- requestCacheHitRate tests ---

func TestRequestCacheHitRate_Various(t *testing.T) {
	tests := []struct {
		name     string
		usage    api.Usage
		wantRate float64
	}{
		{
			name:     "zero input tokens",
			usage:    api.Usage{InputTokens: 0, CachedInputTokens: 50},
			wantRate: 0,
		},
		{
			name:     "negative input tokens",
			usage:    api.Usage{InputTokens: -1, CachedInputTokens: 50},
			wantRate: 0,
		},
		{
			name:     "50% cache hit",
			usage:    api.Usage{InputTokens: 100, CachedInputTokens: 50},
			wantRate: 50.0,
		},
		{
			name:     "100% cache hit",
			usage:    api.Usage{InputTokens: 100, CachedInputTokens: 100},
			wantRate: 100.0,
		},
		{
			name:     "0% cache hit",
			usage:    api.Usage{InputTokens: 100, CachedInputTokens: 0},
			wantRate: 0,
		},
		{
			name:     "25% cache hit",
			usage:    api.Usage{InputTokens: 200, CachedInputTokens: 50},
			wantRate: 25.0,
		},
		{
			name:     "75% cache hit",
			usage:    api.Usage{InputTokens: 400, CachedInputTokens: 300},
			wantRate: 75.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requestCacheHitRate(tt.usage)
			if got != tt.wantRate {
				t.Errorf("requestCacheHitRate() = %f, want %f", got, tt.wantRate)
			}
		})
	}
}
