package repomap

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSymbolCache_SaveLoadNewFieldsRoundTrip(t *testing.T) {
	setProjectMapTestHome(t)

	root := filepath.Join(t.TempDir(), "repo")
	modTime := time.Unix(123, 0).UTC()

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
