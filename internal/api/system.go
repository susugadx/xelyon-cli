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
// BuildSystemField はこの位置で分割して2ブロックにし、最後のブロックに cache_control を設定する。
// Plan Mode 追加時にこのマーカーを挿入することで、system 全体を1つの cache prefix にまとめる。
const SystemPromptCacheBoundary = "\n---XELYON_CACHE_SPLIT---\n"

// BuildSystemField はプロンプトキャッシュ対応のシステムフィールドを構築します。
// キャッシュ有効時は SystemBlock 配列を返し（最後のブロックに cache_control 付き）、
// 無効時は string を返します。
// SystemPromptCacheBoundary が含まれる場合、そこで分割して2ブロックにします。
//
// cache_control は最後の system ブロックに設定します。
// これにより cache prefix は tools + system 全体となり、
// Opus 4.6 の最低キャッシュ可能トークン数（4096）を確実に超えます。
// （以前は parts[0] のみに設定していたため、tools + system[0] が 4096 未満の場合に
// Opus でキャッシュが効かない問題がありました）
func BuildSystemField(systemPrompt string) interface{} {
	cfg := config.GetGlobalConfig()
	if cfg == nil || !cfg.PromptCache.Enabled {
		// 境界マーカーを除去して plain string で返す
		return strings.ReplaceAll(systemPrompt, SystemPromptCacheBoundary, "\n\n")
	}

	parts := strings.SplitN(systemPrompt, SystemPromptCacheBoundary, 2)

	hasDynamic := len(parts) > 1 && strings.TrimSpace(parts[1]) != ""

	if hasDynamic {
		// 2ブロック: 静的部分（cache_control なし）+ 動的部分（cache_control あり）
		// cache_control を最後のブロックに置くことで、prefix = tools + system 全体がキャッシュ対象
		return []SystemBlock{
			{
				Type: "text",
				Text: parts[0],
			},
			{
				Type:         "text",
				Text:         parts[1],
				CacheControl: &CacheControl{Type: "ephemeral"},
			},
		}
	}

	// 1ブロック: 唯一のブロックに cache_control を設定
	return []SystemBlock{
		{
			Type:         "text",
			Text:         parts[0],
			CacheControl: &CacheControl{Type: "ephemeral"},
		},
	}
}
