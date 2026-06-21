package directquery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveImplicitDirectFileQuery(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n\tgo test ./...\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	targets, ok := resolveImplicitDirectFileQuery(execCtx, "Makefile")
	if !ok {
		t.Fatal("expected Makefile to resolve as implicit direct file query")
	}
	if len(targets) != 1 || targets[0].Kind != directQueryTargetFile {
		t.Fatalf("unexpected implicit targets: %+v", targets)
	}

	if _, ok := resolveImplicitDirectFileQuery(execCtx, "config"); ok {
		t.Fatal("expected directory name to stay out of implicit file route")
	}
	if _, ok := resolveImplicitDirectFileQuery(execCtx, "./Makefile"); ok {
		t.Fatal("expected explicit path-like query to be handled by explicit route")
	}
}
