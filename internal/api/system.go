package api

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// CacheControl enables prompt caching for a content block.
type CacheControl struct {
	Type string `json:"type"`          // e.g. "ephemeral"
	TTL  string `json:"ttl,omitempty"` // 延長キャッシュ用（"1h" 等）
}

// NewCacheControl はconfig設定に基づいてCacheControlを生成する。
// CacheTTL が "5m"（デフォルト）ならTTLフィールドなし、"1h" 等なら ttl を設定。
func NewCacheControl() *CacheControl {
	cc := &CacheControl{Type: "ephemeral"}
	cfg := config.GetGlobalConfig()
	if cfg != nil && cfg.PromptCache.CacheTTL != "" && cfg.PromptCache.CacheTTL != "5m" {
		cc.TTL = cfg.PromptCache.CacheTTL
	}
	return cc
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
// Anthropic prompt caching の閾値判定に tools 定義も含められます。
// 実際の最低キャッシュ可能トークン数はモデルごとに異なるため、
// 閾値超過のための安定ブロック追加は prompt builder 側で行います。
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
				CacheControl: NewCacheControl(),
			},
		}
	}

	// 1ブロック: 唯一のブロックに cache_control を設定
	return []SystemBlock{
		{
			Type:         "text",
			Text:         parts[0],
			CacheControl: NewCacheControl(),
		},
	}
}
