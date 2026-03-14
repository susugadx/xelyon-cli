package repomap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setProjectMapTestHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

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

func TestCache_ModTimeChanged(t *testing.T) {
	setProjectMapTestHome(t)

	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	cache := &MapCache{
		RootPath: root,
		Files: map[string]*CacheFile{
			"main.go": {
				ModTime:   info.ModTime().Add(-time.Minute).UTC(),
				LineCount: 1,
			},
		},
	}

	pm := NewProjectMap(root, 1000)
	states, err := pm.buildFileStates([]string{"main.go"}, cache)
	if err != nil {
		t.Fatalf("buildFileStates() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("states length = %d, want 1", len(states))
	}
	if states[0].cached != nil {
		t.Fatal("expected cache miss when modTime changed")
	}
}

func TestCache_NewFile(t *testing.T) {
	setProjectMapTestHome(t)

	root := t.TempDir()
	for _, name := range []string{"main.go", "extra.go"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	info, err := os.Stat(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}

	cache := &MapCache{
		RootPath: root,
		Files: map[string]*CacheFile{
			"main.go": {
				ModTime:   info.ModTime().UTC(),
				LineCount: 1,
				Symbols: []Symbol{
					{Line: 1, Signature: "package main"},
				},
			},
		},
	}

	pm := NewProjectMap(root, 1000)
	states, err := pm.buildFileStates([]string{"main.go", "extra.go"}, cache)
	if err != nil {
		t.Fatalf("buildFileStates() error = %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("states length = %d, want 2", len(states))
	}

	var mainCached, extraCached bool
	for _, state := range states {
		switch state.path {
		case "main.go":
			mainCached = state.cached != nil
		case "extra.go":
			extraCached = state.cached != nil
		}
	}

	if !mainCached {
		t.Fatal("expected cached entry for main.go")
	}
	if extraCached {
		t.Fatal("expected cache miss for new file extra.go")
	}
}

func TestSymbolCache_NewFields(t *testing.T) {
	setProjectMapTestHome(t)

	root := filepath.Join(t.TempDir(), "repo")
	modTime := time.Unix(123, 0).UTC()

	type oldSymbol struct {
		Line      int    `json:"line"`
		Signature string `json:"signature"`
	}
	type oldCacheFile struct {
		ModTime   time.Time   `json:"mod_time"`
		LineCount int         `json:"line_count"`
		Symbols   []oldSymbol `json:"symbols"`
	}
	type oldMapCache struct {
		RootPath  string                   `json:"root_path"`
		UpdatedAt time.Time                `json:"updated_at"`
		Files     map[string]*oldCacheFile `json:"files"`
	}

	cachePath, err := cacheFilePath(root)
	if err != nil {
		t.Fatalf("cacheFilePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	oldData, err := json.Marshal(oldMapCache{
		RootPath:  root,
		UpdatedAt: modTime,
		Files: map[string]*oldCacheFile{
			"main.go": {
				ModTime:   modTime,
				LineCount: 42,
				Symbols: []oldSymbol{
					{Line: 3, Signature: "func Build() error"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(cachePath, oldData, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := loadMapCache(root)
	if err != nil {
		t.Fatalf("loadMapCache() error = %v", err)
	}
	oldLoaded := loaded.Files["main.go"].Symbols[0]
	if oldLoaded.Name != "" || oldLoaded.Kind != "" || oldLoaded.EndLine != 0 || oldLoaded.Exported {
		t.Fatalf("old cache symbol should load zero values for new fields: %+v", oldLoaded)
	}

	cache := &MapCache{
		RootPath: root,
		Files: map[string]*CacheFile{
			"main.go": {
				ModTime:   modTime,
				LineCount: 42,
				Symbols: []Symbol{
					{
						Name:      "Build",
						Kind:      "function",
						Line:      3,
						EndLine:   7,
						Signature: "func Build() error",
						Exported:  true,
					},
				},
			},
		},
	}

	if err := saveMapCache(root, cache); err != nil {
		t.Fatalf("saveMapCache() error = %v", err)
	}

	reloaded, err := loadMapCache(root)
	if err != nil {
		t.Fatalf("loadMapCache() error = %v", err)
	}
	got := reloaded.Files["main.go"].Symbols[0]
	if got.Name != "Build" {
		t.Fatalf("Name = %q, want Build", got.Name)
	}
	if got.Kind != "function" {
		t.Fatalf("Kind = %q, want function", got.Kind)
	}
	if got.EndLine != 7 {
		t.Fatalf("EndLine = %d, want 7", got.EndLine)
	}
	if !got.Exported {
		t.Fatal("Exported = false, want true")
	}
}
