package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/ui"
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

func TestSyncWithRuntimeConfig_ModelUpdate(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-old",
	}
	runtime := NewAgentRuntimeWithConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	// Scenario 1: Update provider specific model in config
	runtime.Config.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-new",
	}

	a.SyncWithRuntimeConfig()

	if a.CurrentModel != "gpt-new" {
		t.Errorf("Expected CurrentModel to be 'gpt-new', got '%s'", a.CurrentModel)
	}
}

func TestSyncWithRuntimeConfig_DefaultModelShadowing(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-default-old"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-specific",
	}
	runtime := NewAgentRuntimeWithConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-specific",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	// Scenario: Update global DefaultModel
	// This simulates user changing 'default_model' via /config
	runtime.Config.DefaultModel = "gpt-default-new"

	a.SyncWithRuntimeConfig()

	// Because provider_models.openai.default_model exists, it should take precedence
	// So CurrentModel should remain "gpt-specific"
	if a.CurrentModel != "gpt-specific" {
		t.Errorf("Expected CurrentModel to remain 'gpt-specific', got '%s'", a.CurrentModel)
	}
}

func TestSyncWithRuntimeConfig_DefaultModelUpdate_WhenNoProviderOverride(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-default-old"
	// Clear provider override for openai
	delete(cfg.ProviderModels, "openai")
	runtime := NewAgentRuntimeWithConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-default-old",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	// Scenario: Update global DefaultModel
	runtime.Config.DefaultModel = "gpt-default-new"

	a.SyncWithRuntimeConfig()

	if a.CurrentModel != "gpt-default-new" {
		t.Errorf("Expected CurrentModel to be 'gpt-default-new', got '%s'", a.CurrentModel)
	}
}

func TestSyncWithRuntimeConfig_RebuildsPromptForClaudeOpus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	cfg.DefaultProvider = "claude"
	cfg.ProviderModels["claude"] = config.ProviderModelConfig{
		DefaultModel: "claude-opus-4-6",
	}
	runtime := NewAgentRuntimeWithConfig(cfg)

	a := &Agent{
		ProviderName:    "claude",
		CurrentModel:    "claude-sonnet-4-6",
		CurrentProvider: &MockProvider{name: "claude"},
		SystemPrompt:    prompt.BuildProviderSystemPromptWithConfig(prompt.SystemPrompt, "claude", "claude-sonnet-4-6", runtime.Config),
		Runtime:         runtime,
	}

	a.SyncWithRuntimeConfig()

	if !strings.Contains(a.SystemPrompt, "### Stable Working Reference") {
		t.Fatal("expected SyncWithGlobalConfig to rebuild the Claude Opus system prompt")
	}
}

func TestSyncWithRuntimeConfig_UsesRuntimeOutputForSwitchWarning(t *testing.T) {
	var out bytes.Buffer
	runtime := NewAgentRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)
	runtime.Config.DefaultProvider = "missing-provider"

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	a.SyncWithRuntimeConfig()

	if !strings.Contains(out.String(), "Warning: Failed to switch provider") {
		t.Fatalf("expected runtime output to contain switch warning, got %q", out.String())
	}
}
