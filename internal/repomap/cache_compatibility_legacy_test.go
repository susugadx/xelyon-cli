package repomap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSymbolCache_LoadLegacyFieldsAsZeroValues(t *testing.T) {
	setProjectMapTestHome(t)

	root := filepath.Join(t.TempDir(), "repo")
	modTime := time.Unix(123, 0).UTC()

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
}
