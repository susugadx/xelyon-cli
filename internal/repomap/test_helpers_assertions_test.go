package repomap

import (
	"path/filepath"
	"testing"
)

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
