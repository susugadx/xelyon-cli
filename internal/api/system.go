package api

import "github.com/susugadx/xelyon-cli/internal/config"

// CacheControl enables prompt caching for a content block.
type CacheControl struct {
	Type string `json:"type"` // e.g. "ephemeral"
}

// SystemBlock represents a system prompt content block.
type SystemBlock struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// BuildSystemField はプロンプトキャッシュ対応のシステムフィールドを構築します。
// config でキャッシュが有効な場合、SystemBlock 配列を返し、無効な場合は string を返します。
func BuildSystemField(systemPrompt string) interface{} {
	cfg := config.GetGlobalConfig()
	if cfg == nil || !cfg.PromptCache.Enabled {
		return systemPrompt
	}

	return []SystemBlock{
		{
			Type: "text",
			Text: systemPrompt,
			CacheControl: &CacheControl{
				Type: "ephemeral",
			},
		},
	}
}
