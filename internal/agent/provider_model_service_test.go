package agent

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
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

func TestSwitchModelForCurrentProvider_GeminiUnsupportedFunctionCallingModelFails(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "gemini"
	cfg.DefaultModel = "gemini-3.5-flash"
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{DefaultModel: "gemini-3.5-flash"})

	var out bytes.Buffer
	provider := &mockCacheClearableProviderForModel{name: "gemini"}
	agent := &Agent{
		ProviderName:      "gemini",
		ProviderConfigKey: "gemini",
		CurrentModel:      "gemini-3.5-flash",
		CurrentProvider:   provider,
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	outcome := agent.SwitchModelForCurrentProvider("gemini-2.0-flash-lite")
	if outcome.ValidationErr == nil {
		t.Fatal("ValidationErr = nil, want Gemini function calling validation error")
	}
	if agent.CurrentModel != "gemini-3.5-flash" {
		t.Fatalf("CurrentModel = %q, want unchanged", agent.CurrentModel)
	}
	if provider.cleared {
		t.Fatal("provider cache should not be cleared when validation fails")
	}
}

func TestSwitchModelForCurrentProvider_GeminiValidatesSavedCandidateConfig(t *testing.T) {
	withConfigCommandHooks(t)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "gemini"
	cfg.DefaultModel = "gemini-3.5-flash"
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
		DefaultModel: "corp-old",
		CatalogModel: "gemini-2.0-flash-lite",
	})
	loadConfigForCommand = func() (*config.Config, error) {
		return config.CloneConfig(cfg), nil
	}
	var saveCalled bool
	saveConfigForCommand = func(cfg *config.Config) error {
		saveCalled = true
		return nil
	}

	var out bytes.Buffer
	provider := &mockCacheClearableProviderForModel{name: "gemini"}
	agent := &Agent{
		ProviderName:      "gemini",
		ProviderConfigKey: "gemini",
		CurrentModel:      "gemini-3.5-flash",
		CurrentProvider:   provider,
		Runtime: &AgentRuntime{
			Config: config.CloneConfig(cfg),
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	outcome := agent.SwitchModelForCurrentProvider("corp-gemini")
	if outcome.ValidationErr == nil {
		t.Fatal("ValidationErr = nil, want saved candidate Gemini catalog_model validation error")
	}
	if !strings.Contains(outcome.ValidationErr.Error(), "catalog_model=gemini-2.0-flash-lite") {
		t.Fatalf("ValidationErr = %v, want stale catalog_model detail", outcome.ValidationErr)
	}
	if saveCalled {
		t.Fatal("saveConfigForCommand should not be called after candidate validation failure")
	}
	if agent.CurrentModel != "gemini-3.5-flash" {
		t.Fatalf("CurrentModel = %q, want unchanged gemini-3.5-flash", agent.CurrentModel)
	}
	if provider.cleared {
		t.Fatal("provider cache should not be cleared when candidate validation fails")
	}
}

func TestSwitchModelForCurrentProvider_AzureDeploymentChangeClearsStaleCatalogModel(t *testing.T) {
	withConfigCommandHooks(t)

	loadConfigForCommand = func() (*config.Config, error) {
		cfg := newProjectMapDisabledConfig()
		cfg.DefaultProvider = "azure"
		cfg.DefaultModel = "dep-a"
		cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
			DefaultModel:    "dep-a",
			CatalogModel:    "gpt-5.4",
			MaxOutputTokens: 64000,
			ModelOverrides:  map[string]config.ModelOverride{"dep-c": {CatalogModel: "gpt-5.5"}},
		})
		return cfg, nil
	}
	var saved *config.Config
	saveConfigForCommand = func(cfg *config.Config) error {
		saved = config.CloneConfig(cfg)
		return nil
	}

	var out bytes.Buffer
	agent := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "dep-a",
		CurrentProvider:   &mockProvider{name: "azure"},
		Runtime: &AgentRuntime{
			Config: newProjectMapDisabledConfig(),
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("dep-a"),
		},
	}

	outcome := agent.SwitchModelForCurrentProvider("dep-b")
	if outcome.LoadConfigErr != nil || outcome.SaveConfigErr != nil || !outcome.ConfigSaved {
		t.Fatalf("outcome persistence = saved:%v load:%v save:%v, want saved without errors", outcome.ConfigSaved, outcome.LoadConfigErr, outcome.SaveConfigErr)
	}
	if saved == nil {
		t.Fatal("saveConfigForCommand was not called")
	}
	pm := saved.ProviderModelsForSave()["azure"]
	if pm.DefaultModel != "dep-b" {
		t.Fatalf("provider_models.azure.default_model = %q, want dep-b", pm.DefaultModel)
	}
	if pm.CatalogModel != "" {
		t.Fatalf("provider_models.azure.catalog_model = %q, want cleared", pm.CatalogModel)
	}
	if pm.MaxOutputTokens != 64000 {
		t.Fatalf("provider_models.azure.max_output_tokens = %d, want preserved", pm.MaxOutputTokens)
	}
	if override := pm.ModelOverrides["dep-c"]; override.CatalogModel != "gpt-5.5" {
		t.Fatalf("provider_models.azure.model_overrides[dep-c] = %#v, want preserved", override)
	}
	resolved := saved.ResolveModelCatalog("azure", "dep-b")
	if resolved.Model != "dep-b" || !resolved.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(azure, dep-b) = %#v, want dep-b configured without catalog", resolved)
	}
}

func TestSwitchModelForCurrentProvider_AzureSameDeploymentPreservesCatalogModel(t *testing.T) {
	withConfigCommandHooks(t)

	loadConfigForCommand = func() (*config.Config, error) {
		cfg := newProjectMapDisabledConfig()
		cfg.DefaultProvider = "azure"
		cfg.DefaultModel = "dep-a"
		cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
			DefaultModel: "dep-a",
			CatalogModel: "gpt-5.4",
		})
		return cfg, nil
	}
	var saved *config.Config
	saveConfigForCommand = func(cfg *config.Config) error {
		saved = config.CloneConfig(cfg)
		return nil
	}

	var out bytes.Buffer
	agent := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "dep-a",
		CurrentProvider:   &mockProvider{name: "azure"},
		Runtime: &AgentRuntime{
			Config: newProjectMapDisabledConfig(),
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("dep-a"),
		},
	}

	outcome := agent.SwitchModelForCurrentProvider("dep-a")
	if outcome.LoadConfigErr != nil || outcome.SaveConfigErr != nil || !outcome.ConfigSaved {
		t.Fatalf("outcome persistence = saved:%v load:%v save:%v, want saved without errors", outcome.ConfigSaved, outcome.LoadConfigErr, outcome.SaveConfigErr)
	}
	if saved == nil {
		t.Fatal("saveConfigForCommand was not called")
	}
	if got := saved.ProviderModelsForSave()["azure"].CatalogModel; got != "gpt-5.4" {
		t.Fatalf("provider_models.azure.catalog_model = %q, want preserved", got)
	}
	if got := saved.ModelCatalogName("azure", "dep-a"); got != "gpt-5.4" {
		t.Fatalf("ModelCatalogName(azure, dep-a) = %q, want gpt-5.4", got)
	}
}

func TestProviderCandidates_DisplayOrderCurrentAndCredentialStatus(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("MOONSHOT_API_KEY", "")
	t.Setenv("AZURE_OPENAI_API_KEY", "")
	t.Setenv("AZURE_OPENAI_BASE_URL", "")
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", filepath.Join(t.TempDir(), "auth"))

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
	wantPrefix := []string{"deepseek", "kimi", "claude", "openai", "openai_subscription", "azure", "gemini"}
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
	if byKey["openai_subscription"].CredentialStatus != ProviderCredentialLoginRequired {
		t.Fatalf("openai_subscription status = %q, want login required", byKey["openai_subscription"].CredentialStatus)
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

func TestProviderCandidates_OpenAISubscriptionLoggedInStatus(t *testing.T) {
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", filepath.Join(t.TempDir(), "auth"))
	if err := openaisubscription.SaveSubscriptionCredential(openaisubscription.DefaultSubscriptionAuthConfig(), openaisubscription.SubscriptionCredential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	agent := &Agent{Runtime: NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())}
	got := agent.ProviderCandidates()
	for _, candidate := range got {
		if candidate.Key == "openai_subscription" {
			if candidate.CredentialStatus != ProviderCredentialLoggedIn {
				t.Fatalf("openai_subscription status = %q, want logged in", candidate.CredentialStatus)
			}
			return
		}
	}
	t.Fatalf("openai_subscription candidate missing: %#v", got)
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

func TestModelCandidates_OpenAISubscriptionAllowlist(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai_subscription"
	agent := &Agent{
		ProviderName:      "openai_subscription",
		ProviderConfigKey: "openai_subscription",
		CurrentModel:      "gpt-5.4-mini",
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	got := agent.ModelCandidates("chatgpt")
	names := modelCandidateNames(got)
	wantPrefix := []string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark"}
	if len(names) < len(wantPrefix) || !reflect.DeepEqual(names[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("subscription model prefix = %v, want %v; all=%v", names[:min(len(names), len(wantPrefix))], wantPrefix, names)
	}
	if c := candidateByName(got, "gpt-5.4-mini"); !c.Current {
		t.Fatalf("gpt-5.4-mini candidate = %#v, want current", c)
	}
	last := got[len(got)-1]
	if !last.Custom || last.Name != "Custom model..." {
		t.Fatalf("last candidate = %#v, want custom model row", last)
	}
}

func TestModelCandidates_IncludesRecommendedGeminiModels(t *testing.T) {
	agent := &Agent{
		ProviderName:      "gemini",
		ProviderConfigKey: "gemini",
		CurrentModel:      "gemini-2.5-flash",
		Runtime:           NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
	}

	got := agent.ModelCandidates("gemini")
	names := modelCandidateNames(got)
	if len(names) == 0 || names[0] != "gemini-3.5-flash" {
		t.Fatalf("gemini candidates prefix = %v, want gemini-3.5-flash first; full=%v", names[:min(len(names), 3)], names)
	}
	if c := candidateByName(got, "gemini-3.5-flash"); c.Name == "" || c.Custom {
		t.Fatalf("gemini-3.5-flash candidate = %#v, want normal runtime candidate", c)
	}
	if c := candidateByName(got, "gemini-3.1-flash-lite"); c.Name == "" || c.Custom {
		t.Fatalf("gemini-3.1-flash-lite candidate = %#v, want normal runtime candidate", c)
	}
	if c := candidateByName(got, "gemini-3.1-pro-preview-customtools"); c.Name == "" || c.Custom {
		t.Fatalf("gemini-3.1-pro-preview-customtools candidate = %#v, want normal runtime candidate", c)
	}
	if c := candidateByName(got, "gemini-3.1-pro"); c.Name != "" {
		t.Fatalf("gemini-3.1-pro candidate = %#v, should not be exposed in picker", c)
	}
	if c := candidateByName(got, "gemini-3.1-pro-preview"); c.Name != "" {
		t.Fatalf("gemini-3.1-pro-preview candidate = %#v, should prefer customtools variant", c)
	}
	if c := candidateByName(got, "gemini-2.0-flash-exp"); c.Name != "" {
		t.Fatalf("gemini-2.0-flash-exp candidate = %#v, should not expose shutdown model", c)
	}
}

func TestModelCandidates_AzureDeploymentOnly(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel:    "corp-gpt55-deployment",
		CatalogModel:    "gpt-5.3-codex",
		MaxOutputTokens: 128000,
		ModelOverrides: map[string]config.ModelOverride{
			"session-deployment": {CatalogModel: "gpt-5.5-pro"},
		},
	})
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
	if got := cfg.ModelCatalogName("azure", "corp-gpt55-deployment"); got != "gpt-5.3-codex" {
		t.Fatalf("ModelCatalogName(default deployment) = %q, want gpt-5.3-codex", got)
	}
	if got := cfg.ModelCatalogName("azure", "session-deployment"); got != "gpt-5.5-pro" {
		t.Fatalf("ModelCatalogName(model override deployment) = %q, want gpt-5.5-pro", got)
	}
	for _, catalogOnly := range []string{"gpt-5.3-codex", "gpt-5.5-pro"} {
		if countCandidateName(got, catalogOnly) != 0 {
			t.Fatalf("azure catalog_model %q should not be displayed as a deployment candidate: %#v", catalogOnly, got)
		}
	}
	if c := candidateByName(got, "session-deployment"); !c.Current {
		t.Fatalf("session deployment candidate = %#v, want current", c)
	}
	if c := candidateByName(got, "corp-gpt55-deployment"); !c.Default {
		t.Fatalf("default deployment candidate = %#v, want default", c)
	}
}

func TestAzureCatalogModelCandidates_OpenAICatalogOnly(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.3-codex",
		ModelOverrides: map[string]config.ModelOverride{
			"session-deployment": {CatalogModel: "gpt-5.5-pro"},
		},
	})
	agent := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "session-deployment",
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	got := agent.AzureCatalogModelCandidates("session-deployment")
	names := modelCandidateNames(got)
	wantPrefix := []string{"gpt-5.5", "gpt-5.5-pro", "gpt-5.4"}
	if !reflect.DeepEqual(names[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("catalog model prefix = %v, want %v; all=%v", names[:len(wantPrefix)], wantPrefix, names)
	}
	if countCandidateName(got, "session-deployment") != 0 {
		t.Fatalf("deployment name should not be displayed as catalog_model candidate: %#v", got)
	}
	if c := candidateByName(got, "gpt-5.5-pro"); !c.Current || !c.Default {
		t.Fatalf("gpt-5.5-pro candidate = %#v, want current/default catalog model", c)
	}
	last := got[len(got)-1]
	if !last.Custom || last.Name != "Custom catalog model..." {
		t.Fatalf("last candidate = %#v, want custom catalog model row", last)
	}
}

func TestAzureCatalogModelCandidates_PreservesExplicitCatalogModelMatchingDeployment(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "gpt-4o",
		CatalogModel: "gpt-4o",
	})
	agent := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "gpt-4o",
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	got := agent.AzureCatalogModelCandidates("gpt-4o")
	if c := candidateByName(got, "gpt-4o"); !c.Current || !c.Default {
		t.Fatalf("gpt-4o candidate = %#v, want explicit matching catalog_model current/default", c)
	}
}

func TestAzureCatalogModelCandidates_PreselectsKnownOpenAIDeploymentWithoutExplicitCatalogModel(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	agent := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "gpt-4o",
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	got := agent.AzureCatalogModelCandidates("gpt-4o")
	if count := countCandidateName(got, "gpt-4o"); count != 1 {
		t.Fatalf("gpt-4o candidate count = %d, want 1; all=%#v", count, got)
	}
	if c := candidateByName(got, "gpt-4o"); !c.Current || !c.Default {
		t.Fatalf("gpt-4o candidate = %#v, want known deployment preselected as catalog_model", c)
	}
}

func TestAzureCatalogModelCandidates_DoesNotPreselectUnknownDeploymentWithoutExplicitCatalogModel(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	agent := &Agent{
		ProviderName:      "azure",
		ProviderConfigKey: "azure",
		CurrentModel:      "gpt-corp-deployment",
		Runtime:           NewAgentRuntimeWithConfig(cfg),
	}

	got := agent.AzureCatalogModelCandidates("gpt-corp-deployment")
	if count := countCandidateName(got, "gpt-corp-deployment"); count != 0 {
		t.Fatalf("unknown deployment candidate count = %d, want 0; all=%#v", count, got)
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

func TestConfigureAzureDeployment_PersistsDeploymentAndCatalogModel(t *testing.T) {
	withConfigCommandHooks(t)

	loadConfigForCommand = func() (*config.Config, error) {
		cfg := newProjectMapDisabledConfig()
		cfg.DefaultProvider = "openai"
		cfg.DefaultModel = "gpt-5.4"
		cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
			ModelOverrides: map[string]config.ModelOverride{
				"corp-codex-deployment": {
					CatalogModel:    "gpt-5.5-pro",
					MaxOutputTokens: 64000,
				},
			},
		})
		return cfg, nil
	}
	var saved *config.Config
	saveConfigForCommand = func(cfg *config.Config) error {
		saved = config.CloneConfig(cfg)
		return nil
	}

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(newProjectMapDisabledConfig(), &out)

	if err := agent.ConfigureAzureDeployment(" corp-codex-deployment ", " gpt-5.3-codex "); err != nil {
		t.Fatalf("ConfigureAzureDeployment() error = %v", err)
	}
	if saved == nil {
		t.Fatal("saveConfigForCommand was not called")
	}
	if saved.DefaultProvider != "azure" {
		t.Fatalf("DefaultProvider = %q, want azure", saved.DefaultProvider)
	}
	if saved.DefaultModel != "corp-codex-deployment" {
		t.Fatalf("DefaultModel = %q, want deployment", saved.DefaultModel)
	}
	if got := saved.GetSelectedModelForProvider("azure"); got != "corp-codex-deployment" {
		t.Fatalf("GetSelectedModelForProvider(azure) = %q, want deployment", got)
	}
	if got := saved.ModelCatalogName("azure", "corp-codex-deployment"); got != "gpt-5.3-codex" {
		t.Fatalf("ModelCatalogName(azure, deployment) = %q, want gpt-5.3-codex", got)
	}
	override, ok := saved.ModelOverrideForProvider("azure", "corp-codex-deployment")
	if !ok {
		t.Fatal("ModelOverrideForProvider(azure, deployment) ok = false, want existing override preserved")
	}
	if override.CatalogModel != "gpt-5.3-codex" || override.MaxOutputTokens != 64000 {
		t.Fatalf("override = %#v, want catalog_model updated and max_output_tokens preserved", override)
	}
	if got := agent.cfg().ModelCatalogName("azure", "corp-codex-deployment"); got != "gpt-5.3-codex" {
		t.Fatalf("runtime ModelCatalogName(azure, deployment) = %q, want gpt-5.3-codex", got)
	}
}

func TestConfigureAndSwitchAzureDeployment_SwitchFailureDoesNotPersistConfig(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")
	t.Setenv("AZURE_OPENAI_BASE_URL", "")
	withConfigCommandHooks(t)

	loadConfigForCommand = func() (*config.Config, error) {
		cfg := newProjectMapDisabledConfig()
		cfg.DefaultProvider = "openai"
		cfg.DefaultModel = "gpt-5.4"
		return cfg, nil
	}
	var saveCalled bool
	saveConfigForCommand = func(cfg *config.Config) error {
		saveCalled = true
		return nil
	}

	var out bytes.Buffer
	runtimeCfg := newProjectMapDisabledConfig()
	runtimeCfg.DefaultProvider = "openai"
	runtimeCfg.DefaultModel = "gpt-5.4"
	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-5.4",
		CurrentProvider:   &mockProvider{name: "openai"},
		Runtime: &AgentRuntime{
			Config: runtimeCfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-5.4"),
		},
	}

	if _, err := agent.ConfigureAndSwitchAzureDeployment("corp-deployment", "gpt-5.4"); err == nil {
		t.Fatal("ConfigureAndSwitchAzureDeployment() error = nil, want Azure credential error")
	}
	if saveCalled {
		t.Fatal("saveConfigForCommand should not be called when provider switch fails")
	}
	if agent.ProviderName != "openai" || agent.ProviderConfigKey != "openai" || agent.CurrentModel != "gpt-5.4" {
		t.Fatalf("agent state = (%q, %q, %q), want unchanged openai/gpt-5.4", agent.ProviderName, agent.ProviderConfigKey, agent.CurrentModel)
	}
	if agent.cfg().DefaultProvider != "openai" {
		t.Fatalf("runtime DefaultProvider = %q, want unchanged openai", agent.cfg().DefaultProvider)
	}
}

func TestConfigureAndSwitchAzureDeployment_PersistsAfterSuccessfulSwitch(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	withConfigCommandHooks(t)

	loadConfigForCommand = func() (*config.Config, error) {
		cfg := newProjectMapDisabledConfig()
		cfg.DefaultProvider = "openai"
		cfg.DefaultModel = "gpt-5.4"
		return cfg, nil
	}
	var saved *config.Config
	saveConfigForCommand = func(cfg *config.Config) error {
		saved = config.CloneConfig(cfg)
		return nil
	}

	var out bytes.Buffer
	runtimeCfg := newProjectMapDisabledConfig()
	runtimeCfg.DefaultProvider = "openai"
	runtimeCfg.DefaultModel = "gpt-5.4"
	agent := &Agent{
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		CurrentModel:      "gpt-5.4",
		CurrentProvider:   &mockProvider{name: "openai"},
		Stats:             NewSessionStats("openai", "gpt-5.4"),
		Runtime: &AgentRuntime{
			Config: runtimeCfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-5.4"),
		},
	}

	if _, err := agent.ConfigureAndSwitchAzureDeployment("corp-deployment", "gpt-5.4"); err != nil {
		t.Fatalf("ConfigureAndSwitchAzureDeployment() error = %v", err)
	}
	if saved == nil {
		t.Fatal("saveConfigForCommand was not called")
	}
	if saved.DefaultProvider != "azure" || saved.DefaultModel != "corp-deployment" {
		t.Fatalf("saved defaults = (%q, %q), want azure/corp-deployment", saved.DefaultProvider, saved.DefaultModel)
	}
	if got := saved.ModelCatalogName("azure", "corp-deployment"); got != "gpt-5.4" {
		t.Fatalf("saved ModelCatalogName(azure, deployment) = %q, want gpt-5.4", got)
	}
	if agent.ProviderName != "azure" || agent.ProviderConfigKey != "azure" || agent.CurrentModel != "corp-deployment" {
		t.Fatalf("agent state = (%q, %q, %q), want azure/corp-deployment", agent.ProviderName, agent.ProviderConfigKey, agent.CurrentModel)
	}
	if got := agent.cfg().ModelCatalogName("azure", "corp-deployment"); got != "gpt-5.4" {
		t.Fatalf("runtime ModelCatalogName(azure, deployment) = %q, want gpt-5.4", got)
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
