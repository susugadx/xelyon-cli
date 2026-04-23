package repomap

import (
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
