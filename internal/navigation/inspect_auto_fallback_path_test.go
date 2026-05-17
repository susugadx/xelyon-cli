package navigation

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestResolveInspectSymbolAuto_FallbackReferenceSearchPathScopesRipgrep(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep is not available")
	}

	dir := setupTestGoFiles(t, map[string]string{
		"app/target.go": "package app\n\nfunc Target() string { return \"app\" }\n",
		"app/use.go":    "package app\n\nfunc UseTarget() string { return Target() }\n",
		"other/use.go":  "package other\n\nfunc UseTarget() string { return Target() }\n",
	})

	result, _, status := ResolveInspectSymbolAuto("Target", filepath.Join(dir, "app"), InspectSymbolAutoOptions{
		Budget:                      FullBudget,
		FallbackReferenceSearchPath: filepath.Join(dir, "app"),
	})

	if status != SymbolAutoSingle {
		t.Fatalf("status = %s, want %s", status, SymbolAutoSingle)
	}
	if len(result.Callers) != 1 {
		t.Fatalf("callers = %+v, want only scoped app caller", result.Callers)
	}
	if got := result.Callers[0].File; got != "app/use.go" {
		t.Fatalf("caller file = %q, want app/use.go", got)
	}
	assertReferencesExcludeFile(t, result.Callers, "other/use.go")
	assertReferencesExcludeFile(t, result.Refs, "other/use.go")
}

func assertReferencesExcludeFile(t *testing.T, refs []Reference, file string) {
	t.Helper()

	for _, ref := range refs {
		if ref.File == file {
			t.Fatalf("references should not include %s: %+v", file, ref)
		}
	}
}
