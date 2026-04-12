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

func TestHandleModelCommand_ListsInstalledModelsAndWarnings(t *testing.T) {
	t.Run("lists installed models", func(t *testing.T) {
		var out bytes.Buffer
		agent := &Agent{
			ProviderName:    "mock",
			CurrentModel:    "test-model",
			CurrentProvider: &mockModelListerProvider{models: []string{"model-a", "model-b"}},
			Runtime: &AgentRuntime{
				UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
			},
		}

		if !handleModelCommand(agent, nil) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Installed models:") || !strings.Contains(out.String(), "model-a") {
			t.Fatalf("output = %q, want installed model list", out.String())
		}
	})

	t.Run("warns when model list fails", func(t *testing.T) {
		var out bytes.Buffer
		agent := &Agent{
			ProviderName:    "mock",
			CurrentModel:    "test-model",
			CurrentProvider: &mockModelListerProvider{err: errors.New("list failed")},
			Runtime: &AgentRuntime{
				UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
			},
		}

		if !handleModelCommand(agent, nil) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Could not list models") {
			t.Fatalf("output = %q, want model listing warning", out.String())
		}
	})
}

func TestHandleModelCommand_ConfigLoadAndSaveWarnings(t *testing.T) {
	t.Run("load config failure keeps session change", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return nil, errors.New("load failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)

		if !handleModelCommand(agent, []string{"new-model"}) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if agent.CurrentModel != "new-model" {
			t.Fatalf("CurrentModel = %q, want %q", agent.CurrentModel, "new-model")
		}
		if !strings.Contains(out.String(), "Warning: Failed to load config") {
			t.Fatalf("output = %q, want load warning", out.String())
		}
	})

	t.Run("save config failure keeps session only change", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return newProjectMapDisabledConfig(), nil
		}
		saveConfigForCommand = func(cfg *config.Config) error {
			return errors.New("save failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)

		if !handleModelCommand(agent, []string{"new-model"}) {
			t.Fatal("handleModelCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Model switched for this session only") {
			t.Fatalf("output = %q, want session-only warning", out.String())
		}
	})
}

func TestHandleConfigCommand_LoadErrorAndModelSaveError(t *testing.T) {
	t.Run("load error is reported", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return nil, errors.New("broken config")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleConfigCommand(agent, []string{"show"}) {
			t.Fatal("handleConfigCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Failed to load config") {
			t.Fatalf("output = %q, want load error", out.String())
		}
	})

	t.Run("model save error is reported", func(t *testing.T) {
		withConfigCommandHooks(t)
		loadConfigForCommand = func() (*config.Config, error) {
			return newProjectMapDisabledConfig(), nil
		}
		saveConfigForCommand = func(cfg *config.Config) error {
			return errors.New("save failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleConfigCommand(agent, []string{"model", "next-model"}) {
			t.Fatal("handleConfigCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Failed to save config") {
			t.Fatalf("output = %q, want save error", out.String())
		}
	})
}

func TestHandleConfigCommand_DelegatesInteractiveMode(t *testing.T) {
	withConfigCommandHooks(t)

	cfg := newProjectMapDisabledConfig()
	loadConfigForCommand = func() (*config.Config, error) {
		return cfg, nil
	}

	menu := &scriptedConfigMenu{
		runResults: []configMenuRunResult{{category: nil, err: errors.New("cancel")}},
	}
	var factoryCalls int
	newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
		factoryCalls++
		return menu
	}

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(cfg, &out)
	if !handleConfigCommand(agent, nil) {
		t.Fatal("handleConfigCommand() = false, want true")
	}
	if factoryCalls != 1 {
		t.Fatalf("config menu factory calls = %d, want 1", factoryCalls)
	}
}

func TestRunInteractiveConfig_ScalarAndStructMapFlows(t *testing.T) {
	t.Run("scalar update saves value", func(t *testing.T) {
		withConfigCommandHooks(t)

		cfg := newProjectMapDisabledConfig()
		categories := []config.ConfigCategory{{Name: "provider"}}
		field := &config.ConfigField{Path: "default_model", FieldType: config.FieldTypeString}
		menu := &scriptedConfigMenu{
			runResults: []configMenuRunResult{
				{category: &categories[0]},
				{category: nil, err: errors.New("cancel")},
			},
			showResults: []configMenuShowResult{
				{field: field},
				{field: nil, err: errors.New("back")},
			},
			editResults: []configMenuEditResult{
				{value: "gpt-next", changed: true},
			},
		}

		buildConfigRegistryForCommand = func(cfg *config.Config) []config.ConfigCategory {
			return categories
		}
		newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
			return menu
		}
		setFieldValueForCommand = func(cfg *config.Config, path string, value interface{}) error {
			cfg.DefaultModel = value.(string)
			return nil
		}

		var saved int
		saveConfigForCommand = func(cfg *config.Config) error {
			saved++
			return nil
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(cfg, &out)
		runInteractiveConfig(agent, cfg)

		if cfg.DefaultModel != "gpt-next" {
			t.Fatalf("DefaultModel = %q, want %q", cfg.DefaultModel, "gpt-next")
		}
		if saved != 1 {
			t.Fatalf("save count = %d, want 1", saved)
		}
		if !strings.Contains(out.String(), "✓ Saved: default_model = gpt-next") {
			t.Fatalf("output = %q, want saved message", out.String())
		}
	})

	t.Run("struct map save error is reported", func(t *testing.T) {
		withConfigCommandHooks(t)

		cfg := newProjectMapDisabledConfig()
		categories := []config.ConfigCategory{{Name: "provider"}}
		field := &config.ConfigField{Path: "provider_models", FieldType: config.FieldTypeStructMap}
		menu := &scriptedConfigMenu{
			runResults: []configMenuRunResult{
				{category: &categories[0]},
				{category: nil, err: errors.New("cancel")},
			},
			showResults: []configMenuShowResult{
				{field: field},
				{field: nil, err: errors.New("back")},
			},
			editResults: []configMenuEditResult{
				{value: nil, changed: true},
			},
		}

		buildConfigRegistryForCommand = func(cfg *config.Config) []config.ConfigCategory {
			return categories
		}
		newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
			return menu
		}
		saveConfigForCommand = func(cfg *config.Config) error {
			return errors.New("save failed")
		}

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(cfg, &out)
		runInteractiveConfig(agent, cfg)

		if !strings.Contains(out.String(), "Error saving: save failed") {
			t.Fatalf("output = %q, want struct-map save error", out.String())
		}
	})
}

func TestIsNonInteractiveConfigSubcommand(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"show"}, true},
		{[]string{"show", "extra"}, false},
		{[]string{"model"}, false},
		{[]string{"model", "gpt-5"}, true},
		{[]string{"other"}, false},
	}

	for _, tt := range tests {
		if got := isNonInteractiveConfigSubcommand(tt.args); got != tt.want {
			t.Fatalf("isNonInteractiveConfigSubcommand(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestHandleUseCommand_HelpAndErrorBranches(t *testing.T) {
	t.Run("usage without args", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleUseCommand(agent, nil) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Usage: /use <provider> [model]") {
			t.Fatalf("output = %q, want usage", out.String())
		}
	})

	t.Run("unknown provider is reported", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleUseCommand(agent, []string{"unknown-provider"}) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Unknown provider") {
			t.Fatalf("output = %q, want unknown provider message", out.String())
		}
	})

	t.Run("already using provider without model prints hint", func(t *testing.T) {
		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		agent.ProviderName = "openai"
		agent.ProviderConfigKey = "openai"
		agent.CurrentModel = "gpt-current"
		if !handleUseCommand(agent, []string{"openai"}) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "Already using openai") || !strings.Contains(out.String(), "Hint: Use '/use <provider> <model>'") {
			t.Fatalf("output = %q, want already-using hint", out.String())
		}
	})

	t.Run("provider switch failure prints setup hint", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "")

		var out bytes.Buffer
		agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
		if !handleUseCommand(agent, []string{"openai"}) {
			t.Fatal("handleUseCommand() = false, want true")
		}
		if !strings.Contains(out.String(), "OPENAI_API_KEY") {
			t.Fatalf("output = %q, want OPENAI_API_KEY setup hint", out.String())
		}
	})
}

func TestHandleProvidersCommand_ShowsAddedCurrentAliasStatus(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)
	agent.ProviderName = "claude"
	agent.ProviderConfigKey = "anthropic"

	if !handleProvidersCommand(agent) {
		t.Fatal("handleProvidersCommand() = false, want true")
	}
	output := out.String()
	if !strings.Contains(output, "anthropic") || !strings.Contains(output, "(API key設定済み)") {
		t.Fatalf("output = %q, want anthropic alias entry with configured status", output)
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
