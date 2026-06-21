package listtool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestBuildListDirFilterIndex_ProjectMapBuildsVisibleAncestors(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	docsDir := filepath.Join(pkgDir, "docs")
	nestedDir := filepath.Join(pkgDir, "nested")
	for _, dir := range []string{docsDir, nestedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	index := buildListDirFilterIndex(pkgDir, root, root, pathmatch.NewMatcher(nil), "go", &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{Path: filepath.ToSlash(filepath.Join("pkg", "nested", "main.go"))},
			{Path: filepath.ToSlash(filepath.Join("pkg", "docs", "README.md"))},
		},
	})

	if !index.hasVisibleDir(nestedDir) {
		t.Fatalf("expected nested dir to stay visible for matching project map file")
	}
	if index.hasVisibleDir(docsDir) {
		t.Fatalf("expected docs dir to stay hidden for non-matching project map file")
	}
}

func TestBuildListDirFilterIndex_PartialProjectMapCompletesByWalk(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	nestedDir := filepath.Join(pkgDir, "nested")
	docsDir := filepath.Join(pkgDir, "docs")
	for _, dir := range []string{nestedDir, docsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "a.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "b.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	index := buildListDirFilterIndex(pkgDir, root, root, pathmatch.NewMatcher(nil), "go", &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{Path: filepath.ToSlash(filepath.Join("pkg", "a.go"))},
		},
	})

	if !index.hasVisibleDir(nestedDir) {
		t.Fatalf("expected filesystem walk to restore nested dir hidden by partial project map")
	}
	if index.hasVisibleDir(docsDir) {
		t.Fatalf("expected non-matching docs dir to stay hidden")
	}
}

func TestBuildListDirFilterIndex_ScanFallbackHonorsIgnoreAndFilter(t *testing.T) {
	root := t.TempDir()
	visibleDir := filepath.Join(root, "pkg", "nested")
	ignoredDir := filepath.Join(root, "node_modules", "dep")
	for _, dir := range []string{visibleDir, ignoredDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(visibleDir, "main.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredDir, "ignored.go"), []byte("package dep\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	index := buildListDirFilterIndex(root, root, root, pathmatch.NewMatcher([]string{"node_modules"}), "go", nil)
	if !index.hasVisibleDir(filepath.Join(root, "pkg")) {
		t.Fatalf("expected pkg subtree to stay visible")
	}
	if index.hasVisibleDir(filepath.Join(root, "node_modules")) {
		t.Fatalf("expected ignored subtree to stay hidden")
	}
}
