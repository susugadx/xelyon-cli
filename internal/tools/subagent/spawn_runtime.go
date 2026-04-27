package subagent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultSubAgentModel = "gpt-5.4-mini"

// spawnConfig は sub-agent 実行直前の確定設定です。
type spawnConfig struct {
	taskType               string
	model                  string
	modelProviderConfigKey string
	cfg                    *config.Config
}

// SpawnRuntimeContext は親 agent の実行時選択を sub-agent 起動へ渡します。
type SpawnRuntimeContext struct {
	CurrentModel string
}

type subAgentModelSelection struct {
	model             string
	providerConfigKey string
}

func normalizeTaskType(taskType string) string {
	if !ValidTaskType(taskType) {
		return TaskTypeExplore
	}
	return taskType
}

func prepareSpawnConfigWithRuntimeContext(cfg *config.Config, provider api.Provider, runtimeCtx SpawnRuntimeContext, taskType, model, reasoningEffort string) (*spawnConfig, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is required")
	}

	normalizedTaskType := normalizeTaskType(taskType)
	mainProvider := config.CanonicalProviderName(provider.Name())
	mainProviderConfigKey := currentProviderConfigKey(provider)
	subCfg, selection, err := cloneConfigForSubWithRuntimeContext(cfg, mainProvider, mainProviderConfigKey, runtimeCtx, normalizedTaskType, model, reasoningEffort)
	if err != nil {
		return nil, err
	}
	if !subCfg.SubAgent.Enabled {
		return nil, fmt.Errorf("sub-agent is disabled")
	}

	return &spawnConfig{
		taskType:               normalizedTaskType,
		model:                  selection.model,
		modelProviderConfigKey: selection.providerConfigKey,
		cfg:                    subCfg,
	}, nil
}

func applySubAgentPrompt(cfg *config.Config, taskType string, provider api.Provider, model string) {
	if cfg == nil || provider == nil {
		return
	}
	cfg.SubAgentPrompt = PromptForTaskTypeWithConfig(taskType, provider.Name(), model, cfg)
}

func cloneConfigForSub(cfg *config.Config, mainProvider, taskType, model, reasoningEffort string) (*config.Config, string, error) {
	cloned, selection, err := cloneConfigForSubWithRuntimeContext(cfg, mainProvider, "", SpawnRuntimeContext{}, taskType, model, reasoningEffort)
	if err != nil {
		return nil, "", err
	}
	return cloned, selection.model, nil
}

func cloneConfigForSubWithRuntimeContext(cfg *config.Config, mainProvider, mainProviderConfigKey string, runtimeCtx SpawnRuntimeContext, taskType, model, reasoningEffort string) (*config.Config, subAgentModelSelection, error) {
	cloned := config.CloneConfig(cfg)
	if cloned == nil {
		cloned = config.DefaultConfig()
	}
	taskType = normalizeTaskType(taskType)

	selection := subAgentModelSelection{model: normalizeSubAgentModel(model)}
	if selection.model != "" {
		selection.providerConfigKey = providerConfigKeyForExplicitSubAgentModelSelection(mainProvider, mainProviderConfigKey)
	}
	if selection.model == "" {
		defaultModel := normalizeSubAgentModel(cloned.SubAgent.DefaultModel)
		if defaultModel != "" && subAgentDefaultModelAppliesToProvider(mainProvider, defaultModel) {
			selection = subAgentModelSelection{
				model:             defaultModel,
				providerConfigKey: providerConfigKeyForSubAgentDefaultSelection(mainProvider, mainProviderConfigKey),
			}
		}
	}
	if selection.model == "" {
		selection = inferSubAgentModelSelection(cloned, mainProvider, mainProviderConfigKey, runtimeCtx.CurrentModel)
	}
	if selection.model == "" {
		fallbackModel, err := fallbackSubAgentModel(mainProvider)
		if err != nil {
			return nil, subAgentModelSelection{}, err
		}
		selection.model = fallbackModel
	}

	effort := strings.TrimSpace(reasoningEffort)
	if effort == "" {
		effort = DefaultEffortForTaskType(taskType)
	}
	if effort == "" {
		effort = strings.TrimSpace(cloned.SubAgent.DefaultEffort)
	}

	if err := applyReasoningEffort(cloned, effort); err != nil {
		return nil, subAgentModelSelection{}, err
	}

	return cloned, selection, nil
}

func normalizeSubAgentModel(model string) string {
	model = strings.TrimSpace(model)
	switch strings.ToLower(model) {
	case "", "sub_agent.default_model":
		return ""
	default:
		return model
	}
}

func subAgentDefaultModelAppliesToProvider(provider, model string) bool {
	if config.CanonicalProviderName(provider) == "azure" && model == defaultSubAgentModel {
		return false
	}
	return true
}

func inferSubAgentModel(cfg *config.Config, provider string) string {
	return inferSubAgentModelSelection(cfg, provider, "", "").model
}

func inferSubAgentModelSelection(cfg *config.Config, provider, providerConfigKey, currentModel string) subAgentModelSelection {
	if model := config.ProviderDefaultSubAgentModel(provider); model != "" {
		return subAgentModelSelection{model: model}
	}
	if cfg == nil {
		return subAgentModelSelection{}
	}

	if config.CanonicalProviderName(provider) == "azure" {
		owner := providerConfigKeyForSubAgentSelection(provider, providerConfigKey)
		if model := normalizeSubAgentModel(currentModel); model != "" {
			return subAgentModelSelection{model: model, providerConfigKey: owner}
		}
		if model := cfg.GetExplicitProviderDefaultModel(provider); model != "" {
			return subAgentModelSelection{model: model, providerConfigKey: owner}
		}
		return subAgentModelSelection{}
	}

	return subAgentModelSelection{model: cfg.GetSelectedModelForProvider(provider)}
}

func fallbackSubAgentModel(provider string) (string, error) {
	if config.CanonicalProviderName(provider) == "azure" {
		return "", fmt.Errorf("azure sub-agent model requires a current deployment, sub_agent.default_model, or provider_models.azure.default_model")
	}
	return defaultSubAgentModel, nil
}

func providerConfigKeyForSubAgentSelection(provider, providerConfigKey string) string {
	if key := config.ActiveProviderConfigKey(providerConfigKey); key != "" {
		return key
	}
	return config.ActiveProviderConfigKey(provider)
}

func providerConfigKeyForSubAgentDefaultSelection(provider, providerConfigKey string) string {
	if config.CanonicalProviderName(provider) != "azure" {
		return ""
	}
	return providerConfigKeyForSubAgentSelection(provider, providerConfigKey)
}

func providerConfigKeyForExplicitSubAgentModelSelection(provider, providerConfigKey string) string {
	if config.CanonicalProviderName(provider) != "azure" {
		return ""
	}
	return providerConfigKeyForSubAgentSelection(provider, providerConfigKey)
}

func applyReasoningEffort(cfg *config.Config, effort string) error {
	switch normalizeReasoningEffort(effort) {
	case "", "off":
		cfg.Thinking.Enabled = false
		return nil
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = normalizeReasoningEffort(effort)
		return nil
	default:
		return fmt.Errorf("invalid reasoning_effort: %s", effort)
	}
}

func normalizeReasoningEffort(effort string) string {
	return strings.ToLower(strings.TrimSpace(effort))
}

type providerConfigKeyProvider interface {
	ProviderConfigKey() string
}

func currentProviderConfigKey(current api.Provider) string {
	if current == nil {
		return ""
	}
	if keyed, ok := current.(providerConfigKeyProvider); ok {
		if key := config.ActiveProviderConfigKey(keyed.ProviderConfigKey()); key != "" {
			return key
		}
	}
	return config.ActiveProviderConfigKey(current.Name())
}

func resolveSubProvider(current api.Provider, cfg *config.Config, model, modelProviderConfigKey string, factory ProviderFactory) (api.Provider, error) {
	if current == nil {
		return nil, fmt.Errorf("provider is required")
	}

	currentName := config.CanonicalProviderName(current.Name())
	currentConfigKey := currentProviderConfigKey(current)
	target := config.CanonicalProviderName(modelProviderConfigKey)
	factoryProviderName := config.ActiveProviderConfigKey(modelProviderConfigKey)
	if target == "" {
		target = currentName
	}
	if factoryProviderName == "" && cfg != nil {
		target = cfg.ResolveProviderForModel(currentName, model)
	}
	if target == "" {
		target = currentName
	}
	if factoryProviderName == "" {
		factoryProviderName = target
	}
	if currentConfigKey != "" && modelProviderConfigKey == "" && config.SameProviderRuntimeIdentity(currentConfigKey, target) {
		factoryProviderName = currentConfigKey
	}
	if factory == nil {
		factory = api.NewProvider
	}
	provider, err := factory(factoryProviderName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", factoryProviderName, err)
	}
	resetSubProviderState(provider)
	return provider, nil
}

func resetSubProviderState(provider api.Provider) {
	if provider == nil {
		return
	}
	if cacheClearable, ok := provider.(api.CacheClearable); ok {
		cacheClearable.ClearCache()
		return
	}
	if responseIDSetter, ok := provider.(interface{ SetResponseID(string) }); ok {
		responseIDSetter.SetResponseID("")
	}
}
