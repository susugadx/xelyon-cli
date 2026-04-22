package repomap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	cachePath, err := cacheFilePath(rootPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newEmptyMapCache(rootPath), nil
		}
		return nil, err
	}

	var cache MapCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if cache.Files == nil {
		cache.Files = map[string]*CacheFile{}
	}
	return &cache, nil
}

func saveMapCache(rootPath string, cache *MapCache) error {
	cachePath, err := cacheFilePath(rootPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}

	cache.RootPath = rootPath
	cache.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0600)
}

func cacheFilePath(rootPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(filepath.Clean(rootPath)))
	name := hex.EncodeToString(sum[:])[:12] + ".json"
	return filepath.Join(home, ".xelyon", "cache", "projectmap", name), nil
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
