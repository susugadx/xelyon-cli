package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestLocalAutoCompressionTokenThresholdForConfig_UsesKnownContextWindow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 80

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "gemini", "gemini-3.1-pro")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 800000 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 800000", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_ReservesMaxOutputHeadroom(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 80

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "deepseek", "deepseek-v4-flash")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 616000 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 616000", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_UsesModelOutputOverrideForHeadroom(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 90
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
		ModelOverrides: map[string]config.ModelOverride{
			"corp-gpt-deployment": {MaxOutputTokens: 250000},
		},
	})

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openai", "corp-gpt-deployment")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 750000 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 750000", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_UsesCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 75
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
	})

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openai", "corp-gpt-deployment")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 750000 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 750000", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_UsesDefaultProviderOutputFallbackForHeadroom(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 99

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openai", "gpt-5.4")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 983616 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 983616", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_UsesProviderOutputOverrideForHeadroom(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 99
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		MaxOutputTokens: 250000,
	})

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openai", "gpt-5.4")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 750000 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 750000", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_ReservesThinkingBudgetWithKnownModelOutput(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 80
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "claude", "claude-sonnet-4-5")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 96000 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 96000", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_ReservesThinkingBudgetWithProviderOutputFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 80
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "claude", "claude-3-5-sonnet")
	if !ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
	}
	if got != 96000 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 96000", got)
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_IgnoresProviderOutputFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.TriggerPercent = 80

	tests := []struct {
		model string
		want  int
	}{
		{"gpt-4", 6553},
		{"gpt-3.5-turbo", 13108},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openai", tt.model)
			if !ok {
				t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = false, want true")
			}
			if got != tt.want {
				t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLocalAutoCompressionTokenThresholdForConfig_UnknownContext(t *testing.T) {
	cfg := config.DefaultConfig()

	got, ok := localAutoCompressionTokenThresholdForConfig(cfg, "openai", "corp-gpt-deployment")
	if ok {
		t.Fatal("localAutoCompressionTokenThresholdForConfig() ok = true, want false")
	}
	if got != 0 {
		t.Fatalf("localAutoCompressionTokenThresholdForConfig() = %d, want 0", got)
	}
}
