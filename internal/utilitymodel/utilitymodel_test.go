package utilitymodel

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type runtimeCaptureProvider struct {
	*testutil.MockProvider
	runtimeConfig *config.Config
}

func (p *runtimeCaptureProvider) SetRuntimeConfig(cfg *config.Config) {
	p.runtimeConfig = cfg
}

func TestEnabledForTask_DefaultTask(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UtilityModel.Enabled = true
	cfg.UtilityModel.Provider = "anthropic"
	cfg.UtilityModel.Model = "claude-haiku"
	cfg.UtilityModel.Tasks = nil

	if !EnabledForTask(cfg, TaskWebSearchCompaction) {
		t.Fatal("EnabledForTask() should allow default web_search_compaction task")
	}
}

func TestRun_UsesConfiguredModelAndIsolatedRuntimeConfig(t *testing.T) {
	originalNewProvider := newProvider
	t.Cleanup(func() {
		newProvider = originalNewProvider
	})

	mock := &runtimeCaptureProvider{MockProvider: testutil.NewMockProvider()}
	mock.ChatWithToolsFunc = func(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
		if systemPrompt != "utility system" {
			t.Fatalf("systemPrompt = %q, want %q", systemPrompt, "utility system")
		}
		if model != "gpt-5.2-mini" {
			t.Fatalf("model = %q, want %q", model, "gpt-5.2-mini")
		}
		if len(history) != 1 || history[0].Role != "user" || history[0].Content != "utility user" {
			t.Fatalf("history = %#v, want single user prompt", history)
		}
		if defs := tools.RegistryFromContext(ctx).GetToolDefinitions(); len(defs) != 0 {
			t.Fatalf("utility model should run without tools, got %d definitions", len(defs))
		}
		return "utility response", nil
	}

	newProvider = func(providerName string) (api.Provider, error) {
		if providerName != "openai" {
			t.Fatalf("providerName = %q, want %q", providerName, "openai")
		}
		return mock, nil
	}

	cfg := config.DefaultConfig()
	cfg.UtilityModel.Enabled = true
	cfg.UtilityModel.Provider = "openai"
	cfg.UtilityModel.Model = "gpt-5.2-mini"
	cfg.Thinking.Enabled = true
	cfg.PromptCache.Enabled = true

	got, err := Run(context.Background(), cfg, TaskWebSearchCompaction, "utility system", "utility user")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got != "utility response" {
		t.Fatalf("Run() = %q, want %q", got, "utility response")
	}

	if mock.runtimeConfig == nil {
		t.Fatal("runtimeConfig should be injected")
	}
	if mock.runtimeConfig == cfg {
		t.Fatal("runtimeConfig should be cloned, not reuse the original config pointer")
	}
	if mock.runtimeConfig.Thinking.Enabled {
		t.Fatal("utility model should disable thinking for lightweight tasks")
	}
	if mock.runtimeConfig.PromptCache.Enabled {
		t.Fatal("utility model should disable prompt cache for one-shot tasks")
	}
	if mock.runtimeConfig.UtilityModel.Enabled {
		t.Fatal("utility model runtime config should disable recursive utility model calls")
	}
}
