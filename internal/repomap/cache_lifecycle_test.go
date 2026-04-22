package repomap

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_SaveLoad(t *testing.T) {
	setProjectMapTestHome(t)

	root := filepath.Join(t.TempDir(), "repo")
	cache := &MapCache{
		RootPath: root,
		Files: map[string]*CacheFile{
			"main.go": {
				ModTime:   time.Unix(123, 0).UTC(),
				LineCount: 42,
				Symbols: []Symbol{
					{Line: 3, Signature: "func Build() error"},
				},
			},
		},
	}

	if err := saveMapCache(root, cache); err != nil {
		t.Fatalf("saveMapCache() error = %v", err)
	}

	loaded, err := loadMapCache(root)
	if err != nil {
		t.Fatalf("loadMapCache() error = %v", err)
	}
	if loaded.RootPath != root {
		t.Fatalf("RootPath = %q, want %q", loaded.RootPath, root)
	}
	if loaded.Files["main.go"].LineCount != 42 {
		t.Fatalf("LineCount = %d, want 42", loaded.Files["main.go"].LineCount)
	}
	if len(loaded.Files["main.go"].Symbols) != 1 {
		t.Fatalf("Symbols length = %d, want 1", len(loaded.Files["main.go"].Symbols))
	}
}

func TestLoadMapCacheWithFallback_InvalidJSON(t *testing.T) {
	setProjectMapTestHome(t)

	root := filepath.Join(t.TempDir(), "repo")
	cachePath, err := cacheFilePath(root)
	if err != nil {
		t.Fatalf("cacheFilePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("{invalid"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cache := loadMapCacheWithFallback(root)
	if cache == nil {
		t.Fatal("loadMapCacheWithFallback() returned nil")
	}
	if cache.RootPath != root {
		t.Fatalf("RootPath = %q, want %q", cache.RootPath, root)
	}
	if len(cache.Files) != 0 {
		t.Fatalf("Files length = %d, want 0", len(cache.Files))
	}
}
