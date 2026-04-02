package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
)

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
// CLI /config default_model と /model コマンドから使用する。
func (a *Agent) SyncDefaultModelToProvider(cfg *config.Config) {
	if a == nil || a.ProviderName == "" {
		return
	}
	if pm, ok := cfg.ProviderModels[a.ProviderName]; ok {
		pm.DefaultModel = cfg.DefaultModel
		cfg.ProviderModels[a.ProviderName] = pm
	}
}
