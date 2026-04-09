package cmd

import "github.com/susugadx/xelyon-cli/internal/config"

// getModel はフラグからモデルを決定する。
// 優先順位: --model > selected provider model resolution
func getModel(cfg *config.Config) string {
	if modelFlag != "" {
		return modelFlag
	}

	if cfg == nil {
		return "deepseek-chat"
	}

	providerName := resolveProviderName(providerFlag, cfg.DefaultProvider)
	return cfg.GetSelectedModelForProvider(providerName)
}
