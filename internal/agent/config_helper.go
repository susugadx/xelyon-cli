package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type providerConfigKeyAware interface {
	ProviderConfigKey() string
}

func providerConfigKeyFromProvider(provider api.Provider) string {
	if provider == nil {
		return ""
	}
	if aware, ok := provider.(providerConfigKeyAware); ok {
		if key := config.NormalizeProviderName(aware.ProviderConfigKey()); key != "" {
			return key
		}
	}
	return config.NormalizeProviderName(provider.Name())
}

func (a *Agent) sessionProviderConfigKey(cfg *config.Config) string {
	if a == nil {
		return ""
	}

	if key := config.NormalizeProviderName(a.ProviderConfigKey); key != "" {
		return key
	}

	runtimeProvider := config.NormalizeProviderName(a.ProviderName)
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

	runtimeProvider := config.NormalizeProviderName(a.ProviderName)
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
	if a == nil || a.ProviderName == "" {
		return
	}
	cfg.SyncProviderDefaultModel(a.sessionProviderConfigKey(cfg), cfg.DefaultModel)
}
