package filefilter

import (
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantType    string
		wantPattern string
	}{
		{name: "token", filter: "  RS  ", wantType: "rs"},
		{name: "leading dot", filter: " .json ", wantType: "json"},
		{name: "glob", filter: "  *_test.go  ", wantPattern: "*_test.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotPattern := Parse(tt.filter)
			if gotType != tt.wantType || gotPattern != tt.wantPattern {
				t.Fatalf("Parse(%q) = (%q, %q), want (%q, %q)", tt.filter, gotType, gotPattern, tt.wantType, tt.wantPattern)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		filter string
		want   bool
	}{
		{name: "empty filter matches", path: "pkg/target.go", filter: "", want: true},
		{name: "language filter matches", path: "pkg/target.go", filter: "go", want: true},
		{name: "language filter rejects", path: "pkg/target.py", filter: "go", want: false},
		{name: "alias matches expanded glob", path: "pkg/types.pyi", filter: "python", want: true},
		{name: "glob matches basename", path: "pkg/target_test.go", filter: "*_test.go", want: true},
		{name: "glob matches path", path: "pkg/generated/mock.go", filter: "pkg/generated/*.go", want: true},
		{name: "double star matches root file", path: "target.ts", filter: "**/*.ts", want: true},
		{name: "double star matches nested file", path: "src/pkg/target.ts", filter: "**/*.ts", want: true},
		{name: "double star matches declaration file", path: "src/pkg/target.d.ts", filter: "**/*.d.ts", want: true},
		{name: "double star matches direct child", path: "src/target.ts", filter: "src/**/*.ts", want: true},
		{name: "double star matches deep child", path: "src/pkg/target.ts", filter: "src/**/*.ts", want: true},
		{name: "double star rejects different root", path: "test/pkg/target.ts", filter: "src/**/*.ts", want: false},
		{name: "mixed star double star matches package source", path: "packages/app/src/lib/target.ts", filter: "packages/*/src/**/*.ts", want: true},
		{name: "mixed star double star rejects package test", path: "packages/app/test/target.ts", filter: "packages/*/src/**/*.ts", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(tt.path, tt.filter); got != tt.want {
				t.Fatalf("Matches(%q, %q) = %v, want %v", tt.path, tt.filter, got, tt.want)
			}
		})
	}
}

func TestTypeMapping(t *testing.T) {
	glob, ok := FileTypeGlob("go")
	if !ok || glob != "*.go" {
		t.Fatalf("FileTypeGlob(go) = (%q, %v), want (*.go, true)", glob, ok)
	}
	globs, ok := FileTypeGlobs("python")
	if !ok || len(globs) != 2 || globs[0] != "*.py" || globs[1] != "*.pyi" {
		t.Fatalf("FileTypeGlobs(python) = (%v, %v), want [*.py *.pyi], true", globs, ok)
	}
	args := RipgrepArgs("go", "*.js")
	if len(args) != 2 || args[0] != "--glob" || args[1] != "*.go" {
		t.Fatalf("RipgrepArgs(go, *.js) = %v, want [--glob *.go]", args)
	}
}

func TestPathBasis(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "pkg", "service.js")
	got := MatchPathWithWorkspace(file, filepath.Join(root, "pkg"), root)
	want := filepath.ToSlash(filepath.Join("pkg", "service.js"))
	if got != want {
		t.Fatalf("MatchPathWithWorkspace() = %q, want %q", got, want)
	}

	basis := ResolveSearchPathBasisWithWorkspace(filepath.Join(root, "pkg"), root)
	if basis.Workdir != root || basis.Target != "pkg" || basis.MatchRoot != root {
		t.Fatalf("ResolveSearchPathBasisWithWorkspace() = %+v, want root-scoped pkg basis", basis)
	}
}
