package api

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

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

// SystemPromptCacheBoundary は system prompt の静的/動的部分の境界マーカー。
// BuildSystemField はこの位置で分割し、前半に cache_control を設定する。
// Plan Mode 追加時にこのマーカーを挿入することで、base prompt のキャッシュを維持する。
const SystemPromptCacheBoundary = "\n---XELYON_CACHE_SPLIT---\n"

// BuildSystemField はプロンプトキャッシュ対応のシステムフィールドを構築します。
// キャッシュ有効時は SystemBlock 配列を返し（静的部分に cache_control 付き）、
// 無効時は string を返します。
// SystemPromptCacheBoundary が含まれる場合、そこで分割して
// 前半（静的）に cache_control、後半（動的）は cache_control なしのブロックにします。
func BuildSystemField(systemPrompt string) interface{} {
	cfg := config.GetGlobalConfig()
	if cfg == nil || !cfg.PromptCache.Enabled {
		// 境界マーカーを除去して plain string で返す
		return strings.ReplaceAll(systemPrompt, SystemPromptCacheBoundary, "\n\n")
	}

	parts := strings.SplitN(systemPrompt, SystemPromptCacheBoundary, 2)

	// 静的部分（base）に cache_control を設定
	blocks := []SystemBlock{
		{
			Type:         "text",
			Text:         parts[0],
			CacheControl: &CacheControl{Type: "ephemeral"},
		},
	}

	// 動的部分があれば別ブロック（cache_control なし）
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		blocks = append(blocks, SystemBlock{
			Type: "text",
			Text: parts[1],
		})
	}

	return blocks
}
