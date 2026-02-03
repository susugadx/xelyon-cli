package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
)

// SyncWithGlobalConfig はグローバル設定（config.GetGlobalConfig）と Agent の状態を同期する。
//
// /config などで設定を変更した場合に、フッター表示・次回API呼び出しに即反映させるために使用する。
// プロバイダー変更が必要な場合は SwitchProvider を呼び出す。
func (a *Agent) SyncWithGlobalConfig() {
	if a == nil {
		return
	}

	cfg := config.GetGlobalConfig()

	// Provider: 変更があれば切り替え（モデルもプロバイダー別デフォルトに更新される）
	if cfg.DefaultProvider != "" && cfg.DefaultProvider != a.ProviderName {
		if err := a.SwitchProvider(cfg.DefaultProvider); err != nil {
			// /config 直後にエラーを出して操作を止めるより、既存プロバイダーで継続
			yellow.Printf("Warning: Failed to switch provider: %v\n", err)
		}
	}

	// Model: プロバイダーの現在モデルとして設定に追従
	// ここでは provider_models の解決も含める
	resolvedModel := cfg.GetModelForProvider(a.ProviderName)
	if resolvedModel == "" {
		resolvedModel = cfg.DefaultModel
	}
	if resolvedModel != "" {
		a.CurrentModel = resolvedModel
	}
}
