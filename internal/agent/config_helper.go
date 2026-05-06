package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type providerConfigKeyAware interface {
	ProviderConfigKey() string
}

type providerConfigKeyMutable interface {
	SetProviderConfigKey(key string)
}

type providerRuntimeNameAware interface {
	RuntimeProviderName() string
}

func providerRuntimeNameFromProvider(provider api.Provider) string {
	if provider == nil {
		return ""
	}
	if aware, ok := provider.(providerRuntimeNameAware); ok {
		if name := config.CanonicalProviderName(aware.RuntimeProviderName()); name != "" {
			return name
		}
	}
	return config.CanonicalProviderName(provider.Name())
}

func providerConfigKeyFromProvider(provider api.Provider) string {
	if provider == nil {
		return ""
	}
	if aware, ok := provider.(providerConfigKeyAware); ok {
		if key := config.ActiveProviderConfigKey(aware.ProviderConfigKey()); key != "" {
			return key
		}
	}
	return config.ActiveProviderConfigKey(provider.Name())
}

func syncProviderConfigKeyToProvider(provider api.Provider, key string) {
	if provider == nil {
		return
	}
	if mutable, ok := provider.(providerConfigKeyMutable); ok {
		mutable.SetProviderConfigKey(key)
	}
}

func (a *Agent) sessionProviderConfigKey(cfg *config.Config) string {
	if a == nil {
		return ""
	}

	if key := config.ActiveProviderConfigKey(a.ProviderConfigKey); key != "" {
		return key
	}

	runtimeProvider := config.ActiveProviderConfigKey(a.ProviderName)
	if cfg != nil {
		if owner := cfg.RuntimeProviderConfigKey(runtimeProvider, a.CurrentModel); owner != "" {
			return owner
		}
		if preferred := cfg.PreferredProviderConfigKey(runtimeProvider); preferred != "" {
			return preferred
		}
	}

	return runtimeProvider
}

func (a *Agent) activeModelProviderConfigKey(cfg *config.Config) string {
	if a == nil {
		return ""
	}

	runtimeProvider := config.ActiveProviderConfigKey(a.ProviderName)
	if cfg != nil {
		if owner := cfg.RuntimeProviderConfigKey(runtimeProvider, a.CurrentModel); owner != "" {
			return owner
		}
	}

	return a.sessionProviderConfigKey(cfg)
}

func (a *Agent) GetProviderConfigKey() string {
	return a.sessionProviderConfigKey(a.cfg())
}

// SaveAndSyncConfig は設定をファイルに保存し、runtime に反映する。
//
// provider_models の個別 override は編集した値をそのまま保存する。
// default_model → provider override の同期は呼び出し側の責務（SyncDefaultModelToProvider 参照）。
func (a *Agent) SaveAndSyncConfig(cfg *config.Config) error {
	if err := config.SaveConfig(cfg); err != nil {
		return err
	}

	if a != nil {
		a.setRuntimeConfig(cfg)
		a.SyncWithRuntimeConfig()
	}

	return nil
}

// SyncDefaultModelToProvider は default_model を明示変更した場合に、
// 現在プロバイダーの provider_models[provider].default_model を同期する。
// 必要なら保存対象の provider_models entry を新規作成する。
// CLI /config default_model と /model コマンドから使用する。
func (a *Agent) SyncDefaultModelToProvider(cfg *config.Config) {
	if a == nil || a.ProviderName == "" || cfg == nil {
		return
	}
	providerKey := a.sessionProviderConfigKey(cfg)
	previousDefaultModel := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel(providerKey))
	nextDefaultModel := strings.TrimSpace(cfg.DefaultModel)
	cfg.SyncProviderDefaultModel(providerKey, nextDefaultModel)
	clearAzureCatalogModelAfterDeploymentChange(cfg, providerKey, previousDefaultModel, nextDefaultModel)
}

func clearAzureCatalogModelAfterDeploymentChange(cfg *config.Config, providerKey, previousDefaultModel, nextDefaultModel string) {
	if cfg == nil || nextDefaultModel == "" || !config.SameProviderRuntimeIdentity(providerKey, "azure") {
		return
	}
	if previousDefaultModel == nextDefaultModel {
		return
	}
	cfg.ClearProviderCatalogModel(providerKey)
}
