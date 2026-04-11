package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type recordingDirCache struct {
	dirs    map[string]string
	getKeys []string
	setKeys []string
}

func (c *recordingDirCache) GetFile(string) (string, bool)              { return "", false }
func (c *recordingDirCache) SetFile(string, string)                     {}
func (c *recordingDirCache) InvalidateFile(string)                      {}
func (c *recordingDirCache) Clear()                                     {}
func (c *recordingDirCache) GetSearch(string, string) (string, bool)    { return "", false }
func (c *recordingDirCache) SetSearch(string, string, string, []string) {}
func (c *recordingDirCache) ClearSearchCache()                          {}
func (c *recordingDirCache) InvalidateSearchCacheForFile(string)        {}
func (c *recordingDirCache) InvalidateDir(string)                       {}
func (c *recordingDirCache) GetDir(path string) (string, bool) {
	c.getKeys = append(c.getKeys, path)
	v, ok := c.dirs[path]
	return v, ok
}
func (c *recordingDirCache) SetDir(path, result string) {
	c.setKeys = append(c.setKeys, path)
	c.dirs[path] = result
}

func TestExecuteListDirWithRuntime_CacheKeySeparatesFilterRoot(t *testing.T) {
	root := t.TempDir()
	chdirForListDirTest(t, root)

	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "main.js"), []byte("export const main = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "other.ts"), []byte("export const other = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &recordingDirCache{dirs: make(map[string]string)}

	withWorkspaceRoot := executeListDirWithRuntime(config.DefaultConfig(), cache, pkgDir, 1, nil, "pkg/*.js", root, nil, "state-a")
	withoutWorkspaceRoot := executeListDirWithRuntime(config.DefaultConfig(), cache, pkgDir, 1, nil, "pkg/*.js", "", nil, "state-b")

	if !strings.Contains(withWorkspaceRoot, "main.js") {
		t.Fatalf("expected workspace-root listing to include main.js, got:\n%s", withWorkspaceRoot)
	}
	if strings.Contains(withoutWorkspaceRoot, "main.js") {
		t.Fatalf("expected scope-relative listing to exclude workspace-relative glob hit, got:\n%s", withoutWorkspaceRoot)
	}
	if len(cache.setKeys) != 2 {
		t.Fatalf("expected distinct cache entries for different filter roots, got set keys %v", cache.setKeys)
	}
	if cache.setKeys[0] == cache.setKeys[1] {
		t.Fatalf("expected cache keys to differ by filter root, got %v", cache.setKeys)
	}
}

func TestExecuteListDirWithRuntime_FilteredCacheRequiresProjectMapStateKey(t *testing.T) {
	root := t.TempDir()
	chdirForListDirTest(t, root)

	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &recordingDirCache{dirs: make(map[string]string)}
	result := executeListDirWithRuntime(config.DefaultConfig(), cache, pkgDir, 1, nil, "go", root, nil, "")
	if !strings.Contains(result, "main.go") {
		t.Fatalf("expected filtered listing to still run without cache state, got:\n%s", result)
	}
	if len(cache.getKeys) != 0 || len(cache.setKeys) != 0 {
		t.Fatalf("expected filtered listing without state key to skip shared cache, got get=%v set=%v", cache.getKeys, cache.setKeys)
	}
}

func TestExecuteListDirWithRuntime_FilteredCacheRefreshesOnProjectMapStateChange(t *testing.T) {
	root := t.TempDir()
	chdirForListDirTest(t, root)

	pkgDir := filepath.Join(root, "pkg")
	nestedDir := filepath.Join(pkgDir, "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cache := &recordingDirCache{dirs: make(map[string]string)}
	before := executeListDirWithRuntime(config.DefaultConfig(), cache, pkgDir, 1, nil, "go", root, &repomap.ProjectMap{
		RootPath: root,
		Files:    nil,
	}, "state-before")
	if strings.Contains(before, "nested/") {
		t.Fatalf("expected no visible nested dir before matching file exists, got:\n%s", before)
	}

	if err := os.WriteFile(filepath.Join(nestedDir, "main.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	after := executeListDirWithRuntime(config.DefaultConfig(), cache, pkgDir, 1, nil, "go", root, &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{Path: filepath.ToSlash(filepath.Join("pkg", "nested", "main.go"))},
		},
	}, "state-after")
	if !strings.Contains(after, "nested/") {
		t.Fatalf("expected state-key change to refresh filtered listing, got:\n%s", after)
	}
	if len(cache.setKeys) != 2 {
		t.Fatalf("expected two cached filtered listings with distinct states, got %v", cache.setKeys)
	}
	if cache.setKeys[0] == cache.setKeys[1] {
		t.Fatalf("expected filtered cache key to include project map state, got %v", cache.setKeys)
	}
}

func TestExecuteListDirWithRuntime_FilteredCacheRefreshesOnProjectMapStateRemoval(t *testing.T) {
	root := t.TempDir()
	chdirForListDirTest(t, root)

	pkgDir := filepath.Join(root, "pkg")
	nestedDir := filepath.Join(pkgDir, "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "main.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &recordingDirCache{dirs: make(map[string]string)}
	before := executeListDirWithRuntime(config.DefaultConfig(), cache, pkgDir, 1, nil, "go", root, &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{Path: filepath.ToSlash(filepath.Join("pkg", "nested", "main.go"))},
		},
	}, "state-before")
	if !strings.Contains(before, "nested/") {
		t.Fatalf("expected matching descendant to be visible before removal, got:\n%s", before)
	}

	if err := os.Remove(filepath.Join(nestedDir, "main.go")); err != nil {
		t.Fatal(err)
	}

	after := executeListDirWithRuntime(config.DefaultConfig(), cache, pkgDir, 1, nil, "go", root, &repomap.ProjectMap{
		RootPath: root,
		Files:    nil,
	}, "state-after")
	if strings.Contains(after, "nested/") {
		t.Fatalf("expected state-key change to drop removed descendant from filtered listing, got:\n%s", after)
	}
	if len(cache.setKeys) != 2 {
		t.Fatalf("expected two cached filtered listings with distinct states, got %v", cache.setKeys)
	}
	if cache.setKeys[0] == cache.setKeys[1] {
		t.Fatalf("expected filtered cache key to change after removal, got %v", cache.setKeys)
	}
}

func TestExecuteListDirWithRuntime_FilteredListingCompletesPartialProjectMapByWalk(t *testing.T) {
	root := t.TempDir()
	chdirForListDirTest(t, root)

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

	result := executeListDirWithRuntime(config.DefaultConfig(), nil, pkgDir, 1, nil, "go", root, &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{Path: filepath.ToSlash(filepath.Join("pkg", "a.go"))},
		},
	}, "state-partial")

	if !strings.Contains(result, "nested/") {
		t.Fatalf("expected filtered listing to keep nested dir visible when project map is partial, got:\n%s", result)
	}
	if strings.Contains(result, "docs/") {
		t.Fatalf("expected non-matching docs dir to stay hidden, got:\n%s", result)
	}
}

func TestExecuteListDirWithRuntimeMode_IgnoreBypassChangesVisibilityAndCacheKey(t *testing.T) {
	root := t.TempDir()
	chdirForListDirTest(t, root)

	targetDir := filepath.Join(root, "node_modules", "dep")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - node_modules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "package.json"), []byte("{\"name\":\"dep\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# dep\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &recordingDirCache{dirs: make(map[string]string)}
	normal := executeListDirWithRuntimeMode(config.DefaultConfig(), cache, targetDir, 1, nil, "json", root, nil, "state-a", listDirApplyIgnores)
	bypass := executeListDirWithRuntimeMode(config.DefaultConfig(), cache, targetDir, 1, nil, "json", root, nil, "state-a", listDirBypassIgnores)

	if strings.Contains(normal, "package.json") {
		t.Fatalf("expected normal list_dir to honor ignores for ignored-tree target, got:\n%s", normal)
	}
	if !strings.Contains(bypass, "package.json") {
		t.Fatalf("expected bypassed list_dir to include ignored-tree package.json, got:\n%s", bypass)
	}
	if strings.Contains(bypass, "README.md") {
		t.Fatalf("expected bypassed list_dir to still honor file_filter, got:\n%s", bypass)
	}
	if len(cache.setKeys) != 2 {
		t.Fatalf("expected separate cache entries for ignore modes, got %v", cache.setKeys)
	}
	if cache.setKeys[0] == cache.setKeys[1] {
		t.Fatalf("expected ignore mode to affect cache key, got %v", cache.setKeys)
	}
}

var _ tools.ToolCacheInterface = (*recordingDirCache)(nil)
