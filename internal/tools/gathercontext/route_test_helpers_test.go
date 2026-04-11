package gathercontext

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func setupRoutePlanFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	for _, dir := range []string{
		filepath.Join(root, "internal", "tools"),
		filepath.Join(root, "pkg"),
		filepath.Join(root, "config"),
		filepath.Join(root, "node_modules", "dep"),
		filepath.Join(root, "pkg", "nested"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "sample.go"):                           "package main\n",
		filepath.Join(root, "nested.go"):                           "package main\n",
		filepath.Join(root, "target.go"):                           "package main\nconst selected = \"root\"\n",
		filepath.Join(root, "README.md"):                           "# root\n",
		filepath.Join(root, "pkg", "target.go"):                    "package pkg\nconst selected = \"subdir\"\n",
		filepath.Join(root, "pkg", "impl.go"):                      "package pkg\n",
		filepath.Join(root, "pkg", "README.md"):                    "# pkg\n",
		filepath.Join(root, "pkg", "impl_test.go"):                 "package pkg\n",
		filepath.Join(root, "pkg", "nested", "worker.go"):          "package nested\n",
		filepath.Join(root, "pkg", "nested", "README.md"):          "nested docs\n",
		filepath.Join(root, "package.json"):                        "{\"name\":\"root-app\"}\n",
		filepath.Join(root, "node_modules", "dep", "package.json"): "{\"name\":\"dep\"}\n",
	})

	return root
}

func newRoutePlanExecCtx(root string, opts ...gatherContextExecCtxOption) tools.ExecutionContext {
	return newGatherContextExecCtx(root, opts...)
}
