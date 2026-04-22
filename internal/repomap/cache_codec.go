package repomap

import (
	"encoding/json"
	"time"
)

func decodeMapCache(data []byte) (*MapCache, error) {
	var cache MapCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

func encodeMapCache(rootPath string, cache *MapCache, updatedAt time.Time) ([]byte, error) {
	if cache == nil {
		cache = newEmptyMapCache(rootPath)
	}
	cache.RootPath = rootPath
	cache.UpdatedAt = updatedAt
	if cache.Files == nil {
		cache.Files = map[string]*CacheFile{}
	}
	return json.MarshalIndent(cache, "", "  ")
}
