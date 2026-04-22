package repomap

import (
	"errors"
	"os"
	"time"
)

// MapCache はプロジェクトマップのキャッシュ。
type MapCache struct {
	RootPath  string                `json:"root_path"`
	UpdatedAt time.Time             `json:"updated_at"`
	Files     map[string]*CacheFile `json:"files"`
}

// CacheFile はファイルごとのキャッシュ。
type CacheFile struct {
	ModTime   time.Time `json:"mod_time"`
	LineCount int       `json:"line_count"`
	Symbols   []Symbol  `json:"symbols,omitempty"`
}

func newEmptyMapCache(rootPath string) *MapCache {
	return &MapCache{
		RootPath: rootPath,
		Files:    map[string]*CacheFile{},
	}
}

func loadMapCacheWithFallback(rootPath string) *MapCache {
	cache, err := loadMapCache(rootPath)
	if err != nil || cache == nil {
		return newEmptyMapCache(rootPath)
	}
	if cache.Files == nil {
		cache.Files = map[string]*CacheFile{}
	}
	return cache
}

func loadMapCache(rootPath string) (*MapCache, error) {
	data, err := readMapCacheData(rootPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newEmptyMapCache(rootPath), nil
		}
		return nil, err
	}

	cache, err := decodeMapCache(data)
	if err != nil {
		return nil, err
	}
	return normalizeLoadedMapCache(rootPath, cache), nil
}

func saveMapCache(rootPath string, cache *MapCache) error {
	data, err := encodeMapCache(rootPath, cache, time.Now().UTC())
	if err != nil {
		return err
	}
	return writeMapCacheData(rootPath, data)
}

func cloneCacheFile(file *CacheFile) *CacheFile {
	if file == nil {
		return nil
	}
	cloned := &CacheFile{
		ModTime:   file.ModTime,
		LineCount: file.LineCount,
	}
	if len(file.Symbols) > 0 {
		cloned.Symbols = append([]Symbol(nil), file.Symbols...)
	}
	return cloned
}

func normalizeLoadedMapCache(rootPath string, cache *MapCache) *MapCache {
	if cache == nil {
		return newEmptyMapCache(rootPath)
	}
	if cache.RootPath == "" {
		cache.RootPath = rootPath
	}
	if cache.Files == nil {
		cache.Files = map[string]*CacheFile{}
	}
	return cache
}
