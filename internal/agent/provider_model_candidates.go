package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
)

type ProviderCandidate = providerpicker.ProviderCandidate
type ProviderCredentialStatus = providerpicker.ProviderCredentialStatus
type ModelCandidate = providerpicker.ModelCandidate

const (
	ProviderCredentialConfigured    = providerpicker.ProviderCredentialConfigured
	ProviderCredentialLoggedIn      = providerpicker.ProviderCredentialLoggedIn
	ProviderCredentialMissingKey    = providerpicker.ProviderCredentialMissingKey
	ProviderCredentialLoginRequired = providerpicker.ProviderCredentialLoginRequired
	ProviderCredentialLocal         = providerpicker.ProviderCredentialLocal
	ProviderCredentialAWSAuth       = providerpicker.ProviderCredentialAWSAuth
)

var listOllamaModelsForCandidates = func(agent *Agent, provider string) ([]string, error) {
	if agent != nil && agent.CurrentProvider != nil && config.SameProviderRuntimeIdentity(agent.ProviderName, provider) {
		if lister, ok := agent.CurrentProvider.(api.ModelLister); ok {
			return lister.ListModels()
		}
	}

	instance, err := api.NewProvider(provider)
	if err != nil {
		return nil, err
	}
	lister, ok := instance.(api.ModelLister)
	if !ok {
		return nil, nil
	}
	return lister.ListModels()
}

// ProviderCandidates は TUI picker 用の provider 候補を表示順で返す。
func (a *Agent) ProviderCandidates() []ProviderCandidate {
	state := a.CurrentProviderModelState()
	keys := llmcatalog.DisplayProviderKeys()
	candidates := make([]ProviderCandidate, 0, len(keys))
	for _, key := range keys {
		desc, ok := llmcatalog.ProviderDescriptorFor(key)
		if !ok {
			continue
		}
		label := desc.DisplayName
		if label == "" {
			label = desc.Key
		}
		candidates = append(candidates, ProviderCandidate{
			Key:              desc.Key,
			Label:            label,
			Current:          isCurrentProviderCandidate(state, desc.Key),
			CredentialStatus: providerCredentialStatus(desc.Key),
		})
	}
	if extra, ok := currentProviderConfigKeyCandidate(state, keys); ok {
		candidates = append(candidates, extra)
	}
	return candidates
}

func isCurrentProviderCandidate(state ProviderModelState, provider string) bool {
	if state.ProviderConfigKey != "" {
		return config.ActiveProviderConfigKey(provider) == state.ProviderConfigKey
	}
	return config.SameProviderRuntimeIdentity(state.CurrentProvider, provider)
}

func currentProviderConfigKeyCandidate(state ProviderModelState, displayKeys []string) (ProviderCandidate, bool) {
	key := strings.TrimSpace(state.ProviderConfigKey)
	if key == "" {
		return ProviderCandidate{}, false
	}
	for _, displayKey := range displayKeys {
		if config.ActiveProviderConfigKey(displayKey) == key {
			return ProviderCandidate{}, false
		}
	}
	if !llmcatalog.IsKnownProvider(key) && !api.IsRegisteredProvider(key) {
		return ProviderCandidate{}, false
	}
	return ProviderCandidate{
		Key:              key,
		Label:            key,
		Current:          true,
		CredentialStatus: providerCredentialStatus(key),
	}, true
}

func providerCredentialStatus(provider string) ProviderCredentialStatus {
	switch config.CanonicalProviderName(provider) {
	case "openai_subscription":
		status := openaisubscription.ReadSubscriptionAuthStatus(openaisubscription.DefaultSubscriptionAuthConfig())
		if status.State == openaisubscription.SubscriptionAuthStateLoggedIn {
			return ProviderCredentialLoggedIn
		}
		return ProviderCredentialLoginRequired
	case "ollama":
		return ProviderCredentialLocal
	case "bedrock":
		return ProviderCredentialAWSAuth
	default:
		if config.ProviderHasAvailableCredential(provider) {
			return ProviderCredentialConfigured
		}
		return ProviderCredentialMissingKey
	}
}

// ModelCandidates は provider picker 用の model/deployment 候補を返す。
func (a *Agent) ModelCandidates(provider string) []ModelCandidate {
	runtimeProvider := config.CanonicalProviderName(provider)
	if runtimeProvider == "azure" {
		return a.azureDeploymentCandidates(provider)
	}

	models := llmcatalog.RecommendedModelNamesForProvider(runtimeProvider)
	if runtimeProvider == "ollama" {
		if liveModels, err := listOllamaModelsForCandidates(a, runtimeProvider); err == nil && len(liveModels) > 0 {
			models = liveModels
		}
	}
	return a.modelCandidatesFromNames(provider, models, "Custom model...")
}

func (a *Agent) azureDeploymentCandidates(provider string) []ModelCandidate {
	return a.modelCandidatesFromNames(provider, nil, "Custom deployment...")
}

// AzureCatalogModelCandidates は Azure deployment に紐づける OpenAI catalog_model 候補を返す。
func (a *Agent) AzureCatalogModelCandidates(deployment string) []ModelCandidate {
	catalogModel := a.azureCatalogModelPreselection(deployment)
	models := llmcatalog.KnownModelNamesForProvider("openai")
	builder := modelCandidateBuilder{
		defaultModel: catalogModel,
		currentModel: catalogModel,
	}
	for _, name := range models {
		builder.add(name)
	}
	builder.add(catalogModel)
	builder.addCustom("Custom catalog model...")
	return builder.candidates
}

func (a *Agent) azureCatalogModelPreselection(deployment string) string {
	deployment = strings.TrimSpace(deployment)
	if deployment == "" {
		deployment = selectedModelForProvider(a.cfg(), "azure")
	}
	if deployment == "" {
		return ""
	}
	if catalogModel := azureExplicitCatalogModelForDeployment(a.cfg(), deployment); catalogModel != "" {
		return catalogModel
	}
	if knownOpenAICatalogModel(deployment) {
		return deployment
	}
	return ""
}

func azureExplicitCatalogModelForDeployment(cfg *config.Config, deployment string) string {
	if cfg == nil {
		return ""
	}
	if override, ok := cfg.ModelOverrideForProvider("azure", deployment); ok {
		if catalogModel := openAICatalogModelName(override.CatalogModel); catalogModel != "" {
			return catalogModel
		}
	}
	pm, ok := cfg.GetExplicitProviderModelConfig("azure")
	if !ok || strings.TrimSpace(pm.DefaultModel) != deployment {
		return ""
	}
	return openAICatalogModelName(pm.CatalogModel)
}

func openAICatalogModelName(model string) string {
	model = strings.TrimSpace(model)
	if model == "" || llmcatalog.InferProviderFromModel(model) != "openai" {
		return ""
	}
	return model
}

func knownOpenAICatalogModel(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	return llmcatalog.IsExactKnownModelNameForProvider("openai", model)
}

func (a *Agent) modelCandidatesFromNames(provider string, names []string, customLabel string) []ModelCandidate {
	state := a.CurrentProviderModelState()
	providerConfigKey := config.ActiveProviderConfigKey(provider)
	if providerConfigKey == "" {
		providerConfigKey = config.CanonicalProviderName(provider)
	}
	defaultModel := selectedModelForProvider(a.cfg(), providerConfigKey)
	currentModel := ""
	if isCurrentProviderCandidate(state, provider) {
		currentModel = strings.TrimSpace(state.CurrentModel)
	}

	builder := modelCandidateBuilder{
		defaultModel: defaultModel,
		currentModel: currentModel,
	}
	for _, name := range names {
		builder.add(name)
	}
	builder.add(currentModel)
	builder.add(defaultModel)
	builder.addCustom(customLabel)
	return builder.candidates
}

func selectedModelForProvider(cfg *config.Config, providerConfigKey string) string {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return strings.TrimSpace(cfg.GetSelectedModelForProvider(providerConfigKey))
}

type modelCandidateBuilder struct {
	defaultModel string
	currentModel string
	seen         map[string]int
	candidates   []ModelCandidate
}

func (b *modelCandidateBuilder) add(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if b.seen == nil {
		b.seen = map[string]int{}
	}
	if idx, ok := b.seen[name]; ok {
		b.candidates[idx].Default = b.candidates[idx].Default || name == b.defaultModel
		b.candidates[idx].Current = b.candidates[idx].Current || name == b.currentModel
		return
	}
	b.seen[name] = len(b.candidates)
	b.candidates = append(b.candidates, ModelCandidate{
		Name:    name,
		Default: name == b.defaultModel,
		Current: name == b.currentModel,
	})
}

func (b *modelCandidateBuilder) addCustom(label string) {
	label = strings.TrimSpace(label)
	if label == "" {
		return
	}
	b.candidates = append(b.candidates, ModelCandidate{Name: label, Custom: true})
}
