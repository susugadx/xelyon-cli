package search

import (
	"path/filepath"
	"testing"
)

func TestStructuredImpactSemanticReferenceFilterOptionsWidensDirectFileScopeOnly(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.ts": "export function buildUser(id: string) { return id }\n",
	})
	subdir := filepath.Join(dir, "packages", "app")

	got := structuredImpactSemanticReferenceFilterOptions(SearchOptions{
		Path:               filepath.Join(dir, "packages", "app", "src", "build.ts"),
		InvocationCWD:      subdir,
		ProjectMapRootPath: dir,
	})

	if got.Path != dir {
		t.Fatalf("Path = %q, want direct-file filter widened to workspace root %q", got.Path, dir)
	}
	if got.InvocationCWD != subdir {
		t.Fatalf("InvocationCWD = %q, want adapter base preserved %q", got.InvocationCWD, subdir)
	}
	if got.ProjectMapRootPath != dir {
		t.Fatalf("ProjectMapRootPath = %q, want workspace root %q", got.ProjectMapRootPath, dir)
	}
}

func TestStructuredImpactSemanticReferenceFilterOptionsPreservesDirectoryScope(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"other/app.ts": "buildUser('out of scope')\n",
	})
	scope := filepath.Join(dir, "src")

	got := structuredImpactSemanticReferenceFilterOptions(SearchOptions{
		Path:          scope,
		InvocationCWD: dir,
	})

	if got.Path != scope {
		t.Fatalf("Path = %q, want directory scope preserved %q", got.Path, scope)
	}
	if got.InvocationCWD != dir {
		t.Fatalf("InvocationCWD = %q, want unchanged %q", got.InvocationCWD, dir)
	}
}
