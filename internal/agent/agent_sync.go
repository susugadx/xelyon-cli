package agent

import (
	"io"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func (a *Agent) currentProviderConfigKey() string {
	if a == nil {
		return ""
	}

	if key := config.ActiveProviderConfigKey(a.ProviderConfigKey); key != "" {
		return key
	}
	if key := providerConfigKeyFromProvider(a.CurrentProvider); key != "" {
		return key
	}
	return config.ActiveProviderConfigKey(a.ProviderName)
}

// SyncWithRuntimeConfig は runtime に保持した設定と Agent の状態を同期する。
//
// /config などで設定を変更した場合に、フッター表示・次回API呼び出しに即反映させるために使用する。
// runtime provider 自体が変わる場合のみ SwitchProvider を呼び出し、
// 同一 runtime provider 内の alias owner 変更は config key の同期だけで反映する。
func (a *Agent) SyncWithRuntimeConfig() {
	if a == nil {
		return
	}

	cfg := a.cfg()
	out := a.output()

	if err := a.syncProviderHistoryRuntimeConfigFromCurrentProject(); err != nil {
		yellow.Fprintf(out, "Warning: Failed to sync provider history reduction config: %v\n", err)
	}

	a.syncRuntimeProviderConfig(cfg, out)

	// Model: プロバイダーの現在モデルとして設定に追従
	// ここでは provider_models の解決も含める
	modelLookupProvider := a.sessionProviderConfigKey(cfg)
	resolvedModel := cfg.GetSelectedModelForProvider(modelLookupProvider)
	if resolvedModel != "" {
		a.setCurrentModelAndSync(resolvedModel)
	}
	a.reconcileSessionForCurrentRuntime()
}

func (a *Agent) syncRuntimeProviderConfig(cfg *config.Config, out io.Writer) {
	if a == nil || cfg == nil {
		return
	}

	currentProviderConfigKey := a.currentProviderConfigKey()
	nextProviderConfigKey := config.ActiveProviderConfigKey(cfg.DefaultProvider)
	if nextProviderConfigKey == "" {
		return
	}

	if !config.SameProviderRuntimeIdentity(nextProviderConfigKey, a.ProviderName) {
		if err := a.SwitchProvider(cfg.DefaultProvider); err != nil {
			// /config 直後にエラーを出して操作を止めるより、既存プロバイダーで継続
			yellow.Fprintf(out, "Warning: Failed to switch provider: %v\n", err)
		}
		return
	}

	if nextProviderConfigKey == currentProviderConfigKey {
		return
	}

	// 同一 runtime provider 内の alias owner 変更は provider 再生成ではなく
	// Agent / provider が参照する config key を差し替えて反映する。
	a.ProviderConfigKey = nextProviderConfigKey
	syncProviderConfigKeyToProvider(a.CurrentProvider, nextProviderConfigKey)
}
