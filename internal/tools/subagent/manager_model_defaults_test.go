package subagent

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// TestManagerSpawn_DefaultModelFollowsMainProvider は DefaultModel 未設定時にメイン provider の最安モデルを選ぶことを確認します。
func TestManagerSpawn_DefaultModelFollowsMainProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	provider := &managerTestProvider{name: "claude"}
	createdProvider := &managerTestProvider{name: "claude"}

	var gotModel string
	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "claude" {
				t.Fatalf("ProviderFactory providerName = %q, want claude", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "read files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "claude-haiku-4-5-20251001" {
		t.Fatalf("resolved model = %q, want claude-haiku-4-5-20251001", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected fresh provider instance to be used")
	}
}

func TestManagerSpawn_OpenAISubscriptionProviderDefaultKeepsSubscriptionProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	provider := &managerTestProvider{name: "OpenAI Subscription", configKey: "openai_subscription"}
	createdProvider := &managerTestProvider{name: "OpenAI Subscription", configKey: "openai_subscription"}

	var gotModel string
	var gotProvider api.Provider
	var factoryProviderName string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			factoryProviderName = providerName
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if factoryProviderName != "openai_subscription" {
		t.Fatalf("ProviderFactory providerName = %q, want openai_subscription", factoryProviderName)
	}
	if gotModel != "gpt-5.4-mini" {
		t.Fatalf("resolved model = %q, want gpt-5.4-mini", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected subscription provider factory result to be used")
	}
}

func TestManagerSpawn_InheritsOpenAISubscriptionWebSearchProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	cfg.WebSearch.Provider = "chatgpt"
	provider := &managerTestProvider{name: "OpenAI Subscription", configKey: "openai_subscription"}
	createdProvider := &managerTestProvider{name: "OpenAI Subscription", configKey: "openai_subscription"}

	var gotWebSearchProvider string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, cfg *config.Config) *RunResult {
			gotWebSearchProvider = cfg.WebSearch.Provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "openai_subscription" {
				t.Fatalf("ProviderFactory providerName = %q, want openai_subscription", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotWebSearchProvider != "chatgpt" {
		t.Fatalf("sub-agent WebSearch.Provider = %q, want inherited chatgpt alias", gotWebSearchProvider)
	}
}

func TestManagerSpawn_PreservesAnthropicAliasOwnerForClaudeRuntimeSubAgent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = false
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {
			DefaultModel:     "shared-claude-model",
			MaxOutputTokens:  11111,
			AnthropicVersion: "2099-01-01",
			AnthropicBeta:    []string{"beta-anthropic"},
		},
		"claude": {
			DefaultModel:     "shared-claude-model",
			MaxOutputTokens:  22222,
			AnthropicVersion: "2023-06-01",
			AnthropicBeta:    []string{"beta-claude"},
		},
	})
	parent := &managerTestProvider{name: "Claude", configKey: "anthropic"}

	var factoryProviderName string
	var gotProviderConfigKey string
	var gotMaxTokens int
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, cfg *config.Config) *RunResult {
			keyed, ok := provider.(interface{ ProviderConfigKey() string })
			if !ok {
				t.Fatal("sub-agent provider should preserve ProviderConfigKey()")
			}
			gotProviderConfigKey = keyed.ProviderConfigKey()
			gotMaxTokens = api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), gotProviderConfigKey, model)
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			factoryProviderName = providerName
			return &managerTestProvider{name: "Claude", configKey: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "shared-claude-model", "off", parent, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	response := manager.Wait([]string{id}, 0)
	if response.Status != "completed" {
		t.Fatalf("Wait().Status = %q, want completed", response.Status)
	}
	if factoryProviderName != "anthropic" {
		t.Fatalf("ProviderFactory providerName = %q, want %q", factoryProviderName, "anthropic")
	}
	if gotProviderConfigKey != "anthropic" {
		t.Fatalf("sub-agent ProviderConfigKey() = %q, want %q", gotProviderConfigKey, "anthropic")
	}
	if gotMaxTokens != 11111 {
		t.Fatalf("GetMaxOutputTokens() = %d, want %d from anthropic alias owner", gotMaxTokens, 11111)
	}
}

// TestManagerSpawn_PlaceholderModelFallsBackToMainProvider はプレースホルダ文字列指定時に provider 推定へフォールバックすることを確認します。
func TestManagerSpawn_PlaceholderModelFallsBackToMainProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	provider := &managerTestProvider{name: "gemini"}
	createdProvider := &managerTestProvider{name: "gemini"}

	var gotModel string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, _ api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "gemini" {
				t.Fatalf("ProviderFactory providerName = %q, want gemini", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "sub_agent.default_model", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "gemini-3.1-flash-lite" {
		t.Fatalf("resolved model = %q, want gemini-3.1-flash-lite", gotModel)
	}
}

// TestManagerSpawn_PlaceholderConfigFallsBackToMainProvider は設定値がプレースホルダ文字列でも provider 推定へフォールバックすることを確認します。
func TestManagerSpawn_PlaceholderConfigFallsBackToMainProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = "sub_agent.default_model"
	provider := &managerTestProvider{name: "claude"}
	createdProvider := &managerTestProvider{name: "claude"}

	var gotModel string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, _ api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "claude" {
				t.Fatalf("ProviderFactory providerName = %q, want claude", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "claude-haiku-4-5-20251001" {
		t.Fatalf("resolved model = %q, want claude-haiku-4-5-20251001", gotModel)
	}
}

// TestManagerSpawn_DefaultModelExplicitOverride は明示設定モデルが provider 推定より優先されることを確認します。
func TestManagerSpawn_DefaultModelExplicitOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = "gpt-5.4-mini"
	currentProvider := &managerTestProvider{name: "gemini"}
	createdProvider := &managerTestProvider{name: "openai"}

	var gotModel string
	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "openai" {
				t.Fatalf("ProviderFactory providerName = %q, want openai", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "gpt-5.4-mini" {
		t.Fatalf("resolved model = %q, want gpt-5.4-mini", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected provider factory result to be used")
	}
}

// TestManagerSpawn_DefaultModelUnknownProviderFallback は不明 provider で OpenAI nano にフォールバックすることを確認します。
func TestManagerSpawn_DefaultModelUnknownProviderFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	currentProvider := &managerTestProvider{name: "custom"}
	createdProvider := &managerTestProvider{name: "openai"}

	var gotModel string
	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "openai" {
				t.Fatalf("ProviderFactory providerName = %q, want openai", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "gpt-5.4-mini" {
		t.Fatalf("resolved model = %q, want gpt-5.4-mini", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected fallback provider factory result to be used")
	}
}
