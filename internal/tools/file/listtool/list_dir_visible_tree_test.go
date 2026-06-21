package listtool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestBuildVisibleListDirTree_FilteredProjectMapIndexKeepsShallowReadDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "go-subtree", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go-subtree", "nested", "main.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxRootDirsShown; i++ {
		dir := filepath.Join(root, "js-only-"+string(rune('a'+i)))
		if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "nested", "index.js"), []byte("export const value = 1;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	readCounts := make(map[string]int)
	readDir := func(path string) ([]os.DirEntry, error) {
		readCounts[path]++
		return os.ReadDir(path)
	}

	filterIndex := buildListDirFilterIndex(root, root, root, pathmatch.NewMatcher(nil), "go", &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{Path: filepath.ToSlash(filepath.Join("go-subtree", "nested", "main.go"))},
		},
	})
	tree := buildVisibleListDirTreeWithReader(root, root, root, "", 1, pathmatch.NewMatcher(nil), "go", filterIndex, true, readDir)
	if len(tree.dirs) != 1 || tree.dirs[0].relPath != "go-subtree/" {
		t.Fatalf("expected only go-subtree to stay visible, got %#v", tree.dirs)
	}
	if len(readCounts) != 1 || readCounts[root] != 1 {
		t.Fatalf("expected filtered shallow build to read only root once, got %#v", readCounts)
	}
}

func TestBuildVisibleListDirTree_DepthOneWithoutFilterDoesNotRecurse(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "nested", "main.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	readCounts := make(map[string]int)
	readDir := func(path string) ([]os.DirEntry, error) {
		readCounts[path]++
		return os.ReadDir(path)
	}

	tree := buildVisibleListDirTreeWithReader(root, root, root, "", 1, pathmatch.NewMatcher(nil), "", nil, true, readDir)
	if len(readCounts) != 1 || readCounts[root] != 1 {
		t.Fatalf("expected shallow unfiltered build to read only root once, got %#v", readCounts)
	}
	if len(tree.dirs) != 1 || tree.dirs[0].name != "a" {
		t.Fatalf("unexpected immediate dirs: %#v", tree.dirs)
	}
	if len(tree.expandedDirs) != 0 {
		t.Fatalf("expected no subtree materialization at depth=1, got %#v", tree.expandedDirs)
	}
}
