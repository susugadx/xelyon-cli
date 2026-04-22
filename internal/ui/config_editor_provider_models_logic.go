package ui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type providerAddTargetStatus int

const (
	providerAddTargetEmpty providerAddTargetStatus = iota
	providerAddTargetDuplicate
	providerAddTargetReady
)

func resolveProviderAddTarget(rawProvider string, existing map[string]config.ProviderModelConfig) (string, providerAddTargetStatus) {
	name := config.NormalizeProviderName(rawProvider)
	if name == "" {
		return "", providerAddTargetEmpty
	}
	if _, ok := existing[name]; ok {
		return name, providerAddTargetDuplicate
	}
	return name, providerAddTargetReady
}

func withAddedProviderModel(existing map[string]config.ProviderModelConfig, provider, model string) map[string]config.ProviderModelConfig {
	updated := cloneProviderModelConfigs(existing)
	updated[provider] = config.ProviderModelConfig{DefaultModel: strings.TrimSpace(model)}
	return updated
}

func cloneProviderModelConfigs(existing map[string]config.ProviderModelConfig) map[string]config.ProviderModelConfig {
	if len(existing) == 0 {
		return make(map[string]config.ProviderModelConfig)
	}

	copied := make(map[string]config.ProviderModelConfig, len(existing))
	for key, value := range existing {
		copied[key] = value
	}
	return copied
}

func selectProviderByInput(input string, providers []string) (string, bool) {
	idx, ok := parseConfigEditorIndex(input, len(providers))
	if !ok {
		return "", false
	}
	return providers[idx], true
}
