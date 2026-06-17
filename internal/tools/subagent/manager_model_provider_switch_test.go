package subagent

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// TestManagerSpawnSwitchesProviderForModel はモデルに応じた provider 切替を確認します。
func TestManagerSpawnSwitchesProviderForModel(t *testing.T) {
	cfg := config.DefaultConfig()
	currentProvider := &managerTestProvider{name: "openai"}
	createdProvider := &managerTestProvider{name: "claude"}

	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, provider api.Provider, _ *config.Config) *RunResult {
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

	id, err := manager.Spawn(context.Background(), "compare", "", "claude-sonnet-4-6", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotProvider != createdProvider {
		t.Fatal("expected provider factory result to be used")
	}
}

func TestManagerSpawn_KeepsGroqProviderForSelectedSlashModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "groq"
	cfg.DefaultModel = "moonshotai/kimi-k2-instruct"
	cfg.SubAgent.DefaultModel = "moonshotai/kimi-k2-instruct"
	currentProvider := &managerTestProvider{name: "groq"}
	createdProvider := &managerTestProvider{name: "groq"}

	var gotProvider api.Provider
	var gotModel string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "groq" {
				t.Fatalf("ProviderFactory providerName = %q, want groq", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "compare", "", "", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "moonshotai/kimi-k2-instruct" {
		t.Fatalf("resolved model = %q, want %q", gotModel, "moonshotai/kimi-k2-instruct")
	}
	if gotProvider != createdProvider {
		t.Fatal("expected groq provider to be retained for selected slash model")
	}
}

func TestManagerSpawn_KeepsOllamaProviderForSelectedForeignLookingModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "ollama"
	cfg.DefaultModel = "deepseek-r1:8b"
	cfg.SubAgent.DefaultModel = "deepseek-r1:8b"
	currentProvider := &managerTestProvider{name: "ollama"}
	createdProvider := &managerTestProvider{name: "ollama"}

	var gotProvider api.Provider
	var gotModel string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "ollama" {
				t.Fatalf("ProviderFactory providerName = %q, want ollama", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "compare", "", "", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "deepseek-r1:8b" {
		t.Fatalf("resolved model = %q, want %q", gotModel, "deepseek-r1:8b")
	}
	if gotProvider != createdProvider {
		t.Fatal("expected ollama provider to be retained for local model")
	}
}
