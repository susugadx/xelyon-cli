package repomap

import (
	"testing"
	"time"
)

func TestApplyCachePolicyToStates_ReusesOnlyMatchedModTime(t *testing.T) {
	modTime := time.Unix(100, 0).UTC()
	states := []fileState{
		{
			path:    "main.go",
			modTime: modTime,
		},
		{
			path:    "stale.go",
			modTime: modTime,
		},
	}
	cache := &MapCache{
		Files: map[string]*CacheFile{
			"main.go": {
				ModTime:   modTime,
				LineCount: 10,
				Symbols: []Symbol{
					{Name: "Build", Line: 3, Signature: "func Build()"},
				},
			},
			"stale.go": {
				ModTime: modTime.Add(-time.Second),
			},
		},
	}

	got := applyCachePolicyToStates(states, cache)
	if got[0].cached == nil {
		t.Fatal("main.go should reuse cache")
	}
	if got[1].cached != nil {
		t.Fatal("stale.go should not reuse stale cache")
	}
}

func TestResolveReusableCacheFile_ReturnsClonedCache(t *testing.T) {
	modTime := time.Unix(200, 0).UTC()
	cache := &MapCache{
		Files: map[string]*CacheFile{
			"main.go": {
				ModTime: modTime,
				Symbols: []Symbol{
					{Name: "Build", Line: 3, Signature: "func Build()"},
				},
			},
		},
	}

	got, ok := resolveReusableCacheFile(cache, "main.go", modTime)
	if !ok {
		t.Fatal("resolveReusableCacheFile() = miss, want hit")
	}
	got.Symbols[0].Name = "Changed"
	if cache.Files["main.go"].Symbols[0].Name != "Build" {
		t.Fatal("expected returned cache file to be cloned")
	}
}

func TestLoadBuildInputCache_ReturnsEmptyCacheOnFailure(t *testing.T) {
	setProjectMapTestHome(t)
	root := t.TempDir()
	cache := loadBuildInputCache(root)
	if cache == nil {
		t.Fatal("loadBuildInputCache() returned nil")
	}
	if cache.Files == nil {
		t.Fatal("cache.Files should be initialized")
	}
}
