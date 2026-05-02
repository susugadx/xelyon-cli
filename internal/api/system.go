package api

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// SystemPromptLayout は system prompt の静的/動的レイアウトを表す。
type SystemPromptLayout struct {
	Static  string
	Dynamic string
}

// CacheControl enables prompt caching for a content block.
type CacheControl struct {
	Type string `json:"type"`          // e.g. "ephemeral"
	TTL  string `json:"ttl,omitempty"` // 延長キャッシュ用（"1h" 等）
}

// NewCacheControlWithConfig は明示指定した設定に基づいて CacheControl を生成する。
func NewCacheControlWithConfig(cfg *config.Config) *CacheControl {
	cc := &CacheControl{Type: "ephemeral"}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
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

// SplitSystemPromptLayout は boundary を解釈して static/dynamic を返す。
// boundary が複数ある場合は dynamic 側に正規化して集約する。
func SplitSystemPromptLayout(systemPrompt string) SystemPromptLayout {
	parts := strings.Split(systemPrompt, SystemPromptCacheBoundary)
	if len(parts) == 1 {
		return SystemPromptLayout{Static: strings.TrimRight(systemPrompt, "\n")}
	}

	layout := SystemPromptLayout{
		Static: strings.TrimRight(parts[0], "\n"),
	}
	for _, part := range parts[1:] {
		layout.AppendDynamic(part)
	}
	return layout
}

// Compose は static/dynamic を boundary 1つに正規化して連結する。
func (l SystemPromptLayout) Compose() string {
	static := strings.TrimRight(l.Static, "\n")
	dynamic := strings.Trim(l.Dynamic, "\n")
	if strings.TrimSpace(dynamic) == "" {
		return static
	}
	if static == "" {
		return dynamic
	}
	return static + SystemPromptCacheBoundary + dynamic
}

// AppendDynamic は dynamic 末尾へ section を追加する。
func (l *SystemPromptLayout) AppendDynamic(section string) {
	if l == nil {
		return
	}
	section = strings.Trim(section, "\n")
	if strings.TrimSpace(section) == "" {
		return
	}
	if strings.TrimSpace(l.Dynamic) == "" {
		l.Dynamic = section
		return
	}
	l.Dynamic = strings.TrimRight(l.Dynamic, "\n") + "\n\n" + section
}

// SetStatic は static ブロックを整形して設定する。
func (l *SystemPromptLayout) SetStatic(static string) {
	if l == nil {
		return
	}
	l.Static = strings.TrimRight(static, "\n")
}

// SetDynamic は dynamic ブロックを整形して設定する。
func (l *SystemPromptLayout) SetDynamic(dynamic string) {
	if l == nil {
		return
	}
	l.Dynamic = strings.Trim(dynamic, "\n")
}

// BuildSystemFieldWithConfig は明示指定した設定でプロンプトキャッシュ対応のシステムフィールドを構築する。
func BuildSystemFieldWithConfig(systemPrompt string, cfg *config.Config) interface{} {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if cfg == nil || !cfg.PromptCache.Enabled {
		// 境界マーカーを除去して plain string で返す
		return strings.ReplaceAll(systemPrompt, SystemPromptCacheBoundary, "\n\n")
	}

	layout := SplitSystemPromptLayout(systemPrompt)
	hasDynamic := strings.TrimSpace(layout.Dynamic) != ""

	if hasDynamic {
		// 2ブロック: 静的部分（cache_control なし）+ 動的部分（cache_control あり）
		// cache_control を最後のブロックに置くことで、prefix = tools + system 全体がキャッシュ対象
		return []SystemBlock{
			{
				Type: "text",
				Text: layout.Static,
			},
			{
				Type:         "text",
				Text:         layout.Dynamic,
				CacheControl: NewCacheControlWithConfig(cfg),
			},
		}
	}

	// 1ブロック: 唯一のブロックに cache_control を設定
	return []SystemBlock{
		{
			Type:         "text",
			Text:         layout.Static,
			CacheControl: NewCacheControlWithConfig(cfg),
		},
	}
}
