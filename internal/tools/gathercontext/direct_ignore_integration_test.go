package gathercontext

import (
	"path/filepath"
	"testing"
)

func TestGatherContext_ScopedExactLookupIgnoresDependencyTrees(t *testing.T) {
	tests := []struct {
		name          string
		ignoredPath   string
		projectConfig string
	}{
		{name: "node_modules", ignoredPath: filepath.Join("node_modules", "dep", "package.json")},
		{name: "vendor", ignoredPath: filepath.Join("vendor", "dep", "package.json")},
		{name: "git", ignoredPath: filepath.Join(".git", "hooks", "package.json")},
		{name: "project config", ignoredPath: filepath.Join("generated-cache", "package.json"), projectConfig: "ignore:\n  patterns:\n    - generated-cache\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			withGatherContextWorkingDir(t, root)

			files := map[string]string{
				filepath.Join(root, "package.json"): "{\"name\":\"root-app\"}\n",
				filepath.Join(root, tt.ignoredPath): "{\"name\":\"dep\"}\n",
			}
			if tt.projectConfig != "" {
				files[filepath.Join(root, "xelyon.yaml")] = tt.projectConfig
			}
			writeGatherContextFiles(t, files)

			result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
				"query":       "package.json",
				"path":        ".",
				"file_filter": "json",
			})
			assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: package.json", `"root-app"`)
			assertGatherContextExcludesAll(t, result, `"dep"`, filepath.ToSlash(tt.ignoredPath))
		})
	}
}

func TestGatherContext_ExactIgnoredTreeFileLookupUsesDirectRead(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "xelyon.yaml"):                         "ignore:\n  patterns:\n    - node_modules\n",
		filepath.Join(root, "package.json"):                        "{\"name\":\"root-app\"}\n",
		filepath.Join(root, "node_modules", "dep", "package.json"): "{\"name\":\"dep\"}\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       filepath.Join("node_modules", "dep", "package.json"),
		"file_filter": "json",
	})
	assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: node_modules/dep/package.json", `"dep"`)
	assertGatherContextExcludesAll(t, result, "No matches found", "Route: Auto search", `"root-app"`)
}

func TestGatherContext_ExplicitIgnoredTreeDirectoryListingUsesDirectRoute(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "xelyon.yaml"):                         "ignore:\n  patterns:\n    - node_modules\n",
		filepath.Join(root, "node_modules", "dep", "package.json"): "{\"name\":\"dep\"}\n",
		filepath.Join(root, "node_modules", "dep", "README.md"):    "# dep\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       filepath.Join("node_modules", "dep") + string(filepath.Separator),
		"file_filter": "json",
	})
	assertGatherContextContainsAll(t, result, "Route: Directory listing", "package.json")
	assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found", "README.md")
}

func TestGatherContext_ExactIgnoredTreeRangeLookupUsesDirectRead(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "xelyon.yaml"):                         "ignore:\n  patterns:\n    - node_modules\n",
		filepath.Join(root, "node_modules", "dep", "package.json"): "{\n  \"name\": \"dep\"\n}\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       filepath.Join("node_modules", "dep", "package.json:2-2"),
		"file_filter": "json",
	})
	assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: node_modules/dep/package.json:2-2", `2:   "name": "dep"`)
	assertGatherContextExcludesAll(t, result, "1: {", "3: }", "No matches found", "Route: Auto search")
}
