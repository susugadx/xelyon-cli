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
	taskType string
	model    string
	cfg      *config.Config
}

func normalizeTaskType(taskType string) string {
	if !ValidTaskType(taskType) {
		return TaskTypeExplore
	}
	return taskType
}

func prepareSpawnConfig(cfg *config.Config, provider api.Provider, taskType, model, reasoningEffort string) (*spawnConfig, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is required")
	}

	normalizedTaskType := normalizeTaskType(taskType)
	mainProvider := config.CanonicalProviderName(provider.Name())
	subCfg, resolvedModel, err := cloneConfigForSub(cfg, mainProvider, normalizedTaskType, model, reasoningEffort)
	if err != nil {
		return nil, err
	}
	if !subCfg.SubAgent.Enabled {
		return nil, fmt.Errorf("sub-agent is disabled")
	}

	return &spawnConfig{
		taskType: normalizedTaskType,
		model:    resolvedModel,
		cfg:      subCfg,
	}, nil
}

func applySubAgentPrompt(cfg *config.Config, taskType string, provider api.Provider, model string) {
	if cfg == nil || provider == nil {
		return
	}
	cfg.SubAgentPrompt = PromptForTaskType(taskType, provider.Name(), model)
}

func cloneConfigForSub(cfg *config.Config, mainProvider, taskType, model, reasoningEffort string) (*config.Config, string, error) {
	cloned := config.CloneConfig(cfg)
	if cloned == nil {
		cloned = config.DefaultConfig()
	}
	taskType = normalizeTaskType(taskType)

	resolvedModel := normalizeSubAgentModel(model)
	if resolvedModel == "" {
		resolvedModel = normalizeSubAgentModel(cloned.SubAgent.DefaultModel)
	}
	if resolvedModel == "" {
		resolvedModel = inferSubAgentModel(mainProvider)
	}
	if resolvedModel == "" {
		resolvedModel = defaultSubAgentModel
	}

	effort := strings.TrimSpace(reasoningEffort)
	if effort == "" {
		effort = DefaultEffortForTaskType(taskType)
	}
	if effort == "" {
		effort = strings.TrimSpace(cloned.SubAgent.DefaultEffort)
	}

	if err := applyReasoningEffort(cloned, effort); err != nil {
		return nil, "", err
	}

	return cloned, resolvedModel, nil
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

func inferSubAgentModel(provider string) string {
	switch config.CanonicalProviderName(provider) {
	case "openai":
		return "gpt-5.4-mini"
	case "claude":
		return "claude-haiku-4-5-20251001"
	case "gemini":
		return "gemini-3.1-flash-lite-preview"
	case "deepseek":
		return "deepseek-chat"
	case "groq":
		return "llama-3.3-70b-versatile"
	case "openrouter":
		return "openai/gpt-5.4-mini"
	default:
		return ""
	}
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

func resolveSubProvider(current api.Provider, cfg *config.Config, model string, factory ProviderFactory) (api.Provider, error) {
	if current == nil {
		return nil, fmt.Errorf("provider is required")
	}

	currentName := config.CanonicalProviderName(current.Name())
	currentConfigKey := currentProviderConfigKey(current)
	target := currentName
	if cfg != nil {
		target = cfg.ResolveProviderForModel(currentName, model)
	}
	if target == "" {
		target = currentName
	}
	factoryProviderName := target
	if currentConfigKey != "" && config.SameProviderRuntimeIdentity(currentConfigKey, target) {
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
