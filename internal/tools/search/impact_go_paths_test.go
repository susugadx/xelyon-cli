package search

import (
	"path/filepath"
	"testing"
)

func TestNormalizeStructuredGoImpactScope_DerivesDefinitionPathHintAndPreservesEvidenceGlob(t *testing.T) {
	dir := t.TempDir()
	opts := SearchOptions{
		Pattern:     "Run",
		Intent:      "impact",
		Path:        dir,
		FilePattern: "packages/app/src/**/*.go",
	}

	scope, ok := normalizeStructuredGoImpactScope(opts)
	if !ok {
		t.Fatal("expected scoped Go glob to normalize")
	}
	want := filepath.Join(dir, "packages", "app", "src")
	if scope.Definition.Path != want {
		t.Fatalf("definition path = %q, want %q", scope.Definition.Path, want)
	}
	if scope.Definition.FilePattern != "" || scope.Definition.FileType != "" {
		t.Fatalf("definition file filter = (%q, %q), want cleared", scope.Definition.FileType, scope.Definition.FilePattern)
	}
	if scope.Evidence.Path != dir {
		t.Fatalf("evidence path = %q, want original path %q", scope.Evidence.Path, dir)
	}
	if scope.Evidence.FilePattern != "packages/app/src/**/*.go" || scope.Evidence.FileType != "" {
		t.Fatalf("evidence file filter = (%q, %q), want (empty, packages/app/src/**/*.go)", scope.Evidence.FileType, scope.Evidence.FilePattern)
	}
}

func TestNormalizeStructuredGoImpactScope_DerivesDefinitionPathHintFromWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	opts := newGoImpactWorkspaceSearchOptions(
		root,
		root,
		filepath.Join(root, "packages", "app"),
		"Run",
		"packages/app/src/**/*.go",
	)

	scope, ok := normalizeStructuredGoImpactScope(opts)
	if !ok {
		t.Fatal("expected scoped Go glob to normalize")
	}
	want := filepath.Join(root, "packages", "app", "src")
	if scope.Definition.Path != want {
		t.Fatalf("definition path = %q, want workspace-root scoped path %q", scope.Definition.Path, want)
	}
	if scope.Evidence.Path != opts.Path {
		t.Fatalf("evidence path = %q, want original path %q", scope.Evidence.Path, opts.Path)
	}
	if scope.Evidence.FilePattern != "packages/app/src/**/*.go" {
		t.Fatalf("evidence file pattern = %q, want original scoped glob", scope.Evidence.FilePattern)
	}
}

func TestNormalizeStructuredGoImpactScope_DerivesRelativeGlobPathHintFromSearchTarget(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "pkg")
	opts := newGoImpactWorkspaceSearchOptions(root, cwd, ".", "Run", "src/**/*.go")

	scope, ok := normalizeStructuredGoImpactScope(opts)
	if !ok {
		t.Fatal("expected scoped Go glob to normalize")
	}
	want := filepath.Join(cwd, "src")
	if scope.Definition.Path != want {
		t.Fatalf("definition path = %q, want search-target scoped path %q", scope.Definition.Path, want)
	}
	if scope.Evidence.Path != opts.Path {
		t.Fatalf("evidence path = %q, want original path %q", scope.Evidence.Path, opts.Path)
	}
	if scope.Evidence.FilePattern != "src/**/*.go" {
		t.Fatalf("evidence file pattern = %q, want original relative scoped glob", scope.Evidence.FilePattern)
	}
}

func TestNormalizeStructuredGoImpactScope_RejectsTargetRelativeGlobForAbsoluteWorkspaceSubdir(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	opts := newGoImpactWorkspaceSearchOptions(root, pkgDir, pkgDir, "Run", "src/**/*.go")

	if scope, ok := normalizeStructuredGoImpactScope(opts); ok {
		t.Fatalf("normalize ok = true with scope %+v, want false because file pattern is workspace-relative and excludes absolute subdir target", scope)
	}
}

func TestNormalizeStructuredGoImpactScope_FilePatternContract(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantOK  bool
	}{
		{name: "basename go glob", pattern: "*.go", wantOK: true},
		{name: "workspace go glob", pattern: "**/*.go", wantOK: true},
		{name: "static scoped go glob", pattern: "packages/app/src/**/*.go", wantOK: true},
		{name: "narrow basename go glob", pattern: "*_test.go", wantOK: false},
		{name: "non recursive scoped go glob", pattern: "packages/app/src/*.go", wantOK: false},
		{name: "wildcard directory scoped go glob", pattern: "packages/*/src/**/*.go", wantOK: false},
		{name: "non go glob", pattern: "*.ts", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := normalizeStructuredGoImpactScope(SearchOptions{
				Pattern:     "Run",
				Intent:      "impact",
				Path:        ".",
				FilePattern: tt.pattern,
			})
			if ok != tt.wantOK {
				t.Fatalf("normalize ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}
