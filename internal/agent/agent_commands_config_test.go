package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// mockCacheClearableProviderForModel はキャッシュクリアと Provider を実装したモック（モデルコマンド用）
type mockCacheClearableProviderForModel struct {
	cleared bool
	name    string
}

func (m *mockCacheClearableProviderForModel) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock-cache"
}

func (m *mockCacheClearableProviderForModel) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "", nil
}

func (m *mockCacheClearableProviderForModel) SupportsImages() bool {
	return false
}

func (m *mockCacheClearableProviderForModel) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return "", nil
}

func (m *mockCacheClearableProviderForModel) IsFunctionCallingEnabled() bool {
	return false
}

func (m *mockCacheClearableProviderForModel) ClearCache() {
	m.cleared = true
}

func TestHandleModelCommand_ClearCache(t *testing.T) {
	agent := &Agent{
		ProviderName: "mock",
		CurrentModel: "old-model",
	}

	mockProvider := &mockCacheClearableProviderForModel{}
	agent.CurrentProvider = mockProvider

	// /model new-model をシミュレート
	args := []string{"new-model"}
	result := handleModelCommand(agent, args)

	// コマンドが正常に処理されたことを確認
	assert.True(t, result)
	// モデルが切り替わったことを確認
	assert.Equal(t, "new-model", agent.CurrentModel)
	// ClearCacheが呼ばれたことを確認
	assert.True(t, mockProvider.cleared, "ClearCache should be called when switching model")
}

func TestHandleModelCommand_RebuildsClaudePromptForOpus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	runtime := NewAgentRuntimeWithConfig(cfg)

	agent := &Agent{
		ProviderName:    "claude",
		CurrentModel:    "claude-sonnet-4-6",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "claude"},
		SystemPrompt:    prompt.BuildProviderSystemPromptWithConfig(prompt.SystemPrompt, "claude", "claude-sonnet-4-6", cfg),
		Runtime:         runtime,
	}

	result := handleModelCommand(agent, []string{"claude-opus-4-6"})

	assert.True(t, result)
	assert.Contains(t, agent.SystemPrompt, "## Workflow Rules")
}

func TestHandleModelCommand_NoArgs_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName:    "mock",
		CurrentModel:    "test-model",
		CurrentProvider: &mockCacheClearableProviderForModel{name: "mock"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	result := handleModelCommand(agent, nil)
	if !result {
		t.Fatal("handleModelCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "Current model: test-model") {
		t.Fatalf("expected runtime output to contain current model, got %q", output)
	}
	if !strings.Contains(output, "Usage: /model <model-name>") {
		t.Fatalf("expected runtime output to contain usage, got %q", output)
	}
}

func TestHandleProvidersCommand_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		ProviderName: "ollama",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	result := handleProvidersCommand(agent)
	if !result {
		t.Fatal("handleProvidersCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "利用可能なプロバイダー") {
		t.Fatalf("expected runtime output to contain providers header, got %q", output)
	}
	if !strings.Contains(output, "/use <provider>") {
		t.Fatalf("expected runtime output to contain usage hint, got %q", output)
	}
	if !strings.Contains(output, "openai") {
		t.Fatalf("expected runtime output to contain provider list, got %q", output)
	}
}
