package api

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// mockRuntimeConfigurableProvider は RuntimeConfigurable を実装するモック
type mockRuntimeConfigurableProvider struct {
	mockProvider
	appliedConfig *config.Config
}

func (m *mockRuntimeConfigurableProvider) SetRuntimeConfig(cfg *config.Config) {
	m.appliedConfig = cfg
}

// mockUIRuntimeConfigurableProvider は UIRuntimeConfigurable を実装するモック
type mockUIRuntimeConfigurableProvider struct {
	mockProvider
	appliedRuntime *ui.Runtime
}

func (m *mockUIRuntimeConfigurableProvider) SetUIRuntime(runtime *ui.Runtime) {
	m.appliedRuntime = runtime
}

func TestIsAdaptiveThinkingModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{
			name:  "claude-opus-4-7 is adaptive",
			model: "claude-opus-4-7",
			want:  true,
		},
		{
			name:  "claude-opus-4.7 dot notation is adaptive",
			model: "claude-opus-4.7",
			want:  true,
		},
		{
			name:  "claude-opus-4-6 is adaptive",
			model: "claude-opus-4-6",
			want:  true,
		},
		{
			name:  "claude-sonnet-4-6 is adaptive",
			model: "claude-sonnet-4-6",
			want:  true,
		},
		{
			name:  "claude-opus-4.6 dot notation is adaptive",
			model: "claude-opus-4.6",
			want:  true,
		},
		{
			name:  "claude-sonnet-4.6 dot notation is adaptive",
			model: "claude-sonnet-4.6",
			want:  true,
		},
		{
			name:  "case insensitive Claude-Opus-4-6",
			model: "Claude-Opus-4-6",
			want:  true,
		},
		{
			name:  "claude-opus-4-5 is not adaptive",
			model: "claude-opus-4-5",
			want:  false,
		},
		{
			name:  "claude-sonnet-4-5 is not adaptive",
			model: "claude-sonnet-4-5",
			want:  false,
		},
		{
			name:  "gpt-5.2 is not adaptive",
			model: "gpt-5.2",
			want:  false,
		},
		{
			name:  "deepseek-chat is not adaptive",
			model: "deepseek-chat",
			want:  false,
		},
		{
			name:  "empty string is not adaptive",
			model: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAdaptiveThinkingModel(tt.model)
			if got != tt.want {
				t.Errorf("isAdaptiveThinkingModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestGetMaxOutputTokens_ClaudeOpus47FamilyUsesCatalogLimit(t *testing.T) {
	tests := []struct {
		provider string
		model    string
	}{
		{provider: "bedrock", model: "global.anthropic.claude-opus-4-7-v1:0"},
		{provider: "bedrock", model: "us.anthropic.claude-opus-4-7-v1:0"},
		{provider: "openrouter", model: "anthropic/claude-opus-4-7"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.model, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.ProviderModels[tt.provider] = config.ProviderModelConfig{
				DefaultModel:    tt.model,
				MaxOutputTokens: 64000,
			}
			cfg.Thinking.Enabled = true
			cfg.Thinking.Level = "xhigh"

			ctx := config.WithContext(context.Background(), cfg)
			if got := GetMaxOutputTokens(ctx, tt.provider, tt.model); got != 128000 {
				t.Fatalf("GetMaxOutputTokens(%s, %q) = %d, want 128000", tt.provider, tt.model, got)
			}
		})
	}
}

func TestLevelToBudgetTokens(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  int
	}{
		{
			name:  "low returns 5000",
			level: "low",
			want:  5000,
		},
		{
			name:  "medium returns 10000",
			level: "medium",
			want:  10000,
		},
		{
			name:  "high returns 20000",
			level: "high",
			want:  20000,
		},
		{
			name:  "xhigh returns 40000",
			level: "xhigh",
			want:  40000,
		},
		{
			name:  "unknown level returns default (10000)",
			level: "unknown",
			want:  10000,
		},
		{
			name:  "empty string returns default (10000)",
			level: "",
			want:  10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevelToBudgetTokens(tt.level)
			if got != tt.want {
				t.Errorf("LevelToBudgetTokens(%q) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestApplyRuntimeConfig(t *testing.T) {
	t.Run("provider implements RuntimeConfigurable", func(t *testing.T) {
		provider := &mockRuntimeConfigurableProvider{
			mockProvider: mockProvider{name: "configurable"},
		}
		cfg := config.DefaultConfig()

		ApplyRuntimeConfig(provider, cfg)

		if provider.appliedConfig != cfg {
			t.Error("ApplyRuntimeConfig() should call SetRuntimeConfig on configurable provider")
		}
	})

	t.Run("provider does not implement RuntimeConfigurable", func(t *testing.T) {
		provider := &mockProvider{name: "basic"}
		cfg := config.DefaultConfig()

		// パニックしないことを確認
		ApplyRuntimeConfig(provider, cfg)
	})

	t.Run("nil provider", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// パニックしないことを確認
		ApplyRuntimeConfig(nil, cfg)
	})

	t.Run("nil config", func(t *testing.T) {
		provider := &mockRuntimeConfigurableProvider{
			mockProvider: mockProvider{name: "configurable"},
		}

		ApplyRuntimeConfig(provider, nil)

		if provider.appliedConfig != nil {
			t.Error("ApplyRuntimeConfig() should not call SetRuntimeConfig when config is nil")
		}
	})
}

func TestApplyUIRuntime(t *testing.T) {
	t.Run("provider implements UIRuntimeConfigurable", func(t *testing.T) {
		provider := &mockUIRuntimeConfigurableProvider{
			mockProvider: mockProvider{name: "ui-configurable"},
		}
		runtime := ui.NewRuntime(nil, nil, nil)

		ApplyUIRuntime(provider, runtime)

		if provider.appliedRuntime != runtime {
			t.Error("ApplyUIRuntime() should call SetUIRuntime on configurable provider")
		}
	})

	t.Run("provider does not implement UIRuntimeConfigurable", func(t *testing.T) {
		provider := &mockProvider{name: "basic"}
		runtime := ui.NewRuntime(nil, nil, nil)

		// パニックしないことを確認
		ApplyUIRuntime(provider, runtime)
	})

	t.Run("nil provider", func(t *testing.T) {
		runtime := ui.NewRuntime(nil, nil, nil)

		// パニックしないことを確認
		ApplyUIRuntime(nil, runtime)
	})

	t.Run("nil runtime", func(t *testing.T) {
		provider := &mockUIRuntimeConfigurableProvider{
			mockProvider: mockProvider{name: "ui-configurable"},
		}

		ApplyUIRuntime(provider, nil)

		if provider.appliedRuntime != nil {
			t.Error("ApplyUIRuntime() should not call SetUIRuntime when runtime is nil")
		}
	})
}
