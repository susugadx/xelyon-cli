package repomap

import (
	"strings"
	"testing"
)

func TestBuildManifest_GenerateManifest(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "README.md", "# test\n")
	writeProjectMapTestFile(t, root, "main.go", "package main\n")
	writeProjectMapTestFile(t, root, "internal/agent/compress.go", "package agent\n")
	writeProjectMapTestFile(t, root, "internal/config/project.go", "package config\n")

	pm := buildProjectManifestForTest(t, root, 4000)
	output := pm.GenerateManifest([]string{"internal/agent"})

	if !strings.Contains(output, "Top-level directories:") {
		t.Fatalf("manifest should contain top-level directories:\n%s", output)
	}
	if !strings.Contains(output, "- internal/") {
		t.Fatalf("manifest should contain internal/ directory:\n%s", output)
	}
	if !strings.Contains(output, "Top-level files:") || !strings.Contains(output, "- README.md") {
		t.Fatalf("manifest should contain top-level files:\n%s", output)
	}
	if !strings.Contains(output, "Priority files:") || !strings.Contains(output, "internal/agent/compress.go") {
		t.Fatalf("manifest should contain prioritized file:\n%s", output)
	}
	if strings.Contains(output, "func ") {
		t.Fatalf("manifest should stay lightweight without symbols:\n%s", output)
	}
}

func TestGenerateManifest_NilPrioritizedPathsReturnsLightweightManifest(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "README.md", "# test\n")
	writeProjectMapTestFile(t, root, "main.go", "package main\n\nfunc Build() {}\n")
	writeProjectMapTestFile(t, root, "internal/agent/compress.go", "package agent\n\nfunc Compress() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	full := pm.Generate()
	manifest := pm.GenerateManifest(nil)

	if manifest == "" {
		t.Fatal("expected non-empty manifest")
	}
	if !strings.Contains(manifest, "Top-level directories:") {
		t.Fatalf("expected top-level directories in manifest:\n%s", manifest)
	}
	if !strings.Contains(manifest, "Top-level files:") {
		t.Fatalf("expected top-level files in manifest:\n%s", manifest)
	}
	if strings.Contains(manifest, "Priority files:") {
		t.Fatalf("stable base manifest should not include priority files:\n%s", manifest)
	}
	if strings.Contains(manifest, "func Build()") || strings.Contains(manifest, "func Compress()") {
		t.Fatalf("manifest should omit symbol dump:\n%s", manifest)
	}
	if len(manifest) >= len(full) {
		t.Fatalf("expected manifest to stay lighter than full map\nmanifest:\n%s\n\nfull:\n%s", manifest, full)
	}
}
