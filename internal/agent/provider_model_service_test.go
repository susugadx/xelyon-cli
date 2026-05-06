package agent

import (
	"bytes"
	"errors"
	"reflect"
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

func TestProviderCandidates_DisplayOrderCurrentAndCredentialStatus(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_API_KEY", "")
	t.Setenv("AZURE_OPENAI_BASE_URL", "")

	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-5.4",
		Runtime:           NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
	}

	got := agent.ProviderCandidates()
	if len(got) < 4 {
		t.Fatalf("ProviderCandidates len = %d, want provider list", len(got))
	}
	wantPrefix := []string{"deepseek", "claude", "openai", "azure", "gemini"}
	for i, want := range wantPrefix {
		if got[i].Key != want {
			t.Fatalf("ProviderCandidates[%d].Key = %q, want %q; all=%#v", i, got[i].Key, want, got)
		}
	}

	byKey := map[string]ProviderCandidate{}
	for _, candidate := range got {
		byKey[candidate.Key] = candidate
	}
	if !byKey["openai"].Current {
		t.Fatalf("openai candidate should be current: %#v", byKey["openai"])
	}
	if byKey["openai"].CredentialStatus != ProviderCredentialConfigured {
		t.Fatalf("openai status = %q, want configured", byKey["openai"].CredentialStatus)
	}
	if byKey["azure"].CredentialStatus != ProviderCredentialMissingKey {
		t.Fatalf("azure status = %q, want missing key", byKey["azure"].CredentialStatus)
	}
	if byKey["ollama"].CredentialStatus != ProviderCredentialLocal {
		t.Fatalf("ollama status = %q, want local", byKey["ollama"].CredentialStatus)
	}
	if byKey["bedrock"].CredentialStatus != ProviderCredentialAWSAuth {
		t.Fatalf("bedrock status = %q, want aws auth", byKey["bedrock"].CredentialStatus)
	}
}

func TestProviderCandidates_AppendsCurrentAliasOwner(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "anthropic-custom",
		Runtime:           NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
	}

	got := agent.ProviderCandidates()
	byKey := map[string]ProviderCandidate{}
	for _, candidate := range got {
		byKey[candidate.Key] = candidate
	}
	if byKey["claude"].Current {
		t.Fatalf("canonical claude candidate should not be current when session owner is anthropic: %#v", byKey["claude"])
	}
	if !byKey["anthropic"].Current {
		t.Fatalf("anthropic alias candidate should be appended as current: %#v", got)
	}
	if byKey["anthropic"].CredentialStatus != ProviderCredentialConfigured {
		t.Fatalf("anthropic status = %q, want configured", byKey["anthropic"].CredentialStatus)
	}
}

func TestModelCandidates_KnownDefaultCurrentCustomStableDedupe(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-session-default"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "gpt-custom-default"})
	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-current",
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	got := agent.ModelCandidates("openai")
	names := modelCandidateNames(got)
	wantPrefix := []string{"gpt-5.5", "gpt-5.5-pro", "gpt-5.4"}
	if !reflect.DeepEqual(names[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("model prefix = %v, want %v; all=%v", names[:len(wantPrefix)], wantPrefix, names)
	}
	if countCandidateName(got, "gpt-current") != 1 {
		t.Fatalf("current model should be added once, got candidates=%#v", got)
	}
	if countCandidateName(got, "gpt-custom-default") != 1 {
		t.Fatalf("default model should be added once, got candidates=%#v", got)
	}
	if c := candidateByName(got, "gpt-current"); !c.Current {
		t.Fatalf("gpt-current candidate = %#v, want current", c)
	}
	if c := candidateByName(got, "gpt-custom-default"); !c.Default {
		t.Fatalf("gpt-custom-default candidate = %#v, want default", c)
	}
	last := got[len(got)-1]
	if !last.Custom || last.Name != "Custom model..." {
		t.Fatalf("last candidate = %#v, want custom model row", last)
	}
}

func TestModelCandidates_AzureDeploymentOnly(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{DefaultModel: "corp-gpt55-deployment"})
	agent := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "session-deployment",
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	got := agent.ModelCandidates("azure")
	names := modelCandidateNames(got)
	want := []string{"session-deployment", "corp-gpt55-deployment", "Custom deployment..."}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("azure candidates = %v, want %v; full=%#v", names, want, got)
	}
	if c := candidateByName(got, "session-deployment"); !c.Current {
		t.Fatalf("session deployment candidate = %#v, want current", c)
	}
	if c := candidateByName(got, "corp-gpt55-deployment"); !c.Default {
		t.Fatalf("default deployment candidate = %#v, want default", c)
	}
}

func TestModelCandidates_OllamaLiveListSuccessAndFallback(t *testing.T) {
	old := listOllamaModelsForCandidates
	t.Cleanup(func() { listOllamaModelsForCandidates = old })

	cfg := newProjectMapDisabledConfig()
	agent := &Agent{Runtime: NewAgentRuntimeWithConfig(cfg)}

	listOllamaModelsForCandidates = func(_ *Agent, _ string) ([]string, error) {
		return []string{"live-a", "live-b"}, nil
	}
	live := modelCandidateNames(agent.ModelCandidates("ollama"))
	if !reflect.DeepEqual(live[:2], []string{"live-a", "live-b"}) {
		t.Fatalf("ollama live prefix = %v, want live list; all=%v", live[:2], live)
	}

	listOllamaModelsForCandidates = func(_ *Agent, _ string) ([]string, error) {
		return nil, errors.New("list failed")
	}
	fallback := modelCandidateNames(agent.ModelCandidates("ollama"))
	if fallback[0] != "qwen2.5-coder:7b" {
		t.Fatalf("ollama fallback first = %q, want known/default fallback; all=%v", fallback[0], fallback)
	}
}

func modelCandidateNames(candidates []ModelCandidate) []string {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.Name)
	}
	return names
}

func countCandidateName(candidates []ModelCandidate, name string) int {
	count := 0
	for _, candidate := range candidates {
		if candidate.Name == name {
			count++
		}
	}
	return count
}

func candidateByName(candidates []ModelCandidate, name string) ModelCandidate {
	for _, candidate := range candidates {
		if candidate.Name == name {
			return candidate
		}
	}
	return ModelCandidate{}
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
