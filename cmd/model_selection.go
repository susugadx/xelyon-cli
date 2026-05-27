package cmd

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// getModel はフラグからモデルを決定する。
// 優先順位: --model > XELYON_MODEL > selected provider model resolution
func getModel(cfg *config.Config) string {
	if modelFlag != "" {
		return modelFlag
	}
	if envModel := strings.TrimSpace(os.Getenv("XELYON_MODEL")); envModel != "" {
		return envModel
	}

	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	providerName := resolveProviderName(providerFlag, cfg.DefaultProvider)
	return cfg.GetSelectedModelForProvider(providerName)
}
