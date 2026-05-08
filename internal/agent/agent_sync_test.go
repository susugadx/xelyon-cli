package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// MockProvider implements api.Provider for testing
type MockProvider struct {
	name      string
	configKey string
}

func (m *MockProvider) Name() string { return m.name }
func (m *MockProvider) ProviderConfigKey() string {
	if m.configKey != "" {
		return m.configKey
	}
	return m.name
}
func (m *MockProvider) SetProviderConfigKey(key string) { m.configKey = key }
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
	cfg := newProjectMapDisabledConfig()
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
		Stats:           NewSessionStats("openai", "gpt-old"),
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-old"),
		},
	}

	// Scenario 1: Update provider specific model in config
	runtime.Config.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel: "gpt-new",
	}

	a.SyncWithRuntimeConfig()

	if a.CurrentModel != "gpt-new" {
		t.Errorf("Expected CurrentModel to be 'gpt-new', got '%s'", a.CurrentModel)
	}
	if a.session == nil || a.session.Model != "gpt-new" {
		t.Fatalf("session.Model = %q, want %q", a.session.Model, "gpt-new")
	}
	if a.Stats == nil || a.Stats.Model != "gpt-new" {
		t.Fatalf("Stats.Model = %q, want %q", a.Stats.Model, "gpt-new")
	}
}

func TestCurrentProviderConfigKey_CanonicalizesAzureDisplayName(t *testing.T) {
	a := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "Azure OpenAI",
		CurrentProvider:   &MockProvider{name: "Azure OpenAI", configKey: "Azure OpenAI"},
	}

	if got := a.currentProviderConfigKey(); got != "azure" {
		t.Fatalf("currentProviderConfigKey() = %q, want azure", got)
	}
}

func TestNewAgentWithRuntime_PreservesMoonshotProviderConfigKey(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "test-key")

	provider, err := api.NewProvider("moonshot")
	if err != nil {
		t.Fatalf("NewProvider(moonshot) error = %v", err)
	}

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "moonshot"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"moonshot": {DefaultModel: "moonshot-custom"},
		"kimi":     {DefaultModel: "kimi-custom"},
	})

	agent := NewAgentWithRuntime("moonshot-custom", provider, false, NewAgentRuntimeWithConfig(cfg))
	if agent.ProviderName != "kimi" {
		t.Fatalf("ProviderName = %q, want kimi", agent.ProviderName)
	}
	if agent.ProviderConfigKey != "moonshot" {
		t.Fatalf("ProviderConfigKey = %q, want moonshot", agent.ProviderConfigKey)
	}
	if got := providerConfigKeyFromProvider(agent.CurrentProvider); got != "moonshot" {
		t.Fatalf("provider config key = %q, want moonshot", got)
	}
	if agent.session == nil || agent.session.ProviderConfigKey != "moonshot" {
		t.Fatalf("session.ProviderConfigKey = %q, want moonshot", agent.session.ProviderConfigKey)
	}
}

func TestSyncWithRuntimeConfig_DefaultModelShadowing(t *testing.T) {
	// Setup
	cfg := newProjectMapDisabledConfig()
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
	cfg := newProjectMapDisabledConfig()
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

func TestSyncWithRuntimeConfig_DefaultModelWinsWhenExplicitEntryHasOnlyNonModelSettings(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-default-new"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		MaxOutputTokens: 99999,
	})
	runtime := NewAgentRuntimeWithConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	a.SyncWithRuntimeConfig()

	if a.CurrentModel != "gpt-default-new" {
		t.Errorf("Expected CurrentModel to be 'gpt-default-new', got '%s'", a.CurrentModel)
	}
}

func TestSyncWithRuntimeConfig_FallsBackToProviderDefaultWhenGlobalModelBelongsToDifferentProvider(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "deepseek-chat"
	runtime := NewAgentRuntimeWithConfig(cfg)

	a := &Agent{
		ProviderName:    "openai",
		CurrentModel:    "gpt-old",
		CurrentProvider: &MockProvider{name: "openai"},
		Runtime:         runtime,
	}

	a.SyncWithRuntimeConfig()

	want := config.DefaultConfig().ProviderModels["openai"].DefaultModel
	if a.CurrentModel != want {
		t.Errorf("Expected CurrentModel to be %q, got %q", want, a.CurrentModel)
	}
}

func TestSyncWithRuntimeConfig_RebuildsPromptForClaudeOpus(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
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

	if !strings.Contains(a.SystemPrompt, "## Workflow Rules") {
		t.Fatal("expected SyncWithRuntimeConfig to rebuild the system prompt")
	}
}

func TestSyncWithRuntimeConfig_UsesRuntimeOutputForSwitchWarning(t *testing.T) {
	var out bytes.Buffer
	runtime := NewAgentRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)
	runtime.Config.ProjectMap.Enabled = false
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

func TestSyncWithRuntimeConfig_DoesNotReswitchWhenAnthropicAliasOwnerIsUnchanged(t *testing.T) {
	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
	})
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	currentProvider := &MockProvider{name: "claude"}
	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "claude-old",
		CurrentProvider:   currentProvider,
		Runtime:           runtime,
	}

	a.SyncWithRuntimeConfig()

	if a.ProviderName != "claude" {
		t.Fatalf("ProviderName = %q, want %q", a.ProviderName, "claude")
	}
	if a.CurrentProvider != currentProvider {
		t.Fatal("CurrentProvider should not be recreated for anthropic/claude alias sync")
	}
	if a.ProviderConfigKey != "anthropic" {
		t.Fatalf("ProviderConfigKey = %q, want %q", a.ProviderConfigKey, "anthropic")
	}
	if a.CurrentModel != "anthropic-custom" {
		t.Fatalf("CurrentModel = %q, want %q", a.CurrentModel, "anthropic-custom")
	}
	if strings.Contains(out.String(), "Warning: Failed to switch provider") {
		t.Fatalf("unexpected switch warning: %q", out.String())
	}
}

func TestSyncWithRuntimeConfig_RebindsAliasOwnerWithinSameRuntimeIdentity(t *testing.T) {
	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})
	cfg.Compression.ProviderThresholds["anthropic"] = 123456
	cfg.Compression.ProviderThresholds["claude"] = 654321

	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	currentProvider := &MockProvider{name: "claude"}
	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		CurrentModel:      "claude-old",
		CurrentProvider:   currentProvider,
		Runtime:           runtime,
	}

	a.SyncWithRuntimeConfig()

	if a.ProviderName != "claude" {
		t.Fatalf("ProviderName = %q, want %q", a.ProviderName, "claude")
	}
	if a.ProviderConfigKey != "anthropic" {
		t.Fatalf("ProviderConfigKey = %q, want %q", a.ProviderConfigKey, "anthropic")
	}
	if providerConfigKeyFromProvider(a.CurrentProvider) != "anthropic" {
		t.Fatalf("provider config key = %q, want %q", providerConfigKeyFromProvider(a.CurrentProvider), "anthropic")
	}
	if a.CurrentProvider != currentProvider {
		t.Fatal("CurrentProvider should be reused when only the config owner alias changes")
	}
	if a.CurrentModel != "anthropic-custom" {
		t.Fatalf("CurrentModel = %q, want %q", a.CurrentModel, "anthropic-custom")
	}
	if got := GetProviderCompressThresholdWithConfig(cfg, a.sessionProviderConfigKey(cfg), a.CurrentModel); got != 123456 {
		t.Fatalf("compression threshold = %d, want %d", got, 123456)
	}
	if strings.Contains(out.String(), "Warning: Failed to switch provider") {
		t.Fatalf("unexpected switch warning: %q", out.String())
	}
}

func TestSyncWithRuntimeConfig_CanonicalizesDisplayNameDefaultProviderOwner(t *testing.T) {
	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "Azure OpenAI"
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{DefaultModel: "corp-gpt55-deployment"})

	currentProvider := &MockProvider{name: "Azure OpenAI", configKey: "azure"}
	a := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "old-deployment",
		CurrentProvider:   currentProvider,
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	a.SyncWithRuntimeConfig()

	if a.ProviderName != "azure" {
		t.Fatalf("ProviderName = %q, want azure", a.ProviderName)
	}
	if a.CurrentProvider != currentProvider {
		t.Fatal("CurrentProvider should be reused for Azure display-name default_provider")
	}
	if a.ProviderConfigKey != "azure" {
		t.Fatalf("ProviderConfigKey = %q, want azure", a.ProviderConfigKey)
	}
	if providerConfigKeyFromProvider(a.CurrentProvider) != "azure" {
		t.Fatalf("provider config key = %q, want azure", providerConfigKeyFromProvider(a.CurrentProvider))
	}
	if a.CurrentModel != "corp-gpt55-deployment" {
		t.Fatalf("CurrentModel = %q, want Azure deployment", a.CurrentModel)
	}
	if strings.Contains(out.String(), "Warning: Failed to switch provider") {
		t.Fatalf("unexpected switch warning: %q", out.String())
	}
}

func TestSyncWithRuntimeConfig_PrefersDefaultProviderAliasModelForSameRuntimeIdentity(t *testing.T) {
	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})

	a := &Agent{
		ProviderName:    "claude",
		CurrentModel:    "claude-old",
		CurrentProvider: &MockProvider{name: "claude"},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	a.SyncWithRuntimeConfig()

	if a.ProviderConfigKey != "anthropic" {
		t.Fatalf("ProviderConfigKey = %q, want %q", a.ProviderConfigKey, "anthropic")
	}
	if a.CurrentModel != "anthropic-custom" {
		t.Fatalf("CurrentModel = %q, want %q", a.CurrentModel, "anthropic-custom")
	}
	if strings.Contains(out.String(), "Warning: Failed to switch provider") {
		t.Fatalf("unexpected switch warning: %q", out.String())
	}
}

func TestSyncWithRuntimeConfig_PrefersSessionProviderConfigKeyWhenDefaultProviderDiffers(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})
	runtime := NewAgentRuntimeWithConfig(cfg)

	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "claude-old",
		CurrentProvider:   &MockProvider{name: "claude"},
		Runtime:           runtime,
	}

	a.SyncWithRuntimeConfig()

	if a.ProviderName != "deepseek" {
		t.Fatalf("ProviderName = %q, want %q", a.ProviderName, "deepseek")
	}
	if a.ProviderConfigKey != "deepseek" {
		t.Fatalf("ProviderConfigKey = %q, want %q", a.ProviderConfigKey, "deepseek")
	}
	if a.CurrentModel != "deepseek-v4-flash" {
		t.Fatalf("CurrentModel = %q, want %q", a.CurrentModel, "deepseek-v4-flash")
	}
}

func TestSyncWithRuntimeConfig_RestoresSavedResponseIDWhenRuntimeReturnsToMatchingContext(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "saved-model"})
	runtime := NewAgentRuntimeWithConfig(cfg)

	provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}}
	session := history.NewSession("old-runtime-model")
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai"
	session.ResponseID = "resp_saved"
	session.ResponseModel = "saved-model"
	session.ResponseProviderName = "openai"
	session.ResponseProviderConfigKey = "openai"

	a := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "different-model",
		CurrentProvider:   provider,
		Runtime:           runtime,
		agentConversationState: agentConversationState{
			session: session,
		},
	}

	a.SyncWithRuntimeConfig()

	if a.CurrentModel != "saved-model" {
		t.Fatalf("CurrentModel = %q, want %q", a.CurrentModel, "saved-model")
	}
	if provider.responseID != "resp_saved" {
		t.Fatalf("provider.responseID = %q, want restored saved response id", provider.responseID)
	}
	if a.session.Model != "saved-model" {
		t.Fatalf("session.Model = %q, want reconciled runtime model", a.session.Model)
	}
}
