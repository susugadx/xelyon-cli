package subagent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type subAgentModelSelection struct {
	model             string
	providerConfigKey string
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

	if err := applySubAgentReasoningEffort(cloned, taskType, reasoningEffort); err != nil {
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
		return subAgentModelSelection{
			model:             model,
			providerConfigKey: providerConfigKeyForSubAgentSelection(provider, providerConfigKey),
		}
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
