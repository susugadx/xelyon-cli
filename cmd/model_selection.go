package cmd

import (
	"github.com/susugadx/xelyon-cli/internal/cliruntime"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// getModel はフラグからモデルを決定する。
// 優先順位: --model > XELYON_MODEL > selected provider model resolution
func getModel(cfg *config.Config) string {
	return cliruntime.GetModel(cfg, cliruntime.ModelSelection{
		ProviderFlag: providerFlag,
		ModelFlag:    modelFlag,
	})
}
