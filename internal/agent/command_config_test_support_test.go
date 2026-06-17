package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
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

type mockModelListerProvider struct {
	mockCacheClearableProviderForModel
	models []string
	err    error
}

func (m *mockModelListerProvider) ListModels() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.models, nil
}

type scriptedConfigMenu struct {
	runResults  []configMenuRunResult
	showResults []configMenuShowResult
	editResults []configMenuEditResult
	runIndex    int
	showIndex   int
	editIndex   int
}

type configMenuRunResult struct {
	category *config.ConfigCategory
	err      error
}

type configMenuShowResult struct {
	field *config.ConfigField
	err   error
}

type configMenuEditResult struct {
	value   interface{}
	changed bool
	err     error
	mutate  func()
}

func (m *scriptedConfigMenu) Run() (*config.ConfigCategory, error) {
	if m.runIndex >= len(m.runResults) {
		return nil, errors.New("unexpected Run call")
	}
	result := m.runResults[m.runIndex]
	m.runIndex++
	return result.category, result.err
}

func (m *scriptedConfigMenu) ShowFieldList(cat *config.ConfigCategory) (*config.ConfigField, error) {
	if m.showIndex >= len(m.showResults) {
		return nil, errors.New("unexpected ShowFieldList call")
	}
	result := m.showResults[m.showIndex]
	m.showIndex++
	return result.field, result.err
}

func (m *scriptedConfigMenu) EditField(field *config.ConfigField) (interface{}, bool, error) {
	if m.editIndex >= len(m.editResults) {
		return nil, false, errors.New("unexpected EditField call")
	}
	result := m.editResults[m.editIndex]
	m.editIndex++
	if result.mutate != nil {
		result.mutate()
	}
	return result.value, result.changed, result.err
}

func withConfigCommandHooks(t *testing.T) {
	t.Helper()

	oldLoad := loadConfigForCommand
	oldSave := saveConfigForCommand
	oldShow := showConfigForCommand
	oldSet := setFieldValueForCommand
	oldBuild := buildConfigRegistryForCommand
	oldMenu := newConfigMenuForCommand

	t.Cleanup(func() {
		loadConfigForCommand = oldLoad
		saveConfigForCommand = oldSave
		showConfigForCommand = oldShow
		setFieldValueForCommand = oldSet
		buildConfigRegistryForCommand = oldBuild
		newConfigMenuForCommand = oldMenu
	})
}

func newConfigCommandTestAgent(cfg *config.Config, out *bytes.Buffer) *Agent {
	if cfg == nil {
		cfg = newProjectMapDisabledConfig()
	}
	cfg.DefaultProvider = "deepseek"
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), out, out)
	return &Agent{
		ProviderName:    "deepseek",
		CurrentModel:    "deepseek-chat",
		CurrentProvider: &mockProvider{name: "deepseek"},
		Runtime:         runtime,
		agentConversationState: agentConversationState{
			session: history.NewSession("deepseek-chat"),
		},
	}
}

type mockResponseIDProvider struct {
	mockProvider
	responseID string
}

func (m *mockResponseIDProvider) HasCachedResponseID() bool {
	return m.responseID != ""
}

func (m *mockResponseIDProvider) SetResponseID(id string) {
	m.responseID = id
}

func (m *mockResponseIDProvider) GetResponseID() string {
	return m.responseID
}

func (m *mockResponseIDProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	return "ok", nil
}
