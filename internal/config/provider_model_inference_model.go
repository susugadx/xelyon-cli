package config

import "github.com/susugadx/xelyon-cli/internal/llmcatalog"

// InferProviderFromModel はモデル名から provider を推定する。
// 推定できない場合は空文字を返す。
func InferProviderFromModel(model string) string {
	return llmcatalog.InferProviderFromModel(model)
}
