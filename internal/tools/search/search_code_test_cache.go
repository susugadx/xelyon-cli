package search

import (
	"sync"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type testSearchCache struct {
	mu          sync.Mutex
	data        map[string]string
	affected    map[string][]string
	dataKeys    map[string]string
	getCalls    int
	setCalls    int
	lastGetPath string
	lastSetPath string
}

func (c *testSearchCache) GetFile(path string) (string, bool) { return "", false }
func (c *testSearchCache) SetFile(path, content string)       {}
func (c *testSearchCache) GetDir(path string) (string, bool)  { return "", false }
func (c *testSearchCache) SetDir(path, result string)         {}
func (c *testSearchCache) InvalidateFile(path string)         {}
func (c *testSearchCache) InvalidateDir(path string)          {}
func (c *testSearchCache) Clear()                             {}
func (c *testSearchCache) ClearSearchCache()                  { tools.NotifySearchCacheCleared() }

func (c *testSearchCache) InvalidateSearchCacheForFile(absPath string) {
	c.mu.Lock()
	deletedKeys := make([]string, 0)
	deleted := false
	for key, files := range c.affected {
		for _, fp := range files {
			if fp == absPath {
				if dataKey, ok := c.dataKeys[key]; ok {
					delete(c.data, dataKey)
					delete(c.dataKeys, key)
				}
				delete(c.affected, key)
				deleted = true
				deletedKeys = append(deletedKeys, key)
				break
			}
		}
	}
	c.mu.Unlock()
	if deleted {
		tools.NotifySearchCacheInvalidatedKeys(deletedKeys)
	}
}

func (c *testSearchCache) GetSearch(pattern, path string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getCalls++
	c.lastGetPath = path
	key := pattern + "|" + path
	if v, ok := c.data[key]; ok {
		return v, true
	}
	return "", false
}

func (c *testSearchCache) SetSearch(pattern, path, result string, affectedFiles []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setCalls++
	c.lastSetPath = path
	key := pattern + "|" + path
	c.data[key] = result
	if c.affected == nil {
		c.affected = make(map[string][]string)
	}
	if c.dataKeys == nil {
		c.dataKeys = make(map[string]string)
	}
	searchKey := singlePatternBundleCacheKey(pattern, path)
	c.affected[searchKey] = append([]string(nil), affectedFiles...)
	c.dataKeys[searchKey] = key
}
