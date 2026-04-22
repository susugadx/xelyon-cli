package repomap

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	states, err := pm.buildFileStates([]string{"main.go"})
	if err != nil {
		t.Fatalf("buildFileStates() error = %v", err)
	}
	states = applyCachePolicyToStates(states, cache)
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
	states, err := pm.buildFileStates([]string{"main.go", "extra.go"})
	if err != nil {
		t.Fatalf("buildFileStates() error = %v", err)
	}
	states = applyCachePolicyToStates(states, cache)
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
