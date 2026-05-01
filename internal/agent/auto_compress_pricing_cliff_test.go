package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
)

func TestAverageOutputTokens(t *testing.T) {
	tests := []struct {
		name  string
		stats *SessionStats
		want  int
	}{
		{
			name:  "nil stats",
			stats: nil,
			want:  0,
		},
		{
			name:  "zero output",
			stats: &SessionStats{OutputTokens: 0, AssistantMessages: 5},
			want:  0,
		},
		{
			name:  "normal",
			stats: &SessionStats{OutputTokens: 3000, AssistantMessages: 3},
			want:  1000,
		},
		{
			name:  "zero messages",
			stats: &SessionStats{OutputTokens: 3000, AssistantMessages: 0},
			want:  3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := averageOutputTokens(tt.stats)
			if got != tt.want {
				t.Fatalf("averageOutputTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestShouldForceCompressForPricingCliff(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		model         string
		currentTokens int
		stats         *SessionStats
		wantProjected int
		wantForce     bool
	}{
		{
			name:          "zero tokens",
			provider:      "claude",
			model:         "claude-sonnet-4-5",
			currentTokens: 0,
			wantProjected: 0,
			wantForce:     false,
		},
		{
			name:          "negative tokens",
			provider:      "claude",
			model:         "claude-sonnet-4-5",
			currentTokens: -100,
			wantProjected: -100,
			wantForce:     false,
		},
		{
			name:          "well below cliff",
			provider:      "claude",
			model:         "claude-sonnet-4-5",
			currentTokens: 100000,
			stats:         &SessionStats{OutputTokens: 2000, AssistantMessages: 1},
			wantProjected: 102000,
			wantForce:     false,
		},
		{
			name:          "Claude pricing cliff手前でforce compressになる",
			provider:      "claude",
			model:         "claude-sonnet-4-5",
			currentTokens: 199000,
			stats:         &SessionStats{OutputTokens: 3000, AssistantMessages: 1},
			wantProjected: 202000,
			wantForce:     true,
		},
		{
			name:          "Gemini cliff未到達ならforce compressにならない",
			provider:      "gemini",
			model:         "gemini-3.1-pro",
			currentTokens: 199500,
			stats:         &SessionStats{OutputTokens: 500, AssistantMessages: 1},
			wantProjected: 200000,
			wantForce:     false,
		},
		{
			name:          "GPT-5.4 long input cliff手前でforce compressになる",
			provider:      "openai",
			model:         "gpt-5.4",
			currentTokens: 271000,
			stats:         &SessionStats{OutputTokens: 2000, AssistantMessages: 1},
			wantProjected: 273000,
			wantForce:     true,
		},
		{
			name:          "Gemini pricing cliff超過済みでもforce compressになる",
			provider:      "gemini",
			model:         "gemini-3.1-pro",
			currentTokens: 250000,
			stats:         nil,
			wantProjected: 250000,
			wantForce:     true,
		},
		{
			name:          "GPT-5.4 long input cliff超過済みでもforce compressになる",
			provider:      "openai",
			model:         "gpt-5.4",
			currentTokens: 300000,
			stats:         nil,
			wantProjected: 300000,
			wantForce:     true,
		},
		{
			name:          "projected tokens cross cliff",
			provider:      "claude",
			model:         "claude-sonnet-4-5",
			currentTokens: 199000,
			stats:         &SessionStats{OutputTokens: 5000, AssistantMessages: 1},
			wantProjected: 204000,
			wantForce:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectedTokens, got := shouldForceCompressForPricingCliff(tt.provider, tt.model, tt.currentTokens, tt.stats)
			if projectedTokens != tt.wantProjected {
				t.Fatalf("projectedTokens = %d, want %d", projectedTokens, tt.wantProjected)
			}
			if got != tt.wantForce {
				t.Fatalf("shouldForceCompressForPricingCliff() = %v, want %v", got, tt.wantForce)
			}
		})
	}
}

func TestShouldForceCompressForPricingCliffForConfig_UsesCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
	})
	stats := &SessionStats{OutputTokens: 2000, AssistantMessages: 1}

	projected, force := shouldForceCompressForPricingCliffForConfig(cfg, "openai", "corp-gpt-deployment", 271000, stats)
	if projected != 273000 {
		t.Fatalf("projected = %d, want 273000", projected)
	}
	if !force {
		t.Fatal("force = false, want true from gpt-5.4 pricing cliff")
	}
}

func TestGetPricingInfo_PricingCliffBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		model     string
		threshold int
		wantBase  float64
		wantHigh  float64
	}{
		{name: "Claude 200K cliff", provider: "claude", model: "claude-sonnet-4-5", threshold: 200000, wantBase: 3.00, wantHigh: 6.00},
		{name: "Gemini 200K cliff", provider: "gemini", model: "gemini-3.1-pro", threshold: 200000, wantBase: 2.00, wantHigh: 4.00},
		{name: "GPT-5.4 272K cliff", provider: "openai", model: "gpt-5.4", threshold: 272000, wantBase: 2.50, wantHigh: 5.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			below := cost.GetPricingInfo(tt.provider, tt.model, tt.threshold-1)
			atThreshold := cost.GetPricingInfo(tt.provider, tt.model, tt.threshold)
			above := cost.GetPricingInfo(tt.provider, tt.model, tt.threshold+1)

			if below.InputCostPerM != tt.wantBase {
				t.Fatalf("below cliff InputCostPerM = %f, want %f", below.InputCostPerM, tt.wantBase)
			}
			if atThreshold.InputCostPerM != tt.wantBase {
				t.Fatalf("at threshold InputCostPerM = %f, want %f", atThreshold.InputCostPerM, tt.wantBase)
			}
			if above.InputCostPerM != tt.wantHigh {
				t.Fatalf("above cliff InputCostPerM = %f, want %f", above.InputCostPerM, tt.wantHigh)
			}
		})
	}
}
