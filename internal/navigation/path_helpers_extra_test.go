package navigation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

func TestLSPLocationFilePath_ResolvesExistingAndFallbackPaths(t *testing.T) {
	rootDir := t.TempDir()
	invocationDir := filepath.Join(rootDir, "workspace")
	rootOnlyDir := filepath.Join(rootDir, "root")
	if err := os.MkdirAll(filepath.Join(invocationDir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(rootOnlyDir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invocationDir, "pkg", "invocation.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootOnlyDir, "pkg", "root.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		file          string
		rootPath      string
		invocationCWD string
		want          string
	}{
		{
			name:          "absolute path is preserved",
			file:          filepath.Join(rootDir, "abs.go"),
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(rootDir, "abs.go"),
		},
		{
			name:          "existing invocation path wins",
			file:          "pkg/invocation.go",
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(invocationDir, "pkg", "invocation.go"),
		},
		{
			name:          "existing root path is used",
			file:          "pkg/root.go",
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(rootOnlyDir, "pkg", "root.go"),
		},
		{
			name:          "missing file falls back to invocation cwd join",
			file:          "pkg/missing.go",
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(invocationDir, "pkg", "missing.go"),
		},
		{
			name:          "missing file falls back to root join",
			file:          "pkg/missing.go",
			rootPath:      rootOnlyDir,
			invocationCWD: "",
			want:          filepath.Join(rootOnlyDir, "pkg", "missing.go"),
		},
		{
			name:          "empty bases return cleaned relative path",
			file:          "pkg/missing.go",
			rootPath:      "",
			invocationCWD: "",
			want:          filepath.Clean(filepath.FromSlash("pkg/missing.go")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lspLocationFilePath(tt.file, tt.rootPath, tt.invocationCWD); got != tt.want {
				t.Fatalf("lspLocationFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCandidateAbsPath_UsesAbsoluteRootAndCWD(t *testing.T) {
	rootDir := t.TempDir()
	rootFile := filepath.Join(rootDir, "pkg", "root.go")
	if err := os.MkdirAll(filepath.Dir(rootFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwdDir := t.TempDir()
	cwdFile := filepath.Join(cwdDir, "cwd.go")
	if err := os.WriteFile(cwdFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwdDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	tests := []struct {
		name string
		cand SymbolCandidate
		want string
	}{
		{
			name: "empty file returns empty path",
			cand: SymbolCandidate{},
			want: "",
		},
		{
			name: "absolute file is cleaned",
			cand: SymbolCandidate{File: rootFile},
			want: filepath.Clean(rootFile),
		},
		{
			name: "root path resolves relative file",
			cand: SymbolCandidate{File: "pkg/root.go", RootPath: rootDir},
			want: rootFile,
		},
		{
			name: "cwd resolves relative file without root",
			cand: SymbolCandidate{File: "cwd.go"},
			want: cwdFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := candidateAbsPath(tt.cand); got != tt.want {
				t.Fatalf("candidateAbsPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPreferredInspectRootPath_ChoosesBroaderKnownRoot(t *testing.T) {
	parent := t.TempDir()
	projectRoot := filepath.Join(parent, "repo")
	symbolRoot := filepath.Join(projectRoot, "pkg")
	unrelated := filepath.Join(parent, "other")
	if err := os.MkdirAll(symbolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		symbolRoot  string
		projectRoot string
		want        string
	}{
		{"prefer project when symbol root empty", "", projectRoot, projectRoot},
		{"prefer symbol when project root empty", symbolRoot, "", symbolRoot},
		{"prefer nested symbol root", symbolRoot, projectRoot, symbolRoot},
		{"prefer nested project root", projectRoot, symbolRoot, symbolRoot},
		{"prefer project on unrelated roots", unrelated, projectRoot, projectRoot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := preferredInspectRootPath(tt.symbolRoot, tt.projectRoot); got != tt.want {
				t.Fatalf("preferredInspectRootPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeResultFilePath_RecoversSnapshotRelativePaths(t *testing.T) {
	rootDir := t.TempDir()
	targetRoot := filepath.Join(rootDir, "repo")
	sourceBase := filepath.Join(targetRoot, "pkg")
	absFile := filepath.Join(targetRoot, "pkg", "run.go")
	if err := os.MkdirAll(filepath.Dir(absFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(targetRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	tests := []struct {
		name       string
		path       string
		targetRoot string
		sourceBase string
		want       string
	}{
		{"absolute path becomes relative", absFile, targetRoot, sourceBase, "pkg/run.go"},
		{"root relative existing path stays relative", "pkg/run.go", targetRoot, sourceBase, "pkg/run.go"},
		{"source base relative file is recovered", "run.go", targetRoot, sourceBase, "pkg/run.go"},
		{"cwd relative file is recovered", filepath.ToSlash(filepath.Join("pkg", "run.go")), targetRoot, "", "pkg/run.go"},
		{"source base fallback keeps nested relative path", "pkg/../missing.go", targetRoot, sourceBase, "pkg/missing.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeResultFilePath(tt.path, tt.targetRoot, tt.sourceBase); got != tt.want {
				t.Fatalf("normalizeResultFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLSPAdapter_NewAndToAbsPath(t *testing.T) {
	if got := NewLSPAdapter(nil, "/workspace"); got != nil {
		t.Fatalf("NewLSPAdapter(nil) = %+v, want nil", got)
	}

	adapter := NewLSPAdapter(&lsp.Client{}, "/workspace")
	if adapter == nil {
		t.Fatal("NewLSPAdapter() should return adapter for non-nil client")
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"absolute path is preserved", "/tmp/file.go", "/tmp/file.go"},
		{"relative path joins root", "pkg/file.go", filepath.Join("/workspace", "pkg/file.go")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adapter.toAbsPath(tt.path); got != tt.want {
				t.Fatalf("toAbsPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLSPAdapter_ConvertLocationsWithoutRootPreservesAbsolutePaths(t *testing.T) {
	adapter := NewLSPAdapter(&lsp.Client{}, "")
	locations := adapter.convertLocations([]lsp.Location{{
		URI: "file:///tmp/example.go",
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 4},
		},
	}})

	if len(locations) != 1 {
		t.Fatalf("len(locations) = %d, want 1", len(locations))
	}
	if locations[0].File != filepath.Clean("/tmp/example.go") {
		t.Fatalf("File = %q, want absolute path", locations[0].File)
	}
}
