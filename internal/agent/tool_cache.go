package agent

import (
	"sync"
	"time"
)

// ToolCache はツール結果のキャッシュ
// read_file, list_dir の結果をキャッシュしてトークン消費を削減
type ToolCache struct {
	files           map[string]cacheEntry
	dirs            map[string]cacheEntry
	searches        map[string]cacheEntry
	negatives       map[string]negativeCacheEntry
	skipPersistence bool
	mu              sync.RWMutex
}

// cacheEntry はキャッシュエントリ
type cacheEntry struct {
	Content       string
	ModTime       time.Time
	AccessedAt    time.Time
	AffectedFiles []string // 検索キャッシュのみ使用
}

// negativeCacheEntry はエラー/空結果のネガティブキャッシュエントリ
type negativeCacheEntry struct {
	ToolName string
	Result   string // 元のエラーメッセージ（1行目）
	CachedAt time.Time
}

// NegativeCacheTTL はネガティブキャッシュの有効期間
const NegativeCacheTTL = 30 * time.Second

// MaxFileCacheSize はキャッシュする最大ファイルサイズ (1MB)
const MaxFileCacheSize = 1024 * 1024

const (
	MaxFileCacheEntries   = 100
	MaxDirCacheEntries    = 50
	MaxSearchCacheEntries = 50
)

// NewToolCache は新しい ToolCache を作成
func NewToolCache() *ToolCache {
	return &ToolCache{
		files:     make(map[string]cacheEntry),
		dirs:      make(map[string]cacheEntry),
		searches:  make(map[string]cacheEntry),
		negatives: make(map[string]negativeCacheEntry),
	}
}

// NewEphemeralToolCache はディスク永続化しない ToolCache を作成する。
func NewEphemeralToolCache() *ToolCache {
	cache := NewToolCache()
	cache.skipPersistence = true
	return cache
}
