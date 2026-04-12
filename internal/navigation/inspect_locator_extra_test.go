package navigation

import (
	"path/filepath"
	"testing"
)

func TestInspectLocatorPathHelpers(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	if got := resolveInspectLocatorPath("", tmp); got != "" {
		t.Fatalf("resolveInspectLocatorPath(empty) = %q, want empty", got)
	}
	if got := resolveInspectLocatorPath("pkg/file.go", ""); got != "" {
		t.Fatalf("resolveInspectLocatorPath(no root) = %q, want empty", got)
	}

	absFile := filepath.Join(tmp, "pkg", "file.go")
	if got := resolveInspectLocatorPath(absFile, filepath.Join(tmp, "other")); got != filepath.Clean(absFile) {
		t.Fatalf("resolveInspectLocatorPath(abs) = %q, want %q", got, filepath.Clean(absFile))
	}

	wantResolved := filepath.Join(tmp, "pkg", "file.go")
	if got := resolveInspectLocatorPath("pkg/file.go", tmp); got != wantResolved {
		t.Fatalf("resolveInspectLocatorPath(rel) = %q, want %q", got, wantResolved)
	}

	if got := cleanInspectResolvedPath("  ./pkg/../file.go  "); got != filepath.Join(tmp, "file.go") {
		t.Fatalf("cleanInspectResolvedPath(rel) = %q, want %q", got, filepath.Join(tmp, "file.go"))
	}
	if got := cleanInspectResolvedPath("   "); got != "" {
		t.Fatalf("cleanInspectResolvedPath(blank) = %q, want empty", got)
	}
}

func TestNewInspectRelatedLocator_ResolvesFallbackAndCleansExplicitPath(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	loc := newInspectRelatedLocator("pkg/file.go", "", tmp, 3, 7, "Build")
	if loc.ResolvedPath != filepath.Join(tmp, "pkg", "file.go") {
		t.Fatalf("newInspectRelatedLocator(blank resolvedPath).ResolvedPath = %q, want %q", loc.ResolvedPath, filepath.Join(tmp, "pkg", "file.go"))
	}

	loc = newInspectRelatedLocator("pkg/file.go", " ./pkg/../other.go ", tmp, 4, 0, "Build")
	if loc.ResolvedPath != filepath.Join(tmp, "other.go") {
		t.Fatalf("newInspectRelatedLocator(explicit resolvedPath).ResolvedPath = %q, want %q", loc.ResolvedPath, filepath.Join(tmp, "other.go"))
	}
	if loc.FilePath != "pkg/file.go" || loc.Line != 4 || loc.Name != "Build" {
		t.Fatalf("locator = %+v, want file/line/name preserved", loc)
	}
}
