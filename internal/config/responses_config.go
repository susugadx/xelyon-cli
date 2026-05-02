package config

import "github.com/susugadx/xelyon-cli/internal/llmcatalog"

// IsResponsesAPIModel はモデルが OpenAI Responses API を使用するか判定する。
// 対応モデルは prefix マッチで自動判定し、設定リストをフォールバックとして使用する。
func (c *Config) IsResponsesAPIModel(model string) bool {
	return llmcatalog.IsOpenAIResponsesModel(model, c.OpenAI.ResponsesAPIModels)
}

// IsProviderResponsesAPIModel は provider/model の実行経路が Responses API か判定する。
// deployment/alias 名は catalog_model に解決してから判定する。
func (c *Config) IsProviderResponsesAPIModel(provider, model string) bool {
	return c.IsProviderResponsesAPIRequest(provider, provider, model)
}

// IsProviderResponsesAPIRequest は実際の provider 実行経路が Responses API か判定する。
//
// runtimeProvider は API 実装を選ぶ provider、catalogProvider は deployment/alias の
// catalog_model 解決に使う provider_models の owner を渡す。
func (c *Config) IsProviderResponsesAPIRequest(runtimeProvider, catalogProvider, model string) bool {
	runtimeProvider = CanonicalProviderName(runtimeProvider)
	if !ProviderSupportsResponsesAPI(runtimeProvider) {
		return false
	}
	if runtimeProvider == "azure" {
		return true
	}
	if catalogProvider == "" {
		catalogProvider = runtimeProvider
	}
	return c.IsResponsesAPIModel(c.ModelCatalogName(catalogProvider, model))
}

// ResponsesStoreEnabled は Responses API の provider-side response 保存を有効にするか返す。
func (c *Config) ResponsesStoreEnabled() bool {
	if c == nil {
		return true
	}
	return c.Responses.Store
}

// ResponsesPersistResponseIDEnabled は session へ response ID を保存・復元するか返す。
func (c *Config) ResponsesPersistResponseIDEnabled() bool {
	if c == nil {
		return true
	}
	return c.Responses.Store && c.Responses.PersistResponseID
}

// ResponsesServerCompactionEnabled は Responses API の server-side context 管理を優先するか返す。
func (c *Config) ResponsesServerCompactionEnabled() bool {
	if c == nil {
		return true
	}
	return c.Responses.Store && c.Responses.ServerCompaction.Enabled
}

// ResponsesServerCompactionCompactThreshold は server-side compaction の閾値設定を返す。
// 0 は provider 側で自動解決する。
func (c *Config) ResponsesServerCompactionCompactThreshold() int {
	if c == nil {
		return 0
	}
	return c.Responses.ServerCompaction.CompactThreshold
}

// ResponsesServerCompactionLocalFallbackEnabled は server-side compaction 不可時に
// local auto-compress へフォールバックするか返す。
func (c *Config) ResponsesServerCompactionLocalFallbackEnabled() bool {
	if c == nil {
		return true
	}
	return c.Responses.ServerCompaction.LocalFallback
}
