package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/listtool"
)

const (
	toolCacheDir     = ".xelyon/cache"
	toolCacheFile    = ".xelyon/cache/tool_cache.json"
	maxCacheFileSize = 10 * 1024 * 1024 // 10MB
)

// persistedCacheEntry はディスク永続化用のエントリ
type persistedCacheEntry struct {
	Content string    `json:"content"`
	ModTime time.Time `json:"mod_time"`
}

// persistedCache はディスク永続化用のキャッシュ全体
type persistedCache struct {
	Files map[string]persistedCacheEntry `json:"files"`
	Dirs  map[string]persistedCacheEntry `json:"dirs"`
}

// Save はファイル・ディレクトリキャッシュをディスクに永続化する。
// searches, negatives はセッション跨ぎで無効になりやすいため永続化しない。
func (c *ToolCache) Save() error {
	c.mu.RLock()
	pc := persistedCache{
		Files: make(map[string]persistedCacheEntry, len(c.files)),
		Dirs:  make(map[string]persistedCacheEntry, len(c.dirs)),
	}
	for k, v := range c.files {
		pc.Files[k] = persistedCacheEntry{Content: v.Content, ModTime: v.ModTime}
	}
	for k, v := range c.dirs {
		pc.Dirs[k] = persistedCacheEntry{Content: v.Content, ModTime: v.ModTime}
	}
	c.mu.RUnlock()

	// エントリがなければ保存不要（既存ファイルも削除）
	if len(pc.Files) == 0 && len(pc.Dirs) == 0 {
		_ = os.Remove(toolCacheFile)
		return nil
	}

	data, err := json.Marshal(pc)
	if err != nil {
		return fmt.Errorf("marshal tool cache: %w", err)
	}

	// サイズ上限チェック
	if len(data) > maxCacheFileSize {
		return nil
	}

	if err := os.MkdirAll(toolCacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.WriteFile(toolCacheFile, data, 0600); err != nil {
		return fmt.Errorf("write tool cache: %w", err)
	}
	return nil
}

// Load はディスクからキャッシュを復元し、mtime 検証して変更済みエントリを破棄する。
func (c *ToolCache) Load() error {
	data, err := os.ReadFile(toolCacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ファイルなし → 正常
		}
		return fmt.Errorf("read tool cache: %w", err)
	}

	// サイズ上限チェック
	if len(data) > maxCacheFileSize {
		_ = os.Remove(toolCacheFile)
		return nil
	}

	var pc persistedCache
	if err := json.Unmarshal(data, &pc); err != nil {
		// 壊れたファイルは削除して続行
		_ = os.Remove(toolCacheFile)
		return nil
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for path, entry := range pc.Files {
		info, err := os.Stat(path)
		if err != nil {
			continue // ファイルが存在しない → スキップ
		}
		if info.ModTime().After(entry.ModTime) {
			continue // 変更されている → スキップ
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		c.files[absPath] = cacheEntry{
			Content:    entry.Content,
			ModTime:    entry.ModTime,
			AccessedAt: now,
		}
	}

	for path, entry := range pc.Dirs {
		key := listtool.NormalizeCacheKey(path)
		if key == "" {
			continue
		}
		dirPath := listtool.CachePhysicalPath(key)
		info, err := os.Stat(dirPath)
		if err != nil {
			continue
		}
		if info.ModTime().After(entry.ModTime) {
			continue
		}
		c.dirs[key] = cacheEntry{
			Content:    entry.Content,
			ModTime:    entry.ModTime,
			AccessedAt: now,
		}
	}

	return nil
}

func pruneOldestEntries(m map[string]cacheEntry, maxEntries int) []string {
	if len(m) <= maxEntries {
		return nil
	}

	// 超過数 + 10% をまとめて削除（頻繁な prune を回避）
	pruneCount := len(m) - maxEntries + maxEntries/10
	if pruneCount < 1 {
		pruneCount = 1
	}

	type kv struct {
		key  string
		time time.Time
	}
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{key: k, time: v.AccessedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].time.Before(items[j].time)
	})
	deletedKeys := make([]string, 0, pruneCount)
	for i := 0; i < pruneCount && i < len(items); i++ {
		delete(m, items[i].key)
		deletedKeys = append(deletedKeys, items[i].key)
	}
	return deletedKeys
}

// インターフェース実装の確認
var _ tools.ToolCacheInterface = (*ToolCache)(nil)
