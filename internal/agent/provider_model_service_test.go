package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestSwitchModelForCurrentProvider_ReturnsOutcomeAndPersistsConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-old"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-old"})
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	var out bytes.Buffer
	provider := &mockCacheClearableProviderForModel{name: "openai"}
	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-old",
		CurrentProvider:   provider,
		Stats:             NewSessionStats("openai", "gpt-old"),
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-old"),
		},
	}

	outcome := agent.SwitchModelForCurrentProvider("gpt-next")
	if outcome.OldModel != "gpt-old" || outcome.NewModel != "gpt-next" {
		t.Fatalf("outcome models = (%q, %q), want (gpt-old, gpt-next)", outcome.OldModel, outcome.NewModel)
	}
	if outcome.LoadConfigErr != nil || outcome.SaveConfigErr != nil || !outcome.ConfigSaved {
		t.Fatalf("outcome persistence = saved:%v load:%v save:%v, want saved without errors", outcome.ConfigSaved, outcome.LoadConfigErr, outcome.SaveConfigErr)
	}
	if out.String() != "" {
		t.Fatalf("SwitchModelForCurrentProvider() wrote output %q, want no user-facing output", out.String())
	}
	if !provider.cleared {
		t.Fatal("current provider cache should be cleared")
	}
	if agent.CurrentModel != "gpt-next" || agent.session.Model != "gpt-next" || agent.Stats.Model != "gpt-next" {
		t.Fatalf("runtime model mirrors = (%q, %q, %q), want gpt-next", agent.CurrentModel, agent.session.Model, agent.Stats.Model)
	}

	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.DefaultModel != "gpt-next" {
		t.Fatalf("DefaultModel = %q, want gpt-next", loaded.DefaultModel)
	}
	if got := loaded.ProviderModelsForSave()["openai"].DefaultModel; got != "gpt-next" {
		t.Fatalf("ProviderModelsForSave()[openai].DefaultModel = %q, want gpt-next", got)
	}
}

func TestSwitchModelForCurrentProvider_ConfigLoadFailureKeepsSessionSwitch(t *testing.T) {
	withConfigCommandHooks(t)
	loadConfigForCommand = func() (*config.Config, error) {
		return nil, errors.New("load failed")
	}
	saveConfigForCommand = func(cfg *config.Config) error {
		t.Fatal("saveConfigForCommand should not be called after load failure")
		return nil
	}

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)

	outcome := agent.SwitchModelForCurrentProvider("session-only-model")
	if outcome.LoadConfigErr == nil {
		t.Fatal("LoadConfigErr = nil, want load failure")
	}
	if outcome.SaveConfigErr != nil || outcome.ConfigSaved {
		t.Fatalf("outcome persistence = saved:%v save:%v, want unsaved without save error", outcome.ConfigSaved, outcome.SaveConfigErr)
	}
	if agent.CurrentModel != "session-only-model" || agent.session.Model != "session-only-model" {
		t.Fatalf("session model = (%q, %q), want session-only-model", agent.CurrentModel, agent.session.Model)
	}
	if out.String() != "" {
		t.Fatalf("SwitchModelForCurrentProvider() wrote output %q, want no warning output", out.String())
	}
}

func TestSwitchModelForCurrentProvider_ConfigSaveFailureKeepsSessionSwitch(t *testing.T) {
	withConfigCommandHooks(t)
	loadConfigForCommand = func() (*config.Config, error) {
		cfg := newProjectMapDisabledConfig()
		cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{DefaultModel: "deepseek-chat"})
		return cfg, nil
	}
	saveConfigForCommand = func(cfg *config.Config) error {
		return errors.New("save failed")
	}

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)

	outcome := agent.SwitchModelForCurrentProvider("session-only-model")
	if outcome.LoadConfigErr != nil || outcome.SaveConfigErr == nil || outcome.ConfigSaved {
		t.Fatalf("outcome persistence = saved:%v load:%v save:%v, want save failure", outcome.ConfigSaved, outcome.LoadConfigErr, outcome.SaveConfigErr)
	}
	if agent.CurrentModel != "session-only-model" || agent.session.Model != "session-only-model" {
		t.Fatalf("session model = (%q, %q), want session-only-model", agent.CurrentModel, agent.session.Model)
	}
	if out.String() != "" {
		t.Fatalf("SwitchModelForCurrentProvider() wrote output %q, want no warning output", out.String())
	}
}

func TestSwitchProviderModel_ReturnsOutcomeWithoutPrinting(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")

	cfg := newProjectMapDisabledConfig()
	var out bytes.Buffer
	oldProvider := &mockCacheClearableProviderForModel{name: "openai"}
	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-old",
		CurrentProvider:   oldProvider,
		Stats:             NewSessionStats("openai", "gpt-old"),
		History: []api.Message{
			{Role: "user", Content: "old task"},
		},
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-old"),
		},
	}
	agent.session.AddMessage("user", "old task", agent.CurrentModel)

	outcome, err := agent.SwitchProviderModel("ollama", "qwen2.5-coder:14b")
	if err != nil {
		t.Fatalf("SwitchProviderModel() error = %v", err)
	}
	if outcome.OldProvider != "openai" || outcome.NewProvider != "ollama" {
		t.Fatalf("outcome providers = (%q, %q), want (openai, ollama)", outcome.OldProvider, outcome.NewProvider)
	}
	if outcome.OldModel != "gpt-old" || outcome.NewModel != "qwen2.5-coder:14b" {
		t.Fatalf("outcome models = (%q, %q), want (gpt-old, qwen2.5-coder:14b)", outcome.OldModel, outcome.NewModel)
	}
	if !outcome.HistoryCleared {
		t.Fatal("HistoryCleared = false, want true")
	}
	if out.String() != "" {
		t.Fatalf("SwitchProviderModel() wrote output %q, want no user-facing output", out.String())
	}
	if !oldProvider.cleared {
		t.Fatal("old provider cache should be cleared")
	}
	if agent.ProviderName != "ollama" || agent.ProviderConfigKey != "ollama" || agent.CurrentModel != "qwen2.5-coder:14b" {
		t.Fatalf("agent provider/model = (%q, %q, %q), want (ollama, ollama, qwen2.5-coder:14b)", agent.ProviderName, agent.ProviderConfigKey, agent.CurrentModel)
	}
	if len(agent.History) != 0 || len(agent.session.Messages) != 0 {
		t.Fatalf("conversation should be reset, got history=%d session=%d", len(agent.History), len(agent.session.Messages))
	}
	if agent.Stats.Provider != "ollama" || agent.Stats.Model != "qwen2.5-coder:14b" {
		t.Fatalf("stats provider/model = (%q, %q), want (ollama, qwen2.5-coder:14b)", agent.Stats.Provider, agent.Stats.Model)
	}
}
