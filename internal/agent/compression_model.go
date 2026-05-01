package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// defaultCompressionModel はプロバイダー別のデフォルト圧縮モデルを返す。
// compression.model が空の場合に使用する。
func defaultCompressionModel(providerName string) string {
	entry, ok := llmcatalog.ProviderDescriptorFor(providerName)
	if !ok {
		return ""
	}
	return strings.TrimSpace(entry.CompressionModel)
}

// getCompressionModel は圧縮に使用するモデルを返す。
// 優先順位: config.compression.model > プロバイダー別デフォルト > 現在モデル。
func (a *Agent) getCompressionModel() string {
	cfg := a.cfg()
	if strings.EqualFold(cfg.Compression.Model, "main") {
		return a.CurrentModel
	}
	if cfg.Compression.Model != "" {
		return cfg.Compression.Model
	}
	if model := defaultCompressionModel(a.ProviderName); model != "" {
		return model
	}
	return a.CurrentModel
}
