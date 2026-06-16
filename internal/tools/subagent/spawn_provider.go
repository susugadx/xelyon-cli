package subagent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

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
