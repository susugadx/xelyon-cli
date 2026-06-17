package subagent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestCloneConfigForSub_AzureUsesConfiguredDeployment(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	_, got, err := cloneConfigForSub(cfg, "azure", TaskTypeExplore, "", "")
	if err != nil {
		t.Fatalf("cloneConfigForSub() error = %v", err)
	}
	if got != "corp-gpt55-deployment" {
		t.Fatalf("cloneConfigForSub() model = %q, want configured Azure deployment", got)
	}
}

func TestCloneConfigForSub_AzureHonorsExplicitSubAgentDeployment(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})
	cfg.SubAgent.DefaultModel = "corp-subagent-deployment"

	_, got, err := cloneConfigForSub(cfg, "azure", TaskTypeExplore, "", "")
	if err != nil {
		t.Fatalf("cloneConfigForSub() error = %v", err)
	}
	if got != "corp-subagent-deployment" {
		t.Fatalf("cloneConfigForSub() model = %q, want explicit Azure sub-agent deployment", got)
	}
}

func TestManagerSpawn_AzureConfiguredDeploymentNameCollisionKeepsAzureProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "gpt-5.4",
		CatalogModel: "gpt-5.4",
	})
	parentProvider := &managerTestProvider{name: "Azure OpenAI", configKey: "azure"}
	createdProvider := &managerTestProvider{name: "Azure OpenAI", configKey: "azure"}

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

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", parentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	response := manager.Wait([]string{id}, 0)
	if response.Status != "completed" {
		t.Fatalf("Wait().Status = %q, want completed", response.Status)
	}
	if factoryProviderName != "azure" {
		t.Fatalf("ProviderFactory providerName = %q, want azure", factoryProviderName)
	}
	if gotModel != "gpt-5.4" {
		t.Fatalf("resolved model = %q, want configured Azure deployment", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected Azure provider factory result to be used")
	}
}

func TestManagerSpawn_AzureCurrentDeploymentNameCollisionKeepsAzureProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	parentProvider := &managerTestProvider{name: "Azure OpenAI", configKey: "azure"}
	createdProvider := &managerTestProvider{name: "Azure OpenAI", configKey: "azure"}

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

	id, err := manager.SpawnWithRuntimeContext(
		context.Background(),
		"inspect files",
		"",
		"",
		"",
		parentProvider,
		cfg,
		SpawnRuntimeContext{CurrentModel: "gpt-5.4"},
	)
	if err != nil {
		t.Fatalf("SpawnWithRuntimeContext() error = %v", err)
	}
	response := manager.Wait([]string{id}, 0)
	if response.Status != "completed" {
		t.Fatalf("Wait().Status = %q, want completed", response.Status)
	}
	if factoryProviderName != "azure" {
		t.Fatalf("ProviderFactory providerName = %q, want azure", factoryProviderName)
	}
	if gotModel != "gpt-5.4" {
		t.Fatalf("resolved model = %q, want current Azure deployment", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected Azure provider factory result to be used")
	}
}

func TestManagerSpawn_AzureExplicitDeploymentNameCollisionKeepsAzureProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	parentProvider := &managerTestProvider{name: "Azure OpenAI", configKey: "azure"}
	createdProvider := &managerTestProvider{name: "Azure OpenAI", configKey: "azure"}

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

	id, err := manager.Spawn(context.Background(), "inspect files", "", "gpt-5.4", "", parentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	response := manager.Wait([]string{id}, 0)
	if response.Status != "completed" {
		t.Fatalf("Wait().Status = %q, want completed", response.Status)
	}
	if factoryProviderName != "azure" {
		t.Fatalf("ProviderFactory providerName = %q, want azure", factoryProviderName)
	}
	if gotModel != "gpt-5.4" {
		t.Fatalf("resolved model = %q, want explicit Azure deployment", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected Azure provider factory result to be used")
	}
}

func TestCloneConfigForSub_AzureWithoutDeploymentFailsInsteadOfPlaceholder(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SubAgent.DefaultModel = ""

	_, _, err := cloneConfigForSub(cfg, "azure", TaskTypeExplore, "", "")
	if err == nil {
		t.Fatal("cloneConfigForSub() should fail without an Azure deployment")
	}
	if !strings.Contains(err.Error(), "azure sub-agent model requires") {
		t.Fatalf("cloneConfigForSub() error = %v, want Azure deployment guidance", err)
	}
}

func TestManagerSpawn_AzureWithoutDeploymentFailsBeforeProviderCreation(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SubAgent.DefaultModel = ""
	parentProvider := &managerTestProvider{name: "Azure OpenAI", configKey: "azure"}
	manager := NewManagerWithOptions(ManagerOptions{
		ProviderFactory: func(providerName string) (api.Provider, error) {
			t.Fatalf("ProviderFactory should not be called, got %q", providerName)
			return nil, nil
		},
	})

	_, err := manager.Spawn(context.Background(), "inspect files", "", "", "", parentProvider, cfg)
	if err == nil {
		t.Fatal("Spawn() should fail without an Azure deployment")
	}
	if !strings.Contains(err.Error(), "azure sub-agent model requires") {
		t.Fatalf("Spawn() error = %v, want Azure deployment guidance", err)
	}
}
