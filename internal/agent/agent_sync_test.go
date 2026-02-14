package agent

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// MockProvider implements api.Provider for testing
type MockProvider struct {
	name string
}

func (m *MockProvider) Name() string { return m.name }
func (m *MockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "", nil
}
func (m *MockProvider) SupportsImages() bool { return false }
func (m *MockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", nil
}
func (m *MockProvider) IsFunctionCallingEnabled() bool { return false }

func TestSyncWithGlobalConfig_ModelUpdate(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-old",
	}
	config.SetGlobalConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &MockProvider{name: "openai"},
	}

	// Scenario 1: Update provider specific model in config
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-new",
	}
	config.SetGlobalConfig(cfg)

	a.SyncWithGlobalConfig()

	if a.CurrentModel != "gpt-new" {
		t.Errorf("Expected CurrentModel to be 'gpt-new', got '%s'", a.CurrentModel)
	}
}

func TestSyncWithGlobalConfig_DefaultModelShadowing(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-default-old"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-specific",
	}
	config.SetGlobalConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-specific",
		CurrentProvider: &MockProvider{name: "openai"},
	}

	// Scenario: Update global DefaultModel
	// This simulates user changing 'default_model' via /config
	cfg.DefaultModel = "gpt-default-new"
	config.SetGlobalConfig(cfg)

	a.SyncWithGlobalConfig()

	// Because provider_models.openai.default_model exists, it should take precedence
	// So CurrentModel should remain "gpt-specific"
	if a.CurrentModel != "gpt-specific" {
		t.Errorf("Expected CurrentModel to remain 'gpt-specific', got '%s'", a.CurrentModel)
	}
}

func TestSyncWithGlobalConfig_DefaultModelUpdate_WhenNoProviderOverride(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-default-old"
	// Clear provider override for openai
	delete(cfg.ProviderModels, "openai")
	config.SetGlobalConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-default-old",
		CurrentProvider: &MockProvider{name: "openai"},
	}

	// Scenario: Update global DefaultModel
	cfg.DefaultModel = "gpt-default-new"
	config.SetGlobalConfig(cfg)

	a.SyncWithGlobalConfig()

	if a.CurrentModel != "gpt-default-new" {
		t.Errorf("Expected CurrentModel to be 'gpt-default-new', got '%s'", a.CurrentModel)
	}
}
