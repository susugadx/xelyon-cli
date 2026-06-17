package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func testProjectMapHasFile(agent *Agent, relPath string) bool {
	if agent == nil || agent.projectMap == nil {
		return false
	}
	relPath = filepath.ToSlash(relPath)
	for _, file := range agent.projectMap.Files {
		if file != nil && file.Path == relPath {
			return true
		}
	}
	return false
}

func markProjectMapTestRoot(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "xelyon.yaml"), []byte("context: test\n"), 0644); err != nil {
		t.Fatalf("WriteFile(xelyon.yaml) error = %v", err)
	}
}
