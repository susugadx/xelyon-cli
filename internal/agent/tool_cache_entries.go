package agent

import (
	"os"
	"sort"
	"time"

	"github.com/susugadx/xelyon-cli/internal/searchcache"
	"github.com/susugadx/xelyon-cli/internal/tools/file/listtool"
)

// GetFile はファイル内容のキャッシュを取得
func (c *ToolCache) GetFile(path string) (string, bool) {
	c.mu.RLock()
	entry, exists := c.files[path]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	// ファイルの mtime をチェック
	info, err := os.Stat(path)
	if err != nil {
		// ファイルが削除された等
		c.InvalidateFile(path)
		return "", false
	}

	// ファイルが変更されていたらキャッシュ無効
	if info.ModTime().After(entry.ModTime) {
		c.InvalidateFile(path)
		return "", false
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.files[path] = entry
	c.mu.Unlock()
	return entry.Content, true
}

// SetFile はファイル内容をキャッシュに保存
func (c *ToolCache) SetFile(path, content string) {
	// 1MB以上はキャッシュしない
	if len(content) > MaxFileCacheSize {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.files[path] = cacheEntry{
		Content:    content,
		ModTime:    info.ModTime(),
		AccessedAt: time.Now(),
	}
	pruneOldestEntries(c.files, MaxFileCacheEntries)
}

// GetDir はディレクトリ一覧のキャッシュを取得
func (c *ToolCache) GetDir(path string) (string, bool) {
	c.mu.RLock()
	entry, exists := c.dirs[path]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	// ディレクトリの mtime をチェック
	dirPath := listtool.CachePhysicalPath(path)
	info, err := os.Stat(dirPath)
	if err != nil {
		c.InvalidateDir(path)
		return "", false
	}

	// ディレクトリが変更されていたらキャッシュ無効
	if info.ModTime().After(entry.ModTime) {
		c.InvalidateDir(path)
		return "", false
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.dirs[path] = entry
	c.mu.Unlock()
	return entry.Content, true
}

// SetDir はディレクトリ一覧をキャッシュに保存
func (c *ToolCache) SetDir(path, result string) {
	key := listtool.NormalizeCacheKey(path)
	if key == "" {
		return
	}
	dirPath := listtool.CachePhysicalPath(key)
	info, err := os.Stat(dirPath)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.dirs[key] = cacheEntry{
		Content:    result,
		ModTime:    info.ModTime(),
		AccessedAt: time.Now(),
	}
	pruneOldestEntries(c.dirs, MaxDirCacheEntries)
}

// InvalidateFile は指定ファイルのキャッシュを無効化
func (c *ToolCache) InvalidateFile(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.files, path)
}

// InvalidateDir は指定ディレクトリのキャッシュを無効化
func (c *ToolCache) InvalidateDir(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	targetPath := listtool.CachePhysicalPath(path)
	for key := range c.dirs {
		if listtool.CachePhysicalPath(key) == targetPath {
			delete(c.dirs, key)
		}
	}
}

// Clear は全キャッシュをクリア
func (c *ToolCache) Clear() {
	c.mu.Lock()
	c.files = make(map[string]cacheEntry)
	c.dirs = make(map[string]cacheEntry)
	c.searches = make(map[string]cacheEntry)
	c.negatives = make(map[string]negativeCacheEntry)
	c.mu.Unlock()
	searchcache.NotifySearchCacheCleared()
}

// Stats はキャッシュの統計情報を返す
func (c *ToolCache) Stats() (files, dirs, searches int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.files), len(c.dirs), len(c.searches)
}

// InvalidateSearchCacheForFile は指定ファイルパスを含む検索キャッシュエントリを無効化
func (c *ToolCache) InvalidateSearchCacheForFile(absPath string) {
	c.mu.Lock()
	deletedKeys := make([]string, 0)
	for key, entry := range c.searches {
		for _, fp := range entry.AffectedFiles {
			if fp == absPath {
				delete(c.searches, key)
				deletedKeys = append(deletedKeys, key)
				break
			}
		}
	}
	c.mu.Unlock()
	if len(deletedKeys) > 0 {
		searchcache.NotifySearchCacheInvalidatedKeys(deletedKeys)
	}
}

// RecentFilePaths は最近アクセスしたファイルキャッシュのパスを新しい順で返す。
func (c *ToolCache) RecentFilePaths(limit int) []string {
	if c == nil || limit <= 0 {
		return nil
	}

	c.mu.RLock()
	type entryWithPath struct {
		path       string
		accessedAt time.Time
	}
	entries := make([]entryWithPath, 0, len(c.files))
	for path, entry := range c.files {
		entries = append(entries, entryWithPath{
			path:       path,
			accessedAt: entry.AccessedAt,
		})
	}
	c.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].accessedAt.After(entries[j].accessedAt)
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.path)
	}
	return paths
}
