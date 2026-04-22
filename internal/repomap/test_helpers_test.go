package repomap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func setProjectMapTestHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func requireRipgrep(t *testing.T) {
	t.Helper()
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}
}

func writeProjectMapTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", relPath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", relPath, err)
	}
}

func buildProjectMapForTest(t *testing.T, root string, maxTokens int, ignoreDirs ...string) *ProjectMap {
	t.Helper()
	setProjectMapTestHome(t)
	pm := NewProjectMap(root, maxTokens, ignoreDirs...)
	if err := pm.Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	return pm
}

func buildProjectManifestForTest(t *testing.T, root string, maxTokens int, ignoreDirs ...string) *ProjectMap {
	t.Helper()
	setProjectMapTestHome(t)
	pm := NewProjectMap(root, maxTokens, ignoreDirs...)
	if err := pm.BuildManifest(); err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	return pm
}

func findFileEntry(t *testing.T, pm *ProjectMap, relPath string) *FileEntry {
	t.Helper()
	for _, file := range pm.Files {
		if file.Path == filepath.ToSlash(relPath) {
			return file
		}
	}
	t.Fatalf("file entry %s not found", relPath)
	return nil
}

func findSymbol(t *testing.T, file *FileEntry, name string) Symbol {
	t.Helper()
	for _, symbol := range file.Symbols {
		if symbol.Name == name {
			return symbol
		}
	}
	t.Fatalf("symbol %s not found in %s", name, file.Path)
	return Symbol{}
}

func signatures(file *FileEntry) []string {
	result := make([]string, 0, len(file.Symbols))
	for _, symbol := range file.Symbols {
		result = append(result, symbol.Signature)
	}
	return result
}
