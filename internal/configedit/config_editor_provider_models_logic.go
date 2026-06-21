package configedit

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// ProviderAddTargetStatus は provider_models 追加入力の判定結果を表す。
type ProviderAddTargetStatus int

const (
	// ProviderAddTargetEmpty は provider 名入力が空だったことを表す。
	ProviderAddTargetEmpty ProviderAddTargetStatus = iota
	// ProviderAddTargetDuplicate は provider が既に存在することを表す。
	ProviderAddTargetDuplicate
	// ProviderAddTargetReady は provider を追加できることを表す。
	ProviderAddTargetReady
)

const (
	providerAddTargetEmpty     = ProviderAddTargetEmpty
	providerAddTargetDuplicate = ProviderAddTargetDuplicate
	providerAddTargetReady     = ProviderAddTargetReady
)

func resolveProviderAddTarget(rawProvider string, existing map[string]config.ProviderModelConfig) (string, ProviderAddTargetStatus) {
	name := config.NormalizeProviderName(rawProvider)
	if name == "" {
		return "", providerAddTargetEmpty
	}
	if _, ok := existing[name]; ok {
		return name, providerAddTargetDuplicate
	}
	return name, providerAddTargetReady
}

// ResolveProviderAddTarget は provider_models に追加する provider 名と追加可否を判定する。
func ResolveProviderAddTarget(rawProvider string, existing map[string]config.ProviderModelConfig) (string, ProviderAddTargetStatus) {
	return resolveProviderAddTarget(rawProvider, existing)
}

func withAddedProviderModel(existing map[string]config.ProviderModelConfig, provider, model string) map[string]config.ProviderModelConfig {
	updated := cloneProviderModelConfigs(existing)
	updated[provider] = config.ProviderModelConfig{DefaultModel: strings.TrimSpace(model)}
	return updated
}

// WithAddedProviderModel は既存 map を破壊せず provider model 設定を追加した map を返す。
func WithAddedProviderModel(existing map[string]config.ProviderModelConfig, provider, model string) map[string]config.ProviderModelConfig {
	return withAddedProviderModel(existing, provider, model)
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

// SelectProviderByInput は番号入力から provider key を選択する。
func SelectProviderByInput(input string, providers []string) (string, bool) {
	return selectProviderByInput(input, providers)
}
