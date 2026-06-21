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

func TestFileTypeGlobMapping(t *testing.T) {
	tests := []struct {
		fileType string
		wantGlob string
		wantOK   bool
	}{
		{fileType: "go", wantGlob: "*.go", wantOK: true},
		{fileType: "py", wantGlob: "*.py", wantOK: true},
		{fileType: "rust", wantGlob: "*.rs", wantOK: true},
		{fileType: "json", wantGlob: "*.json", wantOK: true},
		{fileType: "ex", wantGlob: "*.ex", wantOK: true},
		{fileType: "unknown", wantGlob: "", wantOK: false},
	}

	for _, tt := range tests {
		got, ok := FileTypeGlob(tt.fileType)
		if ok != tt.wantOK || got != tt.wantGlob {
			t.Fatalf("FileTypeGlob(%q) = (%q, %v), want (%q, %v)", tt.fileType, got, ok, tt.wantGlob, tt.wantOK)
		}
	}
}

func TestFileTypeGlobsMapping(t *testing.T) {
	tests := []struct {
		fileType string
		want     []string
		wantOK   bool
	}{
		{fileType: "c", want: []string{"*.c", "*.h", "*.H", "*.[chH].in", "*.cats"}, wantOK: true},
		{fileType: "java", want: []string{"*.java", "*.jsp", "*.jspx", "*.properties"}, wantOK: true},
		{fileType: "sh", want: []string{"*.sh", "*.bash", "*.bashrc", "*.csh", "*.cshrc", "*.env", "*.ksh", "*.kshrc", "*.tcsh", "*.tcshrc", "*.zsh", ".bash_login", ".bash_logout", ".bash_profile", ".bashrc", ".cshrc", ".env", ".kshrc", ".login", ".logout", ".profile", ".tcshrc", ".zlogin", ".zlogout", ".zprofile", ".zshenv", ".zshrc", "bash_login", "bash_logout", "bash_profile", "bashrc", "profile", "zlogin", "zlogout", "zprofile", "zshenv", "zshrc"}, wantOK: true},
		{fileType: "py", want: []string{"*.py", "*.pyi"}, wantOK: true},
		{fileType: "python", want: []string{"*.py", "*.pyi"}, wantOK: true},
		{fileType: "typescript", want: []string{"*.ts", "*.tsx"}, wantOK: true},
		{fileType: "javascript", want: []string{"*.js", "*.jsx", "*.mjs", "*.cjs"}, wantOK: true},
		{fileType: "unknown", want: nil, wantOK: false},
	}

	for _, tt := range tests {
		got, ok := FileTypeGlobs(tt.fileType)
		if ok != tt.wantOK {
			t.Fatalf("FileTypeGlobs(%q) ok = %v, want %v", tt.fileType, ok, tt.wantOK)
		}
		assertStringSlice(t, got, tt.want, "FileTypeGlobs("+tt.fileType+")")
	}
}

func TestRipgrepArgs(t *testing.T) {
	tests := []struct {
		name        string
		fileType    string
		filePattern string
		want        []string
	}{
		{
			name:     "c type uses broadened contract globs",
			fileType: "c",
			want:     []string{"--glob", "*.c", "--glob", "*.h", "--glob", "*.H", "--glob", "*.[chH].in", "--glob", "*.cats"},
		},
		{
			name:     "java type uses broadened contract globs",
			fileType: "java",
			want:     []string{"--glob", "*.java", "--glob", "*.jsp", "--glob", "*.jspx", "--glob", "*.properties"},
		},
		{
			name:     "sh type uses broadened contract globs",
			fileType: "sh",
			want:     []string{"--glob", "*.sh", "--glob", "*.bash", "--glob", "*.bashrc", "--glob", "*.csh", "--glob", "*.cshrc", "--glob", "*.env", "--glob", "*.ksh", "--glob", "*.kshrc", "--glob", "*.tcsh", "--glob", "*.tcshrc", "--glob", "*.zsh", "--glob", ".bash_login", "--glob", ".bash_logout", "--glob", ".bash_profile", "--glob", ".bashrc", "--glob", ".cshrc", "--glob", ".env", "--glob", ".kshrc", "--glob", ".login", "--glob", ".logout", "--glob", ".profile", "--glob", ".tcshrc", "--glob", ".zlogin", "--glob", ".zlogout", "--glob", ".zprofile", "--glob", ".zshenv", "--glob", ".zshrc", "--glob", "bash_login", "--glob", "bash_logout", "--glob", "bash_profile", "--glob", "bashrc", "--glob", "profile", "--glob", "zlogin", "--glob", "zlogout", "--glob", "zprofile", "--glob", "zshenv", "--glob", "zshrc"},
		},
		{
			name:     "cjs token uses glob args",
			fileType: "cjs",
			want:     []string{"--glob", "*.cjs"},
		},
		{
			name:     "javascript alias uses contract globs",
			fileType: "javascript",
			want:     []string{"--glob", "*.js", "--glob", "*.jsx", "--glob", "*.mjs", "--glob", "*.cjs"},
		},
		{
			name:     "python alias includes stub globs",
			fileType: "python",
			want:     []string{"--glob", "*.py", "--glob", "*.pyi"},
		},
		{
			name:     "typescript alias uses contract globs",
			fileType: "typescript",
			want:     []string{"--glob", "*.ts", "--glob", "*.tsx"},
		},
		{
			name:        "file type still overrides file pattern",
			fileType:    "go",
			filePattern: "*.js",
			want:        []string{"--glob", "*.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RipgrepArgs(tt.fileType, tt.filePattern)
			assertStringSlice(t, got, tt.want, "RipgrepArgs")
		})
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

func TestMatchPathUsesSearchBasis(t *testing.T) {
	root := t.TempDir()
	absFile := filepath.Join(root, "pkg", "service.js")

	tests := []struct {
		name string
		path string
		file string
		want string
	}{
		{
			name: "absolute path becomes search-root relative",
			path: root,
			file: absFile,
			want: filepath.ToSlash(filepath.Join("pkg", "service.js")),
		},
		{
			name: "relative file path stays relative",
			path: "pkg",
			file: filepath.ToSlash(filepath.Join("pkg", "service.js")),
			want: filepath.ToSlash(filepath.Join("pkg", "service.js")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchPath(tt.file, tt.path); got != tt.want {
				t.Fatalf("MatchPath(%q, %q) = %q, want %q", tt.file, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchPathWithWorkspacePreservesWorkspaceRelativeGlobBasis(t *testing.T) {
	root := t.TempDir()
	absFile := filepath.Join(root, "pkg", "service.js")

	got := MatchPathWithWorkspace(absFile, filepath.Join(root, "pkg"), root)
	want := filepath.ToSlash(filepath.Join("pkg", "service.js"))
	if got != want {
		t.Fatalf("MatchPathWithWorkspace(%q, %q, %q) = %q, want %q", absFile, filepath.Join(root, "pkg"), root, got, want)
	}
}

func TestResolveSearchPathBasisAlignsExecutionAndMatchRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "pkg", "service.js")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("class UserService {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		path          string
		wantWorkdir   string
		wantTarget    string
		wantMatchRoot string
	}{
		{
			name:          "absolute directory uses root-relative execution basis",
			path:          root,
			wantWorkdir:   root,
			wantTarget:    ".",
			wantMatchRoot: root,
		},
		{
			name:          "absolute file uses parent directory as basis",
			path:          file,
			wantWorkdir:   filepath.Dir(file),
			wantTarget:    filepath.Base(file),
			wantMatchRoot: filepath.Dir(file),
		},
		{
			name:          "relative path stays relative",
			path:          filepath.ToSlash(filepath.Join("pkg", "service.js")),
			wantWorkdir:   "",
			wantTarget:    filepath.ToSlash(filepath.Join("pkg", "service.js")),
			wantMatchRoot: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveSearchPathBasis(tt.path)
			if got.Workdir != tt.wantWorkdir || got.Target != tt.wantTarget || got.MatchRoot != tt.wantMatchRoot {
				t.Fatalf("ResolveSearchPathBasis(%q) = %+v, want workdir=%q target=%q matchRoot=%q", tt.path, got, tt.wantWorkdir, tt.wantTarget, tt.wantMatchRoot)
			}
		})
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

func assertStringSlice(t *testing.T, got, want []string, label string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s len = %d, want %d: got %v want %v", label, len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q: got %v want %v", label, i, got[i], want[i], got, want)
		}
	}
}
