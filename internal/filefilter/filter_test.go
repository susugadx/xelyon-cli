package filefilter

import (
	"os"
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

func TestWorkspaceRelativePath(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "pkg", "service.go")
	outside := filepath.Join(t.TempDir(), "pkg", "service.go")

	tests := []struct {
		name          string
		filePath      string
		workspaceRoot string
		want          string
	}{
		{
			name:          "workspace absolute path becomes relative",
			filePath:      inside,
			workspaceRoot: root,
			want:          "pkg/service.go",
		},
		{
			name:          "outside absolute path is preserved",
			filePath:      outside,
			workspaceRoot: root,
			want:          filepath.ToSlash(filepath.Clean(outside)),
		},
		{
			name:          "relative path stays relative",
			filePath:      filepath.Join("pkg", "service.go"),
			workspaceRoot: root,
			want:          "pkg/service.go",
		},
		{
			name:          "empty workspace keeps absolute basis",
			filePath:      inside,
			workspaceRoot: "",
			want:          filepath.ToSlash(filepath.Clean(inside)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkspaceRelativePath(tt.filePath, tt.workspaceRoot); got != tt.want {
				t.Fatalf("WorkspaceRelativePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveSearchPathBasisWithWorkspace(t *testing.T) {
	root := t.TempDir()
	existingDir := filepath.Join(root, "pkg")
	if err := os.Mkdir(existingDir, 0o700); err != nil {
		t.Fatalf("Mkdir(pkg) error = %v", err)
	}
	existingFile := filepath.Join(existingDir, "service.go")
	if err := os.WriteFile(existingFile, []byte("package pkg\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(service.go) error = %v", err)
	}
	missingAbsolute := filepath.Join(root, "missing", "service.go")

	tests := []struct {
		name          string
		searchPath    string
		workspaceRoot string
		want          SearchPathBasis
	}{
		{
			name:          "workspace root absolute path scopes to dot",
			searchPath:    root,
			workspaceRoot: root,
			want:          SearchPathBasis{Workdir: root, Target: ".", MatchRoot: root},
		},
		{
			name:          "workspace child absolute path scopes to relative target",
			searchPath:    existingFile,
			workspaceRoot: root,
			want:          SearchPathBasis{Workdir: root, Target: "pkg/service.go", MatchRoot: root},
		},
		{
			name:       "existing directory without workspace is its own workdir",
			searchPath: existingDir,
			want:       SearchPathBasis{Workdir: existingDir, Target: ".", MatchRoot: existingDir},
		},
		{
			name:       "existing file without workspace uses parent workdir",
			searchPath: existingFile,
			want:       SearchPathBasis{Workdir: existingDir, Target: "service.go", MatchRoot: existingDir},
		},
		{
			name:       "missing absolute path falls back to target",
			searchPath: missingAbsolute,
			want:       SearchPathBasis{Target: filepath.Clean(missingAbsolute)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveSearchPathBasisWithWorkspace(tt.searchPath, tt.workspaceRoot)
			if got != tt.want {
				t.Fatalf("ResolveSearchPathBasisWithWorkspace() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
