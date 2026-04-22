package repomap

import (
	"strings"
	"testing"
)

func TestBuildManifest_RespectsFileGlobIgnorePatterns(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "assets/app.min.js", "console.log('skip')\n")
	writeProjectMapTestFile(t, root, "assets/app.js", "console.log('keep')\n")

	pm := buildProjectManifestForTest(t, root, 4000, "*.min.js")
	output := pm.GenerateManifest([]string{"assets"})

	if strings.Contains(output, "app.min.js") {
		t.Fatalf("file glob ignore should exclude app.min.js:\n%s", output)
	}
	if !strings.Contains(output, "app.js") {
		t.Fatalf("expected non-ignored file to remain:\n%s", output)
	}
}
