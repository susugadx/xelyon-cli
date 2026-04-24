package agent

import (
	"sort"
	"time"

	"github.com/susugadx/xelyon-cli/internal/searchcache"
)

// searchCacheKey は検索キャッシュのキーを生成
func searchCacheKey(pattern, cacheKey string) string {
	return pattern + "::" + cacheKey
}

// GetSearch は検索結果のキャッシュを取得
func (c *ToolCache) GetSearch(pattern, cacheKey string) (string, bool) {
	key := searchCacheKey(pattern, cacheKey)

	c.mu.RLock()
	entry, exists := c.searches[key]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.searches[key] = entry
	c.mu.Unlock()
	return entry.Content, true
}

// SetSearch は検索結果をキャッシュに保存
func (c *ToolCache) SetSearch(pattern, cacheKey, result string, affectedFiles []string) {
	key := searchCacheKey(pattern, cacheKey)

	c.mu.Lock()
	c.searches[key] = cacheEntry{
		Content:       result,
		AccessedAt:    time.Now(),
		AffectedFiles: append([]string(nil), affectedFiles...),
		// 検索キャッシュはmtimeチェック不要（全クリア方式）
	}
	evicted := pruneOldestEntries(c.searches, MaxSearchCacheEntries)
	c.mu.Unlock()
	if len(evicted) > 0 {
		searchcache.NotifySearchCacheEvicted(evicted)
	}
}

// ClearSearchCache は検索キャッシュをクリア
func (c *ToolCache) ClearSearchCache() {
	c.mu.Lock()
	c.searches = make(map[string]cacheEntry)
	c.mu.Unlock()
	searchcache.NotifySearchCacheCleared()
}

// RecentSearchAffectedFiles は最近アクセスした検索キャッシュ由来の affected files を返す。
func (c *ToolCache) RecentSearchAffectedFiles(limit int) []string {
	return c.recentSearchAffectedFiles(limit, "")
}

// RecentSearchAffectedFilesExcluding は指定検索キーを除いた最近アクセス検索の affected files を返す。
func (c *ToolCache) RecentSearchAffectedFilesExcluding(pattern, cacheKey string, limit int) []string {
	return c.recentSearchAffectedFiles(limit, searchCacheKey(pattern, cacheKey))
}

func (c *ToolCache) recentSearchAffectedFiles(limit int, excludedKey string) []string {
	if c == nil || limit <= 0 {
		return nil
	}

	c.mu.RLock()
	type searchEntry struct {
		affectedFiles []string
		accessedAt    time.Time
	}
	entries := make([]searchEntry, 0, len(c.searches))
	for key, entry := range c.searches {
		if key == excludedKey || len(entry.AffectedFiles) == 0 {
			continue
		}
		entries = append(entries, searchEntry{
			affectedFiles: append([]string(nil), entry.AffectedFiles...),
			accessedAt:    entry.AccessedAt,
		})
	}
	c.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].accessedAt.After(entries[j].accessedAt)
	})

	seen := make(map[string]struct{})
	paths := make([]string, 0, limit)
	for _, entry := range entries {
		for _, path := range entry.affectedFiles {
			if path == "" {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
			if len(paths) >= limit {
				return paths
			}
		}
	}

	return paths
}
