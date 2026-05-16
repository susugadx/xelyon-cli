package search

import (
	"path/filepath"
	"testing"
)

func TestStructuredGoImpactEvidenceFileInScope_ResolvesRelativeEvidenceFromSearchTarget(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"pkg/src/use.go":   "package src\n\nfunc UseTarget() { Target() }\n",
		"other/src/use.go": "package other\n\nfunc UseTarget() { Target() }\n",
	})
	cwd := filepath.Join(root, "pkg")
	opts := newGoImpactWorkspaceSearchOptions(root, cwd, ".", "Target", "src/**/*.go")

	if !structuredGoImpactEvidenceFileInScope("src/use.go", "", opts) {
		t.Fatal("relative in-scope evidence should resolve against the search target")
	}
	if structuredGoImpactEvidenceFileInScope("other/src/use.go", "", opts) {
		t.Fatal("relative sibling evidence should remain out of scope")
	}
}

func TestStructuredGoImpactEvidenceFileInScope_NormalizesWorkspaceGlobDisplayPath(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"packages/app/src/use.go":   "package app\n\nfunc UseTarget() { Target() }\n",
		"packages/other/src/use.go": "package other\n\nfunc UseTarget() { Target() }\n",
	})
	appDir := filepath.Join(root, "packages", "app")
	opts := newGoImpactWorkspaceSearchOptions(root, appDir, appDir, "Target", "packages/app/src/**/*.go")

	if !structuredGoImpactEvidenceFileInScope("src/use.go", filepath.Join(appDir, "src", "use.go"), opts) {
		t.Fatal("CWD-relative evidence should match workspace-relative scoped glob after display path normalization")
	}
	if structuredGoImpactEvidenceFileInScope("src/use.go", filepath.Join(root, "packages", "other", "src", "use.go"), opts) {
		t.Fatal("same CWD-relative display path with sibling resolved path should remain out of scope")
	}
}

func TestStructuredGoImpactEvidenceFileInScope_RejectsTargetRelativeDisplayForAbsoluteWorkspaceSubdir(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"pkg/src/use.go": "package src\n\nfunc UseTarget() { Target() }\n",
	})
	pkgDir := filepath.Join(root, "pkg")
	opts := newGoImpactWorkspaceSearchOptions(root, pkgDir, pkgDir, "Target", "src/**/*.go")

	if structuredGoImpactEvidenceFileInScope("src/use.go", filepath.Join(pkgDir, "src", "use.go"), opts) {
		t.Fatal("CWD-relative display path should not bypass a workspace-relative glob that excludes the absolute subdir target")
	}
}
